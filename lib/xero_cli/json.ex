defmodule Jason do
  @moduledoc """
  Wrapper around Elixir 1.18's built-in :json module.
  Provides a compatible API for JSON encoding/decoding.

  **Requirements:**
  - Elixir 1.18+ (which includes OTP 27+)
  - The :json module is built into OTP 27 and later

  This module provides a thin wrapper around the built-in :json module
  to provide a familiar API similar to the Jason library, while using
  Elixir's native JSON support with zero external dependencies.
  """

  @doc """
  Encodes an Elixir term to JSON.

  ## Options
  - `:pretty` - when true, formats the JSON with indentation (default: false)

  ## Examples

      iex> Jason.encode(%{"key" => "value"})
      {:ok, "{\\"key\\":\\"value\\"}"}

  """
  def encode(data, opts \\ []) do
    try do
      json =
        if Keyword.get(opts, :pretty, false) do
          # Pretty print with indentation
          :json.encode(convert_to_json_term(data))
          |> prettify()
        else
          :json.encode(convert_to_json_term(data))
        end

      {:ok, json}
    rescue
      e -> {:error, "Encoding error: #{inspect(e)}"}
    end
  end

  @doc """
  Decodes a JSON string to an Elixir term.

  ## Examples

      iex> Jason.decode("{\\"key\\":\\"value\\"}")
      {:ok, %{"key" => "value"}}

  """
  def decode(json) when is_binary(json) do
    try do
      result = :json.decode(json)
      {:ok, convert_from_json_term(result)}
    rescue
      e -> {:error, "Decoding error: #{inspect(e)}"}
    end
  end

  # Convert Elixir terms to JSON-compatible terms
  defp convert_to_json_term(map) when is_map(map) do
    Map.new(map, fn {k, v} -> {to_string(k), convert_to_json_term(v)} end)
  end

  defp convert_to_json_term(list) when is_list(list) do
    Enum.map(list, &convert_to_json_term/1)
  end

  defp convert_to_json_term(term), do: term

  # Convert JSON terms to Elixir terms (maps with string keys)
  defp convert_from_json_term(map) when is_map(map) do
    Map.new(map, fn {k, v} -> {k, convert_from_json_term(v)} end)
  end

  defp convert_from_json_term(list) when is_list(list) do
    Enum.map(list, &convert_from_json_term/1)
  end

  defp convert_from_json_term(term), do: term

  # Simple pretty-printing by adding newlines and indentation
  defp prettify(json) do
    json
    |> String.replace("{", "{\n  ")
    |> String.replace("}", "\n}")
    |> String.replace(",", ",\n  ")
    |> String.replace("[", "[\n  ")
    |> String.replace("]", "\n]")
  end
end
