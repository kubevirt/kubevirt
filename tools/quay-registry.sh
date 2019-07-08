#!/bin/bash

set -e

QUAY_USERNAME="${1:-}"
QUAY_PASSWORD="${2:-}"

CLUSTER="${CLUSTER:-OPENSHIFT}"
MARKETPLACE_NAMESPACE="${MARKETPLACE_NAMESPACE:-openshift-marketplace}"
PACKAGE="${PACKAGE:-hco-operatorhub}"
APP_REGISTRY_NAMESPACE="${APP_REGISTRY_NAMESPACE:-kubevirt-hyperconverged}"

if [ "${CLUSTER}" == "KUBERNETES" ]; then
    MARKETPLACE_NAMESPACE="marketplace"
fi

if [ -z "${QUAY_USERNAME}" ]; then
    echo "QUAY_USERNAME not set"
    exit 1
fi
if [ -z "${QUAY_PASSWORD}" ]; then
    echo "QUAY_PASSWORD not set"
    exit 1
fi

TOKEN=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
{
    "user": {
        "username": "'"${QUAY_USERNAME}"'",
        "password": "'"${QUAY_PASSWORD}"'"
    }
}' | jq -r '.token')

echo $TOKEN
if [ "${TOKEN}" == "null" ]; then
   echo "TOKEN was 'null'.  Did you enter the correct quay Username & Password?"
   exit 1
fi

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
