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

source $(dirname "$0")/../common.sh

fail_if_cri_bin_missing
readonly PROM_IMAGE="quay.io/prometheus/prometheus:v2.44.0"

function cleanup() {
    local cleanup_files=("${@:?}")

    for file in "${cleanup_files[@]}"; do
        rm -f "$file"
    done
}

function lint() {
    local target_file="${1:?}"

    ${KUBEVIRT_CRI} run --rm --entrypoint=/bin/promtool \
        -v "$target_file":/tmp/rules.verify:ro,Z "$PROM_IMAGE" \
        check rules /tmp/rules.verify
}

function unit_test() {
    local target_file="${1:?}"
    local tests_file="${2:?}"

    ${KUBEVIRT_CRI} run --rm --entrypoint=/bin/promtool \
        -v "$tests_file":/tmp/rules.test:ro,Z \
        -v "$target_file":/tmp/rules.verify:ro,Z \
        "$PROM_IMAGE" \
        test rules /tmp/rules.test
}

function main() {
    local prom_spec_dumper="${1:?}"
    local tests_file="${2:?}"
    local target_file

    target_file="$(mktemp --tmpdir -u tmp.prom_rules.XXXXX)"
    trap "cleanup $target_file" RETURN EXIT INT

    "$prom_spec_dumper" "$target_file"

    echo "INFO: Rules file content:"
    cat "$target_file"
    echo

    lint "$target_file"
    unit_test "$target_file" "$tests_file"
}

main "$@"
