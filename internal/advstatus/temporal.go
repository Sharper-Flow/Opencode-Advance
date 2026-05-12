package advstatus

import (
	"net"
	"os"
	"strings"
	"time"
)

// TemporalHealthProbe reads temporal.env from the cache dir and probes the
// address via TCP. Returns empty TemporalHealth (not error) when temporal.env
// is missing or ADV state dir doesn't exist (graceful degradation).
func TemporalHealthProbe() (*TemporalHealth, error) {
	addr := readTemporalAddress(cacheDir())
	if addr == "" {
		return nil, nil
	}

	// Verify ADV state dir exists before caring about Temporal health.
	advRoot := advStateRoot()
	if advRoot == "" {
		return nil, nil
	}
	if _, err := os.Stat(advRoot); err != nil {
		return nil, nil
	}

	host, port := parseTemporalAddress(addr)

	// Validate host/port before dialing (prevent injection).
	if !isValidHost(host) || !isValidPort(port) {
		return nil, nil
	}

	reachable := probeTCP(host, port)
	glyph := "T:✗"
	if reachable {
		glyph = "T:✓"
	}

	return &TemporalHealth{
		Reachable: reachable,
		Address:   addr,
		Glyph:     glyph,
	}, nil
}

// TemporalHealthProbeWithDir is the testable variant that accepts explicit
// cache dir and ADV root paths instead of reading from env.
func TemporalHealthProbeWithDir(cacheDirOverride, advRootOverride string) (*TemporalHealth, error) {
	addr := readTemporalAddress(cacheDirOverride)
	if addr == "" {
		return nil, nil
	}

	if advRootOverride != "" {
		if _, err := os.Stat(advRootOverride); err != nil {
			return nil, nil
		}
	}

	host, port := parseTemporalAddress(addr)
	if !isValidHost(host) || !isValidPort(port) {
		return nil, nil
	}

	reachable := probeTCP(host, port)
	glyph := "T:✗"
	if reachable {
		glyph = "T:✓"
	}

	return &TemporalHealth{
		Reachable: reachable,
		Address:   addr,
		Glyph:     glyph,
	}, nil
}

// parseTemporalAddress splits an address into host and port, handling
// IPv6 bracket notation. Defaults port to 7233 when absent.
func parseTemporalAddress(addr string) (host, port string) {
	if strings.HasPrefix(addr, "[") {
		closeIdx := strings.Index(addr, "]")
		if closeIdx < 0 {
			return addr, "7233"
		}
		host = addr[1:closeIdx]
		rest := addr[closeIdx+1:]
		if strings.HasPrefix(rest, ":") {
			port = rest[1:]
		} else {
			port = "7233"
		}
		return host, port
	}

	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 {
		return addr, "7233"
	}
	return addr[:lastColon], addr[lastColon+1:]
}

// isValidHost checks that a hostname contains only safe characters.
func isValidHost(host string) bool {
	if host == "" {
		return false
	}
	for _, c := range host {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '.' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

// isValidPort checks that a port string is numeric.
func isValidPort(port string) bool {
	if port == "" {
		return false
	}
	for _, c := range port {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// probeTCP attempts a quick TCP connection for status-bar use.
func probeTCP(host, port string) bool {
	target := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", target, 25*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
