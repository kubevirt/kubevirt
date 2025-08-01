#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2021 Red Hat, Inc.
#
# This script should be executed through the makefile via `make perftest`.

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}
PROMETHEUS_PORT=${PROMETHEUS_PORT:-30007}

source hack/common.sh
source hack/config.sh

_default_previous_release_registry="quay.io/kubevirt"

previous_release_registry=${PREVIOUS_RELEASE_REGISTRY:-$_default_previous_release_registry}

perftest_docker_prefix=${manifest_docker_prefix-${docker_prefix}}

if [ -z "$kubeconfig" ]; then
    kubeconfig="$KUBECONFIG"
fi

echo 'Performance testing'
export KUBEVIRT_E2E_PERF_TEST=true

echo 'Preparing directory for artifacts'
export AUDIT_CONFIG=${ARTIFACTS}/perfscale-audit-cfg.json
export AUDIT_RESULTS=${ARTIFACTS}/perfscale-audit-results.json
mkdir -p $ARTIFACTS

export TESTS_OUT_DIR=${TESTS_OUT_DIR}
mkdir -p ${TESTS_OUT_DIR}/performance

echo 'ARTIFACTS ' ${ARTIFACTS}
echo 'TESTS_OUT_DIR ' ${TESTS_OUT_DIR}

function perftest() {
    _out/tests/ginkgo -r -slow-spec-threshold=60s $@ _out/tests/tests.test -- ${extra_args} -kubeconfig=${kubeconfig} -container-tag=${docker_tag} -container-tag-alt=${docker_tag_alt} -container-prefix=${perftest_docker_prefix} -image-prefix-alt=${image_prefix_alt} -kubectl-path=${kubectl} -installed-namespace=${namespace} -previous-release-tag=${PREVIOUS_RELEASE_TAG} -previous-release-registry=${previous_release_registry} -deploy-testing-infra=${deploy_testing_infra} -config=${KUBEVIRT_DIR}/tests/default-config.json --artifacts=${ARTIFACTS}
}

function perfaudit() {
    _out/cmd/perfscale-audit/perfscale-audit --config-file=${AUDIT_CONFIG} --results-file=${AUDIT_RESULTS}
}

if [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
    export KUBEVIRT_E2E_FOCUS="${KUBEVIRT_E2E_FOCUS}|\\[sig-performance\\]"
else
    export KUBEVIRT_E2E_FOCUS="\\[sig-performance\\]"
fi

additional_test_args=""
if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
    additional_test_args="${additional_test_args} --skip=${KUBEVIRT_E2E_SKIP}"
fi

if [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
    additional_test_args="${additional_test_args} --focus=${KUBEVIRT_E2E_FOCUS}"
fi

additional_test_args="${additional_test_args} --skip-package test/performance"

perftest ${additional_test_args} ${FUNC_TEST_ARGS}
