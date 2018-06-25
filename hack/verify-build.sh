#!/bin/bash

set -e

if [ -n "$(cluster/virtctl.sh version | grep dirty)" ]; then
    echo "Build is not clean:"
    git status
    exit 1
fi
