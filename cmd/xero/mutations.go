package main

import (
	"fmt"

	"github.com/msmithstubbs/xero-cli/internal/xero"
)

func executeMutation(method, endpoint string, headers map[string]string, payload []byte, emptyMessage string) error {
	if dryRun {
		return emitDryRun(method, endpoint, headers, payload)
	}

	client := xero.NewClient(xeroAPIBase)
	statusCode, body, err := client.Do(method, endpoint, headers, payload)
	if err != nil {
		return internalError("request failed", err)
	}
	if statusCode == 401 {
		return authenticationExpiredError()
	}
	if statusCode < 200 || statusCode >= 300 {
		return apiError(statusCode, body)
	}

	if len(body) == 0 && emptyMessage != "" {
		if resolvedOutputFormat() == outputTable {
			fmt.Fprintln(stdoutWriter, emptyMessage)
			return nil
		}
		return emitData(map[string]any{
			"ok":      true,
			"message": emptyMessage,
		}, nil)
	}

	return emitJSONBody(body, nil)
}
