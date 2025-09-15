#!/usr/bin/env bash
# Rerun KubeVirt Conformance test failures individually
# 1. Runs automation/test.sh once with KUBEVIRT_LABEL_FILTER=Conformance (respecting existing TARGET)
# 2. Parses _out/artifacts/junit.functest.xml for failed test cases
# 3. Re-runs each failed test one-by-one using a lightweight direct ginkgo invocation (bypassing test.sh)
#
# Usage:
#   TARGET=kind-1.29 hack/rerun-conformance-failures.sh
#   TARGET=external KUBECONFIG=/abs/path/kubeconfig hack/rerun-conformance-failures.sh
#
# Optional env vars:
#   TARGET                Provider target (required if not already set)
#   SKIP_BUILD_IMAGES=1   If you patched automation/test.sh to respect this to skip image build
#   EXTRA_TEST_ARGS       Extra args passed to automation/test.sh initial invocation
#   LIGHT_NODES           Parallel nodes for light reruns (default 1)
#   LIGHT_TIMEOUT         Timeout per ginkgo light rerun (default 30m)
#   LIGHT_LABEL_FILTER    Label filter for reruns (default Conformance)
#   LIGHT_EXTRA           Extra args appended to lightweight ginkgo command
#   REUSE_INITIAL_RUN=1   Skip initial full run (assumes junit already present)
#   SKIP_INITIAL_RUN=1    Alias for REUSE_INITIAL_RUN (preferred concise name)
#   MAX_FAILS             Limit number of failures to rerun (default: all)
#   JUNIT_PATH            Explicit path to existing junit file (overrides default)
#
# Requirements:
#   - jq, xmlstarlet or xmllint? (We implement parsing with awk/sed only; no extra deps)
#
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
if [[ -n "${JUNIT_PATH:-}" ]]; then
    JUNIT_FILE="${JUNIT_PATH}"
else
    JUNIT_FILE="${REPO_ROOT}/_out/artifacts/junit.functest.xml"
fi
MERGED_JUNIT_FILE="${REPO_ROOT}/_out/artifacts/junit.functest.merged.xml"
: "${TARGET:?TARGET environment variable must be set (e.g. kind-1.29, external)}"

# Default container registry settings
DOCKER_PREFIX=${DOCKER_PREFIX:-"quay.io/kubevirt"}
DOCKER_TAG=${DOCKER_TAG:-"latest"}

# Lightweight rerun defaults
LIGHT_NODES=${LIGHT_NODES:-1}
LIGHT_TIMEOUT=${LIGHT_TIMEOUT:-30m}
LIGHT_LABEL_FILTER=${LIGHT_LABEL_FILTER:-Conformance}
LIGHT_EXTRA=${LIGHT_EXTRA:-}

# Normalize skip flag (SKIP_INITIAL_RUN takes precedence if set)
if [[ "${SKIP_INITIAL_RUN:-}" == "1" ]]; then
    REUSE_INITIAL_RUN=1
fi

# Run full conformance suite unless skipping
if [[ "${REUSE_INITIAL_RUN:-}" != "1" ]]; then
    echo "[INFO] Running initial Conformance suite via automation/test.sh" >&2
    GINKGO_SKIP="DataVolume|Filesystem|CDI|test_id:1783|test_id:4118|test_id:1514|test_id:1513|test_id:1780|test_id:6479"
    should remain to able resolve the VM IP | Should contain condition when migrating with quota that doesn't have resources for both source and target|test_id:8607|should automatically cancel unschedulable migration after a timeout period|test_id:1853|test_id:6970|test_id:3242|without a specific port number|test_id:6969|with explicit ports used by live migration
    ---
    Should contain condition when migrating with quota that doesn't have resources for both source and target
    should automatically cancel unschedulable migration after a timeout period
    ---
    DataVolume | Filesystem | CDI | test_id:1783 | test_id:4118 | test_id:1514 | test_id:1513 | test_id:1780 | test_id:6479 | should remain to able resolve the VM IP | test_id:8607 | test_id:1853 | test_id:6970 | test_id:3242 | without a specific port number | test_id:6969 | with explicit ports used by live migration

    FUNC_TEST_ARGS=--no-color FUNC_TEST_LABEL_FILTER="--label-filter=(!flake-check)&&(Conformance&&!(single-replica)&&(!QUARANTINE)&&(!requireHugepages2Mi)&&(!requireHugepages1Gi)&&(!SwapTest))" make functest

    (cd "${REPO_ROOT}" &&
        KUBEVIRT_LABEL_FILTER=Conformance \
            KUBEVIRT_E2E_SKIP="${GINKGO_SKIP}" \
            DOCKER_PREFIX="${DOCKER_PREFIX}" \
            DOCKER_TAG="${DOCKER_TAG}" \
            TARGET="${TARGET}" \
            ./automation/test.sh ${EXTRA_TEST_ARGS:-}) || true
else
    echo "[INFO] Skipping initial run (REUSE_INITIAL_RUN=1 or SKIP_INITIAL_RUN=1)" >&2
fi

if [[ "${REUSE_INITIAL_RUN:-}" == "1" && ! -f "${JUNIT_FILE}" ]]; then
    echo "[ERROR] Requested to skip initial run but JUnit file missing: ${JUNIT_FILE}" >&2
    echo "[DEBUG] PWD=$(pwd)" >&2
    echo "[DEBUG] Listing artifacts directory:" >&2
    ls -l "${REPO_ROOT}/_out/artifacts" >&2 || echo "[DEBUG] Could not list artifacts dir" >&2
    exit 3
fi

if [[ ! -f "${JUNIT_FILE}" ]]; then
    echo "[ERROR] JUnit file not found at ${JUNIT_FILE}" >&2
    echo "[DEBUG] If the file exists under a different name use JUNIT_PATH=/abs/path/to/file" >&2
    ls -l "${REPO_ROOT}/_out/artifacts" >&2 || true
    exit 2
fi

# Extract failed test case names from junit. A failure is indicated by a <failure> child element.
# We capture the name attribute of the parent <testcase>.
FAILED_LIST=$(awk '
    /<testcase / {
        name=""; hasfail=0;
        if (match($0, /name="([^"]+)"/, m)) { name=m[1] }
    }
    /<failure/ { hasfail=1 }
    /<\/testcase>/ { if (hasfail && name!="") print name }
' "${JUNIT_FILE}" | sort -u)

if [[ -z "${FAILED_LIST}" ]]; then
    echo "[INFO] No failed tests detected in ${JUNIT_FILE}." >&2
    exit 0
fi

# Optional cap on number of failures
if [[ -n "${MAX_FAILS:-}" ]]; then
    FAILED_LIST=$(echo "${FAILED_LIST}" | head -n "${MAX_FAILS}")
fi

echo "[INFO] Failed tests to rerun:" >&2
echo "${FAILED_LIST}" >&2

RERUN_REPORT_DIR="${REPO_ROOT}/_out/artifacts/reruns"
mkdir -p "${RERUN_REPORT_DIR}"
echo "[INFO] Will merge rerun results into cumulative file: ${MERGED_JUNIT_FILE}" >&2
if [[ ! -f "${MERGED_JUNIT_FILE}" ]]; then
    cp "${JUNIT_FILE}" "${MERGED_JUNIT_FILE}"
fi

PASSED=0
FAILED=0

# Temporarily disable exit on error for the loop
set +e
while IFS= read -r testname; do
    [[ -z "$testname" ]] && continue
    focus_regex=$(printf '%s' "$testname" | awk '{
        gsub(/[]\[\(\){}.+*?^$\\]/, "\\\\&")
        print
    }')
    echo "[RERUN] $testname" >&2
    rerun_tmp_dir="${RERUN_REPORT_DIR}/tmp"
    mkdir -p "${rerun_tmp_dir}"
    rerun_log_prefix=$(echo "$testname" | tr ' /:' '__')
    rerun_junit="${rerun_tmp_dir}/junit.${rerun_log_prefix}.xml"

    # Lightweight direct ginkgo rerun path
    # Preconditions: cluster already up, images synced from initial run, ginkgo/test binary built.
    # We'll build test binary on demand if missing.
    if [[ ! -x "${REPO_ROOT}/_out/tests/ginkgo" ]]; then
        echo "[LIGHT] Building functest binaries (once)" >&2
        (cd "${REPO_ROOT}" && make build-functests >/dev/null)
    fi

    # Ensure artifacts dir for light rerun JUnit
    light_artifacts="${rerun_tmp_dir}/artifacts_${rerun_log_prefix}"
    mkdir -p "${light_artifacts}"

    GINKGO_CMD=("${REPO_ROOT}/_out/tests/ginkgo" -r --timeout="${LIGHT_TIMEOUT}" --label-filter="${LIGHT_LABEL_FILTER}" --skip "DataVolume|Filesystem|CDI|Migration" --focus "${focus_regex}")
    if [[ "${LIGHT_NODES}" != "1" ]]; then
        GINKGO_CMD+=(--nodes "${LIGHT_NODES}")
    fi

    SUITE_ARGS=("${REPO_ROOT}/_out/tests/tests.test" -- -kubeconfig="${KUBECONFIG:?KUBECONFIG must be set for light reruns}" -config="${REPO_ROOT}/tests/default-config.json" -container-tag="${DOCKER_TAG}" -container-prefix="${DOCKER_PREFIX}" --artifacts="${light_artifacts}" -apply-default-e2e-configuration -kubectl-path="$(which kubectl)")

    # Add virtctl if present
    if [[ -x "${REPO_ROOT}/_out/cmd/virtctl/virtctl" ]]; then
        SUITE_ARGS+=(-virtctl-path="${REPO_ROOT}/_out/cmd/virtctl/virtctl")
    fi

    # Add user provided LIGHT_EXTRA (split respecting spaces)
    if [[ -n "${LIGHT_EXTRA}" ]]; then
        # shellcheck disable=SC2206
        extra_parts=(${LIGHT_EXTRA})
        SUITE_ARGS+=("${extra_parts[@]}")
    fi

    # Run test and handle result in subshell to prevent early exit
    (
        "${GINKGO_CMD[@]}" "${SUITE_ARGS[@]}"
        test_rc=$?
        if [[ $test_rc -eq 0 ]]; then
            echo "[RERUN-PASS] $testname" | tee -a "${RERUN_REPORT_DIR}/summary.log"
            echo "PASSED" >"${light_artifacts}/test.result"
        else
            echo "[RERUN-FAIL] $testname (rc=$test_rc)" | tee -a "${RERUN_REPORT_DIR}/summary.log"
            echo "FAILED" >"${light_artifacts}/test.result"
        fi
        exit $test_rc
    )
    rc=$?

    # Update counters based on result file
    if [[ -f "${light_artifacts}/test.result" ]]; then
        if [[ $(cat "${light_artifacts}/test.result") == "PASSED" ]]; then
            ((PASSED++)) || true
        else
            ((FAILED++)) || true
        fi
    fi

    # Attempt to locate JUnit file produced by light run. The standard suite writes junit.functest.xml
    set +e
    candidate_junit=$(find "${light_artifacts}" -maxdepth 2 -type f -name 'junit*.xml' | head -n1 || true)
    if [[ -n "${candidate_junit}" ]]; then
        cp "${candidate_junit}" "${rerun_junit}" || true
    fi

    # Extract updated testcase block from rerun file
    rerun_block=""
    if [[ -f "${rerun_junit}" ]]; then
        rerun_block=$(awk -v name="$testname" 'BEGIN{FS="\n";RS=""} {if($0 ~ "<testcase[^>]*name=\""name"\"") {print $0}}' "${rerun_junit}" 2>/dev/null || true)
        if [[ -z "$rerun_block" ]]; then
            # fallback more manual capture
            rerun_block=$(awk -v name="$testname" 'BEGIN{capture=0} /<testcase/{ if($0 ~ "name=\""name"\"") {capture=1; block=$0; next}} capture{block=block"\n"$0} /<\/testcase>/{ if(capture){print block; exit}}' "${rerun_junit}" 2>/dev/null || true)
        fi
    fi
    if [[ -n "$rerun_block" ]]; then
        # Process junit files with error handling
        if [[ -f "${MERGED_JUNIT_FILE}" ]]; then
            # Remove existing block in merged
            awk -v name="$testname" 'BEGIN{skip=0} /<testcase/{ if($0 ~ "name=\""name"\"") {skip=1}} { if(!skip) print } /<\/testcase>/{ if(skip){skip=0}}' "${MERGED_JUNIT_FILE}" >"${MERGED_JUNIT_FILE}.tmp" 2>/dev/null &&
                mv "${MERGED_JUNIT_FILE}.tmp" "${MERGED_JUNIT_FILE}" 2>/dev/null || true

            # Insert new block before closing testsuite
            awk -v block="$rerun_block" 'BEGIN{printed=0} /<\/testsuite>/{ if(!printed){print block; printed=1} } {print}' "${MERGED_JUNIT_FILE}" >"${MERGED_JUNIT_FILE}.tmp" 2>/dev/null &&
                mv "${MERGED_JUNIT_FILE}.tmp" "${MERGED_JUNIT_FILE}" 2>/dev/null || true

            # Recalculate counts using awk for better XML handling
            total_tests=$(grep -c '<testcase ' "${MERGED_JUNIT_FILE}" 2>/dev/null || echo "0")
            total_failures=$(awk 'BEGIN{FS="[<>]";fail=0} /<testcase/{has=0} /<failure/{has=1} /<\/testcase>/{ if(has) fail++} END{print fail}' "${MERGED_JUNIT_FILE}" 2>/dev/null || echo "0")

            awk -v tests="$total_tests" -v fails="$total_failures" '
                /<testsuite/ {
                    sub(/tests="[0-9]+"/, "tests=\"" tests "\"")
                    sub(/failures="[0-9]+"/, "failures=\"" fails "\"")
                }
                { print }
            ' "${MERGED_JUNIT_FILE}" >"${MERGED_JUNIT_FILE}.tmp" 2>/dev/null &&
                mv "${MERGED_JUNIT_FILE}.tmp" "${MERGED_JUNIT_FILE}" 2>/dev/null || true
        fi
        echo "[INFO] Updated merged junit: tests=${total_tests} failures=${total_failures}" >&2
    fi

done <<<"${FAILED_LIST}"
# Re-enable exit on error after the loop
set -e

echo "[RESULT] Rerun passed=${PASSED} failed=${FAILED}" | tee -a "${RERUN_REPORT_DIR}/summary.log"
echo "[INFO] Final merged JUnit file: ${MERGED_JUNIT_FILE}" >&2

if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
exit 0
