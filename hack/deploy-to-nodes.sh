#!/usr/bin/env bash

source hack/common.sh
source kubevirtci/cluster-up/cluster/$KUBEVIRT_PROVIDER/provider.sh

set -e

# Deploy CNI plugin passt
if [ "${KUBEVIRT_DEPLOY_NET_BINDING_CNI}" == "true" ]; then
    _kubectl create -f $KUBEVIRT_DIR/cmd/cniplugins/passt-binding/passt-binding-ds.yaml
fi
