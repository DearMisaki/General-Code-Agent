package orchestration

import "time"

type CollaborationMode string

const (
	ModeDelegation CollaborationMode = "delegation"
	ModePeer       CollaborationMode = "peer"
	ModeTaskBoard  CollaborationMode = "task_board"
)

type LifecycleState string

const (
	StateCreated   LifecycleState = "created"
	StateReady     LifecycleState = "ready"
	StateRunning   LifecycleState = "running"
	StatePaused    LifecycleState = "paused"
	StateCompleted LifecycleState = "completed"
	StateFailed    LifecycleState = "failed"
	StateArchived  LifecycleState = "archived"
)

type AgentRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	WorkDir string `json:"work_dir,omitempty"`
}

type CollaborationSession struct {
	ID           string            `json:"id"`
	TeamName     string            `json:"team_name,omitempty"`
	Mode         CollaborationMode `json:"mode"`
	Parent       AgentRef          `json:"parent"`
	Participants []AgentRef        `json:"participants"`
	State        LifecycleState    `json:"state"`
	TaskBoardID  string            `json:"task_board_id,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type BoardTaskStatus string

const (
	BoardTaskOpen       BoardTaskStatus = "open"
	BoardTaskClaimed    BoardTaskStatus = "claimed"
	BoardTaskInProgress BoardTaskStatus = "in_progress"
	BoardTaskBlocked    BoardTaskStatus = "blocked"
	BoardTaskReview     BoardTaskStatus = "review"
	BoardTaskDone       BoardTaskStatus = "done"
	BoardTaskFailed     BoardTaskStatus = "failed"
	BoardTaskCancelled  BoardTaskStatus = "cancelled"
)

type BoardTask struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       BoardTaskStatus   `json:"status"`
	Owner        string            `json:"owner,omitempty"`
	Priority     int               `json:"priority"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Result       string            `json:"result,omitempty"`
	Error        string            `json:"error,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type BoardUpdate struct {
	TaskID    string            `json:"task_id"`
	Actor     string            `json:"actor"`
	From      BoardTaskStatus   `json:"from,omitempty"`
	To        BoardTaskStatus   `json:"to"`
	Message   string            `json:"message,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

type MailKind string

const (
	MailKindMessage     MailKind = "message"
	MailKindAssignment  MailKind = "assignment"
	MailKindBoardUpdate MailKind = "board_update"
	MailKindIdle        MailKind = "idle"
	MailKindShutdown    MailKind = "shutdown"
)

type MailEnvelope struct {
	ID        string            `json:"id"`
	Kind      MailKind          `json:"kind"`
	From      string            `json:"from"`
	To        string            `json:"to"`
	Text      string            `json:"text"`
	TeamName  string            `json:"team_name,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}
