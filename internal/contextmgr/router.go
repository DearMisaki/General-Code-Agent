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
		ID:           fmt.Sprintf("handoff-%d", time.Now().UnixNano()),
		FromAgent:    req.FromAgent,
		ToAgent:      req.ToAgent,
		Mode:         req.Mode,
		Summary:      req.Summary,
		Messages:     messages,
		ActiveSkills: copyStringMap(req.ActiveSkills),
		ToolSchemas:  append([]map[string]any(nil), req.ToolSchemas...),
		CreatedAt:    time.Now().UTC(),
	}
}

func (r *Router) BuildForkedConversation(parent *conversation.Manager, task, boilerplate string) *conversation.Manager {
	forked := conversation.NewManager()
	if parent != nil {
		for _, msg := range parent.GetMessages() {
			switch {
			case len(msg.ToolUses) > 0 && len(msg.ToolResults) == 0:
				forked.AddAssistantFull(msg.Content, msg.ThinkingBlocks, msg.ToolUses)
				placeholders := make([]conversation.ToolResultBlock, 0, len(msg.ToolUses))
				for _, tu := range msg.ToolUses {
					placeholders = append(placeholders, conversation.ToolResultBlock{
						ToolUseID: tu.ToolUseID,
						Content:   "(tool execution interrupted by fork)",
					})
				}
				forked.AddToolResultsMessage(placeholders)
			case len(msg.ToolUses) > 0:
				forked.AddAssistantFull(msg.Content, msg.ThinkingBlocks, msg.ToolUses)
			case len(msg.ToolResults) > 0:
				forked.AddToolResultsMessage(msg.ToolResults)
			case msg.Role == "assistant":
				forked.AddAssistantFull(msg.Content, msg.ThinkingBlocks, nil)
			default:
				forked.AddUserMessage(msg.Content)
			}
		}
	}
	if boilerplate != "" {
		forked.AddUserMessage(boilerplate + "\n\nYour task:\n" + task)
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
