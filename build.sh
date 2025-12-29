#!/bin/bash

set -e

PROJECT_DIR="./"
BUILD_DIR="$PROJECT_DIR/build"

mkdir -p $BUILD_DIR

go test ./...

pushd $PROJECT_DIR/cmd/server/
  GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/gtm-linux
popd