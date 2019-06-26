#!/bin/bash -xe

main() {
    TARGET="$0"
    TARGET="${TARGET#./}"
    TARGET="${TARGET%.*}"
    TARGET="${TARGET#*.}"
    echo "TARGET=$TARGET"
    export TARGET

    echo "Run functional tests"
    exec automation/test.sh
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
