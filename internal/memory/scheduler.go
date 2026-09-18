package memory

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type MemoryExtractionSchedulerOptions struct {
	Extract func(context.Context) error
}

type MemoryExtractionScheduler struct {
	opts MemoryExtractionSchedulerOptions

	mu      sync.Mutex
	running bool
	pending bool
	done    chan struct{}
	lastErr error
}

func NewMemoryExtractionScheduler(opts MemoryExtractionSchedulerOptions) *MemoryExtractionScheduler {
	return &MemoryExtractionScheduler{opts: opts, done: make(chan struct{})}
}

func (s *MemoryExtractionScheduler) Execute(ctx context.Context) error {
	if s == nil || s.opts.Extract == nil {
		return nil
	}
	s.mu.Lock()
	if s.running {
		s.pending = true
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.done = make(chan struct{})
	s.mu.Unlock()
	go s.run(ctx)
	return nil
}

func (s *MemoryExtractionScheduler) run(ctx context.Context) {
	defer close(s.done)
	for {
		if err := s.opts.Extract(ctx); err != nil {
			s.mu.Lock()
			s.lastErr = err
			s.mu.Unlock()
		}
		s.mu.Lock()
		if !s.pending {
			s.running = false
			s.mu.Unlock()
			return
		}
		s.pending = false
		s.mu.Unlock()
	}
}

func (s *MemoryExtractionScheduler) Drain(timeoutMs int) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if !s.running {
		err := s.lastErr
		s.mu.Unlock()
		return err
	}
	done := s.done
	s.mu.Unlock()
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeoutMs < 0 {
		timeout = 60 * time.Second
	}
	if timeoutMs == 0 {
		select {
		case <-done:
			return s.lastErr
		default:
			return fmt.Errorf("memory extraction still in flight")
		}
	}
	select {
	case <-done:
		s.mu.Lock()
		err := s.lastErr
		s.mu.Unlock()
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timed out draining memory extraction")
	}
}
