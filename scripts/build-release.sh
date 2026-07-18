#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <version> <output-directory>" >&2
}

fail() {
  echo "build-release: $*" >&2
  exit 1
}

if (( $# != 2 )); then
  usage
  exit 2
fi

version=$1
output_argument=$2
[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] ||
  fail "version must be a semantic version tag such as v1.2.3 or v1.2.3-rc.1"

for command in go tar gzip mktemp; do
  command -v "$command" >/dev/null 2>&1 || fail "required command is unavailable: $command"
done

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
mkdir -p -- "$output_argument"
output_dir=$(cd -- "$output_argument" && pwd -P)
[[ -z $(find "$output_dir" -mindepth 1 -maxdepth 1 -print -quit) ]] ||
  fail "output directory must be empty: $output_dir"

temp_root=$(cd -- "${TMPDIR:-/tmp}" && pwd -P)
staging=$(mktemp -d "$temp_root/voice-asset-mcp-release.XXXXXX")
cleanup() {
  case $staging in
    "$temp_root"/voice-asset-mcp-release.*) rm -rf -- "$staging" ;;
    *) echo "build-release: refusing to clean unexpected path: $staging" >&2 ;;
  esac
}
trap cleanup EXIT

targets=(
  linux/amd64
  linux/arm64
  windows/amd64
  windows/arm64
  darwin/amd64
  darwin/arm64
)
ldflags="-s -w -buildid= -X main.version=$version"

cd -- "$repo_root"
export SOURCE_DATE_EPOCH=0
for target in "${targets[@]}"; do
  IFS=/ read -r goos goarch <<<"$target"
  package="voice-asset-mcp-$version-$goos-$goarch"
  package_dir="$staging/$package"
  extension=
  [[ $goos != windows ]] || extension=.exe
  mkdir -p -- "$package_dir"

  CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch GOWORK=off go build \
    -buildvcs=false -trimpath -ldflags="$ldflags" \
    -o "$package_dir/voice-asset-mcp$extension" ./cmd/voice-asset-mcp
  cp -- LICENSE CHANGELOG.md CONTRACT_VERSION .env.example "$package_dir/"
  tar --sort=name --mtime='@0' --owner=0 --group=0 --numeric-owner \
    -C "$staging" -cf - "$package" | gzip -n >"$staging/$package.tar.gz"
  mv -- "$staging/$package.tar.gz" "$output_dir/"
  rm -rf -- "$package_dir"
done

echo "built ${#targets[@]} release archives in $output_dir"
