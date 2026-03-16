package httpclient

import (
	"net/http"
	"time"
)

// New returns an HTTP client that preserves Go's default proxy behavior while
// allowing callers to set timeouts explicitly.
func New(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment

	client := &http.Client{
		Transport: transport,
	}
	if timeout > 0 {
		client.Timeout = timeout
	}

	return client
}
