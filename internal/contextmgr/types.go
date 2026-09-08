package contextmgr

import (
	"time"

	"mewcode/internal/compact"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/permissions"
	"mewcode/internal/toolresult"
)

const (
	SectionUser        = "user"
	SectionSession     = "session"
	SectionBusiness    = "business"
	SectionExecution   = "execution"
	SectionEnvironment = "environment"
	SectionBudget      = "budget"
)

type RuntimeContext struct {
	ID          string
	User        UserContext
	Session     SessionContext
	Business    BusinessContext
	Execution   ExecutionContext
	Environment EnvironmentContext
	Budget      BudgetContext
	Sources     []ContextSource
}

type UserContext struct {
	SectionName   string
	Instructions  string
	MemoryContent string
}

type SessionContext struct {
	SectionName  string
	SessionID    string
	Messages     []conversation.Message
	MessageCount int
}

type BusinessContext struct {
	SectionName       string
	ActiveSkills      map[string]string
	ToolSchemas       []map[string]any
	DeferredToolNames []string
	Checker           *permissions.Checker
}

type ExecutionContext struct {
	SectionName     string
	AgentID         string
	AgentType       string
	Protocol        string
	WorkDir         string
	Iteration       int
	MaxIterations   int
	ContextWindow   int
	MaxOutputTokens int
}

type EnvironmentContext struct {
	SectionName string
	OS          string
	Arch        string
	Shell       string
	Date        string
	IsGitRepo   bool
	GitBranch   string
	Model       string
}

type BudgetContext struct {
	SectionName      string
	UsageAnchor      compact.UsageAnchor
	CompactTracking  compact.AutoCompactTrackingState
	ReplacementState *toolresult.ContentReplacementState
}

type ContextSource struct {
	Section string
	Kind    string
	Path    string
	Name    string
	FreshAt time.Time
}

type ContextNotice struct {
	Message string
}

type EventType string

type AuditRecord struct {
	ID        string         `json:"id"`
	Time      time.Time      `json:"time"`
	Event     EventType      `json:"event"`
	AgentID   string         `json:"agent_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	ContextID string         `json:"context_id,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type PrepareRequest struct {
	Conversation      *conversation.Manager
	WorkDir           string
	SessionID         string
	Protocol          string
	AgentID           string
	AgentType         string
	Iteration         int
	MaxIterations     int
	ContextWindow     int
	MaxOutputTokens   int
	Model             string
	Client            llm.Client
	Checker           *permissions.Checker
	ToolSchemas       []map[string]any
	DeferredToolNames []string
	ActiveSkills      map[string]string
	Instructions      string
	MemoryContent     string
	Notifications     []string
	UsageAnchor       compact.UsageAnchor
	CompactTracking   compact.AutoCompactTrackingState
	ReplacementState  *toolresult.ContentReplacementState
	Recovery          *compact.RecoveryState
}

type PreparedTurn struct {
	Snapshot         RuntimeContext
	APIConversation  *conversation.Manager
	ToolSchemas      []map[string]any
	Notices          []ContextNotice
	AuditRecords     []AuditRecord
	UsageAnchorReset bool
	CompactTracking  compact.AutoCompactTrackingState
}

type AgentRef struct {
	ID      string
	Type    string
	Name    string
	WorkDir string
}

type HandoffMode string

const (
	HandoffNone    HandoffMode = "none"
	HandoffRecent  HandoffMode = "recent"
	HandoffSummary HandoffMode = "summary"
	HandoffFull    HandoffMode = "full"
	HandoffFork    HandoffMode = "fork"
)

type FilteredItem struct {
	Kind   string
	Reason string
}

type RecoveryFile struct {
	Path    string
	Content string
}

type HandoffPackage struct {
	ID            string
	FromAgent     AgentRef
	ToAgent       AgentRef
	Mode          HandoffMode
	Summary       string
	Messages      []conversation.Message
	ActiveSkills  map[string]string
	ToolSchemas   []map[string]any
	RecoveryFiles []RecoveryFile
	Filtered      []FilteredItem
	CreatedAt     time.Time
}
