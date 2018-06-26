#!/bin/bash

set -e

function report_dirty_build() {
    set +e
    echo "Build is not clean:"
    cluster/virtctl.sh version
    git status
    exit 1
}

# Check that "clean" is reported at least once
if [ -z "$(cluster/virtctl.sh version | grep clean)" ]; then
    report_dirty_build
fi

# Check that "dirty" is never reported
if [ -n "$(cluster/virtctl.sh version | grep dirty)" ]; then
    report_dirty_build
fi
