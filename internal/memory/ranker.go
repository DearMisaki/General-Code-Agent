package memory

import (
	"sort"
	"time"
)

type RankerOptions struct {
	KindWeights map[MemoryKind]float64
}

func DefaultRankerOptions() RankerOptions {
	return RankerOptions{KindWeights: map[MemoryKind]float64{
		KindConstraint:           1.00,
		KindArchitectureDecision: 0.95,
		KindTaskSummary:          0.90,
		KindToolGotcha:           0.80,
		KindProjectFact:          0.75,
		KindPreference:           0.70,
		KindAgentExperience:      0.60,
		KindTeamState:            0.60,
		KindRelationship:         0.50,
		KindEvent:                0.30,
	}}
}

type MemoryRanker struct {
	opts RankerOptions
}

func NewMemoryRanker(opts RankerOptions) *MemoryRanker {
	if len(opts.KindWeights) == 0 {
		opts = DefaultRankerOptions()
	}
	return &MemoryRanker{opts: opts}
}

func (r *MemoryRanker) Rank(results []MemorySearchResult, req RecallRequest) []MemoryBlock {
	blocks := make([]MemoryBlock, 0, len(results))
	for _, result := range results {
		mem := result.Memory
		weight := r.opts.KindWeights[mem.Kind]
		score := result.Score + weight
		if scopeMatchesRequest(mem, req) {
			score += 0.2
		}
		if mem.Confidence > 0 {
			score += mem.Confidence * 0.1
		}
		blocks = append(blocks, MemoryBlock{
			ID:        mem.ID,
			Scope:     mem.Scope,
			Kind:      mem.Kind,
			Text:      mem.Memory,
			Score:     score,
			Reason:    result.Reason,
			Source:    "memory-store",
			CreatedAt: mem.CreatedAt,
			UpdatedAt: mem.UpdatedAt,
			Freshness: MemoryFreshnessText(mem.UpdatedAt.UnixMilli()),
			TokenCost: estimateMemoryTokens(mem.Memory),
			Metadata:  cloneStringMap(mem.Metadata),
		})
	}
	sort.SliceStable(blocks, func(i, j int) bool {
		if blocks[i].Score == blocks[j].Score {
			return blocks[i].UpdatedAt.After(blocks[j].UpdatedAt)
		}
		return blocks[i].Score > blocks[j].Score
	})
	return blocks
}

func scopeMatchesRequest(mem StoredMemory, req RecallRequest) bool {
	return (req.UserID != "" && mem.UserID == req.UserID) ||
		(req.ProjectID != "" && mem.ProjectID == req.ProjectID) ||
		(req.AgentID != "" && mem.AgentID == req.AgentID) ||
		(req.SessionID != "" && mem.SessionID == req.SessionID) ||
		(req.TeamName != "" && mem.TeamName == req.TeamName) ||
		(req.TaskID != "" && mem.TaskID == req.TaskID)
}

func estimateMemoryTokens(text string) int {
	if text == "" {
		return 0
	}
	tokens := len([]rune(text)) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

var _ = time.Time{}
