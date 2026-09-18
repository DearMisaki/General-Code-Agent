package main

import (
	"context"
	"testing"

	"mewcode/internal/config"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
	"mewcode/internal/teams"
	"mewcode/internal/tools"
)

type teammateFakeClient struct{}

func (teammateFakeClient) Stream(context.Context, *conversation.Manager, []map[string]any) (<-chan llm.StreamEvent, <-chan error) {
	events := make(chan llm.StreamEvent)
	errs := make(chan error)
	close(events)
	close(errs)
	return events, errs
}

func TestAttachTeammateMemoryRuntimeWiresMemberAndTeam(t *testing.T) {
	cfg := &config.AppConfig{
		Memory: memory.MemoryConfig{
			LongTerm: memory.LongTermMemoryConfig{
				Enabled:    true,
				Backend:    "local",
				Extraction: memory.MemoryExtractionConfig{Enabled: true},
				Recall:     memory.MemoryRecallConfig{Enabled: true},
			},
		},
	}
	provider := config.ProviderConfig{Protocol: "openai-compat"}
	registry := tools.NewRegistry()
	team := teams.NewTeam("worker-memory-test", teams.ModeInProcess)
	member := team.AddMember("worker", teammateFakeClient{}, registry, provider.Protocol)

	rt, err := attachTeammateMemoryRuntime(cfg, provider, teammateFakeClient{}, registry, team, member)
	if err != nil {
		t.Fatalf("attachTeammateMemoryRuntime() = %v", err)
	}
	if rt == nil {
		t.Fatal("runtime is nil")
	}
	if member.AgentRef.MemoryRecall == nil || member.AgentRef.MemoryExtraction == nil {
		t.Fatal("member agent memory services were not attached")
	}
	if team.MemoryPipeline == nil {
		t.Fatal("team memory pipeline was not attached")
	}
}
