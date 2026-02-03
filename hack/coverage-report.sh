#!/usr/bin/env bash
set -e

# Coverage report using native Go tools
# Merge coverage profiles and generate HTML report

source hack/common.sh
source hack/config.sh

ARTIFACTS=${ARTIFACTS:-_out/artifacts}
mkdir -p ${ARTIFACTS}

# Install gocovmerge if needed
if ! command -V gocovmerge &>/dev/null; then
    go install github.com/wadey/gocovmerge@latest
fi

# Merge coverage files from stdin
gocovmerge $(cat | tr '\n' ' ') > coverprofile.dat

# Install covreport if needed
if ! command -V covreport &>/dev/null; then
    go install github.com/cancue/covreport@latest
fi

covreport -i coverprofile.dat -o "${ARTIFACTS}/coverage.html"
echo "coverage report written to ${ARTIFACTS}/coverage.html"
