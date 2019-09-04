#!/usr/bin/env bash

set -e

image="k8s-1.14.6@sha256:ec29c07c94fce22f37a448cb85ca1fb9215d1854f52573316752d19a1c88bcb3"

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh

function up() {
    ${_cli} run $(_add_common_params)

    # Copy k8s config and kubectl
    ${_cli} scp --prefix $provider_prefix /usr/bin/kubectl - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    ${_cli} scp --prefix $provider_prefix /etc/kubernetes/admin.conf - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
