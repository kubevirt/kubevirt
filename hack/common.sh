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
MANIFEST_TEMPLATES_OUT_DIR=$OUT_DIR/templates/manifests
PYTHON_CLIENT_OUT_DIR=$OUT_DIR/client-python

function build_func_tests() {
    mkdir -p ${TESTS_OUT_DIR}/
    ginkgo build ${KUBEVIRT_DIR}/tests
    mv ${KUBEVIRT_DIR}/tests/tests.test ${TESTS_OUT_DIR}/
}
function build_func_tests_container() {
    local bin_name=tests
    cp ${KUBEVIRT_DIR}/tests/{Dockerfile,entrypoint.sh} \
        ${KUBEVIRT_DIR}/tools/manifest-templator/manifest-templator \
        ${TESTS_OUT_DIR}/
    rsync -ar ${KUBEVIRT_DIR}/manifests/ ${TESTS_OUT_DIR}/manifests
    cd ${TESTS_OUT_DIR}
    docker build \
        -t ${docker_prefix}/${bin_name}:${docker_tag} \
        --label ${job_prefix} \
        --label ${bin_name} .
}

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-k8s-1.10.11}
KUBEVIRT_NUM_NODES=${KUBEVIRT_NUM_NODES:-1}
KUBEVIRT_MEMORY_SIZE=${KUBEVIRT_MEMORY_SIZE:-5120M}

# If set to the name of a branch, the builds will try to use an image of the form kubevirt/{name}:{branche}
# as cache source (--cache-from)
KUBEVIRT_CACHE_FROM=${KUBEVIRT_CACHE_FROM}
# Push images in the form kubevirt/{name}:{branche} to update the build cache for this branch
KUBEVIRT_UPDATE_CACHE_FROM=${KUBEVIRT_UPDATE_CACHE_FROM}

# Use this environment variable to set a custom pkgdir path
# Useful for cross-compilation where the default -pkdir for cross-builds may not be writable
#KUBEVIRT_GO_BASE_PKGDIR="${GOPATH}/crossbuild-cache-root/"

# If on a developer setup, expose ocp on 8443, so that the openshift web console can be used (the port is important because of auth redirects)
if [ -z "${JOB_NAME}" ]; then
    KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --ocp-port 8443"
fi

#If run on jenkins, let us create isolated environments based on the job and
# the executor number
provider_prefix=${JOB_NAME:-${KUBEVIRT_PROVIDER}}${EXECUTOR_NUMBER}
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
