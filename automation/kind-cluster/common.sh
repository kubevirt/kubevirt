#!/bin/bash -e

export CLUSTER_NAME=sriov-ci
export CLUSTER_CONTROL_PLANE="${CLUSTER_NAME}-control-plane"
export CLUSTER_CMD="docker exec -it -d ${CLUSTER_NAME}-control-plane"
export MANIFESTS_DIR="automation/kind-cluster/manifests"
export KUBECONFIG="/root/.kube/kind-config-${CLUSTER_NAME}"
export CONTAINER_REGISTRY_HOST="localhost:5000"
export DOCKER_PREFIX=${CONTAINER_REGISTRY_HOST}/kubevirt
export DOCKER_TAG=devel
export NO_PROXY="localhost,127.0.0.1,172.17.0.2"
