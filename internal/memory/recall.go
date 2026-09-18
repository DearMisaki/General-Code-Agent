package memory

import (
	"context"
	"strings"
)

type RecallRequest struct {
	UserID          string
	AppID           string
	UserQuery       string
	ActivePlan      string
	RecentTools     []string
	AgentID         string
	AgentRole       string
	SessionID       string
	RunID           string
	TeamName        string
	TaskID          string
	ProjectID       string
	TokenBudget     int
	AlreadySurfaced map[string]struct{}
}

type RecallResult struct {
	Blocks   []MemoryBlock
	Rendered string
}

type MemoryRecallOptions struct {
	Store       MemoryStore
	Ranker      *MemoryRanker
	AppID       string
	TokenBudget int
	TopK        int
	Threshold   float64
	Rerank      bool
	Audit       *MemoryAuditWriter
}

type MemoryRecallService struct {
	opts MemoryRecallOptions
}

func NewMemoryRecallService(opts MemoryRecallOptions) *MemoryRecallService {
	if opts.Ranker == nil {
		opts.Ranker = NewMemoryRanker(DefaultRankerOptions())
	}
	if opts.TokenBudget == 0 {
		opts.TokenBudget = 2500
	}
	if opts.TopK == 0 {
		opts.TopK = 12
	}
	return &MemoryRecallService{opts: opts}
}

func (s *MemoryRecallService) Recall(ctx context.Context, req RecallRequest) (RecallResult, error) {
	if s == nil || s.opts.Store == nil {
		return RecallResult{}, nil
	}
	if req.AppID == "" {
		req.AppID = s.opts.AppID
	}
	_ = s.opts.Audit.Append(MemoryAuditRecord{
		Event:     EventMemoryRecallStarted,
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		TeamName:  req.TeamName,
		TaskID:    req.TaskID,
		Summary:   "started memory recall",
	})
	query := buildRecallQuery(req)
	var all []MemorySearchResult
	for _, filters := range recallFilters(req) {
		results, err := s.opts.Store.Search(ctx, SearchMemoryRequest{
			Query:     query,
			Filters:   filters,
			TopK:      s.opts.TopK,
			Threshold: s.opts.Threshold,
			Rerank:    s.opts.Rerank,
		})
		if err != nil {
			_ = s.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryFallbackUsed, Summary: err.Error()})
			return RecallResult{}, err
		}
		all = append(all, results...)
	}
	all = dedupeSearchResults(all, req.AlreadySurfaced)
	blocks := s.opts.Ranker.Rank(all, req)
	budget := req.TokenBudget
	if budget == 0 {
		budget = s.opts.TokenBudget
	}
	blocks = NewMemoryBudgetLimiter(budget).Limit(blocks)
	rendered := RenderMemoryBlocks(blocks)
	_ = s.opts.Audit.Append(MemoryAuditRecord{
		Event:     EventMemoryRecallCompleted,
		SessionID: req.SessionID,
		AgentID:   req.AgentID,
		TeamName:  req.TeamName,
		TaskID:    req.TaskID,
		Summary:   "completed memory recall",
		Metadata:  map[string]string{"count": intString(len(blocks))},
	})
	if rendered != "" {
		_ = s.opts.Audit.Append(MemoryAuditRecord{Event: EventMemoryInjected, SessionID: req.SessionID, AgentID: req.AgentID, Summary: "rendered memory for context"})
	}
	return RecallResult{Blocks: blocks, Rendered: rendered}, nil
}

func buildRecallQuery(req RecallRequest) string {
	parts := []string{req.UserQuery, req.ActivePlan, req.AgentRole, req.TeamName, req.TaskID}
	if len(req.RecentTools) > 0 {
		parts = append(parts, "recent tools: "+strings.Join(req.RecentTools, ", "))
	}
	return strings.Join(nonEmpty(parts), "\n")
}

func recallFilters(req RecallRequest) []MemoryFilters {
	var filters []MemoryFilters
	if req.UserID != "" || req.AppID != "" {
		filters = append(filters, MemoryFilters{UserID: req.UserID, AppID: req.AppID, RunID: req.RunID, Scope: ScopeUser})
	}
	if req.ProjectID != "" {
		filters = append(filters, MemoryFilters{UserID: req.UserID, AppID: req.AppID, ProjectID: req.ProjectID, RunID: req.RunID, Scope: ScopeProject})
	}
	if req.AgentID != "" {
		filters = append(filters, MemoryFilters{UserID: req.UserID, AppID: req.AppID, ProjectID: req.ProjectID, AgentID: req.AgentID, RunID: req.RunID, Scope: ScopeAgent})
	}
	if req.TeamName != "" {
		filters = append(filters, MemoryFilters{UserID: req.UserID, AppID: req.AppID, ProjectID: req.ProjectID, RunID: req.RunID, TeamName: req.TeamName, Scope: ScopeTeam})
	}
	if req.SessionID != "" {
		filters = append(filters, MemoryFilters{UserID: req.UserID, AppID: req.AppID, ProjectID: req.ProjectID, SessionID: req.SessionID, RunID: req.RunID, Scope: ScopeSession})
	}
	return filters
}

func dedupeSearchResults(results []MemorySearchResult, already map[string]struct{}) []MemorySearchResult {
	seen := make(map[string]struct{}, len(results))
	var out []MemorySearchResult
	for _, result := range results {
		id := result.Memory.ID
		if id == "" {
			id = result.Memory.Memory
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if _, ok := already[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, result)
	}
	return out
}

func nonEmpty(in []string) []string {
	var out []string
	for _, value := range in {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func intString(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
