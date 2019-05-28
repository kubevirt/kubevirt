#!/bin/bash -e

function _kubectl() {
    export KUBECONFIG=./cluster/.kubeconfig
    ./cluster/.kubectl "$@"
}

_kubectl "$@"
