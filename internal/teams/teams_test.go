package teams

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"mewcode/internal/orchestration"
)

func TestFileMailBoxRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mb := NewFileMailBox(dir)

	if err := mb.Send("alice", FileMailMessage{From: "bob", Text: "hi"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if err := mb.Send("alice", FileMailMessage{From: "carol", Text: "hello"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	unread, err := mb.ReadUnread("alice")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(unread) != 2 {
		t.Fatalf("expected 2 unread, got %d", len(unread))
	}

	if err := mb.MarkAllRead("alice"); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	unread2, _ := mb.ReadUnread("alice")
	if len(unread2) != 0 {
		t.Errorf("expected 0 unread after MarkAllRead, got %d", len(unread2))
	}
}

func TestFileMailBoxConcurrentSends(t *testing.T) {
	dir := t.TempDir()
	mb := NewFileMailBox(dir)

	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = mb.Send("dest", FileMailMessage{From: "sender", Text: "msg"})
		}(i)
	}
	wg.Wait()

	got, err := mb.ReadUnread("dest")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != n {
		t.Errorf("expected %d messages after concurrent sends, got %d", n, len(got))
	}
}

func TestTeamManagerCRUD(t *testing.T) {
	tm := NewTeamManager()

	team := tm.CreateTeam("alpha", ModeInProcess)
	if team == nil {
		t.Fatal("CreateTeam returned nil")
	}
	if got := tm.GetTeam("alpha"); got != team {
		t.Errorf("Get should return same team instance")
	}
	if names := tm.ListTeams(); len(names) != 1 || names[0] != "alpha" {
		t.Errorf("ListTeams = %v, want [alpha]", names)
	}
	tm.DeleteTeam("alpha")
	if got := tm.GetTeam("alpha"); got != nil {
		t.Error("DeleteTeam did not remove team")
	}
}

func TestTeamCreateDescriptionExplainsTaskBoardWorkflow(t *testing.T) {
	desc := (&TeamCreateTool{}).Description()
	for _, want := range []string{"TaskCreate", "TaskClaim", "TaskUpdate", "shared task board"} {
		if !strings.Contains(desc, want) {
			t.Fatalf("description missing %q:\n%s", want, desc)
		}
	}
}

func TestNewTeamInitializesOrchestrator(t *testing.T) {
	team := NewTeam("orchestrated", ModeInProcess)
	if team.MailBox == nil {
		t.Fatal("mailbox must be initialized")
	}
	if team.Orchestrator == nil {
		t.Fatal("orchestrator must be initialized")
	}
	task, err := team.Orchestrator.CreateBoardTask(orchestration.BoardTask{Title: "wire team"})
	if err != nil {
		t.Fatalf("create board task: %v", err)
	}
	tasks, err := team.Orchestrator.Board.ListTasks()
	if err != nil {
		t.Fatalf("list board tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != task.ID {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
}

func TestDetectBackendFallback(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ITERM_SESSION_ID", "")
	// PATH manipulation: point to an empty dir so `tmux` lookup fails.
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)
	if got := detectBackend(); got != ModeInProcess {
		t.Errorf("expected in-process fallback, got %q", got)
	}
}

func TestDetectBackendPrefersTmuxWhenInside(t *testing.T) {
	t.Setenv("TMUX", "/tmp/sock,1,0")
	if got := detectBackend(); got != ModeTmux {
		t.Errorf("expected tmux when TMUX set, got %q", got)
	}
}

func TestDetectBackendPicksITermWhenInside(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ITERM_SESSION_ID", "w0t0p0:ABC")
	// PATH without tmux so we don't fall back to it.
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)
	if got := detectBackend(); got != ModeITerm {
		t.Errorf("expected iterm, got %q", got)
	}
}

// TestSendMessageToolRoutesToLead pins the fix for the bug where a
// teammate calling SendMessage(to="lead", ...) saw "recipient 'lead' not
// found in any team" because the lead is never registered as a Member.
// The tool must recognize LeadName and route via the sender's team
// mailbox so the lead can read the reply on its next sweep.
func TestSendMessageToolRoutesToLead(t *testing.T) {
	tm := NewTeamManager()
	team := tm.CreateTeam("demo", ModeInProcess)
	team.AddMember("alice", nil, nil, "")

	tool := &SendMessageTool{TeamMgr: tm, SenderName: "alice"}
	res := tool.Execute(context.Background(), map[string]any{
		"to":      LeadName,
		"content": "here is the README summary",
	})
	if res.IsError {
		t.Fatalf("SendMessage to lead errored: %s", res.Output)
	}
	if !strings.Contains(res.Output, LeadName) {
		t.Errorf("expected confirmation mentioning %q, got %q", LeadName, res.Output)
	}

	msgs, err := team.MailBox.ReadUnread(LeadName)
	if err != nil {
		t.Fatalf("read lead inbox: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message in lead inbox, got %d", len(msgs))
	}
	if msgs[0].From != "alice" || msgs[0].Text != "here is the README summary" {
		t.Errorf("unexpected message: %+v", msgs[0])
	}
}

// TestSendMessageToolUnknownSenderToLead guards the failure path: if no
// team contains the sender, sending to the lead can't pick a mailbox and
// must surface a clear error rather than silently dropping the message.
func TestSendMessageToolUnknownSenderToLead(t *testing.T) {
	tm := NewTeamManager()
	tm.CreateTeam("demo", ModeInProcess) // no members added

	tool := &SendMessageTool{TeamMgr: tm, SenderName: "ghost"}
	res := tool.Execute(context.Background(), map[string]any{
		"to":      LeadName,
		"content": "anyone?",
	})
	if !res.IsError {
		t.Fatalf("expected error when sender has no team, got: %s", res.Output)
	}
}

func TestTaskBoardToolsCreateClaimAndComplete(t *testing.T) {
	tm := NewTeamManager()
	team := tm.CreateTeam("board-demo", ModeInProcess)
	team.AddMember("alice", nil, nil, "")

	create := (&TaskCreateTool{TeamMgr: tm}).Execute(context.Background(), map[string]any{
		"team_name":   "board-demo",
		"title":       "implement parser",
		"description": "write parser code",
		"priority":    float64(7),
	})
	if create.IsError {
		t.Fatalf("create errored: %s", create.Output)
	}

	tasks, err := team.Orchestrator.Board.ListTasks()
	if err != nil {
		t.Fatalf("list board: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("got %d tasks, want 1", len(tasks))
	}

	claim := (&TaskClaimTool{TeamMgr: tm}).Execute(context.Background(), map[string]any{
		"team_name":  "board-demo",
		"task_id":    tasks[0].ID,
		"agent_name": "alice",
	})
	if claim.IsError {
		t.Fatalf("claim errored: %s", claim.Output)
	}
	secondClaim := (&TaskClaimTool{TeamMgr: tm}).Execute(context.Background(), map[string]any{
		"team_name":  "board-demo",
		"task_id":    tasks[0].ID,
		"agent_name": "bob",
	})
	if !secondClaim.IsError {
		t.Fatal("second claim should fail")
	}

	update := (&TaskUpdateTool{TeamMgr: tm}).Execute(context.Background(), map[string]any{
		"team_name":  "board-demo",
		"task_id":    tasks[0].ID,
		"agent_name": "alice",
		"status":     string(orchestration.BoardTaskDone),
		"result":     "parser implemented",
	})
	if update.IsError {
		t.Fatalf("update errored: %s", update.Output)
	}
	msgs, err := team.MailBox.ReadUnread(LeadName)
	if err != nil {
		t.Fatalf("read lead inbox: %v", err)
	}
	if len(msgs) == 0 || msgs[len(msgs)-1].Kind != string(orchestration.MailKindBoardUpdate) {
		t.Fatalf("missing board update message: %+v", msgs)
	}
}

func TestTaskListAndGetTools(t *testing.T) {
	tm := NewTeamManager()
	team := tm.CreateTeam("board-list", ModeInProcess)
	task, err := team.Orchestrator.CreateBoardTask(orchestration.BoardTask{Title: "review code"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	list := (&TaskListTool{TeamMgr: tm}).Execute(context.Background(), map[string]any{"team_name": "board-list"})
	if list.IsError || !strings.Contains(list.Output, task.ID) {
		t.Fatalf("list output = %q error=%v", list.Output, list.IsError)
	}
	get := (&TaskGetTool{TeamMgr: tm}).Execute(context.Background(), map[string]any{
		"team_name": "board-list",
		"task_id":   task.ID,
	})
	if get.IsError || !strings.Contains(get.Output, "review code") {
		t.Fatalf("get output = %q error=%v", get.Output, get.IsError)
	}
}

var _ = filepath.Join
var _ = os.Getwd
