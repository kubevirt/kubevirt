#!/bin/bash -xe
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
# Copyright 2021 Red Hat, Inc.
#
# This script checks the defaulting mechanism

function isOperatorConditionSupported() {
  echo "Testing the operator condition"
  if ! ${KUBECTL_BINARY} get crd | grep "operatorconditions.operators.coreos.com"; then
    echo "Not running with OLM, or Operator Condition is not supported. Exiting"
    return 1
  fi
  return 0
}

function getOperatorConditionName() {
  ${KUBECTL_BINARY} get -n "${INSTALLED_NAMESPACE}" OperatorCondition -o name | grep hyperconverged-operator | grep "$1"
}

function getOperatorConditionUpgradeable() {
  ${KUBECTL_BINARY} get -n "${INSTALLED_NAMESPACE}" "$1" -o yaml
}

function printOperatorCondition() {
  if isOperatorConditionSupported; then
    name=$(getOperatorConditionName "$1")
    echo "reading Operator Condition ${name}"
    getOperatorConditionUpgradeable "${name}"
  else
    echo "Operator Condition is not supported"
  fi
}