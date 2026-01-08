package main

import (
	"errors"
	"fmt"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/oauth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/spf13/cobra"
)

var tenantsCmd = &cobra.Command{
	Use:   "tenants",
	Short: "Manage Xero tenants",
}

var tenantsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tenants for the authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		connections, err := oauth.GetConnections(creds.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to fetch tenants: %w", err)
		}
		if len(connections) == 0 {
			return errors.New("no tenants found for this account")
		}

		nameWidth := len("Tenant Name")
		idWidth := len("Tenant ID")
		for _, tenant := range connections {
			if l := len(tenant.TenantName); l > nameWidth {
				nameWidth = l
			}
			if l := len(tenant.TenantID); l > idWidth {
				idWidth = l
			}
		}

		fmt.Println(ui.FormatRow(ui.Pad("Tenant Name", nameWidth), ui.Pad("Tenant ID", idWidth)))
		ui.PrintHeaderLine(nameWidth + idWidth + 3)
		for _, tenant := range connections {
			fmt.Println(ui.FormatRow(ui.Pad(tenant.TenantName, nameWidth), ui.Pad(tenant.TenantID, idWidth)))
		}
		return nil
	},
}

func init() {
	tenantsCmd.AddCommand(tenantsListCmd)
}
