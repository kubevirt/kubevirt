#!/usr/bin/env bash

set -e

source hack/common.sh

path="./pkg/... ./vendor/kubevirt.io/client-go/..."
coverpkg="../kubevirt/pkg/...,../kubevirt/vendor/kubevirt.io/client-go/..."
profile=.coverprofile

go test -cover -covermode=atomic -coverpkg="${coverpkg}" -v -coverprofile=${profile}.tmp ${path}
cat ${profile}.tmp | grep -v generated >${profile}
go tool cover -html=$profile -o ${OUT_DIR}/coverage.html
rm -f ${profile}.tmp
