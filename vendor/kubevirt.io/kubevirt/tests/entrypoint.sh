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

mkdir -p "${RESULTS_DIR}"
mkdir -p "${TEST_MANIFESTS_DIR}"

# Generate manifests for testing
for input in $(find manifests/testing -type f -name "*.yaml.in") ;
do
    # Strip .in suffix from file name
    outfile=${TEST_MANIFESTS_DIR}/$(basename ${input%%.in})
    # Format manifest
    ./manifest-templator "$@" \
        --generated-manifests-dir=manifests/generated \
        --input-file=${input} > ${outfile}
done

# Execute tests
./tests.test "$@"
