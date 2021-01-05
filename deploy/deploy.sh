#!/bin/bash

set -ex

hco_namespace=kubevirt-hyperconverged

# Create the namespaces for the HCO
kubectl create ns $hco_namespace --dry-run=true -o yaml | kubectl apply -f -

# Create additional namespaces needed for HCO components
namespaces=("openshift")
for namespace in ${namespaces[@]}; do
    if [[ $(kubectl get ns ${namespace}) == "" ]]; then
        kubectl create ns ${namespace} --dry-run=true -o yaml | kubectl apply -f -
    fi
done



# Launch all of the CRDs.
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/cluster-network-addons00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/containerized-data-importer00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/kubevirt00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hostpath-provisioner00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/node-maintenance00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance01.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance02.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/scheduling-scale-performance03.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/vm-import-operator00.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco01.crd.yaml
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco02.crd.yaml

# Deploy cert-manager for webhook certificates
kubectl apply  -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cert-manager.yaml
kubectl -n cert-manager wait deployment/cert-manager --for=condition=Available --timeout="300s"
kubectl -n cert-manager wait deployment/cert-manager-webhook --for=condition=Available --timeout="300s"

# Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
kubectl apply  -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cluster_role.yaml
kubectl apply  -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/service_account.yaml
kubectl apply  -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cluster_role_binding.yaml
kubectl apply  -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/webhooks.yaml
kubectl apply  -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/operator.yaml

# Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
kubectl apply  -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/hco.cr.yaml
