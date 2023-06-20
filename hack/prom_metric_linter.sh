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
#
set -e

linter_image_tag="v0.0.1"

PROJECT_ROOT="$(readlink -e "$(dirname "${BASH_SOURCE[0]}")"/../)"
export METRICS_COLLECTOR_PATH="${METRICS_COLLECTOR_PATH:-${PROJECT_ROOT}/tools/prom-metrics-collector}"

if [[ ! -d "$METRICS_COLLECTOR_PATH" ]]; then
    echo "Invalid METRICS_COLLECTOR_PATH: $METRICS_COLLECTOR_PATH is not a valid directory path"
    exit 1
fi

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
    --operator-name=*)
        operator_name="${1#*=}"
        shift
        ;;
    --sub-operator-name=*)
        sub_operator_name="${1#*=}"
        shift
        ;;
    *)
        echo "Invalid argument: $1"
        exit 1
        ;;
    esac
done

# Get the metrics list
go build -o _out/prom-metrics-collector "$METRICS_COLLECTOR_PATH/..."
json_output=$(_out/prom-metrics-collector 2>/dev/null)

# Select container runtime
source hack/cri-bin.sh

# Run the linter by using the prom-metrics-linter Docker container
errors=$($CRI_BIN run -i "quay.io/kubevirt/prom-metrics-linter:$linter_image_tag" \
    --metric-families="$json_output" \
    --operator-name="$operator_name" \
    --sub-operator-name="$sub_operator_name" 2>/dev/null)

# Check if there were any errors, if yes print and fail
if [[ $errors != "" ]]; then
  echo "$errors"
  exit 1
fi
