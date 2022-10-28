load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

URL = "http://repo.ltekieli.com:8080/"

def toolchains():
    if "aarch64-none-linux-gnu" not in native.existing_rules():
        http_archive(
            name = "aarch64-none-linux-gnu",
            build_file = Label("//third_party/toolchains:aarch64-none-linux-gnu.BUILD"),
            urls = [
                "https://developer.arm.com/-/media/Files/downloads/gnu/11.3.rel1/binrel/arm-gnu-toolchain-11.3.rel1-x86_64-aarch64-none-linux-gnu.tar.xz",
            ],
            sha256 = "50cdef6c5baddaa00f60502cc8b59cc11065306ae575ad2f51e412a9b2a90364",
            strip_prefix = "arm-gnu-toolchain-11.3.rel1-x86_64-aarch64-none-linux-gnu",
        )
