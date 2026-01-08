package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var currenciesCmd = &cobra.Command{
	Use:   "currencies",
	Short: "Manage currencies",
}

var currenciesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all currencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		where, _ := cmd.Flags().GetString("where")
		params := url.Values{}
		if where != "" {
			params.Set("where", where)
		}

		endpoint := fmt.Sprintf("%s/Currencies", xeroAPIBase)
		if encoded := params.Encode(); encoded != "" {
			endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
		}

		fmt.Println("Fetching currencies...")
		fmt.Println()

		headers, err := authHeaders(creds)
		if err != nil {
			return err
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

		currencies := getArray(payload, "Currencies")
		displayCurrencies(currencies)
		return nil
	},
}

var currenciesGetCmd = &cobra.Command{
	Use:   "get <currency_code>",
	Short: "Get a single currency by code",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		currencyCode := args[0]
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/Currencies/%s", xeroAPIBase, currencyCode)
		fmt.Printf("Fetching currency %s...\n\n", currencyCode)

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}

		client := xero.NewClient(xeroAPIBase)
		status, body, err := client.Do("GET", endpoint, headers, nil)
		if err != nil {
			return err
		}

		switch status {
		case 200:
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
			currencies := getArray(payload, "Currencies")
			if len(currencies) == 0 {
				fmt.Println("Currency not found.")
				return nil
			}
			if currency, ok := currencies[0].(map[string]any); ok {
				displayCurrencyDetail(currency)
				return nil
			}
			return errors.New("unexpected response format")
		case 401:
			return errors.New("authentication failed. Please run 'xero auth login' again")
		case 404:
			return errors.New("currency not found")
		default:
			return fmt.Errorf("API request failed with status %d: %s", status, string(body))
		}
	},
}

func init() {
	currenciesCmd.AddCommand(currenciesListCmd)
	currenciesCmd.AddCommand(currenciesGetCmd)
	currenciesListCmd.Flags().String("where", "", "Filter currencies with a where clause")
}

func displayCurrencies(items []any) {
	if len(items) == 0 {
		fmt.Println("No currencies found.")
		return
	}

	fmt.Printf("Found %d currency/currencies:\n", len(items))
	fmt.Println()
	ui.PrintHeaderLine(80)
	header := ui.FormatRow(
		ui.Pad("Code", 10),
		ui.Pad("Description", 50),
		ui.Pad("Status", 15),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(80)

	for _, item := range items {
		currency, ok := item.(map[string]any)
		if !ok {
			continue
		}
		code := stringValue(currency, "Code", "N/A")
		description := stringValue(currency, "Description", "N/A")
		status := stringValue(currency, "Status", "ACTIVE")

		row := ui.FormatRow(
			ui.Pad(code, 10),
			ui.Pad(description, 50),
			ui.Pad(status, 15),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(80)
}

func displayCurrencyDetail(currency map[string]any) {
	fmt.Println("Currency Details:")
	fmt.Println()
	ui.PrintHeaderLine(80)

	fmt.Printf("Code:        %s\n", stringValue(currency, "Code", "N/A"))
	fmt.Printf("Description: %s\n", stringValue(currency, "Description", "N/A"))
	fmt.Printf("Status:      %s\n", stringValue(currency, "Status", "ACTIVE"))

	ui.PrintHeaderLine(80)
}
