#!/bin/bash -e

cdi=$1
cdi="${cdi##*/}"

echo $cdi

source ./hack/build/config.sh
source ./cluster/gocli.sh

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

./cluster/kubectl.sh apply -f ./manifests/generated/cdi-controller.yaml
