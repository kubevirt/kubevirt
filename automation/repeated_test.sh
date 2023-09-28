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

# According to https://dl.acm.org/doi/abs/10.1145/3476105
#
# * almost all flaky tests were independent of the execution platform, so environmental issues should be considered with higher priority
# * devs may be unaware of many flakes due to order-dependent tests
# * simple techniques of reversing or shuffling the test order may be more efficient and effective than more sophisticated approaches
# * “One study found that 88% of flaky tests were found to consecutively fail up to a maximum of five times before passing,
#    though another reported finding new flaky tests even after 10,000 test suite runs.”
#
# Therefore we by default
# * use only latest kubevirtci provider,
# * run all changed tests five times and
# * randomize the test order each time

# include defaults for retrieving proper vendored cluster-up version
export DOCKER_TAG_ALT=''
export IMAGE_PREFIX=''
export IMAGE_PREFIX_ALT=''
source hack/config-default.sh

export TIMESTAMP=${TIMESTAMP:-1}

function usage {
    cat <<EOF
usage: [NUM_TESTS=x] [NEW_TESTS=tests/file_1.go|...|tests/file_n.go [TARGET_COMMIT=a1b2c3d4] $0 [TEST_LANE] [--dry-run]

    run tests repeatedly using the set of test files that have been changed or added since last merge commit
    set NEW_TESTS to explicitly name the test files to run

    options:
        NUM_TESTS       how often the test lane is run, default is 5
        NEW_TESTS       what set of tests to run, defaults to all test files added or changed since
                        last merge commit
        TARGET_COMMIT   the commit id to use when fetching the changed test files
                        note: leaving TARGET_COMMIT empty only works if on a git branch different from main.
                        If /clonerefs is at work you need to provide a target commit, as then the latest commit is a
                        merge commit (resulting in no changes detected)
        TEST_LANE       the kubevirtci provider to use, if not given, use latest stable one

    examples:

      1.    NEW_TESTS='tests/operator_test.go' ./automation/repeated_test.sh 'k8s-1.27'

            runs tests/operator_test.go x times on kubevirtci provider k8s-1.27

      2.    NEW_TESTS='tests/operator_test.go' ./automation/repeated_test.sh

            runs tests/operator_test.go x times on latest stable kubevirtci provider found

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
        | grep -vE '_suite(_test)\.go' \
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
    tests_total_for_all_runs_estimate=$(( tests_to_run_estimate * NUM_TESTS ))
    echo -e "Estimates:\ttests_total_estimate: $tests_total_estimate\ttests_total_for_all_runs_estimate: $tests_total_for_all_runs_estimate"
    [ "$tests_total_for_all_runs_estimate" -gt $tests_total_estimate ]
}

ginko_params=''
if (( $# > 0 )); then
    if [[ "$1" =~ -h ]]; then
        usage
        exit 0
    fi

    if [[ "$1" =~ --dry-run ]]; then
        ginko_params='-dry-run'
        shift
    fi
fi

if (( $# > 0 )); then
    TEST_LANE="$1"
    shift
else
    # We only want to use stable providers for flake testing, thus we fetch the k8s version file from kubevirtci.
    # we stop at the first provider that is stable (aka doesn't have an rc or beta or alpha version)
    for k8s_provider in $(cd cluster-up/cluster && ls -rd k8s-[0-9]\.[0-9][0-9]); do
        # shellcheck disable=SC2154
        k8s_provider_version=$(curl --fail "https://raw.githubusercontent.com/kubevirt/kubevirtci/${kubevirtci_git_hash}/cluster-provision/k8s/${k8s_provider#"k8s-"}/version")
        if [[ "${k8s_provider_version}" =~ -(rc|alpha|beta) ]]; then
            echo "Skipping ${k8s_provider_version}"
        else
            TEST_LANE="${k8s_provider}"
            break
        fi
    done
    if [[ ${TEST_LANE} == "" ]]; then
        echo "No stable provider found"
        exit 1
    fi
fi
echo "Test lane: ${TEST_LANE}"
[ -d "cluster-up/cluster/${TEST_LANE}" ] || ( echo "provider ${TEST_LANE} does not exist!"; exit 1 )

if [[ -z ${TARGET_COMMIT-} ]]; then
    # if there's no commit provided default to the latest merge commit
    TARGET_COMMIT=$(git log -1 --format=%H --merges)
fi

if [[ -z ${NEW_TESTS-} ]]; then

    set +e # required due to grep barking when it does not have any input
    NEW_TESTS=$(new_tests "$TARGET_COMMIT")
    set -e

    # skip certain tests for now, as we don't have a strategy currently
    NEW_TESTS=$(echo "$NEW_TESTS" | sed -E 's/\|?[^\|]*(sriov|multus|windows|gpu|mdev)[^\|]*//g' | sed 's/^|//')
fi
if [[ -z "${NEW_TESTS}" ]]; then
    echo "Nothing to test"
    exit 0
fi
echo "Test files touched: $(echo "${NEW_TESTS}" | tr '|' ',')"

NUM_TESTS=${NUM_TESTS-5}
echo "Number of per lane runs: $NUM_TESTS"

if should_skip_test_run_due_to_too_many_tests "${NEW_TESTS}"; then
    echo "Skipping run due to number of tests in total being too high for repeated run."
    exit 0
fi

# for some tests we need three nodes aka two nodes with cpu manager installed, thus we grep whether the skip is present
KUBEVIRT_NUM_NODES=2
# shellcheck disable=SC2046
if grep -q 'SkipTestIfNotEnoughNodesWithCPUManager' $(echo "${NEW_TESTS}" | tr '|' ' '); then
    KUBEVIRT_NUM_NODES=3
fi

trap '{ make cluster-down; }' EXIT SIGINT SIGTERM

# Give the nodes enough memory to run tests in parallel, including tests which involve fedora
export KUBEVIRT_MEMORY_SIZE='9216M'

export KUBEVIRT_NUM_NODES
export KUBEVIRT_WITH_CNAO="true"
export KUBEVIRT_DEPLOY_CDI="true"
export KUBEVIRT_NUM_SECONDARY_NICS=1
export KUBEVIRT_STORAGE="rook-ceph-default"
export KUBEVIRT_DEPLOY_NFS_CSI=true
export KUBEVIRT_DEPLOY_PROMETHEUS=true

export KUBEVIRT_PROVIDER="${TEST_LANE}"

ginko_params="$ginko_params -no-color -succinct --label-filter=!QUARANTINE -randomize-all"
for test_file in $(echo "${NEW_TESTS}" | tr '|' '\n'); do
    ginko_params+=" -focus-file=${test_file}"
done

echo "Test lane: ${TEST_LANE}, preparing cluster up"

if [[ ! "$ginko_params" =~ -dry-run ]]; then
    make cluster-up
    make cluster-sync
else
    NUM_TESTS=1
fi

for i in $(seq 1 "$NUM_TESTS"); do
    echo "Test lane: ${TEST_LANE}, run: $i"
    if ! FUNC_TEST_ARGS="$ginko_params" make functest; then
        echo "Test lane: ${TEST_LANE}, run: $i, tests failed!"
        exit 1
    fi
done
