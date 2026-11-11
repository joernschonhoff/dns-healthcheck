# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.27.4-alpine3.23@sha256:146e3b9c7c1e2ec3d3f5348c8a686151a75e1976e50efac9fc022ebe446a9a82 AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /dns-healthcheck .

# --- final stage: scratch image, binary only ---
FROM scratch

COPY --from=builder /dns-healthcheck /dns-healthcheck

ENTRYPOINT ["/dns-healthcheck"]
