#!/usr/bin/env bash

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi

source ${KUBEVIRTCI_PATH}hack/common.sh
source ${KUBEVIRTCI_CLUSTER_PATH}/$KUBEVIRT_PROVIDER/provider.sh
down
