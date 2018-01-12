#!/bin/bash

KUBEVIRT_DIR="$(
    cd "$(dirname "$0")/../"
    pwd
)"
OUT_DIR=$KUBEVIRT_DIR/_out/
CMD_OUT_DIR=$KUBEVIRT_DIR/_out/cmd/
TESTS_OUT_DIR=$KUBEVIRT_DIR/_out/cmd/
APIDOCS_OUT_DIR=$KUBEVIRT_DIR/_out/apidocs
MANIFESTS_OUT_DIR=$KUBEVIRT_DIR/_out/manifests
