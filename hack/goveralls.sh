#!/usr/bin/env bash
set -eo pipefail

coverprofiles=$(cat | sed "s/ /,/g")

goveralls -v -debug -service=${CI_NAME} -coverprofile=${coverprofiles} -ignore=$(find -regextype posix-egrep -regex ".*generated_mock.*\.go|.*swagger_generated\.go|.*openapi_generated\.go" -printf "%P\n" | paste -d, -s)
