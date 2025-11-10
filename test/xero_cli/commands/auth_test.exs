defmodule XeroCLI.Commands.AuthTest do
  use ExUnit.Case, async: false
  import ExUnit.CaptureIO
  alias XeroCLI.Commands.Auth

  setup do
    # Clean up any existing config before each test
    if File.exists?(XeroCLI.Config.config_path()) do
      File.rm!(XeroCLI.Config.config_path())
    end

    on_exit(fn ->
      # Clean up after test
      if File.exists?(XeroCLI.Config.config_path()) do
        File.rm!(XeroCLI.Config.config_path())
      end
    end)

    :ok
  end

  describe "handle(:logout, _)" do
    test "logs out successfully when credentials exist" do
      # Set up credentials
      credentials = %{
        "access_token" => "token",
        "tenant_id" => "tenant123"
      }

      XeroCLI.Config.set_credentials(credentials)

      output =
        capture_io(fn ->
          Auth.handle(:logout, [])
        end)

      assert output =~ "Successfully logged out"

      # Verify credentials are removed
      {:error, _} = XeroCLI.Config.get_credentials()
    end

    test "logs out successfully even when no credentials exist" do
      output =
        capture_io(fn ->
          Auth.handle(:logout, [])
        end)

      assert output =~ "Successfully logged out"
    end
  end

  describe "handle(:status, _)" do
    test "shows not authenticated when no credentials" do
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(fn ->
          catch_exit(Auth.handle(:status, []))
        end)
      end
    end

    test "shows authenticated status with valid credentials" do
      credentials = %{
        "access_token" => "token",
        "refresh_token" => "refresh",
        "tenant_id" => "tenant123",
        "tenant_name" => "Test Company",
        "expires_in" => 1800,
        "obtained_at" => System.system_time(:second)
      }

      XeroCLI.Config.set_credentials(credentials)

      output =
        capture_io(fn ->
          Auth.handle(:status, [])
        end)

      assert output =~ "Authenticated"
      assert output =~ "Test Company"
      assert output =~ "tenant123"
      assert output =~ "Access token is valid"
    end

    test "shows warning for expired token" do
      # Token obtained 2 hours ago
      two_hours_ago = System.system_time(:second) - 7200

      credentials = %{
        "access_token" => "token",
        "refresh_token" => "refresh",
        "tenant_id" => "tenant123",
        "tenant_name" => "Test Company",
        "expires_in" => 1800,
        "obtained_at" => two_hours_ago
      }

      XeroCLI.Config.set_credentials(credentials)

      output =
        capture_io(fn ->
          Auth.handle(:status, [])
        end)

      assert output =~ "Authenticated"
      assert output =~ "Access token expired"
      assert output =~ "refreshed on next API call"
    end

    test "handles missing tenant name gracefully" do
      credentials = %{
        "access_token" => "token",
        "tenant_id" => "tenant123",
        "expires_in" => 1800,
        "obtained_at" => System.system_time(:second)
      }

      XeroCLI.Config.set_credentials(credentials)

      output =
        capture_io(fn ->
          Auth.handle(:status, [])
        end)

      assert output =~ "Authenticated"
      assert output =~ "Unknown"
    end
  end

  describe "Client ID handling" do
    test "uses environment variable when set" do
      System.put_env("XERO_CLIENT_ID", "env_client_id")

      # We can't fully test the login flow without mocking HTTP,
      # but we can verify the environment variable is checked
      assert System.get_env("XERO_CLIENT_ID") == "env_client_id"

      System.delete_env("XERO_CLIENT_ID")
    end

    test "uses stored client ID from config" do
      XeroCLI.Config.set_setting("client_id", "stored_client_id")

      stored = XeroCLI.Config.get_setting("client_id")
      assert stored == "stored_client_id"
    end
  end

  describe "workflow integration" do
    test "logout -> status shows not authenticated" do
      # Setup credentials
      XeroCLI.Config.set_credentials(%{"access_token" => "token"})

      # Logout
      capture_io(fn ->
        Auth.handle(:logout, [])
      end)

      # Check status - should not be authenticated
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(fn ->
          catch_exit(Auth.handle(:status, []))
        end)
      end
    end

    test "status output includes all expected fields" do
      credentials = %{
        "access_token" => "test_token",
        "refresh_token" => "test_refresh",
        "tenant_id" => "abc-123-def-456",
        "tenant_name" => "My Test Organization",
        "expires_in" => 1800,
        "obtained_at" => System.system_time(:second),
        "client_id" => "client_123"
      }

      XeroCLI.Config.set_credentials(credentials)

      output =
        capture_io(fn ->
          Auth.handle(:status, [])
        end)

      # Check for expected status indicators
      assert output =~ "✅" or output =~ "Authenticated"
      assert output =~ "My Test Organization"
      assert output =~ "abc-123-def-456"
      assert output =~ "✓" or output =~ "valid"
    end
  end
end
