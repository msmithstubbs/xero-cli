defmodule XeroCLI.MixProject do
  use Mix.Project

  def project do
    [
      app: :xero_cli,
      version: "0.1.0",
      elixir: "~> 1.14",
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      escript: escript()
    ]
  end

  defp escript do
    [
      main_module: XeroCLI.CLI,
      name: "xero"
    ]
  end

  # Run "mix help compile.app" to learn about applications.
  def application do
    [
      extra_applications: [:logger, :inets, :ssl, :crypto]
    ]
  end

  # Run "mix help deps" to learn about dependencies.
  defp deps do
    [
      # Using built-in Erlang modules to avoid external dependencies
    ]
  end
end
