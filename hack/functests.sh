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
# Copyright 2017 Red Hat, Inc.
#

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}
KUBEVIRT_E2E_PARALLEL_NODES=${KUBEVIRT_E2E_PARALLEL_NODES:-3}
KUBEVIRT_FUNC_TEST_GINKGO_ARGS=${FUNC_TEST_ARGS:-${KUBEVIRT_FUNC_TEST_GINKGO_ARGS}}

source hack/common.sh
source hack/config.sh

_default_previous_release_registry="quay.io/kubevirt"

previous_release_registry=${PREVIOUS_RELEASE_REGISTRY:-$_default_previous_release_registry}

functest_docker_prefix=${manifest_docker_prefix-${docker_prefix}}

if [[ ${KUBEVIRT_PROVIDER} == os-* ]] || [[ ${KUBEVIRT_PROVIDER} =~ (okd|ocp)-* ]]; then
    oc=${kubectl}
fi

rm -rf $ARTIFACTS
mkdir -p $ARTIFACTS

function functest() {
    KUBEVIRT_FUNC_TEST_SUITE_ARGS="-apply-default-e2e-configuration \
	    -conn-check-ipv4-address=${conn_check_ipv4_address} \
	    -conn-check-ipv6-address=${conn_check_ipv6_address} \
	    -conn-check-dns=${conn_check_dns} \
	    ${KUBEVIRT_FUNC_TEST_SUITE_ARGS}"
    if [[ ${KUBEVIRT_PROVIDER} =~ .*(k8s-1\.16)|(k8s-1\.17)|k8s-sriov.* ]]; then
        echo "Will skip test asserting the cluster is in dual-stack mode."
        KUBEVIRT_FUNC_TEST_SUITE_ARGS="-skip-dual-stack-test ${KUBEVIRT_FUNC_TEST_SUITE_ARGS}"
    fi

    _out/tests/ginkgo -r --slowSpecThreshold 60 $@ _out/tests/tests.test -- -kubeconfig=${kubeconfig} -container-tag=${docker_tag} -container-tag-alt=${docker_tag_alt} -container-prefix=${functest_docker_prefix} -image-prefix-alt=${image_prefix_alt} -oc-path=${oc} -kubectl-path=${kubectl} -gocli-path=${gocli} -installed-namespace=${namespace} -previous-release-tag=${PREVIOUS_RELEASE_TAG} -previous-release-registry=${previous_release_registry} -deploy-testing-infra=${deploy_testing_infra} -config=${KUBEVIRT_DIR}/tests/default-config.json --artifacts=${ARTIFACTS} ${KUBEVIRT_FUNC_TEST_SUITE_ARGS}
}

if [ "$KUBEVIRT_E2E_PARALLEL" == "true" ]; then
    trap "_out/tests/junit-merger -o ${ARTIFACTS}/junit.functest.xml '${ARTIFACTS}/partial.*.xml'" EXIT
    parallel_test_args=""
    serial_test_args=""

    if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
        parallel_test_args="${parallel_test_args} --skip=\\[Serial\\]|${KUBEVIRT_E2E_SKIP}"
        serial_test_args="${serial_test_args} --skip=${KUBEVIRT_E2E_SKIP}"
    else
        parallel_test_args="${parallel_test_args} --skip=\\[Serial\\]"
    fi

    if [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
        parallel_test_args="${parallel_test_args} --focus=${KUBEVIRT_E2E_FOCUS}"
        serial_test_args="${serial_test_args} --focus=\\[Serial\\].*(${KUBEVIRT_E2E_FOCUS})|(${KUBEVIRT_E2E_FOCUS}).*\\[Serial\\]"
    else
        serial_test_args="${serial_test_args} --focus=\\[Serial\\]"
    fi

    return_value=0
    set +e
    functest --nodes=${KUBEVIRT_E2E_PARALLEL_NODES} ${parallel_test_args} ${KUBEVIRT_FUNC_TEST_GINKGO_ARGS}
    return_value="$?"
    set -e
    if [ "$return_value" -ne 0 ] && ! [ "$KUBEVIRT_E2E_RUN_ALL_SUITES" == "true" ]; then
        exit "$return_value"
    fi
    KUBEVIRT_FUNC_TEST_SUITE_ARGS="-junit-output ${ARTIFACTS}/partial.junit.functest.xml ${KUBEVIRT_FUNC_TEST_SUITE_ARGS}"
    functest ${serial_test_args} ${KUBEVIRT_FUNC_TEST_GINKGO_ARGS}
    exit "$return_value"
else
    additional_test_args=""
    if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
        additional_test_args="${additional_test_args} --skip=${KUBEVIRT_E2E_SKIP}"
    fi

    if [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
        additional_test_args="${additional_test_args} --focus=${KUBEVIRT_E2E_FOCUS}"
    fi

    functest ${additional_test_args} ${KUBEVIRT_FUNC_TEST_GINKGO_ARGS}
fi
