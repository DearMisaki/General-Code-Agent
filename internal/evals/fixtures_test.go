package evals

import (
	"slices"
	"strings"
	"testing"
)

func TestSyntheticConversationCreatesRequestedMessages(t *testing.T) {
	conv := SyntheticConversation(10, 0)
	if got := len(conv.GetMessages()); got != 10 {
		t.Fatalf("messages = %d, want 10", got)
	}
}

func TestSyntheticConversationCreatesSingleActualToolResult(t *testing.T) {
	conv := SyntheticConversation(100, 1024)
	msgs := conv.GetMessages()

	toolUses := make(map[string]int)
	var results int
	for _, msg := range msgs {
		if strings.Contains(msg.Content, strings.Repeat("x", 1024)) {
			t.Fatal("tool result payload stored in ordinary message content")
		}
		for _, use := range msg.ToolUses {
			toolUses[use.ToolUseID]++
		}
		for _, result := range msg.ToolResults {
			results++
			if result.Content != strings.Repeat("x", 1024) {
				t.Fatalf("tool result size = %d, want 1024", len(result.Content))
			}
			if toolUses[result.ToolUseID] != 1 {
				t.Fatalf("tool result %q has %d matching tool uses, want 1", result.ToolUseID, toolUses[result.ToolUseID])
			}
		}
	}
	if results != 1 {
		t.Fatalf("tool results = %d, want 1", results)
	}
}

func TestSyntheticToolSchemasCreatesStableNames(t *testing.T) {
	schemas := SyntheticToolSchemas(3)
	names := []string{schemas[0]["name"].(string), schemas[1]["name"].(string), schemas[2]["name"].(string)}
	want := []string{"SyntheticTool0000", "SyntheticTool0001", "SyntheticTool0002"}
	if !slices.Equal(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}
