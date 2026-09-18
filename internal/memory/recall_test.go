package memory

import (
	"context"
	"strings"
	"testing"
)

type recordingMemoryStore struct {
	queries []SearchMemoryRequest
}

func (s *recordingMemoryStore) Add(context.Context, AddMemoryRequest) ([]StoredMemory, error) {
	return nil, nil
}

func (s *recordingMemoryStore) Search(_ context.Context, req SearchMemoryRequest) ([]MemorySearchResult, error) {
	s.queries = append(s.queries, req)
	return nil, nil
}

func (s *recordingMemoryStore) Update(context.Context, string, MemoryPatch) error { return nil }
func (s *recordingMemoryStore) Delete(context.Context, string) error              { return nil }
func (s *recordingMemoryStore) GetAll(context.Context, MemoryFilters, MemoryPage) (MemoryPageResult, error) {
	return MemoryPageResult{}, nil
}
func (s *recordingMemoryStore) History(context.Context, string) ([]MemoryHistoryEvent, error) {
	return nil, nil
}

func TestMemoryRecallServiceSearchesProjectAndUserScopes(t *testing.T) {
	store := NewLocalMemoryStore()
	_, _ = store.Add(context.Background(), AddMemoryRequest{Candidates: []MemoryCandidate{
		{Scope: ScopeUser, Kind: KindPreference, Memory: "User prefers Chinese answers.", Confidence: 0.9, UserID: "u1"},
		{Scope: ScopeProject, Kind: KindArchitectureDecision, Memory: "ContextGateway is the context entrypoint.", Confidence: 0.9, UserID: "u1", ProjectID: "proj"},
	}})
	svc := NewMemoryRecallService(MemoryRecallOptions{
		Store:       store,
		Ranker:      NewMemoryRanker(DefaultRankerOptions()),
		TokenBudget: 100,
	})
	result, err := svc.Recall(context.Background(), RecallRequest{
		UserID:    "u1",
		ProjectID: "proj",
		UserQuery: "How is context prepared? Answer in my preferred language.",
	})
	if err != nil {
		t.Fatalf("Recall() = %v", err)
	}
	if !strings.Contains(result.Rendered, "User prefers Chinese answers") {
		t.Fatalf("rendered missing user memory: %s", result.Rendered)
	}
	if !strings.Contains(result.Rendered, "ContextGateway is the context entrypoint") {
		t.Fatalf("rendered missing project memory: %s", result.Rendered)
	}
}

func TestMemoryRecallServiceAddsConfiguredAppIDToEverySearchFilter(t *testing.T) {
	store := &recordingMemoryStore{}
	svc := NewMemoryRecallService(MemoryRecallOptions{
		Store: store,
		AppID: "mewcode",
	})
	_, err := svc.Recall(context.Background(), RecallRequest{
		ProjectID: "proj",
		AgentID:   "main",
		SessionID: "session-1",
		TeamName:  "team-1",
		UserQuery: "context",
	})
	if err != nil {
		t.Fatalf("Recall() = %v", err)
	}
	if len(store.queries) == 0 {
		t.Fatal("no search queries recorded")
	}
	for _, query := range store.queries {
		if query.Filters.AppID != "mewcode" {
			t.Fatalf("filter missing app_id: %+v", query.Filters)
		}
	}
}

func TestMemoryRecallServiceSearchesUserScopeByAppIDWhenUserIDMissing(t *testing.T) {
	store := NewLocalMemoryStore()
	_, _ = store.Add(context.Background(), AddMemoryRequest{Candidates: []MemoryCandidate{
		{Scope: ScopeUser, Kind: KindPreference, Memory: "User wants the assistant to identify as xyz.", Confidence: 0.9, AppID: "mewcode"},
	}})
	svc := NewMemoryRecallService(MemoryRecallOptions{
		Store:       store,
		AppID:       "mewcode",
		TokenBudget: 100,
	})

	result, err := svc.Recall(context.Background(), RecallRequest{
		UserQuery: "identify xyz",
	})
	if err != nil {
		t.Fatalf("Recall() = %v", err)
	}
	if !strings.Contains(result.Rendered, "identify as xyz") {
		t.Fatalf("rendered missing app-scoped user memory: %s", result.Rendered)
	}
}
