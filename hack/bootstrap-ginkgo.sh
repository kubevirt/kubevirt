#!/usr/bin/env bash

set -e

source $(dirname "$0")/common.sh

FOLDERS="${KUBEVIRT_DIR}/cmd/ ${KUBEVIRT_DIR}/pkg/ ${KUBEVIRT_DIR}/staging/src/kubevirt.io/ ${KUBEVIRT_DIR}/tests/framework/"

ginkgobin=$(realpath _out/tests/ginkgo)
# Find every folder containing tests
for dir in $(find ${FOLDERS} -type f -name '*_test.go' -printf '%h\n' | sort -u); do
    # If there is no file ending with _suite_test.go, bootstrap ginkgo
    SUITE_FILE=$(find $dir -maxdepth 1 -type f -name '*_suite_test.go')
    if [ -z "$SUITE_FILE" ]; then
        echo "Missing test suite entrypoint; attempt to create one automatically"
        (cd $dir && $ginkgobin bootstrap)
    fi
done
