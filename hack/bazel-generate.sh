#!/usr/bin/env bash

# first ensure this file, so that sandbox bootstrapping has a working nogo setup
# without this sourcin hack/bootstraph.sh will fail
cat >vendor/github.com/gordonklaus/ineffassign/pkg/ineffassign/BUILD.bazel <<EOT
# gazelle:ignore
load("@io_bazel_rules_go//go:def.bzl", "go_tool_library")

go_tool_library(
    name = "go_tool_library",
    srcs = ["ineffassign.go"],
    importmap = "kubevirt.io/kubevirt/vendor/github.com/gordonklaus/ineffassign/pkg/ineffassign",
    importpath = "github.com/gordonklaus/ineffassign/pkg/ineffassign",
    visibility = ["//visibility:public"],
    deps = ["@org_golang_x_tools//go/analysis:go_tool_library"],
)
EOT

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/libvirt.org/go/libvirt/BUILD.bazel

# generate BUILD files
bazel run \
    --config=${ARCHITECTURE} \
    //:gazelle -- -exclude vendor/google.golang.org/grpc --exclude cluster-up

# inject changes to libvirt BUILD file
bazel run \
    --config=${ARCHITECTURE} \
    -- @com_github_bazelbuild_buildtools//buildozer 'add cdeps //:libvirt-libs' //vendor/libvirt.org/go/libvirt:go_default_library
# allign BAZEL files to a single format
bazel run \
    --config=${ARCHITECTURE} \
    //:buildifier
