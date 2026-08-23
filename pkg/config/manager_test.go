package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestGetConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	expectedDir := filepath.Clean(tempDir)
	if mgr.GetConfigDir() != expectedDir {
		t.Errorf("expected dir %s, got %s", expectedDir, mgr.GetConfigDir())
	}

	expectedPath := filepath.Join(expectedDir, "config.json")
	if mgr.GetConfigPath() != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, mgr.GetConfigPath())
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewManager(tempDir)

	// Test loading when file does not exist
	cfg, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error loading non-existent config: %v", err)
	}
	if cfg.Owner != "" || cfg.Repo != "" || cfg.Branch != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}

	// Save new config
	newCfg := &Config{
		Owner:  "nicoewok",
		Repo:   "dotdo-win",
		Branch: "main",
	}

	if err := mgr.SaveConfig(newCfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Verify file created
	if _, err := os.Stat(mgr.GetConfigPath()); os.IsNotExist(err) {
		t.Fatalf("config file was not created at %s", mgr.GetConfigPath())
	}

	// Load config back
	loaded, err := mgr.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load config back: %v", err)
	}

	if loaded.Owner != "nicoewok" || loaded.Repo != "dotdo-win" || loaded.Branch != "main" {
		t.Errorf("loaded config mismatched: %+v", loaded)
	}
}

func TestKeyringOperations(t *testing.T) {
	keyring.MockInit()

	mgr := NewManager()

	// Initially not authenticated
	if mgr.IsAuthenticated() {
		t.Error("expected IsAuthenticated to be false initially")
	}

	_, err := mgr.GetToken()
	if err == nil {
		t.Error("expected error retrieving token when none set")
	}

	// Save token
	testToken := "gho_mock_access_token_12345"
	if err := mgr.SaveToken(testToken); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Should be authenticated now
	if !mgr.IsAuthenticated() {
		t.Error("expected IsAuthenticated to be true after saving token")
	}

	// Retrieve token
	gotToken, err := mgr.GetToken()
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	if gotToken != testToken {
		t.Errorf("expected token %s, got %s", testToken, gotToken)
	}

	// Logout
	if err := mgr.Logout(); err != nil {
		t.Fatalf("failed to logout: %v", err)
	}

	// Should not be authenticated after logout
	if mgr.IsAuthenticated() {
		t.Error("expected IsAuthenticated to be false after logout")
	}

	// Logging out again should be idempotent
	if err := mgr.Logout(); err != nil {
		t.Errorf("unexpected error on second logout: %v", err)
	}
}
