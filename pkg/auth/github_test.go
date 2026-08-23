package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultDeviceCodePath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("missing or incorrect Accept header: %s", r.Header.Get("Accept"))
		}

		body, _ := io.ReadAll(r.Body)
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatalf("failed to parse body: %v", err)
		}

		if values.Get("client_id") != "test-client-id" {
			t.Errorf("expected client_id test-client-id, got %s", values.Get("client_id"))
		}
		if values.Get("scope") != "repo" {
			t.Errorf("expected scope repo, got %s", values.Get("scope"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode:      "dev_12345",
			UserCode:        "ABCD-1234",
			VerificationURI: "https://github.com/login/device",
			Interval:        5,
			ExpiresIn:       900,
		})
	}))
	defer server.Close()

	client := NewClient("test-client-id")
	client.BaseURL = server.URL

	resp, err := client.RequestDeviceCode(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.DeviceCode != "dev_12345" {
		t.Errorf("expected DeviceCode dev_12345, got %s", resp.DeviceCode)
	}
	if resp.UserCode != "ABCD-1234" {
		t.Errorf("expected UserCode ABCD-1234, got %s", resp.UserCode)
	}
	if resp.VerificationURI != "https://github.com/login/device" {
		t.Errorf("expected VerificationURI https://github.com/login/device, got %s", resp.VerificationURI)
	}
	if resp.Interval != 5 {
		t.Errorf("expected Interval 5, got %d", resp.Interval)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expected ExpiresIn 900, got %d", resp.ExpiresIn)
	}
}

func TestCheckAccessToken_States(t *testing.T) {
	var step int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != DefaultAccessTokenPath {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("missing or incorrect Accept header")
		}

		current := atomic.AddInt32(&step, 1)
		w.Header().Set("Content-Type", "application/json")

		switch current {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "authorization_pending",
				"error_description": "Pending user authorization",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "slow_down",
				"error_description": "Too fast",
			})
		case 3:
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "expired_token",
				"error_description": "Token expired",
			})
		case 4:
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "access_denied",
				"error_description": "User denied access",
			})
		case 5:
			_ = json.NewEncoder(w).Encode(TokenResponse{
				AccessToken: "gho_test_token_123",
				TokenType:   "bearer",
				Scope:       "repo",
			})
		}
	}))
	defer server.Close()

	client := NewClient("test-client-id")
	client.BaseURL = server.URL

	ctx := context.Background()

	// Step 1: authorization_pending
	_, err := client.CheckAccessToken(ctx, "dev_code")
	if !errors.Is(err, ErrAuthorizationPending) {
		t.Errorf("expected ErrAuthorizationPending, got %v", err)
	}

	// Step 2: slow_down
	_, err = client.CheckAccessToken(ctx, "dev_code")
	if !errors.Is(err, ErrSlowDown) {
		t.Errorf("expected ErrSlowDown, got %v", err)
	}

	// Step 3: expired_token
	_, err = client.CheckAccessToken(ctx, "dev_code")
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}

	// Step 4: access_denied
	_, err = client.CheckAccessToken(ctx, "dev_code")
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}

	// Step 5: success
	tokenResp, err := client.CheckAccessToken(ctx, "dev_code")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if tokenResp.AccessToken != "gho_test_token_123" {
		t.Errorf("expected gho_test_token_123, got %s", tokenResp.AccessToken)
	}
}

func TestPollForAccessToken_Flow(t *testing.T) {
	var count int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&count, 1)
		w.Header().Set("Content-Type", "application/json")

		if c == 1 {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "authorization_pending",
			})
		} else if c == 2 {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "slow_down",
			})
		} else {
			_ = json.NewEncoder(w).Encode(TokenResponse{
				AccessToken: "gho_success_token",
				TokenType:   "bearer",
				Scope:       "repo",
			})
		}
	}))
	defer server.Close()

	client := NewClient("test-client-id")
	client.BaseURL = server.URL
	client.PollIntervalUnit = 5 * time.Millisecond

	tokenResp, err := client.PollForAccessToken(context.Background(), "dev_code", 1, 100)
	if err != nil {
		t.Fatalf("unexpected error in poll: %v", err)
	}
	if tokenResp.AccessToken != "gho_success_token" {
		t.Errorf("expected gho_success_token, got %s", tokenResp.AccessToken)
	}

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected 3 requests during poll, got %d", count)
	}
}

func TestPollForAccessToken_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "authorization_pending",
		})
	}))
	defer server.Close()

	client := NewClient("test-client-id")
	client.BaseURL = server.URL
	client.PollIntervalUnit = 5 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.PollForAccessToken(ctx, "dev_code", 1, 5)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, ErrTimeout) {
		t.Errorf("expected deadline exceeded or ErrTimeout, got %v", err)
	}
}
