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

# CI considerations: $TARGET is used by the jenkins vagrant build, to distinguish what to test
# Currently considered $TARGET values:
#     vagrant-dev: Runs all functional tests on a development vagrant setup
#     vagrant-release: Runs all possible functional tests on a release deployment in vagrant
#     TODO: vagrant-tagged-release: Runs all possible functional tests on a release deployment in vagrant on a tagged release

set -ex

export WORKSPACE="${WORKSPACE:-$PWD}"

kubectl() { cluster/kubectl.sh "$@"; }

export BUILDER_NAME=$TARGET

if [ "$TARGET" = "vagrant-dev"  ]; then
cat > hack/config-local.sh <<EOF
master_ip=192.168.1.2
EOF
export RSYNCD_PORT=${RSYNCD_PORT:-10874}
elif [ "$TARGET" = "vagrant-release"  ]; then
cat > hack/config-local.sh <<EOF
master_ip=192.168.2.2
EOF
export RSYNCD_PORT=${RSYNCD_PORT:-10875}
fi

export VAGRANT_PREFIX=${VARIABLE:-kubevirt}
export VAGRANT_NUM_NODES="${VAGRANT_NUM_NODES:-1}"
# Keep .vagrant files between builds
export VAGRANT_DOTFILE_PATH="${VAGRANT_DOTFILE_PATH:-$WORKSPACE/.vagrant}"

# Install dockerize
export DOCKERIZE_VERSION=v0.3.0
curl -LO https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C $WORKSPACE -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# Make sure that the VM is properly shut down on exit
trap '{ make cluster-down; }' EXIT

# TODO handle complete workspace removal on CI
set +e
make cluster-up
if [ $? -ne 0 ]; then
  vagrant destroy
  set -e
  make cluster-up
fi
set -e

# Build KubeVirt
make

# Make sure we can connect to kubernetes
export APISERVER=$(cat cluster/vagrant-kubernetes/.kubeconfig | grep server | sed -e 's# \+server: https://##' | sed -e 's/\r//')
$WORKSPACE/dockerize -wait tcp://$APISERVER -timeout 300s
# Make sure we don't try to talk to Vagrant host via a proxy
export no_proxy="${APISERVER%:*}"

# Wait for nodes to become ready
while [ -n "$(kubectl get nodes --no-headers | grep -v Ready)" ]; do
   echo "Waiting for all nodes to become ready ..."
   kubectl get nodes --no-headers | >&2 grep -v Ready || true
   sleep 10
done
echo "Nodes are ready:"
kubectl get nodes

# Wait for all kubernetes pods to become ready (dont't wait for kubevirt pods from previous deployments)
sleep 10
while [ -n "$(kubectl get pods -n kube-system -l '!kubevirt.io' --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubernetes pods to become ready ..."
    kubectl get pods -n kube-system --no-headers | >&2 grep -v Running || true
    sleep 10
done

echo "Kubernetes is ready:"
kubectl get pods -n kube-system -l '!kubevirt.io'
echo ""
echo ""

# delete all old traces of kubevirt on the cluster
make cluster-clean

if [ -z "$TARGET" ] || [ "$TARGET" = "vagrant-dev"  ]; then
    make cluster-sync
elif [ "$TARGET" = "vagrant-release"  ]; then
    make cluster-sync
fi

# Wait until kubevirt pods are running
while [ -n "$(kubectl get pods -n kube-system --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubevirt pods to enter the Running state ..."
    kubectl get pods -n kube-system --no-headers | >&2 grep -v Running || true
    sleep 10
done

# Make sure all containers except virt-controller are ready
while [ -n "$(kubectl get pods -n kube-system -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '!/virt-controller/ && /false/')" ]; do
    echo "Waiting for KubeVirt containers to become ready ..."
    kubectl get pods -n kube-system -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '!/virt-controller/ && /false/' || true
    sleep 10
done

# Make sure that at least one virt-controller container is ready
while [ "$(kubectl get pods -n kube-system -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '/virt-controller/ && /true/' | wc -l)" -lt "1" ]; do
    echo "Waiting for KubeVirt virt-controller container to become ready ..."
    kubectl get pods -n kube-system -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '/virt-controller/ && /true/' | wc -l
    sleep 10
done

kubectl get pods -n kube-system
kubectl version

# Disable proxy configuration since it causes test issues
export -n http_proxy
# Run functional tests
FUNC_TEST_ARGS="--ginkgo.noColor" make functest
