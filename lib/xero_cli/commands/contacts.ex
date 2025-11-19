defmodule XeroCLI.Commands.Contacts do
  @moduledoc """
  Handles contact-related commands for Xero CLI.
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

        url = "#{@xero_api_base}/Contacts?#{query_params}"

        IO.puts("👥 Fetching contacts...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                contacts = get_in(data, ["Contacts"]) || []
                display_contacts(contacts)

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
      [contact_id | _rest] ->
        get_contact_by_id(contact_id)

      [] ->
        IO.puts(:stderr, "❌ Error: contact ID required")
        IO.puts("Usage: xero contacts get <contact_id>")
        System.halt(1)
    end
  end

  defp get_contact_by_id(contact_id) do
    case get_valid_credentials() do
      {:ok, credentials} ->
        access_token = Map.get(credentials, "access_token")
        tenant_id = Map.get(credentials, "tenant_id")

        url = "#{@xero_api_base}/Contacts/#{contact_id}"

        IO.puts("👥 Fetching contact #{contact_id}...\n")

        headers = [
          {"authorization", "Bearer #{access_token}"},
          {"xero-tenant-id", tenant_id},
          {"accept", "application/json"}
        ]

        case XeroCLI.HTTP.get(url, headers) do
          {:ok, %{status_code: 200, body: response_body}} ->
            case Jason.decode(response_body) do
              {:ok, data} ->
                contacts = get_in(data, ["Contacts"]) || []

                case contacts do
                  [contact | _] -> display_contact_detail(contact)
                  [] -> IO.puts("Contact not found.")
                end

              {:error, reason} ->
                IO.puts(:stderr, "❌ Failed to parse response: #{inspect(reason)}")
                System.halt(1)
            end

          {:ok, %{status_code: 404}} ->
            IO.puts(:stderr, "❌ Contact not found.")
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

  defp display_contacts([]) do
    IO.puts("No contacts found.")
  end

  defp display_contacts(contacts) do
    IO.puts("Found #{length(contacts)} contact(s):\n")
    IO.puts(String.duplicate("=", 120))

    header =
      format_row([
        pad("Name", 30),
        pad("Email", 35),
        pad("Contact ID", 38),
        pad("Status", 15)
      ])

    IO.puts(header)
    IO.puts(String.duplicate("=", 120))

    Enum.each(contacts, fn contact ->
      name = Map.get(contact, "Name", "N/A")
      email = Map.get(contact, "EmailAddress", "N/A")
      contact_id = Map.get(contact, "ContactID", "N/A")
      status = Map.get(contact, "ContactStatus", "N/A")

      row =
        format_row([
          pad(name, 30),
          pad(email, 35),
          pad(contact_id, 38),
          pad(status, 15)
        ])

      IO.puts(row)
    end)

    IO.puts(String.duplicate("=", 120))
  end

  defp display_contact_detail(contact) do
    IO.puts("Contact Details:\n")
    IO.puts(String.duplicate("=", 80))

    name = Map.get(contact, "Name", "N/A")
    contact_id = Map.get(contact, "ContactID", "N/A")
    email = Map.get(contact, "EmailAddress", "N/A")
    status = Map.get(contact, "ContactStatus", "N/A")
    first_name = Map.get(contact, "FirstName", "N/A")
    last_name = Map.get(contact, "LastName", "N/A")

    IO.puts("Name:             #{name}")
    IO.puts("First Name:       #{first_name}")
    IO.puts("Last Name:        #{last_name}")
    IO.puts("Contact ID:       #{contact_id}")
    IO.puts("Email:            #{email}")
    IO.puts("Status:           #{status}")

    # Display addresses if available
    addresses = Map.get(contact, "Addresses", [])

    if length(addresses) > 0 do
      IO.puts("\nAddresses:")

      Enum.each(addresses, fn address ->
        type = Map.get(address, "AddressType", "N/A")
        line1 = Map.get(address, "AddressLine1", "")
        line2 = Map.get(address, "AddressLine2", "")
        city = Map.get(address, "City", "")
        region = Map.get(address, "Region", "")
        postal_code = Map.get(address, "PostalCode", "")
        country = Map.get(address, "Country", "")

        IO.puts("  #{type}:")
        if line1 != "", do: IO.puts("    #{line1}")
        if line2 != "", do: IO.puts("    #{line2}")

        location_parts =
          [city, region, postal_code, country]
          |> Enum.reject(&(&1 == ""))

        if length(location_parts) > 0 do
          IO.puts("    #{Enum.join(location_parts, ", ")}")
        end
      end)
    end

    # Display phones if available
    phones = Map.get(contact, "Phones", [])

    if length(phones) > 0 do
      IO.puts("\nPhones:")

      Enum.each(phones, fn phone ->
        type = Map.get(phone, "PhoneType", "N/A")
        number = Map.get(phone, "PhoneNumber", "N/A")
        IO.puts("  #{type}: #{number}")
      end)
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
