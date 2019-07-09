#!/bin/bash

set -euxo pipefail

# Install HCO
bash deploy.sh

# Explicitly wait for kubevirt to be deployed
kubectl wait -n kubevirt-hyperconverged kv kubevirt --for condition=Ready --timeout 360s || (echo "KubeVirt not ready in time" && exit 1)

# Wait at least for one pod
while [ -z "$(kubectl get pods -n kubevirt-hyperconverged)" ]; do
    echo "Waiting for at least one pod ..."
    kubectl get pods -n kubevirt-hyperconverged
    sleep 10
done

# Wait until k8s pods are running
while [ -n "$(kubectl get pods -n kubevirt-hyperconverged --no-headers | grep -v Running)" ]; do
    echo "Waiting for HCO pods to enter the Running state ..."
    kubectl get pods -n kubevirt-hyperconverged --no-headers | >&2 grep -v Running || true
    sleep 10
done

# Make sure all containers are ready
while [ -n "$(kubectl get pods -n kubevirt-hyperconverged -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers | grep false)" ]; do
    echo "Waiting for all containers to become ready ..."
    kubectl get pods -n kubevirt-hyperconverged -o'custom-columns=status:status.containerStatuses[*].ready,metadata:metadata.name' --no-headers
    sleep 10
done
