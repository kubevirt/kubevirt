#!/bin/bash

set -ex

RED='\033[0;31m'
NO_COLOR='\033[0m'

globalNamespace=`oc -n openshift-operator-lifecycle-manager get deployments catalog-operator -o jsonpath='{.spec.template.spec.containers[].args[1]}'`
echo "Global Namespace: ${globalNamespace}"

APP_REGISTRY="${APP_REGISTRY:-kubevirt-hyperconverged}"
PACKAGE="${PACKAGE:-kubevirt-hyperconverged}"
CSC_SOURCE="${CSC_SOURCE:-hco-catalogsource-config}"
TARGET_NAMESPACE="${TARGET_NAMESPACE:-kubevirt-hyperconverged}"
CLUSTER="${CLUSTER:-OPENSHIFT}"
MARKETPLACE_NAMESPACE="${MARKETPLACE_NAMESPACE:-openshift-marketplace}"
GLOBAL_NAMESPACE="${GLOBAL_NAMESPACE:-$globalNamespace}"
HCO_VERSION="${HCO_VERSION:-1.0.0}"
HCO_CHANNEL="${HCO_CHANNEL:-1.0.0}"
APPROVAL="${APPROVAL:-Manual}"
CONTENT_ONLY="${CONTENT_ONLY:-}"
KVM_EMULATION="${KVM_EMULATION:-false}"
PRIVATE_REPO="${PRIVATE_REPO:-false}"
QUAY_USERNAME="${QUAY_USERNAME:-}"
QUAY_PASSWORD="${QUAY_PASSWORD:-}"
QUAY_TOKEN="${QUAY_TOKEN:-}"

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

AUTH_TOKEN=""

if [ "$PRIVATE_REPO" = true ]; then
  if [ -z "${QUAY_TOKEN}" ]; then
      if [ -z "${QUAY_USERNAME}" ]; then
          echo "QUAY_USERNAME is unset"
          exit 1
      fi

      if [ -z "${QUAY_PASSWORD}" ]; then
          echo "QUAY_PASSWORD is unset"
          exit 1
      fi

      QUAY_TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
  {
      "user": {
          "username": "'"${QUAY_USERNAME}"'",
          "password": "'"${QUAY_PASSWORD}"'"
      }
  }' | jq -r '.token')

      echo $QUAY_TOKEN
      if [ "${QUAY_TOKEN}" == "null" ]; then
          echo "QUAY_TOKEN was 'null'.  Did you enter the correct quay Username & Password?"
          exit 1
      fi
  fi

  echo "Creating registry secret"
  cat <<EOF | oc create -f -
apiVersion: v1
kind: Secret
metadata:
  name: "quay-registry-${APP_REGISTRY}"
  namespace: "${MARKETPLACE_NAMESPACE}"
type: Opaque
stringData:
      token: "$QUAY_TOKEN"
EOF

  AUTH_TOKEN=$(cat <<EOF
  authorizationToken:
    secretName: "quay-registry-${APP_REGISTRY}"
EOF
)

fi

if [ `oc get OperatorSource "${APP_REGISTRY}" -n "${MARKETPLACE_NAMESPACE}" --no-headers 2> /dev/null | wc -l` -eq 0 ]; then
    echo "Creating OperatorSource"
    cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: "${APP_REGISTRY}"
  namespace: "${MARKETPLACE_NAMESPACE}"
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: "${APP_REGISTRY}"
  displayName: "${APP_REGISTRY}"
  publisher: "Kubevirt"
${AUTH_TOKEN}
EOF
fi

echo "Give the cluster 30 seconds to create the catalogSourceConfig..."
sleep 30

cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: CatalogSourceConfig
metadata:
  name: "${CSC_SOURCE}"
  namespace: "${MARKETPLACE_NAMESPACE}"
spec:
  source: "${APP_REGISTRY}"
  targetNamespace: "${GLOBAL_NAMESPACE}"
  packages: "${PACKAGE}"
  csDisplayName: "HCO Operator"
  csPublisher: "Red Hat"
EOF

echo "Give the cluster 30 seconds to process catalogSourceConfig..."
sleep 30
oc wait deploy $CSC_SOURCE --for condition=available -n $MARKETPLACE_NAMESPACE --timeout="360s"

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

    if [ `oc get operatorgroup -n "${TARGET_NAMESPACE}" --no-headers 2> /dev/null | wc -l` -eq 0 ]; then
    echo "Creating OperatorGroup"
    cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: "${TARGET_NAMESPACE}-group"
  namespace: "${TARGET_NAMESPACE}"
spec: {}
EOF
    fi

    echo "Creating Subscription"
    cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-operatorhub
  namespace: "${TARGET_NAMESPACE}"
spec:
  source: "${CSC_SOURCE}"
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
apiVersion: hco.kubevirt.io/v1alpha1
kind: HyperConverged
metadata:
  name: hyperconverged-cluster
  namespace: "${TARGET_NAMESPACE}"
spec:
  BareMetalPlatform: true
EOF

    echo "Waiting for HCO to get fully deployed"
    oc wait -n ${TARGET_NAMESPACE} hyperconverged hyperconverged-cluster --for condition=Available --timeout=15m
fi
