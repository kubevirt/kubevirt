#!/bin/bash

source ${KUBEVIRT_PATH}hack/config.sh

SYNC_CONFIG=${KUBEVIRT_PATH}cluster/vagrant/sync_config.sh

if [ "$1" == "--init" ]
then
    exec $SYNC_CONFIG
    exit
fi

if [ "$1" == "spice" ]; then
    ${KUBEVIRT_PATH}cmd/virtctl/virtctl spice ${2} -s ${master_ip}:8184 ${3}
    exit
fi

if [ "$1" == "console" ] || [ "$1" == "convert-spec" ]; then
    cmd/virtctl/virtctl "$@" -s http://${master_ip}:8184 
    exit
fi

# Print usage from virtctl and kubectl
if [ "$1" == "--help" ]  || [ "$1" == "-h" ] ; then
    cmd/virtctl/virtctl "$@"
fi

if [ -e  ${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig ] &&
   [ -e ${KUBEVIRT_PATH}cluster/vagrant/.kubectl ] &&
   [ "x$1" == "x--core" ]; then
    shift
    ${KUBEVIRT_PATH}cluster/vagrant/.kubectl --kubeconfig=${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig "$@"
elif [ -e ${KUBEVIRT_PATH}cluster/vagrant/.kubectl ];then
    ${KUBEVIRT_PATH}cluster/vagrant/.kubectl -s http://${master_ip}:8184 "$@"
else
    echo "Did you already run '$SYNC_CONFIG' to deploy kubevirt?"
fi
