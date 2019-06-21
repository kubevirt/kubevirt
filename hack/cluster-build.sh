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
# Copyright 2017 Red Hat, Inc.
#

# This logic moved into the Makefile.
# We're leaving this file around for people who still reference this
# specific script in their development workflow.

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}

source hack/common.sh
source cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

echo "Building ..."

# Build everyting and publish it
${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} IMAGE_PULL_POLICY=${IMAGE_PULL_POLICY} VERBOSITY=${VERBOSITY} ./hack/build-manifests.sh"
${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-build.sh"
${KUBEVIRT_PATH}hack/dockerized "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} DOCKER_TAG_ALT=${DOCKER_TAG_ALT} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/bazel-push-images.sh"

# Make sure that all nodes use the newest images
container=""
container_alias=""
for arg in ${docker_images}; do
    name=$(basename $arg)
    container="${container} ${manifest_docker_prefix}/${name}:${docker_tag} ${manifest_docker_prefix}/${name}:${docker_tag_alt}"
    container_alias="${container_alias} ${manifest_docker_prefix}/${name}:${docker_tag} kubevirt/${name}:${docker_tag}"
done

# OKD provider has different node names and does not have docker
if [[ $KUBEVIRT_PROVIDER =~ okd.* ]]; then
    nodes=("master-0" "worker-0")
    pull_command="podman"
elif [[ $KUBEVIRT_PROVIDER == "external" ]]; then
    nodes=() # in case of external provider we have no control over the nodes
else
    nodes=()
    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        nodes+=("node$(printf "%02d" ${i})")
    done
    pull_command="docker"
fi

for node in ${nodes[@]}; do
    ${KUBEVIRT_PATH}cluster-up/ssh.sh ${node} "echo \"${container}\" | xargs \-\-max-args=1 sudo ${pull_command} pull"
    ${KUBEVIRT_PATH}cluster-up/ssh.sh ${node} "echo \"${container_alias}\" | xargs \-\-max-args=2 sudo ${pull_command} tag"
done

echo "Done"
