#!/usr/bin/env bash
#
# Native RPM freezing tool
# Generates JSON lock files with SHA256 checksums for reproducible builds
#
# Usage: ./hack/rpm-freeze-native.sh <arch> <package_set>
# Example: ./hack/rpm-freeze-native.sh x86_64 launcherbase
#
# This script should be run inside the builder container or a CentOS Stream 9
# environment with dnf and jq installed.
#

set -euo pipefail

# Timing support
START_TIME=$(date +%s.%N)
timing_enabled=${RPM_FREEZE_TIMING:-false}

show_timing() {
    if [[ "${timing_enabled}" == "true" ]]; then
        local end_time=$(date +%s.%N)
        local duration=$(echo "${end_time} - ${START_TIME}" | bc)
        echo ""
        echo "=============================================="
        echo "Timing: ${duration}s (native freeze)"
        echo "=============================================="
    fi
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBEVIRT_DIR="${SCRIPT_DIR}/.."

# Source package definitions
source "${SCRIPT_DIR}/rpm-packages.sh"

# =============================================================================
# Configuration
# =============================================================================

ARCH=${1:-}
PACKAGE_SET=${2:-}
OUTPUT_DIR="${KUBEVIRT_DIR}/rpm-lockfiles"
CACHE_DIR="${KUBEVIRT_DIR}/rpm-cache"
DNF_CONF_DIR="/tmp/kubevirt-rpm-freeze"

if [ -z "${ARCH}" ] || [ -z "${PACKAGE_SET}" ]; then
    echo "Usage: $0 <arch> <package_set>"
    echo ""
    echo "Architectures: x86_64, aarch64, s390x"
    echo "Package sets: $(get_all_package_sets | tr ' ' ', ')"
    echo ""
    echo "Examples:"
    echo "  $0 x86_64 launcherbase"
    echo "  $0 aarch64 handlerbase"
    exit 1
fi

# Validate architecture
case "${ARCH}" in
    x86_64|aarch64|s390x) ;;
    *)
        echo "ERROR: Invalid architecture: ${ARCH}"
        echo "Supported: x86_64, aarch64, s390x"
        exit 1
        ;;
esac

# Get packages for this set (validates package set name)
PACKAGES=$(get_packages "${PACKAGE_SET}" "${ARCH}")
if [ -z "${PACKAGES}" ]; then
    echo "ERROR: Failed to get packages for ${PACKAGE_SET}/${ARCH}"
    exit 1
fi

# Get exclusion patterns (for post-filtering, not dnf install)
EXCLUSIONS=$(get_exclusions "${PACKAGE_SET}")

echo "=============================================="
echo "RPM Freeze: ${PACKAGE_SET} for ${ARCH}"
echo "=============================================="
echo "Packages: ${PACKAGES}"
echo "Post-filter exclusions: ${EXCLUSIONS:-none}"
echo ""

# =============================================================================
# Setup
# =============================================================================

mkdir -p "${OUTPUT_DIR}" "${CACHE_DIR}/${ARCH}/${PACKAGE_SET}" "${DNF_CONF_DIR}"

# Create DNF configuration for target architecture
DNF_CONF="${DNF_CONF_DIR}/dnf-${ARCH}.conf"

cat > "${DNF_CONF}" << EOF
[main]
gpgcheck=1
installonly_limit=3
clean_requirements_on_remove=True
best=False
skip_if_unavailable=False
install_weak_deps=False

[centos-baseos-${ARCH}]
name=CentOS Stream 9 BaseOS ${ARCH}
baseurl=http://mirror.stream.centos.org/9-stream/BaseOS/${ARCH}/os/
gpgcheck=1
gpgkey=https://www.centos.org/keys/RPM-GPG-KEY-CentOS-Official

[centos-appstream-${ARCH}]
name=CentOS Stream 9 AppStream ${ARCH}
baseurl=http://mirror.stream.centos.org/9-stream/AppStream/${ARCH}/os/
gpgcheck=1
gpgkey=https://www.centos.org/keys/RPM-GPG-KEY-CentOS-Official

[centos-crb-${ARCH}]
name=CentOS Stream 9 CRB ${ARCH}
baseurl=http://mirror.stream.centos.org/9-stream/CRB/${ARCH}/os/
gpgcheck=1
gpgkey=https://www.centos.org/keys/RPM-GPG-KEY-CentOS-Official
EOF

# =============================================================================
# Resolve Dependencies
# =============================================================================

echo "Resolving dependencies..."

# Create a temporary installroot to avoid polluting host
INSTALLROOT="${DNF_CONF_DIR}/installroot-${ARCH}-${PACKAGE_SET}"
rm -rf "${INSTALLROOT}"
mkdir -p "${INSTALLROOT}/var/lib/rpm"

# Initialize RPM database in installroot
rpm --root="${INSTALLROOT}" --initdb

# Use dnf to download packages (this resolves dependencies)
# NOTE: We don't use --exclude here because dnf's exclude breaks dependency resolution
# Instead, we'll filter out unwanted packages from the lock file afterward
echo "Downloading packages to cache..."
if ! dnf \
    --config="${DNF_CONF}" \
    --installroot="${INSTALLROOT}" \
    --releasever=9 \
    --setopt=install_weak_deps=False \
    --setopt=tsflags=nodocs \
    --forcearch="${ARCH}" \
    --downloadonly \
    --downloaddir="${CACHE_DIR}/${ARCH}/${PACKAGE_SET}" \
    install -y ${PACKAGES} 2>&1; then
    echo "WARNING: dnf reported errors, but continuing to process downloaded packages"
fi

# =============================================================================
# Generate Lock File
# =============================================================================

echo "Generating lock file..."

LOCK_FILE="${OUTPUT_DIR}/${PACKAGE_SET}-${ARCH}.lock.json"
TEMP_LOCK_FILE="${LOCK_FILE}.tmp"

# Start JSON structure
cat > "${TEMP_LOCK_FILE}" << EOF
{
  "schema_version": "1.0",
  "architecture": "${ARCH}",
  "package_set": "${PACKAGE_SET}",
  "generated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "generator": "rpm-freeze-native.sh",
  "packages": [
EOF

# Process each downloaded RPM
FIRST=true
PACKAGE_COUNT=0
for rpm_file in "${CACHE_DIR}/${ARCH}/${PACKAGE_SET}"/*.rpm; do
    if [ ! -f "${rpm_file}" ]; then
        continue
    fi

    # Extract RPM metadata
    PKG_NAME=$(rpm -qp --queryformat '%{NAME}' "${rpm_file}" 2>/dev/null)
    PKG_VERSION=$(rpm -qp --queryformat '%{VERSION}' "${rpm_file}" 2>/dev/null)
    PKG_RELEASE=$(rpm -qp --queryformat '%{RELEASE}' "${rpm_file}" 2>/dev/null)
    PKG_EPOCH=$(rpm -qp --queryformat '%{EPOCH}' "${rpm_file}" 2>/dev/null)
    PKG_ARCH=$(rpm -qp --queryformat '%{ARCH}' "${rpm_file}" 2>/dev/null)
    PKG_SHA256=$(sha256sum "${rpm_file}" | cut -d' ' -f1)
    PKG_FILENAME=$(basename "${rpm_file}")

    # Handle "(none)" epoch
    if [ "${PKG_EPOCH}" = "(none)" ]; then
        PKG_EPOCH="0"
    fi

    # Add comma separator after first entry
    if [ "${FIRST}" = "true" ]; then
        FIRST=false
    else
        echo "," >> "${TEMP_LOCK_FILE}"
    fi

    # Write package entry (using printf to avoid heredoc issues)
    printf '    {\n' >> "${TEMP_LOCK_FILE}"
    printf '      "name": "%s",\n' "${PKG_NAME}" >> "${TEMP_LOCK_FILE}"
    printf '      "epoch": "%s",\n' "${PKG_EPOCH}" >> "${TEMP_LOCK_FILE}"
    printf '      "version": "%s",\n' "${PKG_VERSION}" >> "${TEMP_LOCK_FILE}"
    printf '      "release": "%s",\n' "${PKG_RELEASE}" >> "${TEMP_LOCK_FILE}"
    printf '      "arch": "%s",\n' "${PKG_ARCH}" >> "${TEMP_LOCK_FILE}"
    printf '      "sha256": "%s",\n' "${PKG_SHA256}" >> "${TEMP_LOCK_FILE}"
    printf '      "filename": "%s"\n' "${PKG_FILENAME}" >> "${TEMP_LOCK_FILE}"
    printf '    }' >> "${TEMP_LOCK_FILE}"

    PACKAGE_COUNT=$((PACKAGE_COUNT + 1))
done

# Close JSON structure
cat >> "${TEMP_LOCK_FILE}" << EOF

  ]
}
EOF

# Validate JSON and format it
if command -v jq &>/dev/null; then
    if jq . "${TEMP_LOCK_FILE}" > "${LOCK_FILE}"; then
        rm -f "${TEMP_LOCK_FILE}"
    else
        echo "ERROR: Generated invalid JSON"
        cat "${TEMP_LOCK_FILE}"
        exit 1
    fi

    # =============================================================================
    # Post-filter: Remove excluded packages from lock file
    # =============================================================================
    # Force ignore packages with their dependencies
    # We remove packages matching the exclusion patterns from the lock file

    if [ -n "${EXCLUSIONS}" ]; then
        echo "Applying post-filter exclusions: ${EXCLUSIONS}"

        # Build jq filter to exclude matching packages
        # jq syntax: select((.name | test("pattern")) | not)
        FILTER_EXPR=""
        for pattern in ${EXCLUSIONS}; do
            # Convert glob pattern to regex (basic conversion)
            regex_pattern=$(echo "${pattern}" | sed 's/\*/.*/g')
            if [ -n "${FILTER_EXPR}" ]; then
                FILTER_EXPR="${FILTER_EXPR} or"
            fi
            FILTER_EXPR="${FILTER_EXPR} (.name | test(\"^${regex_pattern}\$\"))"
        done

        # Apply filter and save - use "| not" for negation in jq
        FILTERED_FILE="${LOCK_FILE}.filtered"
        if jq ".packages |= map(select((${FILTER_EXPR}) | not))" "${LOCK_FILE}" > "${FILTERED_FILE}"; then
            mv "${FILTERED_FILE}" "${LOCK_FILE}"
            # Update package count
            PACKAGE_COUNT=$(jq -r '.packages | length' "${LOCK_FILE}")
            echo "After filtering: ${PACKAGE_COUNT} packages"
        else
            echo "WARNING: Post-filter failed, keeping unfiltered lock file"
            rm -f "${FILTERED_FILE}"
        fi

        # Also remove filtered packages from cache to save space
        for pattern in ${EXCLUSIONS}; do
            rm -f "${CACHE_DIR}/${ARCH}/${PACKAGE_SET}"/${pattern}.rpm 2>/dev/null || true
        done
    fi
else
    mv "${TEMP_LOCK_FILE}" "${LOCK_FILE}"
fi

# =============================================================================
# Summary
# =============================================================================

echo ""
echo "=============================================="
echo "RPM Freeze Complete"
echo "=============================================="
echo "Lock file: ${LOCK_FILE}"
echo "Cache dir: ${CACHE_DIR}/${ARCH}/${PACKAGE_SET}"
echo "Packages:  ${PACKAGE_COUNT}"
echo ""

# List packages
if command -v jq &>/dev/null; then
    echo "Package list:"
    # Use || true to prevent SIGPIPE from head causing script failure with pipefail
    jq -r '.packages[] | "  - \(.name)-\(.epoch):\(.version)-\(.release).\(.arch)"' "${LOCK_FILE}" | head -20 || true
    if [ "${PACKAGE_COUNT}" -gt 20 ]; then
        echo "  ... and $((PACKAGE_COUNT - 20)) more"
    fi
fi

# =============================================================================
# Cleanup
# =============================================================================

rm -rf "${INSTALLROOT}"
rm -f "${DNF_CONF}"

echo ""
echo "Next steps:"
echo "  1. Review lock file: ${LOCK_FILE}"
echo "  2. Verify checksums: ./hack/rpm-verify.sh ${LOCK_FILE}"
echo "  3. Commit lock file to git"

show_timing

exit 0
