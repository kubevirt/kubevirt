#!/usr/bin/env bash

source hack/common.sh
source hack/config.sh

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/libvirt.org/libvirt-go/BUILD.bazel

# generate BUILD files
bazel run \
    --config=${ARCHITECTURE} \
    //:gazelle -- -exclude vendor/google.golang.org/grpc --exclude cluster-up

# inject changes to libvirt BUILD file
bazel run \
    --config=${ARCHITECTURE} \
    -- @com_github_bazelbuild_buildtools//buildozer 'add cdeps //:libvirt-libs //:libvirt-headers' //vendor/libvirt.org/libvirt-go:go_default_library
bazel run \
    --config=${ARCHITECTURE} \
    -- @com_github_bazelbuild_buildtools//buildozer 'add copts -Ibazel-out/k8-fastbuild/genfiles' //vendor/libvirt.org/libvirt-go:go_default_library
bazel run \
    --config=${ARCHITECTURE} \
    -- @com_github_bazelbuild_buildtools//buildozer 'add copts -Ibazel-out/k8-fastbuild/genfiles' //vendor/libvirt.org/libvirt-go:go_default_library
# allign BAZEL files to a single format
bazel run \
    --config=${ARCHITECTURE} \
    //:buildifier
