# typed: false
# frozen_string_literal: true

# This file was generated by GoReleaser. DO NOT EDIT.
class Kit < Formula
  desc "Crazy fast local dev loop."
  homepage "https://github.com/kitproj/kit"
  version "0.1.14"

  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/kitproj/kit/releases/download/v0.1.14/kit_0.1.14_Darwin_x86_64.tar.gz"
      sha256 "ac842cc16184adeac1929a8e67d44388c2e71c7854de5c0efa19ad3551b4b8d8"

      def install
        bin.install "kit"
      end
    end
    if Hardware::CPU.arm?
      url "https://github.com/kitproj/kit/releases/download/v0.1.14/kit_0.1.14_Darwin_arm64.tar.gz"
      sha256 "166966bcb110722f716d95e9cfcd62f1733077016f6f879b8da72701f34089ae"

      def install
        bin.install "kit"
      end
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/kitproj/kit/releases/download/v0.1.14/kit_0.1.14_Linux_arm64.tar.gz"
      sha256 "8a33ae1bce7fe22502f67fba0b257364b39f6600b3f5142dba416ed376085ca8"

      def install
        bin.install "kit"
      end
    end
    if Hardware::CPU.intel?
      url "https://github.com/kitproj/kit/releases/download/v0.1.14/kit_0.1.14_Linux_x86_64.tar.gz"
      sha256 "cac269900043bea2126fd1564d0b291c01ad9a1448e91a535a1ffc07e4574f77"

      def install
        bin.install "kit"
      end
    end
  end
end
