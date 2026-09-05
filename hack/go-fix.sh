#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#

set -e

paths=""
while IFS= read -r line; do
    # read directory from the file and append a wildcard
    [[ -n "${line}" ]] && paths+="./${line}/... "
done <hack/linter/go-fix-paths.txt

# Nothing opted in yet: go fix with no packages would default to '.', so bail out.
if [[ -z "${paths}" ]]; then
    exit 0
fi

if [[ "${1:-}" == "--diff" ]]; then
    # go fix -diff exits 0 even when fixes are pending, so fail on non-empty
    # output instead of the exit code. '|| true' keeps a conflict-skip exit
    # (which does return non-zero) from aborting the capture under 'set -e'.
    diff="$(go fix -diff ${paths} || true)"
    if [[ -n "${diff}" ]]; then
        echo "${diff}"
        echo "go fix found unapplied fixes; run 'make gofix' and commit the result." >&2
        exit 1
    fi
else
    # A single 'go fix' pass applies only non-overlapping fixes and then
    # exits non-zero asking to "Re-run the command to apply more". Loop until
    # no fixes remain (the -diff gate is empty) so one 'make gofix' converges
    # instead of leaving pending fixes behind. '|| true' keeps that partial
    # exit from aborting under 'set -e'.
    for _ in $(seq 1 5); do
        go fix ${paths} || true
        [[ -z "$(go fix -diff ${paths} || true)" ]] && break
    done
fi
