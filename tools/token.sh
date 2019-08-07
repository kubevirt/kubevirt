#!/bin/bash

set -e

QUAY_USERNAME="${1:-}"
QUAY_PASSWORD="${2:-}"

APP_REGISTRY_NAMESPACE="${APP_REGISTRY_NAMESPACE:-rh-osbs-operators}"
if [ -z "${QUAY_USERNAME}" ]; then
    echo "QUAY_USERNAME"
    read QUAY_USERNAME
fi

if [ -z "${QUAY_PASSWORD}" ]; then
    echo "QUAY_PASSWORD"
    read -s QUAY_PASSWORD
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
