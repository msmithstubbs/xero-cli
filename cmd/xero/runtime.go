package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	outputAuto  = "auto"
	outputTable = "table"
	outputJSON  = "json"
	outputJSONL = "jsonl"
)

const (
	exitCodeGeneric    = 1
	exitCodeValidation = 2
	exitCodeAuth       = 3
	exitCodeNotFound   = 4
	exitCodeAPI        = 5
)

var (
	outputFormat string
	fieldsFlag   string
	dryRun       bool
	stdoutWriter io.Writer = os.Stdout
	stderrWriter io.Writer = os.Stderr
)

type commandError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code,omitempty"`
	Details    any    `json:"details,omitempty"`
	Cause      error  `json:"-"`
}

func (e *commandError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *commandError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func validationError(message string) error {
	return &commandError{Code: "validation_error", Message: message}
}

func authError(message string) error {
	return &commandError{Code: "auth_error", Message: message}
}

func notFoundError(message string) error {
	return &commandError{Code: "not_found", Message: message}
}

func apiError(status int, body []byte) error {
	return &commandError{
		Code:       "api_error",
		Message:    fmt.Sprintf("API request failed with status %d", status),
		StatusCode: status,
		Details:    sanitizeBody(body),
	}
}

func internalError(message string, cause error) error {
	return &commandError{Code: "internal_error", Message: message, Cause: cause}
}

func parseResponseError(err error) error {
	return &commandError{Code: "parse_error", Message: "failed to parse response", Cause: err}
}

func unexpectedResponseError() error {
	return &commandError{Code: "unexpected_response", Message: "unexpected response format"}
}

func tenantRequiredError() error {
	return validationError("tenant id is required. Provide --tenant-id or set XERO_TENANT_ID")
}

func authenticationExpiredError() error {
	return authError("authentication failed. Please run 'xero auth login' again")
}

func exitCodeForError(err error) int {
	var cmdErr *commandError
	if errors.As(err, &cmdErr) {
		switch cmdErr.Code {
		case "validation_error":
			return exitCodeValidation
		case "auth_error":
			return exitCodeAuth
		case "not_found":
			return exitCodeNotFound
		case "api_error":
			return exitCodeAPI
		default:
			return exitCodeGeneric
		}
	}
	return exitCodeGeneric
}

func writeCommandError(w io.Writer, err error) {
	mode := resolvedOutputFormat()
	var cmdErr *commandError
	if !errors.As(err, &cmdErr) {
		cmdErr = &commandError{Code: "internal_error", Message: err.Error()}
	}

	if mode == outputJSON || mode == outputJSONL {
		_ = writeJSON(w, map[string]any{
			"ok":    false,
			"error": cmdErr,
		}, false)
		return
	}

	fmt.Fprintln(w, cmdErr.Message)
	if cmdErr.StatusCode != 0 {
		fmt.Fprintf(w, "Status: %d\n", cmdErr.StatusCode)
	}
	if cmdErr.Details != nil {
		fmt.Fprintf(w, "Details: %v\n", cmdErr.Details)
	}
}

func validateOutputFormat(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case outputAuto, outputTable, outputJSON, outputJSONL:
		return nil
	default:
		return validationError("output must be one of: auto, table, json, jsonl")
	}
}

func resolvedOutputFormat() string {
	mode := strings.ToLower(strings.TrimSpace(outputFormat))
	if mode == "" || mode == outputAuto {
		if stdoutIsTTY() {
			return outputTable
		}
		return outputJSON
	}
	return mode
}

func stdoutIsTTY() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func emitData(data any, renderTable func()) error {
	return emitDataWithMode(data, renderTable, resolvedOutputFormat())
}

func emitDataWithMode(data any, renderTable func(), mode string) error {
	selected := applyFieldSelection(data)
	switch mode {
	case outputTable:
		if renderTable != nil {
			renderTable()
			return nil
		}
		return writeJSON(stdoutWriter, selected, true)
	case outputJSON:
		return writeJSON(stdoutWriter, selected, false)
	case outputJSONL:
		return writeJSONL(stdoutWriter, selected)
	default:
		return validationError("output must be one of: auto, table, json, jsonl")
	}
}

func emitJSONBody(body []byte, renderTable func(data any)) error {
	parsed, raw := decodeBody(body)
	mode := resolvedOutputFormat()
	if mode == outputTable && renderTable != nil && parsed != nil {
		renderTable(parsed)
		return nil
	}
	if parsed != nil {
		return emitData(parsed, nil)
	}
	fmt.Fprintln(stdoutWriter, raw)
	return nil
}

func decodeBody(body []byte) (any, string) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{"ok": true}, ""
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err == nil {
		return parsed, ""
	}
	return nil, trimmed
}

func applyFieldSelection(data any) any {
	fields := parseFields(fieldsFlag)
	if len(fields) == 0 {
		return data
	}

	if len(fields) == 1 {
		return lookupField(data, fields[0])
	}

	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		selected[field] = lookupField(data, field)
	}
	return selected
}

func parseFields(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			fields = append(fields, trimmed)
		}
	}
	sort.Strings(fields)
	return fields
}

func lookupField(data any, field string) any {
	current := data
	for _, part := range strings.Split(field, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = obj[part]
	}
	return current
}

func writeJSON(w io.Writer, data any, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

func writeJSONL(w io.Writer, data any) error {
	items, ok := data.([]any)
	if !ok {
		return writeJSON(w, data, false)
	}
	for _, item := range items {
		if err := writeJSON(w, item, false); err != nil {
			return err
		}
	}
	return nil
}

func sanitizeBody(body []byte) any {
	parsed, raw := decodeBody(body)
	if parsed != nil {
		return parsed
	}
	if raw == "" {
		return nil
	}
	return raw
}

func sanitizeHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	sanitized := make(map[string]string, len(headers))
	for key, value := range headers {
		lower := strings.ToLower(key)
		switch lower {
		case "authorization":
			sanitized[key] = "REDACTED"
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}

func emitDryRun(method, endpoint string, headers map[string]string, body []byte) error {
	payload := map[string]any{
		"ok":      true,
		"dry_run": true,
		"request": map[string]any{
			"method":  method,
			"url":     endpoint,
			"headers": sanitizeHeaders(headers),
			"body":    sanitizeBody(body),
		},
	}

	if resolvedOutputFormat() == outputTable {
		fmt.Fprintln(stdoutWriter, "Dry run")
		fmt.Fprintf(stdoutWriter, "Method:  %s\n", method)
		fmt.Fprintf(stdoutWriter, "URL:     %s\n", endpoint)
		fmt.Fprintf(stdoutWriter, "Headers: %v\n", sanitizeHeaders(headers))
		if len(strings.TrimSpace(string(body))) > 0 {
			fmt.Fprintf(stdoutWriter, "Body:    %s\n", strings.TrimSpace(string(body)))
		}
		return nil
	}

	return emitData(payload, nil)
}
