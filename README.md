# dns-healthcheck

> **Hobby project / proof of concept — not production-ready.**
> This was built for personal use and learning. It has not been security audited and comes with no guarantees. Use it at your own risk.

A small static Go binary for DNS liveness checks inside minimal or distroless containers. Drop it into any DNS server image and use it as a native Docker `HEALTHCHECK`.

## What it does

Probes a DNS server over one or more protocols and exits `0` (healthy) or `1` (unhealthy). At least one `--check-*` flag is required.

```
[PASS] udp:53  → health.check (NXDOMAIN, 1.2ms)
[FAIL] dot:853 → health.check (timeout after 1s)
[PASS] doh:443 → health.check (NOERROR, 4.1ms)
result: unhealthy (1/3 checks failed)
```

**NXDOMAIN = pass** — the server responded. **SERVFAIL = fail** — something is broken.

## Supported protocols

| Flag | Protocol | Default port |
|------|----------|-------------|
| `--check-udp` | DNS over UDP (Do53) | 53 |
| `--check-tcp` | DNS over TCP (Do53) | 53 |
| `--check-dot` | DNS over TLS (DoT) | 853 |
| `--check-doh` | DNS over HTTPS (DoH, RFC 8484 POST) | 443 |

## All flags

| Flag | Default | Description |
|------|---------|-------------|
| `--check-udp` | false | Enable UDP check |
| `--check-tcp` | false | Enable TCP check |
| `--check-dot` | false | Enable DoT check |
| `--check-doh` | false | Enable DoH check |
| `--udp-port` | 53 | UDP port |
| `--tcp-port` | 53 | TCP port |
| `--dot-port` | 853 | DoT port |
| `--doh-port` | 443 | DoH port |
| `--name` | health.check | DNS name to query (A record) |
| `--timeout` | 1s | Timeout per check |
| `--quiet` | false | Suppress output, exit code only |

## Usage

### Copy into any container image

```dockerfile
COPY --from=registry.gitlab.com/joern_schonhoff/dns-healthcheck:latest \
    /dns-healthcheck /usr/bin/dns-healthcheck
```

### Dockerfile HEALTHCHECK (dnsdist example)

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=5s \
    CMD ["/usr/bin/dns-healthcheck", "--check-udp", "--check-tcp"]
```

### docker-compose.yml

```yaml
healthcheck:
  test: ["/usr/bin/dns-healthcheck", "--check-udp", "--check-tcp"]
  interval: 30s
  timeout: 5s
  retries: 3
  start_period: 5s
```

## Build

```bash
# Local (Linux binary)
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dns-healthcheck .

# Cross-compile for arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dns-healthcheck-arm64 .
```

Requires Go 1.26+. No CGO. The resulting binary is fully static (~5 MB).

## Distribution

Tagged releases are built and published automatically via GitLab CI:

- **Container image**: `registry.gitlab.com/joern_schonhoff/dns-healthcheck:<tag>` (scratch-based, binary only)
- **Binaries**: attached to each GitLab release as `dns-healthcheck-linux-amd64` and `dns-healthcheck-linux-arm64`

## License

MIT — see [LICENSE](LICENSE).
