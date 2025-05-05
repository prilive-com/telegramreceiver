package telegramreceiver

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	WebhookPort        int
	TLSCertPath        string
	TLSKeyPath         string
	LogFilePath        string
	WebhookSecret      string
	AllowedDomain      string
	RateLimitRequests  float64
	RateLimitBurst     int
	MaxBodySize        int64
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	IdleTimeout        time.Duration
	BreakerMaxRequests uint32
	BreakerInterval    time.Duration
	BreakerTimeout     time.Duration
}

func LoadConfig() (*Config, error) {
	webhookPort, err := strconv.Atoi(getEnv("WEBHOOK_PORT", "8443"))
	if err != nil {
		return nil, err
	}

	rateLimitRequests, err := strconv.ParseFloat(getEnv("RATE_LIMIT_REQUESTS", "10"), 64)
	if err != nil {
		return nil, err
	}

	rateLimitBurst, err := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "20"))
	if err != nil {
		return nil, err
	}

	maxBodySize, err := strconv.ParseInt(getEnv("MAX_BODY_SIZE", "1048576"), 10, 64)
	if err != nil {
		return nil, err
	}

	readTimeout, err := time.ParseDuration(getEnv("READ_TIMEOUT", "10s"))
	if err != nil {
		return nil, err
	}

	writeTimeout, err := time.ParseDuration(getEnv("WRITE_TIMEOUT", "15s"))
	if err != nil {
		return nil, err
	}

	idleTimeout, err := time.ParseDuration(getEnv("IDLE_TIMEOUT", "120s"))
	if err != nil {
		return nil, err
	}

	breakerMaxRequests, err := strconv.ParseUint(getEnv("BREAKER_MAX_REQUESTS", "5"), 10, 32)
	if err != nil {
		return nil, err
	}

	breakerInterval, err := time.ParseDuration(getEnv("BREAKER_INTERVAL", "2m"))
	if err != nil {
		return nil, err
	}

	breakerTimeout, err := time.ParseDuration(getEnv("BREAKER_TIMEOUT", "60s"))
	if err != nil {
		return nil, err
	}

	return &Config{
		WebhookPort:        webhookPort,
		TLSCertPath:        getEnv("TLS_CERT_PATH", ""),
		TLSKeyPath:         getEnv("TLS_KEY_PATH", ""),
		LogFilePath:        getEnv("LOG_FILE_PATH", "logs/telegramreceiver.log"),
		WebhookSecret:      getEnv("WEBHOOK_SECRET", ""),
		AllowedDomain:      getEnv("ALLOWED_DOMAIN", ""),
		RateLimitRequests:  rateLimitRequests,
		RateLimitBurst:     rateLimitBurst,
		MaxBodySize:        maxBodySize,
		ReadTimeout:        readTimeout,
		WriteTimeout:       writeTimeout,
		IdleTimeout:        idleTimeout,
		BreakerMaxRequests: uint32(breakerMaxRequests),
		BreakerInterval:    breakerInterval,
		BreakerTimeout:     breakerTimeout,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
