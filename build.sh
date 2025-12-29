#!/bin/bash
set -e

PROJECT_DIR="./"
BUILD_DIR="$PROJECT_DIR/build"

mkdir -p "$BUILD_DIR"

# Run all tests
go test ./...

# Build the Linux binary
pushd "$PROJECT_DIR/cmd/server/" > /dev/null
GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/gtm-linux"
popd > /dev/null

echo "Build complete. Binaries are in $BUILD_DIR"
