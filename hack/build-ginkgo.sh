#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh

rm -rf "${TESTS_OUT_DIR}"
mkdir -p "${TESTS_OUT_DIR}/tools"

# Build ginkgo using native Go
GOOS=linux GOARCH=$(go env GOARCH) go build -o "${TESTS_OUT_DIR}/ginkgo" vendor/github.com/onsi/ginkgo/v2/ginkgo/main.go
