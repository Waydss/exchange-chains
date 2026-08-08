package cb

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
	"github.com/jackc/pgx/v5"
)

// errChainScoreTooLow signals a chain that does not pass the final score gate.
// var errChainScoreTooLow =

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
	chains     []exchange.Chain
}

// Entry point for the Chain evaluation process. This function looks over the ChainRequestQuene
// for the newly arrived ChainRequest structs.
//
// For the actuall Chain evaluation functionality - refer to the buildClosedChains function
func (s *Service) StartChainRequestWorkers(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(chain_worker_count)

	for range chain_worker_count {
		go func() {
			defer wg.Done()
			for {
				select {
				case req := <-s.queue:
					s.handleChainRequest(ctx, req)
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

// Function for ChainRequest validation. Please make sure to validate the IDs beforehand,
// so we could return HTTP error about it.
func (s *Service) handleChainRequest(ctx context.Context, req ChainRequest) {
	root, err := s.repository.GetAccount(ctx, req.UserID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Println("Chain request skipped: account ID has not been validated")
			return
		}
		log.Println("Chain request skipped: database error")
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
		log.Printf("Chain request skipped: WishedItemID [%d] has not been found for AccountID [%d];", req.WishedItemId, root.ID)
		return
	}

	start := time.Now()
	chains := s.buildClosedChains(ctx, &root, wishIndex)
	log.Printf("Chain request done: account=%d wish=%d chains=%d in %v", root.ID, req.WishedItemId, len(chains), time.Since(start))
}

// Builds a new Chains for provided Account and Wished Item index.
// Index of the latter is based upon the actual DB's ID of this item.
//
// This function uses Directional BFS Algorithm, where edges form on current.give -> others.wish relations.
//
// Chains are written to the Table on each buildChainFromState() call, to ensure data is appearing while the process is still running.
//
// For lookup functionality - refer to the rankNextCandidates() function
func (s *Service) buildClosedChains(ctx context.Context, root *userCandidate, rootWishIndex int) []*exchange.Chain {
	if root == nil || rootWishIndex < 0 || rootWishIndex >= len(root.Wishes) {
		return nil
	}

	rootWish := root.Wishes[rootWishIndex]
	startGive, ok := root.CurrentGive()
	if !ok {
		log.Printf("Build abandoned - Chain's skipped. User (ID: %v) doesn't have any suitable gives for start\n", root.ID)
		return nil
	}
	rootDepthLimit := uint8(allowedDepth(root.ShadowRating))

	results := make([]exchange.Chain, 0)
	queue := make([]*searchState, 0)

	for _, cand := range s.rankNextCandidates(ctx, root.ID, root.ID, startGive, nil) {
		edge := barterEdge{
			From:       root.ID,
			To:         cand.AccountID,
			Give:       startGive,
			Wish:       cand.Wish,
			FuzzyScore: cand.FuzzyScore,
			MatchScore: cand.MatchScore,
			CreatedAt:  maxTime(cand.CurrentGive.UpdatedAt, rootWish.UpdatedAt),
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

		expansions := s.expandLayerParallel(ctx, root, rootWish, queue)
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

	uniq := make([]*exchange.Chain, 0, len(results))
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

func (s *Service) expandLayerParallel(ctx context.Context, root *userCandidate, rootWish exchange.WishedItem, states []*searchState) []stateExpansion {
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
				results <- s.expandState(ctx, root, rootWish, state)
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

func (s *Service) expandState(ctx context.Context, root *userCandidate, rootWish exchange.WishedItem, state *searchState) stateExpansion {
	if state == nil || int(state.Depth) >= max_barters_per_chain {
		return stateExpansion{}
	}

	children := make([]*searchState, 0)
	completed := make([]exchange.Chain, 0)

	if state.Depth >= 1 {
		closingGive, fuzzy, ok := s.bestClosingGive(ctx, state.CurrentID, rootWish)
		if ok {
			closing := &searchState{
				Parent:      state,
				CurrentID:   root.ID,
				CurrentGive: closingGive,
				Depth:       state.Depth + 1,
				DepthLimit:  state.DepthLimit,
				Score:       state.Score,
				Step: barterEdge{
					From:       state.CurrentID,
					To:         root.ID,
					Give:       closingGive,
					Wish:       rootWish,
					FuzzyScore: fuzzy,
					MatchScore: candidateScore(fuzzy, root.PublicRating, root.ShadowRating),
					CreatedAt:  maxTime(closingGive.UpdatedAt, rootWish.UpdatedAt),
				},
			}

			newChain, err := s.buildChainFromState(ctx, root, rootWish, closing)

			if err == nil {
				completed = append(completed, newChain)
			}
		}
	}

	for _, cand := range s.rankNextCandidates(ctx, state.CurrentID, root.ID, state.CurrentGive, state) {
		if cand.AccountID == root.ID {
			continue
		}

		nextState := &searchState{
			Parent:      state,
			CurrentID:   cand.AccountID,
			CurrentGive: cand.CurrentGive,
			Depth:       state.Depth + 1,
			DepthLimit:  minUint8(state.DepthLimit, uint8(allowedDepth(cand.ShadowRating))),
			Score:       state.Score + cand.MatchScore,
			Step: barterEdge{
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
func (s *Service) rankNextCandidates(ctx context.Context, sourceID, rootID int64, give exchange.GivenItem, state *searchState) []candidate {
	if !use_cache {
		return filterCandidates(s.buildCandidatesForGive(ctx, give), sourceID, rootID, state)
	}

	candidates, ok := giveCandidatesCache.Load(give.ID)
	if !ok {
		cacheMisses.Add(1)
		actual, _ := giveCandidatesCache.LoadOrStore(give.ID, s.buildCandidatesForGive(ctx, give))
		candidates = actual
	} else {
		cacheHits.Add(1)
	}

	return filterCandidates(candidates.([]candidate), sourceID, rootID, state)
}

var (
	giveCandidatesCache sync.Map
	accountGivesCache   sync.Map
	cacheHits           atomic.Int64
	cacheMisses         atomic.Int64
	chainWriteMu        sync.Mutex
)

func (s *Service) buildCandidatesForGive(ctx context.Context, give exchange.GivenItem) []candidate {
	possibleCandidates, err := s.repository.FindCandidateAccounts(ctx, give, candidate_limit)
	if err != nil {
		log.Printf("Error while loading chain candidates: %v", err)
		return nil
	}

	candidates := make([]candidate, 0, len(possibleCandidates))

	for _, account := range possibleCandidates {
		for _, wish := range account.Wishes {
			fuzzy := fuzzySimilarity(give.Title, wish.TitleKeywords)
			if fuzzy < min_fuzzy_score {
				continue
			}

			score := candidateScore(fuzzy, account.PublicRating, account.ShadowRating)
			if score < min_final_score {
				continue
			}

			candidates = append(candidates, candidate{
				AccountID:    account.ID,
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

func filterCandidates(candidates []candidate, sourceID, rootID int64, state *searchState) []candidate {
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

func (s *Service) buildChainFromState(ctx context.Context, root *userCandidate, rootWish exchange.WishedItem, state *searchState) (exchange.Chain, error) {
	path := state.Path()
	for i := range path {
		path[i].ID = int64(i + 1)
	}

	convertedPath := make([]exchange.Barter, len(path))
	score := 0.0
	for i, barter := range path {
		score += barter.MatchScore
		convertedPath[i] = barter.VertexToBarter()
	}

	newChain := exchange.Chain{
		RootID:    root.ID,
		RootWish:  rootWish,
		Barters:   convertedPath,
		Signature: chainSignature(exchange.Chain{RootID: root.ID, RootWish: rootWish, Barters: convertedPath}),
		Score:     score,
	}

	if score < min_final_score {
		log.Printf("Chain skipped: score %v/%v", score, min_final_score)
		return exchange.Chain{}, errors.New("chain score below min_final_score")
	}

	chainWriteMu.Lock()

	err := s.repository.CreateChain(ctx, newChain)

	chainWriteMu.Unlock()

	if err != nil {
		log.Printf("Error while writing the chain: %v", err)
	}

	return newChain, nil
}

func currentGive(account *userCandidate) exchange.GivenItem {
	give, _ := account.CurrentGive()
	return give
}

// accountGives returns all searching gives of the account, cached per account ID.
func (s *Service) accountGives(ctx context.Context, accountID int64) []exchange.GivenItem {
	if cached, ok := accountGivesCache.Load(accountID); ok {
		return cached.([]exchange.GivenItem)
	}

	gives, err := s.repository.GetAccountGives(ctx, accountID)
	if err != nil {
		log.Printf("Error while loading account gives for %d: %v", accountID, err)
		return nil
	}

	actual, _ := accountGivesCache.LoadOrStore(accountID, gives)
	return actual.([]exchange.GivenItem)
}

// bestClosingGive picks the give of the last account in the chain that best
// matches the root wish. Reverse of buildCandidatesForGive: it filters the
// account's gives by the root wish (category + price + fuzzy >= min_fuzzy_score)
// and returns the one with the highest fuzzy score.
func (s *Service) bestClosingGive(ctx context.Context, accountID int64, rootWish exchange.WishedItem) (exchange.GivenItem, float64, bool) {
	var bestGive exchange.GivenItem
	bestFuzzy := 0.0
	found := false

	for _, give := range s.accountGives(ctx, accountID) {
		if !wishMatchesGive(rootWish, give) {
			continue
		}

		fuzzy := fuzzySimilarity(give.Title, rootWish.TitleKeywords)
		if fuzzy < min_fuzzy_score {
			continue
		}

		if !found || fuzzy > bestFuzzy {
			bestGive = give
			bestFuzzy = fuzzy
			found = true
		}
	}

	return bestGive, bestFuzzy, found
}
