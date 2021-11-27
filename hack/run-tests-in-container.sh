#!/usr/bin/env bash

set -exuo pipefail

INSTALLED_NAMESPACE=${INSTALLED_NAMESPACE:-"kubevirt-hyperconverged"}
OUTPUT_DIR=${ARTIFACT_DIR:-"$(pwd)/_out"}

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
    computed_test_image=${FUNCTEST_IMAGE}
else
    operator_image="$($KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" get pod -l name=hyperconverged-cluster-operator -o jsonpath='{.items[0] .spec .containers[?(@.name=="hyperconverged-cluster-operator")] .image}')"
    computed_test_image="${operator_image//hyperconverged-cluster-operator/hyperconverged-cluster-functest}"
fi

function cleanup() {
    rv=$?
    if [ "x$rv" != "x0" ]; then
        echo "Error during upgrade: exit status: $rv"
        CMD=${KUBECTL_BINARY} make dump-state
        echo "*** Upgrade test failed ***"
    fi
    exit $rv
}

trap "cleanup" INT TERM EXIT

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

cat <<EOF | $KUBECTL_BINARY create -f -
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: functest
  name: functest
  namespace: $INSTALLED_NAMESPACE
spec:
  containers:
  - name: functest
    args:
    - --config-file
    - hack/testFiles/test_config.yaml
    env:
    - name: INSTALLED_NAMESPACE
      value: $INSTALLED_NAMESPACE
    image: $computed_test_image
    volumeMounts:
      - mountPath: /test/output
        name: output-volume
  - name: copy
    image: $computed_test_image
    command: ["/bin/sh"]
    args: [ "-c", "trap : TERM INT; sleep infinity & wait"]
    volumeMounts:
      - mountPath: /test/output
        name: output-volume
  volumes:
    - name: output-volume
      emptyDir: { }
  serviceAccount: functest
  restartPolicy: Never
EOF

for i in $(seq 1 120); do
  exitCode=$($KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" get pod/functest -o jsonpath='{.status .containerStatuses[?(@.name=="functest")] .state .terminated .exitCode}')

  if [[ -n ${exitCode} ]]; then
    break
  fi

  echo "Waiting for completion... Iteration:$i"
  sleep 10
done

if [[ -z ${exitCode} ]]; then
  $KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" get pod functest -o yaml || true
fi

$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" cp -c=copy functest:/test/output "$OUTPUT_DIR"
$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" logs functest -c functest
$KUBECTL_BINARY -n "${INSTALLED_NAMESPACE}" delete pod functest --ignore-not-found --wait=true


echo "Exiting... Exit code: $exitCode"

# exit non-zero if exit code of functest is not zero
[[ "${exitCode}" == "0" ]]

