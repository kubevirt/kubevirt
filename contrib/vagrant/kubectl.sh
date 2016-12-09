#!/bin/bash

source hack/config.sh

if [ "x$1" == "x--init" ]
then
    exec contrib/vagrant/sync_config.sh
fi

if [ -e contrib/vagrant/.kubeconfig ] && [ -e contrib/vagrant/.kubectl ] && [ "x$1" == "x--core" ];then
    shift
    contrib/vagrant/.kubectl --kubeconfig=contrib/vagrant/.kubeconfig $@
elif [ -e contrib/vagrant/.kubectl ];then
    contrib/vagrant/.kubectl -s http://${master_ip}:8184 $@
else
    echo "Did you already run './contrib/vagrant/sync.sh' to deploy kubevirt?"
fi
