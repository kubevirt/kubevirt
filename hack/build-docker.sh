#!/bin/bash

source hack/config.sh

if [ -z "$1" ]; then
    target="build"
else
    target=$1
shift
fi
shift

if [ $# -eq 0 ]; then
    args=$binaries
else
    args=$@
fi

for arg in $args; do
    if [ "${target}" = "build" ]; then
        (cd $arg; docker $target -t ${docker_prefix}/$(basename $arg):${docker_tag} .)
    elif [ "${target}" = "push" ]; then
        (cd $arg; docker $target ${docker_prefix}/$(basename $arg):${docker_tag})
    fi
done
