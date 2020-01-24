#!/bin/bash -xe

# The ocp provider does not include CNAO so we install it here
if [[ ! $KUBEVIRT_PROVIDER =~ ocp ]]; then
    exit 0
fi

kubectl=./cluster-up/kubectl.sh
version=0.25.0

echo "Installing cluster network addons operator $version"
$kubectl apply -f https://raw.githubusercontent.com/kubevirt/cluster-network-addons-operator/master/manifests/cluster-network-addons/$version/namespace.yaml
$kubectl apply -f https://raw.githubusercontent.com/kubevirt/cluster-network-addons-operator/master/manifests/cluster-network-addons/$version/network-addons-config.crd.yaml
$kubectl apply -f https://raw.githubusercontent.com/kubevirt/cluster-network-addons-operator/master/manifests/cluster-network-addons/$version/operator.yaml

# TODO: Check if there is a conditin for the operator to check
sleep 10

echo "Configuring CNAO components"
$kubectl apply -f https://raw.githubusercontent.com/kubevirt/cluster-network-addons-operator/master/manifests/cluster-network-addons/$version/network-addons-config-example.cr.yaml

$kubectl wait networkaddonsconfig cluster --for condition=Available

$kubectl get networkaddonsconfig cluster -o yaml
