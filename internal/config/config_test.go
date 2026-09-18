package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigParsesMemoryConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	data := []byte(`
providers:
  - name: local
    protocol: openai-compat
    base_url: http://example.test/v1
    model: test-model
memory:
  long_term:
    enabled: true
    backend: mem0-platform
    app_id: mewcode-test
    project_id_strategy: git_remote_or_root_hash
    write_mode: async
    markdown_mirror: true
    api_key: test-key
    base_url: http://mem0.test
    extraction:
      enabled: true
      trigger: every_turn
      min_confidence: 0.8
      max_candidates_per_run: 7
      drain_timeout_ms: 1234
    recall:
      enabled: true
      top_k: 5
      threshold: 0.2
      rerank: true
      token_budget: 999
      exclude_recently_surfaced: true
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() = %v", err)
	}
	lt := cfg.Memory.LongTerm
	if !lt.Enabled || lt.Backend != "mem0-platform" || lt.APIKey != "test-key" {
		t.Fatalf("memory long_term = %+v", lt)
	}
	if !lt.Extraction.Enabled || lt.Extraction.MaxCandidatesPerRun != 7 || lt.Extraction.DrainTimeoutMs != 1234 {
		t.Fatalf("memory extraction = %+v", lt.Extraction)
	}
	if !lt.Recall.Enabled || lt.Recall.TopK != 5 || lt.Recall.TokenBudget != 999 {
		t.Fatalf("memory recall = %+v", lt.Recall)
	}
}

func TestMergeConfigOverridesMemoryConfig(t *testing.T) {
	base := &AppConfig{}
	override := &AppConfig{}
	override.hasMemory = true
	override.Memory.LongTerm.Enabled = true
	override.Memory.LongTerm.Backend = "local"
	override.Memory.LongTerm.Extraction.Enabled = true
	override.Memory.LongTerm.Recall.Enabled = true
	override.Memory.LongTerm.Recall.TopK = 3

	merged := mergeConfig(base, override)
	if !merged.Memory.LongTerm.Enabled || merged.Memory.LongTerm.Backend != "local" {
		t.Fatalf("memory long_term = %+v", merged.Memory.LongTerm)
	}
	if !merged.Memory.LongTerm.Extraction.Enabled {
		t.Fatalf("memory extraction was not merged: %+v", merged.Memory.LongTerm.Extraction)
	}
	if !merged.Memory.LongTerm.Recall.Enabled || merged.Memory.LongTerm.Recall.TopK != 3 {
		t.Fatalf("memory recall was not merged: %+v", merged.Memory.LongTerm.Recall)
	}
}
