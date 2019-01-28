#!/bin/bash

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

docker build -t kubevirt/builder:28-5.0.0 -f ${SCRIPT_DIR}/Dockerfile ${SCRIPT_DIR}
