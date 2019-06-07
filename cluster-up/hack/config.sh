unset docker_prefix master_ip network_provider kubeconfig manifest_docker_prefix

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-${PROVIDER}}

source ${KUBEVIRTCI_PATH}hack/config-default.sh

# Allow different providers to override default config values
test -f "${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/config-provider-${KUBEVIRT_PROVIDER}.sh" && source ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/config-provider-${KUBEVIRT_PROVIDER}.sh

export docker_prefix master_ip network_provider kubeconfig manifest_docker_prefix
