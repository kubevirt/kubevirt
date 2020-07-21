#!/bin/bash

echo "WARNING: THIS SCRIPT WILL BE DEPRECATED SOON. PLEASE USE kustomize/deploy_kustomize.sh INSTEAD."

set -ex

RED='\033[0;31m'
NO_COLOR='\033[0m'

globalNamespace=`oc -n openshift-operator-lifecycle-manager get deployments catalog-operator -o jsonpath='{.spec.template.spec.containers[].args[1]}'`
echo "Global Namespace: ${globalNamespace}"

PACKAGE="${PACKAGE:-kubevirt-hyperconverged}"
CS_SOURCE="${CS_SOURCE:-hco-catalogsource}"

TARGET_NAMESPACE="${TARGET_NAMESPACE:-kubevirt-hyperconverged}"
MARKETPLACE_NAMESPACE="${MARKETPLACE_NAMESPACE:-openshift-marketplace}"
HCO_REGISTRY_IMAGE="${HCO_REGISTRY_IMAGE:-quay.io/kubevirt/hco-container-registry:latest}"
POD_TIMEOUT="${POD_TIMEOUT:-360s}"
HCO_VERSION="${HCO_VERSION:-1.0.0}"
HCO_CHANNEL="${HCO_CHANNEL:-1.0.0}"

CLUSTER="${CLUSTER:-OPENSHIFT}"

GLOBAL_NAMESPACE="${GLOBAL_NAMESPACE:-$globalNamespace}"

APPROVAL="${APPROVAL:-Manual}"
CONTENT_ONLY="${CONTENT_ONLY:-}"
KVM_EMULATION="${KVM_EMULATION:-false}"

RETRIES="${RETRIES:-10}"

oc create ns $TARGET_NAMESPACE || true

if [ "${CLUSTER}" == "KUBERNETES" ]; then
    MARKETPLACE_NAMESPACE="marketplace"
fi

TMP_DIR=$(mktemp -d)

function cleanup_tmp {
    rm -rf $TMP_DIR
}

trap cleanup_tmp EXIT

cleanup_tmp

# Create a Catalog Source backed by a grpc registry
cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${CS_SOURCE}
  namespace: "${MARKETPLACE_NAMESPACE}"
  imagePullPolicy: Always
spec:
  sourceType: grpc
  image: ${HCO_REGISTRY_IMAGE}
  displayName: KubeVirt HyperConverged
  publisher: Red Hat
EOF

echo "Waiting up to ${POD_TIMEOUT} for catalogsource to appear..."
sleep 5
oc wait pods -n "${MARKETPLACE_NAMESPACE}" -l olm.catalogSource="${CS_SOURCE}" --for condition=Ready --timeout="${POD_TIMEOUT}"

echo "Give the cluster 30 seconds to process catalogSource..."
sleep 30

for i in $(seq 1 $RETRIES); do
    echo "Waiting for packagemanifest '${PACKAGE}' to be created in namespace '${TARGET_NAMESPACE}'..."
    oc get packagemanifest -n "${TARGET_NAMESPACE}" "${PACKAGE}" && break
    sleep $i
    if [ "$i" -eq "${RETRIES}" ]; then
      echo "packagemanifest '${PACKAGE}' was never created in namespace '${TARGET_NAMESPACE}'"
      exit 1
    fi
done

SUBSCRIPTION_CONFIG=""
if [ "$KVM_EMULATION" = true ]; then
  SUBSCRIPTION_CONFIG=$(cat <<EOF
  config:
    selector:
      matchLabels:
        name: hyperconverged-cluster-operator
    env:
      - name: KVM_EMULATION
        value: "true"
EOF
)
fi

echo "Content Successfully Created"
if [ -z "${CONTENT_ONLY}" ]; then

    if [ `oc get operatorgroup -n "${TARGET_NAMESPACE}" 2> /dev/null | wc -l` -eq 0 ]; then
    echo "Creating OperatorGroup"
    cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: "${TARGET_NAMESPACE}-group"
  namespace: "${TARGET_NAMESPACE}"
spec:
  targetNamespaces:
  - "${TARGET_NAMESPACE}"
EOF
    fi

    echo "Creating Subscription"
    cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-subscription
  namespace: "${TARGET_NAMESPACE}"
spec:
  source: "${CS_SOURCE}"
  sourceNamespace: "${GLOBAL_NAMESPACE}"
  name: kubevirt-hyperconverged
  startingCSV: "kubevirt-hyperconverged-operator.v${HCO_VERSION}"
  channel: "${HCO_CHANNEL}"
  installPlanApproval: "${APPROVAL}"
${SUBSCRIPTION_CONFIG}
EOF

    echo "Give OLM 60 seconds to process the subscription..."
    sleep 60

    oc get installplan -o yaml -n "${TARGET_NAMESPACE}" $(oc get installplan -n "${TARGET_NAMESPACE}" --no-headers | grep "kubevirt-hyperconverged-operator.v${HCO_VERSION}" | awk '{print $1}') | sed 's/approved: false/approved: true/' | oc apply -n "${TARGET_NAMESPACE}" -f -

    echo "Give OLM 60 seconds to process the installplan..."
    sleep 60

    oc wait pod $(oc get pods -n ${TARGET_NAMESPACE} | grep hco-operator | head -1 | awk '{ print $1 }') --for condition=Ready -n ${TARGET_NAMESPACE} --timeout="360s"

    echo "Creating the HCO's Custom Resource"
    cat <<EOF | oc create -f -
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: kubevirt-hyperconverged
  namespace: "${TARGET_NAMESPACE}"
spec:
  BareMetalPlatform: true
EOF

    echo "Waiting for HCO to get fully deployed"
    oc wait -n ${TARGET_NAMESPACE} hyperconverged kubevirt-hyperconverged --for condition=Available --timeout=15m
    oc wait "$(oc get pods -n ${TARGET_NAMESPACE} -l name=hyperconverged-cluster-operator -o name)" -n "${TARGET_NAMESPACE}" --for condition=Ready --timeout=15m
fi
