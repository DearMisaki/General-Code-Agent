package orchestration

import "testing"

func TestCollaborationModesAreStable(t *testing.T) {
	got := []CollaborationMode{ModeDelegation, ModePeer, ModeTaskBoard}
	want := []CollaborationMode{"delegation", "peer", "task_board"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mode[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBoardStatusesAreStable(t *testing.T) {
	statuses := []BoardTaskStatus{
		BoardTaskOpen,
		BoardTaskClaimed,
		BoardTaskInProgress,
		BoardTaskBlocked,
		BoardTaskReview,
		BoardTaskDone,
		BoardTaskFailed,
		BoardTaskCancelled,
	}
	seen := map[BoardTaskStatus]bool{}
	for _, status := range statuses {
		if status == "" {
			t.Fatal("status must not be empty")
		}
		if seen[status] {
			t.Fatalf("duplicate status: %s", status)
		}
		seen[status] = true
	}
}
