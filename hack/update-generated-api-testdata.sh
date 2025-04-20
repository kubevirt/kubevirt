#!/usr/bin/env bash
#
# Copyright The KubeVirt Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# imitations under the License.

set -e

rm -f staging/src/kubevirt.io/api/apitesting/testdata/HEAD/*.{yaml,json}

# UPDATE_COMPATIBILITY_FIXTURE_DATA=true regenerates fixture data if needed.
# -run //HEAD only runs the test cases comparing against testdata for HEAD.

hack/dockerized "UPDATE_COMPATIBILITY_FIXTURE_DATA=true go test -mod=readonly kubevirt.io/api/apitesting -run //HEAD >/dev/null 2>&1 || true && go test -mod=readonly kubevirt.io/api/apitesting -run //HEAD -count=1"
