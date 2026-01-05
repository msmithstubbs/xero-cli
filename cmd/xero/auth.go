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

	"github.com/msmithstubbs/xero-cli/internal/config"
	"github.com/msmithstubbs/xero-cli/internal/oauth"
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

		authURL, err := oauth.GetAuthURL(clientID)
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

			tokenData, err := oauth.ExchangeCode(code, clientID)
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
			creds := config.Credentials{
				ClientID:     clientID,
				AccessToken:  tokenData.AccessToken,
				RefreshToken: tokenData.RefreshToken,
				TenantID:     tenant.TenantID,
				TenantName:   tenant.TenantName,
				ExpiresIn:    tokenData.ExpiresIn,
				ObtainedAt:   tokenData.ObtainedAt,
			}

			if err := config.SetCredentials(creds); err != nil {
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
		if err := config.DeleteCredentials(); err != nil {
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
		creds, err := config.GetCredentials()
		if err != nil {
			return fmt.Errorf("not authenticated: %w", err)
		}

		fmt.Println("Authenticated")
		fmt.Printf("Organization: %s\n", fallbackString(creds.TenantName, "Unknown"))
		fmt.Printf("Tenant ID: %s\n", fallbackString(creds.TenantID, "Unknown"))

		if oauth.TokenExpired(creds) {
			fmt.Println("Access token expired. It will be refreshed on next API call.")
		} else {
			fmt.Println("Access token is valid")
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
	if env := os.Getenv("XERO_CLIENT_ID"); env != "" {
		fmt.Println("Using Client ID from XERO_CLIENT_ID environment variable")
		return env, nil
	}

	stored, err := config.GetSetting("client_id")
	if err == nil && stored != "" {
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

	fmt.Print("Save Client ID for future use? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" {
		_ = config.SetSetting("client_id", clientID)
	}

	return clientID, nil
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
