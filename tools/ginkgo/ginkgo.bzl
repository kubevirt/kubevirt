# Inspired by https://github.com/onsi/ginkgo/issues/800#issuecomment-829037396
load("@bazel_skylib//lib:shell.bzl", "shell")

def _ginkgo_test_impl(ctx):
    wrapper = ctx.actions.declare_file(ctx.label.name)
    ctx.actions.write(
        output = wrapper,
        content = """#!/usr/bin/env bash
set -e
trap "{merger} -o ${{XML_OUTPUT_FILE}} ${{XML_OUTPUT_FILE}}*" EXIT
{ginkgo} {ginkgo_args} {go_test} -- "$@"
""".format(
            ginkgo = shell.quote(ctx.executable._ginkgo.short_path),
            merger = shell.quote(ctx.executable._merger.short_path),
            ginkgo_args = " ".join([shell.quote(arg) for arg in ctx.attr.ginkgo_args]),
            # Ginkgo requires the precompiled binary end with ".test".
            go_test = shell.quote(ctx.executable.go_test.short_path + ".test"),
        ),
        is_executable = True,
    )

    return [DefaultInfo(
        executable = wrapper,
        runfiles = ctx.runfiles(
            files = ctx.files.data,
            symlinks = {ctx.executable.go_test.short_path + ".test": ctx.executable.go_test},
            transitive_files = depset([], transitive = [ctx.attr._ginkgo.default_runfiles.files, ctx.attr._merger.default_runfiles.files, ctx.attr.go_test.default_runfiles.files]),
        ),
    )]

ginkgo_test = rule(
    implementation = _ginkgo_test_impl,
    attrs = {
        "data": attr.label_list(allow_files = True),
        "go_test": attr.label(executable = True, cfg = "target"),
        "ginkgo_args": attr.string_list(),
        "_ginkgo": attr.label(default = "//vendor/github.com/onsi/ginkgo/ginkgo", executable = True, cfg = "target"),
        "_merger": attr.label(default = "//tools/junit-merger:junit-merger", executable = True, cfg = "target"),
    },
    executable = True,
    test = True,
)
