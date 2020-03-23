#!/usr/bin/env bash

function _kubectl() {
    kubectl "$@"
}

function prepare_config() {
    BASE_PATH=${KUBEVIRTCI_CONFIG_PATH:-$PWD}

    if [ -z "${KUBECONFIG}" ]; then
        echo "KUBECONFIG is not set!"
        exit 1
    fi

    PROVIDER_CONFIG_FILE_PATH="${BASE_PATH}/$KUBEVIRT_PROVIDER/config-provider-$KUBEVIRT_PROVIDER.sh"

    cat > "$PROVIDER_CONFIG_FILE_PATH" <<EOF
kubeconfig=\${KUBECONFIG}
docker_tag=\${DOCKER_TAG}
docker_prefix=\${DOCKER_PREFIX}
manifest_docker_prefix=\${DOCKER_PREFIX}
image_pull_policy=\${IMAGE_PULL_POLICY:-Always}
EOF

    if which oc; then
        echo "oc=$(which oc)" >> "$PROVIDER_CONFIG_FILE_PATH"
    fi

}

# The external cluster is assumed to be up.  Do a simple check
function up() {
    prepare_config
    if ! _kubectl version >/dev/null; then
        echo -e "\n*** Unable to reach external cluster.  Please check configuration ***"
        echo -e "*** Type \"kubectl config view\" for current settings               ***\n"
        exit 1
    fi
    echo "Cluster is up"
}

function down() {
    echo "Not supported by this provider"
}

