#!/bin/bash
set -e

source hack/config.sh

if [ -z "$1" ]; then
    target="install"
else
    target=$1
    shift
fi

if [ $# -eq 0 ]; then
    args=$binaries
else
    args=$@
fi

# forward all commands to all packages if no specific one was requested
# TODO finetune this a little bit more
if [ $# -eq 0 ]; then
    if [ "${target}" = "test" ]; then
        (cd pkg; go ${target} -v ./...)
    else
        (cd pkg; go $target ./...)
    fi
fi

# handle binaries
for arg in $args; do
    if [ "${target}" = "test" ]; then
        (cd $arg; go ${target} -v ./...)
    elif [ "${target}" = "install" ]; then
        (cd $arg; GOBIN=$PWD go ${target} .)
        mkdir -p bin
        ln -sf ../$arg/$(basename $arg) bin/$(basename $arg)
    else
        (cd $arg; go $target ./...)
    fi
done

