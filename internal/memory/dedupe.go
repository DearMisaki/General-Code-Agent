package memory

import (
	"context"
	"strings"
)

type DedupeOptions struct {
	DuplicateThreshold float64
}

type DedupeAction string

const (
	DedupeAdd            DedupeAction = "add"
	DedupeSkip           DedupeAction = "skip"
	DedupeUpdateMetadata DedupeAction = "update_metadata"
	DedupeConflict       DedupeAction = "conflict"
)

type DedupeDecision struct {
	Action DedupeAction
	Match  *StoredMemory
	Reason string
	Score  float64
	Memory MemoryCandidate
}

type Deduper struct {
	store MemoryStore
	opts  DedupeOptions
}

func NewDeduper(store MemoryStore, opts DedupeOptions) *Deduper {
	if opts.DuplicateThreshold == 0 {
		opts.DuplicateThreshold = 0.95
	}
	return &Deduper{store: store, opts: opts}
}

func (d *Deduper) Decide(ctx context.Context, c MemoryCandidate) (DedupeDecision, error) {
	if d == nil || d.store == nil {
		return DedupeDecision{Action: DedupeAdd, Memory: c}, nil
	}
	results, err := d.store.Search(ctx, SearchMemoryRequest{
		Query:   c.Memory,
		Filters: filtersFromCandidate(c),
		TopK:    3,
	})
	if err != nil {
		return DedupeDecision{}, err
	}
	norm := normalizeMemoryText(c.Memory)
	for _, result := range results {
		match := result.Memory
		if normalizeMemoryText(match.Memory) == norm {
			return DedupeDecision{Action: DedupeSkip, Match: &match, Score: 1, Memory: c, Reason: "exact duplicate"}, nil
		}
		if result.Score >= d.opts.DuplicateThreshold {
			if looksContradictory(c.Memory) {
				return DedupeDecision{Action: DedupeConflict, Match: &match, Score: result.Score, Memory: c, Reason: "possible contradiction"}, nil
			}
			if match.Kind == c.Kind {
				return DedupeDecision{Action: DedupeUpdateMetadata, Match: &match, Score: result.Score, Memory: c, Reason: "similar memory"}, nil
			}
		}
	}
	return DedupeDecision{Action: DedupeAdd, Memory: c}, nil
}

func filtersFromCandidate(c MemoryCandidate) MemoryFilters {
	return MemoryFilters{
		UserID:    c.UserID,
		AppID:     c.AppID,
		ProjectID: c.ProjectID,
		AgentID:   c.AgentID,
		SessionID: c.SessionID,
		RunID:     c.RunID,
		TeamName:  c.TeamName,
		TaskID:    c.TaskID,
		Scope:     c.Scope,
	}
}

func normalizeMemoryText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

func looksContradictory(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range []string{"no longer", "instead", "changed to", "replaces", "supersedes"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
