package memory

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type MemoryAuditEvent string

const (
	EventMemoryExtractStarted        MemoryAuditEvent = "memory.extract.started"
	EventMemoryExtractCompleted      MemoryAuditEvent = "memory.extract.completed"
	EventMemoryCandidateFiltered     MemoryAuditEvent = "memory.candidate.filtered"
	EventMemoryCandidateAccepted     MemoryAuditEvent = "memory.candidate.accepted"
	EventMemoryWriteAdded            MemoryAuditEvent = "memory.write.added"
	EventMemoryWriteUpdated          MemoryAuditEvent = "memory.write.updated"
	EventMemoryWriteSkippedDuplicate MemoryAuditEvent = "memory.write.skipped_duplicate"
	EventMemoryWriteConflictDetected MemoryAuditEvent = "memory.write.conflict_detected"
	EventMemoryRecallStarted         MemoryAuditEvent = "memory.recall.started"
	EventMemoryRecallCompleted       MemoryAuditEvent = "memory.recall.completed"
	EventMemoryInjected              MemoryAuditEvent = "memory.injected"
	EventMemoryFallbackUsed          MemoryAuditEvent = "memory.fallback.used"
)

type MemoryAuditRecord struct {
	ID        string            `json:"id,omitempty"`
	Time      time.Time         `json:"time"`
	Event     MemoryAuditEvent  `json:"event"`
	MemoryID  string            `json:"memory_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	AgentID   string            `json:"agent_id,omitempty"`
	TeamName  string            `json:"team_name,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Summary   string            `json:"summary,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type MemoryAuditWriter struct {
	path string
}

func NewMemoryAuditWriter(path string) *MemoryAuditWriter {
	return &MemoryAuditWriter{path: path}
}

func (w *MemoryAuditWriter) Append(record MemoryAuditRecord) error {
	if w == nil || w.path == "" {
		return nil
	}
	if record.Time.IsZero() {
		record.Time = time.Now().UTC()
	}
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(record)
}

func ListMemoryAuditRecords(path string) ([]MemoryAuditRecord, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var records []MemoryAuditRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var record MemoryAuditRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err == nil {
			records = append(records, record)
		}
	}
	return records, scanner.Err()
}
