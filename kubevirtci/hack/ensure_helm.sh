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
# Copyright The KubeVirt Authors.

# Prints the path to a helm binary, downloading one under ${KUBEVIRTCI_CONFIG_PATH}/.tools if needed.
function ensure_helm() {
    if command -v helm >/dev/null 2>&1; then
        echo "Using helm binary: $(command -v helm)" >&2
        command -v helm
        return
    fi

    local version="${HELM_VERSION:-v3.16.4}"

    [[ "${version}" == v* ]] || version="v${version}"

    local os arch install_dir helm_path url tarball
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    case "${arch}" in
        x86_64) arch=amd64 ;;
        aarch64|arm64) arch=arm64 ;;
        *)
            echo "ERROR: unsupported architecture for helm: ${arch}" >&2
            return 1
            ;;
    esac

    install_dir="${KUBEVIRTCI_CONFIG_PATH}/.tools/helm-${version}"
    helm_path="${install_dir}/helm"
    if [ -x "${helm_path}" ]; then
        echo "${helm_path}"
        return
    fi

    mkdir -p "${install_dir}"
    url="https://get.helm.sh/helm-${version}-${os}-${arch}.tar.gz"
    tarball="${install_dir}/helm.tar.gz"
    if ! curl -fsSL -o "${tarball}" "${url}"; then
        echo "ERROR: failed to download helm from ${url}" >&2
        echo "       check that HELM_VERSION=${version} is a real release for ${os}-${arch}." >&2
        rm -f "${tarball}"
        return 1
    fi

    tar -xz -C "${install_dir}" -f "${tarball}" "${os}-${arch}/helm"
    mv "${install_dir}/${os}-${arch}/helm" "${helm_path}"
    rm -f "${tarball}"
    rmdir "${install_dir}/${os}-${arch}" 2>/dev/null || true
    echo "${helm_path}"
}
