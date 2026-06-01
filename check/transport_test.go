package check

import (
	"crypto/tls"
	"encoding/binary"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	testName    = "health.check"
	testTimeout = 3 * time.Second
)

// replyWithRcode reads one DNS query from r and writes back a response with
// the given RCODE over the same transport. For stream (TCP/TLS) connections
// the 2-byte length prefix is handled automatically.
func serveUDP(t *testing.T, pc net.PacketConn, rcode dnsmessage.RCode) {
	t.Helper()
	buf := make([]byte, 512)
	n, addr, err := pc.ReadFrom(buf)
	if err != nil {
		return
	}
	resp := buildDNSResponse(t, buf[:n], rcode)
	pc.WriteTo(resp, addr) //nolint:errcheck
}

func serveTCP(t *testing.T, ln net.Listener, rcode dnsmessage.RCode) {
	t.Helper()
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	msg, err := readTCPDNS(conn)
	if err != nil {
		t.Errorf("serveTCP readTCPDNS: %v", err)
		return
	}
	resp := buildDNSResponse(t, msg, rcode)
	writeTCPDNS(conn, resp) //nolint:errcheck
}

// buildDNSResponse copies the ID from the query and sets the given RCODE.
func buildDNSResponse(t *testing.T, query []byte, rcode dnsmessage.RCode) []byte {
	t.Helper()
	if len(query) < 2 {
		t.Fatal("query too short")
	}
	id := binary.BigEndian.Uint16(query[:2])
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:       id,
		Response: true,
		RCode:    rcode,
	})
	msg, err := b.Finish()
	if err != nil {
		t.Fatalf("buildDNSResponse: %v", err)
	}
	return msg
}

// hostPort splits "host:port" and returns the integer port.
func mustPort(t *testing.T, addr string) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort(%q): %v", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("Atoi(%q): %v", portStr, err)
	}
	return port
}

// ── UDP ───────────────────────────────────────────────────────────────────────

func TestUDP(t *testing.T) {
	for _, tc := range []struct {
		desc  string
		rcode dnsmessage.RCode
		pass  bool
	}{
		{"NOERROR passes", dnsmessage.RCodeSuccess, true},
		{"NXDOMAIN passes", dnsmessage.RCodeNameError, true},
		{"SERVFAIL fails", dnsmessage.RCodeServerFailure, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			pc, err := net.ListenPacket("udp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer pc.Close()

			go serveUDP(t, pc, tc.rcode)

			r := CheckUDP("127.0.0.1", mustPort(t, pc.LocalAddr().String()), testName, testTimeout)
			if r.Passed() != tc.pass {
				t.Errorf("Passed() = %v, want %v — result: %s", r.Passed(), tc.pass, r)
			}
		})
	}
}

func TestUDPConnectionRefused(t *testing.T) {
	// Port 1 is never open; expect a clean failure, not a panic.
	r := CheckUDP("127.0.0.1", 1, testName, time.Second)
	if r.Passed() {
		t.Error("expected failure on refused connection")
	}
}

// ── TCP ───────────────────────────────────────────────────────────────────────

func TestTCP(t *testing.T) {
	for _, tc := range []struct {
		desc  string
		rcode dnsmessage.RCode
		pass  bool
	}{
		{"NOERROR passes", dnsmessage.RCodeSuccess, true},
		{"NXDOMAIN passes", dnsmessage.RCodeNameError, true},
		{"SERVFAIL fails", dnsmessage.RCodeServerFailure, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()

			go serveTCP(t, ln, tc.rcode)

			r := CheckTCP("127.0.0.1", mustPort(t, ln.Addr().String()), testName, testTimeout)
			if r.Passed() != tc.pass {
				t.Errorf("Passed() = %v, want %v — result: %s", r.Passed(), tc.pass, r)
			}
		})
	}
}

// ── DoT ───────────────────────────────────────────────────────────────────────

func TestDoT(t *testing.T) {
	// httptest.NewTLSServer gives us a TLS listener with a self-signed cert.
	// We hijack the connection at the HTTP layer to speak raw DNS-over-TLS instead.
	for _, tc := range []struct {
		desc  string
		rcode dnsmessage.RCode
		pass  bool
	}{
		{"NOERROR passes", dnsmessage.RCodeSuccess, true},
		{"NXDOMAIN passes", dnsmessage.RCodeNameError, true},
		{"SERVFAIL fails", dnsmessage.RCodeServerFailure, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			// Use a raw TLS listener so we control the DNS framing directly.
			cert := httptest.NewTLSServer(nil).TLS // borrow the test cert
			ln, err := tls.Listen("tcp", "127.0.0.1:0", cert)
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()

			go serveTCP(t, ln, tc.rcode)

			r := CheckDoT("127.0.0.1", mustPort(t, ln.Addr().String()), testName, testTimeout)
			if r.Passed() != tc.pass {
				t.Errorf("Passed() = %v, want %v — result: %s", r.Passed(), tc.pass, r)
			}
		})
	}
}

// ── DoH ───────────────────────────────────────────────────────────────────────

func TestDoH(t *testing.T) {
	for _, tc := range []struct {
		desc  string
		rcode dnsmessage.RCode
		pass  bool
	}{
		{"NOERROR passes", dnsmessage.RCodeSuccess, true},
		{"NXDOMAIN passes", dnsmessage.RCodeNameError, true},
		{"SERVFAIL fails", dnsmessage.RCodeServerFailure, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			rcode := tc.rcode // capture for closure
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := readAll(r)
				if err != nil {
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				resp := buildDNSResponse(t, body, rcode)
				w.Header().Set("Content-Type", "application/dns-message")
				w.WriteHeader(http.StatusOK)
				w.Write(resp) //nolint:errcheck
			}))
			defer srv.Close()

			r := CheckDoH("127.0.0.1", mustPort(t, strings.TrimPrefix(srv.URL, "https://")), testName, testTimeout)
			if r.Passed() != tc.pass {
				t.Errorf("Passed() = %v, want %v — result: %s", r.Passed(), tc.pass, r)
			}
		})
	}
}

func TestDoHNon200(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	r := CheckDoH("127.0.0.1", mustPort(t, strings.TrimPrefix(srv.URL, "https://")), testName, testTimeout)
	if r.Passed() {
		t.Error("expected failure on HTTP 404")
	}
}

func readAll(r *http.Request) ([]byte, error) {
	var buf [4096]byte
	n, err := r.Body.Read(buf[:])
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}
	return buf[:n], nil
}
