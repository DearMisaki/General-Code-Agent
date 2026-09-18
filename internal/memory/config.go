package memory

import (
	"fmt"
	"net/http"
	"os"
)

type MemoryConfig struct {
	LongTerm LongTermMemoryConfig `yaml:"long_term"`
}

type LongTermMemoryConfig struct {
	Enabled           bool                   `yaml:"enabled"`
	Backend           string                 `yaml:"backend"`
	AppID             string                 `yaml:"app_id"`
	ProjectIDStrategy string                 `yaml:"project_id_strategy"`
	WriteMode         string                 `yaml:"write_mode"`
	MarkdownMirror    bool                   `yaml:"markdown_mirror"`
	Extraction        MemoryExtractionConfig `yaml:"extraction"`
	Recall            MemoryRecallConfig     `yaml:"recall"`
	APIKey            string                 `yaml:"api_key"`
	BaseURL           string                 `yaml:"base_url"`
	HTTPClient        *http.Client           `yaml:"-"`
}

type MemoryExtractionConfig struct {
	Enabled             bool    `yaml:"enabled"`
	Trigger             string  `yaml:"trigger"`
	MinConfidence       float64 `yaml:"min_confidence"`
	MaxCandidatesPerRun int     `yaml:"max_candidates_per_run"`
	DrainTimeoutMs      int     `yaml:"drain_timeout_ms"`
}

type MemoryRecallConfig struct {
	Enabled                 bool    `yaml:"enabled"`
	TopK                    int     `yaml:"top_k"`
	Threshold               float64 `yaml:"threshold"`
	Rerank                  bool    `yaml:"rerank"`
	TokenBudget             int     `yaml:"token_budget"`
	ExcludeRecentlySurfaced bool    `yaml:"exclude_recently_surfaced"`
}

func NewMemoryStoreFromConfig(config MemoryConfig) (MemoryStore, error) {
	lt := config.LongTerm
	if !lt.Enabled || lt.Backend == "" || lt.Backend == "local-markdown" || lt.Backend == "local" {
		return NewLocalMemoryStore(), nil
	}
	if lt.Backend != "mem0-platform" {
		return nil, fmt.Errorf("unsupported long-term memory backend %q", lt.Backend)
	}
	apiKey := firstNonEmpty(lt.APIKey, os.Getenv("MEM0_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("MEM0_API_KEY is required for mem0-platform memory backend")
	}
	return NewMem0MemoryStore(Mem0StoreOptions{
		BaseURL: firstNonEmpty(lt.BaseURL, os.Getenv("MEM0_BASE_URL")),
		APIKey:  apiKey,
		Client:  lt.HTTPClient,
	}), nil
}
