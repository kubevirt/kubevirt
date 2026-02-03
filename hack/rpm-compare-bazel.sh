#!/usr/bin/env bash

# =============================================================================
# rpm-compare-bazel.sh
# =============================================================================
# Compares native RPM lock files with bazeldnf rpmtree definitions.
# This is a validation script to ensure our native RPM freezing mechanism
# produces identical package sets to what bazeldnf generates.
#
# Usage:
#   ./hack/rpm-compare-bazel.sh [options] [package_set] [arch]
#
# Options:
#   --names-only    Compare only package names (ignore versions)
#   --ignore-arch   Treat noarch packages as matching any arch
#   --verbose       Show detailed package lists
#
# Examples:
#   ./hack/rpm-compare-bazel.sh                           # Compare all (strict)
#   ./hack/rpm-compare-bazel.sh --names-only              # Compare names only
#   ./hack/rpm-compare-bazel.sh --ignore-arch launcherbase x86_64
#
# Exit codes:
#   0 - All comparisons match
#   1 - Differences found or error
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBEVIRT_DIR="${SCRIPT_DIR}/.."
BUILD_BAZEL="${KUBEVIRT_DIR}/rpm/BUILD.bazel"
LOCKFILES_DIR="${KUBEVIRT_DIR}/rpm-lockfiles"

# Options
NAMES_ONLY=false
IGNORE_ARCH=false
ALLOW_EXTRAS=false
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Convert bazeldnf package reference to normalized format
# Input:  "@acl-0__2.3.1-4.el9.x86_64//rpm"
# Output: acl-0:2.3.1-4.el9.x86_64
normalize_bazel_pkg() {
    local pkg="$1"
    
    # Remove quotes, @prefix and //rpm suffix
    pkg="${pkg//\"/}"
    pkg="${pkg#@}"
    pkg="${pkg%//rpm}"
    
    # Handle special characters:
    # __plus__ -> +
    # __caret__ -> ^
    pkg="${pkg//__plus__/+}"
    pkg="${pkg//__caret__/^}"
    
    # Convert epoch separator: the __ between epoch and version becomes :
    # Pattern: name-epoch__version-release.arch
    # Example: acl-0__2.3.1-4.el9.x86_64 -> acl-0:2.3.1-4.el9.x86_64
    # But names can have underscores too, so we need to be careful
    # The pattern is: after -DIGIT__ replace with :
    
    # Use sed for more reliable replacement
    pkg=$(echo "$pkg" | sed -E 's/-([0-9]+)__/-\1:/')
    
    echo "$pkg"
}

# Convert JSON package entry to normalized format
# Input: {"name":"acl","epoch":"0","version":"2.3.1","release":"4.el9","arch":"x86_64",...}
# Output: acl-0:2.3.1-4.el9.x86_64
normalize_json_pkg() {
    local name="$1"
    local epoch="$2"
    local version="$3"
    local release="$4"
    local arch="$5"
    
    echo "${name}-${epoch}:${version}-${release}.${arch}"
}

# Extract packages from bazeldnf rpmtree definition
# Parses rpm/BUILD.bazel for a specific rpmtree target
extract_bazel_packages() {
    local target_name="$1"
    local in_target=false
    local in_rpms=false
    local brace_count=0
    
    while IFS= read -r line; do
        # Look for the target
        if [[ "$line" =~ ^rpmtree\( ]]; then
            in_target=false
            brace_count=1
        fi
        
        if [[ "$line" =~ name[[:space:]]*=[[:space:]]*\"${target_name}\" ]]; then
            in_target=true
        fi
        
        if $in_target; then
            # Track braces to know when we exit the target
            brace_count=$((brace_count + $(echo "$line" | tr -cd '(' | wc -c)))
            brace_count=$((brace_count - $(echo "$line" | tr -cd ')' | wc -c)))
            
            # Look for rpms array start
            if [[ "$line" =~ rpms[[:space:]]*=[[:space:]]*\[ ]]; then
                in_rpms=true
            fi
            
            # Extract package references
            if $in_rpms && [[ "$line" =~ \"@[^\"]+//rpm\" ]]; then
                # Extract all package references from the line
                echo "$line" | grep -oE '"@[^"]+//rpm"' | while read -r pkg; do
                    normalize_bazel_pkg "$pkg"
                done
            fi
            
            # End of rpms array
            if $in_rpms && [[ "$line" =~ \], ]]; then
                in_rpms=false
            fi
            
            # End of target
            if [[ $brace_count -le 0 ]]; then
                break
            fi
        fi
    done < "${BUILD_BAZEL}"
}

# Extract packages from JSON lock file
extract_json_packages() {
    local lock_file="$1"
    
    if [[ ! -f "$lock_file" ]]; then
        return 1
    fi
    
    jq -r '.packages[] | "\(.name)-\(.epoch):\(.version)-\(.release).\(.arch)"' "$lock_file"
}

# Compare two package lists
compare_packages() {
    local pkg_set="$1"
    local arch="$2"
    local bazel_target="${pkg_set}_${arch}"
    local lock_file="${LOCKFILES_DIR}/${pkg_set}-${arch}.lock.json"
    
    # Handle special naming for libguestfs-tools (no arch suffix in some cases)
    if [[ "$pkg_set" == "libguestfs-tools" && "$arch" == "aarch64" ]]; then
        bazel_target="libguestfs-tools"
    fi
    
    echo ""
    echo "=============================================="
    echo "Comparing: ${pkg_set} / ${arch}"
    local mode_desc="strict"
    $NAMES_ONLY && mode_desc="names-only"
    $IGNORE_ARCH && mode_desc="${mode_desc}+ignore-arch"
    $ALLOW_EXTRAS && mode_desc="${mode_desc}+allow-extras"
    echo "Mode: ${mode_desc}"
    echo "=============================================="
    echo "Bazel target: ${bazel_target}"
    echo "Lock file:    ${lock_file}"
    echo ""
    
    # Check if lock file exists
    if [[ ! -f "$lock_file" ]]; then
        log_warn "Lock file not found: ${lock_file}"
        echo "  Run: ./hack/rpm-freeze-native.sh ${arch} ${pkg_set}"
        return 2
    fi
    
    # Extract packages from both sources
    local bazel_pkgs_raw
    local json_pkgs_raw
    
    bazel_pkgs_raw=$(extract_bazel_packages "$bazel_target")
    json_pkgs_raw=$(extract_json_packages "$lock_file")
    
    if [[ -z "$bazel_pkgs_raw" ]]; then
        log_warn "No packages found in Bazel target: ${bazel_target}"
        return 2
    fi
    
    # Apply normalization for comparison
    local bazel_pkgs
    local json_pkgs
    
    bazel_pkgs=$(echo "$bazel_pkgs_raw" | while read -r pkg; do
        normalize_for_comparison "$pkg" "$arch"
    done | sort -u)
    
    json_pkgs=$(echo "$json_pkgs_raw" | while read -r pkg; do
        normalize_for_comparison "$pkg" "$arch"
    done | sort -u)
    
    # Compare
    local bazel_count
    local json_count
    bazel_count=$(echo "$bazel_pkgs" | grep -c . || echo 0)
    json_count=$(echo "$json_pkgs" | grep -c . || echo 0)
    
    echo "Bazel packages: ${bazel_count}"
    echo "JSON packages:  ${json_count}"
    echo ""
    
    # Find differences
    local only_in_bazel
    local only_in_json
    
    only_in_bazel=$(comm -23 <(echo "$bazel_pkgs") <(echo "$json_pkgs") || true)
    only_in_json=$(comm -13 <(echo "$bazel_pkgs") <(echo "$json_pkgs") || true)
    
    local has_error=false
    local has_warning=false
    
    if [[ -n "$only_in_bazel" ]]; then
        has_error=true
        log_error "Packages only in Bazel (MISSING from native lock file):"
        echo "$only_in_bazel" | sed 's/^/  - /'
        echo ""
    fi
    
    if [[ -n "$only_in_json" ]]; then
        if $ALLOW_EXTRAS; then
            has_warning=true
            log_warn "Extra packages in native lock file (not in Bazel) - allowed:"
            echo "$only_in_json" | sed 's/^/  - /'
            echo ""
        else
            has_error=true
            log_error "Packages only in native lock file (not in Bazel):"
            echo "$only_in_json" | sed 's/^/  - /'
            echo ""
        fi
    fi
    
    if $has_error; then
        log_error "MISMATCH: ${pkg_set}/${arch}"
        return 1
    elif $has_warning; then
        log_info "MATCH (with extras): ${pkg_set}/${arch} (Bazel: ${bazel_count}, Native: ${json_count})"
        return 0
    else
        log_info "MATCH: ${pkg_set}/${arch} (${bazel_count} packages)"
        return 0
    fi
}

# Normalize package for comparison based on options
normalize_for_comparison() {
    local pkg="$1"
    local target_arch="$2"
    
    if $NAMES_ONLY; then
        # Extract just the package name (before first -)
        echo "$pkg" | sed -E 's/-[0-9]+:.*//'
    elif $IGNORE_ARCH; then
        # Replace noarch with target arch, or strip arch entirely
        echo "$pkg" | sed -E "s/\.noarch$/.${target_arch}/"
    else
        echo "$pkg"
    fi
}

# =============================================================================
# Main
# =============================================================================

# Parse options
while [[ $# -gt 0 ]]; do
    case "$1" in
        --names-only)
            NAMES_ONLY=true
            shift
            ;;
        --ignore-arch)
            IGNORE_ARCH=true
            shift
            ;;
        --allow-extras)
            ALLOW_EXTRAS=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        -*)
            echo "Unknown option: $1"
            exit 1
            ;;
        *)
            break
            ;;
    esac
done

# Source package definitions
source "${SCRIPT_DIR}/rpm-packages.sh"

FILTER_SET="${1:-}"
FILTER_ARCH="${2:-}"

# Check dependencies
if ! command -v jq &>/dev/null; then
    log_error "jq is required but not installed"
    exit 1
fi

if [[ ! -f "$BUILD_BAZEL" ]]; then
    log_error "BUILD.bazel not found: ${BUILD_BAZEL}"
    exit 1
fi

echo "=============================================="
echo "RPM Lock File vs Bazel Comparison"
echo "=============================================="
echo "BUILD.bazel: ${BUILD_BAZEL}"
echo "Lock files:  ${LOCKFILES_DIR}/"
echo ""

TOTAL=0
PASSED=0
FAILED=0
SKIPPED=0

# Get all package sets
PACKAGE_SETS=$(get_all_package_sets)

for pkg_set in ${PACKAGE_SETS}; do
    # Filter by package set if specified
    if [[ -n "$FILTER_SET" && "$pkg_set" != "$FILTER_SET" ]]; then
        continue
    fi
    
    # Get architectures for this package set
    ARCHITECTURES=$(get_architectures "$pkg_set")
    
    for arch in ${ARCHITECTURES}; do
        # Filter by architecture if specified
        if [[ -n "$FILTER_ARCH" && "$arch" != "$FILTER_ARCH" ]]; then
            continue
        fi
        
        TOTAL=$((TOTAL + 1))
        
        if compare_packages "$pkg_set" "$arch"; then
            PASSED=$((PASSED + 1))
        else
            result=$?
            if [[ $result -eq 2 ]]; then
                SKIPPED=$((SKIPPED + 1))
            else
                FAILED=$((FAILED + 1))
            fi
        fi
    done
done

echo ""
echo "=============================================="
echo "Summary"
echo "=============================================="
echo "Total:   ${TOTAL}"
echo "Passed:  ${PASSED}"
echo "Failed:  ${FAILED}"
echo "Skipped: ${SKIPPED}"
echo ""

if [[ $FAILED -gt 0 ]]; then
    log_error "Some comparisons failed!"
    exit 1
elif [[ $SKIPPED -eq $TOTAL ]]; then
    log_warn "All comparisons skipped (no lock files found)"
    echo ""
    echo "Generate lock files first:"
    echo "  ./hack/rpm-freeze-all.sh"
    exit 1
else
    log_info "All comparisons passed!"
    exit 0
fi
