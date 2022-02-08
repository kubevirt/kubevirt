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
# Copyright 2020 Red Hat, Inc.
#
# Used only with the STDCI
# Builds operator and registry images.


RELEASE_DELTA="${RELEASE_DELTA:-0}"
LATEST_KUBEVIRT=$(curl -L https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt/latest)
LATEST_KUBEVIRT_IMAGE=$(curl -L https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt/${LATEST_KUBEVIRT}/kubevirt-operator.yaml | grep 'OPERATOR_IMAGE' -A1 | tail -n 1 | sed 's/.*value: //g')
LATEST_KUBEVIRT_COMMIT=$(curl -L https://storage.googleapis.com/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt/${LATEST_KUBEVIRT}/commit)
go mod edit -require kubevirt.io/kubevirt@${LATEST_KUBEVIRT_COMMIT}
go mod vendor
KUBEVIRT_IMAGE=${LATEST_KUBEVIRT_IMAGE} hack/build-manifests.sh

container_id=$(podman ps | grep kubevirtci | cut -d ' ' -f 1)
registry_port=$(podman port $container_id | grep 5000 | cut -d ':' -f 2)
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

# Build registry image

export REGISTRY_DOCKERFILE="Dockerfile.registry.kubevirt.nightly"
export REGISTRY_IMAGE_NAME="hco-registry-kubevirt-nightly"
PACKAGE_DIR="./deploy/olm-catalog/community-kubevirt-hyperconverged"
INITIAL_CHANNEL=$(ls -d ${PACKAGE_DIR}/*/ | sort -rV | awk "NR==$((RELEASE_DELTA+1))" | cut -d '/' -f 5)
export REGISTRY_EXTRA_BUILD_ARGS="--build-arg KUBEVIRT_PROVIDER=$KUBEVIRT_PROVIDER --build-arg HCO_VERSION=$INITIAL_CHANNEL"
make bundleRegistry

pwd
make container-clusterserviceversion
ls -al ./test-out


# check images are accessible
CLUSTER_NODES=$(./cluster/kubectl.sh get nodes | grep Ready | cut -d ' ' -f 1)
for NODE in $CLUSTER_NODES; do
    ./cluster/ssh.sh $NODE 'sudo podman pull registry:5000/kubevirt/hco-registry-kubevirt-nightly:latest'
done

