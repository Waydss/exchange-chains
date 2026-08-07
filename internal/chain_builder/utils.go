package cb

import (
	"fmt"
	"strings"
	"time"

	"github.com/L-ka080/DDB-6-case/internal/exchange"
)

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minUint8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

func chainSignature(chain exchange.Chain) string {
	parts := []string{fmt.Sprintf("%d:%d", chain.RootID, chain.RootWish.ID)}
	for _, barter := range chain.Barters {
		parts = append(parts, fmt.Sprintf("%d->%d:%d:%d", barter.From, barter.To, barter.Give.ID, barter.Wish.ID))
	}
	return strings.Join(parts, "|")
}

func allowedDepth(shadowRating int) int {
	switch {
	case shadowRating >= 10:
		return 1
	case shadowRating >= 6:
		return 2
	case shadowRating >= 3:
		return 3
	default:
		return max_barters_per_chain
	}
}

func candidateScore(fuzzy float64, publicRating float64, shadowRating int) float64 {
	return fuzzy*100 + publicRating*3 - float64(shadowRating)*7
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func wishMatchesGive(wish exchange.WishedItem, give exchange.GivenItem) bool {
	if wish.CategoryID != give.CategoryID {
		return false
	}

	return wish.PriceMin <= give.PriceMin && wish.PriceMax >= give.PriceMax

	// return give.Price >= wish.MinPrice && give.Price <= wish.MaxPrice
}
