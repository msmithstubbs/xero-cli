package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var invoicesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an invoice",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		bodyAttrs, err := parseInvoiceBody(cmd)
		if err != nil {
			return err
		}

		contactName, _ := cmd.Flags().GetString("contact")
		contactID, _ := cmd.Flags().GetString("contact-id")
		contactName = strings.TrimSpace(contactName)
		contactID = strings.TrimSpace(contactID)
		contactFromBody, contactFromBodyOk := extractContact(bodyAttrs)
		if contactName != "" && contactID != "" {
			return errors.New("use either --contact or --contact-id, not both")
		}
		if contactName == "" && contactID == "" && !contactFromBodyOk {
			return errors.New("--contact or --contact-id is required (or provide Contact in --body)")
		}

		invoiceType, _ := cmd.Flags().GetString("type")
		invoiceType = strings.TrimSpace(invoiceType)
		if invoiceType == "" {
			invoiceType = "ACCREC"
		}

		status, _ := cmd.Flags().GetString("status")
		status = strings.TrimSpace(status)
		if status == "" {
			status = "DRAFT"
		}
		if cmd.Flags().Changed("status") {
			normalized, err := validateInvoiceStatus(status)
			if err != nil {
				return err
			}
			status = normalized
		}

		lineDesc, _ := cmd.Flags().GetString("line-description")
		lineQty, _ := cmd.Flags().GetFloat64("line-quantity")
		lineUnit, _ := cmd.Flags().GetFloat64("line-unit-amount")
		lineDesc = strings.TrimSpace(lineDesc)
		lineFlagsSet := cmd.Flags().Changed("line-description") ||
			cmd.Flags().Changed("line-quantity") ||
			cmd.Flags().Changed("line-unit-amount")
		if lineFlagsSet {
			if lineDesc == "" {
				return errors.New("--line-description is required when line item fields are set")
			}
			if lineQty <= 0 || lineUnit <= 0 {
				return errors.New("--line-quantity and --line-unit-amount must be greater than 0")
			}
		}

		invoice := cloneMap(bodyAttrs)

		if contactID != "" {
			invoice["Contact"] = map[string]any{"ContactID": contactID}
		} else if contactName != "" {
			invoice["Contact"] = map[string]any{"Name": contactName}
		} else if contactFromBodyOk {
			invoice["Contact"] = contactFromBody
		}

		if cmd.Flags().Changed("type") || !hasKey(invoice, "Type") {
			invoice["Type"] = strings.ToUpper(invoiceType)
		}
		if cmd.Flags().Changed("status") || !hasKey(invoice, "Status") {
			invoice["Status"] = strings.ToUpper(status)
		}

		if shouldSetDefaultDates(cmd, invoice) {
			dateFlag, _ := cmd.Flags().GetString("date")
			dueDateFlag, _ := cmd.Flags().GetString("due-date")
			dueIn, _ := cmd.Flags().GetInt("due-in")
			dateStr, dueDateStr, err := resolveInvoiceDates(dateFlag, dueDateFlag, dueIn)
			if err != nil {
				return err
			}
			invoice["Date"] = dateStr
			invoice["DueDate"] = dueDateStr
		}

		if lineFlagsSet {
			accountCode, _ := cmd.Flags().GetString("account-code")
			taxType, _ := cmd.Flags().GetString("tax-type")
			itemCode, _ := cmd.Flags().GetString("item-code")

			lineItem := map[string]any{
				"Description": lineDesc,
				"Quantity":    lineQty,
				"UnitAmount":  lineUnit,
			}
			if trimmed := strings.TrimSpace(accountCode); trimmed != "" {
				lineItem["AccountCode"] = trimmed
			}
			if trimmed := strings.TrimSpace(taxType); trimmed != "" {
				lineItem["TaxType"] = trimmed
			}
			if trimmed := strings.TrimSpace(itemCode); trimmed != "" {
				lineItem["ItemCode"] = trimmed
			}

			existing := extractLineItems(invoice)
			invoice["LineItems"] = append(existing, lineItem)
		}

		if len(extractLineItems(invoice)) == 0 {
			return errors.New("at least one line item is required; use --line-* flags or provide LineItems in --body")
		}

		if currency, _ := cmd.Flags().GetString("currency"); strings.TrimSpace(currency) != "" {
			invoice["CurrencyCode"] = strings.TrimSpace(currency)
		}
		if amountTypes, _ := cmd.Flags().GetString("line-amount-types"); strings.TrimSpace(amountTypes) != "" {
			invoice["LineAmountTypes"] = strings.TrimSpace(amountTypes)
		}
		if branding, _ := cmd.Flags().GetString("branding-theme-id"); strings.TrimSpace(branding) != "" {
			invoice["BrandingThemeID"] = strings.TrimSpace(branding)
		}
		if reference, _ := cmd.Flags().GetString("reference"); strings.TrimSpace(reference) != "" {
			ref := strings.TrimSpace(reference)
			invType := ""
			if raw, ok := invoice["Type"].(string); ok {
				invType = strings.ToUpper(strings.TrimSpace(raw))
			}
			if invType == "" {
				invType = strings.ToUpper(invoiceType)
			}
			if invType == "ACCPAY" {
				invoice["InvoiceNumber"] = ref
			} else {
				invoice["Reference"] = ref
			}
		}

		payload, err := json.Marshal(map[string]any{"Invoices": []any{invoice}})
		if err != nil {
			return fmt.Errorf("failed to build invoice payload: %w", err)
		}

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}
		headers["content-type"] = "application/json"

		client := xero.NewClient(xeroAPIBase)
		endpoint := fmt.Sprintf("%s/Invoices", xeroAPIBase)
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
	invoicesCmd.AddCommand(invoicesCreateCmd)
	invoicesCreateCmd.Flags().String("contact", "", "Contact name for the invoice")
	invoicesCreateCmd.Flags().String("contact-id", "", "Contact ID for the invoice")
	invoicesCreateCmd.Flags().String("type", "ACCREC", "Invoice type (ACCREC or ACCPAY)")
	invoicesCreateCmd.Flags().String("status", "DRAFT", "Invoice status (DRAFT, SUBMITTED, AUTHORISED, PAID, VOIDED, DELETED)")
	invoicesCreateCmd.Flags().String("date", "", "Invoice date in YYYY-MM-DD (defaults to today)")
	invoicesCreateCmd.Flags().String("due-date", "", "Due date in YYYY-MM-DD (overrides --due-in)")
	invoicesCreateCmd.Flags().Int("due-in", 7, "Number of days after the invoice date for the due date")
	invoicesCreateCmd.Flags().String("body", "", "Raw JSON object of invoice attributes")
	invoicesCreateCmd.Flags().String("line-description", "", "Line item description")
	invoicesCreateCmd.Flags().Float64("line-quantity", 0, "Line item quantity")
	invoicesCreateCmd.Flags().Float64("line-unit-amount", 0, "Line item unit amount")
	invoicesCreateCmd.Flags().String("account-code", "", "Line item account code")
	invoicesCreateCmd.Flags().String("tax-type", "", "Line item tax type")
	invoicesCreateCmd.Flags().String("item-code", "", "Line item item code")
	invoicesCreateCmd.Flags().String("currency", "", "Invoice currency code")
	invoicesCreateCmd.Flags().String("line-amount-types", "", "Line amount types (e.g. Exclusive, Inclusive, NoTax)")
	invoicesCreateCmd.Flags().String("branding-theme-id", "", "Branding theme ID")
	invoicesCreateCmd.Flags().String("reference", "", "Reference text")
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

func parseInvoiceBody(cmd *cobra.Command) (map[string]any, error) {
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

func extractContact(body map[string]any) (map[string]any, bool) {
	if body == nil {
		return nil, false
	}
	raw, ok := body["Contact"].(map[string]any)
	if !ok {
		return nil, false
	}
	if id, ok := raw["ContactID"].(string); ok && strings.TrimSpace(id) != "" {
		return raw, true
	}
	if name, ok := raw["Name"].(string); ok && strings.TrimSpace(name) != "" {
		return raw, true
	}
	return nil, false
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func shouldSetDefaultDates(cmd *cobra.Command, invoice map[string]any) bool {
	if cmd.Flags().Changed("date") || cmd.Flags().Changed("due-date") || cmd.Flags().Changed("due-in") {
		return true
	}
	return !(hasKey(invoice, "Date") || hasKey(invoice, "DueDate"))
}

func extractLineItems(invoice map[string]any) []any {
	if invoice == nil {
		return nil
	}
	if items, ok := invoice["LineItems"].([]any); ok {
		return items
	}
	return nil
}
