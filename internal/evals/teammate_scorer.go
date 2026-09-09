package evals

type TeammateEvalCase struct {
	CaseID                  string                   `json:"case_id"`
	Mode                    string                   `json:"mode"`
	ExpectedToolCalls       []ObservedToolCall       `json:"expected_tool_calls"`
	ExpectedMessages        []ObservedMessage        `json:"expected_messages"`
	ExpectedTaskTransitions []ObservedTaskTransition `json:"expected_task_transitions"`
	MaxAllowedMessages      int                      `json:"max_allowed_messages,omitempty"`
}

type ObservedToolCall struct {
	Name string            `json:"name"`
	Args map[string]string `json:"args,omitempty"`
}

type ObservedMessage struct {
	From   string `json:"from"`
	To     string `json:"to"`
	TaskID string `json:"task_id"`
	Text   string `json:"text"`
}

type ObservedTaskTransition struct {
	TaskID     string `json:"task_id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
}

type TeammateEvalObservation struct {
	ToolCalls       []ObservedToolCall       `json:"tool_calls"`
	Messages        []ObservedMessage        `json:"messages"`
	TaskTransitions []ObservedTaskTransition `json:"task_transitions"`
}

type TeammateEvalScore struct {
	CaseID                  string  `json:"case_id"`
	ToolPrecision           float64 `json:"tool_precision"`
	ToolRecall              float64 `json:"tool_recall"`
	ToolF1                  float64 `json:"tool_f1"`
	MessageDeliveryAccuracy float64 `json:"message_delivery_accuracy"`
	CollaborationCompletion float64 `json:"collaboration_completion"`
	LoopDetected            bool    `json:"loop_detected"`
}

func ScoreTeammateEval(tc TeammateEvalCase, obs TeammateEvalObservation) TeammateEvalScore {
	score := TeammateEvalScore{CaseID: tc.CaseID}

	matchedTools := matchTools(tc.ExpectedToolCalls, obs.ToolCalls)
	score.ToolPrecision = ratio(matchedTools, len(obs.ToolCalls))
	score.ToolRecall = ratio(matchedTools, len(tc.ExpectedToolCalls))
	score.ToolF1 = f1(score.ToolPrecision, score.ToolRecall)

	score.MessageDeliveryAccuracy = ratio(
		matchMessages(tc.ExpectedMessages, obs.Messages),
		len(tc.ExpectedMessages),
	)
	score.CollaborationCompletion = ratio(
		matchTransitions(tc.ExpectedTaskTransitions, obs.TaskTransitions),
		len(tc.ExpectedTaskTransitions),
	)
	score.LoopDetected = hasMessageLoop(obs.Messages)
	return score
}

func matchTools(expected, observed []ObservedToolCall) int {
	matched := 0
	used := make([]bool, len(observed))
	for _, want := range expected {
		for index, got := range observed {
			if used[index] || !toolMatches(want, got) {
				continue
			}
			used[index] = true
			matched++
			break
		}
	}
	return matched
}

func toolMatches(expected, observed ObservedToolCall) bool {
	if expected.Name != observed.Name {
		return false
	}
	for key, value := range expected.Args {
		got, ok := observed.Args[key]
		if !ok || got != value {
			return false
		}
	}
	return true
}

func matchMessages(expected, observed []ObservedMessage) int {
	matched := 0
	used := make([]bool, len(observed))
	for _, want := range expected {
		for index, got := range observed {
			if used[index] || !messageMatches(want, got) {
				continue
			}
			used[index] = true
			matched++
			break
		}
	}
	return matched
}

func messageMatches(expected, observed ObservedMessage) bool {
	return (expected.From == "" || expected.From == observed.From) &&
		(expected.To == "" || expected.To == observed.To) &&
		(expected.TaskID == "" || expected.TaskID == observed.TaskID)
}

func matchTransitions(expected, observed []ObservedTaskTransition) int {
	matched := 0
	used := make([]bool, len(observed))
	for _, want := range expected {
		for index, got := range observed {
			if used[index] || want != got {
				continue
			}
			used[index] = true
			matched++
			break
		}
	}
	return matched
}

func hasMessageLoop(messages []ObservedMessage) bool {
	counts := make(map[messageLoopKey]int)
	for _, message := range messages {
		key := messageLoopKey{from: message.From, to: message.To, text: message.Text}
		counts[key]++
		if counts[key] > 2 {
			return true
		}
	}
	return false
}

type messageLoopKey struct {
	from string
	to   string
	text string
}
