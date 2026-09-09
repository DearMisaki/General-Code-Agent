package orchestration

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type OrchestrationEventType string

const (
	EventSessionCreated      OrchestrationEventType = "session.created"
	EventSessionStateChanged OrchestrationEventType = "session.state_changed"
	EventTaskCreated         OrchestrationEventType = "task.created"
	EventTaskClaimed         OrchestrationEventType = "task.claimed"
	EventTaskUpdated         OrchestrationEventType = "task.updated"
	EventMailSent            OrchestrationEventType = "mail.sent"
	EventAgentSpawned        OrchestrationEventType = "agent.spawned"
	EventAgentCompleted      OrchestrationEventType = "agent.completed"
	EventAgentFailed         OrchestrationEventType = "agent.failed"
)

type OrchestrationEvent struct {
	ID       string                 `json:"id"`
	Time     time.Time              `json:"time"`
	Type     OrchestrationEventType `json:"type"`
	Actor    string                 `json:"actor,omitempty"`
	Summary  string                 `json:"summary,omitempty"`
	Metadata map[string]string      `json:"metadata,omitempty"`
}

type Monitor struct {
	path string
}

func NewMonitor(path string) *Monitor {
	return &Monitor{path: path}
}

func (m *Monitor) AppendEvent(event OrchestrationEvent) error {
	if m == nil || m.path == "" {
		return nil
	}
	if event.ID == "" {
		event.ID = "event_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(m.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	_, err = f.Write([]byte("\n"))
	return err
}

func (m *Monitor) ListEvents() ([]OrchestrationEvent, error) {
	if m == nil || m.path == "" {
		return nil, nil
	}
	f, err := os.Open(m.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []OrchestrationEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var event OrchestrationEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}
