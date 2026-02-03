#!/usr/bin/env bash
#
# Package definitions for RPM freeze - extracted from rpm-deps.sh
# This file is sourced by rpm-freeze-native.sh and other scripts
#
# Usage: source hack/rpm-packages.sh
#

# =============================================================================
# Version Pinning
# =============================================================================
# These versions should match what's currently in rpm-deps.sh
# Override via environment variables when bumping versions

LIBVIRT_VERSION=${LIBVIRT_VERSION:-0:11.9.0-1.el9}
QEMU_VERSION=${QEMU_VERSION:-17:10.1.0-10.el9}
SEABIOS_VERSION=${SEABIOS_VERSION:-0:1.16.3-4.el9}
EDK2_VERSION=${EDK2_VERSION:-0:20241117-8.el9}
LIBGUESTFS_VERSION=${LIBGUESTFS_VERSION:-1:1.54.0-9.el9}
GUESTFSTOOLS_VERSION=${GUESTFSTOOLS_VERSION:-0:1.52.2-5.el9}
PASST_VERSION=${PASST_VERSION:-0:0^20250512.g8ec1341-2.el9}
VIRTIOFSD_VERSION=${VIRTIOFSD_VERSION:-0:1.13.0-1.el9}
SWTPM_VERSION=${SWTPM_VERSION:-0:0.8.0-2.el9}

# Base system package
BASESYSTEM=${BASESYSTEM:-centos-stream-release}

# =============================================================================
# Base Packages (included in most package sets)
# =============================================================================

centos_main="
    acl
    curl-minimal
    vim-minimal
"

centos_extra="
    coreutils-single
    glibc-minimal-langpack
    libcurl-minimal
"

# =============================================================================
# Package Set Definitions
# =============================================================================

# Test image with misc tools
testimage_main="
    device-mapper
    e2fsprogs
    iputils
    nmap-ncat
    procps-ng
    qemu-img-${QEMU_VERSION}
    sevctl
    tar
    targetcli
    util-linux
    which
"

# libvirt-devel for compilation and unit-testing
libvirtdevel_main="
    libvirt-devel-${LIBVIRT_VERSION}
"
libvirtdevel_extra="
    keyutils-libs
    krb5-libs
    libmount
    lz4-libs
"

# Sandbox root for builds
sandboxroot_main="
    findutils
    gcc
    glibc-static
    python3
    sssd-client
"
# bazeldnf resolves fips-provider-next on aarch64 (not openssl-fips-provider)
sandboxroot_aarch64="fips-provider-next"

# Launcher base for virt-launcher
launcherbase_main="
    libvirt-client-${LIBVIRT_VERSION}
    libvirt-daemon-driver-qemu-${LIBVIRT_VERSION}
    passt-${PASST_VERSION}
    qemu-kvm-core-${QEMU_VERSION}
    qemu-kvm-device-usb-host-${QEMU_VERSION}
    swtpm-tools-${SWTPM_VERSION}
"
launcherbase_x86_64="
    edk2-ovmf-${EDK2_VERSION}
    qemu-kvm-device-display-virtio-gpu-${QEMU_VERSION}
    qemu-kvm-device-display-virtio-vga-${QEMU_VERSION}
    qemu-kvm-device-display-virtio-gpu-pci-${QEMU_VERSION}
    qemu-kvm-device-usb-redirect-${QEMU_VERSION}
    seabios-${SEABIOS_VERSION}
"
launcherbase_aarch64="
    edk2-aarch64-${EDK2_VERSION}
    qemu-kvm-device-usb-redirect-${QEMU_VERSION}
    qemu-kvm-device-display-virtio-gpu-${QEMU_VERSION}
    qemu-kvm-device-display-virtio-gpu-pci-${QEMU_VERSION}
"
launcherbase_s390x="
    qemu-kvm-device-display-virtio-gpu-${QEMU_VERSION}
    qemu-kvm-device-display-virtio-gpu-ccw-${QEMU_VERSION}
"
launcherbase_extra="
    findutils
    nftables
    nmap-ncat
    procps-ng
    selinux-policy
    selinux-policy-targeted
    tar
    virtiofsd-${VIRTIOFSD_VERSION}
    xorriso
"

# Handler base for virt-handler
handlerbase_main="
    qemu-img-${QEMU_VERSION}
"
handlerbase_extra="
    findutils
    iproute
    nftables
    procps-ng
    selinux-policy
    selinux-policy-targeted
    tar
    util-linux
    xorriso
"
# bazeldnf resolves fips-provider-next on aarch64 (not openssl-fips-provider)
handlerbase_aarch64="fips-provider-next"

# libguestfs tools
libguestfstools_main="
    libguestfs-${LIBGUESTFS_VERSION}
    guestfs-tools-${GUESTFSTOOLS_VERSION}
    libvirt-daemon-driver-qemu-${LIBVIRT_VERSION}
    qemu-kvm-core-${QEMU_VERSION}
"
libguestfstools_x86_64="
    edk2-ovmf-${EDK2_VERSION}
    seabios-${SEABIOS_VERSION}
    fips-provider-next
"
libguestfstools_s390x="
    edk2-ovmf-${EDK2_VERSION}
"
libguestfstools_extra="
    selinux-policy
    selinux-policy-targeted
"

# Export server base
exportserverbase_main="
    tar
"

# PR helper
pr_helper_main="
    qemu-pr-helper
"
# bazeldnf resolves fips-provider-next on aarch64 (not openssl-fips-provider)
pr_helper_aarch64="fips-provider-next"

# Sidecar shim
sidecar_shim_main="
    python3
"

# Passt tree (standalone) - minimal, just passt and its deps
# Note: bazeldnf resolves glibc-langpack differently per arch
passt_tree_main="
    passt-${PASST_VERSION}
"
# bazeldnf picks glibc-langpack-XX per arch, not glibc-minimal-langpack
passt_tree_x86_64="glibc-langpack-en"
passt_tree_aarch64="glibc-langpack-el"
passt_tree_s390x="glibc-langpack-et"

# =============================================================================
# Helper Functions
# =============================================================================

# Get the list of packages for a given package set and architecture
# Usage: get_packages <package_set> <arch>
get_packages() {
    local pkg_set=$1
    local arch=$2
    local packages=""

    case "${pkg_set}" in
        testimage)
            packages="${centos_main} ${centos_extra} ${testimage_main}"
            ;;
        libvirt-devel)
            packages="${centos_main} ${centos_extra} ${libvirtdevel_main} ${libvirtdevel_extra}"
            ;;
        sandboxroot)
            packages="${centos_main} ${centos_extra} ${sandboxroot_main}"
            case "${arch}" in
                aarch64) packages="${packages} ${sandboxroot_aarch64}" ;;
            esac
            ;;
        launcherbase)
            packages="${centos_main} ${centos_extra} ${launcherbase_main} ${launcherbase_extra}"
            case "${arch}" in
                x86_64)  packages="${packages} ${launcherbase_x86_64}" ;;
                aarch64) packages="${packages} ${launcherbase_aarch64}" ;;
                s390x)   packages="${packages} ${launcherbase_s390x}" ;;
            esac
            ;;
        handlerbase)
            packages="${centos_main} ${centos_extra} ${handlerbase_main} ${handlerbase_extra}"
            case "${arch}" in
                aarch64) packages="${packages} ${handlerbase_aarch64}" ;;
            esac
            ;;
        passt_tree)
            packages="${passt_tree_main}"
            case "${arch}" in
                x86_64)  packages="${packages} ${passt_tree_x86_64}" ;;
                aarch64) packages="${packages} ${passt_tree_aarch64}" ;;
                s390x)   packages="${packages} ${passt_tree_s390x}" ;;
            esac
            ;;
        libguestfs-tools)
            packages="${centos_main} ${centos_extra} ${libguestfstools_main} ${libguestfstools_extra}"
            case "${arch}" in
                x86_64) packages="${packages} ${libguestfstools_x86_64}" ;;
                s390x)  packages="${packages} ${libguestfstools_s390x}" ;;
            esac
            ;;
        exportserverbase)
            packages="${centos_main} ${centos_extra} ${exportserverbase_main}"
            ;;
        pr-helper)
            packages="${centos_main} ${centos_extra} ${pr_helper_main}"
            case "${arch}" in
                aarch64) packages="${packages} ${pr_helper_aarch64}" ;;
            esac
            ;;
        sidecar-shim)
            packages="${centos_main} ${centos_extra} ${sidecar_shim_main}"
            ;;
        *)
            echo "ERROR: Unknown package set: ${pkg_set}" >&2
            return 1
            ;;
    esac

    # Clean up whitespace and return as single line
    echo "${packages}" | tr '\n' ' ' | tr -s ' ' | sed 's/^ *//;s/ *$//'
}

# Get exclusion patterns for dnf (equivalent to bazeldnf --force-ignore-with-dependencies)
# Usage: get_exclusions <package_set>
get_exclusions() {
    local pkg_set=$1

    case "${pkg_set}" in
        launcherbase)
            echo "mozjs60* python*"
            ;;
        handlerbase)
            echo "python*"
            ;;
        libguestfs-tools)
            echo "kernel-* linux-firmware* python3-* mozjs60* libvirt-daemon-kvm* swtpm* man-db* mandoc* dbus*"
            ;;
        *)
            echo ""
            ;;
    esac
}

# Get supported architectures for a package set
# Usage: get_architectures <package_set>
get_architectures() {
    local pkg_set=$1

    case "${pkg_set}" in
        libguestfs-tools)
            # libguestfs-tools only builds for x86_64 and s390x
            echo "x86_64 s390x"
            ;;
        pr-helper)
            # pr-helper only for x86_64 and aarch64
            echo "x86_64 aarch64"
            ;;
        *)
            # Most package sets support all three architectures
            echo "x86_64 aarch64 s390x"
            ;;
    esac
}

# Get all package set names
# Usage: get_all_package_sets
get_all_package_sets() {
    echo "testimage libvirt-devel sandboxroot launcherbase handlerbase passt_tree libguestfs-tools exportserverbase pr-helper sidecar-shim"
}
