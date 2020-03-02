#!/usr/bin/env bash
set -xeuo pipefail

export PATH=$PATH:/usr/local/go/bin/

export GIMME_GO_VERSION=1.12.8
export GOPATH="/go"
export GOBIN="/usr/bin"
source /etc/profile.d/gimme.sh

export DOCKER_PREFIX='dhiller'
export DOCKER_TAG="${PULL_PULL_SHA}"
export KUBEVIRT_PROVIDER=external

oc config view

echo "calling cluster-up to prepare config and check whether cluster is reachable"
bash -x ./cluster-up/up.sh

echo "deploying"
bash -x ./hack/cluster-deploy.sh

echo "testing"
mkdir -p "$ARTIFACT_DIR"
TESTS_TO_FOCUS=$(grep -E -o '\[crit\:high\]' tests/*_test.go | sort | uniq | sed -E 's/tests\/([a-z_]+)\.go\:.*/\1/' | tr '\n' '|' | sed 's/|$//')
FUNC_TEST_ARGS='--ginkgo.noColor --ginkgo.focus='"$TESTS_TO_FOCUS"' --ginkgo.regexScansFilePath=true --junit-output='"$ARTIFACT_DIR"'/junit.functest.xml' \
    bash -x ./hack/functests.sh
