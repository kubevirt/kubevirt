#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh
source hack/install-bazeldnf.sh

# Runs bazeldnf ldd without Bazel by downloading RPMs listed in a
# rpmtree rule, combining them into a tar, and invoking ldd on it.
#   bazeldnf_ldd <rpmtree_name> <rulename> <lib1> [lib2 ...]
function bazeldnf_ldd() {
    local rpmtree_name="$1"
    local rulename="$2"
    shift 2
    local libs=("$@")

    local buildfile="rpm/BUILD.bazel"
    local workspace="WORKSPACE"
    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf ${tmpdir}" RETURN

    # Extract RPM names from the rpmtree rule in BUILD.bazel.
    # Lines look like: "@acl-0__2.3.1-4.el9.x86_64//rpm",
    local rpm_names
    rpm_names=$(sed -n "/name = \"${rpmtree_name}\"/,/^)/{ s/.*\"@\(.*\)\/\/rpm\".*/\1/p }" "${buildfile}")

    local rpm_inputs=()
    for rpm_name in ${rpm_names}; do
        # Get the first URL for this RPM from WORKSPACE.
        # rpm() entries have: urls = [ "http://...", ... ]
        local url
        url=$(sed -n "/name = \"${rpm_name}\"/,/^)/{
            /urls/,/\]/{
                s/.*\"\(http[^\"]*\)\".*/\1/p
            }
        }" "${workspace}" | head -1)

        if [ -z "${url}" ]; then
            echo "ERROR: could not find URL for RPM ${rpm_name} in ${workspace}" >&2
            return 1
        fi

        local rpm_file="${tmpdir}/${rpm_name}.rpm"
        curl -sSL -o "${rpm_file}" "${url}"
        rpm_inputs+=("-i" "${rpm_file}")
    done

    local tar_file="${tmpdir}/combined.tar"
    bazeldnf rpm2tar "${rpm_inputs[@]}" -o "${tar_file}"

    bazeldnf ldd \
        --input "${tar_file}" \
        --rpmtree "${rpmtree_name}" \
        --name "${rulename}" \
        --buildfile "${buildfile}" \
        --workspace "${workspace}" \
        "${libs[@]}"
}

# CentOS Stream version selection (default to 9)
KUBEVIRT_CENTOS_STREAM_VERSION=${KUBEVIRT_CENTOS_STREAM_VERSION:-9}
TARGET_SUFFIX="_cs${KUBEVIRT_CENTOS_STREAM_VERSION}"
CS_CONFIG="cs${KUBEVIRT_CENTOS_STREAM_VERSION}"

# Version-specific package versions
if [ "${KUBEVIRT_CENTOS_STREAM_VERSION}" = "10" ]; then
    # CS10: use unversioned packages (latest available)
    LIBVIRT_VERSION=${LIBVIRT_VERSION:-}
    QEMU_VERSION=${QEMU_VERSION:-}
    SEABIOS_VERSION=${SEABIOS_VERSION:-}
    EDK2_VERSION=${EDK2_VERSION:-}
    LIBGUESTFS_VERSION=${LIBGUESTFS_VERSION:-}
    GUESTFSTOOLS_VERSION=${GUESTFSTOOLS_VERSION:-}
    PASST_VERSION=${PASST_VERSION:-}
    VIRTIOFSD_VERSION=${VIRTIOFSD_VERSION:-}
    SWTPM_VERSION=${SWTPM_VERSION:-}
    LIBNBD_VERSION=${LIBNBD_VERSION:-}
else
    # CS9 defaults (current pinned versions)
    LIBVIRT_VERSION=${LIBVIRT_VERSION:-0:11.10.0-12.el9}
    QEMU_VERSION=${QEMU_VERSION:-17:10.1.0-20.el9}
    SEABIOS_VERSION=${SEABIOS_VERSION:-0:1.16.3-4.el9}
    EDK2_VERSION=${EDK2_VERSION:-0:20241117-8.el9}
    LIBGUESTFS_VERSION=${LIBGUESTFS_VERSION:-1:1.54.0-9.el9}
    GUESTFSTOOLS_VERSION=${GUESTFSTOOLS_VERSION:-0:1.52.2-5.el9}
    PASST_VERSION=${PASST_VERSION:-0:0^20260611.ga9c61ff-1.el9}
    VIRTIOFSD_VERSION=${VIRTIOFSD_VERSION:-0:1.13.0-1.el9}
    SWTPM_VERSION=${SWTPM_VERSION:-0:0.8.0-2.el9}
    LIBNBD_VERSION=${LIBNBD_VERSION:-0:1.20.3-4.el9}
fi

SINGLE_ARCH=${SINGLE_ARCH:-""}
BASESYSTEM=${BASESYSTEM:-"centos-stream-release"}

# Shared library paths for ldd analysis (CS10 adds libfido2)
libvirt_ldd_libs="
  /usr/lib64/libvirt-admin.so.0
  /usr/lib64/libvirt-lxc.so.0
  /usr/lib64/libvirt-qemu.so.0
  /usr/lib64/libvirt.so.0
  /usr/lib64/libkrb5support.so.0
  /usr/lib64/libkeyutils.so.1
  /usr/lib64/liblz4.so.1
  /usr/lib64/libmount.so.1
"
if [ "${KUBEVIRT_CENTOS_STREAM_VERSION}" = "10" ]; then
    libvirt_ldd_libs="/usr/lib64/libfido2.so.1 ${libvirt_ldd_libs}"
fi
libnbd_ldd_libs="/usr/lib64/libnbd.so.0"

# Select repo file based on version
bazeldnf_repos="--repofile rpm/repo-cs${KUBEVIRT_CENTOS_STREAM_VERSION}.yaml"
if [ "${KUBEVIRT_CROSS_ARCH_EMULATION}" ]; then
    bazeldnf_repos="--repofile rpm/repo-virt-preview.yaml ${bazeldnf_repos}"
fi
if [ "${CUSTOM_REPO}" ]; then
    bazeldnf_repos="--repofile ${CUSTOM_REPO} ${bazeldnf_repos}"
fi

# Packages that we want to be included in all container images.
#
# Further down we define per-image package lists, which just like
# this one are split across multiple variables:
#
#   * $foo_main  => packages that we want to have in the image;
#
#   * $foo_ARCH  => same as above, but specific to one architecture;
#
#   * $foo_extra => (indirect) dependencies that can be satisfied by
#                   more than one package.
#
# Listing the "extra" packages explicitly ensures that bazeldnf will
# always reach the same solution, and thus keeps things reproducible

# Version-specific package names
if [ "${KUBEVIRT_CENTOS_STREAM_VERSION}" = "10" ]; then
    # CS10: curl-minimal was replaced by curl
    CURL_PACKAGE="curl"
else
    # CS9: uses curl-minimal
    CURL_PACKAGE="curl-minimal"
fi

centos_main="
  acl
  ${CURL_PACKAGE}
  vim-minimal
"
centos_extra="
  coreutils-single
  glibc-minimal-langpack
  libcurl-minimal
"

# create a rpmtree for our test image with misc. tools.
testimage_main="
  device-mapper
  e2fsprogs
  iputils
  nmap-ncat
  procps-ng
  qemu-img-${QEMU_VERSION}
  tar
  targetcli
  util-linux
  which
"
# sevctl is x86_64-only in CS10, but available for all architectures in CS9
testimage_x86_64="
  sevctl
"
if [ "${KUBEVIRT_CENTOS_STREAM_VERSION}" = "10" ]; then
    testimage_aarch64=""
    testimage_s390x=""
else
    testimage_aarch64="
  sevctl
"
    testimage_s390x="
  sevctl
"
fi

# create a rpmtree for libvirt-devel. libvirt-devel is needed for compilation and unit-testing.
libvirtdevel_main="
  libvirt-devel-${LIBVIRT_VERSION}
"
libvirtdevel_extra="
  keyutils-libs
  krb5-libs
  libmount
  lz4-libs
"

# create a rpmtree for libnbd-devel.
libnbddevel_main="
  libnbd-devel-${LIBNBD_VERSION}
"

# TODO: Remove the sssd-client and use a better sssd config
# This requires a way to inject files into the sandbox via bazeldnf
sandboxroot_main="
  findutils
  gcc
  glibc-static
  python3
  sssd-client
"

# create a rpmtree for virt-launcher and virt-handler. This is the OS for our node-components.
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
if [ "${KUBEVIRT_CROSS_ARCH_EMULATION}" ]; then
    launcherbase_x86_64+="
  qemu-system-aarch64-core
  edk2-aarch64
"
fi
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
  libnbd-${LIBNBD_VERSION}
"

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

libguestfstools_main="
  libguestfs-${LIBGUESTFS_VERSION}
  guestfs-tools-${GUESTFSTOOLS_VERSION}
  libvirt-daemon-driver-qemu-${LIBVIRT_VERSION}
  qemu-kvm-core-${QEMU_VERSION}
"
libguestfstools_x86_64="
  edk2-ovmf-${EDK2_VERSION}
  seabios-${SEABIOS_VERSION}
"

libguestfstools_s390x="
  edk2-ovmf-${EDK2_VERSION}
"
libguestfstools_extra="
  selinux-policy
  selinux-policy-targeted
"

exportserverbase_main="
  tar
"

pr_helper="
  qemu-pr-helper
"

sidecar_shim="
    python3
"

# get latest repo data from repo.yaml
bazeldnf fetch \
    ${bazeldnf_repos}

if [ -z "${SINGLE_ARCH}" ] || [ "${SINGLE_ARCH}" == "x86_64" ]; then

    bazeldnf rpmtree \
        --public --nobest \
        --name testimage_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $testimage_main \
        $testimage_x86_64

    bazeldnf rpmtree \
        --public --nobest \
        --name libvirt-devel_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $libvirtdevel_main \
        $libvirtdevel_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name libnbd-devel_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $libnbddevel_main

    bazeldnf rpmtree \
        --public --nobest \
        --name sandboxroot_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sandboxroot_main

    bazeldnf rpmtree \
        --public --nobest \
        --name launcherbase_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $launcherbase_main \
        $launcherbase_x86_64 \
        $launcherbase_extra

    # bazeldnf resolves noarch RPMs (like edk2-aarch64) under the
    # architecture of the repo that provided them, so edk2-aarch64
    # ends up with an .aarch64 suffix in WORKSPACE and only appears
    # in the aarch64 rpmtree. Inject it into the x86_64 launcher
    # target so the cross-arch EFI firmware is included.
    if [ "${KUBEVIRT_CROSS_ARCH_EMULATION}" ]; then
        edk2_aarch64_entry=$(grep '@edk2-aarch64-.*\.aarch64//rpm' rpm/BUILD.bazel | head -1 | xargs)
        if [ -n "${edk2_aarch64_entry}" ]; then
            sed -i "/name = \"launcherbase_x86_64${TARGET_SUFFIX}\"/,/\]/{
                /@edk2-ovmf-.*x86_64/a\\        ${edk2_aarch64_entry}
            }" rpm/BUILD.bazel
        fi
    fi

    # create a rpmtree for virt-handler
    bazeldnf rpmtree \
        --public --nobest \
        --name handlerbase_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $handlerbase_main \
        $handlerbase_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name passt_tree_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        passt-${PASST_VERSION}

    bazeldnf rpmtree \
        --public --nobest \
        --name libguestfs-tools_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        $centos_main \
        $centos_extra \
        $libguestfstools_main \
        $libguestfstools_x86_64 \
        $libguestfstools_extra \
        ${bazeldnf_repos} \
        --force-ignore-with-dependencies '^(kernel-|linux-firmware)' \
        --force-ignore-with-dependencies '^(python[3]{0,1}-)' \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies '^(libvirt-daemon-kvm|swtpm)' \
        --force-ignore-with-dependencies '^(man-db|mandoc)' \
        --force-ignore-with-dependencies '^dbus'

    bazeldnf rpmtree \
        --public --nobest \
        --name exportserverbase_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $exportserverbase_main

    bazeldnf rpmtree \
        --public --nobest \
        --name pr-helper_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $pr_helper

    bazeldnf rpmtree \
        --public --nobest \
        --name sidecar-shim_x86_64${TARGET_SUFFIX} \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sidecar_shim

    # remove all RPMs which are no longer referenced by a rpmtree
    bazeldnf prune

    # update tar2files targets which act as an adapter between rpms
    # and cc_library which we need for virt-launcher and virt-handler
    bazeldnf_ldd libvirt-devel_x86_64${TARGET_SUFFIX} libvirt-libs_x86_64${TARGET_SUFFIX} \
        ${libvirt_ldd_libs}

    bazeldnf_ldd libnbd-devel_x86_64${TARGET_SUFFIX} libnbd-libs_x86_64${TARGET_SUFFIX} \
        ${libnbd_ldd_libs}

    # Note: sandbox regeneration is done separately after all targets are generated
    # by calling hack/regenerate-sandboxes.sh
fi

if [ -z "${SINGLE_ARCH}" ] || [ "${SINGLE_ARCH}" == "aarch64" ]; then

    bazeldnf rpmtree \
        --public --nobest \
        --name testimage_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $testimage_main \
        $testimage_aarch64

    bazeldnf rpmtree \
        --public --nobest \
        --name libvirt-devel_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $libvirtdevel_main \
        $libvirtdevel_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name libnbd-devel_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $libnbddevel_main

    bazeldnf rpmtree \
        --public --nobest \
        --name sandboxroot_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sandboxroot_main

    bazeldnf rpmtree \
        --public --nobest \
        --name passt_tree_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        passt-${PASST_VERSION}

    bazeldnf rpmtree \
        --public --nobest \
        --name launcherbase_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $launcherbase_main \
        $launcherbase_aarch64 \
        $launcherbase_extra

    if [ "${KUBEVIRT_CROSS_ARCH_EMULATION}" ]; then
        bazeldnf rpmtree \
            --public --nobest \
            --name launcherbase_crossarch_aarch64${TARGET_SUFFIX} \
            --basesystem ${BASESYSTEM} \
            --force-ignore-with-dependencies '^mozjs60' \
            --force-ignore-with-dependencies 'python' \
            ${bazeldnf_repos} \
            qemu-system-x86-core \
            edk2-ovmf \
            seabios
    fi

    # create a rpmtree for virt-handler
    bazeldnf rpmtree \
        --public --nobest \
        --name handlerbase_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $handlerbase_main \
        $handlerbase_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name exportserverbase_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $exportserverbase_main

    bazeldnf rpmtree \
        --public --nobest \
        --name pr-helper_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $pr_helper

    bazeldnf rpmtree \
        --public --nobest \
        --name sidecar-shim_aarch64${TARGET_SUFFIX} --arch aarch64 \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sidecar_shim

    # remove all RPMs which are no longer referenced by a rpmtree
    bazeldnf prune

    # update tar2files targets which act as an adapter between rpms
    # and cc_library which we need for virt-launcher and virt-handler
    bazeldnf_ldd libvirt-devel_aarch64${TARGET_SUFFIX} libvirt-libs_aarch64${TARGET_SUFFIX} \
        ${libvirt_ldd_libs}

    bazeldnf_ldd libnbd-devel_aarch64${TARGET_SUFFIX} libnbd-libs_aarch64${TARGET_SUFFIX} \
        ${libnbd_ldd_libs}

    # Note: sandbox regeneration is done separately
fi

if [ -z "${SINGLE_ARCH}" ] || [ "${SINGLE_ARCH}" == "s390x" ]; then

    bazeldnf rpmtree \
        --public --nobest \
        --name testimage_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $testimage_main \
        $testimage_s390x

    bazeldnf rpmtree \
        --public --nobest \
        --name libvirt-devel_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $libvirtdevel_main \
        $libvirtdevel_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name libnbd-devel_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $libnbddevel_main

    bazeldnf rpmtree \
        --public --nobest \
        --name sandboxroot_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sandboxroot_main

    bazeldnf rpmtree \
        --public --nobest \
        --name launcherbase_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $launcherbase_main \
        $launcherbase_s390x \
        $launcherbase_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name passt_tree_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        passt-${PASST_VERSION}

    # create a rpmtree for virt-handler
    bazeldnf rpmtree \
        --public --nobest \
        --name handlerbase_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        --force-ignore-with-dependencies 'python' \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $handlerbase_main \
        $handlerbase_extra

    bazeldnf rpmtree \
        --public --nobest \
        --name exportserverbase_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $exportserverbase_main

    bazeldnf rpmtree \
        --public --nobest \
        --name libguestfs-tools_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        $centos_main \
        $centos_extra \
        $libguestfstools_main \
        $libguestfstools_s390x \
        $libguestfstools_extra \
        ${bazeldnf_repos} \
        --force-ignore-with-dependencies '^(kernel-|linux-firmware)' \
        --force-ignore-with-dependencies '^(python[3]{0,1}-)' \
        --force-ignore-with-dependencies '^mozjs60' \
        --force-ignore-with-dependencies '^(libvirt-daemon-kvm|swtpm)' \
        --force-ignore-with-dependencies '^(man-db|mandoc)' \
        --force-ignore-with-dependencies '^dbus'

    bazeldnf rpmtree \
        --public --nobest \
        --name sidecar-shim_s390x${TARGET_SUFFIX} --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sidecar_shim

    # remove all RPMs which are no longer referenced by a rpmtree
    bazeldnf prune

    # update tar2files targets which act as an adapter between rpms
    # and cc_library which we need for virt-launcher and virt-handler
    bazeldnf_ldd libvirt-devel_s390x${TARGET_SUFFIX} libvirt-libs_s390x${TARGET_SUFFIX} \
        ${libvirt_ldd_libs}

    bazeldnf_ldd libnbd-devel_s390x${TARGET_SUFFIX} libnbd-libs_s390x${TARGET_SUFFIX} \
        ${libnbd_ldd_libs}

    # Note: sandbox regeneration is done separately
fi

# Sandbox regeneration is triggered automatically on next bazel build
# after the BUILD.bazel hash changes
