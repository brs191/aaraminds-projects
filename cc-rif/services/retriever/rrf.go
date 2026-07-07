package retriever

// RRFScore computes the reciprocal-rank fusion score for the given ranks.
// Ranks are 1-based. A zero or negative rank contributes nothing.
func RRFScore(k int, ranks ...int) float64 {
	if k <= 0 {
		k = defaultRRFK
	}
	var score float64
	for _, rank := range ranks {
		if rank <= 0 {
			continue
		}
		score += 1.0 / float64(k+rank)
	}
	return score
}

func tierWeight(tier string) float64 {
	switch tier {
	case "static":
		return 1.0
	case "inferred-di":
		return 0.9
	case "cross-service":
		return 0.8
	case "inferred-aop":
		return 0.85
	default:
		return 0.75
	}
}
