#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

LIBVIRT_VERSION=${LIBVIRT_VERSION:-0:7.6.0-6.el8}
QEMU_VERSION=${QEMU_VERSION:-15:6.0.0-33.el8s}
SEABIOS_VERSION=${SEABIOS_VERSION:-0:1.14.0-1.el8}
EDK2_VERSION=${EDK2_VERSION:-0:20200602gitca407c7246bf-4.el8}
LIBGUESTFS_VERSION=${LIBGUESTFS_VERSION:-1:1.44.0-3.el8s}
SINGLE_ARCH=${SINGLE_ARCH:-""}

bazeldnf_repos="--repofile repo.yaml"
if [ "${CUSTOM_REPO}" ]; then
    bazeldnf_repos="--repofile ${CUSTOM_REPO} ${bazeldnf_repos}"
fi

# Packages that we want to be included in all container images.
#
# Further down we define per-image package lists, which are just like
# this one are split into two: one for the packages that we actually
# want to have in the image, and one for (indirect) dependencies that
# have more than one way of being resolved. Listing the latter
# explicitly ensures that bazeldnf always reaches the same solution
# and thus keeps things reproducible
centos_base="
  acl
  curl
  vim-minimal
"
centos_extra="
  coreutils-single
  glibc-minimal-langpack
  libcurl-minimal
"

# get latest repo data from repo.yaml
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- fetch ${bazeldnf_repos}

# create a rpmtree for our test image with misc. tools.
testimage_base="
  device-mapper
  e2fsprogs
  iputils
  nmap-ncat
  procps-ng
  qemu-img-${QEMU_VERSION}
  util-linux
  which
"

# create a rpmtree for libvirt-devel. libvirt-devel is needed for compilation and unit-testing.
libvirtdevel_base="
  libvirt-devel-${LIBVIRT_VERSION}
"
libvirtdevel_extra="
  keyutils-libs
  krb5-libs
  libmount
  lz4-libs
"

# TODO: Remove the sssd-client and use a better sssd config
# This requires a way to inject files into the sandbox via bazeldnf
sandbox_base="
  findutils
  gcc
  glibc-static
  python36
  sssd-client
"
# create a rpmtree for virt-launcher and virt-handler. This is the OS for our node-components.
launcherbase_base="
  libvirt-client-${LIBVIRT_VERSION}
  libvirt-daemon-driver-qemu-${LIBVIRT_VERSION}
  qemu-kvm-core-${QEMU_VERSION}
"
launcherbase_x86_64="
  edk2-ovmf-${EDK2_VERSION}
  qemu-kvm-hw-usbredir-${QEMU_VERSION}
  seabios-${SEABIOS_VERSION}
"
launcherbase_aarch64="
  edk2-aarch64-${EDK2_VERSION}
"
launcherbase_extra="
  ethtool
  findutils
  iptables
  nftables
  procps-ng
  selinux-policy
  selinux-policy-targeted
  tar
  xorriso
"

handler_base="
  qemu-img-${QEMU_VERSION}
"

handlerbase_extra="
  findutils
  iproute
  iptables
  nftables
  procps-ng
  selinux-policy
  selinux-policy-targeted
  tar
  util-linux
  xorriso
"

libguestfstools_base="
  libguestfs-tools-${LIBGUESTFS_VERSION}
  libvirt-daemon-driver-qemu-${LIBVIRT_VERSION}
  qemu-kvm-core-${QEMU_VERSION}
  seabios-${SEABIOS_VERSION}
"
libguestfstools_x86_64="
  edk2-ovmf-${EDK2_VERSION}
"

if [ -z "${SINGLE_ARCH}" ] || [ "${SINGLE_ARCH}" == "x86_64" ]; then

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public \
        --name testimage_x86_64 \
        --basesystem centos-stream-release \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $testimage_base

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name libvirt-devel_x86_64 \
        --basesystem centos-stream-release \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $libvirtdevel_base \
        $libvirtdevel_extra

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name sandboxroot_x86_64 \
        --basesystem centos-stream-release \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $sandbox_base

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name launcherbase_x86_64 \
        --basesystem centos-stream-release \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $launcherbase_base \
        $launcherbase_x86_64 \
        $launcherbase_extra

    # create a rpmtree for virt-handler
    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree --public --name handlerbase_x86_64 \
        --basesystem centos-stream-release \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $handler_base \
        $handlerbase_extra

    bazel run \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name libguestfs-tools \
        --basesystem centos-stream-release \
        $centos_base \
        $centos_extra \
        $libguestfstools_base \
        $libguestfstools_x86_64 \
        ${bazeldnf_repos} \
        --force-ignore-with-dependencies '^(kernel-|linux-firmware)' \
        --force-ignore-with-dependencies '^(python[3]{0,1}-|perl[3]{0,1}-)' \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies '^(libvirt-daemon-kvm|swtpm)' \
        --force-ignore-with-dependencies '^(man-db|mandoc)' \
        --force-ignore-with-dependencies '^dbus'

    # remove all RPMs which are no longer referenced by a rpmtree
    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- prune

    # update tar2files targets which act as an adapter between rpms
    # and cc_library which we need for virt-launcher and virt-handler
    bazel run \
        --config=${ARCHITECTURE} \
        //rpm:ldd_x86_64

    # regenerate sandboxes
    rm ${SANDBOX_DIR} -rf
    kubevirt::bootstrap::regenerate x86_64
fi

if [ -z "${SINGLE_ARCH}" ] || [ "${SINGLE_ARCH}" == "aarch64" ]; then

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public \
        --name testimage_aarch64 --arch aarch64 \
        --basesystem centos-stream-release \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $testimage_base

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name libvirt-devel_aarch64 --arch aarch64 \
        --basesystem centos-stream-release \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $libvirtdevel_base \
        $libvirtdevel_extra

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name sandboxroot_aarch64 --arch aarch64 \
        --basesystem centos-stream-release \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $sandbox_base

    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree \
        --public --nobest \
        --name launcherbase_aarch64 --arch aarch64 \
        --basesystem centos-stream-release \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $launcherbase_base \
        $launcherbase_aarch64 \
        $launcherbase_extra

    # create a rpmtree for virt-handler
    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- rpmtree --public --arch=aarch64 --name handlerbase_aarch64 \
        --basesystem centos-stream-release \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_base \
        $centos_extra \
        $handler_base \
        $handlerbase_extra

    # remove all RPMs which are no longer referenced by a rpmtree
    bazel run \
        --config=${ARCHITECTURE} \
        //:bazeldnf -- prune

    # update tar2files targets which act as an adapter between rpms
    # and cc_library which we need for virt-launcher and virt-handler
    bazel run \
        --config=${ARCHITECTURE} \
        //rpm:ldd_aarch64

    # regenerate sandboxes
    rm ${SANDBOX_DIR} -rf
    kubevirt::bootstrap::regenerate aarch64
fi
