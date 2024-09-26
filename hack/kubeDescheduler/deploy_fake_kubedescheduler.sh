#!/bin/bash

#
# Configures, for testing purposes, fake kubeDesheduler CRD and CR to mimic kubeDesheduler APIs when not there
#

set -ex

readonly SCRIPT_DIR=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")

KUBEDES_CRD_NUM=$(
  oc get crds --field-selector=metadata.name=kubedeschedulers.operator.openshift.io -o=name | wc -l
)

if [[ "KUBEDES_CRD_NUM" -eq 0 ]]; then
  echo "Create a CRD for a fake KubeDescheduler"
  oc apply -f "${SCRIPT_DIR}/kube-descheduler-operator.crd.yaml"
  KUBEDES_NS_NUM=$(
    oc get namespaces --field-selector=metadata.name=openshift-kube-descheduler-operator -o=name | wc -l
  )
  if [[ "KUBEDES_NS_NUM" -eq 0 ]]; then
    echo "Creating a namespace for KubeDescheduler"
    oc create namespace openshift-kube-descheduler-operator
  fi
  KUBEDES_CR_NUM=$(
    oc get kubedeschedulers -n=openshift-kube-descheduler-operator --field-selector=metadata.name=cluster -o=name | wc -l
  )
  if [[ "KUBEDES_CR_NUM" -eq 0 ]]; then
    echo "Create a CR for a fake KubeDescheduler"
    oc apply -f "${SCRIPT_DIR}/kubeDescheduler.cr.yaml"
  fi
fi
