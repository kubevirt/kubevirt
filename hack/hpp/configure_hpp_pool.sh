#!/bin/bash

#
# Configures HPP on an OCP cluster using the StoragePool feature.
#
# Requires HPP operator to be deployed on the cluster. It is usually deployed
# as part of CNV by the HCO operator.
#
# See documentation:
# - https://docs.google.com/document/d/1v_kPxJKhy3WYVOIlTRviEpJbigqraE8Hte7BCKJNVBM
#

set -ex

readonly SCRIPT_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")

CLUSTER_PLATFORM=$(
  oc get infrastructure cluster \
    --output=jsonpath='{$.status.platform}'
)

case "${CLUSTER_PLATFORM}" in
  Azure)
    HPP_BACKEND_STORAGE_CLASS=managed-premium
    HPP_VOLUME_SIZE=128Gi
    ;;
  AWS)
    HPP_BACKEND_STORAGE_CLASS=gp2
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

# Create HPP CustomResource and StorageClass using the StoragePool feature
sed "${SCRIPT_DIR}/10_hpp_pool_cr.yaml" \
  -e "s|^\( \+storage\): .*|\1: ${HPP_VOLUME_SIZE}|g" \
  -e "s|^\( \+storageClassName\): .*|\1: ${HPP_BACKEND_STORAGE_CLASS}|g" \
| oc create --filename=-
oc create --filename="${SCRIPT_DIR}/30_hpp_pool_sc.yaml"

# Set HPP-CSI as default StorageClass for the cluster
oc annotate storageclasses --all storageclass.kubernetes.io/is-default-class-
oc annotate storageclass hostpath-csi storageclass.kubernetes.io/is-default-class='true'

# Wait for HPP to be ready
oc wait hostpathprovisioner hostpath-provisioner --for=condition='Available' --timeout='10m'
