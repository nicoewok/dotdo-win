package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nicoewok/dotdo-win/pkg/task"
)

// GetStorageDir returns the storage directory path.
// If customDir is non-empty, it returns customDir.
// Otherwise, it returns default ~/.dotdo.
func GetStorageDir(customDir string) string {
	if customDir != "" {
		return filepath.Clean(customDir)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("USERPROFILE")
		if home == "" {
			home = "."
		}
	}
	return filepath.Join(home, ".dotdo")
}

// GetStoragePath returns the path to tasks.json within storageDir.
func GetStoragePath(storageDir string) string {
	return filepath.Join(storageDir, "tasks.json")
}

// EnsureInitialized checks for the storage folder and file, creating them if missing.
func EnsureInitialized(storageDir string) error {
	dir := GetStorageDir(storageDir)
	path := GetStoragePath(dir)

	// Create directory if missing
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Create tasks.json if missing
	if _, err := os.Stat(path); os.IsNotExist(err) {
		empty := task.List{Tasks: []task.Task{}}
		data, err := json.MarshalIndent(empty, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal empty task list: %w", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return fmt.Errorf("failed to write tasks.json: %w", err)
		}
	}

	return nil
}

// LoadTasks loads task list from tasks.json in storageDir.
func LoadTasks(storageDir string) (task.List, error) {
	dir := GetStorageDir(storageDir)
	path := GetStoragePath(dir)
	var list task.List

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return task.List{Tasks: []task.Task{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return list, fmt.Errorf("failed to read tasks file: %w", err)
	}

	if err := json.Unmarshal(data, &list); err != nil {
		return list, fmt.Errorf("failed to unmarshal tasks file: %w", err)
	}

	return list, nil
}

// SaveTasks saves task list to tasks.json in storageDir and triggers background git sync.
func SaveTasks(storageDir string, list task.List) error {
	dir := GetStorageDir(storageDir)
	path := GetStoragePath(dir)

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	// Sync in the background so API callers don't hang
	go BackgroundGitSync(dir)
	return nil
}

// BackgroundGitSync performs a best-effort git commit and push for background saving.
func BackgroundGitSync(repoPath string) {
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return
	}
	_ = runGit(repoPath, "add", "tasks.json")
	_ = runGit(repoPath, "commit", "-m", "dotdo: sync")
	_ = runGit(repoPath, "push", "origin", "main")
}

// FullGitSync executes a full git add, commit, rebase pull, and push for storageDir.
func FullGitSync(storageDir string) error {
	dir := GetStorageDir(storageDir)

	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("git repository is not initialized in %s", dir)
	}

	status, err := getGitStatus(dir)
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if status != "" {
		if err := runGit(dir, "add", "tasks.json"); err != nil {
			return fmt.Errorf("git add failed: %w", err)
		}
		if err := runGit(dir, "commit", "-m", "dotdo: auto sync update"); err != nil {
			return fmt.Errorf("local git commit failed: %w", err)
		}
	}

	// Try pull rebase on origin master or main
	if err := runGit(dir, "pull", "origin", "master", "--rebase"); err != nil {
		// Fallback attempt for 'main' branch if master fails
		if errMain := runGit(dir, "pull", "origin", "main", "--rebase"); errMain != nil {
			return fmt.Errorf("git pull rebase failed: %w", err)
		}
	}

	if err := runGit(dir, "push", "origin", "master"); err != nil {
		if errMain := runGit(dir, "push", "origin", "main"); errMain != nil {
			return fmt.Errorf("git push failed: %w", err)
		}
	}

	return nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s", stderr.String())
	}
	return nil
}

func getGitStatus(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

