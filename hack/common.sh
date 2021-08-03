#!/usr/bin/env bash

if [ -f cluster-up/hack/common.sh ]; then
    source cluster-up/hack/common.sh
fi

export GOFLAGS="$GOFLAGS -mod=vendor"

KUBEVIRT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../"
    pwd
)"
OUT_DIR=$KUBEVIRT_DIR/_out
SANDBOX_DIR=${KUBEVIRT_DIR}/.bazeldnf/sandbox
VENDOR_DIR=$KUBEVIRT_DIR/vendor
CMD_OUT_DIR=$OUT_DIR/cmd
TESTS_OUT_DIR=$OUT_DIR/tests
APIDOCS_OUT_DIR=$OUT_DIR/apidocs
ARTIFACTS=${ARTIFACTS:-${OUT_DIR}/artifacts}
DIGESTS_DIR=${OUT_DIR}/digests
MANIFESTS_OUT_DIR=$OUT_DIR/manifests
MANIFEST_TEMPLATES_OUT_DIR=$OUT_DIR/templates/manifests
PYTHON_CLIENT_OUT_DIR=$OUT_DIR/client-python
ARCHITECTURE="${BUILD_ARCH:-$(uname -m)}"
HOST_ARCHITECTURE="$(uname -m)"
KUBEVIRT_NO_BAZEL=${KUBEVIRT_NO_BAZEL:-false}

function build_func_tests() {
    mkdir -p "${TESTS_OUT_DIR}/"
    GOPROXY=off \
        go test -c "${KUBEVIRT_DIR}/tests" -o "${TESTS_OUT_DIR}/tests.test"
}

function build_func_tests_image() {
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

# Use this environment variable to set a custom pkgdir path
# Useful for cross-compilation where the default -pkdir for cross-builds may not be writable
#KUBEVIRT_GO_BASE_PKGDIR="${GOPATH}/crossbuild-cache-root/"

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

function go_build() {
    GOPROXY=off go build "$@"
}

# Use this environment variable to set a local path to a custom CA certificate for
# a private HTTPS docker registry. The intention is that this will be merged with the trust
# store in the build environment.

DOCKER_CA_CERT_FILE="${DOCKER_CA_CERT_FILE:-}"
DOCKERIZED_CUSTOM_CA_PATH="/etc/pki/ca-trust/source/anchors/custom-ca.crt"
