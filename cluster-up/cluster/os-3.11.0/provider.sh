#!/usr/bin/env bash

set -e

image="os-3.11.0@sha256:ebc9048f25eb5cc720b8b4eeab7b58b5fa3648d27c9612d87bf338d5dbee46a7"

source ${KUBEVIRTCI_PATH}/cluster/ephemeral-provider-common.sh

function up() {
    ${_cli} run --reverse $(_add_common_params)
    ${_cli} ssh --prefix $provider_prefix node01 -- sudo cp /etc/origin/master/admin.kubeconfig ~vagrant/
    ${_cli} ssh --prefix $provider_prefix node01 -- sudo chown vagrant:vagrant ~vagrant/admin.kubeconfig

    # Copy oc tool and configuration file
    ${_cli} scp --prefix $provider_prefix /usr/bin/oc - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl
    ${_cli} scp --prefix $provider_prefix /etc/origin/master/admin.kubeconfig - >${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig

    # Update Kube config to support unsecured connection
    export KUBECONFIG=${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster node01:8443 --server=https://$(_main_ip):$(_port ocp)
    ${KUBEVIRTCI_CONFIG_PATH}/$KUBEVIRT_PROVIDER/.kubectl config set-cluster node01:8443 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
