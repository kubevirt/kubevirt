#!/usr/bin/bash

set -ex

KUBECTL=${KUBECTL:-kubectl}

echo "Cleaning up ..."
for i in `ls manifests/*.yaml`; do
    $KUBECTL delete -f $i --grace-period 0 2>/dev/null || :
done

sleep 2

echo "Deploying ..."
for i in `ls manifests/*.yaml`; do
    $KUBECTL create -f $i
done
echo "Done"
