package httpclient

import (
	"net/http"
	"testing"
	"time"
)

func TestNewUsesHTTPSProxyFromEnvironment(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.example:8443")
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("NO_PROXY", "")

	client := New(30 * time.Second)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.xero.com/connections", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy lookup failed: %v", err)
	}
	if proxyURL == nil {
		t.Fatal("expected proxy URL from HTTPS_PROXY")
	}
	if got := proxyURL.String(); got != "http://proxy.example:8443" {
		t.Fatalf("expected proxy URL %q, got %q", "http://proxy.example:8443", got)
	}
}
