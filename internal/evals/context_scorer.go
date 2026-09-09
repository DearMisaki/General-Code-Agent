package evals

import (
	"math"
	"sort"
)

type ContextEvalCase struct {
	CaseID              string             `json:"case_id"`
	RequiredContextIDs  []string           `json:"required_context_ids"`
	ForbiddenContextIDs []string           `json:"forbidden_context_ids"`
	RequiredConstraints []string           `json:"required_constraints"`
	Gains               map[string]float64 `json:"gains,omitempty"`
	K                   int                `json:"k,omitempty"`
}

type RankedContext struct {
	ID        string `json:"id"`
	Rank      int    `json:"rank"`
	TokenCost int    `json:"token_cost,omitempty"`
}

type ContextEvalScore struct {
	CaseID            string  `json:"case_id"`
	Precision         float64 `json:"precision"`
	Recall            float64 `json:"recall"`
	F1                float64 `json:"f1"`
	PrecisionAtK      float64 `json:"precision_at_k"`
	RecallAtK         float64 `json:"recall_at_k"`
	MRR               float64 `json:"mrr"`
	NDCGAtK           float64 `json:"ndcg_at_k"`
	ForbiddenSelected int     `json:"forbidden_selected"`
}

func ScoreContextEval(tc ContextEvalCase, selected []RankedContext) ContextEvalScore {
	score := ContextEvalScore{CaseID: tc.CaseID}
	required := stringSet(tc.RequiredContextIDs)
	forbidden := stringSet(tc.ForbiddenContextIDs)

	relevantCount := 0
	seenRelevant := make(map[string]struct{})
	for _, context := range selected {
		if _, ok := required[context.ID]; ok {
			relevantCount++
			seenRelevant[context.ID] = struct{}{}
		}
		if _, ok := forbidden[context.ID]; ok {
			score.ForbiddenSelected++
		}
	}

	score.Precision = ratio(relevantCount, len(selected))
	score.Recall = ratio(len(seenRelevant), len(required))
	score.F1 = f1(score.Precision, score.Recall)

	k := tc.K
	if k <= 0 || k > len(selected) {
		k = len(selected)
	}
	topK := selected[:k]
	topKRelevant := 0
	topKSeenRelevant := make(map[string]struct{})
	for _, context := range topK {
		if _, ok := required[context.ID]; ok {
			topKRelevant++
			topKSeenRelevant[context.ID] = struct{}{}
		}
	}
	score.PrecisionAtK = ratio(topKRelevant, len(topK))
	score.RecallAtK = ratio(len(topKSeenRelevant), len(required))

	for position, context := range selected {
		if _, ok := required[context.ID]; ok {
			rank := context.Rank
			if rank <= 0 {
				rank = position + 1
			}
			score.MRR = 1.0 / float64(rank)
			break
		}
	}
	score.NDCGAtK = ndcgAtK(tc, topK, required)
	return score
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func f1(precision, recall float64) float64 {
	if precision+recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}

func ndcgAtK(tc ContextEvalCase, selected []RankedContext, required map[string]struct{}) float64 {
	gain := func(id string) float64 {
		if tc.Gains != nil {
			return tc.Gains[id]
		}
		if _, ok := required[id]; ok {
			return 1
		}
		return 0
	}

	dcg := 0.0
	for position, context := range selected {
		dcg += gain(context.ID) / math.Log2(float64(position+2))
	}

	idealGains := make([]float64, 0)
	if tc.Gains != nil {
		for _, value := range tc.Gains {
			if value > 0 {
				idealGains = append(idealGains, value)
			}
		}
	} else {
		for range required {
			idealGains = append(idealGains, 1)
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(idealGains)))
	if len(idealGains) > len(selected) {
		idealGains = idealGains[:len(selected)]
	}
	idcg := 0.0
	for position, value := range idealGains {
		idcg += value / math.Log2(float64(position+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}
