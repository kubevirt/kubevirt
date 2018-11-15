#!/bin/bash

set -e

./hack/coverage.sh
goveralls -service=travis-ci -coverprofile=.coverprofile -ignore=$(find -regextype posix-egrep -regex ".*generated_mock.*\.go|.*swagger_generated\.go|.*openapi_generated\.go" -printf "%P\n" | paste -d, -s)
