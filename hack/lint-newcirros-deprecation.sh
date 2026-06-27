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
# Copyright the KubeVirt Authors.
#
# Ensures no new CirrOS usages are introduced outside the known baseline.
# As you migrate a file, remove it from the BASELINE list below.

set -eo pipefail

NEWCIRROS_BASELINE="
pkg/virt-controller/watch/vsock/vsock_test.go
tests/container_disk_test.go
tests/hotplug/pci-ports.go
tests/hotplug/pci_topology.go
tests/network/bindingplugin_slirp.go
tests/network/services.go
tests/network/vmi_istio.go
tests/network/vmi_lifecycle.go
tests/network/vmi_networking.go
tests/performance/density-kwok.go
tests/performance/density.go
tests/stability_test.go
tests/vmi_cloudinit_hook_sidecar_test.go
tests/vmi_cloudinit_test.go
tests/vmi_configuration_test.go
tests/vmi_lifecycle_test.go
tests/vmi_sound_test.go
"

CONTAINERDISK_BASELINE="
tests/container_disk_test.go
tests/storage/objectgraph_test.go
tests/vmi_configuration_test.go
tests/vmi_lifecycle_test.go
"

EXCLUDE_FILES="
tests/containerdisk/containerdisk.go
tests/libvmifact/factory.go
"

exit_code=0

check_pattern() {
    local pattern="$1"
    local baseline="$2"
    local msg="$3"

    raw_files="$(grep -Erl "$pattern" --include='*.go' . || true)"

    if [ -z "$raw_files" ]; then
        return
    fi

    actual_files="$(printf '%s\n' "$raw_files" |
        sed 's|^\./||' |
        sort)"

    while IFS= read -r f; do
        if echo "$EXCLUDE_FILES" | grep -Fqx -- "$f"; then
            continue
        fi
        if ! echo "$baseline" | grep -Fqx -- "$f"; then
            echo "ERROR: $f $msg"
            echo "       Use NewAlpine or NewAlpineWithTestTooling instead."
            echo "       See https://github.com/kubevirt/kubevirt/issues/15043"
            echo ""
            exit_code=1
        fi
    done <<EOF
$actual_files
EOF
}

check_pattern 'libvmifact\.NewCirros\(' "$NEWCIRROS_BASELINE" \
    "uses NewCirros which is deprecated."

check_pattern 'ContainerDiskCirros(CustomLocation)?' "$CONTAINERDISK_BASELINE" \
    "uses ContainerDiskCirros directly which is deprecated."

if [ "$exit_code" -ne 0 ]; then
    exit 1
fi

echo "No new CirrOS usages found outside the baseline."
