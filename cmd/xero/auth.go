package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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
		fmt.Println("Xero CLI - OAuth 2.0 Authentication")
		fmt.Println()

		clientID, err := getClientID()
		if err != nil {
			return err
		}
		if clientID == "" {
			return errors.New("client ID is required")
		}

		codeVerifier, err := getPKCEVerifier()
		if err != nil {
			return err
		}
		if codeVerifier == "" {
			return errors.New("pkce verifier is required")
		}

		authURL, err := oauth.GetAuthURL(clientID, codeVerifier)
		if err != nil {
			return err
		}

		fmt.Println("Please visit the following URL to authorize this application:")
		fmt.Println()
		fmt.Printf("  %s\n", authURL)
		fmt.Println()

		if err := openBrowser(authURL); err == nil {
			fmt.Println("Browser opened automatically")
		} else {
			fmt.Println("Please open the URL manually in your browser")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		server, err := oauth.StartCallbackServer(ctx)
		if err != nil {
			return err
		}

		fmt.Println("\nWaiting for OAuth callback on http://localhost:8888/callback...")

		select {
		case code := <-server.CodeCh:
			cancel()
			_ = server.Server.Shutdown(context.Background())
			fmt.Println()
			fmt.Println("Authorization code received")
			fmt.Println("Exchanging code for access token...")

			tokenData, err := oauth.ExchangeCode(code, clientID, codeVerifier)
			if err != nil {
				return fmt.Errorf("failed to exchange authorization code: %w", err)
			}

			fmt.Println("Access token obtained")
			fmt.Println("Fetching Xero organizations...")

			connections, err := oauth.GetConnections(tokenData.AccessToken)
			if err != nil {
				return fmt.Errorf("failed to fetch organizations: %w", err)
			}
			if len(connections) == 0 {
				return errors.New("no Xero organizations found for this account")
			}

			tenant := connections[0]
			creds := credentials.Credentials{
				ClientID:     clientID,
				AccessToken:  tokenData.AccessToken,
				RefreshToken: tokenData.RefreshToken,
				TenantID:     tenant.TenantID,
				TenantName:   tenant.TenantName,
				ExpiresIn:    tokenData.ExpiresIn,
				ObtainedAt:   tokenData.ObtainedAt,
			}

			if err := credentials.SetCredentials(creds); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}

			fmt.Println("\nSuccessfully authenticated with Xero!")
			fmt.Printf("Organization: %s\n", tenant.TenantName)
			fmt.Printf("Tenant ID: %s\n", tenant.TenantID)
			fmt.Println("\nYou can now use the Xero CLI.")
			return nil

		case err := <-server.ErrCh:
			cancel()
			_ = server.Server.Shutdown(context.Background())
			return fmt.Errorf("authentication failed: %w", err)
		case <-ctx.Done():
			_ = server.Server.Shutdown(context.Background())
			return errors.New("authentication timed out")
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := credentials.DeleteCredentials(); err != nil {
			return fmt.Errorf("failed to logout: %w", err)
		}
		fmt.Println("Successfully logged out. Credentials removed.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := credentials.GetCredentials()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		fmt.Println("Authenticated")
		fmt.Println()

		accessToken := creds.AccessToken
		if oauth.TokenExpired(creds) {
			fmt.Println("Access token expired. Refreshing...")
			tokenData, err := oauth.RefreshToken(creds.RefreshToken, creds.ClientID)
			if err != nil {
				return fmt.Errorf("failed to refresh token: %w", err)
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
			return fmt.Errorf("failed to fetch tenants: %w", err)
		}

		fmt.Printf("Available Tenants (%d):\n", len(connections))
		fmt.Println()

		nameWidth := len("Tenant Name")
		idWidth := len("Tenant ID")
		activeWidth := len("Active")
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
			ui.Pad("Active", activeWidth),
		)
		fmt.Println(header)
		ui.PrintHeaderLine(nameWidth + idWidth + activeWidth + 6)

		for _, conn := range connections {
			name := fallbackString(conn.TenantName, "Unknown")
			id := fallbackString(conn.TenantID, "Unknown")
			active := ""
			if conn.TenantID == creds.TenantID {
				active = "yes"
			}
			row := ui.FormatRow(
				ui.Pad(name, nameWidth),
				ui.Pad(id, idWidth),
				ui.Pad(active, activeWidth),
			)
			fmt.Println(row)
		}

		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

func getClientID() (string, error) {
	stored, err := credentials.GetClientID()
	if err != nil {
		return "", err
	}
	if stored != "" {
		if len(stored) > 8 {
			fmt.Printf("Using saved Client ID: %s...\n", stored[:8])
		} else {
			fmt.Println("Using saved Client ID")
		}
		return stored, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your Xero Client ID: ")
	clientID, _ := reader.ReadString('\n')
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return "", nil
	}

	if err := credentials.SetClientID(clientID); err != nil {
		return "", fmt.Errorf("failed to save client ID: %w", err)
	}

	return clientID, nil
}

func getPKCEVerifier() (string, error) {
	stored, err := credentials.GetPKCEVerifier()
	if err != nil {
		return "", err
	}
	if stored != "" {
		fmt.Println("Using saved PKCE verifier")
		return stored, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your PKCE verifier: ")
	verifier, _ := reader.ReadString('\n')
	verifier = strings.TrimSpace(verifier)
	if verifier == "" {
		return "", nil
	}

	if err := credentials.SetPKCEVerifier(verifier); err != nil {
		return "", fmt.Errorf("failed to save PKCE verifier: %w", err)
	}

	return verifier, nil
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
