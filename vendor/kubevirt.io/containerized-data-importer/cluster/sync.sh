#!/bin/bash -e

cdi=$1
cdi="${cdi##*/}"

echo $cdi

source ./hack/build/config.sh
source ./cluster/gocli.sh

CDI_NAMESPACE=${CDI_NAMESPACE:-kube-system}

# Set controller verbosity to 3 for functional tests.
export VERBOSITY=3

registry_port=$($gocli ports registry | tr -d '\r')
registry=localhost:$registry_port

DOCKER_REPO=${registry} make docker push
DOCKER_REPO="registry:5000" make manifests

# Make sure that all nodes use the newest images
container=""
container_alias=""
images="${@:-${DOCKER_IMAGES}}"
for arg in $images; do
    name=$(basename $arg)
    container="${container} registry:5000/${name}:latest"
done
for i in $(seq 1 ${KUBEVIRT_NUM_NODES}); do
    echo "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs \-\-max-args=1 sudo docker pull"
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" "echo \"${container}\" | xargs \-\-max-args=1 sudo docker pull"
done

# In order to make the cloner work in open shift, we need to give the cdi-sa Service Account privileged rights.
if [[ $(getClusterType) == $OPENSHIFT_IMAGE ]]; then
    ./cluster/kubectl.sh adm policy add-scc-to-user privileged -z cdi-sa -n ${CDI_NAMESPACE}
fi

# Install CDI
./cluster/kubectl.sh apply -f ./manifests/generated/cdi-controller.yaml
# Start functional test HTTP server.
./cluster/kubectl.sh apply -f ./manifests/generated/file-host.yaml
