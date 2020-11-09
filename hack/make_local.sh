#!/usr/bin/env bash
set -ex

LOCAL_DIR=_local
FORMAT=${FORMAT:-txt}

hco_namespace=kubevirt-hyperconverged

set -o allexport
source hack/config
set +o allexport
export WEBHOOK_MODE=false

mkdir -p "${LOCAL_DIR}"
./hack/make_local.py "${LOCAL_DIR}" "${FORMAT}"
sed "s/\(^.*\/operator.yaml$\)/### \1/" deploy/deploy.sh > _local/deploy.sh
sed -i "s|-f https://raw.githubusercontent.com/kubevirt/hyperconverged-cluster-operator/master/deploy|-f deploy|g" _local/deploy.sh

chmod +x _local/deploy.sh

kubectl config set-context --current --namespace=${hco_namespace}
_local/deploy.sh
kubectl apply -f _local/local.yaml
