#!/usr/bin/env bash

set -ex

export GO111MODULE=on
export _sync_only="false"

while true; do
    case "$1" in
    -s | --sync-only)
        _sync_only="true"
        shift 1
        ;;
    --)
        shift
        break
        ;;
    *) break ;;
    esac
done

(
    echo $_sync_only
    cd staging/src/kubevirt.io/api
    if [ "${_sync_only}" == "false" ]; then go get $@ ./...; fi
    # remove compat=1.17 when we move to go 1.18
    go mod tidy -compat=1.17
)
(
    echo $_sync_only
    cd staging/src/kubevirt.io/client-go
    if [ "${_sync_only}" == "false" ]; then go get $@ ./...; fi
    # remove compat=1.17 when we move to go 1.18
    go mod tidy -compat=1.17
)

(
    cd staging/src/github.com/golang/glog
    if [ "${_sync_only}" == "false" ]; then go get $@ ./...; fi
    # remove compat=1.17 when we move to go 1.18
    go mod tidy -compat=1.17
)

(
    cd staging/src/kubevirt.io/client-go/examples/listvms
    if [ "${_sync_only}" == "false" ]; then go get $@ ./...; fi
    # remove compat=1.17 when we move to go 1.18
    go mod tidy -compat=1.17
)

# remove compat=1.17 when we move to go 1.18
go mod tidy -compat=1.17
go mod vendor
