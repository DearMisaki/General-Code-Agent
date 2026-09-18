package extractor

import (
	"fmt"
	"strings"
)

type MemoryExtractionScope struct {
	UserID    string
	AppID     string
	ProjectID string
	AgentID   string
	SessionID string
	RunID     string
	TeamName  string
	TaskID    string
}

func BuildExtractCandidatesPrompt(newMessageCount int, existingMemories string, scope MemoryExtractionScope) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are now acting as the memory extraction sidecar agent. Analyze only the most recent ~%d messages above and extract durable long-term memories.\n\n", newMessageCount)
	b.WriteString("Return valid JSON only. Do not use markdown. The output object must contain a `candidates` array of MemoryCandidate objects.\n\n")
	b.WriteString("MemoryCandidate schema fields: scope, kind, memory, entities, relations, source_message_ids, source_agent_id, user_id, app_id, project_id, agent_id, session_id, run_id, team_name, task_id, confidence, sensitivity, metadata.\n\n")
	b.WriteString("Allowed scope values: user, project, agent, team, session.\n")
	b.WriteString("Allowed kind values: preference, constraint, project_fact, architecture_decision, feedback, tool_gotcha, agent_experience, team_state, task_summary, relationship, event.\n\n")
	b.WriteString("Do not store API keys, tokens, passwords, cookies, secrets, one-off shell output, low-confidence guesses, large source-code blocks, or temporary task chatter.\n")
	b.WriteString("Each memory must be a standalone fact sentence that can be understood without replaying the conversation.\n")
	b.WriteString("If the recent conversation contains explicit memory intent such as `记住`, `remember`, `以后你是`, `call yourself`, or `you are <name/role>`, extract it as a durable preference or constraint unless it is unsafe.\n")
	b.WriteString("For identity instructions like `你是 xyz` or `you are xyz`, store a user-scope preference such as `User wants the assistant to identify as xyz.`\n")
	b.WriteString("Use source_message_ids when message identifiers are visible; otherwise return an empty array.\n\n")
	fmt.Fprintf(&b, "Default scope identifiers: user_id=%q app_id=%q project_id=%q agent_id=%q session_id=%q run_id=%q team_name=%q task_id=%q.\n", scope.UserID, scope.AppID, scope.ProjectID, scope.AgentID, scope.SessionID, scope.RunID, scope.TeamName, scope.TaskID)
	if existingMemories != "" {
		b.WriteString("\nExisting memory manifest:\n")
		b.WriteString(existingMemories)
		b.WriteString("\n")
	}
	b.WriteString("\nExample output:\n")
	b.WriteString(`{"candidates":[{"scope":"project","kind":"architecture_decision","memory":"contextmgr.ContextGateway is the only context preparation entrypoint.","confidence":0.92,"user_id":"`)
	b.WriteString(scope.UserID)
	b.WriteString(`","app_id":"`)
	b.WriteString(scope.AppID)
	b.WriteString(`","project_id":"`)
	b.WriteString(scope.ProjectID)
	b.WriteString(`","source_message_ids":[]}]}`)
	return b.String()
}
