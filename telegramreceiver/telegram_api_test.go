package telegramreceiver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
}

func newTestHandler(updates chan TelegramUpdate) *WebhookHandler {
	return NewWebhookHandler(
		newTestLogger(),
		"test-secret",
		"",
		updates,
		100,   // rate limit requests/sec
		200,   // rate limit burst
		1<<20, // max body size
		5,
		2*time.Minute,
		60*time.Second,
	)
}

func TestWebhookHandler_ValidRequest(t *testing.T) {
	updates := make(chan TelegramUpdate, 10)
	handler := newTestHandler(updates)

	update := TelegramUpdate{
		UpdateID: 12345,
		Message: &Message{
			MessageID: 1,
			Text:      "Hello, World!",
			Chat:      &Chat{ID: 100, Type: "private"},
			From:      &User{ID: 1, FirstName: "Test"},
		},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	select {
	case received := <-updates:
		if received.UpdateID != 12345 {
			t.Errorf("expected update_id 12345, got %d", received.UpdateID)
		}
		if received.Message.Text != "Hello, World!" {
			t.Errorf("expected text 'Hello, World!', got '%s'", received.Message.Text)
		}
	default:
		t.Error("expected update in channel, got none")
	}
}

func TestWebhookHandler_InvalidSecret(t *testing.T) {
	updates := make(chan TelegramUpdate, 10)
	handler := newTestHandler(updates)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	updates := make(chan TelegramUpdate, 10)
	handler := newTestHandler(updates)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rec.Code)
	}
}

func TestWebhookHandler_InvalidJSON(t *testing.T) {
	updates := make(chan TelegramUpdate, 10)
	handler := newTestHandler(updates)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("not valid json")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestWebhookHandler_ChannelBlocked(t *testing.T) {
	// Unbuffered channel will block
	updates := make(chan TelegramUpdate)
	handler := newTestHandler(updates)

	update := TelegramUpdate{UpdateID: 1}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}
}

func TestWebhookHandler_RateLimiting(t *testing.T) {
	updates := make(chan TelegramUpdate, 100)
	// Very low rate limit: 1 request per second, burst of 2
	handler := NewWebhookHandler(
		newTestLogger(),
		"test-secret",
		"",
		updates,
		1, // 1 request/sec
		2, // burst of 2
		1<<20,
		5,
		2*time.Minute,
		60*time.Second,
	)

	update := TelegramUpdate{UpdateID: 1}
	body, _ := json.Marshal(update)

	// First two requests should succeed (burst)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	// Third request should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}
}

func TestWebhookHandler_DomainValidation(t *testing.T) {
	updates := make(chan TelegramUpdate, 10)
	handler := NewWebhookHandler(
		newTestLogger(),
		"test-secret",
		"allowed.example.com",
		updates,
		100,
		200,
		1<<20,
		5,
		2*time.Minute,
		60*time.Second,
	)

	update := TelegramUpdate{UpdateID: 1}
	body, _ := json.Marshal(update)

	// Request with wrong host
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Host = "wrong.example.com"
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", rec.Code)
	}

	// Request with correct host
	req = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Host = "allowed.example.com"
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestWebhookHandler_CallbackQuery(t *testing.T) {
	updates := make(chan TelegramUpdate, 10)
	handler := newTestHandler(updates)

	update := TelegramUpdate{
		UpdateID: 99,
		CallbackQuery: &CallbackQuery{
			ID:   "callback-123",
			Data: "button_clicked",
			From: &User{ID: 42, FirstName: "Test"},
		},
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	select {
	case received := <-updates:
		if received.CallbackQuery == nil {
			t.Fatal("expected callback query, got nil")
		}
		if received.CallbackQuery.Data != "button_clicked" {
			t.Errorf("expected data 'button_clicked', got '%s'", received.CallbackQuery.Data)
		}
	default:
		t.Error("expected update in channel, got none")
	}
}

func TestWebhookHandler_ConcurrentRequests(t *testing.T) {
	updates := make(chan TelegramUpdate, 100)
	handler := newTestHandler(updates)

	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		i := i // capture loop variable
		wg.Go(func() {
			update := TelegramUpdate{UpdateID: i}
			body, _ := json.Marshal(update)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	if successCount != 10 {
		t.Errorf("expected 10 successful requests, got %d", successCount)
	}

	// Drain the channel and count
	close(updates)
	count := 0
	for range updates {
		count++
	}
	if count != 10 {
		t.Errorf("expected 10 updates in channel, got %d", count)
	}
}
