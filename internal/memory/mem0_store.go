package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Mem0StoreOptions struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type Mem0MemoryStore struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewMem0MemoryStore(opts Mem0StoreOptions) *Mem0MemoryStore {
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.mem0.ai"
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Mem0MemoryStore{baseURL: baseURL, apiKey: opts.APIKey, client: client}
}

func (s *Mem0MemoryStore) Add(ctx context.Context, req AddMemoryRequest) ([]StoredMemory, error) {
	var out []StoredMemory
	for _, candidate := range req.Candidates {
		body := map[string]any{
			"messages": []map[string]string{{"role": "user", "content": candidate.Memory}},
			"metadata": metadataFromCandidate(candidate),
		}
		addScope(body, filtersFromCandidate(candidate))
		var parsed mem0AddResponse
		if err := s.do(ctx, http.MethodPost, "/v3/memories/add/", body, &parsed); err != nil {
			return nil, err
		}
		if len(parsed.Results) == 0 {
			out = append(out, storedFromCandidate(candidate, ""))
			continue
		}
		for _, item := range parsed.Results {
			mem := storedFromCandidate(candidate, item.ID)
			if item.Memory != "" {
				mem.Memory = item.Memory
			}
			out = append(out, mem)
		}
	}
	return out, nil
}

func (s *Mem0MemoryStore) Search(ctx context.Context, req SearchMemoryRequest) ([]MemorySearchResult, error) {
	body := map[string]any{
		"query":   req.Query,
		"filters": filtersToMap(req.Filters),
	}
	if req.TopK > 0 {
		body["top_k"] = req.TopK
	}
	if req.Threshold > 0 {
		body["threshold"] = req.Threshold
	}
	if req.Rerank {
		body["rerank"] = true
	}
	var parsed mem0SearchResponse
	if err := s.do(ctx, http.MethodPost, "/v3/memories/search/", body, &parsed); err != nil {
		return nil, err
	}
	out := make([]MemorySearchResult, 0, len(parsed.Results))
	for _, item := range parsed.Results {
		out = append(out, MemorySearchResult{Memory: storedFromMem0(item, req.Filters), Score: item.Score})
	}
	return out, nil
}

func (s *Mem0MemoryStore) Update(ctx context.Context, memoryID string, patch MemoryPatch) error {
	body := map[string]any{}
	if patch.Memory != "" {
		body["memory"] = patch.Memory
	}
	if len(patch.Metadata) > 0 {
		body["metadata"] = patch.Metadata
	}
	return s.do(ctx, http.MethodPut, "/v1/memories/"+memoryID+"/", body, nil)
}

func (s *Mem0MemoryStore) Delete(ctx context.Context, memoryID string) error {
	return s.do(ctx, http.MethodDelete, "/v1/memories/"+memoryID+"/", nil, nil)
}

func (s *Mem0MemoryStore) GetAll(ctx context.Context, filters MemoryFilters, page MemoryPage) (MemoryPageResult, error) {
	body := map[string]any{"filters": filtersToMap(filters)}
	var parsed mem0GetAllResponse
	if err := s.do(ctx, http.MethodPost, "/v3/memories/", body, &parsed); err != nil {
		return MemoryPageResult{}, err
	}
	out := MemoryPageResult{Count: parsed.Count, Next: parsed.Next, Prev: parsed.Previous}
	for _, item := range parsed.Results {
		out.Results = append(out.Results, storedFromMem0(item, filters))
	}
	return out, nil
}

func (s *Mem0MemoryStore) History(_ context.Context, memoryID string) ([]MemoryHistoryEvent, error) {
	return []MemoryHistoryEvent{{MemoryID: memoryID, Operation: "history_unavailable", At: time.Now().UTC()}}, nil
}

func (s *Mem0MemoryStore) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token "+s.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		excerpt := string(data)
		if len(excerpt) > 512 {
			excerpt = excerpt[:512]
		}
		return fmt.Errorf("mem0 %s %s failed: status %d: %s", method, path, resp.StatusCode, excerpt)
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

type mem0SearchResponse struct {
	Results []mem0Memory `json:"results"`
}

type mem0AddResponse struct {
	Results []mem0Memory `json:"results"`
}

type mem0GetAllResponse struct {
	Count    int          `json:"count"`
	Next     string       `json:"next"`
	Previous string       `json:"previous"`
	Results  []mem0Memory `json:"results"`
}

type mem0Memory struct {
	ID        string            `json:"id"`
	Memory    string            `json:"memory"`
	Score     float64           `json:"score"`
	UserID    string            `json:"user_id"`
	AgentID   string            `json:"agent_id"`
	AppID     string            `json:"app_id"`
	RunID     string            `json:"run_id"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func storedFromMem0(item mem0Memory, fallback MemoryFilters) StoredMemory {
	kind, _ := ParseMemoryKind(item.Metadata["kind"])
	scope, _ := ParseMemoryScope(item.Metadata["scope"])
	if scope == "" {
		scope = fallback.Scope
	}
	return StoredMemory{
		ID:        item.ID,
		Scope:     scope,
		Kind:      kind,
		Memory:    item.Memory,
		UserID:    firstNonEmpty(item.UserID, fallback.UserID),
		AppID:     firstNonEmpty(item.AppID, fallback.AppID),
		ProjectID: item.Metadata["project_id"],
		AgentID:   firstNonEmpty(item.AgentID, fallback.AgentID),
		RunID:     firstNonEmpty(item.RunID, fallback.RunID),
		TeamName:  item.Metadata["team_name"],
		TaskID:    item.Metadata["task_id"],
		Metadata:  cloneStringMap(item.Metadata),
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func metadataFromCandidate(c MemoryCandidate) map[string]string {
	meta := cloneStringMap(c.Metadata)
	if meta == nil {
		meta = map[string]string{}
	}
	meta["scope"] = string(c.Scope)
	meta["kind"] = string(c.Kind)
	meta["project_id"] = c.ProjectID
	meta["team_name"] = c.TeamName
	meta["task_id"] = c.TaskID
	meta["source_agent_id"] = c.SourceAgentID
	meta["confidence"] = fmt.Sprintf("%.4f", c.Confidence)
	return meta
}

func storedFromCandidate(c MemoryCandidate, id string) StoredMemory {
	now := time.Now().UTC()
	return StoredMemory{
		ID:               id,
		Scope:            c.Scope,
		Kind:             c.Kind,
		Memory:           c.Memory,
		UserID:           c.UserID,
		AppID:            c.AppID,
		ProjectID:        c.ProjectID,
		AgentID:          c.AgentID,
		SessionID:        c.SessionID,
		RunID:            c.RunID,
		TeamName:         c.TeamName,
		TaskID:           c.TaskID,
		Confidence:       c.Confidence,
		SourceMessageIDs: append([]string(nil), c.SourceMessageIDs...),
		Metadata:         metadataFromCandidate(c),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func filtersToMap(filters MemoryFilters) map[string]any {
	out := map[string]any{}
	if filters.UserID != "" {
		out["user_id"] = filters.UserID
	}
	if filters.AppID != "" {
		out["app_id"] = filters.AppID
	}
	if filters.AgentID != "" {
		out["agent_id"] = filters.AgentID
	}
	if filters.RunID != "" {
		out["run_id"] = filters.RunID
	}
	metadata := map[string]string{}
	if filters.ProjectID != "" {
		metadata["project_id"] = filters.ProjectID
	}
	if filters.TeamName != "" {
		metadata["team_name"] = filters.TeamName
	}
	if filters.TaskID != "" {
		metadata["task_id"] = filters.TaskID
	}
	if filters.Scope != "" {
		metadata["scope"] = string(filters.Scope)
	}
	if filters.Kind != "" {
		metadata["kind"] = string(filters.Kind)
	}
	if len(metadata) > 0 {
		out["metadata"] = metadata
	}
	return out
}

func addScope(body map[string]any, filters MemoryFilters) {
	if filters.UserID != "" {
		body["user_id"] = filters.UserID
	}
	if filters.AppID != "" {
		body["app_id"] = filters.AppID
	}
	if filters.AgentID != "" {
		body["agent_id"] = filters.AgentID
	}
	if filters.RunID != "" {
		body["run_id"] = filters.RunID
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
