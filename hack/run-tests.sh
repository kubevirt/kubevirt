#!/usr/bin/env bash

set -euo pipefail

INSTALLED_NAMESPACE=${INSTALLED_NAMESPACE:-"kubevirt-hyperconverged"}

source hack/common.sh
source cluster/kubevirtci.sh

if [ "${JOB_TYPE}" == "stdci" ]; then
    KUBECONFIG=$(kubevirtci::kubeconfig)
    source ./hack/upgrade-stdci-config
fi

if [[ ${JOB_TYPE} = "prow" ]]; then
    export KUBECTL_BINARY="oc"
else
    export KUBECTL_BINARY="cluster/kubectl.sh"
fi

# when the tests are run in a pod, in-cluster config will be used
KUBECONFIG_FLAG=""
if [[ -n "${KUBECONFIG-}" ]]; then
  KUBECONFIG_FLAG="-kubeconfig="${KUBECONFIG}""
fi

${TEST_OUT_PATH}/func-tests.test -ginkgo.v -installed-namespace="${INSTALLED_NAMESPACE}" -cdi-namespace="${INSTALLED_NAMESPACE}" "${KUBECONFIG_FLAG}"
exit 0
# wait a minute to allow all VMs to be deleted before attempting to change node placement configuration
sleep 60

# Check the webhook, to see if it allow updating of the HyperConverged CR
${KUBECTL_BINARY} patch hco -n "${INSTALLED_NAMESPACE}" kubevirt-hyperconverged -p '{"spec":{"infra":{"nodePlacement":{"tolerations":[{"effect":"NoSchedule","key":"key","operator":"Equal","value":"value"}]}}}}' --type=merge
${KUBECTL_BINARY} patch hco -n "${INSTALLED_NAMESPACE}" kubevirt-hyperconverged -p '{"spec":{"workloads":{"nodePlacement":{"tolerations":[{"effect":"NoSchedule","key":"key","operator":"Equal","value":"value"}]}}}}' --type=merge
# Read the HyperConverged CR
${KUBECTL_BINARY} get hco -n "${INSTALLED_NAMESPACE}" kubevirt-hyperconverged -o yaml

# wait a bit to make sure the VMs are deleted
sleep 60

KUBECTL_BINARY=${KUBECTL_BINARY} ./hack/test_quick_start.sh

./hack/retry.sh 10 30 "KUBECTL_BINARY=${KUBECTL_BINARY} ./hack/check_labels.sh"

# Check the webhook, to see if it allow deleteing of the HyperConverged CR
./hack/retry.sh 10 30 "${KUBECTL_BINARY} delete hco -n ${INSTALLED_NAMESPACE} kubevirt-hyperconverged"
