# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.26.3-alpine3.23@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /dns-healthcheck .

# --- final stage: scratch image, binary only ---
FROM scratch

COPY --from=builder /dns-healthcheck /dns-healthcheck

ENTRYPOINT ["/dns-healthcheck"]
