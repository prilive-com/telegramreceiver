# ── build stage ───────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY . .

# download deps
RUN go mod download

# compile the *example* program (static binary, CGO disabled)
RUN CGO_ENABLED=0 go build -o /app/telegram-example ./example

# ── tiny runtime stage ────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/telegram-example /app/telegram-example

# copy TLS certs into container (compose mounts /tls on top of this)
COPY tls /tls

# expose webhook port
EXPOSE 8443

# run the example bot
CMD ["/app/telegram-example"]
