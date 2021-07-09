#!/usr/bin/env bash

set -e

source hack/common.sh
source hack/config.sh

target=$1

if [ "${target}" = "build" ]; then
    build_func_tests_image
fi

if [ "${target}" = "push" ]; then
    docker push ${docker_prefix}/tests:${docker_tag}
fi
