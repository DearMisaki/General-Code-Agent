package memory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryExtractionSchedulerCoalescesInFlightRuns(t *testing.T) {
	var calls atomic.Int64
	s := NewMemoryExtractionScheduler(MemoryExtractionSchedulerOptions{
		Extract: func(context.Context) error {
			calls.Add(1)
			time.Sleep(20 * time.Millisecond)
			return nil
		},
	})
	_ = s.Execute(context.Background())
	_ = s.Execute(context.Background())
	if err := s.Drain(500); err != nil {
		t.Fatalf("Drain() = %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want initial plus trailing run", got)
	}
}
