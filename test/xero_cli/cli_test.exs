defmodule XeroCLI.CLITest do
  use ExUnit.Case
  import ExUnit.CaptureIO

  describe "parse_args/1" do
    test "parses help flag" do
      output = capture_io(fn ->
        XeroCLI.CLI.main(["--help"])
      end)

      assert output =~ "Xero CLI - Command line tool for interacting with Xero API"
      assert output =~ "USAGE:"
      assert output =~ "xero <command> <subcommand>"
    end

    test "parses -h flag" do
      output = capture_io(fn ->
        XeroCLI.CLI.main(["-h"])
      end)

      assert output =~ "Xero CLI"
    end

    test "shows help when no arguments provided" do
      output = capture_io(fn ->
        XeroCLI.CLI.main([])
      end)

      assert output =~ "Xero CLI"
      assert output =~ "CORE COMMANDS"
    end

    test "shows error for unknown command" do
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(:stderr, fn ->
          catch_exit(XeroCLI.CLI.main(["unknown"]))
        end)
      end
    end

    test "shows error for auth without subcommand" do
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(:stderr, fn ->
          catch_exit(XeroCLI.CLI.main(["auth"]))
        end)
      end
    end

    test "shows error for invoices without subcommand" do
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(:stderr, fn ->
          catch_exit(XeroCLI.CLI.main(["invoices"]))
        end)
      end
    end

    test "shows error for unknown auth subcommand" do
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(:stderr, fn ->
          catch_exit(XeroCLI.CLI.main(["auth", "unknown"]))
        end)
      end
    end

    test "shows error for unknown invoices subcommand" do
      assert_raise ExUnit.AssertionError, fn ->
        capture_io(:stderr, fn ->
          catch_exit(XeroCLI.CLI.main(["invoices", "unknown"]))
        end)
      end
    end
  end

  describe "help output" do
    test "includes auth commands" do
      output = capture_io(fn ->
        XeroCLI.CLI.main(["--help"])
      end)

      assert output =~ "auth login"
      assert output =~ "auth logout"
      assert output =~ "auth status"
    end

    test "includes invoice commands" do
      output = capture_io(fn ->
        XeroCLI.CLI.main(["--help"])
      end)

      assert output =~ "invoices list"
    end

    test "includes examples" do
      output = capture_io(fn ->
        XeroCLI.CLI.main(["--help"])
      end)

      assert output =~ "EXAMPLES"
      assert output =~ "$ xero auth login"
      assert output =~ "$ xero invoices list"
    end
  end
end
