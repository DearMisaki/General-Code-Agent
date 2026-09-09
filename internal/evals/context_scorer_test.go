package evals

import (
	"math"
	"testing"
)

func TestScoreContextEvalComputesPrecisionRecallF1(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		CaseID:              "ctx-001",
		RequiredContextIDs:  []string{"a", "b", "c"},
		ForbiddenContextIDs: []string{"z"},
	}, []RankedContext{
		{ID: "a", Rank: 1},
		{ID: "x", Rank: 2},
		{ID: "b", Rank: 3},
	})
	if score.Precision != 2.0/3.0 || score.Recall != 2.0/3.0 || score.F1 != 2.0/3.0 {
		t.Fatalf("score = %+v", score)
	}
	if score.ForbiddenSelected != 0 {
		t.Fatalf("forbidden selected = %d", score.ForbiddenSelected)
	}
}

func TestScoreContextEvalComputesMRRAndNDCG(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		CaseID:             "ctx-002",
		RequiredContextIDs: []string{"gold"},
		Gains:              map[string]float64{"gold": 3, "silver": 1},
	}, []RankedContext{
		{ID: "noise", Rank: 1},
		{ID: "gold", Rank: 2},
	})
	if score.MRR != 0.5 {
		t.Fatalf("MRR = %f, want 0.5", score.MRR)
	}
	if score.NDCGAtK <= 0 || score.NDCGAtK > 1 {
		t.Fatalf("NDCGAtK = %f", score.NDCGAtK)
	}
}

func TestScoreContextEvalCountsDuplicateSelectionsCorrectly(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"a", "b"},
	}, []RankedContext{
		{ID: "a", Rank: 1},
		{ID: "a", Rank: 2},
	})

	if score.Precision != 0.5 {
		t.Fatalf("Precision = %f, want 0.5", score.Precision)
	}
	if score.Recall != 0.5 {
		t.Fatalf("Recall = %f, want 0.5", score.Recall)
	}
}

func TestScoreContextEvalUsesRankForUnsortedSelections(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"gold", "silver"},
		Gains:              map[string]float64{"gold": 3, "silver": 1},
		K:                  2,
	}, []RankedContext{
		{ID: "gold", Rank: 3},
		{ID: "noise", Rank: 1},
		{ID: "silver", Rank: 2},
	})

	if score.PrecisionAtK != 0.5 || score.RecallAtK != 0.5 {
		t.Fatalf("at K score = %+v, want precision and recall 0.5", score)
	}
	if score.MRR != 0.5 {
		t.Fatalf("MRR = %f, want 0.5", score.MRR)
	}
}

func TestScoreContextEvalDeduplicatesNDCGSelections(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"gold"},
		Gains:              map[string]float64{"gold": 3},
	}, []RankedContext{
		{ID: "gold", Rank: 1},
		{ID: "gold", Rank: 2},
	})

	if score.NDCGAtK != 1 {
		t.Fatalf("NDCGAtK = %f, want 1 after deduplicating selections", score.NDCGAtK)
	}
}

func TestScoreContextEvalLimitsAtKAndDefaultsKToSelectedLength(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"a", "b"},
		K:                  1,
	}, []RankedContext{
		{ID: "a", Rank: 1},
		{ID: "b", Rank: 2},
	})

	if score.PrecisionAtK != 1 || score.RecallAtK != 0.5 {
		t.Fatalf("at K score = %+v", score)
	}

	defaultK := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"a", "b"},
	}, []RankedContext{
		{ID: "a", Rank: 1},
		{ID: "b", Rank: 2},
	})
	if defaultK.PrecisionAtK != 1 || defaultK.RecallAtK != 1 {
		t.Fatalf("default K score = %+v", defaultK)
	}
}

func TestScoreContextEvalPreservesExplicitKWhenFewerContextsAreSelected(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"a", "b"},
		K:                  5,
	}, []RankedContext{
		{ID: "a", Rank: 1},
		{ID: "b", Rank: 2},
	})

	if score.PrecisionAtK != 0.4 || score.RecallAtK != 1 {
		t.Fatalf("at K score = %+v, want precision 0.4 and recall 1", score)
	}
}

func TestScoreContextEvalDuplicateNDCGConsumesRankPosition(t *testing.T) {
	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"gold", "silver"},
		Gains:              map[string]float64{"gold": 3, "silver": 2},
		K:                  3,
	}, []RankedContext{
		{ID: "gold", Rank: 1},
		{ID: "gold", Rank: 2},
		{ID: "silver", Rank: 3},
	})

	want := (3.0 + 2.0/math.Log2(4)) / (3.0 + 2.0/math.Log2(3))
	if math.Abs(score.NDCGAtK-want) > 1e-12 {
		t.Fatalf("NDCGAtK = %.12f, want %.12f", score.NDCGAtK, want)
	}
}

func TestScoreContextEvalHandlesEmptyInputsAndDefaultGains(t *testing.T) {
	empty := ScoreContextEval(ContextEvalCase{CaseID: "empty"}, nil)
	if empty.CaseID != "empty" || empty.Precision != 0 || empty.Recall != 0 || empty.F1 != 0 || empty.MRR != 0 || empty.NDCGAtK != 0 {
		t.Fatalf("empty score = %+v", empty)
	}

	score := ScoreContextEval(ContextEvalCase{
		RequiredContextIDs: []string{"gold"},
	}, []RankedContext{{ID: "gold", Rank: 1}})
	if score.NDCGAtK != 1 {
		t.Fatalf("default-gain NDCGAtK = %f, want 1", score.NDCGAtK)
	}
}
