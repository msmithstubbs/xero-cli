package main

import (
	"testing"

	"github.com/msmithstubbs/xero-cli/internal/credentials"
	"github.com/spf13/cobra"
)

func TestImportedCredentialsFromEnv(t *testing.T) {
	t.Setenv("XERO_CLIENT_ID", "client")
	t.Setenv("XERO_ACCESS_TOKEN", "access")
	t.Setenv("XERO_REFRESH_TOKEN", "refresh")
	t.Setenv("XERO_EXPIRES_IN", "3600")
	t.Setenv("XERO_OBTAINED_AT", "12345")

	cmd := &cobra.Command{}
	cmd.Flags().String("client-id", "", "")
	cmd.Flags().String("access-token", "", "")
	cmd.Flags().String("refresh-token", "", "")
	cmd.Flags().Int64("expires-in", 1800, "")
	cmd.Flags().Int64("obtained-at", 0, "")

	clientID, accessToken, refreshToken, expiresIn, obtainedAt, err := importedCredentials(cmd)
	if err != nil {
		t.Fatalf("importedCredentials failed: %v", err)
	}
	if clientID != "client" || accessToken != "access" || refreshToken != "refresh" {
		t.Fatalf("unexpected imported values: %q %q %q", clientID, accessToken, refreshToken)
	}
	if expiresIn != 3600 || obtainedAt != 12345 {
		t.Fatalf("unexpected timing values: %d %d", expiresIn, obtainedAt)
	}
}

func TestGetPKCEVerifierGeneratesWhenMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cmd := &cobra.Command{}
	cmd.Flags().String("pkce-verifier", "", "")

	verifier, err := getPKCEVerifier(cmd)
	if err != nil {
		t.Fatalf("getPKCEVerifier failed: %v", err)
	}
	if verifier == "" {
		t.Fatal("expected verifier to be generated")
	}

	stored, err := credentials.GetPKCEVerifier()
	if err != nil {
		t.Fatalf("failed to reload stored verifier: %v", err)
	}
	if stored != verifier {
		t.Fatalf("expected stored verifier %q to match generated %q", stored, verifier)
	}
}
