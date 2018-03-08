#!/bin/bash

set -e

image="os-3.9@sha256:67f864198299f658ae7199263d99a8e03afed7d02b252d3ccd2ba6ec434eae4f"

source cluster/ephemeral-provider-common.sh

function up() {
    ${_cli} run --nodes 1 --osp-port 127.0.0.1:8443 --ssh-port 127.0.0.1:2201 --background --registry-port 127.0.0.1:5000 --prefix $PROVIDER --registry-volume kubevirt_registry --base "kubevirtci/${image}"
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
