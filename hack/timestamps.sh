#!/bin/bash

function timestamps::time_wrapper() {
    while IFS= read -r line; do
        printf "%(%T)T: %s\n" "-1" "$line"
    done
}

if [[ "${BASH_SOURCE[0]}" -ef "$0" ]]; then
    set -o pipefail
    /bin/sh "$@" 2>&1 | timestamps::time_wrapper
fi
