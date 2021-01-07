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


set -euo pipefail

KUBECTL_BINARY="${KUBECTL_BINARY:-"kubectl"}"
HCO_NAMESPACE="${HCO_NAMESPACE:-"kubevirt-hyperconverged"}"
HCO_RESOURCE_NAME="${HCO_RESOURCE_NAME:-"kubevirt-hyperconverged"}"

function check_label_value() {
  local labels_as_json="$1"
  local key="$2"
  local expected="$3"

  found="$(echo "$labels_as_json" |jq '."'$key'"' |tr -d '"')"
  if [[ "$found" != "$expected" ]]; then
    echo "Value for label $key is not correct! Expected:'$expected' Found:'$found'"
    exit 1
  fi
}


function check_label_has_value() {
  local labels_as_json="$1"
  local key="$2"

  found="$(echo "$labels_as_json" |jq '."'$key'"' |tr -d '"')"
  if [[ "$found" == "null" ]]; then
    echo "Label $key does not exist!"
    exit 1
  fi
}

function check_labels(){
  local labels_as_json="$1"
  check_label_value "$labels_as_json" "app" "kubevirt-hyperconverged"
  check_label_value "$labels_as_json" "app.kubernetes.io/part-of" "hyperconverged-cluster"
  check_label_value "$labels_as_json" "app.kubernetes.io/managed-by" "hco-operator"

  check_label_has_value "$labels_as_json" "app.kubernetes.io/component"
  check_label_has_value "$labels_as_json" "app.kubernetes.io/version"
}

echo "Fetching related objects..."
related_obj_json="$(${KUBECTL_BINARY} get  hco "${HCO_RESOURCE_NAME}" -n "${HCO_NAMESPACE}" -o jsonpath='{.status .relatedObjects}')"

echo "Putting related objects into array..."
related_obj_array=()
while IFS='' read -r line; do related_obj_array+=("$line"); done < <(echo "$related_obj_json" |jq '.[] | .kind + "," + .name + "," + .namespace' |tr -d '"')

for obj in "${related_obj_array[@]}"
do
  type="$(echo "$obj" |cut -d',' -f1)"
  name="$(echo "$obj" |cut -d',' -f2)"
  namespace="$(echo "$obj" |cut -d',' -f3)"

  namespace_flag=""
  if [[ "$namespace" != "" ]]; then
    namespace_flag="-n $namespace"
  fi

  echo "Checking labels of $type/$name"
  labels_of_ojb="$(${KUBECTL_BINARY} get "$type" "$name" $namespace_flag -o json |jq .metadata.labels)"
  check_labels "$labels_of_ojb"
done

