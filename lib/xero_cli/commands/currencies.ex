defmodule XeroCLI.Commands.Currencies do
  @moduledoc """
  Handles currency-related commands for Xero CLI.
  """

  @xero_api_base "https://api.xero.com/api.xro/2.0"

  def handle(:list, opts) do
    case get_valid_credentials() do
      {:ok, credentials} ->
        access_token = Map.get(credentials, "access_token")
        tenant_id = Map.get(credentials, "tenant_id")

        # Parse options - currencies don't typically support pagination in Xero API
        # but we can filter by where clause if needed
        where = get_option(opts, "--where", nil)

        # Build query parameters
        query_params =
          []
          |> maybe_add_param("where", where)
          |> URI.encode_query()

        url =
          if query_params != "" do
            "#{@xero_api_base}/Currencies?#{query_params}"
          else
            "#{@xero_api_base}/Currencies"
          end

        IO.puts("💱 Fetching currencies...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                currencies = get_in(data, ["Currencies"]) || []
                display_currencies(currencies)

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
      [currency_code | _rest] ->
        get_currency_by_code(currency_code)

      [] ->
        IO.puts(:stderr, "❌ Error: currency code required")
        IO.puts("Usage: xero currencies get <currency_code>")
        System.halt(1)
    end
  end

  defp get_currency_by_code(currency_code) do
    case get_valid_credentials() do
      {:ok, credentials} ->
        access_token = Map.get(credentials, "access_token")
        tenant_id = Map.get(credentials, "tenant_id")

        url = "#{@xero_api_base}/Currencies/#{currency_code}"

        IO.puts("💱 Fetching currency #{currency_code}...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                currencies = get_in(data, ["Currencies"]) || []

                case currencies do
                  [currency | _] -> display_currency_detail(currency)
                  [] -> IO.puts("Currency not found.")
                end

              {:error, reason} ->
                IO.puts(:stderr, "❌ Failed to parse response: #{inspect(reason)}")
                System.halt(1)
            end

          {:ok, %{status_code: 404}} ->
            IO.puts(:stderr, "❌ Currency not found.")
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

  defp display_currencies([]) do
    IO.puts("No currencies found.")
  end

  defp display_currencies(currencies) do
    IO.puts("Found #{length(currencies)} currency/currencies:\n")
    IO.puts(String.duplicate("=", 80))

    header =
      format_row([
        pad("Code", 10),
        pad("Description", 50),
        pad("Status", 15)
      ])

    IO.puts(header)
    IO.puts(String.duplicate("=", 80))

    Enum.each(currencies, fn currency ->
      code = Map.get(currency, "Code", "N/A")
      description = Map.get(currency, "Description", "N/A")

      # Status is not always present in currency responses
      # but we can infer from other fields if available
      status =
        cond do
          Map.has_key?(currency, "Status") -> Map.get(currency, "Status")
          true -> "ACTIVE"
        end

      row =
        format_row([
          pad(code, 10),
          pad(description, 50),
          pad(status, 15)
        ])

      IO.puts(row)
    end)

    IO.puts(String.duplicate("=", 80))
  end

  defp display_currency_detail(currency) do
    IO.puts("Currency Details:\n")
    IO.puts(String.duplicate("=", 80))

    code = Map.get(currency, "Code", "N/A")
    description = Map.get(currency, "Description", "N/A")

    IO.puts("Code:        #{code}")
    IO.puts("Description: #{description}")

    # Display additional fields if available
    if Map.has_key?(currency, "Status") do
      status = Map.get(currency, "Status")
      IO.puts("Status:      #{status}")
    end

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
