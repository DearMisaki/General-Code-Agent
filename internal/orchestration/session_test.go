package orchestration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStoreCreateAndUpdateState(t *testing.T) {
	store := NewSessionStore(filepath.Join(t.TempDir(), "sessions"))
	session, err := store.CreateSession(CollaborationSession{
		TeamName: "demo",
		Mode:     ModeDelegation,
		Parent:   AgentRef{Name: "lead", Type: "lead"},
		Participants: []AgentRef{
			{Name: "worker", Type: "general-purpose"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session id must be assigned")
	}
	if session.State != StateCreated {
		t.Fatalf("state = %s, want %s", session.State, StateCreated)
	}
	if session.CreatedAt.IsZero() || session.UpdatedAt.IsZero() {
		t.Fatalf("timestamps must be set: %+v", session)
	}

	before := session.UpdatedAt
	time.Sleep(time.Millisecond)
	if err := store.UpdateState(session.ID, StateRunning); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, ok, err := store.GetSession(session.ID)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.State != StateRunning {
		t.Fatalf("state = %s, want %s", got.State, StateRunning)
	}
	if !got.UpdatedAt.After(before) {
		t.Fatalf("updated_at did not advance: before=%s after=%s", before, got.UpdatedAt)
	}
}

func TestSessionStoreListSkipsInvalidJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	store := NewSessionStore(dir)
	if _, err := store.CreateSession(CollaborationSession{Mode: ModePeer}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
}
