defmodule XeroCLITest do
  use ExUnit.Case
  doctest XeroCLI

  describe "version/0" do
    test "returns version string" do
      assert XeroCLI.version() == "0.1.0"
    end
  end
end
