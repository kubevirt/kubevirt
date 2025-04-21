#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#

set -e

source $(dirname "$0")/common.sh

FOLDERS="${KUBEVIRT_DIR}/cmd/ ${KUBEVIRT_DIR}/pkg/ ${KUBEVIRT_DIR}/staging/src/kubevirt.io/ ${KUBEVIRT_DIR}/tests/framework/"

ginkgobin=$(realpath _out/tests/ginkgo)
# Find every folder containing tests
for dir in $(find ${FOLDERS} -type f -name '*_test.go' -printf '%h\n' | sort -u); do
    # If there is no file ending with _suite_test.go, bootstrap ginkgo
    SUITE_FILE=$(find $dir -maxdepth 1 -type f -name '*_suite_test.go')
    if [ -z "$SUITE_FILE" ]; then
        echo "Missing test suite entrypoint; attempt to create one automatically"
        (cd $dir && $ginkgobin bootstrap)
    fi
done
