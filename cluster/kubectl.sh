#!/bin/bash

if [ "x$1" == "x--init" ]
then
    exec cluster/sync_config.sh
fi

if [ -e cluster/.kubeconfig ] && [ -e cluster/.kubectl ] && [ "x$1" == "x--core" ];then
    shift
    cluster/.kubectl --kubeconfig=cluster/.kubeconfig $@
elif [ -e cluster/.kubectl ];then
    cluster/.kubectl -s http://192.168.200.2:8184 $@
else
    echo "Did you already run './cluster/sync.sh' to deploy kubevirt?"
fi
