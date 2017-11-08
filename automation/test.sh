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

set -ex

kubectl() { cluster/kubectl.sh --core "$@"; }


# Install GO
eval "$(curl -sL https://raw.githubusercontent.com/travis-ci/gimme/master/gimme | GIMME_GO_VERSION=stable bash)"
export WORKSPACE="${WORKSPACE:-$PWD}"
export GOPATH="${GOPATH:-$WORKSPACE/go}"
export GOBIN="${GOBIN:-$GOPATH/bin}"
export PATH="$GOPATH/bin:$PATH"
export VAGRANT_NUM_NODES="${VAGRANT_NUM_NODES:-1}"

# Install dockerize
export DOCKERIZE_VERSION=v0.3.0
curl -LO https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C $WORKSPACE -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# Keep .vagrant files between builds
export VAGRANT_DOTFILE_PATH="${VAGRANT_DOTFILE_PATH:-$WORKSPACE/.vagrant}"

# Make sure that the VM is properly shut down on exit
trap '{ vagrant halt; }' EXIT

set +e
vagrant up --provider=libvirt
if [ $? -ne 0 ]; then
  # After a workspace cleanup we loose our .vagrant file, this means that we have to clean up libvirt
  vagrant destroy
  virsh destroy kubevirt_master
  virsh undefine kubevirt_master
  virsh destroy kubevirt_node0
  virsh undefine kubevirt_node0
  virsh net-destroy vagrant0
  virsh net-undefine vagrant0
  # Remove now stale images
  vagrant destroy
  set -e
  vagrant up --provider=libvirt
fi
set -e

# Build kubevirt
go get golang.org/x/tools/cmd/goimports
go get -u github.com/golang/dep/cmd/dep
make

# Copy connection details for kubernetes
cluster/kubectl.sh --init

# Make sure we can connect to kubernetes
export APISERVER=$(cat cluster/vagrant/.kubeconfig | grep server | sed -e 's# \+server: https://##' | sed -e 's/\r//')
$WORKSPACE/dockerize -wait tcp://$APISERVER -timeout 120s
# Make sure we don't try to talk to Vagrant host via a proxy
export no_proxy="${APISERVER%:*}"

# Wait for nodes to become ready
while [ -n "$(kubectl get nodes --no-headers | grep -v Ready)" ]; do
   echo "Waiting for all nodes to become ready ..."
   kubectl get nodes --no-headers | >&2 grep -v Ready
   sleep 10
done
echo "Nodes are ready:"
kubectl get nodes

sleep 10
while [ -n "$(kubectl get pods -n kube-system --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubernetes pods to become ready ..."
    kubectl get pods -n kube-system --no-headers | >&2 grep -v Running
    sleep 10
done

echo "Kubernetes is ready:"
kubectl get pods -n kube-system
echo ""
echo ""

# Delete traces from old deployments
kubectl delete deployments --all
kubectl delete ds --all
kubectl delete pods --all

# Deploy kubevirt
cluster/sync.sh

# Wait until kubevirt pods are running
while [ -n "$(kubectl get pods --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubevirt pods to enter the Running state ..."
    kubectl get pods --no-headers | >&2 grep -v Running || true
    sleep 10
done

# Make sure all containers except virt-controller are ready
while [ -n "$(kubectl get pods -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '!/virt-controller/ && /false/')" ]; do
    echo "Waiting for KubeVirt containers to become ready ..."
    kubectl get pods -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '!/virt-controller/ && /false/' || true
    sleep 10
done

# Make sure that at least one virt-controller container is ready
while [ "$(kubectl get pods -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '/virt-controller/ && /true/' | wc -l)" -lt "1" ]; do
    echo "Waiting for KubeVirt virt-controller container to become ready ..."
    kubectl get pods -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | awk '/virt-controller/ && /true/' | wc -l
    sleep 10
done

# Wait until all pods are running
while [ -n "$(kubectl get pods --no-headers --all-namespaces | grep -v Running)" ]; do
    echo "Waiting for pods in all namespaces to enter the Running state ..."
    kubectl get pods --no-headers --all-namespaces | >&2 grep -v Running || true
    sleep 5
done

kubectl get pods --all-namespaces
kubectl version

# Disable proxy configuration since it causes test issues
export -n http_proxy
# Run functional tests
FUNC_TEST_ARGS="--ginkgo.noColor" make functest
