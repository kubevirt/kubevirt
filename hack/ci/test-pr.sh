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
oc expose service docker-registry
registry_host=$(oc get route docker-registry --template='{{ .spec.host }}')
echo "Registry host will be $registry_host"
# now enable the insecure registry
oc patch image.config.openshift.io/cluster --type merge -p '{"spec":{"registrySources":{"insecureRegistries":["'"$registry_host"'"]}}}'

# for some reason the entry point from dockerfile is overridden, so we source gimme go again
source /etc/profile.d/gimme.sh && export GOPATH="/root/go" && go version

# we can only build here, otherwise the image build would be OOMKilled
echo "building images"
./hack/bazel-build.sh
./hack/bazel-build-images.sh

registry_port=5000
registry_pod=$(oc get pod -l app=docker-registry --no-headers -o custom-columns=:metadata.name)
echo "Forwarding $registry_port to pod $registry_pod"
oc port-forward $registry_pod $registry_port:$registry_port &

echo "pushing images"
export DOCKER_PREFIX="localhost:$registry_port"
DOCKER_TAG=$(cat _out/PULL_PULL_SHA)
export DOCKER_TAG
./hack/bazel-push-images.sh

echo "calling cluster-up to prepare config and check whether cluster is reachable"
export KUBEVIRT_PROVIDER=external
./cluster-up/up.sh

echo "building manifests"
export DOCKER_PREFIX="${registry_host}"
./hack/build-manifests.sh

echo "deploying"
./hack/cluster-deploy.sh

echo "testing"
mkdir -p "$ARTIFACT_DIR"
FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus=\[crit:high\] --junit-output='"$ARTIFACT_DIR"'/junit.functest.xml' \
    ./hack/functests.sh
