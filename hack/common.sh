#!/bin/bash

KUBEVIRT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../"
    pwd
)"
OUT_DIR=$KUBEVIRT_DIR/_out
VENDOR_DIR=$KUBEVIRT_DIR/vendor
CMD_OUT_DIR=$OUT_DIR/cmd
TESTS_OUT_DIR=$OUT_DIR/tests
APIDOCS_OUT_DIR=$OUT_DIR/apidocs
MANIFESTS_OUT_DIR=$OUT_DIR/manifests
PYTHON_CLIENT_OUT_DIR=$OUT_DIR/client-python

function build_func_tests() {
    mkdir -p ${TESTS_OUT_DIR}/
    ginkgo build ${KUBEVIRT_DIR}/tests
    mv ${KUBEVIRT_DIR}/tests/tests.test ${TESTS_OUT_DIR}/
}

#If run on jenkins, let us create isolated environments based on the job and
# the executor number
PROVIDER=${PROVIDER:-vagrant-kubernetes}
provider_prefix=${JOB_NAME:-${PROVIDER}}${EXECUTOR_NUMBER}
job_prefix=${JOB_NAME:-kubevirt}${EXECUTOR_NUMBER}
