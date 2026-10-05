package core_middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// DefaultTrustedProxies covers loopback and the private ranges Docker assigns
// to bridge networks. In production Caddy runs in the same compose network as
// the API (deploy/docker-compose.prod.yaml) and the API publishes no port, so
// only containers on that network can reach it.
const DefaultTrustedProxies = "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

// TrustedProxiesFromEnv reads TRUSTED_PROXIES (comma-separated IPs or CIDRs).
// Unset or empty means DefaultTrustedProxies; "none" trusts no proxy, so
// ClientIP is always the TCP peer (use it when the API is exposed directly).
func TrustedProxiesFromEnv() []string {
	return ParseTrustedProxies(os.Getenv("TRUSTED_PROXIES"))
}

// ParseTrustedProxies splits a TRUSTED_PROXIES value; see TrustedProxiesFromEnv.
func ParseTrustedProxies(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultTrustedProxies
	}
	if strings.EqualFold(value, "none") {
		return []string{}
	}
	var proxies []string
	for _, p := range strings.Split(value, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	return proxies
}

// ConfigureClientIP makes c.ClientIP() (per-IP rate limits, audit logs) honor
// X-Forwarded-For only when the TCP peer is one of proxies. Caddy's
// reverse_proxy sets X-Forwarded-For to the real client and drops the incoming
// value from untrusted clients; X-Real-IP is not set by Caddy, so it is not
// read (a client could otherwise send it through). An invalid entry is an error.
func ConfigureClientIP(engine *gin.Engine, proxies []string) error {
	engine.ForwardedByClientIP = true
	engine.RemoteIPHeaders = []string{"X-Forwarded-For"}
	return engine.SetTrustedProxies(proxies)
}
