package orchestration

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestTaskBoardCreateClaimAndComplete(t *testing.T) {
	board := NewTaskBoard(filepath.Join(t.TempDir(), "board.json"))
	task, err := board.CreateTask(BoardTask{
		Title:       "implement parser",
		Description: "write parser tests",
		Priority:    10,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if task.ID == "" {
		t.Fatal("new task must have id")
	}
	if task.Status != BoardTaskOpen {
		t.Fatalf("new task status = %s, want %s", task.Status, BoardTaskOpen)
	}
	if err := board.ClaimTask(task.ID, "alice"); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := board.UpdateTask(task.ID, "alice", BoardTaskDone, "done", "ok"); err != nil {
		t.Fatalf("complete: %v", err)
	}
	got, ok, err := board.GetTask(task.ID)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.Owner != "alice" || got.Status != BoardTaskDone || got.Result != "ok" {
		t.Fatalf("unexpected task: %+v", got)
	}
}

func TestTaskBoardRejectsClaimedTask(t *testing.T) {
	board := NewTaskBoard(filepath.Join(t.TempDir(), "board.json"))
	task, err := board.CreateTask(BoardTask{Title: "review code"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := board.ClaimTask(task.ID, "alice"); err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if err := board.ClaimTask(task.ID, "bob"); err == nil {
		t.Fatal("second claim should fail")
	}
}

func TestTaskBoardListSortsByPriorityThenCreatedAt(t *testing.T) {
	board := NewTaskBoard(filepath.Join(t.TempDir(), "board.json"))
	low, err := board.CreateTask(BoardTask{Title: "low", Priority: 1})
	if err != nil {
		t.Fatalf("create low: %v", err)
	}
	high, err := board.CreateTask(BoardTask{Title: "high", Priority: 5})
	if err != nil {
		t.Fatalf("create high: %v", err)
	}
	alsoHigh, err := board.CreateTask(BoardTask{Title: "also high", Priority: 5})
	if err != nil {
		t.Fatalf("create also high: %v", err)
	}

	tasks, err := board.ListTasks()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := []string{high.ID, alsoHigh.ID, low.ID}
	if len(tasks) != len(want) {
		t.Fatalf("got %d tasks, want %d", len(tasks), len(want))
	}
	for i := range want {
		if tasks[i].ID != want[i] {
			t.Fatalf("task[%d] = %s, want %s", i, tasks[i].ID, want[i])
		}
	}
}

func TestTaskBoardConcurrentClaimHasOneWinner(t *testing.T) {
	board := NewTaskBoard(filepath.Join(t.TempDir(), "board.json"))
	task, err := board.CreateTask(BoardTask{Title: "race"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	var wg sync.WaitGroup
	winners := make(chan string, 8)
	for _, owner := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			if err := board.ClaimTask(task.ID, owner); err == nil {
				winners <- owner
			}
		}(owner)
	}
	wg.Wait()
	close(winners)

	var got []string
	for owner := range winners {
		got = append(got, owner)
	}
	successes := len(got)
	if successes != 1 {
		t.Fatalf("successful claims = %d, want exactly 1", successes)
	}
	stored, ok, err := board.GetTask(task.ID)
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if stored.Owner != got[0] || stored.Status != BoardTaskClaimed {
		t.Fatalf("stored task = %+v, winner = %s", stored, got[0])
	}
}
