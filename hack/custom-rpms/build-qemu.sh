#!/usr/bin/env bash
# Compile and create the rpms, used by hack/custom-rpms.bash
# THIS SCRIPT IS NOT DESIGNED TO BE CALLED DIRECTLY

# QEMU source is copied into the container at /build/qemu
# Container is started with CWD /build
# Contents of /build/output is copied back out of container after build, so should contain any build artifacts needed on the host

set -e

get_qemu_kvm_version_info() {
    # Extract the version and epoch info from qemu-kvm.spec
    SPEC_FILE=$1
    VERSION=$(grep -oP "Version:\s+\\K(.*)" ${SPEC_FILE})
    EPOCH=$(grep -oP "Epoch:\s+\\K(.*)" ${SPEC_FILE})
    RELEASE=$(grep -oP "Release:\s+\\K(\d+)" ${SPEC_FILE})

    echo "$EPOCH:$VERSION-$RELEASE" > /build/output/version.txt
} 

get_qemu_kvm_version_info /build/qemu-kvm/qemu-kvm.spec

QEMU_NAME="qemu-${VERSION}"

# Copy all sources needed by rpmbuild into /root/rpmbuild/SOURCES and the spec file into /root/rpmbuild/SPECS
# QEMU source must be in a tar archive by the name of qemu-${VERSION}.tar.xz where VERSION matches the version field specified in the qemu-kvm.spec file
# The tar must also extract to a folder of the exact same name (qemu-${VERSION})
rpmdev-setuptree
rm -rf ${QEMU_NAME}
mv qemu ${QEMU_NAME}
tar --exclude build -cf /build/${QEMU_NAME}.tar.xz ${QEMU_NAME}
cp /build/qemu-*.tar.xz /root/rpmbuild/SOURCES
cp /build/qemu-kvm/* /root/rpmbuild/SOURCES
cp /build/qemu-kvm/qemu-kvm.spec /root/rpmbuild/SPECS

# Run the RPM build and build a repo
cd /root/rpmbuild/SPECS
rpmbuild -ba qemu-kvm.spec --nocheck --define '_smp_mflags -j12'    # nocheck because QEMU tests take forever
createrepo_c -v --general-compress-type gz /root/rpmbuild/RPMS/x86_64   # Force the compress type. bazeldnf doesn't support zstd at time of writing
