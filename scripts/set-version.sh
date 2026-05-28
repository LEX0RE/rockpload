#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT_DIR/VERSION"

usage() {
    cat <<'USAGE'
Usage:
  scripts/set-version.sh <version>
  scripts/set-version.sh --check
  scripts/set-version.sh --print-tag

Examples:
  scripts/set-version.sh 1.3.0
  scripts/set-version.sh v1.3.0-rc.1
  scripts/set-version.sh 1.3.0-beta.2

The version is stored without a leading "v". Release tags should still use "v":
  git tag "v$(cat VERSION)"
USAGE
}

read_version() {
    tr -d '[:space:]' < "$VERSION_FILE"
}

normalize_version() {
    local version="$1"
    version="${version#"${version%%[![:space:]]*}"}"
    version="${version%"${version##*[![:space:]]}"}"
    version="${version#v}"
    printf '%s\n' "$version"
}

validate_version() {
    local version="$1"
    local semver='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'

    if [[ ! "$version" =~ $semver ]]; then
        echo "Invalid version: $version" >&2
        echo "Expected SemVer like 1.3.0, 1.3.0-rc.1, or v1.3.0-beta.2." >&2
        exit 1
    fi
}

check_version_files() {
    local version
    version="$(read_version)"
    validate_version "$version"

    if grep -En 'rockploadVersion = "[0-9]|VERSION=\$\{ROCKPLOAD_VERSION:-"[0-9]' "$ROOT_DIR/flake.nix" "$ROOT_DIR/scripts/build.sh"; then
        echo "Found a hardcoded release version outside VERSION." >&2
        exit 1
    fi

    echo "Rockpload version is $version"
    echo "Release tag: v$version"
}

case "${1:-}" in
    -h|--help)
        usage
        ;;
    --check)
        check_version_files
        ;;
    --print-tag)
        version="$(read_version)"
        validate_version "$version"
        printf 'v%s\n' "$version"
        ;;
    "")
        usage >&2
        exit 1
        ;;
    *)
        version="$(normalize_version "$1")"
        validate_version "$version"
        printf '%s\n' "$version" > "$VERSION_FILE"
        echo "Updated VERSION to $version"
        echo "Release tag: v$version"
        ;;
esac
