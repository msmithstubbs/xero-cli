package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var paymentsGetCmd = &cobra.Command{
	Use:   "get <payment_id>",
	Short: "Get a single payment by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		paymentID := args[0]
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/Payments/%s", xeroAPIBase, paymentID)
		fmt.Printf("Fetching payment %s...\n\n", paymentID)

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
			payments := getArray(payload, "Payments")
			if len(payments) == 0 {
				fmt.Println("Payment not found.")
				return nil
			}
			if payment, ok := payments[0].(map[string]any); ok {
				displayPaymentDetail(payment)
				return nil
			}
			return errors.New("unexpected response format")
		case 401:
			return errors.New("authentication failed. Please run 'xero auth login' again")
		case 404:
			return errors.New("payment not found")
		default:
			return fmt.Errorf("API request failed with status %d: %s", status, string(body))
		}
	},
}
