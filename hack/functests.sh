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
# Copyright The KubeVirt Authors.
#

set -e

DOCKER_TAG=${DOCKER_TAG:-devel}
DOCKER_TAG_ALT=${DOCKER_TAG_ALT:-devel_alt}
KUBEVIRT_E2E_PARALLEL_NODES=${KUBEVIRT_E2E_PARALLEL_NODES:-4}
KUBEVIRT_FUNC_TEST_GINKGO_ARGS=${FUNC_TEST_ARGS:-${KUBEVIRT_FUNC_TEST_GINKGO_ARGS}}
KUBEVIRT_FUNC_TEST_LABEL_FILTER=${FUNC_TEST_LABEL_FILTER:-${KUBEVIRT_FUNC_TEST_LABEL_FILTER}}
KUBEVIRT_FUNC_TEST_GINKGO_TIMEOUT=${KUBEVIRT_FUNC_TEST_GINKGO_TIMEOUT:-4h}

source hack/common.sh
source hack/config.sh

_default_previous_release_registry="quay.io/kubevirt"

previous_release_registry=${PREVIOUS_RELEASE_REGISTRY:-$_default_previous_release_registry}

functest_docker_prefix=${manifest_docker_prefix-${docker_prefix}}

echo "Using $kubevirt_test_config as test configuration"

if [[ ${KUBEVIRT_PROVIDER} == os-* ]] || [[ ${KUBEVIRT_PROVIDER} =~ (okd|ocp)-* ]]; then
    oc=${kubectl}
fi

virtctl_path=$(pwd)/_out/cmd/virtctl/virtctl
example_guest_agent_path=$(pwd)/_out/cmd/example-guest-agent/example-guest-agent

rm -rf $ARTIFACTS
mkdir -p $ARTIFACTS

function functest() {
    KUBEVIRT_FUNC_TEST_SUITE_ARGS="--ginkgo.trace
	    -apply-default-e2e-configuration \
	    -conn-check-ipv4-address=${conn_check_ipv4_address} \
	    -conn-check-ipv6-address=${conn_check_ipv6_address} \
	    -conn-check-dns=${conn_check_dns} \
	    -migration-network-nic=${migration_network_nic} \
	    ${KUBEVIRT_FUNC_TEST_SUITE_ARGS}"
    if [[ ${KUBEVIRT_PROVIDER} =~ .*(k8s-sriov).* ]] || [[ ${KUBEVIRT_SINGLE_STACK} == "true" ]]; then
        echo "Will skip test asserting the cluster is in dual-stack mode."
        KUBEVIRT_FUNC_TEST_SUITE_ARGS="-skip-dual-stack-test ${KUBEVIRT_FUNC_TEST_SUITE_ARGS}"
    fi

    _out/tests/ginkgo -timeout=${KUBEVIRT_FUNC_TEST_GINKGO_TIMEOUT} -r "$@" _out/tests/tests.test -- -kubeconfig=${kubeconfig} -container-tag=${docker_tag} -container-tag-alt=${docker_tag_alt} -container-prefix=${functest_docker_prefix} -image-prefix-alt=${image_prefix_alt} -oc-path=${oc} -kubectl-path=${kubectl} -installed-namespace=${namespace} -previous-release-tag=${PREVIOUS_RELEASE_TAG} -previous-release-registry=${previous_release_registry} -deploy-testing-infra=${deploy_testing_infra} -config=${kubevirt_test_config} --artifacts=${ARTIFACTS} --operator-manifest-path=${OPERATOR_MANIFEST_PATH} --testing-manifest-path=${TESTING_MANIFEST_PATH} ${KUBEVIRT_FUNC_TEST_SUITE_ARGS} -virtctl-path=${virtctl_path} -example-guest-agent-path=${example_guest_agent_path}
}

additional_test_args=""
if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
    additional_test_args="${additional_test_args} --skip=${KUBEVIRT_E2E_SKIP}"
fi

if [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
    additional_test_args="${additional_test_args} --focus=${KUBEVIRT_E2E_FOCUS}"
fi

if [ "$KUBEVIRT_E2E_PARALLEL" == "true" ]; then
    additional_test_args="--nodes=${KUBEVIRT_E2E_PARALLEL_NODES} ${additional_test_args}"
fi

set -x
functest ${additional_test_args} ${KUBEVIRT_FUNC_TEST_GINKGO_ARGS} "${KUBEVIRT_FUNC_TEST_LABEL_FILTER}"
