package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/spf13/cobra"
)

var paymentsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a payment",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		bodyAttrs, err := parsePaymentBodyRaw(cmd)
		if err != nil {
			return err
		}

		invoiceID, _ := cmd.Flags().GetString("invoice-id")
		creditNoteID, _ := cmd.Flags().GetString("credit-note-id")
		prepaymentID, _ := cmd.Flags().GetString("prepayment-id")
		overpaymentID, _ := cmd.Flags().GetString("overpayment-id")
		accountID, _ := cmd.Flags().GetString("account-id")
		reference, _ := cmd.Flags().GetString("reference")
		paymentType, _ := cmd.Flags().GetString("payment-type")

		invoiceID = strings.TrimSpace(invoiceID)
		creditNoteID = strings.TrimSpace(creditNoteID)
		prepaymentID = strings.TrimSpace(prepaymentID)
		overpaymentID = strings.TrimSpace(overpaymentID)
		accountID = strings.TrimSpace(accountID)
		reference = strings.TrimSpace(reference)
		paymentType = strings.TrimSpace(paymentType)

		var payment map[string]any
		var payload []byte
		batchPayload := false

		if bodyAttrs != nil {
			switch value := bodyAttrs.(type) {
				case map[string]any:
					if normalizePaymentsWrapper(value) {
						if paymentFlagsSet(cmd) {
							return validationError("--body Payments cannot be combined with payment flags")
						}
					payload, err = json.Marshal(value)
					if err != nil {
						return fmt.Errorf("failed to build payments payload: %w", err)
					}
					batchPayload = true
				} else {
					payment = cloneMap(value)
				}
				case []any:
					if paymentFlagsSet(cmd) {
						return validationError("--body array cannot be combined with payment flags")
					}
				payload, err = json.Marshal(map[string]any{"Payments": value})
				if err != nil {
					return fmt.Errorf("failed to build payments payload: %w", err)
				}
				batchPayload = true
				default:
					return validationError("input must be a JSON object or array")
				}
			}

		if payment == nil && !batchPayload {
			payment = map[string]any{}
		}

		if !batchPayload {
			targetCount := 0
			if invoiceID != "" {
				targetCount++
			}
			if creditNoteID != "" {
				targetCount++
			}
			if prepaymentID != "" {
				targetCount++
			}
			if overpaymentID != "" {
				targetCount++
			}
			if targetCount > 1 {
				return validationError("use only one of --invoice-id, --credit-note-id, --prepayment-id, or --overpayment-id")
			}

			switch {
			case invoiceID != "":
				payment["Invoice"] = map[string]any{"InvoiceID": invoiceID}
			case creditNoteID != "":
				payment["CreditNote"] = map[string]any{"CreditNoteID": creditNoteID}
			case prepaymentID != "":
				payment["Prepayment"] = map[string]any{"PrepaymentID": prepaymentID}
			case overpaymentID != "":
				payment["Overpayment"] = map[string]any{"OverpaymentID": overpaymentID}
			}

			if accountID != "" {
				payment["Account"] = map[string]any{"AccountID": accountID}
			}

			if cmd.Flags().Changed("amount") {
				amount, _ := cmd.Flags().GetFloat64("amount")
				if amount <= 0 {
					return validationError("--amount must be greater than 0")
				}
				payment["Amount"] = amount
			}

			if cmd.Flags().Changed("date") {
				dateFlag, _ := cmd.Flags().GetString("date")
				dateFlag = strings.TrimSpace(dateFlag)
				if dateFlag == "" {
					return validationError("--date cannot be empty")
				}
				if _, err := time.Parse("2006-01-02", dateFlag); err != nil {
					return validationError("invalid --date; expected YYYY-MM-DD")
				}
				payment["Date"] = dateFlag
			}

			if cmd.Flags().Changed("reference") {
				if reference == "" {
					return validationError("--reference cannot be empty")
				}
				payment["Reference"] = reference
			}

			if cmd.Flags().Changed("currency-rate") {
				rate, _ := cmd.Flags().GetFloat64("currency-rate")
				if rate <= 0 {
					return validationError("--currency-rate must be greater than 0")
				}
				payment["CurrencyRate"] = rate
			}

			if cmd.Flags().Changed("payment-type") {
				if paymentType == "" {
					return validationError("--payment-type cannot be empty")
				}
				payment["PaymentType"] = paymentType
			}

			if cmd.Flags().Changed("is-reconciled") {
				isReconciled, _ := cmd.Flags().GetBool("is-reconciled")
				payment["IsReconciled"] = isReconciled
			}

			if !hasPaymentTarget(payment) {
				return validationError("payment target is required (invoice, credit note, prepayment, or overpayment)")
			}
			if !hasKey(payment, "Account") {
				return validationError("--account-id is required (or provide Account in --body)")
			}
			if !hasKey(payment, "Amount") {
				return validationError("--amount is required (or provide Amount in --body)")
			}

			payload, err = json.Marshal(payment)
			if err != nil {
				return fmt.Errorf("failed to build payment payload: %w", err)
			}
		}

		params := url.Values{}
		if cmd.Flags().Changed("summarize-errors") {
			if !batchPayload {
				return validationError("--summarize-errors requires a Payments array in --body")
			}
			summarize, _ := cmd.Flags().GetBool("summarize-errors")
			params.Set("summarizeErrors", fmt.Sprintf("%t", summarize))
		}

		endpoint := fmt.Sprintf("%s/Payments", xeroAPIBase)
		if encoded := params.Encode(); encoded != "" {
			endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}
		headers["content-type"] = "application/json"
		if idempotency, _ := cmd.Flags().GetString("idempotency-key"); strings.TrimSpace(idempotency) != "" {
			headers["Idempotency-Key"] = strings.TrimSpace(idempotency)
		}

		return executeMutation("POST", endpoint, headers, payload, "")
	},
}

func init() {
	paymentsCreateCmd.Flags().String("invoice-id", "", "Invoice ID to apply payment to")
	paymentsCreateCmd.Flags().String("credit-note-id", "", "Credit note ID to apply payment to")
	paymentsCreateCmd.Flags().String("prepayment-id", "", "Prepayment ID to apply payment to")
	paymentsCreateCmd.Flags().String("overpayment-id", "", "Overpayment ID to apply payment to")
	paymentsCreateCmd.Flags().String("account-id", "", "Bank account ID to apply payment from")
	paymentsCreateCmd.Flags().Float64("amount", 0, "Payment amount")
	paymentsCreateCmd.Flags().String("date", "", "Payment date in YYYY-MM-DD")
	paymentsCreateCmd.Flags().String("reference", "", "Payment reference")
	paymentsCreateCmd.Flags().Float64("currency-rate", 0, "Currency rate for the payment")
	paymentsCreateCmd.Flags().String("payment-type", "", "Payment type for the payment")
	paymentsCreateCmd.Flags().Bool("is-reconciled", false, "Whether the payment is reconciled")
	addStructuredInputFlags(paymentsCreateCmd, "Raw JSON object of payment attributes or a Payments array")
	paymentsCreateCmd.Flags().String("body", "", "Raw JSON object of payment attributes or a Payments array")
	paymentsCreateCmd.Flags().Bool("summarize-errors", false, "Summarize validation errors in the response")
	paymentsCreateCmd.Flags().String("idempotency-key", "", "Idempotency key for safely retrying requests")
}

func parsePaymentBodyRaw(cmd *cobra.Command) (any, error) {
	decoded, err := parseStructuredJSONInput(cmd)
	if err != nil || decoded == nil {
		return nil, err
	}

	switch decoded.(type) {
	case map[string]any, []any:
		return decoded, nil
	default:
		return nil, validationError("input must be a JSON object or array")
	}
}

func parsePaymentBodyObject(cmd *cobra.Command) (map[string]any, error) {
	body, err := parsePaymentBodyRaw(cmd)
	if err != nil || body == nil {
		return nil, err
	}
	obj, ok := body.(map[string]any)
	if !ok {
		return nil, validationError("input must be a JSON object")
	}
	if normalizePaymentsWrapper(obj) {
		return nil, validationError("input must be a single Payment object, not Payments")
	}
	return obj, nil
}

func hasPaymentTarget(payment map[string]any) bool {
	if payment == nil {
		return false
	}
	return hasKey(payment, "Invoice") ||
		hasKey(payment, "CreditNote") ||
		hasKey(payment, "Prepayment") ||
		hasKey(payment, "Overpayment")
}

func normalizePaymentsWrapper(value map[string]any) bool {
	if hasKey(value, "Payments") {
		return true
	}
	if payments, ok := value["payments"]; ok {
		value["Payments"] = payments
		delete(value, "payments")
		return true
	}
	return false
}

func paymentFlagsSet(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("invoice-id") ||
		cmd.Flags().Changed("credit-note-id") ||
		cmd.Flags().Changed("prepayment-id") ||
		cmd.Flags().Changed("overpayment-id") ||
		cmd.Flags().Changed("account-id") ||
		cmd.Flags().Changed("amount") ||
		cmd.Flags().Changed("date") ||
		cmd.Flags().Changed("reference") ||
		cmd.Flags().Changed("currency-rate") ||
		cmd.Flags().Changed("payment-type") ||
		cmd.Flags().Changed("is-reconciled")
}
