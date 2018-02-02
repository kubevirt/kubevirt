#!/bin/bash

KUBEVIRT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../"
    pwd
)"
OUT_DIR=$KUBEVIRT_DIR/_out/
CMD_OUT_DIR=$KUBEVIRT_DIR/_out/cmd/
TESTS_OUT_DIR=$KUBEVIRT_DIR/_out/tests/
APIDOCS_OUT_DIR=$KUBEVIRT_DIR/_out/apidocs
MANIFESTS_OUT_DIR=$KUBEVIRT_DIR/_out/manifests
PYTHON_CLIENT_OUT_DIR=$KUBEVIRT_DIR/_out/client-python

function build_func_tests() {
    mkdir -p ${TESTS_OUT_DIR}/
    ginkgo build ${KUBEVIRT_DIR}/tests
    mv ${KUBEVIRT_DIR}/tests/tests.test ${TESTS_OUT_DIR}/
}
