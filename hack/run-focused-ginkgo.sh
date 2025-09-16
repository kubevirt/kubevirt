#!/usr/bin/env bash
# Run a focused subset of KubeVirt tests directly with ginkgo, bypassing automation/test.sh.
# Useful for quickly iterating on a single failing Conformance test.
#
# Prerequisites:
#  - Cluster is already up (make cluster-up, or external provider env ready)
#  - KUBECONFIG points to the target cluster
#  - Functional test binaries built (run `make build-functests` or any prior full test invocation)
#
# Environment variables:
#  FOCUS        Regex to match test names (KUBEVIRT_E2E_FOCUS equivalent)
#  SKIP         Regex to skip test names (KUBEVIRT_E2E_SKIP equivalent)
#  LABEL_FILTER Ginkgo label filter expression (defaults to Conformance)
#  TIMEOUT      Overall ginkgo timeout (default 2h)
#  NODES        Parallel nodes (default 1 for easier debugging)
#  ARTIFACTS    Directory for artifacts (default _out/artifacts/focused)
#  EXTRA        Extra arguments appended after the standard ones
#
# Examples:
#   FOCUS='GuestAgent Readiness Probe' hack/run-focused-ginkgo.sh
#   FOCUS='\\[test_id:1234\\]' hack/run-focused-ginkgo.sh
#   FOCUS='networking.*icmp' LABEL_FILTER='(Conformance && sig-network)' hack/run-focused-ginkgo.sh
#
set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "${REPO_ROOT}" || exit 1

: "${KUBECONFIG:?KUBECONFIG must be set}"

LABEL_FILTER=${LABEL_FILTER:-Conformance}
FOCUS=${FOCUS:-}
SKIP=${SKIP:-}
TIMEOUT=${TIMEOUT:-2h}
NODES=${NODES:-1}
ARTIFACTS=${ARTIFACTS:-${REPO_ROOT}/_out/artifacts/focused}
EXTRA=${EXTRA:-}

mkdir -p "${ARTIFACTS}"

if [[ ! -x _out/tests/ginkgo ]]; then
    echo "[INFO] ginkgo binary not found, building functests..." >&2
    make build-functests >/dev/null
fi

GINKGO_CMD=(_out/tests/ginkgo -r --timeout="${TIMEOUT}" --label-filter="${LABEL_FILTER}")

if [[ -n "${FOCUS}" ]]; then
    GINKGO_CMD+=(--focus "${FOCUS}")
fi
if [[ -n "${SKIP}" ]]; then
    GINKGO_CMD+=(--skip "${SKIP}")
fi

if [[ "${NODES}" != "1" ]]; then
    GINKGO_CMD+=(--nodes "${NODES}")
fi

# Test binary and its suite args (mirror minimal set used in hack/functests.sh)
SUITE_ARGS=(_out/tests/tests.test -- -kubeconfig="${KUBECONFIG}" -config=tests/default-config.json -container-tag=latest -container-prefix=quay.io/kubevirt --artifacts="${ARTIFACTS}")

# Preserve virtctl path if built
if [[ -x _out/cmd/virtctl/virtctl ]]; then
    SUITE_ARGS+=(-virtctl-path="${REPO_ROOT}/_out/cmd/virtctl/virtctl")
fi

# Apply default e2e config unless explicitly disabled
SUITE_ARGS+=(-apply-default-e2e-configuration)

# Append any extra args user provided
if [[ -n "${EXTRA}" ]]; then
    # shellcheck disable=SC2206
    SUITE_ARGS+=(${EXTRA})
fi

set -x
"${GINKGO_CMD[@]}" "${SUITE_ARGS[@]}" || exit_code=$? || true
set +x

exit ${exit_code:-0}
