#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh

LIBVIRT_VERSION=0:7.0.0-12
SEABIOS_VERSION=0:1.14.0-1
QEMU_VERSION=15:5.2.0-15

# Define some base packages to avoid dependency flipping
# since some dependencies can be satisfied by multiple packages
basesystem="glibc-langpack-en coreutils-single libcurl-minimal curl-minimal fedora-logos-httpd vim-minimal"

# get latest repo data from repo.yaml
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- fetch

# create a rpmtree for our test image with misc. tools.
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --name testimage_x86_64 \
    $basesystem \
    qemu-img \
    which \
    procps-ng \
    nmap-ncat \
    iputils \
    e2fsprogs \
    device-mapper \
    util-linux

bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --arch=aarch64 --name testimage_aarch64 \
    $basesystem \
    qemu-img \
    which \
    procps-ng \
    nmap-ncat \
    iputils \
    e2fsprogs \
    device-mapper \
    util-linux

# create a rpmtree for libvirt-devel. libvirt-devel is needed for compilation and unit-testing.
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --name libvirt-devel_x86_64 \
    $basesystem \
    libvirt-devel-${LIBVIRT_VERSION} \
    keyutils-libs \
    krb5-libs \
    libmount \
    lz4-libs

bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --arch=aarch64 --name libvirt-devel_aarch64 \
    $basesystem \
    libvirt-devel-${LIBVIRT_VERSION} \
    keyutils-libs \
    krb5-libs \
    libmount \
    lz4-libs

# create a rpmtree for virt-launcher.
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --name launcherbase_x86_64 \
    --force-ignore-with-dependencies '^mozjs60' \
    $basesystem \
    libvirt-daemon-driver-qemu-${LIBVIRT_VERSION} \
    libvirt-client-${LIBVIRT_VERSION} \
    qemu-kvm-core-${QEMU_VERSION} \
    seabios-${SEABIOS_VERSION} \
    xorriso \
    selinux-policy selinux-policy-targeted \
    nftables \
    findutils \
    procps-ng \
    iptables \
    tar

bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --arch=aarch64 --name launcherbase_aarch64 \
    --force-ignore-with-dependencies '^mozjs60' \
    $basesystem \
    libvirt-daemon-driver-qemu-${LIBVIRT_VERSION} \
    libvirt-client-${LIBVIRT_VERSION} \
    qemu-kvm-core-${QEMU_VERSION} \
    xorriso \
    selinux-policy selinux-policy-targeted \
    nftables \
    findutils \
    procps-ng \
    iptables \
    tar

# create a rpmtree for virt-handler
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --arch=aarch64 --name handlerbase_aarch64 \
    $basesystem \
    qemu-img-${QEMU_VERSION} \
    xorriso \
    nftables \
    findutils \
    procps-ng \
    iptables \
    iproute \
    util-linux \
    tar

bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --name handlerbase_x86_64 \
    $basesystem \
    qemu-img-${QEMU_VERSION} \
    xorriso \
    nftables \
    findutils \
    procps-ng \
    iptables \
    iproute \
    util-linux \
    tar

bazel run \
    //:bazeldnf -- rpmtree --public --name libguestfs-tools \
    $basesystem \
    libguestfs \
    libguestfs-tools \
    --force-ignore-with-dependencies '^(kernel-|linux-firmware)' \
    --force-ignore-with-dependencies '^(python[3]{0,1}-|perl[3]{0,1}-)' \
    --force-ignore-with-dependencies '^(mesa-|libwayland-|selinux-policy|mozjs60)' \
    --force-ignore-with-dependencies '^(libvirt-daemon-driver-storage|swtpm)' \
    --force-ignore-with-dependencies '^(man-db|mandoc)' \
    --force-ignore-with-dependencies '^(dbus|glusterfs|libX11|qemu-kvm-block|trousers|usbredir)' \
    --force-ignore-with-dependencies '^(gstreamer1|kbd|libX)'

# remove all RPMs which are no longer referenced by a rpmtree
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- prune

# FIXME: For an unknown reason the run target afterwards can get
# out dated tar files, build them explicitly first.
bazel build \
    --config=${ARCHITECTURE} \
    //rpm:libvirt-devel_x86_64

bazel build \
    --config=${ARCHITECTURE} \
    //rpm:libvirt-devel_aarch64
# update tar2files targets which act as an adapter between rpms
# and cc_library which we need for virt-launcher and virt-handler
bazel run \
    --config=${ARCHITECTURE} \
    //rpm:ldd_x86_64

bazel run \
    --config=${ARCHITECTURE} \
    //rpm:ldd_aarch64
