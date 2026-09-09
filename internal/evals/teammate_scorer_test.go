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

func TestScoreTeammateEvalRequiresExpectedEmptyStringArgToExist(t *testing.T) {
	score := ScoreTeammateEval(TeammateEvalCase{
		ExpectedToolCalls: []ObservedToolCall{{
			Name: "SendMessage",
			Args: map[string]string{"to": ""},
		}},
	}, TeammateEvalObservation{
		ToolCalls: []ObservedToolCall{{Name: "SendMessage"}},
	})

	if score.ToolRecall != 0 || score.ToolPrecision != 0 {
		t.Fatalf("tool score = %+v, want no match for missing expected arg", score)
	}
}

func TestScoreTeammateEvalTreatsEmptyExpectedMessageFieldsAsWildcards(t *testing.T) {
	score := ScoreTeammateEval(TeammateEvalCase{
		ExpectedMessages: []ObservedMessage{{From: "lead"}},
	}, TeammateEvalObservation{
		Messages: []ObservedMessage{{From: "lead", To: "worker", TaskID: "task-1"}},
	})

	if score.MessageDeliveryAccuracy != 1 {
		t.Fatalf("delivery accuracy = %f, want 1", score.MessageDeliveryAccuracy)
	}
}

func TestScoreTeammateEvalComputesCollaborationCompletionFromTransitions(t *testing.T) {
	score := ScoreTeammateEval(TeammateEvalCase{
		ExpectedTaskTransitions: []ObservedTaskTransition{
			{TaskID: "task-1", FromStatus: "pending", ToStatus: "running"},
			{TaskID: "task-1", FromStatus: "running", ToStatus: "done"},
		},
	}, TeammateEvalObservation{
		TaskTransitions: []ObservedTaskTransition{{
			TaskID: "task-1", FromStatus: "pending", ToStatus: "running",
		}},
	})

	if score.CollaborationCompletion != 0.5 {
		t.Fatalf("completion = %f, want 0.5", score.CollaborationCompletion)
	}
}

func TestScoreTeammateEvalDetectsMessageLoopOnlyAfterThirdIdenticalMessage(t *testing.T) {
	messages := []ObservedMessage{
		{From: "lead", To: "worker", Text: "retry"},
		{From: "lead", To: "worker", Text: "retry"},
	}
	if score := ScoreTeammateEval(TeammateEvalCase{}, TeammateEvalObservation{Messages: messages}); score.LoopDetected {
		t.Fatal("loop detected after only two identical messages")
	}

	messages = append(messages, ObservedMessage{From: "lead", To: "worker", Text: "retry"})
	if score := ScoreTeammateEval(TeammateEvalCase{}, TeammateEvalObservation{Messages: messages}); !score.LoopDetected {
		t.Fatal("loop not detected after three identical messages")
	}
}

func TestScoreTeammateEvalMatchesDuplicateToolsOneToOne(t *testing.T) {
	score := ScoreTeammateEval(TeammateEvalCase{
		ExpectedToolCalls: []ObservedToolCall{
			{Name: "Bash"},
			{Name: "Bash"},
		},
	}, TeammateEvalObservation{
		ToolCalls: []ObservedToolCall{{Name: "Bash"}},
	})

	if score.ToolPrecision != 1 || score.ToolRecall != 0.5 {
		t.Fatalf("tool score = %+v, want one-to-one duplicate matching", score)
	}
}
