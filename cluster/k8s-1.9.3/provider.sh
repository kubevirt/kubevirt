#!/bin/bash

set -e

image="k8s-1.9.3@sha256:ead8cbdf16e205acfe66ec4b03e31974217e07808da1d9127409337d4959ace7"

source cluster/ephemeral-provider-common.sh

function up() {
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    ${_cli} run --nodes ${num_nodes} --k8s-port 127.0.0.1:8443 --ssh-port 127.0.0.1:2201 --background --registry-port 127.0.0.1:5000 --prefix $PROVIDER --registry-volume kubevirt_registry --base "kubevirtci/${image}"
    ${_cli} ssh --prefix $PROVIDER node01 sudo chown vagrant:vagrant /etc/kubernetes/admin.conf

    chmod 0600 ${KUBEVIRT_PATH}cluster/vagrant.key
    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/vagrant.key -P 2201"

    # Copy k8s config and kubectl
    scp ${OPTIONS} vagrant@127.0.0.1:/usr/bin/kubectl ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    chmod u+x cluster/vagrant-kubernetes/.kubectl
    scp ${OPTIONS} vagrant@127.0.0.1:/etc/kubernetes/admin.conf ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig

    # Set server and disable tls check
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster kubernetes --server=https://$(_main_ip):8443
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster kubernetes --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
