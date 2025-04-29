package telegramreceiver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

// WebhookHandler handles incoming webhook requests from Telegram.
type WebhookHandler struct {
	logger        *slog.Logger
	webhookSecret string
	messageChan   chan<- TelegramUpdate // channel to pass messages to your application logic
}

// TelegramUpdate represents the structure of incoming messages from Telegram.
type TelegramUpdate struct {
	UpdateID int             `json:"update_id"`
	Message  json.RawMessage `json:"message"` // We'll decode the message content later as needed
}

// ServeHTTP handles the incoming HTTP POST requests from Telegram.
func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		wh.logger.Warn("Invalid HTTP method", "method", r.Method)
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	if wh.webhookSecret != "" {
		if receivedSecret := r.Header.Get("X-Telegram-Bot-Api-Secret-Token"); receivedSecret != wh.webhookSecret {
			wh.logger.Warn("Webhook secret mismatch")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.logger.Error("Failed to read body", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var update TelegramUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		wh.logger.Error("Failed to unmarshal update", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	wh.logger.Info("Telegram update received", "update_id", update.UpdateID)

	// Forward entire update to your app
	select {
	case wh.messageChan <- update:
		wh.logger.Debug("Update forwarded successfully", "update_id", update.UpdateID)
	default:
		wh.logger.Error("Message channel blocked", "update_id", update.UpdateID)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}
}
