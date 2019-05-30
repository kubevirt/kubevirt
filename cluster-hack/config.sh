unset docker_prefix master_ip network_provider kubeconfig manifest_docker_prefix

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-${PROVIDER}}

source ${KUBEVIRT_PATH}cluster-hack/config-default.sh

# Allow different providers to override default config values
test -f "${KUBEVIRT_PATH}cluster-hack/config-provider-${KUBEVIRT_PROVIDER}.sh" && source ${KUBEVIRT_PATH}cluster-hack/config-provider-${KUBEVIRT_PROVIDER}.sh

# Let devs override any default variables, to avoid needing
# to change the version controlled config-default.sh file
test -f "${KUBEVIRT_PATH}cluster-hack/config-local.sh" && source ${KUBEVIRT_PATH}cluster-hack/config-local.sh

export docker_prefix master_ip network_provider kubeconfig manifest_docker_prefix
