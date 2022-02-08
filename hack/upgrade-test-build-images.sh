#!/bin/bash -ex
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
# Copyright 2019 Red Hat, Inc.
#
# Used only with the STDCI
# Builds operator and registry images.

source hack/cri-bin.sh

container_id=$($CRI_BIN ps | grep kubevirtci | cut -d ' ' -f 1)
registry_port=$($CRI_BIN port $container_id | grep 5000 | cut -d ':' -f 2)
registry=localhost:$registry_port

echo "INFO: registry: $registry"

export REGISTRY_NAMESPACE=kubevirt
export IMAGE_REGISTRY=$registry
export CONTAINER_TAG=latest
make container-build-operator container-push-operator

# check images are accessible
CLUSTER_NODES=$(./cluster/kubectl.sh get nodes | grep Ready | cut -d ' ' -f 1)
for NODE in $CLUSTER_NODES; do
    ./cluster/ssh.sh $NODE 'sudo podman pull registry:5000/kubevirt/hyperconverged-cluster-operator'
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster/ssh.sh $NODE 'sudo sysctl -w user.max_user_namespaces=1024'
done

# Build upgrade registry image

export REGISTRY_DOCKERFILE="Dockerfile.registry.upgrade"
export REGISTRY_IMAGE_NAME="hco-registry-upgrade"
export REGISTRY_EXTRA_BUILD_ARGS="--build-arg KUBEVIRT_PROVIDER=$KUBEVIRT_PROVIDER"
make bundleRegistry

pwd
make container-clusterserviceversion
ls -al ./test-out


# check images are accessible
CLUSTER_NODES=$(./cluster/kubectl.sh get nodes | grep Ready | cut -d ' ' -f 1)
for NODE in $CLUSTER_NODES; do
    ./cluster/ssh.sh $NODE 'sudo podman pull registry:5000/kubevirt/hco-registry-upgrade:latest'
done

