package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func readStructuredInput(cmd *cobra.Command) ([]byte, error) {
	input, _ := cmd.Flags().GetString("input")
	inputFile, _ := cmd.Flags().GetString("input-file")
	body, _ := cmd.Flags().GetString("body")

	sourceCount := 0
	if strings.TrimSpace(input) != "" {
		sourceCount++
	}
	if strings.TrimSpace(inputFile) != "" {
		sourceCount++
	}
	if strings.TrimSpace(body) != "" {
		sourceCount++
	}
	if sourceCount > 1 {
		return nil, validationError("use only one of --input, --input-file, or --body")
	}
	if strings.TrimSpace(input) != "" {
		return []byte(input), nil
	}
	if strings.TrimSpace(inputFile) != "" {
		return readPathOrStdin(strings.TrimSpace(inputFile))
	}
	if strings.TrimSpace(body) != "" {
		return []byte(body), nil
	}
	return nil, nil
}

func readBinaryInputPath(cmd *cobra.Command) (string, error) {
	filePath, _ := cmd.Flags().GetString("file")
	inputFile, _ := cmd.Flags().GetString("input-file")
	filePath = strings.TrimSpace(filePath)
	inputFile = strings.TrimSpace(inputFile)

	if filePath != "" && inputFile != "" {
		return "", validationError("use only one of --file or --input-file")
	}
	if inputFile != "" {
		return inputFile, nil
	}
	return filePath, nil
}

func readPathOrStdin(path string) ([]byte, error) {
	if path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, internalError("failed to read stdin", err)
		}
		return data, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, internalError(fmt.Sprintf("failed to read %s", path), err)
	}
	return data, nil
}

func parseStructuredJSONInput(cmd *cobra.Command) (any, error) {
	input, err := readStructuredInput(cmd)
	if err != nil || input == nil {
		return nil, err
	}

	var decoded any
	if err := json.Unmarshal(input, &decoded); err != nil {
		return nil, validationError(fmt.Sprintf("invalid JSON input: %v", err))
	}
	return decoded, nil
}

func parseStructuredJSONObjectInput(cmd *cobra.Command) (map[string]any, error) {
	input, err := parseStructuredJSONInput(cmd)
	if err != nil || input == nil {
		return nil, err
	}

	obj, ok := input.(map[string]any)
	if !ok {
		return nil, validationError("input must be a JSON object")
	}
	return obj, nil
}

func addStructuredInputFlags(cmd *cobra.Command, usage string) {
	cmd.Flags().String("input", "", usage)
	cmd.Flags().String("input-file", "", "Path to a JSON file, or '-' to read JSON from stdin")
}
