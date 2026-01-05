package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xero",
	Short: "Xero CLI - Command line tool for interacting with Xero API",
	Long:  "A command-line interface for interacting with the Xero API, modeled after the GitHub CLI.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(invoicesCmd)
	rootCmd.AddCommand(contactsCmd)
	rootCmd.AddCommand(accountsCmd)
	rootCmd.AddCommand(currenciesCmd)
	rootCmd.AddCommand(bankTransactionsCmd)
}
