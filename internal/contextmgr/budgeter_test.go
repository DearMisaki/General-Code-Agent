package contextmgr

import (
	"context"
	"strings"
	"testing"

	"mewcode/internal/conversation"
	"mewcode/internal/toolresult"
)

func TestBudgeterAppliesToolResultReplacement(t *testing.T) {
	conv := conversation.NewManager()
	large := strings.Repeat("x", toolresult.SingleResultLimit+1)
	conv.AddAssistantFull("", nil, []conversation.ToolUseBlock{{
		ToolUseID: "tool-1",
		ToolName:  "Bash",
		Arguments: map[string]any{"command": "printf"},
	}})
	conv.AddToolResultsMessage([]conversation.ToolResultBlock{{
		ToolUseID: "tool-1",
		Content:   large,
	}})

	state := toolresult.New()
	result, err := NewBudgeter().PrepareBudget(context.Background(), BudgetRequest{
		Conversation:     conv,
		WorkDir:          t.TempDir(),
		ContextWindow:    200000,
		MaxOutputTokens:  8192,
		ReplacementState: state,
	})
	if err != nil {
		t.Fatalf("budget: %v", err)
	}
	msgs := result.APIConversation.GetMessages()
	got := msgs[1].ToolResults[0].Content
	if !strings.HasPrefix(got, "[Result of ") {
		t.Fatalf("tool result was not replaced: %q", got[:40])
	}
	if len(result.ToolResultRecords) != 1 {
		t.Fatalf("records = %d", len(result.ToolResultRecords))
	}
}
