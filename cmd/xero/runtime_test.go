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
	t.Cleanup(func() {
		stdoutWriter = originalStdout
		fieldsFlag = originalFields
	})

	var out bytes.Buffer
	stdoutWriter = &out
	fieldsFlag = "Invoices"

	payload := map[string]any{
		"Invoices": []any{
			map[string]any{"InvoiceID": "123"},
		},
		"Unused": true,
	}

	if err := emitDataWithMode(payload, nil, outputJSON); err != nil {
		t.Fatalf("emitDataWithMode failed: %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode output: %v", err)
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
	t.Cleanup(func() {
		stdoutWriter = originalStdout
		outputFormat = originalOutput
	})

	var out bytes.Buffer
	stdoutWriter = &out
	outputFormat = outputJSON

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

	request := decoded["request"].(map[string]any)
	headers := request["headers"].(map[string]any)
	if headers["authorization"] != "REDACTED" {
		t.Fatalf("expected authorization to be redacted, got %#v", headers["authorization"])
	}
}
