#!/usr/bin/env bash
set -ex

LOCAL_DIR=_local
FORMAT=${FORMAT:-txt}
DEBUG_OPERATOR=${DEBUG_OPERATOR:-true}
DEBUG_WEBHOOK=${DEBUG_WEBHOOK:-false}
hco_namespace=kubevirt-hyperconverged

set -o allexport
source hack/config
set +o allexport
export WEBHOOK_MODE=false

mkdir -p "${LOCAL_DIR}"
./hack/generate_local_env.py "${LOCAL_DIR}" "${FORMAT}"

# don't deploy operator, webhook and the HCO CR.
sed "s/\(^.*\/hco.cr.yaml$\)/### \1/" deploy/deploy.sh > _local/deploy.sh
sed -i "s|-f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/main/deploy|-f deploy|g" _local/deploy.sh

chmod +x _local/deploy.sh

kubectl config set-context --current --namespace=${hco_namespace}
_local/deploy.sh

if [ "${DEBUG_OPERATOR}" == "true" ]; then
  kubectl --namespace=${hco_namespace} scale deploy hyperconverged-cluster-operator --replicas=0
else
  kubectl --namespace=${hco_namespace} scale deploy hyperconverged-cluster-operator --replicas=1
fi

if [ "${DEBUG_WEBHOOK}" == "true" ]; then
  kubectl --namespace=${hco_namespace} scale deploy hyperconverged-cluster-webhook --replicas=0
  hack/local_webhook.sh
  # telepresence will create it
  kubectl --namespace=${hco_namespace}  delete service hyperconverged-cluster-webhook-service --ignore-not-found
else
  kubectl --namespace=${hco_namespace} scale deploy hyperconverged-cluster-webhook --replicas=1
fi


