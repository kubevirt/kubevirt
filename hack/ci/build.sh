#!/usr/bin/env bash

# poor mans replacement for PULL_PULL_SHA provided by prow - use the second parent commit id from the merge commit
mkdir -p _out/
merge_commit=$(git --no-pager log -1 --merges --format=%H)
git --no-pager show ${merge_commit} --format=%P | tr -d '\n' | cut -d ' ' -f 2 >_out/PULL_PULL_SHA
PULL_PULL_SHA=$(cat _out/PULL_PULL_SHA)
export PULL_PULL_SHA

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
