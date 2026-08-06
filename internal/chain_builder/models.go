package cb

import "github.com/L-ka080/DDB-6-case/internal/chain_builder/entities"

type candidate struct {
	AccountID    uint
	PublicRating float64
	ShadowRating int
	CurrentGive  entities.GivedItem
	Wish         entities.WishedItem
	FuzzyScore   float64
	MatchScore   float64
}

type searchState struct {
	Parent      *searchState
	CurrentID   uint
	CurrentGive entities.GivedItem
	Step        entities.Barter
	Depth       uint8
	DepthLimit  uint8
	Score       float64
}

func (s *searchState) HasInPath(accountID uint) bool {
	for cur := s; cur != nil; cur = cur.Parent {
		if cur.CurrentID == accountID {
			return true
		}
	}

	return false
}

func (s *searchState) PathAccountIDs() []uint {
	if s == nil {
		return nil
	}

	ids := make([]uint, 0, s.Depth)
	for cur := s; cur != nil; cur = cur.Parent {
		ids = append(ids, cur.CurrentID)
	}

	return ids
}

func (s *searchState) Path() []entities.Barter {
	if s == nil {
		return nil
	}

	path := make([]entities.Barter, 0, s.Depth)
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
