#!/usr/bin/env bash
set -xeuo pipefail

export DOCKER_PREFIX='kubevirtnightlybuilds'
export DOCKER_TAG="latest"
export KUBEVIRT_PROVIDER=external

echo "calling cluster-up to prepare config and check whether cluster is reachable"
bash -x ./cluster-up/up.sh

echo "deploying"
bash -x ./hack/cluster-deploy.sh

echo "testing"
mkdir -p "$ARTIFACT_DIR"
FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus=\[crit:high\] --junit-output='"$ARTIFACT_DIR"'/junit.functest.xml' \
    bash -x ./hack/functests.sh
