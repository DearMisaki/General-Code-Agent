package contextmgr

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"mewcode/internal/evals"
)

func TestBenchmarkPrepareRequestCreatesFreshConversation(t *testing.T) {
	gateway := NewGateway(GatewayOptions{})
	first := newBenchmarkPrepareRequest(10, 0, 1, 1)
	if _, err := gateway.PrepareTurn(context.Background(), first); err != nil {
		t.Fatalf("prepare first benchmark request: %v", err)
	}

	second := newBenchmarkPrepareRequest(10, 0, 1, 1)
	if got, want := second.Conversation.Len(), 10; got != want {
		t.Fatalf("fresh benchmark conversation length = %d, want %d", got, want)
	}
}

func TestBenchmarkPrepareRequestExercisesToolResultReplacement(t *testing.T) {
	req := newBenchmarkPrepareRequest(100, 1<<20, 0, 0)
	req.WorkDir = t.TempDir()
	turn, err := NewGateway(GatewayOptions{}).PrepareTurn(context.Background(), req)
	if err != nil {
		t.Fatalf("prepare tool-result benchmark request: %v", err)
	}

	for _, msg := range turn.APIConversation.GetMessages() {
		for _, result := range msg.ToolResults {
			if strings.HasPrefix(result.Content, "[Result of ") {
				return
			}
		}
	}
	t.Fatal("benchmark request did not exercise tool-result replacement")
}

func TestDeferredToolsBenchmarkRequestRendersDeferredToolNames(t *testing.T) {
	req := newDeferredToolsBenchmarkRequest(3)
	turn, err := NewGateway(GatewayOptions{}).PrepareTurn(context.Background(), req)
	if err != nil {
		t.Fatalf("prepare deferred-tools benchmark request: %v", err)
	}
	if len(req.DeferredToolNames) != 3 {
		t.Fatalf("deferred tool names = %d, want 3", len(req.DeferredToolNames))
	}

	messages := turn.APIConversation.GetMessages()
	for _, name := range []string{"SyntheticTool0000", "SyntheticTool0001", "SyntheticTool0002"} {
		found := false
		for _, msg := range messages {
			if strings.Contains(msg.Content, name) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("rendered conversation does not expose deferred tool %q", name)
		}
	}
}

func BenchmarkGatewayPrepareTurn(b *testing.B) {
	for _, tc := range []struct {
		name            string
		messages        int
		toolResultBytes int
		notifications   int
		memories        int
	}{
		{"messages_10", 10, 0, 0, 0},
		{"messages_1000", 1000, 0, 10, 10},
		{"tool_result_1mb", 100, 1 << 20, 0, 0},
		{"notifications_100", 100, 0, 100, 0},
	} {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkGatewayPrepareTurn(b, tc.messages, tc.toolResultBytes, tc.notifications, tc.memories)
		})
	}
}

func BenchmarkGatewayPrepareTurnLargeConversation(b *testing.B) {
	benchmarkGatewayPrepareTurn(b, 1000, 0, 10, 10)
}

func BenchmarkGatewayPrepareTurnLargeToolResult(b *testing.B) {
	benchmarkGatewayPrepareTurn(b, 100, 1<<20, 0, 0)
}

func BenchmarkGatewayPrepareTurnManyNotifications(b *testing.B) {
	benchmarkGatewayPrepareTurn(b, 100, 0, 100, 0)
}

func BenchmarkGatewayPrepareTurnManyDeferredTools(b *testing.B) {
	for _, count := range []int{0, 100, 1000} {
		b.Run(fmt.Sprintf("tools_%d", count), func(b *testing.B) {
			gateway := NewGateway(GatewayOptions{})
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				req := newDeferredToolsBenchmarkRequest(count)
				b.StartTimer()
				if _, err := gateway.PrepareTurn(context.Background(), req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchmarkGatewayPrepareTurn(b *testing.B, messages, toolResultBytes, notifications, memories int) {
	gateway := NewGateway(GatewayOptions{})
	workDir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		req := newBenchmarkPrepareRequest(messages, toolResultBytes, notifications, memories)
		req.WorkDir = workDir
		b.StartTimer()
		if _, err := gateway.PrepareTurn(context.Background(), req); err != nil {
			b.Fatal(err)
		}
	}
}

func newBenchmarkPrepareRequest(messages, toolResultBytes, notifications, memories int) PrepareRequest {
	return PrepareRequest{
		Conversation:    evals.SyntheticConversation(messages, toolResultBytes),
		Instructions:    "benchmark instructions",
		MemoryContent:   evals.SyntheticMemoryBlocks(memories),
		Notifications:   evals.SyntheticNotifications(notifications),
		ContextWindow:   200000,
		MaxOutputTokens: 8192,
	}
}

func newDeferredToolsBenchmarkRequest(count int) PrepareRequest {
	return PrepareRequest{
		Conversation:      evals.SyntheticConversation(100, 0),
		DeferredToolNames: evals.SyntheticDeferredToolNames(count),
		ContextWindow:     200000,
		MaxOutputTokens:   8192,
	}
}
