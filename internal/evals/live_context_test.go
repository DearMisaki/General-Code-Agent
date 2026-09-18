package evals

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
)

func TestContextLiveRunnerSeedsMemoryAndScoresRecalledLogicalIDs(t *testing.T) {
	store := &fakeLiveStore{
		searchResults: []memory.MemorySearchResult{{
			Memory: memory.StoredMemory{
				ID:       "mem-physical-1",
				Memory:   "用户说他的名字是 xyz",
				Metadata: map[string]string{"eval_logical_id": "identity.name"},
			},
			Score: 0.91,
		}},
	}
	client := &fakeLiveClient{text: "你的名字是 xyz。"}
	runner := NewContextLiveRunner(ContextLiveRunnerOptions{
		Store:     store,
		Client:    client,
		Provider:  "fake",
		Model:     "fake-model",
		UserID:    "eval-user",
		AppID:     "mewcode-eval",
		AgentID:   "main",
		ProjectID: "/repo",
		SessionID: "eval-session",
		RunID:     "eval-run",
	})

	result := runner.RunCase(context.Background(), ContextEvalCase{
		CaseID:                 "identity",
		Query:                  "我是谁？",
		MemorySeeds:            []EvalMemorySeed{{ID: "identity.name", Memory: "用户说他的名字是 xyz", Scope: "user", Kind: "preference"}},
		RequiredContextIDs:     []string{"identity.name"},
		ExpectedAnswerContains: []string{"xyz"},
		K:                      1,
	})

	if result.Error != "" {
		t.Fatalf("RunCase error = %s", result.Error)
	}
	if len(store.addRequests) != 1 {
		t.Fatalf("Add calls = %d, want 1", len(store.addRequests))
	}
	candidate := store.addRequests[0].Candidates[0]
	if candidate.UserID != "eval-user" || !strings.Contains(candidate.AppID, "eval-run") {
		t.Fatalf("candidate scope = user %q app %q", candidate.UserID, candidate.AppID)
	}
	if candidate.Metadata["eval_logical_id"] != "identity.name" || candidate.Metadata["eval_case_id"] != "identity" || candidate.Metadata["eval_run_id"] != "eval-run" {
		t.Fatalf("candidate metadata = %+v", candidate.Metadata)
	}
	if !store.searched {
		t.Fatal("Search was not called")
	}
	if len(store.searchRequests) == 0 || !strings.Contains(store.searchRequests[0].Filters.AppID, "eval-run") {
		t.Fatalf("search filters = %+v, want app_id scoped by eval run", store.searchRequests)
	}
	if !client.streamed {
		t.Fatal("LLM stream was not called")
	}
	if result.Score.Recall != 1 || result.Score.Precision != 1 || result.Score.MRR != 1 {
		t.Fatalf("score = %+v", result.Score)
	}
	if !result.AnswerContainsPassed || !strings.Contains(result.LLMText, "xyz") {
		t.Fatalf("answer check failed: %+v", result)
	}
}

func TestContextLiveRunnerReturnsCaseErrorWhenSeedFails(t *testing.T) {
	runner := NewContextLiveRunner(ContextLiveRunnerOptions{
		Store:     &fakeLiveStore{addErr: errors.New("mem0 unavailable")},
		Client:    &fakeLiveClient{text: "unused"},
		UserID:    "eval-user",
		AppID:     "mewcode-eval",
		AgentID:   "main",
		ProjectID: "/repo",
		SessionID: "eval-session",
		RunID:     "eval-run",
	})

	result := runner.RunCase(context.Background(), ContextEvalCase{
		CaseID:             "seed-fails",
		Query:              "hello",
		MemorySeeds:        []EvalMemorySeed{{ID: "seed", Memory: "fact"}},
		RequiredContextIDs: []string{"seed"},
	})

	if !strings.Contains(result.Error, "mem0 unavailable") {
		t.Fatalf("error = %q", result.Error)
	}
}

type fakeLiveStore struct {
	addRequests    []memory.AddMemoryRequest
	searchResults  []memory.MemorySearchResult
	searchRequests []memory.SearchMemoryRequest
	addErr         error
	searched       bool
}

func (s *fakeLiveStore) Add(_ context.Context, req memory.AddMemoryRequest) ([]memory.StoredMemory, error) {
	s.addRequests = append(s.addRequests, req)
	if s.addErr != nil {
		return nil, s.addErr
	}
	out := make([]memory.StoredMemory, 0, len(req.Candidates))
	for _, candidate := range req.Candidates {
		out = append(out, memory.StoredMemory{ID: "physical-" + candidate.ID, Memory: candidate.Memory, Metadata: candidate.Metadata})
	}
	return out, nil
}

func (s *fakeLiveStore) Search(_ context.Context, req memory.SearchMemoryRequest) ([]memory.MemorySearchResult, error) {
	s.searched = true
	s.searchRequests = append(s.searchRequests, req)
	return s.searchResults, nil
}

func (s *fakeLiveStore) Update(context.Context, string, memory.MemoryPatch) error { return nil }
func (s *fakeLiveStore) Delete(context.Context, string) error                     { return nil }
func (s *fakeLiveStore) GetAll(context.Context, memory.MemoryFilters, memory.MemoryPage) (memory.MemoryPageResult, error) {
	return memory.MemoryPageResult{}, nil
}
func (s *fakeLiveStore) History(context.Context, string) ([]memory.MemoryHistoryEvent, error) {
	return nil, nil
}

type fakeLiveClient struct {
	text     string
	streamed bool
}

func (c *fakeLiveClient) Stream(_ context.Context, _ *conversation.Manager, _ []map[string]any) (<-chan llm.StreamEvent, <-chan error) {
	events := make(chan llm.StreamEvent, 2)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		events <- llm.TextDelta{Text: c.text}
		events <- llm.StreamEnd{StopReason: "end_turn"}
	}()
	go func() {
		defer close(errs)
	}()
	c.streamed = true
	return events, errs
}
