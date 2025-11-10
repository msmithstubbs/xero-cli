defmodule XeroCLI.HTTPTest do
  use ExUnit.Case
  alias XeroCLI.HTTP

  describe "parse_response/1" do
    test "parses successful JSON response" do
      response = {:ok, %{status_code: 200, body: "{\"key\":\"value\"}"}}
      assert HTTP.parse_response(response) == {:ok, %{"key" => "value"}}
    end

    test "parses successful response with array" do
      response = {:ok, %{status_code: 200, body: "[1,2,3]"}}
      assert HTTP.parse_response(response) == {:ok, [1, 2, 3]}
    end

    test "returns plain text for non-JSON 200 response" do
      response = {:ok, %{status_code: 200, body: "plain text"}}
      assert HTTP.parse_response(response) == {:ok, "plain text"}
    end

    test "handles 201 created response" do
      response = {:ok, %{status_code: 201, body: "{\"created\":true}"}}
      assert HTTP.parse_response(response) == {:ok, %{"created" => true}}
    end

    test "handles 204 no content" do
      response = {:ok, %{status_code: 204, body: ""}}
      assert HTTP.parse_response(response) == {:ok, ""}
    end

    test "returns error for 400 bad request" do
      response = {:ok, %{status_code: 400, body: "{\"error\":\"bad request\"}"}}
      {:error, {code, data}} = HTTP.parse_response(response)
      assert code == 400
      assert data == %{"error" => "bad request"}
    end

    test "returns error for 401 unauthorized" do
      response = {:ok, %{status_code: 401, body: "{\"error\":\"unauthorized\"}"}}
      {:error, {code, data}} = HTTP.parse_response(response)
      assert code == 401
      assert data == %{"error" => "unauthorized"}
    end

    test "returns error for 404 not found" do
      response = {:ok, %{status_code: 404, body: "Not Found"}}
      {:error, {code, body}} = HTTP.parse_response(response)
      assert code == 404
      assert body == "Not Found"
    end

    test "returns error for 500 server error" do
      response = {:ok, %{status_code: 500, body: "{\"error\":\"server error\"}"}}
      {:error, {code, data}} = HTTP.parse_response(response)
      assert code == 500
      assert data == %{"error" => "server error"}
    end

    test "handles error tuple" do
      response = {:error, "connection failed"}
      assert HTTP.parse_response(response) == {:error, "connection failed"}
    end

    test "handles error with complex reason" do
      response = {:error, {:timeout, "request timed out"}}
      assert HTTP.parse_response(response) == {:error, {:timeout, "request timed out"}}
    end
  end

  describe "HTTP methods" do
    test "start initializes inets and ssl" do
      # This should not crash
      assert HTTP.start() == :ok
    end
  end

  describe "response structure" do
    test "successful response has expected structure" do
      # This test verifies the structure we expect from HTTP calls
      # We can't make real HTTP calls in tests, but we can verify the structure

      expected_response = %{
        status_code: 200,
        headers: %{"content-type" => "application/json"},
        body: "{\"test\":\"data\"}"
      }

      result = {:ok, expected_response}
      {:ok, parsed} = HTTP.parse_response(result)
      assert parsed == %{"test" => "data"}
    end

    test "error response has expected structure" do
      expected_response = %{
        status_code: 401,
        headers: %{"content-type" => "application/json"},
        body: "{\"error\":\"invalid token\"}"
      }

      result = {:ok, expected_response}
      {:error, {code, data}} = HTTP.parse_response(result)
      assert code == 401
      assert data == %{"error" => "invalid token"}
    end
  end

  describe "JSON parsing in responses" do
    test "handles nested JSON structures" do
      response = {:ok, %{status_code: 200, body: "{\"user\":{\"name\":\"John\",\"age\":30}}"}}
      {:ok, parsed} = HTTP.parse_response(response)
      assert parsed == %{"user" => %{"name" => "John", "age" => 30}}
    end

    test "handles arrays in response" do
      response = {:ok, %{status_code: 200, body: "{\"items\":[{\"id\":1},{\"id\":2}]}"}}
      {:ok, parsed} = HTTP.parse_response(response)
      assert parsed == %{"items" => [%{"id" => 1}, %{"id" => 2}]}
    end

    test "handles empty JSON object" do
      response = {:ok, %{status_code: 200, body: "{}"}}
      {:ok, parsed} = HTTP.parse_response(response)
      assert parsed == %{}
    end

    test "handles empty JSON array" do
      response = {:ok, %{status_code: 200, body: "[]"}}
      {:ok, parsed} = HTTP.parse_response(response)
      assert parsed == []
    end
  end

  describe "status code ranges" do
    test "treats all 2xx as success" do
      for status <- [200, 201, 202, 203, 204, 205, 206] do
        response = {:ok, %{status_code: status, body: "{\"ok\":true}"}}
        assert {:ok, _} = HTTP.parse_response(response)
      end
    end

    test "treats 3xx as error" do
      for status <- [300, 301, 302, 303, 304] do
        response = {:ok, %{status_code: status, body: "redirect"}}
        assert {:error, _} = HTTP.parse_response(response)
      end
    end

    test "treats 4xx as error" do
      for status <- [400, 401, 403, 404, 405] do
        response = {:ok, %{status_code: status, body: "error"}}
        assert {:error, _} = HTTP.parse_response(response)
      end
    end

    test "treats 5xx as error" do
      for status <- [500, 501, 502, 503, 504] do
        response = {:ok, %{status_code: status, body: "error"}}
        assert {:error, _} = HTTP.parse_response(response)
      end
    end
  end
end
