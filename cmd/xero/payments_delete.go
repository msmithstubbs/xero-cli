package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var paymentsDeleteCmd = &cobra.Command{
	Use:   "delete <payment_id>",
	Short: "Delete a payment",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("payment_id is required")
		}
		paymentID := strings.TrimSpace(args[0])
		if paymentID == "" {
			return errors.New("payment_id is required")
		}

		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{"Status": "DELETED"})
		if err != nil {
			return fmt.Errorf("failed to build delete payload: %w", err)
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}
		headers["content-type"] = "application/json"
		if idempotency, _ := cmd.Flags().GetString("idempotency-key"); strings.TrimSpace(idempotency) != "" {
			headers["Idempotency-Key"] = strings.TrimSpace(idempotency)
		}

		client := xero.NewClient(xeroAPIBase)
		endpoint := fmt.Sprintf("%s/Payments/%s", xeroAPIBase, paymentID)
		statusCode, body, err := client.Do("POST", endpoint, headers, payload)
		if err != nil {
			return err
		}
		if statusCode == 401 {
			return errors.New("authentication failed. Please run 'xero auth login' again")
		}
		if statusCode < 200 || statusCode >= 300 {
			return fmt.Errorf("API request failed with status %d: %s", statusCode, string(body))
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
	paymentsDeleteCmd.Flags().String("idempotency-key", "", "Idempotency key for safely retrying requests")
}
