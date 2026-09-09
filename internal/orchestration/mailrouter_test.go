package orchestration

import (
	"path/filepath"
	"strings"
	"testing"
)

type recordingMailbox struct {
	recipient string
	message   MailboxMessage
}

func (m *recordingMailbox) Send(recipient string, msg MailboxMessage) error {
	m.recipient = recipient
	m.message = msg
	return nil
}

func TestMailRouterSendEnvelopeWritesTypedMailboxMessage(t *testing.T) {
	_ = filepath.Join(t.TempDir(), "inboxes")
	mb := &recordingMailbox{}
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
	if mb.recipient != "alice" {
		t.Fatalf("recipient = %s, want alice", mb.recipient)
	}
	msg := mb.message
	if msg.ID == "" {
		t.Fatal("message id must be assigned")
	}
	if msg.Kind != string(MailKindAssignment) || msg.TeamName != "demo" || msg.TaskID != "task_1" {
		t.Fatalf("unexpected typed fields: %+v", msg)
	}
	if msg.From != "lead" || msg.Text != "implement task" || msg.Timestamp == "" {
		t.Fatalf("legacy fields not preserved: %+v", msg)
	}
	if msg.Metadata["priority"] != "10" {
		t.Fatalf("metadata not preserved: %+v", msg.Metadata)
	}
}

func TestFormatEnvelopeTextIncludesTaskID(t *testing.T) {
	text := FormatEnvelopeText(MailEnvelope{
		Kind:   MailKindBoardUpdate,
		From:   "alice",
		To:     "lead",
		TaskID: "task_7",
		Text:   "ready for review",
	})
	if !strings.Contains(text, "task_7") || !strings.Contains(text, "ready for review") {
		t.Fatalf("formatted text missing task context: %q", text)
	}
}
