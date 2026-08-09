package cb

import "strings"

// Fuzzy search evaluation of the Item's names
//
// Uses Levenshtein Distance method of calculating the score (0-1)
func fuzzySimilarity(a, b string) float64 {
	a = normalizeText(a)
	b = normalizeText(b)
	if a == b {
		return 1
	}
	if a == "" || b == "" {
		return 0
	}

	maxLen := len([]rune(a))
	if l := len([]rune(b)); l > maxLen {
		maxLen = l
	}
	if maxLen == 0 {
		return 1
	}

	distance := levenshteinDistance([]rune(a), []rune(b))
	score := 1 - float64(distance)/float64(maxLen)
	if score < 0 {
		return 0
	}
	return score
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	return s
}

func levenshteinDistance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			deletion := prev[j] + 1
			insertion := cur[j-1] + 1
			substitution := prev[j-1] + cost
			cur[j] = minInt(deletion, minInt(insertion, substitution))
		}
		copy(prev, cur)
	}

	return prev[len(b)]
}
