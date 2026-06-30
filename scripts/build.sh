#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_PATH="$ROOT_DIR/releases"
ARCH="amd64"
PACKAGE_NAME="rockpload"
VERSION_FILE="$ROOT_DIR/VERSION"

rm -rf "$RELEASE_PATH"
mkdir -p "$RELEASE_PATH"

if [ -n "${ROCKPLOAD_VERSION:-}" ]; then
    VERSION="$ROCKPLOAD_VERSION"
elif [ -f "$VERSION_FILE" ]; then
    VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
else
    VERSION="dev"
fi

###################
# GENERATE ASSETS #
###################

echo "Generating Assets"
go generate

###############
# PREPARATION #
###############

VERSION="${VERSION#v}"
VERSION_CLEAN=$(echo "$VERSION" | sed 's/\//-/g')
echo "Building version $VERSION_CLEAN"

###############
# BUILD LINUX #
###############

OS="linux"
OUTPUT="${PACKAGE_NAME}_${VERSION_CLEAN}-${OS}"
FILE="$RELEASE_PATH/$OUTPUT"

echo "Building $OUTPUT..."

# TODO Temp skip CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH go build -C "$ROOT_DIR" -mod=vendor -trimpath -o "$FILE" -ldflags "-s -w -X main.Version=$VERSION_CLEAN"

#################
# BUILD WINDOWS #
#################

OS="windows"
EXT=".exe"
OUTPUT="${PACKAGE_NAME}_${VERSION_CLEAN}-${OS}${EXT}"
FILE="$RELEASE_PATH/$OUTPUT"

echo "Building $OUTPUT..."

# TODO Temp skip CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH go build -C "$ROOT_DIR" -mod=vendor -trimpath -o "$FILE" -ldflags "-s -w -X main.Version=$VERSION_CLEAN -H windowsgui"

#############
# BUILD MAC # (CURRENTLY NOT SUPPORTED)
#############


#############################
# Removing Generated Assets #
#############################

rm -f "$ROOT_DIR"/*.syso

#################
# BUILD ANDROID #
#################

if [ "${BUILD_ANDROID:-false}" = "true" ]; then
    "$ROOT_DIR/scripts/build-android.sh"
fi
