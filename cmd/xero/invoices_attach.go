package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/msmithstubbs/xero-cli/internal/auth"
	"github.com/msmithstubbs/xero-cli/internal/xero"
	"github.com/spf13/cobra"
)

var invoicesAttachCmd = &cobra.Command{
	Use:   "attach <invoice_id>",
	Short: "Attach a PDF to an invoice",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		creds, err := auth.GetValidCredentials()
		if err != nil {
			return err
		}

		invoiceID := strings.TrimSpace(args[0])
		if invoiceID == "" {
			return validationError("invoice_id is required")
		}

		filePath, err := readBinaryInputPath(cmd)
		if err != nil {
			return err
		}
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			return validationError("--file is required")
		}

		name, _ := cmd.Flags().GetString("name")
		attachment, attachmentName, err := loadInvoiceAttachment(filePath, strings.TrimSpace(name))
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf(
			"%s/Invoices/%s/Attachments/%s",
			xeroAPIBase,
			invoiceID,
			url.PathEscape(attachmentName),
		)

		headers, err := authHeaders(creds)
		if err != nil {
			return err
		}
		headers["content-type"] = "application/pdf"

		if dryRun {
			return emitDryRun("POST", endpoint, headers, attachment)
		}

		client := xero.NewClient(xeroAPIBase)
		status, body, err := client.Do("POST", endpoint, headers, attachment)
		if err != nil {
			return internalError("request failed", err)
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose {
			fmt.Fprintf(stdoutWriter, "HTTP Status: %d\nResponse: %s\n", status, string(body))
		}

		if status == 401 {
			return authenticationExpiredError()
		}
		if status < 200 || status >= 300 {
			return apiError(status, body)
		}

		if len(body) == 0 {
			message := fmt.Sprintf("Attachment %s uploaded.", attachmentName)
			if resolvedOutputFormat() == outputTable {
				fmt.Fprintln(stdoutWriter, message)
				return nil
			}
			return emitData(map[string]any{
				"ok":      true,
				"message": message,
			}, nil)
		}
		return emitJSONBody(body, nil)
	},
}

func init() {
	invoicesCmd.AddCommand(invoicesAttachCmd)
	invoicesAttachCmd.Flags().String("file", "", "Path to PDF file to attach")
	invoicesAttachCmd.Flags().String("input-file", "", "Path to a PDF file, or '-' to read a PDF from stdin")
	invoicesAttachCmd.Flags().String("name", "", "Attachment file name (defaults to the PDF base name)")
	invoicesAttachCmd.Flags().Bool("verbose", false, "Print raw API response to stdout")
}

func loadInvoiceAttachment(path, name string) ([]byte, string, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", errors.New("file is empty")
	}

	attachmentName := strings.TrimSpace(name)
	if attachmentName == "" {
		if path == "-" {
			return nil, "", errors.New("--name is required when --file is '-'")
		}
		attachmentName = filepath.Base(path)
	}
	if attachmentName == "" {
		return nil, "", errors.New("attachment name is required")
	}

	if !strings.HasSuffix(strings.ToLower(attachmentName), ".pdf") {
		return nil, "", errors.New("attachment must be a .pdf file")
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, "", errors.New("file does not appear to be a PDF")
	}

	return data, attachmentName, nil
}
