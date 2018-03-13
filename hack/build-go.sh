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
    target="install"
else
    target=$1
    shift
fi

if [ $# -eq 0 ]; then
    args=$binaries
    build_tests="true"
else
    args=$@
fi

# forward all commands to all packages if no specific one was requested
# TODO finetune this a little bit more
if [ $# -eq 0 ]; then
    if [ "${target}" = "test" ]; then
        (
            cd ${KUBEVIRT_DIR}/pkg
            go ${target} -v ./...
        )
    else
        (
            cd ${KUBEVIRT_DIR}/pkg
            go $target ./...
        )
        (
            cd ${KUBEVIRT_DIR}/tests
            go $target ./...
        )
    fi
fi

# handle binaries

if [ "${target}" = "install" ]; then
    rm -rf ${CMD_OUT_DIR}
fi

for arg in $args; do
    if [ "${target}" = "test" ]; then
        (
            cd $arg
            go ${target} -v ./...
        )
    elif [ "${target}" = "install" ]; then
        eval "$(go env)"
        BIN_NAME=$(basename $arg)
        ARCH_BASENAME=${BIN_NAME}-$(git describe --always --tags)
        ARCHBIN=${ARCH_BASENAME}-${GOHOSTOS}-${GOHOSTARCH}
        mkdir -p ${CMD_OUT_DIR}/${BIN_NAME}
        (
            cd $arg
            go vet ./...
            go build -o ${CMD_OUT_DIR}/${BIN_NAME}/${ARCHBIN}
            (cd ${CMD_OUT_DIR}/${BIN_NAME} && ln -sf ${ARCHBIN} ${BIN_NAME})

            # build virtctl binaries for other operating systems
            if [ "${BIN_NAME}" = "virtctl" ]; then
                if [[ "${GOHOSTOS}" != "linux" ]] || [[ "${GOHOSTARCH}" != "amd64" ]]; then
                    GOOS=linux GOARCH=amd64 go build -o ${CMD_OUT_DIR}/${BIN_NAME}/${ARCH_BASENAME}-linux-amd64
                fi
                if [[ "${GOHOSTOS}" != "darwin" ]] || [[ "${GOHOSTARCH}" != "amd64" ]]; then
                    GOOS=darwin GOARCH=amd64 go build -o ${CMD_OUT_DIR}/${BIN_NAME}/${ARCH_BASENAME}-darwin-amd64
                fi
                if [[ "${GOHOSTOS}" != "windows" ]] || [[ "${GOHOSTARCH}" != "amd64" ]]; then
                    GOOS=windows GOARCH=amd64 go build -o ${CMD_OUT_DIR}/${BIN_NAME}/${ARCH_BASENAME}-windows-amd64.exe
                fi
            fi
        )
    else
        (
            cd $arg
            go $target ./...
        )
    fi
done

if [[ "${target}" == "install" && "${build_tests}" == "true" ]]; then
    build_func_tests
fi
