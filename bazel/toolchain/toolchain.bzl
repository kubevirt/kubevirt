def register_all_toolchains():
    native.register_toolchains(
        # CS10 toolchains first (more specific, with target_settings)
        "//bazel/toolchain/x86_64-none-linux-gnu:x86_64_linux_toolchain_cs10",
        "//bazel/toolchain/aarch64-none-linux-gnu:aarch64_linux_toolchain_cs10",
        "//bazel/toolchain/s390x-none-linux-gnu:s390x_linux_toolchain_cs10",
        # Default toolchains (CS9)
        "//bazel/toolchain/s390x-none-linux-gnu:s390x_linux_toolchain",
        "//bazel/toolchain/aarch64-none-linux-gnu:aarch64_linux_toolchain",
        "//bazel/toolchain/x86_64-none-linux-gnu:x86_64_linux_toolchain",
    )
