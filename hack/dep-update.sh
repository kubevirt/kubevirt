#!/usr/bin/env bash

set -ex

export GO111MODULE=on

(
    cd staging/src/kubevirt.io/client-go
    go get $@ ./...
    go mod tidy
)

(
    cd staging/src/github.com/golang/glog
    go get $@ ./...
    go mod tidy
)

(
    cd staging/src/kubevirt.io/client-go/examples/listvms
    go get $@ ./...
    go mod tidy
)

go mod tidy
go mod vendor
