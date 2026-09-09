package evals

import (
	"fmt"
	"strings"

	"mewcode/internal/conversation"
	"mewcode/internal/orchestration"
)

func SyntheticConversation(size int, toolResultBytes int) *conversation.Manager {
	conv := conversation.NewManager()
	if size <= 0 {
		return conv
	}

	ordinaryMessages := size
	if toolResultBytes > 0 {
		ordinaryMessages = max(0, size-2)
	}
	for i := 0; i < ordinaryMessages; i++ {
		if i%2 == 0 {
			conv.AddUserMessage(fmt.Sprintf("[ctx-%04d] user request %d", i, i))
			continue
		}
		conv.AddAssistantMessage(fmt.Sprintf("[ctx-%04d] assistant response %d", i, i))
	}
	if toolResultBytes <= 0 {
		return conv
	}

	toolUseID := fmt.Sprintf("synthetic-tool-use-%04d", ordinaryMessages)
	if size > 1 {
		conv.AddAssistantFull("synthetic benchmark tool call", nil, []conversation.ToolUseBlock{{
			ToolUseID: toolUseID,
			ToolName:  "SyntheticTool0000",
			Arguments: map[string]any{"fixture": "deterministic"},
		}})
	}
	conv.AddToolResultMessage(toolUseID, strings.Repeat("x", toolResultBytes), false)
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

func SyntheticDeferredToolNames(count int) []string {
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, fmt.Sprintf("SyntheticTool%04d", i))
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
