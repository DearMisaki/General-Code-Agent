package memory

import (
	"context"
	"time"
)

type MemoryStore interface {
	Add(ctx context.Context, req AddMemoryRequest) ([]StoredMemory, error)
	Search(ctx context.Context, req SearchMemoryRequest) ([]MemorySearchResult, error)
	Update(ctx context.Context, memoryID string, patch MemoryPatch) error
	Delete(ctx context.Context, memoryID string) error
	GetAll(ctx context.Context, filters MemoryFilters, page MemoryPage) (MemoryPageResult, error)
	History(ctx context.Context, memoryID string) ([]MemoryHistoryEvent, error)
}

type AddMemoryRequest struct {
	Candidates []MemoryCandidate
}

type StoredMemory struct {
	ID               string
	Scope            MemoryScope
	Kind             MemoryKind
	Memory           string
	UserID           string
	AppID            string
	ProjectID        string
	AgentID          string
	SessionID        string
	RunID            string
	TeamName         string
	TaskID           string
	Confidence       float64
	SourceMessageIDs []string
	Metadata         map[string]string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SearchMemoryRequest struct {
	Query     string
	Filters   MemoryFilters
	TopK      int
	Threshold float64
	Rerank    bool
}

type MemorySearchResult struct {
	Memory StoredMemory
	Score  float64
	Reason string
}

type MemoryFilters struct {
	UserID    string
	AppID     string
	ProjectID string
	AgentID   string
	SessionID string
	RunID     string
	TeamName  string
	TaskID    string
	Scope     MemoryScope
	Kind      MemoryKind
}

type MemoryPatch struct {
	Memory   string
	Metadata map[string]string
}

type MemoryPage struct {
	Page     int
	PageSize int
}

type MemoryPageResult struct {
	Count   int
	Next    string
	Prev    string
	Results []StoredMemory
}

type MemoryHistoryEvent struct {
	MemoryID  string
	Operation string
	At        time.Time
	Metadata  map[string]string
}
