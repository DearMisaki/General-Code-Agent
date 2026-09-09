package teams

import (
	"context"
	"fmt"
	"strings"

	"mewcode/internal/orchestration"
	"mewcode/internal/tools"
)

// SendMessageTool allows agents to send messages to named teammates.
type SendMessageTool struct {
	TeamMgr    *TeamManager
	SenderName string
}

func (t *SendMessageTool) Name() string                 { return "SendMessage" }
func (t *SendMessageTool) Category() tools.ToolCategory { return tools.CategoryCommand }
func (t *SendMessageTool) Description() string {
	return "Send a message to another named agent in the team. The recipient will see it on their next turn."
}

func (t *SendMessageTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"to": map[string]any{
					"type":        "string",
					"description": "Name of the recipient agent",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Message content to send",
				},
			},
			"required": []string{"to", "content"},
		},
	}
}

func (t *SendMessageTool) Execute(ctx context.Context, args map[string]any) tools.ToolResult {
	to, _ := args["to"].(string)
	content, _ := args["content"].(string)
	if to == "" || content == "" {
		return tools.ToolResult{Output: "Error: 'to' and 'content' are required", IsError: true}
	}

	// The lead is not registered as a Member (it lives in the parent
	// process and only reads from its own mailbox), so route to it by
	// finding any team the sender belongs to.
	if to == LeadName {
		for _, teamName := range t.TeamMgr.ListTeams() {
			team := t.TeamMgr.GetTeam(teamName)
			if team == nil {
				continue
			}
			if _, ok := team.Members[t.SenderName]; ok {
				team.SendMessage(t.SenderName, LeadName, content)
				return tools.ToolResult{
					Output: fmt.Sprintf("Message sent to %s.", LeadName),
				}
			}
		}
		return tools.ToolResult{
			Output:  fmt.Sprintf("Error: cannot find team for sender '%s'", t.SenderName),
			IsError: true,
		}
	}

	// Find a registered teammate with this name, or fall back to the
	// file-based mailbox. In tmux/iTerm mode each teammate runs in a
	// separate process and only knows about itself in Members, so the
	// in-memory lookup will miss peers. The mailbox write always works
	// because all processes share the same inbox directory on disk.
	for _, teamName := range t.TeamMgr.ListTeams() {
		team := t.TeamMgr.GetTeam(teamName)
		if team == nil {
			continue
		}
		if _, ok := team.Members[to]; ok {
			team.SendMessage(t.SenderName, to, content)
			return tools.ToolResult{
				Output: fmt.Sprintf("Message sent to %s.", to),
			}
		}
		// Recipient not in Members but we belong to this team — write
		// directly to the file mailbox so external-process peers pick
		// it up on their next poll.
		if _, ok := team.Members[t.SenderName]; ok {
			team.SendMessage(t.SenderName, to, content)
			return tools.ToolResult{
				Output: fmt.Sprintf("Message sent to %s.", to),
			}
		}
	}

	return tools.ToolResult{
		Output:  fmt.Sprintf("Error: recipient '%s' not found in any team", to),
		IsError: true,
	}
}

// TeamCreateTool creates a new agent team.
type TeamCreateTool struct {
	TeamMgr *TeamManager
}

func (t *TeamCreateTool) Name() string                 { return "TeamCreate" }
func (t *TeamCreateTool) Category() tools.ToolCategory { return tools.CategoryCommand }
func (t *TeamCreateTool) Description() string {
	return `Create a new team for coordinating multiple agents.

## When to Use

Use this tool proactively whenever:
- The user explicitly asks to use a team, swarm, or group of agents
- The user mentions wanting agents to work together, coordinate, or collaborate
- A task requires sequential or parallel collaboration between multiple agents

When in doubt about whether a task warrants a team, prefer spawning a team.

## Team Workflow

1. **Create a team** with TeamCreate
2. Use **TaskCreate** to put durable work items on the shared task board when multiple agents need ownership, dependencies, or review state
3. **Spawn teammates** using the Agent tool with team_name and name parameters — this is REQUIRED to create long-running team members
4. Teammates use **TaskClaim** and **TaskUpdate** to coordinate shared task board state
5. Teammates work independently and communicate via **SendMessage**
6. When a teammate finishes, it sends its result to "lead" via SendMessage, then goes idle
7. The lead collects and synthesizes all teammate results

## CRITICAL: Spawning Teammates

To add a member to a team, you MUST pass both team_name and name to the Agent tool:
` + "```" + `
Agent({
  "team_name": "<team name from step 1>",
  "name": "<member name, e.g. reviewer>",
  "prompt": "...",
  "description": "..."
})
` + "```" + `
Without team_name, the agent runs as a one-shot sub-agent that blocks and returns inline — it will NOT be a team member.

## Teammate Idle State

Teammates go idle after every turn — this is completely normal. A teammate going idle after sending a message does NOT mean they are done or unavailable. Sending a message to an idle teammate wakes them up.

## Communication

- Use SendMessage to talk to teammates by name
- Messages from teammates arrive as system reminders at the start of each turn
- Messages are delivered automatically — you do NOT need to manually check your inbox`
}

func (t *TeamCreateTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name": map[string]any{
					"type":        "string",
					"description": "Name for the team",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "What this team will work on",
				},
			},
			"required": []string{"team_name"},
		},
	}
}

func (t *TeamCreateTool) Execute(ctx context.Context, args map[string]any) tools.ToolResult {
	name, _ := args["team_name"].(string)
	if name == "" {
		return tools.ToolResult{Output: "Error: team_name is required", IsError: true}
	}

	// Deduplicate: if name exists, append suffix
	baseName := name
	for i := 2; t.TeamMgr.GetTeam(name) != nil; i++ {
		name = fmt.Sprintf("%s-%d", baseName, i)
	}

	mode := detectBackend()
	team := t.TeamMgr.CreateTeam(name, mode)

	desc, _ := args["description"].(string)
	return tools.ToolResult{
		Output: fmt.Sprintf("Team \"%s\" created (mode: %s). Use Agent tool with team_name=\"%s\" to add teammates.\nDescription: %s",
			team.Name, team.Mode, team.Name, desc),
	}
}

// TeamDeleteTool deletes an agent team and stops all members.
type TeamDeleteTool struct {
	TeamMgr *TeamManager
}

func (t *TeamDeleteTool) Name() string { return "TeamDelete" }

func (t *TeamDeleteTool) Category() tools.ToolCategory { return tools.CategoryCommand }
func (t *TeamDeleteTool) Description() string {
	return "Delete a team, stopping all its members."
}

func (t *TeamDeleteTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name": map[string]any{
					"type":        "string",
					"description": "Name of the team to delete",
				},
			},
			"required": []string{"team_name"},
		},
	}
}

func (t *TeamDeleteTool) Execute(ctx context.Context, args map[string]any) tools.ToolResult {
	name, _ := args["team_name"].(string)
	if name == "" {
		return tools.ToolResult{Output: "Error: team_name is required", IsError: true}
	}

	team := t.TeamMgr.GetTeam(name)
	if team == nil {
		return tools.ToolResult{
			Output:  fmt.Sprintf("Error: team '%s' not found", name),
			IsError: true,
		}
	}

	memberCount := len(team.Members)
	var memberNames []string
	for n := range team.Members {
		memberNames = append(memberNames, n)
	}

	t.TeamMgr.DeleteTeam(name)
	return tools.ToolResult{
		Output: fmt.Sprintf("Team \"%s\" deleted. Stopped %d member(s): %s", name, memberCount, strings.Join(memberNames, ", ")),
	}
}

type TaskCreateTool struct {
	TeamMgr *TeamManager
}

func (t *TaskCreateTool) Name() string                 { return "TaskCreate" }
func (t *TaskCreateTool) Category() tools.ToolCategory { return tools.CategoryCommand }
func (t *TaskCreateTool) Description() string {
	return "Create a durable task on a team's shared task board."
}
func (t *TaskCreateTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name":    map[string]any{"type": "string"},
				"title":        map[string]any{"type": "string"},
				"description":  map[string]any{"type": "string"},
				"priority":     map[string]any{"type": "number"},
				"dependencies": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"team_name", "title"},
		},
	}
}
func (t *TaskCreateTool) Execute(_ context.Context, args map[string]any) tools.ToolResult {
	team, res := teamWithOrchestrator(t.TeamMgr, stringArg(args, "team_name"))
	if res.IsError {
		return res
	}
	task, err := team.Orchestrator.CreateBoardTask(orchestration.BoardTask{
		Title:        stringArg(args, "title"),
		Description:  stringArg(args, "description"),
		Priority:     intArg(args, "priority"),
		Dependencies: stringSliceArg(args, "dependencies"),
	})
	if err != nil {
		return tools.ToolResult{Output: fmt.Sprintf("Error creating task: %s", err), IsError: true}
	}
	return tools.ToolResult{Output: fmt.Sprintf("Task %s created: %s", task.ID, task.Title)}
}

type TaskListTool struct {
	TeamMgr *TeamManager
}

func (t *TaskListTool) Name() string                 { return "TaskList" }
func (t *TaskListTool) Category() tools.ToolCategory { return tools.CategoryRead }
func (t *TaskListTool) Description() string {
	return "List tasks on a team's shared task board."
}
func (t *TaskListTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name": map[string]any{"type": "string"},
				"status":    map[string]any{"type": "string"},
				"owner":     map[string]any{"type": "string"},
			},
			"required": []string{"team_name"},
		},
	}
}
func (t *TaskListTool) Execute(_ context.Context, args map[string]any) tools.ToolResult {
	team, res := teamWithOrchestrator(t.TeamMgr, stringArg(args, "team_name"))
	if res.IsError {
		return res
	}
	tasks, err := team.Orchestrator.Board.ListTasks()
	if err != nil {
		return tools.ToolResult{Output: fmt.Sprintf("Error listing tasks: %s", err), IsError: true}
	}
	statusFilter := stringArg(args, "status")
	ownerFilter := stringArg(args, "owner")
	var lines []string
	for _, task := range tasks {
		if statusFilter != "" && string(task.Status) != statusFilter {
			continue
		}
		if ownerFilter != "" && task.Owner != ownerFilter {
			continue
		}
		lines = append(lines, formatTaskLine(task))
	}
	if len(lines) == 0 {
		return tools.ToolResult{Output: "No tasks found."}
	}
	return tools.ToolResult{Output: strings.Join(lines, "\n")}
}

type TaskGetTool struct {
	TeamMgr *TeamManager
}

func (t *TaskGetTool) Name() string                 { return "TaskGet" }
func (t *TaskGetTool) Category() tools.ToolCategory { return tools.CategoryRead }
func (t *TaskGetTool) Description() string          { return "Get one task from a team's shared task board." }
func (t *TaskGetTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name": map[string]any{"type": "string"},
				"task_id":   map[string]any{"type": "string"},
			},
			"required": []string{"team_name", "task_id"},
		},
	}
}
func (t *TaskGetTool) Execute(_ context.Context, args map[string]any) tools.ToolResult {
	team, res := teamWithOrchestrator(t.TeamMgr, stringArg(args, "team_name"))
	if res.IsError {
		return res
	}
	task, ok, err := team.Orchestrator.Board.GetTask(stringArg(args, "task_id"))
	if err != nil {
		return tools.ToolResult{Output: fmt.Sprintf("Error reading task: %s", err), IsError: true}
	}
	if !ok {
		return tools.ToolResult{Output: "Error: task not found", IsError: true}
	}
	return tools.ToolResult{Output: formatTaskDetail(task)}
}

type TaskClaimTool struct {
	TeamMgr *TeamManager
}

func (t *TaskClaimTool) Name() string                 { return "TaskClaim" }
func (t *TaskClaimTool) Category() tools.ToolCategory { return tools.CategoryCommand }
func (t *TaskClaimTool) Description() string {
	return "Claim an open task on a team's shared task board."
}
func (t *TaskClaimTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name":  map[string]any{"type": "string"},
				"task_id":    map[string]any{"type": "string"},
				"agent_name": map[string]any{"type": "string"},
			},
			"required": []string{"team_name", "task_id", "agent_name"},
		},
	}
}
func (t *TaskClaimTool) Execute(_ context.Context, args map[string]any) tools.ToolResult {
	team, res := teamWithOrchestrator(t.TeamMgr, stringArg(args, "team_name"))
	if res.IsError {
		return res
	}
	taskID := stringArg(args, "task_id")
	owner := stringArg(args, "agent_name")
	if err := team.Orchestrator.AssignBoardTask(taskID, owner); err != nil {
		return tools.ToolResult{Output: fmt.Sprintf("Error claiming task: %s", err), IsError: true}
	}
	return tools.ToolResult{Output: fmt.Sprintf("Task %s claimed by %s.", taskID, owner)}
}

type TaskUpdateTool struct {
	TeamMgr *TeamManager
}

func (t *TaskUpdateTool) Name() string                 { return "TaskUpdate" }
func (t *TaskUpdateTool) Category() tools.ToolCategory { return tools.CategoryCommand }
func (t *TaskUpdateTool) Description() string {
	return "Update task status, progress, result, or error on a team's shared task board."
}
func (t *TaskUpdateTool) Schema() map[string]any {
	return map[string]any{
		"name":        t.Name(),
		"description": t.Description(),
		"input_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"team_name":  map[string]any{"type": "string"},
				"task_id":    map[string]any{"type": "string"},
				"agent_name": map[string]any{"type": "string"},
				"status":     map[string]any{"type": "string"},
				"message":    map[string]any{"type": "string"},
				"result":     map[string]any{"type": "string"},
			},
			"required": []string{"team_name", "task_id", "agent_name", "status"},
		},
	}
}
func (t *TaskUpdateTool) Execute(_ context.Context, args map[string]any) tools.ToolResult {
	team, res := teamWithOrchestrator(t.TeamMgr, stringArg(args, "team_name"))
	if res.IsError {
		return res
	}
	taskID := stringArg(args, "task_id")
	actor := stringArg(args, "agent_name")
	status := orchestration.BoardTaskStatus(stringArg(args, "status"))
	if status == orchestration.BoardTaskDone {
		if err := team.Orchestrator.CompleteBoardTask(taskID, actor, stringArg(args, "result")); err != nil {
			return tools.ToolResult{Output: fmt.Sprintf("Error updating task: %s", err), IsError: true}
		}
		return tools.ToolResult{Output: fmt.Sprintf("Task %s updated to %s.", taskID, status)}
	}
	if err := team.Orchestrator.Board.UpdateTask(taskID, actor, status, stringArg(args, "message"), stringArg(args, "result")); err != nil {
		return tools.ToolResult{Output: fmt.Sprintf("Error updating task: %s", err), IsError: true}
	}
	_ = team.Orchestrator.Mail.SendEnvelope(orchestration.MailEnvelope{
		Kind:     orchestration.MailKindBoardUpdate,
		From:     actor,
		To:       LeadName,
		TeamName: team.Name,
		TaskID:   taskID,
		Text:     orchestration.FormatEnvelopeText(orchestration.MailEnvelope{TaskID: taskID, Text: string(status)}),
	})
	return tools.ToolResult{Output: fmt.Sprintf("Task %s updated to %s.", taskID, status)}
}

func teamWithOrchestrator(tm *TeamManager, name string) (*Team, tools.ToolResult) {
	if name == "" {
		return nil, tools.ToolResult{Output: "Error: team_name is required", IsError: true}
	}
	if tm == nil {
		return nil, tools.ToolResult{Output: "Error: team manager is not configured", IsError: true}
	}
	team := tm.GetTeam(name)
	if team == nil {
		return nil, tools.ToolResult{Output: fmt.Sprintf("Error: team '%s' not found", name), IsError: true}
	}
	if team.Orchestrator == nil || team.Orchestrator.Board == nil {
		return nil, tools.ToolResult{Output: fmt.Sprintf("Error: team '%s' has no task board", name), IsError: true}
	}
	return team, tools.ToolResult{}
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return value
}

func intArg(args map[string]any, key string) int {
	switch value := args[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)
	if !ok {
		if strings, ok := args[key].([]string); ok {
			return append([]string(nil), strings...)
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func formatTaskLine(task orchestration.BoardTask) string {
	return fmt.Sprintf("%s [%s] owner=%s priority=%d title=%s", task.ID, task.Status, task.Owner, task.Priority, task.Title)
}

func formatTaskDetail(task orchestration.BoardTask) string {
	return fmt.Sprintf("ID: %s\nTitle: %s\nStatus: %s\nOwner: %s\nPriority: %d\nDescription: %s\nResult: %s\nError: %s",
		task.ID, task.Title, task.Status, task.Owner, task.Priority, task.Description, task.Result, task.Error)
}
