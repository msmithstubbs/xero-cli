package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestValidateOutputFormat(t *testing.T) {
	valid := []string{"auto", "table", "json", "jsonl"}
	for _, mode := range valid {
		if err := validateOutputFormat(mode); err != nil {
			t.Fatalf("expected %q to be valid: %v", mode, err)
		}
	}

	if err := validateOutputFormat("xml"); err == nil {
		t.Fatal("expected xml to be rejected")
	}
}

func TestEmitDataWithModeJSONHonorsFields(t *testing.T) {
	originalStdout := stdoutWriter
	originalFields := fieldsFlag
	originalRedact := redactOutput
	t.Cleanup(func() {
		stdoutWriter = originalStdout
		fieldsFlag = originalFields
		redactOutput = originalRedact
	})

	var out bytes.Buffer
	stdoutWriter = &out
	fieldsFlag = "Invoices"
	redactOutput = true

	payload := map[string]any{
		"Invoices": []any{
			map[string]any{"InvoiceID": "123"},
		},
		"Unused": true,
	}

	if err := emitDataWithMode(payload, nil, outputJSON); err != nil {
		t.Fatalf("emitDataWithMode failed: %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode output: %v", err)
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", envelope["ok"])
	}

	data, ok := envelope["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %#v", envelope["data"])
	}

	serialized, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to remarshal data: %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(serialized, &decoded); err != nil {
		t.Fatalf("failed to decode selected data: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected one invoice in output, got %d", len(decoded))
	}
	if decoded[0]["InvoiceID"] != "123" {
		t.Fatalf("expected InvoiceID to survive field selection, got %#v", decoded[0]["InvoiceID"])
	}
}

func TestEmitDataWithModeJSONL(t *testing.T) {
	originalStdout := stdoutWriter
	t.Cleanup(func() {
		stdoutWriter = originalStdout
	})

	var out bytes.Buffer
	stdoutWriter = &out

	data := []any{
		map[string]any{"id": "1"},
		map[string]any{"id": "2"},
	}
	if err := emitDataWithMode(data, nil, outputJSONL); err != nil {
		t.Fatalf("emitDataWithMode failed: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("expected 2 jsonl lines, got %d", len(lines))
	}
}

func TestWriteCommandErrorJSON(t *testing.T) {
	originalOutput := outputFormat
	t.Cleanup(func() {
		outputFormat = originalOutput
	})

	outputFormat = outputJSON
	var out bytes.Buffer
	writeCommandError(&out, apiError(400, []byte(`{"Detail":"bad request"}`)))

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode error output: %v", err)
	}

	if okValue, ok := decoded["ok"].(bool); !ok || okValue {
		t.Fatalf("expected ok=false, got %#v", decoded["ok"])
	}
}

func TestEmitDryRunRedactsAuthorization(t *testing.T) {
	originalStdout := stdoutWriter
	originalOutput := outputFormat
	originalRedact := redactOutput
	t.Cleanup(func() {
		stdoutWriter = originalStdout
		outputFormat = originalOutput
		redactOutput = originalRedact
	})

	var out bytes.Buffer
	stdoutWriter = &out
	outputFormat = outputJSON
	redactOutput = true

	err := emitDryRun("POST", "https://example.test", map[string]string{
		"authorization": "Bearer secret",
		"accept":        "application/json",
	}, []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("emitDryRun failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode dry run output: %v", err)
	}

	request := decoded["data"].(map[string]any)["request"].(map[string]any)
	headers := request["headers"].(map[string]any)
	if headers["authorization"] != "REDACTED" {
		t.Fatalf("expected authorization to be redacted, got %#v", headers["authorization"])
	}
}

func TestSanitizeValueRedactsSensitiveKeys(t *testing.T) {
	value := map[string]any{
		"access_token": "secret",
		"nested": map[string]any{
			"client_id": "abc",
		},
	}

	sanitized := sanitizeValue(value).(map[string]any)
	if sanitized["access_token"] != "REDACTED" {
		t.Fatalf("expected access_token to be redacted, got %#v", sanitized["access_token"])
	}
	nested := sanitized["nested"].(map[string]any)
	if nested["client_id"] != "REDACTED" {
		t.Fatalf("expected nested client_id to be redacted, got %#v", nested["client_id"])
	}
}
