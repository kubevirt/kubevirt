#!/bin/bash

set -e

PROJECT_ROOT="$(readlink -e $(dirname "$BASH_SOURCE[0]")/../)"

CLUSTER="${CLUSTER:-OPENSHIFT}"
MARKETPLACE_NAMESPACE="${MARKETPLACE_NAMESPACE:-openshift-marketplace}"
PACKAGE="${PACKAGE:-hco-operatorhub}"
APP_REGISTRY_NAMESPACE="${APP_REGISTRY_NAMESPACE:-kubevirt-hyperconverged}"

if [ "${CLUSTER}" == "KUBERNETES" ]; then
    MARKETPLACE_NAMESPACE="marketplace"
fi

if [ -z "${QUAY_USERNAME}" ]; then
    echo "QUAY_USERNAME"
    read QUAY_USERNAME
fi

if [ -z "${QUAY_PASSWORD}" ]; then
    echo "QUAY_PASSWORD"
    read -s QUAY_PASSWORD
fi

TOKEN=$("${PROJECT_ROOT}"/tools/token.sh $QUAY_USERNAME $QUAY_PASSWORD)

cat <<EOF | oc create -f -
apiVersion: v1
kind: Secret
metadata:
  name: quay-registry-$APP_REGISTRY_NAMESPACE
  namespace: "${MARKETPLACE_NAMESPACE}"
type: Opaque
stringData:
      token: "$TOKEN"
EOF

cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: "${APP_REGISTRY_NAMESPACE}"
  namespace: "${MARKETPLACE_NAMESPACE}"
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: $APP_REGISTRY_NAMESPACE
  displayName: "${APP_REGISTRY_NAMESPACE}"
  publisher: "Red Hat"
  authorizationToken:
    secretName: quay-registry-$APP_REGISTRY_NAMESPACE
EOF

cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: CatalogSourceConfig
metadata:
  name: hco-catalogsource-config
  namespace: "${MARKETPLACE_NAMESPACE}"
spec:
  targetNamespace: kubevirt-hyperconverged
  packages: "${PACKAGE}"
  csDisplayName: "CNV Operators"
  csPublisher: "Red Hat"
EOF
