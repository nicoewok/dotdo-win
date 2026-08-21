package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nicoewok/dotdo-win/pkg/store"
	"github.com/nicoewok/dotdo-win/pkg/task"
)

var (
	ErrEmptyTitle    = errors.New("task title cannot be empty")
	ErrTaskExists    = errors.New("task already exists")
	ErrTaskNotFound  = errors.New("task not found")
	ErrInvalidStatus = errors.New("invalid task status")
)

// Service provides high-level task management operations.
type Service struct {
	storageDir string
}

// NewService creates a new Service instance.
// If customDir is provided (non-empty), it overrides the default ~/.dotdo directory.
func NewService(customDir ...string) *Service {
	dir := ""
	if len(customDir) > 0 {
		dir = customDir[0]
	}
	return &Service{
		storageDir: store.GetStorageDir(dir),
	}
}

// StorageDir returns the directory path configured for this service.
func (s *Service) StorageDir() string {
	return s.storageDir
}

// Init ensures the storage directory and tasks.json file are created.
func (s *Service) Init() error {
	return store.EnsureInitialized(s.storageDir)
}

// ListTasks returns all tasks, optionally filtering only pending tasks (status != "done").
// The tasks are returned sorted by due date.
func (s *Service) ListTasks(pendingOnly bool) ([]task.Task, error) {
	list, err := store.LoadTasks(s.storageDir)
	if err != nil {
		return nil, err
	}

	list.SortByDueDate()

	if !pendingOnly {
		return list.Tasks, nil
	}

	var pending []task.Task
	for _, t := range list.Tasks {
		if t.Status != "done" {
			pending = append(pending, t)
		}
	}
	return pending, nil
}

// AddTask adds a new task with the given title and due date.
// Returns an error if the title is empty or if a task with the exact title already exists.
func (s *Service) AddTask(title string, due time.Time) (*task.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrEmptyTitle
	}

	if err := s.Init(); err != nil {
		return nil, err
	}

	list, err := store.LoadTasks(s.storageDir)
	if err != nil {
		return nil, err
	}

	for _, t := range list.Tasks {
		if strings.EqualFold(t.Title, title) {
			return nil, fmt.Errorf("%w: %s", ErrTaskExists, title)
		}
	}

	maxID := 0
	for _, t := range list.Tasks {
		if t.ID > maxID {
			maxID = t.ID
		}
	}

	newTask := task.Task{
		ID:     maxID + 1,
		Title:  title,
		Status: "todo",
		Due:    due,
	}

	list.Tasks = append(list.Tasks, newTask)
	if err := store.SaveTasks(s.storageDir, list); err != nil {
		return nil, err
	}

	return &newTask, nil
}

// SetTaskStatus updates the status of a task matching the given title.
// Valid statuses are "todo", "doing", and "done".
func (s *Service) SetTaskStatus(title string, status string) (*task.Task, error) {
	title = strings.TrimSpace(title)
	status = strings.ToLower(strings.TrimSpace(status))

	if status != "todo" && status != "doing" && status != "done" {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, status)
	}

	list, err := store.LoadTasks(s.storageDir)
	if err != nil {
		return nil, err
	}

	foundIdx := -1
	for i, t := range list.Tasks {
		if strings.EqualFold(t.Title, title) {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, title)
	}

	list.Tasks[foundIdx].Status = status
	if err := store.SaveTasks(s.storageDir, list); err != nil {
		return nil, err
	}

	return &list.Tasks[foundIdx], nil
}

// MarkDoing sets a task's status to "doing".
func (s *Service) MarkDoing(title string) (*task.Task, error) {
	return s.SetTaskStatus(title, "doing")
}

// MarkDone sets a task's status to "done".
func (s *Service) MarkDone(title string) (*task.Task, error) {
	return s.SetTaskStatus(title, "done")
}

// RemoveDoneTasks removes all tasks with status "done" from storage.
// Returns the count of removed tasks.
func (s *Service) RemoveDoneTasks() (int, error) {
	list, err := store.LoadTasks(s.storageDir)
	if err != nil {
		return 0, err
	}

	initialCount := len(list.Tasks)
	var kept []task.Task

	for _, t := range list.Tasks {
		if t.Status != "done" {
			kept = append(kept, t)
		}
	}

	removedCount := initialCount - len(kept)
	if removedCount == 0 {
		return 0, nil
	}

	list.Tasks = kept
	if err := store.SaveTasks(s.storageDir, list); err != nil {
		return 0, err
	}

	return removedCount, nil
}

// DeleteTask removes a single task matching title from storage.
func (s *Service) DeleteTask(title string) (*task.Task, error) {
	title = strings.TrimSpace(title)
	list, err := store.LoadTasks(s.storageDir)
	if err != nil {
		return nil, err
	}

	foundIdx := -1
	var removed task.Task
	for i, t := range list.Tasks {
		if strings.EqualFold(t.Title, title) {
			foundIdx = i
			removed = t
			break
		}
	}

	if foundIdx == -1 {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, title)
	}

	list.Tasks = append(list.Tasks[:foundIdx], list.Tasks[foundIdx+1:]...)
	if err := store.SaveTasks(s.storageDir, list); err != nil {
		return nil, err
	}

	return &removed, nil
}

// DeleteTaskByID removes a single task matching ID from storage.
func (s *Service) DeleteTaskByID(id int) (*task.Task, error) {
	list, err := store.LoadTasks(s.storageDir)
	if err != nil {
		return nil, err
	}

	foundIdx := -1
	var removed task.Task
	for i, t := range list.Tasks {
		if t.ID == id {
			foundIdx = i
			removed = t
			break
		}
	}

	if foundIdx == -1 {
		return nil, fmt.Errorf("%w: id %d", ErrTaskNotFound, id)
	}

	list.Tasks = append(list.Tasks[:foundIdx], list.Tasks[foundIdx+1:]...)
	if err := store.SaveTasks(s.storageDir, list); err != nil {
		return nil, err
	}

	return &removed, nil
}

// Sync performs a full Git synchronization (pull with rebase and push) for storage.
func (s *Service) Sync() error {
	return store.FullGitSync(s.storageDir)
}

