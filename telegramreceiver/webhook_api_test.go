package telegramreceiver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSetWebhook(t *testing.T) {
	tests := []struct {
		name        string
		handler     func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful webhook registration",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "setWebhook") {
					t.Errorf("expected setWebhook in path, got %s", r.URL.Path)
				}

				body, _ := io.ReadAll(r.Body)
				var req setWebhookRequest
				json.Unmarshal(body, &req)

				if req.URL != "https://example.com/webhook" {
					t.Errorf("expected URL https://example.com/webhook, got %s", req.URL)
				}
				if req.SecretToken != "secret123" {
					t.Errorf("expected secret token secret123, got %s", req.SecretToken)
				}

				json.NewEncoder(w).Encode(map[string]any{
					"ok":     true,
					"result": true,
				})
			},
			wantErr: false,
		},
		{
			name: "telegram API error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{
					"ok":          false,
					"error_code":  400,
					"description": "Bad Request: invalid URL",
				})
			},
			wantErr:     true,
			errContains: "invalid URL",
		},
		{
			name: "unauthorized error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{
					"ok":          false,
					"error_code":  401,
					"description": "Unauthorized",
				})
			},
			wantErr:     true,
			errContains: "Unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer server.Close()

			client := &http.Client{
				Transport: &testTransport{
					baseURL:    server.URL,
					httpClient: server.Client(),
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := SetWebhookWithClient(ctx, client, SecretToken("test-token"), "https://example.com/webhook", "secret123")

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
			}
		})
	}
}

func TestDeleteWebhook(t *testing.T) {
	tests := []struct {
		name               string
		dropPendingUpdates bool
		handler            func(w http.ResponseWriter, r *http.Request)
		wantErr            bool
	}{
		{
			name:               "successful deletion without dropping updates",
			dropPendingUpdates: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var req deleteWebhookRequest
				json.Unmarshal(body, &req)

				if req.DropPendingUpdates != false {
					t.Error("expected DropPendingUpdates to be false")
				}

				json.NewEncoder(w).Encode(map[string]any{
					"ok":     true,
					"result": true,
				})
			},
			wantErr: false,
		},
		{
			name:               "successful deletion with dropping updates",
			dropPendingUpdates: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				var req deleteWebhookRequest
				json.Unmarshal(body, &req)

				if req.DropPendingUpdates != true {
					t.Error("expected DropPendingUpdates to be true")
				}

				json.NewEncoder(w).Encode(map[string]any{
					"ok":     true,
					"result": true,
				})
			},
			wantErr: false,
		},
		{
			name:               "API error",
			dropPendingUpdates: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{
					"ok":          false,
					"error_code":  401,
					"description": "Unauthorized",
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer server.Close()

			client := &http.Client{
				Transport: &testTransport{
					baseURL:    server.URL,
					httpClient: server.Client(),
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := DeleteWebhookWithClient(ctx, client, SecretToken("test-token"), tt.dropPendingUpdates)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetWebhookInfo(t *testing.T) {
	tests := []struct {
		name        string
		handler     func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		wantURL     string
		wantPending int
	}{
		{
			name: "successful info retrieval",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}

				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"result": map[string]any{
						"url":                    "https://example.com/webhook",
						"has_custom_certificate": false,
						"pending_update_count":   5,
						"max_connections":        40,
					},
				})
			},
			wantErr:     false,
			wantURL:     "https://example.com/webhook",
			wantPending: 5,
		},
		{
			name: "no webhook set",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{
					"ok": true,
					"result": map[string]any{
						"url":                    "",
						"has_custom_certificate": false,
						"pending_update_count":   0,
					},
				})
			},
			wantErr:     false,
			wantURL:     "",
			wantPending: 0,
		},
		{
			name: "API error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]any{
					"ok":          false,
					"error_code":  401,
					"description": "Unauthorized",
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.handler))
			defer server.Close()

			client := &http.Client{
				Transport: &testTransport{
					baseURL:    server.URL,
					httpClient: server.Client(),
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			info, err := GetWebhookInfoWithClient(ctx, client, SecretToken("test-token"))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if info.URL != tt.wantURL {
				t.Errorf("expected URL %q, got %q", tt.wantURL, info.URL)
			}
			if info.PendingUpdateCount != tt.wantPending {
				t.Errorf("expected pending count %d, got %d", tt.wantPending, info.PendingUpdateCount)
			}
		})
	}
}

func TestTelegramAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TelegramAPIError
		expected string
	}{
		{
			name: "with code and description",
			err: &TelegramAPIError{
				Code:        401,
				Description: "Unauthorized",
			},
			expected: "telegram API error [401]: Unauthorized",
		},
		{
			name: "with wrapped error",
			err: &TelegramAPIError{
				Code:        500,
				Description: "Internal error",
				Err:         io.EOF,
			},
			expected: "telegram API error [500]: Internal error: EOF",
		},
		{
			name: "description only",
			err: &TelegramAPIError{
				Description: "Something went wrong",
			},
			expected: "telegram API error: Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
