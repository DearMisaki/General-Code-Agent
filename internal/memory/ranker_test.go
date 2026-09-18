package memory

import (
	"testing"
	"time"
)

func TestMemoryRankerPrioritizesConstraints(t *testing.T) {
	now := time.Now()
	results := []MemorySearchResult{
		{Memory: StoredMemory{ID: "pref", Kind: KindPreference, Memory: "User likes concise answers.", UpdatedAt: now}, Score: 0.9},
		{Memory: StoredMemory{ID: "constraint", Kind: KindConstraint, Memory: "Do not read internal/tui.", UpdatedAt: now}, Score: 0.8},
	}
	ranked := NewMemoryRanker(DefaultRankerOptions()).Rank(results, RecallRequest{ProjectID: "proj"})
	if ranked[0].ID != "constraint" {
		t.Fatalf("first memory = %q, want constraint", ranked[0].ID)
	}
}
