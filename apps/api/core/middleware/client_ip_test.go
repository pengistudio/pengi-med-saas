package core_middleware

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
)

func clientIPRouter(t *testing.T, proxies []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := ConfigureClientIP(r, proxies); err != nil {
		t.Fatalf("configure: %v", err)
	}
	r.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.ClientIP()) })
	return r
}

func clientIP(r *gin.Engine, remoteAddr string, headers map[string]string) string {
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remoteAddr
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Body.String()
}

func TestClientIP_UntrustedPeerCannotSpoofForwardedFor(t *testing.T) {
	r := clientIPRouter(t, ParseTrustedProxies(""))
	got := clientIP(r, "203.0.113.7:4321", map[string]string{"X-Forwarded-For": "1.2.3.4"})
	if got != "203.0.113.7" {
		t.Fatalf("ClientIP = %q, want the remote address 203.0.113.7", got)
	}
}

func TestClientIP_TrustedProxyForwardsClient(t *testing.T) {
	r := clientIPRouter(t, ParseTrustedProxies(""))
	// Caddy in the compose bridge network.
	got := clientIP(r, "172.18.0.5:4321", map[string]string{"X-Forwarded-For": "198.51.100.20"})
	if got != "198.51.100.20" {
		t.Fatalf("ClientIP = %q, want the forwarded client 198.51.100.20", got)
	}
}

func TestClientIP_TrustedProxyIgnoresSpoofedHopsBeforeIt(t *testing.T) {
	r := clientIPRouter(t, ParseTrustedProxies(""))
	// A client-supplied XFF that a proxy appended to: the rightmost untrusted hop wins.
	got := clientIP(r, "172.18.0.5:4321", map[string]string{"X-Forwarded-For": "1.2.3.4, 198.51.100.20"})
	if got != "198.51.100.20" {
		t.Fatalf("ClientIP = %q, want 198.51.100.20", got)
	}
}

func TestClientIP_XRealIPIsIgnored(t *testing.T) {
	r := clientIPRouter(t, ParseTrustedProxies(""))
	got := clientIP(r, "172.18.0.5:4321", map[string]string{"X-Real-IP": "1.2.3.4"})
	if got != "172.18.0.5" {
		t.Fatalf("ClientIP = %q, want the peer 172.18.0.5", got)
	}
}

func TestClientIP_NoneTrustsNoProxy(t *testing.T) {
	r := clientIPRouter(t, ParseTrustedProxies("none"))
	got := clientIP(r, "172.18.0.5:4321", map[string]string{"X-Forwarded-For": "198.51.100.20"})
	if got != "172.18.0.5" {
		t.Fatalf("ClientIP = %q, want the peer 172.18.0.5", got)
	}
}

func TestParseTrustedProxies(t *testing.T) {
	got := ParseTrustedProxies(" 10.1.0.0/16 , 192.0.2.1 ,")
	if want := []string{"10.1.0.0/16", "192.0.2.1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := ParseTrustedProxies("NONE"); len(got) != 0 {
		t.Fatalf("none: got %v, want empty", got)
	}
}

func TestConfigureClientIP_RejectsInvalidEntry(t *testing.T) {
	if err := ConfigureClientIP(gin.New(), []string{"not-an-ip"}); err == nil {
		t.Fatal("want error for an invalid proxy entry")
	}
}
