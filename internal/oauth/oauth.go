package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/msmithstubbs/xero-cli/internal/credentials"
)

const (
	xeroAuthURL        = "https://login.xero.com/identity/connect/authorize"
	xeroTokenURL       = "https://identity.xero.com/connect/token"
	xeroConnectionsURL = "https://api.xero.com/connections"
	redirectURI        = "http://localhost:8888/callback"
)

type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	ObtainedAt   int64  `json:"obtained_at"`
}

type Connection struct {
	TenantID   string `json:"tenantId"`
	TenantName string `json:"tenantName"`
}

func GetAuthURL(clientID string) (string, string, error) {
	codeVerifier, err := codeVerifierFromEnv()
	if err != nil {
		return "", "", err
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", "offline_access openid profile email accounting.transactions accounting.contacts accounting.settings")
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	return fmt.Sprintf("%s?%s", xeroAuthURL, params.Encode()), codeVerifier, nil
}

func ExchangeCode(code, clientID, codeVerifier string) (*TokenData, error) {
	if codeVerifier == "" {
		return nil, errors.New("code verifier not found. Please restart the authentication process")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", codeVerifier)

	data, err := postForm(xeroTokenURL, form)
	if err != nil {
		return nil, err
	}
	data.ObtainedAt = time.Now().Unix()
	return data, nil
}

func RefreshToken(refreshToken, clientID string) (*TokenData, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", clientID)

	data, err := postForm(xeroTokenURL, form)
	if err != nil {
		return nil, err
	}
	data.ObtainedAt = time.Now().Unix()
	return data, nil
}

func GetConnections(accessToken string) ([]Connection, error) {
	req, err := http.NewRequest(http.MethodGet, xeroConnectionsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("authorization", "Bearer "+accessToken)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to get connections with status %d: %s", resp.StatusCode, string(body))
	}

	var connections []Connection
	if err := json.Unmarshal(body, &connections); err != nil {
		return nil, err
	}
	return connections, nil
}

func TokenExpired(creds *credentials.Credentials) bool {
	if creds == nil || creds.ObtainedAt == 0 {
		return true
	}
	expiresIn := creds.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 1800
	}
	current := time.Now().Unix()
	return current >= creds.ObtainedAt+expiresIn-300
}

func codeVerifierFromEnv() (string, error) {
	if env := os.Getenv("XERO_PKCE_VERIFIER"); env != "" {
		return env, nil
	}
	return generateCodeVerifier()
}

type CallbackServer struct {
	Server *http.Server
	CodeCh chan string
	ErrCh  chan error
}

func StartCallbackServer(ctx context.Context) (*CallbackServer, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, failureHTML())
			errCh <- errors.New("no authorization code found")
			return
		}
		io.WriteString(w, successHTML())
		codeCh <- code
	})

	srv := &http.Server{
		Addr:              ":8888",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	return &CallbackServer{Server: srv, CodeCh: codeCh, ErrCh: errCh}, nil
}

func postForm(url string, form url.Values) (*TokenData, error) {
	resp, err := http.PostForm(url, form)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var data TokenData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func generateCodeVerifier() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func successHTML() string {
	return `<!DOCTYPE html>
<html>
<head><title>Xero CLI - Authentication Successful</title></head>
<body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
  <h1 style="color: #13B5EA;">Authentication Successful</h1>
  <p>You can close this window and return to your terminal.</p>
</body>
</html>`
}

func failureHTML() string {
	return `<!DOCTYPE html>
<html>
<head><title>Xero CLI - Authentication Failed</title></head>
<body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
  <h1 style="color: #ff0000;">Authentication Failed</h1>
  <p>No authorization code found. Please try again.</p>
</body>
</html>`
}
