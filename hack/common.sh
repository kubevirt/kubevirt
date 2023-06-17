#!/usr/bin/env bash

determine_cri_bin() {
    if [ "${KUBEVIRT_CRI}" = "podman" ]; then
        echo podman
    elif [ "${KUBEVIRT_CRI}" = "docker" ]; then
        echo docker
    else
        if podman ps >/dev/null 2>&1; then
            echo podman
        elif docker ps >/dev/null 2>&1; then
            echo docker
        else
            echo ""
        fi
    fi
}

fail_if_cri_bin_missing() {
    if [ -z "${KUBEVIRT_CRI}" ]; then
        echo >&2 "no working container runtime found. Neither docker nor podman seems to work."
        exit 1
    fi
}

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
ARCHITECTURE="${ARCHITECTURE:-$(uname -m)}"
HOST_ARCHITECTURE="$(uname -m)"
KUBEVIRT_NO_BAZEL=${KUBEVIRT_NO_BAZEL:-false}
KUBEVIRT_RELEASE=${KUBEVIRT_RELEASE:-false}
OPERATOR_MANIFEST_PATH=$MANIFESTS_OUT_DIR/release/kubevirt-operator.yaml
TESTING_MANIFEST_PATH=$MANIFESTS_OUT_DIR/testing
KUBEVIRT_CRI="$(determine_cri_bin)"

function build_func_tests_image() {
    local bin_name=tests
    cp ${KUBEVIRT_DIR}/tests/{Dockerfile,entrypoint.sh} \
        ${KUBEVIRT_DIR}/tools/manifest-templator/manifest-templator \
        ${TESTS_OUT_DIR}/
    rsync -ar ${KUBEVIRT_DIR}/manifests/ ${TESTS_OUT_DIR}/manifests
    cd ${TESTS_OUT_DIR}
    ${KUBEVIRT_CRI} build \
        -t ${docker_prefix}/${bin_name}:${docker_tag} \
        --label ${job_prefix} \
        --label ${bin_name} .
}

# Use this environment variable to set a custom pkgdir path
# Useful for cross-compilation where the default -pkdir for cross-builds may not be writable
#KUBEVIRT_GO_BASE_PKGDIR="${GOPATH}/crossbuild-cache-root/"

# Use this environment variable to specify additional tags for the go build in hack/build-go.sh.
# To specify tags in the bazel build modify/overwrite the build target in .bazelrc instead.
if [ -z "$KUBEVIRT_GO_BUILD_TAGS" ]; then
    KUBEVIRT_GO_BUILD_TAGS="selinux"
else
    KUBEVIRT_GO_BUILD_TAGS="selinux,${KUBEVIRT_GO_BUILD_TAGS}"
fi

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

# We are formatting the architecture name here to ensure that
# it is consistent with the platform name specified in ../.bazelrc
# if the second argument is set, the function formats arch name for
# image tag.
function format_archname() {
    local local_platform=$(uname -m)
    local platform=$1
    local tag=$2

    if [ $# -lt 1 ]; then
        echo ${local_platform}
    else
        case ${platform} in
        x86_64 | amd64)
            [[ $tag ]] && echo "amd64" && return
            arch="x86_64"
            echo ${arch}
            ;;
        crossbuild-aarch64 | aarch64 | arm64)
            [[ $tag ]] && echo "arm64" && return
            if [ ${local_platform} != "aarch64" ]; then
                arch="crossbuild-aarch64"
            else
                arch="aarch64"
            fi
            echo ${arch}
            ;;
        *)
            echo "ERROR: invalid Arch, ${platform}, only support x86_64 and aarch64"
            exit 1
            ;;
        esac
    fi
}
