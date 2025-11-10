defmodule XeroCLI.JSONTest do
  use ExUnit.Case
  alias Jason

  describe "encode/1" do
    test "encodes null" do
      {:ok, json} = Jason.encode(nil)
      assert json == "null"
    end

    test "encodes booleans" do
      {:ok, json_true} = Jason.encode(true)
      assert json_true == "true"

      {:ok, json_false} = Jason.encode(false)
      assert json_false == "false"
    end

    test "encodes numbers" do
      {:ok, json_int} = Jason.encode(42)
      assert json_int == "42"

      {:ok, json_float} = Jason.encode(3.14)
      assert json_float =~ "3.14"
    end

    test "encodes strings" do
      {:ok, json} = Jason.encode("hello")
      assert json == "\"hello\""
    end

    test "encodes strings with special characters" do
      {:ok, json} = Jason.encode("hello\nworld")
      assert json == "\"hello\\nworld\""

      {:ok, json2} = Jason.encode("with \"quotes\"")
      assert json2 == "\"with \\\"quotes\\\"\""
    end

    test "encodes empty array" do
      {:ok, json} = Jason.encode([])
      assert json == "[]"
    end

    test "encodes array with elements" do
      {:ok, json} = Jason.encode([1, 2, 3])
      assert json == "[1,2,3]"
    end

    test "encodes mixed array" do
      {:ok, json} = Jason.encode([1, "two", true, nil])
      assert json == "[1,\"two\",true,null]"
    end

    test "encodes empty map" do
      {:ok, json} = Jason.encode(%{})
      assert json == "{}"
    end

    test "encodes map with string keys" do
      {:ok, json} = Jason.encode(%{"name" => "John", "age" => 30})
      assert json =~ "\"name\""
      assert json =~ "\"John\""
      assert json =~ "\"age\""
      assert json =~ "30"
    end

    test "encodes nested structures" do
      data = %{
        "user" => %{
          "name" => "Alice",
          "tags" => ["admin", "user"]
        }
      }

      {:ok, json} = Jason.encode(data)
      assert json =~ "\"user\""
      assert json =~ "\"name\""
      assert json =~ "\"Alice\""
      assert json =~ "\"tags\""
      assert json =~ "\"admin\""
    end

    test "encodes atom keys as strings" do
      {:ok, json} = Jason.encode(%{name: "Bob"})
      assert json =~ "\"name\""
      assert json =~ "\"Bob\""
    end
  end

  describe "decode/1" do
    test "decodes null" do
      {:ok, value} = Jason.decode("null")
      assert value == nil
    end

    test "decodes booleans" do
      {:ok, value_true} = Jason.decode("true")
      assert value_true == true

      {:ok, value_false} = Jason.decode("false")
      assert value_false == false
    end

    test "decodes numbers" do
      {:ok, int_value} = Jason.decode("42")
      assert int_value == 42

      {:ok, float_value} = Jason.decode("3.14")
      assert float_value == 3.14
    end

    test "decodes strings" do
      {:ok, value} = Jason.decode("\"hello\"")
      assert value == "hello"
    end

    test "decodes strings with escaped characters" do
      {:ok, value} = Jason.decode("\"hello\\nworld\"")
      assert value == "hello\nworld"

      {:ok, value2} = Jason.decode("\"with \\\"quotes\\\"\"")
      assert value2 == "with \"quotes\""
    end

    test "decodes empty array" do
      {:ok, value} = Jason.decode("[]")
      assert value == []
    end

    test "decodes array with elements" do
      {:ok, value} = Jason.decode("[1,2,3]")
      assert value == [1, 2, 3]
    end

    test "decodes array with mixed types" do
      {:ok, value} = Jason.decode("[1,\"two\",true,null]")
      assert value == [1, "two", true, nil]
    end

    test "decodes empty object" do
      {:ok, value} = Jason.decode("{}")
      assert value == %{}
    end

    test "decodes object with properties" do
      {:ok, value} = Jason.decode("{\"name\":\"John\",\"age\":30}")
      assert value == %{"name" => "John", "age" => 30}
    end

    test "decodes nested structures" do
      json = "{\"user\":{\"name\":\"Alice\",\"tags\":[\"admin\",\"user\"]}}"
      {:ok, value} = Jason.decode(json)

      assert value == %{
               "user" => %{
                 "name" => "Alice",
                 "tags" => ["admin", "user"]
               }
             }
    end

    test "decodes with whitespace" do
      json = """
      {
        "name": "Bob",
        "age": 25
      }
      """

      {:ok, value} = Jason.decode(json)
      assert value == %{"name" => "Bob", "age" => 25}
    end

    test "handles negative numbers" do
      {:ok, value} = Jason.decode("-42")
      assert value == -42

      {:ok, value2} = Jason.decode("-3.14")
      assert value2 == -3.14
    end
  end

  describe "round-trip encoding and decoding" do
    test "encodes and decodes simple map" do
      original = %{"key" => "value", "number" => 123}
      {:ok, json} = Jason.encode(original)
      {:ok, decoded} = Jason.decode(json)
      assert decoded == original
    end

    test "encodes and decodes complex nested structure" do
      original = %{
        "users" => [
          %{"name" => "Alice", "active" => true, "score" => 98.5},
          %{"name" => "Bob", "active" => false, "score" => 87.3}
        ],
        "total" => 2
      }

      {:ok, json} = Jason.encode(original)
      {:ok, decoded} = Jason.decode(json)
      assert decoded == original
    end
  end

  describe "error handling" do
    test "returns error for invalid JSON" do
      result = Jason.decode("{invalid json}")
      assert match?({:error, _}, result)
    end

    test "returns error for unclosed string" do
      result = Jason.decode("\"unclosed")
      assert match?({:error, _}, result)
    end

    test "returns error for unclosed array" do
      result = Jason.decode("[1,2,3")
      assert match?({:error, _}, result)
    end

    test "returns error for unclosed object" do
      result = Jason.decode("{\"key\":\"value\"")
      assert match?({:error, _}, result)
    end
  end
end
