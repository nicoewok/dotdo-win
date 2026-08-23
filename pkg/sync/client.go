package sync

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/nicoewok/dotdo-win/pkg/task"
)

const (
	DefaultBaseURL = "https://api.github.com"
	DefaultPath    = "tasks.json"
	DefaultRepo    = ".dotdo"
)

var (
	ErrNotFound     = errors.New("file not found on github")
	ErrUnauthorized = errors.New("unauthorized github token")
	ErrConflict     = errors.New("github file conflict (sha mismatch)")
)

// Client handles REST-based synchronization with GitHub's Contents API.
type Client struct {
	Token      string
	Owner      string
	Repo       string
	Path       string
	BaseURL    string
	HTTPClient *http.Client

	mu  sync.RWMutex
	sha string
}

// NewClient creates a new REST sync Client.
func NewClient(token, owner string, customRepo ...string) *Client {
	repo := DefaultRepo
	if len(customRepo) > 0 && customRepo[0] != "" {
		repo = customRepo[0]
	}
	return &Client{
		Token: token,
		Owner: owner,
		Repo:  repo,
		Path:  DefaultPath,
	}
}

// GetSHA returns the currently cached in-memory SHA.
func (c *Client) GetSHA() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sha
}

// SetSHA manually sets the cached in-memory SHA.
func (c *Client) SetSHA(sha string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sha = sha
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/")
	}
	return DefaultBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

// FetchTasks pulls tasks.json from GET /repos/{owner}/{repo}/contents/{path}.
// Decodes base64 content into the tasks slice and stores the file's SHA in memory.
// Fallback: If GET returns 404 (file does not exist), clears SHA and returns an empty tasks slice.
func (c *Client) FetchTasks(ctx context.Context) ([]task.Task, error) {
	if c.Token == "" {
		return nil, errors.New("token is required")
	}
	if c.Owner == "" {
		return nil, errors.New("owner is required")
	}

	repo := c.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	path := c.Path
	if path == "" {
		path = DefaultPath
	}

	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL(), c.Owner, repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "DotDoApp")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		c.SetSHA("")
		return []task.Task{}, nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code fetching tasks: %d", resp.StatusCode)
	}

	var contentResp struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		SHA      string `json:"sha"`
		Encoding string `json:"encoding"`
		Content  string `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
		return nil, fmt.Errorf("failed to decode contents response: %w", err)
	}

	// Cache the SHA in memory
	c.SetSHA(contentResp.SHA)

	// Clean newlines and whitespace from base64 content
	cleanContent := strings.ReplaceAll(contentResp.Content, "\n", "")
	cleanContent = strings.ReplaceAll(cleanContent, "\r", "")
	cleanContent = strings.TrimSpace(cleanContent)

	rawBytes, err := base64.StdEncoding.DecodeString(cleanContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %w", err)
	}

	var taskList task.List
	if err := json.Unmarshal(rawBytes, &taskList); err != nil {
		// Fallback: try unmarshaling as raw slice
		var rawSlice []task.Task
		if errSlice := json.Unmarshal(rawBytes, &rawSlice); errSlice == nil {
			return rawSlice, nil
		}
		return nil, fmt.Errorf("failed to unmarshal tasks json: %w", err)
	}

	return taskList.Tasks, nil
}

// SaveTasks marshals the tasks slice to JSON, base64-encodes it, and sends PUT /repos/{owner}/{repo}/contents/{path}.
// Includes commit message, encoded payload, and cached SHA (omitted if empty for new file creation).
// Updates in-memory SHA with the new SHA returned by the PUT response.
func (c *Client) SaveTasks(ctx context.Context, tasks []task.Task, message string) error {
	if c.Token == "" {
		return errors.New("token is required")
	}
	if c.Owner == "" {
		return errors.New("owner is required")
	}

	repo := c.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	path := c.Path
	if path == "" {
		path = DefaultPath
	}
	if message == "" {
		message = "dotdo: sync tasks"
	}

	taskList := task.List{Tasks: tasks}
	data, err := json.MarshalIndent(taskList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	encodedContent := base64.StdEncoding.EncodeToString(data)

	cachedSHA := c.GetSHA()

	putPayload := struct {
		Message string `json:"message"`
		Content string `json:"content"`
		SHA     string `json:"sha,omitempty"`
	}{
		Message: message,
		Content: encodedContent,
		SHA:     cachedSHA,
	}

	bodyBytes, err := json.Marshal(putPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal put payload: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL(), c.Owner, repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create save request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DotDoApp")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return ErrConflict
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code saving tasks: %d", resp.StatusCode)
	}

	var putResp struct {
		Content struct {
			SHA string `json:"sha"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&putResp); err == nil && putResp.Content.SHA != "" {
		c.SetSHA(putResp.Content.SHA)
	}

	return nil
}
