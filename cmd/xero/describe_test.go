package main

import "testing"

func TestBuildCommandDescriptionRootIncludesSubcommands(t *testing.T) {
	description := buildCommandDescription(rootCmd, true)

	if description.Name != "xero" {
		t.Fatalf("expected root name xero, got %q", description.Name)
	}
	if len(description.Subcommands) == 0 {
		t.Fatal("expected root description to include subcommands")
	}

	foundDescribe := false
	for _, child := range description.Subcommands {
		if child.Name == "describe" {
			foundDescribe = true
			break
		}
	}
	if !foundDescribe {
		t.Fatal("expected describe command to be listed")
	}
}

func TestBuildCommandDescriptionMarksMutationAndInheritedFlags(t *testing.T) {
	description := buildCommandDescription(paymentsCreateCmd, false)

	if !description.Mutating {
		t.Fatal("expected payments create to be marked mutating")
	}
	if !description.SupportsDryRun {
		t.Fatal("expected payments create to support dry run")
	}
	if !description.SupportsStructuredInput {
		t.Fatal("expected payments create to support structured input")
	}

	hasOutput := false
	hasTenant := false
	for _, flag := range description.Flags {
		if flag.Name == "output" && flag.Persistent {
			hasOutput = true
		}
		if flag.Name == "tenant-id" && flag.Persistent {
			hasTenant = true
		}
	}

	if !hasOutput {
		t.Fatal("expected inherited output flag to be described")
	}
	if !hasTenant {
		t.Fatal("expected inherited tenant-id flag to be described")
	}
}

func TestBuildCommandDescriptionIncludesHTTPSProxyForNetworkCommands(t *testing.T) {
	description := buildCommandDescription(authLoginCmd, false)

	foundHTTPSProxy := false
	for _, env := range description.EnvVars {
		if env == "HTTPS_PROXY" {
			foundHTTPSProxy = true
			break
		}
	}

	if !foundHTTPSProxy {
		t.Fatal("expected HTTPS_PROXY to be described for network commands")
	}
}
