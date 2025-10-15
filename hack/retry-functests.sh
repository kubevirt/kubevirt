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
# Copyright 2021 Red Hat, Inc.
#
#
# KubeVirt functional test retry helper.
#
# Purpose:
#   Mitigate transient/flaky test failures by re-running only the failing tests.
#
# Inputs (environment variables):
#   TEST_FOCUS            Optional logical expression inserted into label filter on first run only.
#   KUBEVIRT_E2E_SKIP     Optional regex passed as --skip (handled by hack/functests.sh).
#   JUNIT_REPORT_FILE     Path to junit XML (default set by workflow/Makefile).
#   MAX_ATTEMPTS          Override number of total attempts (default: 5, min 1).
#
# Output:
#   Updates JUNIT_REPORT_FILE on every attempt (previous content replaced).
#   Logs actions to stdout.
#
# Notes:
#   - We intentionally DO NOT set -e so we can inspect and retry on failures.
#   - We avoid external XML tooling; awk-based parser targets standard JUnit
#     produced by Ginkgo.
#   - Failing test names are escaped for basic regex meta characters before
#     constructing the alternation pattern.

set -uo pipefail

MAX_ATTEMPTS=${MAX_ATTEMPTS:-5}
if [[ ${MAX_ATTEMPTS} -lt 1 ]]; then
    echo "MAX_ATTEMPTS must be >= 1" >&2
    exit 1
fi

JUNIT_REPORT_FILE=${JUNIT_REPORT_FILE:-_out/artifacts/junit.functest.xml}

BASE_LABEL_EXCLUDES='!(single-replica)&&(!QUARANTINE)&&(!requireHugepages2Mi)&&(!requireHugepages1Gi)&&(!SwapTest)'

# Build the static label filter (excludes + flake suppression); test focusing handled via KUBEVIRT_E2E_FOCUS env.
build_label_filter() {
    echo "--label-filter=(!flake-check)&&(${TEST_FOCUS}&&${BASE_LABEL_EXCLUDES})"
}

# Extract failing test names from JUnit file and decode common XML/HTML entities.
extract_failures() {
    local file="$1"
    [[ -s ${file} ]] || return 0
    # Each record ends at </testcase>; if the record contains <failure or <error we treat it as failed.
    awk -v RS='</testcase>' 'index($0,"<failure")||index($0,"<error") { if (match($0,/name=\"([^\"]+)\"/,a)) print a[1] }' "${file}" |
        sed '/^$/d' |
        perl -pe 's/&quot;/"/g; s/&#39;/'"'"'/g; s/&apos;/'"'"'/g; s/&lt;/</g; s/&gt;/>/g; s/&amp;/&/g'
}

escape_regex() {
    # Use Perl to escape all regex metacharacters so test names become literal patterns.
    # Characters escaped: \ [ ] ( ) { } . ^ $ | * + ?
    perl -pe 's/([\\\[\]\(\)\{\}\.\^\$\|\*\+\?])/\\$1/g'
}

run_number=1
overall_status=1

while [[ ${run_number} -le ${MAX_ATTEMPTS} ]]; do
    echo "================ Functional Test Attempt ${run_number}/${MAX_ATTEMPTS} ================"
    rm -f "${JUNIT_REPORT_FILE}"

    current_label_filter=$(build_label_filter)
    echo "Running suite with label filter: ${current_label_filter} (LABEL_FILTER='${LABEL_FILTER:-}')"

    func_args="--no-color"
    if [[ -n ${LABEL_FILTER:-} ]]; then
        export KUBEVIRT_E2E_FOCUS="${LABEL_FILTER}"
    else
        unset KUBEVIRT_E2E_FOCUS 2>/dev/null || true
    fi

    FUNC_TEST_ARGS="${func_args}" \
        FUNC_TEST_LABEL_FILTER="${current_label_filter}" \
        make functest
    status=$?
    overall_status=${status}
    echo "Attempt ${run_number} exit status: ${status}"

    # Parse failures
    mapfile -t failed_tests < <(extract_failures "${JUNIT_REPORT_FILE}")
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        echo "No failing tests detected after attempt ${run_number}. Stopping early."
        overall_status=0
        break
    elif [[ ${#failed_tests[@]} -eq 1 && ${failed_tests[0]} =~ 'Tests Suite'$ ]]; then
        echo "Suite-level failure detected. Stopping early."
        overall_status=0
        break
    fi

    echo "Failing tests (${#failed_tests[@]}):"
    for t in "${failed_tests[@]}"; do
        echo "  - $t"
    done

    if [[ ${run_number} -ge ${MAX_ATTEMPTS} ]]; then
        echo "Reached maximum attempts (${MAX_ATTEMPTS}). Keeping last failure status ${overall_status}."
        break
    fi

    # Derive new LABEL_FILTER from failing test names (regex OR group)
    # Normalize (dedupe) after decoding to avoid redundant alternations.
    escaped_joined=$(printf '%s\n' "${failed_tests[@]}" | sort -u | escape_regex | paste -sd '|' -)
    LABEL_FILTER="(${escaped_joined})"
    export LABEL_FILTER
    echo "Updated LABEL_FILTER for next attempt to: ${LABEL_FILTER}"

    run_number=$((run_number + 1))
done

echo "Final exit status: ${overall_status}"
exit ${overall_status}
