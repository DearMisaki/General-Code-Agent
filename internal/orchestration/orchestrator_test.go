package orchestration

import (
	"path/filepath"
	"testing"
)

func newTestOrchestrator(t *testing.T) (*Orchestrator, *recordingMailbox) {
	t.Helper()
	dir := t.TempDir()
	mb := &recordingMailbox{}
	return &Orchestrator{
		TeamName: "demo",
		Board:    NewTaskBoard(filepath.Join(dir, "board.json")),
		Mail:     NewMailRouter("demo", mb),
		Sessions: NewSessionStore(filepath.Join(dir, "sessions")),
	}, mb
}

func TestOrchestratorStartDelegationCreatesSession(t *testing.T) {
	orch, _ := newTestOrchestrator(t)
	session, err := orch.StartDelegation(AgentRef{Name: "lead", Type: "lead"}, AgentRef{Name: "worker", Type: "general-purpose"}, "task_1")
	if err != nil {
		t.Fatalf("start delegation: %v", err)
	}
	if session.Mode != ModeDelegation || session.State != StateCreated || session.TaskBoardID != "task_1" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if len(session.Participants) != 1 || session.Participants[0].Name != "worker" {
		t.Fatalf("unexpected participants: %+v", session.Participants)
	}
}

func TestOrchestratorStartPeerSessionCreatesSession(t *testing.T) {
	orch, _ := newTestOrchestrator(t)
	session, err := orch.StartPeerSession(AgentRef{Name: "lead", Type: "lead"}, []AgentRef{
		{Name: "alice", Type: "reviewer"},
		{Name: "bob", Type: "builder"},
	})
	if err != nil {
		t.Fatalf("start peer: %v", err)
	}
	if session.Mode != ModePeer || len(session.Participants) != 2 {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestOrchestratorAssignBoardTaskSendsAssignment(t *testing.T) {
	orch, mb := newTestOrchestrator(t)
	task, err := orch.CreateBoardTask(BoardTask{Title: "implement parser", Priority: 5})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := orch.AssignBoardTask(task.ID, "alice"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if mb.recipient != "alice" {
		t.Fatalf("recipient = %s, want alice", mb.recipient)
	}
	if mb.message.Kind != string(MailKindAssignment) || mb.message.TaskID != task.ID {
		t.Fatalf("unexpected assignment message: %+v", mb.message)
	}
}

func TestOrchestratorCompleteBoardTaskNotifiesLead(t *testing.T) {
	orch, mb := newTestOrchestrator(t)
	task, err := orch.CreateBoardTask(BoardTask{Title: "implement parser"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := orch.AssignBoardTask(task.ID, "alice"); err != nil {
		t.Fatalf("assign: %v", err)
	}
	if err := orch.CompleteBoardTask(task.ID, "alice", "done"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if mb.recipient != "lead" {
		t.Fatalf("recipient = %s, want lead", mb.recipient)
	}
	if mb.message.Kind != string(MailKindBoardUpdate) || mb.message.TaskID != task.ID {
		t.Fatalf("unexpected board update: %+v", mb.message)
	}
}
