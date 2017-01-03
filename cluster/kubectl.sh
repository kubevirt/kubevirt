#!/bin/bash

source ${KUBEVIRT_PATH}hack/config.sh

SYNC_CONFIG=${KUBEVIRT_PATH}cluster/vagrant/sync_config.sh

if [ "x$1" == "x--init" ]
then
    exec $SYNC_CONFIG
fi

if [ -e  ${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig ] &&
   [ -e ${KUBEVIRT_PATH}cluster/vagrant/.kubectl ] &&
   [ "x$1" == "x--core" ]; then
    shift
    ${KUBEVIRT_PATH}cluster/vagrant/.kubectl --kubeconfig=${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig $@
elif [ -e ${KUBEVIRT_PATH}cluster/vagrant/.kubectl ];then
    ${KUBEVIRT_PATH}cluster/vagrant/.kubectl -s http://${master_ip}:8184 $@
else
    echo "Did you already run '$SYNC_CONFIG' to deploy kubevirt?"
fi
