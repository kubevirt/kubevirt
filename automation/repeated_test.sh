#!/bin/bash

SCRIPT_PATH="$(
    cd "$(dirname "${BASH_SOURCE[0]}")/" || exit
    echo "$(pwd)/"
)"

NUM_TESTS=${NUM_TESTS:-3}
if [[ -z ${NEW_TESTS} ]]; then
    NEW_TESTS=$("${SCRIPT_PATH}"/new_tests.sh)
    # skip certain tests for now, as we don't have a strategy currently
    NEW_TESTS=$(echo "$NEW_TESTS" | sed -E 's/\|?[^\|]*(sriov|multus|windows|genie|gpu)[^\|]*//g')
fi
if [[ -z "${NEW_TESTS}" ]]; then
    echo "Nothing to test"
    exit 0
fi

TEST_LANES=( 'k8s-1.14.6' 'k8s-1.15.1' 'k8s-1.16.2' )

trap '{ make cluster-down; }' EXIT SIGINT SIGTERM

for lane in "${TEST_LANES[@]}"; do

    echo "test lane: $lane, preparing cluster up"

    export KUBEVIRT_PROVIDER="$lane"
    export KUBEVIRT_NUM_NODES=2
    make cluster-up

    for i in $(seq 1 "$NUM_TESTS"); do
        echo "test lane: $lane, run: $i"
        make cluster-sync
        ginko_params="--ginkgo.noColor --ginkgo.focus=${NEW_TESTS} --ginkgo.regexScansFilePath=true"
        FUNC_TEST_ARGS="$ginko_params" make functest
        if [[ $? -ne 0 ]]; then
            echo "test lane: $lane, run: $i, tests failed!"
            exit 1
        fi
    done

    make cluster-down

done

exit 0
