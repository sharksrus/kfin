#!/usr/bin/env bash
set -euo pipefail

REPO="sharksrus/kfin"
VERSION="${1:-}"
OUTPUT_PATH="${2:-Formula/kfin.rb}"

if [[ -z "${VERSION}" ]]; then
  echo "Usage: $0 <version-tag> [output-path]" >&2
  echo "Example: $0 v0.0.1 Formula/kfin.rb" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "ERROR: gh is required" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

assets=(
  "kfin_darwin_arm64.tar.gz.sha256"
  "kfin_linux_amd64.tar.gz.sha256"
  "kfin_linux_arm64.tar.gz.sha256"
)

for asset in "${assets[@]}"; do
  gh release download "${VERSION}" -R "${REPO}" -D "${tmpdir}" -p "${asset}" --clobber >/dev/null
  if [[ ! -f "${tmpdir}/${asset}" ]]; then
    echo "ERROR: missing downloaded checksum file: ${asset}" >&2
    exit 1
  fi
done

sha_darwin_arm64="$(awk '{print $1}' "${tmpdir}/kfin_darwin_arm64.tar.gz.sha256")"
sha_linux_amd64="$(awk '{print $1}' "${tmpdir}/kfin_linux_amd64.tar.gz.sha256")"
sha_linux_arm64="$(awk '{print $1}' "${tmpdir}/kfin_linux_arm64.tar.gz.sha256")"

mkdir -p "$(dirname "${OUTPUT_PATH}")"

cat > "${OUTPUT_PATH}" <<RUBY
class Kfin < Formula
  desc "Kubernetes cost visibility CLI"
  homepage "https://github.com/sharksrus/kfin"
  version "${VERSION#v}"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/sharksrus/kfin/releases/download/${VERSION}/kfin_darwin_arm64.tar.gz"
      sha256 "${sha_darwin_arm64}"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/sharksrus/kfin/releases/download/${VERSION}/kfin_linux_amd64.tar.gz"
      sha256 "${sha_linux_amd64}"
    end

    if Hardware::CPU.arm?
      url "https://github.com/sharksrus/kfin/releases/download/${VERSION}/kfin_linux_arm64.tar.gz"
      sha256 "${sha_linux_arm64}"
    end
  end

  def install
    bin.install "kfin"
  end

  test do
    output = shell_output("#{bin}/kfin --version")
    assert_match "kfin", output
  end
end
RUBY

echo "Wrote ${OUTPUT_PATH}"
