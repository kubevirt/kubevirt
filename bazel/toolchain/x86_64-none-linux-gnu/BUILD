load(":cc_toolchain_config.bzl", "cc_toolchain_config")

package(default_visibility = ["//visibility:public"])

filegroup(name = "empty")

cc_toolchain_config(name = "x86_64_toolchain_config")

cc_toolchain(
    name = "x86_64_toolchain",
    all_files = ":empty",
    compiler_files = ":empty",
    dwp_files = ":empty",
    linker_files = ":empty",
    objcopy_files = ":empty",
    strip_files = ":empty",
    toolchain_config = ":x86_64_toolchain_config",
    toolchain_identifier = "x86_64-toolchain",
)

cc_toolchain_suite(
    name = "gcc_toolchain",
    tags = ["manual"],
    toolchains = {
        "x86_64": ":x86_64_toolchain",
    },
)

toolchain(
    name = "x86_64_linux_toolchain",
    exec_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
    target_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
    toolchain = ":x86_64_toolchain",
    toolchain_type = "@bazel_tools//tools/cpp:toolchain_type",
)
