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

# Populate an environment variable with the version info needed.
# It should be used for everything which needs a version when building (not generating)
# IMPORTANT:
# RIGHT NOW ONLY RELEVANT FOR BUILDING, GENERATING CODE OUTSIDE OF GIT
# IS NOT NEEDED NOR RECOMMENDED AT THIS STAGE.

function kubevirt_version() {
    if [ -n "${KUBEVIRT_VERSION}" ]; then
        echo ${KUBEVIRT_VERSION}
    elif [ -d ${KUBEVIRT_DIR}/.git ]; then
        echo "$(git describe --always --tags)"
    else
        echo "undefined"
    fi
}
KUBEVIRT_VERSION="$(kubevirt_version)"
