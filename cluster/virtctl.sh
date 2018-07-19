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
# Copyright 2018 Red Hat, Inc.
#

set -e

source $(dirname "$0")/../hack/common.sh

source ${KUBEVIRT_DIR}/cluster/$KUBEVIRT_PROVIDER/provider.sh
source ${KUBEVIRT_DIR}/hack/config.sh

CONFIG_ARGS=

if [ -n "$kubeconfig" ]; then
    CONFIG_ARGS="--kubeconfig=${kubeconfig}"
elif [ -n "$KUBECONFIG" ]; then
    CONFIG_ARGS="--kubeconfig=${KUBECONFIG}"
fi

VIRTCTL=virtctl
if [ "$(uname -s)" = "Darwin" ]; then
    VIRTCTL='virtctl-darwin'
fi

${KUBEVIRT_DIR}/_out/cmd/virtctl/${VIRTCTL} $CONFIG_ARGS "$@"
