package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMem0MemoryStoreSearchUsesV3EndpointAndFilters(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"results":[{"id":"m1","memory":"User prefers Chinese.","score":0.9,"metadata":{"kind":"preference","scope":"user"}}]}`))
	}))
	defer server.Close()

	store := NewMem0MemoryStore(Mem0StoreOptions{BaseURL: server.URL, APIKey: "key"})
	results, err := store.Search(context.Background(), SearchMemoryRequest{
		Query:   "language preference",
		Filters: MemoryFilters{UserID: "u1"},
		TopK:    3,
	})
	if err != nil {
		t.Fatalf("Search() = %v", err)
	}
	if gotPath != "/v3/memories/search/" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Token key" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotBody["query"] != "language preference" {
		t.Fatalf("body query = %v", gotBody["query"])
	}
	filters, _ := gotBody["filters"].(map[string]any)
	if filters["user_id"] != "u1" {
		t.Fatalf("filters = %+v, want user_id", filters)
	}
	if len(results) != 1 || results[0].Memory.ID != "m1" {
		t.Fatalf("results = %+v", results)
	}
}

func TestMem0MemoryStoreUsesAppIDForSearchAndAdd(t *testing.T) {
	var searchBody map[string]any
	var addBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/memories/search/":
			_ = json.NewDecoder(r.Body).Decode(&searchBody)
			_, _ = w.Write([]byte(`{"results":[]}`))
		case "/v3/memories/add/":
			_ = json.NewDecoder(r.Body).Decode(&addBody)
			_, _ = w.Write([]byte(`{"results":[{"id":"m1","memory":"ContextGateway is the context entrypoint.","app_id":"mewcode","metadata":{"kind":"architecture_decision","scope":"project","project_id":"proj"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	store := NewMem0MemoryStore(Mem0StoreOptions{BaseURL: server.URL, APIKey: "key"})
	_, err := store.Search(context.Background(), SearchMemoryRequest{
		Query:   "context",
		Filters: MemoryFilters{AppID: "mewcode", ProjectID: "proj", Scope: ScopeProject},
	})
	if err != nil {
		t.Fatalf("Search() = %v", err)
	}
	searchFilters, _ := searchBody["filters"].(map[string]any)
	if searchFilters["app_id"] != "mewcode" {
		t.Fatalf("search filters = %+v, want app_id", searchFilters)
	}

	_, err = store.Add(context.Background(), AddMemoryRequest{Candidates: []MemoryCandidate{{
		Scope:      ScopeProject,
		Kind:       KindArchitectureDecision,
		Memory:     "ContextGateway is the context entrypoint.",
		ProjectID:  "proj",
		AppID:      "mewcode",
		Confidence: 0.9,
	}}})
	if err != nil {
		t.Fatalf("Add() = %v", err)
	}
	if addBody["app_id"] != "mewcode" {
		t.Fatalf("add body = %+v, want app_id", addBody)
	}
}
