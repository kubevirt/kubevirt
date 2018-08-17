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

source hack/build/common.sh
source hack/build/config.sh

kubectl=${CDI_DIR}/cluster/.kubectl

if [[ ${TARGET} == openshift* ]]; then
    oc=${kubectl}
fi

echo ${kubectl}

kubeconfig=${CDI_DIR}/cluster/.kubeconfig
# ${TESTS_OUT_DIR}/tests.test -kubeconfig=${kubeconfig} -oc-path=${oc} -kubectl-path=${kubectl} -test.timeout 90m ${FUNC_TEST_ARGS}
go test -v ./tests/... -args -ginkgo.v -kubeconfig=${kubeconfig} -oc-path=${oc} -kubectl-path=${kubectl} -test.timeout 90m ${FUNC_TEST_ARGS}
