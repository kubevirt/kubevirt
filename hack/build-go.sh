#!/bin/bash

source hack/config.sh

if [ -z "$1" ]; then
    target="build"
else
    target=$1
    shift
fi

if [ $# -eq 0 ]; then
    args=$binaries
else
    args=$@
fi

for arg in $args; do
    if [ "${target}" = "test" ]; then
        (cd $arg; go ${target} -v ./...)
    elif [ "${target}" = "build" ]; then
        (cd $arg; go ${target} .)
    else
        (cd $arg; go $target ./...)
    fi
done
