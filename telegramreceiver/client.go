package telegramreceiver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// Client is the main entry point for receiving Telegram updates.
// Use New() or NewFromConfig() to create a Client.
type Client struct {
	config  ClientConfig
	updates chan TelegramUpdate

	// Internal components (created on Start)
	pollingClient  *LongPollingClient
	webhookHandler *WebhookHandler
}

// validate is the shared validator instance
var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Use json tags in error messages
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	// Register custom bot token validator
	validate.RegisterValidation("bottoken", validateBotTokenField)
}

// validateBotTokenField is a validator.Func for bot token format
func validateBotTokenField(fl validator.FieldLevel) bool {
	token := fl.Field().String()
	if token == "" {
		return true // Let 'required' handle empty
	}
	return ValidateBotToken(SecretToken(token)) == nil
}

// New creates a new Client with the given bot token and options.
// This is the recommended way to create a Client programmatically.
//
// Example:
//
//	client, err := telegramreceiver.New(os.Getenv("TELEGRAM_BOT_TOKEN"),
//	    telegramreceiver.WithMode(telegramreceiver.ModeLongPolling),
//	    telegramreceiver.WithPolling(30, 100),
//	    telegramreceiver.WithLogger(logger),
//	)
func New(botToken string, opts ...Option) (*Client, error) {
	cfg := DefaultClientConfig()
	cfg.BotToken = botToken

	// Apply options
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	// Validate
	if err := validateClientConfig(&cfg); err != nil {
		return nil, err
	}

	return newClient(cfg)
}

// NewFromConfig creates a Client by loading configuration from multiple sources.
// Configuration precedence (highest to lowest):
//  1. Programmatic options (opts...)
//  2. Environment variables (TELEGRAM_*)
//  3. Config file (if path provided)
//  4. Default values
//
// Example:
//
//	client, err := telegramreceiver.NewFromConfig("config.yaml",
//	    telegramreceiver.WithLogger(logger),  // Override from config
//	)
func NewFromConfig(configPath string, opts ...Option) (*Client, error) {
	cfg, err := LoadClientConfig(configPath, opts...)
	if err != nil {
		return nil, err
	}
	return newClient(*cfg)
}

// LoadClientConfig loads configuration from file, env vars, and applies options.
func LoadClientConfig(configPath string, opts ...Option) (*ClientConfig, error) {
	k := koanf.New(".")

	// 1. DEFAULTS (lowest priority)
	if err := k.Load(structs.Provider(DefaultClientConfig(), "koanf"), nil); err != nil {
		return nil, fmt.Errorf("loading defaults: %w", err)
	}

	// 2. CONFIG FILE (if exists)
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("loading config file: %w", err)
			}
		}
	}

	// 3. ENVIRONMENT VARIABLES (TELEGRAM_*)
	if err := k.Load(env.Provider("TELEGRAM_", ".", func(s string) string {
		// TELEGRAM_BOT_TOKEN -> bot_token
		key := strings.ToLower(strings.TrimPrefix(s, "TELEGRAM_"))
		return strings.ReplaceAll(key, "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env vars: %w", err)
	}

	// Unmarshal to struct
	var cfg ClientConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	// 4. PROGRAMMATIC OPTIONS (highest priority)
	for _, opt := range opts {
		opt.apply(&cfg)
	}

	// Validate
	if err := validateClientConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateClientConfig validates the configuration and returns user-friendly errors.
func validateClientConfig(cfg *ClientConfig) error {
	// Custom validation logic
	if cfg.BotToken == "" {
		return fmt.Errorf("bot_token: required (set via TELEGRAM_BOT_TOKEN env var)")
	}

	if err := ValidateBotToken(SecretToken(cfg.BotToken)); err != nil {
		return fmt.Errorf("bot_token: %w (format: 123456789:ABCdefGHI...)", err)
	}

	if cfg.Mode == ModeLongPolling {
		if cfg.PollingTimeout < 0 || cfg.PollingTimeout > 60 {
			return fmt.Errorf("polling_timeout: must be between 0 and 60")
		}
		if cfg.PollingLimit < 1 || cfg.PollingLimit > 100 {
			return fmt.Errorf("polling_limit: must be between 1 and 100")
		}
	}

	if cfg.Mode == ModeWebhook {
		if cfg.WebhookPort < 1 || cfg.WebhookPort > 65535 {
			return fmt.Errorf("webhook_port: must be between 1 and 65535")
		}
	}

	return nil
}

// newClient creates the internal client from validated config.
func newClient(cfg ClientConfig) (*Client, error) {
	return &Client{
		config:  cfg,
		updates: make(chan TelegramUpdate, 100),
	}, nil
}

// Config returns a copy of the client configuration.
func (c *Client) Config() ClientConfig {
	return c.config
}

// Updates returns the channel for receiving Telegram updates.
func (c *Client) Updates() <-chan TelegramUpdate {
	return c.updates
}

// Start begins receiving updates based on the configured mode.
func (c *Client) Start(ctx context.Context) error {
	switch c.config.Mode {
	case ModeLongPolling:
		return c.startPolling(ctx)
	case ModeWebhook:
		return c.startWebhook(ctx)
	default:
		return fmt.Errorf("unknown receiver mode: %s", c.config.Mode)
	}
}

// Stop gracefully stops receiving updates.
func (c *Client) Stop() {
	if c.pollingClient != nil {
		c.pollingClient.Stop()
	}
}

// IsHealthy returns health status for Kubernetes probes.
func (c *Client) IsHealthy() bool {
	if c.pollingClient != nil {
		return c.pollingClient.IsHealthy()
	}
	return true
}

// WebhookHandler returns the HTTP handler for webhook mode.
// Use this to integrate with your own HTTP server.
func (c *Client) WebhookHandler() http.Handler {
	if c.webhookHandler == nil {
		logger := c.config.Logger
		if logger == nil {
			var err error
			loggerWrapper, err := NewLogger(0, c.config.LogFilePath)
			if err == nil {
				logger = loggerWrapper.Logger
			}
		}

		c.webhookHandler = NewWebhookHandler(
			logger,
			c.config.WebhookSecret,
			c.config.AllowedDomain,
			c.updates,
			c.config.RateLimitRequests,
			c.config.RateLimitBurst,
			c.config.MaxBodySize,
			c.config.BreakerMaxRequests,
			c.config.BreakerInterval,
			c.config.BreakerTimeout,
		)
	}
	return c.webhookHandler
}

// startPolling starts the long polling client.
func (c *Client) startPolling(ctx context.Context) error {
	logger := c.config.Logger
	if logger == nil {
		loggerWrapper, err := NewLogger(0, c.config.LogFilePath)
		if err != nil {
			return fmt.Errorf("creating logger: %w", err)
		}
		logger = loggerWrapper.Logger
	}

	var opts []LongPollingOption
	if c.config.PollingMaxErrors != 10 {
		opts = append(opts, WithMaxErrors(c.config.PollingMaxErrors))
	}
	if len(c.config.AllowedUpdates) > 0 {
		opts = append(opts, WithAllowedUpdates(c.config.AllowedUpdates))
	}
	if c.config.PollingDeleteWebhook {
		opts = append(opts, WithDeleteWebhook(true))
	}
	if c.config.RetryInitialDelay > 0 || c.config.RetryMaxDelay > 0 {
		opts = append(opts, WithRetryConfig(
			c.config.RetryInitialDelay,
			c.config.RetryMaxDelay,
			c.config.RetryBackoffFactor,
		))
	}
	if c.config.HTTPClient != nil {
		opts = append(opts, WithHTTPClient(c.config.HTTPClient.(*http.Client)))
	}

	c.pollingClient = NewLongPollingClient(
		SecretToken(c.config.BotToken),
		c.updates,
		logger,
		c.config.PollingTimeout,
		c.config.PollingLimit,
		c.config.BreakerMaxRequests,
		c.config.BreakerInterval,
		c.config.BreakerTimeout,
		opts...,
	)

	return c.pollingClient.Start(ctx)
}

// startWebhook starts the webhook server.
func (c *Client) startWebhook(ctx context.Context) error {
	// For webhook, we just set up the handler
	// The actual server is started by the user or via StartWebhookServer()
	_ = c.WebhookHandler()
	return nil
}
