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
# Copyright 2017 Red Hat, Inc.
#

# CI considerations: $TARGET is used by the jenkins build, to distinguish what to test
# Currently considered $TARGET values:
#     vagrant-dev: Runs all functional tests on a development vagrant setup (deprecated)
#     vagrant-release: Runs all possible functional tests on a release deployment in vagrant (deprecated)
#     kubernetes-dev: Runs all functional tests on a development kubernetes setup
#     kubernetes-release: Runs all functional tests on a release kubernetes setup
#     openshift-release: Runs all functional tests on a release openshift setup
#     TODO: vagrant-tagged-release: Runs all possible functional tests on a release deployment in vagrant on a tagged release

set -ex

export WORKSPACE="${WORKSPACE:-$PWD}"

if [[ $TARGET =~ openshift-.* ]]; then
  if [[ $TARGET =~ .*-crio-.* ]]; then
    export KUBEVIRT_PROVIDER="os-3.9.0-crio"
  else
    export KUBEVIRT_PROVIDER="os-3.10.0"
  fi
elif [[ $TARGET =~ .*-1.9.3-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-1.9.3"
else
  export KUBEVIRT_PROVIDER="k8s-1.10.3"
fi

export KUBEVIRT_NUM_NODES=2
export NFS_WINDOWS_DIR=${NFS_WINDOWS_DIR:-/home/nfs/images/windows2016}

kubectl() { cluster/kubectl.sh "$@"; }

export NAMESPACE="${NAMESPACE:-kube-system}"

# Make sure that the VM is properly shut down on exit
trap '{ make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP

make cluster-down
make cluster-up

# Wait for nodes to become ready
set +e
kubectl get nodes --no-headers
kubectl_rc=$?
while [ $kubectl_rc -ne 0 ] || [ -n "$(kubectl get nodes --no-headers | grep NotReady)" ]; do
    echo "Waiting for all nodes to become ready ..."
    kubectl get nodes --no-headers
    kubectl_rc=$?
    sleep 10
done
set -e

echo "Nodes are ready:"
kubectl get nodes

make cluster-sync

# OpenShift is running important containers under default namespace
namespaces=(kube-system default)
if [[ $NAMESPACE != "kube-system" ]]; then
  namespaces+=($NAMESPACE)
fi

timeout=300
sample=30

for i in ${namespaces[@]}; do
  # Wait until kubevirt pods are running
  current_time=0
  while [ -n "$(kubectl get pods -n $i --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubevirt pods to enter the Running state ..."
    kubectl get pods -n $i --no-headers | >&2 grep -v Running || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      exit 1
    fi
  done

  # Make sure all containers are ready
  current_time=0
  while [ -n "$(kubectl get pods -n $i -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false)" ]; do
    echo "Waiting for KubeVirt containers to become ready ..."
    kubectl get pods -n $i -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      exit 1
    fi
  done
  kubectl get pods -n $i
done

kubectl version

ginko_params="--ginkgo.noColor --junit-output=$WORKSPACE/junit.xml"

# Prepare PV for windows testing
if [[ -d $NFS_WINDOWS_DIR ]] && [[ $TARGET =~ windows.* ]]; then
  kubectl create -f - <<EOF
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: disk-windows
  labels:
    kubevirt.io/test: "windows"
spec:
  capacity:
    storage: 30Gi
  accessModes:
    - ReadWriteOnce
  nfs:
    server: "nfs"
    path: /
EOF
  ginko_params="$ginko_params --ginkgo.focus=Windows"
fi

# Run functional tests
FUNC_TEST_ARGS=$ginko_params make functest
