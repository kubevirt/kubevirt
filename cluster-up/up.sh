#!/usr/bin/env bash

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi


source ${KUBEVIRTCI_PATH}hack/common.sh
source ${KUBEVIRTCI_CLUSTER_PATH}/$KUBEVIRT_PROVIDER/provider.sh
up

# check if the environment has a corrupted host
if [[ $(${KUBEVIRTCI_PATH}kubectl.sh get nodes | grep localhost) != "" ]]; then
    echo "The environment has a corrupted host"
    exit 1
fi
