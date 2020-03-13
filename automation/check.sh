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
# Copyright 2020 Red Hat, Inc.
#

set -ex

make generate
if [[ -n "$(git status --porcelain)" ]]; then
    echo "It seems like you need to run
  `make generate`. Please run it and commit the changes";
  git status --porcelain; false;
fi
if diff <(git grep -c '') <(git grep -cI '') | egrep -v -e 'docs/.*\.png|swagger-ui' -e 'vendor/*' -e 'assets/*' | grep '^<'; then
  echo "Binary files are present in git repostory."; false;
fi
make
if [[ -n "$(git status --porcelain)" ]] ; then
    echo "It seems like you need to run
  `make`. Please run it and commit the changes"; git status --porcelain; false;
fi
make build-verify # verify that we set version on the packages built by bazel
# TODO: make goverall
make bazel-test;
make build-verify # verify that we set version on the packages built by go(goveralls depends on go-build target)
make apidocs
make client-python
# TODO: Use tag if present instead of latest
make manifests DOCKER_PREFIX="docker.io/kubevirt" DOCKER_TAG=latest # skip getting old CSVs here (no QUAY_REPOSITORY), verification might fail because of stricter rules over time; falls back to latest if not on a tag
make olm-verify
make prom-rules-verify;


