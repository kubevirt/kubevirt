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
# Copyright 2022 IBM, Inc.
#

set -ex

kubectl() { KUBEVIRTCI_VERBOSE=false kubevirtci/cluster-up/kubectl.sh "$@"; }
export IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

_prometheus_port_forward_pid=""
trap "clean_up" EXIT SIGINT SIGTERM SIGQUIT
clean_up() {
  kill -9 $_prometheus_port_forward_pid 2> /dev/null | exit 0
  make cluster-clean
}

echo "Nodes are ready:"
kubectl get nodes

echo "Cluster sync: push docker images and deploy kubevirt"
make cluster-sync

# OpenShift is running important containers under default namespace
namespaces=(kubevirt default)
if [[ $NAMESPACE != "kubevirt" ]]; then
  namespaces+=($NAMESPACE)
fi

timeout=300
sample=30

for i in ${namespaces[@]}; do
  # Wait until kubevirt pods are running or completed
  current_time=0
  while [ -n "$(kubectl get pods -n $i --no-headers | grep -v -E 'Running|Completed')" ]; do
    echo "Waiting for kubevirt pods to enter the Running/Completed state ..."
    kubectl get pods -n $i --no-headers | >&2 grep -v -E 'Running|Completed' || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      echo "Dump kubevirt state:"
      make dump
      exit 1
    fi
  done

  # Make sure all containers are ready
  current_time=0
  while [ -n "$(kubectl get pods -n $i --field-selector=status.phase==Running -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false)" ]; do
    echo "Waiting for KubeVirt containers to become ready ..."
    kubectl get pods -n $i --field-selector=status.phase==Running -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      echo "Dump kubevirt state:"
      make dump
      exit 1
    fi
  done
  kubectl get pods -n $i
done

# build perfscale tools
make bazel-build

export DOCKER_PREFIX="${DOCKER_PREFIX:-registry:5000/kubevirt}"
export DOCKER_TAG="${DOCKER_TAG:-devel}"
export PERFAUDIT="${PERFAUDIT:-true}"
export PROMETHEUS_PORT=${PROMETHEUS_PORT:-30007}

# expose prometheus in an external kubernetes cluster
if [[ (${PERFAUDIT} == "true" || ${PERFAUDIT} == "True") && ${KUBEVIRT_PROVIDER} == "external" ]]; then
  kubectl -n openshift-monitoring port-forward service/prometheus-operated ${PROMETHEUS_PORT} &> /dev/null &
  _prometheus_port_forward_pid=$1
fi

./hack/perfscale-tests.sh
