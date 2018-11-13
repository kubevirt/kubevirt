#!/bin/bash

set -e

path=${1:-./pkg/...}

go test -cover -v -coverprofile=.coverprofile $(go list ${path})
