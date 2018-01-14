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

PROVIDER=${PROVIDER:-vagrant-kubernetes}

source hack/common.sh
source hack/config.sh

# Get k8s version and set environment variable KUBE_VERSION
if [ "$PROVIDER" = "vagrant-openshift" ]; then
    KUBE_VERSION=$(./cluster/kubectl.sh version | grep kubernetes | head -n 1 | cut -f 2 -d ' ')
else
    KUBE_VERSION=$(./cluster/kubectl.sh version | head -n 1 | cut -f 3 -d ',' | cut -f 2 -d ':')
fi
KUBE_VERSION=$(echo $KUBE_VERSION | cut -f 1 -d '+' | cut -f 2 -d 'v')
export KUBE_VERSION

${TESTS_OUT_DIR}/tests.test -kubeconfig=${kubeconfig} -test.timeout 30m ${FUNC_TEST_ARGS}
