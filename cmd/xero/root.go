package main

import (
	"os"

	"github.com/spf13/cobra"
)

var tenantOverride string

var rootCmd = &cobra.Command{
	Use:           "xero",
	Short:         "Xero CLI - Command line tool for interacting with Xero API",
	Long:          "A command-line interface for interacting with the Xero API, modeled after the GitHub CLI.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return validateOutputFormat(outputFormat)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		writeCommandError(stderrWriter, err)
		os.Exit(exitCodeForError(err))
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&tenantOverride, "tenant-id", "", "Tenant ID to use for this request")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", outputAuto, "Output format: auto, table, json, jsonl")
	rootCmd.PersistentFlags().StringVar(&fieldsFlag, "fields", "", "Comma-separated field paths to include in JSON output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Preview mutating requests without sending them")
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(invoicesCmd)
	rootCmd.AddCommand(contactsCmd)
	rootCmd.AddCommand(accountsCmd)
	rootCmd.AddCommand(currenciesCmd)
	rootCmd.AddCommand(bankingCmd)
	rootCmd.AddCommand(paymentsCmd)
	rootCmd.AddCommand(tenantsCmd)
}
