load(":cc_toolchain_config.bzl", "cc_toolchain_config")
load(":cc_toolchain_config_cs10.bzl", "cc_toolchain_config_cs10")

package(default_visibility = ["//visibility:public"])

filegroup(name = "empty")

# CentOS Stream 9 toolchain (GCC 11)
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

# CentOS Stream 10 toolchain (GCC 14)
cc_toolchain_config_cs10(name = "x86_64_toolchain_config_cs10")

cc_toolchain(
    name = "x86_64_toolchain_cs10",
    all_files = ":empty",
    compiler_files = ":empty",
    dwp_files = ":empty",
    linker_files = ":empty",
    objcopy_files = ":empty",
    strip_files = ":empty",
    toolchain_config = ":x86_64_toolchain_config_cs10",
    toolchain_identifier = "x86_64-toolchain-cs10",
)

cc_toolchain_suite(
    name = "gcc_toolchain",
    tags = ["manual"],
    toolchains = {
        "x86_64": ":x86_64_toolchain",
    },
)

# CS9 toolchain - default, lower priority
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

# CS10 toolchain - selected when centos_stream_version=10
toolchain(
    name = "x86_64_linux_toolchain_cs10",
    exec_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
    target_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
    target_settings = [
        "//rpm:is_cs10",
    ],
    toolchain = ":x86_64_toolchain_cs10",
    toolchain_type = "@bazel_tools//tools/cpp:toolchain_type",
)
