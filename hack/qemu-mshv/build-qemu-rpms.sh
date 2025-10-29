#!/bin/bash

while getopts r:v: flag; do
    case "${flag}" in
    r) QEMU_REPO=${OPTARG} ;;
    v) QEMU_VERSION=${OPTARG} ;;
    *)
        echo "Invalid option"
        exit 1
        ;;
    esac
done

if [ -z "$QEMU_REPO" ] || [ -z "$QEMU_VERSION" ]; then
    echo "Usage: $0 -r <QEMU_REPO> -v <QEMU_VERSION>"
    exit 1
fi

git clone ${QEMU_REPO} qemu-src

cd qemu-src/

sed -i "s/Version:.*$/Version: ${QEMU_VERSION}/" qemu.spec

curl -L ${QEMU_REPO}/archive/refs/tags/v${QEMU_VERSION}.tar.gz -o qemu-${QEMU_VERSION}.tar.xz

docker rm -f qemu-build

docker run -td \
    --name qemu-build \
    -v $(pwd):/qemu-src \
    registry.gitlab.com/libvirt/libvirt/ci-centos-stream-9

# Build qemu RPM
docker exec -w /qemu-src qemu-build bash -c "
  set -ex
  mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
  cp qemu.spec ~/rpmbuild/SPECS
  cp qemu-${QEMU_VERSION}.tar.xz ~/rpmbuild/SOURCES/
  cd ~/rpmbuild/SPECS
  dnf update -y
  dnf -y install createrepo
  dnf builddep -y qemu.spec
  rpmbuild -ba qemu.spec
  cd ~/rpmbuild/RPMS
  createrepo --general-compress-type=gz --checksum=sha256 x86_64
"

cd ../

docker cp qemu-build:/root/rpmbuild/RPMS ./rpms-qemu

cat >./rpms-qemu/build-info.json <<EOF
{
  "qemu_version": "0:${QEMU_VERSION}-1.el9"
}
EOF
