#!/bin/bash

set -e

image="os-3.9@sha256:03671c22bd5b08224926852fcb821dc11ed928044f2310e13c751edb7d0ce9a4"

source cluster/ephemeral-provider-common.sh

function up() {
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    ${_cli} run --nodes ${num_nodes} --reverse --random-ports --background --prefix $provider_prefix --registry-volume $(_registry_volume) --base "kubevirtci/${image}"
    ${_cli} ssh --prefix $provider_prefix node01 sudo cp /etc/origin/master/admin.kubeconfig ~vagrant/
    ${_cli} ssh --prefix $provider_prefix node01 sudo chown vagrant:vagrant ~vagrant/admin.kubeconfig

    chmod 0600 ${KUBEVIRT_PATH}cluster/vagrant.key
    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/vagrant.key -P $(_port ssh)"

    # Copy oc tool and configuration file
    scp ${OPTIONS} vagrant@$(_main_ip):/usr/local/bin/oc ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    chmod u+x ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    scp ${OPTIONS} vagrant@$(_main_ip):~vagrant/admin.kubeconfig ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig

    # Update Kube config to support unsecured connection
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster node01:8443 --server=https://$(_main_ip):$(_port osp)
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster node01:8443 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
