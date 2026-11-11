package app

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func proxyPrefixes(values []string) ([]netip.Prefix, error) {
	prefixes := []netip.Prefix{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(value)
		if err != nil || prefix.Bits() == 0 {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS harus berisi CIDR proxy yang spesifik, bukan seluruh internet")
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

// X-Real-IP is accepted only from an explicitly configured proxy. The supplied
// Caddy configuration overwrites it with the socket peer, discarding user input.
// X-Forwarded-For is intentionally not consumed as an unverified header chain.
func (a *App) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	for _, prefix := range a.trustedProxies {
		if prefix.Contains(peer) {
			forwarded, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("X-Real-IP")))
			if err == nil && forwarded.Zone() == "" {
				return forwarded.Unmap().String()
			}
			break
		}
	}
	return peer.String()
}
