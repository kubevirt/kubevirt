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
# Copyright 2018 Red Hat, Inc.
#

set -e

>&2 echo "WARNING: usage of '${BASH_SOURCE[0]}' is deprecated!"
>&2 echo "         see: https://github.com/kubevirt/kubevirtci/issues/1277"

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi

source ${KUBEVIRTCI_PATH}/hack/common.sh
source ${KUBEVIRTCI_CLUSTER_PATH}/$KUBEVIRT_PROVIDER/provider.sh
source ${KUBEVIRTCI_PATH}/hack/config.sh

CONFIG_ARGS=

if [ -n "$kubeconfig" ]; then
    CONFIG_ARGS="--kubeconfig=${kubeconfig}"
elif [ -n "$KUBECONFIG" ]; then
    CONFIG_ARGS="--kubeconfig=${KUBECONFIG}"
fi

KUBEVIRT_OUT_PATH=${KUBEVIRTCI_PATH}/../_out
if [ ! -d ${KUBEVIRT_OUT_PATH} ]; then
    # see https://github.com/kubevirt/kubevirt/pull/12872
    >&2 echo "WARNING: $KUBEVIRT_OUT_PATH not found, falling back to parent"
    KUBEVIRT_OUT_PATH=${KUBEVIRTCI_PATH}/../../_out
    >&2 echo "         $KUBEVIRT_OUT_PATH"
fi
${KUBEVIRT_OUT_PATH}/cmd/virtctl/virtctl $CONFIG_ARGS "$@"

