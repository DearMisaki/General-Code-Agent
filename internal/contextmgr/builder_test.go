package contextmgr

import (
	"testing"

	"mewcode/internal/conversation"
)

func TestBuilderBuildsSnapshotWithoutMutatingConversation(t *testing.T) {
	conv := conversation.NewManager()
	conv.AddUserMessage("hello")

	builder := NewBuilder()
	snap := builder.Build(PrepareRequest{
		Conversation:      conv,
		WorkDir:           "/repo",
		SessionID:         "session-1",
		Protocol:          "anthropic",
		AgentID:           "agent-1",
		AgentType:         "main",
		Iteration:         2,
		ContextWindow:     200000,
		MaxOutputTokens:   8192,
		Instructions:      "follow repo rules",
		MemoryContent:     "prefers Chinese",
		ActiveSkills:      map[string]string{"review": "read carefully"},
		ToolSchemas:       []map[string]any{{"name": "ReadFile"}},
		DeferredToolNames: []string{"mcp__docs__search"},
	})

	if conv.Len() != 1 {
		t.Fatalf("builder mutated conversation, len=%d", conv.Len())
	}
	if snap.Session.SessionID != "session-1" {
		t.Fatalf("session id = %q", snap.Session.SessionID)
	}
	if snap.Session.MessageCount != 1 {
		t.Fatalf("message count = %d", snap.Session.MessageCount)
	}
	if snap.User.Instructions != "follow repo rules" {
		t.Fatalf("instructions not copied")
	}
	if snap.Business.ActiveSkills["review"] != "read carefully" {
		t.Fatalf("active skill not copied")
	}
	if snap.Business.ToolSchemas[0]["name"] != "ReadFile" {
		t.Fatalf("tool schema not copied")
	}
	if snap.Business.DeferredToolNames[0] != "mcp__docs__search" {
		t.Fatalf("deferred tool name not copied")
	}
	if snap.Execution.WorkDir != "/repo" || snap.Execution.Iteration != 2 {
		t.Fatalf("execution context not populated")
	}
	if len(snap.Sources) == 0 {
		t.Fatalf("expected sources")
	}
}
