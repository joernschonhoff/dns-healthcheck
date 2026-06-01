package dns

import (
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"health.check", false},
		{"example.com", false},
		{"a.b.c.d.example.internal", false},
		{"", false}, // root query — valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := BuildQuery(tt.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildQuery(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Parse the produced bytes back with dnsmessage to verify correctness.
			var p dnsmessage.Parser
			hdr, err := p.Start(msg)
			if err != nil {
				t.Fatalf("failed to parse produced query: %v", err)
			}
			if hdr.OpCode != 0 {
				t.Errorf("expected standard query opcode 0, got %d", hdr.OpCode)
			}
			if !hdr.RecursionDesired {
				t.Error("expected RD bit set")
			}

			q, err := p.Question()
			if err != nil {
				t.Fatalf("failed to read question: %v", err)
			}
			if q.Type != dnsmessage.TypeA {
				t.Errorf("expected QTYPE A, got %v", q.Type)
			}
			if q.Class != dnsmessage.ClassINET {
				t.Errorf("expected QCLASS IN, got %v", q.Class)
			}
		})
	}
}

func TestBuildQueryIsRandom(t *testing.T) {
	// Two queries for the same name must produce different IDs.
	a, err := BuildQuery("health.check")
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildQuery("health.check")
	if err != nil {
		t.Fatal(err)
	}
	// IDs are bytes 0-1 of the message.
	if a[0] == b[0] && a[1] == b[1] {
		t.Error("two consecutive queries produced the same ID — randomness may be broken")
	}
}

func TestParseRcode(t *testing.T) {
	tests := []struct {
		desc      string
		rcode     dnsmessage.RCode
		wantRcode int
		wantPass  bool
	}{
		{"NOERROR", dnsmessage.RCodeSuccess, RcodeNoError, true},
		{"NXDOMAIN", dnsmessage.RCodeNameError, RcodeNXDomain, true},
		{"SERVFAIL", dnsmessage.RCodeServerFailure, RcodeServFail, false},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			msg := buildResponse(t, tt.rcode)

			got, err := ParseRcode(msg)
			if err != nil {
				t.Fatalf("ParseRcode error: %v", err)
			}
			if got != tt.wantRcode {
				t.Errorf("got rcode %d, want %d", got, tt.wantRcode)
			}
		})
	}
}

func TestParseRcodeTooShort(t *testing.T) {
	_, err := ParseRcode([]byte{0x00, 0x01})
	if err == nil {
		t.Error("expected error for truncated message, got nil")
	}
}

// buildResponse constructs a minimal DNS response with the given RCODE.
func buildResponse(t *testing.T, rcode dnsmessage.RCode) []byte {
	t.Helper()
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:       1,
		Response: true,
		RCode:    rcode,
	})
	msg, err := b.Finish()
	if err != nil {
		t.Fatalf("building response: %v", err)
	}
	return msg
}
