# Context Layer Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a first-class `internal/contextmgr` layer that owns context construction, budgeting, handoff, lifecycle, and audit while preserving current agent behavior.

**Architecture:** Add `internal/contextmgr` as the Context Layer and migrate callers to it in stages. Existing `conversation`, `compact`, and `toolresult` packages remain the lower-level compatibility primitives; the new gateway becomes the agent loop's single pre-send context entry point.

**Tech Stack:** Go 1.25, standard library, existing `conversation`, `compact`, `toolresult`, `permissions`, `prompt`, `tools`, `session`, `agents`, and `teams` packages.

**Spec:** `docs/superpowers/specs/2026-09-08-context-layer-design.md`

## Global Constraints

- Do not read or modify `internal/tui` unless a compile error proves a public API integration requires it.
- Preserve LLM adapter request formats.
- Preserve session JSONL compatibility.
- Preserve existing compact boundary behavior.
- Preserve prompt-cache-sensitive fork behavior, including thinking blocks and placeholder tool results for incomplete tool calls.
- Never create orphan `tool_result` messages.
- Keep audit write failures non-fatal.
- Do not remove `internal/compact` or `internal/toolresult` during this plan.
- Use `GOCACHE=/private/tmp/mewcode-go-cache` for Go commands in restricted sandboxes.

---

## Checkpoints

- [ ] **Checkpoint 1: Context model is explicit and stable**
  - [ ] Define `RuntimeContext` as the top-level snapshot type.
  - [ ] Define `UserContext`, `SessionContext`, `BusinessContext`, `ExecutionContext`, `EnvironmentContext`, and `BudgetContext`.
  - [ ] Define stable section constants: `user`, `session`, `business`, `execution`, `environment`, `budget`.
  - [ ] Define `PrepareRequest` and `PreparedTurn` as the gateway boundary.
  - [ ] Define handoff primitives: `HandoffPackage`, `HandoffMode`, `AgentRef`, and `FilteredItem`.
  - [ ] Add tests proving handoff modes are unique and section names are stable.
  - [ ] Commit the new type model.

- [ ] **Checkpoint 2: Context Builder can create snapshots without side effects**
  - [ ] Build snapshots from an existing `conversation.Manager`.
  - [ ] Copy conversation messages without mutating the original manager.
  - [ ] Copy active skills and tool schema slices into snapshot fields.
  - [ ] Populate execution metadata: agent ID, agent type, protocol, workdir, iteration, and token limits.
  - [ ] Populate environment metadata from `prompt.DetectEnvironment`.
  - [ ] Record context sources for memory, instructions, skills, tools, conversation, and runtime environment.
  - [ ] Add tests proving the builder does not mutate conversation state.
  - [ ] Commit the builder.

- [ ] **Checkpoint 3: Renderer preserves current reminder behavior**
  - [ ] Render long-term instructions and memory into the same semantic reminder content used today.
  - [ ] Render plan mode reminder through `prompt.BuildPlanModeReminder`.
  - [ ] Keep `Checker.PlanFilePath` synchronized when plan mode is active.
  - [ ] Render notification messages from `NotificationFn`.
  - [ ] Render active skill SOPs under an `Active Skills` section.
  - [ ] Render deferred tool names and `ToolSearch` loading instructions.
  - [ ] Add tests proving all existing reminder categories appear in rendered output.
  - [ ] Commit the renderer.

- [ ] **Checkpoint 4: Audit log records context-layer events**
  - [ ] Define audit event constants for prepare, render, tool-result budget, compact, handoff, tool exposure, recovery file reads, and recovery skills.
  - [ ] Define `AuditRecord` with agent, session, context, summary, time, and metadata fields.
  - [ ] Implement append-only JSONL writing under `.mewcode/context/audit.jsonl`.
  - [ ] Ensure empty or nil audit writers are no-ops.
  - [ ] Keep audit write failures non-fatal at call sites.
  - [ ] Add tests proving valid JSONL is appended.
  - [ ] Commit the audit log.

- [ ] **Checkpoint 5: Budgeter wraps existing size-control behavior**
  - [ ] Route tool-result replacement through `toolresult.Apply`.
  - [ ] Persist tool-result replacement records through `toolresult.AppendRecords`.
  - [ ] Route automatic compaction through `compact.ManageContext`.
  - [ ] Route manual compaction through `compact.ForceCompact`.
  - [ ] Preserve `compact.AutoCompactTrackingState` across calls.
  - [ ] Signal when usage anchors must be reset after compaction.
  - [ ] Add tests proving large tool results are spilled/replaced through the wrapper.
  - [ ] Commit the budgeter.

- [ ] **Checkpoint 6: Gateway becomes the pre-send context entry point**
  - [ ] Compose builder, renderer, budgeter, and audit writer behind `ContextGateway`.
  - [ ] Build a snapshot before rendering.
  - [ ] Render context reminders into a temporary conversation copy.
  - [ ] Apply budget management to produce `APIConversation`.
  - [ ] Return prepared tool schemas unchanged.
  - [ ] Emit audit records for prepare, tool exposure, compaction, and tool-result budgeting.
  - [ ] Add tests proving rendered context appears in the conversation sent to the LLM.
  - [ ] Commit the gateway.

- [ ] **Checkpoint 7: Recovery tracking is owned by the context layer**
  - [ ] Add `RecoveryTracker` as a facade over `compact.RecoveryState`.
  - [ ] Support file-read recovery recording.
  - [ ] Support skill-invocation recovery recording.
  - [ ] Support compact recovery attachment rendering.
  - [ ] Keep nil receiver behavior safe.
  - [ ] Add tests proving recovery attachments include files, skills, and tools.
  - [ ] Commit the recovery facade.

- [ ] **Checkpoint 8: Router owns handoff and fork conversation construction**
  - [ ] Add `Router.BuildHandoff` for `none`, `recent`, `summary`, `full`, and `fork` modes.
  - [ ] Add `Router.BuildForkedConversation` to preserve existing fork semantics.
  - [ ] Preserve thinking blocks in forked conversations.
  - [ ] Preserve completed tool-use/tool-result pairs.
  - [ ] Insert placeholder tool results for incomplete assistant tool-use messages.
  - [ ] Add tests proving no orphan `tool_result` is created.
  - [ ] Commit the router.

- [ ] **Checkpoint 9: Lifecycle facade exposes clear and compact operations**
  - [ ] Add `LifecycleManager` with budgeter and recovery dependencies.
  - [ ] Add `Clear` to reset context recovery state.
  - [ ] Add `ForceCompact` to route manual compaction through the budgeter.
  - [ ] Keep nil lifecycle behavior safe.
  - [ ] Add tests proving clear resets recovery.
  - [ ] Commit lifecycle facade.

- [ ] **Checkpoint 10: Agent loop uses ContextGateway**
  - [ ] Add `ContextGateway` field to `agent.Agent`.
  - [ ] Initialize a default gateway in `agent.New`.
  - [ ] Replace direct memory injection with gateway preparation.
  - [ ] Replace direct plan reminder injection with gateway rendering.
  - [ ] Replace direct notification injection with gateway rendering.
  - [ ] Replace direct active skill reminder injection with gateway rendering.
  - [ ] Replace direct deferred tool reminder injection with gateway rendering.
  - [ ] Replace direct `compact.ManageContext` and `toolresult.Apply` calls with gateway output.
  - [ ] Preserve compact event emission.
  - [ ] Preserve `PreSend` and `PostReceive` hook ordering.
  - [ ] Add agent tests proving gateway-rendered context reaches the LLM client.
  - [ ] Commit the agent loop migration.

- [ ] **Checkpoint 11: Recovery recording is routed through lifecycle**
  - [ ] Add `ContextLifecycle` field to `agent.Agent`.
  - [ ] Share one `compact.RecoveryState` between legacy fields and `RecoveryTracker`.
  - [ ] Route successful `ReadFile` snapshots through `ContextLifecycle.Recovery`.
  - [ ] Add `Agent.RecordSkillInvocation` as the skill recovery integration point.
  - [ ] Update `LoadSkillTool` to use the recovery host when available.
  - [ ] Keep fallback behavior for old direct `RecoveryState` use.
  - [ ] Run `contextmgr`, `agent`, and `skills` tests.
  - [ ] Commit recovery routing.

- [ ] **Checkpoint 12: Fork flow is migrated to Router**
  - [ ] Replace `agents.buildForkedConversation` call with `contextmgr.Router.BuildForkedConversation`.
  - [ ] Keep fork boilerplate owned by `internal/agents`.
  - [ ] Delete the duplicated local fork builder after equivalence tests pass.
  - [ ] Verify nested fork guard behavior remains unchanged.
  - [ ] Verify prompt-cache-sensitive replay remains byte-shape compatible.
  - [ ] Run fork-related `internal/agents` tests.
  - [ ] Commit fork router migration.

- [ ] **Checkpoint 13: Sub-agent and team spawns produce handoff audit**
  - [ ] Build `HandoffNone` packages for definition-based synchronous sub-agents.
  - [ ] Build `HandoffNone` packages for background sub-agents.
  - [ ] Build `HandoffNone` packages for team teammates.
  - [ ] Include destination agent type, name, workdir, and message count in audit metadata.
  - [ ] Keep existing sub-agent context visibility unchanged.
  - [ ] Keep team mailbox behavior unchanged.
  - [ ] Run `internal/agents` and `internal/contextmgr` tests.
  - [ ] Commit handoff audit integration.

- [ ] **Checkpoint 14: Manual lifecycle APIs are available**
  - [ ] Add `Agent.ClearContextState`.
  - [ ] Add `Agent.ForceCompactContext`.
  - [ ] Reset active skills, replacement state, recovery state, and lifecycle state on clear.
  - [ ] Route manual compact through `LifecycleManager.ForceCompact`.
  - [ ] Avoid modifying `internal/tui` under this plan unless explicitly approved.
  - [ ] Add tests for clear and nil-client force compact behavior.
  - [ ] Commit lifecycle API integration.

- [ ] **Checkpoint 15: Non-TUI regression suite is clean**
  - [ ] Run all `internal/contextmgr` tests.
  - [ ] Run core non-TUI package tests.
  - [ ] Run `go test ./...` once with temp `GOCACHE`.
  - [ ] If only `internal/tui` fails, report it without inspecting TUI internals.
  - [ ] Inspect `git diff --stat` for scope creep.
  - [ ] Commit final regression fixes.

---

## File Structure

- Create `internal/contextmgr/types.go`: public data model for runtime context, request/response types, notices, handoff, recovery, and audit event names.
- Create `internal/contextmgr/builder.go`: build structured `RuntimeContext` snapshots from agent inputs without mutating conversation.
- Create `internal/contextmgr/render.go`: render memory, plan, notifications, active skills, and deferred tools into conversation reminders.
- Create `internal/contextmgr/budgeter.go`: wrap `compact.ManageContext`, `compact.ForceCompact`, `toolresult.Apply`, and `toolresult.AppendRecords`.
- Create `internal/contextmgr/gateway.go`: orchestrate build, render, lifecycle budget, tool-result replacement, and audit for each turn.
- Create `internal/contextmgr/recovery.go`: own recovery tracking for recently read files and active skills while delegating to `compact.RecoveryState` initially.
- Create `internal/contextmgr/audit.go`: append JSONL audit records under `.mewcode/context/audit.jsonl`.
- Create `internal/contextmgr/router.go`: construct handoff packages and forked conversations.
- Create `internal/contextmgr/lifecycle.go`: clear/reset APIs and compact APIs used by `/clear`, `/compact`, and future resume wiring.
- Create focused tests under `internal/contextmgr/*_test.go`.
- Modify `internal/agent/agent.go`: replace direct pre-send context orchestration with `ContextGateway`.
- Modify `internal/agents/agent_tool.go`: use `contextmgr.Router` for forked conversation and handoff package construction.
- Modify `internal/skills/load_skill_tool.go`: record skill recovery through context manager where the host provides it.
- Modify tests in `internal/agent` and `internal/agents` only when behavior is intentionally routed through the new layer.

---

### Task 1: Context Model and Request Types

**Files:**
- Create: `internal/contextmgr/types.go`
- Test: `internal/contextmgr/types_test.go`

**Interfaces:**
- Produces: `RuntimeContext`, `UserContext`, `SessionContext`, `BusinessContext`, `ExecutionContext`, `EnvironmentContext`, `BudgetContext`, `PrepareRequest`, `PreparedTurn`, `ContextNotice`, `ContextSource`, `HandoffPackage`, `HandoffMode`, `AgentRef`, `FilteredItem`
- Consumes: `conversation.Manager`, `conversation.Message`, `permissions.Checker`, `compact.UsageAnchor`, `compact.AutoCompactTrackingState`, `toolresult.ContentReplacementState`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"testing"
)

func TestHandoffModeConstants(t *testing.T) {
	modes := []HandoffMode{HandoffNone, HandoffRecent, HandoffSummary, HandoffFull, HandoffFork}
	seen := map[HandoffMode]bool{}
	for _, mode := range modes {
		if mode == "" {
			t.Fatalf("handoff mode must not be empty")
		}
		if seen[mode] {
			t.Fatalf("duplicate handoff mode: %q", mode)
		}
		seen[mode] = true
	}
}

func TestRuntimeContextSectionsHaveStableNames(t *testing.T) {
	ctx := RuntimeContext{
		User:        UserContext{SectionName: SectionUser},
		Session:     SessionContext{SectionName: SectionSession},
		Business:    BusinessContext{SectionName: SectionBusiness},
		Execution:   ExecutionContext{SectionName: SectionExecution},
		Environment: EnvironmentContext{SectionName: SectionEnvironment},
		Budget:      BudgetContext{SectionName: SectionBudget},
	}
	if ctx.User.SectionName != "user" {
		t.Fatalf("user section = %q", ctx.User.SectionName)
	}
	if ctx.Session.SectionName != "session" {
		t.Fatalf("session section = %q", ctx.Session.SectionName)
	}
	if ctx.Business.SectionName != "business" {
		t.Fatalf("business section = %q", ctx.Business.SectionName)
	}
	if ctx.Execution.SectionName != "execution" {
		t.Fatalf("execution section = %q", ctx.Execution.SectionName)
	}
	if ctx.Environment.SectionName != "environment" {
		t.Fatalf("environment section = %q", ctx.Environment.SectionName)
	}
	if ctx.Budget.SectionName != "budget" {
		t.Fatalf("budget section = %q", ctx.Budget.SectionName)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run 'TestHandoffModeConstants|TestRuntimeContextSectionsHaveStableNames' -count=1`

Expected: FAIL because `internal/contextmgr` and the referenced types do not exist.

- [ ] **Step 3: Implement the model types**

```go
package contextmgr

import (
	"time"

	"mewcode/internal/compact"
	"mewcode/internal/conversation"
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

type PrepareRequest struct {
	Conversation     *conversation.Manager
	WorkDir          string
	SessionID        string
	Protocol         string
	AgentID          string
	AgentType        string
	Iteration        int
	MaxIterations    int
	ContextWindow    int
	MaxOutputTokens  int
	Model            string
	Checker          *permissions.Checker
	ToolSchemas      []map[string]any
	DeferredToolNames []string
	ActiveSkills     map[string]string
	Instructions     string
	MemoryContent    string
	Notifications    []string
	UsageAnchor      compact.UsageAnchor
	CompactTracking  compact.AutoCompactTrackingState
	ReplacementState *toolresult.ContentReplacementState
	Recovery         *compact.RecoveryState
}

type PreparedTurn struct {
	Snapshot         RuntimeContext
	APIConversation *conversation.Manager
	ToolSchemas     []map[string]any
	Notices         []ContextNotice
	AuditRecords    []AuditRecord
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
	ID           string
	FromAgent    AgentRef
	ToAgent      AgentRef
	Mode         HandoffMode
	Summary      string
	Messages     []conversation.Message
	ActiveSkills map[string]string
	ToolSchemas  []map[string]any
	RecoveryFiles []RecoveryFile
	Filtered     []FilteredItem
	CreatedAt    time.Time
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run 'TestHandoffModeConstants|TestRuntimeContextSectionsHaveStableNames' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/types.go internal/contextmgr/types_test.go
git commit -m "feat: add context layer model types"
```

---

### Task 2: Context Builder

**Files:**
- Create: `internal/contextmgr/builder.go`
- Test: `internal/contextmgr/builder_test.go`

**Interfaces:**
- Consumes: `PrepareRequest`
- Produces: `type Builder struct{}`, `func NewBuilder() *Builder`, `func (b *Builder) Build(req PrepareRequest) RuntimeContext`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"testing"

	"mewcode/internal/conversation"
)

func TestBuilderBuildsSnapshotWithoutMutatingConversation(t *testing.T) {
	conv := conversation.NewManager()
	conv.AddUserMessage("hello")

	builder := NewBuilder()
	snap := builder.Build(PrepareRequest{
		Conversation: conv,
		WorkDir: "/repo",
		SessionID: "session-1",
		Protocol: "anthropic",
		AgentID: "agent-1",
		AgentType: "main",
		Iteration: 2,
		ContextWindow: 200000,
		MaxOutputTokens: 8192,
		Instructions: "follow repo rules",
		MemoryContent: "prefers Chinese",
		ActiveSkills: map[string]string{"review": "read carefully"},
		ToolSchemas: []map[string]any{{"name": "ReadFile"}},
		DeferredToolNames: []string{"mcp__docs__search"},
	})

	if conv.Len() != 1 {
		t.Fatalf("builder mutated conversation, len=%d", conv.Len())
	}
	if snap.Session.SessionID != "session-1" {
		t.Fatalf("session id = %q", snap.Session.SessionID)
	}
	if snap.Session.MessageCount != 1 {
		t.Fatalf("message count = %d", snap.Session.MessageCount)
	}
	if snap.User.Instructions != "follow repo rules" {
		t.Fatalf("instructions not copied")
	}
	if snap.Business.ActiveSkills["review"] != "read carefully" {
		t.Fatalf("active skill not copied")
	}
	if snap.Business.ToolSchemas[0]["name"] != "ReadFile" {
		t.Fatalf("tool schema not copied")
	}
	if snap.Execution.WorkDir != "/repo" || snap.Execution.Iteration != 2 {
		t.Fatalf("execution context not populated")
	}
	if len(snap.Sources) == 0 {
		t.Fatalf("expected sources")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestBuilderBuildsSnapshotWithoutMutatingConversation -count=1`

Expected: FAIL because `NewBuilder` is not implemented.

- [ ] **Step 3: Implement builder**

```go
package contextmgr

import (
	"os"
	"runtime"
	"time"

	"mewcode/internal/conversation"
	"mewcode/internal/prompt"
)

type Builder struct{}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) Build(req PrepareRequest) RuntimeContext {
	now := time.Now()
	var messages []conversation.Message
	if req.Conversation != nil {
		messages = req.Conversation.GetMessages()
	}

	activeSkills := make(map[string]string, len(req.ActiveSkills))
	for k, v := range req.ActiveSkills {
		activeSkills[k] = v
	}

	env := prompt.DetectEnvironment(req.WorkDir)
	if env.OS == "" {
		env.OS = runtime.GOOS
	}
	if env.Arch == "" {
		env.Arch = runtime.GOARCH
	}
	if env.Shell == "" {
		env.Shell = os.Getenv("SHELL")
	}

	snap := RuntimeContext{
		ID: req.SessionID,
		User: UserContext{
			SectionName: SectionUser,
			Instructions: req.Instructions,
			MemoryContent: req.MemoryContent,
		},
		Session: SessionContext{
			SectionName: SectionSession,
			SessionID: req.SessionID,
			Messages: messages,
			MessageCount: len(messages),
		},
		Business: BusinessContext{
			SectionName: SectionBusiness,
			ActiveSkills: activeSkills,
			ToolSchemas: append([]map[string]any(nil), req.ToolSchemas...),
			DeferredToolNames: append([]string(nil), req.DeferredToolNames...),
			Checker: req.Checker,
		},
		Execution: ExecutionContext{
			SectionName: SectionExecution,
			AgentID: req.AgentID,
			AgentType: req.AgentType,
			Protocol: req.Protocol,
			WorkDir: req.WorkDir,
			Iteration: req.Iteration,
			MaxIterations: req.MaxIterations,
			ContextWindow: req.ContextWindow,
			MaxOutputTokens: req.MaxOutputTokens,
		},
		Environment: EnvironmentContext{
			SectionName: SectionEnvironment,
			OS: env.OS,
			Arch: env.Arch,
			Shell: env.Shell,
			Date: env.Date,
			IsGitRepo: env.IsGitRepo,
			GitBranch: env.GitBranch,
			Model: req.Model,
		},
		Budget: BudgetContext{
			SectionName: SectionBudget,
			UsageAnchor: req.UsageAnchor,
			CompactTracking: req.CompactTracking,
			ReplacementState: req.ReplacementState,
		},
	}

	if req.Instructions != "" {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionUser, Kind: "instructions", FreshAt: now})
	}
	if req.MemoryContent != "" {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionUser, Kind: "memory", FreshAt: now})
	}
	if len(activeSkills) > 0 {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionBusiness, Kind: "skill", FreshAt: now})
	}
	if len(req.ToolSchemas) > 0 || len(req.DeferredToolNames) > 0 {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionBusiness, Kind: "tool-registry", FreshAt: now})
	}
	if len(messages) > 0 {
		snap.Sources = append(snap.Sources, ContextSource{Section: SectionSession, Kind: "conversation", FreshAt: now})
	}
	snap.Sources = append(snap.Sources, ContextSource{Section: SectionEnvironment, Kind: "runtime", FreshAt: now})

	return snap
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestBuilderBuildsSnapshotWithoutMutatingConversation -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/builder.go internal/contextmgr/builder_test.go
git commit -m "feat: build runtime context snapshots"
```

---

### Task 3: Renderer for Existing System Reminders

**Files:**
- Create: `internal/contextmgr/render.go`
- Test: `internal/contextmgr/render_test.go`

**Interfaces:**
- Consumes: `RuntimeContext`, `PrepareRequest`
- Produces: `type Renderer struct{}`, `func NewRenderer() *Renderer`, `func (r *Renderer) Render(req PrepareRequest, snap RuntimeContext) []string`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"strings"
	"testing"

	"mewcode/internal/permissions"
)

func TestRendererIncludesCurrentReminderSections(t *testing.T) {
	req := PrepareRequest{
		Iteration: 1,
		WorkDir: "/repo",
		Notifications: []string{"team idle"},
		DeferredToolNames: []string{"mcp__docs__search"},
		Checker: &permissions.Checker{Mode: permissions.ModePlan},
	}
	snap := RuntimeContext{
		User: UserContext{Instructions: "repo rules", MemoryContent: "memory body"},
		Business: BusinessContext{ActiveSkills: map[string]string{"review": "review SOP"}},
	}

	out := NewRenderer().Render(req, snap)
	joined := strings.Join(out, "\n---\n")
	for _, want := range []string{"repo rules", "memory body", "team idle", "Active Skills", "review SOP", "deferred tools", "mcp__docs__search", "Plan Mode"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("rendered reminders missing %q:\n%s", want, joined)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestRendererIncludesCurrentReminderSections -count=1`

Expected: FAIL because renderer does not exist.

- [ ] **Step 3: Implement renderer**

```go
package contextmgr

import (
	"strings"

	"mewcode/internal/permissions"
	"mewcode/internal/planfile"
	"mewcode/internal/prompt"
)

type Renderer struct{}

func NewRenderer() *Renderer { return &Renderer{} }

func (r *Renderer) Render(req PrepareRequest, snap RuntimeContext) []string {
	var reminders []string

	if snap.User.Instructions != "" || snap.User.MemoryContent != "" {
		var sections []string
		if snap.User.Instructions != "" {
			sections = append(sections, "# mewcodeMd\nCodebase and user instructions are shown below. Be sure to adhere to these instructions. IMPORTANT: These instructions OVERRIDE any default behavior and you MUST follow them exactly as written.\n\n"+snap.User.Instructions)
		}
		if snap.User.MemoryContent != "" {
			sections = append(sections, "# autoMemory\n"+snap.User.MemoryContent)
		}
		reminders = append(reminders, "As you answer the user's questions, you can use the following context:\n"+
			strings.Join(sections, "\n\n")+
			"\n\n      IMPORTANT: this context may or may not be relevant to your tasks. You should not respond to this context unless it is highly relevant to your task.")
	}

	if req.Checker != nil && req.Checker.Mode == permissions.ModePlan {
		planPath := planfile.GetOrCreatePlanPath(req.WorkDir)
		req.Checker.PlanFilePath = planPath
		planExists := planfile.PlanExists(req.WorkDir)
		reminders = append(reminders, prompt.BuildPlanModeReminder(planPath, planExists, req.Iteration))
	}

	reminders = append(reminders, req.Notifications...)

	if active := renderActiveSkills(snap.Business.ActiveSkills); active != "" {
		reminders = append(reminders, active)
	}

	if len(req.DeferredToolNames) > 0 {
		reminders = append(reminders, "The following deferred tools are available via ToolSearch. Their schemas are NOT loaded - use ToolSearch with query \"select:<name>[,<name>...]\" to load tool schemas before calling them:\n"+strings.Join(req.DeferredToolNames, "\n"))
	}

	return reminders
}

func renderActiveSkills(active map[string]string) string {
	if len(active) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("# Active Skills\n\nThe following Skill SOPs are pinned to the environment context. Follow each SOP when its triggering condition applies.\n\n")
	for name, body := range active {
		sb.WriteString("## Active Skill: ")
		sb.WriteString(name)
		sb.WriteString("\n\n")
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}
	return sb.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestRendererIncludesCurrentReminderSections -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/render.go internal/contextmgr/render_test.go
git commit -m "feat: render context reminders"
```

---

### Task 4: Audit Log

**Files:**
- Create: `internal/contextmgr/audit.go`
- Test: `internal/contextmgr/audit_test.go`

**Interfaces:**
- Produces: `AuditRecord`, `AuditWriter`, `NewAuditWriter`, `AuditWriter.Append`, event constants
- Consumes: workdir string

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditWriterAppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	writer := NewAuditWriter(dir)
	err := writer.Append(AuditRecord{
		ID: "rec-1",
		Time: time.Unix(1, 0).UTC(),
		Event: EventContextPrepare,
		AgentID: "agent-1",
		SessionID: "session-1",
		ContextID: "ctx-1",
		Summary: "prepared",
		Metadata: map[string]any{"messages": float64(2)},
	})
	if err != nil {
		t.Fatalf("append audit: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".mewcode", "context", "audit.jsonl"))
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, `"event":"context.prepare"`) {
		t.Fatalf("missing event: %s", line)
	}
	if !strings.Contains(line, `"agent_id":"agent-1"`) {
		t.Fatalf("missing agent id: %s", line)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestAuditWriterAppendsJSONL -count=1`

Expected: FAIL because audit types do not exist.

- [ ] **Step 3: Implement audit writer**

```go
package contextmgr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type EventType string

const (
	EventContextPrepare          EventType = "context.prepare"
	EventContextRender           EventType = "context.render"
	EventContextToolResultBudget EventType = "context.tool_result_budget"
	EventContextCompact          EventType = "context.compact"
	EventContextHandoff          EventType = "context.handoff"
	EventContextToolExposure     EventType = "context.tool_exposure"
	EventContextRecoveryFileRead EventType = "context.recovery_file_read"
	EventContextRecoverySkill    EventType = "context.recovery_skill"
)

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

type AuditWriter struct {
	workDir string
}

func NewAuditWriter(workDir string) *AuditWriter {
	return &AuditWriter{workDir: workDir}
}

func (w *AuditWriter) Append(record AuditRecord) error {
	if w == nil || w.workDir == "" {
		return nil
	}
	if record.Time.IsZero() {
		record.Time = time.Now().UTC()
	}
	path := filepath.Join(w.workDir, ".mewcode", "context", "audit.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	_, err = f.Write([]byte("\n"))
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestAuditWriterAppendsJSONL -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/audit.go internal/contextmgr/audit_test.go
git commit -m "feat: add context audit log"
```

---

### Task 5: Budgeter Wrapper

**Files:**
- Create: `internal/contextmgr/budgeter.go`
- Test: `internal/contextmgr/budgeter_test.go`

**Interfaces:**
- Produces: `Budgeter`, `BudgetResult`, `NewBudgeter`, `PrepareBudget`
- Consumes: `compact.ManageContext`, `toolresult.Apply`, `toolresult.AppendRecords`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"context"
	"strings"
	"testing"

	"mewcode/internal/conversation"
	"mewcode/internal/toolresult"
)

func TestBudgeterAppliesToolResultReplacement(t *testing.T) {
	conv := conversation.NewManager()
	large := strings.Repeat("x", toolresult.SingleResultLimit+1)
	conv.AddAssistantFull("", nil, []conversation.ToolUseBlock{{
		ToolUseID: "tool-1",
		ToolName: "Bash",
		Arguments: map[string]any{"command": "printf"},
	}})
	conv.AddToolResultsMessage([]conversation.ToolResultBlock{{
		ToolUseID: "tool-1",
		Content: large,
	}})

	state := toolresult.New()
	result, err := NewBudgeter().PrepareBudget(context.Background(), BudgetRequest{
		Conversation: conv,
		WorkDir: t.TempDir(),
		ContextWindow: 200000,
		MaxOutputTokens: 8192,
		ReplacementState: state,
	})
	if err != nil {
		t.Fatalf("budget: %v", err)
	}
	msgs := result.APIConversation.GetMessages()
	got := msgs[1].ToolResults[0].Content
	if !strings.HasPrefix(got, "[Result of ") {
		t.Fatalf("tool result was not replaced: %q", got[:40])
	}
	if len(result.ToolResultRecords) != 1 {
		t.Fatalf("records = %d", len(result.ToolResultRecords))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestBudgeterAppliesToolResultReplacement -count=1`

Expected: FAIL because `Budgeter` is not implemented.

- [ ] **Step 3: Implement budgeter**

```go
package contextmgr

import (
	"context"

	"mewcode/internal/compact"
	"mewcode/internal/conversation"
	"mewcode/internal/llm"
	"mewcode/internal/toolresult"
)

type Budgeter struct{}

func NewBudgeter() *Budgeter { return &Budgeter{} }

type BudgetRequest struct {
	Conversation     *conversation.Manager
	Client           llm.Client
	WorkDir          string
	SessionID        string
	ContextWindow    int
	MaxOutputTokens  int
	ReplacementState *toolresult.ContentReplacementState
	Recovery         *compact.RecoveryState
	ToolSchemas      []map[string]any
	UsageAnchor      compact.UsageAnchor
	CompactTracking  compact.AutoCompactTrackingState
}

type BudgetResult struct {
	APIConversation   *conversation.Manager
	ToolResultRecords []toolresult.Record
	CompactMessage    string
	CompactTracking   compact.AutoCompactTrackingState
	UsageAnchorReset  bool
}

func (b *Budgeter) PrepareBudget(ctx context.Context, req BudgetRequest) (BudgetResult, error) {
	tracking := req.CompactTracking
	var compactMessage string
	var usageAnchorReset bool

	if req.Client != nil && req.Conversation != nil {
		msg, err := compact.ManageContext(ctx, req.Conversation, req.Client, req.WorkDir, req.SessionID, req.ContextWindow, req.MaxOutputTokens, &tracking, req.Recovery, req.ToolSchemas, req.UsageAnchor)
		if err != nil {
			return BudgetResult{CompactTracking: tracking}, err
		}
		if msg != "" {
			compactMessage = msg
			usageAnchorReset = true
		}
	}

	state := req.ReplacementState
	if state == nil {
		state = toolresult.New()
	}
	apiConv, records, err := toolresult.Apply(req.Conversation, req.WorkDir, state)
	if err != nil {
		return BudgetResult{CompactTracking: tracking}, err
	}
	if len(records) > 0 {
		_ = toolresult.AppendRecords(req.WorkDir, records)
	}

	return BudgetResult{
		APIConversation: apiConv,
		ToolResultRecords: records,
		CompactMessage: compactMessage,
		CompactTracking: tracking,
		UsageAnchorReset: usageAnchorReset,
	}, nil
}

func (b *Budgeter) ForceCompact(ctx context.Context, req BudgetRequest) (string, error) {
	if req.Client == nil {
		return "", nil
	}
	return compact.ForceCompact(ctx, req.Conversation, req.Client, req.WorkDir, req.SessionID, req.ContextWindow, req.Recovery, req.ToolSchemas)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestBudgeterAppliesToolResultReplacement -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/budgeter.go internal/contextmgr/budgeter_test.go
git commit -m "feat: wrap context budgeting"
```

---

### Task 6: Context Gateway

**Files:**
- Create: `internal/contextmgr/gateway.go`
- Test: `internal/contextmgr/gateway_test.go`

**Interfaces:**
- Produces: `ContextGateway`, `NewGateway`, `PrepareTurn`
- Consumes: `Builder`, `Renderer`, `Budgeter`, `AuditWriter`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"context"
	"strings"
	"testing"

	"mewcode/internal/conversation"
	"mewcode/internal/toolresult"
)

func TestGatewayPrepareTurnRendersAndBudgets(t *testing.T) {
	conv := conversation.NewManager()
	conv.AddUserMessage("start")

	gateway := NewGateway(GatewayOptions{
		Builder: NewBuilder(),
		Renderer: NewRenderer(),
		Budgeter: NewBudgeter(),
		Audit: NewAuditWriter(t.TempDir()),
	})
	prepared, err := gateway.PrepareTurn(context.Background(), PrepareRequest{
		Conversation: conv,
		WorkDir: t.TempDir(),
		SessionID: "session-1",
		Protocol: "anthropic",
		Iteration: 1,
		ContextWindow: 200000,
		MaxOutputTokens: 8192,
		Instructions: "repo rules",
		MemoryContent: "memory body",
		ReplacementState: toolresult.New(),
	})
	if err != nil {
		t.Fatalf("prepare turn: %v", err)
	}
	if prepared.APIConversation == nil {
		t.Fatalf("missing api conversation")
	}
	msgs := prepared.APIConversation.GetMessages()
	joined := ""
	for _, m := range msgs {
		joined += m.Content + "\n"
	}
	if !strings.Contains(joined, "repo rules") || !strings.Contains(joined, "memory body") {
		t.Fatalf("missing rendered context:\n%s", joined)
	}
	if len(prepared.AuditRecords) == 0 {
		t.Fatalf("expected audit records")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestGatewayPrepareTurnRendersAndBudgets -count=1`

Expected: FAIL because gateway does not exist.

- [ ] **Step 3: Implement gateway**

```go
package contextmgr

import (
	"context"
	"fmt"
	"time"

	"mewcode/internal/conversation"
)

type GatewayOptions struct {
	Builder  *Builder
	Renderer *Renderer
	Budgeter *Budgeter
	Audit    *AuditWriter
}

type ContextGateway struct {
	builder  *Builder
	renderer *Renderer
	budgeter *Budgeter
	audit    *AuditWriter
}

func NewGateway(opts GatewayOptions) *ContextGateway {
	g := &ContextGateway{
		builder: opts.Builder,
		renderer: opts.Renderer,
		budgeter: opts.Budgeter,
		audit: opts.Audit,
	}
	if g.builder == nil {
		g.builder = NewBuilder()
	}
	if g.renderer == nil {
		g.renderer = NewRenderer()
	}
	if g.budgeter == nil {
		g.budgeter = NewBudgeter()
	}
	return g
}

func (g *ContextGateway) PrepareTurn(ctx context.Context, req PrepareRequest) (PreparedTurn, error) {
	snap := g.builder.Build(req)

	renderConv := conversation.NewManager()
	if req.Conversation != nil {
		renderConv.AppendMessages(req.Conversation.GetMessages())
	}
	for _, reminder := range g.renderer.Render(req, snap) {
		renderConv.AddSystemReminder(reminder)
	}

	budget, err := g.budgeter.PrepareBudget(ctx, BudgetRequest{
		Conversation: renderConv,
		WorkDir: req.WorkDir,
		SessionID: req.SessionID,
		ContextWindow: req.ContextWindow,
		MaxOutputTokens: req.MaxOutputTokens,
		ReplacementState: req.ReplacementState,
		Recovery: req.Recovery,
		ToolSchemas: req.ToolSchemas,
		UsageAnchor: req.UsageAnchor,
		CompactTracking: req.CompactTracking,
	})
	if err != nil {
		return PreparedTurn{}, err
	}

	records := []AuditRecord{{
		ID: fmt.Sprintf("%d", time.Now().UnixNano()),
		Time: time.Now().UTC(),
		Event: EventContextPrepare,
		AgentID: req.AgentID,
		SessionID: req.SessionID,
		ContextID: snap.ID,
		Summary: "prepared context turn",
		Metadata: map[string]any{"messages": snap.Session.MessageCount},
	}}
	if len(req.DeferredToolNames) > 0 {
		records = append(records, AuditRecord{
			ID: fmt.Sprintf("%d-tool-exposure", time.Now().UnixNano()),
			Time: time.Now().UTC(),
			Event: EventContextToolExposure,
			AgentID: req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary: "deferred tools exposed",
			Metadata: map[string]any{"count": len(req.DeferredToolNames)},
		})
	}
	if budget.CompactMessage != "" {
		records = append(records, AuditRecord{
			ID: fmt.Sprintf("%d-compact", time.Now().UnixNano()),
			Time: time.Now().UTC(),
			Event: EventContextCompact,
			AgentID: req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary: budget.CompactMessage,
		})
	}
	if len(budget.ToolResultRecords) > 0 {
		records = append(records, AuditRecord{
			ID: fmt.Sprintf("%d-budget", time.Now().UnixNano()),
			Time: time.Now().UTC(),
			Event: EventContextToolResultBudget,
			AgentID: req.AgentID,
			SessionID: req.SessionID,
			ContextID: snap.ID,
			Summary: "tool result replacements applied",
			Metadata: map[string]any{"count": len(budget.ToolResultRecords)},
		})
	}

	for _, record := range records {
		if g.audit != nil {
			_ = g.audit.Append(record)
		}
	}

	return PreparedTurn{
		Snapshot: snap,
		APIConversation: budget.APIConversation,
		ToolSchemas: req.ToolSchemas,
		AuditRecords: records,
		UsageAnchorReset: budget.UsageAnchorReset,
		CompactTracking: budget.CompactTracking,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestGatewayPrepareTurnRendersAndBudgets -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/gateway.go internal/contextmgr/gateway_test.go
git commit -m "feat: add context gateway"
```

---

### Task 7: Recovery Facade

**Files:**
- Create: `internal/contextmgr/recovery.go`
- Test: `internal/contextmgr/recovery_test.go`

**Interfaces:**
- Produces: `RecoveryTracker`, `NewRecoveryTracker`, `RecordFileRead`, `RecordSkillInvocation`, `CompactState`
- Consumes: `compact.RecoveryState`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"strings"
	"testing"
)

func TestRecoveryTrackerBuildsCompactAttachment(t *testing.T) {
	tracker := NewRecoveryTracker()
	tracker.RecordFileRead("a.go", "package a")
	tracker.RecordSkillInvocation("review", "review SOP")

	attachment := tracker.BuildAttachment([]map[string]any{{"name": "ReadFile", "description": "Read a file\nmore"}})
	for _, want := range []string{"Recently read files", "a.go", "package a", "Active skills", "review", "Available tools", "ReadFile"} {
		if !strings.Contains(attachment, want) {
			t.Fatalf("missing %q in attachment:\n%s", want, attachment)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestRecoveryTrackerBuildsCompactAttachment -count=1`

Expected: FAIL because recovery facade does not exist.

- [ ] **Step 3: Implement recovery facade**

```go
package contextmgr

import "mewcode/internal/compact"

type RecoveryTracker struct {
	state *compact.RecoveryState
}

func NewRecoveryTracker() *RecoveryTracker {
	return &RecoveryTracker{state: compact.NewRecoveryState()}
}

func WrapRecoveryState(state *compact.RecoveryState) *RecoveryTracker {
	if state == nil {
		state = compact.NewRecoveryState()
	}
	return &RecoveryTracker{state: state}
}

func (r *RecoveryTracker) CompactState() *compact.RecoveryState {
	if r == nil {
		return nil
	}
	return r.state
}

func (r *RecoveryTracker) RecordFileRead(path, content string) {
	if r == nil || r.state == nil {
		return
	}
	r.state.RecordFileRead(path, content)
}

func (r *RecoveryTracker) RecordSkillInvocation(name, body string) {
	if r == nil || r.state == nil {
		return
	}
	r.state.RecordSkillInvocation(name, body)
}

func (r *RecoveryTracker) BuildAttachment(toolSchemas []map[string]any) string {
	if r == nil {
		return ""
	}
	return compact.BuildRecoveryAttachment(r.state, toolSchemas)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestRecoveryTrackerBuildsCompactAttachment -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/recovery.go internal/contextmgr/recovery_test.go
git commit -m "feat: add context recovery facade"
```

---

### Task 8: Router and Handoff Package

**Files:**
- Create: `internal/contextmgr/router.go`
- Test: `internal/contextmgr/router_test.go`

**Interfaces:**
- Produces: `Router`, `NewRouter`, `BuildHandoff`, `BuildForkedConversation`
- Consumes: `conversation.Manager`, `conversation.Message`, `HandoffMode`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import (
	"strings"
	"testing"

	"mewcode/internal/conversation"
)

func TestRouterForkPatchesIncompleteToolUse(t *testing.T) {
	parent := conversation.NewManager()
	parent.AddUserMessage("read")
	parent.AddAssistantFull("calling", nil, []conversation.ToolUseBlock{{
		ToolUseID: "tool-1",
		ToolName: "ReadFile",
		Arguments: map[string]any{"file_path": "a.go"},
	}})

	router := NewRouter()
	forked := router.BuildForkedConversation(parent, "continue task", "fork boilerplate")
	msgs := forked.GetMessages()
	if len(msgs) != 4 {
		t.Fatalf("messages = %d", len(msgs))
	}
	if len(msgs[2].ToolResults) != 1 {
		t.Fatalf("missing placeholder tool result")
	}
	if got := msgs[2].ToolResults[0].Content; got != "(tool execution interrupted by fork)" {
		t.Fatalf("placeholder = %q", got)
	}
	if !strings.Contains(msgs[3].Content, "continue task") {
		t.Fatalf("missing task in final message")
	}
}

func TestRouterBuildsRecentHandoff(t *testing.T) {
	parent := conversation.NewManager()
	for i := 0; i < 8; i++ {
		parent.AddUserMessage("u")
	}
	router := NewRouter()
	pkg := router.BuildHandoff(HandoffRequest{
		FromAgent: AgentRef{ID: "main"},
		ToAgent: AgentRef{ID: "worker"},
		Mode: HandoffRecent,
		Conversation: parent,
		RecentMessages: 3,
	})
	if pkg.Mode != HandoffRecent {
		t.Fatalf("mode = %q", pkg.Mode)
	}
	if len(pkg.Messages) != 3 {
		t.Fatalf("messages = %d", len(pkg.Messages))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run 'TestRouterForkPatchesIncompleteToolUse|TestRouterBuildsRecentHandoff' -count=1`

Expected: FAIL because router does not exist.

- [ ] **Step 3: Implement router**

```go
package contextmgr

import (
	"fmt"
	"time"

	"mewcode/internal/conversation"
)

type Router struct{}

func NewRouter() *Router { return &Router{} }

type HandoffRequest struct {
	FromAgent      AgentRef
	ToAgent        AgentRef
	Mode           HandoffMode
	Conversation   *conversation.Manager
	RecentMessages int
	Summary        string
	ActiveSkills   map[string]string
	ToolSchemas    []map[string]any
}

func (r *Router) BuildHandoff(req HandoffRequest) HandoffPackage {
	var messages []conversation.Message
	if req.Conversation != nil {
		all := req.Conversation.GetMessages()
		switch req.Mode {
		case HandoffNone:
			messages = nil
		case HandoffRecent:
			n := req.RecentMessages
			if n <= 0 {
				n = 5
			}
			if len(all) > n {
				messages = append(messages, all[len(all)-n:]...)
			} else {
				messages = append(messages, all...)
			}
		default:
			messages = append(messages, all...)
		}
	}
	return HandoffPackage{
		ID: fmt.Sprintf("handoff-%d", time.Now().UnixNano()),
		FromAgent: req.FromAgent,
		ToAgent: req.ToAgent,
		Mode: req.Mode,
		Summary: req.Summary,
		Messages: messages,
		ActiveSkills: copyStringMap(req.ActiveSkills),
		ToolSchemas: append([]map[string]any(nil), req.ToolSchemas...),
		CreatedAt: time.Now().UTC(),
	}
}

func (r *Router) BuildForkedConversation(parent *conversation.Manager, task, boilerplate string) *conversation.Manager {
	forked := conversation.NewManager()
	if parent != nil {
		for _, msg := range parent.GetMessages() {
			if len(msg.ToolUses) > 0 && len(msg.ToolResults) == 0 {
				forked.AddAssistantFull(msg.Content, msg.ThinkingBlocks, msg.ToolUses)
				var placeholders []conversation.ToolResultBlock
				for _, tu := range msg.ToolUses {
					placeholders = append(placeholders, conversation.ToolResultBlock{
						ToolUseID: tu.ToolUseID,
						Content: "(tool execution interrupted by fork)",
					})
				}
				forked.AddToolResultsMessage(placeholders)
				continue
			}
			if len(msg.ToolUses) > 0 {
				forked.AddAssistantFull(msg.Content, msg.ThinkingBlocks, msg.ToolUses)
				continue
			}
			if len(msg.ToolResults) > 0 {
				forked.AddToolResultsMessage(msg.ToolResults)
				continue
			}
			if msg.Role == "assistant" {
				forked.AddAssistantFull(msg.Content, msg.ThinkingBlocks, nil)
				continue
			}
			forked.AddUserMessage(msg.Content)
		}
	}
	if boilerplate != "" {
		forked.AddUserMessage(boilerplate+"\n\nYour task:\n"+task)
	} else {
		forked.AddUserMessage(task)
	}
	return forked
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run 'TestRouterForkPatchesIncompleteToolUse|TestRouterBuildsRecentHandoff' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/router.go internal/contextmgr/router_test.go
git commit -m "feat: add context handoff router"
```

---

### Task 9: Lifecycle Facade

**Files:**
- Create: `internal/contextmgr/lifecycle.go`
- Test: `internal/contextmgr/lifecycle_test.go`

**Interfaces:**
- Produces: `LifecycleManager`, `NewLifecycleManager`, `Clear`, `ForceCompact`
- Consumes: `Budgeter`, `RecoveryTracker`

- [ ] **Step 1: Write the failing tests**

```go
package contextmgr

import "testing"

func TestLifecycleClearResetsRecovery(t *testing.T) {
	lc := NewLifecycleManager(NewBudgeter(), NewRecoveryTracker())
	lc.Recovery().RecordFileRead("a.go", "package a")
	lc.Clear()
	if got := lc.Recovery().BuildAttachment(nil); got != "" {
		t.Fatalf("recovery not cleared:\n%s", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestLifecycleClearResetsRecovery -count=1`

Expected: FAIL because lifecycle facade does not exist.

- [ ] **Step 3: Implement lifecycle facade**

```go
package contextmgr

import (
	"context"
)

type LifecycleManager struct {
	budgeter *Budgeter
	recovery *RecoveryTracker
}

func NewLifecycleManager(budgeter *Budgeter, recovery *RecoveryTracker) *LifecycleManager {
	if budgeter == nil {
		budgeter = NewBudgeter()
	}
	if recovery == nil {
		recovery = NewRecoveryTracker()
	}
	return &LifecycleManager{budgeter: budgeter, recovery: recovery}
}

func (m *LifecycleManager) Recovery() *RecoveryTracker {
	if m == nil {
		return nil
	}
	return m.recovery
}

func (m *LifecycleManager) Clear() {
	if m == nil {
		return
	}
	m.recovery = NewRecoveryTracker()
}

func (m *LifecycleManager) ForceCompact(ctx context.Context, req BudgetRequest) (string, error) {
	if m == nil {
		return "", nil
	}
	return m.budgeter.ForceCompact(ctx, req)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestLifecycleClearResetsRecovery -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/contextmgr/lifecycle.go internal/contextmgr/lifecycle_test.go
git commit -m "feat: add context lifecycle facade"
```

---

### Task 10: Wire Gateway Into Agent Loop

**Files:**
- Modify: `internal/agent/agent.go`
- Test: `internal/agent/agent_test.go`

**Interfaces:**
- Consumes: `contextmgr.ContextGateway.PrepareTurn`
- Produces: `Agent.ContextGateway *contextmgr.ContextGateway`

- [ ] **Step 1: Write the failing test**

Add a test using the existing fake LLM client pattern in `internal/agent/agent_test.go`:

```go
func TestAgentUsesContextGatewayForReminders(t *testing.T) {
	client := &fakeClient{
		events: []llm.StreamEvent{
			llm.TextDelta{Text: "done"},
			llm.StreamEnd{StopReason: "end_turn"},
		},
	}
	reg := tools.NewRegistry()
	ag := agent.New(client, reg, "anthropic")
	ag.Instructions = "repo rules"
	ag.MemoryContent = "memory body"
	ag.ContextGateway = contextmgr.NewGateway(contextmgr.GatewayOptions{})

	conv := conversation.NewManager()
	conv.AddUserMessage("hello")
	for range ag.Run(context.Background(), conv) {
	}

	if client.lastConv == nil {
		t.Fatalf("client did not receive conversation")
	}
	joined := ""
	for _, msg := range client.lastConv.GetMessages() {
		joined += msg.Content + "\n"
	}
	if !strings.Contains(joined, "repo rules") || !strings.Contains(joined, "memory body") {
		t.Fatalf("context gateway reminders missing:\n%s", joined)
	}
}
```

If the local fake client is named differently, adapt the receiver field but preserve this assertion:
the conversation passed to `Client.Stream` must contain the rendered context reminders.

- [ ] **Step 2: Run test to verify it fails**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/agent -run TestAgentUsesContextGatewayForReminders -count=1`

Expected: FAIL because `Agent.ContextGateway` does not exist.

- [ ] **Step 3: Add `ContextGateway` field and default initialization**

Modify `Agent`:

```go
ContextGateway *contextmgr.ContextGateway
```

Modify `New`:

```go
ContextGateway: contextmgr.NewGateway(contextmgr.GatewayOptions{}),
```

Import:

```go
"mewcode/internal/contextmgr"
```

- [ ] **Step 4: Replace pre-send context orchestration**

Inside `Run`, replace direct calls for memory injection, plan reminder, notifications, active skills
reminder, deferred tool reminder, `compact.ManageContext`, and `toolresult.Apply` with:

```go
notifications := []string(nil)
if a.NotificationFn != nil {
	notifications = a.NotificationFn()
}

prepared, err := a.ContextGateway.PrepareTurn(ctx, contextmgr.PrepareRequest{
	Conversation: conv,
	WorkDir: a.WorkDir,
	SessionID: a.SessionID,
	Protocol: a.Protocol,
	Iteration: iteration,
	MaxIterations: a.MaxIterations,
	ContextWindow: a.ContextWindow,
	MaxOutputTokens: a.MaxOutputTokens,
	Checker: a.Checker,
	ToolSchemas: toolSchemas,
	DeferredToolNames: a.Registry.GetDeferredToolNames(),
	ActiveSkills: a.activeSkills,
	Instructions: a.Instructions,
	MemoryContent: a.MemoryContent,
	Notifications: notifications,
	UsageAnchor: usageAnchor,
	CompactTracking: a.compactTracking,
	ReplacementState: a.ReplacementState,
	Recovery: a.RecoveryState,
})
if err != nil {
	ch <- ErrorEvent{Message: err.Error()}
	return
}
if prepared.UsageAnchorReset {
	usageAnchor = compact.UsageAnchor{}
}
a.compactTracking = prepared.CompactTracking
apiConv := prepared.APIConversation
```

Keep `toolSchemas := a.currentToolSchemas()` before the prepare call.

Remove the now-unused local `apiConv, newRecords, _ := toolresult.Apply(...)` block and the direct
system reminder injections it replaces.

- [ ] **Step 5: Preserve compact event emission**

After `PrepareTurn`, emit compact notices from audit records:

```go
for _, record := range prepared.AuditRecords {
	if record.Event == contextmgr.EventContextCompact && record.Summary != "" {
		ch <- CompactEvent{Message: record.Summary}
	}
}
```

- [ ] **Step 6: Run agent tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/agent -count=1`

Expected: PASS.

- [ ] **Step 7: Run context manager tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -count=1`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/agent/agent.go internal/agent/agent_test.go
git commit -m "refactor: route agent context through gateway"
```

---

### Task 11: Route Recovery Recording Through Context Layer

**Files:**
- Modify: `internal/agent/agent.go`
- Modify: `internal/skills/load_skill_tool.go`
- Test: `internal/contextmgr/recovery_test.go`
- Test: `internal/agent/skills_test.go`

**Interfaces:**
- Consumes: `contextmgr.RecoveryTracker`
- Produces: `Agent.ContextLifecycle *contextmgr.LifecycleManager`

- [ ] **Step 1: Write failing agent recovery test**

Add to `internal/contextmgr/recovery_test.go`:

```go
func TestRecoveryTrackerNilSafe(t *testing.T) {
	var tracker *RecoveryTracker
	tracker.RecordFileRead("a.go", "package a")
	tracker.RecordSkillInvocation("review", "body")
	if got := tracker.BuildAttachment(nil); got != "" {
		t.Fatalf("nil tracker attachment = %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails or passes**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestRecoveryTrackerNilSafe -count=1`

Expected: PASS if Task 7 already implemented nil safety; otherwise FAIL and fix with the Task 7 nil checks.

- [ ] **Step 3: Add lifecycle field to `Agent`**

Modify `Agent`:

```go
ContextLifecycle *contextmgr.LifecycleManager
```

Modify `New`:

```go
RecoveryState: compact.NewRecoveryState(),
ContextLifecycle: contextmgr.NewLifecycleManager(nil, contextmgr.WrapRecoveryState(compact.NewRecoveryState())),
```

Use a single shared recovery state:

```go
recovery := compact.NewRecoveryState()
return &Agent{
	...
	RecoveryState: recovery,
	ContextLifecycle: contextmgr.NewLifecycleManager(nil, contextmgr.WrapRecoveryState(recovery)),
}
```

- [ ] **Step 4: Replace direct recovery file write**

In `executeSingleTool`, replace:

```go
a.RecoveryState.RecordFileRead(p, string(data))
```

with:

```go
if a.ContextLifecycle != nil && a.ContextLifecycle.Recovery() != nil {
	a.ContextLifecycle.Recovery().RecordFileRead(p, string(data))
} else {
	a.RecoveryState.RecordFileRead(p, string(data))
}
```

- [ ] **Step 5: Expose skill recovery through skill host**

If `LoadSkillTool` currently records directly into `compact.RecoveryState`, change its host-facing
integration to prefer a small optional interface:

```go
type RecoveryHost interface {
	RecordSkillInvocation(name, body string)
}
```

Implement on `Agent`:

```go
func (a *Agent) RecordSkillInvocation(name, body string) {
	if a.ContextLifecycle != nil && a.ContextLifecycle.Recovery() != nil {
		a.ContextLifecycle.Recovery().RecordSkillInvocation(name, body)
		return
	}
	if a.RecoveryState != nil {
		a.RecoveryState.RecordSkillInvocation(name, body)
	}
}
```

- [ ] **Step 6: Run tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr ./internal/agent ./internal/skills -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/contextmgr/recovery_test.go internal/agent/agent.go internal/skills/load_skill_tool.go internal/agent/skills_test.go
git commit -m "refactor: record recovery through context lifecycle"
```

---

### Task 12: Migrate Fork Conversation to Router

**Files:**
- Modify: `internal/agents/agent_tool.go`
- Test: `internal/agents/agent_tool_test.go`
- Test: `internal/contextmgr/router_test.go`

**Interfaces:**
- Consumes: `contextmgr.NewRouter().BuildForkedConversation`
- Produces: no new public API

- [ ] **Step 1: Add router equivalence test**

Add to `internal/contextmgr/router_test.go`:

```go
func TestBuildForkedConversationPreservesThinkingBlocks(t *testing.T) {
	parent := conversation.NewManager()
	parent.AddAssistantFull("thinking result", []conversation.ThinkingBlock{{
		Thinking: "private summary",
		Signature: "sig-1",
	}}, nil)

	forked := NewRouter().BuildForkedConversation(parent, "task", "boilerplate")
	msgs := forked.GetMessages()
	if len(msgs[0].ThinkingBlocks) != 1 {
		t.Fatalf("thinking blocks not preserved")
	}
	if msgs[0].ThinkingBlocks[0].Signature != "sig-1" {
		t.Fatalf("signature = %q", msgs[0].ThinkingBlocks[0].Signature)
	}
}
```

- [ ] **Step 2: Run router tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestBuildForkedConversationPreservesThinkingBlocks -count=1`

Expected: PASS after Task 8.

- [ ] **Step 3: Replace local fork builder**

In `internal/agents/agent_tool.go`, replace:

```go
forkedConv := buildForkedConversation(t.Conversation, prompt)
```

with:

```go
forkedConv := contextmgr.NewRouter().BuildForkedConversation(t.Conversation, prompt, forkBoilerplate)
```

Import:

```go
"mewcode/internal/contextmgr"
```

- [ ] **Step 4: Remove duplicated local function**

Delete local `buildForkedConversation` from `internal/agents/agent_tool.go` after tests prove router
coverage. Keep `forkBoilerplate` and fork constants in `agents` for now.

- [ ] **Step 5: Run agent-tool tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/agents -run 'Test.*Fork|TestAgent' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/agents/agent_tool.go internal/contextmgr/router_test.go internal/agents/agent_tool_test.go
git commit -m "refactor: use context router for forks"
```

---

### Task 13: Handoff Packages for Sub-Agents and Teams

**Files:**
- Modify: `internal/agents/agent_tool.go`
- Test: `internal/agents/agent_tool_test.go`
- Test: `internal/contextmgr/router_test.go`

**Interfaces:**
- Consumes: `contextmgr.Router.BuildHandoff`
- Produces: handoff audit records for sync, async, and teammate spawns

- [ ] **Step 1: Add handoff metadata test**

Add to `internal/contextmgr/router_test.go`:

```go
func TestHandoffCopiesSkillsAndTools(t *testing.T) {
	conv := conversation.NewManager()
	conv.AddUserMessage("hello")
	pkg := NewRouter().BuildHandoff(HandoffRequest{
		FromAgent: AgentRef{ID: "main", Type: "main"},
		ToAgent: AgentRef{ID: "worker", Type: "general-purpose"},
		Mode: HandoffRecent,
		Conversation: conv,
		RecentMessages: 1,
		ActiveSkills: map[string]string{"review": "body"},
		ToolSchemas: []map[string]any{{"name": "ReadFile"}},
	})
	if pkg.ActiveSkills["review"] != "body" {
		t.Fatalf("missing active skill")
	}
	if pkg.ToolSchemas[0]["name"] != "ReadFile" {
		t.Fatalf("missing tool schema")
	}
}
```

- [ ] **Step 2: Run test**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run TestHandoffCopiesSkillsAndTools -count=1`

Expected: PASS.

- [ ] **Step 3: Build handoff package in `runSync`**

At the start of `runSync`, after resolving `spec`, create:

```go
handoff := contextmgr.NewRouter().BuildHandoff(contextmgr.HandoffRequest{
	FromAgent: contextmgr.AgentRef{ID: "parent", Type: "main"},
	ToAgent: contextmgr.AgentRef{ID: description, Type: spec.Name},
	Mode: contextmgr.HandoffNone,
	Conversation: nil,
})
_ = handoff
```

Use `HandoffNone` because definition-based sub-agents currently do not inherit parent conversation.
This records the architectural boundary without changing behavior.

- [ ] **Step 4: Build handoff package in `runAsync`**

Create the same package with `Mode: contextmgr.HandoffNone` before `SpawnSubAgent`.

- [ ] **Step 5: Build handoff package in `runAsTeammate`**

Create the same package with:

```go
Mode: contextmgr.HandoffNone
ToAgent: contextmgr.AgentRef{ID: memberName, Type: subagentType, WorkDir: workdir}
```

- [ ] **Step 6: Add audit append helper**

Add an unexported helper in `internal/agents/agent_tool.go`:

```go
func auditHandoff(workDir string, pkg contextmgr.HandoffPackage) {
	_ = contextmgr.NewAuditWriter(workDir).Append(contextmgr.AuditRecord{
		ID: pkg.ID,
		Time: pkg.CreatedAt,
		Event: contextmgr.EventContextHandoff,
		AgentID: pkg.FromAgent.ID,
		Summary: string(pkg.Mode),
		Metadata: map[string]any{
			"to_agent": pkg.ToAgent.ID,
			"to_type": pkg.ToAgent.Type,
			"messages": len(pkg.Messages),
		},
	})
}
```

Call this helper after each package is built. Use `t.ParentChecker` only for permissions; do not
couple audit to permissions.

- [ ] **Step 7: Run tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/agents ./internal/contextmgr -count=1`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/agents/agent_tool.go internal/agents/agent_tool_test.go internal/contextmgr/router_test.go
git commit -m "feat: audit subagent handoffs"
```

---

### Task 14: Manual Compact and Clear Lifecycle Integration

**Files:**
- Modify: `internal/agent/agent.go`
- Modify: command handlers outside `internal/tui` only if available
- Test: `internal/contextmgr/lifecycle_test.go`
- Test: `internal/agent/agent_test.go`

**Interfaces:**
- Consumes: `LifecycleManager.Clear`, `LifecycleManager.ForceCompact`
- Produces: context lifecycle integration points

- [ ] **Step 1: Add lifecycle force-compact nil-client test**

Add to `internal/contextmgr/lifecycle_test.go`:

```go
func TestLifecycleForceCompactWithNilClientIsNoop(t *testing.T) {
	lc := NewLifecycleManager(NewBudgeter(), NewRecoveryTracker())
	msg, err := lc.ForceCompact(context.Background(), BudgetRequest{})
	if err != nil {
		t.Fatalf("force compact: %v", err)
	}
	if msg != "" {
		t.Fatalf("message = %q", msg)
	}
}
```

- [ ] **Step 2: Run lifecycle tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -run 'TestLifecycle' -count=1`

Expected: PASS.

- [ ] **Step 3: Add Agent clear API**

Add to `internal/agent/agent.go`:

```go
func (a *Agent) ClearContextState() {
	a.ClearActiveSkills()
	a.ReplacementState = toolresult.New()
	a.RecoveryState = compact.NewRecoveryState()
	a.ContextLifecycle = contextmgr.NewLifecycleManager(nil, contextmgr.WrapRecoveryState(a.RecoveryState))
}
```

- [ ] **Step 4: Add Agent manual compact API**

Add:

```go
func (a *Agent) ForceCompactContext(ctx context.Context, conv *conversation.Manager) (string, error) {
	if a.ContextLifecycle == nil {
		a.ContextLifecycle = contextmgr.NewLifecycleManager(nil, contextmgr.WrapRecoveryState(a.RecoveryState))
	}
	return a.ContextLifecycle.ForceCompact(ctx, contextmgr.BudgetRequest{
		Conversation: conv,
		Client: a.Client,
		WorkDir: a.WorkDir,
		SessionID: a.SessionID,
		ContextWindow: a.ContextWindow,
		MaxOutputTokens: a.MaxOutputTokens,
		Recovery: a.RecoveryState,
		ToolSchemas: a.currentToolSchemas(),
	})
}
```

- [ ] **Step 5: Leave TUI command wiring for a follow-up if it requires reading `internal/tui`**

Because this plan's global constraint says not to read or modify `internal/tui`, stop at agent-level
APIs unless non-TUI command dispatch code exists. Document in the final implementation summary that
TUI `/clear` and `/compact` can be switched to these APIs later.

- [ ] **Step 6: Run tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr ./internal/agent -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/contextmgr/lifecycle_test.go internal/agent/agent.go internal/agent/agent_test.go
git commit -m "feat: expose context lifecycle APIs"
```

---

### Task 15: Package-Level Regression Run

**Files:**
- Modify: only files required to fix compile or behavior regressions from prior tasks
- Test: package tests excluding direct TUI investigation

**Interfaces:**
- Consumes: all previous task outputs
- Produces: passing non-TUI test suite

- [ ] **Step 1: Run context package tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/contextmgr -count=1`

Expected: PASS.

- [ ] **Step 2: Run core non-TUI package tests**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./internal/agent ./internal/agents ./internal/compact ./internal/conversation ./internal/llm ./internal/mcp ./internal/memory ./internal/permissions ./internal/session ./internal/skills ./internal/teams ./internal/toolresult ./internal/tools ./internal/worktree -count=1`

Expected: PASS.

- [ ] **Step 3: Run full package test once**

Run: `env GOCACHE=/private/tmp/mewcode-go-cache go test ./... -count=1`

Expected: PASS. If this fails only in `internal/tui`, report the failure and do not inspect or edit TUI unless the user explicitly approves lifting the constraint.

- [ ] **Step 4: Inspect git diff**

Run: `git diff --stat`

Expected: changes are limited to `internal/contextmgr`, `internal/agent`, `internal/agents`, `internal/skills`, and docs.

- [ ] **Step 5: Commit final regression fixes**

```bash
git add internal/contextmgr internal/agent internal/agents internal/skills docs/superpowers/specs/2026-09-08-context-layer-design.md docs/superpowers/plans/2026-09-08-context-layer.md
git commit -m "test: verify context layer refactor"
```

---

## Self-Review

**Spec coverage:** This plan covers the spec's builder, gateway, budgeter, recovery, router, lifecycle, and audit requirements. Store is represented by audit persistence under `.mewcode/context/audit.jsonl`; full snapshot persistence is intentionally deferred because the spec says first implementation only writes audit JSONL.

**Placeholder scan:** The plan avoids placeholder language. Each task includes concrete files, interfaces, test commands, and implementation snippets.

**Type consistency:** `PrepareRequest`, `PreparedTurn`, `RuntimeContext`, `BudgetRequest`, `BudgetResult`, `RecoveryTracker`, `Router`, `HandoffPackage`, and audit event names are defined before later tasks consume them.

**Risk notes:** Task 10 is the highest-risk step because it changes `Agent.Run` turn preparation. Keep that diff focused and preserve event emission order around `PreSend`, `PostReceive`, and compact notifications. Task 12 is prompt-cache-sensitive; compare the old and new forked conversation message sequence before deleting the local helper.
