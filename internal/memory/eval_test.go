package memory

import "testing"

func TestScoreMemoryEvalCaseComputesRecallPrecisionF1(t *testing.T) {
	result := ScoreMemoryEvalCase(MemoryEvalCase{
		RequiredMemoryIDs:  []string{"a", "b"},
		ForbiddenMemoryIDs: []string{"x"},
	}, []MemoryBlock{
		{ID: "a"},
		{ID: "x"},
	})
	if result.Recall != 0.5 {
		t.Fatalf("recall = %v, want 0.5", result.Recall)
	}
	if result.Precision != 0.5 {
		t.Fatalf("precision = %v, want 0.5", result.Precision)
	}
	if result.ForbiddenHits != 1 {
		t.Fatalf("forbidden hits = %d, want 1", result.ForbiddenHits)
	}
}
