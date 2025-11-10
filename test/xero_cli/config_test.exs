defmodule XeroCLI.ConfigTest do
  use ExUnit.Case, async: false
  import Bitwise
  alias XeroCLI.Config

  setup do
    # Clean up any existing config before each test
    if File.exists?(Config.config_path()) do
      File.rm!(Config.config_path())
    end

    on_exit(fn ->
      # Clean up after test
      if File.exists?(Config.config_path()) do
        File.rm!(Config.config_path())
      end
    end)

    :ok
  end

  describe "config_path/0" do
    test "returns path in user home directory" do
      path = Config.config_path()
      assert String.ends_with?(path, ".xero-cli/config.json")
      assert Path.basename(path) == "config.json"
    end
  end

  describe "ensure_config_dir/0" do
    test "creates config directory if it doesn't exist" do
      config_dir = Config.ensure_config_dir()
      assert File.dir?(config_dir)
      assert String.ends_with?(config_dir, ".xero-cli")
    end

    test "returns existing directory if already created" do
      dir1 = Config.ensure_config_dir()
      dir2 = Config.ensure_config_dir()
      assert dir1 == dir2
      assert File.dir?(dir1)
    end
  end

  describe "read_config/0" do
    test "returns empty map when config doesn't exist" do
      assert Config.read_config() == {:ok, %{}}
    end

    test "reads existing config file" do
      config_data = %{"key" => "value", "number" => 42}
      :ok = Config.write_config(config_data)

      assert Config.read_config() == {:ok, config_data}
    end

    test "returns error for invalid JSON" do
      Config.ensure_config_dir()
      path = Config.config_path()
      File.write!(path, "invalid json")

      {:error, reason} = Config.read_config()
      assert reason =~ "Invalid config file format"
    end
  end

  describe "write_config/1" do
    test "creates config file with correct data" do
      config_data = %{"test" => "data", "value" => 123}
      assert Config.write_config(config_data) == :ok

      path = Config.config_path()
      assert File.exists?(path)

      # Verify content
      {:ok, content} = File.read(path)
      {:ok, parsed} = Jason.decode(content)
      assert parsed == config_data
    end

    test "sets file permissions to 600" do
      config_data = %{"secure" => "data"}
      :ok = Config.write_config(config_data)

      path = Config.config_path()
      {:ok, stat} = File.stat(path)

      # Check that file is only readable/writable by owner
      # Note: This test may behave differently on Windows
      case :os.type() do
        {:unix, _os_name} ->
          # On Unix, mode should be 0o100600 (regular file with 600 permissions)
          assert (stat.mode &&& 0o777) == 0o600

        _ ->
          # Skip permission check on non-Unix systems
          :ok
      end
    end

    test "overwrites existing config" do
      :ok = Config.write_config(%{"first" => "config"})
      :ok = Config.write_config(%{"second" => "config"})

      {:ok, config} = Config.read_config()
      assert config == %{"second" => "config"}
    end
  end

  describe "get_credentials/0" do
    test "returns error when not authenticated" do
      {:error, reason} = Config.get_credentials()
      assert reason =~ "Not authenticated"
    end

    test "returns credentials when they exist" do
      credentials = %{
        "access_token" => "token123",
        "refresh_token" => "refresh123",
        "tenant_id" => "tenant123"
      }

      :ok = Config.set_credentials(credentials)

      assert Config.get_credentials() == {:ok, credentials}
    end
  end

  describe "set_credentials/1" do
    test "stores credentials in config" do
      credentials = %{
        "access_token" => "abc123",
        "refresh_token" => "xyz789",
        "tenant_id" => "tenant_id_123"
      }

      assert Config.set_credentials(credentials) == :ok

      {:ok, stored_creds} = Config.get_credentials()
      assert stored_creds == credentials
    end

    test "overwrites existing credentials" do
      old_creds = %{"access_token" => "old"}
      new_creds = %{"access_token" => "new"}

      :ok = Config.set_credentials(old_creds)
      :ok = Config.set_credentials(new_creds)

      {:ok, stored} = Config.get_credentials()
      assert stored == new_creds
    end

    test "preserves other config settings" do
      :ok = Config.set_setting("other_setting", "value")
      :ok = Config.set_credentials(%{"access_token" => "token"})

      assert Config.get_setting("other_setting") == "value"
    end
  end

  describe "delete_credentials/0" do
    test "removes credentials from config" do
      credentials = %{"access_token" => "token"}
      :ok = Config.set_credentials(credentials)

      assert Config.delete_credentials() == :ok

      {:error, reason} = Config.get_credentials()
      assert reason =~ "Not authenticated"
    end

    test "preserves other config settings" do
      :ok = Config.set_credentials(%{"access_token" => "token"})
      :ok = Config.set_setting("keep_this", "value")
      :ok = Config.delete_credentials()

      assert Config.get_setting("keep_this") == "value"
    end

    test "succeeds even when no credentials exist" do
      assert Config.delete_credentials() == :ok
    end
  end

  describe "get_setting/2" do
    test "returns default when setting doesn't exist" do
      assert Config.get_setting("nonexistent", "default") == "default"
    end

    test "returns nil default when not specified" do
      assert Config.get_setting("nonexistent") == nil
    end

    test "returns stored setting" do
      :ok = Config.set_setting("my_setting", "my_value")
      assert Config.get_setting("my_setting") == "my_value"
    end
  end

  describe "set_setting/2" do
    test "stores setting in config" do
      assert Config.set_setting("setting1", "value1") == :ok
      assert Config.get_setting("setting1") == "value1"
    end

    test "overwrites existing setting" do
      :ok = Config.set_setting("setting", "old")
      :ok = Config.set_setting("setting", "new")
      assert Config.get_setting("setting") == "new"
    end

    test "stores different types of values" do
      :ok = Config.set_setting("string", "text")
      :ok = Config.set_setting("number", 42)
      :ok = Config.set_setting("boolean", true)
      :ok = Config.set_setting("map", %{"nested" => "value"})

      assert Config.get_setting("string") == "text"
      assert Config.get_setting("number") == 42
      assert Config.get_setting("boolean") == true
      assert Config.get_setting("map") == %{"nested" => "value"}
    end
  end

  describe "integration tests" do
    test "complete workflow with credentials and settings" do
      # Set some settings
      :ok = Config.set_setting("client_id", "my_client_id")
      :ok = Config.set_setting("theme", "dark")

      # Set credentials
      credentials = %{
        "access_token" => "token123",
        "refresh_token" => "refresh123",
        "tenant_id" => "tenant123",
        "expires_in" => 1800
      }

      :ok = Config.set_credentials(credentials)

      # Verify everything is stored
      assert Config.get_setting("client_id") == "my_client_id"
      assert Config.get_setting("theme") == "dark"
      {:ok, stored_creds} = Config.get_credentials()
      assert stored_creds == credentials

      # Update credentials
      updated_creds = Map.put(credentials, "access_token", "new_token")
      :ok = Config.set_credentials(updated_creds)

      # Verify settings still exist and credentials are updated
      assert Config.get_setting("client_id") == "my_client_id"
      {:ok, new_stored} = Config.get_credentials()
      assert new_stored["access_token"] == "new_token"

      # Delete credentials
      :ok = Config.delete_credentials()

      # Verify settings still exist but credentials are gone
      assert Config.get_setting("client_id") == "my_client_id"
      {:error, _} = Config.get_credentials()
    end
  end
end
