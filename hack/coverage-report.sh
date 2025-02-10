#!/usr/bin/env bash
set -e

# shellcheck disable=SC2046
bazel run //vendor/github.com/wadey/gocovmerge:gocovmerge -- $(cat | sed "s# # ${BUILD_WORKING_DIRECTORY}/#g" | sed "s#^#${BUILD_WORKING_DIRECTORY}/#") >coverprofile.dat
ARTIFACTS=${ARTIFACTS:-_out/artifacts}
mkdir -p ${ARTIFACTS}
if ! command -V covreport; then go install github.com/cancue/covreport@latest; fi
covreport -i coverprofile.dat -o "${ARTIFACTS}/coverage.html"
echo "coverage report written to ${ARTIFACTS}/coverage.html"
