#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "usage: $0 <version> <release-directory> [--require-sbom]" >&2
}

fail() {
  echo "verify-release: $*" >&2
  exit 1
}

if (( $# < 2 || $# > 3 )); then
  usage
  exit 2
fi

version=$1
release_argument=$2
require_sbom=false
if (( $# == 3 )); then
  [[ $3 == --require-sbom ]] || {
    usage
    exit 2
  }
  require_sbom=true
fi

[[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$ ]] ||
  fail "invalid semantic version tag: $version"
[[ -d $release_argument ]] || fail "release directory does not exist: $release_argument"
release_dir=$(cd -- "$release_argument" && pwd -P)
repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)

for command in go tar sha256sum mktemp cmp grep awk; do
  command -v "$command" >/dev/null 2>&1 || fail "required command is unavailable: $command"
done

[[ -f $release_dir/SHA256SUMS && ! -L $release_dir/SHA256SUMS ]] ||
  fail "SHA256SUMS must be a regular, non-symlink file"
if $require_sbom; then
  [[ -f $release_dir/voice-asset-mcp-source.cdx.json && ! -L $release_dir/voice-asset-mcp-source.cdx.json ]] ||
    fail "required CycloneDX SBOM is missing"
fi

targets=(
  linux/amd64
  linux/arm64
  windows/amd64
  windows/arm64
  darwin/amd64
  darwin/arm64
)
expected_archives=()
for target in "${targets[@]}"; do
  IFS=/ read -r goos goarch <<<"$target"
  expected_archives+=("voice-asset-mcp-$version-$goos-$goarch.tar.gz")
done
mapfile -t expected_archives < <(printf '%s\n' "${expected_archives[@]}" | LC_ALL=C sort)

shopt -s nullglob dotglob
archive_paths=("$release_dir"/*.tar.gz)
mapfile -t actual_archives < <(
  for archive in "${archive_paths[@]}"; do basename -- "$archive"; done | LC_ALL=C sort
)
[[ $(printf '%s\n' "${actual_archives[@]}") == $(printf '%s\n' "${expected_archives[@]}") ]] ||
  fail "release directory does not contain exactly the six expected archives"

artifact_paths=("$release_dir"/*.tar.gz "$release_dir"/*.json)
mapfile -t expected_checksum_names < <(
  for artifact in "${artifact_paths[@]}"; do
    [[ -f $artifact && ! -L $artifact ]] || fail "artifact must be a regular, non-symlink file: $artifact"
    printf './%s\n' "$(basename -- "$artifact")"
  done | LC_ALL=C sort
)
mapfile -t checksum_names < <(awk '{name = $2; sub(/^\*/, "", name); print name}' "$release_dir/SHA256SUMS" | LC_ALL=C sort)
[[ $(printf '%s\n' "${checksum_names[@]}") == $(printf '%s\n' "${expected_checksum_names[@]}") ]] ||
  fail "SHA256SUMS does not cover exactly the release artifacts"
while read -r checksum name extra; do
  [[ -z ${extra:-} && $checksum =~ ^[0-9a-f]{64}$ && $name == \*./* ]] ||
    fail "malformed SHA256SUMS entry"
done <"$release_dir/SHA256SUMS"
(
  cd -- "$release_dir"
  sha256sum -c SHA256SUMS
)

temp_root=$(cd -- "${TMPDIR:-/tmp}" && pwd -P)
staging=$(mktemp -d "$temp_root/voice-asset-mcp-verify.XXXXXX")
cleanup() {
  case $staging in
    "$temp_root"/voice-asset-mcp-verify.*) rm -rf -- "$staging" ;;
    *) echo "verify-release: refusing to clean unexpected path: $staging" >&2 ;;
  esac
}
trap cleanup EXIT
host_goos=$(go env GOOS)
host_goarch=$(go env GOARCH)

for target in "${targets[@]}"; do
  IFS=/ read -r goos goarch <<<"$target"
  package="voice-asset-mcp-$version-$goos-$goarch"
  archive="$release_dir/$package.tar.gz"
  extension=
  [[ $goos != windows ]] || extension=.exe

  listing="$staging/$package.list"
  tar -tzf "$archive" >"$listing"
  [[ -s $listing ]] || fail "archive is empty: $archive"
  while IFS= read -r entry; do
    [[ $entry != /* && ! $entry =~ (^|/)\.\.(/|$) ]] || fail "unsafe archive path: $entry"
    case $entry in
      "$package" | "$package/" | "$package/"*) ;;
      *) fail "archive entry escapes its package root: $entry" ;;
    esac
  done <"$listing"

  extract_dir="$staging/extract-$goos-$goarch"
  mkdir -p -- "$extract_dir"
  tar -xzf "$archive" -C "$extract_dir"
  package_dir="$extract_dir/$package"
  [[ -d $package_dir ]] || fail "package directory is missing: $package"
  [[ -z $(find "$package_dir" -type l -print -quit) ]] || fail "package contains a symbolic link: $package"

  expected_top=(.env.example CHANGELOG.md CONTRACT_VERSION LICENSE "voice-asset-mcp$extension")
  mapfile -t expected_top < <(printf '%s\n' "${expected_top[@]}" | LC_ALL=C sort)
  top_paths=("$package_dir"/*)
  mapfile -t actual_top < <(
    for path in "${top_paths[@]}"; do basename -- "$path"; done | LC_ALL=C sort
  )
  [[ $(printf '%s\n' "${actual_top[@]}") == $(printf '%s\n' "${expected_top[@]}") ]] ||
    fail "unexpected top-level package contents: $package"
  cmp -- "$repo_root/CONTRACT_VERSION" "$package_dir/CONTRACT_VERSION" >/dev/null ||
    fail "CONTRACT_VERSION differs from the repository pin: $package"

  binary="$package_dir/voice-asset-mcp$extension"
  [[ -f $binary && ! -L $binary ]] || fail "binary is missing: $binary"
  metadata=$(go version -m "$binary") || fail "cannot inspect Go metadata: $binary"
  grep -Fq $'\tbuild\tGOOS='"$goos" <<<"$metadata" || fail "wrong GOOS in $binary"
  grep -Fq $'\tbuild\tGOARCH='"$goarch" <<<"$metadata" || fail "wrong GOARCH in $binary"
  grep -aFq -- "$version" "$binary" || fail "version was not injected into $binary"
  if [[ $goos == "$host_goos" && $goarch == "$host_goarch" ]]; then
    runtime_version=$("$binary" --version) || fail "host MCP --version command failed"
    [[ $runtime_version == "$version" ]] || fail "host MCP reported the wrong version: $runtime_version"
  fi
done

echo "verified ${#targets[@]} MCP archives and SHA-256 checksums"
