package telegramreceiver

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
)

/* ---------- types ---------- */

type WebhookHandler struct {
	logger        *slog.Logger
	webhookSecret string
	allowedDomain string

	Updates     chan TelegramUpdate
	limiter     *rate.Limiter
	breaker     *gobreaker.CircuitBreaker
	bufferPool  sync.Pool
	maxBodySize int64
}

/* ---------- constructor ---------- */

// All tunables are injected – nothing hard-coded.
func NewWebhookHandler(
	logger *slog.Logger,
	webhookSecret string,
	allowedDomain string,
	updates chan TelegramUpdate,

	rateLimitReq float64,
	rateLimitBurst int,
	maxBodySize int64,

	breakerMaxReq uint32,
	breakerInterval time.Duration,
	breakerTimeout time.Duration,
) *WebhookHandler {

	cbSettings := gobreaker.Settings{
		Name:        "WebhookCircuitBreaker",
		MaxRequests: breakerMaxReq,
		Interval:    breakerInterval,
		Timeout:     breakerTimeout,
	}

	return &WebhookHandler{
		logger:        logger,
		webhookSecret: webhookSecret,
		allowedDomain: allowedDomain,
		Updates:       updates,
		limiter:       rate.NewLimiter(rate.Limit(rateLimitReq), rateLimitBurst),
		breaker:       gobreaker.NewCircuitBreaker(cbSettings),
		maxBodySize:   maxBodySize,
		bufferPool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, maxBodySize)
				return &b // store pointer to avoid SA6002 allocation warning
			},
		},
	}
}

/* ---------- HTTP handler ---------- */

func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	/* rate-limit check */
	if !wh.limiter.Allow() {
		wh.fail(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	/* everything else (wrapped by circuit-breaker) */
	_, err := wh.breaker.Execute(func() (interface{}, error) {
		/* domain + secret + method validation */
		if wh.allowedDomain != "" && r.Host != wh.allowedDomain {
			return nil, errors.New("forbidden")
		}
		if wh.webhookSecret != "" &&
			subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Telegram-Bot-Api-Secret-Token")), []byte(wh.webhookSecret)) != 1 {
			return nil, errors.New("unauthorized")
		}
		if r.Method != http.MethodPost {
			return nil, errors.New("method not allowed")
		}

		/* pooled buffer */
		bufPtr := wh.bufferPool.Get().(*[]byte)
		buffer := *bufPtr
		defer wh.bufferPool.Put(bufPtr)

		r.Body = http.MaxBytesReader(w, r.Body, wh.maxBodySize)
		n, err := io.ReadFull(r.Body, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		defer r.Body.Close()

		var upd TelegramUpdate
		if err := json.Unmarshal(buffer[:n], &upd); err != nil {
			return nil, err
		}

		select {
		case wh.Updates <- upd:
			wh.logger.Info("update forwarded", "update_id", upd.UpdateID)
		default:
			return nil, errors.New("updates channel blocked")
		}
		return nil, nil
	})

	if err != nil {
		switch err.Error() {
		case "forbidden":
			wh.fail(w, err.Error(), http.StatusForbidden)
		case "unauthorized":
			wh.fail(w, err.Error(), http.StatusUnauthorized)
		case "method not allowed":
			wh.fail(w, err.Error(), http.StatusMethodNotAllowed)
		case "updates channel blocked":
			wh.fail(w, err.Error(), http.StatusServiceUnavailable)
		default:
			wh.fail(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (wh *WebhookHandler) fail(w http.ResponseWriter, msg string, code int) {
	wh.logger.Error(msg)
	http.Error(w, msg, code)
}
