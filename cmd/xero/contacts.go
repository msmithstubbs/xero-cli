package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/ui"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

const xeroAPIBase = "https://api.xero.com/api.xro/2.0"

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts",
}

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		page, _ := cmd.Flags().GetInt("page")
		pageSize, _ := cmd.Flags().GetInt("page-size")

		params := url.Values{}
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		if pageSize > 0 {
			params.Set("pageSize", fmt.Sprintf("%d", pageSize))
		}
		endpoint := fmt.Sprintf("%s/Contacts?%s", xeroAPIBase, params.Encode())

		if resolvedOutputFormat() == outputTable {
			fmt.Println("Fetching contacts...")
			fmt.Println()
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

		if status == 401 {
			return authenticationExpiredError()
		}
		if status < 200 || status >= 300 {
			return apiError(status, body)
		}

		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			return parseResponseError(err)
		}

		contacts := getArray(payload, "Contacts")
		return emitData(payload, func() {
			displayContacts(contacts)
		})
	},
}

var contactsGetCmd = &cobra.Command{
	Use:   "get <contact_id>",
	Short: "Get a single contact by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactID := args[0]
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/Contacts/%s", xeroAPIBase, contactID)
		if resolvedOutputFormat() == outputTable {
			fmt.Printf("Fetching contact %s...\n\n", contactID)
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
			contacts := getArray(payload, "Contacts")
			if len(contacts) == 0 {
				return notFoundError("contact not found")
			}
			if contact, ok := contacts[0].(map[string]any); ok {
				return emitData(payload, func() {
					displayContactDetail(contact)
				})
			}
			return unexpectedResponseError()
		case 401:
			return authenticationExpiredError()
		case 404:
			return notFoundError("contact not found")
		default:
			return apiError(status, body)
		}
	},
}

func init() {
	contactsCmd.AddCommand(contactsListCmd)
	contactsCmd.AddCommand(contactsGetCmd)
	contactsListCmd.Flags().Int("page", 1, "Page number for pagination")
	contactsListCmd.Flags().Int("page-size", 100, "Number of items per page")
}

func displayContacts(items []any) {
	if len(items) == 0 {
		fmt.Println("No contacts found.")
		return
	}

	fmt.Printf("Found %d contact(s):\n", len(items))
	fmt.Println()
	ui.PrintHeaderLine(120)
	header := ui.FormatRow(
		ui.Pad("Name", 30),
		ui.Pad("Email", 35),
		ui.Pad("Contact ID", 38),
		ui.Pad("Status", 15),
	)
	fmt.Println(header)
	ui.PrintHeaderLine(120)

	for _, item := range items {
		contact, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringValue(contact, "Name", "N/A")
		email := stringValue(contact, "EmailAddress", "N/A")
		contactID := stringValue(contact, "ContactID", "N/A")
		status := stringValue(contact, "ContactStatus", "N/A")

		row := ui.FormatRow(
			ui.Pad(name, 30),
			ui.Pad(email, 35),
			ui.Pad(contactID, 38),
			ui.Pad(status, 15),
		)
		fmt.Println(row)
	}

	ui.PrintHeaderLine(120)
}

func displayContactDetail(contact map[string]any) {
	fmt.Println("Contact Details:")
	fmt.Println()
	ui.PrintHeaderLine(80)

	fmt.Printf("Name:             %s\n", stringValue(contact, "Name", "N/A"))
	fmt.Printf("First Name:       %s\n", stringValue(contact, "FirstName", "N/A"))
	fmt.Printf("Last Name:        %s\n", stringValue(contact, "LastName", "N/A"))
	fmt.Printf("Contact ID:       %s\n", stringValue(contact, "ContactID", "N/A"))
	fmt.Printf("Email:            %s\n", stringValue(contact, "EmailAddress", "N/A"))
	fmt.Printf("Status:           %s\n", stringValue(contact, "ContactStatus", "N/A"))

	addresses := getArray(contact, "Addresses")
	if len(addresses) > 0 {
		fmt.Println("\nAddresses:")
		for _, addressItem := range addresses {
			address, ok := addressItem.(map[string]any)
			if !ok {
				continue
			}
			addrType := stringValue(address, "AddressType", "N/A")
			line1 := stringValue(address, "AddressLine1", "")
			line2 := stringValue(address, "AddressLine2", "")
			city := stringValue(address, "City", "")
			region := stringValue(address, "Region", "")
			postal := stringValue(address, "PostalCode", "")
			country := stringValue(address, "Country", "")

			fmt.Printf("  %s:\n", addrType)
			if line1 != "" {
				fmt.Printf("    %s\n", line1)
			}
			if line2 != "" {
				fmt.Printf("    %s\n", line2)
			}
			location := strings.TrimSpace(strings.Join(filterEmpty([]string{city, region, postal, country}), ", "))
			if location != "" {
				fmt.Printf("    %s\n", location)
			}
		}
	}

	phones := getArray(contact, "Phones")
	if len(phones) > 0 {
		fmt.Println("\nPhones:")
		for _, phoneItem := range phones {
			phone, ok := phoneItem.(map[string]any)
			if !ok {
				continue
			}
			typeValue := stringValue(phone, "PhoneType", "N/A")
			number := stringValue(phone, "PhoneNumber", "N/A")
			fmt.Printf("  %s: %s\n", typeValue, number)
		}
	}

	ui.PrintHeaderLine(80)
}
