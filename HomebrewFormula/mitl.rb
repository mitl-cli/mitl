#!/usr/bin/env ruby
# HomebrewFormula/mitl.rb - Homebrew formula for mitl
# To be hosted in mitl-cli/homebrew-tap repository

class Mitl < Formula
  desc "MY Tool Launch - 10x faster Docker alternative for Mac"
  homepage "https://github.com/mitl-cli/mitl"
  version "0.1.0-alpha"
  license "MIT"

  if Hardware::CPU.arm?
    url "https://github.com/mitl-cli/mitl/releases/download/v0.1.0-alpha/mitl-darwin-arm64.tar.gz"
    sha256 "b6f31fcb5bdc0a79faf012346b5d618a658a9c0b82e4425ff962b0c492ff26e6"
  else
    url "https://github.com/mitl-cli/mitl/releases/download/v0.1.0-alpha/mitl-darwin-amd64.tar.gz"
    sha256 "1c517f3e64cdd05493dd76e18853a70fcd80003805115396f5a11a769f647bb8"
  end

  depends_on macos: :monterey

  def caveats
    <<~EOS
      #{bold}🏹 mitl installed successfully!#{reset}

      #{bold}Quick Start:#{reset}
        mitl doctor          # Check your setup
        mitl run npm test    # Run any command

      #{bold}For maximum performance on Apple Silicon:#{reset}
        Install Apple's Container framework (5-10x faster than Docker)
        Download from: developer.apple.com/virtualization

      #{bold}Get started:#{reset}
        https://github.com/mitl-cli/mitl#quick-start
    EOS
  end

  def install
    bin.install "mitl"

    # Install shell completions
    generate_completions_from_executable(bin/"mitl", "completion")

    # Create config directory
    (var/"mitl").mkpath

    # Install man page if it exists
    man1.install "mitl.1" if File.exist?("mitl.1")
  end

  test do
    system "#{bin}/mitl", "version"
    system "#{bin}/mitl", "doctor"
  end

  def post_install
    if Hardware::CPU.arm? && !File.exist?("/usr/bin/container")
      opoo "Apple Container not found. mitl will use Docker (slower)."
      opoo "Install Apple Container for 5-10x speed boost!"
    end

    unless File.exist?("/usr/local/bin/docker") || File.exist?("/opt/homebrew/bin/docker")
      opoo "No container runtime found. Install Docker or Podman:"
      opoo "  brew install --cask docker"
    end
  end

  private

  def bold
    "\033[1m"
  end

  def reset
    "\033[0m"
  end
end
