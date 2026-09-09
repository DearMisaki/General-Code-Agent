package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunContextEvalWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "out.jsonl")
	caseDir := filepath.Join(dir, "cases")
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caseDir, "case.json"), []byte(`{"case_id":"ctx-001","required_context_ids":["a"]}`), 0644); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"--mode", "context", "--cases", caseDir, "--out", out})
	if err != nil {
		t.Fatalf("run() = %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"case_id":"ctx-001"`) {
		t.Fatalf("output = %s", data)
	}
}
