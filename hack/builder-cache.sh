#!/usr/bin/env bash

set -e

source hack/common.sh
source hack/config.sh

target=$1

if [ "${target}" = "push" ]; then
    docker push kubevirt/builder-cache:${KUBEVIRT_UPDATE_CACHE_FROM}
fi

if [ "${target}" = "pull" ]; then
    docker pull kubevirt/builder-cache:${KUBEVIRT_CACHE_FROM}
fi
