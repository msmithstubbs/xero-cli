package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var paymentsCmd = &cobra.Command{
	Use:   "payments",
	Short: "Manage payments",
	Long:  "Manage payments for invoices, credit notes, prepayments, and overpayments.",
}

var paymentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List payments",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		where, _ := cmd.Flags().GetString("where")
		order, _ := cmd.Flags().GetString("order")
		ifModifiedSince, _ := cmd.Flags().GetString("if-modified-since")

		params := url.Values{}
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		if pageSize > 0 {
			params.Set("pageSize", fmt.Sprintf("%d", pageSize))
		}
		if strings.TrimSpace(where) != "" {
			params.Set("where", strings.TrimSpace(where))
		}
		if strings.TrimSpace(order) != "" {
			params.Set("order", strings.TrimSpace(order))
		}

		endpoint := fmt.Sprintf("%s/Payments", xeroAPIBase)
		if encoded := params.Encode(); encoded != "" {
			endpoint = fmt.Sprintf("%s?%s", endpoint, encoded)
		}

		fmt.Println("Fetching payments...")
		fmt.Println()

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}
		if strings.TrimSpace(ifModifiedSince) != "" {
			headers["If-Modified-Since"] = strings.TrimSpace(ifModifiedSince)
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

		payments := getArray(payload, "Payments")
		displayPayments(payments)
		return nil
	},
}

func init() {
	paymentsCmd.AddCommand(paymentsListCmd)
	paymentsCmd.AddCommand(paymentsGetCmd)
	paymentsCmd.AddCommand(paymentsCreateCmd)
	paymentsCmd.AddCommand(paymentsUpdateCmd)
	paymentsCmd.AddCommand(paymentsDeleteCmd)

	paymentsListCmd.Flags().Int("page", 1, "Page number for pagination")
	paymentsListCmd.Flags().Int("page-size", 100, "Number of items per page")
	paymentsListCmd.Flags().String("where", "", "Filter payments with a where clause (e.g. Status==\"AUTHORISED\")")
	paymentsListCmd.Flags().String("order", "", "Order payments by field, e.g. \"Amount ASC\"")
	paymentsListCmd.Flags().String("if-modified-since", "", "Filter payments modified since ISO 8601 time")
}

func displayPayments(items []any) {
	if len(items) == 0 {
		fmt.Println("No payments found.")
		return
	}

	fmt.Printf("Found %d payment(s):\n", len(items))
	fmt.Println()
	ui.PrintHeaderLine(140)
	header := ui.FormatRow(
		ui.Pad("Payment ID", 38),
		ui.Pad("Date", 12),
		ui.Pad("Amount", 12),
		ui.Pad("Reference", 22),
		ui.Pad("Status", 12),
		ui.Pad("Applied To", 40),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(140)

	for _, item := range items {
		payment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		paymentID := stringValue(payment, "PaymentID", "N/A")
		date := formatDate(payment["Date"])
		amount := formatCurrency(payment["Amount"])
		reference := stringValue(payment, "Reference", "N/A")
		status := stringValue(payment, "Status", "N/A")
		target := paymentTarget(payment)

		row := ui.FormatRow(
			ui.Pad(paymentID, 38),
			ui.Pad(date, 12),
			ui.Pad(amount, 12),
			ui.Pad(reference, 22),
			ui.Pad(status, 12),
			ui.Pad(target, 40),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(140)
}

func paymentTarget(payment map[string]any) string {
	if invoice, ok := payment["Invoice"].(map[string]any); ok {
		if number := strings.TrimSpace(stringValue(invoice, "InvoiceNumber", "")); number != "" {
			return "Invoice " + number
		}
		if id := strings.TrimSpace(stringValue(invoice, "InvoiceID", "")); id != "" {
			return "Invoice " + id
		}
	}
	if creditNote, ok := payment["CreditNote"].(map[string]any); ok {
		if number := strings.TrimSpace(stringValue(creditNote, "CreditNoteNumber", "")); number != "" {
			return "CreditNote " + number
		}
		if id := strings.TrimSpace(stringValue(creditNote, "CreditNoteID", "")); id != "" {
			return "CreditNote " + id
		}
	}
	if prepayment, ok := payment["Prepayment"].(map[string]any); ok {
		if id := strings.TrimSpace(stringValue(prepayment, "PrepaymentID", "")); id != "" {
			return "Prepayment " + id
		}
	}
	if overpayment, ok := payment["Overpayment"].(map[string]any); ok {
		if id := strings.TrimSpace(stringValue(overpayment, "OverpaymentID", "")); id != "" {
			return "Overpayment " + id
		}
	}
	return "N/A"
}

func displayPaymentDetail(payment map[string]any) {
	fmt.Println("Payment Details:")
	fmt.Println()
	ui.PrintHeaderLine(80)

	fmt.Printf("Payment ID:       %s\n", stringValue(payment, "PaymentID", "N/A"))
	fmt.Printf("Status:           %s\n", stringValue(payment, "Status", "N/A"))
	fmt.Printf("Date:             %s\n", formatDate(payment["Date"]))
	fmt.Printf("Amount:           %s\n", formatCurrency(payment["Amount"]))
	fmt.Printf("Reference:        %s\n", stringValue(payment, "Reference", "N/A"))
	fmt.Printf("Currency Rate:    %s\n", formatRate(payment["CurrencyRate"]))
	fmt.Printf("Payment Type:     %s\n", stringValue(payment, "PaymentType", "N/A"))
	fmt.Printf("Is Reconciled:    %s\n", formatBool(payment["IsReconciled"]))

	if account, ok := payment["Account"].(map[string]any); ok {
		fmt.Println()
		fmt.Printf("Account ID:       %s\n", stringValue(account, "AccountID", "N/A"))
		fmt.Printf("Account Code:     %s\n", stringValue(account, "Code", "N/A"))
		fmt.Printf("Account Name:     %s\n", stringValue(account, "Name", "N/A"))
	}

	if target := paymentTarget(payment); target != "N/A" {
		fmt.Println()
		fmt.Printf("Applied To:       %s\n", target)
	}
}

func formatBool(value any) string {
	if v, ok := value.(bool); ok {
		if v {
			return "true"
		}
		return "false"
	}
	return "N/A"
}

func formatRate(value any) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.6f", v)
	case float32:
		return fmt.Sprintf("%.6f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	}
	return "N/A"
}
