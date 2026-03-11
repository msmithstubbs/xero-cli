package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var invoicesCmd = &cobra.Command{
	Use:   "invoices",
	Short: "Manage invoices",
	Long:  "Manage invoices in Xero.",
}

var invoicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all invoices",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")
		status, _ := cmd.Flags().GetString("status")

		params := url.Values{}
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		if pageSize > 0 {
			params.Set("pageSize", fmt.Sprintf("%d", pageSize))
		}
		if status != "" {
			params.Set("where", buildWhereClause(status))
		}

		endpoint := fmt.Sprintf("%s/Invoices?%s", xeroAPIBase, params.Encode())

		if resolvedOutputFormat() == outputTable {
			fmt.Println("Fetching invoices...")
			fmt.Println()
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}

		client := xero.NewClient(xeroAPIBase)
		statusCode, body, err := client.Do("GET", endpoint, headers, nil)
		if err != nil {
			return internalError("request failed", err)
		}

		if statusCode == 401 {
			return authenticationExpiredError()
		}
		if statusCode < 200 || statusCode >= 300 {
			return apiError(statusCode, body)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return parseResponseError(err)
		}

		invoices := getArray(payload, "Invoices")
		return emitData(payload, func() {
			displayInvoices(invoices)
		})
	},
}

func init() {
	invoicesCmd.AddCommand(invoicesListCmd)
	invoicesListCmd.Flags().Int("page", 1, "Page number for pagination")
	invoicesListCmd.Flags().Int("page-size", 100, "Number of items per page")
	invoicesListCmd.Flags().String("status", "", "Filter invoices by status")
}

func buildWhereClause(status string) string {
	return fmt.Sprintf("Status==\"%s\"", strings.ToUpper(status))
}

func displayInvoices(items []any) {
	if len(items) == 0 {
		fmt.Println("No invoices found.")
		return
	}

	fmt.Printf("Found %d invoice(s):\n", len(items))
	fmt.Println()
	ui.PrintHeaderLine(120)
	header := ui.FormatRow(
		ui.Pad("Invoice Number", 20),
		ui.Pad("Type", 10),
		ui.Pad("Contact", 25),
		ui.Pad("Date", 12),
		ui.Pad("Due Date", 12),
		ui.Pad("Status", 12),
		ui.Pad("Total", 15),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(120)

	for _, item := range items {
		invoice, ok := item.(map[string]any)
		if !ok {
			continue
		}
		number := stringValue(invoice, "InvoiceNumber", "N/A")
		typeValue := stringValue(invoice, "Type", "N/A")
		contactName := "N/A"
		if contact, ok := invoice["Contact"].(map[string]any); ok {
			contactName = stringValue(contact, "Name", "N/A")
		}
		date := formatDate(invoice["Date"])
		dueDate := formatDate(invoice["DueDate"])
		status := stringValue(invoice, "Status", "N/A")
		total := formatCurrency(invoice["Total"])

		row := ui.FormatRow(
			ui.Pad(number, 20),
			ui.Pad(typeValue, 10),
			ui.Pad(contactName, 25),
			ui.Pad(date, 12),
			ui.Pad(dueDate, 12),
			ui.Pad(statusWithEmoji(status), 12),
			ui.Pad(total, 15),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(120)
}

func formatDate(value any) string {
	if value == nil {
		return "N/A"
	}
	if s, ok := value.(string); ok {
		if strings.HasPrefix(s, "/Date(") {
			trimmed := strings.TrimPrefix(s, "/Date(")
			trimmed = strings.TrimSuffix(trimmed, ")/")
			if millis, err := parseInt64(trimmed); err == nil {
				return time.Unix(millis/1000, 0).UTC().Format("2006-01-02")
			}
		}
		if s != "" {
			return s
		}
	}
	return "N/A"
}

func formatCurrency(value any) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("$%.2f", v)
	case float32:
		return fmt.Sprintf("$%.2f", v)
	case int:
		return fmt.Sprintf("$%.2f", float64(v))
	case int64:
		return fmt.Sprintf("$%.2f", float64(v))
	}
	return "$0.00"
}

func statusWithEmoji(status string) string {
	switch strings.ToUpper(status) {
	case "PAID":
		return "PAID"
	case "AUTHORISED":
		return "AUTH"
	case "DRAFT":
		return "DRAFT"
	case "VOIDED":
		return "VOID"
	case "DELETED":
		return "DEL"
	default:
		return status
	}
}

func parseInt64(value string) (int64, error) {
	var parsed int64
	_, err := fmt.Sscanf(value, "%d", &parsed)
	return parsed, err
}
