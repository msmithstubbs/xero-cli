package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/credentials"
)

func authHeaders(creds *credentials.Credentials) (map[string]string, error) {
	tenantID := strings.TrimSpace(tenantOverride)
	if tenantID == "" {
		tenantID = strings.TrimSpace(os.Getenv("XERO_TENANT_ID"))
	}
	if tenantID == "" {
		return nil, tenantRequiredError()
	}
	return map[string]string{
		"authorization":  "Bearer " + creds.AccessToken,
		"xero-tenant-id": tenantID,
		"accept":         "application/json",
	}, nil
}

func stringValue(data map[string]any, key, fallback string) string {
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case string:
			if v != "" {
				return v
			}
		case fmt.Stringer:
			return v.String()
		}
	}
	return fallback
}

func getArray(data map[string]any, key string) []any {
	if value, ok := data[key]; ok {
		switch v := value.(type) {
		case []any:
			return v
		}
	}
	return nil
}

func filterEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func validateInvoiceStatus(value string) (string, error) {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return "", errors.New("invoice status cannot be empty")
	}
	switch trimmed {
	case "DRAFT", "SUBMITTED", "AUTHORISED", "PAID", "VOIDED", "DELETED":
		return trimmed, nil
	default:
		return "", fmt.Errorf("invalid invoice status %q; use DRAFT, SUBMITTED, AUTHORISED, PAID, VOIDED, or DELETED", value)
	}
}
