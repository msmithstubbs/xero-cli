package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestReadStructuredInputRejectsMultipleSources(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("input", "", "")
	cmd.Flags().String("input-file", "", "")
	cmd.Flags().String("body", "", "")

	if err := cmd.Flags().Set("input", `{"a":1}`); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("body", `{"b":2}`); err != nil {
		t.Fatal(err)
	}

	if _, err := readStructuredInput(cmd); err == nil {
		t.Fatal("expected multiple input sources to fail")
	}
}

func TestParseStructuredJSONObjectInputFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.json")
	if err := os.WriteFile(path, []byte(`{"Name":"Acme"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("input", "", "")
	cmd.Flags().String("input-file", "", "")
	cmd.Flags().String("body", "", "")
	if err := cmd.Flags().Set("input-file", path); err != nil {
		t.Fatal(err)
	}

	decoded, err := parseStructuredJSONObjectInput(cmd)
	if err != nil {
		t.Fatalf("parseStructuredJSONObjectInput failed: %v", err)
	}
	if decoded["Name"] != "Acme" {
		t.Fatalf("expected Name=Acme, got %#v", decoded["Name"])
	}
}

func TestExecuteMutationDryRun(t *testing.T) {
	originalStdout := stdoutWriter
	originalOutput := outputFormat
	originalDryRun := dryRun
	t.Cleanup(func() {
		stdoutWriter = originalStdout
		outputFormat = originalOutput
		dryRun = originalDryRun
	})

	var out bytes.Buffer
	stdoutWriter = &out
	outputFormat = outputJSON
	dryRun = true

	err := executeMutation("POST", "https://example.test/resource", map[string]string{
		"authorization": "Bearer secret",
	}, []byte(`{"hello":"world"}`), "")
	if err != nil {
		t.Fatalf("executeMutation dry run failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode dry run output: %v", err)
	}
	if decoded["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got %#v", decoded["dry_run"])
	}
}
