#!/bin/bash
set -euo pipefail

export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_PATH="$ROOT_DIR/releases"
PACKAGE_NAME="rockpload"
VERSION_FILE="$ROOT_DIR/VERSION"
APP_ID="gg.lexore.rockpload"
APP_NAME="Rockpload"
KEYSTORE_ENV_FILE="$ROOT_DIR/keystore/.env"

if ! command -v fyne-cross >/dev/null 2>&1; then
    echo "Missing fyne-cross. Install it with:"
    echo "  go install github.com/fyne-io/fyne-cross@latest"
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

if ls "$ROOT_DIR/$RELEASE_PATH/"*.apk >/dev/null 2>&1; then
    rm -f "$ROOT_DIR/$RELEASE_PATH/"*.apk "$OUTPUT"
fi
rm -rf "$ROOT_DIR/fyne-cross/"

echo "Building Android APK $OUTPUT..."

if [ -f "$KEYSTORE_ENV_FILE" ]; then
    echo "Loading secrets from $KEYSTORE_ENV_FILE"
    export $(grep -v '^#' "$KEYSTORE_ENV_FILE" | xargs)
else
    echo "Warning: $KEYSTORE_ENV_FILE not found, skipping."
fi

echo "$ROCKPLOAD_KEYSTORE_PATH"

(
    cd "$ROOT_DIR"
    CGO_ENABLED=1 fyne-cross \
        android \
        -env "GOTOOLCHAIN=auto" \
        -app-id "$APP_ID" \
        -name "$APP_NAME" \
        -icon app/assets/logo.png \
        -app-version "$ANDROID_APP_VERSION" \
        -app-build "$APP_BUILD" \
        -metadata "rockploadVersion=$VERSION_CLEAN"
)

APK_FILE="$(find "$ROOT_DIR/fyne-cross" -maxdepth 3 -name '*.apk' -print -quit)"
if [ -z "$APK_FILE" ]; then
    echo "Fyne-Cross completed but no APK was found in $ROOT_DIR"
    exit 1
fi

mv "$APK_FILE" "$OUTPUT"
rm -rf "$ROOT_DIR/fyne-cross/"
echo "Android APK created: $OUTPUT"

echo "--- Android Installation ---"
if command -v adb >/dev/null 2>&1; then
    if adb devices | grep -E -v 'List of devices attached|^$' >/dev/null; then
        echo "Android device/emulator detected. Installing APK automatically..."
        
        adb install -r "$OUTPUT"
        
        echo "Installation completed."
        echo "To view logs run: adb logcat | grep -i 'component=rockpload'"
    else
        echo "No Android device connected."
        echo "To install manually: adb install $OUTPUT"
        echo "To view logs: adb logcat | grep -i 'component=rockpload'"
    fi
else
    echo "adb command not found."
    echo "To install manually: adb install $OUTPUT"
    echo "To view logs: adb logcat | grep -i 'component=rockpload'"
fi