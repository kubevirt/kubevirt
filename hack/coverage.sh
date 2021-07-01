#!/usr/bin/env bash
set -e

source hack/common.sh

if [ "${CI}" == "true" ]; then
    cat >>ci.bazelrc <<EOF
coverage --cache_test_results=no --runs_per_test=1
EOF
fi

bazel coverage \
    --config=${ARCHITECTURE} \
    --@io_bazel_rules_go//go/config:race \
    --test_output=errors -- //staging/src/kubevirt.io/client-go/... //pkg/... //cmd/... -//cmd/virtctl/...
