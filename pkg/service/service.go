package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nicoewok/dotdo-win/pkg/config"
	"github.com/nicoewok/dotdo-win/pkg/store"
	"github.com/nicoewok/dotdo-win/pkg/sync"
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

	newTask := task.Task{
		ID:     list.NextID(),
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

// Pull fetches tasks from GitHub via REST Contents API (GET /repos/{owner}/.dotdo/contents/tasks.json)
// and overwrites local tasks.json with the fetched remote tasks.
func (s *Service) Pull() error {
	cfgMgr := config.NewManager()
	token, err := cfgMgr.GetToken()
	if err != nil || token == "" {
		return errors.New("not connected to GitHub (no token in credential manager)")
	}

	cfg, _ := cfgMgr.LoadConfig()
	owner := cfg.Owner
	if owner == "" {
		owner, _ = ValidateGithubToken(token)
	}
	if owner == "" {
		return errors.New("cannot determine GitHub owner")
	}

	repo := cfg.Repo
	if repo == "" {
		repo = ".dotdo"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	syncClient := sync.NewClient(token, owner, repo)
	remoteTasks, fetchErr := syncClient.FetchTasks(ctx)
	if fetchErr != nil {
		return fmt.Errorf("failed to pull tasks from GitHub: %w", fetchErr)
	}

	// Overwrite local tasks.json with fetched remote tasks and deduplicate any duplicate IDs
	taskList := task.List{Tasks: remoteTasks}
	_ = taskList.DeduplicateIDs()

	if err := store.SaveTasks(s.storageDir, taskList); err != nil {
		return fmt.Errorf("failed to write local tasks.json: %w", err)
	}

	return nil
}

// Push sends local tasks.json to GitHub via REST Contents API (PUT /repos/{owner}/.dotdo/contents/tasks.json).
func (s *Service) Push() error {
	cfgMgr := config.NewManager()
	token, err := cfgMgr.GetToken()
	if err != nil || token == "" {
		return errors.New("not connected to GitHub (no token in credential manager)")
	}

	cfg, _ := cfgMgr.LoadConfig()
	owner := cfg.Owner
	if owner == "" {
		owner, _ = ValidateGithubToken(token)
	}
	if owner == "" {
		return errors.New("cannot determine GitHub owner")
	}

	repo := cfg.Repo
	if repo == "" {
		repo = ".dotdo"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	syncClient := sync.NewClient(token, owner, repo)

	// Fetch current SHA first if not cached
	_, _ = syncClient.FetchTasks(ctx)

	localTasks, err := s.ListTasks(false)
	if err != nil {
		return err
	}

	if err := syncClient.SaveTasks(ctx, localTasks, "dotdo: sync tasks"); err != nil {
		return fmt.Errorf("failed to push tasks to GitHub: %w", err)
	}

	return nil
}

// Sync performs synchronization: pulls latest tasks (overwriting local tasks.json)
// and pushes local tasks to GitHub.
func (s *Service) Sync() error {
	// Try REST pull first
	pullErr := s.Pull()
	if pullErr == nil {
		// Also push after pulling to keep remote synced
		_ = s.Push()
		return nil
	}

	// If remote file did not exist (or 404), try pushing local tasks to create it remotely
	if pushErr := s.Push(); pushErr == nil {
		return nil
	}

	// Fallback to git CLI sync if configured
	return store.FullGitSync(s.storageDir)
}

// ValidateGithubToken tests a token against GitHub API https://api.github.com/user
// Returns the GitHub username (login) if valid, or an error if invalid.
func ValidateGithubToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("token cannot be empty")
	}

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "DotDoApp")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("invalid GitHub Personal Access Token (401 Unauthorized)")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err == nil && user.Login != "" {
		return user.Login, nil
	}

	return "", nil
}

// ConfigureGithub sets up GitHub integration with the given token, owner, and repo.
// Default repo is ".dotdo" if empty. Owner is auto-detected via GitHub API if empty.
// Securely stores token in Windows Credential Manager and non-sensitive settings in %LOCALAPPDATA%\DotDo\config.json.
// NOTE: config.json is NEVER pushed to Git; only tasks.json is synced.
func (s *Service) ConfigureGithub(token string, owner string, repo string) error {
	token = strings.TrimSpace(token)
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)

	if repo == "" {
		repo = ".dotdo"
	}

	if token == "" {
		return errors.New("token cannot be empty")
	}

	// Validate token against GitHub API and auto-detect username if owner is empty
	validatedUser, valErr := ValidateGithubToken(token)
	if valErr != nil {
		return valErr
	}

	if owner == "" && validatedUser != "" {
		owner = validatedUser
	}

	// 1. Save token securely in Windows Credential Manager under service "DotDoApp"
	cfgMgr := config.NewManager()
	if err := cfgMgr.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token in credential manager: %w", err)
	}

	// 2. Save non-sensitive config in %LOCALAPPDATA%\DotDo\config.json
	_ = cfgMgr.SaveConfig(&config.Config{
		Owner:  owner,
		Repo:   repo,
		Branch: "main",
	})

	// 3. Save local store config
	repoStr := ""
	if owner != "" && repo != "" {
		repoStr = fmt.Sprintf("%s/%s", owner, repo)
	}
	_ = store.SaveConfig(s.storageDir, store.Config{
		SyncBranch: "main",
		GithubRepo: repoStr,
		Owner:      owner,
	})

	// 4. Immediately perform Sync (pull remote tasks and overwrite local tasks.json, or push if 404)
	_ = s.Sync()

	return nil
}

// DisconnectGithub removes credentials from Windows Credential Manager and clears GitHub connection.
func (s *Service) DisconnectGithub() error {
	cfgMgr := config.NewManager()
	_ = cfgMgr.Logout()

	cfg, _ := store.LoadConfig(s.storageDir)
	cfg.GithubRepo = ""
	_ = store.SaveConfig(s.storageDir, cfg)

	return nil
}
