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
TESTS_TO_FOCUS=$(grep -E -o '\[crit\:high\]' tests/*_test.go | sort | uniq | sed -E 's/tests\/([a-z_]+)\.go\:.*/\1/' | tr '\n' '|' | sed 's/|$//')
FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus='"$TESTS_TO_FOCUS"' --ginkgo.regexScansFilePath=true --junit-output='"$ARTIFACT_DIR"'/junit.functest.xml' \
    bash -x ./hack/functests.sh
