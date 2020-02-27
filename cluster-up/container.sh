#!/usr/bin/env bash

set -e

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi

source ${KUBEVIRTCI_PATH}/hack/common.sh

if [[ $KUBEVIRT_PROVIDER =~ (ocp|okd).* ]]; then

    # If it's terminal make it interactive
    test -t 1 && USE_TTY="-it"

    docker exec $USE_TTY ${provider_prefix}-cluster $@
else
    echo "connect is supported only for ocp / okd, (or KUBEVIRT_PROVIDER isnt exported ?)"
fi
