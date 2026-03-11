package main

import (
	"encoding/json"
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
		if resolvedOutputFormat() == outputTable {
			fmt.Printf("Fetching payment %s...\n\n", paymentID)
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
			payments := getArray(payload, "Payments")
			if len(payments) == 0 {
				return notFoundError("payment not found")
			}
			if payment, ok := payments[0].(map[string]any); ok {
				return emitData(payload, func() {
					displayPaymentDetail(payment)
				})
			}
			return unexpectedResponseError()
		case 401:
			return authenticationExpiredError()
		case 404:
			return notFoundError("payment not found")
		default:
			return apiError(status, body)
		}
	},
}
