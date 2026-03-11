package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/spf13/cobra"
)

var paymentsUpdateCmd = &cobra.Command{
	Use:   "update <payment_id>",
	Short: "Update a payment (status only)",
	Long:  "Update a payment. Xero only supports setting Status=DELETED for payments.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		paymentID := strings.TrimSpace(args[0])
		if paymentID == "" {
			return validationError("payment_id is required")
		}

		bodyAttrs, err := parsePaymentBodyObject(cmd)
		if err != nil {
			return err
		}

		payment := cloneMap(bodyAttrs)

		if cmd.Flags().Changed("status") {
			status, _ := cmd.Flags().GetString("status")
			status = strings.TrimSpace(status)
			if status == "" {
				return validationError("--status cannot be empty")
			}
			payment["Status"] = strings.ToUpper(status)
		}

		if raw, ok := payment["Status"].(string); ok && strings.TrimSpace(raw) != "" {
			status := strings.ToUpper(strings.TrimSpace(raw))
			if status != "DELETED" {
				return validationError("only Status=DELETED is supported for payments")
			}
			payment["Status"] = status
		}

		if len(payment) == 0 || !hasKey(payment, "Status") {
			return validationError("payment status is required; use --status DELETED or --body")
		}

		payload, err := json.Marshal(payment)
		if err != nil {
			return fmt.Errorf("failed to build payment payload: %w", err)
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
	paymentsUpdateCmd.Flags().String("status", "", "Payment status (Xero supports DELETED)")
	addStructuredInputFlags(paymentsUpdateCmd, "Raw JSON object of payment attributes")
	paymentsUpdateCmd.Flags().String("body", "", "Raw JSON object of payment attributes")
	paymentsUpdateCmd.Flags().String("idempotency-key", "", "Idempotency key for safely retrying requests")
}
