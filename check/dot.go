package check

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"gitlab.com/joern_schonhoff/dns-healthcheck/dns"
)

// CheckDoT performs a DNS over TLS health check (RFC 7858).
// TLS certificate verification is always disabled — the target is always
// 127.0.0.1, which rarely has a valid IP SAN.
func CheckDoT(host string, port int, name string, timeout time.Duration) Result {
	label := fmt.Sprintf("dot:%d", port)

	query, err := dns.BuildQuery(name)
	if err != nil {
		return Result{Label: label, Name: name, Err: err}
	}

	start := time.Now()
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: timeout},
		"tcp",
		fmt.Sprintf("%s:%d", host, port),
		&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // target is always 127.0.0.1, no valid IP SAN
			ServerName:         host,
		},
	)
	if err != nil {
		return errResult(label, name, 0, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout)) //nolint:errcheck

	return exchangeStreamDNS(conn, label, name, query, start)
}
