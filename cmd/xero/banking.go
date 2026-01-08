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

func init() {
	bankingCmd.AddCommand(bankingTransactionsCmd)
	bankingTransactionsCmd.Flags().String("file", "", "Path to JSON file containing bank transactions")
	bankingTransactionsCmd.Flags().Bool("summarize-errors", false, "Summarize validation errors in the response")
	bankingTransactionsCmd.Flags().Int("unitdp", 0, "Unit decimal places for line items")
	bankingTransactionsCmd.Flags().String("idempotency-key", "", "Idempotency key for safe retries")
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
