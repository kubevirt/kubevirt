#!/usr/bin/env bash
set -eo pipefail

# Goveralls using native Go tools
# Merge coverage and submit to coveralls

source hack/common.sh
source hack/config.sh

# Install gocovmerge if needed
if ! command -V gocovmerge &>/dev/null; then
    go install github.com/wadey/gocovmerge@latest
fi

# Merge coverage files from stdin
gocovmerge $(cat | tr '\n' ' ') > coverprofile.dat

# Submit to goveralls
goveralls -service=${CI_NAME} -jobid=${PROW_JOB_ID} -coverprofile=coverprofile.dat -ignore=$(find -regextype posix-egrep -regex ".*[^/]*(generated[^/]*|\.pb)\.go" -printf "%P\n" | paste -d, -s)
