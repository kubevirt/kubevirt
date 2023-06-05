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
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2023 Red Hat, Inc.
#
#

set -e

PROJECT_ROOT="$(readlink -e "$(dirname "${BASH_SOURCE[0]}")"/../../)"
export METRICS_DOC_PATH="${METRICS_DOC_PATH:-${PROJECT_ROOT}/docs/metrics.md}"
export METRICS_COLLECTOR_PATH="${METRICS_COLLECTOR_PATH:-${PROJECT_ROOT}/tools/prom-metrics-collector}"

if [[ ! -f "$METRICS_DOC_PATH" ]]; then
    echo "Invalid METRICS_DOC_PATH: $METRICS_DOC_PATH is not a valid file path"
    exit 1
fi

if [[ ! -d "$METRICS_COLLECTOR_PATH" ]]; then
    echo "Invalid METRICS_COLLECTOR_PATH: $METRICS_COLLECTOR_PATH is not a valid directory path"
    exit 1
fi

# Get the metrics list
go build -o _out/prom-metrics-collector "$METRICS_COLLECTOR_PATH/..."
json_output=$(_out/prom-metrics-collector "$METRICS_DOC_PATH" 2>/dev/null)

echo "$json_output"
