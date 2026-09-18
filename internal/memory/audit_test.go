package memory

import (
	"path/filepath"
	"testing"
)

func TestMemoryAuditWriterAppendAndList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-audit.jsonl")
	w := NewMemoryAuditWriter(path)
	err := w.Append(MemoryAuditRecord{
		Event:    EventMemoryWriteAdded,
		MemoryID: "mem_1",
		Summary:  "added memory",
	})
	if err != nil {
		t.Fatalf("Append() = %v", err)
	}
	records, err := ListMemoryAuditRecords(path)
	if err != nil {
		t.Fatalf("ListMemoryAuditRecords() = %v", err)
	}
	if len(records) != 1 || records[0].MemoryID != "mem_1" {
		t.Fatalf("records = %+v, want mem_1", records)
	}
}
