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
# Copyright The KubeVirt Authors.
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

KUBEVIRT_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")

# include defaults for retrieving proper vendored cluster-up version
export DOCKER_TAG_ALT=''
export IMAGE_PREFIX=''
export IMAGE_PREFIX_ALT=''
source hack/config-default.sh

export TIMESTAMP=${TIMESTAMP:-1}

function usage() {
    cat <<EOF
usage: [NUM_TESTS=x] \
       [NEW_TESTS=file_name.json] \
       [TARGET_COMMIT_RANGE=a1b2c3d4] $0 [TEST_LANE] [--dry-run]

    run set of tests repeatedly using a json file containing the names of the tests that have been
    changed or added since last merge commit
    hint: set NEW_TESTS to explicitly name the json file containing test names to run

    options:
        NUM_TESTS           how many times the test lane is run, default is 5
        NEW_TESTS           the json file containing the (textual) names of the
                            tests to run, according to what is shown in the
                            junit.xml file
        TARGET_COMMIT_RANGE the commit id to use when fetching the changed
                            test files
                            note: leaving TARGET_COMMIT_RANGE empty only works
                            if on a git branch different from main.
                            If /clonerefs is at work you need to provide a
                            target commit, as then the latest commit is a
                            merge commit (resulting in no changes detected)
        TEST_LANE           the kubevirtci provider to use, if not given,
                            use latest stable one

    examples:

      1.    NEW_TESTS='/tmp/changed-tests.json' ./automation/repeated_test.sh 'k8s-1.27'

            runs set of tests contained in /tmp/changed-tests.json x times
            on kubevirtci provider k8s-1.27

      2.    NEW_TESTS='/tmp/changed-tests.json' ./automation/repeated_test.sh

            runs set of tests contained in /tmp/changed-tests.json x times
            on latest stable kubevirtci provider found

EOF
}

function new_tests() {
    local target_commit_range
    target_commit_range="$1"
    if [ -n "${target_commit_range}" ]; then
        target_commit_range='-r '"${target_commit_range}"
    fi

    # The CANNIER project provides (among others) the command `extract changed-tests` that extracts the names of
    # changed tests from a commit range into a json file.
    #
    # For reference - the latter is part of a bigger effort implementing a new re-run strategy
    # leveraging ML to predict test flakiness. Initial implementation of the CANNIER approach set of tools is done here:
    # https://github.com/kubevirt/project-infra/pull/3930

    tmp_dir="$(mktemp -d)"
    podman run --rm \
        -v "${KUBEVIRT_ROOT}:/kubevirt/" \
        -v "${tmp_dir}:/tmp" \
        quay.io/kubevirtci/cannier:v20250227-c93cf50 \
        extract changed-tests ${target_commit_range} \
        -p /kubevirt \
        -t /kubevirt/tests/ \
        -o /tmp/changed-tests.json

    echo "${tmp_dir}/changed-tests.json"
}

# taking into account that each changed test is run several times
# and considering overhead of cluster-up etc., we should skip the run
# if the total number of tests for all runs gets higher than the total
# number of tests
function should_skip_test_run_due_to_too_many_tests() {
    local new_tests="$1"
    local test_start_pattern='(Specify|It|Entry)\('
    local tests_total_estimate=0
    while IFS= read -r -d '' test_file_name; do
        tests_total_estimate=$((tests_total_estimate + $(grep -hcE "${test_start_pattern}" "$test_file_name")))
    done < <(find tests/ -name '*.go' -print0)
    local tests_to_run_estimate
    tests_to_run_estimate=$(jq '. | length' "${new_tests}")
    local tests_total_for_all_runs_estimate
    tests_total_for_all_runs_estimate=$((tests_to_run_estimate * NUM_TESTS))
    echo -e "Estimates:\ttests_total_estimate: $tests_total_estimate\ttests_total_for_all_runs_estimate: $tests_total_for_all_runs_estimate"
    [ "$tests_total_for_all_runs_estimate" -gt $tests_total_estimate ]
}

ginkgo_params=''
if (($# > 0)); then
    if [[ "$1" =~ -h ]]; then
        usage
        exit 0
    fi

    if [[ "$1" =~ --dry-run ]]; then
        ginkgo_params='-dry-run'
        shift
    fi
fi

if (($# > 0)); then
    TEST_LANE="$1"
    shift
else
    # We only want to use stable providers for flake testing, thus we fetch the k8s version file from kubevirtci.
    # we stop at the first provider that is stable (aka doesn't have an rc or beta or alpha version)
    for k8s_provider in $(cd kubevirtci/cluster-up/cluster && ls -rd k8s-[0-9]\.[0-9][0-9]); do
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
[ -d "kubevirtci/cluster-up/cluster/${TEST_LANE}" ] || (
    echo "provider ${TEST_LANE} does not exist!"
    exit 1
)

if [[ -z ${TARGET_COMMIT_RANGE-} ]]; then
    # if there's no commit provided default to the latest merge commit
    TARGET_COMMIT_RANGE="$(git log -1 --format=%H --merges).."
fi

if [[ -z ${NEW_TESTS-} ]]; then
    NEW_TESTS=$(new_tests "$TARGET_COMMIT_RANGE")
fi

if [[ -z "${NEW_TESTS}" ]]; then
    echo "Nothing to test"
    exit 0
fi

tests_changed=$(jq '. | length' "${NEW_TESTS}")
echo "Tests changed: ${tests_changed}"
if ((tests_changed == 0)); then
    echo "Nothing to test"
    exit 0
fi

NUM_TESTS=${NUM_TESTS-5}
echo "Number of per lane runs: $NUM_TESTS"

if should_skip_test_run_due_to_too_many_tests "${NEW_TESTS}"; then
    echo "Skipping run due to number of tests in total being too high for repeated run."
    exit 0
fi

# for some tests we need three nodes aka two nodes with cpu manager installed
# TODO: check whether no test with label `RequiresTwoWorkerNodesWithCPUManager` is present, in that case use two nodes
KUBEVIRT_NUM_NODES=3

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
export KUBEVIRT_DEPLOY_NET_BINDING_CNI=true

export KUBEVIRT_PROVIDER="${TEST_LANE}"

# add_to_label_filter appends the given label and separator to
# $label_filter which is passed to Ginkgo --filter-label flag.
# How to use:
# - Run tests with label
#     add_to_label_filter '(mylabel)' ','
# - Dont run tests with label:
#     add_to_label_filter '(!mylabel)' '&&'
add_to_label_filter() {
    local label=$1
    local separator=$2
    if [[ -z $label_filter ]]; then
        label_filter="${1}"
    else
        label_filter="${label_filter}${separator}${1}"
    fi
}

label_filter="${KUBEVIRT_LABEL_FILTER:-}"

# skip certain tests on flake lane
add_to_label_filter "(!(SRIOV,Multus,Windows,GPU,VGPU))" "&&"

add_to_label_filter '(!QUARANTINE)' '&&'
add_to_label_filter '(!exclude-native-ssh)' '&&'
add_to_label_filter '(!no-flake-check)' '&&'
# check-tests-for-flake does not support Istio tests, remove this filtering once it does.
add_to_label_filter '(!Istio)' '&&'
rwofs_sc=$(jq -er .storageRWOFileSystem "${kubevirt_test_config}")
if [[ "${rwofs_sc}" == "local" ]]; then
    # local is a primitive non CSI storage class that doesn't support expansion
    add_to_label_filter "(!RequiresVolumeExpansion)" "&&"
fi

label_filter="(flake-check)||(${label_filter})"
ginkgo_params="$ginkgo_params -no-color -succinct --label-filter=${label_filter} -randomize-all"
if [[ -n ${NEW_TESTS} ]]; then
    readarray -t test_names <<<"$(jq -r '.[]' "${NEW_TESTS}")"
    for test_name in "${test_names[@]}"; do
        ginkgo_params+=" -focus='${test_name}'"
    done
fi

echo "Test lane: ${TEST_LANE}, preparing cluster up"

if [[ ! "$ginkgo_params" =~ -dry-run ]]; then
    make cluster-up
    make cluster-sync
else
    # Ginkgo only performs -dryRun in serial mode.
    export KUBEVIRT_E2E_PARALLEL="false"
    NUM_TESTS=1
fi

for i in $(seq 1 "$NUM_TESTS"); do
    echo "Test lane: ${TEST_LANE}, run: $i"
    if ! FUNC_TEST_ARGS="$ginkgo_params" make functest; then
        echo "Test lane: ${TEST_LANE}, run: $i, tests failed!"
        exit 1
    fi
done
