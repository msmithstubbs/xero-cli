package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = "zero-cli"
	configFileName = "tunnel"
)

var ErrConfigAccess = errors.New("config access error")

type Credentials struct {
	ClientID     string `json:"client_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	ObtainedAt   int64  `json:"obtained_at"`
}

type tunnelConfig struct {
	ClientID     string `json:"client_id,omitempty"`
	PKCEVerifier string `json:"pkce_verifier,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	ObtainedAt   int64  `json:"obtained_at,omitempty"`
}

func GetCredentials() (*Credentials, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.AccessToken == "" || cfg.RefreshToken == "" {
		return nil, errors.New("not authenticated. Run 'xero auth login' first.")
	}

	return &Credentials{
		ClientID:     cfg.ClientID,
		AccessToken:  cfg.AccessToken,
		RefreshToken: cfg.RefreshToken,
		ExpiresIn:    cfg.ExpiresIn,
		ObtainedAt:   cfg.ObtainedAt,
	}, nil
}

func SetCredentials(creds Credentials) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.ClientID = creds.ClientID
	cfg.AccessToken = creds.AccessToken
	cfg.RefreshToken = creds.RefreshToken
	cfg.ExpiresIn = creds.ExpiresIn
	cfg.ObtainedAt = creds.ObtainedAt
	return writeConfig(cfg)
}

func DeleteCredentials() error {
	cfg, err := loadConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	cfg.AccessToken = ""
	cfg.RefreshToken = ""
	cfg.ExpiresIn = 0
	cfg.ObtainedAt = 0

	if cfg.ClientID == "" && cfg.PKCEVerifier == "" {
		return removeConfigFile()
	}

	return writeConfig(cfg)
}

func GetClientID() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	return cfg.ClientID, nil
}

func SetClientID(clientID string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.ClientID = clientID
	return writeConfig(cfg)
}

func GetPKCEVerifier() (string, error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", err
	}
	return cfg.PKCEVerifier, nil
}

func SetPKCEVerifier(verifier string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	cfg.PKCEVerifier = verifier
	return writeConfig(cfg)
}

func loadConfig() (tunnelConfig, error) {
	path, err := configPath()
	if err != nil {
		return tunnelConfig{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return tunnelConfig{}, nil
		}
		return tunnelConfig{}, wrapConfigErr(path, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return tunnelConfig{}, nil
	}

	var cfg tunnelConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return tunnelConfig{}, wrapConfigErr(path, fmt.Errorf("failed to decode tunnel config: %w", err))
	}

	return cfg, nil
}

func writeConfig(cfg tunnelConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return wrapConfigErr(path, err)
	}

	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return wrapConfigErr(path, err)
	}

	return nil
}

func removeConfigFile() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return wrapConfigErr(path, err)
	}
	return nil
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrConfigAccess, err)
	}
	return filepath.Join(home, ".config", configDirName, configFileName), nil
}

func wrapConfigErr(path string, err error) error {
	return fmt.Errorf("%w: %s: %v", ErrConfigAccess, path, err)
}
