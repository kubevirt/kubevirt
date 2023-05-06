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

PROJECT_ROOT="$(readlink -e "$(dirname "${BASH_SOURCE[0]}")"/../)"
metrics_doc_path="${PROJECT_ROOT}/docs/metrics.md"

# Run the metric name linter and store the output in errors
errors=$(go run "${PROJECT_ROOT}"/tools/prom-metric-linter/*.go "$metrics_doc_path" 2>&1)

# Check if there were any errors, if yes print and fail
if [[ $errors != "" ]]; then
    echo "$errors"
    exit 1
fi
