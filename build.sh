#!/bin/bash

RELEASE_PATH="releases"
ARCH="amd64"
PACKAGE_NAME="rockpload"

rm -rf $RELEASE_PATH
mkdir -p $RELEASE_PATH

VERSION="1.0.0"
echo "Building version $VERSION"

###############
# BUILD LINUX #
###############

OS="linux"
OUTPUT="${PACKAGE_NAME}_${VERSION}-${OS}"
FILE="$RELEASE_PATH/$OUTPUT"

echo "Building $OUTPUT..."

CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH go build -o "$FILE" -ldflags "-X main.Version=$VERSION"

#################
# BUILD WINDOWS #
#################

OS="windows"
EXT=".exe"
OUTPUT="${PACKAGE_NAME}_${VERSION}-${OS}${EXT}"
FILE="$RELEASE_PATH/$OUTPUT"

echo "Building $OUTPUT..."

CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH go build -o "$FILE" -ldflags "-X main.Version=$VERSION -H windowsgui"

#############
# BUILD MAC # (CURRENTLY NOT SUPPORTED)
#############