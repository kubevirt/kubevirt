#!/usr/bin/env bash
#
# Generate Containerfile from RPM lock file
#
# Usage: ./hack/containerfile-from-lock.sh <lock_file> [output_containerfile]
#
# This generates a multi-stage Containerfile that:
# 1. Installs RPMs from the lock file in a CentOS Stream 9 builder stage
# 2. Creates a minimal layer with only the necessary files
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# =============================================================================
# Configuration
# =============================================================================

LOCK_FILE=${1:-}
OUTPUT_FILE=${2:-}

if [ -z "${LOCK_FILE}" ] || [ ! -f "${LOCK_FILE}" ]; then
    echo "Usage: $0 <lock_file> [output_containerfile]"
    echo ""
    echo "Examples:"
    echo "  $0 rpm-lockfiles/launcherbase-x86_64.lock.json"
    echo "  $0 rpm-lockfiles/handlerbase-aarch64.lock.json build/Containerfile.rpms"
    exit 1
fi

# Check for jq
if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required but not installed"
    exit 1
fi

# Extract metadata from lock file
ARCH=$(jq -r '.architecture' "${LOCK_FILE}")
PACKAGE_SET=$(jq -r '.package_set' "${LOCK_FILE}")
GENERATED=$(jq -r '.generated' "${LOCK_FILE}")
PACKAGE_COUNT=$(jq -r '.packages | length' "${LOCK_FILE}")

# Default output file
if [ -z "${OUTPUT_FILE}" ]; then
    OUTPUT_FILE="Containerfile.${PACKAGE_SET}-${ARCH}"
fi

echo "Generating Containerfile from: ${LOCK_FILE}"
echo "  Architecture: ${ARCH}"
echo "  Package set:  ${PACKAGE_SET}"
echo "  Packages:     ${PACKAGE_COUNT}"
echo "  Output:       ${OUTPUT_FILE}"
echo ""

# =============================================================================
# File Lists per Package Set
# =============================================================================
# These define which files to copy from the RPM installer stage
# This is more selective than copying entire directories

get_copy_paths() {
    local pkg_set=$1

    case "${pkg_set}" in
        launcherbase)
            cat << 'EOF'
# QEMU binaries and libraries
/usr/bin/qemu-img
/usr/bin/qemu-pr-helper
/usr/libexec/qemu-kvm
/usr/share/qemu-kvm/
/usr/lib64/qemu-kvm/

# Libvirt
/usr/bin/virsh
/usr/bin/virt-qemu-run
/usr/sbin/virtqemud
/usr/sbin/virtlogd
/usr/lib64/libvirt*.so*
/usr/lib64/libvirt/
/etc/libvirt/

# Firmware
/usr/share/seabios/
/usr/share/OVMF/
/usr/share/edk2/
/usr/share/AAVMF/

# SWTPM
/usr/bin/swtpm*
/usr/lib64/swtpm/

# Passt
/usr/bin/passt
/usr/bin/pasta

# Virtiofsd
/usr/libexec/virtiofsd

# Utilities
/usr/bin/tar
/usr/bin/xorriso
/usr/bin/ncat
/usr/bin/nft
/usr/sbin/nft
/usr/bin/find
/usr/bin/ps
/usr/bin/pgrep

# SELinux policy
/etc/selinux/
/usr/share/selinux/
EOF
            ;;
        handlerbase)
            cat << 'EOF'
# QEMU image tool
/usr/bin/qemu-img

# Utilities
/usr/bin/tar
/usr/bin/xorriso
/usr/bin/nft
/usr/sbin/nft
/usr/bin/find
/usr/bin/ps
/usr/bin/ip
/usr/sbin/ip

# SELinux policy
/etc/selinux/
/usr/share/selinux/
EOF
            ;;
        exportserverbase)
            cat << 'EOF'
/usr/bin/tar
/usr/bin/curl
EOF
            ;;
        libguestfs-tools)
            cat << 'EOF'
# Libguestfs
/usr/bin/guestfish
/usr/bin/guestmount
/usr/bin/virt-*
/usr/lib64/guestfs/
/usr/share/guestfs/

# QEMU for libguestfs
/usr/libexec/qemu-kvm
/usr/share/qemu-kvm/
/usr/lib64/qemu-kvm/

# Libvirt
/usr/sbin/virtqemud
/usr/lib64/libvirt*.so*

# Firmware
/usr/share/seabios/
/usr/share/OVMF/
/usr/share/edk2/

# SELinux
/etc/selinux/
/usr/share/selinux/
EOF
            ;;
        testimage)
            cat << 'EOF'
/usr/bin/qemu-img
/usr/bin/tar
/usr/bin/ncat
/usr/bin/ping
/usr/bin/ps
/usr/sbin/targetcli
/usr/bin/e2fsck
/usr/sbin/mkfs.ext4
EOF
            ;;
        *)
            # Default: copy common utilities
            cat << 'EOF'
/usr/bin/tar
/usr/bin/curl
EOF
            ;;
    esac
}

# =============================================================================
# Generate Containerfile
# =============================================================================

# Map architecture to container platform
case "${ARCH}" in
    x86_64)  PLATFORM_ARCH="amd64" ;;
    aarch64) PLATFORM_ARCH="arm64" ;;
    s390x)   PLATFORM_ARCH="s390x" ;;
    *)       PLATFORM_ARCH="${ARCH}" ;;
esac

cat > "${OUTPUT_FILE}" << EOF
# Generated Containerfile for ${PACKAGE_SET} (${ARCH})
# Source: ${LOCK_FILE}
# Generated: $(date -u +%Y-%m-%dT%H:%M:%SZ)
# Original lock file generated: ${GENERATED}
#
# Build: podman build --platform linux/${PLATFORM_ARCH} -f ${OUTPUT_FILE} .
#

# =============================================================================
# Stage 1: RPM Installer
# =============================================================================
FROM quay.io/centos/centos:stream9 AS rpm-installer

# Install exact package versions from lock file
RUN dnf install -y --setopt=install_weak_deps=False \\
EOF

# Add each package from the lock file
jq -r '.packages[] | "    \(.name)-\(.epoch):\(.version)-\(.release).\(.arch)"' "${LOCK_FILE}" | \
    sed 's/:0:/:/' | \
    while IFS= read -r pkg; do
        echo "    ${pkg} \\"
    done >> "${OUTPUT_FILE}"

cat >> "${OUTPUT_FILE}" << 'EOF'
    && dnf clean all \
    && rm -rf /var/cache/dnf

# Create package manifest for verification
RUN rpm -qa --queryformat '%{NAME}-%{EPOCH}:%{VERSION}-%{RELEASE}.%{ARCH}\n' | \
    sed 's/:(none):/:0:/' | sort > /tmp/installed-packages.txt

EOF

cat >> "${OUTPUT_FILE}" << EOF
# =============================================================================
# Stage 2: Minimal RPM Layer
# =============================================================================
# This stage extracts only the necessary files for the final image
FROM rpm-installer AS rpm-layer

# Create directory structure
RUN mkdir -p /rpm-root/usr/bin /rpm-root/usr/sbin /rpm-root/usr/lib64 \\
    /rpm-root/usr/libexec /rpm-root/usr/share /rpm-root/etc

# Copy only required files (selective, not entire directories)
EOF

# Add copy commands for this package set
get_copy_paths "${PACKAGE_SET}" | grep -v '^#' | grep -v '^$' | while read -r path; do
    # Handle wildcards and directories
    if [[ "${path}" == */ ]]; then
        # Directory - copy if exists
        echo "RUN if [ -d \"${path}\" ]; then mkdir -p /rpm-root$(dirname ${path}) && cp -a ${path} /rpm-root${path}; fi" >> "${OUTPUT_FILE}"
    elif [[ "${path}" == *'*'* ]]; then
        # Wildcard - use shell glob
        echo "RUN for f in ${path}; do [ -e \"\$f\" ] && mkdir -p /rpm-root\$(dirname \"\$f\") && cp -a \"\$f\" /rpm-root\$(dirname \"\$f\")/; done 2>/dev/null || true" >> "${OUTPUT_FILE}"
    else
        # Single file
        echo "RUN if [ -e \"${path}\" ]; then mkdir -p /rpm-root$(dirname ${path}) && cp -a ${path} /rpm-root${path}; fi" >> "${OUTPUT_FILE}"
    fi
done

cat >> "${OUTPUT_FILE}" << EOF

# Copy shared libraries that binaries depend on
RUN for bin in /rpm-root/usr/bin/* /rpm-root/usr/sbin/* /rpm-root/usr/libexec/*; do \\
        [ -f "\$bin" ] && [ -x "\$bin" ] && \\
        ldd "\$bin" 2>/dev/null | grep "=> /" | awk '{print \$3}' | \\
        while read lib; do \\
            [ -f "\$lib" ] && mkdir -p /rpm-root\$(dirname "\$lib") && \\
            cp -an "\$lib" /rpm-root"\$lib" 2>/dev/null || true; \\
        done; \\
    done || true

# Copy ld-linux for dynamic linking
RUN cp -a /lib64/ld-linux-*.so* /rpm-root/lib64/ 2>/dev/null || \\
    cp -a /lib/ld-linux-*.so* /rpm-root/lib/ 2>/dev/null || true

# =============================================================================
# Stage 3: Final Image (use as base for application images)
# =============================================================================
FROM scratch AS ${PACKAGE_SET}-${ARCH}

COPY --from=rpm-layer /rpm-root/ /

# Labels
LABEL org.opencontainers.image.description="RPM base layer for ${PACKAGE_SET} (${ARCH})"
LABEL org.opencontainers.image.architecture="${ARCH}"
LABEL org.kubevirt.rpm-lockfile="${LOCK_FILE}"
EOF

echo "" >> "${OUTPUT_FILE}"

# =============================================================================
# Summary
# =============================================================================

echo ""
echo "Containerfile generated: ${OUTPUT_FILE}"
echo ""
echo "Build commands:"
echo "  podman build --platform linux/${PLATFORM_ARCH} --target ${PACKAGE_SET}-${ARCH} \\"
echo "    -t kubevirt-rpms-${PACKAGE_SET}:${ARCH} -f ${OUTPUT_FILE} ."
echo ""
echo "Usage in application Containerfile:"
echo "  FROM kubevirt-rpms-${PACKAGE_SET}:${ARCH} AS rpm-base"
echo "  FROM gcr.io/distroless/base-debian12:nonroot"
echo "  COPY --from=rpm-base / /"
echo "  COPY virt-launcher /usr/bin/"
