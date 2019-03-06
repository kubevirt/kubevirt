#!/usr/bin/env bash

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/github.com/libvirt/libvirt-go/BUILD.bazel

# generate BUILD files
bazel run //:gazelle

# inject changes to libvirt BUILD file
bazel run --run_under='cd /root/go/src/kubevirt.io/kubevirt &&' -- @com_github_bazelbuild_buildtools//buildozer 'add cdeps //:libvirt-libs //:libvirt-headers' //vendor/github.com/libvirt/libvirt-go:go_default_library
bazel run --run_under='cd /root/go/src/kubevirt.io/kubevirt &&' -- @com_github_bazelbuild_buildtools//buildozer 'add copts -Ibazel-out/k8-fastbuild/genfiles' //vendor/github.com/libvirt/libvirt-go:go_default_library

# allign BAZEL files to a single format
bazel run //:buildifier
