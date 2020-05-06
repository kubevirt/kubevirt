#!/usr/bin/env bash

# bazel will fail if either HOME or USER are not set
HOME=$(pwd)
export HOME
USER='kubeadmin'
export USER

mkdir -p _out/
if [ -z ${PULL_PULL_SHA} ]; then
    # poor mans replacement for PULL_PULL_SHA provided by prow - use the second parent commit id from the merge commit
    merge_commit=$(git --no-pager log -1 --merges --format=%H)
    PULL_PULL_SHA=$(git --no-pager show ${merge_commit} --format=%P | tr -d '\n' | cut -d ' ' -f 2)
fi
echo ${PULL_PULL_SHA} >_out/PULL_PULL_SHA
export PULL_PULL_SHA

# build dump
CMD_OUT_DIR="$(pwd)/_out/cmd"
export CMD_OUT_DIR
mkdir -p "$CMD_OUT_DIR/dump/"
GOPROXY=off GOFLAGS=-mod=vendor go build -o "$CMD_OUT_DIR/dump/dump" ./cmd/dump
./hack/build-func-tests.sh

rm -rf _ci-configs/

# to avoid any permission problems we reset access rights recursively
chmod -R 777 .
