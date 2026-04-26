package utils

import "strings"

func ClosestMatch(target string, candidates []string, maxDistance int) (string, bool) {
	target = strings.TrimSpace(strings.ToLower(target))
	if target == "" || len(candidates) == 0 {
		return "", false
	}

	bestCandidate := ""
	bestDistance := maxDistance + 1

	for _, candidate := range candidates {
		normalized := strings.TrimSpace(strings.ToLower(candidate))
		if normalized == "" {
			continue
		}

		distance := levenshtein(target, normalized)
		if distance < bestDistance {
			bestDistance = distance
			bestCandidate = candidate
		}
	}

	if bestCandidate == "" || bestDistance > maxDistance {
		return "", false
	}

	return bestCandidate, true
}

func levenshtein(left, right string) int {
	if left == right {
		return 0
	}
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}

	previous := make([]int, len(right)+1)
	current := make([]int, len(right)+1)
	for i := range previous {
		previous[i] = i
	}

	for i := 1; i <= len(left); i++ {
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 0
			if left[i-1] != right[j-1] {
				cost = 1
			}

			deletion := previous[j] + 1
			insertion := current[j-1] + 1
			substitution := previous[j-1] + cost

			current[j] = min(deletion, insertion, substitution)
		}
		copy(previous, current)
	}

	return previous[len(right)]
}

func min(values ...int) int {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}
