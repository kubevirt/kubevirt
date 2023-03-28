#!/bin/bash

#
# Configures HPP-CSI on a SNO cluster using the StoragePool feature.
#
# Deploys two storage classes
# * hostpath-csi-basic - uses root filesystem of the worker nodes
# * hostpath-csi-pvc-block - utilize another storage class as a backend
#
# Requires HPP operator to be deployed on the cluster.

set -ex

readonly SCRIPT_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")
VOLUME_BINDING_MODE="WaitForFirstConsumer"

CLUSTER_PLATFORM=$(
  oc get infrastructure cluster \
    --output=jsonpath='{$.status.platform}'
)

case "${CLUSTER_PLATFORM}" in
  Azure)
    HPP_BACKEND_STORAGE_CLASS=managed-csi
    HPP_VOLUME_SIZE=128Gi
    ;;
  AWS)
    HPP_BACKEND_STORAGE_CLASS=gp3-csi
    HPP_VOLUME_SIZE=128Gi
    ;;
  GCP)
    HPP_BACKEND_STORAGE_CLASS=standard-csi
    HPP_VOLUME_SIZE=128Gi
    ;;
  BareMetal|None)
    HPP_BACKEND_STORAGE_CLASS=local-block-hpp
    HPP_VOLUME_SIZE=5Gi
    ;;
  *)
    echo "[ERROR] Unsupported cluster platform: [${CLUSTER_PLATFORM}]" >&2
    exit 1
    ;;
esac

# Create HPP CustomResource using the StoragePool feature
sed "${SCRIPT_DIR}/10_hpp_pool_cr.yaml" \
  -e "s|^\( \+storage\): .*|\1: ${HPP_VOLUME_SIZE}|g" \
  -e "s|^\( \+storageClassName\): .*|\1: ${HPP_BACKEND_STORAGE_CLASS}|g" \
| oc create --filename=-

# Create HPP StorageClass using the StoragePool feature
oc create --filename="${SCRIPT_DIR}/20_hpp_pool_sc.yaml"
