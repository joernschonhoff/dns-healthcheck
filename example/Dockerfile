# syntax=docker/dockerfile:1

# Pull the pre-built healthcheck binary from the registry
FROM registry.gitlab.com/joern_schonhoff/dns-healthcheck:latest AS healthcheck

# ── bind9 with embedded healthcheck ──────────────────────────────────────────
FROM internetsystemsconsortium/bind9:9.21@sha256:0410063274548dbba85093c1a05182a8c4b077417465492d0d35c008c60bb618

COPY --from=healthcheck /dns-healthcheck /usr/bin/dns-healthcheck

COPY named.conf    /etc/bind/named.conf
COPY zones/        /etc/bind/zones/

EXPOSE 53/udp 53/tcp

HEALTHCHECK --interval=15s --timeout=5s --retries=3 --start-period=5s \
    CMD ["/usr/bin/dns-healthcheck", \
         "--check-udp", "--check-tcp", \
         "--name", "health.example.internal"]
