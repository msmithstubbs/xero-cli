defmodule XeroCLI.HTTP do
  @moduledoc """
  HTTP client wrapper using Erlang's built-in :httpc module.
  """

  def start do
    :inets.start()
    :ssl.start()
  end

  def get(url, headers \\ []) do
    request(:get, url, headers, "")
  end

  def post(url, body, headers \\ []) do
    request(:post, url, headers, body)
  end

  def put(url, body, headers \\ []) do
    request(:put, url, headers, body)
  end

  def delete(url, headers \\ []) do
    request(:delete, url, headers, "")
  end

  defp request(method, url, headers, body) do
    start()

    url_charlist = String.to_charlist(url)
    headers_list = Enum.map(headers, fn {k, v} -> {String.to_charlist(k), String.to_charlist(v)} end)

    request =
      if body != "" do
        content_type =
          case List.keyfind(headers_list, ~c"content-type", 0) do
            {_, ct} -> ct
            nil -> ~c"application/json"
          end

        {url_charlist, headers_list, content_type, body}
      else
        {url_charlist, headers_list}
      end

    http_options = [
      ssl: [
        verify: :verify_peer,
        cacerts: :public_key.cacerts_get(),
        customize_hostname_check: [
          match_fun: :public_key.pkix_verify_hostname_match_fun(:https)
        ]
      ]
    ]

    options = [body_format: :binary]

    case :httpc.request(method, request, http_options, options) do
      {:ok, {{_version, status_code, _reason}, response_headers, response_body}} ->
        headers_map =
          Enum.into(response_headers, %{}, fn {k, v} ->
            {to_string(k), to_string(v)}
          end)

        {:ok,
         %{
           status_code: status_code,
           headers: headers_map,
           body: to_string(response_body)
         }}

      {:error, reason} ->
        {:error, "HTTP request failed: #{inspect(reason)}"}
    end
  end

  def parse_response({:ok, %{status_code: code, body: body}}) when code in 200..299 do
    case Jason.decode(body) do
      {:ok, data} -> {:ok, data}
      {:error, _} -> {:ok, body}
    end
  end

  def parse_response({:ok, %{status_code: code, body: body}}) do
    case Jason.decode(body) do
      {:ok, data} -> {:error, {code, data}}
      {:error, _} -> {:error, {code, body}}
    end
  end

  def parse_response({:error, reason}) do
    {:error, reason}
  end
end
