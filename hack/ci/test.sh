#!/usr/bin/env bash

set -xeuo pipefail

### TODO: START this is to be moved into a dedicated template that creates gcp clusters w/ nested-virt enabled
function enable_nested_virt_with_image() {
  if [ -z "$1" ]; then
    echo "image not provided"
    exit 1
  fi

  NESTED_VIRT_IMAGE="$1"

  oc get machineset -n openshift-machine-api -o json >/tmp/machinesets.json
  MACHINE_IMAGE=$(jq -r .items[0].spec.template.spec.providerSpec.value.disks[0].image /tmp/machinesets.json)
  sed -i 's/'"$MACHINE_IMAGE"'/'"$NESTED_VIRT_IMAGE"'/g' /tmp/machinesets.json
  oc apply -f /tmp/machinesets.json
  oc scale --replicas=0 machineset --all -n openshift-machine-api
  oc get machines -n openshift-machine-api -o json >/tmp/machines.json
  num_machines=$(jq '.items | length' /tmp/machines.json)
  while [ "$num_machines" -ne "3" ]; do
      sleep 5
      oc get machines -n openshift-machine-api -o json >/tmp/machines.json
      num_machines=$(jq '.items | length' /tmp/machines.json)
  done
  oc scale --replicas=1 machineset --all -n openshift-machine-api
  while [ "$num_machines" -ne "6" ]; do
      sleep 5
      oc get machines -n openshift-machine-api -o json >/tmp/machines.json
      num_machines=$(jq '.items | length' /tmp/machines.json)
  done
  while [ $(oc get nodes | wc -l) -ne "7" ]; do oc get nodes; sleep 5; done
  nodes_ready=false
  while ! "$nodes_ready"; do sleep 5 && if ! oc get nodes | grep NotReady; then nodes_ready=true; fi; done
  oc project default
  # apply kvm device plugin
  oc apply -f https://raw.githubusercontent.com/kubevirt/kubernetes-device-plugins/master/manifests/kvm-ds.yml
  workers=$(oc get nodes | grep worker | awk '{ print $1 }')
  workers_each=($workers)
  for i in {0..2}; do
      if ! oc debug node/"${workers_each[i]}" -- ls /dev/kvm; then oc debug node/"${workers_each[i]}" -- ls /dev/kvm; fi
  done
}

# Colin Walters created 2 images for testing/reuse in openshift-gce-devel, they are:
# - rhcos42-nested-virt,
# - rhcos43-nested-virt
enable_nested_virt_with_image "rhcos43-nested-virt"

### TODO: END to be moved into dedicated template

function check_basics() {
  echo "checking kubectl and oc"
  which kubectl || exit 1
  which oc || exit 1

  echo "checking nodes for cluster"
  # in CI, this cmd fails unless you provide a ns
  oc -n default get nodes

  echo "checking configuration"
  env | grep KUBE
  kubectl config view

  echo "checking configuration location"
  echo "KUBECONFIG: ${KUBECONFIG}"
}

function deploy_kubevirt() {
  # calling cluster-up will prepare a config that is sourced later on and check whether cluster is reachable
  # TODO: kubevirtci#199 remove patching of external provider
  (cd cluster-up/cluster/external && curl -L -O -o provider.sh https://raw.githubusercontent.com/dhiller/kubevirtci/fix-external-provider/cluster-up/cluster/external/provider.sh)
  bash -x ./cluster-up/up.sh

  echo "deploying"
  bash -x ./hack/cluster-deploy.sh
}

function run_kubevirt_functests() {
  if [ -z "$1" ]; then
    echo "tests to focus on not provided"
    exit 1
  fi

  if [ -z "$2" ]; then
    echo "artifacts dir not provided"
    exit 1
  fi

  TESTS_TO_FOCUS="$1"
  ARTIFACTS="$2"
  export ARTIFACTS

  echo "testing"
  mkdir -p "$ARTIFACTS"
  FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus='"$TESTS_TO_FOCUS"' --ginkgo.regexScansFilePath=true --junit-output='"$ARTIFACTS"'/junit.functest.xml' bash -x ./hack/functests.sh
}

export PATH=$PATH:/usr/local/go/bin/

# TODO: instead of pulling from docker registry we should directly create the KubeVirt latest within the Dockerfile.ci
# or at least use the official kubevirt registry
export DOCKER_PREFIX='dhiller'
export DOCKER_TAG="latest"
export KUBEVIRT_PROVIDER=external
export GIMME_GO_VERSION=1.12.8
export GOPATH="/go"
export GOBIN="/usr/bin"

# TODO: is this really required? I think we're consuming the built version with this image?
source /etc/profile.d/gimme.sh

TESTS_TO_FOCUS=$(grep -E -o '\[crit\:high\]' tests/*_test.go | sort | uniq | sed -E 's/tests\/([a-z_]+)\_test\.go\:.*/\1/' | tr '\n' '|' | sed 's/|$//')

check_basics
deploy_kubevirt
run_kubevirt_functests "$TESTS_TO_FOCUS" "$ARTIFACT_DIR"
