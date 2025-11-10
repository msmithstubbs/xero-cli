defmodule XeroCLI.OAuth do
  @moduledoc """
  Handles OAuth 2.0 authentication flow for Xero API.
  Uses PKCE (Proof Key for Code Exchange) for security.
  """

  @xero_auth_url "https://login.xero.com/identity/connect/authorize"
  @xero_token_url "https://identity.xero.com/connect/token"
  @xero_connections_url "https://api.xero.com/connections"
  @redirect_uri "http://localhost:8888/callback"

  def get_auth_url(client_id) do
    # Generate PKCE code verifier and challenge
    code_verifier = generate_code_verifier()
    code_challenge = generate_code_challenge(code_verifier)

    # Store code verifier for later use
    XeroCLI.Config.set_setting("pkce_verifier", code_verifier)

    params = %{
      "response_type" => "code",
      "client_id" => client_id,
      "redirect_uri" => @redirect_uri,
      "scope" =>
        "offline_access openid profile email accounting.transactions accounting.contacts accounting.settings",
      "code_challenge" => code_challenge,
      "code_challenge_method" => "S256"
    }

    query_string = URI.encode_query(params)
    "#{@xero_auth_url}?#{query_string}"
  end

  def exchange_code(code, client_id) do
    # Retrieve stored code verifier
    code_verifier = XeroCLI.Config.get_setting("pkce_verifier")

    if code_verifier == nil do
      {:error, "Code verifier not found. Please restart the authentication process."}
    else
      params = %{
        "grant_type" => "authorization_code",
        "client_id" => client_id,
        "code" => code,
        "redirect_uri" => @redirect_uri,
        "code_verifier" => code_verifier
      }

      body = URI.encode_query(params)

      headers = [
        {"content-type", "application/x-www-form-urlencoded"}
      ]

      case XeroCLI.HTTP.post(@xero_token_url, body, headers) do
        {:ok, %{status_code: 200, body: response_body}} ->
          case Jason.decode(response_body) do
            {:ok, token_data} ->
              # Clean up the code verifier
              XeroCLI.Config.set_setting("pkce_verifier", nil)

              # Add timestamp for token expiration tracking
              token_data_with_timestamp =
                Map.put(token_data, "obtained_at", System.system_time(:second))

              {:ok, token_data_with_timestamp}

            {:error, reason} ->
              {:error, "Failed to parse token response: #{inspect(reason)}"}
          end

        {:ok, %{status_code: status, body: body}} ->
          {:error, "Token exchange failed with status #{status}: #{body}"}

        {:error, reason} ->
          {:error, "Token exchange request failed: #{inspect(reason)}"}
      end
    end
  end

  def refresh_token(refresh_token, client_id) do
    params = %{
      "grant_type" => "refresh_token",
      "refresh_token" => refresh_token,
      "client_id" => client_id
    }

    body = URI.encode_query(params)

    headers = [
      {"content-type", "application/x-www-form-urlencoded"}
    ]

    case XeroCLI.HTTP.post(@xero_token_url, body, headers) do
      {:ok, %{status_code: 200, body: response_body}} ->
        case Jason.decode(response_body) do
          {:ok, token_data} ->
            token_data_with_timestamp =
              Map.put(token_data, "obtained_at", System.system_time(:second))

            {:ok, token_data_with_timestamp}

          {:error, reason} ->
            {:error, "Failed to parse token response: #{inspect(reason)}"}
        end

      {:ok, %{status_code: status, body: body}} ->
        {:error, "Token refresh failed with status #{status}: #{body}"}

      {:error, reason} ->
        {:error, "Token refresh request failed: #{inspect(reason)}"}
    end
  end

  def get_connections(access_token) do
    headers = [
      {"authorization", "Bearer #{access_token}"},
      {"content-type", "application/json"}
    ]

    case XeroCLI.HTTP.get(@xero_connections_url, headers) do
      {:ok, %{status_code: 200, body: response_body}} ->
        case Jason.decode(response_body) do
          {:ok, connections} -> {:ok, connections}
          {:error, reason} -> {:error, "Failed to parse connections: #{inspect(reason)}"}
        end

      {:ok, %{status_code: status, body: body}} ->
        {:error, "Failed to get connections with status #{status}: #{body}"}

      {:error, reason} ->
        {:error, "Connections request failed: #{inspect(reason)}"}
    end
  end

  def token_expired?(credentials) do
    case Map.get(credentials, "obtained_at") do
      nil ->
        true

      obtained_at ->
        expires_in = Map.get(credentials, "expires_in", 1800)
        current_time = System.system_time(:second)
        # Consider token expired if less than 5 minutes remaining
        current_time >= obtained_at + expires_in - 300
    end
  end

  # Generate a random code verifier for PKCE
  defp generate_code_verifier do
    :crypto.strong_rand_bytes(32)
    |> Base.url_encode64(padding: false)
  end

  # Generate code challenge from verifier using SHA256
  defp generate_code_challenge(verifier) do
    :crypto.hash(:sha256, verifier)
    |> Base.url_encode64(padding: false)
  end

  def start_callback_server do
    {:ok, listen_socket} =
      :gen_tcp.listen(8888, [
        :binary,
        packet: :raw,
        active: false,
        reuseaddr: true
      ])

    IO.puts("\n📡 Waiting for OAuth callback on http://localhost:8888/callback...")
    accept_connection(listen_socket)
  end

  defp accept_connection(listen_socket) do
    {:ok, socket} = :gen_tcp.accept(listen_socket)

    case :gen_tcp.recv(socket, 0, 30_000) do
      {:ok, data} ->
        request = to_string(data)
        code = extract_code_from_request(request)

        if code do
          response = """
          HTTP/1.1 200 OK\r
          Content-Type: text/html\r
          Connection: close\r
          \r
          <html>
          <head><title>Xero CLI - Authentication Successful</title></head>
          <body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
            <h1 style="color: #13B5EA;">✓ Authentication Successful!</h1>
            <p>You can close this window and return to your terminal.</p>
          </body>
          </html>
          """

          :gen_tcp.send(socket, response)
          :gen_tcp.close(socket)
          :gen_tcp.close(listen_socket)
          {:ok, code}
        else
          error_response = """
          HTTP/1.1 400 Bad Request\r
          Content-Type: text/html\r
          Connection: close\r
          \r
          <html>
          <head><title>Xero CLI - Authentication Failed</title></head>
          <body style="font-family: Arial, sans-serif; text-align: center; padding: 50px;">
            <h1 style="color: #ff0000;">✗ Authentication Failed</h1>
            <p>No authorization code found. Please try again.</p>
          </body>
          </html>
          """

          :gen_tcp.send(socket, error_response)
          :gen_tcp.close(socket)
          :gen_tcp.close(listen_socket)
          {:error, "No authorization code found"}
        end

      {:error, reason} ->
        :gen_tcp.close(socket)
        :gen_tcp.close(listen_socket)
        {:error, "Failed to receive callback: #{inspect(reason)}"}
    end
  end

  defp extract_code_from_request(request) do
    case Regex.run(~r/GET \/callback\?code=([^&\s]+)/, request) do
      [_, code] -> URI.decode(code)
      _ -> nil
    end
  end
end
