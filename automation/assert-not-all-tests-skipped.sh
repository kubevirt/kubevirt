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
# Copyright 2021 Red Hat, Inc.
#
#

set -euo pipefail

function usage() {
    cat <<EOF
usage: $0 <junit_xml_file>

    Check that the number of skipped testcases is lesser than the number of testcases,
    exit with non-zero exit code if that is not the case.

EOF
}

function main() {
    if [ "$#" -lt 1 ]; then
        usage
        exit 1
    fi
    if [ ! -f "$1" ]; then
        usage
        echo "Test results file $1 does not exist or is not a file!"
        exit 1
    fi
    local test_results_file="$1"

    number_of_testcases=$(grep -c '<testcase' "$test_results_file")
    number_of_skipped_testcases=$(grep -c '<skipped' "$test_results_file")

    echo "Testcases executed:" "$number_of_testcases"
    echo "Testcases skipped:" "$number_of_skipped_testcases"

    if [ ! "$number_of_testcases" -gt "$number_of_skipped_testcases" ]; then
        echo "ERROR: all test cases have been skipped!"
        exit 1
    fi
}

main "$@"
