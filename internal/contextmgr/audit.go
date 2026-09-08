package contextmgr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	EventContextPrepare          EventType = "context.prepare"
	EventContextRender           EventType = "context.render"
	EventContextToolResultBudget EventType = "context.tool_result_budget"
	EventContextCompact          EventType = "context.compact"
	EventContextHandoff          EventType = "context.handoff"
	EventContextToolExposure     EventType = "context.tool_exposure"
	EventContextRecoveryFileRead EventType = "context.recovery_file_read"
	EventContextRecoverySkill    EventType = "context.recovery_skill"
)

type AuditWriter struct {
	workDir string
}

func NewAuditWriter(workDir string) *AuditWriter {
	return &AuditWriter{workDir: workDir}
}

func (w *AuditWriter) Append(record AuditRecord) error {
	if w == nil || w.workDir == "" {
		return nil
	}
	if record.Time.IsZero() {
		record.Time = time.Now().UTC()
	}
	path := filepath.Join(w.workDir, ".mewcode", "context", "audit.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	_, err = f.Write([]byte("\n"))
	return err
}
