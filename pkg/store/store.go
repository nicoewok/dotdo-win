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

// Config represents application settings stored in config.json.
type Config struct {
	SyncBranch string `json:"sync_branch"`
	GithubRepo string `json:"github_repo,omitempty"`
	Owner      string `json:"owner,omitempty"`
}

// IsGithubConnected returns true if github_repo is non-empty in config.json.
func IsGithubConnected(storageDir string) bool {
	cfg, err := LoadConfig(storageDir)
	if err != nil {
		return false
	}
	return cfg.GithubRepo != ""
}

// GetStorageDir returns the storage directory path.
// If customDir is non-empty, it returns customDir.
// Otherwise, it returns default AppData/dotdo (%APPDATA%\dotdo on Windows).
func GetStorageDir(customDir string) string {
	if customDir != "" {
		return filepath.Clean(customDir)
	}
	appData := os.Getenv("APPDATA")
	if appData == "" {
		cfgDir, err := os.UserConfigDir()
		if err == nil && cfgDir != "" {
			appData = cfgDir
		} else {
			home, _ := os.UserHomeDir()
			appData = home
		}
	}
	return filepath.Join(appData, "dotdo")
}

// GetStoragePath returns the path to tasks.json within storageDir.
func GetStoragePath(storageDir string) string {
	return filepath.Join(storageDir, "tasks.json")
}

// GetConfigPath returns the path to config.json within storageDir.
func GetConfigPath(storageDir string) string {
	return filepath.Join(storageDir, "config.json")
}

// EnsureGitignore ensures that .gitignore exists and ignores config.json.
func EnsureGitignore(dir string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return os.WriteFile(gitignorePath, []byte("config.json\n"), 0644)
	}
	return nil
}

// EnsureInitialized checks for storage folder, tasks.json, config.json, and .gitignore, creating them if missing.
func EnsureInitialized(storageDir string) error {
	dir := GetStorageDir(storageDir)
	path := GetStoragePath(dir)
	cfgPath := GetConfigPath(dir)

	// Create directory if missing
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Ensure .gitignore ignores config.json
	_ = EnsureGitignore(dir)

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

	// Create config.json if missing
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		defaultCfg := Config{
			SyncBranch: "main",
		}
		data, err := json.MarshalIndent(defaultCfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(cfgPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config.json: %w", err)
		}
	}

	return nil
}

// LoadConfig loads application configuration from config.json.
func LoadConfig(storageDir string) (Config, error) {
	dir := GetStorageDir(storageDir)
	cfgPath := GetConfigPath(dir)
	var cfg Config

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return Config{SyncBranch: "main"}, nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return cfg, nil
}

// SaveConfig saves configuration to config.json.
func SaveConfig(storageDir string, cfg Config) error {
	dir := GetStorageDir(storageDir)
	cfgPath := GetConfigPath(dir)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
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

	// Ensure duplicate IDs are detected and resolved automatically
	_ = list.DeduplicateIDs()

	return list, nil
}

// SaveTasks saves task list to tasks.json in storageDir.
func SaveTasks(storageDir string, list task.List) error {
	dir := GetStorageDir(storageDir)
	path := GetStoragePath(dir)

	// Deduplicate IDs before saving
	_ = list.DeduplicateIDs()

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %w", err)
	}

	return nil
}

// FullGitSync executes a full git add, commit, rebase pull, and push for storageDir.
// NOTE: Only tasks.json is staged; config.json is NEVER added or committed to git.
func FullGitSync(storageDir string) error {
	dir := GetStorageDir(storageDir)

	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("git repository is not initialized in %s", dir)
	}

	_ = EnsureGitignore(dir)

	status, err := getGitStatus(dir)
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if status != "" {
		if err := runGit(dir, "add", "tasks.json"); err != nil {
			return fmt.Errorf("git add failed: %w", err)
		}
		if err := runGit(dir, "commit", "-m", "dotdo: auto sync update"); err != nil {
			// Commit might fail if no changes to tasks.json, continue with pull/push
		}
	}

	// Try pull rebase on origin master or main
	if err := runGit(dir, "pull", "origin", "master", "--rebase"); err != nil {
		if errMain := runGit(dir, "pull", "origin", "main", "--rebase"); errMain != nil {
			// Ignore pull error if branch doesn't exist yet remotely
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
	setHideWindow(cmd)
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
	setHideWindow(cmd)
	out, err := cmd.Output()
	return string(out), err
}
