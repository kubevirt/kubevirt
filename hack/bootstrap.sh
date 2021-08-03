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

sandbox_root=${SANDBOX_DIR}/default/root
sandbox_hash="c435f4256302f2dca6a1613ec6816c20f75cee20"

function kubevirt::bootstrap::regenerate() {
    (
        if [ -f "${SANDBOX_DIR}/${sandbox_hash}" ]; then
            kubevirt::bootstrap::sandbox_config
            echo "Sandbox is up to date"
            return
        fi
        echo "Regenerating sandbox"
        cd ${KUBEVIRT_DIR}
        rm ${SANDBOX_DIR} -rf
        rm .bazeldnf/sandbox.bazelrc -f
        bazel run --config ${HOST_ARCHITECTURE} //rpm:sandbox_${1}

        local sha=$(kubevirt::bootstrap::sha256)
        sed -i "/^[[:blank:]]*sandbox_hash[[:blank:]]*=/s/=.*/=\"${sha}\"/" hack/bootstrap.sh
        touch ${SANDBOX_DIR}/${sha}

        kubevirt::bootstrap::sandbox_config
    )
}

function kubevirt::bootstrap::sandbox_config() {
    cat <<EOT >.bazeldnf/sandbox.bazelrc
build --sandbox_add_mount_pair=${sandbox_root}/usr/:/usr/
build --sandbox_add_mount_pair=${sandbox_root}/lib64:/lib64
build --sandbox_add_mount_pair=${sandbox_root}/lib:/lib
build --sandbox_add_mount_pair=${sandbox_root}/bin:/bin
EOT
}

function kubevirt::bootstrap::sha256() {
    (
        cd ${KUBEVIRT_DIR}
        sha256sum rpm/BUILD.bazel | head -c 40
    )
}

kubevirt::bootstrap::regenerate ${HOST_ARCHITECTURE}
