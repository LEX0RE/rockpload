#!/bin/bash

RELEASE_PATH="releases"
ARCH="amd64"
PACKAGE_NAME="rockpload"

rm -rf $RELEASE_PATH
mkdir -p $RELEASE_PATH

VERSION=${ROCKPLOAD_VERSION:-"1.1.1"}
VERSION_CLEAN=$(echo $VERSION | sed 's/\//-/g')
echo "Building version $VERSION_CLEAN"

###############
# BUILD LINUX #
###############

OS="linux"
OUTPUT="${PACKAGE_NAME}_${VERSION_CLEAN}-${OS}"
FILE="$RELEASE_PATH/$OUTPUT"

echo "Building $OUTPUT..."

CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH go build -trimpath -o "$FILE" -ldflags "-s -w -X main.Version=$VERSION_CLEAN"

#################
# BUILD WINDOWS #
#################

OS="windows"
EXT=".exe"
OUTPUT="${PACKAGE_NAME}_${VERSION_CLEAN}-${OS}${EXT}"
FILE="$RELEASE_PATH/$OUTPUT"

echo "Building $OUTPUT..."

CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH go build -trimpath -o "$FILE" -ldflags "-s -w -X main.Version=$VERSION_CLEAN -H windowsgui"

#############
# BUILD MAC # (CURRENTLY NOT SUPPORTED)
#############