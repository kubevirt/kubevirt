#!/usr/bin/env bash

set -eExuo pipefail

export WORKSPACE="${WORKSPACE:-$PWD}"
readonly ARTIFACTS_PATH="${ARTIFACTS-$WORKSPACE/exported-artifacts}"
mkdir -p "$ARTIFACTS_PATH"

trap "{ make cluster-down; cp -r _out/artifacts/conformance ${ARTIFACTS_PATH}; }" EXIT SIGINT SIGTERM SIGQUIT

make cluster-up
make cluster-sync
make conformance
