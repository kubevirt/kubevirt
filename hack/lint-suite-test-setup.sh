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
# Copyright the KubeVirt Authors.
#
# Ensures all Ginkgo test suite files use KubeVirtTestSuiteSetup instead of
# the plain ginkgo bootstrap (RegisterFailHandler + RunSpecs). The shared
# setup provides log redirection, gomega format settings, and bazel JUnit
# XML integration.
#
# See https://github.com/kubevirt/kubevirt/issues/15675

set -eo pipefail

# Files that legitimately cannot use KubeVirtTestSuiteSetup:
#   - tests/tests_suite_test.go: functional test suite with its own setup
#   - staging/.../apitesting: not a ginkgo suite (regular go test)
#   - staging/.../client-go/log: inlines the logic to avoid circular import
EXCLUDE_FILES="
tests/tests_suite_test.go
staging/src/kubevirt.io/api/apitesting/apitesting_suite_test.go
staging/src/kubevirt.io/client-go/log/log_suite_test.go
"

exit_code=0

for f in $(find . -name '*_suite_test.go' -not -path './vendor/*' | sed 's|^\./||' | sort); do
    if echo "$EXCLUDE_FILES" | grep -Fqx -- "$f"; then
        continue
    fi

    if ! grep -q 'KubeVirtTestSuiteSetup' "$f"; then
        echo "ERROR: $f does not use testutils.KubeVirtTestSuiteSetup(t)"
        echo "       Replace the plain ginkgo bootstrap with:"
        echo "           testutils.KubeVirtTestSuiteSetup(t)"
        echo "       See https://github.com/kubevirt/kubevirt/issues/15675"
        echo ""
        exit_code=1
    fi
done

if [ "$exit_code" -ne 0 ]; then
    exit 1
fi

echo "All suite test files use KubeVirtTestSuiteSetup."
