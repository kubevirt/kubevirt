#!/usr/bin/env bash
set -e

source hack/common.sh
source hack/config.sh

# Run fuzz tests using native Go
# Note: Go native fuzz testing requires Go 1.18+

echo "Running fuzz tests..."

# Run Go fuzz tests with race detector
go test \
    -race \
    -fuzz=. \
    -fuzztime=30s \
    ./pkg/...

echo "Fuzz tests completed."
