#!/usr/bin/env bash

set -e

path=${1:-./pkg/...}
profile=.coverprofile

packages=$(go list ${path})

# in rare cases go test with coverage seems to swallow test
# failure output, so we keep the exit code
set +e
go test -cover -v -coverprofile=$profile ${packages}
return_value=$?

# in case of exit code we explicitly test
# each package again to have the output of all
# failing tests
if [[ $return_value -ne 0 ]]; then
    for package in ${packages}; do
        go test ${package}
    done
    exit $return_value
fi

set -e
go tool cover -html=$profile -o coverage.html
