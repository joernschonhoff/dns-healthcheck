# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.27.3-alpine3.23@sha256:895e5feab11a4b3774d24ee3c9579028456b82ad37f42a9eb6f82adaf776a201 AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /dns-healthcheck .

# --- final stage: scratch image, binary only ---
FROM scratch

COPY --from=builder /dns-healthcheck /dns-healthcheck

ENTRYPOINT ["/dns-healthcheck"]
