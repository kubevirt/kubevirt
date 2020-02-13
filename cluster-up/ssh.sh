#!/usr/bin/env bash
set -e

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi

test -t 1 && USE_TTY="-it"

source ${KUBEVIRTCI_PATH}/hack/common.sh

source ${KUBEVIRTCI_CLUSTER_PATH}/$KUBEVIRT_PROVIDER/provider.sh
source ${KUBEVIRTCI_PATH}/hack/config.sh

node=$1

if [ -z "$node" ]; then
    echo "node name required as argument"
    echo "okd/ocp example: ./ssh master-0"
    echo "k8s example: ./ssh node01"
    exit 1
fi

if [[ $KUBEVIRT_PROVIDER =~ (ocp|okd).* ]]; then

    # Get the exact virsh domain name
    domain=$($KUBEVIRTCI_PATH/container.sh virsh list  |grep $node |awk '{print $2}')

    # Get the virsh address for the node
    ip=$($KUBEVIRTCI_PATH/container.sh virsh domifaddr $domain |grep vnet |head -n 1 | awk '{print $4}' | sed "s/\/.*//g")

    # Ignore the node argument
    shift

    # Run the passed arguments into the oc node
    $KUBEVIRTCI_PATH/container.sh ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -q -lcore core@$ip -i /vagrant.key $@

elif [[ $KUBEVIRT_PROVIDER =~ kind.* ]]; then
    _ssh_into_node "$@"
else
    ${_cli} --prefix $provider_prefix ssh "$@"
fi
