package tui

import (
	"context"
	"testing"

	"mewcode/internal/agent"
	"mewcode/internal/config"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
	"mewcode/internal/tools"
)

type tuiMemoryFakeClient struct{}

func (tuiMemoryFakeClient) Stream(context.Context, *conversation.Manager, []map[string]any) (<-chan llm.StreamEvent, <-chan error) {
	events := make(chan llm.StreamEvent)
	errs := make(chan error)
	close(events)
	close(errs)
	return events, errs
}

func TestNewKeepsMemoryConfigForMainAgentRuntime(t *testing.T) {
	cfg := memory.MemoryConfig{
		LongTerm: memory.LongTermMemoryConfig{
			Enabled:    true,
			Backend:    "local",
			Extraction: memory.MemoryExtractionConfig{Enabled: true},
			Recall:     memory.MemoryRecallConfig{Enabled: true},
		},
	}
	m := New(nil, nil, nil, cfg)
	if !m.memoryConfig.LongTerm.Enabled || m.memoryConfig.LongTerm.Backend != "local" {
		t.Fatalf("memory config not stored on model: %+v", m.memoryConfig.LongTerm)
	}
}

func TestAttachMainMemoryRuntimeWiresAgentAndTeam(t *testing.T) {
	m := New(nil, nil, nil, memory.MemoryConfig{
		LongTerm: memory.LongTermMemoryConfig{
			Enabled:    true,
			Backend:    "local",
			Extraction: memory.MemoryExtractionConfig{Enabled: true},
			Recall:     memory.MemoryRecallConfig{Enabled: true},
		},
	})
	m.client = tuiMemoryFakeClient{}
	m.registry = tools.NewRegistry()
	m.conversation = conversation.NewManager()
	m.sessionID = "session-1"
	m.teamMgr = nil

	ag := agent.New(m.client, m.registry, "openai-compat")
	if err := m.attachMainMemoryRuntime(ag, t.TempDir(), "openai-compat"); err != nil {
		t.Fatalf("attachMainMemoryRuntime() = %v", err)
	}
	if ag.MemoryRecall == nil || ag.MemoryExtraction == nil {
		t.Fatal("main agent memory services were not attached")
	}
	if m.memoryRuntime == nil {
		t.Fatal("model did not keep memory runtime for drain")
	}
}

var _ = config.AppConfig{}
