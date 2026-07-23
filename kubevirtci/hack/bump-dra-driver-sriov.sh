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

set -euo pipefail

UPSTREAM_REPO="${UPSTREAM_REPO:-https://github.com/k8snetworkplumbingwg/dra-driver-sriov.git}"
NAMESPACE="dra-driver-sriov"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

UPSTREAM_SHA="$(git ls-remote --exit-code "${UPSTREAM_REPO}" refs/heads/main | cut -c1-7)"

UPSTREAM_CHECKOUT="${TMP_DIR}/dra-driver-sriov"
GENERATED_MANIFEST="${TMP_DIR}/sriov_dra.yaml"

git clone --quiet --depth 1 "${UPSTREAM_REPO}" "${UPSTREAM_CHECKOUT}"

helm template sriov-dra \
    "${UPSTREAM_CHECKOUT}/deployments/helm/dra-driver-sriov/" \
    --set image.tag="${UPSTREAM_SHA}" \
    --set kubeletPlugin.enableDeviceMetadata=true \
    -n "${NAMESPACE}" | \
    sed '/helm.sh\/chart:/d' | \
    sed '/app.kubernetes.io\/version:/d' | \
    sed '/app.kubernetes.io\/managed-by: Helm/d' >"${GENERATED_MANIFEST}"

mapfile -t TARGET_MANIFESTS < <(find cluster-up/cluster -path "*/sriov-components/manifests/dra/sriov_dra.yaml" -type f)
if (( ${#TARGET_MANIFESTS[@]} == 0 )); then
    echo "ERROR: no sriov_dra.yaml manifests found under cluster-up/cluster"
    exit 1
fi

for manifest in "${TARGET_MANIFESTS[@]}"; do
    cp "${GENERATED_MANIFEST}" "${manifest}"
    echo "Updated ${manifest}"
done

echo "Done. Updated ${#TARGET_MANIFESTS[@]} manifests with upstream SHA ${UPSTREAM_SHA}."
