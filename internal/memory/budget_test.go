package memory

import (
	"strings"
	"testing"
)

func TestMemoryBudgetLimiterKeepsWithinBudget(t *testing.T) {
	blocks := []MemoryBlock{
		{ID: "a", Kind: KindConstraint, Text: "Do not read internal/tui.", TokenCost: 5},
		{ID: "b", Kind: KindEvent, Text: strings.Repeat("old event ", 100), TokenCost: 100},
	}
	kept := NewMemoryBudgetLimiter(10).Limit(blocks)
	if len(kept) != 1 || kept[0].ID != "a" {
		t.Fatalf("kept = %+v, want only constraint", kept)
	}
}
