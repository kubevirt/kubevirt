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
# Copyright 2023 Red Hat, Inc.
#

set -ex

# Check the deployment name has been updated after upgrade
NEW_DEPLOYMENT_NAME="kubevirt-console-plugin"
OLD_DEPLOYMENT_NAME="kubevirt-plugin"
[[ $(${KUBECTL_BINARY} get deployment ${NEW_DEPLOYMENT_NAME} -n ${INSTALLED_NAMESPACE}) ]]
[[ ! $(${KUBECTL_BINARY} get deployment ${OLD_DEPLOYMENT_NAME} -n ${INSTALLED_NAMESPACE}) ]]

# Check the service name has been updated after upgrade
NEW_SERVICE_NAME="${NEW_DEPLOYMENT_NAME}-service"
OLD_SERVICE_NAME="${OLD_DEPLOYMENT_NAME}-service"
[[ $(${KUBECTL_BINARY} get svc ${NEW_SERVICE_NAME} -n ${INSTALLED_NAMESPACE}) ]]
[[ ! $(${KUBECTL_BINARY} get svc ${OLD_SERVICE_NAME} -n ${INSTALLED_NAMESPACE}) ]]

# Check the ConsolePlugin points to the new service name
[[ $(${KUBECTL_BINARY} get consoleplugin ${OLD_DEPLOYMENT_NAME} -o jsonpath='{.spec.backend.service.name}') == "${NEW_SERVICE_NAME}" ]]

