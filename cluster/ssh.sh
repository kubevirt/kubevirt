#!/usr/bin/env bash
set -e

KUBEVIRT_PATH="$(
    cd "$(dirname "$BASH_SOURCE[0]")/../"
    echo "$(pwd)/"
)"

source ${KUBEVIRT_PATH}/cluster-hack/common.sh

test -t 1 && USE_TTY="-it"
source ${KUBEVIRT_PATH}/cluster/$KUBEVIRT_PROVIDER/provider.sh
source ${KUBEVIRT_PATH}/cluster-hack/config.sh

ssh_key=${KUBEVIRT_PATH}/cluster-hack/common.key
chmod 600 $ssh_key

node=$1

if [ -z "$node" ]; then
    echo "node name required as argument"
    echo "okd example: ./ssh master-0"
    echo "k8s example: ./ssh node01"
    exit 1
fi

if [[ $provider_prefix =~ okd.* ]]; then
    ports=$($KUBEVIRT_PATH/cluster/cli.sh --prefix $provider_prefix ports --container-name cluster)

    if [[ $node =~ worker-0.* ]]; then
        port=$(echo "$ports" | grep 2202 | awk -F':' '{print $2}')
    elif [[ $node =~ master-0.* ]]; then
        port=$(echo "$ports" | grep 2201 | awk -F':' '{print $2}')
    fi

    if [ -z "$port" ]; then
        echo "no ssh port found for $node"
        exit 1
    fi
    ssh -lcore -p $port core@127.0.0.1 -i ${ssh_key}
else
    ${_cli} --prefix $provider_prefix ssh "$1"
fi
