# Context Layer Architecture Design

## Goal

Replace the current scattered context-management flow with an explicit Context Layer that builds,
filters, budgets, transfers, persists, and audits the runtime information shown to agents.

The existing conversation, compact, and tool-result packages remain as compatibility primitives at
the start of the migration. The new layer becomes the architectural owner of context behavior.

## Current State

Context behavior currently lives across several packages:

- `internal/agent`: injects reminders, invokes context compaction, applies tool-result budgeting,
  handles active skills, and records recent file reads.
- `internal/conversation`: stores LLM message history.
- `internal/compact`: summarizes older conversation history and keeps a recent tail.
- `internal/toolresult`: spills large tool results and snips stale outputs.
- `internal/memory`: supplies long-term memory prompt text.
- `internal/agents` and `internal/teams`: manually copy or synthesize context for sub-agents,
  forks, teammates, and worktrees.
- `internal/permissions`: defines runtime visibility and action boundaries.

This works, but the architecture treats context as a side effect of prompt construction. There is no
single component that can answer what context exists, where it came from, why it was included, what
was filtered, or what was transferred to another agent.

## Target Architecture

Add `internal/contextmgr` as the explicit Context Layer.

```text
internal/contextmgr/
  types.go
  builder.go
  gateway.go
  store.go
  router.go
  lifecycle.go
  budgeter.go
  audit.go
  render.go
```

The Context Layer has five responsibilities:

1. Build a structured runtime context snapshot for every agent turn.
2. Render selected snapshot sections into the conversation sent to the LLM.
3. Apply size management through tool-result budgeting and conversation compaction.
4. Build handoff packages for fork, sub-agent, and team flows.
5. Persist audit events for compaction, spill, snip, handoff, reminder injection, and tool exposure.

## Runtime Context Model

Runtime context is represented as five typed sections plus budget metadata.

```go
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
```

`UserContext` starts with user instructions and memory prompt content. It can later grow into real
user preferences without changing agent-loop code.

`SessionContext` owns the current conversation, session ID, recent turn retention policy, tool-use
history, and compact boundary information.

`BusinessContext` owns active skills, allowed tool filters, full tool schema listing, deferred tool
names, MCP server/tool exposure, and permission boundaries.

`ExecutionContext` owns agent identity, agent type, iteration number, protocol, workdir, fork/team
metadata, worktree metadata, max iterations, and model budget settings.

`EnvironmentContext` owns OS, architecture, shell, current date, git repo state, branch, model, and
provider metadata.

`BudgetContext` owns context window, max output tokens, current usage anchor, auto-compact tracking,
tool-result replacement state, and recovery snapshots.

## Gateway Contract

`ContextGateway` is the only component `Agent.Run` calls before sending a turn to the LLM.

```go
type PrepareRequest struct {
	Conversation   *conversation.Manager
	WorkDir        string
	SessionID      string
	Protocol       string
	Iteration      int
	ContextWindow  int
	MaxOutputTokens int
	Checker        *permissions.Checker
	ToolSchemas    []map[string]any
	ActiveSkills   map[string]string
	Instructions   string
	MemoryContent  string
	Notifications  []string
	UsageAnchor    compact.UsageAnchor
}

type PreparedTurn struct {
	Snapshot        RuntimeContext
	APIConversation *conversation.Manager
	ToolSchemas     []map[string]any
	Notices          []ContextNotice
	Records          []audit.Record
	UsageAnchorReset bool
}
```

The gateway performs these steps in order:

1. Build a `RuntimeContext` snapshot from request inputs.
2. Render snapshot sections that must be visible as system reminders.
3. Apply lifecycle and budget management.
4. Apply tool-result replacement to create an API conversation.
5. Append audit records.
6. Return the prepared conversation and tool schemas.

`Agent.Run` remains responsible for streaming model events, executing tools, appending assistant
messages, appending tool results, handling permission requests, and publishing agent events.

## Builder

`Builder` collects runtime context without mutating the conversation. It records the source of every
section using:

```go
type ContextSource struct {
	Section string
	Kind    string
	Path    string
	Name    string
	FreshAt time.Time
}
```

Examples:

- Memory content source: `Kind: "memory"`.
- Active skill source: `Kind: "skill"`.
- Deferred tool source: `Kind: "tool-registry"`.
- Environment source: `Kind: "runtime"`.
- Session source: `Kind: "conversation"`.

## Renderer

`Renderer` converts selected structured sections into message text. It replaces ad hoc reminder
construction in `internal/agent`.

Initial rendered sections:

- Long-term instructions and memory.
- Plan mode reminder.
- Notification reminders.
- Active skill reminders.
- Deferred tool reminders.

Rendering must preserve current behavior first. Later changes can improve format once tests prove
compatibility.

## Budgeter

`Budgeter` owns size control.

It wraps:

- `toolresult.Apply`
- `toolresult.AppendRecords`
- `compact.ManageContext`
- `compact.ForceCompact`
- `compact.RecoveryState`

The first implementation keeps these packages intact and delegates to them. This avoids rewriting
well-tested token and spill behavior during the architectural migration.

## Store

`Store` persists context-layer artifacts under `.mewcode/context`.

```text
.mewcode/context/
  audit.jsonl
  snapshots/
```

The first implementation only writes audit JSONL. Snapshot persistence is introduced as an interface
with no mandatory writes so session compatibility stays unchanged.

Session JSONL remains in `.mewcode/sessions`. Compact boundaries continue to use the existing
session format.

## Router and Handoff

`Router` creates handoff packages for:

- forked agents
- definition-based sub-agents
- background agents
- team teammates
- worktree-isolated agents

```go
type HandoffPackage struct {
	ID             string
	FromAgent      AgentRef
	ToAgent        AgentRef
	Mode           HandoffMode
	Summary        string
	Messages       []conversation.Message
	ActiveSkills   map[string]string
	ToolSchemas    []map[string]any
	RecoveryFiles  []RecoveryFile
	Filtered       []FilteredItem
	CreatedAt      time.Time
}
```

Supported modes:

- `none`: no parent context.
- `recent`: keep only recent messages.
- `summary`: compact parent context into a summary plus recent tail.
- `full`: preserve parent conversation, patching incomplete tool-use blocks.
- `fork`: byte-stable replay for prompt-cache-sensitive forks.

Fork mode must preserve the current behavior: replay thinking blocks, tool uses, and tool results,
and insert placeholder tool results when the parent has incomplete tool calls.

## Lifecycle

`LifecycleManager` owns context transitions:

- create snapshot
- render snapshot
- apply budget
- compact when thresholds are crossed
- record recovery data after tool execution
- archive context audit records
- clear active context on `/clear`

The first migration wires lifecycle into the gateway. Later lifecycle work can move `/compact`,
`/clear`, resume, and memory extraction deeper into the context layer.

## Audit

Audit records are append-only JSONL.

```go
type Record struct {
	ID        string         `json:"id"`
	Time      time.Time      `json:"time"`
	Event     EventType      `json:"event"`
	AgentID   string         `json:"agent_id,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	ContextID string         `json:"context_id,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
```

Initial event types:

- `context.prepare`
- `context.render`
- `context.tool_result_budget`
- `context.compact`
- `context.handoff`
- `context.tool_exposure`
- `context.recovery_file_read`
- `context.recovery_skill`

Audit write failures are non-fatal. The agent must continue when audit persistence fails.

## Migration Phases

Phase 1 creates `internal/contextmgr` and unit tests without changing behavior.

Phase 2 changes `Agent.Run` to use `ContextGateway.PrepareTurn` instead of directly injecting
memory, plan, notifications, active skills, deferred tool reminders, compaction, and tool-result
budgeting.

Phase 3 changes `Agent.executeSingleTool` to report recovery file reads through the context layer.

Phase 4 changes `AgentTool` fork/sub-agent/team code to use `Router` handoff construction.

Phase 5 moves manual `/compact`, `/clear`, and resume integration onto lifecycle APIs while keeping
session JSONL compatibility.

## Compatibility Rules

- Do not change LLM adapter request formats in this migration.
- Do not change session JSONL plain message records.
- Do not remove `compact` or `toolresult`; delegate to them first.
- Do not require TUI rewrites during Phase 1 or Phase 2.
- Preserve prompt-cache-sensitive fork replay behavior.
- Never split an assistant tool-use message from its user tool-result message.
- Keep audit failures non-fatal.

## Success Criteria

- `Agent.Run` no longer manually orchestrates all context injection and budgeting.
- Context construction is testable through `contextmgr` unit tests.
- Existing compact and tool-result behavior remains covered by current tests.
- Handoff modes are explicit and tested.
- Context audit records exist for prepare, render, budget, compact, recovery, and handoff.
- Existing sessions can still be loaded and compact boundaries still work.
