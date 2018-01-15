#!/bin/bash

set -e

source $(dirname "$0")/common.sh

# Find every folder containing tests
for dir in $(find ${KUBEVIRT_DIR}/pkg/ -type f -name '*_test.go' -printf '%h\n' | sort -u); do
    # If there is no file ending with _suite_test.go, bootstrap ginkgo
    SUITE_FILE=$(find $dir -maxdepth 1 -type f -name '*_suite_test.go')
    if [ -z "$SUITE_FILE" ]; then
        (cd $dir && ginkgo bootstrap || :)
    fi
done
