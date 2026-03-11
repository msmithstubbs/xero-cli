package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/msmithstubbs/xero-cli/internal/credentials"
	"github.com/msmithstubbs/xero-cli/internal/oauth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Xero",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Xero via OAuth 2.0",
	RunE: func(cmd *cobra.Command, args []string) error {
		if resolvedOutputFormat() == outputTable {
			fmt.Println("Xero CLI - OAuth 2.0 Authentication")
			fmt.Println()
		}

		clientID, err := getClientID(cmd)
		if err != nil {
			return err
		}
		if clientID == "" {
			return validationError("client ID is required; provide --client-id or set XERO_CLIENT_ID")
		}

		codeVerifier, err := getPKCEVerifier(cmd)
		if err != nil {
			return err
		}
		if codeVerifier == "" {
			return validationError("pkce verifier is required")
		}

		authURL, err := oauth.GetAuthURL(clientID, codeVerifier)
		if err != nil {
			return err
		}

		if resolvedOutputFormat() == outputTable {
			fmt.Println("Please visit the following URL to authorize this application:")
			fmt.Println()
			fmt.Printf("  %s\n", authURL)
			fmt.Println()
		}

		noBrowser, _ := cmd.Flags().GetBool("no-browser")
		authCode, _ := cmd.Flags().GetString("auth-code")
		authCode = strings.TrimSpace(authCode)
		if authCode == "" && !noBrowser {
			if err := openBrowser(authURL); err == nil {
				if resolvedOutputFormat() == outputTable {
					fmt.Println("Browser opened automatically")
				}
			} else if resolvedOutputFormat() == outputTable {
				fmt.Println("Please open the URL manually in your browser")
			}
		}

		if authCode != "" {
			return completeLogin(clientID, codeVerifier, authCode)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		server, err := oauth.StartCallbackServer(ctx)
		if err != nil {
			return err
		}

		if resolvedOutputFormat() == outputTable {
			fmt.Println("\nWaiting for OAuth callback on http://localhost:8888/callback...")
		}

		select {
		case code := <-server.CodeCh:
			cancel()
			_ = server.Server.Shutdown(context.Background())
			return completeLogin(clientID, codeVerifier, code)

		case err := <-server.ErrCh:
			cancel()
			_ = server.Server.Shutdown(context.Background())
			return internalError("authentication failed", err)
		case <-ctx.Done():
			_ = server.Server.Shutdown(context.Background())
			return authError("authentication timed out")
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := credentials.DeleteCredentials(); err != nil {
			return internalError("failed to logout", err)
		}
		return emitData(map[string]any{
			"ok":      true,
			"message": "Successfully logged out. Credentials removed.",
		}, func() {
			fmt.Println("Successfully logged out. Credentials removed.")
		})
	},
}

var authImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import credentials for non-interactive use",
	RunE: func(cmd *cobra.Command, args []string) error {
		clientID, accessToken, refreshToken, expiresIn, obtainedAt, err := importedCredentials(cmd)
		if err != nil {
			return err
		}

		if err := credentials.SetCredentials(credentials.Credentials{
			ClientID:     clientID,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    expiresIn,
			ObtainedAt:   obtainedAt,
		}); err != nil {
			return internalError("failed to save credentials", err)
		}

		return emitData(map[string]any{
			"ok":      true,
			"message": "Credentials imported.",
		}, func() {
			fmt.Println("Credentials imported.")
		})
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := credentials.GetCredentials()
		if err != nil {
			if errors.Is(err, credentials.ErrConfigAccess) {
				fmt.Fprintln(os.Stderr, "Unable to access credentials at ~/.config/xero-cli/credentials.toml. Check permissions and try again.")
			}
			return authError("not authenticated. Run 'xero auth login' first.")
		}

		accessToken := creds.AccessToken
		if oauth.TokenExpired(creds) {
			if resolvedOutputFormat() == outputTable {
				fmt.Println("Access token expired. Refreshing...")
			}
			tokenData, err := oauth.RefreshToken(creds.RefreshToken, creds.ClientID)
			if err != nil {
				return internalError("failed to refresh token", err)
			}

			creds.AccessToken = tokenData.AccessToken
			creds.RefreshToken = tokenData.RefreshToken
			creds.ExpiresIn = tokenData.ExpiresIn
			creds.ObtainedAt = tokenData.ObtainedAt

			if err := credentials.SetCredentials(*creds); err != nil {
				return fmt.Errorf("failed to save refreshed credentials: %w", err)
			}
			accessToken = tokenData.AccessToken
		}

		connections, err := oauth.GetConnections(accessToken)
		if err != nil {
			return internalError("failed to fetch tenants", err)
		}

		if resolvedOutputFormat() != outputTable {
			return emitData(map[string]any{
				"Authenticated": true,
				"Tenants":       connections,
				"TokenExpired":  oauth.TokenExpired(creds),
			}, nil)
		}

		fmt.Println("Authenticated")
		fmt.Println()

		fmt.Printf("Available Tenants (%d):\n", len(connections))
		fmt.Println()

		nameWidth := len("Tenant Name")
		idWidth := len("Tenant ID")
		for _, conn := range connections {
			name := fallbackString(conn.TenantName, "Unknown")
			id := fallbackString(conn.TenantID, "Unknown")
			if l := len(name); l > nameWidth {
				nameWidth = l
			}
			if l := len(id); l > idWidth {
				idWidth = l
			}
		}

		header := ui.FormatRow(
			ui.Pad("Tenant Name", nameWidth),
			ui.Pad("Tenant ID", idWidth),
		)
		fmt.Println(header)
		ui.PrintHeaderLine(nameWidth + idWidth + 3)

		for _, conn := range connections {
			name := fallbackString(conn.TenantName, "Unknown")
			id := fallbackString(conn.TenantID, "Unknown")
			row := ui.FormatRow(
				ui.Pad(name, nameWidth),
				ui.Pad(id, idWidth),
			)
			fmt.Println(row)
		}

		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authImportCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)

	authLoginCmd.Flags().String("client-id", "", "Xero client ID (defaults to XERO_CLIENT_ID or saved config)")
	authLoginCmd.Flags().String("pkce-verifier", "", "PKCE verifier (defaults to XERO_PKCE_VERIFIER, saved config, or a generated value)")
	authLoginCmd.Flags().String("auth-code", "", "Authorization code to exchange directly")
	authLoginCmd.Flags().Bool("no-browser", false, "Do not open the browser automatically")

	authImportCmd.Flags().String("client-id", "", "Xero client ID (defaults to XERO_CLIENT_ID)")
	authImportCmd.Flags().String("access-token", "", "Access token (defaults to XERO_ACCESS_TOKEN)")
	authImportCmd.Flags().String("refresh-token", "", "Refresh token (defaults to XERO_REFRESH_TOKEN)")
	authImportCmd.Flags().Int64("expires-in", 1800, "Token expiry in seconds (defaults to XERO_EXPIRES_IN or 1800)")
	authImportCmd.Flags().Int64("obtained-at", 0, "Unix timestamp when the token was obtained (defaults to now or XERO_OBTAINED_AT)")
}

func getClientID(cmd *cobra.Command) (string, error) {
	stored, err := credentials.GetClientID()
	if err != nil {
		if errors.Is(err, credentials.ErrConfigAccess) {
			fmt.Fprintln(os.Stderr, "Unable to access credentials at ~/.config/xero-cli/credentials.toml. Check permissions and try again.")
		}
		return "", internalError("failed to load client ID", err)
	}

	flagValue, _ := cmd.Flags().GetString("client-id")
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		if err := credentials.SetClientID(trimmed); err != nil {
			return "", internalError("failed to save client ID", err)
		}
		return trimmed, nil
	}

	if envValue := strings.TrimSpace(os.Getenv("XERO_CLIENT_ID")); envValue != "" {
		if err := credentials.SetClientID(envValue); err != nil {
			return "", internalError("failed to save client ID", err)
		}
		return envValue, nil
	}
	if stored != "" {
		if resolvedOutputFormat() == outputTable && len(stored) > 8 {
			fmt.Printf("Using saved Client ID: %s...\n", stored[:8])
		} else if resolvedOutputFormat() == outputTable {
			fmt.Println("Using saved Client ID")
		}
		return stored, nil
	}

	if !stdoutIsTTY() {
		return "", nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your Xero Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", nil
	}

	if err := credentials.SetClientID(clientID); err != nil {
		return "", internalError("failed to save client ID", err)
	}

	return clientID, nil
}

func getPKCEVerifier(cmd *cobra.Command) (string, error) {
	stored, err := credentials.GetPKCEVerifier()
	if err != nil {
		if errors.Is(err, credentials.ErrConfigAccess) {
			fmt.Fprintln(os.Stderr, "Unable to access credentials at ~/.config/xero-cli/credentials.toml. Check permissions and try again.")
		}
		return "", internalError("failed to load PKCE verifier", err)
	}

	flagValue, _ := cmd.Flags().GetString("pkce-verifier")
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		if err := credentials.SetPKCEVerifier(trimmed); err != nil {
			return "", internalError("failed to save PKCE verifier", err)
		}
		return trimmed, nil
	}

	if envValue := strings.TrimSpace(os.Getenv("XERO_PKCE_VERIFIER")); envValue != "" {
		if err := credentials.SetPKCEVerifier(envValue); err != nil {
			return "", internalError("failed to save PKCE verifier", err)
		}
		return envValue, nil
	}
	if stored != "" {
		if resolvedOutputFormat() == outputTable {
			fmt.Println("Using saved PKCE verifier")
		}
		return stored, nil
	}

	verifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		return "", internalError("failed to generate PKCE verifier", err)
	}

	if err := credentials.SetPKCEVerifier(verifier); err != nil {
		return "", internalError("failed to save PKCE verifier", err)
	}

	return verifier, nil
}

func completeLogin(clientID, codeVerifier, code string) error {
	if resolvedOutputFormat() == outputTable {
		fmt.Println()
		fmt.Println("Authorization code received")
		fmt.Println("Exchanging code for access token...")
	}

	tokenData, err := oauth.ExchangeCode(code, clientID, codeVerifier)
	if err != nil {
		return internalError("failed to exchange authorization code", err)
	}

	if resolvedOutputFormat() == outputTable {
		fmt.Println("Access token obtained")
		fmt.Println("Fetching Xero organizations...")
	}

	connections, err := oauth.GetConnections(tokenData.AccessToken)
	if err != nil {
		return internalError("failed to fetch organizations", err)
	}
	if len(connections) == 0 {
		return notFoundError("no Xero organizations found for this account")
	}

	creds := credentials.Credentials{
		ClientID:     clientID,
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		ExpiresIn:    tokenData.ExpiresIn,
		ObtainedAt:   tokenData.ObtainedAt,
	}

	if err := credentials.SetCredentials(creds); err != nil {
		return internalError("failed to save credentials", err)
	}

	tenant := connections[0]
	return emitData(map[string]any{
		"Authenticated": true,
		"Organization":  tenant.TenantName,
		"TenantID":      tenant.TenantID,
		"TenantName":    tenant.TenantName,
	}, func() {
		fmt.Println("\nSuccessfully authenticated with Xero!")
		fmt.Printf("Organization: %s\n", tenant.TenantName)
		fmt.Printf("Tenant ID: %s\n", tenant.TenantID)
		fmt.Println("\nYou can now use the Xero CLI.")
	})
}

func importedCredentials(cmd *cobra.Command) (string, string, string, int64, int64, error) {
	clientID, _ := cmd.Flags().GetString("client-id")
	accessToken, _ := cmd.Flags().GetString("access-token")
	refreshToken, _ := cmd.Flags().GetString("refresh-token")
	expiresIn, _ := cmd.Flags().GetInt64("expires-in")
	obtainedAt, _ := cmd.Flags().GetInt64("obtained-at")

	clientID = fallbackEnv(strings.TrimSpace(clientID), "XERO_CLIENT_ID")
	accessToken = fallbackEnv(strings.TrimSpace(accessToken), "XERO_ACCESS_TOKEN")
	refreshToken = fallbackEnv(strings.TrimSpace(refreshToken), "XERO_REFRESH_TOKEN")
	if expiresIn == 1800 {
		if raw := strings.TrimSpace(os.Getenv("XERO_EXPIRES_IN")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return "", "", "", 0, 0, validationError("XERO_EXPIRES_IN must be an integer")
			}
			expiresIn = parsed
		}
	}
	if obtainedAt == 0 {
		if raw := strings.TrimSpace(os.Getenv("XERO_OBTAINED_AT")); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return "", "", "", 0, 0, validationError("XERO_OBTAINED_AT must be an integer")
			}
			obtainedAt = parsed
		} else {
			obtainedAt = time.Now().Unix()
		}
	}

	switch {
	case clientID == "":
		return "", "", "", 0, 0, validationError("client ID is required; provide --client-id or set XERO_CLIENT_ID")
	case accessToken == "":
		return "", "", "", 0, 0, validationError("access token is required; provide --access-token or set XERO_ACCESS_TOKEN")
	case refreshToken == "":
		return "", "", "", 0, 0, validationError("refresh token is required; provide --refresh-token or set XERO_REFRESH_TOKEN")
	}

	return clientID, accessToken, refreshToken, expiresIn, obtainedAt, nil
}

func fallbackEnv(value, key string) string {
	if value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(key))
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return errors.New("unknown OS")
	}

	return cmd.Start()
}

func fallbackString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
