load(":cc_toolchain_config.bzl", "cc_toolchain_config")

package(default_visibility = ["//visibility:public"])

filegroup(name = "empty")

cc_toolchain_config(name = "s390x_toolchain_config")

cc_toolchain(
    name = "s390x_toolchain",
    all_files = ":empty",
    compiler_files = ":empty",
    dwp_files = ":empty",
    linker_files = ":empty",
    objcopy_files = ":empty",
    strip_files = ":empty",
    toolchain_config = ":s390x_toolchain_config",
    toolchain_identifier = "s390x-toolchain",
)

cc_toolchain_suite(
    name = "gcc_toolchain",
    tags = ["manual"],
    toolchains = {
        "s390x": ":s390x_toolchain",
    },
)

toolchain(
    name = "s390x_linux_toolchain",
    exec_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
    target_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:s390x",
    ],
    toolchain = ":s390x_toolchain",
    toolchain_type = "@bazel_tools//tools/cpp:toolchain_type",
)
