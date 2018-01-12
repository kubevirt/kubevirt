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

source hack/config.sh
KUBEVIRT_DIR="$(
    cd "$(dirname "$0")/../"
    pwd
)"
OUT_DIR=$KUBEVIRT_DIR/_out/cmd/

if [ -z "$1" ]; then
    target="build"
else
    target=$1
    shift
fi

if [ $# -eq 0 ]; then
    args=$docker_images
elif [ "$1" = "optional" ]; then
    args=$optional_docker_images
else
    args=$@
fi

for arg in $args; do
    if [ "${target}" = "build" ]; then
        BIN_NAME=$(basename $arg)
        rsync -avz --exclude "**/*.md" --exclude "**/*.go" --exclude "**/.*" $arg/ ${OUT_DIR}/${BIN_NAME}/
        # TODO the version of docker we're using in our vagrant dev environment
        # does not support using ARGS in FROM field.
        # https://docs.docker.com/engine/reference/builder/#understand-how-arg-and-from-interact
        # Because of this we have to manipulate the Dockerfile for kubevirt containers
        # that depend on other kubevirt containers.
        cat $arg/Dockerfile | sed s/registry-disk-v1alpha/registry-disk-v1alpha\:$docker_tag/g >${OUT_DIR}/${BIN_NAME}/Dockerfile
        (
            cd ${OUT_DIR}/${BIN_NAME}/
            docker $target -t ${docker_prefix}/${BIN_NAME}:${docker_tag} .
        )
    elif [ "${target}" = "push" ]; then
        (
            BIN_NAME=$(basename $arg)
            cd ${OUT_DIR}/${BIN_NAME}/
            docker push ${docker_prefix}/${BIN_NAME}:${docker_tag}
        )
    fi
done
