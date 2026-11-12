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
# * almost all flaky tests were independent of the execution platform,
#   so environmental issues should be considered with higher priority
# * devs may be unaware of many flakes due to order-dependent tests
# * simple techniques of reversing or shuffling the test order may be more
#   efficient and effective than more sophisticated approaches
# * "One study found that 88% of flaky tests were found to consecutively fail
#    up to a maximum of five times before passing, though another reported
#    finding new flaky tests even after 10,000 test suite runs."
#
# Therefore we by default
# * use only latest kubevirtci provider,
# * run all changed tests five times and
# * randomize the test order each time

KUBEVIRT_ROOT=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")

# include defaults for retrieving proper vendored cluster-up version
export IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"
export DOCKER_TAG_ALT=''
export IMAGE_PREFIX=''
export IMAGE_PREFIX_ALT=''
source hack/config-default.sh

GINKGO_TESTS_IMAGE="${GINKGO_TESTS_IMAGE-quay.io/kubevirtci/ginkgo-tests:v202611101645-0f94693}"

export TIMESTAMP=${TIMESTAMP:-1}

function usage() {
    cat <<EOF
usage: [NUM_TESTS=x] \
       [NEW_TESTS=file_name.json] \
       [TARGET_COMMIT_RANGE=a1b2c3d4..e5f60718] $0 [TEST_LANE] [--dry-run]

    run set of tests repeatedly using a json file containing the names of the tests that have been
    changed or added since last merge commit
    hint: set NEW_TESTS to explicitly name the json file containing test names to run

    options:
        GINKGO_TESTS_IMAGE        the container image to use to execute cannier
                             command
        NUM_TESTS            how many times the test lane is run, default is 5
        INVOKE_FUNCTEST_ONCE if set to 'true', instead of invoking functests
                             binary NUM_TESTS amount times, invoke it only once
                             and use the Ginkgo flag '--repeat' instead to run
                             the test NUM_TESTS times
        NEW_TESTS            the json file containing the (textual) names of the
                             tests to run, according to what is shown in the
                             junit.xml file
        TARGET_COMMIT_RANGE  the commit id to use when fetching the changed
                             test files
                             note: leaving TARGET_COMMIT_RANGE empty only works
                             if on a git branch different from main.
                             If /clonerefs is at work you need to provide a
                             target commit, as then the latest commit is a
                             merge commit (resulting in no changes detected)
        TEST_LANE            the kubevirtci provider to use, if not given,
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
        target_commit_range=('-r' "${target_commit_range}")
    fi

    tmp_dir="$(mktemp -d)"
    # ignore stderr during execution caused by ginkgo compile
    podman run --rm \
        -v "${KUBEVIRT_ROOT}:/kubevirt/:z" \
        -v "${tmp_dir}:/tmp:z" \
        "${GINKGO_TESTS_IMAGE}" \
        changed "${target_commit_range[@]}" \
        -p /kubevirt \
        -t tests/ \
        -o /tmp/changed-tests.json \
        -l /tmp/changed-tests-labels.json > /dev/null 2>&1

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

function detect_environment_profile() {
    local labels_file="$1"

    if [[ ! -f "$labels_file" ]] || [[ ! -s "$labels_file" ]]; then
        echo "base"
        return
    fi

    local all_labels
    all_labels=$(jq -r '[.[].labels[]] | unique | .[]' "$labels_file" 2>/dev/null || true)

    if [[ -z "$all_labels" ]]; then
        echo "base"
        return
    fi

    local has_network=false
    local has_gpu=false
    local has_dra_cpu=false

    while IFS= read -r label; do
        case "$label" in
            SRIOV|Multus|DRA-Network|in-place-hotplug-NICs|Istio)
                has_network=true ;;
            GPU|VGPU|DRA-GPU)
                has_gpu=true ;;
            DRA-CPU)
                has_dra_cpu=true ;;
        esac
    done <<< "$all_labels"

    if $has_network; then
        echo "network"
    elif $has_gpu; then
        echo "gpu"
    elif $has_dra_cpu; then
        echo "dra-cpu"
    else
        echo "base"
    fi
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
    TEST_LANE="$(<./kubevirtci/stable_provider.txt)"
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
    echo "TARGET_COMMIT_RANGE: $TARGET_COMMIT_RANGE"
    NEW_TESTS=$(new_tests "$TARGET_COMMIT_RANGE")
    echo "NEW_TESTS: $NEW_TESTS"
fi

if [[ -z "${NEW_TESTS}" ]]; then
    echo "Nothing to test"
    exit 0
fi

LABELS_FILE="${NEW_TESTS%.json}-labels.json"
ENVIRONMENT_PROFILE=$(detect_environment_profile "${LABELS_FILE}")
echo "Environment profile: ${ENVIRONMENT_PROFILE}"

tests_changed=$(jq '. | length' "${NEW_TESTS}")
echo "Tests changed: ${tests_changed}"
if ((tests_changed == 0)); then
    echo "Nothing to test"
    exit 0
fi

NUM_TESTS=${NUM_TESTS-5}
echo "Number of per lane runs: $NUM_TESTS"

if [[ "${JOB_NAME-}" == *dequarantine* ]]; then
    echo "Dequarantine lane: skipping repetition count check"
elif should_skip_test_run_due_to_too_many_tests "${NEW_TESTS}"; then
    echo "Skipping run due to number of tests in total being too high for repeated run."
    exit 0
fi

# Tests triggered by a Prow batch should stop after the first failure to save CI resources
if [[ -n ${PROW_JOB_ID:-} && ${JOB_TYPE} == "batch" ]]; then
  ginkgo_params="$ginkgo_params --fail-fast"
fi

if [[ -f "${LABELS_FILE}" ]] && jq -e '[.[].labels[] | select(. == "requires-two-worker-nodes-with-cpu-manager" or . == "requires-three-schedulable-nodes")] | length > 0' "${LABELS_FILE}" > /dev/null 2>&1; then
    KUBEVIRT_NUM_NODES=3
else
    KUBEVIRT_NUM_NODES=2
fi

trap '{ ret=$?; make cluster-down || true; exit $ret; }' EXIT SIGINT SIGTERM

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
export KUBEVIRT_DEPLOY_NETWORK_RESOURCES_INJECTOR=true
export KUBEVIRT_FLANNEL=true

export KUBEVIRT_NFS_DIR=${KUBEVIRT_NFS_DIR:-/var/lib/containers/nfs-data}

export KUBEVIRT_PROVIDER="${TEST_LANE}"

detect_centos_stream_version

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
        label_filter="${label}"
    else
        label_filter="${label_filter}${separator}${label}"
    fi
}

add_feature_gate() {
  if [ -z "${FEATURE_GATES:-}" ]; then
    export FEATURE_GATES="$1"
  else
    export FEATURE_GATES="$FEATURE_GATES,$1"
  fi
}

add_feature_gate "Plugins"
add_feature_gate "OCIExport"

case "$ENVIRONMENT_PROFILE" in
  network)
    export KUBEVIRT_DEPLOY_ISTIO=true
    export KUBEVIRT_SECONDARY_IFACES_TO_IP=eth1
    export KUBEVIRT_DEPLOY_NET_DRA_DRIVER=true
    export KUBEVIRT_WITH_SRIOV=true
    export KUBEVIRT_FUNC_TEST_SUITE_ARGS="${KUBEVIRT_FUNC_TEST_SUITE_ARGS:-} -emulated-sriov=true"
    ;;
  gpu)
    export KUBEVIRT_USE_FAKE_VFIO=true
    export DRA_DRIVER_PROFILE=vfio-gpu
    export FAKE_PCI_DEVICES=8
    export DRA_DRIVER_NAME="vfio-gpu.example.com"
    add_feature_gate "GPUsWithDRA"
    ;;
  dra-cpu)
    export KUBEVIRT_USE_DRA_CPU=true
    export KUBEVIRT_NUM_NUMA_NODES=2
    export KUBEVIRT_NUM_VCPU=12
    add_feature_gate "CPUsWithDRA"
    ;;
esac

label_filter="${KUBEVIRT_LABEL_FILTER:-}"

case "$ENVIRONMENT_PROFILE" in
  network)
    add_to_label_filter "(!(Windows,GPU,VGPU,DRA-GPU,DRA-CPU))" "&&"
    ;;
  gpu)
    add_to_label_filter "(!(SRIOV,Multus,Windows,in-place-hotplug-NICs,DRA-Network,DRA-CPU))" "&&"
    add_to_label_filter '(!Istio)' '&&'
    ;;
  dra-cpu)
    add_to_label_filter "(!(SRIOV,Multus,Windows,GPU,VGPU,in-place-hotplug-NICs,DRA-GPU,DRA-Network))" "&&"
    add_to_label_filter '(!Istio)' '&&'
    ;;
  *)
    add_to_label_filter "(!(SRIOV,Multus,Windows,GPU,VGPU,in-place-hotplug-NICs,DRA-GPU,DRA-Network,DRA-CPU))" "&&"
    add_to_label_filter '(!Istio)' '&&'
    ;;
esac

add_to_label_filter '(!QUARANTINE)' '&&'
add_to_label_filter '(!no-flake-check)' '&&'
add_to_label_filter '(!requireHugepages1Gi)' '&&'
add_to_label_filter '(!USB)' '&&'
add_to_label_filter '(!requires-arm64)' '&&'
add_to_label_filter '(!requires-s390x)' '&&'
add_to_label_filter '(!requires-cross-arch-emulation)' '&&'
add_to_label_filter '(!requires-arm64-secure-boot)' '&&'
add_to_label_filter '(!RequiresPersistentReservation)' '&&'
add_to_label_filter '(!RequiresVSOCKLocalNamespace)' '&&'

if [[ ${KUBEVIRT_NUM_NODES:-} = "1" && ${KUBEVIRT_INFRA_REPLICAS:-} = "1" ]]; then
  add_to_label_filter '!(multi-replica)' '&&'
else
  add_to_label_filter '!(single-replica)' '&&'
fi

if ! kubectl get clusterversion version &>/dev/null; then
  add_to_label_filter '(!OpenShift)' '&&'
fi

label_filter="(flake-check)||(${label_filter})"
ginkgo_params="$ginkgo_params -no-color -succinct --label-filter=${label_filter} -randomize-all"
if [[ -n ${NEW_TESTS} ]]; then
    readarray -t test_names <<<"$(jq -r '.[]' "${NEW_TESTS}")"
    for test_name in "${test_names[@]}"; do
        echo "test name: ${test_name}"
        # escape steps:
        # 1) whitespaces - this avoids skipping tests
        # 2) square brackets - which make the regex bail
        escaped=$(echo "${test_name// /\\s}" | sed -E 's/([][])/\\\1/g')
        echo "test name escaped: ${escaped}"
        ginkgo_params+=" -focus=${escaped}"
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

INVOKE_FUNCTEST_ONCE="${INVOKE_FUNCTEST_ONCE-}"
if [[ "$INVOKE_FUNCTEST_ONCE" == 'true' ]]; then
    echo "Test lane: ${TEST_LANE}"
    if ! FUNC_TEST_ARGS="--repeat=$(( NUM_TESTS - 1 )) $ginkgo_params" make functest; then
        echo "Test lane: ${TEST_LANE}, tests failed!"
        exit 1
    fi
else
    for i in $(seq 1 "$NUM_TESTS"); do
        echo "Test lane: ${TEST_LANE}, run: $i"
        if ! FUNC_TEST_ARGS="$ginkgo_params" make functest; then
            echo "Test lane: ${TEST_LANE}, run: $i, tests failed!"
            exit 1
        fi
    done
fi
