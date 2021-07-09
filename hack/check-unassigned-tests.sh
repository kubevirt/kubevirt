#!/bin/bash

main() {
    skip="SRIOV|GPU|\\[sig-operator\\]|\\[sig-network\\]|\\[sig-storage\\]|\\[sig-compute\\]"
    result=$(FUNC_TEST_ARGS="-dryRun -skip=${skip}" make functest)
    total_tests=$(echo "${result}" | grep "Ran[[:space:]].*of" | awk '{print $2}')
    if [ "${total_tests}" != "0" ]; then
        echo "Found ${total_tests} tests not asigned to any SIG, please check: ${result}"
        exit 1
    fi
}

main "${@}"
