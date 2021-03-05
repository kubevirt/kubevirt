#!/bin/bash

main(){
    result=$(FUNC_TEST_ARGS='-dryRun -focus=QUARANTINE' make functest)
    quarantined_tests=$(echo "$result" | grep '^Ran.*of.*Specs' | awk '{print $2}')
    if [ "${quarantined_tests}" != "0" ]; then
        automation/test.sh
    fi
}

main "$@"
