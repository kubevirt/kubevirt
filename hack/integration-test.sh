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
# Copyright 2023 Red Hat, Inc.
#

set -e

INTEG_TEST_IMAGE=${INTEG_TEST_IMAGE:-"quay.io/kubevirt/builder:2306271234-e00d9fcf9"}
PODMAN_SOCKET=${PODMAN_SOCKET:-"/run/podman/podman.sock"}

detect_podman_socket() {
    if curl --unix-socket "${PODMAN_SOCKET}" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
        echo "${PODMAN_SOCKET}"
    fi
}

if [ "${CONTAINER_CLIENT}" = "podman" ]; then
    _cri_bin="podman --remote --url=unix://$(detect_podman_socket)"
elif [ "${CONTAINER_CLIENT}" = "docker" ]; then
    _cri_bin=docker
else
    _cri_socket=$(detect_podman_socket)
    if [ -n "$_cri_socket" ]; then
        _cri_bin="podman --remote --url=unix://$_cri_socket"
        echo >&2 "selecting podman as container runtime"
    elif docker ps >/dev/null 2>&1; then
        _cri_bin=docker
        echo >&2 "selecting docker as container runtime"
    else
        echo >&2 "no working container runtime found. Neither docker nor podman seems to work."
        exit 1
    fi
fi

gocachemount="-v ${HOME}/.cache/go-build:/root/.cache/go-build"
test -t 1 && USE_TTY="-it"
# Unfortunately, `--cap-add NET_ADMIN` is not enough to replace `--privileged`.
# Some of the /proc/sys/net fields are read-only:
# open /proc/sys/net/ipv4/conf/testDummy99/route_localnet: read-only file system
_cli="${_cri_bin} run --privileged --rm ${USE_TTY} ${gocachemount} -v ./:/workspace:Z -w /workspace ${INTEG_TEST_IMAGE}"

$_cli go test -v \
    /workspace/pkg/network/driver/nmstate/... \
    --run-integration-tests \
    ${NULL}
