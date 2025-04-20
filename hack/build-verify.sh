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

function report_dirty_build() {
    set +e
    echo "Build is not clean:"
    hack/virtctl.sh version
    git status
    exit 1
}

# Check that "clean" is reported at least once
if [ -z "$(hack/virtctl.sh version | grep clean)" ]; then
    report_dirty_build
fi

# Check that "dirty" is never reported
if [ -n "$(hack/virtctl.sh version | grep dirty)" ]; then
    report_dirty_build
fi
