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
# How long to wait for containers to start up and become ready
# (Specified in duplicates of START_WAIT_DELAY)
export START_WAIT_TIMEOUT=${START_WAIT_TIMEOUT:-60}
# How long to wait between checks for container readiness (In seconds)
export START_WAIT_DELAY=10

kubectl() { cluster/kubectl.sh "$@"; }

kubectl_or_die() {
    if ! kubectl "$@"; then
        echo "kubectl invocation failed, did the cluster crash?"
        exit 1
    fi
}

retry_check() {
    local attempt

    let attempt=START_WAIT_TIMEOUT
    while true; do
        "$@" && break
        if (( --attempt <= 0 )); then
            echo "Timed out waiting"
            exit 1
        fi
        sleep $START_WAIT_DELAY
    done
}

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

nodes_ready() {
    local kctl_out

    echo "Waiting for all nodes to become ready ..."
    kctl_out="$(kubectl_or_die get nodes --no-headers)" || exit 128
    grep -v Ready <<<"$kctl_out" && return 1
    return 0
}

pods_up() {
    local description="${1:?}"
    local filter
    local kctl_out
    if [[ $# -ge 2 ]]; then
        filter=(-l "$2")
    else
        filter=()
    fi

    echo "Waiting for $description pods to enter the Running state ..."
    kctl_out="$(
        kubectl_or_die get pods -n kube-system --no-headers "${filter[@]}"
    )" || exit 128
    grep -v Running <<<"$kctl_out" && return 1
    return 0
}

# Wait for nodes to become ready
retry_check nodes_ready
echo "Nodes are ready:"
kubectl_or_die get nodes

# Wait for all kubernetes pods to become ready (dont't wait for kubevirt pods from previous deployments)
retry_check pods_up 'kubernetes' '!kubevirt.io'

echo "Kubernetes is ready:"
kubectl_or_die get pods -n kube-system -l '!kubevirt.io'
echo ""
echo ""

# delete all old traces of kubevirt on the cluster
make cluster-clean

if [ -z "$TARGET" ] || [ "$TARGET" = "vagrant-dev"  ]; then
    make cluster-sync
elif [ "$TARGET" = "vagrant-release"  ]; then
    make cluster-sync
fi

containers_ready() {
    local description="${1:?}"
    local mode="${2:?}"
    shift 2
    local grep_args=("${@}")
    local kctl_out

    echo "Waiting for $description to become ready ..."
    kctl_out="$(
        kubectl_or_die get pods -n kube-system \
            -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' \
            --no-headers
    )" || exit 128
    grep "${grep_args[@]}" <<<"$kctl_out" \
    | if [[ "$mode" == all ]]; then
        grep 'false' && return 1
        return 0
    else
        grep 'true' && return 0
        return 1
    fi
}

# Wait until kubevirt pods are running
retry_check pods_up 'KubeVirt'

# Make sure all containers except virt-controller are ready
retry_check containers_ready 'KubeVirt containers' all -v 'virt-controller'

# Make sure that at least one virt-controller container is ready
retry_check containers_ready 'KubeVirt virt-controller container' any 'virt-controller'

kubectl_or_die get pods -n kube-system
kubectl version

# Disable proxy configuration since it causes test issues
export -n http_proxy
# Run functional tests
FUNC_TEST_ARGS="--ginkgo.noColor" make functest
