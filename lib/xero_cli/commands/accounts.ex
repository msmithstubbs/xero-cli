defmodule XeroCLI.Commands.Accounts do
  @moduledoc """
  Handles account-related commands for Xero CLI.
  """

  @xero_api_base "https://api.xero.com/api.xro/2.0"

  def handle(:list, opts) do
    case get_valid_credentials() do
      {:ok, credentials} ->
        access_token = Map.get(credentials, "access_token")
        tenant_id = Map.get(credentials, "tenant_id")

        # Parse options
        page = get_option(opts, "--page", "1") |> String.to_integer()
        page_size = get_option(opts, "--page-size", "100") |> String.to_integer()

        # Build query parameters
        query_params =
          []
          |> maybe_add_param("page", page)
          |> maybe_add_param("pageSize", page_size)
          |> URI.encode_query()

        url = "#{@xero_api_base}/Accounts?#{query_params}"

        IO.puts("💰 Fetching accounts...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                accounts = get_in(data, ["Accounts"]) || []
                display_accounts(accounts)

              {:error, reason} ->
                IO.puts(:stderr, "❌ Failed to parse response: #{inspect(reason)}")
                System.halt(1)
            end

          {:ok, %{status_code: 401}} ->
            IO.puts(:stderr, "❌ Authentication failed. Please run 'xero auth login' again.")
            System.halt(1)

          {:ok, %{status_code: status, body: body}} ->
            IO.puts(:stderr, "❌ API request failed with status #{status}")
            IO.puts(:stderr, "Response: #{body}")
            System.halt(1)

          {:error, reason} ->
            IO.puts(:stderr, "❌ Request failed: #{inspect(reason)}")
            System.halt(1)
        end

      {:error, reason} ->
        IO.puts(:stderr, "❌ #{reason}")
        IO.puts("Please run 'xero auth login' first.")
        System.halt(1)
    end
  end

  def handle(:get, opts) do
    case opts do
      [account_id | _rest] ->
        get_account_by_id(account_id)

      [] ->
        IO.puts(:stderr, "❌ Error: account ID required")
        IO.puts("Usage: xero accounts get <account_id>")
        System.halt(1)
    end
  end

  defp get_account_by_id(account_id) do
    case get_valid_credentials() do
      {:ok, credentials} ->
        access_token = Map.get(credentials, "access_token")
        tenant_id = Map.get(credentials, "tenant_id")

        url = "#{@xero_api_base}/Accounts/#{account_id}"

        IO.puts("💰 Fetching account #{account_id}...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                accounts = get_in(data, ["Accounts"]) || []

                case accounts do
                  [account | _] -> display_account_detail(account)
                  [] -> IO.puts("Account not found.")
                end

              {:error, reason} ->
                IO.puts(:stderr, "❌ Failed to parse response: #{inspect(reason)}")
                System.halt(1)
            end

          {:ok, %{status_code: 404}} ->
            IO.puts(:stderr, "❌ Account not found.")
            System.halt(1)

          {:ok, %{status_code: 401}} ->
            IO.puts(:stderr, "❌ Authentication failed. Please run 'xero auth login' again.")
            System.halt(1)

          {:ok, %{status_code: status, body: body}} ->
            IO.puts(:stderr, "❌ API request failed with status #{status}")
            IO.puts(:stderr, "Response: #{body}")
            System.halt(1)

          {:error, reason} ->
            IO.puts(:stderr, "❌ Request failed: #{inspect(reason)}")
            System.halt(1)
        end

      {:error, reason} ->
        IO.puts(:stderr, "❌ #{reason}")
        IO.puts("Please run 'xero auth login' first.")
        System.halt(1)
    end
  end

  defp get_valid_credentials do
    case XeroCLI.Config.get_credentials() do
      {:ok, credentials} ->
        # Check if token is expired
        if XeroCLI.OAuth.token_expired?(credentials) do
          IO.puts("🔄 Access token expired. Refreshing...")

          refresh_token = Map.get(credentials, "refresh_token")
          client_id = Map.get(credentials, "client_id")

          case XeroCLI.OAuth.refresh_token(refresh_token, client_id) do
            {:ok, new_token_data} ->
              # Update credentials with new token
              updated_credentials =
                credentials
                |> Map.put("access_token", Map.get(new_token_data, "access_token"))
                |> Map.put("refresh_token", Map.get(new_token_data, "refresh_token"))
                |> Map.put("expires_in", Map.get(new_token_data, "expires_in", 1800))
                |> Map.put("obtained_at", Map.get(new_token_data, "obtained_at"))

              case XeroCLI.Config.set_credentials(updated_credentials) do
                :ok ->
                  IO.puts("✓ Token refreshed\n")
                  {:ok, updated_credentials}

                {:error, reason} ->
                  {:error, "Failed to save refreshed credentials: #{reason}"}
              end

            {:error, reason} ->
              {:error, "Failed to refresh token: #{reason}"}
          end
        else
          {:ok, credentials}
        end

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp get_option(opts, flag, default) do
    case Enum.find_index(opts, &(&1 == flag)) do
      nil -> default
      index -> Enum.at(opts, index + 1, default)
    end
  end

  defp maybe_add_param(params, _key, nil), do: params

  defp maybe_add_param(params, key, value) do
    [{key, value} | params]
  end

  defp display_accounts([]) do
    IO.puts("No accounts found.")
  end

  defp display_accounts(accounts) do
    IO.puts("Found #{length(accounts)} account(s):\n")
    IO.puts(String.duplicate("=", 120))

    header =
      format_row([
        pad("Code", 12),
        pad("Name", 35),
        pad("Type", 20),
        pad("Account ID", 38),
        pad("Status", 12)
      ])

    IO.puts(header)
    IO.puts(String.duplicate("=", 120))

    Enum.each(accounts, fn account ->
      code = Map.get(account, "Code", "N/A")
      name = Map.get(account, "Name", "N/A")
      type = Map.get(account, "Type", "N/A")
      account_id = Map.get(account, "AccountID", "N/A")
      status = Map.get(account, "Status", "N/A")

      row =
        format_row([
          pad(code, 12),
          pad(name, 35),
          pad(type, 20),
          pad(account_id, 38),
          pad(status, 12)
        ])

      IO.puts(row)
    end)

    IO.puts(String.duplicate("=", 120))
  end

  defp display_account_detail(account) do
    IO.puts("Account Details:\n")
    IO.puts(String.duplicate("=", 80))

    code = Map.get(account, "Code", "N/A")
    name = Map.get(account, "Name", "N/A")
    account_id = Map.get(account, "AccountID", "N/A")
    type = Map.get(account, "Type", "N/A")
    status = Map.get(account, "Status", "N/A")
    description = Map.get(account, "Description", "N/A")
    tax_type = Map.get(account, "TaxType", "N/A")
    class = Map.get(account, "Class", "N/A")
    enable_payments = Map.get(account, "EnablePaymentsToAccount", false)
    show_in_expense_claims = Map.get(account, "ShowInExpenseClaims", false)

    IO.puts("Account Code:           #{code}")
    IO.puts("Name:                   #{name}")
    IO.puts("Account ID:             #{account_id}")
    IO.puts("Type:                   #{type}")
    IO.puts("Status:                 #{status}")
    IO.puts("Description:            #{description}")
    IO.puts("Tax Type:               #{tax_type}")
    IO.puts("Class:                  #{class}")
    IO.puts("Enable Payments:        #{enable_payments}")
    IO.puts("Show in Expense Claims: #{show_in_expense_claims}")

    IO.puts(String.duplicate("=", 80))
  end

  defp format_row(columns) do
    Enum.join(columns, " | ")
  end

  defp pad(str, width) do
    str = to_string(str)

    cond do
      String.length(str) > width ->
        String.slice(str, 0, width - 3) <> "..."

      String.length(str) < width ->
        String.pad_trailing(str, width)

      true ->
        str
    end
  end
end
