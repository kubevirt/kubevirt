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

set -exuo pipefail

SCRIPT_PATH=$(dirname "$(realpath "$0")")

kubevirtci_path="$(realpath "${SCRIPT_PATH}/../../..")/"
PROVIDER_PATH="${kubevirtci_path}/cluster-up/cluster/${KUBEVIRT_PROVIDER}"

RUN_KUBEVIRT_CONFORMANCE=${RUN_KUBEVIRT_CONFORMANCE:-"false"}

(
  cd $kubevirtci_path
  kubectl="./cluster-up/kubectl.sh"
  echo "Wait for pods to be ready.."
  timeout 5m bash -c "until ${kubectl} wait --for=condition=Ready pod --timeout=30s --all  -A; do sleep 1; done"
  timeout 5m bash -c "until ${kubectl} wait --for=condition=Ready pod --timeout=30s -n kube-system --all; do sleep 1; done"
  ${kubectl} get nodes
  ${kubectl} get pods -A
  echo ""
  
  nodes=$(${kubectl} get nodes --no-headers | awk '{print $1}')
  for node in $nodes; do
    node_exec="docker exec ${node}"
    echo "[$node] network interfaces status:"
    ${node_exec} ip a
    echo ""
    echo "[$node] route table:"
    ${node_exec} ip r
    echo ""
    echo "[$node] hosts file:"
    ${node_exec} cat /etc/hosts
    echo ""
    echo "[$node] resolve config:"
    ${node_exec} cat /etc/resolv.conf
    echo ""
  done

  if [ "$RUN_KUBEVIRT_CONFORMANCE" == "true" ]; then
    nightly_build_base_url="https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt"
    latest=$(curl -sL "${nightly_build_base_url}/latest")

    echo "Deploy latest nighly build Kubevirt"
    if [ "$(kubectl get kubevirts -n kubevirt kubevirt -ojsonpath='{.status.phase}')" != "Deployed" ]; then
      ${kubectl} apply -f "${nightly_build_base_url}/${latest}/kubevirt-operator.yaml"
      ${kubectl} apply -f "${nightly_build_base_url}/${latest}/kubevirt-cr.yaml"
    fi
    ${kubectl} wait -n kubevirt kv kubevirt --for condition=Available --timeout 15m

    echo "Run latest nighly build Kubevirt conformance tests"
    kubevirt_plugin="--plugin ${nightly_build_base_url}/${latest}/conformance.yaml"
    SONOBUOY_EXTRA_ARGS="${SONOBUOY_EXTRA_ARGS} ${kubevirt_plugin}"

    commit=$(curl -sL "${nightly_build_base_url}/${latest}/commit")
    commit="${commit:0:9}"
    container_tag="--plugin-env kubevirt-conformance.CONTAINER_TAG=${latest}_${commit}"
    SONOBUOY_EXTRA_ARGS="${SONOBUOY_EXTRA_ARGS} ${container_tag}"
    
    hack/conformance.sh ${PROVIDER_PATH}/conformance.json
  fi 
)
