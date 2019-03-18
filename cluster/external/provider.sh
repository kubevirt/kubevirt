#!/usr/bin/env bash

function _kubectl() {
    kubectl "$@"
}

function prepare_config() {
    cat >hack/config-provider-external.sh <<EOF
docker_tag=devel
docker_prefix=${DOCKER_PREFIX}
manifest_docker_prefix=${DOCKER_PREFIX}
image_pull_policy=${IMAGE_PULL_POLICY:-Always}
EOF
}

# The external cluster is assumed to be up.  Do a simple check
function up() {
    prepare_config
    _kubectl version >/dev/null
    if [ $? -ne 0 ]; then
        echo -e "\n*** Unable to reach external cluster.  Please check configuration ***"
        echo -e "*** Type \"kubectl config view\" for current settings               ***\n"
        exit 1
    fi
    echo "Cluster is up"
}

function down() {
    echo "Not supported by this provider"
}

function build() {
    prepare_config
    # Build code and manifests
    ${KUBEVIRT_PATH}hack/dockerized "DOCKER_TAG=${DOCKER_TAG} DOCKER_PREFIX=${docker_prefix} KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER} ./hack/build-manifests.sh"
    DOCKER_PREFIX=${docker_prefix} DOCKER_TAG=${docker_tag} make build bazel-push-images
}
