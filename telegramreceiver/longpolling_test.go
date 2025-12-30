package telegramreceiver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLongPollingClient_Start(t *testing.T) {
	tests := []struct {
		name          string
		deleteWebhook func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
	}{
		{
			name: "successful start after webhook deletion",
			deleteWebhook: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"ok":     true,
					"result": true,
				})
			},
			wantErr: false,
		},
		{
			name: "fails when webhook deletion fails",
			deleteWebhook: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]any{
					"ok":          false,
					"error_code":  401,
					"description": "Unauthorized",
				})
			},
			wantErr:     true,
			errContains: "delete webhook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.URL.Path, "deleteWebhook") {
					tt.deleteWebhook(w, r)
					return
				}
				if strings.Contains(r.URL.Path, "getUpdates") {
					requestCount++
					// Return empty updates to keep polling
					json.NewEncoder(w).Encode(map[string]any{
						"ok":     true,
						"result": []TelegramUpdate{},
					})
					return
				}
			}))
			defer server.Close()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			updates := make(chan TelegramUpdate, 10)

			client := NewLongPollingClient(
				SecretToken("test-token"),
				updates,
				logger,
				1, // short timeout for tests
				10,
				100*time.Millisecond,
				5,
				time.Minute,
				time.Minute,
				WithHTTPClient(&http.Client{
					Timeout: 5 * time.Second,
					Transport: &testTransport{
						baseURL:    server.URL,
						httpClient: server.Client(),
					},
				}),
			)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := client.Start(ctx)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Let it poll once
			time.Sleep(200 * time.Millisecond)
			client.Stop()

			if !client.Running() == false {
				t.Error("expected client to be stopped")
			}
		})
	}
}

func TestLongPollingClient_DoubleStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"ok":     true,
			"result": []TelegramUpdate{},
		})
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithHTTPClient(&http.Client{
			Transport: &testTransport{
				baseURL:    server.URL,
				httpClient: server.Client(),
			},
		}),
	)

	ctx := context.Background()

	// First start should succeed
	if err := client.Start(ctx); err != nil {
		t.Fatalf("first start failed: %v", err)
	}
	defer client.Stop()

	// Second start should fail
	err := client.Start(ctx)
	if err != ErrPollingAlreadyRunning {
		t.Errorf("expected ErrPollingAlreadyRunning, got %v", err)
	}
}

func TestLongPollingClient_ReceivesUpdates(t *testing.T) {
	updateCounter := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "deleteWebhook") {
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
			return
		}
		if strings.Contains(r.URL.Path, "getUpdates") {
			updateCounter++
			if updateCounter == 1 {
				// Return some updates on first call
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"result": []map[string]any{
						{
							"update_id": 100,
							"message": map[string]any{
								"message_id": 1,
								"text":       "Hello",
								"chat":       map[string]any{"id": 123, "type": "private"},
								"from":       map[string]any{"id": 456, "first_name": "Test"},
								"date":       1234567890,
							},
						},
						{
							"update_id": 101,
							"message": map[string]any{
								"message_id": 2,
								"text":       "World",
								"chat":       map[string]any{"id": 123, "type": "private"},
								"from":       map[string]any{"id": 456, "first_name": "Test"},
								"date":       1234567891,
							},
						},
					},
				})
			} else {
				// Return empty on subsequent calls
				json.NewEncoder(w).Encode(map[string]any{
					"ok":     true,
					"result": []TelegramUpdate{},
				})
			}
			return
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithHTTPClient(&http.Client{
			Transport: &testTransport{
				baseURL:    server.URL,
				httpClient: server.Client(),
			},
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer client.Stop()

	// Collect updates
	var received []TelegramUpdate
	timeout := time.After(2 * time.Second)

	for {
		select {
		case update := <-updates:
			received = append(received, update)
			if len(received) >= 2 {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:

	if len(received) != 2 {
		t.Errorf("expected 2 updates, got %d", len(received))
	}

	if len(received) > 0 && received[0].Message.Text != "Hello" {
		t.Errorf("expected first message text 'Hello', got %q", received[0].Message.Text)
	}

	if len(received) > 1 && received[1].Message.Text != "World" {
		t.Errorf("expected second message text 'World', got %q", received[1].Message.Text)
	}
}

func TestLongPollingClient_GracefulShutdown(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		if strings.Contains(r.URL.Path, "deleteWebhook") {
			json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
			return
		}
		// Simulate long polling delay
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{
			"ok":     true,
			"result": []TelegramUpdate{},
		})
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithHTTPClient(&http.Client{
			Transport: &testTransport{
				baseURL:    server.URL,
				httpClient: server.Client(),
			},
		}),
	)

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	// Let it run for a bit
	time.Sleep(300 * time.Millisecond)

	// Stop should complete without hanging
	done := make(chan struct{})
	go func() {
		client.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Stop() timed out")
	}

	if client.Running() {
		t.Error("client should not be running after Stop()")
	}
}

func TestLongPollingClient_IsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithHTTPClient(&http.Client{
			Transport: &testTransport{
				baseURL:    server.URL,
				httpClient: server.Client(),
			},
		}),
		WithMaxErrors(5),
	)

	// Not running yet - should not be healthy
	if client.IsHealthy() {
		t.Error("client should not be healthy before starting")
	}

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	defer client.Stop()

	// Running - should be healthy
	if !client.IsHealthy() {
		t.Error("client should be healthy when running")
	}

	// Check consecutive errors starts at 0
	if client.ConsecutiveErrors() != 0 {
		t.Errorf("expected 0 consecutive errors, got %d", client.ConsecutiveErrors())
	}
}

func TestLongPollingClient_WithMaxErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithMaxErrors(3),
	)

	if client.maxErrors != 3 {
		t.Errorf("expected maxErrors=3, got %d", client.maxErrors)
	}
}

func TestLongPollingClient_WithAllowedUpdates(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	allowedTypes := []string{"message", "callback_query"}
	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithAllowedUpdates(allowedTypes),
	)

	if len(client.allowedUpdates) != 2 {
		t.Errorf("expected 2 allowed updates, got %d", len(client.allowedUpdates))
	}
	if client.allowedUpdates[0] != "message" {
		t.Errorf("expected first allowed update 'message', got %q", client.allowedUpdates[0])
	}
}

func TestLongPollingClient_UnlimitedErrors(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithMaxErrors(0), // Unlimited
	)

	if client.maxErrors != 0 {
		t.Errorf("expected maxErrors=0, got %d", client.maxErrors)
	}
}

func TestLongPollingClient_DoubleStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": true})
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	updates := make(chan TelegramUpdate, 10)

	client := NewLongPollingClient(
		SecretToken("test-token"),
		updates,
		logger,
		1,
		10,
		100*time.Millisecond,
		5,
		time.Minute,
		time.Minute,
		WithHTTPClient(&http.Client{
			Transport: &testTransport{
				baseURL:    server.URL,
				httpClient: server.Client(),
			},
		}),
	)

	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	// Double stop should not panic (sync.Once protection)
	client.Stop()
	client.Stop() // Should not panic
}

// testTransport intercepts HTTP requests and redirects them to the test server.
type testTransport struct {
	baseURL    string
	httpClient *http.Client
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to our test server
	newURL := t.baseURL + req.URL.Path
	if req.URL.RawQuery != "" {
		newURL += "?" + req.URL.RawQuery
	}

	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header

	return t.httpClient.Transport.RoundTrip(newReq)
}
