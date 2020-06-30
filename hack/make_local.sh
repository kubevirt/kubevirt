#!/usr/bin/env bash
set -ex

LOCAL_DIR=_local

mkdir -p "${LOCAL_DIR}"
./hack/make_local.py "${LOCAL_DIR}"
sed "s/\(^.*\/operator.yaml$\)/### \1/" deploy/deploy.sh > _local/deploy.sh
chmod +x _local/deploy.sh

_local/deploy.sh
kubectl apply -f _local/local.yaml
