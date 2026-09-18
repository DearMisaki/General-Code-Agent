package memory

import (
	"fmt"
	"strings"
	"time"
)

type MemoryScope string

const (
	ScopeUser    MemoryScope = "user"
	ScopeProject MemoryScope = "project"
	ScopeAgent   MemoryScope = "agent"
	ScopeTeam    MemoryScope = "team"
	ScopeSession MemoryScope = "session"
)

var memoryScopes = []MemoryScope{ScopeUser, ScopeProject, ScopeAgent, ScopeTeam, ScopeSession}

func ParseMemoryScope(raw string) (MemoryScope, bool) {
	for _, scope := range memoryScopes {
		if string(scope) == raw {
			return scope, true
		}
	}
	return "", false
}

type MemoryKind string

const (
	KindPreference           MemoryKind = "preference"
	KindConstraint           MemoryKind = "constraint"
	KindProjectFact          MemoryKind = "project_fact"
	KindArchitectureDecision MemoryKind = "architecture_decision"
	KindFeedback             MemoryKind = "feedback"
	KindToolGotcha           MemoryKind = "tool_gotcha"
	KindAgentExperience      MemoryKind = "agent_experience"
	KindTeamState            MemoryKind = "team_state"
	KindTaskSummary          MemoryKind = "task_summary"
	KindRelationship         MemoryKind = "relationship"
	KindEvent                MemoryKind = "event"
)

var memoryKinds = []MemoryKind{
	KindPreference,
	KindConstraint,
	KindProjectFact,
	KindArchitectureDecision,
	KindFeedback,
	KindToolGotcha,
	KindAgentExperience,
	KindTeamState,
	KindTaskSummary,
	KindRelationship,
	KindEvent,
}

func ParseMemoryKind(raw string) (MemoryKind, bool) {
	for _, kind := range memoryKinds {
		if string(kind) == raw {
			return kind, true
		}
	}
	return "", false
}

type Sensitivity string

const (
	SensitivityPublic     Sensitivity = "public"
	SensitivityInternal   Sensitivity = "internal"
	SensitivitySensitive  Sensitivity = "sensitive"
	SensitivityCredential Sensitivity = "credential"
)

type MemoryEntity struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type MemoryRelation struct {
	Subject   MemoryEntity `json:"subject"`
	Predicate string       `json:"predicate"`
	Object    MemoryEntity `json:"object"`
	Evidence  []string     `json:"evidence,omitempty"`
}

type MemoryCandidate struct {
	ID               string            `json:"id,omitempty"`
	Scope            MemoryScope       `json:"scope"`
	Kind             MemoryKind        `json:"kind"`
	Memory           string            `json:"memory"`
	Entities         []MemoryEntity    `json:"entities,omitempty"`
	Relations        []MemoryRelation  `json:"relations,omitempty"`
	SourceMessageIDs []string          `json:"source_message_ids,omitempty"`
	SourceAgentID    string            `json:"source_agent_id,omitempty"`
	UserID           string            `json:"user_id,omitempty"`
	AppID            string            `json:"app_id,omitempty"`
	ProjectID        string            `json:"project_id,omitempty"`
	AgentID          string            `json:"agent_id,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	RunID            string            `json:"run_id,omitempty"`
	TeamName         string            `json:"team_name,omitempty"`
	TaskID           string            `json:"task_id,omitempty"`
	Confidence       float64           `json:"confidence"`
	Sensitivity      Sensitivity       `json:"sensitivity,omitempty"`
	ValidFrom        time.Time         `json:"valid_from,omitempty"`
	ValidUntil       *time.Time        `json:"valid_until,omitempty"`
	TTL              *time.Duration    `json:"ttl,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

func (c MemoryCandidate) Validate() error {
	if _, ok := ParseMemoryScope(string(c.Scope)); !ok {
		return fmt.Errorf("invalid memory scope %q", c.Scope)
	}
	if _, ok := ParseMemoryKind(string(c.Kind)); !ok {
		return fmt.Errorf("invalid memory kind %q", c.Kind)
	}
	if strings.TrimSpace(c.Memory) == "" {
		return fmt.Errorf("memory text is required")
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	return nil
}

type MemoryBlock struct {
	ID        string
	Scope     MemoryScope
	Kind      MemoryKind
	Text      string
	Score     float64
	Reason    string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
	Freshness string
	TokenCost int
	Metadata  map[string]string
}
