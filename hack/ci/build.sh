#!/usr/bin/env bash

export DOCKER_PREFIX='kubevirtnightlybuilds'
export DOCKER_TAG="latest"
export KUBEVIRT_PROVIDER=external

bash -x ./hack/build-manifests.sh

# build dump
CMD_OUT_DIR="$(pwd)/_out/cmd"
export CMD_OUT_DIR
mkdir -p "$CMD_OUT_DIR/dump/"
GOPROXY=off GOFLAGS=-mod=vendor go build -o "$CMD_OUT_DIR/dump/dump" ./cmd/dump
bash -x ./hack/build-func-tests.sh

rm -rf _ci-configs/

# to avoid any permission problems we reset access rights recursively
chmod -R 777 .
