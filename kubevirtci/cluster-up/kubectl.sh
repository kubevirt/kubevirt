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

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/"
        echo "$(pwd)/"
    )"
fi

source ${KUBEVIRTCI_PATH}/hack/common.sh
source ${KUBEVIRTCI_CLUSTER_PATH}/$KUBEVIRT_PROVIDER/provider.sh
source ${KUBEVIRTCI_PATH}/hack/config.sh

if [ "$1" == "console" ] || [ "$1" == "vnc" ] || [ "$1" == "start" ] || [ "$1" == "stop" ] || [ "$1" == "migrate" ]; then
    ${KUBEVIRTCI_PATH}/virtctl.sh "$@"
elif [ "$1" == "virt" ]; then
    shift
    ${KUBEVIRTCI_PATH}/virtctl.sh "$@"
else
    _kubectl "$@"
fi
