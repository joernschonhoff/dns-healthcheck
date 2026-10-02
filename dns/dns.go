// Package dns wraps golang.org/x/net/dns/dnsmessage to provide minimal
// DNS query building and response RCODE parsing for health checks.
package dns

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"golang.org/x/net/dns/dnsmessage"
)

// RCODE values as defined in RFC 1035.
const (
	RcodeNoError  = int(dnsmessage.RCodeSuccess)
	RcodeServFail = int(dnsmessage.RCodeServerFailure)
	RcodeNXDomain = int(dnsmessage.RCodeNameError)
	RcodeRefused  = int(dnsmessage.RCodeRefused)
)

// BuildQuery encodes a DNS A-record query in wire format.
func BuildQuery(name string) ([]byte, error) {
	var idBytes [2]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return nil, fmt.Errorf("generating query id: %w", err)
	}
	id := binary.BigEndian.Uint16(idBytes[:])

	// dnsmessage requires a fully-qualified name (trailing dot).
	fqdn := name
	if len(fqdn) == 0 || fqdn[len(fqdn)-1] != '.' {
		fqdn += "."
	}

	qname, err := dnsmessage.NewName(fqdn)
	if err != nil {
		return nil, fmt.Errorf("invalid dns name %q: %w", name, err)
	}

	buf := make([]byte, 0, 64)
	b := dnsmessage.NewBuilder(buf, dnsmessage.Header{
		ID:               id,
		RecursionDesired: true,
	})

	if err := b.StartQuestions(); err != nil {
		return nil, err
	}
	if err := b.Question(dnsmessage.Question{
		Name:  qname,
		Type:  dnsmessage.TypeA,
		Class: dnsmessage.ClassINET,
	}); err != nil {
		return nil, err
	}

	msg, err := b.Finish()
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// ParseRcode parses the RCODE from a raw DNS response.
func ParseRcode(resp []byte) (int, error) {
	var p dnsmessage.Parser
	hdr, err := p.Start(resp)
	if err != nil {
		return 0, fmt.Errorf("parsing dns response: %w", err)
	}
	return int(hdr.RCode), nil
}

// RcodeName returns the standard DNS name for a known RCODE (e.g. "NXDOMAIN").
// Falls back to dnsmessage's own representation for unknown codes.
func RcodeName(rcode int) string {
	switch rcode {
	case RcodeNoError:
		return "NOERROR"
	case RcodeServFail:
		return "SERVFAIL"
	case RcodeNXDomain:
		return "NXDOMAIN"
	case RcodeRefused:
		return "REFUSED"
	default:
		return dnsmessage.RCode(rcode).String()
	}
}
