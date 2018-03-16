#!/bin/bash

set -e

image="k8s-1.9.3@sha256:ead8cbdf16e205acfe66ec4b03e31974217e07808da1d9127409337d4959ace7"

source cluster/ephemeral-provider-common.sh

function up() {
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    ${_cli} run --nodes ${num_nodes} --random-ports --background --prefix $_prefix --registry-volume $(_registry_volume) --base "kubevirtci/${image}"
    echo lala
    ${_cli} ssh --prefix $_prefix node01 sudo chown vagrant:vagrant /etc/kubernetes/admin.conf

    chmod 0600 ${KUBEVIRT_PATH}cluster/vagrant.key
    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/vagrant.key -P $(_port ssh)"

    # Copy k8s config and kubectl
    scp ${OPTIONS} vagrant@$(_main_ip):/usr/bin/kubectl ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    chmod u+x cluster/vagrant-kubernetes/.kubectl
    scp ${OPTIONS} vagrant@$(_main_ip):/etc/kubernetes/admin.conf ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster kubernetes --server=https://$(_main_ip):$(_port k8s)
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
