package check

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"gitlab.com/joern_schonhoff/dns-healthcheck/dns"
)

// CheckDoH performs a DNS over HTTPS health check (RFC 8484 POST).
// TLS certificate verification is always disabled — the target is always
// 127.0.0.1, which rarely has a valid IP SAN.
func CheckDoH(host string, port int, name string, timeout time.Duration) Result {
	label := fmt.Sprintf("doh:%d", port)

	query, err := dns.BuildQuery(name)
	if err != nil {
		return Result{Label: label, Name: name, Err: err}
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // target is always 127.0.0.1, no valid IP SAN
			},
		},
	}

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("https://%s:%d/dns-query", host, port),
		bytes.NewReader(query),
	)
	if err != nil {
		return Result{Label: label, Name: name, Err: err}
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return errResult(label, name, elapsed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errResult(label, name, elapsed, fmt.Errorf("unexpected HTTP status: %s", resp.Status))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return errResult(label, name, elapsed, err)
	}

	return parseResult(label, name, elapsed, body)
}
