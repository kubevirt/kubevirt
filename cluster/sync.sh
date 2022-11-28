#!/bin/bash -ex

source ./hack/common.sh
source ./_kubevirtci/cluster-up/cluster/ephemeral-provider-common.sh

registry_port=$(${_cri_bin} ps | grep -Po '(?<=0.0.0.0:)\d+(?=->5000\/tcp)' | head -n 1)
if [ -z "$registry_port" ]
then
      >&2 echo "unable to get the registry port"
      exit 1
fi

registry=localhost:$registry_port

# Cleanup previously generated manifests
rm -rf _out/

# Copy release manifests as a base for generated ones, this should make it possible to upgrade
cp -r deploy _out/

# Sed from quay.io to local registry
sed -r -i 's|: quay.io/kubevirt/hyperconverged-cluster-operator(@sha256)?:.*$|: registry:5000/kubevirt/hyperconverged-cluster-operator:latest|g' _out/operator.yaml
sed -r -i 's|: quay.io/kubevirt/hyperconverged-cluster-webhook(@sha256)?:.*$|: registry:5000/kubevirt/hyperconverged-cluster-webhook:latest|g' _out/operator.yaml

CMD="./cluster/kubectl.sh" ./hack/clean.sh

IMAGE_REGISTRY=$registry make container-build-operator container-push-operator container-build-webhook container-push-webhook

nodes=()
if [[ $KUBEVIRT_PROVIDER =~ (okd|ocp).* ]]; then
    for okd_node in "master-0" "worker-0"; do
        node=$(./cluster/kubectl.sh get nodes | grep -o '[^ ]*'${okd_node}'[^ ]*')
        nodes+=(${node})
    done
    pull_command="podman"
else
    for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
        nodes+=("node$(printf "%02d" ${i})")
    done
    pull_command="docker"
fi

${_cri_bin} ps -a

for node in ${nodes[@]}; do
    ./cluster/ssh.sh ${node} "echo registry:5000/kubevirt/hyperconverged-cluster-operator | xargs \-\-max-args=1 sudo ${pull_command} pull"
    ./cluster/ssh.sh ${node} "echo registry:5000/kubevirt/hyperconverged-cluster-webhook | xargs \-\-max-args=1 sudo ${pull_command} pull"
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster/ssh.sh ${node} "echo user.max_user_namespaces=1024 | xargs \-\-max-args=1 sudo sysctl -w"
done

# Deploy the HCO
CMD="./cluster/kubectl.sh" HCO_IMAGE="registry:5000/kubevirt/hyperconverged-cluster-operator:latest" WEBHOOK_IMAGE="registry:5000/kubevirt/hyperconverged-cluster-webhook:latest" ./hack/deploy.sh
