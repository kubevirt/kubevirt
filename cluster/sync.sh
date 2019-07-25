#!/bin/bash -e

registry_port=$(./cluster/cli.sh ports registry | tr -d '\r')
registry=localhost:$registry_port

CMD="./cluster/kubectl.sh" ./hack/clean.sh

IMAGE_REGISTRY=$registry make docker-build-operator docker-push-operator

for i in $(seq 1 ${CLUSTER_NUM_NODES}); do
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" "sudo docker pull registry:5000/kubevirt/hyperconverged-cluster-operator"
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" 'sudo sysctl -w user.max_user_namespaces=1024'
done

CMD="./cluster/kubectl.sh" HCO_IMAGE="registry:5000/kubevirt/hyperconverged-cluster-operator:latest" ./hack/deploy.sh
