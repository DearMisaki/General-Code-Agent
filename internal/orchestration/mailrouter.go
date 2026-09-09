package orchestration

import (
	"fmt"
	"strings"
	"time"
)

type MailboxMessage struct {
	ID        string            `json:"id,omitempty"`
	Kind      string            `json:"kind,omitempty"`
	From      string            `json:"from"`
	Text      string            `json:"text"`
	Timestamp string            `json:"timestamp"`
	Read      bool              `json:"read"`
	Color     string            `json:"color,omitempty"`
	Summary   string            `json:"summary,omitempty"`
	TeamName  string            `json:"team_name,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Mailbox interface {
	Send(recipient string, msg MailboxMessage) error
}

type MailRouter struct {
	teamName string
	mailbox  Mailbox
}

func NewMailRouter(teamName string, mailbox Mailbox) *MailRouter {
	return &MailRouter{teamName: teamName, mailbox: mailbox}
}

func (r *MailRouter) SendEnvelope(env MailEnvelope) error {
	if r == nil || r.mailbox == nil {
		return fmt.Errorf("mailbox is required")
	}
	if env.To == "" {
		return fmt.Errorf("recipient is required")
	}
	if env.ID == "" {
		env.ID = fmt.Sprintf("mail_%d", time.Now().UnixNano())
	}
	if env.TeamName == "" {
		env.TeamName = r.teamName
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = time.Now().UTC()
	}
	if env.Kind == "" {
		env.Kind = MailKindMessage
	}
	return r.mailbox.Send(env.To, MailboxMessage{
		ID:        env.ID,
		Kind:      string(env.Kind),
		From:      env.From,
		Text:      env.Text,
		Timestamp: env.CreatedAt.Format(time.RFC3339Nano),
		TeamName:  env.TeamName,
		TaskID:    env.TaskID,
		Metadata:  env.Metadata,
		Summary:   string(env.Kind),
	})
}

func FormatEnvelopeText(env MailEnvelope) string {
	var parts []string
	if env.TaskID != "" {
		parts = append(parts, "task="+env.TaskID)
	}
	if env.Text != "" {
		parts = append(parts, env.Text)
	}
	if len(parts) == 0 {
		return string(env.Kind)
	}
	return strings.Join(parts, " ")
}
