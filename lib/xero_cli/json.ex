defmodule Jason do
  @moduledoc """
  Simple JSON encoder/decoder using Erlang's :json module (OTP 27+)
  or a basic implementation for older versions.
  """

  def encode(data, _opts \\ []) do
    try do
      # Try using Erlang's built-in JSON encoder (OTP 27+)
      {:ok, :json.encode(convert_to_erlang_term(data))}
    rescue
      UndefinedFunctionError ->
        # Fallback to manual encoding
        {:ok, manual_encode(data)}
    end
  end

  def decode(json) when is_binary(json) do
    try do
      # Try using Erlang's built-in JSON decoder (OTP 27+)
      {:ok, convert_from_erlang_term(:json.decode(json))}
    rescue
      UndefinedFunctionError ->
        # Fallback to manual decoding
        manual_decode(json)
    end
  end

  # Convert Elixir terms to Erlang terms for :json module
  defp convert_to_erlang_term(map) when is_map(map) do
    Map.new(map, fn {k, v} -> {to_string(k), convert_to_erlang_term(v)} end)
  end

  defp convert_to_erlang_term(list) when is_list(list) do
    Enum.map(list, &convert_to_erlang_term/1)
  end

  defp convert_to_erlang_term(term), do: term

  # Convert Erlang terms back to Elixir terms
  defp convert_from_erlang_term(map) when is_map(map) do
    Map.new(map, fn {k, v} -> {k, convert_from_erlang_term(v)} end)
  end

  defp convert_from_erlang_term(list) when is_list(list) do
    Enum.map(list, &convert_from_erlang_term/1)
  end

  defp convert_from_erlang_term(term), do: term

  # Manual encoding fallback
  defp manual_encode(nil), do: "null"
  defp manual_encode(true), do: "true"
  defp manual_encode(false), do: "false"
  defp manual_encode(atom) when is_atom(atom), do: "\"#{atom}\""
  defp manual_encode(num) when is_number(num), do: to_string(num)

  defp manual_encode(str) when is_binary(str) do
    escaped =
      str
      |> String.replace("\\", "\\\\")
      |> String.replace("\"", "\\\"")
      |> String.replace("\n", "\\n")
      |> String.replace("\r", "\\r")
      |> String.replace("\t", "\\t")

    "\"#{escaped}\""
  end

  defp manual_encode(list) when is_list(list) do
    items = Enum.map(list, &manual_encode/1)
    "[#{Enum.join(items, ",")}]"
  end

  defp manual_encode(map) when is_map(map) do
    pairs =
      map
      |> Enum.map(fn {k, v} -> "#{manual_encode(to_string(k))}:#{manual_encode(v)}" end)

    "{#{Enum.join(pairs, ",")}}"
  end

  # Manual decoding fallback (basic implementation)
  defp manual_decode(json) do
    try do
      {result, _rest} = do_parse(String.trim(json))
      {:ok, result}
    rescue
      e -> {:error, "Parse error: #{inspect(e)}"}
    end
  end

  defp do_parse("null" <> rest), do: {nil, rest}
  defp do_parse("true" <> rest), do: {true, rest}
  defp do_parse("false" <> rest), do: {false, rest}

  defp do_parse("{" <> rest) do
    parse_object(rest, %{})
  end

  defp do_parse("[" <> rest) do
    parse_array(rest, [])
  end

  defp do_parse("\"" <> rest) do
    parse_string(rest, "")
  end

  defp do_parse(json) do
    # Try to parse as number
    case Float.parse(json) do
      {num, rest} -> {num, rest}
      :error -> raise "Invalid JSON"
    end
  end

  defp parse_string("\"" <> rest, acc), do: {acc, rest}

  defp parse_string("\\" <> <<char, rest::binary>>, acc) do
    char =
      case char do
        ?n -> "\n"
        ?r -> "\r"
        ?t -> "\t"
        ?" -> "\""
        ?\\ -> "\\"
        _ -> <<char>>
      end

    parse_string(rest, acc <> char)
  end

  defp parse_string(<<char, rest::binary>>, acc) do
    parse_string(rest, acc <> <<char>>)
  end

  defp parse_object(json, acc) do
    json = String.trim_leading(json)

    case json do
      "}" <> rest ->
        {acc, rest}

      "\"" <> _ ->
        {key, rest} = parse_string(String.trim_leading(json, "\""), "")
        rest = String.trim_leading(rest)
        ":" <> rest = rest
        rest = String.trim_leading(rest)
        {value, rest} = parse_value(rest)
        rest = String.trim_leading(rest)

        case rest do
          "," <> rest ->
            parse_object(String.trim_leading(rest), Map.put(acc, key, value))

          "}" <> rest ->
            {Map.put(acc, key, value), rest}
        end
    end
  end

  defp parse_array(json, acc) do
    json = String.trim_leading(json)

    case json do
      "]" <> rest ->
        {Enum.reverse(acc), rest}

      _ ->
        {value, rest} = parse_value(json)
        rest = String.trim_leading(rest)

        case rest do
          "," <> rest ->
            parse_array(String.trim_leading(rest), [value | acc])

          "]" <> rest ->
            {Enum.reverse([value | acc]), rest}
        end
    end
  end

  defp parse_value(json) do
    json = String.trim_leading(json)

    cond do
      String.starts_with?(json, "null") ->
        {nil, String.slice(json, 4..-1//1)}

      String.starts_with?(json, "true") ->
        {true, String.slice(json, 4..-1//1)}

      String.starts_with?(json, "false") ->
        {false, String.slice(json, 5..-1//1)}

      String.starts_with?(json, "\"") ->
        parse_string(String.slice(json, 1..-1//1), "")

      String.starts_with?(json, "{") ->
        parse_object(String.slice(json, 1..-1//1), %{})

      String.starts_with?(json, "[") ->
        parse_array(String.slice(json, 1..-1//1), [])

      true ->
        # Parse number
        {num_str, rest} =
          Enum.split_while(String.to_charlist(json), fn c ->
            c in ?0..?9 or c == ?. or c == ?- or c == ?+
          end)

        num_str = to_string(num_str)

        num =
          if String.contains?(num_str, ".") do
            String.to_float(num_str)
          else
            String.to_integer(num_str)
          end

        {num, to_string(rest)}
    end
  end
end
