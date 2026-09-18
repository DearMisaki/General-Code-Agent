package memory

type MemoryEvalCase struct {
	CaseID              string
	RequiredMemoryIDs   []string
	ForbiddenMemoryIDs  []string
	RequiredConstraints []string
}

type MemoryEvalResult struct {
	Recall        float64
	Precision     float64
	F1            float64
	ForbiddenHits int
	TokenCount    int
}

func ScoreMemoryEvalCase(eval MemoryEvalCase, selected []MemoryBlock) MemoryEvalResult {
	required := set(eval.RequiredMemoryIDs)
	forbidden := set(eval.ForbiddenMemoryIDs)
	var relevant int
	var forbiddenHits int
	var tokenCount int
	for _, block := range selected {
		if _, ok := required[block.ID]; ok {
			relevant++
		}
		if _, ok := forbidden[block.ID]; ok {
			forbiddenHits++
		}
		tokenCount += block.TokenCost
	}
	recall := ratio(relevant, len(required))
	precision := ratio(relevant, len(selected))
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return MemoryEvalResult{
		Recall:        recall,
		Precision:     precision,
		F1:            f1,
		ForbiddenHits: forbiddenHits,
		TokenCount:    tokenCount,
	}
}

func set(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}
