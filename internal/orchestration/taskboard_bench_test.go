package orchestration

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func BenchmarkTaskBoardClaimContention(b *testing.B) {
	for _, contenders := range []int{2, 10, 100} {
		b.Run(fmt.Sprintf("contenders_%d", contenders), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				board := NewTaskBoard(filepath.Join(b.TempDir(), fmt.Sprintf("board-%d.json", i)))
				task, err := board.CreateTask(BoardTask{Title: "claim me"})
				if err != nil {
					b.Fatal(err)
				}
				var wg sync.WaitGroup
				wg.Add(contenders)
				for c := 0; c < contenders; c++ {
					go func(c int) {
						defer wg.Done()
						_ = board.ClaimTask(task.ID, fmt.Sprintf("agent-%d", c))
					}(c)
				}
				wg.Wait()
			}
		})
	}
}

func BenchmarkTaskBoardList(b *testing.B) {
	for _, count := range []int{10, 1000, 10000} {
		b.Run(fmt.Sprintf("tasks_%d", count), func(b *testing.B) {
			board := NewTaskBoard(filepath.Join(b.TempDir(), "board.json"))
			if err := seedTaskBoardBenchmarkTasks(board, count); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := board.ListTasks(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func seedTaskBoardBenchmarkTasks(board *TaskBoard, count int) error {
	tasks := make([]BoardTask, count)
	for i := range tasks {
		tasks[i] = BoardTask{
			ID:        fmt.Sprintf("task-%d", i),
			Title:     fmt.Sprintf("task-%d", i),
			Status:    BoardTaskOpen,
			Priority:  i % 10,
			CreatedAt: time.Unix(int64(i), 0).UTC(),
			UpdatedAt: time.Unix(int64(i), 0).UTC(),
		}
	}
	return board.write(boardFile{Tasks: tasks})
}

func BenchmarkTaskBoardUpdateLargeHistory(b *testing.B) {
	board := NewTaskBoard(filepath.Join(b.TempDir(), "board.json"))
	task := BoardTask{
		ID:        "task-1",
		Title:     "update me",
		Status:    BoardTaskInProgress,
		Owner:     "agent-a",
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}
	history := make([]BoardUpdate, 10000)
	for i := range history {
		history[i] = BoardUpdate{
			TaskID:    task.ID,
			Actor:     "agent-a",
			From:      BoardTaskInProgress,
			To:        BoardTaskInProgress,
			Message:   fmt.Sprintf("update-%d", i),
			CreatedAt: time.Unix(int64(i), 0).UTC(),
		}
	}
	fixture := boardFile{Tasks: []BoardTask{task}, History: history}
	if err := board.write(fixture); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := board.write(fixture); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := board.UpdateTask(task.ID, "agent-a", BoardTaskInProgress, "benchmark update", ""); err != nil {
			b.Fatal(err)
		}
	}
}
