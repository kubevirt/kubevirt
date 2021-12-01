#!/usr/bin/env bash

set -e

source hack/common.sh
source hack/config.sh

fail_if_cri_bin_missing
target=$1

if [ "${target}" = "build" ]; then
    build_func_tests_image
fi

if [ "${target}" = "push" ]; then
    ${KUBEVIRT_CRI} push ${docker_prefix}/tests:${docker_tag}
fi
