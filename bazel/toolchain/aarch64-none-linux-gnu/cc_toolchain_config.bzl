load("@bazel_tools//tools/build_defs/cc:action_names.bzl", "ACTION_NAMES")
load(
    "@bazel_tools//tools/cpp:cc_toolchain_config_lib.bzl",
    "feature",
    "flag_group",
    "flag_set",
    "tool_path",
)

all_link_actions = [
    ACTION_NAMES.cpp_link_executable,
    ACTION_NAMES.cpp_link_dynamic_library,
    ACTION_NAMES.cpp_link_nodeps_dynamic_library,
]

all_compile_actions = [
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

def _impl(ctx):
    tool_paths = [
        tool_path(
            name = "ar",
            path = "wrappers/aarch64-none-linux-gnu-ar",
        ),
        tool_path(
            name = "cpp",
            path = "wrappers/aarch64-none-linux-gnu-cpp",
        ),
        tool_path(
            name = "gcc",
            path = "wrappers/aarch64-none-linux-gnu-gcc",
        ),
        tool_path(
            name = "gcov",
            path = "wrappers/aarch64-none-linux-gnu-gcov",
        ),
        tool_path(
            name = "ld",
            path = "wrappers/aarch64-none-linux-gnu-ld",
        ),
        tool_path(
            name = "nm",
            path = "wrappers/aarch64-none-linux-gnu-nm",
        ),
        tool_path(
            name = "objdump",
            path = "wrappers/aarch64-none-linux-gnu-objdump",
        ),
        tool_path(
            name = "strip",
            path = "wrappers/aarch64-none-linux-gnu-strip",
        ),
    ]

    default_compiler_flags = feature(
        name = "default_compiler_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = all_compile_actions,
                flag_groups = [
                    flag_group(
                        flags = [
                            "-no-canonical-prefixes",
                            "-fno-canonical-system-headers",
                            "-Wno-builtin-macro-redefined",
                            "-D__DATE__=\"redacted\"",
                            "-D__TIMESTAMP__=\"redacted\"",
                            "-D__TIME__=\"redacted\"",
                        ],
                    ),
                ],
            ),
        ],
    )

    default_linker_flags = feature(
        name = "default_linker_flags",
        enabled = True,
        flag_sets = [
            flag_set(
                actions = all_link_actions,
                flag_groups = ([
                    flag_group(
                        flags = [
                            "-lstdc++",
                        ],
                    ),
                ]),
            ),
        ],
    )

    features = [
        default_compiler_flags,
        default_linker_flags,
    ]

    return cc_common.create_cc_toolchain_config_info(
        ctx = ctx,
        cxx_builtin_include_directories = [
            "/proc/self/cwd/external/aarch64-none-linux-gnu/aarch64-none-linux-gnu/include/c++/10.2.1/",
            "/proc/self/cwd/external/aarch64-none-linux-gnu/aarch64-none-linux-gnu/libc/usr/include/",
            "/proc/self/cwd/external/aarch64-none-linux-gnu/lib/gcc/aarch64-none-linux-gnu/10.2.1/include/",
            "/proc/self/cwd/external/aarch64-none-linux-gnu/aarch64-none-linux-gnu/libc/lib/",
        ],
        features = features,
        toolchain_identifier = "aarch64-toolchain",
        host_system_name = "local",
        target_system_name = "unknown",
        target_cpu = "unknown",
        target_libc = "unknown",
        compiler = "unknown",
        abi_version = "unknown",
        abi_libc_version = "unknown",
        tool_paths = tool_paths,
    )

cc_toolchain_config = rule(
    implementation = _impl,
    attrs = {},
    provides = [CcToolchainConfigInfo],
)
