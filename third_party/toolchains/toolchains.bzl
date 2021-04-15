load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

URL = "http://repo.ltekieli.com:8080/"

def toolchains():
    if "aarch64-none-linux-gnu" not in native.existing_rules():
        http_archive(
            name = "aarch64-none-linux-gnu",
            build_file = Label("//third_party/toolchains:aarch64-none-linux-gnu.BUILD"),
            url = "https://developer.arm.com/-/media/Files/downloads/gnu-a/10.2-2020.11/binrel/gcc-arm-10.2-2020.11-x86_64-aarch64-none-linux-gnu.tar.xz",
            sha256 = "fe7f72330216612de44891ebe5e228eed7c0c051ac090c395b2b33115c6f5408",
            strip_prefix = "gcc-arm-10.2-2020.11-x86_64-aarch64-none-linux-gnu",
        )
