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
#   MAX_FAILS             Limit number of failures to rerun (default: all)
#
# Requirements:
#   - jq, xmlstarlet or xmllint? (We implement parsing with awk/sed only; no extra deps)
#
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
JUNIT_FILE="${REPO_ROOT}/_out/artifacts/junit.functest.xml"
MERGED_JUNIT_FILE="${REPO_ROOT}/_out/artifacts/junit.functest.merged.xml"
: "${TARGET:?TARGET environment variable must be set (e.g. kind-1.29, external)}"

# Lightweight rerun defaults
LIGHT_NODES=${LIGHT_NODES:-1}
LIGHT_TIMEOUT=${LIGHT_TIMEOUT:-30m}
LIGHT_LABEL_FILTER=${LIGHT_LABEL_FILTER:-Conformance}
LIGHT_EXTRA=${LIGHT_EXTRA:-}

# Run full conformance suite unless skipping
if [[ "${REUSE_INITIAL_RUN:-}" != "1" ]]; then
  echo "[INFO] Running initial Conformance suite via automation/test.sh" >&2
  ( cd "${REPO_ROOT}" && KUBEVIRT_LABEL_FILTER=Conformance TARGET="${TARGET}" ./automation/test.sh ${EXTRA_TEST_ARGS:-} ) || true
else
  echo "[INFO] Skipping initial run (REUSE_INITIAL_RUN=1)" >&2
fi

if [[ ! -f "${JUNIT_FILE}" ]]; then
  echo "[ERROR] JUnit file not found at ${JUNIT_FILE}" >&2
  exit 2
fi

# Extract failed test case names from junit. A failure is indicated by a <failure> child element.
# We capture the name attribute of the parent <testcase>.
FAILED_LIST=$(awk 'BEGIN{FS="[<>]"} /<testcase/{name="";hasfail=0; for(i=1;i<=NF;i++){if($i ~ /name=\"/){sub(/.*name=\"/,"",$i); sub(/\".*/,"",$i); name=$i}}} /<failure/{hasfail=1} /<\/testcase>/{ if(hasfail&&name!=""){print name} }' "${JUNIT_FILE}" | sort -u )

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

while IFS= read -r testname; do
  [[ -z "$testname" ]] && continue
  focus_regex=$(printf '%s' "$testname" | sed -E 's/[](){}.+*?^$|\\/\\&/g')
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

  GINKGO_CMD=( "${REPO_ROOT}/_out/tests/ginkgo" -r --timeout="${LIGHT_TIMEOUT}" --label-filter="${LIGHT_LABEL_FILTER}" --focus "${focus_regex}" )
  if [[ "${LIGHT_NODES}" != "1" ]]; then
    GINKGO_CMD+=( --nodes "${LIGHT_NODES}" )
  fi

  SUITE_ARGS=( "${REPO_ROOT}/_out/tests/tests.test" -- -kubeconfig="${KUBECONFIG:?KUBECONFIG must be set for light reruns}" -config=tests/default-config.json -container-tag=latest -container-prefix=quay.io/kubevirt --artifacts="${light_artifacts}" -apply-default-e2e-configuration )

  # Add virtctl if present
  if [[ -x "${REPO_ROOT}/_out/cmd/virtctl/virtctl" ]]; then
    SUITE_ARGS+=( -virtctl-path="${REPO_ROOT}/_out/cmd/virtctl/virtctl" )
  fi

  # Add user provided LIGHT_EXTRA (split respecting spaces)
  if [[ -n "${LIGHT_EXTRA}" ]]; then
    # shellcheck disable=SC2206
    extra_parts=( ${LIGHT_EXTRA} )
    SUITE_ARGS+=( "${extra_parts[@]}" )
  fi

  set +e
  (cd "${REPO_ROOT}" && "${GINKGO_CMD[@]}" "${SUITE_ARGS[@]}")
  rc=$?
  set -e
  if [[ $rc -eq 0 ]]; then
    echo "[RERUN-PASS] $testname" | tee -a "${RERUN_REPORT_DIR}/summary.log"
    ((PASSED++))
  else
    echo "[RERUN-FAIL] $testname (rc=$rc)" | tee -a "${RERUN_REPORT_DIR}/summary.log"
    ((FAILED++))
  fi

  # Attempt to locate JUnit file produced by light run. The standard suite writes junit.functest.xml
  candidate_junit=$(find "${light_artifacts}" -maxdepth 2 -type f -name 'junit*.xml' | head -n1 || true)
  if [[ -n "${candidate_junit}" ]]; then
    cp "${candidate_junit}" "${rerun_junit}" || true
  fi

  # Extract updated testcase block from rerun file
  rerun_block=$(awk -v name="$testname" 'BEGIN{FS="\n";RS=""} {if($0 ~ "<testcase[^>]*name=\""name"\"") {print $0}}' "${rerun_junit}" 2>/dev/null || true)
  if [[ -z "$rerun_block" ]]; then
    # fallback more manual capture
    rerun_block=$(awk -v name="$testname" 'BEGIN{capture=0} /<testcase/{ if($0 ~ "name=\""name"\"") {capture=1; block=$0; next}} capture{block=block"\n"$0} /<\/testcase>/{ if(capture){print block; exit}}' "${rerun_junit}")
  fi
  if [[ -n "$rerun_block" ]]; then
    # Remove existing block in merged
    awk -v name="$testname" 'BEGIN{skip=0} /<testcase/{ if($0 ~ "name=\""name"\"") {skip=1}} { if(!skip) print } /<\/testcase>/{ if(skip){skip=0}}' "${MERGED_JUNIT_FILE}" > "${MERGED_JUNIT_FILE}.tmp" && mv "${MERGED_JUNIT_FILE}.tmp" "${MERGED_JUNIT_FILE}"
    # Insert new block before closing testsuite
    awk -v block="$rerun_block" 'BEGIN{printed=0} /<\/testsuite>/{ if(!printed){print block; printed=1} } {print}' "${MERGED_JUNIT_FILE}" > "${MERGED_JUNIT_FILE}.tmp" && mv "${MERGED_JUNIT_FILE}.tmp" "${MERGED_JUNIT_FILE}"
    # Recalculate counts
    total_tests=$(grep -c '<testcase ' "${MERGED_JUNIT_FILE}" || true)
    total_failures=$(awk 'BEGIN{FS="[<>]";fail=0} /<testcase/{has=0} /<failure/{has=1} /<\/testcase>/{ if(has) fail++} END{print fail}' "${MERGED_JUNIT_FILE}")
    sed -E -i "s/(<testsuite[^>]*tests=)\"[0-9]+\"/\1\"${total_tests}\"/" "${MERGED_JUNIT_FILE}" || true
    sed -E -i "s/(<testsuite[^>]*failures=)\"[0-9]+\"/\1\"${total_failures}\"/" "${MERGED_JUNIT_FILE}" || true
    echo "[INFO] Updated merged junit: tests=${total_tests} failures=${total_failures}" >&2
  fi

done <<< "${FAILED_LIST}"

echo "[RESULT] Rerun passed=${PASSED} failed=${FAILED}" | tee -a "${RERUN_REPORT_DIR}/summary.log"
echo "[INFO] Final merged JUnit file: ${MERGED_JUNIT_FILE}" >&2

if [[ $FAILED -gt 0 ]]; then
  exit 1
fi
exit 0
