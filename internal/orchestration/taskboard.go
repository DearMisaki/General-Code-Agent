package orchestration

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type TaskBoard struct {
	path string
}

type boardFile struct {
	Tasks   []BoardTask   `json:"tasks"`
	History []BoardUpdate `json:"history,omitempty"`
}

func NewTaskBoard(path string) *TaskBoard {
	return &TaskBoard{path: path}
}

func (b *TaskBoard) CreateTask(task BoardTask) (BoardTask, error) {
	var created BoardTask
	err := b.withLock(func(data boardFile) (boardFile, error) {
		now := time.Now().UTC()
		if task.ID == "" {
			task.ID = fmt.Sprintf("task_%d", now.UnixNano())
		}
		if task.Status == "" {
			task.Status = BoardTaskOpen
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = now
		}
		task.UpdatedAt = now
		data.Tasks = append(data.Tasks, task)
		data.History = append(data.History, BoardUpdate{
			TaskID:    task.ID,
			Actor:     "system",
			To:        task.Status,
			Message:   "task created",
			CreatedAt: now,
		})
		created = task
		return data, nil
	})
	return created, err
}

func (b *TaskBoard) ClaimTask(taskID, owner string) error {
	if taskID == "" || owner == "" {
		return fmt.Errorf("task id and owner are required")
	}
	return b.withLock(func(data boardFile) (boardFile, error) {
		idx := findTask(data.Tasks, taskID)
		if idx < 0 {
			return data, fmt.Errorf("task not found: %s", taskID)
		}
		task := data.Tasks[idx]
		if task.Status != BoardTaskOpen {
			return data, fmt.Errorf("task %s is not open", taskID)
		}
		now := time.Now().UTC()
		task.Owner = owner
		task.Status = BoardTaskClaimed
		task.UpdatedAt = now
		data.Tasks[idx] = task
		data.History = append(data.History, BoardUpdate{
			TaskID:    taskID,
			Actor:     owner,
			From:      BoardTaskOpen,
			To:        BoardTaskClaimed,
			Message:   "task claimed",
			CreatedAt: now,
		})
		return data, nil
	})
}

func (b *TaskBoard) UpdateTask(taskID, actor string, status BoardTaskStatus, message, result string) error {
	if taskID == "" || actor == "" || status == "" {
		return fmt.Errorf("task id, actor, and status are required")
	}
	return b.withLock(func(data boardFile) (boardFile, error) {
		idx := findTask(data.Tasks, taskID)
		if idx < 0 {
			return data, fmt.Errorf("task not found: %s", taskID)
		}
		task := data.Tasks[idx]
		if task.Owner != "" && task.Owner != actor && actor != "lead" {
			return data, fmt.Errorf("task %s is owned by %s", taskID, task.Owner)
		}
		now := time.Now().UTC()
		from := task.Status
		task.Status = status
		if task.Owner == "" && actor != "lead" {
			task.Owner = actor
		}
		switch status {
		case BoardTaskDone:
			task.Result = result
			task.Error = ""
		case BoardTaskFailed:
			task.Error = message
		default:
			if result != "" {
				task.Result = result
			}
		}
		task.UpdatedAt = now
		data.Tasks[idx] = task
		data.History = append(data.History, BoardUpdate{
			TaskID:    taskID,
			Actor:     actor,
			From:      from,
			To:        status,
			Message:   message,
			CreatedAt: now,
		})
		return data, nil
	})
}

func (b *TaskBoard) GetTask(taskID string) (BoardTask, bool, error) {
	data, err := b.read()
	if err != nil {
		return BoardTask{}, false, err
	}
	idx := findTask(data.Tasks, taskID)
	if idx < 0 {
		return BoardTask{}, false, nil
	}
	return data.Tasks[idx], true, nil
}

func (b *TaskBoard) ListTasks() ([]BoardTask, error) {
	data, err := b.read()
	if err != nil {
		return nil, err
	}
	tasks := append([]BoardTask(nil), data.Tasks...)
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Priority != tasks[j].Priority {
			return tasks[i].Priority > tasks[j].Priority
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
}

func (b *TaskBoard) withLock(fn func(boardFile) (boardFile, error)) error {
	if b == nil || b.path == "" {
		return fmt.Errorf("task board path is required")
	}
	if err := os.MkdirAll(filepath.Dir(b.path), 0o755); err != nil {
		return err
	}
	lockFile := b.path + ".lock"
	var lockFd *os.File
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		lockFd, err = os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			break
		}
		if !os.IsExist(err) {
			return err
		}
		if info, statErr := os.Stat(lockFile); statErr == nil && time.Since(info.ModTime()) > 10*time.Second {
			_ = os.Remove(lockFile)
		}
		time.Sleep(time.Duration(5+rand.Intn(96)) * time.Millisecond)
	}
	if lockFd == nil {
		return err
	}
	_ = lockFd.Close()
	defer os.Remove(lockFile)

	data, err := b.read()
	if err != nil {
		return err
	}
	data, err = fn(data)
	if err != nil {
		return err
	}
	return b.write(data)
}

func (b *TaskBoard) read() (boardFile, error) {
	var data boardFile
	if b == nil || b.path == "" {
		return data, fmt.Errorf("task board path is required")
	}
	raw, err := os.ReadFile(b.path)
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}
	if len(raw) == 0 {
		return data, nil
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return boardFile{}, err
	}
	return data, nil
}

func (b *TaskBoard) write(data boardFile) error {
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, b.path)
}

func findTask(tasks []BoardTask, taskID string) int {
	for i := range tasks {
		if tasks[i].ID == taskID {
			return i
		}
	}
	return -1
}
