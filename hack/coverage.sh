#!/usr/bin/env bash
set -e

source hack/common.sh
source hack/config.sh

# Run coverage tests using native Go

WHAT=${WHAT:-./staging/src/kubevirt.io/client-go/... ./pkg/... ./cmd/...}

echo "Running coverage tests..."

# Create coverage output directory
mkdir -p _out/coverage

# Run Go tests with coverage
go test \
    -race \
    -coverprofile=_out/coverage/coverage.out \
    -covermode=atomic \
    ${WHAT}

echo "Coverage profile written to _out/coverage/coverage.out"
