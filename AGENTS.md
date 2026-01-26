After completing any reasonable amount of work, run `go build` to check the project compiles.

Commit changes at regular intervals, such as completing a feature or fixing a bug.

You can test the Xero cli again the Demo Company tenant.
If demo company isn't available ask the user to run `xero auth login` and grant access again.
ONLY use the Demo Company tenant.

Xero API docs are available at https://developer.xero.com/documentation/api/accounting/overview

## Command Help Documentation

All new commands must provide built-in help using Cobra's fields:

**Required for every command:**
- `Use` - Command name with argument placeholders (e.g., `"get <contact_id>"`)
- `Short` - One-line description shown in parent command's help

**Required for parent commands (commands with subcommands):**
- `Long` - Detailed description explaining the command's purpose

**Required for all flags:**
- Descriptive help text as the third parameter to `Flags()` calls
- Include format specifications (e.g., `"Invoice date in YYYY-MM-DD"`)
- Document default values in help text when meaningful (e.g., `"defaults to today"`)
- Provide examples for complex values (e.g., `"Filter with where clause (e.g. Status==\"AUTHORISED\")"`)

Example:
```go
var exampleCmd = &cobra.Command{
    Use:   "example <id>",
    Short: "Brief one-line description",
    Long:  "Detailed description for parent commands.",
    Args:  cobra.ExactArgs(1),
    RunE:  runExample,
}

func init() {
    exampleCmd.Flags().String("date", "", "Date in YYYY-MM-DD format (defaults to today)")
    exampleCmd.Flags().String("status", "DRAFT", "Status (DRAFT, SUBMITTED, AUTHORISED)")
}
