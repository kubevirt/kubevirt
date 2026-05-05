load("@rules_pkg//:pkg.bzl", "pkg_tar")
load("//bazel:rpm_macros.bzl", "tar2files")

def passt_repair_for_arch(arch):
    tar_label = "//rpm:passt_tree_{}".format(arch)
    binary_name = "passt_repair_binary_{}".format(arch)
    tar_name = "passt_repair_tar_{}".format(arch)

    tar2files(
        name = binary_name,
        files = {"/usr/bin": ["passt-repair"]},
        tar = tar_label,
    )

    pkg_tar(
        name = tar_name,
        srcs = [":{}/usr/bin/passt-repair".format(binary_name)],
        mode = "0744",
        package_dir = "/usr/bin",
    )
