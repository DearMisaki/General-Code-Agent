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

func TestRunLiveTeammateEvalIsNotImplemented(t *testing.T) {
	err := run([]string{"--mode", "teammate", "--runner", "live", "--cases", t.TempDir(), "--config", "config.yaml"})
	if err == nil || !strings.Contains(err.Error(), "live teammate eval is not implemented") {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunLiveContextEvalRequiresConfig(t *testing.T) {
	err := run([]string{"--mode", "context", "--runner", "live", "--cases", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "--config is required") {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunLiveContextEvalRequiresMem0Backend(t *testing.T) {
	dir := t.TempDir()
	caseDir := filepath.Join(dir, "cases")
	if err := os.MkdirAll(caseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(caseDir, "case.json"), []byte(`{"case_id":"ctx-live","query":"hello","required_context_ids":["a"]}`), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	configData := []byte(`providers:
  - name: fake
    protocol: openai-compat
    base_url: http://127.0.0.1:9/v1
    model: fake
memory:
  long_term:
    enabled: true
    backend: local
`)
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatal(err)
	}

	err := run([]string{"--mode", "context", "--runner", "live", "--cases", caseDir, "--config", configPath, "--provider", "fake"})
	if err == nil || !strings.Contains(err.Error(), "requires mem0-platform") {
		t.Fatalf("run() error = %v", err)
	}
}
