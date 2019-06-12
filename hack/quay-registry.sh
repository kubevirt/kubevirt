#!/bin/bash

set -e

QUAY_USERNAME="${1:-}"
QUAY_PASSWORD="${2:-}"

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
  name: quay-registry-$REGISTRY_NAMESPACE
  namespace: openshift-marketplace
type: Opaque
stringData:
      token: "$TOKEN"
EOF

cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: "${REGISTRY_NAMESPACE}"
  namespace: openshift-marketplace
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: $REGISTRY_NAMESPACE
  displayName: "${REGISTRY_NAMESPACE}"
  publisher: "Red Hat"
  authorizationToken:
    secretName: quay-registry-$REGISTRY_NAMESPACE
EOF

cat <<EOF | oc create -f -
apiVersion: operators.coreos.com/v1
kind: CatalogSourceConfig
metadata:
  name: hco-catalogsource
  namespace: openshift-marketplace
spec:
  targetNamespace: kubevirt-hyperconverged
  packages: kubevirt-hyperconverged
  csDisplayName: "CNV Operators"
  csPublisher: "Red Hat"
EOF
