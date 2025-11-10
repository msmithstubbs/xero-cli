defmodule XeroCLI do
  @moduledoc """
  Xero CLI - A command-line interface for interacting with the Xero API.

  This tool provides a GitHub CLI-like interface for managing your Xero
  accounting data from the command line. It supports OAuth 2.0 authentication
  and various API operations.

  ## Features

  - OAuth 2.0 authentication with PKCE
  - Automatic token refresh
  - Secure credential storage
  - Invoice management
  - And more to come...

  ## Usage

  See the CLI module for command documentation.
  """

  @doc """
  Returns the version of the Xero CLI.
  """
  def version do
    "0.1.0"
  end
end
