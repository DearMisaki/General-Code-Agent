package memory

import (
	"context"
	"testing"
)

func TestDeduperSkipsExactDuplicate(t *testing.T) {
	store := NewLocalMemoryStore()
	_, _ = store.Add(context.Background(), AddMemoryRequest{
		Candidates: []MemoryCandidate{{
			Scope:      ScopeProject,
			Kind:       KindProjectFact,
			Memory:     "ContextGateway prepares context before each model turn.",
			Confidence: 0.9,
			ProjectID:  "proj",
		}},
	})
	d := NewDeduper(store, DedupeOptions{DuplicateThreshold: 0.95})
	decision, err := d.Decide(context.Background(), MemoryCandidate{
		Scope:      ScopeProject,
		Kind:       KindProjectFact,
		Memory:     "ContextGateway prepares context before each model turn.",
		Confidence: 0.9,
		ProjectID:  "proj",
	})
	if err != nil {
		t.Fatalf("Decide() = %v", err)
	}
	if decision.Action != DedupeSkip {
		t.Fatalf("action = %q, want %q", decision.Action, DedupeSkip)
	}
}
