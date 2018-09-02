#!/usr/bin/env bash
set -e

go test -cover -v -coverprofile=.coverprofile ./pkg/...
goveralls -service=travis-ci -coverprofile=.coverprofile -ignore=$(find -regextype posix-egrep -regex ".*generated_mock.*\.go|.*swagger_generated\.go|.*openapi_generated\.go" -printf "%P\n" | paste -d, -s)
