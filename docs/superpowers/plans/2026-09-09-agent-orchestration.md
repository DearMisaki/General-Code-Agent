# Multi-Agent Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a unified multi-agent collaboration system that supports parent-child delegation, peer collaboration, and shared task-board coordination, with all agent-to-agent communication transported through the existing file mailbox.

**Architecture:** Introduce an explicit orchestration layer under `internal/orchestration` that owns agent registry, collaboration sessions, task board, routing policy, lifecycle status, and audit-facing events. Keep existing `internal/agents`, `internal/teams`, `internal/contextmgr`, and `internal/worktree` as compatibility primitives, then migrate `AgentTool`, team tools, and teammate runners to the orchestration API in stages.

**Tech Stack:** Go 1.25, existing `agent`, `agents`, `teams`, `contextmgr`, `conversation`, `tools`, `worktree`, and file-backed JSON storage.

**Spec:** Based on the referenced Agent Layer article: Agent Layer should contain Registry, Factory, Scheduler, Orchestrator, Lifecycle Manager, and Monitor; Agent state should be separated into Memory State, Execution State, and Context State; multi-agent collaboration should use explicit modes instead of direct ad hoc calls.

## Global Constraints

- Do not read or modify `internal/tui` unless a compile error proves a public API integration requires it.
- Preserve existing `Agent` tool call shape for one-shot sub-agent and fork behavior.
- Preserve existing `TeamCreate`, `TeamDelete`, and `SendMessage` behavior while adding task-board capabilities.
- Preserve file mailbox as the only cross-process transport.
- Do not replace `internal/contextmgr`; use it for handoff and audit where appropriate.
- Do not break prompt-cache-sensitive fork replay.
- All mailbox writes must be file-lock protected.
- All orchestration state writes must be atomic enough for multi-process teammate mode.
- Audit and monitor writes must be non-fatal.
- Tests must use `MEWCODE_TEAMS_DIR` or temp directories to avoid polluting the repository.

---

## Article Takeaways

The article's Agent Layer design maps to this project as follows:

- Agent Registry: the current `agents.AgentLoader`, builtin `SubAgentSpec`, and future team member registry.
- Agent Factory: current `AgentTool.runSync`, `runAsync`, `runFork`, and `teams.SpawnTeammate`.
- Agent Scheduler: current `TaskManager` plus teammate idle polling, but currently not unified.
- Agent Orchestrator: currently split across `AgentTool`, `teams.TeamManager`, mailbox notifications, and manual prompts.
- Lifecycle Manager: partially exists through `TaskStatus`, teammate `Active`, `Cancel`, shutdown messages, and context lifecycle.
- Agent Monitor: partially exists through `TeammateProgress`, task notifications, and context audit.

The target design is to make these responsibilities explicit without rewriting working execution primitives.

---

## Target Collaboration Modes

### Mode 1: Parent-Child Delegation

Parent-child delegation is the current one-shot sub-agent flow:

- Parent calls `Agent` with `subagent_type`.
- Child receives a task prompt and limited tools.
- Parent blocks for sync mode or receives task completion later for async mode.
- Child returns final result to parent, not to arbitrary peers.

The orchestration layer should represent this as a `CollaborationSession` with `ModeDelegation`, a parent `AgentRef`, one child `AgentRef`, and a `TaskID`.

### Mode 2: Peer Agent Collaboration

Peer collaboration is long-running teammate mode:

- A team contains multiple named members.
- Each member has its own conversation and lifecycle.
- Members communicate through `SendMessage`.
- Messages are persisted in per-agent inbox files.
- Lead receives teammate updates through the lead inbox.

The orchestration layer should represent this as `ModePeer`, where all named agents are participants and the mailbox is the shared communication bus.

### Mode 3: Shared Task Board

Shared task-board mode adds durable coordination on top of the file mailbox:

- A team has a board file.
- Tasks have owner, status, priority, dependencies, result, and timestamps.
- Agents claim tasks, update progress, block tasks, complete tasks, or hand off tasks.
- Task-board updates are also sent as mailbox notifications so idle agents can react.

The orchestration layer should represent this as `ModeTaskBoard`, where the board is the source of truth for work allocation and mailbox messages are the wake-up and notification mechanism.

---

## File Structure

- Create `internal/orchestration/types.go`: collaboration modes, agent refs, lifecycle states, task-board types, event types.
- Create `internal/orchestration/registry.go`: registry facade over agent definitions and live team members.
- Create `internal/orchestration/session.go`: collaboration session model and JSON persistence.
- Create `internal/orchestration/taskboard.go`: file-backed shared task board with locking.
- Create `internal/orchestration/mailrouter.go`: typed mailbox envelope builder and sender using `teams.FileMailBox`.
- Create `internal/orchestration/orchestrator.go`: high-level API for delegation, peer messaging, board operations, and lifecycle transitions.
- Create `internal/orchestration/monitor.go`: non-fatal event sink and summary helpers.
- Create focused tests under `internal/orchestration/*_test.go`.
- Modify `internal/teams/filemailbox.go`: add typed message IDs and optional metadata while preserving old JSON fields.
- Modify `internal/teams/tools.go`: add task-board tools and route `SendMessage` through the new mail router where available.
- Modify `internal/teams/runner.go`: inject task-board notifications into teammate turns.
- Modify `internal/agents/agent_tool.go`: create orchestration sessions for sync, async, fork, and teammate spawns.
- Modify `internal/agents/subagent.go`: keep `TaskManager` compatibility but allow orchestration task IDs to mirror status.

---

## Public Model

```go
package orchestration

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
```

---

## Storage Layout

```text
.mewcode/teams/
  <team>/
    inboxes/
      lead.json
      <member>.json
    board.json
    sessions/
      <session-id>.json
    events.jsonl
```

`inboxes/*.json` remain the cross-process message transport. `board.json` is the durable shared task board. `events.jsonl` is append-only monitoring/audit material and must not block execution if writing fails.

---

## Task 1: Add Orchestration Domain Types

**Files:**
- Create: `internal/orchestration/types.go`
- Test: `internal/orchestration/types_test.go`

**Interfaces:**
- Produces: `CollaborationMode`, `LifecycleState`, `AgentRef`, `CollaborationSession`, `BoardTaskStatus`, `BoardTask`, `BoardUpdate`, `MailEnvelope`
- Consumes: standard library `time`

- [ ] **Step 1: Write failing tests for stable constants**

Create `internal/orchestration/types_test.go`:

```go
package orchestration

import "testing"

func TestCollaborationModesAreStable(t *testing.T) {
	got := []CollaborationMode{ModeDelegation, ModePeer, ModeTaskBoard}
	want := []CollaborationMode{"delegation", "peer", "task_board"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mode[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBoardStatusesAreStable(t *testing.T) {
	statuses := []BoardTaskStatus{
		BoardTaskOpen,
		BoardTaskClaimed,
		BoardTaskInProgress,
		BoardTaskBlocked,
		BoardTaskReview,
		BoardTaskDone,
		BoardTaskFailed,
		BoardTaskCancelled,
	}
	seen := map[BoardTaskStatus]bool{}
	for _, status := range statuses {
		if status == "" {
			t.Fatal("status must not be empty")
		}
		if seen[status] {
			t.Fatalf("duplicate status: %s", status)
		}
		seen[status] = true
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration -count=1
```

Expected: FAIL because package/types do not exist.

- [ ] **Step 3: Implement types**

Create `internal/orchestration/types.go` with the public model shown above and add:

```go
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
```

- [ ] **Step 4: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/types.go internal/orchestration/types_test.go
git commit -m "feat: add orchestration domain types"
```

---

## Task 2: Add File-Backed Task Board

**Files:**
- Create: `internal/orchestration/taskboard.go`
- Test: `internal/orchestration/taskboard_test.go`

**Interfaces:**
- Consumes: `BoardTask`, `BoardTaskStatus`, `BoardUpdate`
- Produces: `TaskBoard`, `NewTaskBoard(path string) *TaskBoard`, `CreateTask`, `ClaimTask`, `UpdateTask`, `ListTasks`, `GetTask`

- [ ] **Step 1: Write failing task-board tests**

Create tests that prove:

- `CreateTask` writes `board.json`.
- `ClaimTask` succeeds only when the task is open.
- `UpdateTask` appends history and preserves owner.
- Concurrent `ClaimTask` calls result in exactly one winner.

Use temp dirs and no repository files:

```go
func TestTaskBoardCreateClaimAndComplete(t *testing.T) {
	board := NewTaskBoard(filepath.Join(t.TempDir(), "board.json"))
	task, err := board.CreateTask(BoardTask{
		Title:       "implement parser",
		Description: "write parser tests",
		Priority:    10,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if task.Status != BoardTaskOpen {
		t.Fatalf("new task status = %s", task.Status)
	}
	if err := board.ClaimTask(task.ID, "alice"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := board.UpdateTask(task.ID, "alice", BoardTaskDone, "done", "ok"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, ok, err := board.GetTask(task.ID)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.Owner != "alice" || got.Status != BoardTaskDone || got.Result != "ok" {
		t.Fatalf("unexpected task: %+v", got)
	}
}
```

- [ ] **Step 2: Implement task board locking**

Use a lock file next to `board.json`, following `teams.FileMailBox.withLock` semantics:

- lock path: `board.json.lock`
- retry up to 10 times
- remove stale lock older than 10 seconds
- read after acquiring lock
- write JSON with indent

- [ ] **Step 3: Implement task operations**

Required semantics:

- `CreateTask` assigns `task_<unix-nano>` when ID is empty.
- New tasks default to `open`.
- `ClaimTask` can claim only `open` tasks.
- `UpdateTask` requires owner match unless actor is `lead`.
- `UpdateTask` updates `UpdatedAt`, status, message-derived error/result.
- `ListTasks` returns tasks sorted by priority descending then creation time ascending.

- [ ] **Step 4: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/taskboard.go internal/orchestration/taskboard_test.go
git commit -m "feat: add file backed task board"
```

---

## Task 3: Add Typed Mail Router on Top of File Mailbox

**Files:**
- Create: `internal/orchestration/mailrouter.go`
- Test: `internal/orchestration/mailrouter_test.go`
- Modify: `internal/teams/filemailbox.go`

**Interfaces:**
- Consumes: `teams.FileMailBox`, `teams.FileMailMessage`, `MailEnvelope`
- Produces: `MailRouter`, `SendEnvelope`, `FormatEnvelopeText`

- [ ] **Step 1: Extend mailbox message compatibility**

Modify `teams.FileMailMessage` by adding optional fields only:

```go
ID       string            `json:"id,omitempty"`
Kind     string            `json:"kind,omitempty"`
TeamName string            `json:"team_name,omitempty"`
TaskID   string            `json:"task_id,omitempty"`
Metadata map[string]string `json:"metadata,omitempty"`
```

Existing JSON remains valid because the new fields are optional.

- [ ] **Step 2: Write failing router tests**

Test that `SendEnvelope` writes a mailbox message with both legacy `Text` and typed metadata:

```go
func TestMailRouterSendEnvelopeWritesTypedMailboxMessage(t *testing.T) {
	mb := teams.NewFileMailBox(filepath.Join(t.TempDir(), "inboxes"))
	router := NewMailRouter("demo", mb)
	env := MailEnvelope{
		Kind:     MailKindAssignment,
		From:     "lead",
		To:       "alice",
		Text:     "implement task",
		TaskID:   "task_1",
		Metadata: map[string]string{"priority": "10"},
	}
	if err := router.SendEnvelope(env); err != nil {
		t.Fatalf("send: %v", err)
	}
	msgs, err := mb.ReadUnread("alice")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Kind != string(MailKindAssignment) || msgs[0].TaskID != "task_1" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
}
```

- [ ] **Step 3: Implement `MailRouter`**

`NewMailRouter(teamName string, mailbox *teams.FileMailBox) *MailRouter`

`SendEnvelope(env MailEnvelope) error` should:

- assign ID if empty.
- assign `TeamName` if empty.
- assign `CreatedAt` if zero.
- call `mailbox.Send(env.To, teams.FileMailMessage{...})`.
- keep `Text` readable for old consumers.

- [ ] **Step 4: Run mailbox and orchestration tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration ./internal/teams -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/mailrouter.go internal/orchestration/mailrouter_test.go internal/teams/filemailbox.go
git commit -m "feat: add typed mailbox routing"
```

---

## Task 4: Add Collaboration Session Store

**Files:**
- Create: `internal/orchestration/session.go`
- Test: `internal/orchestration/session_test.go`

**Interfaces:**
- Consumes: `CollaborationSession`
- Produces: `SessionStore`, `CreateSession`, `UpdateState`, `GetSession`, `ListSessions`

- [ ] **Step 1: Write failing session store tests**

Tests must prove:

- creating a session writes `<session-id>.json`.
- updating lifecycle state changes `UpdatedAt`.
- listing sessions excludes invalid JSON files.

- [ ] **Step 2: Implement session persistence**

Use layout:

```text
<team-dir>/sessions/<session-id>.json
```

Use safe write:

- marshal JSON.
- write to `*.tmp`.
- rename to final path.

- [ ] **Step 3: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/orchestration/session.go internal/orchestration/session_test.go
git commit -m "feat: persist collaboration sessions"
```

---

## Task 5: Add Orchestrator Facade

**Files:**
- Create: `internal/orchestration/orchestrator.go`
- Test: `internal/orchestration/orchestrator_test.go`

**Interfaces:**
- Consumes: `TaskBoard`, `MailRouter`, `SessionStore`
- Produces: `Orchestrator`, `StartDelegation`, `StartPeerSession`, `CreateBoardTask`, `AssignBoardTask`, `CompleteBoardTask`

- [ ] **Step 1: Write failing orchestrator tests**

Test these behaviors:

- `StartDelegation` creates a `ModeDelegation` session with parent and child refs.
- `StartPeerSession` creates a `ModePeer` session with all participants.
- `AssignBoardTask` claims or assigns a task and sends an assignment envelope to the owner.
- `CompleteBoardTask` updates board task and sends a board update to lead.

- [ ] **Step 2: Implement facade**

`Orchestrator` should be a small coordination object:

```go
type Orchestrator struct {
	TeamName string
	Board    *TaskBoard
	Mail     *MailRouter
	Sessions *SessionStore
}
```

It should not create LLM clients or run agents. It only creates durable orchestration state and sends mailbox notifications.

- [ ] **Step 3: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/orchestration/orchestrator.go internal/orchestration/orchestrator_test.go
git commit -m "feat: add multi agent orchestrator facade"
```

---

## Task 6: Wire Orchestrator into Team

**Files:**
- Modify: `internal/teams/teams.go`
- Modify: `internal/teams/tools.go`
- Test: `internal/teams/teams_test.go`

**Interfaces:**
- Consumes: `orchestration.Orchestrator`
- Produces: `Team.Orchestrator`, board-aware `TeamCreate`

- [ ] **Step 1: Add orchestration field to Team**

Add:

```go
Orchestrator *orchestration.Orchestrator
```

`NewTeam` should create:

- mailbox at `<teamsBaseDir>/<team>/inboxes`
- board at `<teamsBaseDir>/<team>/board.json`
- sessions at `<teamsBaseDir>/<team>/sessions`
- mail router wrapping the same file mailbox

- [ ] **Step 2: Update tests**

Add a test proving `NewTeam("demo", ModeInProcess)` initializes non-nil `MailBox` and `Orchestrator`, and that board operations write under the same team directory.

- [ ] **Step 3: Run teams tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/teams ./internal/orchestration -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/teams/teams.go internal/teams/teams_test.go
git commit -m "feat: attach orchestrator to teams"
```

---

## Task 7: Add Shared Task Board Tools

**Files:**
- Modify: `internal/teams/tools.go`
- Test: `internal/teams/teams_test.go`
- Test: `internal/agents/tool_filter_test.go`

**Interfaces:**
- Produces tools: `TaskCreate`, `TaskList`, `TaskGet`, `TaskUpdate`, `TaskClaim`
- Consumes: `Team.Orchestrator.Board`, `Team.Orchestrator.Mail`

- [ ] **Step 1: Define tool schemas**

Add:

- `TaskCreateTool`: `team_name`, `title`, `description`, `priority`, `dependencies`.
- `TaskListTool`: `team_name`, optional `status`, optional `owner`.
- `TaskGetTool`: `team_name`, `task_id`.
- `TaskClaimTool`: `team_name`, `task_id`, `agent_name`.
- `TaskUpdateTool`: `team_name`, `task_id`, `agent_name`, `status`, optional `message`, optional `result`.

- [ ] **Step 2: Write tests**

Tests should prove:

- lead can create/list/get tasks.
- teammate can claim an open task.
- another teammate cannot claim an already claimed task.
- completing a task sends a mailbox notification to lead.

- [ ] **Step 3: Implement tools**

Each tool should look up team through `TeamManager.GetTeam`. If the team or orchestrator is nil, return a `tools.ToolResult` error. Avoid panics.

- [ ] **Step 4: Update tool filter allowlists**

Ensure team members can use:

- `TaskCreate`
- `TaskList`
- `TaskGet`
- `TaskUpdate`
- `TaskClaim`
- `SendMessage`

Keep destructive team administration tools restricted to lead unless already allowed by current policy.

- [ ] **Step 5: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/teams ./internal/agents -count=1
```

- [ ] **Step 6: Commit**

```bash
git add internal/teams/tools.go internal/teams/teams_test.go internal/agents/tool_filter_test.go
git commit -m "feat: add shared task board tools"
```

---

## Task 8: Route Agent Spawns Through Collaboration Sessions

**Files:**
- Modify: `internal/agents/agent_tool.go`
- Test: `internal/agents/agent_tool_test.go`

**Interfaces:**
- Consumes: `contextmgr.Router`, `orchestration.Orchestrator`
- Produces: session creation for sync sub-agent, async sub-agent, fork, and teammate paths

- [ ] **Step 1: Add tests for session creation**

Use a fake or temp team orchestrator and assert:

- `runSync` creates `ModeDelegation`.
- `runAsync` creates `ModeDelegation`.
- `runFork` creates `ModeDelegation` with child type `fork`.
- `runAsTeammate` creates or updates `ModePeer`.

- [ ] **Step 2: Implement integration**

In `AgentTool` paths:

- For sync/async/fork without team, use a lightweight per-workdir session store under `.mewcode/agents/sessions`.
- For teammate mode, use `team.Orchestrator`.
- Keep existing execution behavior unchanged.
- Continue using `contextmgr.NewRouter().BuildHandoff` for context handoff audit.

- [ ] **Step 3: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/agents ./internal/orchestration -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/agents/agent_tool.go internal/agents/agent_tool_test.go
git commit -m "feat: track agent spawns as collaboration sessions"
```

---

## Task 9: Inject Board Notifications into Teammate Turns

**Files:**
- Modify: `internal/teams/runner.go`
- Test: `internal/teams/runner_test.go`

**Interfaces:**
- Consumes: typed mailbox messages with `Kind == "assignment"` or `Kind == "board_update"`
- Produces: board-aware teammate prompts

- [ ] **Step 1: Write tests**

Add tests proving:

- assignment messages render with task ID, sender, and text.
- board updates render separately from free-form messages.
- shutdown messages still stop the runner.

- [ ] **Step 2: Update prompt formatting**

Modify `formatInboundAsPrompt` to group messages:

```text
You have new messages from your team:

Assignments:
- task_1 from lead: implement parser

Board updates:
- task_1 from reviewer: status=review message=ready

Messages:
- from alice: please check file X
```

Keep old messages with empty `Kind` in the `Messages` group.

- [ ] **Step 3: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/teams -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/teams/runner.go internal/teams/runner_test.go
git commit -m "feat: surface task board updates in teammate prompts"
```

---

## Task 10: Add Monitor Events

**Files:**
- Create: `internal/orchestration/monitor.go`
- Test: `internal/orchestration/monitor_test.go`
- Modify: `internal/teams/teams.go`

**Interfaces:**
- Produces: `Monitor`, `AppendEvent`, `ListEvents`, `EventType`

- [ ] **Step 1: Write tests**

Prove:

- appending writes one JSONL line.
- invalid event files do not panic list.
- append failure can be ignored by callers.

- [ ] **Step 2: Implement monitor**

Event types:

- `session.created`
- `session.state_changed`
- `task.created`
- `task.claimed`
- `task.updated`
- `mail.sent`
- `agent.spawned`
- `agent.completed`
- `agent.failed`

- [ ] **Step 3: Integrate non-fatal monitor writes**

Add monitor writes to:

- task-board operations.
- mail router sends.
- session state transitions.

All call sites must ignore monitor errors.

- [ ] **Step 4: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration ./internal/teams -count=1
```

- [ ] **Step 5: Commit**

```bash
git add internal/orchestration/monitor.go internal/orchestration/monitor_test.go internal/orchestration/taskboard.go internal/orchestration/mailrouter.go internal/orchestration/session.go
git commit -m "feat: record orchestration monitor events"
```

---

## Task 11: Update User-Facing Tool Guidance

**Files:**
- Modify: `internal/teams/tools.go`
- Modify: `internal/agents/agent_tool.go`
- Test: existing schema/description assertions if present

**Interfaces:**
- Consumes: new collaboration modes and task-board tools
- Produces: clearer model instructions for when to use delegation, peer team, or task board

- [ ] **Step 1: Update `Agent` tool description**

Clarify:

- no `team_name`: one-shot parent-child delegation.
- `team_name` + `name`: long-running peer teammate.
- when multiple teammates need shared ownership, use TeamCreate plus TaskCreate/TaskClaim/TaskUpdate.

- [ ] **Step 2: Update `TeamCreate` tool description**

Add task board workflow:

```text
1. TeamCreate
2. TaskCreate for shared work items
3. Agent with team_name/name to spawn teammates
4. TaskClaim/TaskUpdate to coordinate state
5. SendMessage for discussion and final handoff
```

- [ ] **Step 3: Run tests**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/teams ./internal/agents -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/teams/tools.go internal/agents/agent_tool.go
git commit -m "docs: clarify multi agent orchestration tools"
```

---

## Task 12: Non-TUI Regression Verification

**Files:**
- No planned code changes unless tests expose regressions outside TUI.

**Interfaces:**
- Consumes: all previous tasks
- Produces: verified non-TUI orchestration system

- [ ] **Step 1: Run focused orchestration suite**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration ./internal/teams ./internal/agents -count=1
```

- [ ] **Step 2: Run context integration suite**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr ./internal/agent ./internal/skills -count=1
```

- [ ] **Step 3: Run non-TUI packages**

Run:

```bash
env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/orchestration ./internal/contextmgr ./internal/agent ./internal/agents ./internal/compact ./internal/conversation ./internal/llm ./internal/mcp ./internal/memory ./internal/permissions ./internal/session ./internal/skills ./internal/teams ./internal/toolresult ./internal/tools ./internal/worktree -count=1
```

- [ ] **Step 4: Inspect scope**

Run:

```bash
git diff --stat
```

Expected: changes are limited to orchestration, teams, agents, tests, and docs.

- [ ] **Step 5: Commit final fixes**

```bash
git add internal/orchestration internal/teams internal/agents docs/superpowers/plans/2026-09-09-agent-orchestration.md
git commit -m "test: verify multi agent orchestration"
```

---

## Open Design Decisions

- Board ownership rule: this plan uses “owner or lead can update” as the initial rule because it is simple and testable.
- Peer discovery: this plan keeps named teammates as the first-class address space and does not introduce broadcast until direct messaging and board notifications are stable.
- Scheduler scope: this plan implements durable coordination first. Priority-based automatic dispatch can be added after board operations are stable.
- Transport scope: mailbox remains JSON-file based. No sockets, database, or message queue are introduced in this phase.

## Success Criteria

- Parent-child delegation, peer collaboration, and task-board coordination all create explicit orchestration sessions.
- All agent-to-agent communication still flows through file mailboxes.
- Shared task-board state is durable across teammate processes.
- Teammates see assignments and board updates as turn-start prompts.
- Existing one-shot Agent and fork behavior remains compatible.
- Existing TeamCreate/SendMessage workflow remains compatible.
- Non-TUI tests pass or failures are documented as pre-existing/environmental.
