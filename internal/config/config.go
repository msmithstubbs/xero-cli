package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	configDirName  = ".xero-cli"
	configFileName = "config.toml"
	legacyFileName = "config.json"
)

type Config struct {
	Credentials *Credentials `toml:"credentials"`
	Settings    Settings     `toml:"settings"`
}

type Settings struct {
	ClientID     string `toml:"client_id"`
	PKCEVerifier string `toml:"pkce_verifier"`
}

type Credentials struct {
	ClientID     string `toml:"client_id"`
	AccessToken  string `toml:"access_token"`
	RefreshToken string `toml:"refresh_token"`
	TenantID     string `toml:"tenant_id"`
	TenantName   string `toml:"tenant_name"`
	ExpiresIn    int64  `toml:"expires_in"`
	ObtainedAt   int64  `toml:"obtained_at"`
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

func LegacyConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName, legacyFileName), nil
}

func EnsureConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, configDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func ReadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}

	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		var cfg Config
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
		return cfg, nil
	}

	migrated, err := maybeMigrateLegacyConfig()
	if err != nil {
		return Config{}, err
	}
	if migrated {
		return ReadConfig()
	}

	return Config{}, nil
}

func WriteConfig(cfg Config) error {
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return nil
}

func GetCredentials() (*Credentials, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Credentials == nil {
		return nil, errors.New("not authenticated. Run 'xero auth login' first.")
	}
	return cfg.Credentials, nil
}

func SetCredentials(creds Credentials) error {
	cfg, _ := ReadConfig()
	cfg.Credentials = &creds
	return WriteConfig(cfg)
}

func DeleteCredentials() error {
	cfg, _ := ReadConfig()
	cfg.Credentials = nil
	return WriteConfig(cfg)
}

func GetSetting(key string) (string, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return "", err
	}
	return getSettingFromConfig(cfg, key), nil
}

func SetSetting(key, value string) error {
	cfg, _ := ReadConfig()
	setSettingOnConfig(&cfg, key, value)
	return WriteConfig(cfg)
}

func getSettingFromConfig(cfg Config, key string) string {
	switch key {
	case "client_id":
		return cfg.Settings.ClientID
	case "pkce_verifier":
		return cfg.Settings.PKCEVerifier
	default:
		return ""
	}
}

func setSettingOnConfig(cfg *Config, key, value string) {
	switch key {
	case "client_id":
		cfg.Settings.ClientID = value
	case "pkce_verifier":
		cfg.Settings.PKCEVerifier = value
	}
}

type legacyConfig struct {
	Credentials map[string]any `json:"credentials"`
	ClientID    string         `json:"client_id"`
	PKCE        string         `json:"pkce_verifier"`
}

func maybeMigrateLegacyConfig() (bool, error) {
	legacyPath, err := LegacyConfigPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(legacyPath); err != nil {
		return false, nil
	}
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return false, err
	}
	var legacy legacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return false, err
	}

	cfg := Config{}
	cfg.Settings.ClientID = legacy.ClientID
	cfg.Settings.PKCEVerifier = legacy.PKCE
	if len(legacy.Credentials) > 0 {
		creds := Credentials{}
		creds.ClientID = stringField(legacy.Credentials, "client_id")
		creds.AccessToken = stringField(legacy.Credentials, "access_token")
		creds.RefreshToken = stringField(legacy.Credentials, "refresh_token")
		creds.TenantID = stringField(legacy.Credentials, "tenant_id")
		creds.TenantName = stringField(legacy.Credentials, "tenant_name")
		creds.ExpiresIn = int64Field(legacy.Credentials, "expires_in", 1800)
		creds.ObtainedAt = int64Field(legacy.Credentials, "obtained_at", time.Now().Unix())
		cfg.Credentials = &creds
	}

	if err := WriteConfig(cfg); err != nil {
		return false, err
	}
	return true, nil
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		case []byte:
			return string(t)
		}
	}
	return ""
}

func int64Field(m map[string]any, key string, fallback int64) int64 {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case int64:
			return t
		case int:
			return int64(t)
		case float64:
			return int64(t)
		case json.Number:
			if n, err := t.Int64(); err == nil {
				return n
			}
		}
	}
	return fallback
}
