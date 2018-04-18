#!/bin/bash

set -e

image="os-3.9.0@sha256:416e63f01ede46bd669f58b6e65c403dc5d1c560159950943491a7d06683321b"

source cluster/ephemeral-provider-common.sh

function up() {
    ${_cli} run --reverse $(_add_common_params)
    ${_cli} ssh --prefix $provider_prefix node01 sudo cp /etc/origin/master/admin.kubeconfig ~vagrant/
    ${_cli} ssh --prefix $provider_prefix node01 sudo chown vagrant:vagrant ~vagrant/admin.kubeconfig

    chmod 0600 ${KUBEVIRT_PATH}cluster/vagrant.key
    OPTIONS="-o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ${KUBEVIRT_PATH}cluster/vagrant.key -P $(_port ssh)"

    # Copy oc tool and configuration file
    scp ${OPTIONS} vagrant@$(_main_ip):/usr/bin/oc ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    chmod u+x ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl
    scp ${OPTIONS} vagrant@$(_main_ip):~vagrant/admin.kubeconfig ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig

    # Update Kube config to support unsecured connection
    export KUBECONFIG=${KUBEVIRT_PATH}cluster/$PROVIDER/.kubeconfig
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster node01:8443 --server=https://$(_main_ip):$(_port osp)
    ${KUBEVIRT_PATH}cluster/$PROVIDER/.kubectl config set-cluster node01:8443 --insecure-skip-tls-verify=true

    # Make sure that local config is correct
    prepare_config
}
