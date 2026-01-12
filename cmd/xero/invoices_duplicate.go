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

var invoicesDuplicateLastCmd = &cobra.Command{
	Use:   "duplicate-last",
	Short: "Duplicate the most recent invoice for a contact as a draft",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		contactName, _ := cmd.Flags().GetString("contact")
		contactName = strings.TrimSpace(contactName)
		if contactName == "" {
			return errors.New("--contact is required")
		}

		dateFlag, _ := cmd.Flags().GetString("date")
		dueDateFlag, _ := cmd.Flags().GetString("due-date")
		dueIn, _ := cmd.Flags().GetInt("due-in")
		dateStr, dueDateStr, err := resolveInvoiceDates(dateFlag, dueDateFlag, dueIn)
		if err != nil {
			return err
		}

		addDesc, _ := cmd.Flags().GetString("add-line-description")
		addQty, _ := cmd.Flags().GetFloat64("add-line-quantity")
		addUnit, _ := cmd.Flags().GetFloat64("add-line-unit-amount")
		if addDesc == "" && (addQty != 0 || addUnit != 0) {
			return errors.New("--add-line-description is required when quantity or unit amount is provided")
		}
		if addDesc != "" && (addQty <= 0 || addUnit <= 0) {
			return errors.New("--add-line-quantity and --add-line-unit-amount must be greater than 0")
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}

		client := xero.NewClient(xeroAPIBase)
		sourceInvoice, err := fetchLatestInvoiceForContact(client, headers, contactName)
		if err != nil {
			return err
		}

		lineItems := extractInvoiceLineItems(sourceInvoice)
		if len(lineItems) == 0 {
			return errors.New("source invoice has no line items to duplicate")
		}

		if addDesc != "" {
			lineItems = append(lineItems, buildAddedLineItem(lineItems, addDesc, addQty, addUnit))
		}

		newInvoice, err := buildDuplicateInvoicePayload(sourceInvoice, dateStr, dueDateStr, lineItems)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{"Invoices": []any{newInvoice}})
		if err != nil {
			return fmt.Errorf("failed to build invoice payload: %w", err)
		}

		headers["content-type"] = "application/json"
		endpoint := fmt.Sprintf("%s/Invoices", xeroAPIBase)
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
	invoicesCmd.AddCommand(invoicesDuplicateLastCmd)
	invoicesDuplicateLastCmd.Flags().String("contact", "", "Contact name to duplicate the latest invoice for")
	invoicesDuplicateLastCmd.Flags().String("date", "", "Invoice date in YYYY-MM-DD (defaults to today)")
	invoicesDuplicateLastCmd.Flags().String("due-date", "", "Due date in YYYY-MM-DD (overrides --due-in)")
	invoicesDuplicateLastCmd.Flags().Int("due-in", 7, "Number of days after the invoice date for the due date")
	invoicesDuplicateLastCmd.Flags().String("add-line-description", "", "Optional line item description to append")
	invoicesDuplicateLastCmd.Flags().Float64("add-line-quantity", 0, "Quantity for the appended line item")
	invoicesDuplicateLastCmd.Flags().Float64("add-line-unit-amount", 0, "Unit amount for the appended line item")
}

func fetchLatestInvoiceForContact(client *xero.Client, headers map[string]string, contactName string) (map[string]any, error) {
	params := url.Values{}
	params.Set("where", fmt.Sprintf("Contact.Name==\"%s\"", escapeWhereValue(contactName)))
	params.Set("order", "Date DESC")
	params.Set("page", "1")
	params.Set("pageSize", "1")

	endpoint := fmt.Sprintf("%s/Invoices?%s", xeroAPIBase, params.Encode())
	status, body, err := client.Do("GET", endpoint, headers, nil)
	if err != nil {
		return nil, err
	}
	if status == 401 {
		return nil, errors.New("authentication failed. Please run 'xero auth login' again")
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", status, string(body))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	invoices := getArray(payload, "Invoices")
	if len(invoices) == 0 {
		return nil, fmt.Errorf("no invoices found for contact %q", contactName)
	}

	invoice, ok := invoices[0].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected invoice response format")
	}

	if items := extractInvoiceLineItems(invoice); len(items) > 0 {
		return invoice, nil
	}

	invoiceID := stringValue(invoice, "InvoiceID", "")
	if invoiceID == "" {
		return nil, errors.New("invoice ID missing; cannot fetch full invoice details")
	}
	return fetchInvoiceByID(client, headers, invoiceID)
}

func fetchInvoiceByID(client *xero.Client, headers map[string]string, invoiceID string) (map[string]any, error) {
	endpoint := fmt.Sprintf("%s/Invoices/%s", xeroAPIBase, invoiceID)
	status, body, err := client.Do("GET", endpoint, headers, nil)
	if err != nil {
		return nil, err
	}
	if status == 401 {
		return nil, errors.New("authentication failed. Please run 'xero auth login' again")
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", status, string(body))
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	invoices := getArray(payload, "Invoices")
	if len(invoices) == 0 {
		return nil, errors.New("invoice not found")
	}
	invoice, ok := invoices[0].(map[string]any)
	if !ok {
		return nil, errors.New("unexpected invoice response format")
	}
	return invoice, nil
}

func resolveInvoiceDates(dateFlag, dueDateFlag string, dueIn int) (string, string, error) {
	baseDate := time.Now()
	if strings.TrimSpace(dateFlag) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(dateFlag))
		if err != nil {
			return "", "", fmt.Errorf("invalid --date; expected YYYY-MM-DD")
		}
		baseDate = parsed
	}

	var dueDate time.Time
	if strings.TrimSpace(dueDateFlag) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(dueDateFlag))
		if err != nil {
			return "", "", fmt.Errorf("invalid --due-date; expected YYYY-MM-DD")
		}
		dueDate = parsed
	} else {
		if dueIn <= 0 {
			dueIn = 7
		}
		dueDate = baseDate.AddDate(0, 0, dueIn)
	}

	return baseDate.Format("2006-01-02"), dueDate.Format("2006-01-02"), nil
}

func escapeWhereValue(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(escaped, "\"", "\\\"")
}

func extractInvoiceLineItems(invoice map[string]any) []any {
	raw, ok := invoice["LineItems"].([]any)
	if !ok {
		return nil
	}

	lineItems := make([]any, 0, len(raw))
	for _, item := range raw {
		lineItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		filtered := map[string]any{}
		for _, key := range []string{
			"Description",
			"Quantity",
			"UnitAmount",
			"AccountCode",
			"TaxType",
			"ItemCode",
			"Tracking",
			"DiscountRate",
		} {
			if value, ok := lineItem[key]; ok {
				filtered[key] = value
			}
		}
		if len(filtered) > 0 {
			lineItems = append(lineItems, filtered)
		}
	}
	return lineItems
}

func buildAddedLineItem(existing []any, description string, quantity, unitAmount float64) map[string]any {
	line := map[string]any{
		"Description": description,
		"Quantity":    quantity,
		"UnitAmount":  unitAmount,
	}
	if len(existing) > 0 {
		if first, ok := existing[0].(map[string]any); ok {
			if value, ok := first["AccountCode"]; ok {
				line["AccountCode"] = value
			}
			if value, ok := first["TaxType"]; ok {
				line["TaxType"] = value
			}
		}
	}
	return line
}

func buildDuplicateInvoicePayload(source map[string]any, dateStr, dueDateStr string, lineItems []any) (map[string]any, error) {
	contact, ok := source["Contact"].(map[string]any)
	if !ok {
		return nil, errors.New("source invoice missing contact information")
	}
	contactPayload := map[string]any{}
	if contactID := stringValue(contact, "ContactID", ""); contactID != "" {
		contactPayload["ContactID"] = contactID
	} else if contactName := stringValue(contact, "Name", ""); contactName != "" {
		contactPayload["Name"] = contactName
	}
	if len(contactPayload) == 0 {
		return nil, errors.New("source invoice contact missing ID or name")
	}

	newInvoice := map[string]any{
		"Type":      stringValue(source, "Type", "ACCREC"),
		"Contact":   contactPayload,
		"Date":      dateStr,
		"DueDate":   dueDateStr,
		"Status":    "DRAFT",
		"LineItems": lineItems,
	}

	if currency := stringValue(source, "CurrencyCode", ""); currency != "" {
		newInvoice["CurrencyCode"] = currency
	}
	if amountType := stringValue(source, "LineAmountTypes", ""); amountType != "" {
		newInvoice["LineAmountTypes"] = amountType
	}
	if branding := stringValue(source, "BrandingThemeID", ""); branding != "" {
		newInvoice["BrandingThemeID"] = branding
	}
	if reference := stringValue(source, "Reference", ""); reference != "" {
		newInvoice["Reference"] = reference
	}

	return newInvoice, nil
}
