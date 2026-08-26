#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RELEASE_PATH="$ROOT_DIR/releases"
PACKAGE_NAME="rockpload"
VERSION_FILE="$ROOT_DIR/VERSION"
APP_ID="gg.lexore.rockpload"
APP_NAME="Rockpload"
IS_SIGNED="${ROCKPLOAD_SIGNED:-true}"

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

IFS='.' read -r -a VERSION_PARTS <<< "$ANDROID_APP_VERSION"
VERSION_MAJOR="${VERSION_PARTS[0]:-0}"
VERSION_MINOR="${VERSION_PARTS[1]:-0}"
VERSION_PATCH="${VERSION_PARTS[2]:-0}"
if [ "$VERSION_MINOR" -ge 100 ] || [ "$VERSION_PATCH" -ge 100 ]; then
    echo "ANDROID_APP_VERSION=$ANDROID_APP_VERSION has a minor or patch >= 100." >&2
    echo "The versionCode formula (major*10000 + minor*100 + patch) can't represent that without colliding with the next major/minor. Widen the formula before releasing." >&2
    exit 1
fi
DEFAULT_APP_BUILD=$((VERSION_MAJOR * 10000 + VERSION_MINOR * 100 + VERSION_PATCH))
APP_BUILD="${ANDROID_APP_BUILD:-$DEFAULT_APP_BUILD}"
OUTPUT="$RELEASE_PATH/${PACKAGE_NAME}_${VERSION_CLEAN}-android.apk"

mkdir -p "$RELEASE_PATH"
rm -f "$ROOT_DIR"/*.apk "$OUTPUT"

echo "Building Android APK $OUTPUT..."

if [ "$IS_SIGNED" = "true" ]; then
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

    APK_FILE="$(find "$ROOT_DIR" -maxdepth 1 -name '*.aab' -print -quit)"
    if [ -z "$APK_FILE" ]; then
        echo "Fyne completed but no AAB was found in $ROOT_DIR"
        exit 1
    fi
else
    (
        cd "$ROOT_DIR"
        fyne package \
            -os android \
            -appID "$APP_ID" \
            -name "$APP_NAME" \
            -icon app/assets/logo.png \
            -appVersion "$ANDROID_APP_VERSION" \
            -appBuild "$APP_BUILD" \
            -metadata "rockploadVersion=$VERSION_CLEAN"
    )

    APK_FILE="$(find "$ROOT_DIR" -maxdepth 1 -name '*.apk' -print -quit)"
    if [ -z "$APK_FILE" ]; then
        echo "Fyne completed but no APK was found in $ROOT_DIR"
        exit 1
    fi

    mv "$APK_FILE" "$OUTPUT"
fi

echo "Android AAB created: $APK_FILE"
