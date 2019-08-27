#!/bin/bash

set -x

# Create the namespaces for the HCO
kubectl create ns kubevirt-hyperconverged

# Create additional namespaces needed for HCO components
namespaces=("openshift" "openshift-machine-api")
for namespace in ${namespaces[@]}; do
    if [[ $(kubectl get ns ${namespace}) == "" ]]; then
        kubectl create ns ${namespace}
    fi
done

# Switch to the HCO namespace.
kubectl config set-context $(kubectl config current-context) --namespace=kubevirt-hyperconverged

# Launch all of the CRDs.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/hco.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/kubevirt.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/cdi.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/cna.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/common-template-bundles.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/node-labeller-bundles.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/template-validator.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/metrics-aggregation.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/nodemaintenance.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/mro.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/crds/v2vvmware.crd.yaml

# Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cluster_role.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/service_account.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/cluster_role_binding.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/operator.yaml

# Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/hco.cr.yaml
