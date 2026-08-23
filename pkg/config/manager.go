package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	ServiceName = "dotdo"
	TokenKey    = "access_token"
)

// Config represents non-sensitive configuration stored in %LOCALAPPDATA%\dotdo\config.json.
type Config struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

// Manager manages application configuration and credentials.
type Manager struct {
	customDir string
}

// NewManager creates a new Manager instance.
// Option customDir overrides default %LOCALAPPDATA%\dotdo storage directory (useful for testing).
func NewManager(customDir ...string) *Manager {
	dir := ""
	if len(customDir) > 0 {
		dir = customDir[0]
	}
	return &Manager{customDir: dir}
}

// GetConfigDir returns the directory path for configuration.
// Defaults to %LOCALAPPDATA%\dotdo (or fallback %APPDATA%\dotdo / user config dir).
func (m *Manager) GetConfigDir() string {
	if m.customDir != "" {
		return filepath.Clean(m.customDir)
	}

	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		cfgDir, err := os.UserConfigDir()
		if err == nil && cfgDir != "" {
			localAppData = cfgDir
		} else {
			appData := os.Getenv("APPDATA")
			if appData != "" {
				localAppData = appData
			} else {
				home, _ := os.UserHomeDir()
				localAppData = home
			}
		}
	}
	return filepath.Join(localAppData, "dotdo")
}

// GetConfigPath returns the full path to config.json within the config directory.
func (m *Manager) GetConfigPath() string {
	return filepath.Join(m.GetConfigDir(), "config.json")
}

// LoadConfig reads non-sensitive configuration from %LOCALAPPDATA%\DotDo\config.json.
func (m *Manager) LoadConfig() (*Config, error) {
	path := m.GetConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// SaveConfig writes non-sensitive configuration to %LOCALAPPDATA%\DotDo\config.json.
func (m *Manager) SaveConfig(cfg *Config) error {
	dir := m.GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := m.GetConfigPath()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveToken saves the OAuth access_token securely into Windows Credential Manager under service "DotDoApp".
func (m *Manager) SaveToken(token string) error {
	if token == "" {
		return errors.New("cannot save empty token")
	}
	return keyring.Set(ServiceName, TokenKey, token)
}

// GetToken retrieves the OAuth access_token from Windows Credential Manager.
func (m *Manager) GetToken() (string, error) {
	token, err := keyring.Get(ServiceName, TokenKey)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve access token: %w", err)
	}
	return token, nil
}

// IsAuthenticated checks whether a valid token exists in Windows Credential Manager.
func (m *Manager) IsAuthenticated() bool {
	token, err := m.GetToken()
	return err == nil && token != ""
}

// Logout deletes the OAuth access_token from Windows Credential Manager under service "DotDoApp".
func (m *Manager) Logout() error {
	err := keyring.Delete(ServiceName, TokenKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("failed to delete access token: %w", err)
	}
	return nil
}

// Package-level helper functions using the default Manager:

// LoadConfig reads non-sensitive configuration using default manager.
func LoadConfig() (*Config, error) {
	return NewManager().LoadConfig()
}

// SaveConfig writes non-sensitive configuration using default manager.
func SaveConfig(cfg *Config) error {
	return NewManager().SaveConfig(cfg)
}

// SaveToken saves the OAuth access_token into Windows Credential Manager under service "DotDoApp".
func SaveToken(token string) error {
	return NewManager().SaveToken(token)
}

// GetToken retrieves the OAuth access_token from Windows Credential Manager.
func GetToken() (string, error) {
	return NewManager().GetToken()
}

// IsAuthenticated returns true if a token exists in Windows Credential Manager.
func IsAuthenticated() bool {
	return NewManager().IsAuthenticated()
}

// Logout deletes the OAuth access_token from Windows Credential Manager under service "DotDoApp".
func Logout() error {
	return NewManager().Logout()
}
