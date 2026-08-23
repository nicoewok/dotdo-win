package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL         = "https://github.com"
	DefaultDeviceCodePath  = "/login/device/code"
	DefaultAccessTokenPath = "/login/oauth/access_token"
	DefaultScope           = "repo"
	DefaultInterval        = 5
	GrantTypeDeviceCode    = "urn:ietf:params:oauth:grant-type:device_code"
)

var (
	ErrAuthorizationPending = errors.New("authorization_pending")
	ErrSlowDown             = errors.New("slow_down")
	ErrTokenExpired         = errors.New("expired_token")
	ErrAccessDenied         = errors.New("access_denied")
	ErrTimeout              = errors.New("authorization request timed out")
)

// DeviceCodeResponse represents the response from the initial device code request.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
	ExpiresIn       int    `json:"expires_in"`
}

// TokenResponse represents a successful token response from GitHub.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// Client interacts with GitHub OAuth Device Flow APIs.
type Client struct {
	ClientID         string
	BaseURL          string
	HTTPClient       *http.Client
	PollIntervalUnit time.Duration // Defaults to time.Second if 0
}

// NewClient returns a new GitHub OAuth Client.
func NewClient(clientID string) *Client {
	return &Client{
		ClientID: clientID,
	}
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

func (c *Client) pollUnit() time.Duration {
	if c.PollIntervalUnit > 0 {
		return c.PollIntervalUnit
	}
	return time.Second
}

// RequestDeviceCode initiates the GitHub OAuth Device Flow by posting client_id and scope.
func (c *Client) RequestDeviceCode(ctx context.Context, scope string) (*DeviceCodeResponse, error) {
	if c.ClientID == "" {
		return nil, errors.New("client_id is required")
	}
	if scope == "" {
		scope = DefaultScope
	}

	endpoint := c.baseURL() + DefaultDeviceCodePath

	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device code request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status code: %d", resp.StatusCode)
	}

	var deviceResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return nil, fmt.Errorf("failed to decode device code response: %w", err)
	}

	if deviceResp.DeviceCode == "" {
		return nil, errors.New("invalid response from github: missing device_code")
	}

	if deviceResp.Interval <= 0 {
		deviceResp.Interval = DefaultInterval
	}

	return &deviceResp, nil
}

// CheckAccessToken makes a single query to the GitHub access token endpoint.
func (c *Client) CheckAccessToken(ctx context.Context, deviceCode string) (*TokenResponse, error) {
	if c.ClientID == "" {
		return nil, errors.New("client_id is required")
	}
	if deviceCode == "" {
		return nil, errors.New("device_code is required")
	}

	endpoint := c.baseURL() + DefaultAccessTokenPath

	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("device_code", deviceCode)
	v.Set("grant_type", GrantTypeDeviceCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create access token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected http status code: %d", resp.StatusCode)
	}

	var raw struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		Scope            string `json:"scope"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		ErrorURI         string `json:"error_uri"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode access token response: %w", err)
	}

	if raw.Error != "" {
		switch raw.Error {
		case "authorization_pending":
			return nil, ErrAuthorizationPending
		case "slow_down":
			return nil, ErrSlowDown
		case "expired_token":
			if raw.ErrorDescription != "" {
				return nil, fmt.Errorf("%w: %s", ErrTokenExpired, raw.ErrorDescription)
			}
			return nil, ErrTokenExpired
		case "access_denied":
			if raw.ErrorDescription != "" {
				return nil, fmt.Errorf("%w: %s", ErrAccessDenied, raw.ErrorDescription)
			}
			return nil, ErrAccessDenied
		default:
			return nil, fmt.Errorf("oauth error: %s (%s)", raw.Error, raw.ErrorDescription)
		}
	}

	if raw.AccessToken == "" {
		return nil, errors.New("empty access_token received")
	}

	return &TokenResponse{
		AccessToken: raw.AccessToken,
		TokenType:   raw.TokenType,
		Scope:       raw.Scope,
	}, nil
}

// PollForAccessToken polls the GitHub access token endpoint at the specified interval
// until authorization completes, times out, or receives a terminal error.
func (c *Client) PollForAccessToken(ctx context.Context, deviceCode string, interval int, expiresIn int) (*TokenResponse, error) {
	pollInterval := interval
	if pollInterval <= 0 {
		pollInterval = DefaultInterval
	}

	unit := c.pollUnit()

	if expiresIn > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(expiresIn)*unit)
		defer cancel()
	}

	timer := time.NewTimer(time.Duration(pollInterval) * unit)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrTimeout
			}
			return nil, ctx.Err()

		case <-timer.C:
			tokenResp, err := c.CheckAccessToken(ctx, deviceCode)
			if err == nil {
				return tokenResp, nil
			}

			if errors.Is(err, ErrAuthorizationPending) {
				timer.Reset(time.Duration(pollInterval) * unit)
				continue
			}

			if errors.Is(err, ErrSlowDown) {
				pollInterval += 5
				timer.Reset(time.Duration(pollInterval) * unit)
				continue
			}

			return nil, err
		}
	}
}

// RequestDeviceCode is a package-level helper to initiate the device flow.
func RequestDeviceCode(ctx context.Context, clientID string, scope string) (*DeviceCodeResponse, error) {
	return NewClient(clientID).RequestDeviceCode(ctx, scope)
}

// PollForAccessToken is a package-level helper to poll for the access token.
func PollForAccessToken(ctx context.Context, clientID string, deviceCode string, interval int, expiresIn int) (*TokenResponse, error) {
	return NewClient(clientID).PollForAccessToken(ctx, deviceCode, interval, expiresIn)
}
