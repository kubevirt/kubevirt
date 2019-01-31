#!/bin/bash -e

cdi=$1
cdi="${cdi##*/}"

echo $cdi

source ./hack/build/config.sh
source ./cluster/gocli.sh

CDI_NAMESPACE=${CDI_NAMESPACE:-cdi}

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
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster/cli.sh ssh "node$(printf "%02d" ${i})" "sudo sysctl -w user.max_user_namespaces=1024"
done

# Install CDI
./cluster/kubectl.sh apply -f ./manifests/generated/cdi-operator.yaml
./cluster/kubectl.sh apply -f ./manifests/generated/cdi-operator-cr.yaml
./cluster/kubectl.sh wait cdis.cdi.kubevirt.io/cdi --for=condition=running --timeout=120s
# Start functional test HTTP server.
./cluster/kubectl.sh apply -f ./manifests/generated/file-host.yaml
./cluster/kubectl.sh apply -f ./manifests/generated/registry-host.yaml
