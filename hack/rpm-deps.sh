#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh

LIBVIRT_VERSION=0:6.6.0-8
SEABIOS_VERSION=0:1.14.0-1
QEMU_VERSION=15:5.1.0-16

# Define some base packages to avoid dependency flipping
# since some dependencies can be satisfied by multiple packages
basesystem="glibc-langpack-en coreutils-single libcurl-minimal curl-minimal fedora-logos-httpd"

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
    qemu-guest-agent \
    stress \
    dmidecode \
    virt-what \
    which \
    nginx \
    scsi-target-utils \
    procps-ng \
    nmap-ncat \
    iputils \
    e2fsprogs

bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --arch=ppc64le --name testimage_ppc64le \
    $basesystem \
    qemu-img \
    qemu-guest-agent \
    stress \
    nginx \
    scsi-target-utils \
    procps-ng \
    nmap-ncat \
    iputils \
    e2fsprogs

# create a rpmtree for libvirt-devel. libvirt-devel is needed for compilation and unit-testing.
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --name libvirt-devel_x86_64 $basesystem libvirt-devel-${LIBVIRT_VERSION}

# create a rpmtree for virt-launcher and virt-handler. This is the OS for our node-components.
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- rpmtree --public --name launcherbase_x86_64 \
    $basesystem \
    libverto-libev \
    libvirt-daemon-driver-qemu-${LIBVIRT_VERSION} \
    libvirt-client-${LIBVIRT_VERSION} \
    libvirt-daemon-driver-storage-core-${LIBVIRT_VERSION} \
    qemu-kvm-${QEMU_VERSION} \
    seabios-${SEABIOS_VERSION} \
    genisoimage \
    selinux-policy selinux-policy-targeted \
    nftables \
    findutils \
    procps-ng \
    iptables

# remove all RPMs which are no longer referenced by a rpmtree
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- prune

# update tar2files targets which act as an adapter between rpms
# and cc_library which we need for virt-launcher and virt-handler
bazel run \
    --config=${ARCHITECTURE} \
    //rpm:ldd
