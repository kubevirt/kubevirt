#!/bin/bash
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

source hack/common.sh
source hack/config.sh

functest_container_prefix=${manifest_container_prefix-${container_prefix}}

if [[ ${TARGET} == openshift* ]]; then
    oc=${kubectl}
fi

# Some tooling is built utilizing lowercase kubeconfig and others utilizing
# uppercase KUBECONFIG E.G. cluster/ tools
# Test for the normal lower case first and if it doesn't exists use uppercase.
if [ -n "$kubeconfig" ]; then
    KUBECONFIG_ARGS="-kubeconfig=${kubeconfig}"
elif [ -n "$KUBECONFIG" ]; then
    KUBECONFIG_ARGS="-kubeconfig=${KUBECONFIG}"
fi

${TESTS_OUT_DIR}/tests.test ${KUBECONFIG_ARGS} -container-tag=${docker_tag} -container-prefix=${functest_docker_prefix} -oc-path=${oc} -kubectl-path=${kubectl} -test.timeout 180m ${FUNC_TEST_ARGS} -installed-namespace=${namespace} -deploy-testing-infra -path-to-testing-infra-manifests=${MANIFESTS_OUT_DIR}/testing

