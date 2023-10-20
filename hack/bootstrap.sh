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
# Copyright 2021 Red Hat, Inc.
#
set -e

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    source hack/common.sh
    source hack/config.sh
fi
KUBEVIRT_NO_BAZEL=${KUBEVIRT_NO_BAZEL:-false}
HOST_ARCHITECTURE="$(uname -m)"

sandbox_root=${SANDBOX_DIR}/default/root
sandbox_hash="1314bca8ef513d8c0ad742ea1c479aa51bc81880"

function kubevirt::bootstrap::regenerate() {
    (
        if kubevirt::bootstrap::sandbox_exists; then
            kubevirt::bootstrap::sandbox_config
            echo "Sandbox is up to date"
            return
        fi
        echo "Regenerating sandbox"
        cd ${KUBEVIRT_DIR}
        rm ${SANDBOX_DIR} -rf
        rm .bazeldnf/sandbox.bazelrc -f
        # Run gazelle to ensure that nogo has all build files resolved and that we can bootstrap the env.
        # This is necessary since some steps remove the vendor build files and nogo would be broken then.
        KUBEVIRT_BOOTSTRAPPING=true bazel run --config=${ARCHITECTURE} //:gazelle -- -exclude vendor/google.golang.org/grpc --exclude cluster-up
        KUBEVIRT_BOOTSTRAPPING=true bazel run --config ${HOST_ARCHITECTURE} //rpm:sandbox_${1}
        bazel clean

        local sha=$(kubevirt::bootstrap::sha256)
        sed -i "/^[[:blank:]]*sandbox_hash[[:blank:]]*=/s/=.*/=\"${sha}\"/" hack/bootstrap.sh
        touch ${SANDBOX_DIR}/${sha}

        kubevirt::bootstrap::sandbox_config
    )
}

function kubevirt::bootstrap::sandbox_exists() {
    ls ${SANDBOX_DIR}/${sandbox_hash} >/dev/null 2>&1
}

function kubevirt::bootstrap::sandbox_config() {
    cat <<EOT >.bazeldnf/sandbox.bazelrc
build --sandbox_add_mount_pair=${sandbox_root}/usr/:/usr/
build --sandbox_add_mount_pair=${sandbox_root}/lib64:/lib64
build --sandbox_add_mount_pair=${sandbox_root}/lib:/lib
build --sandbox_add_mount_pair=${sandbox_root}/bin:/bin

build --incompatible_enable_cc_toolchain_resolution --platforms=//bazel/platforms:x86_64-none-linux-gnu
EOT
}

function kubevirt::bootstrap::sha256() {
    (
        cd ${KUBEVIRT_DIR}
        sha256sum rpm/BUILD.bazel | head -c 40
    )
}

if [ "${KUBEVIRT_NO_BAZEL}" != "true" ]; then
    kubevirt::bootstrap::regenerate ${HOST_ARCHITECTURE}
fi
