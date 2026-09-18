package memoryruntime

import (
	"context"
	"testing"

	"mewcode/internal/agent"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
	"mewcode/internal/teams"
	"mewcode/internal/tools"
)

type fakeClient struct{}

func (fakeClient) Stream(context.Context, *conversation.Manager, []map[string]any) (<-chan llm.StreamEvent, <-chan error) {
	events := make(chan llm.StreamEvent)
	errs := make(chan error)
	close(events)
	close(errs)
	return events, errs
}

func TestBuildCreatesEnabledMemoryRuntime(t *testing.T) {
	rt, err := Build(BuildOptions{
		Config: memory.MemoryConfig{
			LongTerm: memory.LongTermMemoryConfig{
				Enabled: true,
				Backend: "local",
				Extraction: memory.MemoryExtractionConfig{
					Enabled:             true,
					MinConfidence:       0.8,
					MaxCandidatesPerRun: 3,
					DrainTimeoutMs:      25,
				},
				Recall: memory.MemoryRecallConfig{
					Enabled:     true,
					TopK:        4,
					TokenBudget: 500,
					Threshold:   0.1,
					Rerank:      true,
				},
			},
		},
		Client:       fakeClient{},
		Registry:     tools.NewRegistry(),
		Conversation: conversation.NewManager(),
		ProjectRoot:  t.TempDir(),
		Protocol:     "openai-compat",
		SessionID:    "session-1",
		AgentID:      "main",
	})
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	if rt.Store == nil || rt.Pipeline == nil || rt.Recall == nil || rt.Extraction == nil {
		t.Fatalf("runtime not fully wired: %+v", rt)
	}
}

func TestRuntimeAttachWiresAgentAndTeam(t *testing.T) {
	rt, err := Build(BuildOptions{
		Config: memory.MemoryConfig{
			LongTerm: memory.LongTermMemoryConfig{
				Enabled:    true,
				Backend:    "local",
				Extraction: memory.MemoryExtractionConfig{Enabled: true},
				Recall:     memory.MemoryRecallConfig{Enabled: true},
			},
		},
		Client:       fakeClient{},
		Registry:     tools.NewRegistry(),
		Conversation: conversation.NewManager(),
		ProjectRoot:  t.TempDir(),
		Protocol:     "openai-compat",
	})
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	ag := agent.New(fakeClient{}, tools.NewRegistry(), "openai-compat")
	rt.AttachAgent(ag)
	if ag.ContextGateway == nil || ag.ContextLifecycle == nil {
		t.Fatal("context management was not initialized on agent")
	}
	if ag.MemoryRecall == nil || ag.MemoryExtraction == nil {
		t.Fatal("memory services were not attached to agent")
	}

	team := teams.NewTeam("runtime-test", teams.ModeInProcess)
	rt.AttachTeam(team)
	if team.MemoryPipeline == nil {
		t.Fatal("memory pipeline was not attached to team")
	}
}

func TestBuildLeavesMemoryDisabledWhenConfigDisabled(t *testing.T) {
	rt, err := Build(BuildOptions{
		Config:      memory.MemoryConfig{LongTerm: memory.LongTermMemoryConfig{Enabled: false}},
		ProjectRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	if rt.Recall != nil || rt.Extraction != nil || rt.Pipeline != nil {
		t.Fatalf("disabled runtime should not expose active memory services: %+v", rt)
	}
}
