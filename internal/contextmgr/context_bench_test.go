package contextmgr

import (
	"context"
	"fmt"
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
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		req := newBenchmarkPrepareRequest(messages, toolResultBytes, notifications, memories)
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
		Conversation:    evals.SyntheticConversation(100, 0),
		ToolSchemas:     evals.SyntheticToolSchemas(count),
		ContextWindow:   200000,
		MaxOutputTokens: 8192,
	}
}
