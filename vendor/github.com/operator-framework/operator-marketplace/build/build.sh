#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/build/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="marketplace-operator"
REPO_PATH="github.com/operator-framework/operator-marketplace/"
BUILD_PATH="${REPO_PATH}/cmd/manager"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
