# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.27.2-alpine3.23@sha256:2ac5c2a64f1f970b5120fe21c6a5e3d9190b196a9ead95797564738ecd07a8a2 AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /dns-healthcheck .

# --- final stage: scratch image, binary only ---
FROM scratch

COPY --from=builder /dns-healthcheck /dns-healthcheck

ENTRYPOINT ["/dns-healthcheck"]
