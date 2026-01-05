package auth

import (
	"fmt"

	"github.com/msmithstubbs/xero-cli/internal/config"
	"github.com/msmithstubbs/xero-cli/internal/oauth"
)

func GetValidCredentials() (*config.Credentials, error) {
	creds, err := config.GetCredentials()
	if err != nil {
		return nil, err
	}

	if oauth.TokenExpired(creds) {
		fmt.Println("Access token expired. Refreshing...")
		if creds.RefreshToken == "" || creds.ClientID == "" {
			return nil, fmt.Errorf("missing refresh token or client ID")
		}
		newToken, err := oauth.RefreshToken(creds.RefreshToken, creds.ClientID)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		updated := *creds
		updated.AccessToken = newToken.AccessToken
		updated.RefreshToken = newToken.RefreshToken
		if newToken.ExpiresIn != 0 {
			updated.ExpiresIn = newToken.ExpiresIn
		}
		updated.ObtainedAt = newToken.ObtainedAt

		if err := config.SetCredentials(updated); err != nil {
			return nil, fmt.Errorf("failed to save refreshed credentials: %w", err)
		}
		fmt.Println("Token refreshed")
		return &updated, nil
	}

	return creds, nil
}
