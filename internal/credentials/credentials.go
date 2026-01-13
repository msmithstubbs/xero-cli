package credentials

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	keyringService   = "xero-cli"
	credentialsEntry = "credentials"
	clientIDEntry    = "client_id"
	pkceEntry        = "pkce_verifier"
)

type Credentials struct {
	ClientID     string `json:"client_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TenantID     string `json:"tenant_id"`
	TenantName   string `json:"tenant_name"`
	ExpiresIn    int64  `json:"expires_in"`
	ObtainedAt   int64  `json:"obtained_at"`
}

func GetCredentials() (*Credentials, error) {
	value, err := keyring.Get(keyringService, credentialsEntry)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, errors.New("not authenticated. Run 'xero auth login' first.")
		}
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal([]byte(value), &creds); err != nil {
		return nil, fmt.Errorf("failed to decode stored credentials: %w", err)
	}
	return &creds, nil
}

func SetCredentials(creds Credentials) error {
	payload, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	return keyring.Set(keyringService, credentialsEntry, string(payload))
}

func DeleteCredentials() error {
	err := keyring.Delete(keyringService, credentialsEntry)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}

func GetClientID() (string, error) {
	return getValue(clientIDEntry)
}

func SetClientID(clientID string) error {
	return setValue(clientIDEntry, clientID)
}

func GetPKCEVerifier() (string, error) {
	return getValue(pkceEntry)
}

func SetPKCEVerifier(verifier string) error {
	return setValue(pkceEntry, verifier)
}

func getValue(key string) (string, error) {
	value, err := keyring.Get(keyringService, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func setValue(key, value string) error {
	return keyring.Set(keyringService, key, value)
}
