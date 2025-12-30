package telegramreceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const telegramAPIBaseURL = "https://api.telegram.org/bot"

// WebhookInfo contains information about the current webhook status.
type WebhookInfo struct {
	URL                          string   `json:"url"`
	HasCustomCertificate         bool     `json:"has_custom_certificate"`
	PendingUpdateCount           int      `json:"pending_update_count"`
	IPAddress                    string   `json:"ip_address,omitempty"`
	LastErrorDate                int64    `json:"last_error_date,omitempty"`
	LastErrorMessage             string   `json:"last_error_message,omitempty"`
	LastSynchronizationErrorDate int64    `json:"last_synchronization_error_date,omitempty"`
	MaxConnections               int      `json:"max_connections,omitempty"`
	AllowedUpdates               []string `json:"allowed_updates,omitempty"`
}

// telegramResponse is the generic response structure from Telegram API.
type telegramResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
}

// setWebhookRequest is the request body for setWebhook API call.
type setWebhookRequest struct {
	URL                string   `json:"url"`
	SecretToken        string   `json:"secret_token,omitempty"`
	MaxConnections     int      `json:"max_connections,omitempty"`
	AllowedUpdates     []string `json:"allowed_updates,omitempty"`
	DropPendingUpdates bool     `json:"drop_pending_updates,omitempty"`
}

// deleteWebhookRequest is the request body for deleteWebhook API call.
type deleteWebhookRequest struct {
	DropPendingUpdates bool `json:"drop_pending_updates,omitempty"`
}

// httpClient is an interface for HTTP operations to enable testing.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultHTTPClient returns a configured HTTP client for Telegram API calls.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// SetWebhook registers a webhook URL with Telegram.
// This should be called when starting in webhook mode with a URL configured.
func SetWebhook(ctx context.Context, botToken SecretToken, webhookURL, secretToken string) error {
	return SetWebhookWithClient(ctx, defaultHTTPClient(), botToken, webhookURL, secretToken)
}

// SetWebhookWithClient registers a webhook URL using a custom HTTP client.
// Use this for testing or when you need custom HTTP configuration.
func SetWebhookWithClient(ctx context.Context, client httpClient, botToken SecretToken, webhookURL, secretToken string) error {
	reqBody := setWebhookRequest{
		URL:            webhookURL,
		SecretToken:    secretToken,
		MaxConnections: 40, // Telegram default
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return &TelegramAPIError{Description: "failed to marshal request", Err: err}
	}

	url := fmt.Sprintf("%s%s/setWebhook", telegramAPIBaseURL, botToken.Value())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &TelegramAPIError{Description: "failed to create request", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &TelegramAPIError{Description: "failed to send request", Err: err}
	}
	defer resp.Body.Close()

	return parseAPIResponse(resp)
}

// DeleteWebhook removes the current webhook from Telegram.
// This must be called before starting long polling mode.
func DeleteWebhook(ctx context.Context, botToken SecretToken, dropPendingUpdates bool) error {
	return DeleteWebhookWithClient(ctx, defaultHTTPClient(), botToken, dropPendingUpdates)
}

// DeleteWebhookWithClient removes the webhook using a custom HTTP client.
// Use this for testing or when you need custom HTTP configuration.
func DeleteWebhookWithClient(ctx context.Context, client httpClient, botToken SecretToken, dropPendingUpdates bool) error {
	reqBody := deleteWebhookRequest{
		DropPendingUpdates: dropPendingUpdates,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return &TelegramAPIError{Description: "failed to marshal request", Err: err}
	}

	url := fmt.Sprintf("%s%s/deleteWebhook", telegramAPIBaseURL, botToken.Value())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &TelegramAPIError{Description: "failed to create request", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return &TelegramAPIError{Description: "failed to send request", Err: err}
	}
	defer resp.Body.Close()

	return parseAPIResponse(resp)
}

// GetWebhookInfo retrieves information about the current webhook configuration.
// Useful for diagnostics and verifying webhook status.
func GetWebhookInfo(ctx context.Context, botToken SecretToken) (*WebhookInfo, error) {
	return GetWebhookInfoWithClient(ctx, defaultHTTPClient(), botToken)
}

// GetWebhookInfoWithClient retrieves webhook info using a custom HTTP client.
// Use this for testing or when you need custom HTTP configuration.
func GetWebhookInfoWithClient(ctx context.Context, client httpClient, botToken SecretToken) (*WebhookInfo, error) {
	url := fmt.Sprintf("%s%s/getWebhookInfo", telegramAPIBaseURL, botToken.Value())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &TelegramAPIError{Description: "failed to create request", Err: err}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, &TelegramAPIError{Description: "failed to send request", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &TelegramAPIError{Description: "failed to read response", Err: err}
	}

	var telegramResp telegramResponse
	if err := json.Unmarshal(respBody, &telegramResp); err != nil {
		return nil, &TelegramAPIError{Description: "failed to parse response", Err: err}
	}

	if !telegramResp.OK {
		return nil, &TelegramAPIError{
			Code:        telegramResp.ErrorCode,
			Description: telegramResp.Description,
		}
	}

	var info WebhookInfo
	if err := json.Unmarshal(telegramResp.Result, &info); err != nil {
		return nil, &TelegramAPIError{Description: "failed to parse webhook info", Err: err}
	}

	return &info, nil
}

// parseAPIResponse handles the common Telegram API response parsing.
func parseAPIResponse(resp *http.Response) error {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &TelegramAPIError{Description: "failed to read response", Err: err}
	}

	var telegramResp telegramResponse
	if err := json.Unmarshal(respBody, &telegramResp); err != nil {
		return &TelegramAPIError{Description: "failed to parse response", Err: err}
	}

	if !telegramResp.OK {
		return &TelegramAPIError{
			Code:        telegramResp.ErrorCode,
			Description: telegramResp.Description,
		}
	}

	return nil
}
