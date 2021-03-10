#!/bin/bash -xe

remote=$(echo $1 | sed "s/:.*//")
branch=$(echo $1 | sed "s/.*://")
version=$(echo $KUBEVIRT_PROVIDER | sed "s/k8s-//")

cleanup() {
    rm -rf kubevirtci
}

provision() {
    (
        cd kubevirtci/cluster-provision/k8s/$version
        ../provision.sh
    )
}

trap cleanup EXIT

git clone https://github.com/$remote/kubevirtci -b $branch
rsync kubevirtci/cluster-up cluster-up
rsync -rt --links ./kubevirtci/cluster-up/* ./cluster-up/

provision

echo "export KUBEVIRTCI_GOCLI_CONTAINER=quay.io/kubevirtci/gocli:latest" >> cluster-up/hack/common.sh
echo "export KUBEVIRTCI_PROVISION_CHECK=1" >> cluster-up/hack/common.sh
