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

CDI_OPERATOR_URL=$(curl --silent "https://api.github.com/repos/kubevirt/containerized-data-importer/releases/latest" \
 | grep browser_download_url | grep "cdi-operator.yaml\"" | cut -d'"' -f4)
KUBEVIRT_OPERATOR_URL=$(curl --silent "https://api.github.com/repos/kubevirt/kubevirt/releases/latest" \
 | grep browser_download_url | grep "kubevirt-operator.yaml\"" | cut -d'"' -f4)
CNA_URL_PREFIX="https://raw.githubusercontent.com/kubevirt/cluster-network-addons-operator/master/manifests/cluster-network-addons/0.3.0"
SSP_URL_PREFIX="https://raw.githubusercontent.com/MarSik/kubevirt-ssp-operator/master/cluster/1.0.0"
KUBECTL=$(which kubectl 2> /dev/null)

KWEBUI_URL="https://raw.githubusercontent.com/kubevirt/web-ui-operator/master/deploy/operator.yaml"

if [ -z "${KUBECTL}" ]; then
    CMD=oc
else
    CMD=kubectl
fi
