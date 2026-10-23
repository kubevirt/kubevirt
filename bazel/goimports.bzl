"""Bazel rule for running goimports across the workspace."""

load("@bazel_skylib//lib:paths.bzl", "paths")
load("@bazel_skylib//lib:shell.bzl", "shell")
load("@io_bazel_rules_go//go:def.bzl", "go_context")

def _goimports_impl(ctx):
    go = go_context(ctx)
    args = [
        "-d=" + str(ctx.attr.display_diffs).lower(),
        "-e=" + str(ctx.attr.report_all_errors).lower(),
        "-l=" + str(ctx.attr.list).lower(),
        "-local=" + ",".join(ctx.attr.local),
        "-v=" + str(ctx.attr.verbose).lower(),
        "-w=" + str(ctx.attr.write).lower(),
    ]

    exclude_paths_str = ""
    if ctx.attr.exclude_paths:
        exclude_paths = ["-not -path %s" % shell.quote(path) for path in ctx.attr.exclude_paths]
        exclude_paths_str = " ".join(exclude_paths)

    exclude_files_str = ""
    if ctx.attr.exclude_files:
        exclude_files = ["-not -name %s" % shell.quote(file) for file in ctx.attr.exclude_files]
        exclude_files_str = " ".join(exclude_files)

    out_file = ctx.actions.declare_file(ctx.label.name + ".bash")
    substitutions = {
        "@@PREFIX_DIR_PATH@@": shell.quote(paths.dirname(ctx.attr.prefix)),
        "@@PREFIX_BASE_NAME@@": shell.quote(paths.basename(ctx.attr.prefix)),
        "@@ARGS@@": shell.array_literal(args),
        "@@GOIMPORTS_SHORT_PATH@@": shell.quote(ctx.executable._goimports.short_path),
        "@@GO_SHORT_PATH@@": shell.quote(go.go.short_path),
        "@@EXCLUDE_PATHS@@": exclude_paths_str,
        "@@EXCLUDE_FILES@@": exclude_files_str,
    }
    ctx.actions.expand_template(
        template = ctx.file._runner,
        output = out_file,
        substitutions = substitutions,
        is_executable = True,
    )
    runfiles = ctx.runfiles(files = [ctx.executable._goimports, go.go])
    return [DefaultInfo(
        files = depset([out_file]),
        runfiles = runfiles,
        executable = out_file,
    )]

_goimports = rule(
    implementation = _goimports_impl,
    attrs = {
        "prefix": attr.string(mandatory = True),
        "exclude_paths": attr.string_list(
            allow_empty = True,
            default = ["./vendor/*"],
        ),
        "exclude_files": attr.string_list(allow_empty = True),
        "display_diffs": attr.bool(),
        "_go_context_data": attr.label(
            default = "@io_bazel_rules_go//:go_context_data",
        ),
        "report_all_errors": attr.bool(),
        "list": attr.bool(),
        "local": attr.string_list(),
        "verbose": attr.bool(),
        "write": attr.bool(),
        "_goimports": attr.label(
            default = "@org_golang_x_tools//cmd/goimports",
            cfg = "exec",
            executable = True,
        ),
        "_runner": attr.label(
            default = "//bazel:goimports.runner.bash.template",
            allow_single_file = True,
        ),
    },
    toolchains = ["@io_bazel_rules_go//go:toolchain"],
    executable = True,
)

def goimports(**kwargs):
    tags = list(kwargs.get("tags", []))
    if "manual" not in tags:
        tags.append("manual")
        kwargs["tags"] = tags
    _goimports(**kwargs)
