defmodule XeroCLI.Config do
  @moduledoc """
  Handles configuration and credential storage for Xero CLI.
  Stores credentials securely in the user's home directory.
  """

  @config_dir ".xero-cli"
  @config_file "config.json"

  def config_path do
    Path.join([System.user_home!(), @config_dir, @config_file])
  end

  def ensure_config_dir do
    dir = Path.join(System.user_home!(), @config_dir)
    File.mkdir_p!(dir)
    dir
  end

  def read_config do
    path = config_path()

    if File.exists?(path) do
      case File.read(path) do
        {:ok, content} ->
          case Jason.decode(content) do
            {:ok, config} -> {:ok, config}
            {:error, _} -> {:error, "Invalid config file format"}
          end

        {:error, reason} ->
          {:error, "Failed to read config: #{inspect(reason)}"}
      end
    else
      {:ok, %{}}
    end
  end

  def write_config(config) do
    ensure_config_dir()
    path = config_path()

    case Jason.encode(config, pretty: true) do
      {:ok, json} ->
        case File.write(path, json) do
          :ok ->
            # Set file permissions to 600 (read/write for owner only)
            File.chmod!(path, 0o600)
            :ok

          {:error, reason} ->
            {:error, "Failed to write config: #{inspect(reason)}"}
        end

      {:error, reason} ->
        {:error, "Failed to encode config: #{inspect(reason)}"}
    end
  end

  def get_credentials do
    case read_config() do
      {:ok, config} ->
        case Map.get(config, "credentials") do
          nil -> {:error, "Not authenticated. Run 'xero auth login' first."}
          creds -> {:ok, creds}
        end

      {:error, reason} ->
        {:error, reason}
    end
  end

  def set_credentials(credentials) do
    case read_config() do
      {:ok, config} ->
        updated_config = Map.put(config, "credentials", credentials)
        write_config(updated_config)

      {:error, _} ->
        write_config(%{"credentials" => credentials})
    end
  end

  def delete_credentials do
    case read_config() do
      {:ok, config} ->
        updated_config = Map.delete(config, "credentials")
        write_config(updated_config)

      {:error, _} ->
        :ok
    end
  end

  def get_setting(key, default \\ nil) do
    case read_config() do
      {:ok, config} -> Map.get(config, key, default)
      {:error, _} -> default
    end
  end

  def set_setting(key, value) do
    case read_config() do
      {:ok, config} ->
        updated_config = Map.put(config, key, value)
        write_config(updated_config)

      {:error, _} ->
        write_config(%{key => value})
    end
  end
end
