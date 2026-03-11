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

		inputPath, err := readBinaryInputPath(cmd)
		if err != nil {
			return err
		}
		if inputPath == "" {
			return validationError("--file is required")
		}

		payload, err := buildBankTransactionsPayload(inputPath)
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

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}
		headers["content-type"] = "application/json"
		if idempotency, _ := cmd.Flags().GetString("idempotency-key"); idempotency != "" {
			headers["Idempotency-Key"] = idempotency
		}

		return executeMutation("POST", endpoint, headers, payload, "")
	},
}

var bankingTransactionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List bank transactions",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		accountID, _ := cmd.Flags().GetString("account-id")
		order, _ := cmd.Flags().GetString("order")

		params := url.Values{}
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		if pageSize > 0 {
			params.Set("pageSize", fmt.Sprintf("%d", pageSize))
		}
		if accountID != "" {
			params.Set("where", fmt.Sprintf("BankAccount.AccountID==Guid(\"%s\")", accountID))
		}
		if strings.TrimSpace(order) != "" {
			params.Set("order", order)
		}

		endpoint := fmt.Sprintf("%s/BankTransactions?%s", xeroAPIBase, params.Encode())

		if resolvedOutputFormat() == outputTable {
			fmt.Println("Fetching bank transactions...")
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

		transactions := getArray(payload, "BankTransactions")
		return emitData(payload, func() {
			displayBankTransactions(transactions)
		})
	},
}

var bankingTransactionsGetCmd = &cobra.Command{
	Use:   "get <transaction_id>",
	Short: "Get a single bank transaction by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		transactionID := args[0]
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/BankTransactions/%s", xeroAPIBase, transactionID)
		if resolvedOutputFormat() == outputTable {
			fmt.Printf("Fetching bank transaction %s...\n\n", transactionID)
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
			transactions := getArray(payload, "BankTransactions")
			if len(transactions) == 0 {
				return notFoundError("bank transaction not found")
			}
			if transaction, ok := transactions[0].(map[string]any); ok {
				return emitData(payload, func() {
					displayBankTransactionDetail(transaction)
				})
			}
			return unexpectedResponseError()
		case 401:
			return authenticationExpiredError()
		case 404:
			return notFoundError("bank transaction not found")
		default:
			return apiError(status, body)
		}
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

		if resolvedOutputFormat() == outputTable {
			fmt.Println("Fetching bank accounts...")
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
			displayBankAccounts(accounts)
		})
	},
}

func init() {
	bankingCmd.AddCommand(bankingTransactionsCmd)
	bankingTransactionsCmd.AddCommand(bankingTransactionsListCmd)
	bankingTransactionsCmd.AddCommand(bankingTransactionsGetCmd)
	bankingCmd.AddCommand(bankingListAccountsCmd)
	bankingTransactionsCmd.Flags().String("file", "", "Path to JSON file containing bank transactions")
	bankingTransactionsCmd.Flags().String("input-file", "", "Path to a JSON file, or '-' to read JSON from stdin")
	bankingTransactionsCmd.Flags().Bool("summarize-errors", false, "Summarize validation errors in the response")
	bankingTransactionsCmd.Flags().Int("unitdp", 0, "Unit decimal places for line items")
	bankingTransactionsCmd.Flags().String("idempotency-key", "", "Idempotency key for safe retries")
	bankingTransactionsListCmd.Flags().Int("page", 1, "Page number for pagination")
	bankingTransactionsListCmd.Flags().Int("page-size", 100, "Number of items per page")
	bankingTransactionsListCmd.Flags().String("account-id", "", "Filter transactions by bank account ID")
	bankingTransactionsListCmd.Flags().String("order", "", "Order transactions by field, e.g. \"Date DESC\"")
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

func displayBankTransactions(items []any) {
	if len(items) == 0 {
		fmt.Println("No bank transactions found.")
		return
	}

	fmt.Printf("Found %d bank transaction(s):\n", len(items))
	fmt.Println()
	ui.PrintHeaderLine(150)
	header := ui.FormatRow(
		ui.Pad("Date", 12),
		ui.Pad("Type", 10),
		ui.Pad("Contact", 25),
		ui.Pad("Bank Account", 25),
		ui.Pad("Status", 12),
		ui.Pad("Total", 12),
		ui.Pad("Transaction ID", 40),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(150)

	for _, item := range items {
		transaction, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typeValue := stringValue(transaction, "Type", "N/A")
		status := stringValue(transaction, "Status", "N/A")
		date := formatDate(transaction["Date"])
		total := formatCurrency(transaction["Total"])
		transactionID := stringValue(transaction, "BankTransactionID", "N/A")

		contactName := "N/A"
		if contact, ok := transaction["Contact"].(map[string]any); ok {
			contactName = stringValue(contact, "Name", "N/A")
		}

		bankAccountName := "N/A"
		if bankAccount, ok := transaction["BankAccount"].(map[string]any); ok {
			bankAccountName = stringValue(bankAccount, "Name", "N/A")
		}

		row := ui.FormatRow(
			ui.Pad(date, 12),
			ui.Pad(typeValue, 10),
			ui.Pad(contactName, 25),
			ui.Pad(bankAccountName, 25),
			ui.Pad(status, 12),
			ui.Pad(total, 12),
			ui.Pad(transactionID, 40),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(150)
}

func displayBankTransactionDetail(transaction map[string]any) {
	fmt.Println("Bank Transaction Details:")
	fmt.Println()
	ui.PrintHeaderLine(90)

	fmt.Printf("Transaction ID:  %s\n", stringValue(transaction, "BankTransactionID", "N/A"))
	fmt.Printf("Type:            %s\n", stringValue(transaction, "Type", "N/A"))
	fmt.Printf("Status:          %s\n", stringValue(transaction, "Status", "N/A"))
	fmt.Printf("Date:            %s\n", formatDate(transaction["Date"]))
	fmt.Printf("Total:           %s\n", formatCurrency(transaction["Total"]))
	fmt.Printf("Reference:       %s\n", stringValue(transaction, "Reference", "N/A"))
	fmt.Printf("Currency:        %s\n", stringValue(transaction, "CurrencyCode", "N/A"))
	fmt.Printf("Line Amounts:    %s\n", stringValue(transaction, "LineAmountTypes", "N/A"))

	contactName := "N/A"
	contactID := "N/A"
	if contact, ok := transaction["Contact"].(map[string]any); ok {
		contactName = stringValue(contact, "Name", "N/A")
		contactID = stringValue(contact, "ContactID", "N/A")
	}
	fmt.Printf("Contact:         %s\n", contactName)
	fmt.Printf("Contact ID:      %s\n", contactID)

	bankAccountName := "N/A"
	bankAccountID := "N/A"
	if bankAccount, ok := transaction["BankAccount"].(map[string]any); ok {
		bankAccountName = stringValue(bankAccount, "Name", "N/A")
		bankAccountID = stringValue(bankAccount, "AccountID", "N/A")
	}
	fmt.Printf("Bank Account:    %s\n", bankAccountName)
	fmt.Printf("Bank Account ID: %s\n", bankAccountID)

	ui.PrintHeaderLine(90)
}
