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

func TestSyntheticToolResultSize(t *testing.T) {
	conv := SyntheticConversation(2, 1024)
	msgs := conv.GetMessages()
	if !strings.Contains(msgs[1].Content, strings.Repeat("x", 1024)) {
		t.Fatalf("tool result payload missing requested size")
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
