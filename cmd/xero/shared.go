package main

import (
	"fmt"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/config"
)

func authHeaders(creds *config.Credentials) map[string]string {
	tenantID := creds.TenantID
	if strings.TrimSpace(tenantOverride) != "" {
		tenantID = strings.TrimSpace(tenantOverride)
	}
	return map[string]string{
		"authorization":  "Bearer " + creds.AccessToken,
		"xero-tenant-id": tenantID,
		"accept":         "application/json",
	}
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
