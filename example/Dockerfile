# syntax=docker/dockerfile:1

# Pull the pre-built healthcheck binary from the registry
FROM registry.gitlab.com/joern_schonhoff/dns-healthcheck:latest AS healthcheck

# ── bind9 with embedded healthcheck ──────────────────────────────────────────
FROM internetsystemsconsortium/bind9:9.20@sha256:8fdd4005c2af3afd3f528650da44fea9eb4365c15f49b2ede96a6fa2257079a1

COPY --from=healthcheck /dns-healthcheck /usr/bin/dns-healthcheck

COPY named.conf    /etc/bind/named.conf
COPY zones/        /etc/bind/zones/

EXPOSE 53/udp 53/tcp

HEALTHCHECK --interval=15s --timeout=5s --retries=3 --start-period=5s \
    CMD ["/usr/bin/dns-healthcheck", \
         "--check-udp", "--check-tcp", \
         "--name", "health.example.internal"]
