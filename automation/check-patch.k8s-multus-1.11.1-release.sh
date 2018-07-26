#!/bin/bash -e
# Patch/PR testing wrapper that sets the corresponding TARGET
#

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    TARGET="$0"
    TARGET="${TARGET#./}"
    TARGET="${TARGET%.*}"
    TARGET="${TARGET#*.}"
    echo "TARGET=$TARGET"
    export TARGET
    exec automation/test.sh
fi
