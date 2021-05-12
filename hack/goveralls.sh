#!/usr/bin/env bash
set -eo pipefail

bazel run //vendor/github.com/wadey/gocovmerge:gocovmerge -- $(cat | sed "s# # ${BUILD_WORKING_DIRECTORY}/#g" | sed "s#^#${BUILD_WORKING_DIRECTORY}/#") >coverprofile.dat
goveralls -service=${CI_NAME} -jobid=${PROW_JOB_ID} -coverprofile=coverprofile.dat -ignore=$(find -regextype posix-egrep -regex ".*generated_mock.*\.go|.*swagger_generated\.go|.*openapi_generated\.go" -printf "%P\n" | paste -d, -s)
