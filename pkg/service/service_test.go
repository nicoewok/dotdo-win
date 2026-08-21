package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestService(t *testing.T) (*Service, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "dotdo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	svc := NewService(tempDir)
	return svc, tempDir
}

func TestService_InitAndStorage(t *testing.T) {
	svc, tempDir := setupTestService(t)
	defer os.RemoveAll(tempDir)

	if err := svc.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	tasksFile := filepath.Join(tempDir, "tasks.json")
	if _, err := os.Stat(tasksFile); os.IsNotExist(err) {
		t.Fatalf("tasks.json was not created in %s", tempDir)
	}
}

func TestService_AddAndListTasks(t *testing.T) {
	svc, tempDir := setupTestService(t)
	defer os.RemoveAll(tempDir)

	now := time.Now()
	dueTomorrow := now.Add(24 * time.Hour)

	// Add first task
	t1, err := svc.AddTask("Task One", time.Time{})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if t1.Title != "Task One" || t1.Status != "todo" || t1.ID != 1 {
		t.Errorf("unexpected task returned: %+v", t1)
	}

	// Add second task with due date
	t2, err := svc.AddTask("Task Two", dueTomorrow)
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}
	if t2.Title != "Task Two" || t2.ID != 2 {
		t.Errorf("unexpected task returned: %+v", t2)
	}

	// Duplicate title error check
	_, err = svc.AddTask("Task One", time.Time{})
	if err == nil {
		t.Errorf("expected error when adding duplicate task title")
	}

	// Empty title error check
	_, err = svc.AddTask("  ", time.Time{})
	if err == nil {
		t.Errorf("expected error when adding empty task title")
	}

	// List tasks (task with due date should sort first)
	allTasks, err := svc.ListTasks(false)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(allTasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(allTasks))
	}
	if allTasks[0].Title != "Task Two" {
		t.Errorf("expected task with due date to be sorted first, got %s", allTasks[0].Title)
	}
}

func TestService_StatusTransitions(t *testing.T) {
	svc, tempDir := setupTestService(t)
	defer os.RemoveAll(tempDir)

	_, err := svc.AddTask("Buy Milk", time.Time{})
	if err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Mark doing
	tDoing, err := svc.MarkDoing("Buy Milk")
	if err != nil {
		t.Fatalf("MarkDoing failed: %v", err)
	}
	if tDoing.Status != "doing" {
		t.Errorf("expected status 'doing', got '%s'", tDoing.Status)
	}

	// Mark done
	tDone, err := svc.MarkDone("Buy Milk")
	if err != nil {
		t.Fatalf("MarkDone failed: %v", err)
	}
	if tDone.Status != "done" {
		t.Errorf("expected status 'done', got '%s'", tDone.Status)
	}

	// List pending tasks should be empty
	pending, err := svc.ListTasks(true)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending tasks, got %d", len(pending))
	}
}

func TestService_RemoveDoneTasks(t *testing.T) {
	svc, tempDir := setupTestService(t)
	defer os.RemoveAll(tempDir)

	svc.AddTask("Task 1", time.Time{})
	svc.AddTask("Task 2", time.Time{})
	svc.MarkDone("Task 1")

	removedCount, err := svc.RemoveDoneTasks()
	if err != nil {
		t.Fatalf("RemoveDoneTasks failed: %v", err)
	}
	if removedCount != 1 {
		t.Errorf("expected 1 task removed, got %d", removedCount)
	}

	allTasks, err := svc.ListTasks(false)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(allTasks) != 1 || allTasks[0].Title != "Task 2" {
		t.Errorf("unexpected remaining tasks: %+v", allTasks)
	}
}

func TestService_DeleteSingleTask(t *testing.T) {
	svc, tempDir := setupTestService(t)
	defer os.RemoveAll(tempDir)

	t1, _ := svc.AddTask("Task A", time.Time{})
	svc.AddTask("Task B", time.Time{})

	removed, err := svc.DeleteTaskByID(t1.ID)
	if err != nil {
		t.Fatalf("DeleteTaskByID failed: %v", err)
	}
	if removed.Title != "Task A" {
		t.Errorf("expected deleted task title 'Task A', got '%s'", removed.Title)
	}

	remaining, err := svc.ListTasks(false)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Title != "Task B" {
		t.Errorf("unexpected remaining tasks: %+v", remaining)
	}
}

