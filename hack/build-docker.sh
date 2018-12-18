#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

set -e

source hack/common.sh
source hack/config.sh

if [ -z "$1" ]; then
    target="build"
else
    target=$1
    shift
fi

if [ $# -eq 0 ]; then
    args=$docker_images
    build_tests="true"
else
    args=$@
fi

for arg in $args; do
    BUILDER_EXTRA_ARGS=""
    BIN_NAME=$(basename $arg)
    if [ "${target}" = "build" ]; then
        (
            if [ -n "$KUBEVIRT_CACHE_FROM" ]; then
                BUILDER_EXTRA_ARGS="${BUILDER_EXTRA_ARGS} --cache-from kubevirt/${BIN_NAME}:${KUBEVIRT_CACHE_FROM}"
            fi
            if [ -n "$KUBEVIRT_UPDATE_CACHE_FROM" ]; then
                BUILDER_EXTRA_ARGS="${BUILDER_EXTRA_ARGS} -t kubevirt/${BIN_NAME}:${KUBEVIRT_UPDATE_CACHE_FROM}"
            fi
            cd ${CMD_OUT_DIR}/${BIN_NAME}/
            docker build -t ${docker_prefix}/${BIN_NAME}:${docker_tag} --label ${job_prefix} --label ${BIN_NAME} ${BUILDER_EXTRA_ARGS} .
        )
    elif [ "${target}" = "push" ]; then
        (
            cd ${CMD_OUT_DIR}/${BIN_NAME}/
            docker push ${docker_prefix}/${BIN_NAME}:${docker_tag}
        )
    fi
done

if [ $# -eq 0 ]; then
    args=$docker_images_cacheable
else
    args=$@
fi

if [ "${target}" = "push-cache" ]; then
    docker push kubevirt/builder-cache:${KUBEVIRT_UPDATE_CACHE_FROM}
    if [ -n "$KUBEVIRT_UPDATE_CACHE_FROM" ]; then
        for arg in $args; do
            BIN_NAME=$(basename $arg)
            docker push kubevirt/${BIN_NAME}:${KUBEVIRT_UPDATE_CACHE_FROM}
        done
    fi
fi

if [ "${target}" = "pull-cache" ]; then
    docker pull kubevirt/builder-cache:${KUBEVIRT_CACHE_FROM}
    if [ -n "$KUBEVIRT_CACHE_FROM" ]; then
        for arg in $args; do
            BIN_NAME=$(basename $arg)
            docker pull kubevirt/${BIN_NAME}:${KUBEVIRT_CACHE_FROM} || true
        done
    fi
fi

if [[ "${build_tests}" == "true" ]]; then
    if [[ "${target}" == "build" ]]; then
        build_func_tests_container
    fi
    if [[ "${target}" == "push" ]]; then
        cd ${TESTS_OUT_DIR}
        docker $target ${docker_prefix}/tests:${docker_tag}
    fi
fi
