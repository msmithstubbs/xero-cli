package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var bankingCmd = &cobra.Command{
	Use:   "banking",
	Short: "Manage banking operations",
}

var bankingTransactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Create bank transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return errors.New("--file is required")
		}

		payload, err := buildBankTransactionsPayload(filePath)
		if err != nil {
			return err
		}

		params := url.Values{}
		if cmd.Flags().Changed("summarize-errors") {
			summarize, _ := cmd.Flags().GetBool("summarize-errors")
			params.Set("summarizeErrors", fmt.Sprintf("%t", summarize))
		}
		if cmd.Flags().Changed("unitdp") {
			unitdp, _ := cmd.Flags().GetInt("unitdp")
			if unitdp > 0 {
				params.Set("unitdp", fmt.Sprintf("%d", unitdp))
			}
		}

		endpoint := fmt.Sprintf("%s/BankTransactions", xeroAPIBase)
		if encoded := params.Encode(); encoded != "" {
			endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
		}

		headers := authHeaders(creds)
		headers["content-type"] = "application/json"
		if idempotency, _ := cmd.Flags().GetString("idempotency-key"); idempotency != "" {
			headers["Idempotency-Key"] = idempotency
		}

		client := xero.NewClient(xeroAPIBase)
		status, body, err := client.Do("POST", endpoint, headers, payload)
		if err != nil {
			return err
		}

		if status == 401 {
			return errors.New("authentication failed. Please run 'xero auth login' again")
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("API request failed with status %d: %s", status, string(body))
		}

		formatted, err := prettyJSON(body)
		if err != nil {
			fmt.Println(string(body))
			return nil
		}
		fmt.Println(formatted)
		return nil
	},
}

var bankingListAccountsCmd = &cobra.Command{
	Use:   "list-accounts",
	Short: "List bank accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		params := url.Values{}
		params.Set("where", "Type==\"BANK\"")
		endpoint := fmt.Sprintf("%s/Accounts?%s", xeroAPIBase, params.Encode())

		fmt.Println("Fetching bank accounts...")
		fmt.Println()

		headers := authHeaders(creds)
		if cmd.Flags().Changed("tenant-id") {
			if tenantID, _ := cmd.Flags().GetString("tenant-id"); strings.TrimSpace(tenantID) != "" {
				headers["xero-tenant-id"] = strings.TrimSpace(tenantID)
			}
		}

		client := xero.NewClient(xeroAPIBase)
		status, body, err := client.Do("GET", endpoint, headers, nil)
		if err != nil {
			return err
		}

		if status == 401 {
			return errors.New("authentication failed. Please run 'xero auth login' again")
		}
		if status < 200 || status >= 300 {
			return fmt.Errorf("API request failed with status %d: %s", status, string(body))
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		accounts := getArray(payload, "Accounts")
		displayBankAccounts(accounts)
		return nil
	},
}

func init() {
	bankingCmd.AddCommand(bankingTransactionsCmd)
	bankingCmd.AddCommand(bankingListAccountsCmd)
	bankingTransactionsCmd.Flags().String("file", "", "Path to JSON file containing bank transactions")
	bankingTransactionsCmd.Flags().Bool("summarize-errors", false, "Summarize validation errors in the response")
	bankingTransactionsCmd.Flags().Int("unitdp", 0, "Unit decimal places for line items")
	bankingTransactionsCmd.Flags().String("idempotency-key", "", "Idempotency key for safe retries")
	bankingListAccountsCmd.Flags().String("tenant-id", "", "Tenant ID to use for this request")
}

func buildBankTransactionsPayload(path string) ([]byte, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, errors.New("file is empty")
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	switch value := decoded.(type) {
	case []any:
		return json.Marshal(map[string]any{"BankTransactions": value})
	case map[string]any:
		if hasKey(value, "BankTransactions") || hasKey(value, "bankTransactions") {
			return json.Marshal(value)
		}
		return json.Marshal(map[string]any{"BankTransactions": []any{value}})
	default:
		return nil, errors.New("unexpected JSON format")
	}
}

func prettyJSON(input []byte) (string, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, input, "", "  "); err != nil {
		return "", err
	}
	return out.String(), nil
}

func hasKey(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}

func displayBankAccounts(items []any) {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		account, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(stringValue(account, "Type", ""), "BANK") {
			filtered = append(filtered, account)
		}
	}

	if len(filtered) == 0 {
		fmt.Println("No bank accounts found.")
		return
	}

	fmt.Printf("Found %d bank account(s):\n", len(filtered))
	fmt.Println()
	ui.PrintHeaderLine(90)
	header := ui.FormatRow(
		ui.Pad("Name", 45),
		ui.Pad("Account ID", 40),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(90)

	for _, account := range filtered {
		name := stringValue(account, "Name", "N/A")
		accountID := stringValue(account, "AccountID", "N/A")
		row := ui.FormatRow(
			ui.Pad(name, 45),
			ui.Pad(accountID, 40),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(90)
}
