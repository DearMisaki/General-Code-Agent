package orchestration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMonitorAppendAndListEvents(t *testing.T) {
	monitor := NewMonitor(filepath.Join(t.TempDir(), "events.jsonl"))
	if err := monitor.AppendEvent(OrchestrationEvent{
		Type:    EventTaskCreated,
		Actor:   "lead",
		Summary: "task created",
		Metadata: map[string]string{
			"task_id": "task_1",
		},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	events, err := monitor.ListEvents()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Type != EventTaskCreated || events[0].Metadata["task_id"] != "task_1" {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func TestMonitorListSkipsInvalidLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	monitor := NewMonitor(path)
	if err := monitor.AppendEvent(OrchestrationEvent{Type: EventMailSent}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := os.WriteFile(path, append([]byte("{bad json\n"), mustRead(t, path)...), 0o644); err != nil {
		t.Fatalf("write bad line: %v", err)
	}
	events, err := monitor.ListEvents()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
}

func TestMonitorNilAndEmptyPathAreNoops(t *testing.T) {
	var monitor *Monitor
	if err := monitor.AppendEvent(OrchestrationEvent{Type: EventAgentSpawned}); err != nil {
		t.Fatalf("nil append should not fail: %v", err)
	}
	if err := NewMonitor("").AppendEvent(OrchestrationEvent{Type: EventAgentSpawned}); err != nil {
		t.Fatalf("empty path append should not fail: %v", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
