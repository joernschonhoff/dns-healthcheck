# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.26.5-alpine3.23@sha256:622e56dbc11a8cfe87cafa2331e9a201877271cbff918af53d3be315f3da88cc AS builder

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /dns-healthcheck .

# --- final stage: scratch image, binary only ---
FROM scratch

COPY --from=builder /dns-healthcheck /dns-healthcheck

ENTRYPOINT ["/dns-healthcheck"]
