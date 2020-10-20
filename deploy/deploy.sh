#!/bin/bash

set -x

hco_namespace=kubevirt-hyperconverged

# Create the namespaces for the HCO
kubectl create ns $hco_namespace

# Create additional namespaces needed for HCO components
namespaces=("openshift")
for namespace in ${namespaces[@]}; do
    if [[ $(kubectl get ns ${namespace}) == "" ]]; then
        kubectl create ns ${namespace}
    fi
done



# Launch all of the CRDs.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/cluster-network-addons00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/containerized-data-importer00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/kubevirt00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hostpath-provisioner00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/node-maintenance00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance01.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance02.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance03.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/vm-import-operator00.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco01.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco02.crd.yaml

# Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
kubectl create -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cluster_role.yaml
kubectl create -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/service_account.yaml
kubectl create -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cluster_role_binding.yaml
kubectl create -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/operator.yaml

# Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
kubectl create -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/hco.cr.yaml
