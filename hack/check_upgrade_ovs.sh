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
# Copyright 2020 Red Hat, Inc.
#

set -ex

if [[ -z ${PREVIOUS_OVS_ANNOTATION} && ${PREVIOUS_OVS_STATE} == '{}' ]] || [[ ${PREVIOUS_OVS_ANNOTATION} == 'true' ]]; then
  # if the annotation did not exist and OVS was running, or if it existed and equal to 'true' - OVS should be deployed in new version

  echo "check that OVS annotation in HCO CR in new version is set to true"
  [[ $(${CMD} get HyperConverged kubevirt-hyperconverged -n kubevirt-hyperconverged -o jsonpath='{.metadata.annotations.deployOVS}') == 'true' ]]

  echo "check that OVS exists in CNAO CR Spec"
  [[ $(${CMD} get networkaddonsconfigs cluster -o jsonpath='{.spec.ovs}') == '{}' ]]

  echo "check that OVS DaemonSet exists"
  [[ $(${CMD} get ds ovs-cni-amd64 -n kubevirt-hyperconverged --no-headers --ignore-not-found | wc -l) == '1' ]]

elif [[ -z ${PREVIOUS_OVS_ANNOTATION} && -z ${PREVIOUS_OVS_STATE} ]] || [[ -n ${PREVIOUS_OVS_ANNOTATION} && ${PREVIOUS_OVS_ANNOTATION} != 'true' ]]; then
  # if the annotation did not exist and OVS was not running,
  # or if the annotation existed and was not 'true' - OVS should not be deployed in new version

  echo "check that OVS annotation in HCO CR in new version is set to false"
  [[ $(${CMD} get HyperConverged kubevirt-hyperconverged -n kubevirt-hyperconverged -o jsonpath='{.metadata.annotations.deployOVS}') == 'false' ]]

  echo "check that OVS does not exist in CNAO CR Spec"
  [[ $(${CMD} get networkaddonsconfigs cluster -o jsonpath='{.spec.ovs}') == '' ]]

  echo "check that OVS DaemonSet does not exist"
  [[ $(${CMD} get ds ovs-cni-amd64 -n kubevirt-hyperconverged --no-headers --ignore-not-found | wc -l) == '0' ]]

else
  echo "OVS opt-in test did not run. PREVIOUS_OVS_ANNOTATION=${PREVIOUS_OVS_ANNOTATION}, PREVIOUS_OVS_STATE=${PREVIOUS_OVS_STATE}"
  exit 1
fi

echo "OVS Opt-in test completed successfully."

