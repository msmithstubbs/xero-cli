package credentials

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	configDirName  = "xero-cli"
	configFileName = "credentials.toml"
)

var ErrConfigAccess = errors.New("config access error")

type Credentials struct {
	ClientID     string `toml:"client_id"`
	AccessToken  string `toml:"access_token"`
	RefreshToken string `toml:"refresh_token"`
	ExpiresIn    int64  `toml:"expires_in"`
	ObtainedAt   int64  `toml:"obtained_at"`
}

type configFile struct {
	ClientID     string `toml:"client_id,omitempty"`
	PKCEVerifier string `toml:"pkce_verifier,omitempty"`
	AccessToken  string `toml:"access_token,omitempty"`
	RefreshToken string `toml:"refresh_token,omitempty"`
	ExpiresIn    int64  `toml:"expires_in,omitempty"`
	ObtainedAt   int64  `toml:"obtained_at,omitempty"`
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

func loadConfig() (configFile, error) {
	path, err := configPath()
	if err != nil {
		return configFile{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configFile{}, nil
		}
		return configFile{}, wrapConfigErr(path, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return configFile{}, nil
	}

	var cfg configFile
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return configFile{}, wrapConfigErr(path, fmt.Errorf("failed to decode config: %w", err))
	}

	return cfg, nil
}

func writeConfig(cfg configFile) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return wrapConfigErr(path, err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return wrapConfigErr(path, err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
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
