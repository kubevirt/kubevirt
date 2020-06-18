#!/usr/bin/env bash
set -euo pipefail

# bazel will fail if either HOME or USER are not set
HOME=$(pwd)
export HOME
USER='kubeadmin'
export USER

echo "setting up ephemeral image registry"
oc create -f hack/ci/resources/docker-registry.yaml
oc wait deployment registry --for condition=available --timeout=180s
while ! oc get service docker-registry; do
    echo 'waiting for service'
    sleep 1
done
cluster_ip=$(oc get service docker-registry -o custom-columns=:.spec.clusterIP --no-headers)
registry_port=5000
echo "Registry cluster_ip: $cluster_ip"
# now enable the insecure registry
oc patch image.config.openshift.io/cluster --type merge -p '{"spec":{"registrySources":{"insecureRegistries":["'"$cluster_ip:$registry_port"'"]}}}'

# for some reason the entry point from dockerfile is overridden, so we source gimme go again
source /etc/profile.d/gimme.sh && export GOPATH="/root/go" && go version

# we can only build here, otherwise the image build would be OOMKilled
echo "building images"
./hack/bazel-build.sh
./hack/bazel-build-images.sh

echo "pushing images"
export DOCKER_PREFIX="localhost:$registry_port"
DOCKER_TAG=$(cat _out/PULL_PULL_SHA)
export DOCKER_TAG
set +e
port_forward_pid=0
while true; do
    registry_pod=$(oc get pod -l app=docker-registry --no-headers -o custom-columns=:metadata.name)
    echo "Forwarding $registry_port to pod $registry_pod"
    oc port-forward $registry_pod $registry_port:$registry_port &
    port_forward_pid=$!
    ./hack/bazel-push-images.sh
    return_code=$?
    # kill port forwarding to avoid interferences during testing
    kill $port_forward_pid
    if [ $return_code -eq 0 ]; then
        break
    fi
done
set -e

echo "calling cluster-up to prepare config and check whether cluster is reachable"
export KUBEVIRT_PROVIDER=external
./cluster-up/up.sh

echo "building manifests"
export DOCKER_PREFIX="$cluster_ip:$registry_port"
# force using DOCKER_TAG
#rm ./bazel-bin/push-virt-operator.digest
./hack/build-manifests.sh

echo "deploying"
set +e
./hack/cluster-deploy.sh
if [ $? -ne 0 ]; then
    echo "--- Manifests:"
    cat _out/manifests/release/*.yaml
    echo "--- pods:"
    for pod_name in $(oc get pods -n kubevirt -o=custom-columns=:.metadata.name --no-headers); do
        echo " -- pod: $pod_name:"
        oc describe pod $pod_name -n kubevirt
    done
    oc describe pod $registry_pod
    echo "KUBEVIRT DEPLOYMENT FAILED!"
    exit 1
fi
set -e

echo "testing"
mkdir -p "$ARTIFACT_DIR"
FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus=\[crit:high\] --junit-output='"$ARTIFACT_DIR"'/junit.functest.xml' \
    ./hack/functests.sh
