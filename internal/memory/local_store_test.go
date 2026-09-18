package memory

import (
	"context"
	"testing"
)

func TestLocalMemoryStoreAddAndSearch(t *testing.T) {
	store := NewLocalMemoryStore()
	added, err := store.Add(context.Background(), AddMemoryRequest{
		Candidates: []MemoryCandidate{{
			Scope:      ScopeProject,
			Kind:       KindProjectFact,
			Memory:     "ContextGateway injects long-term memory before budgeting.",
			Confidence: 0.9,
			ProjectID:  "proj",
		}},
	})
	if err != nil || len(added) != 1 {
		t.Fatalf("Add() = (%v, %v), want one memory", added, err)
	}
	results, err := store.Search(context.Background(), SearchMemoryRequest{
		Query:   "where is long-term memory injected",
		Filters: MemoryFilters{ProjectID: "proj"},
		TopK:    5,
	})
	if err != nil || len(results) != 1 {
		t.Fatalf("Search() = (%v, %v), want one result", results, err)
	}
	if results[0].Memory.ID != added[0].ID {
		t.Fatalf("Search returned ID %q, want %q", results[0].Memory.ID, added[0].ID)
	}
}
