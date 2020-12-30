#!/usr/bin/env bash
set -e

source hack/common.sh

bazel coverage \
    --config=${ARCHITECTURE} \
    --features race \
    --test_output=errors -- //staging/src/kubevirt.io/client-go/... //pkg/... //cmd/...
