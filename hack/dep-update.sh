#!/usr/bin/env bash

set -e

export GO111MODULE=on

(
    cd staging/src/kubevirt.io/client-go
    go mod tidy
)

(
    cd staging/src/github.com/golang/glog
    go mod tidy
)

(
    cd staging/src/kubevirt.io/client-go/examples/listvms
    go mod tidy
)

go mod tidy
go mod vendor
