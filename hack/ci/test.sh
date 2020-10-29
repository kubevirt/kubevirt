#!/usr/bin/env bash
set -euo pipefail

echo "Running tests"
mkdir -p "$ARTIFACT_DIR"
# required to be set for test binary
export ARTIFACTS=${ARTIFACT_DIR}

tests.test -v=5 -kubeconfig=${KUBECONFIG} -container-tag=${DOCKER_TAG} -container-tag-alt= -container-prefix=${DOCKER_PREFIX} -image-prefix-alt=-kv -oc-path=/bin/oc -kubectl-path=/bin/kubectl -gocli-path=$(pwd)/cluster-up/cli.sh -test.timeout 420m -ginkgo.noColor -ginkgo.succinct -ginkgo.slowSpecThreshold=60 '-ginkgo.focus=\[rfe_id:273\]\[crit:high\]' -junit-output=${ARTIFACT_DIR}/junit.functest.xml -installed-namespace=kubevirt -previous-release-tag= -previous-release-registry=index.docker.io/kubevirt -deploy-testing-infra=false
