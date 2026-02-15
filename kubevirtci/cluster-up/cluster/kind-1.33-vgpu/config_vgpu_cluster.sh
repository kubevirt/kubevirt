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
# Copyright the KubeVirt Authors.
#
# Note: When running with fake vGPU, this script is called from the provider
# after the cluster is up. The fake vGPU module should already be loaded on the host.

set -xe

SCRIPT_PATH=$(dirname "$(realpath "$0")")

source ${SCRIPT_PATH}/vgpu-node/node.sh

# KUBEVIRTCI_PATH should be set by the caller
if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(cd "$(dirname "$BASH_SOURCE[0]")/../../" && pwd)"
fi

echo "KUBEVIRTCI_PATH: ${KUBEVIRTCI_PATH}"
source ${KUBEVIRTCI_PATH}/cluster/kind/common.sh
echo "_kubectl: ${_kubectl}"

# Get cluster nodes
nodes=($(_kubectl get nodes -o custom-columns=:.metadata.name --no-headers))
echo "Cluster nodes: ${nodes[*]}"

# Remount sysfs as read-write in Kind nodes
node::remount_sysfs "${nodes[*]}"

# Discover and display GPU/mdev information
echo ""
echo "=== vGPU/mdev Discovery ==="
node::discover_host_gpus
echo ""
node::list_mdev_types
echo ""
node::list_mdev_devices
echo ""

# Optional: Install mesa for vGPU display support
# Set VGPU_DISPLAY=true to enable display support (requires mesa-injector webhook)
if [ "${VGPU_DISPLAY:-false}" == "true" ]; then
    echo ""
    echo "=== Installing Mesa for vGPU Display Support ==="
    node::install_mesa "${nodes[*]}"
    
    echo ""
    echo "To enable display support, deploy the mesa-injector webhook:"
    echo "  ${SCRIPT_PATH}/mesa-injector/deploy.sh deploy"
    echo ""
fi

_kubectl get nodes
