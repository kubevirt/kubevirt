#!/bin/bash -ex

mkdir -p test-out || true

export IMAGE_TAG=latest

UPGRADE_VERSION=100.0.0

CLUSTER_DIR=/registry/kubevirt-hyperconverged/${UPGRADE_VERSION}

CLUSTER_FILE=${CLUSTER_DIR}/kubevirt-hyperconverged-operator.v${UPGRADE_VERSION}.clusterserviceversion.yaml

echo ${CLUSTER_FILE}

docker run --entrypoint ls  ${IMAGE_REGISTRY}/kubevirt/hco-registry-upgrade:${IMAGE_TAG} ${CLUSTER_DIR} || true

docker run --entrypoint cat  ${IMAGE_REGISTRY}/kubevirt/hco-registry-upgrade:${IMAGE_TAG} ${CLUSTER_FILE} > ./test-out/clusterserviceversion.yaml

ls -al test-out

cat ./test-out/clusterserviceversion.yaml


