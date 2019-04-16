unset binaries docker_images docker_prefix docker_tag docker_tag_alt manifest_templates \
    master_ip network_provider kubeconfig manifest_docker_prefix namespace image_pull_policy verbosity \
    csv_version package_name

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-${PROVIDER}}

source ${KUBEVIRT_PATH}hack/config-default.sh

# Allow different providers to override default config values
test -f "${KUBEVIRT_PATH}hack/config-provider-${KUBEVIRT_PROVIDER}.sh" && source ${KUBEVIRT_PATH}hack/config-provider-${KUBEVIRT_PROVIDER}.sh

# Let devs override any default variables, to avoid needing
# to change the version controlled config-default.sh file
test -f "${KUBEVIRT_PATH}hack/config-local.sh" && source ${KUBEVIRT_PATH}hack/config-local.sh

export binaries docker_images docker_prefix docker_tag docker_tag_alt manifest_templates \
    master_ip network_provider kubeconfig manifest_docker_prefix namespace image_pull_policy verbosity \
    csv_version package_name
