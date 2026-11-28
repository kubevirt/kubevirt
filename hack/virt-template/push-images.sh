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
# Copyright The KubeVirt Authors.
#

# Mirror the pinned virt-template images into DOCKER_PREFIX.
# make push leaves BUILD_ARCH unset and copies every architecture.
# cluster-sync sets BUILD_ARCH and copies that one architecture.
# --preserve-digests keeps the upstream digest. A destination that already
# has that digest is left unchanged, so copying onto quay.io/kubevirt does
# not rewrite the upstream tag.

# Do not use nounset. hack/common.sh sources kubevirtci, which reads
# KUBEVIRTCI_PATH before assigning it.
set -eo pipefail

source hack/common.sh
source hack/config.sh
source hack/virt-template/default.sh

source_registry="quay.io/kubevirt"
virt_template_targets="virt-template-apiserver virt-template-controller"

# One architecture when BUILD_ARCH is a single value. A comma-separated list
# still copies the full manifest, because one tag cannot hold a subset.
single_oci_arch=""
if [[ -n ${BUILD_ARCH:-} && ${BUILD_ARCH} != *","* ]]; then
    if ! single_oci_arch="$(format_archname "${BUILD_ARCH}" tag)"; then
        echo "${single_oci_arch}" >&2
        exit 1
    fi
fi

dest_is_insecure() {
    local host="${docker_prefix%%/*}"
    case "${host}" in
    localhost | localhost:* | 127.0.0.1 | 127.0.0.1:* | registry | registry:*)
        return 0
        ;;
    *)
        return 1
        ;;
    esac
}

digest_for_target() {
    case "$1" in
    virt-template-apiserver)
        echo "${virt_template_apiserver_digest}"
        ;;
    virt-template-controller)
        echo "${virt_template_controller_digest}"
        ;;
    *)
        echo "unknown virt-template image: $1" >&2
        return 1
        ;;
    esac
}

platform_digest() {
    local target="$1"
    local oci_arch="$2"
    local digest

    digest="$(skopeo inspect --raw "docker://${source_registry}/${target}:${virt_template_version}" |
        jq -er --arg arch "${oci_arch}" '
            [ .manifests[]
              | select(.platform.architecture == $arch and .platform.os == "linux")
              | .digest
            ] | if length == 1 then .[0] else empty end')" || {
        echo "missing ${oci_arch} manifest for ${target}" >&2
        return 1
    }
    echo "${digest}"
}

copy_virt_template_image() {
    local target="$1"
    local repository_prefix="$2"
    local dest="${docker_prefix}/${repository_prefix}${target}:${virt_template_version}"
    local expected_digest
    local src
    local inspect_args=()
    local copy_args=(--preserve-digests)
    local existing

    if [[ -n ${single_oci_arch} ]]; then
        expected_digest="$(platform_digest "${target}" "${single_oci_arch}")"
    else
        expected_digest="$(digest_for_target "${target}")"
        copy_args+=(--all)
    fi
    src="${source_registry}/${target}@${expected_digest}"

    if [[ -z ${expected_digest} ]]; then
        echo "missing manifest digest for ${target}" >&2
        return 1
    fi

    if dest_is_insecure; then
        inspect_args+=(--tls-verify=false)
        copy_args+=(--dest-tls-verify=false)
    fi

    existing="$(skopeo inspect --format '{{.Digest}}' "${inspect_args[@]}" "docker://${dest}" 2>/dev/null || true)"
    if [[ ${existing} == "${expected_digest}" ]]; then
        echo "Skipping ${dest}; digest ${expected_digest} is already present"
        return 0
    fi

    echo "Copying ${src} to ${dest}"
    skopeo copy "${copy_args[@]}" "docker://${src}" "docker://${dest}"
}

for target in ${virt_template_targets}; do
    copy_virt_template_image "${target}" "${image_prefix}"
done

if [[ -n ${image_prefix_alt} ]]; then
    for target in ${virt_template_targets}; do
        copy_virt_template_image "${target}" "${image_prefix_alt}"
    done
fi
