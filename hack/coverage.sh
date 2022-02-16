#!/usr/bin/env bash
set -e

source hack/common.sh
source hack/bootstrap.sh

if [ "${CI}" == "true" ]; then
    cat >>ci.bazelrc <<EOF
coverage --cache_test_results=no --runs_per_test=1
EOF
fi

bazel coverage \
    --config=${ARCHITECTURE} \
    --features race \
    --test_output=errors -- //staging/src/kubevirt.io/client-go/... //pkg/... //cmd/...
