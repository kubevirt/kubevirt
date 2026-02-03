#!/usr/bin/env bash
#
# Generate all RPM lock files for all package sets and architectures
#
# Usage: ./hack/rpm-freeze-all.sh [--single-arch ARCH] [--single-set PACKAGE_SET]
#
# This script should be run inside the builder container or a CentOS Stream 9
# environment with dnf and jq installed.
#

set -euo pipefail

# Timing support
START_TIME=$(date +%s.%N)

show_timing() {
    local end_time=$(date +%s.%N)
    local duration=$(echo "${end_time} - ${START_TIME}" | bc)
    echo ""
    echo "=============================================="
    echo "Total Time: ${duration}s (native freeze-all)"
    echo "=============================================="
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source package definitions
source "${SCRIPT_DIR}/rpm-packages.sh"

# =============================================================================
# Parse Arguments
# =============================================================================

SINGLE_ARCH=""
SINGLE_SET=""
PARALLEL=${PARALLEL:-false}

while [[ $# -gt 0 ]]; do
    case $1 in
        --single-arch)
            SINGLE_ARCH="$2"
            shift 2
            ;;
        --single-set)
            SINGLE_SET="$2"
            shift 2
            ;;
        --parallel)
            PARALLEL=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--single-arch ARCH] [--single-set PACKAGE_SET] [--parallel]"
            echo ""
            echo "Options:"
            echo "  --single-arch ARCH     Only process specified architecture"
            echo "  --single-set SET       Only process specified package set"
            echo "  --parallel             Run freeze operations in parallel"
            echo ""
            echo "Examples:"
            echo "  $0                              # Generate all lock files"
            echo "  $0 --single-arch x86_64         # Only x86_64 architecture"
            echo "  $0 --single-set launcherbase    # Only launcherbase package set"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# =============================================================================
# Main
# =============================================================================

echo "=============================================="
echo "RPM Freeze All - Generating Lock Files"
echo "=============================================="
echo ""

# Get package sets to process
if [ -n "${SINGLE_SET}" ]; then
    PACKAGE_SETS="${SINGLE_SET}"
else
    PACKAGE_SETS=$(get_all_package_sets)
fi

# Track results
TOTAL=0
SUCCESS=0
FAILED=0
FAILED_LIST=""

# Process each package set
for pkg_set in ${PACKAGE_SETS}; do
    # Get architectures for this package set
    if [ -n "${SINGLE_ARCH}" ]; then
        ARCHITECTURES="${SINGLE_ARCH}"
    else
        ARCHITECTURES=$(get_architectures "${pkg_set}")
    fi

    for arch in ${ARCHITECTURES}; do
        TOTAL=$((TOTAL + 1))
        echo "----------------------------------------------"
        echo "Processing: ${pkg_set} / ${arch}"
        echo "----------------------------------------------"

        if "${SCRIPT_DIR}/rpm-freeze-native.sh" "${arch}" "${pkg_set}"; then
            SUCCESS=$((SUCCESS + 1))
            echo "SUCCESS: ${pkg_set}-${arch}"
        else
            FAILED=$((FAILED + 1))
            FAILED_LIST="${FAILED_LIST} ${pkg_set}-${arch}"
            echo "FAILED: ${pkg_set}-${arch}"
        fi
        echo ""
    done
done

# =============================================================================
# Summary
# =============================================================================

echo "=============================================="
echo "RPM Freeze All - Summary"
echo "=============================================="
echo "Total:   ${TOTAL}"
echo "Success: ${SUCCESS}"
echo "Failed:  ${FAILED}"

if [ "${FAILED}" -gt 0 ]; then
    echo ""
    echo "Failed package sets:"
    for item in ${FAILED_LIST}; do
        echo "  - ${item}"
    done
    exit 1
fi

echo ""
echo "All lock files generated successfully!"
echo ""
echo "Lock files are in: rpm-lockfiles/"
echo ""
echo "Next steps:"
echo "  1. Review the lock files"
echo "  2. Run verification: ./hack/rpm-verify.sh"
echo "  3. Commit lock files to git"

show_timing
