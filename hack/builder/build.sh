#!/usr/bin/env bash

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

. ${SCRIPT_DIR}/version.sh

docker build -t kubevirt/builder:${VERSION} -f ${SCRIPT_DIR}/Dockerfile ${SCRIPT_DIR}
