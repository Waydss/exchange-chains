package cb

import (
	"context"
	"log"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/chain_builder/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// The maximum length of the chain.
	// The Shadow Rating limitation of it is evaluated with allowedDepth() function
	max_barters_per_chain = 5

	// Value for the SQL request filtering while getting
	// the possible candidates for the next BFS layer
	candidate_limit = 100

	// TODO min_fuzzy_score needs further finetuning

	// NEEDS FINETUNING!. Used for sorting possible candidates for the chain layer.
	// Refer to fuzzySimilarity() function to get more details
	min_fuzzy_score = 0.45

	// TODO min_final_score needs further finetuning

	// NEEDS FINETUNING! Used for sorting possible candidates for the chain layer.
	min_final_score = 10

	// Depricated: caching already helps a lot in deeper BFS layers (3 and 4),
	// so it will be removed soon, when i'll get to it
	use_cache = true

	// It is not recomended to change this value to a higher number,
	// because the expandLayerParallel() function uses go's GOMAXPROCS
	// value to evaluate the amount of threads avaliable
	chain_worker_count = 1
)

type stateExpansion struct {
	nextStates []*searchState
	chains     []entities.Chain
}

// var ChainRequestQuene []ChainRequest = []ChainRequest{}
const chain_request_queue_size = 100

var ChainRequestQuene = make(chan ChainRequest, chain_request_queue_size)

func EnqueueChainRequest(req ChainRequest) {
	ChainRequestQuene <- req
}

// Entry point for the Chain evaluation process. This function looks over the ChainRequestQuene
// for the newly arrived ChainRequest structs.
//
// For the actuall Chain evaluation functionality - refer to the buildClosedChains function
func StartChainRequestWorkers(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(chain_worker_count)

	for range chain_worker_count {
		go func() {
			defer wg.Done()
			for {
				select {
				case req := <-ChainRequestQuene:
					handleChainRequest(req)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	return done
}

// Function for ChainRequest validation
func handleChainRequest(req ChainRequest) {
	root := req.UserAccount
	if root == nil {
		log.Println("Chain request skipped: account pointer is nil")
		return
	}

	wishIndex := -1
	for i := range root.Wishes {
		if root.Wishes[i].ID == req.WishedItemId {
			wishIndex = i
			break
		}
	}
	if wishIndex < 0 {
		log.Printf("Chain request skipped: wished item %d not found for account %d; total wishes: %v", req.WishedItemId, root.Id, len(root.Wishes))
		return
	}

	start := time.Now()
	chains := buildClosedChains(root, wishIndex)
	log.Printf("Chain request done: account=%d wish=%d chains=%d in %v", root.Id, req.WishedItemId, len(chains), time.Since(start))
}

// Builds a new Chains for provided Account and Wished Item index.
// Index of the latter is based upon the actual DB's ID of this item.
//
// This function uses Directional BFS Algorithm, where edges form on current.give -> others.wish relations.
//
// Chains are written to the Table on each buildChainFromState() call, to ensure data is appearing while the process is still running.
//
// For lookup functionality - refer to the rankNextCandidates() function
func buildClosedChains(root *entities.Account, rootWishIndex int) []*entities.Chain {
	if root == nil || rootWishIndex < 0 || rootWishIndex >= len(root.Wishes) {
		return nil
	}

	rootWish := root.Wishes[rootWishIndex]
	startGive, ok := root.CurrentGive()
	if !ok {
		return nil
	}
	rootDepthLimit := uint8(allowedDepth(root.ShadowRating))

	results := make([]entities.Chain, 0)
	queue := make([]*searchState, 0)

	for _, cand := range rankNextCandidates(root.Id, root.Id, startGive, nil) {
		edge := entities.Barter{
			From:       root.Id,
			To:         cand.AccountID,
			Give:       startGive,
			Wish:       cand.Wish,
			FuzzyScore: cand.FuzzyScore,
			MatchScore: cand.MatchScore,
			CreatedAt:  maxTime(startGive.UpdatedAt, cand.Wish.UpdatedAt),
		}

		nextState := &searchState{
			CurrentID:   cand.AccountID,
			CurrentGive: cand.CurrentGive,
			Step:        edge,
			Depth:       1,
			DepthLimit:  minUint8(rootDepthLimit, uint8(allowedDepth(cand.ShadowRating))),
			Score:       cand.MatchScore,
		}

		if nextState.Depth > nextState.DepthLimit {
			continue
		}

		queue = append(queue, nextState)
	}

	searchStart := time.Now()

	for len(queue) > 0 {
		layerStart := time.Now()
		layerSize := len(queue)
		layerDepth := queue[0].Depth

		expansions := expandLayerParallel(root, rootWish, queue)
		queue = queue[:0]

		layerChains := 0
		for _, expansion := range expansions {
			for i := range expansion.chains {
				chain := expansion.chains[i]
				log.Printf("CHAIN #%d found [t=%.2fs] depth=%d score=%.2f barters=%d", len(results)+layerChains+1, time.Since(searchStart).Seconds(), layerDepth, chain.Score, len(chain.Barters))
			}
			layerChains += len(expansion.chains)
			results = append(results, expansion.chains...)
			queue = append(queue, expansion.nextStates...)
		}

		log.Printf("LAYER depth=%d size=%d -> next=%d time=%.2fs chains=%d", layerDepth, layerSize, len(queue), time.Since(layerStart).Seconds(), layerChains)
	}

	if len(results) == 0 {
		return nil
	}

	uniq := make([]*entities.Chain, 0, len(results))
	seen := map[string]bool{}
	for _, chain := range results {
		sig := chainSignature(chain)
		if seen[sig] {
			continue
		}
		seen[sig] = true
		uniq = append(uniq, &chain)
	}

	hits := cacheHits.Load()
	misses := cacheMisses.Load()
	if hits+misses > 0 {
		log.Printf("CACHE stats: hits=%d misses=%d (hit=%.1f%%)", hits, misses, 100*float64(hits)/float64(hits+misses))
	}

	return uniq
}

func expandLayerParallel(root *entities.Account, rootWish entities.WishedItem, states []*searchState) []stateExpansion {
	if len(states) == 0 {
		return nil
	}

	workerCount := runtime.GOMAXPROCS(0)

	if workerCount < 1 {
		workerCount = 1
	}

	if workerCount > len(states) {
		workerCount = len(states)
	}

	jobs := make(chan *searchState)
	results := make(chan stateExpansion, len(states))
	var wg sync.WaitGroup
	wg.Add(workerCount)

	for range workerCount {
		go func() {
			defer wg.Done()
			for state := range jobs {
				results <- expandState(root, rootWish, state)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, state := range states {
			jobs <- state
		}
		close(jobs)
	}()

	out := make([]stateExpansion, 0, len(states))
	for expansion := range results {
		out = append(out, expansion)
	}

	return out
}

func expandState(root *entities.Account, rootWish entities.WishedItem, state *searchState) stateExpansion {
	if state == nil || int(state.Depth) >= max_barters_per_chain {
		return stateExpansion{}
	}

	children := make([]*searchState, 0)
	completed := make([]entities.Chain, 0)

	if state.Depth >= 1 && wishMatchesGive(rootWish, state.CurrentGive) {
		fuzzy := fuzzySimilarity(state.CurrentGive.Name, rootWish.Name)
		closing := &searchState{
			Parent:      state,
			CurrentID:   root.Id,
			CurrentGive: state.CurrentGive,
			Depth:       state.Depth + 1,
			DepthLimit:  state.DepthLimit,
			Score:       state.Score,
			Step: entities.Barter{
				From:       state.CurrentID,
				To:         root.Id,
				Give:       state.CurrentGive,
				Wish:       rootWish,
				FuzzyScore: fuzzy,
				MatchScore: candidateScore(fuzzy, root.PublicRating, root.ShadowRating),
				CreatedAt:  maxTime(state.CurrentGive.UpdatedAt, rootWish.UpdatedAt),
			},
		}
		completed = append(completed, buildChainFromState(root, rootWish, closing))
	}

	for _, cand := range rankNextCandidates(state.CurrentID, root.Id, state.CurrentGive, state) {
		if cand.AccountID == root.Id {
			continue
		}

		nextState := &searchState{
			Parent:      state,
			CurrentID:   cand.AccountID,
			CurrentGive: cand.CurrentGive,
			Depth:       state.Depth + 1,
			DepthLimit:  minUint8(state.DepthLimit, uint8(allowedDepth(cand.ShadowRating))),
			Score:       state.Score + cand.MatchScore,
			Step: entities.Barter{
				From:       state.CurrentID,
				To:         cand.AccountID,
				Give:       state.CurrentGive,
				Wish:       cand.Wish,
				FuzzyScore: cand.FuzzyScore,
				MatchScore: cand.MatchScore,
				CreatedAt:  maxTime(state.CurrentGive.UpdatedAt, cand.Wish.UpdatedAt),
			},
		}

		if nextState.Depth > nextState.DepthLimit {
			continue
		}

		children = append(children, nextState)
	}

	return stateExpansion{nextStates: children, chains: completed}
}

// Creates a collection of possible candidates based on their wishes.
//
// Getting all possible candidates for constructing a layer brokes into 3 steps: 1. filtering request to the database; 2. further filtering to ensure rootId is present in the slice; 3. sorting on final_score (refer to candidateScore()).
//
// Then current graph state is extending with this data and Chain building continues.
func rankNextCandidates(sourceID, rootID uint, give entities.GivedItem, state *searchState) []candidate {
	if !use_cache {
		return filterCandidates(buildCandidatesForGive(give), sourceID, rootID, state)
	}

	candidates, ok := giveCandidatesCache.Load(give.ID)
	if !ok {
		cacheMisses.Add(1)
		actual, _ := giveCandidatesCache.LoadOrStore(give.ID, buildCandidatesForGive(give))
		candidates = actual
	} else {
		cacheHits.Add(1)
	}

	return filterCandidates(candidates.([]candidate), sourceID, rootID, state)
}

var (
	giveCandidatesCache sync.Map
	cacheHits           atomic.Int64
	cacheMisses         atomic.Int64
	chainWriteMu        sync.Mutex
)

func buildCandidatesForGive(give entities.GivedItem) []candidate {
	possibleCandidates := make([]entities.Account, 0)

	// FIXME We need to start using gorm, because i can't live without it...
	database.DB.Model(&entities.Account{}).
		Select("DISTINCT accounts.id, accounts.name, accounts.public_rating, accounts.shadow_rating").
		Joins("JOIN wished_items ON wished_items.user_id = accounts.id").
		Where("wished_items.category = ? AND wished_items.min_price <= ? AND wished_items.max_price >= ?", give.Category, give.Price, give.Price).
		Preload("Gives", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("updated_at")
		}).
		Preload("Wishes", "wished_items.category = ? AND wished_items.min_price <= ? AND wished_items.max_price >= ?", give.Category, give.Price, give.Price).
		Limit(candidate_limit).
		Find(&possibleCandidates)

	candidates := make([]candidate, 0, len(possibleCandidates))

	for _, account := range possibleCandidates {
		for _, wish := range account.Wishes {
			fuzzy := fuzzySimilarity(give.Name, wish.Name)
			if fuzzy < min_fuzzy_score {
				continue
			}

			score := candidateScore(fuzzy, account.PublicRating, account.ShadowRating)
			if score < min_final_score {
				continue
			}

			candidates = append(candidates, candidate{
				AccountID:    account.Id,
				PublicRating: account.PublicRating,
				ShadowRating: account.ShadowRating,
				CurrentGive:  currentGive(&account),
				Wish:         wish,
				FuzzyScore:   fuzzy,
				MatchScore:   score,
			})
		}
	}

	return candidates
}

func filterCandidates(candidates []candidate, sourceID, rootID uint, state *searchState) []candidate {
	filtered := make([]candidate, 0, len(candidates))

	for _, cand := range candidates {
		if cand.AccountID == sourceID {
			continue
		}

		if cand.AccountID != rootID && state != nil && state.HasInPath(cand.AccountID) {
			continue
		}

		filtered = append(filtered, cand)
	}

	// TODO Needs further testing, to validate if this chunk of code could be removed or not
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].MatchScore == filtered[j].MatchScore {
			if filtered[i].FuzzyScore == filtered[j].FuzzyScore {
				return filtered[i].PublicRating > filtered[j].PublicRating
			}
			return filtered[i].FuzzyScore > filtered[j].FuzzyScore
		}
		return filtered[i].MatchScore > filtered[j].MatchScore
	})

	if len(filtered) > candidate_limit {
		filtered = filtered[:candidate_limit]
	}

	return filtered
}

func buildChainFromState(root *entities.Account, rootWish entities.WishedItem, state *searchState) entities.Chain {
	path := state.Path()
	for i := range path {
		path[i].Id = uint(i + 1)
	}

	score := 0.0
	for _, barter := range path {
		score += barter.MatchScore
	}

	newChain := entities.Chain{
		Id:        root.Id,
		UserId:    root.Id,
		RootId:    root.Id,
		RootWish:  rootWish,
		Barters:   path,
		Signature: chainSignature(entities.Chain{RootId: root.Id, RootWish: rootWish, Barters: path}),
		Complete:  len(path) > 0 && len(path) <= max_barters_per_chain,
		Score:     score,
	}

	dbChain := newChain
	dbChain.Id = 0
	dbChain.UserId = root.Id
	dbChain.Signature = newChain.Signature

	chainWriteMu.Lock()

	// FIXME We need to start using gorm, because i can't live without it...
	err := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&dbChain).Error
	chainWriteMu.Unlock()
	if err != nil {
		log.Printf("Error while writing the chain: %v", err)
	}

	return newChain
}

func currentGive(account *entities.Account) entities.GivedItem {
	give, _ := account.CurrentGive()
	return give
}
