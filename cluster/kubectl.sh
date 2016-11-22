#!/bin/bash
if [ -e cluster/.kubeconfig ] && [ -e cluster/.kubectl ];then
    cluster/.kubectl --kubeconfig=cluster/.kubeconfig $@
else
    echo "Did you already run './cluster/sync.sh' to deploy kubevirt?"
fi
