#!/bin/bash

set -x

# Create the namespaces for the HCO
kubectl create ns kubevirt-hyperconverged

# Switch to the HCO namespace.
kubectl config set-context $(kubectl config current-context) --namespace=kubevirt-hyperconverged

# Launch all of the CRDs.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/hco.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/kubevirt.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/cdi.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/cna.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/ssp.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/kwebui.crd.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/nodemaintenance.crd.yaml

# Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/cluster_role.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/service_account.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/cluster_role_binding.yaml
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/operator.yaml

# Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
kubectl create -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy/converged/crds/hco.cr.yaml
