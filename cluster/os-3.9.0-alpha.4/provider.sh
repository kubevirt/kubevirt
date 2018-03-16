#!/bin/bash

set -e

image="os-3.9@sha256:a3c66710e0f4d55e81d5b2d32e89c074074cc14b216941818bde0d68cf4b0a12"

source cluster/ephemeral-provider-common.sh

function up() {
    # Add one, 0 here means no node at all, but in the kubevirt repo it means master-only
    local num_nodes=${VAGRANT_NUM_NODES-0}
    num_nodes=$((num_nodes + 1))
    ${_cli} run --nodes ${num_nodes} --reverse --osp-port 127.0.0.1:8443 --ssh-port 127.0.0.1:2201 --background --registry-port 127.0.0.1:5000 --prefix $PROVIDER --registry-volume kubevirt_registry --base "kubevirtci/${image}"
    ${_cli} ssh --prefix $PROVIDER node01 sudo cp /etc/origin/master/admin.kubeconfig ~vagrant/
    ${_cli} ssh --prefix $PROVIDER node01 sudo chown vagrant:vagrant ~vagrant/admin.kubeconfig

    chmod 0600 ${KUBEVIRT_PATH}cluster/vagrant.key
    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/vagrant.key -P 2201"

    # Copy oc tool and configuration file
    scp ${OPTIONS} vagrant@127.0.0.1:/usr/local/bin/oc ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    chmod u+x ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    scp ${OPTIONS} vagrant@127.0.0.1:~vagrant/admin.kubeconfig ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig

    # Update Kube config to support unsecured connection
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster node01:8443 --server=https://$(_main_ip):8443
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster node01:8443 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
