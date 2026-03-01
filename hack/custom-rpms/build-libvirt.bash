#!/usr/bin/env bash
# Compile and create the rpms, used by hack/custom-rpms.bash
# THIS SCRIPT IS NOT DESIGNED TO BE CALLED DIRECTLY

# Libvirt source is copied into the container at /build/libvirt
# Container is started with CWD /build
# Contents of /build/libvirt/build is copied back out of container after build, so should contain any build artifacts needed on the host
set -e
cd libvirt
git status &> /dev/null     # For some reason the git repo is marked as dirty when it isn't, running status fixes it!?
meson setup build -Dsystem=true -Ddriver_qemu=enabled       # Options recommended by libvirt build docs
ninja -C build dist                                         
rpmbuild -ta    /build/libvirt/build/meson-dist/libvirt-*.tar.xz  
# Create repomd.xml
createrepo_c -v --general-compress-type gz /root/rpmbuild/RPMS/x86_64   # Force the compress type. bazeldnf doesn't support zstd at time of writing
# Write the libvirt build version to file
meson introspect build --projectinfo | jq -r ".version" > /build/libvirt/build/version.txt