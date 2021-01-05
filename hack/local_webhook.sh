#!/usr/bin/env bash
set -ex

hco_namespace=kubevirt-hyperconverged

kubectl apply -n $hco_namespace -f deploy/webhooks.yaml

i=0
until kubectl -n $hco_namespace get secret hyperconverged-cluster-webhook-service-cert
do
  if [[ "$i" -gt 100 ]]; then
    echo "TIMEOUT!!! Check cert-manager pods in your cluster and certificate objects in your namespace."
    exit 1
  fi

  echo "Waiting for secret 'hyperconverged-cluster-webhook-service-cert'. Try: $i"
  ((i=i+1))
  sleep 3s
done


mkdir -p ./_local/certs

kubectl -n $hco_namespace get secret hyperconverged-cluster-webhook-service-cert -o jsonpath='{.data .tls\.crt}'  |base64 -d > ./_local/certs/apiserver.crt
kubectl -n $hco_namespace get secret hyperconverged-cluster-webhook-service-cert -o jsonpath='{.data .tls\.key}'  |base64 -d > ./_local/certs/apiserver.key

