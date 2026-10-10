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
    go mod tidy
)
(
    echo $_sync_only
    cd staging/src/kubevirt.io/client-go
    if [ "${_sync_only}" == "false" ]; then go get $@ ./...; fi
    go mod tidy
)

go mod tidy
go work vendor

# Go 1.27 "go work sync" writes the workspace-selected k8s.io/client-go version,
# the root go.mod placeholder (v12.0.0+incompatible), into the staging module.
# That placeholder only works through the root replace, so restore the version
# that replace points at, then re-tidy and re-vendor so vendor/modules.txt
# matches the final go.mod files. (The version must be read before the sync,
# while vendor/modules.txt is still consistent.)
client_go_version="$(go list -m -f "{{.Replace.Version}}" k8s.io/client-go)"
go work sync
(
    cd staging/src/kubevirt.io/client-go
    go mod edit -require="k8s.io/client-go@${client_go_version}"
    go mod tidy
)
go work vendor
