// Package check implements DNS health checks over UDP, TCP, DoT, and DoH.
package check

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"gitlab.com/joern_schonhoff/dns-healthcheck/dns"
)

// Result holds the outcome of a single DNS health check.
type Result struct {
	Label   string        // e.g. "udp:53", "dot:853"
	Name    string        // DNS name queried
	Rcode   int           // DNS RCODE (only meaningful when Err == nil)
	Elapsed time.Duration // round-trip time
	Err     error         // non-nil on network or protocol error
}

// Passed reports whether the check is considered healthy.
// NOERROR and NXDOMAIN both pass; SERVFAIL and network errors fail.
func (r Result) Passed() bool {
	if r.Err != nil {
		return false
	}
	return r.Rcode == dns.RcodeNoError || r.Rcode == dns.RcodeNXDomain
}

// String returns the log line for this result, e.g.:
//
//	[PASS] udp:53 → health.check (NXDOMAIN, 1.2ms)
//	[FAIL] dot:853 → health.check (timeout after 3s)
func (r Result) String() string {
	status := "PASS"
	if !r.Passed() {
		status = "FAIL"
	}
	var detail string
	if r.Err != nil {
		detail = r.Err.Error()
	} else {
		detail = fmt.Sprintf("%s, %.1fms", dns.RcodeName(r.Rcode), float64(r.Elapsed.Microseconds())/1000.0)
	}
	return fmt.Sprintf("[%s] %s \u2192 %s (%s)", status, r.Label, r.Name, detail)
}

// errResult returns a failed Result with the given error.
func errResult(label, name string, elapsed time.Duration, err error) Result {
	return Result{Label: label, Name: name, Elapsed: elapsed, Err: err}
}

// parseResult builds a Result by parsing the RCODE from a raw DNS response.
func parseResult(label, name string, elapsed time.Duration, resp []byte) Result {
	rcode, err := dns.ParseRcode(resp)
	if err != nil {
		return errResult(label, name, elapsed, err)
	}
	return Result{Label: label, Name: name, Rcode: rcode, Elapsed: elapsed}
}

// writeTCPDNS writes a DNS message prefixed with a 2-byte length (RFC 7858).
func writeTCPDNS(w io.Writer, msg []byte) error {
	var hdr [2]byte
	binary.BigEndian.PutUint16(hdr[:], uint16(len(msg)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(msg)
	return err
}

// readTCPDNS reads a length-prefixed DNS message (RFC 7858).
func readTCPDNS(r io.Reader) ([]byte, error) {
	var hdr [2]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("reading length prefix: %w", err)
	}
	buf := make([]byte, binary.BigEndian.Uint16(hdr[:]))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("reading dns message: %w", err)
	}
	return buf, nil
}
