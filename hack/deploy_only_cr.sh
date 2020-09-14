#!/bin/bash -ex

HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"
HCO_DEPLOYMENT_NAME=hco-operator

echo "KUBEVIRT_PROVIDER: $KUBEVIRT_PROVIDER"

if [ -n "$KUBEVIRT_PROVIDER" ]; then
  echo "Running on STDCI ${KUBEVIRT_PROVIDER}"
  source ./hack/upgrade-stdci-config
else
  echo "Running on OpenShift CI"
  source ./hack/upgrade-openshiftci-config
fi

function cleanup() {
    rv=$?
    if [ "x$rv" != "x0" ]; then
        echo "Error during HCO CR deployment: exit status: $rv"
        make dump-state
        echo "*** HCO CR deployment failed ***"
    fi
    exit $rv
}

trap "cleanup" INT TERM EXIT

if [[ -n "${KVM_EMULATION}" ]]; then
  SUBSCRIPTION_NAME=$(oc get subscription -n "${HCO_NAMESPACE}" -o name)
  # cut the type prefix, e.g. subscription.operators.coreos.com/kubevirt-hyperconverged => kubevirt-hyperconverged
  SUBSCRIPTION_NAME=${SUBSCRIPTION_NAME/*\//}

  TMP_DIR=$(mktemp -d)
  cat > "${TMP_DIR}/subscription-patch.yaml" << EOF
spec:
  config:
    selector:
      matchLabels:
        name: hyperconverged-cluster-operator
    env:
    - name: 'KVM_EMULATION'
      value: "${KVM_EMULATION}"
EOF

  ${CMD} patch -n "${HCO_NAMESPACE}" Subscription "${SUBSCRIPTION_NAME}" --patch="$(cat "${TMP_DIR}/subscription-patch.yaml")" --type=merge

  # give it some time to take place
  sleep 60
  # wait for the HCO to run with the new configurations
  ${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"
fi

${CMD} apply -n kubevirt-hyperconverged -f deploy/hco.cr.yaml

${CMD} wait -n "${HCO_NAMESPACE}" "${HCO_KIND}" "${HCO_RESOURCE_NAME}" --for condition=Available --timeout="30m"
