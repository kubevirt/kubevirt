load(":cc_toolchain_config.bzl", "cc_toolchain_config")
load(":cc_toolchain_config_cs10.bzl", "cc_toolchain_config_cs10")

package(default_visibility = ["//visibility:public"])

filegroup(name = "empty")

# CentOS Stream 9 toolchain (GCC 12)
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

# CentOS Stream 10 toolchain (GCC 14)
cc_toolchain_config_cs10(name = "s390x_toolchain_config_cs10")

cc_toolchain(
    name = "s390x_toolchain_cs10",
    all_files = ":empty",
    compiler_files = ":empty",
    dwp_files = ":empty",
    linker_files = ":empty",
    objcopy_files = ":empty",
    strip_files = ":empty",
    toolchain_config = ":s390x_toolchain_config_cs10",
    toolchain_identifier = "s390x-toolchain-cs10",
)

cc_toolchain_suite(
    name = "gcc_toolchain",
    tags = ["manual"],
    toolchains = {
        "s390x": ":s390x_toolchain",
    },
)

# CS9 toolchain - default, lower priority
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

# CS10 toolchain - selected when centos_stream_version=10
toolchain(
    name = "s390x_linux_toolchain_cs10",
    exec_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:x86_64",
    ],
    target_compatible_with = [
        "@platforms//os:linux",
        "@platforms//cpu:s390x",
    ],
    target_settings = [
        "//rpm:is_cs10",
    ],
    toolchain = ":s390x_toolchain_cs10",
    toolchain_type = "@bazel_tools//tools/cpp:toolchain_type",
)
