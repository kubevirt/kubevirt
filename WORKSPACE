workspace(name = "kubevirt")

load("//third_party:deps.bzl", "deps")

deps()

# register crosscompiler toolchains
load("//bazel/toolchain:toolchain.bzl", "register_all_toolchains")

register_all_toolchains()

load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "rules_python",
    sha256 = "778197e26c5fbeb07ac2a2c5ae405b30f6cb7ad1f5510ea6fdac03bded96cc6f",
    urls = [
        "https://github.com/bazelbuild/rules_python/releases/download/0.2.0/rules_python-0.2.0.tar.gz",
        "https://storage.googleapis.com/builddeps/778197e26c5fbeb07ac2a2c5ae405b30f6cb7ad1f5510ea6fdac03bded96cc6f",
    ],
)

# Additional bazel rules
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "rules_proto",
    sha256 = "bc12122a5ae4b517fa423ea03a8d82ea6352d5127ea48cb54bc324e8ab78493c",
    strip_prefix = "rules_proto-af6481970a34554c6942d993e194a9aed7987780",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/af6481970a34554c6942d993e194a9aed7987780.tar.gz",
        "https://storage.googleapis.com/builddeps/bc12122a5ae4b517fa423ea03a8d82ea6352d5127ea48cb54bc324e8ab78493c",
    ],
)

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "69de5c704a05ff37862f7e0f5534d4f479418afc21806c887db544a316f3cb6b",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
        "https://storage.googleapis.com/builddeps/69de5c704a05ff37862f7e0f5534d4f479418afc21806c887db544a316f3cb6b",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "62ca106be173579c0a167deb23358fdfe71ffa1e4cfdddf5582af26520f1c66f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
        "https://storage.googleapis.com/builddeps/62ca106be173579c0a167deb23358fdfe71ffa1e4cfdddf5582af26520f1c66f",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "95d39fd84ff4474babaf190450ee034d958202043e366b9fc38f438c9e6c3334",
    strip_prefix = "rules_docker-0.16.0",
    urls = [
        "https://github.com/bazelbuild/rules_docker/releases/download/v0.16.0/rules_docker-v0.16.0.tar.gz",
        "https://storage.googleapis.com/builddeps/95d39fd84ff4474babaf190450ee034d958202043e366b9fc38f438c9e6c3334",
    ],
)

http_archive(
    name = "com_github_ash2k_bazel_tools",
    sha256 = "80ba082177c93e43a7c085a8566c7f11654dbae41da7da0da52e0ed2e917cd12",
    strip_prefix = "bazel-tools-6e2a416f565062955735edcfae881cdba2b7abf7",
    urls = [
        "https://github.com/ash2k/bazel-tools/archive/6e2a416f565062955735edcfae881cdba2b7abf7.zip",
        "https://storage.googleapis.com/builddeps/80ba082177c93e43a7c085a8566c7f11654dbae41da7da0da52e0ed2e917cd12",
    ],
)

# Disk images
http_file(
    name = "alpine_image",
    sha256 = "5a4b2588afd32e7024dd61d9558b77b03a4f3189cb4c9fc05e9e944fb780acdd",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso",
        "https://storage.googleapis.com/builddeps/5a4b2588afd32e7024dd61d9558b77b03a4f3189cb4c9fc05e9e944fb780acdd",
    ],
)

http_file(
    name = "alpine_image_aarch64",
    sha256 = "1a37be6e94bf2e102e9997c3c5bd433787299640a42d3df1597fd9dc5e1a81c7",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.11/releases/aarch64/alpine-virt-3.11.5-aarch64.iso",
        "https://storage.googleapis.com/builddeps/1a37be6e94bf2e102e9997c3c5bd433787299640a42d3df1597fd9dc5e1a81c7",
    ],
)

http_file(
    name = "cirros_image",
    sha256 = "a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    urls = [
        "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img",
        "https://storage.googleapis.com/builddeps/a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    ],
)

http_file(
    name = "cirros_image_aarch64",
    sha256 = "46c4bd31c1b39152bafe3265c8e3551dd6bc672dfee6713dc736f5e20a348e63",
    urls = [
        "https://download.cirros-cloud.net/0.5.0/cirros-0.5.0-aarch64-disk.img",
        "https://storage.googleapis.com/builddeps/46c4bd31c1b39152bafe3265c8e3551dd6bc672dfee6713dc736f5e20a348e63",
    ],
)

http_file(
    name = "fedora_image",
    sha256 = "423a4ce32fa32c50c11e3d3ff392db97a762533b81bef9d00599de518a7469c8",
    urls = ["https://storage.googleapis.com/builddeps/423a4ce32fa32c50c11e3d3ff392db97a762533b81bef9d00599de518a7469c8"],
)

http_file(
    name = "fedora_image_aarch64",
    sha256 = "b367755c664a2d7a26955bbfff985855adfa2ca15e908baf15b4b176d68d3967",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/32/Cloud/aarch64/images/Fedora-Cloud-Base-32-1.6.aarch64.qcow2",
        "https://storage.googleapis.com/builddeps/b367755c664a2d7a26955bbfff985855adfa2ca15e908baf15b4b176d68d3967",
    ],
)

http_file(
    name = "microlivecd_image",
    sha256 = "ae449ae8c0f73b1a7e2c394bc5385e7ab01d8fc000f5b074bc8b2aaabf931eac",
    urls = [
        "https://github.com/jean-edouard/microlivecd/releases/download/0.1/microlivecd_amd64.iso",
        "https://storage.googleapis.com/builddeps/ae449ae8c0f73b1a7e2c394bc5385e7ab01d8fc000f5b074bc8b2aaabf931eac",
    ],
)

http_file(
    name = "microlivecd_image_ppc64el",
    sha256 = "eae431d68b9dc5fab422f4b90d4204cbc28c39518780c4822970a4bef42f7c7f",
    urls = [
        "https://github.com/jean-edouard/microlivecd/releases/download/0.1/microlivecd_ppc64el.iso",
        "https://storage.googleapis.com/builddeps/eae431d68b9dc5fab422f4b90d4204cbc28c39518780c4822970a4bef42f7c7f",
    ],
)

http_file(
    name = "microlivecd_image_aarch64",
    sha256 = "2d9a7790fa6347251aacd997384b30962bc60dfe4eb9f0c2bd76b42f54d04b8d",
    urls = [
        "https://github.com/jean-edouard/microlivecd/releases/download/0.2/microlivecd_arm64.iso",
        "https://storage.googleapis.com/builddeps/2d9a7790fa6347251aacd997384b30962bc60dfe4eb9f0c2bd76b42f54d04b8d",
    ],
)

http_file(
    name = "virtio_win_image",
    sha256 = "7bf7f53e30c69a360f89abb3d2cc19cc978f533766b1b2270c2d8344edf9b3ef",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.171-1/virtio-win-0.1.171.iso",
        "https://storage.googleapis.com/builddeps/7bf7f53e30c69a360f89abb3d2cc19cc978f533766b1b2270c2d8344edf9b3ef",
    ],
)

http_archive(
    name = "bazeldnf",
    sha256 = "f6c9293d36914755c8fc808a2145edddcde2a26525afde1b9356bd63968b2d94",
    strip_prefix = "bazeldnf-0.1.0",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.1.0.tar.gz",
        "https://storage.googleapis.com/builddeps/f6c9293d36914755c8fc808a2145edddcde2a26525afde1b9356bd63968b2d94",
    ],
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)
load("@bazeldnf//:deps.bzl", "bazeldnf_dependencies", "rpm")

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.16.6",
    nogo = "@//:nogo_vet",
)

load("@com_github_ash2k_bazel_tools//goimports:deps.bzl", "goimports_dependencies")

goimports_dependencies()

load(
    "@bazel_gazelle//:deps.bzl",
    "gazelle_dependencies",
    "go_repository",
)

gazelle_dependencies()

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    "git_repository",
)

# Winrmcli dependencies
go_repository(
    name = "com_github_masterzen_winrmcli",
    commit = "c85a68ee8b6e3ac95af2a5fd62d2f41c9e9c5f32",
    importpath = "github.com/masterzen/winrm-cli",
)

# Winrmcp deps
go_repository(
    name = "com_github_packer_community_winrmcp",
    commit = "c76d91c1e7db27b0868c5d09e292bb540616c9a2",
    importpath = "github.com/packer-community/winrmcp",
)

go_repository(
    name = "com_github_masterzen_winrm_cli",
    commit = "6f0c57dee4569c04f64c44c335752b415e5d73a7",
    importpath = "github.com/masterzen/winrm-cli",
)

go_repository(
    name = "com_github_masterzen_winrm",
    commit = "1d17eaf15943ca3554cdebb3b1b10aaa543a0b7e",
    importpath = "github.com/masterzen/winrm",
)

go_repository(
    name = "com_github_nu7hatch_gouuid",
    commit = "179d4d0c4d8d407a32af483c2354df1d2c91e6c3",
    importpath = "github.com/nu7hatch/gouuid",
)

go_repository(
    name = "com_github_dylanmei_iso8601",
    commit = "2075bf119b58e5576c6ed9f867b8f3d17f2e54d4",
    importpath = "github.com/dylanmei/iso8601",
)

go_repository(
    name = "com_github_gofrs_uuid",
    commit = "abfe1881e60ef34074c1b8d8c63b42565c356ed6",
    importpath = "github.com/gofrs/uuid",
)

go_repository(
    name = "com_github_christrenkamp_goxpath",
    commit = "c5096ec8773dd9f554971472081ddfbb0782334e",
    importpath = "github.com/ChrisTrenkamp/goxpath",
)

go_repository(
    name = "com_github_azure_go_ntlmssp",
    commit = "4a21cbd618b459155f8b8ee7f4491cd54f5efa77",
    importpath = "github.com/Azure/go-ntlmssp",
)

go_repository(
    name = "com_github_masterzen_simplexml",
    commit = "31eea30827864c9ab643aa5a0d5b2d4988ec8409",
    importpath = "github.com/masterzen/simplexml",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "4def268fd1a49955bfb3dda92fe3db4f924f2285",
    importpath = "golang.org/x/crypto",
)

# override rules_docker issue with this dependency
# rules_docker 0.16 uses 0.1.4, bit since there the checksum changed, which is very weird, going with 0.1.4.1 to
go_repository(
    name = "com_github_google_go_containerregistry",
    importpath = "github.com/google/go-containerregistry",
    sha256 = "bc0136a33f9c1e4578a700f7afcdaa1241cfff997d6bba695c710d24c5ae26bd",
    strip_prefix = "google-go-containerregistry-efb2d62",
    type = "tar.gz",
    urls = ["https://api.github.com/repos/google/go-containerregistry/tarball/efb2d62d93a7705315b841d0544cb5b13565ff2a"],  # v0.1.4.1
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load(
    "@io_bazel_rules_docker//cc:image.bzl",
    _cc_image_repos = "repositories",
)

_cc_image_repos()

# nispor library
new_local_repository(
    name = "nispor",
    build_file_content = """
load("@io_bazel_rules_docker//cc:image.bzl", "cc_image")
load("@bazel_tools//tools/build_defs/pkg:pkg.bzl", "pkg_tar")

exports_files(["lib64/libnispor.so.1.1.1"])

cc_import(
    name = "lib",
    shared_library = "lib64/libnispor.so.1.1.1",
    hdrs = ["include/nispor.h"],
    visibility = ["//visibility:public"],
)

pkg_tar(
    name = "libpkg",
    package_dir = "/usr/lib64",
    mode = "0755",
    srcs = ["lib64/libnispor.so.1.1.1"],
    visibility = ["//visibility:public"],
)
""",
    path = "nispor",
)

# bazel docker rules
load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_image",
    "container_pull",
)
load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

# Pull base image fedora31
# WARNING: please update any automated process to push this image to quay.io
# instead of index.docker.io
container_pull(
    name = "fedora",
    digest = "sha256:5e2b864cfe165fa7da6606b29a9e60549eb7cc9ae7fb574614110d1494b0f0c2",
    registry = "quay.io",
    repository = "kubevirtci/fedora",
    tag = "31",
)

# As rpm package in https://dl.fedoraproject.org/pub/fedora/linux/releases/31 is empty, we use fedora 32 here.
# TODO add fedora image to quay.io
container_pull(
    name = "fedora_aarch64",
    digest = "sha256:425676dd30f2c85ba3593b82040ce03341cd6dc4e38838e57c8bc5eef95b5f81",
    registry = "index.docker.io",
    repository = "library/fedora",
    tag = "32",
)

# Pull fedora 32 customize container-disk
# WARNING: please update any automated process to push this image to quay.io
# instead of index.docker.io
# TODO build fedora_sriov_lane for multi-arch
container_pull(
    name = "fedora_sriov_lane",
    digest = "sha256:6f66ee747d62c354c0d36e640f8c97d6be0b6ad88a9e8c0180496ac55cba31bf",
    registry = "quay.io",
    repository = "kubevirtci/fedora-sriov-testing",
)

container_pull(
    name = "fedora_sriov_lane_aarch64",
    digest = "sha256:6f66ee747d62c354c0d36e640f8c97d6be0b6ad88a9e8c0180496ac55cba31bf",
    registry = "quay.io",
    repository = "kubevirtci/fedora-sriov-testing",
)

# Pull go_image_base
container_pull(
    name = "go_image_base",
    digest = "sha256:f65536ce108fcc41cdcd5cb101006fcb82b9a1527409263feb9e34032f00bda0",
    registry = "gcr.io",
    repository = "distroless/base",
)

container_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:789c477fbd30a7d85435450306e54f20c53938e40af644284a229d852db30dde",
    registry = "gcr.io",
    repository = "distroless/base",
)

# Pull nfs-server image
# WARNING: please update any automated process to push this image to quay.io
# instead of index.docker.io
# TODO build nfs-server for multi-arch
container_pull(
    name = "nfs-server",
    digest = "sha256:8c1fa882dddb2885c4152e9ce632c466f4b8dce29339455e9b6bfe71f0a3d3ef",
    registry = "quay.io",
    repository = "kubevirtci/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
)

container_pull(
    name = "nfs-server_aarch64",
    digest = "sha256:8c1fa882dddb2885c4152e9ce632c466f4b8dce29339455e9b6bfe71f0a3d3ef",
    registry = "quay.io",
    repository = "kubevirtci/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
)

# Pull fedora container-disk preconfigured with ci tooling
# like stress and qemu guest agent pre-configured
# TODO build fedora_with_test_tooling for multi-arch
container_pull(
    name = "fedora_with_test_tooling",
    digest = "sha256:ce36d2b4f81b038fba0b61b1bb1ac7f671d47687fb1f9d7ddedd22742cc79dd9",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
)

container_pull(
    name = "fedora_with_test_tooling_aarch64",
    digest = "sha256:9ec3e137bff093597d192f5a4e346f25b614c3a94216b857de0e3d75b68bfb17",
    registry = "quay.io",
    repository = "kubevirt/fedora-with-test-tooling",
)

container_pull(
    name = "alpine-ext-kernel-boot-demo-container-base",
    digest = "sha256:a2ddb2f568bf3814e594a14bc793d5a655a61d5983f3561d60d02afa7bbc56b4",
    registry = "quay.io",
    repository = "kubevirt/alpine-ext-kernel-boot-demo",
)

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

http_archive(
    name = "io_bazel_rules_container_rpm",
    sha256 = "151261f1b81649de6e36f027c945722bff31176f1340682679cade2839e4b1e1",
    strip_prefix = "rules_container_rpm-0.0.5",
    urls = [
        "https://github.com/rmohr/rules_container_rpm/archive/v0.0.5.tar.gz",
        "https://storage.googleapis.com/builddeps/151261f1b81649de6e36f027c945722bff31176f1340682679cade2839e4b1e1",
    ],
)

http_file(
    name = "libguestfs-appliance",
    sha256 = "3e49a80d688b165ce8535369aa352f80eb832001f0ae6f2e1af1b4e9986421c5",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/appliance-1.44.0-linux-5.11.22-100-fedora32.tar.xz",
    ],
)

# Get container-disk-v1alpha RPM's
http_file(
    name = "qemu-img",
    sha256 = "669250ad47aad5939cf4d1b88036fd95a94845d8e0bbdb05e933f3d2fe262fea",
    urls = ["https://storage.googleapis.com/builddeps/669250ad47aad5939cf4d1b88036fd95a94845d8e0bbdb05e933f3d2fe262fea"],
)

# some repos which are not part of go_rules anymore
go_repository(
    name = "com_github_golang_glog",
    importpath = "github.com/golang/glog",
    sum = "h1:VKtxabqXZkF25pY9ekfRL6a582T4P37/31XEstQ5p58=",
    version = "v0.0.0-20160126235308-23def4e6c14b",
)

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    sum = "h1:M5a8xTlYTxwMn5ZFkwhRabsygDY5G8TYLyQDBxJNAxE=",
    version = "v1.30.0",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    sum = "h1:oWX7TPOiFAMXLq8o0ikBYfCJVlRHBcsciT5bXOrH628=",
    version = "v0.0.0-20190311183353-d8887717615a",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    sum = "h1:g61tztE5qeGQ89tm6NTjjM9VPIm088od1l6aSorWRWg=",
    version = "v0.3.0",
)

register_toolchains("//:py_toolchain")

go_repository(
    name = "org_golang_x_mod",
    build_file_generation = "on",
    build_file_proto_mode = "disable",
    importpath = "golang.org/x/mod",
    sum = "h1:RM4zey1++hCTbCVQfnWeKs9/IEsaBLA8vTkd0WVtmH4=",
    version = "v0.3.0",
)

go_repository(
    name = "org_golang_x_xerrors",
    build_file_generation = "on",
    build_file_proto_mode = "disable",
    importpath = "golang.org/x/xerrors",
    sum = "h1:go1bK/D/BFZV2I8cIQd1NKEZ+0owSTG1fDTci4IqFcE=",
    version = "v0.0.0-20200804184101-5ec99f83aff1",
)

bazeldnf_dependencies()

rpm(
    name = "acl-0__2.2.53-9.fc33.aarch64",
    sha256 = "e7dab71e07c48e8a370873ee66455f7befdde76c2b94f33d4276c807779e7f71",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/a/acl-2.2.53-9.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/a/acl-2.2.53-9.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/a/acl-2.2.53-9.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/a/acl-2.2.53-9.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "acl-0__2.2.53-9.fc33.x86_64",
    sha256 = "92c1615d385b32088f78a6574a2bf89a6bb29d9858abdd71471ef5113ef0831f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/acl-2.2.53-9.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/acl-2.2.53-9.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/acl-2.2.53-9.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/a/acl-2.2.53-9.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "alternatives-0__1.14-3.fc33.aarch64",
    sha256 = "675137138c943198fb726dd943fc08a212a8b0ead1d95a13f767a960108b80ac",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/a/alternatives-1.14-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/a/alternatives-1.14-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/a/alternatives-1.14-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/a/alternatives-1.14-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "alternatives-0__1.14-3.fc33.x86_64",
    sha256 = "2200dd65dff57b773532153d3626ecb5914bd7826c42c689ca34be3f60ac3fe2",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/alternatives-1.14-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/alternatives-1.14-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/alternatives-1.14-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/a/alternatives-1.14-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "attr-0__2.4.48-10.fc33.x86_64",
    sha256 = "1a3b95c248ceae0d5a5dab151aa967828d1781c058ba7afda47a4ee3384b4af3",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/attr-2.4.48-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/attr-2.4.48-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/attr-2.4.48-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/a/attr-2.4.48-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "audit-libs-0__3.0.3-1.fc33.aarch64",
    sha256 = "1059d513fd15f5034b0fb48b6a4d11efacf1f5fdba853df2932b7f5235ebbdf5",
    urls = [
        "https://mirrors.xtom.ee/fedora/updates/33/Everything/aarch64/Packages/a/audit-libs-3.0.3-1.fc33.aarch64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/updates/33/Everything/aarch64/Packages/a/audit-libs-3.0.3-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/a/audit-libs-3.0.3-1.fc33.aarch64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/aarch64/Packages/a/audit-libs-3.0.3-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "audit-libs-0__3.0.3-1.fc33.x86_64",
    sha256 = "ac9648bdcda7fa82a8b19d5b2bd0714037085d1f2030f4f848c02456c926a297",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/a/audit-libs-3.0.3-1.fc33.x86_64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/a/audit-libs-3.0.3-1.fc33.x86_64.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/a/audit-libs-3.0.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/a/audit-libs-3.0.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "augeas-libs-0__1.12.0-6.fc33.x86_64",
    sha256 = "a509a2ac5edc981181650f924da8b308a27129bd794dced7e4f9bfd7ce589543",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/a/augeas-libs-1.12.0-6.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/a/augeas-libs-1.12.0-6.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/a/augeas-libs-1.12.0-6.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/a/augeas-libs-1.12.0-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.16-7.fc33.aarch64",
    sha256 = "82a16e5bdd335e3d3d4812fd074308021a8432b89e6a2c1bc24b60cf3c967f7f",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.16-7.fc33.x86_64",
    sha256 = "5b17b2acf46bceb62d62fc8b3a7a8cf25efd7224fd612c95944e03aa933ba73a",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "basesystem-0__11-10.fc33.aarch64",
    sha256 = "f4efaa5bc8382246d8230ece8bacebd3c29eb9fd52b509b1e6575e643953851b",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
    ],
)

rpm(
    name = "basesystem-0__11-10.fc33.x86_64",
    sha256 = "f4efaa5bc8382246d8230ece8bacebd3c29eb9fd52b509b1e6575e643953851b",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/b/basesystem-11-10.fc33.noarch.rpm",
    ],
)

rpm(
    name = "bash-0__5.0.17-2.fc33.aarch64",
    sha256 = "278a1a1515db1bdda811747358fc64c2fa95f1709cf70646518952b62dd6c591",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/b/bash-5.0.17-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/bash-5.0.17-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/b/bash-5.0.17-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/bash-5.0.17-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "bash-0__5.0.17-2.fc33.x86_64",
    sha256 = "c59a621f3cdd5e073b3c1ef9cd8fd9d7e02d77d94be05330390eac05f77b5b60",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bash-5.0.17-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bash-5.0.17-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bash-5.0.17-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/b/bash-5.0.17-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "binutils-0__2.35-18.fc33.x86_64",
    sha256 = "550b4a118f9ec68b20ef9e1dc265cbf4044932b17225df4573191a59ffb88ac7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/b/binutils-2.35-18.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/b/binutils-2.35-18.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/b/binutils-2.35-18.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/b/binutils-2.35-18.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "binutils-gold-0__2.35-18.fc33.x86_64",
    sha256 = "b362690f0fbadcd4c11c5393d2b7da62803fe1c5b88fc3b18533762ea399527c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/b/binutils-gold-2.35-18.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/b/binutils-gold-2.35-18.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/b/binutils-gold-2.35-18.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/b/binutils-gold-2.35-18.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "btrfs-progs-0__5.13-1.fc33.x86_64",
    sha256 = "9b7338148a15f91ccf297b15e27ef4edbb8c234a38f742a606e4babaac468a5a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/b/btrfs-progs-5.13-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/b/btrfs-progs-5.13-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/b/btrfs-progs-5.13-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/b/btrfs-progs-5.13-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-4.fc33.aarch64",
    sha256 = "0256c1d649d9a30a3a5748f39adea8e31043fb679e160c4504fa4445faadc2d1",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/b/bzip2-1.0.8-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/bzip2-1.0.8-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/b/bzip2-1.0.8-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/bzip2-1.0.8-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-4.fc33.x86_64",
    sha256 = "4286e638411ffee177dd05ebb2d58b5e86a26bf0151a7d4c3c2f1e6999b78522",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bzip2-1.0.8-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bzip2-1.0.8-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bzip2-1.0.8-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/b/bzip2-1.0.8-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-4.fc33.aarch64",
    sha256 = "7ff5ca47bd625e4db19a49da01b3784830988bd12c364e4c466b67c81a218476",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-4.fc33.x86_64",
    sha256 = "79d722ced9766b7a0661e498b0408cec9cb6ea048ec67ea052bdf0949b65dd54",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ca-certificates-0__2021.2.50-1.0.fc33.aarch64",
    sha256 = "bfc524db1a5566ebb2b5b8915854d2215f71af4159e1e538794f2fcce11ef040",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
    ],
)

rpm(
    name = "ca-certificates-0__2021.2.50-1.0.fc33.x86_64",
    sha256 = "bfc524db1a5566ebb2b5b8915854d2215f71af4159e1e538794f2fcce11ef040",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/ca-certificates-2021.2.50-1.0.fc33.noarch.rpm",
    ],
)

rpm(
    name = "checkpolicy-0__3.1-3.fc33.aarch64",
    sha256 = "e289920abfed7e65cd018d7bf663aae94f914b2bf3b427c333d99270f92cb7c0",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/c/checkpolicy-3.1-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/checkpolicy-3.1-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/c/checkpolicy-3.1-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/checkpolicy-3.1-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "checkpolicy-0__3.1-3.fc33.x86_64",
    sha256 = "c6db4defb99e600890ad91ec6eac65e75394e1ddc02daea3622a647775cb5f5d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/checkpolicy-3.1-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/checkpolicy-3.1-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/checkpolicy-3.1-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/checkpolicy-3.1-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "compat-readline5-0__5.2-37.fc33.x86_64",
    sha256 = "d37fb057cd371d93c2b3903544bbd3d30683242867ebfd7996866494c9b71021",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/compat-readline5-5.2-37.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/compat-readline5-5.2-37.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/compat-readline5-5.2-37.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/compat-readline5-5.2-37.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-21.fc33.aarch64",
    sha256 = "0f173e7045de7b5d3c7c0ee115381ecb4d564bd424bdff36bb3716a657b33bb4",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/c/coreutils-single-8.32-21.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/c/coreutils-single-8.32-21.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/c/coreutils-single-8.32-21.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/c/coreutils-single-8.32-21.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-21.fc33.x86_64",
    sha256 = "6406a660d2a48fee1f5cff443cbaefbea5469af83f45526ceb396ac09aaaadbf",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/coreutils-single-8.32-21.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/c/coreutils-single-8.32-21.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/c/coreutils-single-8.32-21.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/coreutils-single-8.32-21.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "cpio-0__2.13-8.fc33.x86_64",
    sha256 = "e86b1c2c512192248d8e510015c5e65241bef056338b414a1fb388df35e75330",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cpio-2.13-8.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cpio-2.13-8.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cpio-2.13-8.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/cpio-2.13-8.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-24.fc33.aarch64",
    sha256 = "d6d08d7d9405e7d83477fba28a33b2651988b0d6041cf637ebaa3dc8bff25638",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/c/cracklib-2.9.6-24.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cracklib-2.9.6-24.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/c/cracklib-2.9.6-24.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cracklib-2.9.6-24.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-24.fc33.x86_64",
    sha256 = "d43821773988f753ba824c731f62af463216f3a84e39c0199c5768b062423a8c",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cracklib-2.9.6-24.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cracklib-2.9.6-24.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cracklib-2.9.6-24.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/cracklib-2.9.6-24.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "crypto-policies-0__20200918-1.git85dccc5.fc33.aarch64",
    sha256 = "b21925570643c716d57353e2d2e2f05ad1ed75743ffced9343d8115d3960ee0e",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "crypto-policies-0__20200918-1.git85dccc5.fc33.x86_64",
    sha256 = "b21925570643c716d57353e2d2e2f05ad1ed75743ffced9343d8115d3960ee0e",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/crypto-policies-20200918-1.git85dccc5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "cryptsetup-0__2.3.6-1.fc33.x86_64",
    sha256 = "ced6d7fd8ed0e3d9d2b174748733b567945dda7877d3d66c18d8e423050da5d3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-2.3.6-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-2.3.6-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-2.3.6-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-2.3.6-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.6-1.fc33.aarch64",
    sha256 = "f0641a498cbbb392d477f35d861adcdcb87f54adc21930008d412c4612702095",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.6-1.fc33.x86_64",
    sha256 = "e4a7425fe86c23d4ab941a486cda819502128c8e13cbc70d5a898a380317c4da",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.6-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "curl-minimal-0__7.71.1-9.fc33.aarch64",
    sha256 = "8be83f9bd7599684435b1664d7d5bdb0b8b44a486db71c35888ddd89b2dcb4f6",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/c/curl-minimal-7.71.1-9.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/c/curl-minimal-7.71.1-9.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/c/curl-minimal-7.71.1-9.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/c/curl-minimal-7.71.1-9.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "curl-minimal-0__7.71.1-9.fc33.x86_64",
    sha256 = "54ebe1773a3adf5287ca4396b72cc6d0e26670c72a6e08b3e2802e2cf3d03acf",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/curl-minimal-7.71.1-9.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/c/curl-minimal-7.71.1-9.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/c/curl-minimal-7.71.1-9.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/c/curl-minimal-7.71.1-9.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-6.fc33.aarch64",
    sha256 = "5a518b461e1a7de5c2d3430adaf8896dffd6e22a63cfea5e092130d09fbe1c23",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-6.fc33.x86_64",
    sha256 = "7cb1635c2eaec68363207e08f43340deb99a48496edb25f7c42efa2be462b028",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-6.fc33.aarch64",
    sha256 = "0477f216e01c78607cbbf4bdf25c1014720f340f3de4776a01e8dfba407804f3",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-6.fc33.x86_64",
    sha256 = "35b5cf88dd0c861c498a04618c95d18740fbc15af228db333323f8885efd9f57",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-6.fc33.aarch64",
    sha256 = "787e1490b188fc6ff9747bbd21e23cb9cbf1bfc00ae5844dc91ee80534dc6215",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-6.fc33.x86_64",
    sha256 = "6b1d965a722a5ef3f53ce486b72c7aba5f9d1afcbce952227d66adac8665c270",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "daxctl-libs-0__71.1-1.fc33.x86_64",
    sha256 = "50c3ca09fcc055f3b5448ee75a9c0b69208e6fb8980d02a85b5e740fb1c1eacc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/daxctl-libs-71.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/daxctl-libs-71.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/daxctl-libs-71.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/daxctl-libs-71.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dbus-1__1.12.20-2.fc33.aarch64",
    sha256 = "2fa5c426b772c282228429ca24317b002121a35938874dc0388b5b8a347a371d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/d/dbus-1.12.20-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/dbus-1.12.20-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/d/dbus-1.12.20-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/dbus-1.12.20-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "dbus-1__1.12.20-2.fc33.x86_64",
    sha256 = "e1a52d4de212a8d40f5f54030e3214b1080bbd936783721b2f942c716c3d78bf",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-1.12.20-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-1.12.20-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-1.12.20-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/dbus-1.12.20-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dbus-broker-0__28-3.fc33.aarch64",
    sha256 = "49172a401b26c8163dab63cc934bc37eb03e40aa97bcfab57acab516107ae161",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/d/dbus-broker-28-3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/d/dbus-broker-28-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/d/dbus-broker-28-3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/d/dbus-broker-28-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "dbus-broker-0__28-3.fc33.x86_64",
    sha256 = "31295dc305968c7caaa1e7a5d446239965ba9759a5fcef1e1211efce279a4a9b",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dbus-broker-28-3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dbus-broker-28-3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dbus-broker-28-3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dbus-broker-28-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-2.fc33.aarch64",
    sha256 = "2a2bf7b072831968262f9e6d046a925f3c2fcee2a984114b83130802b6e714fb",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-2.fc33.x86_64",
    sha256 = "2a2bf7b072831968262f9e6d046a925f3c2fcee2a984114b83130802b6e714fb",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/dbus-common-1.12.20-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-2.fc33.x86_64",
    sha256 = "37874b92e316af462e6035ed4e274770a7053302743f5a2db637940c4ae7a551",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-libs-1.12.20-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-libs-1.12.20-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dbus-libs-1.12.20-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/dbus-libs-1.12.20-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-0__1.02.173-1.fc33.aarch64",
    sha256 = "2cca39ca6e3c78698d28ece07dd4b9e1f6ac55f197583540a4b5df783bdb990c",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/d/device-mapper-1.02.173-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/device-mapper-1.02.173-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/d/device-mapper-1.02.173-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/device-mapper-1.02.173-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-0__1.02.173-1.fc33.x86_64",
    sha256 = "3d0f1d848a92a8401ca6c8778f9a9a329af8a8420ae14a5c8c99ccbcbd97ebb7",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/device-mapper-1.02.173-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-event-0__1.02.173-1.fc33.x86_64",
    sha256 = "68242b0ea47075bd78ef4bbab44520d2061582ad8ebf57fd4027fdac77f256f0",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.173-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-event-libs-0__1.02.173-1.fc33.x86_64",
    sha256 = "605a07738477a5a7d9c536f84e7df5b3f7c607125c08223151cab4dae1e8b9cb",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.173-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-libs-0__1.02.173-1.fc33.aarch64",
    sha256 = "694ed46b1e411e7df03ed5cf6f8f47d3af3d9d38b5ca640bf022aa223dcdf0d8",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-libs-0__1.02.173-1.fc33.x86_64",
    sha256 = "9539c6e7a76422600939d661382634d7912e0669aa7e273fdf14b1fcde5b0652",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.173-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-7.fc33.aarch64",
    sha256 = "bba1c54ca259e849440a47443b5d89023c7c4b5faeb1a4996b5119864d457c26",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-7.fc33.x86_64",
    sha256 = "b3ddc1bd2758ca68bb770072f5d541e3986c70e71065cd91d3829a62b22f54b0",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/device-mapper-multipath-libs-0.8.4-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-persistent-data-0__0.8.5-4.fc33.x86_64",
    sha256 = "f7e8201cb8e3fb9269c47c1ca758aebcd529a7a1578bd520d74074943e96b3e9",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dhcp-client-12__4.4.2-9.b1.fc33.x86_64",
    sha256 = "5a712f8fa25d0a3e23e833a5a707657f3d242ec4685d79edd1eefd857e2e8d81",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dhcp-client-4.4.2-9.b1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dhcp-client-4.4.2-9.b1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dhcp-client-4.4.2-9.b1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dhcp-client-4.4.2-9.b1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dhcp-common-12__4.4.2-9.b1.fc33.x86_64",
    sha256 = "8ba3b6275243d04461f54578a9d35be279dda846b8d9bb9a1ffd19388742fe86",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dhcp-common-4.4.2-9.b1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dhcp-common-4.4.2-9.b1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dhcp-common-4.4.2-9.b1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dhcp-common-4.4.2-9.b1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "diffutils-0__3.7-7.fc33.aarch64",
    sha256 = "0a7c18e98100c119db590976abdd08e9afbd606b09f4f0f01c2989feb80fef8a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/d/diffutils-3.7-7.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/d/diffutils-3.7-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/d/diffutils-3.7-7.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/d/diffutils-3.7-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "diffutils-0__3.7-7.fc33.x86_64",
    sha256 = "771190da938657df1479390c745ddfe6252ffe5fefe95f9440e4952b77cf35be",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/diffutils-3.7-7.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/diffutils-3.7-7.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/diffutils-3.7-7.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/diffutils-3.7-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dmidecode-1__3.2-8.fc33.x86_64",
    sha256 = "858d47c7d613d31a40e5e750f949e9a23b47eb7c9e7de85cd03f64181cc6640a",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dmidecode-3.2-8.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dmidecode-3.2-8.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dmidecode-3.2-8.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/dmidecode-3.2-8.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dnf-0__4.8.0-1.fc33.x86_64",
    sha256 = "2ce4a9cde4339f64ee68f800d5cddece63bf106ac321499b79c0459c1d4b3025",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-4.8.0-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-4.8.0-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dnf-4.8.0-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-4.8.0-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "dnf-data-0__4.8.0-1.fc33.x86_64",
    sha256 = "9aa8b00ac6a06ea2db5f66cc2f76ca979b650e135839e037581acb60a1afa6ff",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-data-4.8.0-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-data-4.8.0-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dnf-data-4.8.0-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-data-4.8.0-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "dnf-plugins-core-0__4.0.22-1.fc33.x86_64",
    sha256 = "cd90ba56638e7853bd8c62edf158cc88d12e4e970aa00b7c31a10655e7e0248e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "dosfstools-0__4.1-12.fc33.x86_64",
    sha256 = "e8b414d97aed9eebe7155567b9eb10ebc2254398926ed38fbcb79a2da5175ba5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dosfstools-4.1-12.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dosfstools-4.1-12.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/d/dosfstools-4.1-12.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/d/dosfstools-4.1-12.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "dracut-0__055-3.fc33.x86_64",
    sha256 = "93c6e3666a9b0c6055cf6f2262221127b065de238af1f0617bbf2fe3310e622a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dracut-055-3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/d/dracut-055-3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/d/dracut-055-3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/d/dracut-055-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.6-4.fc33.aarch64",
    sha256 = "f90f2fa2e52ad3d6c44f33644329efbc47a74ff590ee27a3c485f164dac2022f",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.6-4.fc33.x86_64",
    sha256 = "184262d3114e289deac7fe53e7bf6c5867f6cc6892c828bb105d8793884ec9db",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.6-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-4.fc33.aarch64",
    sha256 = "5e611b8620249b9c614f631fca188418811b1cab80c35813fdf102605e398767",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-4.fc33.x86_64",
    sha256 = "aebd92e625196d0455167dd14a959ab202223d0d3abf567a8bb808d8c89023e8",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.6-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "edk2-aarch64-0__20200801stable-3.fc33.aarch64",
    sha256 = "6ed871556b36694f815ae323ebc2ab7d628033b0c70ec8fd5c39e28b34100cdf",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-3.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-3.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-3.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-3.fc33.noarch.rpm",
    ],
)

rpm(
    name = "edk2-ovmf-0__20200801stable-3.fc33.x86_64",
    sha256 = "252de3c31ef2f044b40508aef7ba722dda9bad44bcbc52649a6648f54f9020d1",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-3.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-3.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-3.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-3.fc33.noarch.rpm",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.185-2.fc33.x86_64",
    sha256 = "29baa496795f92188012e2eb2037d40e71459542fe2d652190397814efb59195",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.185-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.185-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.185-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.185-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.185-2.fc33.aarch64",
    sha256 = "4459853986012d5857680d293ab44941749bbec9b2f846c032c2a81114a18029",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.185-2.fc33.x86_64",
    sha256 = "4459853986012d5857680d293ab44941749bbec9b2f846c032c2a81114a18029",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.185-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.185-2.fc33.aarch64",
    sha256 = "5d3f83f1903eed51bfd05ff27469d9d5a3283219f19f598844618da0b1c07870",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libelf-0.185-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libelf-0.185-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libelf-0.185-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libelf-0.185-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.185-2.fc33.x86_64",
    sha256 = "2247822d6f22b40296cad9aa7de4955546d15289e7570227c657e083791af043",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libelf-0.185-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libelf-0.185-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libelf-0.185-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libelf-0.185-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "elfutils-libs-0__0.185-2.fc33.aarch64",
    sha256 = "ca79c6260b91e45963f3a620f38c683d24c4c76d0cc1ca878067bad564faa759",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libs-0.185-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libs-0.185-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libs-0.185-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/e/elfutils-libs-0.185-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "elfutils-libs-0__0.185-2.fc33.x86_64",
    sha256 = "21c7eb74ccbd13f10010a4b04ebed51c07d3ecf21522bbebab6e698970e1e453",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libs-0.185-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libs-0.185-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libs-0.185-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/elfutils-libs-0.185-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "expat-0__2.4.1-1.fc33.aarch64",
    sha256 = "853c5c371f8c6193d3df53521bd98d5e365998b398f407ff5ea0d5aea3b28485",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/e/expat-2.4.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/e/expat-2.4.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/e/expat-2.4.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/e/expat-2.4.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "expat-0__2.4.1-1.fc33.x86_64",
    sha256 = "d407e7b349bab74f6b7767b23805cd2eb480c464e68831af9881da0cbf47097a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/expat-2.4.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/e/expat-2.4.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/e/expat-2.4.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/e/expat-2.4.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "fedora-gpg-keys-0__33-5.aarch64",
    sha256 = "60458c315a03bdd0ae84b2e1bf80d51c18bc65099addc55383ca556c72e6edef",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
    ],
)

rpm(
    name = "fedora-gpg-keys-0__33-5.x86_64",
    sha256 = "60458c315a03bdd0ae84b2e1bf80d51c18bc65099addc55383ca556c72e6edef",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-gpg-keys-33-5.noarch.rpm",
    ],
)

rpm(
    name = "fedora-logos-httpd-0__30.0.2-5.fc33.aarch64",
    sha256 = "80ec0b61ff35376a226026aa0db8a2a012f54d777398f37477de4316c3dd9ca0",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "fedora-logos-httpd-0__30.0.2-5.fc33.x86_64",
    sha256 = "80ec0b61ff35376a226026aa0db8a2a012f54d777398f37477de4316c3dd9ca0",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-common-0__33-4.aarch64",
    sha256 = "64835f399490036b175241b8d4ef16a26829c3f6276a901a79a7a7f16ff8388d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-common-33-4.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-common-33-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-common-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-common-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-common-0__33-4.x86_64",
    sha256 = "64835f399490036b175241b8d4ef16a26829c3f6276a901a79a7a7f16ff8388d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-common-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-common-33-4.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-common-33-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-common-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-container-0__33-4.aarch64",
    sha256 = "10af1cbaa2f48ffb949a9c677fa28c0e0230116aec739c1b94f2e229144731a4",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-container-33-4.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-container-33-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-container-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-container-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-container-0__33-4.x86_64",
    sha256 = "10af1cbaa2f48ffb949a9c677fa28c0e0230116aec739c1b94f2e229144731a4",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-container-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-container-33-4.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-container-33-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-container-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-identity-container-0__33-4.aarch64",
    sha256 = "726de55a6465ed3b71c295c41b4ecb49c291f47a28f6a5cb1fa8171db895411c",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-container-33-4.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-container-33-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-container-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-container-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-identity-snappy-0__33-4.x86_64",
    sha256 = "bcbdf7f32a66c6e819a1f9f8f75b0e6629a9c72701fb562328a8987f9cb0d81a",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-snappy-33-4.noarch.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-snappy-33-4.noarch.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-snappy-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-snappy-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-identity-soas-0__33-4.aarch64",
    sha256 = "470d4b5d2281860443ddce032f3a7044d2c79952c7384c87aa51882b11cc4dde",
    urls = [
        "https://mirrors.xtom.ee/fedora/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-identity-soas-0__33-4.x86_64",
    sha256 = "470d4b5d2281860443ddce032f3a7044d2c79952c7384c87aa51882b11cc4dde",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-soas-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-release-identity-xfce-0__33-4.x86_64",
    sha256 = "1f638049c6a94d5432f672f052858642ff3aa1bc0814f249bf839202165dc2d2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-xfce-33-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-xfce-33-4.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-xfce-33-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-release-identity-xfce-33-4.noarch.rpm",
    ],
)

rpm(
    name = "fedora-repos-0__33-5.aarch64",
    sha256 = "12636321f3701fc57a1d97ea59f0395180a4d2d470a51ffcf3c407f28024d315",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-repos-33-5.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/f/fedora-repos-33-5.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-repos-33-5.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/f/fedora-repos-33-5.noarch.rpm",
    ],
)

rpm(
    name = "fedora-repos-0__33-5.x86_64",
    sha256 = "12636321f3701fc57a1d97ea59f0395180a4d2d470a51ffcf3c407f28024d315",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-repos-33-5.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-repos-33-5.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/f/fedora-repos-33-5.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/f/fedora-repos-33-5.noarch.rpm",
    ],
)

rpm(
    name = "file-0__5.39-3.fc33.x86_64",
    sha256 = "1ef4150dbe503b704c3d420ea9210eb8a62b6f5d5f4afb432239dfedecf8ef0d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/file-5.39-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/file-5.39-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/file-5.39-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/file-5.39-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "file-libs-0__5.39-3.fc33.x86_64",
    sha256 = "1d694765c7aa5e8ccb1509c26976642998db953da0723c9b06d8d4bdf1b87f2e",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/file-libs-5.39-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/file-libs-5.39-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/file-libs-5.39-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/file-libs-5.39-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "filesystem-0__3.14-3.fc33.aarch64",
    sha256 = "da4099138efb6fd069feede5d7e4cd371e9f69a9e363cee5fd58ab79c03840b0",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/f/filesystem-3.14-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/filesystem-3.14-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/f/filesystem-3.14-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/filesystem-3.14-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "filesystem-0__3.14-3.fc33.x86_64",
    sha256 = "2d9ed3be09813ff727751a6db3a839e49630257df9ab5a21204335f4ca49fecc",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/filesystem-3.14-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/filesystem-3.14-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/filesystem-3.14-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/filesystem-3.14-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "findutils-1__4.7.0-7.fc33.aarch64",
    sha256 = "5f15c98b05cb2d576d771288d6a3cd1ce81d4a0a6963b7c287cc816f86dd31bf",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/f/findutils-4.7.0-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/findutils-4.7.0-7.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/f/findutils-4.7.0-7.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/findutils-4.7.0-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "findutils-1__4.7.0-7.fc33.x86_64",
    sha256 = "0fc62ef8c645c239295982a2e6436bd3604c367d82b48e145b65d20c1f9e8b35",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/findutils-4.7.0-7.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/findutils-4.7.0-7.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/findutils-4.7.0-7.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/findutils-4.7.0-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "fuse-0__2.9.9-10.fc33.x86_64",
    sha256 = "4506efd1efbe7df7ace842060b3ecc0e53d182650a2ab56c1de1d91336430308",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-2.9.9-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-2.9.9-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-2.9.9-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/fuse-2.9.9-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "fuse-common-0__3.9.4-1.fc33.x86_64",
    sha256 = "7bd88b5035fb70ed35977a1b97fafd472aa2a044e54ea314eeb7960d1ed37975",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-common-3.9.4-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-common-3.9.4-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-common-3.9.4-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/fuse-common-3.9.4-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-10.fc33.aarch64",
    sha256 = "807a974476e61323941761828906340a483a6926324b920bc9a9b4434c0b82fa",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-10.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-10.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-10.fc33.x86_64",
    sha256 = "af6c6f788555064ff9c7d3b32b2d4edde5e33e958384a909459ce33940755971",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gawk-0__5.1.0-2.fc33.aarch64",
    sha256 = "03fc2036ddf506103dde29e4cf42d7f7fccf1a644c5314a6ac7d0b52453065bc",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/g/gawk-5.1.0-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gawk-5.1.0-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/g/gawk-5.1.0-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gawk-5.1.0-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gawk-0__5.1.0-2.fc33.x86_64",
    sha256 = "eeb4165863f2c905e81eaace836a808dd9be4ee3fd2aab70c8fa3ea8e499c300",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gawk-5.1.0-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gawk-5.1.0-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gawk-5.1.0-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/gawk-5.1.0-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gdbm-libs-1__1.19-1.fc33.aarch64",
    sha256 = "d2ca8c325b6df45021cef977a1bdbd8a956ff75c8a5ab0d9ae808f337fe89dcc",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/gdbm-libs-1.19-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/gdbm-libs-1.19-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/gdbm-libs-1.19-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/gdbm-libs-1.19-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gdbm-libs-1__1.19-1.fc33.x86_64",
    sha256 = "84d164b25063c542596dd28569d96b13959d202a237094283617daee998bbc68",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gdbm-libs-1.19-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gdbm-libs-1.19-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gdbm-libs-1.19-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gdbm-libs-1.19-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gdisk-0__1.0.8-1.fc33.x86_64",
    sha256 = "36dd01fe33b27ab5849cc54e143d63d1c081265cc25f1448136a9130fbd9b3c9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gdisk-1.0.8-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gdisk-1.0.8-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gdisk-1.0.8-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gdisk-1.0.8-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "genisoimage-0__1.1.11-46.fc33.x86_64",
    sha256 = "7c9ac0bd2bbccfa507c2a3d3d7a9febf6e68f9750d56795d40993134723de4ef",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-46.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-46.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-46.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-46.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gettext-0__0.21-3.fc33.aarch64",
    sha256 = "3dc1b908b2350f21d9423eb8efd3362d574cf057c310b39b09572f82bbd2e4bc",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/g/gettext-0.21-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gettext-0.21-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/g/gettext-0.21-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gettext-0.21-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gettext-0__0.21-3.fc33.x86_64",
    sha256 = "41bb26740843bb610a6e4415a61f82e448a206a83016099f8bab48e63c090dca",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gettext-0.21-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gettext-0.21-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gettext-0.21-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/gettext-0.21-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-3.fc33.aarch64",
    sha256 = "92073a99b21c39ed235f30021380fe3aac063f4fe5f431b9c9ccfbc3600c1b2a",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/g/gettext-libs-0.21-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gettext-libs-0.21-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/g/gettext-libs-0.21-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gettext-libs-0.21-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-3.fc33.x86_64",
    sha256 = "fc0ec75b2c135ba8a382c9b1f24e25bfb687f6ff7168e8d1e0ac40b6f73cb4ea",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gettext-libs-0.21-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gettext-libs-0.21-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gettext-libs-0.21-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/gettext-libs-0.21-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "glib2-0__2.66.8-1.fc33.aarch64",
    sha256 = "fc7db32b3e22519b2a35effed02006ff4fba6adfc0eb155e930ac0c916326616",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/glib2-2.66.8-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/glib2-2.66.8-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/glib2-2.66.8-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/glib2-2.66.8-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "glib2-0__2.66.8-1.fc33.x86_64",
    sha256 = "fd8073319996557fc4d74f52f4f6e142424c3d6e3acbfdefb2eac02064999863",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glib2-2.66.8-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/glib2-2.66.8-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/glib2-2.66.8-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glib2-2.66.8-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "glibc-0__2.32-10.fc33.aarch64",
    sha256 = "70429f4e08de273090dba0bf335f8ee7a1e6573a03d0fd67b2c72235a9215402",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-2.32-10.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/glibc-2.32-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-2.32-10.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-2.32-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-0__2.32-10.fc33.x86_64",
    sha256 = "16ab0c5c5501e493a96f2b4d191621bf045afc45f4f0180ebfafe2e5e4100490",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-2.32-10.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-2.32-10.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/glibc-2.32-10.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-2.32-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "glibc-common-0__2.32-10.fc33.aarch64",
    sha256 = "0ea7d19c0802eb1d8e04147b141363205cbbc41579d8dee527ff09a87baf3b38",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-common-2.32-10.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/glibc-common-2.32-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-common-2.32-10.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-common-2.32-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-common-0__2.32-10.fc33.x86_64",
    sha256 = "e9198a5dea87477a2c35ed51bbffcd3560f2282f5fc08b278847546ba9e2c771",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-common-2.32-10.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-common-2.32-10.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/glibc-common-2.32-10.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-common-2.32-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "glibc-langpack-en-0__2.32-10.fc33.aarch64",
    sha256 = "01a725ad507637e29ba601acd964df5a8894a45ae91480b2dd3d4cf51b0aa320",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-langpack-en-2.32-10.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/glibc-langpack-en-2.32-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-langpack-en-2.32-10.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/glibc-langpack-en-2.32-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-langpack-en-0__2.32-10.fc33.x86_64",
    sha256 = "b920b532289833b043aef05d9ddfc3568cba8039ddea85aeea27900321a359fc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-langpack-en-2.32-10.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-langpack-en-2.32-10.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/glibc-langpack-en-2.32-10.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/glibc-langpack-en-2.32-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gmp-1__6.2.0-5.fc33.aarch64",
    sha256 = "22311bb1367441335ce7c18a9bd243979a1998daf0b9e52f858391c26f48916e",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/g/gmp-6.2.0-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gmp-6.2.0-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/g/gmp-6.2.0-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gmp-6.2.0-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gmp-1__6.2.0-5.fc33.x86_64",
    sha256 = "159a52ff2593a73b64b7c2b14720fbf55786a871b698e4267c468f38a0dabb4c",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gmp-6.2.0-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gmp-6.2.0-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gmp-6.2.0-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/gmp-6.2.0-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gnupg2-0__2.2.25-2.fc33.x86_64",
    sha256 = "e252724e4abca4e2a715bab5c75f32b6f4e3b0e2518348014d28d31559d06b68",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnupg2-2.2.25-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnupg2-2.2.25-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gnupg2-2.2.25-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnupg2-2.2.25-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.6.16-1.fc33.aarch64",
    sha256 = "905b5a2c8895f06d4608d595dc5a53c11ebaec133caed9f765c749e163986df8",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-3.6.16-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/gnutls-3.6.16-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-3.6.16-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-3.6.16-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.6.16-1.fc33.x86_64",
    sha256 = "2180eefdec6a7e641fca23366f820e7ac25251cc3b2025dbde6ddfe4acf044a3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-3.6.16-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-3.6.16-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gnutls-3.6.16-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-3.6.16-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.16-1.fc33.aarch64",
    sha256 = "02fba17b27e862b5e02bafd5f2308a440bbc1bffb2a25b5fe6f884d6a0d8d91a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-dane-3.6.16-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/gnutls-dane-3.6.16-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-dane-3.6.16-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-dane-3.6.16-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.16-1.fc33.x86_64",
    sha256 = "c7f8a1ad471d4780d4f52795c893cd3b3f2d9373752b052bf942f769abf20675",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-dane-3.6.16-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-dane-3.6.16-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gnutls-dane-3.6.16-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-dane-3.6.16-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.16-1.fc33.aarch64",
    sha256 = "c57855db376f682d0d6c8db0431ce614d098af83ad1e112a00108189cc9dd6eb",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-utils-3.6.16-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/gnutls-utils-3.6.16-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-utils-3.6.16-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/gnutls-utils-3.6.16-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.16-1.fc33.x86_64",
    sha256 = "933d8048a32c2ff96c4271106c344b2e6d2244ba7f4531511a52b9ade4ea6aa2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-utils-3.6.16-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-utils-3.6.16-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gnutls-utils-3.6.16-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gnutls-utils-3.6.16-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gperftools-libs-0__2.8.1-1.fc33.aarch64",
    sha256 = "dc7b9de2314ba469cbce71ec2f690f163aab3b8de40fb684b5d156065eb039bd",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/gperftools-libs-2.8.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/gperftools-libs-2.8.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/gperftools-libs-2.8.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/gperftools-libs-2.8.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gperftools-libs-0__2.8.1-1.fc33.x86_64",
    sha256 = "63fc4325aa7ca83aba17d6f5a8d4504924290b770f84410d0d910958f38f86d8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gperftools-libs-2.8.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/gperftools-libs-2.8.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/gperftools-libs-2.8.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/gperftools-libs-2.8.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gpgme-0__1.14.0-2.fc33.x86_64",
    sha256 = "35f883f8d430eb53d5e14745889a612398a959f7e35de81341416fd0012b04e7",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gpgme-1.14.0-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gpgme-1.14.0-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gpgme-1.14.0-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/gpgme-1.14.0-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "grep-0__3.4-6.fc33.aarch64",
    sha256 = "29c9d1e2e1f954c30103acf4f65af9ca705c0ce3d8f404bbac091322288355ae",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/grep-3.4-6.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/grep-3.4-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/grep-3.4-6.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/grep-3.4-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "grep-0__3.4-6.fc33.x86_64",
    sha256 = "498f3a08ee5bfd9077b893f34e1a63bc12ab526bf4cc7f36c87af22e63e710d5",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/grep-3.4-6.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/grep-3.4-6.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/grep-3.4-6.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/grep-3.4-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "groff-base-0__1.22.4-4.fc33.aarch64",
    sha256 = "dd8406dd92f69856f3f5477658a09169847ff8ac45907cf8ddd1b18cdefce5c2",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/g/groff-base-1.22.4-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/g/groff-base-1.22.4-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/g/groff-base-1.22.4-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/g/groff-base-1.22.4-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "groff-base-0__1.22.4-4.fc33.x86_64",
    sha256 = "fc503ed9739391a7c4de7e45ab86893f234383ef379315000d95a8fe23802370",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/groff-base-1.22.4-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/g/groff-base-1.22.4-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/g/groff-base-1.22.4-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/g/groff-base-1.22.4-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "gzip-0__1.10-3.fc33.aarch64",
    sha256 = "1e86196c96925970800643fdbbdc4960efd00ad5607651e4b261e153fb207c74",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/g/gzip-1.10-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gzip-1.10-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/g/gzip-1.10-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/g/gzip-1.10-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "gzip-0__1.10-3.fc33.x86_64",
    sha256 = "c8d043738df1538d58276fb8279a03bb50faee33ec1c2e87116ab5cc5327ea9a",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gzip-1.10-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gzip-1.10-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/g/gzip-1.10-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/g/gzip-1.10-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "hexedit-0__1.2.13-18.fc33.x86_64",
    sha256 = "b5d7e48ed92684e2cbbd07a1e7dbbb8656b266f9c02c053a7bc1c486cdefabd3",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/h/hexedit-1.2.13-18.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/h/hexedit-1.2.13-18.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/h/hexedit-1.2.13-18.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/h/hexedit-1.2.13-18.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "hivex-0__1.3.20-1.fc33.x86_64",
    sha256 = "a60957cd6605035b1661fdad5b2345727f652d7c37011bc49480eb33340ae558",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/h/hivex-1.3.20-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/h/hivex-1.3.20-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/h/hivex-1.3.20-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/h/hivex-1.3.20-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "hwdata-0__0.349-1.fc33.aarch64",
    sha256 = "bef8a5c28dc2c9cb7fb7a71092c3c1d54cffd19f701ab02329559786409056cb",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "hwdata-0__0.349-1.fc33.x86_64",
    sha256 = "bef8a5c28dc2c9cb7fb7a71092c3c1d54cffd19f701ab02329559786409056cb",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/h/hwdata-0.349-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "ima-evm-utils-0__1.3.2-1.fc33.x86_64",
    sha256 = "5821e5fdcd8f622a8197b6840d3d20f27eb51aee3299576e11b69bb2411964e1",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/ima-evm-utils-1.3.2-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/i/ima-evm-utils-1.3.2-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/i/ima-evm-utils-1.3.2-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/ima-evm-utils-1.3.2-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ipcalc-0__0.4.1-2.fc33.x86_64",
    sha256 = "7cf59e66b948e4cb70fcebae01b2f43b57ccb17d530c9da13fd683d592f7c4ca",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/i/ipcalc-0.4.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/i/ipcalc-0.4.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/i/ipcalc-0.4.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/i/ipcalc-0.4.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "iproute-0__5.9.0-1.fc33.aarch64",
    sha256 = "693b21eb4a97d4542dabecda6a2550a87e2edf704c1225d8d2c58d2dd1a32209",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "iproute-0__5.9.0-1.fc33.x86_64",
    sha256 = "73a086245b54c933bde244861fb37ad52a129fc99031a44da4f226c09e3c08c8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "iproute-tc-0__5.9.0-1.fc33.aarch64",
    sha256 = "72111e38fa2a82175e65aafdb819ee842082097dad51ae4616bc473d4c59d601",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "iproute-tc-0__5.9.0-1.fc33.x86_64",
    sha256 = "fbb9428ebe82db38e485f2e4e4b9f48a3fe8ccc8627137336906a9f033dec0b6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "iptables-0__1.8.5-6.fc33.aarch64",
    sha256 = "3a268ee172cf5001a0d2cd9c995baf6eef5784f648d4c2406e6fecd83ae06595",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/i/iptables-1.8.5-6.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/i/iptables-1.8.5-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/i/iptables-1.8.5-6.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/i/iptables-1.8.5-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "iptables-0__1.8.5-6.fc33.x86_64",
    sha256 = "c9304b83d784806c45e739a5ff55c1c0f256309c8eafc74afcee516f4decaaa6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iptables-1.8.5-6.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/i/iptables-1.8.5-6.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/i/iptables-1.8.5-6.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iptables-1.8.5-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.5-6.fc33.aarch64",
    sha256 = "5d141b538bcb249a9beaa70247dd06177546208960b46cb125e5b5e8548edaa0",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/i/iptables-libs-1.8.5-6.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/i/iptables-libs-1.8.5-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/i/iptables-libs-1.8.5-6.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/i/iptables-libs-1.8.5-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.5-6.fc33.x86_64",
    sha256 = "23864097e94e18b7f7429522aa869c872ff284cb0d63c24da3742b7e9e54de1e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iptables-libs-1.8.5-6.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/i/iptables-libs-1.8.5-6.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/i/iptables-libs-1.8.5-6.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iptables-libs-1.8.5-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "iputils-0__20200821-1.fc33.aarch64",
    sha256 = "9195691c2bdcf46c8908a832e8e270b7c22913c959e5941efcfc7fd3b424dbce",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/i/iputils-20200821-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/i/iputils-20200821-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/i/iputils-20200821-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/i/iputils-20200821-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "iputils-0__20200821-1.fc33.x86_64",
    sha256 = "9ab24d7a66316a8b15ed3fc63713b97630e2c2172ce23014ae73131de87ed4cc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iputils-20200821-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/i/iputils-20200821-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/i/iputils-20200821-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/i/iputils-20200821-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ipxe-roms-qemu-0__20200823-1.git4bd064de.fc33.x86_64",
    sha256 = "22eb521e2287314dc681aa4add51ffe33038bdf5f174395b76f2876e1a262011",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20200823-1.git4bd064de.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20200823-1.git4bd064de.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20200823-1.git4bd064de.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20200823-1.git4bd064de.fc33.noarch.rpm",
    ],
)

rpm(
    name = "jansson-0__2.13.1-1.fc33.aarch64",
    sha256 = "7d19769cfebaf42f2b38c82c69f014c20038323c3fb95aaafb8f92ad1754fcc1",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/j/jansson-2.13.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/j/jansson-2.13.1-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/j/jansson-2.13.1-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/j/jansson-2.13.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "jansson-0__2.13.1-1.fc33.x86_64",
    sha256 = "c2ac735bec37389cacbeaf08493f155414925af91e91c734d6cc34bef47be83a",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/j/jansson-2.13.1-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/j/jansson-2.13.1-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/j/jansson-2.13.1-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/j/jansson-2.13.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "json-c-0__0.14-7.fc33.aarch64",
    sha256 = "eaa03a6585283ff62c349c45308624505f6df04a1c2cd1811788bdaa5bae0f4f",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/j/json-c-0.14-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/j/json-c-0.14-7.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/j/json-c-0.14-7.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/j/json-c-0.14-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "json-c-0__0.14-7.fc33.x86_64",
    sha256 = "bacb143e31174848a89c9a2a84421c2d66a45fa1e7272a7e5c18f624e3316750",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/j/json-c-0.14-7.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/j/json-c-0.14-7.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/j/json-c-0.14-7.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/j/json-c-0.14-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "json-glib-0__1.6.2-1.fc33.aarch64",
    sha256 = "446a4d25f249d7e737afaab5297bdabf8f1441029317cb15f69642d96c5e05d4",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/j/json-glib-1.6.2-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/j/json-glib-1.6.2-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/j/json-glib-1.6.2-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/j/json-glib-1.6.2-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "json-glib-0__1.6.2-1.fc33.x86_64",
    sha256 = "da06a19ffa5c535fe02f915f18145c880a42f063d96bb174063dcc4c0bd170e8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/j/json-glib-1.6.2-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/j/json-glib-1.6.2-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/j/json-glib-1.6.2-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/j/json-glib-1.6.2-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "kbd-0__2.3.0-2.fc33.x86_64",
    sha256 = "8772a0ee3f0e3c8fb2917b7286af74f07d49a5741ba2b7ab0fa4c90934b10902",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-2.3.0-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-2.3.0-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-2.3.0-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/k/kbd-2.3.0-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "kbd-legacy-0__2.3.0-2.fc33.x86_64",
    sha256 = "369bbe9d84cf172759a50311ef8aae03e53b704eca9f6b88349b13261426a398",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-legacy-2.3.0-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-legacy-2.3.0-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-legacy-2.3.0-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/k/kbd-legacy-2.3.0-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "kbd-misc-0__2.3.0-2.fc33.x86_64",
    sha256 = "b285c51c3bee3dfd38e2aa9f39086e27d84d5708060e71c6923b4e70ccbd6495",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-misc-2.3.0-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-misc-2.3.0-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kbd-misc-2.3.0-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/k/kbd-misc-2.3.0-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "kde-filesystem-0__4-64.fc33.aarch64",
    sha256 = "e01c5438a4a2a2d5f0c05749405b8748655edf8befdd1323c7df28e5be149984",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/k/kde-filesystem-4-64.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/k/kde-filesystem-4-64.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/k/kde-filesystem-4-64.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/k/kde-filesystem-4-64.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "kde-filesystem-0__4-64.fc33.x86_64",
    sha256 = "b0a7b263a7449a3c22ddd21269ee1efed7d2d41f99ca1bf18a190d98094b0668",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kde-filesystem-4-64.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kde-filesystem-4-64.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/k/kde-filesystem-4-64.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/k/kde-filesystem-4-64.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "kernel-debug-core-0__5.13.6-100.fc33.x86_64",
    sha256 = "eff633e658fc521276bcab118359ae6b898e24c4da8f2d95ad798066a9897ad4",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/k/kernel-debug-core-5.13.6-100.fc33.x86_64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/k/kernel-debug-core-5.13.6-100.fc33.x86_64.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/k/kernel-debug-core-5.13.6-100.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/k/kernel-debug-core-5.13.6-100.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.1-1.fc33.aarch64",
    sha256 = "3c5a545e68dcc34fac9654d906a6f863a1d79afc37da806d50fea86875ecd9db",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.1-1.fc33.x86_64",
    sha256 = "e3a71188d243474792a20b995c2f58fd3bde23a25da85fcda757faf7b05217e1",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "kf5-filesystem-0__5.79.0-1.fc33.aarch64",
    sha256 = "73f7f94055404a81eeffc997f18c1014a44a66c051cd5722b4ed008142ab87e6",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/k/kf5-filesystem-5.79.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/k/kf5-filesystem-5.79.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/k/kf5-filesystem-5.79.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/k/kf5-filesystem-5.79.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "kf5-filesystem-0__5.79.0-1.fc33.x86_64",
    sha256 = "e442624ce86743d334f3272024722e4f38c972c6a7e9612be4262d97fbc43cfd",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/kf5-filesystem-5.79.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/k/kf5-filesystem-5.79.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/k/kf5-filesystem-5.79.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/kf5-filesystem-5.79.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "kmod-0__29-2.fc33.aarch64",
    sha256 = "c6cfce9e3b0f5a2d1da30b144019356470f16ccc1a873e62d14dcd4ce8e1348b",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/k/kmod-29-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/k/kmod-29-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/k/kmod-29-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/k/kmod-29-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "kmod-0__29-2.fc33.x86_64",
    sha256 = "4efb633d935fb86b96204ff8f01bbfc20e86ec1ea315feedb419d9d6b48cf24b",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/kmod-29-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/k/kmod-29-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/k/kmod-29-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/kmod-29-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "kmod-libs-0__29-2.fc33.aarch64",
    sha256 = "a3159b2822abb5e2bdd95673006fa71b77c9e93b5cf1617e71dfa12e5e8324db",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/k/kmod-libs-29-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/k/kmod-libs-29-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/k/kmod-libs-29-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/k/kmod-libs-29-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "kmod-libs-0__29-2.fc33.x86_64",
    sha256 = "8779b1377d020ff7e6d0064a5d10d1849d459363ae27cd2f806463c63132230a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/kmod-libs-29-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/k/kmod-libs-29-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/k/kmod-libs-29-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/kmod-libs-29-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-30.fc33.aarch64",
    sha256 = "b0372dda989b9332964316063568647a4fc8ef617a978ee48c1e55d3eec3fa7d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/k/krb5-libs-1.18.2-30.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/k/krb5-libs-1.18.2-30.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/k/krb5-libs-1.18.2-30.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/k/krb5-libs-1.18.2-30.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-30.fc33.x86_64",
    sha256 = "74e625533760c3f3d8d7249367f3a536a7ce5b61e39f33569ec0006ccfa877ed",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/krb5-libs-1.18.2-30.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/k/krb5-libs-1.18.2-30.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/k/krb5-libs-1.18.2-30.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/k/krb5-libs-1.18.2-30.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "less-0__551-4.fc33.x86_64",
    sha256 = "d835bbf3799b4514447cdffff2367e2352828e61902346fe8575e1bba132a540",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/less-551-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/less-551-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/less-551-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/less-551-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libacl-0__2.2.53-9.fc33.aarch64",
    sha256 = "b8f488c703052104819b2b65127554a307c15d171348d7fa9ef0f43c987e57f8",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libacl-2.2.53-9.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libacl-2.2.53-9.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libacl-2.2.53-9.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libacl-2.2.53-9.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libacl-0__2.2.53-9.fc33.x86_64",
    sha256 = "5f7479b7577de892f42e4492ebe5674fdfff19d14aaa800db8b18162853e15b0",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libacl-2.2.53-9.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libacl-2.2.53-9.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libacl-2.2.53-9.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libacl-2.2.53-9.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libaio-0__0.3.111-10.fc33.aarch64",
    sha256 = "a2f2ee3465c4495e1b4f10c9dad5dacc9e9679cc8d1153cf8155066ae56303db",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libaio-0.3.111-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libaio-0.3.111-10.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libaio-0.3.111-10.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libaio-0.3.111-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libaio-0__0.3.111-10.fc33.x86_64",
    sha256 = "51ae3b86c7a6fd64ed187574b3a0a7e3a58f533a6db80e3bf44be99f5fd72f50",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libaio-0.3.111-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libaio-0.3.111-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libaio-0.3.111-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libaio-0.3.111-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libarchive-0__3.5.1-1.fc33.aarch64",
    sha256 = "000f6c4eb00af6310ffef067c788ad9ca7bc48d79c96e02a7736e389b5f5bb02",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libarchive-3.5.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libarchive-3.5.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libarchive-3.5.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libarchive-3.5.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libarchive-0__3.5.1-1.fc33.x86_64",
    sha256 = "dd56b15f902451ea6db11147eeb0193f1b44cbf4b35894a4dfd38e28a0b9c04d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libarchive-3.5.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libarchive-3.5.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libarchive-3.5.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libarchive-3.5.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libargon2-0__20171227-5.fc33.aarch64",
    sha256 = "45e09aff0e5eec9566bcca5bfc97c72c4f5ea66f1fad97a3ec314714d3144ebc",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libargon2-20171227-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libargon2-20171227-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libargon2-20171227-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libargon2-20171227-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libargon2-0__20171227-5.fc33.x86_64",
    sha256 = "f87a7db3ba17f6cd201de31b73768c93b4679bee33a97507723dc0eaed373f50",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libargon2-20171227-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libargon2-20171227-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libargon2-20171227-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libargon2-20171227-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libassuan-0__2.5.3-4.fc33.x86_64",
    sha256 = "974486fc5c90c575512df8964302b42cf912fcb1d588a931c628f2a49469468f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libassuan-2.5.3-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libassuan-2.5.3-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libassuan-2.5.3-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libassuan-2.5.3-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libattr-0__2.4.48-10.fc33.aarch64",
    sha256 = "b13579c92235fbefb8ae93bb1c8224f622991334b73dcacc9f1b727728792ef4",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libattr-2.4.48-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libattr-2.4.48-10.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libattr-2.4.48-10.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libattr-2.4.48-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libattr-0__2.4.48-10.fc33.x86_64",
    sha256 = "99f27025aedb0cd4a652f4a42bb176122253d6522e7e5ff5d162dd3a787ca135",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libattr-2.4.48-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libattr-2.4.48-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libattr-2.4.48-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libattr-2.4.48-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libblkid-0__2.36.1-1.fc33.aarch64",
    sha256 = "311015bbb160a705b019ff4230175dc6b2179e5a97228bad4bbe52deac0db523",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libblkid-2.36.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libblkid-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libblkid-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libblkid-2.36.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libblkid-0__2.36.1-1.fc33.x86_64",
    sha256 = "bbe3024eff73efafd4fb0100c28b2c8a51e54df0cc74e87493b9cbe78e837b3a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libblkid-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libblkid-2.36.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libblkid-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libblkid-2.36.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libburn-0__1.5.4-2.fc33.aarch64",
    sha256 = "bbcd3617cb21039799f110b41bca985fc5d53a88e7cefe83628cef4f56e9ef76",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libburn-0__1.5.4-2.fc33.x86_64",
    sha256 = "6c0148821faf182d642084a46c9d530c2f6b947b8bbc3700622f08f4c10b652c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libcap-0__2.48-2.fc33.aarch64",
    sha256 = "52effcf793fd68f4218830e1945748af0fd12d46b5457ce55923bba3e22f1775",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libcap-2.48-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libcap-2.48-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libcap-2.48-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libcap-2.48-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libcap-0__2.48-2.fc33.x86_64",
    sha256 = "faf6aa22a112309fd9b751ea9da0e185126df6b3aa10accb1039eb3ad6af6813",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcap-2.48-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcap-2.48-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libcap-2.48-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcap-2.48-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libcap-ng-0__0.8-1.fc33.aarch64",
    sha256 = "70372612cc83892498fbae3065dd42f39db6696d528e9e267035a8b2260ec502",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libcap-ng-0.8-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libcap-ng-0.8-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libcap-ng-0.8-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libcap-ng-0.8-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libcap-ng-0__0.8-1.fc33.x86_64",
    sha256 = "c33cf40de2cdb38c36f830b0fcbca1ee89984c706682677d45f5bfe436bf4010",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcap-ng-0.8-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcap-ng-0.8-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcap-ng-0.8-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libcap-ng-0.8-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libcbor-0__0.5.0-7.fc33.aarch64",
    sha256 = "a8cc95d9c1e34195a6a44b2859def2f28634d64cfb13c7b1117ac24730cfee9d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libcbor-0.5.0-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libcbor-0.5.0-7.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libcbor-0.5.0-7.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libcbor-0.5.0-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libcbor-0__0.5.0-7.fc33.x86_64",
    sha256 = "d15f02eb237a82e87f8e0709adccd09be7ca0d20e5d1df68810bc90a7fc211c9",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcbor-0.5.0-7.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcbor-0.5.0-7.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcbor-0.5.0-7.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libcbor-0.5.0-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libcom_err-0__1.45.6-4.fc33.aarch64",
    sha256 = "891241a801785e731e7c59caa4927739e24fd1265108ba2a9fe4fd788aa44012",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libcom_err-1.45.6-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libcom_err-1.45.6-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libcom_err-1.45.6-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libcom_err-1.45.6-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libcom_err-0__1.45.6-4.fc33.x86_64",
    sha256 = "a2ab34c05b4d64c156a1083c92b595113a1576680d7ec7a59797fa89ef09b45c",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcom_err-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcom_err-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libcom_err-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libcom_err-1.45.6-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libcomps-0__0.1.17-1.fc33.x86_64",
    sha256 = "e02636032fc71da2d96ad0d4fe6858fef501247f9a8dbfcebaa05aba03bbfe9d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcomps-0.1.17-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcomps-0.1.17-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libcomps-0.1.17-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcomps-0.1.17-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libconfig-0__1.7.2-6.fc33.x86_64",
    sha256 = "0b54eaa87839ff993d14002db810bd292a6983de17171b3c36636d26768d56c1",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libconfig-1.7.2-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libconfig-1.7.2-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libconfig-1.7.2-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libconfig-1.7.2-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.71.1-9.fc33.aarch64",
    sha256 = "6621ff963618a0a2546331eeae0f0a768de0d44f991d8d0500f997f5194d0cc8",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libcurl-minimal-7.71.1-9.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libcurl-minimal-7.71.1-9.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libcurl-minimal-7.71.1-9.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libcurl-minimal-7.71.1-9.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.71.1-9.fc33.x86_64",
    sha256 = "d25b5a1b27837fb6f747683168e7b813cf188a72aa442adff82b249438c88bda",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcurl-minimal-7.71.1-9.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcurl-minimal-7.71.1-9.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libcurl-minimal-7.71.1-9.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libcurl-minimal-7.71.1-9.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libdb-0__5.3.28-45.fc33.aarch64",
    sha256 = "777e5481c5160e0892407b490ba7fba5ebd9ce6da68bf901523af1c1fbab978a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libdb-5.3.28-45.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libdb-5.3.28-45.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libdb-5.3.28-45.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libdb-5.3.28-45.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libdb-0__5.3.28-45.fc33.x86_64",
    sha256 = "23a28b6de28b3640d40ffb50ed4c5a9a9491df18e4160392065c355b389ab473",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdb-5.3.28-45.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdb-5.3.28-45.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libdb-5.3.28-45.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdb-5.3.28-45.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libdb-utils-0__5.3.28-45.fc33.x86_64",
    sha256 = "a2c27edb856dd8320bd3cdfc5b89c4de522120b45d3c10a1636a86accf51453d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdb-utils-5.3.28-45.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdb-utils-5.3.28-45.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libdb-utils-5.3.28-45.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdb-utils-5.3.28-45.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libdnf-0__0.63.1-1.fc33.x86_64",
    sha256 = "58245ff4d349b18c0e9622e4a83034a2702ee89d1309d39777277844f61294f1",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdnf-0.63.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdnf-0.63.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libdnf-0.63.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libdnf-0.63.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libeconf-0__0.4.0-1.fc33.aarch64",
    sha256 = "8d1d14128e2c29d3965323b9029eb07290e53c53a53793489d1a72381b16588a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libeconf-0.4.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libeconf-0.4.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libeconf-0.4.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libeconf-0.4.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libeconf-0__0.4.0-1.fc33.x86_64",
    sha256 = "dd0f73e6d70766a916ec1ddfda13f5615060dc2cc47e249d609b064b24ced8ec",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libeconf-0.4.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libeconf-0.4.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libeconf-0.4.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libeconf-0.4.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libevent-0__2.1.8-10.fc33.aarch64",
    sha256 = "238146c95630843041b3f078e4eda1d78d8eea8af4132a5c1cde7ad76433a5fd",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libevent-2.1.8-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libevent-2.1.8-10.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libevent-2.1.8-10.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libevent-2.1.8-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libevent-0__2.1.8-10.fc33.x86_64",
    sha256 = "d9af737a48ee0d8cc03dd0fb18e576b9829471ba45f161d29fe41b071ea3190d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libevent-2.1.8-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libevent-2.1.8-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libevent-2.1.8-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libevent-2.1.8-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libfdisk-0__2.36.1-1.fc33.aarch64",
    sha256 = "a7a50c6bff1871e5f8ead1e67cfef87130a267c0f48aaf9cebc878fabbc182cd",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libfdisk-2.36.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libfdisk-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libfdisk-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libfdisk-2.36.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libfdisk-0__2.36.1-1.fc33.x86_64",
    sha256 = "343d6f2e2b5b65f46bc52d2a4b05fe138bdf30ad8874916ce5873d2fad91ea83",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libfdisk-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libfdisk-2.36.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libfdisk-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libfdisk-2.36.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libfdt-0__1.6.0-3.fc33.aarch64",
    sha256 = "1bcd35cce257dfbda39783568ef92086173c4794a57bc69aa339a08f853c7f7d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libfdt-1.6.0-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libfdt-1.6.0-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libfdt-1.6.0-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libfdt-1.6.0-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libffi-0__3.1-26.fc33.aarch64",
    sha256 = "aeb3ad6cea3959372b60ccc543e9b48e007d8e4fd66e911269e971ba5d00529b",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libffi-3.1-26.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libffi-3.1-26.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libffi-3.1-26.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libffi-3.1-26.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libffi-0__3.1-26.fc33.x86_64",
    sha256 = "b877711628f940c9faef140a34a5227f6fb428f505a13048b786a28a16d7d24c",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libffi-3.1-26.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libffi-3.1-26.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libffi-3.1-26.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libffi-3.1-26.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libfido2-0__1.4.0-3.fc33.aarch64",
    sha256 = "0ea73f7d31338b8e5c766791d4d01ff2a9f77087a774e99c8e0b5d573a44444b",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libfido2-1.4.0-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libfido2-1.4.0-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libfido2-1.4.0-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libfido2-1.4.0-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libfido2-0__1.4.0-3.fc33.x86_64",
    sha256 = "aa02e6efd3cfcf909a52fef4615ef47517264ae0f0802ab3787753a05d923698",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libfido2-1.4.0-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libfido2-1.4.0-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libfido2-1.4.0-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libfido2-1.4.0-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libgcc-0__10.3.1-1.fc33.aarch64",
    sha256 = "dd31ca3879f18e7d12d5d5e82fbea09f217de885f9889dbb5978423a02ad091e",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libgcc-0__10.3.1-1.fc33.x86_64",
    sha256 = "9a4e1ef5561bcf98486fa4b742b0cadfa69f8e226392d52418c3041055fe34aa",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.8-1.fc33.aarch64",
    sha256 = "765bc633c08e11526800035989fa7900b72a8f16f0e681914e66dab7a0d488a0",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgcrypt-1.8.8-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libgcrypt-1.8.8-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgcrypt-1.8.8-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgcrypt-1.8.8-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.8-1.fc33.x86_64",
    sha256 = "42b1f9f75082b55f71866621fa246050f35f805a20c18b04dbb543ba60b68c56",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgcrypt-1.8.8-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgcrypt-1.8.8-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libgcrypt-1.8.8-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgcrypt-1.8.8-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libgomp-0__10.3.1-1.fc33.aarch64",
    sha256 = "3b72eb01422e5c688aa35a97f28fb300aac11cca704dce3bec6ea40ef6f7d932",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libgomp-0__10.3.1-1.fc33.x86_64",
    sha256 = "19cbab143370bd6ba28d2734ad74b9cc1c27dd10165f104cad707a9a4d6068fa",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libgpg-error-0__1.41-1.fc33.aarch64",
    sha256 = "a18ec50ff9fc7ef1863c61bbb580179f50967a2b3f607f201b9a6a835ebc754d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgpg-error-1.41-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libgpg-error-1.41-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgpg-error-1.41-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libgpg-error-1.41-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libgpg-error-0__1.41-1.fc33.x86_64",
    sha256 = "b6e2da9808d1b74cb75e0f4bd912fd46c869d9a56ebb3123e0008ff4faf805ab",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgpg-error-1.41-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgpg-error-1.41-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libgpg-error-1.41-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libgpg-error-1.41-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libguestfs-1__1.44.1-1.fc33.x86_64",
    sha256 = "bc7178755c8417e6f887f078e2d7bcc40822fabe120cf172d16349a1c2d061e7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-1.44.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-1.44.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-1.44.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-1.44.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libguestfs-tools-1__1.44.1-1.fc33.x86_64",
    sha256 = "00f7ff6d324b64fc02c941e902a3a1556144932092be5b73cfd09fc440726aa3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-1.44.1-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-1.44.1-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-1.44.1-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-1.44.1-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "libguestfs-tools-c-1__1.44.1-1.fc33.x86_64",
    sha256 = "98298a1a21237ee3ff6f7f7d63138d7b7f5e67ce7f8880cf61d6133730a053eb",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libibverbs-0__35.0-1.fc33.aarch64",
    sha256 = "b4aa2e3b7c157a89b5496491765a3f993fc262a8e0c1bfd2c179bd17c96587ee",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libibverbs-35.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libibverbs-35.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libibverbs-35.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libibverbs-35.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libibverbs-0__35.0-1.fc33.x86_64",
    sha256 = "e9e6d6c08798754d9e224c3a11c98f084ba75c8c03fecfacdaee1eae4db3f40e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libibverbs-35.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libibverbs-35.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libibverbs-35.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libibverbs-35.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libidn2-0__2.3.2-1.fc33.aarch64",
    sha256 = "4047cb30a12e5305563b78039e8f097f823e8f4eefa7ade05c8b5767cb2c3ce0",
    urls = [
        "https://mirrors.xtom.ee/fedora/updates/33/Everything/aarch64/Packages/l/libidn2-2.3.2-1.fc33.aarch64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/updates/33/Everything/aarch64/Packages/l/libidn2-2.3.2-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libidn2-2.3.2-1.fc33.aarch64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/aarch64/Packages/l/libidn2-2.3.2-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libidn2-0__2.3.2-1.fc33.x86_64",
    sha256 = "9100d00a12573a3b6745a57fb2265e75adea5389dabf6cad6a1d6852923ac39b",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/l/libidn2-2.3.2-1.fc33.x86_64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/l/libidn2-2.3.2-1.fc33.x86_64.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/l/libidn2-2.3.2-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libidn2-2.3.2-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-2.fc33.aarch64",
    sha256 = "3bcb5f60d1eff2b49106b052cb9a0f8e511a27d64a0a45447e6166847e2646e2",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-2.fc33.x86_64",
    sha256 = "51990e518dcf11395812563d37017ac3c17811a1f5f639d4ad7ba9d754ca150c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libisofs-0__1.5.4-1.fc33.aarch64",
    sha256 = "18621f4003887f6aed0a366aefdf7ce127a0697f53b23e45007ea5c41016f5ab",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libisofs-0__1.5.4-1.fc33.x86_64",
    sha256 = "b2fb98a7c53ba93131ef9daace8caff65a8950210e236646d0de791d5ddc0e71",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libkcapi-0__1.2.1-1.fc33.x86_64",
    sha256 = "d3a7c8de03e69fd1b7b60f34adb4415022a464dbe4230a2cb171a86203ad0e49",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-1.2.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-1.2.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-1.2.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-1.2.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libkcapi-hmaccalc-0__1.2.1-1.fc33.x86_64",
    sha256 = "a09e28093b3e0122ddd9bba0d634b889c963e01851f9c8b726cc673e0f3eb398",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-hmaccalc-1.2.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-hmaccalc-1.2.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-hmaccalc-1.2.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libkcapi-hmaccalc-1.2.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libksba-0__1.3.5-13.fc33.x86_64",
    sha256 = "2f485beaa7a53ffbc350b47a3429e053e8b5953761ea5adfe2a45376a21a1842",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libksba-1.3.5-13.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libksba-1.3.5-13.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libksba-1.3.5-13.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libksba-1.3.5-13.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-12.fc33.aarch64",
    sha256 = "500d779a33341568f466a1eb8227b313ebcf09220928d3c650d8b2e6f074d57d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libmnl-1.0.4-12.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libmnl-1.0.4-12.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libmnl-1.0.4-12.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libmnl-1.0.4-12.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-12.fc33.x86_64",
    sha256 = "b6773a2567060a6b8ed602f442e4bb8ce6885b43616d7247d26fa1b0c8e8536f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libmnl-1.0.4-12.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libmnl-1.0.4-12.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libmnl-1.0.4-12.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libmnl-1.0.4-12.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libmodulemd-0__2.13.0-1.fc33.x86_64",
    sha256 = "b97f960ed6243733bfd16e14379509b051dc1cffc500b960c6271809a814978f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libmodulemd-2.13.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libmodulemd-2.13.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libmodulemd-2.13.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libmodulemd-2.13.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libmount-0__2.36.1-1.fc33.aarch64",
    sha256 = "9b7ab092d8a39b7e95f4b0ab8a509f7bd47cb23bb3192f9ae1f8b0114854159d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libmount-2.36.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libmount-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libmount-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libmount-2.36.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libmount-0__2.36.1-1.fc33.x86_64",
    sha256 = "e193d2a1e547d5d573b5585593e3946e01205d080f49880fd4640a0d9a146109",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libmount-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libmount-2.36.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libmount-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libmount-2.36.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.7-5.fc33.aarch64",
    sha256 = "591d0b233356db831ac6913028208a03397d7be0e363382493769f72d8494a33",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.7-5.fc33.x86_64",
    sha256 = "51a5b539067f77e16e569de900c2f013c648ca0b70b3aaea9bcab2f7b46b1fb7",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-18.fc33.aarch64",
    sha256 = "cf85816b6bea52547353342a7649a70155e9158f7e01cd02faf6ec0d847222ae",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-18.fc33.x86_64",
    sha256 = "b12a0d496eaf77f2ecc4a282857c756abdbc81de46249a9559abaa525d85c3d9",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-18.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libnftnl-0__1.1.7-3.fc33.aarch64",
    sha256 = "90216dbc020553dc56a49f9b272fcc8e78d1c1e7e78bd5983ac6fd8d5a50a6ec",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libnftnl-1.1.7-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnftnl-1.1.7-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libnftnl-1.1.7-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnftnl-1.1.7-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libnftnl-0__1.1.7-3.fc33.x86_64",
    sha256 = "984f215f7f0fe4961026939892ab651416899244ed2230b3aa4c82e18d7dfbed",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnftnl-1.1.7-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnftnl-1.1.7-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnftnl-1.1.7-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libnftnl-1.1.7-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-1.fc33.aarch64",
    sha256 = "0129a91d8281969f2cd35fc996d84a79eb30a3b34b68e2ce41d864034a2caddb",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libnghttp2-1.43.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libnghttp2-1.43.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libnghttp2-1.43.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libnghttp2-1.43.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-1.fc33.x86_64",
    sha256 = "690098a3fa9b1905d6cd0d0960574fd40653d508a7eb23294e42549c896178aa",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libnghttp2-1.43.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libnghttp2-1.43.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libnghttp2-1.43.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libnghttp2-1.43.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-5.fc33.aarch64",
    sha256 = "f5d74b77dc9ec7e2bc4436d2641834ccf69404b58786581732541ae4651b26a0",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libnl3-3.5.0-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnl3-3.5.0-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libnl3-3.5.0-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnl3-3.5.0-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-5.fc33.x86_64",
    sha256 = "d4d05d7acb8a093c1c05dfdd47689e126084ecb0ed3134e224f791a3c51aa982",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnl3-3.5.0-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnl3-3.5.0-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnl3-3.5.0-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libnl3-3.5.0-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-8.20180605git4a062cf.fc33.aarch64",
    sha256 = "1653f80966ac5485d134a63cc22fc28e6f160b4f8c365b2a3b56f4b1fc3a9009",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-8.20180605git4a062cf.fc33.x86_64",
    sha256 = "15a7e5a788e1f285bb3254638a7ed8159462508283df8079efb4699e17eed46d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-8.20180605git4a062cf.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libpcap-14__1.10.1-1.fc33.aarch64",
    sha256 = "ee06c2ef40938afb48e0fef4bbe6dad6b980b545365f75c77206846e2d5b9e41",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpcap-1.10.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libpcap-1.10.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpcap-1.10.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpcap-1.10.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libpcap-14__1.10.1-1.fc33.x86_64",
    sha256 = "74f5babfa4abaedc817336563ed3caa5eb0d6957a7fc57597a2b3d4609ba722b",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpcap-1.10.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpcap-1.10.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libpcap-1.10.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpcap-1.10.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libpipeline-0__1.5.2-5.fc33.x86_64",
    sha256 = "d8bd62cca42c062048078d622cf6a81570012f85ccaea2b3cbeb27053e2749ca",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libpipeline-1.5.2-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libpipeline-1.5.2-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libpipeline-1.5.2-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libpipeline-1.5.2-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libpkgconf-0__1.7.3-5.fc33.aarch64",
    sha256 = "7981f42013e26b9b34216ea06e5b68ca2950b67343729eff9373eea358760186",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpkgconf-1.7.3-5.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libpkgconf-1.7.3-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpkgconf-1.7.3-5.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpkgconf-1.7.3-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libpkgconf-0__1.7.3-5.fc33.x86_64",
    sha256 = "cb51366698228de723f6c5bd80608d9921d921712d13fb09c10392d4d2b54eca",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpkgconf-1.7.3-5.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpkgconf-1.7.3-5.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libpkgconf-1.7.3-5.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpkgconf-1.7.3-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libpmem-0__1.9-4.fc33.x86_64",
    sha256 = "40733112fe0c290619a7a7f6b76dccfb6919ac0271672f41b7fb96ce076dcf55",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libpmem-1.9-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libpmem-1.9-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libpmem-1.9-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libpmem-1.9-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libpng-2__1.6.37-6.fc33.aarch64",
    sha256 = "612c1ef9ef2d24d9c7112afa0191db59f8e82acd3b7de765342dff130b04cff3",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpng-1.6.37-6.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libpng-1.6.37-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpng-1.6.37-6.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpng-1.6.37-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libpng-2__1.6.37-6.fc33.x86_64",
    sha256 = "55961786d2e1d417ef8120cef5379d3563a744854ae24b67a839d52f2f5303aa",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpng-1.6.37-6.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpng-1.6.37-6.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libpng-1.6.37-6.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpng-1.6.37-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-2.fc33.aarch64",
    sha256 = "aa35b9922d3ebdb886a7f494f50a1079fd1fea78227a00e7bb31074b1e4e306f",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpwquality-1.4.4-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libpwquality-1.4.4-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpwquality-1.4.4-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libpwquality-1.4.4-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-2.fc33.x86_64",
    sha256 = "a74da9782ea20e65d910693073c6a7eea388800b5297680757564393db4479f3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpwquality-1.4.4-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpwquality-1.4.4-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libpwquality-1.4.4-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libpwquality-1.4.4-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "librdmacm-0__35.0-1.fc33.aarch64",
    sha256 = "713007531baa8797b8e9e78fec9658b4d79e4caae009fe17aafa1e2563199892",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/librdmacm-35.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/librdmacm-35.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/librdmacm-35.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/librdmacm-35.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "librdmacm-0__35.0-1.fc33.x86_64",
    sha256 = "7d4edcf369eacfb85c6d1a32acbe5d24dedc96c92c204771baf6ac28832df674",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/librdmacm-35.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/librdmacm-35.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/librdmacm-35.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/librdmacm-35.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "librepo-0__1.14.1-1.fc33.x86_64",
    sha256 = "52748d7562497a3181ece56afd76b7bedb289efebdea74783f4e942db3d90d23",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/librepo-1.14.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/librepo-1.14.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/librepo-1.14.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/librepo-1.14.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libreport-filesystem-0__2.15.2-2.fc33.x86_64",
    sha256 = "27ae38550b577f2f416bc19a4fa9801f6745fdebf14a2129e5aafafecefa50ec",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libreport-filesystem-2.15.2-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libreport-filesystem-2.15.2-2.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libreport-filesystem-2.15.2-2.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libreport-filesystem-2.15.2-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "libseccomp-0__2.5.0-3.fc33.aarch64",
    sha256 = "ab5a824d402c717bfe8e01cfb216a70fd4a7e1d66d2d7baa80ac6ad6581081c9",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libseccomp-2.5.0-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libseccomp-2.5.0-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libseccomp-2.5.0-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libseccomp-2.5.0-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libseccomp-0__2.5.0-3.fc33.x86_64",
    sha256 = "964e39835b59c76b7eb3f78c460bfc6e7acfb0c40b901775c7e8a7204537f8a7",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libseccomp-2.5.0-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libseccomp-2.5.0-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libseccomp-2.5.0-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libseccomp-2.5.0-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libselinux-0__3.1-2.fc33.aarch64",
    sha256 = "3e50b11882b29b9590a3cdb8dcb80098fd8606ef5824f01838c981c4c4007e3b",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libselinux-3.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libselinux-3.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libselinux-3.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libselinux-3.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libselinux-0__3.1-2.fc33.x86_64",
    sha256 = "898d9c9911a8e9b6933d3a7e52350f0dbb92e24ba9b00959cfaf451cec43661a",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libselinux-3.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libselinux-3.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libselinux-3.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libselinux-3.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libselinux-utils-0__3.1-2.fc33.aarch64",
    sha256 = "3fc62021ddf35477e84c45485d0da54aae743ab1318a2559c51be53c501ac200",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libselinux-utils-3.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libselinux-utils-3.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libselinux-utils-3.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libselinux-utils-3.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libselinux-utils-0__3.1-2.fc33.x86_64",
    sha256 = "59c4b3c0c1d150e80d64c4b63e477956116ffcdfffbc0fd47759a0d45a06bed5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libselinux-utils-3.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libselinux-utils-3.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libselinux-utils-3.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libselinux-utils-3.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libsemanage-0__3.1-2.fc33.aarch64",
    sha256 = "fa29fa0aaf613c902663d4d6f6fa4dbc4127a96fc9fd72f4c154df08e3e3febc",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libsemanage-3.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libsemanage-3.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libsemanage-3.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libsemanage-3.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libsemanage-0__3.1-2.fc33.x86_64",
    sha256 = "0c84b9965d221a5da3b62ba620b7cf69f0f77cdbc1b89a1c48e6df3cdfda258e",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsemanage-3.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsemanage-3.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsemanage-3.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libsemanage-3.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libsepol-0__3.1-3.fc33.aarch64",
    sha256 = "19bedd354211c58bd9ec935b3087c47ba1f34bb43bd06e0a66e751f6027ed841",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libsepol-3.1-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libsepol-3.1-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libsepol-3.1-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libsepol-3.1-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libsepol-0__3.1-3.fc33.x86_64",
    sha256 = "3da666241b0c46a3e6d172e028ce657d02bc6b9c7e2c12757ce629bdfee07a97",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsepol-3.1-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsepol-3.1-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsepol-3.1-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libsepol-3.1-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-11.fc33.aarch64",
    sha256 = "86e82ce52eaa68f6bab73fcbfd9d800f7a8bccb92081cdce0a37f91345a16ae9",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libsigsegv-2.11-11.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libsigsegv-2.11-11.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libsigsegv-2.11-11.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libsigsegv-2.11-11.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-11.fc33.x86_64",
    sha256 = "d0ea70f74990b543d2c294a764b77a75f1f2354f5e98e5c9638dffc1e1a71c1f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsigsegv-2.11-11.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsigsegv-2.11-11.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libsigsegv-2.11-11.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libsigsegv-2.11-11.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libsmartcols-0__2.36.1-1.fc33.aarch64",
    sha256 = "8c626d832b53805e4c3a375f90453a0115cb0de9823114ca85beee6bf2a8b076",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libsmartcols-2.36.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libsmartcols-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libsmartcols-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libsmartcols-2.36.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libsmartcols-0__2.36.1-1.fc33.x86_64",
    sha256 = "23101871fd241e376f029d3343a949b9e4b6b2595acf348dc8b1daf9a73eca07",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libsmartcols-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libsmartcols-2.36.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libsmartcols-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libsmartcols-2.36.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libsolv-0__0.7.17-1.fc33.x86_64",
    sha256 = "9053b9694522945cfe3a160cc8a0022f2418367b2d72cb1429d00cff91e1f8e5",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libsolv-0.7.17-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libsolv-0.7.17-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libsolv-0.7.17-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libsolv-0.7.17-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libss-0__1.45.6-4.fc33.aarch64",
    sha256 = "6a64673d0fe956f00ef5ede73463d4d913068b9b15d0c1c7424bce66bfbba883",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libss-1.45.6-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libss-1.45.6-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libss-1.45.6-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libss-1.45.6-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libss-0__1.45.6-4.fc33.x86_64",
    sha256 = "59604aca347019a53f2e09fb37eacb8bf882cfa731211b1161576ff2baf5e623",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libss-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libss-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libss-1.45.6-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libss-1.45.6-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libssh-0__0.9.5-1.fc33.aarch64",
    sha256 = "4b0690e8642e37f66dd8b1daabae44d272874d75cd89046ce4ab76cb7e36cdd4",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libssh-0.9.5-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libssh-0.9.5-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libssh-0.9.5-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libssh-0.9.5-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libssh-0__0.9.5-1.fc33.x86_64",
    sha256 = "fc74fb07362c326bb364d069789c1b8153263202160641337fbf22cd12c19ecf",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh-0.9.5-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh-0.9.5-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh-0.9.5-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libssh-0.9.5-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libssh-config-0__0.9.5-1.fc33.aarch64",
    sha256 = "7c1a3d7eca1254f8b39563a3dac133dfb14e6daa86ec2b1ad291958d9dfdbc38",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "libssh-config-0__0.9.5-1.fc33.x86_64",
    sha256 = "7c1a3d7eca1254f8b39563a3dac133dfb14e6daa86ec2b1ad291958d9dfdbc38",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libssh-config-0.9.5-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "libssh2-0__1.9.0-6.fc33.aarch64",
    sha256 = "2d1893a74099d09c6ca26647f655fcb6d49f21b9ddcdd4d6e45987ce608b95f1",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libssh2-1.9.0-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libssh2-1.9.0-6.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libssh2-1.9.0-6.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libssh2-1.9.0-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libssh2-0__1.9.0-6.fc33.x86_64",
    sha256 = "e6dadbdb507b557025a1086173ef70a4de6a49d854c9acd41f53da4d8c8c5584",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh2-1.9.0-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh2-1.9.0-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libssh2-1.9.0-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libssh2-1.9.0-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__10.3.1-1.fc33.aarch64",
    sha256 = "f48e12e90fcce4535ec24195fddba695da29818941d0ef5cc7be79d3296d2d59",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__10.3.1-1.fc33.x86_64",
    sha256 = "c85e6cc7739124bc999dfb872a6e046be69bf379f19c1d80a24a8d250d86ae25",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-3.fc33.aarch64",
    sha256 = "51783850b22d87c778b804d8c60b8eaa890e4bdd9ec81a3ba6bfa840feb5c8c7",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-3.fc33.x86_64",
    sha256 = "185725ccd1171d5883bb05cb42694ac573ff3b3c5168adbba806af7de466b1eb",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libtextstyle-0__0.21-3.fc33.aarch64",
    sha256 = "875539783799e863af7004007dab3ea449a6f8dfc99d6612ecbf82816ad2df7a",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libtextstyle-0.21-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libtextstyle-0.21-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libtextstyle-0.21-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libtextstyle-0.21-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libtextstyle-0__0.21-3.fc33.x86_64",
    sha256 = "173f904f0734916905f51ab634267edf8519f72977ada75bacd4dfcf94d96016",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libtextstyle-0.21-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libtextstyle-0.21-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libtextstyle-0.21-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libtextstyle-0.21-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libtirpc-0__1.2.6-4.rc4.fc33.aarch64",
    sha256 = "4504d1dd9ce2d1b4ad6557a6e21f36e8e09d592457c7a40873fc310b6211b26e",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libtirpc-0__1.2.6-4.rc4.fc33.x86_64",
    sha256 = "057da6c0d07111ea9569c7e24b457206d3a7ab804b4e88d81c02cff1ead3539e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libtirpc-1.2.6-4.rc4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libtpms-0__0.8.4-1.20210624gita594c4692a.fc33.aarch64",
    sha256 = "fc93046066a095d0589023c6c4033635f237cbbea600953ff3eee42b7ce69444",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libtpms-0__0.8.4-1.20210624gita594c4692a.fc33.x86_64",
    sha256 = "17a8c867e175622eb1c7a3baa472a0ee3e089ca2030fc38782039047d9d109af",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libtpms-0.8.4-1.20210624gita594c4692a.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-9.fc33.aarch64",
    sha256 = "bb6abb35d98c552f313ebe609982c35ee4ec07d8ab8c6d9d2d864e29d35ea616",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libunistring-0.9.10-9.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libunistring-0.9.10-9.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libunistring-0.9.10-9.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libunistring-0.9.10-9.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-9.fc33.x86_64",
    sha256 = "a5ed095938b6ef997cbad403a5f46f64b71db54c8e35c7d0b93d0ca6e5fa88a7",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libunistring-0.9.10-9.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libunistring-0.9.10-9.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libunistring-0.9.10-9.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libunistring-0.9.10-9.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libunwind-0__1.4.0-4.fc33.aarch64",
    sha256 = "fa1e6a6529c0de1dc7a1245546d630fc97639fe87533975a92e04e1ad5c5b7bd",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libunwind-1.4.0-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libunwind-1.4.0-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libunwind-1.4.0-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libunwind-1.4.0-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libunwind-0__1.4.0-4.fc33.x86_64",
    sha256 = "01957e4ebfb63766b22fb9d865d8c8e13b945a4a49cc14af7261e9d1bc6279f2",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libunwind-1.4.0-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libunwind-1.4.0-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libunwind-1.4.0-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libunwind-1.4.0-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libusal-0__1.1.11-46.fc33.x86_64",
    sha256 = "cbd4f98d2e61d317ca511e7ceedac43b87558a148ce916cf8e8cc64637372623",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libusal-1.1.11-46.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libusal-1.1.11-46.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libusal-1.1.11-46.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libusal-1.1.11-46.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libusbx-0__1.0.24-2.fc33.aarch64",
    sha256 = "76a5c0d3823d84f55152f984f6fa47aa79503da538d1e33b02a946568cfa7344",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libusbx-0__1.0.24-2.fc33.x86_64",
    sha256 = "888f1bf001f0bc4f336ae9e9f0ee7da4f44df553fe77b7f6e0397b029a352836",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-2.fc33.aarch64",
    sha256 = "e56021ee9e6fd5200d3af531876a02dbfd4704441e732603a636c44adb4461f4",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libutempter-1.2.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libutempter-1.2.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libutempter-1.2.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libutempter-1.2.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-2.fc33.x86_64",
    sha256 = "3d0ed8ce643128450960b07873746465ae1ce288d14a235641dc1ab145cef688",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libutempter-1.2.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libutempter-1.2.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libutempter-1.2.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libutempter-1.2.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libuuid-0__2.36.1-1.fc33.aarch64",
    sha256 = "0d140fd97a17dbb7f331712876ec48bfe897efa0369eca3c08c1fe1777f0bb48",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libuuid-2.36.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libuuid-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libuuid-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libuuid-2.36.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libuuid-0__2.36.1-1.fc33.x86_64",
    sha256 = "a49bba77d31af50eececfce69853cc694abed86ee210208dae74838815704977",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libuuid-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libuuid-2.36.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libuuid-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libuuid-2.36.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libverto-0__0.3.0-10.fc33.aarch64",
    sha256 = "506efb4322165f1d310b9881dadb22f59564635e3a2b26c7e08602c0115dd314",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libverto-0.3.0-10.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libverto-0.3.0-10.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libverto-0.3.0-10.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libverto-0.3.0-10.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libverto-0__0.3.0-10.fc33.x86_64",
    sha256 = "37bb459e5079332144ee5bd4858657df635a6dc5ed67d25807072b8df7cc1ac5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libverto-0.3.0-10.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libverto-0.3.0-10.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libverto-0.3.0-10.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libverto-0.3.0-10.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-bash-completion-0__7.0.0-12.fc33.aarch64",
    sha256 = "b3d659e6e7b1bb3f9342f9357aa12475b6331bc65a39b662cfbb75ea8e821ada",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-aarch64/02116091-libvirt/libvirt-bash-completion-7.0.0-12.fc33.aarch64.rpm"],
)

rpm(
    name = "libvirt-bash-completion-0__7.0.0-12.fc33.x86_64",
    sha256 = "d947d1283f2b3f6fa5b9c7be1fac9e7866fa970bea5794ae77dafbb948c5d78b",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-bash-completion-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libvirt-client-0__7.0.0-12.fc33.aarch64",
    sha256 = "f248397a63daf71846b81567b3900d62168be3810de1e3e19e456e943847c7ee",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-aarch64/02116091-libvirt/libvirt-client-7.0.0-12.fc33.aarch64.rpm"],
)

rpm(
    name = "libvirt-client-0__7.0.0-12.fc33.x86_64",
    sha256 = "fa51355b9c34d6ca30101cd5a23b1a52bee818c78b7743913cb0d347edffd319",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-client-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-0__7.0.0-12.fc33.aarch64",
    sha256 = "774507311156940fb837c3211a9335ee6bbb266064617b45bf5226ad6b5063dc",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-aarch64/02116091-libvirt/libvirt-daemon-7.0.0-12.fc33.aarch64.rpm"],
)

rpm(
    name = "libvirt-daemon-0__7.0.0-12.fc33.x86_64",
    sha256 = "bccfe9a290f4df1659dc42a96d14d8c82ace32ff80dbe4041278b9a34a45930b",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-daemon-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.0.0-12.fc33.aarch64",
    sha256 = "cd3e821f685668c1ef624a2f0c58b07d35b810fae17532c2f8c69da56cd9c93d",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-aarch64/02116091-libvirt/libvirt-daemon-driver-qemu-7.0.0-12.fc33.aarch64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.0.0-12.fc33.x86_64",
    sha256 = "382e0776a40cb0e82178c8697325da8b024304949805e57c692fccba8aa1d95b",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-daemon-driver-qemu-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__7.0.0-12.fc33.x86_64",
    sha256 = "0eb25e05d2e9183f29d3b2a72a8f2acbb12ff4d88df0ac2f8da7a5a252445a13",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-daemon-driver-secret-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libvirt-devel-0__7.0.0-12.fc33.aarch64",
    sha256 = "c381c657851378665b2d8267f21955b829b957bb6228aed92d9828a174185852",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-aarch64/02116091-libvirt/libvirt-devel-7.0.0-12.fc33.aarch64.rpm"],
)

rpm(
    name = "libvirt-devel-0__7.0.0-12.fc33.x86_64",
    sha256 = "62c0d995655471efb7be1ecbfa28420c64ad89b5be50eb48a8d4a763b7f4b002",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-devel-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libvirt-libs-0__7.0.0-12.fc33.aarch64",
    sha256 = "374f58bf038821c9b0005966c0ec540c6e5152f6045686c93ce7e71ccb27b9f6",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-aarch64/02116091-libvirt/libvirt-libs-7.0.0-12.fc33.aarch64.rpm"],
)

rpm(
    name = "libvirt-libs-0__7.0.0-12.fc33.x86_64",
    sha256 = "65d4b2807a0438d4e71d762e0dc86b463a5ddcf398f9f97b83ca58c64f3b0a92",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-33-x86_64/02116091-libvirt/libvirt-libs-7.0.0-12.fc33.x86_64.rpm"],
)

rpm(
    name = "libwsman1-0__2.6.8-17.fc33.aarch64",
    sha256 = "57d6d44bc95b22d00d1195a65b52c9f67073d5bbc36c72fa6d6eef3641870a53",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libwsman1-2.6.8-17.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libwsman1-2.6.8-17.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libwsman1-2.6.8-17.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libwsman1-2.6.8-17.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libwsman1-0__2.6.8-17.fc33.x86_64",
    sha256 = "6b38e2f34200cc6eb0f57c9757f367a99cd64bb92dba34698c3c0ee5a473d9c9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libwsman1-2.6.8-17.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libwsman1-2.6.8-17.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libwsman1-2.6.8-17.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libwsman1-2.6.8-17.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.23-1.fc33.aarch64",
    sha256 = "cd0cf291ee668ca5109cc218b2e7fbac7c2f537532c68e1410fe176f4f343ac8",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libxcrypt-4.4.23-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libxcrypt-4.4.23-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libxcrypt-4.4.23-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libxcrypt-4.4.23-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.23-1.fc33.x86_64",
    sha256 = "d2077aa02f520136e50f023c1a2851caaf1830f4941b29bbbb2f7e2d4dd2af5a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libxcrypt-4.4.23-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libxcrypt-4.4.23-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libxcrypt-4.4.23-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libxcrypt-4.4.23-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libxkbcommon-0__1.0.1-1.fc33.aarch64",
    sha256 = "3d83f736fff09470371b11f1987d8d59c7a91849f8f73cfdbebf3fc85fec5700",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libxkbcommon-0__1.0.1-1.fc33.x86_64",
    sha256 = "4507a3f68f13d9a7efebc8f812becf6e493106dab092e03fd56bcf6dca9b39c4",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libxkbcommon-1.0.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libxml2-0__2.9.12-4.fc33.aarch64",
    sha256 = "a1249fd4fb8b2b6735254be0c04106b8fb197eb2db76743e81c0bd37449cd6be",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libxml2-2.9.12-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libxml2-2.9.12-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libxml2-2.9.12-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libxml2-2.9.12-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libxml2-0__2.9.12-4.fc33.x86_64",
    sha256 = "e71ffc4d92a3b1baae855c8fcbb93e5b2e3510449d6f89a2bd208e7b89b0020a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libxml2-2.9.12-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libxml2-2.9.12-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libxml2-2.9.12-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libxml2-2.9.12-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libyaml-0__0.2.5-3.fc33.x86_64",
    sha256 = "fa1cb9fae24eac5a489ba1995574d9e249dfd2a4d3a27ef06a980ca00d9bbf4c",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libyaml-0.2.5-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libyaml-0.2.5-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/libyaml-0.2.5-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/libyaml-0.2.5-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "libzstd-0__1.5.0-1.fc33.aarch64",
    sha256 = "671cd12c421bfd080f32019948cd8e3d0f38755dd57c294f14eb6a196c07f0c3",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/libzstd-1.5.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/libzstd-1.5.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/libzstd-1.5.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/libzstd-1.5.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "libzstd-0__1.5.0-1.fc33.x86_64",
    sha256 = "2c0d974ddc788990b38d5d61bd7d5f25ada666bf997e7dc46d9a21a465397a35",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libzstd-1.5.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/libzstd-1.5.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/libzstd-1.5.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/libzstd-1.5.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "linux-atm-libs-0__2.5.1-27.fc33.aarch64",
    sha256 = "16f02899aed36b6738707b0a4dafaa7ff6dcb54a37dcfa745b6869dfbab240c8",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "linux-atm-libs-0__2.5.1-27.fc33.x86_64",
    sha256 = "dcaa79dabf9ad8a7b5cc4cd3913b3667bf207450921f6f80a8208ab120c077d3",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-27.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "linux-firmware-0__20210716-121.fc33.x86_64",
    sha256 = "68e3c9ad2824889a0418e8d5334438d412a7ad774080257b97a7432dcfce3a42",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-20210716-121.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-20210716-121.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-20210716-121.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-20210716-121.fc33.noarch.rpm",
    ],
)

rpm(
    name = "linux-firmware-whence-0__20210716-121.fc33.x86_64",
    sha256 = "646f8c3088d1e8b223b516a3c9650a3dd34304e37855d7ad46bb415a009f2f94",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-whence-20210716-121.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-whence-20210716-121.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-whence-20210716-121.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/linux-firmware-whence-20210716-121.fc33.noarch.rpm",
    ],
)

rpm(
    name = "lsof-0__4.93.2-4.fc33.aarch64",
    sha256 = "feba3a83b50c43b899b4c2e1e448b852e9c90b2a257f2eafe1be7755ba661949",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/lsof-4.93.2-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lsof-4.93.2-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/lsof-4.93.2-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lsof-4.93.2-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "lsof-0__4.93.2-4.fc33.x86_64",
    sha256 = "170188bd8a452f70c9b489b248ba74895df4012ccd4c4155f0fa59dcfefd2e60",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lsof-4.93.2-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lsof-4.93.2-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lsof-4.93.2-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/lsof-4.93.2-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lsscsi-0__0.32-1.fc33.x86_64",
    sha256 = "c0908eab32f1cd7f0dbec4989bf48aa6400bcf8c8b4b64406bddf689f35f7143",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/lsscsi-0.32-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/lsscsi-0.32-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/lsscsi-0.32-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/lsscsi-0.32-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lua-libs-0__5.4.3-1.fc33.aarch64",
    sha256 = "8fe55e8e17c35e5f3f8d4c6b796bd05d938150ed3cc0e5a3a53ffbe170cd11f7",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/l/lua-libs-5.4.3-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/l/lua-libs-5.4.3-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/l/lua-libs-5.4.3-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/l/lua-libs-5.4.3-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "lua-libs-0__5.4.3-1.fc33.x86_64",
    sha256 = "70c3d9491cff17279cbc29b87a2a1cf062c4cc8c2e487626f8de411d19cf1234",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/lua-libs-5.4.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/l/lua-libs-5.4.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/l/lua-libs-5.4.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/l/lua-libs-5.4.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lvm2-0__2.03.10-1.fc33.x86_64",
    sha256 = "1d0378ffc0575f8627445aa666533e4558235d830adb61927069e4682eca3104",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lvm2-2.03.10-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lvm2-2.03.10-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lvm2-2.03.10-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/lvm2-2.03.10-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lvm2-libs-0__2.03.10-1.fc33.x86_64",
    sha256 = "dbc237320a73c44c38124da66469d199a49c3361d416f9e7354b9e106043938c",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.10-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.10-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.10-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.10-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.1-3.fc33.aarch64",
    sha256 = "50412043630f1e2d798202907dda5fba28b4059627e28b66c12d2afc98bc6930",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.1-3.fc33.x86_64",
    sha256 = "099c665f716f7039f8f81e0b00d359b3808ecd4c5cd933f51b129c81c19544e5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lzo-0__2.10-3.fc33.aarch64",
    sha256 = "33438d15d07b0acdde3f8606b4373532af57f25a2c05c9569ace8752a4cb33fb",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/lzo-2.10-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lzo-2.10-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/lzo-2.10-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lzo-2.10-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "lzo-0__2.10-3.fc33.x86_64",
    sha256 = "52c386eefee700baa2befdca5c065bf8d61688d7703e00c80ca8ceee30cbe503",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lzo-2.10-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lzo-2.10-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lzo-2.10-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/lzo-2.10-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "lzop-0__1.04-5.fc33.aarch64",
    sha256 = "6124cfb85319d3a5a4a43ef8dd3cbf05cf87f615de23b4a96d291923f0c1cf53",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/l/lzop-1.04-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lzop-1.04-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/l/lzop-1.04-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/l/lzop-1.04-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "lzop-0__1.04-5.fc33.x86_64",
    sha256 = "ac1aafc48ea9025cdc416022bae4d5a83ae008553f527feb6b1c2f0ae9b8ecf4",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lzop-1.04-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lzop-1.04-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/l/lzop-1.04-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/l/lzop-1.04-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "man-db-0__2.9.2-6.fc33.x86_64",
    sha256 = "392c6aa83abdd0fb90de42d8650fe6c9fd4028b4bfddcc8189bb82480fe2a140",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/m/man-db-2.9.2-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/m/man-db-2.9.2-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/m/man-db-2.9.2-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/m/man-db-2.9.2-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "mdadm-0__4.1-6.fc33.x86_64",
    sha256 = "8cbb151246a861a9637ef339231a6c64400b3d5dd62cdf8eb78fc6e509ad994d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/m/mdadm-4.1-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/m/mdadm-4.1-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/m/mdadm-4.1-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/m/mdadm-4.1-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "mozjs78-0__78.12.0-1.fc33.aarch64",
    sha256 = "91e38a9ec52f6e62cd0dc3025ce18e7d2248b7ca2348ad7853a3a475c09936b7",
    urls = [
        "https://mirrors.xtom.ee/fedora/updates/33/Everything/aarch64/Packages/m/mozjs78-78.12.0-1.fc33.aarch64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/updates/33/Everything/aarch64/Packages/m/mozjs78-78.12.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/m/mozjs78-78.12.0-1.fc33.aarch64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/aarch64/Packages/m/mozjs78-78.12.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "mozjs78-0__78.12.0-1.fc33.x86_64",
    sha256 = "298b697339f701601049cd4cbb2ec3ec1ddf585dccc1a801b39b94327cf7f681",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/m/mozjs78-78.12.0-1.fc33.x86_64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/m/mozjs78-78.12.0-1.fc33.x86_64.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/m/mozjs78-78.12.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/m/mozjs78-78.12.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-7.fc33.aarch64",
    sha256 = "f4ea5de704058d7bdedd2bb12cdd936bd971b7243498cc829d0df0c5dbc93c86",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/m/mpfr-4.1.0-7.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/m/mpfr-4.1.0-7.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/m/mpfr-4.1.0-7.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/m/mpfr-4.1.0-7.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-7.fc33.x86_64",
    sha256 = "141954a00c6fd0d2184483d8079f56dc578302aabf9344a105527fa1ba69a2f6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/m/mpfr-4.1.0-7.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/m/mpfr-4.1.0-7.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/m/mpfr-4.1.0-7.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/m/mpfr-4.1.0-7.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "mtools-0__4.0.31-1.fc33.x86_64",
    sha256 = "26c6168ca8b7f2fc0e288fd4ca58c83bafd523514e2518689e282ecf7e3681db",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/m/mtools-4.0.31-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/m/mtools-4.0.31-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/m/mtools-4.0.31-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/m/mtools-4.0.31-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ncurses-0__6.2-3.20200222.fc33.aarch64",
    sha256 = "45c3c9f0af99e35d7e945aa5e54e1488c3f70b458608ab4b6882b0c3dd5e7bb2",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/ncurses-6.2-3.20200222.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/ncurses-6.2-3.20200222.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/ncurses-6.2-3.20200222.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/ncurses-6.2-3.20200222.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "ncurses-0__6.2-3.20200222.fc33.x86_64",
    sha256 = "f20e6a7d425bac2891a7f6628bcfcc8553efc2e3a841b0395cd3729c16138aa1",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-6.2-3.20200222.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-6.2-3.20200222.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-6.2-3.20200222.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/ncurses-6.2-3.20200222.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-3.20200222.fc33.aarch64",
    sha256 = "3ba2028d4649a5f9e6c77785e09dc5d711f5856c5c91c923ff3f46ea4430f4df",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-3.20200222.fc33.x86_64",
    sha256 = "3ba2028d4649a5f9e6c77785e09dc5d711f5856c5c91c923ff3f46ea4430f4df",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/ncurses-base-6.2-3.20200222.fc33.noarch.rpm",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-3.20200222.fc33.aarch64",
    sha256 = "0ee8d448ba3b455d707bef95d8eb8670f2015fc9f2bb729fdc843e8336f3575d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-3.20200222.fc33.x86_64",
    sha256 = "6aa5ec2a16eb602969378982f1d7983acb2fad63198042235224a9e3ebe27e06",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/ncurses-libs-6.2-3.20200222.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ndctl-libs-0__71.1-1.fc33.x86_64",
    sha256 = "e59c9104a0ed99b109f56d97b7a85b9024aff4317f2cdfb3f6cc984e758dcd78",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/ndctl-libs-71.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/n/ndctl-libs-71.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/n/ndctl-libs-71.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/ndctl-libs-71.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "nettle-0__3.6-3.fc33.aarch64",
    sha256 = "5e38e6fa4413958e7f555d7d7a37cda0412d58556b75fdb78dd1f227110b0653",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/nettle-3.6-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/nettle-3.6-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/nettle-3.6-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/nettle-3.6-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "nettle-0__3.6-3.fc33.x86_64",
    sha256 = "5d8870dad6187c05f1a4599bf9fe16f8e3d3254c87a93a7f21bcf50579e10a07",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nettle-3.6-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nettle-3.6-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nettle-3.6-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/nettle-3.6-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "nftables-1__0.9.3-8.fc33.aarch64",
    sha256 = "b2075fa165940e5b72ee9a30ee3e305aef25434dd7125bfe1cb7a85617b88e29",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/n/nftables-0.9.3-8.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/n/nftables-0.9.3-8.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/n/nftables-0.9.3-8.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/n/nftables-0.9.3-8.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "nftables-1__0.9.3-8.fc33.x86_64",
    sha256 = "2bfa685773c94eb28d1c36f2ada08d1e50696931c1c16036a5e8ab3dee2a3146",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/nftables-0.9.3-8.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/n/nftables-0.9.3-8.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/n/nftables-0.9.3-8.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/nftables-0.9.3-8.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "nginx-1__1.20.1-3.fc33.aarch64",
    sha256 = "96c815b4e6a57aefb9967926145f63802b21b146a23dd362b920beef79aac132",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/n/nginx-1.20.1-3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/n/nginx-1.20.1-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/n/nginx-1.20.1-3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/n/nginx-1.20.1-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "nginx-1__1.20.1-3.fc33.x86_64",
    sha256 = "e3a3d964236f10c440920c08761ca2f5fb4a044d5af1deb8a473274b7654e5de",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/nginx-1.20.1-3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/n/nginx-1.20.1-3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/n/nginx-1.20.1-3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/nginx-1.20.1-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "nginx-filesystem-1__1.20.1-3.fc33.aarch64",
    sha256 = "749da91395935056069434b3cb82719c9b0e98512ceca5cd84ac772884337835",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
    ],
)

rpm(
    name = "nginx-filesystem-1__1.20.1-3.fc33.x86_64",
    sha256 = "749da91395935056069434b3cb82719c9b0e98512ceca5cd84ac772884337835",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/n/nginx-filesystem-1.20.1-3.fc33.noarch.rpm",
    ],
)

rpm(
    name = "nginx-mimetypes-0__2.1.49-2.fc33.aarch64",
    sha256 = "e860501275c9073f199354766d9ccd99afc0b97fff8acae8e8184d4f02799d38",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "nginx-mimetypes-0__2.1.49-2.fc33.x86_64",
    sha256 = "e860501275c9073f199354766d9ccd99afc0b97fff8acae8e8184d4f02799d38",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.49-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "nmap-ncat-2__7.80-5.fc33.aarch64",
    sha256 = "95dd4cae81ea529ecf09ba4e6252625227da0dbfecd18f94d6ec9b2c0a2ec89c",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/nmap-ncat-7.80-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/nmap-ncat-7.80-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/nmap-ncat-7.80-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/nmap-ncat-7.80-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "nmap-ncat-2__7.80-5.fc33.x86_64",
    sha256 = "e95e3bc3abd0adadc8588440c68a0d7fea32f13ce32dab441bf47cfaca2798e4",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nmap-ncat-7.80-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nmap-ncat-7.80-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/nmap-ncat-7.80-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/nmap-ncat-7.80-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "npth-0__1.6-5.fc33.x86_64",
    sha256 = "31b24e5b45ac87710d84a334b57978b858d1ea723645d40a1b020d26f7ab87aa",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/npth-1.6-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/npth-1.6-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/npth-1.6-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/npth-1.6-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ntfs-3g-2__2017.3.23-14.fc33.x86_64",
    sha256 = "eaec107252826d1a5176caaee8663bebe1a049ebcc85ec4d190df51f4623d68d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-14.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-14.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-14.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-14.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ntfs-3g-system-compression-0__1.0-4.fc33.x86_64",
    sha256 = "079bae0466224156aed1b479aef713d69df81a47cb5312fba150e8378b2ecd36",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "ntfsprogs-2__2017.3.23-14.fc33.x86_64",
    sha256 = "e50ddc5d617a568c247a4c9023a0d7a02ac98a6d48a36b9791ccbb63b7b2a226",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-14.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-14.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-14.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-14.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.14-1.fc33.aarch64",
    sha256 = "3a76106cad35a14805ba46cab5299a35bf3395f4f3af2426d08793b1cea55f22",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/numactl-libs-2.0.14-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/numactl-libs-2.0.14-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/numactl-libs-2.0.14-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/numactl-libs-2.0.14-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.14-1.fc33.x86_64",
    sha256 = "4638c05022355530097a6500e94fd04328383b50b03cf19a4538df317c129238",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/numactl-libs-2.0.14-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/numactl-libs-2.0.14-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/numactl-libs-2.0.14-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/numactl-libs-2.0.14-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "numad-0__0.5-32.20150602git.fc33.aarch64",
    sha256 = "ccc5a8f4cfa786fec209144c139d24d3004fa639eb52982f1bc38d4f70ece176",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/n/numad-0.5-32.20150602git.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/numad-0.5-32.20150602git.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/n/numad-0.5-32.20150602git.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/n/numad-0.5-32.20150602git.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "numad-0__0.5-32.20150602git.fc33.x86_64",
    sha256 = "add1e73d27b1541781323820be04b1ed56507903cfd16f76ea8a0b3b39463467",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/numad-0.5-32.20150602git.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/numad-0.5-32.20150602git.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/n/numad-0.5-32.20150602git.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/n/numad-0.5-32.20150602git.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "openldap-0__2.4.50-5.fc33.aarch64",
    sha256 = "9f0856617538fb641df75099ba4cb8e5dba0667ecb0ae99339c28539f59b1b3c",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/o/openldap-2.4.50-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/o/openldap-2.4.50-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/o/openldap-2.4.50-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/o/openldap-2.4.50-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "openldap-0__2.4.50-5.fc33.x86_64",
    sha256 = "8edd2c807b5277829c1b9434c8eef06190f492b78c706e8f4a212b4169646e01",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/o/openldap-2.4.50-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/o/openldap-2.4.50-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/o/openldap-2.4.50-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/o/openldap-2.4.50-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "openssl-1__1.1.1k-1.fc33.aarch64",
    sha256 = "93e0d59d9f07328a62665355044dc383cc83bf40fe37d2a157995aead9fb43ed",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "openssl-1__1.1.1k-1.fc33.x86_64",
    sha256 = "c3f796604d12b64c21d08384e41c7aac9e86dc028ca55d1bbe61b9705958ece2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-1.fc33.aarch64",
    sha256 = "93e378602e7bd9e903a2bfc04f43469dd1f6c6e5d786782445d2f2d35ec9ea76",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-1.fc33.x86_64",
    sha256 = "3ceed0f6a58eb99f85ae10a0888af440b167ed88b1d64358cf994bb68acdefb6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-2.fc33.aarch64",
    sha256 = "338154cbc58da6c2972c2670bb5d1aeddfdc4e7f65ab20403c32f52948e7b077",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-2.fc33.x86_64",
    sha256 = "44891526d7ba01d8e02d7e91214190e0799e0896c00c9d3b58209415795c9b1f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-2.fc33.aarch64",
    sha256 = "c32c04513360b1548b0e459ccbb886f2daf129be85f682ccd795673d09d8f12f",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-2.fc33.x86_64",
    sha256 = "6c969fdbd5edd104db2379fb7e93966cbe10c5b69032000883978e2d956e5e8c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pam-0__1.4.0-11.fc33.aarch64",
    sha256 = "7b20e49bb0355aa28e217b67e14980f957af2066e71339e4b776fa98e571951d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pam-1.4.0-11.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pam-1.4.0-11.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pam-1.4.0-11.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pam-1.4.0-11.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pam-0__1.4.0-11.fc33.x86_64",
    sha256 = "2ce1aa050e5352d91031273ffb0b85b24e7b7d38f31db62da75ca482ed160bc6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pam-1.4.0-11.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pam-1.4.0-11.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pam-1.4.0-11.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pam-1.4.0-11.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "parted-0__3.3-5.fc33.x86_64",
    sha256 = "bc293f7c965c95f4c48dcf76b157fb1faa323cb04e273ee35e99b4a4b5887979",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/parted-3.3-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/parted-3.3-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/parted-3.3-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/parted-3.3-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pciutils-0__3.7.0-3.fc33.aarch64",
    sha256 = "9ed46338c12582b988f4d4905c4a5efc4f1d37831db9de4ebdada2202e7ad85f",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pciutils-0__3.7.0-3.fc33.x86_64",
    sha256 = "8b5d2ba55dda94724ce0b81ea340786003480263341eac33f431096c1a8a5379",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pciutils-libs-0__3.7.0-3.fc33.aarch64",
    sha256 = "61bbd79d595f5d2c60309a802a4005e8892b14014175e389579731bf529e1bc1",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pciutils-libs-0__3.7.0-3.fc33.x86_64",
    sha256 = "1f88efb8e360d474bb928d42e22a64c90f086c2abbafa4080f436da1bb711b8e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pcre-0__8.44-2.fc33.aarch64",
    sha256 = "5e488b96605e3799b666cf121e1e6690799fb60d4b036969018426aa35a1a7c1",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre-8.44-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pcre-8.44-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre-8.44-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre-8.44-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pcre-0__8.44-2.fc33.x86_64",
    sha256 = "56aab3400a2e087a0d4b73a5f05f1bdf8bdb2a9d38182e3ada653f1f442c8313",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre-8.44-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre-8.44-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pcre-8.44-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre-8.44-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pcre2-0__10.36-4.fc33.aarch64",
    sha256 = "647e786334e32eb09833d50d86d8103f5502f523a1f9adf8bb96ada1cab0c74a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre2-10.36-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pcre2-10.36-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre2-10.36-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre2-10.36-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pcre2-0__10.36-4.fc33.x86_64",
    sha256 = "40580c78b25180c9f366ad73aa0d109723693513e2a239aec6c44c8d0c33c4b6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre2-10.36-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre2-10.36-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pcre2-10.36-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre2-10.36-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.36-4.fc33.aarch64",
    sha256 = "d03029e5b2852c5b3312dc28b08c8904e999bccdfebdde4d5efbbdeeb78b8b1d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.36-4.fc33.x86_64",
    sha256 = "d03029e5b2852c5b3312dc28b08c8904e999bccdfebdde4d5efbbdeeb78b8b1d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Carp-0__1.50-457.fc33.aarch64",
    sha256 = "e5e022524532e7058cb71ae47ed5ae909b03b9feb6769174f35321e5afbc0ab8",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Carp-0__1.50-457.fc33.x86_64",
    sha256 = "e5e022524532e7058cb71ae47ed5ae909b03b9feb6769174f35321e5afbc0ab8",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Carp-1.50-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Class-Struct-0__0.66-471.fc33.aarch64",
    sha256 = "0a5c87ffe750274f7009c5d5cc5b0fe79d840f70e3b337ea0e26841a0e12b8a3",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Class-Struct-0__0.66-471.fc33.x86_64",
    sha256 = "0a5c87ffe750274f7009c5d5cc5b0fe79d840f70e3b337ea0e26841a0e12b8a3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Class-Struct-0.66-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Config-General-0__2.63-13.fc33.aarch64",
    sha256 = "d2b36626569d0eb2e2524d4d63fef106b8df3a109a3c72586827b8cf35bd0360",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Config-General-0__2.63-13.fc33.x86_64",
    sha256 = "d2b36626569d0eb2e2524d4d63fef106b8df3a109a3c72586827b8cf35bd0360",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-13.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-DynaLoader-0__1.47-471.fc33.aarch64",
    sha256 = "db7460a3fd4a66eb1b8c72f29306a95d9d0415f0eb40ca9a99dbb0390a66814d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-DynaLoader-1.47-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-DynaLoader-1.47-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-DynaLoader-1.47-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-DynaLoader-1.47-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-DynaLoader-0__1.47-471.fc33.x86_64",
    sha256 = "80af5e171fc72fb147c456c39601f32bb3d2cecfc0e91730f69241d92d5bb1a9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-DynaLoader-1.47-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-DynaLoader-1.47-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-DynaLoader-1.47-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-DynaLoader-1.47-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-Encode-4__3.08-458.fc33.aarch64",
    sha256 = "6213e51e78a9f74218ea0b8ca450c075d4b3c6e4c743f46af5d204a60064f717",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-Encode-4__3.08-458.fc33.x86_64",
    sha256 = "63e39c8f6144b94549fe60589c23efae7c40c167d974a062b4ed1c02dd7e79b6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-English-0__1.11-471.fc33.aarch64",
    sha256 = "e656675774b04078fcf3d5e7d4df49bf19769fa7627791f81ad5cd50075c6ecc",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-English-0__1.11-471.fc33.x86_64",
    sha256 = "e656675774b04078fcf3d5e7d4df49bf19769fa7627791f81ad5cd50075c6ecc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-English-1.11-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Errno-0__1.30-471.fc33.aarch64",
    sha256 = "c075cce7279237319678c911b7e238ea673f82b05c6bf47ef8df1b0c87c3c00a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Errno-1.30-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Errno-1.30-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Errno-1.30-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Errno-1.30-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-Errno-0__1.30-471.fc33.x86_64",
    sha256 = "628b1f4d04a3cc5028d92bfd3dd5692cb4075fb3e7f8ffb8eef0ca2cd0905b06",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Errno-1.30-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Errno-1.30-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Errno-1.30-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Errno-1.30-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-Exporter-0__5.74-458.fc33.aarch64",
    sha256 = "0e0b6a361d6326a1d1da76002148529c255623e597d4ab4d67897f438aa23d95",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Exporter-0__5.74-458.fc33.x86_64",
    sha256 = "0e0b6a361d6326a1d1da76002148529c255623e597d4ab4d67897f438aa23d95",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Fcntl-0__1.13-471.fc33.aarch64",
    sha256 = "699fdfa85a5d96ef618659039afce125c01f82012c41f6e4e5d1d0a728d89879",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Fcntl-1.13-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Fcntl-1.13-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Fcntl-1.13-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Fcntl-1.13-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-Fcntl-0__1.13-471.fc33.x86_64",
    sha256 = "a717261d6ea1c8ffa3ecf5d1aed10c487fd9449301923d0dbaad7d0da6f2ba83",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Fcntl-1.13-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Fcntl-1.13-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Fcntl-1.13-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Fcntl-1.13-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-File-Basename-0__2.85-471.fc33.aarch64",
    sha256 = "e23464437ee0b126be89f28b9557925ba937ad72cbcbe4a64f5473fb3fd027df",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-Basename-0__2.85-471.fc33.x86_64",
    sha256 = "e23464437ee0b126be89f28b9557925ba937ad72cbcbe4a64f5473fb3fd027df",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Basename-2.85-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-Path-0__2.18-1.fc33.aarch64",
    sha256 = "fa69403d6baf75a911b220a4f12a189a62130dd2983edf92597815b863b38444",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-Path-0__2.18-1.fc33.x86_64",
    sha256 = "fa69403d6baf75a911b220a4f12a189a62130dd2983edf92597815b863b38444",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Path-2.18-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-Temp-1__0.231.100-1.fc33.aarch64",
    sha256 = "2ed121bb2f22c7f46cec5048145d2adbc4babcfbb10cb1f50f87c7fac1e9e486",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-Temp-1__0.231.100-1.fc33.x86_64",
    sha256 = "2ed121bb2f22c7f46cec5048145d2adbc4babcfbb10cb1f50f87c7fac1e9e486",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-Temp-0.231.100-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-stat-0__1.09-471.fc33.aarch64",
    sha256 = "fad654208a476b3281a474bdb177e62d9a4b411bf0f664fddd9b7a7cb8ff564c",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-File-stat-0__1.09-471.fc33.x86_64",
    sha256 = "fad654208a476b3281a474bdb177e62d9a4b411bf0f664fddd9b7a7cb8ff564c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-File-stat-1.09-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-FileHandle-0__2.03-471.fc33.aarch64",
    sha256 = "056dcbbc626b9e855772912ce71ad103c64bfeee34a168038ab2eab3b941e519",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-FileHandle-0__2.03-471.fc33.x86_64",
    sha256 = "056dcbbc626b9e855772912ce71ad103c64bfeee34a168038ab2eab3b941e519",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-FileHandle-2.03-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.52-1.fc33.aarch64",
    sha256 = "7ce8898c6b1865d85625c74947b3f2257303b2a3316d64f1f68f89a76ac57286",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.52-1.fc33.x86_64",
    sha256 = "7ce8898c6b1865d85625c74947b3f2257303b2a3316d64f1f68f89a76ac57286",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Getopt-Long-2.52-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Getopt-Std-0__1.12-471.fc33.aarch64",
    sha256 = "5a26527412fc011b733060c592a7f03ebbed1461a68833c68c63ac70ef7080f6",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Getopt-Std-0__1.12-471.fc33.x86_64",
    sha256 = "5a26527412fc011b733060c592a7f03ebbed1461a68833c68c63ac70ef7080f6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Getopt-Std-1.12-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.076-457.fc33.aarch64",
    sha256 = "39ce930bc92f6f6ef48dbae17daf1be2bbc357a524732b12db8e85e9a8c481c5",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.076-457.fc33.x86_64",
    sha256 = "39ce930bc92f6f6ef48dbae17daf1be2bbc357a524732b12db8e85e9a8c481c5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-IO-0__1.43-471.fc33.aarch64",
    sha256 = "cceb1fd488475aed986ae427a5ab48e8ddfc1df3359ffba4fd1328f46a7d2e92",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-IO-1.43-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-IO-1.43-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-IO-1.43-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-IO-1.43-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-IO-0__1.43-471.fc33.x86_64",
    sha256 = "ed2d30b2a5ab585a8b7e6bc17684c5636ca74c84b54d5ec24be5b44b8de7ed86",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-IO-1.43-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-IO-1.43-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-IO-1.43-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-IO-1.43-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-IPC-Open3-0__1.21-471.fc33.aarch64",
    sha256 = "618496301e8cb4e6de4a443fd89358ff2307de87ee71feca20cad62a930190e7",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-IPC-Open3-0__1.21-471.fc33.x86_64",
    sha256 = "618496301e8cb4e6de4a443fd89358ff2307de87ee71feca20cad62a930190e7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-IPC-Open3-1.21-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.16-1.fc33.aarch64",
    sha256 = "47032dee12d205f84e11ea376862ca19da4bd54531b6aa9f1e291cab618526b0",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.16-1.fc33.x86_64",
    sha256 = "dd117e8fd758a274f280848f3ee9b36d3f187f9425691a98157a0bf5cc8e9c97",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.16-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-POSIX-0__1.94-471.fc33.aarch64",
    sha256 = "d33434e09c9b1a6579fa336e292193430a8b1b7a9a1881ddc2249ce64348ba88",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-POSIX-1.94-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-POSIX-1.94-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-POSIX-1.94-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-POSIX-1.94-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-POSIX-0__1.94-471.fc33.x86_64",
    sha256 = "a7ae861d633c9e898423597097f0dd295072e5148f8ad5ec59d9454d47e05fe7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-POSIX-1.94-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-POSIX-1.94-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-POSIX-1.94-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-POSIX-1.94-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-PathTools-0__3.78-458.fc33.aarch64",
    sha256 = "9e3681736a5bf3b7b3848ffc08745df124ec1f224709c9b3deb3be8992459cd2",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-PathTools-3.78-458.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-PathTools-3.78-458.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-PathTools-3.78-458.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-PathTools-3.78-458.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-PathTools-0__3.78-458.fc33.x86_64",
    sha256 = "a262c4d2632185d9ef2ee164a7c4c1f78133ce494aefdda7eeb6ed00a74f93cf",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-PathTools-3.78-458.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-PathTools-3.78-458.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-PathTools-3.78-458.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-PathTools-3.78-458.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-457.fc33.aarch64",
    sha256 = "af09b56e1d378210fa7803b1e37a39f7a4aea30b4383205fa00158300dc6c8a5",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-457.fc33.x86_64",
    sha256 = "af09b56e1d378210fa7803b1e37a39f7a4aea30b4383205fa00158300dc6c8a5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28.01-458.fc33.aarch64",
    sha256 = "954bcb74d73a76b0515c874cb759ca2c130419fa4940fc2a3772e4244e4576ab",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28.01-458.fc33.x86_64",
    sha256 = "954bcb74d73a76b0515c874cb759ca2c130419fa4940fc2a3772e4244e4576ab",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.42-1.fc33.aarch64",
    sha256 = "2cb34fad408d8fe8abda658f15a2cf28d390dfefab2e22fb6c858a7f28209074",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.42-1.fc33.x86_64",
    sha256 = "2cb34fad408d8fe8abda658f15a2cf28d390dfefab2e22fb6c858a7f28209074",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Simple-3.42-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__2.01-1.fc33.aarch64",
    sha256 = "fabaa923fa248f40e0a752875828db22be5da61db039558febfbee50cda9754c",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__2.01-1.fc33.x86_64",
    sha256 = "fabaa923fa248f40e0a752875828db22be5da61db039558febfbee50cda9754c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-4__1.55-457.fc33.aarch64",
    sha256 = "83ebd0c0ac3d53192f1be8528fa1945ad54ed9fae71331dbed7cc1709dbdbf1d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-4__1.55-457.fc33.x86_64",
    sha256 = "2b6ee7559017ae31fb817764d3d07a22262ac82d7f342f878794e7b0f096d5d8",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.55-457.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-SelectSaver-0__1.02-471.fc33.aarch64",
    sha256 = "486b9e13fb311411997be5a4869f54fd8b8b8e2a7260d4fdade12f16a7c781e6",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-SelectSaver-0__1.02-471.fc33.x86_64",
    sha256 = "486b9e13fb311411997be5a4869f54fd8b8b8e2a7260d4fdade12f16a7c781e6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-SelectSaver-1.02-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Socket-4__2.031-1.fc33.aarch64",
    sha256 = "21527a78e964c788eb601e766c5ad2833e07ffa4d2be5537064865a88982996b",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-Socket-4__2.031-1.fc33.x86_64",
    sha256 = "83bc45c1ab1355e6e1853c6cbc2f1ed0516d085cdc4810dcaf356e90bd04ad32",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-Storable-1__3.21-457.fc33.aarch64",
    sha256 = "a08edc9165d474fd2bf3a9a9bdc4dd158e7e5c98e2d4880ad8e85c113fd9af0f",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Storable-3.21-457.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Storable-3.21-457.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Storable-3.21-457.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Storable-3.21-457.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-Storable-1__3.21-457.fc33.x86_64",
    sha256 = "68939314088cf7d9fc4c157da2de6b83805dbee1149228f147170a022bc5436e",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Storable-3.21-457.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Storable-3.21-457.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Storable-3.21-457.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Storable-3.21-457.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-Symbol-0__1.08-471.fc33.aarch64",
    sha256 = "8b1941864a8de95ea565c4e4a5ee88e593b380bc89bb36df4e3fe915920faac2",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Symbol-0__1.08-471.fc33.x86_64",
    sha256 = "8b1941864a8de95ea565c4e4a5ee88e593b380bc89bb36df4e3fe915920faac2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Symbol-1.08-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Sys-Guestfs-1__1.44.1-1.fc33.x86_64",
    sha256 = "ad9f12cc01bd443a66c1bfda1a4e2505bb88f8f98493e0b4ab83697ad2186207",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Sys-Guestfs-1.44.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Sys-Guestfs-1.44.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-Sys-Guestfs-1.44.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-Sys-Guestfs-1.44.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__5.01-458.fc33.aarch64",
    sha256 = "5a658d4b9d5cd91a8d6355e1eb7fc783480cac4c23139506510eaf8fe022cb47",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__5.01-458.fc33.x86_64",
    sha256 = "5a658d4b9d5cd91a8d6355e1eb7fc783480cac4c23139506510eaf8fe022cb47",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-457.fc33.aarch64",
    sha256 = "1d0b09de8dd909c601451e66966094b6c38c72ac125f18e9568b1eebfb3c14a5",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-457.fc33.x86_64",
    sha256 = "1d0b09de8dd909c601451e66966094b6c38c72ac125f18e9568b1eebfb3c14a5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-457.fc33.aarch64",
    sha256 = "c19748d0da838388a30669965fa4959236bb4e7421b1186ed5f591792db0ec33",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-457.fc33.x86_64",
    sha256 = "c19748d0da838388a30669965fa4959236bb4e7421b1186ed5f591792db0ec33",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-457.fc33.aarch64",
    sha256 = "ce46a4e75ce7a08c55d5da99602ae4b0049eb29285df8f7dc434e2e7a6d9d24f",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-457.fc33.x86_64",
    sha256 = "ce46a4e75ce7a08c55d5da99602ae4b0049eb29285df8f7dc434e2e7a6d9d24f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Time-Local-2__1.300-4.fc33.aarch64",
    sha256 = "f034b6399c6c147dad56ea1f381a33b3a52f17023d9ab7b4fc8690813f9bf099",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-Time-Local-2__1.300-4.fc33.x86_64",
    sha256 = "f034b6399c6c147dad56ea1f381a33b3a52f17023d9ab7b4fc8690813f9bf099",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-base-0__2.27-471.fc33.aarch64",
    sha256 = "9e0d5e866d718ffc655d0cad2ad071e8cf0d963508f1a3615b9de3a2fad23865",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-base-0__2.27-471.fc33.x86_64",
    sha256 = "9e0d5e866d718ffc655d0cad2ad071e8cf0d963508f1a3615b9de3a2fad23865",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-base-2.27-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-constant-0__1.33-458.fc33.aarch64",
    sha256 = "7ff00f97f7229b747a04ba0c12b6260e63d8f995813ea596c165df64930fce07",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-constant-0__1.33-458.fc33.x86_64",
    sha256 = "7ff00f97f7229b747a04ba0c12b6260e63d8f995813ea596c165df64930fce07",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-constant-1.33-458.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-hivex-0__1.3.20-1.fc33.x86_64",
    sha256 = "b746dd7753754fb5fba33adfbd20787adff7e0723ab8c080e984b81dcfc39348",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-hivex-1.3.20-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-hivex-1.3.20-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-hivex-1.3.20-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-hivex-1.3.20-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-if-0__0.60.800-471.fc33.aarch64",
    sha256 = "79b02791a9400db190dd4023f0769fd7f9f3ccd70c7db02d1a62d3547819c250",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-if-0__0.60.800-471.fc33.x86_64",
    sha256 = "79b02791a9400db190dd4023f0769fd7f9f3ccd70c7db02d1a62d3547819c250",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-if-0.60.800-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-interpreter-4__5.32.1-471.fc33.aarch64",
    sha256 = "4fcfd472ba7c987d94843834ebbe6d9d99751f39e2e4561573255dd7f81bbe1f",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-interpreter-5.32.1-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-interpreter-5.32.1-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-interpreter-5.32.1-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-interpreter-5.32.1-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-interpreter-4__5.32.1-471.fc33.x86_64",
    sha256 = "e3915187f2c6f3175b76872266f82d4f04bddb465d80cf60b5d354c4e4f1d69a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-interpreter-5.32.1-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-interpreter-5.32.1-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-interpreter-5.32.1-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-interpreter-5.32.1-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-libintl-perl-0__1.31-8.fc33.x86_64",
    sha256 = "0a7477134c730ec391a3202d3db1916f8845e8ebbbec24178f76a7895c52b5c2",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-libintl-perl-1.31-8.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-libintl-perl-1.31-8.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-libintl-perl-1.31-8.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-libintl-perl-1.31-8.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-libs-4__5.32.1-471.fc33.aarch64",
    sha256 = "d25e59a1b31b8621be3f1c52719fde12ce4062dfd7bad3a563bbe6ebeb2cc36c",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-libs-5.32.1-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-libs-5.32.1-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-libs-5.32.1-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-libs-5.32.1-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-libs-4__5.32.1-471.fc33.x86_64",
    sha256 = "cbf2e517e49f609c7420a584f8535bd7220408695e4140d274185d0797f223b9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-libs-5.32.1-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-libs-5.32.1-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-libs-5.32.1-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-libs-5.32.1-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-locale-0__1.09-471.fc33.x86_64",
    sha256 = "eba4b9984c245abbb1ac48a8bb1b287f369f1b241785f5c1713f8b9e342fe6d4",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-locale-1.09-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-locale-1.09-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-locale-1.09-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-locale-1.09-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-macros-4__5.32.1-471.fc33.aarch64",
    sha256 = "667ccd6f5786c714112aee64705e01e8988519ff9717579d3de5afb77f7fe168",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-macros-4__5.32.1-471.fc33.x86_64",
    sha256 = "667ccd6f5786c714112aee64705e01e8988519ff9717579d3de5afb77f7fe168",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-macros-5.32.1-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-mro-0__1.23-471.fc33.aarch64",
    sha256 = "f1655f8c572e5139b7a3850e3cdc46aa58df41551574d813c988538616a1d86e",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-mro-1.23-471.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-mro-1.23-471.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-mro-1.23-471.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-mro-1.23-471.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "perl-mro-0__1.23-471.fc33.x86_64",
    sha256 = "c9ece268327cd8e42e9cdbb884f2a1820bb4d71c63f8af157aed7246e6f4525e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-mro-1.23-471.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-mro-1.23-471.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-mro-1.23-471.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-mro-1.23-471.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "perl-overload-0__1.31-471.fc33.aarch64",
    sha256 = "ca30b3529f7e1ee96dc7704663c32a1faa778b9d9d9a6e8a2e712ddccd311607",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-overload-0__1.31-471.fc33.x86_64",
    sha256 = "ca30b3529f7e1ee96dc7704663c32a1faa778b9d9d9a6e8a2e712ddccd311607",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-overload-1.31-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-overloading-0__0.02-471.fc33.aarch64",
    sha256 = "b7617a65e55bc3b06df553a9352476de130b17885e42584c97432d27de5e3420",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-overloading-0__0.02-471.fc33.x86_64",
    sha256 = "b7617a65e55bc3b06df553a9352476de130b17885e42584c97432d27de5e3420",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-overloading-0.02-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-parent-1__0.238-457.fc33.aarch64",
    sha256 = "a997aa7de2e1726461d9f6aea8feb8ac772a430a8337d4a75c161178d64a8711",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-parent-1__0.238-457.fc33.x86_64",
    sha256 = "a997aa7de2e1726461d9f6aea8feb8ac772a430a8337d4a75c161178d64a8711",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-parent-0.238-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-podlators-1__4.14-457.fc33.aarch64",
    sha256 = "29ffa425b0ad188798e10850407cf0efe1a402b50d47f26adc665d8f37011612",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-podlators-1__4.14-457.fc33.x86_64",
    sha256 = "29ffa425b0ad188798e10850407cf0efe1a402b50d47f26adc665d8f37011612",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/perl-podlators-4.14-457.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-subs-0__1.03-471.fc33.aarch64",
    sha256 = "ea1a7fc5dc1cb85dd2666a0e29b2da4bad470adce382da6574790229aa9d543a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-subs-0__1.03-471.fc33.x86_64",
    sha256 = "ea1a7fc5dc1cb85dd2666a0e29b2da4bad470adce382da6574790229aa9d543a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-subs-1.03-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-vars-0__1.05-471.fc33.aarch64",
    sha256 = "90f0a3ac479bc8f7035b3ee1c7263f0d90bbcd423645ec618504b2efb963d445",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "perl-vars-0__1.05-471.fc33.x86_64",
    sha256 = "90f0a3ac479bc8f7035b3ee1c7263f0d90bbcd423645ec618504b2efb963d445",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/perl-vars-1.05-471.fc33.noarch.rpm",
    ],
)

rpm(
    name = "pixman-0__0.40.0-2.fc33.aarch64",
    sha256 = "7bff4fbefdd619d691d3c71b2f774b1fd1ca2115bcaf389df8734166ff6c2357",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/pixman-0.40.0-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/pixman-0.40.0-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/pixman-0.40.0-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/pixman-0.40.0-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pixman-0__0.40.0-2.fc33.x86_64",
    sha256 = "908dd2b915aa93b93c0b88f6471135bc59ad33f99751b66931f51a9ac078de3b",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/pixman-0.40.0-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/pixman-0.40.0-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/pixman-0.40.0-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/pixman-0.40.0-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pkgconf-0__1.7.3-5.fc33.aarch64",
    sha256 = "239cba10fa96b50ed12b65f6efc498047a25dc2eb7168254af50db80dcfc96a7",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-1.7.3-5.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-1.7.3-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-1.7.3-5.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-1.7.3-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pkgconf-0__1.7.3-5.fc33.x86_64",
    sha256 = "c89a62b955bd7a1a8cb25f5435aa7dd03084158f185c4cd4b9c6c1caec6565f1",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-1.7.3-5.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-1.7.3-5.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-1.7.3-5.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-1.7.3-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.7.3-5.fc33.aarch64",
    sha256 = "d81cc33bad30b374e7853c943047cb70e2c4547d5028cb795ae08ca5bfa24716",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.7.3-5.fc33.x86_64",
    sha256 = "d81cc33bad30b374e7853c943047cb70e2c4547d5028cb795ae08ca5bfa24716",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-m4-1.7.3-5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.7.3-5.fc33.aarch64",
    sha256 = "003f8d21f5475b994181d43b5b141004f8cc990a6a93ce3082af89214aa1d345",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.7.3-5.fc33.x86_64",
    sha256 = "1b8f7479a6248a2d793a08d0f46dc4f6bab4c6c590bfa21778fed3a75af7945b",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/pkgconf-pkg-config-1.7.3-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "policycoreutils-0__3.1-4.fc33.aarch64",
    sha256 = "c65cb537d6338d39331d17d68a159e619b87a6641708821a3ba77e839620b5b7",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-3.1-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-3.1-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-3.1-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-3.1-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "policycoreutils-0__3.1-4.fc33.x86_64",
    sha256 = "17b93c1a952ad5bb1adeaeda811bbb014d7e5cc5b355a8d52c66737ece2a1f3e",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-3.1-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-3.1-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-3.1-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-3.1-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.1-4.fc33.aarch64",
    sha256 = "969d257b6dde56ed640f6097e7056bebc580cfb57a5351f07b2d18abd251340d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.1-4.fc33.x86_64",
    sha256 = "969d257b6dde56ed640f6097e7056bebc580cfb57a5351f07b2d18abd251340d",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.1-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "polkit-0__0.117-2.fc33.1.aarch64",
    sha256 = "1d199bea4b39569c37baae3d5c790e59171e8da374bc34352b7fb4a681968724",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/polkit-0.117-2.fc33.1.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/polkit-0.117-2.fc33.1.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/polkit-0.117-2.fc33.1.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/polkit-0.117-2.fc33.1.aarch64.rpm",
    ],
)

rpm(
    name = "polkit-0__0.117-2.fc33.1.x86_64",
    sha256 = "97e0deee75189e2de363f5214a86298af2cb1efb6722dcb0933dbfb48ee417dc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/polkit-0.117-2.fc33.1.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/polkit-0.117-2.fc33.1.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/polkit-0.117-2.fc33.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/polkit-0.117-2.fc33.1.x86_64.rpm",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-2.fc33.1.aarch64",
    sha256 = "123c6512c425587b1554c0aaf666d952bb51737b7b1d9a71cbee44cf45d851bd",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/polkit-libs-0.117-2.fc33.1.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/polkit-libs-0.117-2.fc33.1.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/polkit-libs-0.117-2.fc33.1.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/polkit-libs-0.117-2.fc33.1.aarch64.rpm",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-2.fc33.1.x86_64",
    sha256 = "1b40099a0278bd2925ce73e195b146ab15a0cbff50217dda30bcf61e106c7834",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/polkit-libs-0.117-2.fc33.1.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/polkit-libs-0.117-2.fc33.1.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/polkit-libs-0.117-2.fc33.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/polkit-libs-0.117-2.fc33.1.x86_64.rpm",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-18.fc33.aarch64",
    sha256 = "e5c3748a7c6bd607b01c4d28215ee13741f0c61c4ac3973faf7c59e5c1d25eeb",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-18.fc33.x86_64",
    sha256 = "24da45cbdddb74b51d4c7b0633e3038008e22959647d821d1e8eb716168f348f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-18.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "popt-0__1.18-2.fc33.aarch64",
    sha256 = "8b783548eda0c5d2b26754164d33cd9b4e09bde888a16a3288847c4696db40c7",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/popt-1.18-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/popt-1.18-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/popt-1.18-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/popt-1.18-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "popt-0__1.18-2.fc33.x86_64",
    sha256 = "cc87778dd52ee4ae352a1b995f4fccc4f5e2e681221f0cde738a02fea17370b1",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/popt-1.18-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/popt-1.18-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/popt-1.18-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/popt-1.18-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "procps-ng-0__3.3.16-2.fc33.aarch64",
    sha256 = "d5952d13ee3038d81dc4db761fec3ccf24deb8dbc5e621a7f33632014af0c035",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "procps-ng-0__3.3.16-2.fc33.x86_64",
    sha256 = "69591ef3d0d1144d97cacf804487e391f215cbdff760542cb65ce56d86dac1c7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-3.fc33.aarch64",
    sha256 = "0015d9856bcbdf1a2d044bc1ce1ee4b5c18431fbe320fbf6b617452b00a23ef8",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/protobuf-c-1.3.3-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/protobuf-c-1.3.3-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/protobuf-c-1.3.3-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/protobuf-c-1.3.3-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-3.fc33.x86_64",
    sha256 = "a50bbb0bb697c317090f06a43d0ca319d28744f154fc338b19094eda3cd12a00",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/protobuf-c-1.3.3-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/protobuf-c-1.3.3-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/protobuf-c-1.3.3-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/protobuf-c-1.3.3-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "psmisc-0__23.3-4.fc33.aarch64",
    sha256 = "ab54d7cc985d4c7b3e574e7aadeda2cbca1e76ee71d5d843e2e2eaf59d046838",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/psmisc-23.3-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/psmisc-23.3-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/psmisc-23.3-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/psmisc-23.3-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "psmisc-0__23.3-4.fc33.x86_64",
    sha256 = "b78eceaa9d622467cdb364d10656e0a65bdcf47cc20cd34f349b3f1c6c789ff5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/psmisc-23.3-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/psmisc-23.3-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/psmisc-23.3-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/psmisc-23.3-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python-pip-wheel-0__20.2.2-2.fc33.aarch64",
    sha256 = "cd404e9abe898e738ea1037f9051b0e6908171b35e4d1a8a80184c13a50f22bf",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python-pip-wheel-0__20.2.2-2.fc33.x86_64",
    sha256 = "cd404e9abe898e738ea1037f9051b0e6908171b35e4d1a8a80184c13a50f22bf",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python-pip-wheel-20.2.2-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python-setuptools-wheel-0__49.1.3-2.fc33.aarch64",
    sha256 = "333fe6c2a9774daee73388788752fc8fe227d996caa4d2d75fcf7e8db8347537",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python-setuptools-wheel-0__49.1.3-2.fc33.x86_64",
    sha256 = "333fe6c2a9774daee73388788752fc8fe227d996caa4d2d75fcf7e8db8347537",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python-setuptools-wheel-49.1.3-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-0__3.9.6-2.fc33.aarch64",
    sha256 = "848ed8fa3aa01aac96dd56e21c99844352dca2dd4254ec36494d0fcd1f2c2d5e",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-3.9.6-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/python3-3.9.6-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-3.9.6-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-3.9.6-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "python3-0__3.9.6-2.fc33.x86_64",
    sha256 = "b165605408ccd1664ca5a419dc2ed7df39d6a276115a0db07eb3db2f9cef692b",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-3.9.6-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-3.9.6-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-3.9.6-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-3.9.6-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-audit-0__3.0.3-1.fc33.aarch64",
    sha256 = "f8a8e1cf8852192286aa85036412d01b58c6cb9729c43361ddb2f39bc8820af9",
    urls = [
        "https://mirrors.xtom.ee/fedora/updates/33/Everything/aarch64/Packages/p/python3-audit-3.0.3-1.fc33.aarch64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-audit-3.0.3-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-audit-3.0.3-1.fc33.aarch64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-audit-3.0.3-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "python3-audit-0__3.0.3-1.fc33.x86_64",
    sha256 = "546454745b3637b2681e99e21231eb50f45a269791b649c2092029207c4a9309",
    urls = [
        "https://ftp.byfly.by/pub/fedoraproject.org/linux/updates/33/Everything/x86_64/Packages/p/python3-audit-3.0.3-1.fc33.x86_64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-audit-3.0.3-1.fc33.x86_64.rpm",
        "https://mirror.23m.com/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-audit-3.0.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-audit-3.0.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-dateutil-1__2.8.1-2.fc33.x86_64",
    sha256 = "c1c8e77d2f5ef170e3c9a0b01552eb4c3b22c9af10f39eb48ba1bbaea4cc6828",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-dateutil-2.8.1-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-dateutil-2.8.1-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-dateutil-2.8.1-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-dateutil-2.8.1-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-dbus-0__1.2.16-3.fc33.x86_64",
    sha256 = "82de6ab9664b5d79f62beb16afe00ad99d79e768c95aee8cca40c588196e9931",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-dbus-1.2.16-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-dbus-1.2.16-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-dbus-1.2.16-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-dbus-1.2.16-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-distro-0__1.5.0-4.fc33.x86_64",
    sha256 = "bc8c25957141f6eac2f7a965964f2a06b0ef658e12ca1786de5194f2835345f6",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-distro-1.5.0-4.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-distro-1.5.0-4.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-distro-1.5.0-4.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-distro-1.5.0-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-dnf-0__4.8.0-1.fc33.x86_64",
    sha256 = "9ae24a964c5918668558ddae5e5cd4ff40892e882e3bb1191a8ee8fe631f555a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-4.8.0-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-4.8.0-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-4.8.0-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-4.8.0-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-dnf-plugins-core-0__4.0.22-1.fc33.x86_64",
    sha256 = "f07f99a86b4b9a87fcc5f8a88dc504aa6c1630c55f27ec5e690df2c33a9b0458",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-dnf-plugins-core-4.0.22-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-gpg-0__1.14.0-2.fc33.x86_64",
    sha256 = "8b98a717abbd38d8aa4cc7c403985e591518d23db5f74489bed2dcd410c8a1ac",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-gpg-1.14.0-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-gpg-1.14.0-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-gpg-1.14.0-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-gpg-1.14.0-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-hawkey-0__0.63.1-1.fc33.x86_64",
    sha256 = "fa41bf154b8cfed94e27f8d06ff33606a65b5f2abe58ed43cbfae47e083794f3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-hawkey-0.63.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-hawkey-0.63.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-hawkey-0.63.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-hawkey-0.63.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libcomps-0__0.1.17-1.fc33.x86_64",
    sha256 = "420e54a303a7db5a6acf8b2bd693192f4a44c656434a1a4c632ea5a483d442dc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libcomps-0.1.17-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libcomps-0.1.17-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-libcomps-0.1.17-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libcomps-0.1.17-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libdnf-0__0.63.1-1.fc33.x86_64",
    sha256 = "38ed537da81e80d3e95736aaf604b623ef4687081ed20b719241ec5f36ff41d3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libdnf-0.63.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libdnf-0.63.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-libdnf-0.63.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libdnf-0.63.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libs-0__3.9.6-2.fc33.aarch64",
    sha256 = "71c61bde986b19a797e090acbfe8421cfbf81d2dbc1d3d1381b17a7ccc51a18f",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-libs-3.9.6-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/python3-libs-3.9.6-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-libs-3.9.6-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-libs-3.9.6-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libs-0__3.9.6-2.fc33.x86_64",
    sha256 = "9059824c6b60155f3bb0914c73291dc5200532d4c72a9520ece9d5767445cb8f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libs-3.9.6-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libs-3.9.6-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-libs-3.9.6-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-libs-3.9.6-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libselinux-0__3.1-2.fc33.aarch64",
    sha256 = "0cad78eacdcf7dc53d692787269b5da44be4eb35a0b98ae8aca8b180831fdf45",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-libselinux-3.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-libselinux-3.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-libselinux-3.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-libselinux-3.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libselinux-0__3.1-2.fc33.x86_64",
    sha256 = "01a57ff02d3050490a7acc3265de4c2395c2ce7e4f5d3a1f2a453508a7f51284",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-libselinux-3.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-libselinux-3.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-libselinux-3.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-libselinux-3.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.1-2.fc33.aarch64",
    sha256 = "639ea1fb8bbfd9326bf8e4658a99779a10e712721f661c05b3b66b7424177a1b",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-libsemanage-3.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-libsemanage-3.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-libsemanage-3.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-libsemanage-3.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.1-2.fc33.x86_64",
    sha256 = "127da89c822f7495a46f86eb5442da9918e61000b98825132c5edee5add83296",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-libsemanage-3.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-libsemanage-3.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-libsemanage-3.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-libsemanage-3.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-policycoreutils-0__3.1-4.fc33.aarch64",
    sha256 = "fd0687173f01ca3046ea0a33ad371f0584fe9eefed491f9c0e516982370e09bb",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-policycoreutils-0__3.1-4.fc33.x86_64",
    sha256 = "fd0687173f01ca3046ea0a33ad371f0584fe9eefed491f9c0e516982370e09bb",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.1-4.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-rpm-0__4.16.1.3-1.fc33.x86_64",
    sha256 = "c48bba88b5bc88e741c34fe94caa3dfd76dbe17b320639c4caf7a5814aa661be",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-rpm-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-rpm-4.16.1.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-rpm-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-rpm-4.16.1.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-setools-0__4.3.0-5.fc33.aarch64",
    sha256 = "251fa497d84556879121bc12598daff1b4c993c4b6ba94d8905733cdeea39dfb",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "python3-setools-0__4.3.0-5.fc33.x86_64",
    sha256 = "80d364c9512d2021ab15fbcc685761b1363e233d6bb1a0384438f8bafccc76ac",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "python3-setuptools-0__49.1.3-2.fc33.aarch64",
    sha256 = "5254de2c527ec7adf677fb94e2f9328e88a22a15bc7d5f9ee780768676b6f563",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-setuptools-0__49.1.3-2.fc33.x86_64",
    sha256 = "5254de2c527ec7adf677fb94e2f9328e88a22a15bc7d5f9ee780768676b6f563",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/p/python3-setuptools-49.1.3-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "python3-six-0__1.15.0-2.fc33.x86_64",
    sha256 = "cdad5f33eb5005d565e3301eea9dbdfd74b6b231c5c914a1711b35c7c266bc86",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-six-1.15.0-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-six-1.15.0-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/p/python3-six-1.15.0-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/p/python3-six-1.15.0-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "qemu-img-15__5.2.0-15.fc33.aarch64",
    sha256 = "9bb88edb80f54e6b0f4356c56d57f1f78422f860774d6d2dfa2773276379b536",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-33-aarch64/02142033-qemu-kvm/qemu-img-5.2.0-15.fc33.aarch64.rpm"],
)

rpm(
    name = "qemu-img-15__5.2.0-15.fc33.x86_64",
    sha256 = "d2929afafcbc29448a8969f93669ac66e6443a58089e4155c3e12b5c48f19538",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-33-x86_64/02142033-qemu-kvm/qemu-img-5.2.0-15.fc33.x86_64.rpm"],
)

rpm(
    name = "qemu-kvm-common-15__5.2.0-15.fc33.aarch64",
    sha256 = "a582145532b2010930748f0ee8a5894aedbf2f9bd00b9abe8d70c787502331e8",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-33-aarch64/02142033-qemu-kvm/qemu-kvm-common-5.2.0-15.fc33.aarch64.rpm"],
)

rpm(
    name = "qemu-kvm-common-15__5.2.0-15.fc33.x86_64",
    sha256 = "27fc07879ca5588fcc4ae21eed6cda6a20d36db7c538b576192c432aeebd38be",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-33-x86_64/02142033-qemu-kvm/qemu-kvm-common-5.2.0-15.fc33.x86_64.rpm"],
)

rpm(
    name = "qemu-kvm-core-15__5.2.0-15.fc33.aarch64",
    sha256 = "9befb434fcf209b2e195acfca2e386bdad7f274f4bc2988c64bf843dbb2af1bd",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-33-aarch64/02142033-qemu-kvm/qemu-kvm-core-5.2.0-15.fc33.aarch64.rpm"],
)

rpm(
    name = "qemu-kvm-core-15__5.2.0-15.fc33.x86_64",
    sha256 = "453fac30d0ab4109baa0cc7e402bc3306b634601e1ca43ee51d378f1d907b2ba",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-33-x86_64/02142033-qemu-kvm/qemu-kvm-core-5.2.0-15.fc33.x86_64.rpm"],
)

rpm(
    name = "qrencode-libs-0__4.0.2-6.fc33.aarch64",
    sha256 = "c9fafb74e8561426b6d750416582968ac45fa682e65db70018edb2ccdbcb874d",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "qrencode-libs-0__4.0.2-6.fc33.x86_64",
    sha256 = "edac6ebce4c4b01843b09d55bed7dfdb08f1b9cad2e631bdcc2692df859c7a31",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "rdma-core-0__35.0-1.fc33.aarch64",
    sha256 = "9c9a84a749bf326f15563773f7459b6d7f78e38127ae5243568e9dd7f5f9cf3a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/r/rdma-core-35.0-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/r/rdma-core-35.0-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/r/rdma-core-35.0-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/r/rdma-core-35.0-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "rdma-core-0__35.0-1.fc33.x86_64",
    sha256 = "7db3b43c81fc06508045073a45e9a8bc0baf2d191eb7f48f2e10bd7bab094b99",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rdma-core-35.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/r/rdma-core-35.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/r/rdma-core-35.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rdma-core-35.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "readline-0__8.0-5.fc33.aarch64",
    sha256 = "c8b8da766839a2a310f32bf661c6286fd092103111c92749401eb3312d6584a3",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/r/readline-8.0-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/r/readline-8.0-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/r/readline-8.0-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/r/readline-8.0-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "readline-0__8.0-5.fc33.x86_64",
    sha256 = "b2ae93e93832a2ee3ad70637722704df839e7a6123c966a1c61c86b663c736a3",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/r/readline-8.0-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/r/readline-8.0-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/r/readline-8.0-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/r/readline-8.0-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-1.fc33.aarch64",
    sha256 = "4910c09fe1083dc4b6b879d8d6b503553d92594e3f4507be55ae8f02e37aa895",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-4.16.1.3-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/r/rpm-4.16.1.3-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-4.16.1.3-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-4.16.1.3-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-1.fc33.x86_64",
    sha256 = "45e7cc65ee20cd1c288ecca379b7edf94b39cceb1e7b7e1f4493af14a5e2fc3f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-4.16.1.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/r/rpm-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-4.16.1.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "rpm-build-libs-0__4.16.1.3-1.fc33.x86_64",
    sha256 = "74de607dcebdc6e8189002106509e1267c4e66194a84145e97929d1f9e084d05",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-build-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-build-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/r/rpm-build-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-build-libs-4.16.1.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-1.fc33.aarch64",
    sha256 = "6bad58e517629b6a8192dcec7672f44d415e67cbef1dd2e985bde02f266780ff",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-libs-4.16.1.3-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/r/rpm-libs-4.16.1.3-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-libs-4.16.1.3-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-libs-4.16.1.3-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-1.fc33.x86_64",
    sha256 = "3149539f92082e97357f6eacd092af0e01bd6cba862d9426b8627c48f2e6a0e7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/r/rpm-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-libs-4.16.1.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-1.fc33.aarch64",
    sha256 = "addab697aca8321f667bc28fd78b78c1c239285ec23a31b21fc7c0eee12f4ea1",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-1.fc33.x86_64",
    sha256 = "4f46dec8f8f7e46034ff57909ff3c611c7afb44b94204283240fffe43a7c1bc1",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.16.1.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "rpm-sign-libs-0__4.16.1.3-1.fc33.x86_64",
    sha256 = "b6861222f7f84138080e01a32ee5dd7a6c284de9407679bb95679ff133062752",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-sign-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-sign-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/r/rpm-sign-libs-4.16.1.3-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/r/rpm-sign-libs-4.16.1.3-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "scrub-0__2.6.1-2.fc33.x86_64",
    sha256 = "3bfefb480fe1eeb8f0041be5803194e8b110524d06800ba7be90c0db0110d10d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "scsi-target-utils-0__1.0.79-2.fc33.aarch64",
    sha256 = "98b2ff9cb078d8087a237e48ba54a7817a68c2bb0873b31271e76d094a962780",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "scsi-target-utils-0__1.0.79-2.fc33.x86_64",
    sha256 = "7a5f06e2273c889153da0024586f3009419b99a4c4e85850501d52a018d5e366",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "seabios-0__1.14.0-1.fc33.x86_64",
    sha256 = "77079cb8f1053ab98671ed8081584cbafda502f2c5bd7d02d087f182ee29266f",
    urls = ["https://download.copr.fedorainfracloud.org/results/@kubevirt/seabios-1.14.0-1.el8/fedora-33-x86_64/01822781-seabios/seabios-1.14.0-1.fc33.x86_64.rpm"],
)

rpm(
    name = "seabios-bin-0__1.14.0-1.fc33.x86_64",
    sha256 = "9bc6d2a010f61a3cb2d094194ae58a79363f725632abf97067d176819d2cd760",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/seabios-bin-1.14.0-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/seabios-bin-1.14.0-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/seabios-bin-1.14.0-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/seabios-bin-1.14.0-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.14.0-1.fc33.x86_64",
    sha256 = "db41a0588904c245df622c7d04397e9ba58eed75baa7a9823b03e3bebf453210",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/seavgabios-bin-1.14.0-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/seavgabios-bin-1.14.0-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/seavgabios-bin-1.14.0-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/seavgabios-bin-1.14.0-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "sed-0__4.8-5.fc33.aarch64",
    sha256 = "0bde87180700f75304bc5b6919dbf627c8794b507684e7f211dd0ea8ddc18f98",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/s/sed-4.8-5.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/sed-4.8-5.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/s/sed-4.8-5.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/sed-4.8-5.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "sed-0__4.8-5.fc33.x86_64",
    sha256 = "be84b00378dbeb0b8e276ec62aa94b73f53a9ec02349deb91b2e3b59558a8fd1",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sed-4.8-5.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sed-4.8-5.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sed-4.8-5.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/sed-4.8-5.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.6-39.fc33.aarch64",
    sha256 = "586c8d82f66bde7c3853c406f6bec5b732947dad0199203ba3a69568117266c3",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.6-39.fc33.x86_64",
    sha256 = "586c8d82f66bde7c3853c406f6bec5b732947dad0199203ba3a69568117266c3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-3.14.6-39.fc33.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.6-39.fc33.aarch64",
    sha256 = "8296a40f715031f30410a0299a5443239db92300aadf960a74a0ad65a1715647",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.6-39.fc33.x86_64",
    sha256 = "8296a40f715031f30410a0299a5443239db92300aadf960a74a0ad65a1715647",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.6-39.fc33.noarch.rpm",
    ],
)

rpm(
    name = "setup-0__2.13.7-2.fc33.aarch64",
    sha256 = "74d8bf336378256d01cbdb40a8972b0c00ea4b7d433a5c9d5dad704ed5188555",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "setup-0__2.13.7-2.fc33.x86_64",
    sha256 = "74d8bf336378256d01cbdb40a8972b0c00ea4b7d433a5c9d5dad704ed5188555",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/setup-2.13.7-2.fc33.noarch.rpm",
    ],
)

rpm(
    name = "sg3_utils-0__1.45-3.fc33.aarch64",
    sha256 = "76677a181fec694ba8c701f30ab277aa8c327245d8acf9caf5af06f09a81e099",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-1.45-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-1.45-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-1.45-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-1.45-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "sg3_utils-0__1.45-3.fc33.x86_64",
    sha256 = "657572b76a9b312c984db7863d235a0164b35e6a6b9c5cf576bbea360d3cba29",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-1.45-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-1.45-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-1.45-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-1.45-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "sg3_utils-libs-0__1.45-3.fc33.aarch64",
    sha256 = "7d92e54ee42486088d92ea2220d36b33c0afea82984b9050692ba6e59c6780e7",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "sg3_utils-libs-0__1.45-3.fc33.x86_64",
    sha256 = "3021f9102907bb46a82646a3c23c0c03f78a9bc864aaabccd5581cca5bf10704",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.45-3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "sgabios-bin-1__0.20180715git-5.fc33.x86_64",
    sha256 = "40af59ac7229ba71e4611a068f8298fdcd485ee7a763b1e05f55ea7277938c57",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-5.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-5.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-5.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-5.fc33.noarch.rpm",
    ],
)

rpm(
    name = "shadow-utils-2__4.8.1-6.fc33.aarch64",
    sha256 = "b970d6a0a84e89223cf935b9c2ad887b462651926df5aa5ad4bbae17e973050a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/shadow-utils-4.8.1-6.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/shadow-utils-4.8.1-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/shadow-utils-4.8.1-6.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/shadow-utils-4.8.1-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "shadow-utils-2__4.8.1-6.fc33.x86_64",
    sha256 = "5be2aa0259fc2f06731ea1d8d36d286e2e331edf39cb117984dad5cb71d70f8e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/shadow-utils-4.8.1-6.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/shadow-utils-4.8.1-6.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/shadow-utils-4.8.1-6.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/shadow-utils-4.8.1-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "snappy-0__1.1.8-4.fc33.aarch64",
    sha256 = "2d72eabbcab8da5c1996b16d7795e9077d5700131d76af209259be7d7ca8c5ae",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/s/snappy-1.1.8-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/snappy-1.1.8-4.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/s/snappy-1.1.8-4.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/s/snappy-1.1.8-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "snappy-0__1.1.8-4.fc33.x86_64",
    sha256 = "0e8032be4085a8b3193d8507852d41fb6e6b757e6958cfd9542676255fe4d3d4",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/snappy-1.1.8-4.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/snappy-1.1.8-4.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/snappy-1.1.8-4.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/snappy-1.1.8-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-1.fc33.aarch64",
    sha256 = "aa0ec281edea00858302ddb983830d627437ed75cc5081d2a7f6691379489bf8",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/sqlite-libs-3.34.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/sqlite-libs-3.34.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/sqlite-libs-3.34.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/sqlite-libs-3.34.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-1.fc33.x86_64",
    sha256 = "ed176422a687d280bcf0f55a5556501e0db324b2fdfbb48c6113e844f1baf3b9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/sqlite-libs-3.34.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/sqlite-libs-3.34.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/sqlite-libs-3.34.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/sqlite-libs-3.34.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "squashfs-tools-0__4.4-2.20200513gitc570c61.fc33.x86_64",
    sha256 = "882bed6efcb5c905fb4aca877fb9ad67f9ca9bdfb4c8349716c0d049bdcf0a06",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/squashfs-tools-4.4-2.20200513gitc570c61.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/squashfs-tools-4.4-2.20200513gitc570c61.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/squashfs-tools-4.4-2.20200513gitc570c61.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/squashfs-tools-4.4-2.20200513gitc570c61.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "supermin-0__5.2.1-1.fc33.x86_64",
    sha256 = "0f91ac66bedb36543d06190c1403a906f25b27b9905dfdda5cd451006eb0b076",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "swtpm-0__0.6.0-1.20210607gitea627b3.fc33.aarch64",
    sha256 = "f8f5111df56a4b94ceca94211a95b715ec1b701cc30fbeb6e952f5b696111f14",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "swtpm-0__0.6.0-1.20210607gitea627b3.fc33.x86_64",
    sha256 = "40d94bc96c90eeace03e9d1d4f06487f3f7fd38dd582b5be46c01883855497b7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "swtpm-libs-0__0.6.0-1.20210607gitea627b3.fc33.aarch64",
    sha256 = "15d97c2fba7f6227db7294fa4824405033a38f1587a60f6f605abc814e3bcd09",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "swtpm-libs-0__0.6.0-1.20210607gitea627b3.fc33.x86_64",
    sha256 = "fa61b6d850c42ce674c40db3726a2b6eb765adb3ed4e493bc3777d16ba8eb719",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "swtpm-tools-0__0.6.0-1.20210607gitea627b3.fc33.aarch64",
    sha256 = "843928af61063cd174ba945f50988a4627d537c3d891718a8e48817fc9c7a941",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "swtpm-tools-0__0.6.0-1.20210607gitea627b3.fc33.x86_64",
    sha256 = "6b4859b45602c0b66e9ca8c203efb6fb7fa5906c5816b578af209c4e4e9ac618",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "syslinux-0__6.04-0.16.fc33.x86_64",
    sha256 = "304cb90329d2eabe3d44e774758d296e594c0a647a2009212529e8eb0cb94c21",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-6.04-0.16.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-6.04-0.16.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-6.04-0.16.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/syslinux-6.04-0.16.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "syslinux-extlinux-0__6.04-0.16.fc33.x86_64",
    sha256 = "a4c50b0175c112e5cbfc180e3468eca871ba1289f9e287f9e397b163ddd764a6",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.16.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.16.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.16.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.16.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "syslinux-extlinux-nonlinux-0__6.04-0.16.fc33.x86_64",
    sha256 = "af2777ec0c6ee867b5ecad024de55fce1afccb5567cd48501611e073536d332f",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
    ],
)

rpm(
    name = "syslinux-nonlinux-0__6.04-0.16.fc33.x86_64",
    sha256 = "a89afa93b4570d941838daa89cfaa671fa11559626c9ed976b15a63a9b3c5e4e",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.16.fc33.noarch.rpm",
    ],
)

rpm(
    name = "systemd-0__246.15-1.fc33.aarch64",
    sha256 = "0671b867ce55aeef2a25b7b079f79c577b01c1c44dc0b2c797480ae4ed06a62a",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-246.15-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/systemd-246.15-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-246.15-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-246.15-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-0__246.15-1.fc33.x86_64",
    sha256 = "33a402be41f0156585c01363c461514d4e96fd727ae2f765a525d0e0db0f9323",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-246.15-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-246.15-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/systemd-246.15-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-246.15-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "systemd-container-0__246.15-1.fc33.aarch64",
    sha256 = "47fde31277b18dab2d3a4b564bf2d3ecf361f92b8d1edd531d143ab89d90ec67",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-container-246.15-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/systemd-container-246.15-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-container-246.15-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-container-246.15-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-container-0__246.15-1.fc33.x86_64",
    sha256 = "78f8bfa3a4e6e8b2fda2230e70f9cff571c1fd6647e4d97a771860704b6d2dbb",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-container-246.15-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-container-246.15-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/systemd-container-246.15-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-container-246.15-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "systemd-libs-0__246.15-1.fc33.aarch64",
    sha256 = "ce0e4b2a5c2715427bc046cbfb3fc13c50d86beaa567a957b2de97a4c38e43c0",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-libs-246.15-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/systemd-libs-246.15-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-libs-246.15-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-libs-246.15-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-libs-0__246.15-1.fc33.x86_64",
    sha256 = "009f27dcccbc749ce6ece223258504367fc41081384668c0170a0014167b5780",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-libs-246.15-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-libs-246.15-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/systemd-libs-246.15-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-libs-246.15-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "systemd-pam-0__246.15-1.fc33.aarch64",
    sha256 = "754156887a1786609ce736c09053a8b8dbabbf14cb9cf0ed70ee566074496b5e",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-pam-246.15-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/systemd-pam-246.15-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-pam-246.15-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-pam-246.15-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-pam-0__246.15-1.fc33.x86_64",
    sha256 = "b56fa2b398e44201dbdd241c09c95873825c873ef07c82feb3d86d028ab1aefe",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-pam-246.15-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-pam-246.15-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/systemd-pam-246.15-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-pam-246.15-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__246.15-1.fc33.aarch64",
    sha256 = "e6f4e9d9a9ac04527d86e674fce1f6deec109b9b7762bf83bc0b3fbe72139bf9",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__246.15-1.fc33.x86_64",
    sha256 = "e6f4e9d9a9ac04527d86e674fce1f6deec109b9b7762bf83bc0b3fbe72139bf9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-rpm-macros-246.15-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "systemd-udev-0__246.15-1.fc33.x86_64",
    sha256 = "74648d5fb5f5e87a32d8a8a0f464e5356d4981751ce04dafe56c1b0a70fb5cf2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-udev-246.15-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-udev-246.15-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/s/systemd-udev-246.15-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/s/systemd-udev-246.15-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "tar-2__1.32-6.fc33.aarch64",
    sha256 = "5a692f3e7457c4e4e2c0a4b305aceba8d5c91c54bde56a64f2901013a2856dc5",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/t/tar-1.32-6.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/t/tar-1.32-6.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/t/tar-1.32-6.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/t/tar-1.32-6.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "tar-2__1.32-6.fc33.x86_64",
    sha256 = "871dc18514b9b64bcff6c4c61fd4c1a9f4c1e46cddd6f6934b4ee93662541aca",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/t/tar-1.32-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/t/tar-1.32-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/t/tar-1.32-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/t/tar-1.32-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "tpm2-tss-0__3.0.4-1.fc33.x86_64",
    sha256 = "8d508e8265f106a7203ed1914e2a90f1903b5c49ed67fb592871658cb382afab",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/tpm2-tss-3.0.4-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/t/tpm2-tss-3.0.4-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/t/tpm2-tss-3.0.4-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/tpm2-tss-3.0.4-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "trousers-0__0.3.14-4.fc33.aarch64",
    sha256 = "cecae4dcd8b383433c5d30107b9bf342340ab9d492a8ab8a79080f1e07c74ea5",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/t/trousers-0.3.14-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/t/trousers-0.3.14-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/t/trousers-0.3.14-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/t/trousers-0.3.14-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "trousers-0__0.3.14-4.fc33.x86_64",
    sha256 = "88c72f34f6f2dc11a72f4d2117753af14006473a7a0960b50cf32d7f8cf5691c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/trousers-0.3.14-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/t/trousers-0.3.14-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/t/trousers-0.3.14-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/trousers-0.3.14-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "trousers-lib-0__0.3.14-4.fc33.aarch64",
    sha256 = "6f8cac5cfdd2125866a189dd7cdac522688ce2851fb79734b6add2616f3a210d",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/t/trousers-lib-0.3.14-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/t/trousers-lib-0.3.14-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/t/trousers-lib-0.3.14-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/t/trousers-lib-0.3.14-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "trousers-lib-0__0.3.14-4.fc33.x86_64",
    sha256 = "a00967da073030cbaa21cb1dcfbf98320ba01d825ba3b30840f6c4cd7b6a9506",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/trousers-lib-0.3.14-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/t/trousers-lib-0.3.14-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/t/trousers-lib-0.3.14-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/trousers-lib-0.3.14-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "tzdata-0__2021a-1.fc33.aarch64",
    sha256 = "4f451cb7e2f24240f8851c9317d9c565dbea94d10e793cb579d3c22ff5b33540",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "tzdata-0__2021a-1.fc33.x86_64",
    sha256 = "4f451cb7e2f24240f8851c9317d9c565dbea94d10e793cb579d3c22ff5b33540",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/t/tzdata-2021a-1.fc33.noarch.rpm",
    ],
)

rpm(
    name = "unbound-libs-0__1.13.1-1.fc33.aarch64",
    sha256 = "72010693fa0a887dc45fd31b3e68e7a20aa39e7b5e376d6a76775f74de9c5415",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/u/unbound-libs-1.13.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/u/unbound-libs-1.13.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/u/unbound-libs-1.13.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/u/unbound-libs-1.13.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "unbound-libs-0__1.13.1-1.fc33.x86_64",
    sha256 = "7bf2d711ae8933cfd8d55dadddb033fcc1485cff77e2a1b373d47883cfd67ec7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/u/unbound-libs-1.13.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/u/unbound-libs-1.13.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/u/unbound-libs-1.13.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/u/unbound-libs-1.13.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "usbredir-0__0.10.0-1.fc33.x86_64",
    sha256 = "15c68ef8af4f5d9b82673eedc56c9fabc91f461cdd3ef2b9c250c5c7748b8001",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/u/usbredir-0.10.0-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/u/usbredir-0.10.0-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/u/usbredir-0.10.0-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/u/usbredir-0.10.0-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "userspace-rcu-0__0.12.1-2.fc33.aarch64",
    sha256 = "2009cb5bb6f29b0dd70d8e5d1488e806e5634c53420a91a38f290ab8940b32ac",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "userspace-rcu-0__0.12.1-2.fc33.x86_64",
    sha256 = "3dd12ce86657ef9b40829e74e79d7933cf815ced3b24628c8883aad178393876",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/u/userspace-rcu-0.12.1-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "util-linux-0__2.36.1-1.fc33.aarch64",
    sha256 = "27d3d536156b6db9b769c4f8577af9dd62fb6b26c570d2177520c99b3b8b18f5",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/u/util-linux-2.36.1-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/u/util-linux-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/u/util-linux-2.36.1-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/u/util-linux-2.36.1-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "util-linux-0__2.36.1-1.fc33.x86_64",
    sha256 = "3a0f24014ccc213f0a6317371a4d2dd8ea7311bd0a318d240a6e6e33ac83ef73",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/u/util-linux-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/u/util-linux-2.36.1-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/u/util-linux-2.36.1-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/u/util-linux-2.36.1-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.3182-1.fc33.aarch64",
    sha256 = "69cef1c9bdbc2df55bfe155f4ae71640dd2879686a4af3c0fd67705ef4b6e5c2",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/v/vim-minimal-8.2.3182-1.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/v/vim-minimal-8.2.3182-1.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/v/vim-minimal-8.2.3182-1.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/v/vim-minimal-8.2.3182-1.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.3182-1.fc33.x86_64",
    sha256 = "46f2dbe5516fae22f09a405f43d3e912421ffcd5ef5fb4675c2c87cd1cdb7186",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/v/vim-minimal-8.2.3182-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/v/vim-minimal-8.2.3182-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/v/vim-minimal-8.2.3182-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/v/vim-minimal-8.2.3182-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "which-0__2.21-20.fc33.aarch64",
    sha256 = "872cdf7f0ff3009c8e8f9a2539f215a2d38f0520d7e39ef21a78ba72cf648d71",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/w/which-2.21-20.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/w/which-2.21-20.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/w/which-2.21-20.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/w/which-2.21-20.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "which-0__2.21-20.fc33.x86_64",
    sha256 = "caa8f3aebd5fd12202d3ff568c4cb1be7f0be824be9d2b676bee36c81e13134b",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/w/which-2.21-20.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/w/which-2.21-20.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/w/which-2.21-20.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/w/which-2.21-20.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.30-3.fc33.aarch64",
    sha256 = "35196132ff5c616da01afede106f96ce15841e1aa0827a7ef8f1f175067534c5",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.30-3.fc33.x86_64",
    sha256 = "35196132ff5c616da01afede106f96ce15841e1aa0827a7ef8f1f175067534c5",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/x/xkeyboard-config-2.30-3.fc33.noarch.rpm",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-2.fc33.aarch64",
    sha256 = "5d8f79b3e5c62617eb5cf7890ffe23b234c78de728e70d2e3ab6ee3db35014f9",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-2.fc33.x86_64",
    sha256 = "a2e3a2dd3c4dff8d44ccd7c6c8b9457107e8916979ca12d1984b9967c101f4ec",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "xz-0__5.2.5-4.fc33.aarch64",
    sha256 = "ec14838c35155279ebef4988c0d03672ab6ea6b68f2faf2ee82ff01417bae656",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/x/xz-5.2.5-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/x/xz-5.2.5-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/x/xz-5.2.5-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/x/xz-5.2.5-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "xz-0__5.2.5-4.fc33.x86_64",
    sha256 = "9fbdb741d5796c16e764719b77f05376134c9dfba44d296ac9a9d20af98e0d5c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/x/xz-5.2.5-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/x/xz-5.2.5-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/x/xz-5.2.5-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/x/xz-5.2.5-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-4.fc33.aarch64",
    sha256 = "c88b40620209645160c086a8b474cc6c4a1b2d28e95d4b0517aa7d58e4e95fdb",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/x/xz-libs-5.2.5-4.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/x/xz-libs-5.2.5-4.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/x/xz-libs-5.2.5-4.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/x/xz-libs-5.2.5-4.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-4.fc33.x86_64",
    sha256 = "fb687ca58b810f43aab51ca093ee95e2ea70bb2deb5521dd0cc6d4d53ab143ed",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/x/xz-libs-5.2.5-4.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/x/xz-libs-5.2.5-4.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/x/xz-libs-5.2.5-4.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/x/xz-libs-5.2.5-4.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "yajl-0__2.1.0-15.fc33.aarch64",
    sha256 = "05b788619e15a389015f14b76c1d93109eb14f3a9d0afeba1d0c119f412b30d0",
    urls = [
        "https://mirrors.xtom.de/fedora/releases/33/Everything/aarch64/os/Packages/y/yajl-2.1.0-15.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/33/Everything/aarch64/os/Packages/y/yajl-2.1.0-15.fc33.aarch64.rpm",
        "https://mirror.ihost.md/fedora/releases/33/Everything/aarch64/os/Packages/y/yajl-2.1.0-15.fc33.aarch64.rpm",
        "https://www.mirrorservice.org/sites/dl.fedoraproject.org/pub/fedora/linux/releases/33/Everything/aarch64/os/Packages/y/yajl-2.1.0-15.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "yajl-0__2.1.0-15.fc33.x86_64",
    sha256 = "bbfd6ad6b0aa4adeae2770ecfee7521ee487de40294bfda2e5ffedf21fbffae2",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/y/yajl-2.1.0-15.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/y/yajl-2.1.0-15.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/y/yajl-2.1.0-15.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/y/yajl-2.1.0-15.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "zchunk-libs-0__1.1.15-1.fc33.x86_64",
    sha256 = "fbe2e7fc4b2111070598d1a7dd26c640ac54e508af201c694601321d93151c72",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/z/zchunk-libs-1.1.15-1.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/z/zchunk-libs-1.1.15-1.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/z/zchunk-libs-1.1.15-1.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/z/zchunk-libs-1.1.15-1.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "zerofree-0__1.1.1-6.fc33.x86_64",
    sha256 = "1cc54aa2662378ffab333f9341afc41b8ecaf64a6fa1432b2d23d208040d0ebd",
    urls = [
        "https://mirror.vpsnet.com/fedora/linux/releases/33/Everything/x86_64/os/Packages/z/zerofree-1.1.1-6.fc33.x86_64.rpm",
        "https://mirror.karneval.cz/pub/linux/fedora/linux/releases/33/Everything/x86_64/os/Packages/z/zerofree-1.1.1-6.fc33.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/33/Everything/x86_64/os/Packages/z/zerofree-1.1.1-6.fc33.x86_64.rpm",
        "https://mirror.telepoint.bg/fedora/releases/33/Everything/x86_64/os/Packages/z/zerofree-1.1.1-6.fc33.x86_64.rpm",
    ],
)

rpm(
    name = "zlib-0__1.2.11-23.fc33.aarch64",
    sha256 = "82f6070fd2851f8da9636a6d802c0adb4f0db562f58595406d0e428b4c2160a7",
    urls = [
        "https://mirror.yandex.ru/fedora/linux/updates/33/Everything/aarch64/Packages/z/zlib-1.2.11-23.fc33.aarch64.rpm",
        "https://fedora.mirror.liteserver.nl/linux/updates/33/Everything/aarch64/Packages/z/zlib-1.2.11-23.fc33.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/aarch64/Packages/z/zlib-1.2.11-23.fc33.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/aarch64/Packages/z/zlib-1.2.11-23.fc33.aarch64.rpm",
    ],
)

rpm(
    name = "zlib-0__1.2.11-23.fc33.x86_64",
    sha256 = "0a112a32975de398821aef60297faa4359c04bfca37e258cddec03e387c350b5",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/33/Everything/x86_64/Packages/z/zlib-1.2.11-23.fc33.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/33/Everything/x86_64/Packages/z/zlib-1.2.11-23.fc33.x86_64.rpm",
        "https://fedora.ip-connect.info/linux/updates/33/Everything/x86_64/Packages/z/zlib-1.2.11-23.fc33.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/33/Everything/x86_64/Packages/z/zlib-1.2.11-23.fc33.x86_64.rpm",
    ],
)
