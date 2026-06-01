package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gitlab.com/joern_schonhoff/dns-healthcheck/check"
)

func main() {
	// Protocol enable flags
	checkUDP := flag.Bool("check-udp", false, "Enable UDP (Do53) check")
	checkTCP := flag.Bool("check-tcp", false, "Enable TCP (Do53) check")
	checkDoT := flag.Bool("check-dot", false, "Enable DoT check")
	checkDoH := flag.Bool("check-doh", false, "Enable DoH check")

	// Port flags
	udpPort := flag.Int("udp-port", 53, "UDP port")
	tcpPort := flag.Int("tcp-port", 53, "TCP port")
	dotPort := flag.Int("dot-port", 853, "DoT port")
	dohPort := flag.Int("doh-port", 443, "DoH port")

	// Common flags
	name    := flag.String("name", "health.check", "DNS name to query (A record)")
	timeout := flag.Duration("timeout", 1*time.Second, "Timeout per check")
	quiet   := flag.Bool("quiet", false, "Suppress output, return exit code only")

	flag.Parse()

	if !*checkUDP && !*checkTCP && !*checkDoT && !*checkDoH {
		fmt.Fprintln(os.Stderr, "error: at least one --check-* flag must be specified")
		flag.Usage()
		os.Exit(1)
	}

	const host = "127.0.0.1"

	type entry struct {
		enabled bool
		fn      func() check.Result
	}
	checks := []entry{
		{*checkUDP, func() check.Result { return check.CheckUDP(host, *udpPort, *name, *timeout) }},
		{*checkTCP, func() check.Result { return check.CheckTCP(host, *tcpPort, *name, *timeout) }},
		{*checkDoT, func() check.Result { return check.CheckDoT(host, *dotPort, *name, *timeout) }},
		{*checkDoH, func() check.Result { return check.CheckDoH(host, *dohPort, *name, *timeout) }},
	}

	var results []check.Result
	for _, c := range checks {
		if c.enabled {
			results = append(results, c.fn())
		}
	}

	failed := 0
	for _, r := range results {
		if !r.Passed() {
			failed++
		}
		if !*quiet {
			fmt.Println(r.String())
		}
	}

	if !*quiet {
		total := len(results)
		if failed == 0 {
			fmt.Printf("result: healthy (%d/%d checks passed)\n", total, total)
		} else {
			fmt.Printf("result: unhealthy (%d/%d checks failed)\n", failed, total)
		}
	}

	if failed > 0 {
		os.Exit(1)
	}
}
