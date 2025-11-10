defmodule XeroCLI.Commands.Auth do
  @moduledoc """
  Handles authentication commands for Xero CLI.
  """

  def handle(:login, _opts) do
    IO.puts("🔐 Xero CLI - OAuth 2.0 Authentication")
    IO.puts("")

    # Prompt for client ID
    client_id = get_client_id()

    if client_id == nil do
      IO.puts(:stderr, "\n❌ Error: Client ID is required")
      IO.puts("Please provide your Xero OAuth 2.0 Client ID.")
      IO.puts("You can create one at: https://developer.xero.com/app/manage")
      System.halt(1)
    end

    # Generate auth URL
    auth_url = XeroCLI.OAuth.get_auth_url(client_id)

    IO.puts("Please visit the following URL to authorize this application:\n")
    IO.puts("  #{auth_url}\n")

    # Try to open the browser automatically
    case open_browser(auth_url) do
      :ok -> IO.puts("✓ Browser opened automatically")
      {:error, _} -> IO.puts("⚠ Please open the URL manually in your browser")
    end

    # Start callback server and wait for the authorization code
    case XeroCLI.OAuth.start_callback_server() do
      {:ok, code} ->
        IO.puts("\n✓ Authorization code received")
        IO.puts("Exchanging code for access token...")

        case XeroCLI.OAuth.exchange_code(code, client_id) do
          {:ok, token_data} ->
            # Get tenant/organization connections
            access_token = Map.get(token_data, "access_token")

            IO.puts("✓ Access token obtained")
            IO.puts("Fetching Xero organizations...")

            case XeroCLI.OAuth.get_connections(access_token) do
              {:ok, connections} when is_list(connections) and length(connections) > 0 ->
                # Store the first tenant ID
                tenant = List.first(connections)
                tenant_id = Map.get(tenant, "tenantId")
                tenant_name = Map.get(tenant, "tenantName", "Unknown")

                credentials = %{
                  "client_id" => client_id,
                  "access_token" => access_token,
                  "refresh_token" => Map.get(token_data, "refresh_token"),
                  "tenant_id" => tenant_id,
                  "tenant_name" => tenant_name,
                  "expires_in" => Map.get(token_data, "expires_in", 1800),
                  "obtained_at" => Map.get(token_data, "obtained_at")
                }

                case XeroCLI.Config.set_credentials(credentials) do
                  :ok ->
                    IO.puts("\n✅ Successfully authenticated with Xero!")
                    IO.puts("Organization: #{tenant_name}")
                    IO.puts("Tenant ID: #{tenant_id}")
                    IO.puts("\nYou can now use the Xero CLI.")

                  {:error, reason} ->
                    IO.puts(:stderr, "\n❌ Failed to save credentials: #{reason}")
                    System.halt(1)
                end

              {:ok, []} ->
                IO.puts(:stderr, "\n❌ No Xero organizations found for this account")
                System.halt(1)

              {:error, reason} ->
                IO.puts(:stderr, "\n❌ Failed to fetch organizations: #{reason}")
                System.halt(1)
            end

          {:error, reason} ->
            IO.puts(:stderr, "\n❌ Failed to exchange authorization code: #{reason}")
            System.halt(1)
        end

      {:error, reason} ->
        IO.puts(:stderr, "\n❌ Authentication failed: #{reason}")
        System.halt(1)
    end
  end

  def handle(:logout, _opts) do
    case XeroCLI.Config.delete_credentials() do
      :ok ->
        IO.puts("✅ Successfully logged out. Credentials removed.")

      {:error, reason} ->
        IO.puts(:stderr, "❌ Failed to logout: #{reason}")
        System.halt(1)
    end
  end

  def handle(:status, _opts) do
    case XeroCLI.Config.get_credentials() do
      {:ok, credentials} ->
        tenant_name = Map.get(credentials, "tenant_name", "Unknown")
        tenant_id = Map.get(credentials, "tenant_id", "Unknown")

        IO.puts("✅ Authenticated")
        IO.puts("Organization: #{tenant_name}")
        IO.puts("Tenant ID: #{tenant_id}")

        if XeroCLI.OAuth.token_expired?(credentials) do
          IO.puts("⚠ Access token expired. It will be refreshed on next API call.")
        else
          IO.puts("✓ Access token is valid")
        end

      {:error, reason} ->
        IO.puts("❌ Not authenticated")
        IO.puts("Reason: #{reason}")
        System.halt(1)
    end
  end

  defp get_client_id do
    # First, check if it's set in environment variable
    case System.get_env("XERO_CLIENT_ID") do
      nil ->
        # Check if it's stored in config
        case XeroCLI.Config.get_setting("client_id") do
          nil ->
            # Prompt user to enter it
            IO.write("Enter your Xero Client ID: ")
            client_id = IO.gets("") |> String.trim()

            if client_id != "" do
              # Optionally save it for future use
              IO.write("Save Client ID for future use? (y/n): ")
              response = IO.gets("") |> String.trim() |> String.downcase()

              if response == "y" do
                XeroCLI.Config.set_setting("client_id", client_id)
              end

              client_id
            else
              nil
            end

          stored_id ->
            IO.puts("Using saved Client ID: #{String.slice(stored_id, 0..7)}...")
            stored_id
        end

      env_id ->
        IO.puts("Using Client ID from XERO_CLIENT_ID environment variable")
        env_id
    end
  end

  defp open_browser(url) do
    case :os.type() do
      {:unix, :darwin} -> System.cmd("open", [url])
      {:unix, _} -> System.cmd("xdg-open", [url])
      {:win32, _} -> System.cmd("cmd", ["/c", "start", url])
      _ -> {:error, "Unknown OS"}
    end

    :ok
  rescue
    _ -> {:error, "Failed to open browser"}
  end
end
