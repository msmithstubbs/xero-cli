package main

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type commandDescription struct {
	Name                    string               `json:"name"`
	Path                    string               `json:"path"`
	Use                     string               `json:"use"`
	Short                   string               `json:"short,omitempty"`
	Long                    string               `json:"long,omitempty"`
	Hidden                  bool                 `json:"hidden,omitempty"`
	Runnable                bool                 `json:"runnable"`
	Mutating                bool                 `json:"mutating"`
	SupportsDryRun          bool                 `json:"supports_dry_run"`
	SupportsStructuredInput bool                 `json:"supports_structured_input"`
	SupportsStructuredOut   bool                 `json:"supports_structured_output"`
	EnvVars                 []string             `json:"env_vars,omitempty"`
	Flags                   []flagDescription    `json:"flags,omitempty"`
	Subcommands             []commandDescription `json:"subcommands,omitempty"`
}

type flagDescription struct {
	Name       string `json:"name"`
	Shorthand  string `json:"shorthand,omitempty"`
	Type       string `json:"type,omitempty"`
	Default    string `json:"default,omitempty"`
	Usage      string `json:"usage,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
	Required   bool   `json:"required,omitempty"`
}

var describeCmd = &cobra.Command{
	Use:   "describe [command...]",
	Short: "Describe the CLI in machine-readable form",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := rootCmd
		if len(args) > 0 {
			found, _, err := rootCmd.Find(args)
			if err != nil {
				return validationError(err.Error())
			}
			target = found
		}
		return emitDataWithMode(buildCommandDescription(target, len(args) == 0), nil, outputJSON)
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func buildCommandDescription(cmd *cobra.Command, recursive bool) commandDescription {
	flags := collectFlags(cmd)
	description := commandDescription{
		Name:                    cmd.Name(),
		Path:                    cmd.CommandPath(),
		Use:                     cmd.Use,
		Short:                   cmd.Short,
		Long:                    cmd.Long,
		Hidden:                  cmd.Hidden,
		Runnable:                cmd.Runnable(),
		Mutating:                isMutatingCommand(cmd),
		SupportsDryRun:          isMutatingCommand(cmd),
		SupportsStructuredInput: supportsStructuredInput(cmd),
		SupportsStructuredOut:   true,
		EnvVars:                 envVarsForCommand(cmd),
		Flags:                   flags,
	}

	children := visibleChildren(cmd)
	if recursive {
		description.Subcommands = make([]commandDescription, 0, len(children))
		for _, child := range children {
			description.Subcommands = append(description.Subcommands, buildCommandDescription(child, false))
		}
	}

	return description
}

func visibleChildren(cmd *cobra.Command) []*cobra.Command {
	children := make([]*cobra.Command, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Name() < children[j].Name()
	})
	return children
}

func collectFlags(cmd *cobra.Command) []flagDescription {
	items := make([]flagDescription, 0)
	seen := make(map[string]bool)

	appendFlag := func(flag *pflag.Flag, persistent bool) {
		if flag == nil || seen[flag.Name] {
			return
		}
		seen[flag.Name] = true
		items = append(items, flagDescription{
			Name:       flag.Name,
			Shorthand:  flag.Shorthand,
			Type:       flag.Value.Type(),
			Default:    flag.DefValue,
			Usage:      flag.Usage,
			Persistent: persistent,
			Required:   cmd.Flag(flag.Name) != nil && cmd.Flag(flag.Name).Annotations != nil && len(cmd.Flag(flag.Name).Annotations[cobra.BashCompOneRequiredFlag]) > 0,
		})
	}

	cmd.NonInheritedFlags().VisitAll(func(flag *pflag.Flag) {
		appendFlag(flag, false)
	})
	cmd.InheritedFlags().VisitAll(func(flag *pflag.Flag) {
		appendFlag(flag, true)
	})

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

func isMutatingCommand(cmd *cobra.Command) bool {
	switch cmd.CommandPath() {
	case "xero contacts create",
		"xero invoices attach",
		"xero invoices create",
		"xero invoices update",
		"xero payments create",
		"xero payments delete",
		"xero payments update",
		"xero banking transactions":
		return true
	default:
		return false
	}
}

func supportsStructuredInput(cmd *cobra.Command) bool {
	for _, flagName := range []string{"body", "file", "input", "input-file"} {
		if cmd.Flags().Lookup(flagName) != nil {
			return true
		}
	}
	return false
}

func envVarsForCommand(cmd *cobra.Command) []string {
	envs := make([]string, 0, 3)
	switch cmd.CommandPath() {
	case "xero auth login":
		envs = append(envs, "XERO_CLIENT_ID", "XERO_PKCE_VERIFIER")
	}

	if strings.HasPrefix(cmd.CommandPath(), "xero auth") || strings.HasPrefix(cmd.CommandPath(), "xero describe") || strings.HasPrefix(cmd.CommandPath(), "xero tenants") {
		sort.Strings(envs)
		return envs
	}

	envs = append(envs, "XERO_TENANT_ID")
	sort.Strings(envs)
	return envs
}
