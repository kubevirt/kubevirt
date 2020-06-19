#!/usr/bin/env bash
set -xeuo pipefail

release_base_url="https://gcsweb.apps.ovirt.org/gcs/kubevirt-prow/devel/nightly/release/kubevirt/kubevirt"
release_date=$(curl -L "${release_base_url}/latest")
release_url="${release_base_url}/${release_date}"
commit=$(curl -L "${release_url}/commit")

export DOCKER_PREFIX='kubevirtnightlybuilds'
DOCKER_TAG="${release_date}_$(echo ${commit} | cut -c 1-9)"
export DOCKER_TAG

echo "deploying kubevirt from nightly build"
oc create -f "${release_url}/kubevirt-operator.yaml"
oc create -f "${release_url}/kubevirt-cr.yaml"

echo "Deploying test infrastructure"
for testinfra_file in $(curl -L "${release_url}/testing/" | grep -oE 'https://[^"]*\.yaml'); do
    oc create -f ${testinfra_file}
done

set +e
oc wait -n kubevirt kv kubevirt --for condition=Available --timeout 15m
return_code=$?
set -e
if [ ${return_code} -ne 0 ]; then
    echo "Dumping KubeVirt state"
    hack/dump.sh
    exit ${return_code}
fi

echo "calling cluster-up to prepare config and check whether cluster is reachable"
export KUBEVIRT_PROVIDER=external
bash -x ./cluster-up/up.sh

echo "testing"
mkdir -p "$ARTIFACT_DIR"
FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus=\[crit:high\] --junit-output='"$ARTIFACT_DIR"'/junit.functest.xml' \
    bash -x ./hack/functests.sh
