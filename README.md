# telegramreceiver

**telegramreceiver** is a production‑ready, Go 1.24+ library and companion example application that lets you consume Telegram bot updates via an HTTPS webhook, forward them into your own code (queues, databases, OpenAI calls …) and run resiliently in Docker or Kubernetes.

### This librarry can receive only text messages!

---

## ✨ What you get

| Capability            | Details                                                                                            |
| --------------------- | -------------------------------------------------------------------------------------------------- |
| Secure webhook server | HTTPS (TLS 1.2+), optional Cloudflare client‑certs, host check, Telegram secret‑token check        |
| Resilience            | Rate‑limiter (\_golang.org/x/time/rate\_), circuit‑breaker (\_sony/gobreaker\_), graceful shutdown |
| Performance           | Request‑body max‑size guard, sync.Pool buffer reuse, buffered update channel                       |
| Observability         | Go 1.24 `log/slog` JSON logs, structured errors                                                    |
| Configurable          | Everything via env‑vars → Config struct (defaults supplied)                                        |
| Container‑ready       | Multi‑stage Dockerfile, Docker‑Compose example, environment file sample                            |
| Tests                 | Unit tests for config loader (more welcome!)                                                       |

---

## 📦 Installation

```bash
go get github.com/prilive-com/telegramreceiver/telegramreceiver
```

*(The trailing `/telegramreceiver` is required because the library code lives in that sub‑folder.)*

---

## 🏗️ Architecture

```
┌────────┐   HTTPS Webhook   ┌────────────┐        Channel         ┌──────────────┐
│Telegram│ ────────────────▶ │   Handler   │ ─────────────────────▶ │Your App Logic│
└────────┘                   │(rate‑limit)│                        └──────────────┘
                             │(circuit‑br)│
                             └────────────┘
```

* **WebhookHandler**

  * Validates host, secret token, HTTP method.
  * Rate‑limits & circuit‑breaks.
  * Reads JSON update into pooled buffer → forwards whole raw update on `Updates` channel.
* **StartWebhookServer**

  * Small HTTPS server with configurable read/write/idle timeouts.
  * Graceful shutdown on `context.Cancel`.
* **Config**

  * Populated entirely from environment variables with sensible defaults.

---

## ⚡ Quick start (example application)

1. **Clone** repository and edit `.env` with your real paths & tokens.
2. **TLS**: place valid cert/key in `tls/` (or point to external volume).
3. **Build & run**:

   ```bash
   docker compose build
   docker compose up -d
   docker compose logs -f telegram-bot
   ```
4. **Register** webhook with Telegram:

   ```bash
   curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
        -H "Content-Type: application/json" \
        -d '{"url":"https://your.domain.com:8443","secret_token":"<WEBHOOK_SECRET>"}'
   ```
5. Send a message to your bot → you’ll see pretty‑printed JSON in the logs.

Full walk‑through lives in the README [Docker Compose section](#docker-compose-deploy).

---

## 🔑 All environment variables

| Variable                                        | Default                     | Description                                           |
| ----------------------------------------------- | --------------------------- | ----------------------------------------------------- |
| `WEBHOOK_PORT`                                  | `8443`                      | HTTPS listen port inside the container                |
| `TLS_CERT_PATH`, `TLS_KEY_PATH`                 | *(none)*                    | Paths **inside the container** to certificate and key |
| `LOG_FILE_PATH`                                 | `logs/telegramreceiver.log` | File plus JSON logs to stdout                         |
| `WEBHOOK_SECRET`                                |                             | Secret token sent back to Telegram for verification   |
| `ALLOWED_DOMAIN`                                |                             | If set, request’s `Host` header must match            |
| `RATE_LIMIT_REQUESTS`                           |  `10`                       | Allowed requests per second                           |
| `RATE_LIMIT_BURST`                              |  `20`                       | Extra burst tokens                                    |
| `MAX_BODY_SIZE`                                 |  `1048576`                  | Max bytes read from body (1 MiB default)              |
| `READ_TIMEOUT`, `WRITE_TIMEOUT`, `IDLE_TIMEOUT` | `10s`,`15s`,`120s`          | Connection timeouts                                   |
| `BREAKER_MAX_REQUESTS`                          |  `5`                        | Requests allowed in half‑open state                   |
| `BREAKER_INTERVAL`                              |  `2m`                       | Window that resets failure counters                   |
| `BREAKER_TIMEOUT`                               |  `60s`                      | How long breaker stays open                           |

---

## 🧩 Integrating the library in **your** code

```go
import (
    "log/slog"
    "github.com/prilive-com/telegramreceiver/telegramreceiver"
)

func main() {
    cfg, _ := telegramreceiver.LoadConfig()

    logger, _ := telegramreceiver.NewLogger(slog.LevelInfo, cfg.LogFilePath)
    updates := make(chan telegramreceiver.TelegramUpdate, 100)

    handler := telegramreceiver.NewWebhookHandler(
        logger,
        cfg.WebhookSecret,
        cfg.AllowedDomain,
        updates,
        cfg.RateLimitRequests,
        cfg.RateLimitBurst,
        cfg.MaxBodySize,
        cfg.BreakerMaxRequests,
        cfg.BreakerInterval,
        cfg.BreakerTimeout,
    )

    go telegramreceiver.StartWebhookServer(context.Background(), cfg, handler, logger)

    for upd := range updates {
        // 🪄  do something interesting
    }
}
```

Add your own business logic right after reading from `updates`.

---

## 🐳 Docker Compose deploy <a name="docker-compose-deploy"></a>

See `Dockerfile`, `docker-compose.yml`, and `.env` in the repo for a ready‑made deployment that you can adapt for production CI/CD pipelines.

---

## 🗺️ Roadmap / contributions

| Planned                                | Status |
| -------------------------------------- | ------ |
| Benchmark suite for buffer‑pool tuning | ─      |
| Kubernetes Helm chart                  | coming |
| CI workflow (Go Vet/Lint/Test)         | coming |

PRs & issues are welcome—please follow conventional commits and run `go vet && go test` before submitting.

---

## 📜 License

MIT © 2025 Prilive Com
