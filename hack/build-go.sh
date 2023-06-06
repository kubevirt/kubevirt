#!/usr/bin/env bash
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
source hack/version.sh

source hack/go-build-functests.sh

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

PLATFORM=$(uname -m)
case ${PLATFORM} in
x86_64* | i?86_64* | amd64*)
    ARCH="amd64"
    ;;
aarch64* | arm64*)
    ARCH="arm64"
    ;;
*)
    echo "invalid Arch, only support x86_64 and aarch64"
    exit 1
    ;;
esac

# forward all commands to all packages if no specific one was requested
# TODO finetune this a little bit more
if [ $# -eq 0 ]; then
    if [ "${target}" = "test" ]; then
        (
            # Ignoring container-disk-v2alpha since it is written in C, not in go
            go ${target} -v -tags "${KUBEVIRT_GO_BUILD_TAGS}" --ignore=container-disk-v2alpha ./cmd/...
        )
        (
            go ${target} -v -tags "${KUBEVIRT_GO_BUILD_TAGS}" -race ./pkg/...
        )
    else
        (
            go $target -tags "${KUBEVIRT_GO_BUILD_TAGS}" ./pkg/...
            GO111MODULE=off go $target -tags "${KUBEVIRT_GO_BUILD_TAGS}" ./staging/src/kubevirt.io/...
        )
        (
            go $target -tags "${KUBEVIRT_GO_BUILD_TAGS}" ./tests/...
        )
    fi
fi

# Return a pkgdir parameter based on os and arch
function pkg_dir() {
    if [ -n "${KUBEVIRT_GO_BASE_PKGDIR}" ]; then
        echo "-pkgdir ${KUBEVIRT_GO_BASE_PKGDIR}/$1-$2"
    fi
}

# handle binaries

if [ "${target}" = "install" ]; then
    # Delete all binaries which are not present in the binaries variable to avoid branch inconsistencies
    to_delete=$(comm -23 <(find ${CMD_OUT_DIR} -mindepth 1 -maxdepth 1 -type d | sort) <(echo $binaries | sed -e 's/cmd\///g' -e 's/ /\n/g' | sed -e "s#^#${CMD_OUT_DIR}/#" | sort))
    rm -rf ${to_delete}

    (
        if [ -z "$BIN_NAME" ] || [[ $BIN_NAME == *"container-disk"* ]]; then
            mkdir -p ${CMD_OUT_DIR}/container-disk-v2alpha
            cd cmd/container-disk-v2alpha
            # the containerdisk bianry needs to be static, as it runs in a scratch container
            echo "building static binary container-disk"
            gcc -static -o ${CMD_OUT_DIR}/container-disk-v2alpha/container-disk main.c
        fi
    )
fi

for arg in $args; do
    if [ "${target}" = "test" ]; then
        (
            go ${target} -v -tags "${KUBEVIRT_GO_BUILD_TAGS}" ./$arg/...
        )
    elif [ "${target}" = "install" ]; then
        eval "$(go env)"
        BIN_NAME=$(basename $arg)
        ARCH_BASENAME=${BIN_NAME}-${KUBEVIRT_VERSION}
        mkdir -p ${CMD_OUT_DIR}/${BIN_NAME}
        (
            go vet ./$arg/...

            cd $arg

            # always build and link the binary based on CPU Architecture
            LINUX_NAME=${ARCH_BASENAME}-linux-${ARCH}

            echo "building dynamic binary $BIN_NAME"
            GOOS=linux GOARCH=${ARCH} go_build -tags "${KUBEVIRT_GO_BUILD_TAGS}" -o ${CMD_OUT_DIR}/${BIN_NAME}/${LINUX_NAME} -ldflags "$(kubevirt::version::ldflags)" $(pkg_dir linux ${ARCH})

            (cd ${CMD_OUT_DIR}/${BIN_NAME} && ln -sf ${LINUX_NAME} ${BIN_NAME})

            kubevirt::version::get_version_vars
            echo "$KUBEVIRT_GIT_VERSION" >${CMD_OUT_DIR}/${BIN_NAME}/.version

            # build virtctl for all architectures if requested
            if [ "${BIN_NAME}" = "virtctl" -a "${KUBEVIRT_RELEASE}" = "true" ]; then
                for arch in amd64 arm64; do
                    for os in linux darwin windows; do
                        if [ "${os}" = "windows" ]; then
                            extension=".exe"
                        else
                            extension=""
                        fi

                        GOOS=${os} GOARCH=${arch} go_build -tags "${KUBEVIRT_GO_BUILD_TAGS}" -o ${CMD_OUT_DIR}/${BIN_NAME}/${ARCH_BASENAME}-${os}-${arch}${extension} -ldflags "$(kubevirt::version::ldflags)" $(pkg_dir ${os} ${arch})
                        # Create symlinks to the latest binary
                        (cd ${CMD_OUT_DIR}/${BIN_NAME} && ln -sf ${ARCH_BASENAME}-${os}-${arch}${extension} ${BIN_NAME}-${os}-${arch}${extension})
                    done
                done
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
