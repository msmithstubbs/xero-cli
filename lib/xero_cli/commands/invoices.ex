defmodule XeroCLI.Commands.Invoices do
  @moduledoc """
  Handles invoice-related commands for Xero CLI.
  """

  @xero_api_base "https://api.xero.com/api.xro/2.0"

  def handle(:list, opts) do
    # Get credentials and ensure token is valid
    case get_valid_credentials() do
      {:ok, credentials} ->
        access_token = Map.get(credentials, "access_token")
        tenant_id = Map.get(credentials, "tenant_id")

        # Parse options
        page = get_option(opts, "--page", "1") |> String.to_integer()
        page_size = get_option(opts, "--page-size", "100") |> String.to_integer()
        status = get_option(opts, "--status", nil)

        # Build query parameters
        query_params =
          []
          |> maybe_add_param("page", page)
          |> maybe_add_param("pageSize", page_size)
          |> maybe_add_param("where", build_where_clause(status))
          |> URI.encode_query()

        url = "#{@xero_api_base}/Invoices?#{query_params}"

        IO.puts("📄 Fetching invoices...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                invoices = get_in(data, ["Invoices"]) || []
                display_invoices(invoices)

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

  defp build_where_clause(nil), do: nil

  defp build_where_clause(status) do
    "Status==\"#{String.upcase(status)}\""
  end

  defp display_invoices([]) do
    IO.puts("No invoices found.")
  end

  defp display_invoices(invoices) do
    IO.puts("Found #{length(invoices)} invoice(s):\n")
    IO.puts(String.duplicate("=", 120))

    header =
      format_row([
        pad("Invoice Number", 20),
        pad("Type", 10),
        pad("Contact", 25),
        pad("Date", 12),
        pad("Due Date", 12),
        pad("Status", 12),
        pad("Total", 15)
      ])

    IO.puts(header)
    IO.puts(String.duplicate("=", 120))

    Enum.each(invoices, fn invoice ->
      number = Map.get(invoice, "InvoiceNumber", "N/A")
      type = Map.get(invoice, "Type", "N/A")
      contact_name = get_in(invoice, ["Contact", "Name"]) || "N/A"
      date = Map.get(invoice, "Date", "N/A") |> format_date()
      due_date = Map.get(invoice, "DueDate", "N/A") |> format_date()
      status = Map.get(invoice, "Status", "N/A")
      total = Map.get(invoice, "Total", 0) |> format_currency()

      row =
        format_row([
          pad(number, 20),
          pad(type, 10),
          pad(contact_name, 25),
          pad(date, 12),
          pad(due_date, 12),
          pad(status_with_emoji(status), 12),
          pad(total, 15)
        ])

      IO.puts(row)
    end)

    IO.puts(String.duplicate("=", 120))
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

  defp format_date(date_string) when is_binary(date_string) do
    # Xero returns dates in format like "/Date(1234567890000)/"
    case Regex.run(~r/\/Date\((\d+)\)/, date_string) do
      [_, timestamp] ->
        timestamp
        |> String.to_integer()
        |> div(1000)
        |> DateTime.from_unix!()
        |> Calendar.strftime("%Y-%m-%d")

      _ ->
        date_string
    end
  end

  defp format_date(_), do: "N/A"

  defp format_currency(amount) when is_number(amount) do
    "$#{:erlang.float_to_binary(amount / 1, decimals: 2)}"
  end

  defp format_currency(_), do: "$0.00"

  defp status_with_emoji(status) do
    case String.upcase(to_string(status)) do
      "PAID" -> "✓ PAID"
      "AUTHORISED" -> "○ AUTH"
      "DRAFT" -> "◐ DRAFT"
      "VOIDED" -> "✗ VOID"
      "DELETED" -> "✗ DEL"
      other -> other
    end
  end
end
