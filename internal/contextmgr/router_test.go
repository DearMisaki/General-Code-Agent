package contextmgr

import (
	"strings"
	"testing"

	"mewcode/internal/conversation"
)

func TestRouterForkPatchesIncompleteToolUse(t *testing.T) {
	parent := conversation.NewManager()
	parent.AddUserMessage("read")
	parent.AddAssistantFull("calling", nil, []conversation.ToolUseBlock{{
		ToolUseID: "tool-1",
		ToolName:  "ReadFile",
		Arguments: map[string]any{"file_path": "a.go"},
	}})

	router := NewRouter()
	forked := router.BuildForkedConversation(parent, "continue task", "fork boilerplate")
	msgs := forked.GetMessages()
	if len(msgs) != 4 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if len(msgs[2].ToolResults) != 1 {
		t.Fatalf("missing placeholder tool result")
	}
	if got := msgs[2].ToolResults[0].Content; got != "(tool execution interrupted by fork)" {
		t.Fatalf("placeholder = %q", got)
	}
	if !strings.Contains(msgs[3].Content, "continue task") {
		t.Fatalf("missing task in final message")
	}
}

func TestRouterBuildsRecentHandoff(t *testing.T) {
	parent := conversation.NewManager()
	for i := 0; i < 8; i++ {
		parent.AddUserMessage("u")
	}
	router := NewRouter()
	pkg := router.BuildHandoff(HandoffRequest{
		FromAgent:      AgentRef{ID: "main"},
		ToAgent:        AgentRef{ID: "worker"},
		Mode:           HandoffRecent,
		Conversation:   parent,
		RecentMessages: 3,
	})
	if pkg.Mode != HandoffRecent {
		t.Fatalf("mode = %q", pkg.Mode)
	}
	if len(pkg.Messages) != 3 {
		t.Fatalf("messages = %d", len(pkg.Messages))
	}
}

func TestBuildForkedConversationPreservesThinkingBlocks(t *testing.T) {
	parent := conversation.NewManager()
	parent.AddAssistantFull("thinking result", []conversation.ThinkingBlock{{
		Thinking:  "private summary",
		Signature: "sig-1",
	}}, nil)

	forked := NewRouter().BuildForkedConversation(parent, "task", "boilerplate")
	msgs := forked.GetMessages()
	if len(msgs[0].ThinkingBlocks) != 1 {
		t.Fatalf("thinking blocks not preserved")
	}
	if msgs[0].ThinkingBlocks[0].Signature != "sig-1" {
		t.Fatalf("signature = %q", msgs[0].ThinkingBlocks[0].Signature)
	}
}

func TestHandoffCopiesSkillsAndTools(t *testing.T) {
	conv := conversation.NewManager()
	conv.AddUserMessage("hello")
	pkg := NewRouter().BuildHandoff(HandoffRequest{
		FromAgent:      AgentRef{ID: "main", Type: "main"},
		ToAgent:        AgentRef{ID: "worker", Type: "general-purpose"},
		Mode:           HandoffRecent,
		Conversation:   conv,
		RecentMessages: 1,
		ActiveSkills:   map[string]string{"review": "body"},
		ToolSchemas:    []map[string]any{{"name": "ReadFile"}},
	})
	if pkg.ActiveSkills["review"] != "body" {
		t.Fatalf("missing active skill")
	}
	if pkg.ToolSchemas[0]["name"] != "ReadFile" {
		t.Fatalf("missing tool schema")
	}
}
