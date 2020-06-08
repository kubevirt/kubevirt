#!/bin/bash -e
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
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
# Copyright 2020 Red Hat, Inc.
#

last_running_func=""
trap '{ [[ $last_running_func != "" ]] && echo "Last running function was:" $last_running_func; }' EXIT ERR

# For development you can change this list, but to revert it before merging.
declare -a functions=( 
    test_generate 
    test_binaries
    test_make 
    test_goveralls_bazel_test 
    test_apidocs 
    test_client_python 
    test_manifests 
    test_olm_verify 
    test_prom_rules_verify    
)

function test_generate {
    make generate
    if [[ -n "$(git status --porcelain)" ]] ; then
        echo "It seems like you need to run 'make generate'. Please run it and commit the changes"
        git status --porcelain; false
    fi
}

function test_binaries {
    if diff <(git grep -c '') <(git grep -cI '') | egrep -v -e 'docs/.*\.png|swagger-ui' -e 'vendor/*' -e 'assets/*' | grep '^<'; then
        echo "Binary files are present in git repostory."; false
    fi
}

function test_make {
    make
    if [[ -n "$(git status --porcelain)" ]] ; then
        echo "It seems like you need to run 'make'. Please run it and commit the changes"; git status --porcelain; false
    fi
    
    make build-verify # verify that we set version on the packages built by bazel
}

function test_goveralls_bazel_test {
    if [[ $TRAVIS_REPO_SLUG == "kubevirt/kubevirt" && $TRAVIS_CPU_ARCH == "amd64" ]]; then
        echo "Running goveralls"
        make goveralls
    else
        echo "Running bazel-test"
        make bazel-test
    fi

    make build-verify # verify that we set version on the packages built by go (goveralls depends on go-build target)
}

function test_apidocs {
    make apidocs
}

function test_client_python {
    make client-python
}

function test_manifests {
    make manifests DOCKER_PREFIX="docker.io/kubevirt" DOCKER_TAG=$TRAVIS_TAG # skip getting old CSVs here (no QUAY_REPOSITORY), verification might fail because of stricter rules over time; falls back to latest if not on a tag
}

function test_olm_verify {
    make olm-verify
}

function test_prom_rules_verify {
    if [[ $TRAVIS_CPU_ARCH == "amd64" ]]; then
        make prom-rules-verify
    fi
}

function main() {
    FUNC_LIST="${functions[*]// /|}"
    if [ $# -ne 0 ]; then
        echo "Overriding function list from arguments"
        FUNC_LIST="$@"
    fi

    echo "Running functions:" $FUNC_LIST

    for f in ${functions[@]}; do
        last_running_func=$(echo $f)
        echo "*** Starting $f ***"
        $f
        echo "*** Ended $f ***"
        last_running_func=""
    done
}

main "$@"
