package evals

import (
	"fmt"
	"strings"

	"mewcode/internal/conversation"
	"mewcode/internal/orchestration"
)

func SyntheticConversation(size int, toolResultBytes int) *conversation.Manager {
	conv := conversation.NewManager()
	payload := strings.Repeat("x", max(0, toolResultBytes))
	for i := 0; i < size; i++ {
		if i%2 == 0 {
			conv.AddUserMessage(fmt.Sprintf("[ctx-%04d] user request %d", i, i))
			continue
		}
		content := fmt.Sprintf("[ctx-%04d] assistant response %d", i, i)
		if toolResultBytes > 0 {
			content += "\n" + payload
		}
		conv.AddAssistantMessage(content)
	}
	return conv
}

func SyntheticMemoryBlocks(count int) string {
	var b strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&b, "- [mem-%04d] durable fact %d\n", i, i)
	}
	return b.String()
}

func SyntheticNotifications(count int) []string {
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, fmt.Sprintf("<notification id=\"note-%04d\">ready</notification>", i))
	}
	return out
}

func SyntheticToolSchemas(count int) []map[string]any {
	out := make([]map[string]any, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, map[string]any{
			"name":        fmt.Sprintf("SyntheticTool%04d", i),
			"description": "synthetic benchmark tool",
			"parameters":  map[string]any{"type": "object"},
		})
	}
	return out
}

func SyntheticTaskBoardTasks(count int) []orchestration.BoardTask {
	out := make([]orchestration.BoardTask, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, orchestration.BoardTask{
			Title:    fmt.Sprintf("task %04d", i),
			Priority: count - i,
		})
	}
	return out
}
