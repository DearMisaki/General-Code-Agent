package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type LocalMemoryStore struct {
	mu       sync.Mutex
	nextID   int
	memories map[string]StoredMemory
	history  map[string][]MemoryHistoryEvent
}

func NewLocalMemoryStore() *LocalMemoryStore {
	return &LocalMemoryStore{
		memories: make(map[string]StoredMemory),
		history:  make(map[string][]MemoryHistoryEvent),
	}
}

func (s *LocalMemoryStore) Add(_ context.Context, req AddMemoryRequest) ([]StoredMemory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	out := make([]StoredMemory, 0, len(req.Candidates))
	for _, c := range req.Candidates {
		if err := c.Validate(); err != nil {
			return nil, err
		}
		s.nextID++
		id := c.ID
		if id == "" {
			id = fmt.Sprintf("local_mem_%d", s.nextID)
		}
		mem := StoredMemory{
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
			Metadata:         cloneStringMap(c.Metadata),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.memories[id] = mem
		s.history[id] = append(s.history[id], MemoryHistoryEvent{MemoryID: id, Operation: "add", At: now})
		out = append(out, mem)
	}
	return out, nil
}

func (s *LocalMemoryStore) Search(_ context.Context, req SearchMemoryRequest) ([]MemorySearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}
	var results []MemorySearchResult
	for _, mem := range s.memories {
		if !matchesMemoryFilters(mem, req.Filters) {
			continue
		}
		score := localScore(req.Query, mem.Memory)
		if strings.EqualFold(strings.TrimSpace(req.Query), strings.TrimSpace(mem.Memory)) {
			score = 1
		}
		if req.Threshold > 0 && score < req.Threshold {
			continue
		}
		if score == 0 && strings.TrimSpace(req.Query) != "" {
			continue
		}
		results = append(results, MemorySearchResult{Memory: mem, Score: score})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Memory.UpdatedAt.After(results[j].Memory.UpdatedAt)
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (s *LocalMemoryStore) Update(_ context.Context, memoryID string, patch MemoryPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	mem, ok := s.memories[memoryID]
	if !ok {
		return fmt.Errorf("memory %q not found", memoryID)
	}
	if patch.Memory != "" {
		mem.Memory = patch.Memory
	}
	if len(patch.Metadata) > 0 {
		if mem.Metadata == nil {
			mem.Metadata = make(map[string]string)
		}
		for k, v := range patch.Metadata {
			mem.Metadata[k] = v
		}
	}
	mem.UpdatedAt = time.Now().UTC()
	s.memories[memoryID] = mem
	s.history[memoryID] = append(s.history[memoryID], MemoryHistoryEvent{MemoryID: memoryID, Operation: "update", At: mem.UpdatedAt})
	return nil
}

func (s *LocalMemoryStore) Delete(_ context.Context, memoryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.memories, memoryID)
	s.history[memoryID] = append(s.history[memoryID], MemoryHistoryEvent{MemoryID: memoryID, Operation: "delete", At: time.Now().UTC()})
	return nil
}

func (s *LocalMemoryStore) GetAll(_ context.Context, filters MemoryFilters, page MemoryPage) (MemoryPageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []StoredMemory
	for _, mem := range s.memories {
		if matchesMemoryFilters(mem, filters) {
			all = append(all, mem)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].CreatedAt.Before(all[j].CreatedAt) })
	return MemoryPageResult{Count: len(all), Results: all}, nil
}

func (s *LocalMemoryStore) History(_ context.Context, memoryID string) ([]MemoryHistoryEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]MemoryHistoryEvent(nil), s.history[memoryID]...), nil
}

func matchesMemoryFilters(mem StoredMemory, filters MemoryFilters) bool {
	if filters.UserID != "" && mem.UserID != filters.UserID {
		return false
	}
	if filters.AppID != "" && mem.AppID != filters.AppID {
		return false
	}
	if filters.ProjectID != "" && mem.ProjectID != filters.ProjectID {
		return false
	}
	if filters.AgentID != "" && mem.AgentID != filters.AgentID {
		return false
	}
	if filters.SessionID != "" && mem.SessionID != filters.SessionID {
		return false
	}
	if filters.RunID != "" && mem.RunID != filters.RunID {
		return false
	}
	if filters.TeamName != "" && mem.TeamName != filters.TeamName {
		return false
	}
	if filters.TaskID != "" && mem.TaskID != filters.TaskID {
		return false
	}
	if filters.Scope != "" && mem.Scope != filters.Scope {
		return false
	}
	if filters.Kind != "" && mem.Kind != filters.Kind {
		return false
	}
	return true
}

func localScore(query, memory string) float64 {
	q := strings.Fields(strings.ToLower(query))
	m := strings.ToLower(memory)
	var hits int
	seen := make(map[string]struct{}, len(q))
	for _, term := range q {
		term = strings.Trim(term, ".,:;!?\"'`()[]{}")
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		if strings.Contains(m, term) {
			hits++
		}
	}
	if len(seen) == 0 {
		return 0
	}
	return float64(hits) / float64(len(seen))
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
