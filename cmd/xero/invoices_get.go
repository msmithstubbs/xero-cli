package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var invoicesGetCmd = &cobra.Command{
	Use:   "get <invoice_id>",
	Short: "Get a single invoice by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		invoiceID := strings.TrimSpace(args[0])
		if invoiceID == "" {
			return errors.New("invoice_id is required")
		}

		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/Invoices/%s", xeroAPIBase, invoiceID)
		fmt.Printf("Fetching invoice %s...\n\n", invoiceID)

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
			invoices := getArray(payload, "Invoices")
			if len(invoices) == 0 {
				fmt.Println("Invoice not found.")
				return nil
			}
			if invoice, ok := invoices[0].(map[string]any); ok {
				displayInvoiceDetail(invoice)
				return nil
			}
			return errors.New("unexpected response format")
		case 401:
			return errors.New("authentication failed. Please run 'xero auth login' again")
		case 404:
			return errors.New("invoice not found")
		default:
			return fmt.Errorf("API request failed with status %d: %s", status, string(body))
		}
	},
}

func init() {
	invoicesCmd.AddCommand(invoicesGetCmd)
}

func displayInvoiceDetail(invoice map[string]any) {
	fmt.Println("Invoice Details:")
	fmt.Println()
	ui.PrintHeaderLine(80)

	fmt.Printf("Invoice ID:       %s\n", stringValue(invoice, "InvoiceID", "N/A"))
	fmt.Printf("Invoice Number:   %s\n", stringValue(invoice, "InvoiceNumber", "N/A"))
	fmt.Printf("Type:             %s\n", stringValue(invoice, "Type", "N/A"))
	fmt.Printf("Status:           %s\n", stringValue(invoice, "Status", "N/A"))
	fmt.Printf("Reference:        %s\n", stringValue(invoice, "Reference", "N/A"))

	if contact, ok := invoice["Contact"].(map[string]any); ok {
		fmt.Printf("Contact:          %s\n", stringValue(contact, "Name", "N/A"))
		fmt.Printf("Contact ID:       %s\n", stringValue(contact, "ContactID", "N/A"))
	}

	fmt.Printf("Date:             %s\n", formatDate(invoice["Date"]))
	fmt.Printf("Due Date:         %s\n", formatDate(invoice["DueDate"]))
	fmt.Printf("Currency:         %s\n", stringValue(invoice, "CurrencyCode", "N/A"))
	fmt.Printf("Line Amount Type: %s\n", stringValue(invoice, "LineAmountTypes", "N/A"))

	lineItems := getArray(invoice, "LineItems")
	if len(lineItems) > 0 {
		fmt.Println("\nLine Items:")
		ui.PrintHeaderLine(80)
		header := ui.FormatRow(
			ui.Pad("Description", 35),
			ui.Pad("Qty", 8),
			ui.Pad("Unit Price", 12),
			ui.Pad("Account", 10),
			ui.Pad("Amount", 12),
		)
		fmt.Println(header)
		ui.PrintHeaderLine(80)

		for _, item := range lineItems {
			lineItem, ok := item.(map[string]any)
			if !ok {
				continue
			}
			desc := stringValue(lineItem, "Description", "N/A")
			if len(desc) > 35 {
				desc = desc[:32] + "..."
			}
			qty := formatQuantity(lineItem["Quantity"])
			unitAmount := formatCurrency(lineItem["UnitAmount"])
			accountCode := stringValue(lineItem, "AccountCode", "N/A")
			lineAmount := formatCurrency(lineItem["LineAmount"])

			row := ui.FormatRow(
				ui.Pad(desc, 35),
				ui.Pad(qty, 8),
				ui.Pad(unitAmount, 12),
				ui.Pad(accountCode, 10),
				ui.Pad(lineAmount, 12),
			)
			fmt.Println(row)
		}
		ui.PrintHeaderLine(80)
	}

	fmt.Println("\nTotals:")
	fmt.Printf("  Sub Total:      %s\n", formatCurrency(invoice["SubTotal"]))
	fmt.Printf("  Total Tax:      %s\n", formatCurrency(invoice["TotalTax"]))
	fmt.Printf("  Total:          %s\n", formatCurrency(invoice["Total"]))
	fmt.Printf("  Amount Due:     %s\n", formatCurrency(invoice["AmountDue"]))
	fmt.Printf("  Amount Paid:    %s\n", formatCurrency(invoice["AmountPaid"]))

	ui.PrintHeaderLine(80)
}

func formatQuantity(value any) string {
	switch v := value.(type) {
	case float64:
		if v == float64(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%.2f", v)
	case float32:
		if v == float32(int(v)) {
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%.2f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	}
	return "0"
}
