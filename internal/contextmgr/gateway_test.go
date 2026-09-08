package contextmgr

import (
	"context"
	"strings"
	"testing"

	"mewcode/internal/conversation"
	"mewcode/internal/toolresult"
)

func TestGatewayPrepareTurnRendersAndBudgets(t *testing.T) {
	conv := conversation.NewManager()
	conv.AddUserMessage("start")

	gateway := NewGateway(GatewayOptions{
		Builder:  NewBuilder(),
		Renderer: NewRenderer(),
		Budgeter: NewBudgeter(),
		Audit:    NewAuditWriter(t.TempDir()),
	})
	prepared, err := gateway.PrepareTurn(context.Background(), PrepareRequest{
		Conversation:     conv,
		WorkDir:          t.TempDir(),
		SessionID:        "session-1",
		Protocol:         "anthropic",
		AgentID:          "agent-1",
		Iteration:        1,
		ContextWindow:    200000,
		MaxOutputTokens:  8192,
		Instructions:     "repo rules",
		MemoryContent:    "memory body",
		ReplacementState: toolresult.New(),
	})
	if err != nil {
		t.Fatalf("prepare turn: %v", err)
	}
	if prepared.APIConversation == nil {
		t.Fatalf("missing api conversation")
	}
	msgs := prepared.APIConversation.GetMessages()
	joined := ""
	for _, m := range msgs {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "repo rules") || !strings.Contains(joined, "memory body") {
		t.Fatalf("missing rendered context:\n%s", joined)
	}
	if len(prepared.AuditRecords) == 0 {
		t.Fatalf("expected audit records")
	}
}
