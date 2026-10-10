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
# Copyright 2019 Red Hat, Inc.
#

set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# Opt-in E2E coverage instrumentation (VIRTCNV E2E code coverage).
# When --build-cover is passed, component binaries are built with Go coverage
# instrumentation so functests can collect a coverage profile. Production
# builds are unaffected unless this flag is explicitly set.
BUILD_COVER=false
for arg in "$@"; do
    case "${arg}" in
    --build-cover)
        BUILD_COVER=true
        ;;
    esac
done

coverage_args=()
if [ "${BUILD_COVER}" == "true" ]; then
    echo "Building images with E2E coverage instrumentation (--build-cover)"
    coverage_args=(
        --collect_code_coverage
        --instrumentation_filter='//cmd/...,//pkg/...,//staging/src/kubevirt.io/client-go/...'
        --@io_bazel_rules_go//go/config:cover_format=go_cover
        --@io_bazel_rules_go//go/config:tags=coverage_e2e
        # covermode is atomic automatically: rules_go v0.60.0 hardcodes
        # cover_mode = "atomic" whenever coverage is enabled
        # (go/private/actions/compilepkg.bzl L123-126), precisely because the
        # runtime/coverage APIs require it. No cover_mode flag is needed (or
        # exposed) -- atomic is guaranteed, so runtime/coverage.ClearCounters()
        # (per-test reset) works with these binaries.
    )
fi

# vars are uninteresting for the build step, they are interesting for the push step only
other_images_default="
    //cmd/sidecars:sidecar-shim-image
    //cmd/libguestfs:libguestfs-tools-image
    //cmd/test-helpers:test-helpers-image
    //cmd/example-node-hook-plugin:example-node-hook-plugin-image
    //containerimages:alpine-container-disk-image
    //containerimages:alpine-with-test-tooling
    //containerimages:fedora-with-test-tooling
    //images/disks-images-provider:disks-images-provider-image
    //images/vm-killer:vm-killer-image
"

other_images_x86_64_aarch64="
    //cmd/sidecars/smbios:example-hook-sidecar-image
    //cmd/sidecars/disk-mutation:example-disk-mutation-hook-sidecar-image
    //cmd/sidecars/cloudinit:example-cloudinit-hook-sidecar-image
    //cmd/plugin-sidecars/test-launcher-hook:test-launcher-hook-sidecar-image
    //cmd/sidecars/network-passt-binding:network-passt-binding-image
    //cmd/sidecars/test-network-vhostuser-dra-binding:test-network-vhostuser-dra-binding-image
    //cmd/pr-helper:pr-helper-image
    //containerimages:cirros-container-disk-image
    //containerimages:cirros-custom-container-disk-image
    //containerimages:virtio-container-disk-image
    //images/winrmcli:winrmcli-image
    //tests:conformance_image
"

other_images_x86_64_only="
    //containerimages:fedora-realtime
    //containerimages:alpine-ext-kernel-boot-demo-container
    //cmd/cniplugins/passt-binding/cmd:network-passt-binding-cni-image
"

case ${ARCHITECTURE} in
"s390x" | "crossbuild-s390x")
    other_images="
        $other_images_default
    "
    ;;
"aarch64" | "crossbuild-aarch64")
    other_images="
        $other_images_default
        $other_images_x86_64_aarch64
    "
    ;;
*)
    other_images="
        $other_images_default
        $other_images_x86_64_aarch64
        $other_images_x86_64_only
    "
    ;;
esac

bazel build \
    --config=${ARCHITECTURE} ${BAZEL_CS_CONFIG} \
    "${coverage_args[@]}" \
    --define container_prefix= \
    --define image_prefix= \
    --define container_tag= \
    //cmd/virt-operator:virt-operator-image //cmd/virt-api:virt-api-image //cmd/virt-controller:virt-controller-image \
    //cmd/virt-handler:virt-handler-image //cmd/virt-launcher:virt-launcher-image //cmd/virt-exportproxy:virt-exportproxy-image \
    //cmd/virt-exportserver:virt-exportserver-image //cmd/synchronization-controller:virt-synchronization-controller-image ${other_images[@]}

rm -rf ${DIGESTS_DIR}/${ARCHITECTURE}
mkdir -p ${DIGESTS_DIR}/${ARCHITECTURE}

for f in $(find bazel-bin/ -name '*.json.sha256' | grep -v 'version-container'); do
    dir=${DIGESTS_DIR}/${ARCHITECTURE}/$(dirname $f)
    mkdir -p ${dir}
    cp -f ${f} ${dir}/$(basename ${f})
done
