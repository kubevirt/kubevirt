"""
Shared C++ toolchain configuration template for cross-compilation.

This module provides common configuration for all architectures (aarch64, s390x, x86_64)
to avoid duplication across cc_toolchain_config files.
"""

load("@bazel_tools//tools/build_defs/cc:action_names.bzl", "ACTION_NAMES")
load(
    "@bazel_tools//tools/cpp:cc_toolchain_config_lib.bzl",
    "feature",
    "flag_group",
    "flag_set",
    "tool_path",
)

def _get_all_link_actions():
    """Returns list of all link action names."""
    return [
        ACTION_NAMES.cpp_link_executable,
        ACTION_NAMES.cpp_link_dynamic_library,
        ACTION_NAMES.cpp_link_nodeps_dynamic_library,
    ]

def _get_all_compile_actions():
    """Returns list of all compile action names."""
    return [
        ACTION_NAMES.assemble,
        ACTION_NAMES.c_compile,
        ACTION_NAMES.clif_match,
        ACTION_NAMES.cpp_compile,
        ACTION_NAMES.cpp_header_parsing,
        ACTION_NAMES.cpp_module_codegen,
        ACTION_NAMES.cpp_module_compile,
        ACTION_NAMES.linkstamp_compile,
        ACTION_NAMES.lto_backend,
        ACTION_NAMES.preprocess_assemble,
    ]

def _get_tool_paths(tool_prefix):
    """
    Returns tool paths for a given tool prefix.

    Args:
        tool_prefix: Prefix for tools (e.g., "aarch64-linux-gnu-", "x86_64-linux-gnu-", or "")

    Returns:
        List of tool_path() rules
    """
    return [
        tool_path(name = "ar", path = "/usr/bin/{}ar".format(tool_prefix)),
        tool_path(name = "cpp", path = "/usr/bin/{}cpp".format(tool_prefix)),
        tool_path(name = "gcc", path = "/usr/bin/{}gcc".format(tool_prefix)),
        tool_path(name = "gcov", path = "/usr/bin/{}gcov".format(tool_prefix)),
        tool_path(name = "ld", path = "/usr/bin/{}ld".format(tool_prefix)),
        tool_path(name = "nm", path = "/usr/bin/{}nm".format(tool_prefix)),
        tool_path(name = "objcopy", path = "/usr/bin/{}objcopy".format(tool_prefix)),
        tool_path(name = "objdump", path = "/usr/bin/{}objdump".format(tool_prefix)),
        tool_path(name = "strip", path = "/usr/bin/{}strip".format(tool_prefix)),
    ]

def _get_common_features():
    """
    Returns list of common compiler features.

    Returns:
        List of feature() rules that apply to all architectures
    """
    return [
        feature(
            name = "default_compiler_flags",
            enabled = True,
            flag_sets = [
                flag_set(
                    actions = _get_all_compile_actions(),
                    flag_groups = [
                        flag_group(flags = [
                            "-Wall",
                            "-Wextra",
                            "-std=c++17",
                            "-fPIC",
                            "-fvisibility=hidden",
                        ]),
                    ],
                ),
            ],
        ),
        feature(
            name = "hardening",
            enabled = True,
            flag_sets = [
                flag_set(
                    actions = _get_all_compile_actions() + _get_all_link_actions(),
                    flag_groups = [
                        flag_group(flags = [
                            "-D_FORTIFY_SOURCE=2",
                            "-fstack-protector",
                        ]),
                    ],
                ),
            ],
        ),
        feature(
            name = "opt",
            flag_sets = [
                flag_set(
                    actions = _get_all_compile_actions(),
                    flag_groups = [
                        flag_group(flags = [
                            "-O2",
                            "-DNDEBUG",
                        ]),
                    ],
                    with_features = [{"features": ["opt"]}],
                ),
            ],
        ),
        feature(
            name = "dbg",
            flag_sets = [
                flag_set(
                    actions = _get_all_compile_actions(),
                    flag_groups = [
                        flag_group(flags = ["-g"]),
                    ],
                    with_features = [{"features": ["dbg"]}],
                ),
            ],
        ),
        feature(
            name = "treat_warnings_as_errors",
            flag_sets = [
                flag_set(
                    actions = _get_all_compile_actions(),
                    flag_groups = [
                        flag_group(flags = ["-Werror"]),
                    ],
                ),
            ],
        ),
        feature(
            name = "supports_dynamic_linker",
            enabled = True,
        ),
        feature(name = "no_legacy_features"),
    ]

def _get_link_features():
    """
    Returns list of link-specific features.

    Returns:
        List of feature() rules for linking
    """
    return [
        feature(
            name = "default_linker_flags",
            enabled = True,
            flag_sets = [
                flag_set(
                    actions = _get_all_link_actions(),
                    flag_groups = [
                        flag_group(flags = [
                            "-lstdc++",
                        ]),
                    ],
                ),
            ],
        ),
    ]

def create_cc_toolchain_config(
        ctx,
        toolchain_identifier,
        tool_prefix,
        target_cpu,
        target_os = "linux"):
    """
    Creates a cc_toolchain_config provider for a specific architecture.

    This function generates the configuration for a C++ toolchain targeting
    a specific architecture and OS. It parametrizes over tool prefix to handle
    different architectures (aarch64, s390x, x86_64) with minimal duplication.

    Args:
        ctx: Rule context
        toolchain_identifier: Unique identifier for this toolchain (e.g., "aarch64-toolchain")
        tool_prefix: Prefix for cross-compiler tools (e.g., "aarch64-linux-gnu-", or "")
        target_cpu: Target CPU architecture (aarch64, s390x, x86_64)
        target_os: Target OS (default: "linux")

    Returns:
        cc_toolchain_config provider
    """
    all_link_actions = _get_all_link_actions()
    all_compile_actions = _get_all_compile_actions()

    tool_paths = _get_tool_paths(tool_prefix)

    features = (
        _get_common_features() +
        _get_link_features() +
        [
            feature(
                name = "supports_pic",
                enabled = True,
            ),
        ]
    )

    return struct(
        ctx = ctx,
        features = features,
        action_configs = [],
        tool_paths = tool_paths,
        toolchain_identifier = toolchain_identifier,
        host_system_name = target_os,
        target_system_name = target_os,
        target_cpu = target_cpu,
        target_libc = "glibc",
        compiler = "gcc",
        abi_libc_version = "glibc_2.31",
        abi_version = "gcc",
        cc_target_os = None,
        builtin_sysroot = None,
        make_variables = [],
    )
