#!/usr/bin/env bash
set -ex

# edit priority class labels, causing HCO to recreate it, making sure that the relatedObjects list contains the new priorityClass

# check that the new priority class was created and that it does not include the new label
function check_priority_class() {
  set -ex
  OLD_PC_UID=$1

  # make sure the object was actually changed
  NEW_PC_UID=$(${KUBECTL_BINARY} get priorityclass -n ${INSTALLED_NAMESPACE} kubevirt-cluster-critical -o jsonpath='{.metadata.uid}')
  [[ "${OLD_PC_UID}" != "${NEW_PC_UID}" ]]

  # read the UID from the HyperConverged relatedObject list
  NEW_PC_REF_UID=$(${KUBECTL_BINARY} get hco -n ${INSTALLED_NAMESPACE} kubevirt-hyperconverged -o json | jq -r '.status.relatedObjects[] | select(.kind == "PriorityClass" and .name == "kubevirt-cluster-critical") | .uid')

  [[ "${NEW_PC_UID}" == "${NEW_PC_REF_UID}" ]]

  # make sure the new label was removed:
  TEST_LABEL=$(${KUBECTL_BINARY} get priorityclass -n ${INSTALLED_NAMESPACE} kubevirt-cluster-critical -o jsonpath='{.metadata.labels.test}')
  [[ -z "${TEST_LABEL}" ]]
}

export -f check_priority_class

OLD_PC_REF_UID=$(${KUBECTL_BINARY} get hco -n ${INSTALLED_NAMESPACE} kubevirt-hyperconverged -o json | jq -r '.status.relatedObjects[] | select(.kind == "PriorityClass" and .name == "kubevirt-cluster-critical") | .uid')
OLD_PC_UID=$(${KUBECTL_BINARY} get priorityclass -n ${INSTALLED_NAMESPACE} kubevirt-cluster-critical -o jsonpath='{.metadata.uid}')
[[ "${OLD_PC_REF_UID}" == "${OLD_PC_UID}" ]]

${KUBECTL_BINARY} patch priorityclass kubevirt-cluster-critical -n ${INSTALLED_NAMESPACE} --type=json -p='[{"op": "add", "path": "/metadata/labels/test", "value": "test"}]'
./hack/retry.sh 3 10 "check_priority_class ${OLD_PC_UID}"