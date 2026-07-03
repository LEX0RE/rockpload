#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_PATH="$ROOT_DIR/releases"
PACKAGE_NAME="rockpload"
VERSION_FILE="$ROOT_DIR/VERSION"
APP_ID="gg.lexore.rockpload"
APP_NAME="Rockpload"

if ! command -v fyne >/dev/null 2>&1; then
    echo "Missing fyne CLI. Install it with:"
    echo "  go install fyne.io/tools/cmd/fyne@latest"
    exit 1
fi

if [ -n "${ROCKPLOAD_VERSION:-}" ]; then
    VERSION="$ROCKPLOAD_VERSION"
elif [ -f "$VERSION_FILE" ]; then
    VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
else
    VERSION="dev"
fi

VERSION="${VERSION#v}"
VERSION_CLEAN=$(echo "$VERSION" | sed 's/\//-/g')

IOS_APP_VERSION="$VERSION_CLEAN"
if [[ "$IOS_APP_VERSION" == *-* ]]; then
    IOS_APP_VERSION="${IOS_APP_VERSION%%-*}"
fi
if [[ ! "$IOS_APP_VERSION" =~ ^[0-9]+(\.[0-9]+){0,2}$ ]]; then
    IOS_APP_VERSION="0.0.1"
fi
APP_BUILD="${IOS_APP_BUILD:-1}"

mkdir -p "$RELEASE_PATH"

echo "Building iOS App..."

(
    cd "$ROOT_DIR"
    fyne package \
        -os iossimulator \
        -appID "$APP_ID" \
        -name "$APP_NAME" \
        -icon app/assets/logo.png \
        -appVersion "$IOS_APP_VERSION" \
        -appBuild "$APP_BUILD" \
        -certificate "" \
        -profile ""
)

APP_FOLDER="$(find "$ROOT_DIR" -maxdepth 1 -name '*.app' -print -quit)"
if [ -z "$APP_FOLDER" ]; then
    echo "Fyne completed but no .app folder was found in $ROOT_DIR"
    exit 1
fi

IPA_NAME="${PACKAGE_NAME}_${VERSION_CLEAN}-ios.ipa"
OUTPUT="$RELEASE_PATH/$IPA_NAME"

echo "Creating IPA for sideloading: $OUTPUT"
cd "$ROOT_DIR"
rm -rf Payload
mkdir Payload

find "$APP_FOLDER" -maxdepth 3

mv "$APP_FOLDER" Payload/
zip -qr "$OUTPUT" Payload
rm -rf Payload

echo "iOS build completed: $OUTPUT"