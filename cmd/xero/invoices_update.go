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

var invoicesUpdateCmd = &cobra.Command{
	Use:   "update <invoice_id>",
	Short: "Update an invoice",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		invoiceID := strings.TrimSpace(args[0])
		if invoiceID == "" {
			return errors.New("invoice_id is required")
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

		invoice := cloneMap(bodyAttrs)
		invoice["InvoiceID"] = invoiceID

		if contactID != "" {
			invoice["Contact"] = map[string]any{"ContactID": contactID}
		} else if contactName != "" {
			invoice["Contact"] = map[string]any{"Name": contactName}
		} else if contactFromBodyOk {
			invoice["Contact"] = contactFromBody
		}

		invoiceType, _ := cmd.Flags().GetString("type")
		if cmd.Flags().Changed("type") {
			invoiceType = strings.TrimSpace(invoiceType)
			if invoiceType == "" {
				return errors.New("--type cannot be empty")
			}
			invoice["Type"] = strings.ToUpper(invoiceType)
		}

		status, _ := cmd.Flags().GetString("status")
		if cmd.Flags().Changed("status") {
			status = strings.TrimSpace(status)
			if status == "" {
				return errors.New("--status cannot be empty")
			}
			invoice["Status"] = strings.ToUpper(status)
		}

		if err := applyInvoiceUpdateDates(cmd, invoice); err != nil {
			return err
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

		if cmd.Flags().Changed("currency") {
			if currency, _ := cmd.Flags().GetString("currency"); strings.TrimSpace(currency) == "" {
				return errors.New("--currency cannot be empty")
			} else {
				invoice["CurrencyCode"] = strings.TrimSpace(currency)
			}
		}
		if cmd.Flags().Changed("line-amount-types") {
			if amountTypes, _ := cmd.Flags().GetString("line-amount-types"); strings.TrimSpace(amountTypes) == "" {
				return errors.New("--line-amount-types cannot be empty")
			} else {
				invoice["LineAmountTypes"] = strings.TrimSpace(amountTypes)
			}
		}
		if cmd.Flags().Changed("branding-theme-id") {
			if branding, _ := cmd.Flags().GetString("branding-theme-id"); strings.TrimSpace(branding) == "" {
				return errors.New("--branding-theme-id cannot be empty")
			} else {
				invoice["BrandingThemeID"] = strings.TrimSpace(branding)
			}
		}
		if cmd.Flags().Changed("reference") {
			if reference, _ := cmd.Flags().GetString("reference"); strings.TrimSpace(reference) == "" {
				return errors.New("--reference cannot be empty")
			} else {
				invoice["Reference"] = strings.TrimSpace(reference)
			}
		}

		if len(invoice) == 1 {
			return errors.New("no invoice fields provided; use flags or --body")
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
	invoicesCmd.AddCommand(invoicesUpdateCmd)
	invoicesUpdateCmd.Flags().String("contact", "", "Contact name for the invoice")
	invoicesUpdateCmd.Flags().String("contact-id", "", "Contact ID for the invoice")
	invoicesUpdateCmd.Flags().String("type", "", "Invoice type (ACCREC or ACCPAY)")
	invoicesUpdateCmd.Flags().String("status", "", "Invoice status")
	invoicesUpdateCmd.Flags().String("date", "", "Invoice date in YYYY-MM-DD")
	invoicesUpdateCmd.Flags().String("due-date", "", "Due date in YYYY-MM-DD (overrides --due-in)")
	invoicesUpdateCmd.Flags().Int("due-in", 0, "Number of days after today (or --date) for the due date")
	invoicesUpdateCmd.Flags().String("body", "", "Raw JSON object of invoice attributes")
	invoicesUpdateCmd.Flags().String("line-description", "", "Line item description")
	invoicesUpdateCmd.Flags().Float64("line-quantity", 0, "Line item quantity")
	invoicesUpdateCmd.Flags().Float64("line-unit-amount", 0, "Line item unit amount")
	invoicesUpdateCmd.Flags().String("account-code", "", "Line item account code")
	invoicesUpdateCmd.Flags().String("tax-type", "", "Line item tax type")
	invoicesUpdateCmd.Flags().String("item-code", "", "Line item item code")
	invoicesUpdateCmd.Flags().String("currency", "", "Invoice currency code")
	invoicesUpdateCmd.Flags().String("line-amount-types", "", "Line amount types (e.g. Exclusive, Inclusive, NoTax)")
	invoicesUpdateCmd.Flags().String("branding-theme-id", "", "Branding theme ID")
	invoicesUpdateCmd.Flags().String("reference", "", "Reference text")
}

func applyInvoiceUpdateDates(cmd *cobra.Command, invoice map[string]any) error {
	dateChanged := cmd.Flags().Changed("date")
	dueDateChanged := cmd.Flags().Changed("due-date")
	dueInChanged := cmd.Flags().Changed("due-in")

	if !dateChanged && !dueDateChanged && !dueInChanged {
		return nil
	}

	var baseDate time.Time
	if dateChanged {
		dateFlag, _ := cmd.Flags().GetString("date")
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(dateFlag))
		if err != nil {
			return fmt.Errorf("invalid --date; expected YYYY-MM-DD")
		}
		invoice["Date"] = parsed.Format("2006-01-02")
		baseDate = parsed
	} else {
		baseDate = time.Now()
	}

	if dueDateChanged {
		dueDateFlag, _ := cmd.Flags().GetString("due-date")
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(dueDateFlag))
		if err != nil {
			return fmt.Errorf("invalid --due-date; expected YYYY-MM-DD")
		}
		invoice["DueDate"] = parsed.Format("2006-01-02")
		return nil
	}

	if dueInChanged {
		dueIn, _ := cmd.Flags().GetInt("due-in")
		if dueIn <= 0 {
			return errors.New("--due-in must be greater than 0")
		}
		dueDate := baseDate.AddDate(0, 0, dueIn)
		invoice["DueDate"] = dueDate.Format("2006-01-02")
	}

	return nil
}
