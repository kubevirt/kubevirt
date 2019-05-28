#!/bin/bash -e

SCRIPT_ROOT=$(dirname "${BASH_SOURCE}")/..

source ${SCRIPT_ROOT}/cluster/gocli.sh

if [[ -t 1 ]]; then
    $gocli_interactive "$@"
else
    $gocli "$@"
fi
