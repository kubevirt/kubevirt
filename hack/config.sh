unset binaries docker_images docker_prefix docker_tag manifest_templates \
    master_ip network_provider kubeconfig manifest_docker_prefix namespace image_pull_policy verbosity

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-${PROVIDER}}

source ${KUBEVIRT_PATH}hack/config-default.sh source ${KUBEVIRT_PATH}hack/config-${KUBEVIRT_PROVIDER}.sh

# Allow different providers to override default config values
test -f "hack/config-provider-${KUBEVIRT_PROVIDER}.sh" && source hack/config-provider-${KUBEVIRT_PROVIDER}.sh

# Let devs override any default variables, to avoid needing
# to change the version controlled config-default.sh file
test -f "hack/config-local.sh" && source hack/config-local.sh

export binaries docker_images docker_prefix docker_tag manifest_templates \
    master_ip network_provider kubeconfig namespace image_pull_policy verbosity
