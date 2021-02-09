#!/usr/bin/env bash

set -euo pipefail

INSTALLED_NAMESPACE=${INSTALLED_NAMESPACE:-"kubevirt-hyperconverged"}

source hack/common.sh
source cluster/kubevirtci.sh

export KUBECTL_BINARY="kubectl"

if [ "${JOB_TYPE}" == "stdci" ]; then
    KUBECONFIG=$(kubevirtci::kubeconfig)
    source ./hack/upgrade-stdci-config
    KUBECTL_BINARY="cluster/kubectl.sh"
fi

if [[ ${JOB_TYPE} = "prow" ]]; then
    KUBECTL_BINARY="oc"
    component=hyperconverged-cluster-functest
    computed_test_image=`eval echo ${IMAGE_FORMAT}`
else
    operator_image="$($KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" get pod -l name=hyperconverged-cluster-operator -o jsonpath='{.items[0] .spec .containers[?(@.name=="hyperconverged-cluster-operator")] .image}')"
    computed_test_image="${operator_image//hyperconverged-cluster-operator/hyperconverged-cluster-functest}"
fi

# the test image can be overwritten by the caller
FUNC_TEST_IMAGE=${FUNC_TEST_IMAGE:-${computed_test_image}}

echo "Running tests with $FUNC_TEST_IMAGE"

$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" create serviceaccount functest \
  --dry-run -o yaml  |$KUBECTL_BINARY apply -f -

$KUBECTL_BINARY create clusterrolebinding functest-cluster-admin \
    --clusterrole=cluster-admin \
    --serviceaccount="${INSTALLED_NAMESPACE}":functest \
    --dry-run -o yaml  |$KUBECTL_BINARY apply -f -

$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" delete pod functest --ignore-not-found --wait=true

$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" run functest \
 --image="$FUNC_TEST_IMAGE" --serviceaccount=functest \
 --env="INSTALLED_NAMESPACE=${INSTALLED_NAMESPACE}" \
 --restart=Never

phase="Running"
for i in $(seq 1 60); do
  phase=$($KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" get pod/functest -o jsonpath='{.status.phase}')

  if [[ "${phase}" == "Succeeded" || "${phase}" == "Failed" ]]; then
    break
  fi

  echo "Waiting for completion... Iteration:$i Phase:$phase"
  sleep 10
done

$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" logs functest

echo "Exiting... Last phase status: $phase"

# exit non-zero if the last phase is not Succeeded
[[ "${phase}" == "Succeeded" ]]

