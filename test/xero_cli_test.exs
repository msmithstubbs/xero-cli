defmodule XeroCLITest do
  use ExUnit.Case
  doctest XeroCLI

  test "greets the world" do
    assert XeroCLI.hello() == :world
  end
end
