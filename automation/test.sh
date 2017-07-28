#!/bin/bash
#
# This file is part of the kubevirt project
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
export GOPATH=$WORKSPACE/go
export GOBIN=$WORKSPACE/go/bin
export PATH=$GOPATH/bin:$PATH
export VAGRANT_NUM_NODES=1

# Install dockerize
export DOCKERIZE_VERSION=v0.3.0
curl -LO https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C $WORKSPACE -xzvf dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# Keep .vagrant files between builds
export VAGRANT_DOTFILE_PATH=$WORKSPACE/.vagrant

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
go get -u github.com/Masterminds/glide
make

# Copy connection details for kubernetes
cluster/kubectl.sh --init

# Make sure we can connect to kubernetes
export APISERVER=$(cat cluster/vagrant/.kubeconfig | grep server | sed -e 's# \+server: https://##' | sed -e 's/\r//')
$WORKSPACE/dockerize -wait tcp://$APISERVER -timeout 120s

# Wait for nodes to become ready
while [ -n "$(kubectl get nodes --no-headers | grep -v Ready)" ]; do
   echo "Waiting for all nodes to become ready ..."
   kubectl get nodes --no-headers | >&2 grep -v Ready
   sleep 10
done
echo "Nodes are ready:"
kubectl get nodes

echo "Work around bug https://github.com/kubernetes/kubernetes/issues/31123: Force a recreate of the discovery pods"
kubectl delete pods -n kube-system -l k8s-app=kube-discovery
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
kubectl delete pods --all

# Deploy kubevirt
cluster/sync.sh

# Wait until kubevirt is ready
while [ -n "$(kubectl get pods --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubevirt pods to become ready ..."
    kubectl get pods --no-headers | >&2 grep -v Running
    sleep 10
done
kubectl get pods
cluster/kubectl.sh version

# Run functional tests
FUNC_TEST_ARGS="--ginkgo.noColor" make functest
