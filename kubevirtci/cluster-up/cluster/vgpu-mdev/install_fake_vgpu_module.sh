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

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBEVIRTCI_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

export KUBEVIRTCI_PATH="${KUBEVIRTCI_ROOT}/cluster-up/"
export KUBEVIRTCI_CONFIG_PATH="${KUBEVIRTCI_CONFIG_PATH:-${KUBEVIRTCI_ROOT}/../_ci-configs}"
export KUBEVIRT_PROVIDER="${KUBEVIRT_PROVIDER:?Set KUBEVIRT_PROVIDER to the running cluster provider}"
export KUBECONFIG="${KUBECONFIG:-${KUBEVIRTCI_CONFIG_PATH}/${KUBEVIRT_PROVIDER}/.kubeconfig}"

if [[ -z "${KUBEVIRTCI_TAG:-}" && -z "${KUBEVIRTCI_GOCLI_CONTAINER:-}" ]]; then
    KUBEVIRTCI_TAG=$(curl --fail --silent --show-error --location \
        https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)
    export KUBEVIRTCI_TAG
fi

echo "Installing fake vGPU kernel module on ${KUBEVIRT_PROVIDER} worker nodes..."
"${SCRIPT_DIR}/setup_vgpu_mdev.sh" full-setup

echo "Fake vGPU kernel module installed on ${KUBEVIRT_PROVIDER} worker nodes."
