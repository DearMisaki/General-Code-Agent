package contextmgr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditWriterAppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	writer := NewAuditWriter(dir)
	err := writer.Append(AuditRecord{
		ID:        "rec-1",
		Time:      time.Unix(1, 0).UTC(),
		Event:     EventContextPrepare,
		AgentID:   "agent-1",
		SessionID: "session-1",
		ContextID: "ctx-1",
		Summary:   "prepared",
		Metadata:  map[string]any{"messages": float64(2)},
	})
	if err != nil {
		t.Fatalf("append audit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".mewcode", "context", "audit.jsonl"))
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, `"event":"context.prepare"`) {
		t.Fatalf("missing event: %s", line)
	}
	if !strings.Contains(line, `"agent_id":"agent-1"`) {
		t.Fatalf("missing agent id: %s", line)
	}
}
