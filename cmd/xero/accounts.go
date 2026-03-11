package main

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage accounts",
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		params := url.Values{}
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		if pageSize > 0 {
			params.Set("pageSize", fmt.Sprintf("%d", pageSize))
		}
		endpoint := fmt.Sprintf("%s/Accounts?%s", xeroAPIBase, params.Encode())

		if resolvedOutputFormat() == outputTable {
			fmt.Println("Fetching accounts...")
			fmt.Println()
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}

		client := xero.NewClient(xeroAPIBase)
		status, body, err := client.Do("GET", endpoint, headers, nil)
		if err != nil {
			return internalError("request failed", err)
		}

		if status == 401 {
			return authenticationExpiredError()
		}
		if status < 200 || status >= 300 {
			return apiError(status, body)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return parseResponseError(err)
		}

		accounts := getArray(payload, "Accounts")
		return emitData(payload, func() {
			displayAccounts(accounts)
		})
	},
}

var accountsGetCmd = &cobra.Command{
	Use:   "get <account_id>",
	Short: "Get a single account by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := args[0]
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/Accounts/%s", xeroAPIBase, accountID)
		if resolvedOutputFormat() == outputTable {
			fmt.Printf("Fetching account %s...\n\n", accountID)
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}

		client := xero.NewClient(xeroAPIBase)
		status, body, err := client.Do("GET", endpoint, headers, nil)
		if err != nil {
			return internalError("request failed", err)
		}

		switch status {
		case 200:
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				return parseResponseError(err)
			}
			accounts := getArray(payload, "Accounts")
			if len(accounts) == 0 {
				return notFoundError("account not found")
			}
			if account, ok := accounts[0].(map[string]any); ok {
				return emitData(payload, func() {
					displayAccountDetail(account)
				})
			}
			return unexpectedResponseError()
		case 401:
			return authenticationExpiredError()
		case 404:
			return notFoundError("account not found")
		default:
			return apiError(status, body)
		}
	},
}

func init() {
	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsGetCmd)
	accountsListCmd.Flags().Int("page", 1, "Page number for pagination")
	accountsListCmd.Flags().Int("page-size", 100, "Number of items per page")
}

func displayAccounts(items []any) {
	if len(items) == 0 {
		fmt.Println("No accounts found.")
		return
	}

	fmt.Printf("Found %d account(s):\n", len(items))
	fmt.Println()
	ui.PrintHeaderLine(120)
	header := ui.FormatRow(
		ui.Pad("Code", 12),
		ui.Pad("Name", 35),
		ui.Pad("Type", 20),
		ui.Pad("Account ID", 38),
		ui.Pad("Status", 12),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(120)

	for _, item := range items {
		account, ok := item.(map[string]any)
		if !ok {
			continue
		}
		code := stringValue(account, "Code", "N/A")
		name := stringValue(account, "Name", "N/A")
		typeValue := stringValue(account, "Type", "N/A")
		accountID := stringValue(account, "AccountID", "N/A")
		status := stringValue(account, "Status", "N/A")

		row := ui.FormatRow(
			ui.Pad(code, 12),
			ui.Pad(name, 35),
			ui.Pad(typeValue, 20),
			ui.Pad(accountID, 38),
			ui.Pad(status, 12),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(120)
}

func displayAccountDetail(account map[string]any) {
	fmt.Println("Account Details:")
	fmt.Println()
	ui.PrintHeaderLine(80)

	fmt.Printf("Code:            %s\n", stringValue(account, "Code", "N/A"))
	fmt.Printf("Name:            %s\n", stringValue(account, "Name", "N/A"))
	fmt.Printf("Account ID:      %s\n", stringValue(account, "AccountID", "N/A"))
	fmt.Printf("Type:            %s\n", stringValue(account, "Type", "N/A"))
	fmt.Printf("Status:          %s\n", stringValue(account, "Status", "N/A"))

	ui.PrintHeaderLine(80)
}
