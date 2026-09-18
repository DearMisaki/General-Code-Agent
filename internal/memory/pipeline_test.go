package memory

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMemoryWritePipelineFiltersDedupesAndWrites(t *testing.T) {
	store := NewLocalMemoryStore()
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	pipeline := NewMemoryWritePipeline(MemoryWritePipelineOptions{
		Store:   store,
		Policy:  NewMemoryPolicyFilter(PolicyOptions{MinConfidence: 0.75}),
		Deduper: NewDeduper(store, DedupeOptions{DuplicateThreshold: 0.95}),
		Audit:   NewMemoryAuditWriter(auditPath),
	})
	result, err := pipeline.WriteCandidates(context.Background(), []MemoryCandidate{
		{Scope: ScopeProject, Kind: KindProjectFact, Memory: "ContextGateway prepares context.", Confidence: 0.9, ProjectID: "proj"},
		{Scope: ScopeProject, Kind: KindProjectFact, Memory: "The password is secret.", Confidence: 0.9, ProjectID: "proj"},
	})
	if err != nil {
		t.Fatalf("WriteCandidates() = %v", err)
	}
	if result.Added != 1 || result.Filtered != 1 {
		t.Fatalf("result = %+v, want one added and one filtered", result)
	}
	records, _ := ListMemoryAuditRecords(auditPath)
	if len(records) == 0 {
		t.Fatal("expected audit records")
	}
}
