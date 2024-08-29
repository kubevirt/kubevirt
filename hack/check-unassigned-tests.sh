#!/bin/bash

main() {
    skip="SRIOV|GPU|\\[sig-operator\\]|\\[sig-network\\]|\\[sig-storage\\]|\\[sig-compute\\]|\\[sig-performance\\]|\\[sig-compute-realtime\\]|\\[sig-monitoring\\]"
    result=$(FUNC_TEST_ARGS="--dry-run --no-color -skip=${skip}" make functest)
    total_tests=$(echo "${result}" | sed -n "s/.*Ran \([0-9]\+\) of .* Specs in .* seconds.*/\1/p")
    if [ "${total_tests}" != "0" ]; then
        echo "Found ${total_tests} tests not assigned to any SIG, please check: ${result}"
        exit 1
    fi
}

main "${@}"
