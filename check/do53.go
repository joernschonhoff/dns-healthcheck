package check

import (
	"fmt"
	"net"
	"time"

	"gitlab.com/joern_schonhoff/dns-healthcheck/dns"
)

// CheckUDP performs a DNS over UDP (Do53) health check.
func CheckUDP(host string, port int, name string, timeout time.Duration) Result {
	label := fmt.Sprintf("udp:%d", port)

	query, err := dns.BuildQuery(name)
	if err != nil {
		return Result{Label: label, Name: name, Err: err}
	}

	start := time.Now()
	conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return errResult(label, name, 0, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout)) //nolint:errcheck

	if _, err := conn.Write(query); err != nil {
		return errResult(label, name, time.Since(start), err)
	}

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	elapsed := time.Since(start)
	if err != nil {
		return errResult(label, name, elapsed, err)
	}

	return parseResult(label, name, elapsed, buf[:n])
}

// CheckTCP performs a DNS over TCP (Do53) health check.
func CheckTCP(host string, port int, name string, timeout time.Duration) Result {
	label := fmt.Sprintf("tcp:%d", port)

	query, err := dns.BuildQuery(name)
	if err != nil {
		return Result{Label: label, Name: name, Err: err}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), timeout)
	if err != nil {
		return errResult(label, name, 0, err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(timeout)) //nolint:errcheck

	return exchangeStreamDNS(conn, label, name, query, start)
}

// exchangeStreamDNS sends a length-framed DNS query over an already-open stream
// connection (TCP or TLS), reads one response, and returns the parsed result.
func exchangeStreamDNS(conn net.Conn, label, name string, query []byte, start time.Time) Result {
	if err := writeTCPDNS(conn, query); err != nil {
		return errResult(label, name, time.Since(start), err)
	}
	resp, err := readTCPDNS(conn)
	elapsed := time.Since(start)
	if err != nil {
		return errResult(label, name, elapsed, err)
	}
	return parseResult(label, name, elapsed, resp)
}
