package evals

import "testing"

func TestScoreTeammateEvalComputesToolF1AndDeliveryAccuracy(t *testing.T) {
	score := ScoreTeammateEval(TeammateEvalCase{
		CaseID: "team-001",
		ExpectedToolCalls: []ObservedToolCall{
			{Name: "TeamCreate", Args: map[string]string{"team_name": "alpha"}},
			{Name: "SendMessage", Args: map[string]string{"to": "worker"}},
		},
		ExpectedMessages: []ObservedMessage{
			{From: "lead", To: "worker", TaskID: "task-1"},
		},
	}, TeammateEvalObservation{
		ToolCalls: []ObservedToolCall{
			{Name: "TeamCreate", Args: map[string]string{"team_name": "alpha"}},
			{Name: "SendMessage", Args: map[string]string{"to": "worker"}},
			{Name: "Bash"},
		},
		Messages: []ObservedMessage{
			{From: "lead", To: "worker", TaskID: "task-1"},
		},
	})
	if score.ToolPrecision != 2.0/3.0 || score.ToolRecall != 1.0 {
		t.Fatalf("tool score = %+v", score)
	}
	if score.MessageDeliveryAccuracy != 1.0 {
		t.Fatalf("delivery accuracy = %f", score.MessageDeliveryAccuracy)
	}
}
