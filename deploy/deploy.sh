#!/bin/bash

set -ex

hco_namespace=kubevirt-hyperconverged

IS_OPENSHIFT=${IS_OPENSHIFT:-false}
if kubectl api-resources |grep clusterversions |grep config.openshift.io; then
  IS_OPENSHIFT="true"
fi

# Create the namespaces for the HCO
kubectl create ns $hco_namespace --dry-run=true -o yaml | kubectl apply -f -

# Create additional namespaces needed for HCO components
namespaces=("openshift")
for namespace in ${namespaces[@]}; do
    if [[ $(kubectl get ns ${namespace}) == "" ]]; then
        kubectl create ns ${namespace} --dry-run=true -o yaml | kubectl apply -f -
    fi
done

# Exclude Openshift specific resources if not on OCP/OKD
LABEL_SELECTOR_ARG=""
if [ "$IS_OPENSHIFT" != "true" ]; then
    LABEL_SELECTOR_ARG="-l name!=ssp-operator,name!=hyperconverged-cluster-cli-download"
fi

# Launch all of the CRDs.
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/cluster-network-addons00.crd.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/containerized-data-importer00.crd.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/hco00.crd.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/kubevirt00.crd.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/hostpath-provisioner00.crd.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/node-maintenance00.crd.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/scheduling-scale-performance00.crd.yaml
# TODO: Wait for https://github.com/kubevirt/hyperconverged-cluster-operator/pull/1866 to be re-reverted
# kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/crds/tekton-tasks-operator00.crd.yaml

# Deploy cert-manager for webhook certificates
kubectl apply ${LABEL_SELECTOR_ARG} -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/cert-manager.yaml
kubectl -n cert-manager wait deployment/cert-manager --for=condition=Available --timeout="300s"
kubectl -n cert-manager wait deployment/cert-manager-webhook --for=condition=Available --timeout="300s"

# Launch all of the Service Accounts, Cluster Role(Binding)s, and Operators.
kubectl apply ${LABEL_SELECTOR_ARG} -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/cluster_role.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/service_account.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/cluster_role_binding.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/webhooks.yaml
kubectl apply ${LABEL_SELECTOR_ARG} -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/operator.yaml

kubectl -n $hco_namespace wait deployment/hyperconverged-cluster-webhook --for=condition=Available --timeout="300s"

# Create an HCO CustomResource, which creates the KubeVirt CR, launching KubeVirt.
kubectl apply ${LABEL_SELECTOR_ARG} -n $hco_namespace -f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy/hco.cr.yaml
