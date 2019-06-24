#!/usr/bin/env bash

set -e

source hack/common.sh
source hack/config.sh

target=$1

if [ "${target}" = "push-cache" ]; then
    docker push kubevirt/builder-cache:${KUBEVIRT_UPDATE_CACHE_FROM}
fi

if [ "${target}" = "pull-cache" ]; then
    docker pull kubevirt/builder-cache:${KUBEVIRT_CACHE_FROM}
fi
