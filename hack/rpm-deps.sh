#!/usr/bin/env bash

set -ex

source hack/common.sh
source hack/config.sh

basesystem="glibc-langpack-en coreutils-single libcurl-minimal curl-minimal"

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

# remove all RPMs which are no longer referenced by a rpmtree
bazel run \
    --config=${ARCHITECTURE} \
    //:bazeldnf -- prune
