#!/bin/bash
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
# Copyright 2019 Red Hat, Inc.
#
set -euo pipefail

export TIMESTAMP=${TIMESTAMP:-1}

function usage {
    cat <<EOF
usage: [NUM_TESTS=x] [NEW_TESTS=test1_test|...|testn_test] $0 [kubevirtci_provider[ kubevirtci_provider ...]]

    run test lanes repeatedly using the set of test files that have been
    changed or added since last merge commit, set NEW_TESTS to explicitly name the tests to run)

    options:
        NUM_TESTS       how often each test lane is run, default is 3
        NEW_TESTS       what set of tests to run, defaults to all test files added or changed since
                        last merge commit
        TARGET_COMMIT   the commit id to use when fetching the changed test files
                        note: leaving TARGET_COMMIT empty only works if on a git branch different from main.
                        If /clonerefs is at work you need to provide a target commit, as then the latest commit is a
                        merge commit (resulting in no changes detected)

    example:

        NEW_TESTS='operator_test' ./automation/repeated_test.sh 'k8s-1.16.2'

        runs tests/operator_test.go three times on kubevirtci provider k8s-1.16.2

EOF
}

function new_tests {
    local target_commit
    target_commit="$1"
    # 1. fetch the names of all added, copied, modified or renamed files
    #    from within the tests/ directory
    # 2. print only the last column of the line (in case of rename this is the new name)
    # 3. grep all files ending with '.go' but not with '_suite.go'
    # 4. replace newline with `|`
    # 5. remove last `|`
    git diff --diff-filter=ACMR --name-status "${target_commit}".. -- tests/ \
        | awk '{print $NF}' \
        | grep '\.go' \
        | grep -v '_suite\.go' \
        | tr '\n' '|' \
        | sed -E 's/\|$//'
}

# taking into account that each file containing changed tests is run several times per default
# and considering overhead of cluster-up etc., we should just skip the run
# if the total number of tests for all runs gets higher than the total number of tests
function should_skip_test_run_due_to_too_many_tests() {
    local new_tests="$1"
    local test_start_pattern='(Specify|It|Entry)\('
    local tests_total_estimate=0
    while IFS= read -r -d '' test_file_name; do
        tests_total_estimate=$(( tests_total_estimate + $(grep -hcE "${test_start_pattern}" "$test_file_name") ))
    done < <(find tests/ -name '*.go' -print0)
    local tests_to_run_estimate=0
    for test_file_name in $(echo "${new_tests}" | tr '|' '\n'); do
        set +e
        tests_to_run_estimate=$(( tests_to_run_estimate + $(grep -hcE "${test_start_pattern}" "$test_file_name") ))
        set -e
    done
    local tests_total_for_all_runs_estimate
    tests_total_for_all_runs_estimate=$(( tests_to_run_estimate * NUM_TESTS * ${#TEST_LANES[@]} ))
    echo -e "Estimates:\ttests_total_estimate: $tests_total_estimate\ttests_total_for_all_runs_estimate: $tests_total_for_all_runs_estimate"
    [ "$tests_total_for_all_runs_estimate" -gt $tests_total_estimate ]
}


if (( $# > 0 )); then
    if [[ "$1" =~ -h ]]; then
        usage
        exit 0
    fi
fi

if (( $# > 0 )); then
    declare -a TEST_LANES
    max="$#"
    while [ ! $max -lt 1 ]; do
        max=$((max-1))
        TEST_LANES[$max]="$1"
        shift
    done
else
    mapfile -t TEST_LANES < <( find cluster-up/cluster -name 'k8s-[0-9].[0-9][0-9]' -print | sort -rV | grep -oE 'k8s.*' | head -3 )
fi
echo "Test lanes: ${TEST_LANES[*]}"
for lane in "${TEST_LANES[@]}"; do
    [ -d "cluster-up/cluster/$lane" ] || ( echo "provider $lane does not exist!"; exit 1 )
done

if [[ -z ${TARGET_COMMIT-} ]]; then
    # if there's no commit provided default to the latest merge commit
    TARGET_COMMIT=$(git log -1 --format=%H --merges)
fi

if [[ -z ${NEW_TESTS-} ]]; then

    set +e # required due to grep barking when it does not have any input
    NEW_TESTS=$(new_tests $TARGET_COMMIT)
    set -e

    # skip certain tests for now, as we don't have a strategy currently
    NEW_TESTS=$(echo "$NEW_TESTS" | sed -E 's/\|?[^\|]*(sriov|multus|windows|gpu|mdev)[^\|]*//g' | sed 's/^|//')
fi
if [[ -z "${NEW_TESTS}" ]]; then
    echo "Nothing to test"
    exit 0
fi
echo "Test files touched: $(echo ${NEW_TESTS} | tr '|' ',')"

NUM_TESTS=${NUM_TESTS-3}
echo "Number of per lane runs: $NUM_TESTS"

if should_skip_test_run_due_to_too_many_tests "${NEW_TESTS}"; then
    echo "Skipping run due to number of tests in total being too high for repeated run."
    exit 0
fi

trap '{ make cluster-down; }' EXIT SIGINT SIGTERM

for lane in "${TEST_LANES[@]}"; do

    echo "test lane: $lane, preparing cluster up"

    export KUBEVIRT_PROVIDER="$lane"
    export KUBEVIRT_NUM_NODES=2
    export KUBEVIRT_WITH_CNAO="true"
    export KUBEVIRT_DEPLOY_CDI="true"
    export KUBEVIRT_NUM_SECONDARY_NICS=1
    make cluster-up

    for i in $(seq 1 "$NUM_TESTS"); do
        echo "test lane: $lane, run: $i"
        make cluster-sync
        ginko_params="-noColor -succinct -slowSpecThreshold=30 -focus=${NEW_TESTS} -skip=QUARANTINE -regexScansFilePath=true"
        FUNC_TEST_ARGS="$ginko_params" make functest
        if [[ $? -ne 0 ]]; then
            echo "test lane: $lane, run: $i, tests failed!"
            exit 1
        fi
    done

    make cluster-down

done

exit 0
