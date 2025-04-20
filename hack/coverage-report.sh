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

set -e

# shellcheck disable=SC2046
bazel run //vendor/github.com/wadey/gocovmerge:gocovmerge -- $(cat | sed "s# # ${BUILD_WORKING_DIRECTORY}/#g" | sed "s#^#${BUILD_WORKING_DIRECTORY}/#") >coverprofile.dat
ARTIFACTS=${ARTIFACTS:-_out/artifacts}
mkdir -p ${ARTIFACTS}
if ! command -V covreport; then go install github.com/cancue/covreport@latest; fi
covreport -i coverprofile.dat -o "${ARTIFACTS}/coverage.html"
echo "coverage report written to ${ARTIFACTS}/coverage.html"
