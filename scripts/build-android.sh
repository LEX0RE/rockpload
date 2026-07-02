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
ANDROID_APP_VERSION="$VERSION_CLEAN"
if [[ "$ANDROID_APP_VERSION" == *-* ]]; then
    ANDROID_APP_VERSION="${ANDROID_APP_VERSION%%-*}"
fi
if [[ ! "$ANDROID_APP_VERSION" =~ ^[0-9]+(\.[0-9]+){0,2}$ ]]; then
    ANDROID_APP_VERSION="0.0.1"
fi
APP_BUILD="${ANDROID_APP_BUILD:-1}"
OUTPUT="$RELEASE_PATH/${PACKAGE_NAME}_${VERSION_CLEAN}-android.apk"

mkdir -p "$RELEASE_PATH"
rm -f "$ROOT_DIR"/*.apk "$OUTPUT"

echo "Building Android APK $OUTPUT..."

(
    cd "$ROOT_DIR"
    fyne release \
        -os android \
        -appID "$APP_ID" \
        -name "$APP_NAME" \
        -icon app/assets/logo.png \
        -appVersion "$ANDROID_APP_VERSION" \
        -appBuild "$APP_BUILD" \
        -metadata "rockploadVersion=$VERSION_CLEAN" \
        -keyStore "keystore/rockpload.keystore" \
        -keyStorePass "$ROCKPLOAD_KEYSTORE_PASS" \
        -keyPass "$ROCKPLOAD_KEYSTORE_PASS" \
        -keyName "$ROCKPLOAD_KEY_ALIAS"
)

APK_FILE="$(find "$ROOT_DIR" -maxdepth 1 -name '*.apk' -print -quit)"
if [ -z "$APK_FILE" ]; then
    echo "Fyne completed but no APK was found in $ROOT_DIR"
    exit 1
fi

mv "$APK_FILE" "$OUTPUT"
echo "Android APK created: $OUTPUT"
