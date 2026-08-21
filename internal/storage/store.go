package storage

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
)

func LoadTasks() (List, error) {

	path := GetStoragePath()
	var list List

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return List{Tasks: []Task{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return list, err
	}

	err = json.Unmarshal(data, &list)
	return list, err
}

func SaveTasks(list List) error {
	path := GetStoragePath()
	data, _ := json.MarshalIndent(list, "", "  ")

	err := os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}

	// Sync in the background so the CLI doesn't hang
	go sync(filepath.Dir(path))
	return nil
}

func GetStorageDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dotdo")
}

func GetStoragePath() string {
	return filepath.Join(GetStorageDir(), "tasks.json")
}

func sync(repoPath string) {
	// Check if .git exists first
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return
	}
	exec.Command("git", "-C", repoPath, "add", "tasks.json").Run()
	exec.Command("git", "-C", repoPath, "commit", "-m", "dotdo: sync").Run()
	exec.Command("git", "-C", repoPath, "push", "origin", "main").Run()
}
