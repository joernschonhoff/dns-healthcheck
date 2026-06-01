package check

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"gitlab.com/joern_schonhoff/dns-healthcheck/dns"
)

// ── Result.Passed ─────────────────────────────────────────────────────────────

func TestResultPassed(t *testing.T) {
	tests := []struct {
		desc string
		r    Result
		want bool
	}{
		{"NOERROR passes", Result{Rcode: dns.RcodeNoError}, true},
		{"NXDOMAIN passes", Result{Rcode: dns.RcodeNXDomain}, true},
		{"SERVFAIL fails", Result{Rcode: dns.RcodeServFail}, false},
		{"network error fails", Result{Err: errors.New("timeout")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.r.Passed(); got != tt.want {
				t.Errorf("Passed() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── Result.String ─────────────────────────────────────────────────────────────

func TestResultStringPass(t *testing.T) {
	r := Result{
		Label:   "udp:53",
		Name:    "health.check",
		Rcode:   dns.RcodeNXDomain,
		Elapsed: 1200 * time.Microsecond,
	}
	got := r.String()
	want := "[PASS] udp:53 → health.check (NXDOMAIN, 1.2ms)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResultStringFail(t *testing.T) {
	r := Result{
		Label: "dot:853",
		Name:  "health.check",
		Err:   errors.New("i/o timeout"),
	}
	got := r.String()
	want := "[FAIL] dot:853 → health.check (i/o timeout)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── TCP framing round-trip ────────────────────────────────────────────────────

func TestTCPFramingRoundTrip(t *testing.T) {
	messages := [][]byte{
		{0x01, 0x02, 0x03, 0x04},
		make([]byte, 512),
		{0xff},
	}

	for _, msg := range messages {
		client, server := net.Pipe()

		// Write in a goroutine so the pipe doesn't block.
		errCh := make(chan error, 1)
		go func() {
			errCh <- writeTCPDNS(client, msg)
			client.Close()
		}()

		got, err := readTCPDNS(server)
		server.Close()

		if werr := <-errCh; werr != nil {
			t.Fatalf("writeTCPDNS error: %v", werr)
		}
		if err != nil {
			t.Fatalf("readTCPDNS error: %v", err)
		}
		if len(got) != len(msg) {
			t.Fatalf("length mismatch: got %d, want %d", len(got), len(msg))
		}
		for i := range msg {
			if got[i] != msg[i] {
				t.Fatalf("byte %d mismatch: got %x, want %x", i, got[i], msg[i])
			}
		}
	}
}

func TestReadTCPDNSTruncated(t *testing.T) {
	client, server := net.Pipe()

	go func() {
		// Write a length prefix claiming 10 bytes but only send 3.
		client.Write([]byte{0x00, 0x0a, 0x01, 0x02, 0x03})
		client.Close()
	}()

	_, err := readTCPDNS(server)
	server.Close()

	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("expected io.ErrUnexpectedEOF, got %v", err)
	}
}
