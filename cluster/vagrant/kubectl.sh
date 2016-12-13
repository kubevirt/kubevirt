#!/bin/bash

source hack/config.sh

SYNC_CONFIG=cluster/vagrant/sync_config.sh

if [ "x$1" == "x--init" ]
then
    exec $SYNC_CONFIG
fi

if [ -e cluster/vagrant/.kubeconfig ] && [ -e cluster/vagrant/.kubectl ] && [ "x$1" == "x--core" ];then
    shift
    cluster/vagrant/.kubectl --kubeconfig=cluster/vagrant/.kubeconfig $@
elif [ -e cluster/vagrant/.kubectl ];then
    cluster/vagrant/.kubectl -s http://${master_ip}:8184 $@
else
    echo "Did you already run '$SYNC_CONFIG' to deploy kubevirt?"
fi
