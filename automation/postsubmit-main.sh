#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#


set -exuo pipefail

GIT_ASKPASS="$(pwd)/automation/git-askpass.sh"
[ -f "$GIT_ASKPASS" ] || exit 1
export GIT_ASKPASS

export DOCKER_TAG=""

make bazel-build-verify

make build-verify # verify that we set version on the packages built by go (goveralls depends on go-build target)
make apidocs
make client-python
make manifests DOCKER_PREFIX="$DOCKER_PREFIX" DOCKER_TAG="$DOCKER_TAG" # skip getting old CSVs here (no QUAY_REPOSITORY), verification might fail because of stricter rules over time; falls back to latest if not on a tag
make olm-verify
make prom-rules-verify

make manifests
make build-functests

bash hack/gen-swagger-doc/deploy.sh
bash hack/gen-client-python/deploy.sh
hack/publish-staging.sh
