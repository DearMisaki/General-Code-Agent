package contextmgr

import (
	"context"
	"fmt"
	"testing"

	"mewcode/internal/evals"
)

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
			gateway := NewGateway(GatewayOptions{})
			conv := evals.SyntheticConversation(tc.messages, tc.toolResultBytes)
			req := PrepareRequest{
				Conversation:    conv,
				Instructions:    "benchmark instructions",
				MemoryContent:   evals.SyntheticMemoryBlocks(tc.memories),
				Notifications:   evals.SyntheticNotifications(tc.notifications),
				ContextWindow:   200000,
				MaxOutputTokens: 8192,
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := gateway.PrepareTurn(context.Background(), req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGatewayPrepareTurnManyDeferredTools(b *testing.B) {
	for _, count := range []int{0, 100, 1000} {
		b.Run(fmt.Sprintf("tools_%d", count), func(b *testing.B) {
			gateway := NewGateway(GatewayOptions{})
			req := PrepareRequest{
				Conversation:    evals.SyntheticConversation(100, 0),
				ToolSchemas:     evals.SyntheticToolSchemas(count),
				ContextWindow:   200000,
				MaxOutputTokens: 8192,
			}
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := gateway.PrepareTurn(context.Background(), req); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
