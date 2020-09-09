#!/bin/bash -e

HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"

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
        echo "Error during upgrade: exit status: $rv"
        make dump-state
        echo "*** Upgrade test failed ***"
    fi
    exit $rv
}

trap "cleanup" INT TERM EXIT

${CMD} apply -n kubevirt-hyperconverged -f deploy/hco.cr.yaml

${CMD} wait -n "${HCO_NAMESPACE}" "${HCO_KIND}" "${HCO_RESOURCE_NAME}" --for condition=Available --timeout="30m"
${CMD} wait deployment "${HCO_DEPLOYMENT_NAME}" --for condition=Available -n "${HCO_NAMESPACE}" --timeout="30m"
