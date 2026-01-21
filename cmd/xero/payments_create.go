package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/xero"
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

		bodyAttrs, err := parsePaymentBody(cmd)
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

		payment := cloneMap(bodyAttrs)

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
			return errors.New("use only one of --invoice-id, --credit-note-id, --prepayment-id, or --overpayment-id")
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
				return errors.New("--amount must be greater than 0")
			}
			payment["Amount"] = amount
		}

		if cmd.Flags().Changed("date") {
			dateFlag, _ := cmd.Flags().GetString("date")
			dateFlag = strings.TrimSpace(dateFlag)
			if dateFlag == "" {
				return errors.New("--date cannot be empty")
			}
			if _, err := time.Parse("2006-01-02", dateFlag); err != nil {
				return errors.New("invalid --date; expected YYYY-MM-DD")
			}
			payment["Date"] = dateFlag
		}

		if cmd.Flags().Changed("reference") {
			if reference == "" {
				return errors.New("--reference cannot be empty")
			}
			payment["Reference"] = reference
		}

		if cmd.Flags().Changed("currency-rate") {
			rate, _ := cmd.Flags().GetFloat64("currency-rate")
			if rate <= 0 {
				return errors.New("--currency-rate must be greater than 0")
			}
			payment["CurrencyRate"] = rate
		}

		if cmd.Flags().Changed("payment-type") {
			if paymentType == "" {
				return errors.New("--payment-type cannot be empty")
			}
			payment["PaymentType"] = paymentType
		}

		if cmd.Flags().Changed("is-reconciled") {
			isReconciled, _ := cmd.Flags().GetBool("is-reconciled")
			payment["IsReconciled"] = isReconciled
		}

		if !hasPaymentTarget(payment) {
			return errors.New("payment target is required (invoice, credit note, prepayment, or overpayment)")
		}
		if !hasKey(payment, "Account") {
			return errors.New("--account-id is required (or provide Account in --body)")
		}
		if !hasKey(payment, "Amount") {
			return errors.New("--amount is required (or provide Amount in --body)")
		}

		payload, err := json.Marshal(payment)
		if err != nil {
			return fmt.Errorf("failed to build payment payload: %w", err)
		}

		params := url.Values{}
		if cmd.Flags().Changed("summarize-errors") {
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
	paymentsCreateCmd.Flags().String("body", "", "Raw JSON object of payment attributes")
	paymentsCreateCmd.Flags().Bool("summarize-errors", false, "Summarize validation errors in the response")
	paymentsCreateCmd.Flags().String("idempotency-key", "", "Idempotency key for safely retrying requests")
}

func parsePaymentBody(cmd *cobra.Command) (map[string]any, error) {
	body, _ := cmd.Flags().GetString("body")
	if strings.TrimSpace(body) == "" {
		return nil, nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		return nil, fmt.Errorf("invalid --body JSON: %w", err)
	}

	obj, ok := decoded.(map[string]any)
	if !ok {
		return nil, errors.New("--body must be a JSON object")
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
