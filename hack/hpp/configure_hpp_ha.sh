#!/bin/bash

#
# Configures HPP-CSI on an OCP cluster using the StoragePool feature.
#
# Deploys two storage classes
# * hostpath-csi-basic - uses root filesystem of the worker nodes
# * hostpath-csi-pvc-block - utilize another storage class as a backend
#
# Requires HPP operator to be deployed on the cluster.

set -ex

readonly SCRIPT_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")
HPP_VOLUME_SIZE=${HPP_VOLUME_SIZE:-${VOLUME_SIZE:-70}}Gi


CLUSTER_PLATFORM=$(
  oc get infrastructure cluster \
    --output=jsonpath='{$.status.platform}'
)

case "${CLUSTER_PLATFORM}" in
  Azure)
    HPP_BACKEND_STORAGE_CLASS=managed-csi
    ;;
  AWS)
    HPP_BACKEND_STORAGE_CLASS=gp3-csi
    ;;
  GCP)
    HPP_BACKEND_STORAGE_CLASS=standard-csi
    ;;
  BareMetal)
    HPP_BACKEND_STORAGE_CLASS=ocs-storagecluster-ceph-rbd
    ;;
  None)
  # UPI Installation
    HPP_BACKEND_STORAGE_CLASS=${HPP_BACKEND_STORAGE_CLASS:-ocs-storagecluster-ceph-rbd}
    ;;
  *)
    echo "[ERROR] Unsupported cluster platform: [${CLUSTER_PLATFORM}]" >&2
    exit 1
    ;;
esac


# Create HPP CustomResource using the StoragePool feature
sed "${SCRIPT_DIR}/10_hpp_pool_cr.yaml" \
  -e "s|^\( \+storageClassName\): .*|\1: ${HPP_BACKEND_STORAGE_CLASS}|g" \
| oc create --filename=-

# Create HPP StorageClass using the StoragePool feature
oc create --filename="${SCRIPT_DIR}/20_hpp_pool_sc.yaml"
