defmodule XeroCLI.CLI do
  @moduledoc """
  Main CLI entry point for the Xero CLI tool.
  Modeled after GitHub CLI's command structure.
  """

  def main(args) do
    case parse_args(args) do
      {:auth, subcommand, opts} ->
        XeroCLI.Commands.Auth.handle(subcommand, opts)

      {:invoices, subcommand, opts} ->
        XeroCLI.Commands.Invoices.handle(subcommand, opts)

      {:contacts, subcommand, opts} ->
        XeroCLI.Commands.Contacts.handle(subcommand, opts)

      {:accounts, subcommand, opts} ->
        XeroCLI.Commands.Accounts.handle(subcommand, opts)

      {:currencies, subcommand, opts} ->
        XeroCLI.Commands.Currencies.handle(subcommand, opts)

      {:help} ->
        print_help()

      {:error, message} ->
        IO.puts(:stderr, "Error: #{message}")
        print_help()
        System.halt(1)
    end
  end

  defp parse_args([]) do
    {:help}
  end

  defp parse_args(["--help" | _]) do
    {:help}
  end

  defp parse_args(["-h" | _]) do
    {:help}
  end

  defp parse_args(["auth" | rest]) do
    case rest do
      ["login" | opts] -> {:auth, :login, opts}
      ["logout" | opts] -> {:auth, :logout, opts}
      ["status" | opts] -> {:auth, :status, opts}
      [] -> {:error, "auth command requires a subcommand"}
      [unknown | _] -> {:error, "unknown auth subcommand: #{unknown}"}
    end
  end

  defp parse_args(["invoices" | rest]) do
    case rest do
      ["list" | opts] -> {:invoices, :list, opts}
      [] -> {:error, "invoices command requires a subcommand"}
      [unknown | _] -> {:error, "unknown invoices subcommand: #{unknown}"}
    end
  end

  defp parse_args(["contacts" | rest]) do
    case rest do
      ["list" | opts] -> {:contacts, :list, opts}
      ["get" | opts] -> {:contacts, :get, opts}
      [] -> {:error, "contacts command requires a subcommand"}
      [unknown | _] -> {:error, "unknown contacts subcommand: #{unknown}"}
    end
  end

  defp parse_args(["accounts" | rest]) do
    case rest do
      ["list" | opts] -> {:accounts, :list, opts}
      ["get" | opts] -> {:accounts, :get, opts}
      [] -> {:error, "accounts command requires a subcommand"}
      [unknown | _] -> {:error, "unknown accounts subcommand: #{unknown}"}
    end
  end

  defp parse_args(["currencies" | rest]) do
    case rest do
      ["list" | opts] -> {:currencies, :list, opts}
      ["get" | opts] -> {:currencies, :get, opts}
      [] -> {:error, "currencies command requires a subcommand"}
      [unknown | _] -> {:error, "unknown currencies subcommand: #{unknown}"}
    end
  end

  defp parse_args([unknown | _]) do
    {:error, "unknown command: #{unknown}"}
  end

  defp print_help do
    IO.puts("""
    Xero CLI - Command line tool for interacting with Xero API

    USAGE:
      xero <command> <subcommand> [flags]

    CORE COMMANDS:
      auth         Authenticate with Xero
      invoices     Manage invoices
      contacts     Manage contacts
      accounts     Manage accounts
      currencies   Manage currencies

    AUTH COMMANDS:
      xero auth login     Log in to Xero via OAuth 2.0
      xero auth logout    Log out and remove credentials
      xero auth status    Check authentication status

    INVOICE COMMANDS:
      xero invoices list  List all invoices

    CONTACT COMMANDS:
      xero contacts list           List all contacts
      xero contacts get <id>       Get a single contact by ID

    ACCOUNT COMMANDS:
      xero accounts list           List all accounts
      xero accounts get <id>       Get a single account by ID

    CURRENCY COMMANDS:
      xero currencies list         List all currencies
      xero currencies get <code>   Get a single currency by code (e.g., USD, EUR)

    FLAGS:
      --help, -h       Show help for command
      --page <num>     Page number for pagination (default: 1)
      --page-size <n>  Number of items per page (default: 100)

    EXAMPLES:
      $ xero auth login
      $ xero invoices list
      $ xero contacts list --page 2 --page-size 50
      $ xero contacts get abc123-def456-ghi789
      $ xero accounts list
      $ xero accounts get xyz789-uvw456-rst123
      $ xero currencies list
      $ xero currencies get USD

    Learn more at: https://developer.xero.com/
    """)
  end
end
