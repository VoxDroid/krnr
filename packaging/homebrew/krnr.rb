class Krnr < Formula
  desc "Kernel Runner — a cross-platform CLI to manage a command registry"
  homepage "https://github.com/VoxDroid/krnr"
  url "https://github.com/VoxDroid/krnr/archive/refs/tags/v0.0.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}")
  end

  test do
    assert_match "krnr", shell_output("#{bin}/krnr --help")
  end
end
