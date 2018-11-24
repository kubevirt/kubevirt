#!/bin/bash

set -e

path=${1:-./pkg/...}
profile=.coverprofile

go test -cover -v -coverprofile=$profile $(go list ${path})
go tool cover -html=$profile -o coverage.html
