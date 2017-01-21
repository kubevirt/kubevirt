#!/usr/bin/bash

set -ex

KUBECTL=${KUBECTL:-kubectl}

echo "Cleaning up ..."
# Work around https://github.com/kubernetes/kubernetes/issues/33517
cluster/kubectl.sh --core delete -f manifests/virt-handler.yaml --cascade=false --grace-period 0 2>/dev/null || :
cluster/kubectl.sh --core delete pods -l=daemon=virt-handler --grace-period 0 2>/dev/null || :

# Delete everything else
for i in `ls manifests/*.yaml`; do
    $KUBECTL delete -f $i --grace-period 0 2>/dev/null || :
done

sleep 2

echo "Deploying ..."
for i in `ls manifests/*.yaml`; do
    $KUBECTL create -f $i
done
echo "Done"
