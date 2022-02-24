def register_all_toolchains():
    native.register_toolchains(
        "//bazel/toolchain/aarch64-none-linux-gnu:aarch64_linux_toolchain",
        "//bazel/toolchain/x86_64-none-linux-gnu:x86_64_linux_toolchain",
    )
