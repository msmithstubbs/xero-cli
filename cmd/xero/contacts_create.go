package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var contactsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a contact",
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		bodyAttrs, err := parseContactBody(cmd)
		if err != nil {
			return err
		}

		name, _ := cmd.Flags().GetString("name")
		firstName, _ := cmd.Flags().GetString("first-name")
		lastName, _ := cmd.Flags().GetString("last-name")
		email, _ := cmd.Flags().GetString("email")
		name = strings.TrimSpace(name)
		firstName = strings.TrimSpace(firstName)
		lastName = strings.TrimSpace(lastName)
		email = strings.TrimSpace(email)

		contact := cloneMap(bodyAttrs)

		if cmd.Flags().Changed("name") {
			if name == "" {
				return errors.New("--name cannot be empty")
			}
			contact["Name"] = name
		}
		if cmd.Flags().Changed("first-name") {
			if firstName == "" {
				return errors.New("--first-name cannot be empty")
			}
			contact["FirstName"] = firstName
		}
		if cmd.Flags().Changed("last-name") {
			if lastName == "" {
				return errors.New("--last-name cannot be empty")
			}
			contact["LastName"] = lastName
		}
		if cmd.Flags().Changed("email") {
			if email == "" {
				return errors.New("--email cannot be empty")
			}
			contact["EmailAddress"] = email
		}

		if !hasKey(contact, "Name") && !hasKey(contact, "FirstName") && !hasKey(contact, "LastName") {
			return errors.New("--name is required (or provide Name/FirstName/LastName in --body)")
		}

		payload, err := json.Marshal(map[string]any{"Contacts": []any{contact}})
		if err != nil {
			return fmt.Errorf("failed to build contact payload: %w", err)
		}

		params := url.Values{}
		if cmd.Flags().Changed("summarize-errors") {
			summarize, _ := cmd.Flags().GetBool("summarize-errors")
			params.Set("summarizeErrors", fmt.Sprintf("%t", summarize))
		}

		endpoint := fmt.Sprintf("%s/Contacts", xeroAPIBase)
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
	contactsCmd.AddCommand(contactsCreateCmd)
	contactsCreateCmd.Flags().String("name", "", "Contact name")
	contactsCreateCmd.Flags().String("first-name", "", "Contact first name")
	contactsCreateCmd.Flags().String("last-name", "", "Contact last name")
	contactsCreateCmd.Flags().String("email", "", "Contact email address")
	contactsCreateCmd.Flags().String("body", "", "Raw JSON object of contact attributes")
	contactsCreateCmd.Flags().Bool("summarize-errors", false, "Summarize validation errors in the response")
	contactsCreateCmd.Flags().String("idempotency-key", "", "Idempotency key for safely retrying requests")
}

func parseContactBody(cmd *cobra.Command) (map[string]any, error) {
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
