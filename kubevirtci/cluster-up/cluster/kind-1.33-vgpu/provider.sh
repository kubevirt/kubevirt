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

set -e

DEFAULT_CLUSTER_NAME="vgpu"
DEFAULT_HOST_PORT=5000
ALTERNATE_HOST_PORT=5001
export CLUSTER_NAME=${CLUSTER_NAME:-$DEFAULT_CLUSTER_NAME}

if [ $CLUSTER_NAME == $DEFAULT_CLUSTER_NAME ]; then
    export HOST_PORT=$DEFAULT_HOST_PORT
else
    export HOST_PORT=$ALTERNATE_HOST_PORT
fi

# Use fake vGPU by default (set to "false" to require real hardware)
export USE_FAKE_VGPU=${USE_FAKE_VGPU:-true}

function set_kind_params() {
    version=$(cat "${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/version")
    export KIND_VERSION="${KIND_VERSION:-$version}"

    image=$(cat "${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/image")
    export KIND_NODE_IMAGE="${KIND_NODE_IMAGE:-$image}"
}

function configure_registry_proxy() {
    [ "$CI" != "true" ] && return

    echo "Configuring cluster nodes to work with CI mirror-proxy..."

    local -r ci_proxy_hostname="docker-mirror-proxy.kubevirt-prow.svc"
    local -r kind_binary_path="${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kind"
    local -r configure_registry_proxy_script="${KUBEVIRTCI_PATH}/cluster/kind/configure-registry-proxy.sh"

    KIND_BIN="$kind_binary_path" PROXY_HOSTNAME="$ci_proxy_hostname" $configure_registry_proxy_script
}

function validate_fake_vgpu() {
    if [ "$USE_FAKE_VGPU" != "true" ]; then
        echo "USE_FAKE_VGPU is not true, skipping fake vGPU validation"
        return 0
    fi

    echo "Validating fake vGPU setup..."
    
    local setup_script="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/setup-fake-vgpu-host.sh"
    
    # Check if module is loaded
    if ! lsmod | grep -q "fake_nvidia_vgpu"; then
        echo ""
        echo "ERROR: fake_nvidia_vgpu module is not loaded."
        echo ""
        echo "Please run the setup script first (requires sudo):"
        echo "  sudo $setup_script setup"
        echo ""
        echo "Or see: ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/fake-nvidia-vgpu/README.md"
        return 1
    fi
    
    # Check if mdev types exist
    if [ ! -d "/sys/class/mdev_bus/nvidia/mdev_supported_types/nvidia-222" ]; then
        echo ""
        echo "ERROR: mdev types not found at /sys/class/mdev_bus/nvidia/mdev_supported_types/"
        echo ""
        echo "The module may not have loaded correctly. Check dmesg for errors."
        return 1
    fi
    
    # Check if mdev instances exist
    local mdev_count=$(ls /sys/bus/mdev/devices/ 2>/dev/null | wc -l)
    if [ "$mdev_count" -eq 0 ]; then
        echo ""
        echo "ERROR: No mdev instances found."
        echo ""
        echo "Please run the setup script first (requires sudo):"
        echo "  sudo $setup_script setup"
        echo ""
        return 1
    fi
    
    echo "Found $mdev_count mdev instance(s)"
    echo "Fake vGPU validation passed"
}

function cleanup_fake_vgpu() {
    if [ "$USE_FAKE_VGPU" == "true" ]; then
        local setup_script="${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/setup-fake-vgpu-host.sh"
        echo ""
        echo "NOTE: To cleanup fake vGPU resources, run:"
        echo "  sudo $setup_script cleanup"
        echo ""
    fi
}

function _add_mdev_mounts() {
    # Add mdev_bus mount for fake vGPU support
    # Note: The fake vGPU module creates devices under /sys/devices/virtual/nvidia/
    cat <<EOF >> ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
  - containerPath: /sys/class/mdev_bus
    hostPath: /sys/class/mdev_bus
  - containerPath: /sys/bus/mdev
    hostPath: /sys/bus/mdev
  - containerPath: /sys/devices/virtual/nvidia
    hostPath: /sys/devices/virtual/nvidia
EOF
}

function up() {
    # Validate fake vGPU if enabled, otherwise check for real hardware
    if [ "$USE_FAKE_VGPU" == "true" ]; then
        validate_fake_vgpu || exit 1
    else
        # print hardware info for easier debugging based on logs
        echo 'Available cards'
        ${CRI_BIN} run --rm --cap-add=SYS_RAWIO quay.io/phoracek/lspci@sha256:0f3cacf7098202ef284308c64e3fc0ba441871a846022bb87d65ff130c79adb1 sh -c "lspci -k | grep -EA2 'VGA|3D'"
        echo ""
    fi

    cp $KIND_MANIFESTS_DIR/kind.yaml ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/kind.yaml
    _add_extra_mounts
    
    # Add mdev mounts for vGPU support
    _add_mdev_mounts
    
    kind_up

    configure_registry_proxy

    # remove the rancher.io kind default storageClass
    _kubectl delete sc standard

    ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/config_vgpu_cluster.sh

    echo "$KUBEVIRT_PROVIDER cluster '$CLUSTER_NAME' is ready"
}

set_kind_params

source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
