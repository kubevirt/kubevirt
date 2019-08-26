#!/usr/bin/env bash
set -ex

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi

source ${KUBEVIRTCI_PATH}/hack/common.sh

test -t 1 && USE_TTY="-it"
source ${KUBEVIRTCI_PATH}/cluster/$KUBEVIRT_PROVIDER/provider.sh
source ${KUBEVIRTCI_PATH}/hack/config.sh

ssh_key=${KUBEVIRTCI_PATH}/hack/common.key
chmod 600 $ssh_key
node=$1

if [ -z "$node" ]; then
    echo "node name required as argument"
    echo "okd example: ./ssh master-0"
    echo "k8s example: ./ssh node01"
    exit 1
fi

if [[ $KUBEVIRT_PROVIDER =~ okd.* ]]; then
    ports=$(${KUBEVIRTCI_PATH}cli.sh --prefix $provider_prefix ports --container-name cluster)

    if [[ $node =~ worker-0.* ]]; then
        port=$(echo "$ports" | grep 2202 | awk -F':' '{print $2}')
    elif [[ $node =~ master-0.* ]]; then
        port=$(echo "$ports" | grep 2201 | awk -F':' '{print $2}')
    fi

    if [ -z "$port" ]; then
        echo "no ssh port found for $node"
        exit 1
    fi
    shift
    ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -q -lcore -p $port core@127.0.0.1 -i ${ssh_key} $@
elif [[ $KUBEVIRT_PROVIDER =~ kind.* ]]; then
    _ssh_into_node "$@"
else
    ${_cli} --prefix $provider_prefix ssh "$@"
fi
