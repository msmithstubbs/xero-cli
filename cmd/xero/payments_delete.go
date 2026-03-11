package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
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
			return validationError("payment_id is required")
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

		endpoint := fmt.Sprintf("%s/Payments/%s", xeroAPIBase, paymentID)
		return executeMutation("POST", endpoint, headers, payload, "")
	},
}

func init() {
	paymentsDeleteCmd.Flags().String("idempotency-key", "", "Idempotency key for safely retrying requests")
}
