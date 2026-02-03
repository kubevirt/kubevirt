#!/usr/bin/env bash
#
# Verify RPM lock files - check that cached RPMs match lock file checksums
#
# Usage: ./hack/rpm-verify.sh [lock_file...]
#
# If no lock files specified, verifies all lock files in rpm-lockfiles/
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBEVIRT_DIR="${SCRIPT_DIR}/.."
LOCKFILES_DIR="${KUBEVIRT_DIR}/rpm-lockfiles"
CACHE_DIR="${KUBEVIRT_DIR}/rpm-cache"

# =============================================================================
# Functions
# =============================================================================

verify_lock_file() {
    local lock_file=$1
    local errors=0

    if [ ! -f "${lock_file}" ]; then
        echo "ERROR: Lock file not found: ${lock_file}"
        return 1
    fi

    # Extract metadata
    local arch pkg_set
    arch=$(jq -r '.architecture' "${lock_file}")
    pkg_set=$(jq -r '.package_set' "${lock_file}")
    local cache_path="${CACHE_DIR}/${arch}/${pkg_set}"

    echo "Verifying: ${lock_file}"
    echo "  Architecture: ${arch}"
    echo "  Package set:  ${pkg_set}"
    echo "  Cache path:   ${cache_path}"

    if [ ! -d "${cache_path}" ]; then
        echo "  WARNING: Cache directory not found - skipping SHA256 verification"
        echo "  (Run rpm-freeze-native.sh to populate cache)"
        return 0
    fi

    # Verify each package
    local total verified missing mismatch
    total=$(jq -r '.packages | length' "${lock_file}")
    verified=0
    missing=0
    mismatch=0

    while IFS='|' read -r filename expected_sha256; do
        local rpm_path="${cache_path}/${filename}"

        if [ ! -f "${rpm_path}" ]; then
            echo "  MISSING: ${filename}"
            missing=$((missing + 1))
            continue
        fi

        local actual_sha256
        actual_sha256=$(sha256sum "${rpm_path}" | cut -d' ' -f1)

        if [ "${actual_sha256}" = "${expected_sha256}" ]; then
            verified=$((verified + 1))
        else
            echo "  MISMATCH: ${filename}"
            echo "    Expected: ${expected_sha256}"
            echo "    Actual:   ${actual_sha256}"
            mismatch=$((mismatch + 1))
            errors=$((errors + 1))
        fi
    done < <(jq -r '.packages[] | "\(.filename)|\(.sha256)"' "${lock_file}")

    echo "  Results: ${verified}/${total} verified, ${missing} missing, ${mismatch} mismatched"

    if [ "${mismatch}" -gt 0 ]; then
        return 1
    fi

    return 0
}

# =============================================================================
# Main
# =============================================================================

# Check for jq
if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required but not installed"
    exit 1
fi

# Determine which lock files to verify
LOCK_FILES=()
if [ $# -gt 0 ]; then
    LOCK_FILES=("$@")
else
    if [ -d "${LOCKFILES_DIR}" ]; then
        for f in "${LOCKFILES_DIR}"/*.lock.json; do
            [ -f "$f" ] && LOCK_FILES+=("$f")
        done
    fi
fi

if [ ${#LOCK_FILES[@]} -eq 0 ]; then
    echo "No lock files found to verify"
    echo ""
    echo "Usage: $0 [lock_file...]"
    echo ""
    echo "If no lock files specified, verifies all files in rpm-lockfiles/"
    exit 0
fi

echo "=============================================="
echo "RPM Lock File Verification"
echo "=============================================="
echo ""

# Verify each lock file
TOTAL=${#LOCK_FILES[@]}
SUCCESS=0
FAILED=0

for lock_file in "${LOCK_FILES[@]}"; do
    echo ""
    if verify_lock_file "${lock_file}"; then
        SUCCESS=$((SUCCESS + 1))
    else
        FAILED=$((FAILED + 1))
    fi
done

# =============================================================================
# Summary
# =============================================================================

echo ""
echo "=============================================="
echo "Verification Summary"
echo "=============================================="
echo "Total:   ${TOTAL}"
echo "Success: ${SUCCESS}"
echo "Failed:  ${FAILED}"

if [ "${FAILED}" -gt 0 ]; then
    echo ""
    echo "ERROR: Some lock files failed verification"
    exit 1
fi

echo ""
echo "All lock files verified successfully!"
