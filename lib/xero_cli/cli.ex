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

  defp parse_args([unknown | _]) do
    {:error, "unknown command: #{unknown}"}
  end

  defp print_help do
    IO.puts("""
    Xero CLI - Command line tool for interacting with Xero API

    USAGE:
      xero <command> <subcommand> [flags]

    CORE COMMANDS:
      auth        Authenticate with Xero
      invoices    Manage invoices

    AUTH COMMANDS:
      xero auth login     Log in to Xero via OAuth 2.0
      xero auth logout    Log out and remove credentials
      xero auth status    Check authentication status

    INVOICE COMMANDS:
      xero invoices list  List all invoices

    FLAGS:
      --help, -h  Show help for command

    EXAMPLES:
      $ xero auth login
      $ xero invoices list

    Learn more at: https://developer.xero.com/
    """)
  end
end
