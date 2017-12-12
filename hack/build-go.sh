#!/bin/bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2017 Red Hat, Inc.
#

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
        (
            cd pkg
            go ${target} -v ./...
        )
    elif [ "${target}" = "functest" ]; then
        (cd tests; go test -kubeconfig=../cluster/vagrant/.kubeconfig -timeout 30m ${FUNC_TEST_ARGS})
        exit
    else
        (
            cd pkg
            go $target ./...
        )
        (
            cd tests
            go $target ./...
        )
    fi
fi

# handle binaries
for arg in $args; do
    if [ "${target}" = "test" ]; then
        (
            cd $arg
            go ${target} -v ./...
        )
    elif [ "${target}" = "install" ]; then
        eval "$(go env)"
        ARCHBIN=$(basename $arg)-$(git describe --always)-$GOHOSTOS-$GOHOSTARCH
        ALIASLNK=$(basename $arg)
        rm $arg/$ALIASLNK $arg/$(basename $arg)-*-$GOHOSTOS-$GOHOSTARCH || :
        (
            cd $arg
            GOBIN=$PWD go build -o $ARCHBIN
        )
        mkdir -p bin
        ln -sf $ARCHBIN $arg/$ALIASLNK
        ln -sf ../$arg/$ARCHBIN bin/$ALIASLNK
    else
        (
            cd $arg
            go $target ./...
        )
    fi
done
