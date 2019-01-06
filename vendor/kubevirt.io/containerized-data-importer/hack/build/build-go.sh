#!/usr/bin/env bash

#Copyright 2018 The CDI Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -eo pipefail

script_dir="$(readlink -f $(dirname $0))"
source "${script_dir}"/common.sh
source "${script_dir}"/config.sh

mkdir -p ${BIN_DIR}
mkdir -p ${CMD_OUT_DIR}

if [ -z "$1" ]; then
    go_opt="build"
else
    go_opt=$1
    shift
fi

targets="$@"

if [ "${go_opt}" == "test" ]; then
    if [ -z "${targets}" ]; then
        targets="${CDI_PKGS}"
    fi
    for tgt in ${targets}; do
        (
            cd $tgt
            go test -v ./...
        )
    done
elif [ "${go_opt}" == "build" ]; then
    if [ -z "${targets}" ]; then
        targets="${BINARIES}"
    fi
    for tgt in ${targets}; do
        BIN_NAME=$(basename ${tgt})
        BIN_PATH=${tgt%/}
        outFile=${OUT_DIR}/${BIN_PATH}/${BIN_NAME}
        outLink=${BIN_DIR}/${BIN_NAME}
        rm -f ${outFile}
        rm -f ${outLink}
        (
            cd $tgt

            # Only build executables for linux amd64
            GOOS=linux GOARCH=amd64 go build -o ${outFile} -ldflags '-extldflags "static"'

            ln -sf ${outFile} ${outLink}
        )
    done
else # Pass go commands directly on to packages except vendor
    if [ -z ${targets} ]; then
        targets=$(allPkgs) # pkg/client is generated code, ignore it
    fi
    for tgt in ${targets}; do
        (
            cd $tgt
            go ${go_opt} ./...
        )
    done
fi
