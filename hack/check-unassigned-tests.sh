#!/bin/bash
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


main() {
    skip="SRIOV|GPU|\\[sig-operator\\]|\\[sig-network\\]|\\[sig-storage\\]|\\[sig-compute\\]|\\[sig-compute-migrations\\]|\\[sig-performance\\]|\\[sig-compute-realtime\\]|\\[sig-monitoring\\]"
    result=$(FUNC_TEST_ARGS="--dry-run --no-color -skip=${skip}" make functest)
    total_tests=$(echo "${result}" | sed -n "s/.*Ran \([0-9]\+\) of .* Specs in .* seconds.*/\1/p")
    if [ "${total_tests}" != "0" ]; then
        echo "Found ${total_tests} tests not assigned to any SIG, please check: ${result}"
        exit 1
    fi
    labels="!(SRIOV,GPU,sig-operator,sig-network,sig-storage,sig-compute,sig-compute-migrations,sig-performance,sig-compute-realtime,sig-monitoring)"
    result=$(FUNC_TEST_ARGS="--dry-run --no-color --label-filter=${labels}" make functest)
    total_tests=$(echo "${result}" | sed -n "s/.*Ran \([0-9]\+\) of .* Specs in .* seconds.*/\1/p")
    if [ "${total_tests}" != "0" ]; then
        echo "Found ${total_tests} tests not assigned to any SIG, please check: ${result}"
        exit 1
    fi
}

main "${@}"
