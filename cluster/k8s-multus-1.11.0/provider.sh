#!/bin/bash

set -e

image="k8s-multus-1.11.0@sha256:ec8bbe50e588708c625487086f6eac9082f4ee490ee5f481e475f699e34aa55a"

source cluster/ephemeral-provider-common.sh

function up() {
    ${_cli} run $(_add_common_params)

    # Copy k8s config and kubectl
    ${_cli} scp --prefix $provider_prefix /usr/bin/kubectl - >${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubectl
    chmod u+x ${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubectl
    ${_cli} scp --prefix $provider_prefix /etc/kubernetes/admin.conf - >${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRT_PATH}cluster/$KUBEVIRT_PROVIDER/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config

    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        cluster/kubectl.sh create clusterrolebinding multus-node-"node$(printf "%02d" ${i})" \
            --clusterrole=multus-crd-overpowered \
            --user=system:node:"node$(printf "%02d" ${i})"
    done
}
