#!/bin/bash

source ${KUBEVIRT_PATH}hack/config.sh

SYNC_CONFIG=${KUBEVIRT_PATH}cluster/vagrant/sync_config.sh

if [ "x$1" == "x--init" ]
then
    exec $SYNC_CONFIG
    exit
fi

if [ "x$1" == "xspice" ]; then
        viewer=${SPICE_VIEWER:-remote\-viewer}
        if [ "x$3" == "x--details" ]; then
            curl -sS http://${master_ip}:8184/apis/kubevirt.io/v1alpha1/namespaces/default/vms/$2/spice  -H"Accept:text/plain"
        else
            curl -sS http://${master_ip}:8184/apis/kubevirt.io/v1alpha1/namespaces/default/vms/$2/spice > ${KUBEVIRT_PATH}cluster/.console.vv  -H"Accept:text/plain"
            echo $viewer
            $viewer ${KUBEVIRT_PATH}cluster/.console.vv
        fi
        exit
fi

if [ "x$1" == "xconsole" ]; then
    cmd/virtctl/virtctl "$@" -s http://${master_ip}:8184 
    exit
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
