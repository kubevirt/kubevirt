#!/usr/bin/env bash

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

. ${SCRIPT_DIR}/version.sh

docker tag kubevirt/builder:${VERSION} docker.io/kubevirt/builder:${VERSION}
docker push docker.io/kubevirt/builder:${VERSION}
