package cb

import (
	"time"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

// Internal model for chain_builder package,
// used for storing information for Given and Wished items
// along side the useful for this package User information
type userCandidate struct {
	ID           int64
	Wishes       []exchange.WishedItem
	Gives        []exchange.GivenItem
	PublicRating float64
	ShadowRating int
}

// Internal model for chain_builder package,
// used for representation of the edge (vertex) construction in the graph
type barterVertex struct {
	ID         int64
	From       int64
	To         int64
	Give       exchange.GivenItem
	Wish       exchange.WishedItem
	FuzzyScore float64
	MatchScore float64
	CreatedAt  time.Time
}

func (v barterVertex) VertexToBarter() exchange.Barter {
	return exchange.Barter{
		ID:        v.ID,
		From:      v.From,
		Give:      v.Give,
		To:        v.To,
		Wish:      v.Wish,
		CreatedAt: v.CreatedAt,
	}
}

// Returns the latest version of Give item, determined by the UpdateAt time.
//
// Probably not that useful currently, if items updated by creating a new entry each time
func (a userCandidate) CurrentGive() (exchange.GivenItem, bool) {
	if len(a.Gives) == 0 {
		return exchange.GivenItem{}, false
	}
	best := a.Gives[0]
	for _, give := range a.Gives[1:] {
		if give.UpdatedAt.After(best.UpdatedAt) {
			best = give
		}
	}
	return best, true
}

func (a userCandidate) CurrentWish() (exchange.WishedItem, bool) {
	if len(a.Wishes) == 0 {
		return exchange.WishedItem{}, false
	}
	best := a.Wishes[0]
	for _, wish := range a.Wishes[1:] {
		if wish.UpdatedAt.After(best.UpdatedAt) {
			best = wish
		}
	}
	return best, true
}

// Internal model for chain_builder package,
// used as representation of Vetecies in the BFS Graph
type candidate struct {
	AccountID    int64
	PublicRating float64
	ShadowRating int
	CurrentGive  exchange.GivenItem
	Wish         exchange.WishedItem
	FuzzyScore   float64
	MatchScore   float64
}

// Internal model for chain_builder package,
// used for representation of current BFS lookup state for further building of the graph
type searchState struct {
	Parent      *searchState
	CurrentID   int64
	CurrentGive exchange.GivenItem
	Step        barterVertex
	Depth       uint8
	DepthLimit  uint8
	Score       float64
}

func (s *searchState) HasInPath(accountID int64) bool {
	for cur := s; cur != nil; cur = cur.Parent {
		if cur.CurrentID == accountID {
			return true
		}
	}

	return false
}

func (s *searchState) PathAccountIDs() []int64 {
	if s == nil {
		return nil
	}

	ids := make([]int64, 0, s.Depth)
	for cur := s; cur != nil; cur = cur.Parent {
		ids = append(ids, cur.CurrentID)
	}

	return ids
}

func (s *searchState) Path() []barterVertex {
	if s == nil {
		return nil
	}

	path := make([]barterVertex, 0, s.Depth)
	for cur := s; cur != nil; cur = cur.Parent {
		if cur.Depth == 0 {
			continue
		}
		path = append(path, cur.Step)
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}
