package evals

import (
	"math"
	"sort"
)

type ContextEvalCase struct {
	CaseID                 string             `json:"case_id"`
	RequiredContextIDs     []string           `json:"required_context_ids"`
	ForbiddenContextIDs    []string           `json:"forbidden_context_ids"`
	RequiredConstraints    []string           `json:"required_constraints"`
	Gains                  map[string]float64 `json:"gains,omitempty"`
	K                      int                `json:"k,omitempty"`
	Query                  string             `json:"query,omitempty"`
	Conversation           []EvalMessage      `json:"conversation,omitempty"`
	MemorySeeds            []EvalMemorySeed   `json:"memory_seeds,omitempty"`
	ActivePlan             []string           `json:"active_plan,omitempty"`
	RecentTools            []string           `json:"recent_tools,omitempty"`
	ExpectedAnswerContains []string           `json:"expected_answer_contains,omitempty"`
	AgentID                string             `json:"agent_id,omitempty"`
	ProjectID              string             `json:"project_id,omitempty"`
	TokenBudget            int                `json:"token_budget,omitempty"`
}

type EvalMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type EvalMemorySeed struct {
	ID        string            `json:"id"`
	Memory    string            `json:"memory"`
	Scope     string            `json:"scope,omitempty"`
	Kind      string            `json:"kind,omitempty"`
	UserID    string            `json:"user_id,omitempty"`
	AppID     string            `json:"app_id,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	AgentID   string            `json:"agent_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	RunID     string            `json:"run_id,omitempty"`
	TeamName  string            `json:"team_name,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
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
	ordered := rankSelections(selected)

	seenRelevant := make(map[string]struct{})
	for _, context := range ordered {
		if _, ok := required[context.ID]; ok {
			seenRelevant[context.ID] = struct{}{}
		}
		if _, ok := forbidden[context.ID]; ok {
			score.ForbiddenSelected++
		}
	}

	score.Precision = ratio(len(seenRelevant), len(ordered))
	score.Recall = ratio(len(seenRelevant), len(required))
	score.F1 = f1(score.Precision, score.Recall)

	k := tc.K
	if k <= 0 {
		k = len(ordered)
	}
	topKCount := min(k, len(ordered))
	topK := ordered[:topKCount]
	topKSeenRelevant := make(map[string]struct{})
	for _, context := range topK {
		if _, ok := required[context.ID]; ok {
			topKSeenRelevant[context.ID] = struct{}{}
		}
	}
	score.PrecisionAtK = ratio(len(topKSeenRelevant), k)
	score.RecallAtK = ratio(len(topKSeenRelevant), len(required))

	for _, context := range ordered {
		if _, ok := required[context.ID]; ok {
			score.MRR = 1.0 / float64(context.Rank)
			break
		}
	}
	score.NDCGAtK = ndcgAtK(tc, topK, required, k)
	return score
}

func rankSelections(selected []RankedContext) []RankedContext {
	ordered := append([]RankedContext(nil), selected...)
	maxRank := 0
	for _, context := range ordered {
		if context.Rank > maxRank {
			maxRank = context.Rank
		}
	}
	nextFallbackRank := maxRank + 1
	for index := range ordered {
		if ordered[index].Rank <= 0 {
			ordered[index].Rank = nextFallbackRank
			nextFallbackRank++
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Rank < ordered[j].Rank
	})
	return ordered
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

func ndcgAtK(tc ContextEvalCase, selected []RankedContext, required map[string]struct{}, k int) float64 {
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
	seenIDs := make(map[string]struct{}, len(selected))
	for position, context := range selected {
		if _, seen := seenIDs[context.ID]; seen {
			continue
		}
		seenIDs[context.ID] = struct{}{}
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
	if len(idealGains) > k {
		idealGains = idealGains[:k]
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
