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
    sha256 = "934c9ceb552e84577b0faf1e5a2f0450314985b4d8712b2b70717dc679fdc01b",
    urls = [
        "https://github.com/bazelbuild/rules_python/releases/download/0.3.0/rules_python-0.3.0.tar.gz",
        "https://storage.googleapis.com/builddeps/934c9ceb552e84577b0faf1e5a2f0450314985b4d8712b2b70717dc679fdc01b",
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
    sha256 = "6a2af09c6a598a3c4e4fec9af78334fbec2b3c16473f4e2c692fe2e567dc6f56",
    strip_prefix = "bazeldnf-0.5.1",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.5.1.tar.gz",
        "https://storage.googleapis.com/builddeps/6a2af09c6a598a3c4e4fec9af78334fbec2b3c16473f4e2c692fe2e567dc6f56",
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
    digest = "sha256:14193941e1fe74f2189536263c71479abbd296dc93b75e8a7f97f0b31e78b71e",
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

# TODO build fedora_realtime for multi-arch
container_pull(
    name = "fedora_realtime",
    digest = "sha256:437f4e02986daf0058239f4a282d32304dcac629d5d1b4c75a74025f1ce22811",
    registry = "quay.io",
    repository = "kubevirt/fedora-realtime-container-disk",
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
    sha256 = "51d38a062d1b91bd7cb3dd8e68354aae86f6a889b4bb68a358b3ab55030dc0c9",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/appliance-1.44.0-linux-4.18.0-338-centos8.tar.xz",
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
    name = "acl-0__2.2.53-1.el8.aarch64",
    sha256 = "47c2cc5872174c548de1096dc5673ee91349209d89e0193a4793955d6865b3b1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/acl-2.2.53-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/47c2cc5872174c548de1096dc5673ee91349209d89e0193a4793955d6865b3b1",
    ],
)

rpm(
    name = "acl-0__2.2.53-1.el8.x86_64",
    sha256 = "227de6071cd3aeca7e10ad386beaf38737d081e06350d02208a3f6a2c9710385",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/acl-2.2.53-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/227de6071cd3aeca7e10ad386beaf38737d081e06350d02208a3f6a2c9710385",
    ],
)

rpm(
    name = "attr-0__2.4.48-3.el8.x86_64",
    sha256 = "da1464c73554bd77756428d592f0cb9a8f65604c22c3b3b2b7db14b35f5ad178",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/attr-2.4.48-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da1464c73554bd77756428d592f0cb9a8f65604c22c3b3b2b7db14b35f5ad178",
    ],
)

rpm(
    name = "audit-libs-0__3.0-0.17.20191104git1c2f876.el8.aarch64",
    sha256 = "11811c556a3bdc9c572c0ab67d3106bd1de3406c9d471de03e028f041b5785c3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/audit-libs-3.0-0.17.20191104git1c2f876.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/11811c556a3bdc9c572c0ab67d3106bd1de3406c9d471de03e028f041b5785c3",
    ],
)

rpm(
    name = "audit-libs-0__3.0-0.17.20191104git1c2f876.el8.x86_64",
    sha256 = "e7da6b155db78fb2015c40663fec6e475a44b21b1c2124496cf23f862e021db8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/audit-libs-3.0-0.17.20191104git1c2f876.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e7da6b155db78fb2015c40663fec6e475a44b21b1c2124496cf23f862e021db8",
    ],
)

rpm(
    name = "augeas-libs-0__1.12.0-6.el8.x86_64",
    sha256 = "60e90e9c353066b0d08136aacfd6731a0eef918ca3ab59d7ee117e5c0b7ba723",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/augeas-libs-1.12.0-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/60e90e9c353066b0d08136aacfd6731a0eef918ca3ab59d7ee117e5c0b7ba723",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.12-8.el8.aarch64",
    sha256 = "a69b87111415322e6586ba6b35494d77af7d9d58b2d9dfaf0360e4f827622dd2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/autogen-libopts-5.18.12-8.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a69b87111415322e6586ba6b35494d77af7d9d58b2d9dfaf0360e4f827622dd2",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.12-8.el8.x86_64",
    sha256 = "c73af033015bfbdbe8a43e162b098364d148517d394910f8db5d33b76b93aa48",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/autogen-libopts-5.18.12-8.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c73af033015bfbdbe8a43e162b098364d148517d394910f8db5d33b76b93aa48",
    ],
)

rpm(
    name = "basesystem-0__11-5.el8.aarch64",
    sha256 = "48226934763e4c412c1eb65df314e6879720b4b1ebcb3d07c126c9526639cb68",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/basesystem-11-5.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48226934763e4c412c1eb65df314e6879720b4b1ebcb3d07c126c9526639cb68",
    ],
)

rpm(
    name = "basesystem-0__11-5.el8.x86_64",
    sha256 = "48226934763e4c412c1eb65df314e6879720b4b1ebcb3d07c126c9526639cb68",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/basesystem-11-5.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48226934763e4c412c1eb65df314e6879720b4b1ebcb3d07c126c9526639cb68",
    ],
)

rpm(
    name = "bash-0__4.4.20-3.el8.aarch64",
    sha256 = "e5cbf67dbddd24bd6f40e980a9185827c6480a30cea408733dc0b22241fd5d96",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bash-4.4.20-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e5cbf67dbddd24bd6f40e980a9185827c6480a30cea408733dc0b22241fd5d96",
    ],
)

rpm(
    name = "bash-0__4.4.20-3.el8.x86_64",
    sha256 = "f5da563e3446ecf16a12813b885b57243cb6181a5def815bf4cafaa35a0eefc5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bash-4.4.20-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f5da563e3446ecf16a12813b885b57243cb6181a5def815bf4cafaa35a0eefc5",
    ],
)

rpm(
    name = "bind-export-libs-32__9.11.26-6.el8.x86_64",
    sha256 = "fa086b16f9c9cb29b088547630b79e930bdef641088a1987d3c1c920a7e65585",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bind-export-libs-9.11.26-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fa086b16f9c9cb29b088547630b79e930bdef641088a1987d3c1c920a7e65585",
    ],
)

rpm(
    name = "binutils-0__2.30-110.el8.aarch64",
    sha256 = "81ac620ec14922a509a2db9cbc4225ced69539159b51f4dbf80f477224b3e3f5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/binutils-2.30-110.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/81ac620ec14922a509a2db9cbc4225ced69539159b51f4dbf80f477224b3e3f5",
    ],
)

rpm(
    name = "binutils-0__2.30-110.el8.x86_64",
    sha256 = "b1c05c24fbcc72aa4375bf952b8ad4e9c1b3f9163963d9eb696f61637985e44d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/binutils-2.30-110.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b1c05c24fbcc72aa4375bf952b8ad4e9c1b3f9163963d9eb696f61637985e44d",
    ],
)

rpm(
    name = "bzip2-0__1.0.6-26.el8.aarch64",
    sha256 = "b18d9f23161d7d5de93fa72a56c645762deefbc0f3e5a095bb8d9e3cf09521e6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.6-26.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b18d9f23161d7d5de93fa72a56c645762deefbc0f3e5a095bb8d9e3cf09521e6",
    ],
)

rpm(
    name = "bzip2-0__1.0.6-26.el8.x86_64",
    sha256 = "78596f457c3d737a97a4edfe9a03a01f593606379c281701ab7f7eba13ecaf18",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.6-26.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/78596f457c3d737a97a4edfe9a03a01f593606379c281701ab7f7eba13ecaf18",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.6-26.el8.aarch64",
    sha256 = "a4451cae0e8a3307228ed8ac7dc9bab7de77fcbf2004141daa7f986f5dc9b381",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.6-26.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a4451cae0e8a3307228ed8ac7dc9bab7de77fcbf2004141daa7f986f5dc9b381",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.6-26.el8.x86_64",
    sha256 = "19d66d152b745dbd49cea9d21c52aec0ec4d4321edef97a342acd3542404fa31",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.6-26.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19d66d152b745dbd49cea9d21c52aec0ec4d4321edef97a342acd3542404fa31",
    ],
)

rpm(
    name = "ca-certificates-0__2021.2.50-82.el8.aarch64",
    sha256 = "1fad1d1f8b56e6967863aeb60f5fa3615e6a35b0f6532d8a23066e6823b50860",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ca-certificates-2021.2.50-82.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1fad1d1f8b56e6967863aeb60f5fa3615e6a35b0f6532d8a23066e6823b50860",
    ],
)

rpm(
    name = "ca-certificates-0__2021.2.50-82.el8.x86_64",
    sha256 = "1fad1d1f8b56e6967863aeb60f5fa3615e6a35b0f6532d8a23066e6823b50860",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ca-certificates-2021.2.50-82.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1fad1d1f8b56e6967863aeb60f5fa3615e6a35b0f6532d8a23066e6823b50860",
    ],
)

rpm(
    name = "centos-gpg-keys-1__8-3.el8.aarch64",
    sha256 = "79cda0505d8dd88b8277c1af9c55021319a0e516df8d24c893d740eac1d74feb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-8-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79cda0505d8dd88b8277c1af9c55021319a0e516df8d24c893d740eac1d74feb",
    ],
)

rpm(
    name = "centos-gpg-keys-1__8-3.el8.x86_64",
    sha256 = "79cda0505d8dd88b8277c1af9c55021319a0e516df8d24c893d740eac1d74feb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-8-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79cda0505d8dd88b8277c1af9c55021319a0e516df8d24c893d740eac1d74feb",
    ],
)

rpm(
    name = "centos-stream-release-0__8.6-1.el8.aarch64",
    sha256 = "3b3b86cb51f62632995ace850fbed9efc65381d639f1e1c5ceeff7ccf2dd6151",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-stream-release-8.6-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3b3b86cb51f62632995ace850fbed9efc65381d639f1e1c5ceeff7ccf2dd6151",
    ],
)

rpm(
    name = "centos-stream-release-0__8.6-1.el8.x86_64",
    sha256 = "3b3b86cb51f62632995ace850fbed9efc65381d639f1e1c5ceeff7ccf2dd6151",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-release-8.6-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3b3b86cb51f62632995ace850fbed9efc65381d639f1e1c5ceeff7ccf2dd6151",
    ],
)

rpm(
    name = "centos-stream-repos-0__8-3.el8.aarch64",
    sha256 = "bd0c7fe3f1f6a08f4658cc0cc9b1c1a91e38f8bf60c3af2ed2ee220523ded269",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-8-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd0c7fe3f1f6a08f4658cc0cc9b1c1a91e38f8bf60c3af2ed2ee220523ded269",
    ],
)

rpm(
    name = "centos-stream-repos-0__8-3.el8.x86_64",
    sha256 = "bd0c7fe3f1f6a08f4658cc0cc9b1c1a91e38f8bf60c3af2ed2ee220523ded269",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-8-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd0c7fe3f1f6a08f4658cc0cc9b1c1a91e38f8bf60c3af2ed2ee220523ded269",
    ],
)

rpm(
    name = "chkconfig-0__1.19.1-1.el8.aarch64",
    sha256 = "be370bfc2f375cdbfc1079b19423142236770cf67caf74cdb12a7aef8a29c8c5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/chkconfig-1.19.1-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/be370bfc2f375cdbfc1079b19423142236770cf67caf74cdb12a7aef8a29c8c5",
    ],
)

rpm(
    name = "chkconfig-0__1.19.1-1.el8.x86_64",
    sha256 = "561b5fdadd60370b5d0a91b7ed35df95d7f60650cbade8c7e744323982ac82db",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/chkconfig-1.19.1-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/561b5fdadd60370b5d0a91b7ed35df95d7f60650cbade8c7e744323982ac82db",
    ],
)

rpm(
    name = "coreutils-single-0__8.30-12.el8.aarch64",
    sha256 = "2a72f27d58b3e9364a872fb089322a570477fc108ceeb5d304a2b831ab6f3e23",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.30-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2a72f27d58b3e9364a872fb089322a570477fc108ceeb5d304a2b831ab6f3e23",
    ],
)

rpm(
    name = "coreutils-single-0__8.30-12.el8.x86_64",
    sha256 = "2eb9c891de4f7281c7068351ffab36f93a8fa1e9d16b694c70968ed1a66a5f04",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.30-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2eb9c891de4f7281c7068351ffab36f93a8fa1e9d16b694c70968ed1a66a5f04",
    ],
)

rpm(
    name = "cpio-0__2.12-11.el8.x86_64",
    sha256 = "e16977e134123c69edc860829d45a5c751ad4befb5576a4a6812b31d6a1ba273",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cpio-2.12-11.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e16977e134123c69edc860829d45a5c751ad4befb5576a4a6812b31d6a1ba273",
    ],
)

rpm(
    name = "cpp-0__8.5.0-3.el8.aarch64",
    sha256 = "e5671f0fdb50f642a57ae51a608b0f6a1845559884618bc79ca76876f866fc69",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/cpp-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e5671f0fdb50f642a57ae51a608b0f6a1845559884618bc79ca76876f866fc69",
    ],
)

rpm(
    name = "cpp-0__8.5.0-3.el8.x86_64",
    sha256 = "8ded2f85aa7df71564892282353c18e7f1a57a9689e046faf17376d055d868a9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/cpp-8.5.0-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ded2f85aa7df71564892282353c18e7f1a57a9689e046faf17376d055d868a9",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-15.el8.aarch64",
    sha256 = "54efb853142572e1c2872e351838fc3657b662722ff6b2913d1872d4752a0eb8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cracklib-2.9.6-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/54efb853142572e1c2872e351838fc3657b662722ff6b2913d1872d4752a0eb8",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-15.el8.x86_64",
    sha256 = "dbbc9e20caabc30070354d91f61f383081f6d658e09d3c09e6df8764559e5aca",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cracklib-2.9.6-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dbbc9e20caabc30070354d91f61f383081f6d658e09d3c09e6df8764559e5aca",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-15.el8.aarch64",
    sha256 = "d61741af0ffe96c55f588dd164b9c3c93e7c7175c7e616db25990ab3e16e0f22",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cracklib-dicts-2.9.6-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d61741af0ffe96c55f588dd164b9c3c93e7c7175c7e616db25990ab3e16e0f22",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-15.el8.x86_64",
    sha256 = "f1ce23ee43c747a35367dada19ca200a7758c50955ccc44aa946b86b647077ca",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cracklib-dicts-2.9.6-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1ce23ee43c747a35367dada19ca200a7758c50955ccc44aa946b86b647077ca",
    ],
)

rpm(
    name = "crypto-policies-0__20210617-1.gitc776d3e.el8.aarch64",
    sha256 = "2a8f9e5119a034801904185dcbf1bc29db67e9e9b0cf5893615722d7bb33099c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/crypto-policies-20210617-1.gitc776d3e.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2a8f9e5119a034801904185dcbf1bc29db67e9e9b0cf5893615722d7bb33099c",
    ],
)

rpm(
    name = "crypto-policies-0__20210617-1.gitc776d3e.el8.x86_64",
    sha256 = "2a8f9e5119a034801904185dcbf1bc29db67e9e9b0cf5893615722d7bb33099c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/crypto-policies-20210617-1.gitc776d3e.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2a8f9e5119a034801904185dcbf1bc29db67e9e9b0cf5893615722d7bb33099c",
    ],
)

rpm(
    name = "cryptsetup-0__2.3.3-4.el8.x86_64",
    sha256 = "fab7d620fb953f64b8a01b93c835e4a8a59aa1e9459e58dc3211db493d6f2c35",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-2.3.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fab7d620fb953f64b8a01b93c835e4a8a59aa1e9459e58dc3211db493d6f2c35",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.3-4.el8.aarch64",
    sha256 = "c94d212f77d5d83ba1bd22a5c6b5e92590d5c4cb412950ec22d1309d79e2fc0e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cryptsetup-libs-2.3.3-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c94d212f77d5d83ba1bd22a5c6b5e92590d5c4cb412950ec22d1309d79e2fc0e",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.3-4.el8.x86_64",
    sha256 = "679d78e677c3be4a5ee747feee9bbc4ccf59d489321da44253048b7d76beba97",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-libs-2.3.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/679d78e677c3be4a5ee747feee9bbc4ccf59d489321da44253048b7d76beba97",
    ],
)

rpm(
    name = "curl-0__7.61.1-22.el8.aarch64",
    sha256 = "522b718e08eb3ef7c2b9af21e84c624f503b72b533a1bc5c8f70d7d302e87e93",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/curl-7.61.1-22.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/522b718e08eb3ef7c2b9af21e84c624f503b72b533a1bc5c8f70d7d302e87e93",
    ],
)

rpm(
    name = "curl-0__7.61.1-22.el8.x86_64",
    sha256 = "3dd394e5b9403846d3068978bbe63f76b30a1d6801a7b3a93bbb3a9e64881e53",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/curl-7.61.1-22.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3dd394e5b9403846d3068978bbe63f76b30a1d6801a7b3a93bbb3a9e64881e53",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-5.el8.aarch64",
    sha256 = "7dcb85af91070dca195ad82b91476d6cbbb4fef192e2f5c0a318d228ffedfbac",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-2.1.27-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7dcb85af91070dca195ad82b91476d6cbbb4fef192e2f5c0a318d228ffedfbac",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-5.el8.x86_64",
    sha256 = "41cf36b5d082794509fece3681e8b7a0000574efef834c611820d00ecf6f2d78",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-2.1.27-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/41cf36b5d082794509fece3681e8b7a0000574efef834c611820d00ecf6f2d78",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-5.el8.aarch64",
    sha256 = "f02f26dc5be5410aa233a0b50821df2c63f81772aef7f31c4be557319f1080e1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-gssapi-2.1.27-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f02f26dc5be5410aa233a0b50821df2c63f81772aef7f31c4be557319f1080e1",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-5.el8.x86_64",
    sha256 = "a7af455a12a4df52523efc4be6f4da1065d1e83c73209844ba331c00d1d409a3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-gssapi-2.1.27-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a7af455a12a4df52523efc4be6f4da1065d1e83c73209844ba331c00d1d409a3",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-5.el8.aarch64",
    sha256 = "36d4e208921238b99c822a5f1686120c0c227fc02dc6e3258c2c71d62492a1e7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-lib-2.1.27-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/36d4e208921238b99c822a5f1686120c0c227fc02dc6e3258c2c71d62492a1e7",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-5.el8.x86_64",
    sha256 = "c421b9c029abac796ade606f96d638e06a6d4ce5c2d499abd05812c306d25143",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-lib-2.1.27-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c421b9c029abac796ade606f96d638e06a6d4ce5c2d499abd05812c306d25143",
    ],
)

rpm(
    name = "daxctl-libs-0__71.1-2.el8.x86_64",
    sha256 = "c879ce8eea2780a4cf4ffe67661fe56aaf2e6c110b45a1ead147925065a2eeb2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/daxctl-libs-71.1-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c879ce8eea2780a4cf4ffe67661fe56aaf2e6c110b45a1ead147925065a2eeb2",
    ],
)

rpm(
    name = "dbus-1__1.12.8-14.el8.aarch64",
    sha256 = "107a781be497f1a51ffd370aba59dbc4de3d7f89802830c66051dc51a5ec185b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-1.12.8-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/107a781be497f1a51ffd370aba59dbc4de3d7f89802830c66051dc51a5ec185b",
    ],
)

rpm(
    name = "dbus-1__1.12.8-14.el8.x86_64",
    sha256 = "a61f7b7bccd0168f654f54e7a1acfb597bf018bbda267140d2049e58563c6f12",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-1.12.8-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a61f7b7bccd0168f654f54e7a1acfb597bf018bbda267140d2049e58563c6f12",
    ],
)

rpm(
    name = "dbus-common-1__1.12.8-14.el8.aarch64",
    sha256 = "7baac88adafdc5958fb818c7685d3c6548f6e2e585e4435ceee4a168edc3597e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.8-14.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7baac88adafdc5958fb818c7685d3c6548f6e2e585e4435ceee4a168edc3597e",
    ],
)

rpm(
    name = "dbus-common-1__1.12.8-14.el8.x86_64",
    sha256 = "7baac88adafdc5958fb818c7685d3c6548f6e2e585e4435ceee4a168edc3597e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.8-14.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7baac88adafdc5958fb818c7685d3c6548f6e2e585e4435ceee4a168edc3597e",
    ],
)

rpm(
    name = "dbus-daemon-1__1.12.8-14.el8.aarch64",
    sha256 = "69e6fa2fa4a60384e21913b69cf4ddd6a21148e3d984a4ff0cbe651a2986f738",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-daemon-1.12.8-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/69e6fa2fa4a60384e21913b69cf4ddd6a21148e3d984a4ff0cbe651a2986f738",
    ],
)

rpm(
    name = "dbus-daemon-1__1.12.8-14.el8.x86_64",
    sha256 = "c15824e278323ba2ef0e3fab5c2c39d04137485dfa0298e43d19a6d2ca667f6c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-daemon-1.12.8-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c15824e278323ba2ef0e3fab5c2c39d04137485dfa0298e43d19a6d2ca667f6c",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.8-14.el8.aarch64",
    sha256 = "9738cb7597fa6dd4e3bee9159e813e6188894f98852fb896b95437f7fc8dbd8d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.12.8-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9738cb7597fa6dd4e3bee9159e813e6188894f98852fb896b95437f7fc8dbd8d",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.8-14.el8.x86_64",
    sha256 = "7533e19781d1b7e354315b15ef3d3011a8f1eec8980f9e8a6c633af3a806db2a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.8-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7533e19781d1b7e354315b15ef3d3011a8f1eec8980f9e8a6c633af3a806db2a",
    ],
)

rpm(
    name = "dbus-tools-1__1.12.8-14.el8.aarch64",
    sha256 = "da2dd7c4192fbafc3dfda1769b03fa27ec1855dd54963e774eb404f44a85b8e7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-tools-1.12.8-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da2dd7c4192fbafc3dfda1769b03fa27ec1855dd54963e774eb404f44a85b8e7",
    ],
)

rpm(
    name = "dbus-tools-1__1.12.8-14.el8.x86_64",
    sha256 = "6032a05a8c33bc9d6be816d4172e8bb24a18ec873c127f5bee94da9210130e8b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-tools-1.12.8-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6032a05a8c33bc9d6be816d4172e8bb24a18ec873c127f5bee94da9210130e8b",
    ],
)

rpm(
    name = "device-mapper-8__1.02.181-1.el8.aarch64",
    sha256 = "75017861aeef44550dad70302fa9bca2814515d10283bda29501698fd6c93490",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.181-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/75017861aeef44550dad70302fa9bca2814515d10283bda29501698fd6c93490",
    ],
)

rpm(
    name = "device-mapper-8__1.02.181-1.el8.x86_64",
    sha256 = "f07a8c527c540ab3984a2b8c537188b11b11ee19f5362b48dcdc2d976cc14ffd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.181-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f07a8c527c540ab3984a2b8c537188b11b11ee19f5362b48dcdc2d976cc14ffd",
    ],
)

rpm(
    name = "device-mapper-event-8__1.02.181-1.el8.x86_64",
    sha256 = "08345e8633a8cc8d653ae5bf9de4dad3fcc2126e7b0337714f4f7b2cb7ff9d24",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-1.02.181-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/08345e8633a8cc8d653ae5bf9de4dad3fcc2126e7b0337714f4f7b2cb7ff9d24",
    ],
)

rpm(
    name = "device-mapper-event-libs-8__1.02.181-1.el8.x86_64",
    sha256 = "4daaeff710f6e3a8379d24fbe70e816bf3925a7f797491e59926cafa7eae3ca0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-libs-1.02.181-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4daaeff710f6e3a8379d24fbe70e816bf3925a7f797491e59926cafa7eae3ca0",
    ],
)

rpm(
    name = "device-mapper-libs-8__1.02.181-1.el8.aarch64",
    sha256 = "9c07c99ed19ea5864d11d7e656f60b7a16bce9894e283f048011cfbfc955a15e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.181-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9c07c99ed19ea5864d11d7e656f60b7a16bce9894e283f048011cfbfc955a15e",
    ],
)

rpm(
    name = "device-mapper-libs-8__1.02.181-1.el8.x86_64",
    sha256 = "d5a65da7489ef48ee365a81a197b99b7a46c242fdbc4cc42b12976bf73aca564",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.181-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5a65da7489ef48ee365a81a197b99b7a46c242fdbc4cc42b12976bf73aca564",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-19.el8.aarch64",
    sha256 = "3fdab7e2480a8bef23801206e5da46e3d09a670f903f2dde5a34780873a90869",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.4-19.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3fdab7e2480a8bef23801206e5da46e3d09a670f903f2dde5a34780873a90869",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-19.el8.x86_64",
    sha256 = "70604e18521fadae6334374851e1ed35952b754ff4f14f7f15916f7ee0ddbfcd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.4-19.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/70604e18521fadae6334374851e1ed35952b754ff4f14f7f15916f7ee0ddbfcd",
    ],
)

rpm(
    name = "device-mapper-persistent-data-0__0.9.0-5.el8.x86_64",
    sha256 = "ab9b4f25ba6d9fb2cedaaeea5465cae6fe625612ccc55daaf8551964b6433534",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-persistent-data-0.9.0-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ab9b4f25ba6d9fb2cedaaeea5465cae6fe625612ccc55daaf8551964b6433534",
    ],
)

rpm(
    name = "dhcp-client-12__4.3.6-45.el8.x86_64",
    sha256 = "17b8055a23085caee5db3a3b35d5571f228865bdfb06d0bdfbe4cdba52e82350",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-client-4.3.6-45.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/17b8055a23085caee5db3a3b35d5571f228865bdfb06d0bdfbe4cdba52e82350",
    ],
)

rpm(
    name = "dhcp-common-12__4.3.6-45.el8.x86_64",
    sha256 = "d64da85b0e013679fb06eeadad74465d9556d7ad21e97beb1b5f61fb0658cd10",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-common-4.3.6-45.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d64da85b0e013679fb06eeadad74465d9556d7ad21e97beb1b5f61fb0658cd10",
    ],
)

rpm(
    name = "dhcp-libs-12__4.3.6-45.el8.x86_64",
    sha256 = "7e55750fb56250d3f8ad91edbf4b865a09b7baf7797dc4eaa668b473043efeca",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-libs-4.3.6-45.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e55750fb56250d3f8ad91edbf4b865a09b7baf7797dc4eaa668b473043efeca",
    ],
)

rpm(
    name = "diffutils-0__3.6-6.el8.aarch64",
    sha256 = "8cbebc0fa970ceca4f479ee292eaad155084987be2cf7f97bbafe4a529319c98",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/diffutils-3.6-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8cbebc0fa970ceca4f479ee292eaad155084987be2cf7f97bbafe4a529319c98",
    ],
)

rpm(
    name = "diffutils-0__3.6-6.el8.x86_64",
    sha256 = "c515d78c64a93d8b469593bff5800eccd50f24b16697ab13bdce81238c38eb77",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/diffutils-3.6-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c515d78c64a93d8b469593bff5800eccd50f24b16697ab13bdce81238c38eb77",
    ],
)

rpm(
    name = "dmidecode-1__3.2-10.el8.x86_64",
    sha256 = "cd2f140bd1718b4403ad66568155264b69231748a3c813e2feaac1b704da62c6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dmidecode-3.2-10.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cd2f140bd1718b4403ad66568155264b69231748a3c813e2feaac1b704da62c6",
    ],
)

rpm(
    name = "dnf-0__4.7.0-4.el8.x86_64",
    sha256 = "7233eddc7da5e6adbe5c2199cc51de7688cdfa44d46e99721bba6f2f1d08f0ca",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dnf-4.7.0-4.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7233eddc7da5e6adbe5c2199cc51de7688cdfa44d46e99721bba6f2f1d08f0ca",
    ],
)

rpm(
    name = "dnf-plugins-core-0__4.0.21-3.el8.x86_64",
    sha256 = "a0b33e9e8224401aad9ba415c20587dd91326f851da2cde469c029803f756a28",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dnf-plugins-core-4.0.21-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a0b33e9e8224401aad9ba415c20587dd91326f851da2cde469c029803f756a28",
    ],
)

rpm(
    name = "dosfstools-0__4.1-6.el8.x86_64",
    sha256 = "40676b73567e195228ba2a8bb53692f88f88d43612564613fb168383eee57f6a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dosfstools-4.1-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/40676b73567e195228ba2a8bb53692f88f88d43612564613fb168383eee57f6a",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.6-2.el8.aarch64",
    sha256 = "ac016de4d762f554820fcc7081025a9cc9a9aaec171fcf377c18f9d3b1365e2d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.45.6-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ac016de4d762f554820fcc7081025a9cc9a9aaec171fcf377c18f9d3b1365e2d",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.6-2.el8.x86_64",
    sha256 = "f46dee25409f262173a127102dd9c7a3aced4feecf935a2340a043c17b1f8c61",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.45.6-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f46dee25409f262173a127102dd9c7a3aced4feecf935a2340a043c17b1f8c61",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-2.el8.aarch64",
    sha256 = "e20ee66f3b3bc94aa689ad1e220c7ae787a689ec4a10916c14fb744ceb5e06a4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.45.6-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e20ee66f3b3bc94aa689ad1e220c7ae787a689ec4a10916c14fb744ceb5e06a4",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-2.el8.x86_64",
    sha256 = "037d854bec991cd4f0827ff5903b33847ef6965f4aba44252f30f412d49afdac",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.45.6-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/037d854bec991cd4f0827ff5903b33847ef6965f4aba44252f30f412d49afdac",
    ],
)

rpm(
    name = "edk2-aarch64-0__20200602gitca407c7246bf-4.el8.aarch64",
    sha256 = "1a2f9802fefaa0e6fab41557b0c2f02968d443a71d0bd83eae98882b0ce7df6d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20200602gitca407c7246bf-4.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1a2f9802fefaa0e6fab41557b0c2f02968d443a71d0bd83eae98882b0ce7df6d",
    ],
)

rpm(
    name = "edk2-ovmf-0__20200602gitca407c7246bf-4.el8.x86_64",
    sha256 = "3fe8746248ada6d0421d3108aa1db0c602a65b09aa5ddf9201ce8305529a129b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20200602gitca407c7246bf-4.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3fe8746248ada6d0421d3108aa1db0c602a65b09aa5ddf9201ce8305529a129b",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.185-1.el8.aarch64",
    sha256 = "30ceeb5a6cadaeccdbde088bfb52ba88190fa530c11f4a2aafd62b4b4ad6b404",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.185-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/30ceeb5a6cadaeccdbde088bfb52ba88190fa530c11f4a2aafd62b4b4ad6b404",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.185-1.el8.x86_64",
    sha256 = "30ceeb5a6cadaeccdbde088bfb52ba88190fa530c11f4a2aafd62b4b4ad6b404",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.185-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/30ceeb5a6cadaeccdbde088bfb52ba88190fa530c11f4a2aafd62b4b4ad6b404",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.185-1.el8.aarch64",
    sha256 = "25788279ab5869acfcaf46186ef08dc6908d6a90f6f4ff6ba9474a1fde3870fd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.185-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/25788279ab5869acfcaf46186ef08dc6908d6a90f6f4ff6ba9474a1fde3870fd",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.185-1.el8.x86_64",
    sha256 = "b56349ce3abac926fad2ef8366080e0823c4719235e72cb47306f4e9a39a0d66",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.185-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b56349ce3abac926fad2ef8366080e0823c4719235e72cb47306f4e9a39a0d66",
    ],
)

rpm(
    name = "elfutils-libs-0__0.185-1.el8.aarch64",
    sha256 = "cb7464d6e1440b4218eb668edaa67b6a43ecd647d8915a6e96d5f955ad69f09c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.185-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb7464d6e1440b4218eb668edaa67b6a43ecd647d8915a6e96d5f955ad69f09c",
    ],
)

rpm(
    name = "elfutils-libs-0__0.185-1.el8.x86_64",
    sha256 = "abfb7d93009c64a38d1e938093eb109ad344b150272ac644dc8ea6a3bd64adef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.185-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/abfb7d93009c64a38d1e938093eb109ad344b150272ac644dc8ea6a3bd64adef",
    ],
)

rpm(
    name = "ethtool-2__5.8-7.el8.aarch64",
    sha256 = "6bf45ab001060360948e03de9f3d6f676f6dd5b6ed11b5d4a3dc0080907ad6a1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ethtool-5.8-7.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6bf45ab001060360948e03de9f3d6f676f6dd5b6ed11b5d4a3dc0080907ad6a1",
    ],
)

rpm(
    name = "ethtool-2__5.8-7.el8.x86_64",
    sha256 = "c11b2edae722a386ea97575910ad57f233b4e6c2ac1fdefa44990f59eb0f7fa8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ethtool-5.8-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c11b2edae722a386ea97575910ad57f233b4e6c2ac1fdefa44990f59eb0f7fa8",
    ],
)

rpm(
    name = "expat-0__2.2.5-4.el8.aarch64",
    sha256 = "16356a5f29d0b191e84e37c92f9b6a3cd2ef683c84dd37c065f3461ad5abef03",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/expat-2.2.5-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/16356a5f29d0b191e84e37c92f9b6a3cd2ef683c84dd37c065f3461ad5abef03",
    ],
)

rpm(
    name = "expat-0__2.2.5-4.el8.x86_64",
    sha256 = "0c451ef9a9cd603a35aaab1a6c4aba83103332bed7c2b7393c48631f9bb50158",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/expat-2.2.5-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c451ef9a9cd603a35aaab1a6c4aba83103332bed7c2b7393c48631f9bb50158",
    ],
)

rpm(
    name = "file-0__5.33-20.el8.x86_64",
    sha256 = "9729d5fd2ecbf6902329585a4acdd09f2f591673802ca89dd575ba8351991814",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/file-5.33-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9729d5fd2ecbf6902329585a4acdd09f2f591673802ca89dd575ba8351991814",
    ],
)

rpm(
    name = "file-libs-0__5.33-20.el8.x86_64",
    sha256 = "216250c4239243c7692981146d9a0eb08434c9f8d4b1321ef31302e9dbf08384",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/file-libs-5.33-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/216250c4239243c7692981146d9a0eb08434c9f8d4b1321ef31302e9dbf08384",
    ],
)

rpm(
    name = "filesystem-0__3.8-6.el8.aarch64",
    sha256 = "e6c3fa94860eda0bc2ae6b1b78acd1159cbed355a03e7bec8b3defa1d90782b6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/filesystem-3.8-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e6c3fa94860eda0bc2ae6b1b78acd1159cbed355a03e7bec8b3defa1d90782b6",
    ],
)

rpm(
    name = "filesystem-0__3.8-6.el8.x86_64",
    sha256 = "50bdb81d578914e0e88fe6b13550b4c30aac4d72f064fdcd78523df7dd2f64da",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/filesystem-3.8-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/50bdb81d578914e0e88fe6b13550b4c30aac4d72f064fdcd78523df7dd2f64da",
    ],
)

rpm(
    name = "findutils-1__4.6.0-20.el8.aarch64",
    sha256 = "985479064966d05aa82010ed5b8905942e47e2bebb919c9c1bd004a28addad1d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/findutils-4.6.0-20.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/985479064966d05aa82010ed5b8905942e47e2bebb919c9c1bd004a28addad1d",
    ],
)

rpm(
    name = "findutils-1__4.6.0-20.el8.x86_64",
    sha256 = "811eb112646b7d87773c65af47efdca975468f3e5df44aa9944e30de24d83890",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/findutils-4.6.0-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/811eb112646b7d87773c65af47efdca975468f3e5df44aa9944e30de24d83890",
    ],
)

rpm(
    name = "fuse-0__2.9.7-12.el8.x86_64",
    sha256 = "2465c0c3b3d9519a3f9ae2ffe3e2c0bc61dca6fcb6ae710a6c7951007f498864",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-2.9.7-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2465c0c3b3d9519a3f9ae2ffe3e2c0bc61dca6fcb6ae710a6c7951007f498864",
    ],
)

rpm(
    name = "fuse-common-0__3.2.1-12.el8.x86_64",
    sha256 = "3f947e1e56d0b0210f9ccbc4483f8b6bfb100cfd79ea1efac3336a8d624ec0d6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-common-3.2.1-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f947e1e56d0b0210f9ccbc4483f8b6bfb100cfd79ea1efac3336a8d624ec0d6",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.7-12.el8.aarch64",
    sha256 = "0431ac0a9ad2ae9d657a66e9a5dc9326b232732e9967088990c09e826c6f3071",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/fuse-libs-2.9.7-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0431ac0a9ad2ae9d657a66e9a5dc9326b232732e9967088990c09e826c6f3071",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.7-12.el8.x86_64",
    sha256 = "6c6c98e2ddc2210ca377b0ef0c6bb694abd23f33413acadaedc1760da5bcc079",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.7-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6c6c98e2ddc2210ca377b0ef0c6bb694abd23f33413acadaedc1760da5bcc079",
    ],
)

rpm(
    name = "gawk-0__4.2.1-2.el8.aarch64",
    sha256 = "1597024288d637f0865ca9be73fb1f2e5c495005fa9ca5b3aacc6d8ab8f444a8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gawk-4.2.1-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1597024288d637f0865ca9be73fb1f2e5c495005fa9ca5b3aacc6d8ab8f444a8",
    ],
)

rpm(
    name = "gawk-0__4.2.1-2.el8.x86_64",
    sha256 = "bc0d36db80589a9797b8c343cd80f5ad5f42b9afc88f8a46666dc1d8f5317cfe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gawk-4.2.1-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc0d36db80589a9797b8c343cd80f5ad5f42b9afc88f8a46666dc1d8f5317cfe",
    ],
)

rpm(
    name = "gcc-0__8.5.0-3.el8.aarch64",
    sha256 = "bbb255f57b707bbe487a881fb757d0d83785a2f53ceb335997ca4ee1f76f7c80",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gcc-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bbb255f57b707bbe487a881fb757d0d83785a2f53ceb335997ca4ee1f76f7c80",
    ],
)

rpm(
    name = "gcc-0__8.5.0-3.el8.x86_64",
    sha256 = "a2ec761aed245daf2f446c91e1be61dfbea6f313e8a37cd6b25021c440dba7a9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gcc-8.5.0-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a2ec761aed245daf2f446c91e1be61dfbea6f313e8a37cd6b25021c440dba7a9",
    ],
)

rpm(
    name = "gdbm-1__1.18-1.el8.aarch64",
    sha256 = "b7d0b4b922429354ffe7ddac90c8cd448229571b8d8e4c342110edadfe809f99",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gdbm-1.18-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b7d0b4b922429354ffe7ddac90c8cd448229571b8d8e4c342110edadfe809f99",
    ],
)

rpm(
    name = "gdbm-1__1.18-1.el8.x86_64",
    sha256 = "76d81e433a5291df491d2e289de9b33d4e5b98dcf48fd0a003c2767415d3e0aa",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdbm-1.18-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/76d81e433a5291df491d2e289de9b33d4e5b98dcf48fd0a003c2767415d3e0aa",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18-1.el8.aarch64",
    sha256 = "a7d04ae40ad91ba0ea93e4971a35585638f6adf8dbe1ed4849f643b6b64a5871",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gdbm-libs-1.18-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a7d04ae40ad91ba0ea93e4971a35585638f6adf8dbe1ed4849f643b6b64a5871",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18-1.el8.x86_64",
    sha256 = "3a3cb5a11f8e844cd1bf7c0e7bb6c12cc63e743029df50916ce7e6a9f8a4e169",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.18-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a3cb5a11f8e844cd1bf7c0e7bb6c12cc63e743029df50916ce7e6a9f8a4e169",
    ],
)

rpm(
    name = "gdisk-0__1.0.3-8.el8.x86_64",
    sha256 = "d1a8d5f78d6878d9396a7a72778789c9d4d519b00813c99e2f579058727f5346",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdisk-1.0.3-8.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d1a8d5f78d6878d9396a7a72778789c9d4d519b00813c99e2f579058727f5346",
    ],
)

rpm(
    name = "genisoimage-0__1.1.11-39.el8.x86_64",
    sha256 = "f98e67e6ed49e1ff2f4c1d8dea7aa139aaff69020013e458d2f3d8bd9d2c91b2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/genisoimage-1.1.11-39.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f98e67e6ed49e1ff2f4c1d8dea7aa139aaff69020013e458d2f3d8bd9d2c91b2",
    ],
)

rpm(
    name = "gettext-0__0.19.8.1-17.el8.aarch64",
    sha256 = "5f0c37488d3017b052039ddb8d9189a38c252af97884264959334237109c7e7c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gettext-0.19.8.1-17.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5f0c37488d3017b052039ddb8d9189a38c252af97884264959334237109c7e7c",
    ],
)

rpm(
    name = "gettext-0__0.19.8.1-17.el8.x86_64",
    sha256 = "829c842bbd79dca18d37198414626894c44e5b8faf0cce0054ca0ba6623ae136",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gettext-0.19.8.1-17.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/829c842bbd79dca18d37198414626894c44e5b8faf0cce0054ca0ba6623ae136",
    ],
)

rpm(
    name = "gettext-libs-0__0.19.8.1-17.el8.aarch64",
    sha256 = "882f23e0250a2d4aea49abb4ec8e11a9a3869ccdd812c796b6f85341ff9d30a2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gettext-libs-0.19.8.1-17.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/882f23e0250a2d4aea49abb4ec8e11a9a3869ccdd812c796b6f85341ff9d30a2",
    ],
)

rpm(
    name = "gettext-libs-0__0.19.8.1-17.el8.x86_64",
    sha256 = "ade52756aaf236e77dadd6cf97716821141c2759129ca7808524ab79607bb4c4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gettext-libs-0.19.8.1-17.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ade52756aaf236e77dadd6cf97716821141c2759129ca7808524ab79607bb4c4",
    ],
)

rpm(
    name = "glib2-0__2.56.4-157.el8.aarch64",
    sha256 = "943c9769c9d098cca94335a4ce39e266b7db424c2c98f83c08851fe28e5023a2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glib2-2.56.4-157.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/943c9769c9d098cca94335a4ce39e266b7db424c2c98f83c08851fe28e5023a2",
    ],
)

rpm(
    name = "glib2-0__2.56.4-157.el8.x86_64",
    sha256 = "c219858ad479a1c03a958b7da3c14f78689ea9f3207c56ac689306f4068d74b2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glib2-2.56.4-157.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c219858ad479a1c03a958b7da3c14f78689ea9f3207c56ac689306f4068d74b2",
    ],
)

rpm(
    name = "glibc-0__2.28-167.el8.aarch64",
    sha256 = "773065a902634b232d854242a7695daac4c7278f175736bdeec38a443eee29af",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/773065a902634b232d854242a7695daac4c7278f175736bdeec38a443eee29af",
    ],
)

rpm(
    name = "glibc-0__2.28-167.el8.x86_64",
    sha256 = "033fb54df0a62404659f0fd2724e9acd4d4ceaec2b2e62de1f5e28a98a5ade3d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/033fb54df0a62404659f0fd2724e9acd4d4ceaec2b2e62de1f5e28a98a5ade3d",
    ],
)

rpm(
    name = "glibc-common-0__2.28-167.el8.aarch64",
    sha256 = "865557ace673806c7c676e0c8880d88c29adb466108ce752dc118b27f8f87a06",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-common-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/865557ace673806c7c676e0c8880d88c29adb466108ce752dc118b27f8f87a06",
    ],
)

rpm(
    name = "glibc-common-0__2.28-167.el8.x86_64",
    sha256 = "201a0811fd5dfa7218c8c5503241a7af6e9953ab27993bb98ce82fe0557c3092",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-common-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/201a0811fd5dfa7218c8c5503241a7af6e9953ab27993bb98ce82fe0557c3092",
    ],
)

rpm(
    name = "glibc-devel-0__2.28-167.el8.aarch64",
    sha256 = "37280a911cb95aeca4e210e16518e0144524f3196cf32a3d1fef9f431d6ec559",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-devel-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/37280a911cb95aeca4e210e16518e0144524f3196cf32a3d1fef9f431d6ec559",
    ],
)

rpm(
    name = "glibc-devel-0__2.28-167.el8.x86_64",
    sha256 = "867755c166575433ede069708f753f4a2ee86c9da0e97efc6743f88ac7137b95",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-devel-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/867755c166575433ede069708f753f4a2ee86c9da0e97efc6743f88ac7137b95",
    ],
)

rpm(
    name = "glibc-gconv-extra-0__2.28-167.el8.aarch64",
    sha256 = "b0796eae9a9d6694ffae8902e1752a1c9b40c0397007e63e34730f46670f9652",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-gconv-extra-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b0796eae9a9d6694ffae8902e1752a1c9b40c0397007e63e34730f46670f9652",
    ],
)

rpm(
    name = "glibc-gconv-extra-0__2.28-167.el8.x86_64",
    sha256 = "0861a710e202266324906226ba02755552a416af0941d0c34febecf31b802cb9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-gconv-extra-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0861a710e202266324906226ba02755552a416af0941d0c34febecf31b802cb9",
    ],
)

rpm(
    name = "glibc-headers-0__2.28-167.el8.aarch64",
    sha256 = "7f36a3e25ce54e7f0af81f7e6786eed9e53f3921e4e600faed3b1d73af2f80c1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-headers-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7f36a3e25ce54e7f0af81f7e6786eed9e53f3921e4e600faed3b1d73af2f80c1",
    ],
)

rpm(
    name = "glibc-headers-0__2.28-167.el8.x86_64",
    sha256 = "6e120e79cb371e8beda951e9785daea067ccbc1b927af8a76ef162be9f7b609e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-headers-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6e120e79cb371e8beda951e9785daea067ccbc1b927af8a76ef162be9f7b609e",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-167.el8.aarch64",
    sha256 = "9d67803e9b395a77afe7007d4d366d4317f274a3c46823eca9d165982552447d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9d67803e9b395a77afe7007d4d366d4317f274a3c46823eca9d165982552447d",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-167.el8.x86_64",
    sha256 = "95dc34a107d3292061c324fc034d2b71ac3934584d53d296a3a6e28b2a840f37",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95dc34a107d3292061c324fc034d2b71ac3934584d53d296a3a6e28b2a840f37",
    ],
)

rpm(
    name = "glibc-static-0__2.28-167.el8.aarch64",
    sha256 = "da13ed2554d5d4adafb4ca3af665f91f364e3a41cda4f8c32dcfb31ad4d9de30",
    urls = [
        "http://mirror.centos.org/centos/8-stream/PowerTools/aarch64/os/Packages/glibc-static-2.28-167.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da13ed2554d5d4adafb4ca3af665f91f364e3a41cda4f8c32dcfb31ad4d9de30",
    ],
)

rpm(
    name = "glibc-static-0__2.28-167.el8.x86_64",
    sha256 = "1be6bad93bc841d78a1e7d983c8bb1caba62b72a8f56b9b2361a7a3d2c78e172",
    urls = [
        "http://mirror.centos.org/centos/8-stream/PowerTools/x86_64/os/Packages/glibc-static-2.28-167.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1be6bad93bc841d78a1e7d983c8bb1caba62b72a8f56b9b2361a7a3d2c78e172",
    ],
)

rpm(
    name = "gmp-1__6.1.2-10.el8.aarch64",
    sha256 = "8d407f8ad961169fca2ee5e22e824cbc2d2b5fedca9701896cc492d4cb788603",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gmp-6.1.2-10.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8d407f8ad961169fca2ee5e22e824cbc2d2b5fedca9701896cc492d4cb788603",
    ],
)

rpm(
    name = "gmp-1__6.1.2-10.el8.x86_64",
    sha256 = "3b96e2c7d5cd4b49bfde8e52c8af6ff595c91438e50856e468f14a049d8511e2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gmp-6.1.2-10.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b96e2c7d5cd4b49bfde8e52c8af6ff595c91438e50856e468f14a049d8511e2",
    ],
)

rpm(
    name = "gnupg2-0__2.2.20-2.el8.x86_64",
    sha256 = "42842cc39272d095d01d076982d4e9aa4888c7b2a1c26ebed6fb6ef9a02680ba",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gnupg2-2.2.20-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/42842cc39272d095d01d076982d4e9aa4888c7b2a1c26ebed6fb6ef9a02680ba",
    ],
)

rpm(
    name = "gnutls-0__3.6.16-4.el8.aarch64",
    sha256 = "f97d55f7bdf6fe126e7a1446563af7ee4c1bb7ee3a2a9b12b6df1cdd344da47e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gnutls-3.6.16-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f97d55f7bdf6fe126e7a1446563af7ee4c1bb7ee3a2a9b12b6df1cdd344da47e",
    ],
)

rpm(
    name = "gnutls-0__3.6.16-4.el8.x86_64",
    sha256 = "51bae480875ce4f8dd76b0af177c88eb1bd33faa910dbd64e574ef8c7ada1d03",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gnutls-3.6.16-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/51bae480875ce4f8dd76b0af177c88eb1bd33faa910dbd64e574ef8c7ada1d03",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.16-4.el8.aarch64",
    sha256 = "df78e84002d6ba09e37901b2f85f462a160beda734e98876c8baba0c71caf638",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.6.16-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/df78e84002d6ba09e37901b2f85f462a160beda734e98876c8baba0c71caf638",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.16-4.el8.x86_64",
    sha256 = "122d2a8e70c4cb857803e8b3673ca8dc572ba21ce790064abc4c99cca0f94b3f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.6.16-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/122d2a8e70c4cb857803e8b3673ca8dc572ba21ce790064abc4c99cca0f94b3f",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.16-4.el8.aarch64",
    sha256 = "1421e7f87f559b398b9bd289ee10c79b38a0505613761b4499ad9747aafb7da6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.6.16-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1421e7f87f559b398b9bd289ee10c79b38a0505613761b4499ad9747aafb7da6",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.16-4.el8.x86_64",
    sha256 = "58bc517e7d159bffa96db5cb5fd132e7e1798b8685ebb35d22a62ab6db51ced7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.6.16-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/58bc517e7d159bffa96db5cb5fd132e7e1798b8685ebb35d22a62ab6db51ced7",
    ],
)

rpm(
    name = "grep-0__3.1-6.el8.aarch64",
    sha256 = "7ffd6e95b0554466e97346b2f41fb5279aedcb29ae07828f63d06a8dedd7cd51",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/grep-3.1-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7ffd6e95b0554466e97346b2f41fb5279aedcb29ae07828f63d06a8dedd7cd51",
    ],
)

rpm(
    name = "grep-0__3.1-6.el8.x86_64",
    sha256 = "3f8ffe48bb481a5db7cbe42bf73b839d872351811e5df41b2f6697c61a030487",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/grep-3.1-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f8ffe48bb481a5db7cbe42bf73b839d872351811e5df41b2f6697c61a030487",
    ],
)

rpm(
    name = "gzip-0__1.9-12.el8.aarch64",
    sha256 = "1fe57a2d38c0d449efd06fa3e498e49f1952829f612d657418a7496458c0cb7c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gzip-1.9-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1fe57a2d38c0d449efd06fa3e498e49f1952829f612d657418a7496458c0cb7c",
    ],
)

rpm(
    name = "gzip-0__1.9-12.el8.x86_64",
    sha256 = "6d995888083240517e8eb5e0c8d8c22e63ac46de3b4bcd3c61e14959558800dd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gzip-1.9-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6d995888083240517e8eb5e0c8d8c22e63ac46de3b4bcd3c61e14959558800dd",
    ],
)

rpm(
    name = "hexedit-0__1.2.13-12.el8.x86_64",
    sha256 = "4538e44d3ebff3f9323b59171767bca2b7f5244dd90141de101856ad4f4643f5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/hexedit-1.2.13-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4538e44d3ebff3f9323b59171767bca2b7f5244dd90141de101856ad4f4643f5",
    ],
)

rpm(
    name = "hivex-0__1.3.18-23.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "d24f86d286bd2294de8b3c2931c3f851495cd12f76a24705425635f55eaf1147",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/hivex-1.3.18-23.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d24f86d286bd2294de8b3c2931c3f851495cd12f76a24705425635f55eaf1147",
    ],
)

rpm(
    name = "info-0__6.5-6.el8.aarch64",
    sha256 = "187a1fbb7e2992dfa777c7ca5c2f7369ecb85e4be4a483e6c0c6036e02bacf95",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/info-6.5-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/187a1fbb7e2992dfa777c7ca5c2f7369ecb85e4be4a483e6c0c6036e02bacf95",
    ],
)

rpm(
    name = "info-0__6.5-6.el8.x86_64",
    sha256 = "611da4957e11f4621f53b5d7d491bcba09854de4fad8a5be34e762f4f36b1102",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/info-6.5-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/611da4957e11f4621f53b5d7d491bcba09854de4fad8a5be34e762f4f36b1102",
    ],
)

rpm(
    name = "ipcalc-0__0.2.4-4.el8.x86_64",
    sha256 = "dea18976861575d40ffca814dee08a225376c7828a5afc9e5d0a383edd3d8907",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ipcalc-0.2.4-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dea18976861575d40ffca814dee08a225376c7828a5afc9e5d0a383edd3d8907",
    ],
)

rpm(
    name = "iproute-0__5.12.0-4.el8.aarch64",
    sha256 = "e5a745d26d55a485e6da72acc186bee17085c4f43e3e945919a97da714ef996c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-5.12.0-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e5a745d26d55a485e6da72acc186bee17085c4f43e3e945919a97da714ef996c",
    ],
)

rpm(
    name = "iproute-0__5.12.0-4.el8.x86_64",
    sha256 = "c8316f61dc49f4bc976f1bf0283bd6be9387846637b5ef620606ac1e6415769d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-5.12.0-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c8316f61dc49f4bc976f1bf0283bd6be9387846637b5ef620606ac1e6415769d",
    ],
)

rpm(
    name = "iproute-tc-0__5.12.0-4.el8.aarch64",
    sha256 = "88d3c6908954ddd71cf2af26fe31579bd385b87c708380219a7ac8a7ae3bb680",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-tc-5.12.0-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/88d3c6908954ddd71cf2af26fe31579bd385b87c708380219a7ac8a7ae3bb680",
    ],
)

rpm(
    name = "iproute-tc-0__5.12.0-4.el8.x86_64",
    sha256 = "943f8e803f85d65938c4becf7dd5c09faf57c97ca33f3fe9d8e7975becffa597",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-tc-5.12.0-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/943f8e803f85d65938c4becf7dd5c09faf57c97ca33f3fe9d8e7975becffa597",
    ],
)

rpm(
    name = "iptables-0__1.8.4-21.el8.aarch64",
    sha256 = "93e2f05889c69a12213df8c0a22d98c4746c12a11529cc581edd0d57e3bbfba1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-1.8.4-21.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/93e2f05889c69a12213df8c0a22d98c4746c12a11529cc581edd0d57e3bbfba1",
    ],
)

rpm(
    name = "iptables-0__1.8.4-21.el8.x86_64",
    sha256 = "70a61c876185414ff734d77676d8ae3162b70f9fe0e8ae270cbd6fff96465e5f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-1.8.4-21.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/70a61c876185414ff734d77676d8ae3162b70f9fe0e8ae270cbd6fff96465e5f",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-21.el8.aarch64",
    sha256 = "4d14ddc87ee1f4eec6d6c93176cf56cccc9e06f5f434e8bc5bcab7b73e5d7f37",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.4-21.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4d14ddc87ee1f4eec6d6c93176cf56cccc9e06f5f434e8bc5bcab7b73e5d7f37",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-21.el8.x86_64",
    sha256 = "89aa7541f46cee75f15d530d60c8d996932dbdec1b26c1a570a68e0822dd544f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.4-21.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89aa7541f46cee75f15d530d60c8d996932dbdec1b26c1a570a68e0822dd544f",
    ],
)

rpm(
    name = "iputils-0__20180629-7.el8.aarch64",
    sha256 = "6a8bdae6d069605468b2a153403187aa157c6d7ec59dd7097f160ea2b10c4899",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iputils-20180629-7.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a8bdae6d069605468b2a153403187aa157c6d7ec59dd7097f160ea2b10c4899",
    ],
)

rpm(
    name = "iputils-0__20180629-7.el8.x86_64",
    sha256 = "3c3d251a417e4d325b7075221d631df7411e25e3fc000528e3f2bd39a6bcc3af",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iputils-20180629-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3c3d251a417e4d325b7075221d631df7411e25e3fc000528e3f2bd39a6bcc3af",
    ],
)

rpm(
    name = "ipxe-roms-qemu-0__20181214-8.git133f4c47.el8.x86_64",
    sha256 = "36d152d9372177f7418c609e71b3a3b3c683a505df85d1d1c43b1730955ff024",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/ipxe-roms-qemu-20181214-8.git133f4c47.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/36d152d9372177f7418c609e71b3a3b3c683a505df85d1d1c43b1730955ff024",
    ],
)

rpm(
    name = "isl-0__0.16.1-6.el8.aarch64",
    sha256 = "b9bd73b0edcd9573548853bd44f5a58919d9de77d8b1304a4176c7fad726b472",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/isl-0.16.1-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b9bd73b0edcd9573548853bd44f5a58919d9de77d8b1304a4176c7fad726b472",
    ],
)

rpm(
    name = "isl-0__0.16.1-6.el8.x86_64",
    sha256 = "0cbdbdf53c8c12f48493bdae47d2bda45425011e67801a5827d164d6e10759ae",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/isl-0.16.1-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0cbdbdf53c8c12f48493bdae47d2bda45425011e67801a5827d164d6e10759ae",
    ],
)

rpm(
    name = "jansson-0__2.11-3.el8.aarch64",
    sha256 = "b8bd21e036c68bb8fbb9f21e6b5f6998fc3558f55a4b902d5d85664d5929134a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/jansson-2.11-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b8bd21e036c68bb8fbb9f21e6b5f6998fc3558f55a4b902d5d85664d5929134a",
    ],
)

rpm(
    name = "jansson-0__2.11-3.el8.x86_64",
    sha256 = "a06e1d34df03aaf429d290d5c281356fefe0ad510c229189405b88b3c0f40374",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/jansson-2.11-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a06e1d34df03aaf429d290d5c281356fefe0ad510c229189405b88b3c0f40374",
    ],
)

rpm(
    name = "json-c-0__0.13.1-3.el8.aarch64",
    sha256 = "3bb6aa6c7aa0c3186c3dbce23661ec709c43c0e87a22a7e952148f515e2bfc82",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/json-c-0.13.1-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3bb6aa6c7aa0c3186c3dbce23661ec709c43c0e87a22a7e952148f515e2bfc82",
    ],
)

rpm(
    name = "json-c-0__0.13.1-3.el8.x86_64",
    sha256 = "5035057553b61cb389c67aa2c29d99c8e0c1677369dad179d683942ccee90b3f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/json-c-0.13.1-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5035057553b61cb389c67aa2c29d99c8e0c1677369dad179d683942ccee90b3f",
    ],
)

rpm(
    name = "json-glib-0__1.4.4-1.el8.aarch64",
    sha256 = "01e70480bb032d5e0b60c5e732d4302d3a0ce73d1502a1729280d2b36e7e1c1a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/json-glib-1.4.4-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01e70480bb032d5e0b60c5e732d4302d3a0ce73d1502a1729280d2b36e7e1c1a",
    ],
)

rpm(
    name = "json-glib-0__1.4.4-1.el8.x86_64",
    sha256 = "98a6386df94fc9595365c3ecbc630708420fa68d1774614a723dec4a55e84b9c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/json-glib-1.4.4-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/98a6386df94fc9595365c3ecbc630708420fa68d1774614a723dec4a55e84b9c",
    ],
)

rpm(
    name = "kernel-headers-0__4.18.0-348.2.1.el8_5.aarch64",
    sha256 = "d9143f73a6aba8d86ea92886530161db15dfb42547eb10be6e8abae66c7e80d7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kernel-headers-4.18.0-348.2.1.el8_5.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d9143f73a6aba8d86ea92886530161db15dfb42547eb10be6e8abae66c7e80d7",
    ],
)

rpm(
    name = "kernel-headers-0__4.18.0-348.2.1.el8_5.x86_64",
    sha256 = "929a567ed8f80e2fa701425338013c4453be02d3c7d3b9b6c6733b38c559711d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kernel-headers-4.18.0-348.2.1.el8_5.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/929a567ed8f80e2fa701425338013c4453be02d3c7d3b9b6c6733b38c559711d",
    ],
)

rpm(
    name = "keyutils-libs-0__1.5.10-9.el8.aarch64",
    sha256 = "c5af4350099a98929777412fb23e74c3bd2d7d8bbd09c2969a59d45937738aad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/keyutils-libs-1.5.10-9.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c5af4350099a98929777412fb23e74c3bd2d7d8bbd09c2969a59d45937738aad",
    ],
)

rpm(
    name = "keyutils-libs-0__1.5.10-9.el8.x86_64",
    sha256 = "423329269c719b96ada88a27325e1923e764a70672e0dc6817e22eff07a9af7b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/keyutils-libs-1.5.10-9.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/423329269c719b96ada88a27325e1923e764a70672e0dc6817e22eff07a9af7b",
    ],
)

rpm(
    name = "kmod-0__25-18.el8.aarch64",
    sha256 = "22cd4d2563a814440d0c766e0153ef230d460ccb141c497f1cbd4723968832bc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-25-18.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/22cd4d2563a814440d0c766e0153ef230d460ccb141c497f1cbd4723968832bc",
    ],
)

rpm(
    name = "kmod-0__25-18.el8.x86_64",
    sha256 = "d48173b5826ab4f09c3d06758266be6a9bfc992f58cdc1c1244982b71a75463c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-25-18.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d48173b5826ab4f09c3d06758266be6a9bfc992f58cdc1c1244982b71a75463c",
    ],
)

rpm(
    name = "kmod-libs-0__25-18.el8.aarch64",
    sha256 = "9fec275ea16aaea202613606599e262e9806ef791342a62366d7d6936bc2ec3c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-libs-25-18.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9fec275ea16aaea202613606599e262e9806ef791342a62366d7d6936bc2ec3c",
    ],
)

rpm(
    name = "kmod-libs-0__25-18.el8.x86_64",
    sha256 = "8caf89ee7b7546fc39ebe58bf7447c9cd47ca8c4b2c0d9228de4b3087e0cb64e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-libs-25-18.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8caf89ee7b7546fc39ebe58bf7447c9cd47ca8c4b2c0d9228de4b3087e0cb64e",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-14.el8.aarch64",
    sha256 = "965eef9e09df948fc4a7fc4628111cb4e8018dd1e3496e56970c2e1909349dc6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.18.2-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/965eef9e09df948fc4a7fc4628111cb4e8018dd1e3496e56970c2e1909349dc6",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-14.el8.x86_64",
    sha256 = "898e38dba327b96336006633042ff6e138fbcafca248192ad0b43257c1d16904",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.18.2-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/898e38dba327b96336006633042ff6e138fbcafca248192ad0b43257c1d16904",
    ],
)

rpm(
    name = "less-0__530-1.el8.x86_64",
    sha256 = "f94172554b8ceeab97b560d0b05c2e2df4b2e737471adce6eca82fd3209be254",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/less-530-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f94172554b8ceeab97b560d0b05c2e2df4b2e737471adce6eca82fd3209be254",
    ],
)

rpm(
    name = "libacl-0__2.2.53-1.el8.aarch64",
    sha256 = "c4cfed85e5a0db903ad134b4327b1714e5453fcf5c4348ec93ab344860a970ef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libacl-2.2.53-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c4cfed85e5a0db903ad134b4327b1714e5453fcf5c4348ec93ab344860a970ef",
    ],
)

rpm(
    name = "libacl-0__2.2.53-1.el8.x86_64",
    sha256 = "4973664648b7ed9278bf29074ec6a60a9f660aa97c23a283750483f64429d5bb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libacl-2.2.53-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4973664648b7ed9278bf29074ec6a60a9f660aa97c23a283750483f64429d5bb",
    ],
)

rpm(
    name = "libaio-0__0.3.112-1.el8.aarch64",
    sha256 = "3bcb1ade26c217ead2da81c92b7ef78026c4a78383d28b6e825a7b840cae97fa",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libaio-0.3.112-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3bcb1ade26c217ead2da81c92b7ef78026c4a78383d28b6e825a7b840cae97fa",
    ],
)

rpm(
    name = "libaio-0__0.3.112-1.el8.x86_64",
    sha256 = "2c63399bee449fb6e921671a9bbf3356fda73f890b578820f7d926202e98a479",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libaio-0.3.112-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c63399bee449fb6e921671a9bbf3356fda73f890b578820f7d926202e98a479",
    ],
)

rpm(
    name = "libarchive-0__3.3.3-1.el8.aarch64",
    sha256 = "e6ddc29b56fcbabe7bcd1ff1535a72c0d4477176a6321b13006d2aa65477ff9d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libarchive-3.3.3-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e6ddc29b56fcbabe7bcd1ff1535a72c0d4477176a6321b13006d2aa65477ff9d",
    ],
)

rpm(
    name = "libarchive-0__3.3.3-1.el8.x86_64",
    sha256 = "57e908e16c0b5e63d0d97902e80660aa26543e0a17a5b78a41528889ea9cefb5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libarchive-3.3.3-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/57e908e16c0b5e63d0d97902e80660aa26543e0a17a5b78a41528889ea9cefb5",
    ],
)

rpm(
    name = "libasan-0__8.5.0-3.el8.aarch64",
    sha256 = "de13de2b2229035f99848a0774edf3aedc249e44d959835a15acf0c4dded62c0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libasan-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de13de2b2229035f99848a0774edf3aedc249e44d959835a15acf0c4dded62c0",
    ],
)

rpm(
    name = "libassuan-0__2.5.1-3.el8.x86_64",
    sha256 = "b49e8c674e462e3f494e825c5fca64002008cbf7a47bf131aa98b7f41678a6eb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libassuan-2.5.1-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b49e8c674e462e3f494e825c5fca64002008cbf7a47bf131aa98b7f41678a6eb",
    ],
)

rpm(
    name = "libatomic-0__8.5.0-3.el8.aarch64",
    sha256 = "8b55c4642b8644fc8b601032820c2e6c9025e1d4fdec2dcdeff724982c3d1ee8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libatomic-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8b55c4642b8644fc8b601032820c2e6c9025e1d4fdec2dcdeff724982c3d1ee8",
    ],
)

rpm(
    name = "libattr-0__2.4.48-3.el8.aarch64",
    sha256 = "6a6db7eab6e53dccc54116d2ddf86b02db4cff332a58b868f7ba778a99666c58",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libattr-2.4.48-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a6db7eab6e53dccc54116d2ddf86b02db4cff332a58b868f7ba778a99666c58",
    ],
)

rpm(
    name = "libattr-0__2.4.48-3.el8.x86_64",
    sha256 = "a02e1344ccde1747501ceeeff37df4f18149fb79b435aa22add08cff6bab3a5a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libattr-2.4.48-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a02e1344ccde1747501ceeeff37df4f18149fb79b435aa22add08cff6bab3a5a",
    ],
)

rpm(
    name = "libblkid-0__2.32.1-28.el8.aarch64",
    sha256 = "4eb804f201b7ff9d79f5d5c82c898a8b6f0daf3e2a43e4032790868238ec9e6e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libblkid-2.32.1-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4eb804f201b7ff9d79f5d5c82c898a8b6f0daf3e2a43e4032790868238ec9e6e",
    ],
)

rpm(
    name = "libblkid-0__2.32.1-28.el8.x86_64",
    sha256 = "20bcec1c3a9ca196c26749d9bb67dc6a77039f46ec078915531ae3f5205ee693",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libblkid-2.32.1-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20bcec1c3a9ca196c26749d9bb67dc6a77039f46ec078915531ae3f5205ee693",
    ],
)

rpm(
    name = "libbpf-0__0.4.0-1.el8.aarch64",
    sha256 = "700453875d00ba325459253a8395d6628f6d3a76fbad46ed24fb978c9ce66439",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libbpf-0.4.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/700453875d00ba325459253a8395d6628f6d3a76fbad46ed24fb978c9ce66439",
    ],
)

rpm(
    name = "libbpf-0__0.4.0-1.el8.x86_64",
    sha256 = "d131c3e7309b262921d54a510a6ba141c6b4f71214f0f4b98b4ac4d843b4baa4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libbpf-0.4.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d131c3e7309b262921d54a510a6ba141c6b4f71214f0f4b98b4ac4d843b4baa4",
    ],
)

rpm(
    name = "libburn-0__1.4.8-3.el8.aarch64",
    sha256 = "5ae88291a28b2a86efb6cdc8ff67baaf73dad1428c858c8b0fa9e8df0f0f041c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libburn-1.4.8-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5ae88291a28b2a86efb6cdc8ff67baaf73dad1428c858c8b0fa9e8df0f0f041c",
    ],
)

rpm(
    name = "libburn-0__1.4.8-3.el8.x86_64",
    sha256 = "d4b0815ced6c1ec209b78fee4e2c1ee74efcd401d5462268b47d94a28ebfaf31",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libburn-1.4.8-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d4b0815ced6c1ec209b78fee4e2c1ee74efcd401d5462268b47d94a28ebfaf31",
    ],
)

rpm(
    name = "libcap-0__2.26-5.el8.aarch64",
    sha256 = "b83ba0e5356e0d972bcfcf4efa36602f2ff27c18d6facf2ad54aba855563131e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcap-2.26-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b83ba0e5356e0d972bcfcf4efa36602f2ff27c18d6facf2ad54aba855563131e",
    ],
)

rpm(
    name = "libcap-0__2.26-5.el8.x86_64",
    sha256 = "336b54ee3f1509d49250863cdfff11b498d43e6a0252273cb801d86ee1919d38",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcap-2.26-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/336b54ee3f1509d49250863cdfff11b498d43e6a0252273cb801d86ee1919d38",
    ],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.el8.aarch64",
    sha256 = "cbbbb1771fe9cfaa3284837e5e02cd2101190504ea0baa0278c9cfb2b169073c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcap-ng-0.7.11-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cbbbb1771fe9cfaa3284837e5e02cd2101190504ea0baa0278c9cfb2b169073c",
    ],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.el8.x86_64",
    sha256 = "15c3c696ec2e21f48e951f426d3c77b53b579605b8dd89843b35c9ab9b1d7e69",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcap-ng-0.7.11-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/15c3c696ec2e21f48e951f426d3c77b53b579605b8dd89843b35c9ab9b1d7e69",
    ],
)

rpm(
    name = "libcom_err-0__1.45.6-2.el8.aarch64",
    sha256 = "adcc252cfead341c4258526cc6064d32f4a5709d3667ef15d66716e636a28783",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcom_err-1.45.6-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/adcc252cfead341c4258526cc6064d32f4a5709d3667ef15d66716e636a28783",
    ],
)

rpm(
    name = "libcom_err-0__1.45.6-2.el8.x86_64",
    sha256 = "21ac150aca09ddc50c667bf369c4c4937630f959ebec5f19c62560576ca18fd3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcom_err-1.45.6-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21ac150aca09ddc50c667bf369c4c4937630f959ebec5f19c62560576ca18fd3",
    ],
)

rpm(
    name = "libconfig-0__1.5-9.el8.x86_64",
    sha256 = "a4a2c7c0e2f454abae61dddbf4286a0b3617a8159fd20659bddbcedd8eaaa80c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libconfig-1.5-9.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a4a2c7c0e2f454abae61dddbf4286a0b3617a8159fd20659bddbcedd8eaaa80c",
    ],
)

rpm(
    name = "libcroco-0__0.6.12-4.el8_2.1.aarch64",
    sha256 = "0022ec2580783f68e603e9d4751478c28f2b383c596b4e896469077748771bfe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcroco-0.6.12-4.el8_2.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0022ec2580783f68e603e9d4751478c28f2b383c596b4e896469077748771bfe",
    ],
)

rpm(
    name = "libcroco-0__0.6.12-4.el8_2.1.x86_64",
    sha256 = "87f2a4d80cf4f6a958f3662c6a382edefc32a5ad2c364a7f3c40337cf2b1e8ba",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcroco-0.6.12-4.el8_2.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/87f2a4d80cf4f6a958f3662c6a382edefc32a5ad2c364a7f3c40337cf2b1e8ba",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.61.1-22.el8.aarch64",
    sha256 = "175a4530f5139bd05a3ececdaeb24de882166ca541e29c1f4b9415aef787fc2f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.61.1-22.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/175a4530f5139bd05a3ececdaeb24de882166ca541e29c1f4b9415aef787fc2f",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.61.1-22.el8.x86_64",
    sha256 = "28b062f4d5d39535aa7fd20ffe2a5fbd25fa4c84782445c3d936ccc9db3dba19",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.61.1-22.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/28b062f4d5d39535aa7fd20ffe2a5fbd25fa4c84782445c3d936ccc9db3dba19",
    ],
)

rpm(
    name = "libdb-0__5.3.28-42.el8_4.aarch64",
    sha256 = "7ab75211c6fca91324039d3c2eb73903f2da73c17d6edaf8e997462ce4fbb46c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libdb-5.3.28-42.el8_4.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7ab75211c6fca91324039d3c2eb73903f2da73c17d6edaf8e997462ce4fbb46c",
    ],
)

rpm(
    name = "libdb-0__5.3.28-42.el8_4.x86_64",
    sha256 = "058f77432592f4337039cbb7a4e5f680020d8b85a477080c01d96a7728de6934",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libdb-5.3.28-42.el8_4.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/058f77432592f4337039cbb7a4e5f680020d8b85a477080c01d96a7728de6934",
    ],
)

rpm(
    name = "libdb-utils-0__5.3.28-42.el8_4.aarch64",
    sha256 = "84d0f5ae6a2bb4855d800c8e26be44bd06ac5f3c286a7877310bddabec12477a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libdb-utils-5.3.28-42.el8_4.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/84d0f5ae6a2bb4855d800c8e26be44bd06ac5f3c286a7877310bddabec12477a",
    ],
)

rpm(
    name = "libdb-utils-0__5.3.28-42.el8_4.x86_64",
    sha256 = "ceb3dbd9e0d39d3e6b566eaf05359de4dd9a18d09da9238f2319f66f7cfebf7b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libdb-utils-5.3.28-42.el8_4.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ceb3dbd9e0d39d3e6b566eaf05359de4dd9a18d09da9238f2319f66f7cfebf7b",
    ],
)

rpm(
    name = "libevent-0__2.1.8-5.el8.aarch64",
    sha256 = "a7fed3b521d23e60539dcbd548bda2a62f0d745a99dd5feeb43b6539f7f88232",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libevent-2.1.8-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a7fed3b521d23e60539dcbd548bda2a62f0d745a99dd5feeb43b6539f7f88232",
    ],
)

rpm(
    name = "libevent-0__2.1.8-5.el8.x86_64",
    sha256 = "746bac6bb011a586d42bd82b2f8b25bac72c9e4bbd4c19a34cf88eadb1d83873",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libevent-2.1.8-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/746bac6bb011a586d42bd82b2f8b25bac72c9e4bbd4c19a34cf88eadb1d83873",
    ],
)

rpm(
    name = "libfdisk-0__2.32.1-28.el8.aarch64",
    sha256 = "dbe365d0d44beafe99de8fa82e9789f873955e0ce1f66bebb785acca98ae3743",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libfdisk-2.32.1-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dbe365d0d44beafe99de8fa82e9789f873955e0ce1f66bebb785acca98ae3743",
    ],
)

rpm(
    name = "libfdisk-0__2.32.1-28.el8.x86_64",
    sha256 = "58cb137adc7edde3eb60b1ee8aac7de0198093621d6f37398b6023c79cd5bb06",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libfdisk-2.32.1-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/58cb137adc7edde3eb60b1ee8aac7de0198093621d6f37398b6023c79cd5bb06",
    ],
)

rpm(
    name = "libfdt-0__1.6.0-1.el8.aarch64",
    sha256 = "a2f3c86d18ee25ce4764a1df0854c63b615db37291ef9780e649f0123a92acf5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libfdt-1.6.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a2f3c86d18ee25ce4764a1df0854c63b615db37291ef9780e649f0123a92acf5",
    ],
)

rpm(
    name = "libffi-0__3.1-22.el8.aarch64",
    sha256 = "9d7e9a47e16b3edd1f9ce69c44bf485e8498cb6ced68e354b4c24936cd015bb5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libffi-3.1-22.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9d7e9a47e16b3edd1f9ce69c44bf485e8498cb6ced68e354b4c24936cd015bb5",
    ],
)

rpm(
    name = "libffi-0__3.1-22.el8.x86_64",
    sha256 = "3991890c6b556a06923002b0ad511c0e2d85e93cb0618758e68d72f95676b4e6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libffi-3.1-22.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3991890c6b556a06923002b0ad511c0e2d85e93cb0618758e68d72f95676b4e6",
    ],
)

rpm(
    name = "libgcc-0__8.5.0-3.el8.aarch64",
    sha256 = "f1f10022c95ef2ff496b3a358f6fa7f7474fecd4840ac0fac689075356a09689",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcc-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f1f10022c95ef2ff496b3a358f6fa7f7474fecd4840ac0fac689075356a09689",
    ],
)

rpm(
    name = "libgcc-0__8.5.0-3.el8.x86_64",
    sha256 = "59ce0d8c0aa6ed7ca399b20ffa125b096554e05127faea9da3602c215e811685",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcc-8.5.0-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/59ce0d8c0aa6ed7ca399b20ffa125b096554e05127faea9da3602c215e811685",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-6.el8.aarch64",
    sha256 = "e51932a986acc83e12f81396d532b58aacfa2b553fee84f1e62ffada1029bfd8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.8.5-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e51932a986acc83e12f81396d532b58aacfa2b553fee84f1e62ffada1029bfd8",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-6.el8.x86_64",
    sha256 = "f53997b3c5a858b3f2c640b1a2f2fcc1ba9f698bf12ae1b6ff5097d9095caa5e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.8.5-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f53997b3c5a858b3f2c640b1a2f2fcc1ba9f698bf12ae1b6ff5097d9095caa5e",
    ],
)

rpm(
    name = "libgomp-0__8.5.0-3.el8.aarch64",
    sha256 = "2d0278ec7c49b088bb7e1444a241154372d3ee560dc3d3564ffcf327c5e32c4f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgomp-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2d0278ec7c49b088bb7e1444a241154372d3ee560dc3d3564ffcf327c5e32c4f",
    ],
)

rpm(
    name = "libgomp-0__8.5.0-3.el8.x86_64",
    sha256 = "f67d405acce5a03b67fa7a01726b9a697d7f2f810ae6f6d4db295f2b2a0fe7ff",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgomp-8.5.0-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f67d405acce5a03b67fa7a01726b9a697d7f2f810ae6f6d4db295f2b2a0fe7ff",
    ],
)

rpm(
    name = "libgpg-error-0__1.31-1.el8.aarch64",
    sha256 = "b953729a0a2be24749aeee9f00853fdc3227737971cf052a999a37ac36387cd9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgpg-error-1.31-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b953729a0a2be24749aeee9f00853fdc3227737971cf052a999a37ac36387cd9",
    ],
)

rpm(
    name = "libgpg-error-0__1.31-1.el8.x86_64",
    sha256 = "845a0732d9d7a01b909124cd8293204764235c2d856227c7a74dfa0e38113e34",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgpg-error-1.31-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/845a0732d9d7a01b909124cd8293204764235c2d856227c7a74dfa0e38113e34",
    ],
)

rpm(
    name = "libguestfs-1__1.44.0-3.el8s.x86_64",
    sha256 = "dd8b612fb7f8b199c989e3d15bbeb101f8cc710e7a7b9880c09ce285dfd1eaad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libguestfs-1.44.0-3.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd8b612fb7f8b199c989e3d15bbeb101f8cc710e7a7b9880c09ce285dfd1eaad",
    ],
)

rpm(
    name = "libguestfs-tools-1__1.44.0-3.el8s.x86_64",
    sha256 = "de36b65d1686762617e567379f8d22bd7a5167b223aac5cfa0311cfcca7950a3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libguestfs-tools-1.44.0-3.el8s.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de36b65d1686762617e567379f8d22bd7a5167b223aac5cfa0311cfcca7950a3",
    ],
)

rpm(
    name = "libguestfs-tools-c-1__1.44.0-3.el8s.x86_64",
    sha256 = "8ae0f79d1a6b83b23c3b0d1d2bb126cadea908818d310a3c069c0c089418bd3f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libguestfs-tools-c-1.44.0-3.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ae0f79d1a6b83b23c3b0d1d2bb126cadea908818d310a3c069c0c089418bd3f",
    ],
)

rpm(
    name = "libibverbs-0__37.1-1.el8.aarch64",
    sha256 = "49e443b8c3bbf916f93ff7bf42a2685e3c80d0877265dc436fd6462880f622c4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libibverbs-37.1-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/49e443b8c3bbf916f93ff7bf42a2685e3c80d0877265dc436fd6462880f622c4",
    ],
)

rpm(
    name = "libibverbs-0__37.1-1.el8.x86_64",
    sha256 = "824778825b8d3bbe79f24b890e378a91e3b233613854dcf1ebd14c93fd618d21",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libibverbs-37.1-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/824778825b8d3bbe79f24b890e378a91e3b233613854dcf1ebd14c93fd618d21",
    ],
)

rpm(
    name = "libidn2-0__2.2.0-1.el8.aarch64",
    sha256 = "b62589101a60a365ef34447cae78f62e6dba560d403dc56c87036709ea00ad88",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libidn2-2.2.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b62589101a60a365ef34447cae78f62e6dba560d403dc56c87036709ea00ad88",
    ],
)

rpm(
    name = "libidn2-0__2.2.0-1.el8.x86_64",
    sha256 = "7e08785bd3cc0e09f9ab4bf600b98b705203d552cbb655269a939087987f1694",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libidn2-2.2.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e08785bd3cc0e09f9ab4bf600b98b705203d552cbb655269a939087987f1694",
    ],
)

rpm(
    name = "libisoburn-0__1.4.8-4.el8.aarch64",
    sha256 = "3ff828ef16f6033227d71207bc1b00983b826172fe7c575cd7590a72d846d831",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libisoburn-1.4.8-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3ff828ef16f6033227d71207bc1b00983b826172fe7c575cd7590a72d846d831",
    ],
)

rpm(
    name = "libisoburn-0__1.4.8-4.el8.x86_64",
    sha256 = "7aa030310250b462d90895d8c04ce47695722d86f5470930fdf8bfba0570c4dc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libisoburn-1.4.8-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7aa030310250b462d90895d8c04ce47695722d86f5470930fdf8bfba0570c4dc",
    ],
)

rpm(
    name = "libisofs-0__1.4.8-3.el8.aarch64",
    sha256 = "2e5435efba38348be8d33a43e5abbffc85f7c5a9504ebe6451b87c44006b3b4c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libisofs-1.4.8-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2e5435efba38348be8d33a43e5abbffc85f7c5a9504ebe6451b87c44006b3b4c",
    ],
)

rpm(
    name = "libisofs-0__1.4.8-3.el8.x86_64",
    sha256 = "66b7bcc256b62736f7b3d33fa65c6a91a17e08c61484a7c3748f4f86b4589bc7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libisofs-1.4.8-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/66b7bcc256b62736f7b3d33fa65c6a91a17e08c61484a7c3748f4f86b4589bc7",
    ],
)

rpm(
    name = "libksba-0__1.3.5-7.el8.x86_64",
    sha256 = "e6d3476e9996fb49632744be169f633d92900f5b7151db233501167a9018d240",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libksba-1.3.5-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e6d3476e9996fb49632744be169f633d92900f5b7151db233501167a9018d240",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-6.el8.aarch64",
    sha256 = "fbe4f2cb2660ebe3cb90a73c7dfbd978059af138356e46c9a93049761c0467ef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmnl-1.0.4-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fbe4f2cb2660ebe3cb90a73c7dfbd978059af138356e46c9a93049761c0467ef",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-6.el8.x86_64",
    sha256 = "30fab73ee155f03dbbd99c1e30fe59dfba4ae8fdb2e7213451ccc36d6918bfcc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmnl-1.0.4-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/30fab73ee155f03dbbd99c1e30fe59dfba4ae8fdb2e7213451ccc36d6918bfcc",
    ],
)

rpm(
    name = "libmount-0__2.32.1-28.el8.aarch64",
    sha256 = "2e0e94196aaaf205e6bda61d9379b789140d49ac3547d14fad573d012759e54d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmount-2.32.1-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2e0e94196aaaf205e6bda61d9379b789140d49ac3547d14fad573d012759e54d",
    ],
)

rpm(
    name = "libmount-0__2.32.1-28.el8.x86_64",
    sha256 = "a20142c98ca558697f68d07aee33d98759c45d1307fdc86ed1553c52f5b7bb96",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmount-2.32.1-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a20142c98ca558697f68d07aee33d98759c45d1307fdc86ed1553c52f5b7bb96",
    ],
)

rpm(
    name = "libmpc-0__1.1.0-9.1.el8.aarch64",
    sha256 = "9701bd94db9b467e11590b2de375a122ab61aa8d624be7df22631a6da91c79e4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libmpc-1.1.0-9.1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9701bd94db9b467e11590b2de375a122ab61aa8d624be7df22631a6da91c79e4",
    ],
)

rpm(
    name = "libmpc-0__1.1.0-9.1.el8.x86_64",
    sha256 = "93c2232d1885ec6265159f4669aeb13335a80e74d3ae0832f624678d87ea3638",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libmpc-1.1.0-9.1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/93c2232d1885ec6265159f4669aeb13335a80e74d3ae0832f624678d87ea3638",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.6-5.el8.aarch64",
    sha256 = "4e43b0f85746f74064b082fdf6914ba4e9fe386651b1c39aeaecc702b2a59fc0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnetfilter_conntrack-1.0.6-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4e43b0f85746f74064b082fdf6914ba4e9fe386651b1c39aeaecc702b2a59fc0",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.6-5.el8.x86_64",
    sha256 = "224100af3ecfc80c416796ec02c7c4dd113a38d42349d763485f3b42f260493f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnetfilter_conntrack-1.0.6-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/224100af3ecfc80c416796ec02c7c4dd113a38d42349d763485f3b42f260493f",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-13.el8.aarch64",
    sha256 = "8422fbc84108abc9a89fe98cef9cd18ad1788b4dc6a9ec0bba1836b772fcaeda",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnfnetlink-1.0.1-13.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8422fbc84108abc9a89fe98cef9cd18ad1788b4dc6a9ec0bba1836b772fcaeda",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-13.el8.x86_64",
    sha256 = "cec98aa5fbefcb99715921b493b4f92d34c4eeb823e9c8741aa75e280def89f1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.1-13.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cec98aa5fbefcb99715921b493b4f92d34c4eeb823e9c8741aa75e280def89f1",
    ],
)

rpm(
    name = "libnftnl-0__1.1.5-4.el8.aarch64",
    sha256 = "c85fbf0045e810a8a7df257799a82e32fee141db8119e9f1eb7abdb96553127f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnftnl-1.1.5-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c85fbf0045e810a8a7df257799a82e32fee141db8119e9f1eb7abdb96553127f",
    ],
)

rpm(
    name = "libnftnl-0__1.1.5-4.el8.x86_64",
    sha256 = "c1bb77ed45ae47dc068445c6dfa4b70b273a3daf8cd82b9fa7a50e3d59abe3c1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnftnl-1.1.5-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c1bb77ed45ae47dc068445c6dfa4b70b273a3daf8cd82b9fa7a50e3d59abe3c1",
    ],
)

rpm(
    name = "libnghttp2-0__1.33.0-3.el8_2.1.aarch64",
    sha256 = "23e9ff009c2316652c3bcd96a8b69b5bc26f2acd46214f652a7ce26a572cbabb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.33.0-3.el8_2.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/23e9ff009c2316652c3bcd96a8b69b5bc26f2acd46214f652a7ce26a572cbabb",
    ],
)

rpm(
    name = "libnghttp2-0__1.33.0-3.el8_2.1.x86_64",
    sha256 = "0126a384853d46484dec98601a4cb4ce58b2e0411f8f7ef09937174dd5975bac",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.33.0-3.el8_2.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0126a384853d46484dec98601a4cb4ce58b2e0411f8f7ef09937174dd5975bac",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-1.el8.aarch64",
    sha256 = "851a9cebfb68b8c301231b1121f573311fbb165ace0f4b1a599fa42f80113df9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnl3-3.5.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/851a9cebfb68b8c301231b1121f573311fbb165ace0f4b1a599fa42f80113df9",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-1.el8.x86_64",
    sha256 = "21c65dbf3b506a37828b13c205077f4b70fddb4b1d1c929dec01661238108059",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnl3-3.5.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21c65dbf3b506a37828b13c205077f4b70fddb4b1d1c929dec01661238108059",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-2.20180605git4a062cf.el8.aarch64",
    sha256 = "b33276781f442757afd5e066ead95ec79927f2aed608a368420f230d5ee28686",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnsl2-1.2.0-2.20180605git4a062cf.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b33276781f442757afd5e066ead95ec79927f2aed608a368420f230d5ee28686",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-2.20180605git4a062cf.el8.x86_64",
    sha256 = "5846c73edfa2ff673989728e9621cce6a1369eb2f8a269ac5205c381a10d327a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnsl2-1.2.0-2.20180605git4a062cf.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5846c73edfa2ff673989728e9621cce6a1369eb2f8a269ac5205c381a10d327a",
    ],
)

rpm(
    name = "libpcap-14__1.9.1-5.el8.aarch64",
    sha256 = "239019a8aadb26e4b015d99f7fe49e80c2d1dfa227f7c71322dca2a2a85c2de1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpcap-1.9.1-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/239019a8aadb26e4b015d99f7fe49e80c2d1dfa227f7c71322dca2a2a85c2de1",
    ],
)

rpm(
    name = "libpcap-14__1.9.1-5.el8.x86_64",
    sha256 = "7f429477c26b4650a3eca4a27b3972ff0857c843bdb4d8fcb02086da111ce5fd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpcap-1.9.1-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f429477c26b4650a3eca4a27b3972ff0857c843bdb4d8fcb02086da111ce5fd",
    ],
)

rpm(
    name = "libpkgconf-0__1.4.2-1.el8.aarch64",
    sha256 = "8f3e34df67e6c4a20bd7617f17d1199f0441a626fbab8059ddc6bf06c7ff4e78",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpkgconf-1.4.2-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8f3e34df67e6c4a20bd7617f17d1199f0441a626fbab8059ddc6bf06c7ff4e78",
    ],
)

rpm(
    name = "libpkgconf-0__1.4.2-1.el8.x86_64",
    sha256 = "a76ff4cf270d2e38106a4bba1880c3a0899d186cd4e1986d7e97c01b934e13b7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpkgconf-1.4.2-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a76ff4cf270d2e38106a4bba1880c3a0899d186cd4e1986d7e97c01b934e13b7",
    ],
)

rpm(
    name = "libpmem-0__1.9.2-1.module_el8.5.0__plus__756__plus__4cdc1762.x86_64",
    sha256 = "82a3a0bb6541ed6110a700488977ecf0aaf2650203356ea2f13fbc9410640706",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libpmem-1.9.2-1.module_el8.5.0+756+4cdc1762.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82a3a0bb6541ed6110a700488977ecf0aaf2650203356ea2f13fbc9410640706",
    ],
)

rpm(
    name = "libpng-2__1.6.34-5.el8.aarch64",
    sha256 = "d7bd4e7a7ff4424266c0f6030bf444de0bea88d0540ff4caf4f7f6c2bac175f6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpng-1.6.34-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d7bd4e7a7ff4424266c0f6030bf444de0bea88d0540ff4caf4f7f6c2bac175f6",
    ],
)

rpm(
    name = "libpng-2__1.6.34-5.el8.x86_64",
    sha256 = "cc2f054cf7ef006faf0b179701838ff8632c3ac5f45a0199a13f9c237f632b82",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpng-1.6.34-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cc2f054cf7ef006faf0b179701838ff8632c3ac5f45a0199a13f9c237f632b82",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-3.el8.aarch64",
    sha256 = "64e55ddddc1dd27e05097c9222e73052f6f20f9d2f7605f46922b7756adeb0b5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpwquality-1.4.4-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/64e55ddddc1dd27e05097c9222e73052f6f20f9d2f7605f46922b7756adeb0b5",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-3.el8.x86_64",
    sha256 = "e42ec1259c966909507a6b4c4cd25b183268d4516dd9a8d60078c8a4b6df0014",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.4-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e42ec1259c966909507a6b4c4cd25b183268d4516dd9a8d60078c8a4b6df0014",
    ],
)

rpm(
    name = "librdmacm-0__37.1-1.el8.aarch64",
    sha256 = "51b0954cb1bd4e694763f4e49e5a2b964d54e38f13232b75d9c06f577be7642a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/librdmacm-37.1-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/51b0954cb1bd4e694763f4e49e5a2b964d54e38f13232b75d9c06f577be7642a",
    ],
)

rpm(
    name = "librdmacm-0__37.1-1.el8.x86_64",
    sha256 = "df44b6cb1a470fe35ec2a652c8b70d469ff002b86941301796cc2eadf3769820",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/librdmacm-37.1-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df44b6cb1a470fe35ec2a652c8b70d469ff002b86941301796cc2eadf3769820",
    ],
)

rpm(
    name = "libreport-filesystem-0__2.9.5-15.el8.x86_64",
    sha256 = "b9cfde532f94e32540b51c74547da69bb06045e5d03c4e7d4be909dbcf929887",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libreport-filesystem-2.9.5-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b9cfde532f94e32540b51c74547da69bb06045e5d03c4e7d4be909dbcf929887",
    ],
)

rpm(
    name = "libseccomp-0__2.5.2-1.el8.aarch64",
    sha256 = "2460f610a00c11b7070ff75d27fb22fab4b8d67c856da2ffb097cf3eff28f365",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libseccomp-2.5.2-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2460f610a00c11b7070ff75d27fb22fab4b8d67c856da2ffb097cf3eff28f365",
    ],
)

rpm(
    name = "libseccomp-0__2.5.2-1.el8.x86_64",
    sha256 = "4a6322832274a9507108719de9af48406ee0fcfc54c9906b9450e1ae231ede4b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libseccomp-2.5.2-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a6322832274a9507108719de9af48406ee0fcfc54c9906b9450e1ae231ede4b",
    ],
)

rpm(
    name = "libselinux-0__2.9-5.el8.aarch64",
    sha256 = "9474fe348bd9e3a7a6ffe7813538e979e80ddb970b074e4e79bd122b4ece8b64",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libselinux-2.9-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9474fe348bd9e3a7a6ffe7813538e979e80ddb970b074e4e79bd122b4ece8b64",
    ],
)

rpm(
    name = "libselinux-0__2.9-5.el8.x86_64",
    sha256 = "89e54e0975b9c87c45d3478d9f8bcc3f19a90e9ef16062a524af4a8efc059e1f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libselinux-2.9-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89e54e0975b9c87c45d3478d9f8bcc3f19a90e9ef16062a524af4a8efc059e1f",
    ],
)

rpm(
    name = "libselinux-utils-0__2.9-5.el8.aarch64",
    sha256 = "e4613455147d283b222fcff5ef0f85b3a1a323893ed884db8950e51936e97c52",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libselinux-utils-2.9-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e4613455147d283b222fcff5ef0f85b3a1a323893ed884db8950e51936e97c52",
    ],
)

rpm(
    name = "libselinux-utils-0__2.9-5.el8.x86_64",
    sha256 = "5063fe914f04ca203e3f28529021c40ef01ad8ed33330fafc0f658581a78b722",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libselinux-utils-2.9-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5063fe914f04ca203e3f28529021c40ef01ad8ed33330fafc0f658581a78b722",
    ],
)

rpm(
    name = "libsemanage-0__2.9-6.el8.aarch64",
    sha256 = "ccb929460b2e9f3fc477b5f040b8e9de1faab4492e696aac4d4eafd4d82b7ba3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsemanage-2.9-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ccb929460b2e9f3fc477b5f040b8e9de1faab4492e696aac4d4eafd4d82b7ba3",
    ],
)

rpm(
    name = "libsemanage-0__2.9-6.el8.x86_64",
    sha256 = "6ba1f1f26bc8e261a813883e0cbcd7b0f542109e797fb6092afba8dc7f1ea269",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsemanage-2.9-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6ba1f1f26bc8e261a813883e0cbcd7b0f542109e797fb6092afba8dc7f1ea269",
    ],
)

rpm(
    name = "libsepol-0__2.9-3.el8.aarch64",
    sha256 = "e9d2e6252228076c270850b51b7205baed31c1c3c2ccdb9d3280c9b0de5d652a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsepol-2.9-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e9d2e6252228076c270850b51b7205baed31c1c3c2ccdb9d3280c9b0de5d652a",
    ],
)

rpm(
    name = "libsepol-0__2.9-3.el8.x86_64",
    sha256 = "f91e372ffa25c4c82ae7e001565cf5ff73048c407083493555025fdb5fc4c14a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsepol-2.9-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f91e372ffa25c4c82ae7e001565cf5ff73048c407083493555025fdb5fc4c14a",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-5.el8.aarch64",
    sha256 = "b377f4e8bcdc750ed0be94f97bdbfbb12843c458fbc1d5d507f92ad04aaf592b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsigsegv-2.11-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b377f4e8bcdc750ed0be94f97bdbfbb12843c458fbc1d5d507f92ad04aaf592b",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-5.el8.x86_64",
    sha256 = "02d728cf74eb47005babeeab5ac68ca04472c643203a1faef0037b5f33710fe2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsigsegv-2.11-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/02d728cf74eb47005babeeab5ac68ca04472c643203a1faef0037b5f33710fe2",
    ],
)

rpm(
    name = "libsmartcols-0__2.32.1-28.el8.aarch64",
    sha256 = "2f037740a6275018d75377a0b50a60866215f7e086f44c609835a1d08c629ce4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.32.1-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2f037740a6275018d75377a0b50a60866215f7e086f44c609835a1d08c629ce4",
    ],
)

rpm(
    name = "libsmartcols-0__2.32.1-28.el8.x86_64",
    sha256 = "da4240251c7d94968ef0b834751c789fbe49993655aa059a1533ff5f187cce6d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.32.1-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da4240251c7d94968ef0b834751c789fbe49993655aa059a1533ff5f187cce6d",
    ],
)

rpm(
    name = "libss-0__1.45.6-2.el8.aarch64",
    sha256 = "be9516ec31fa9282fa26a30d86eb13e195274b4910b3180d2e627e2bb7baa671",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libss-1.45.6-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/be9516ec31fa9282fa26a30d86eb13e195274b4910b3180d2e627e2bb7baa671",
    ],
)

rpm(
    name = "libss-0__1.45.6-2.el8.x86_64",
    sha256 = "e66194044367e413e733c0aeebfa04ec7acaef3c330fb3f331b79976152fdf37",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libss-1.45.6-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e66194044367e413e733c0aeebfa04ec7acaef3c330fb3f331b79976152fdf37",
    ],
)

rpm(
    name = "libssh-0__0.9.6-3.el8.aarch64",
    sha256 = "4e7b5c73bf2ff1dc42904d96b86891ab3d2ccc27ba0e6d71de4984f9b1e71d65",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libssh-0.9.6-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4e7b5c73bf2ff1dc42904d96b86891ab3d2ccc27ba0e6d71de4984f9b1e71d65",
    ],
)

rpm(
    name = "libssh-0__0.9.6-3.el8.x86_64",
    sha256 = "56db2bbc7028a0b031250b262a70d37de96edeb8832836e426d7a2b9d35bab12",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libssh-0.9.6-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56db2bbc7028a0b031250b262a70d37de96edeb8832836e426d7a2b9d35bab12",
    ],
)

rpm(
    name = "libssh-config-0__0.9.6-3.el8.aarch64",
    sha256 = "e9e954ba21bac58e3aebaf52bf824758fe4c2ad09d75171b3009a214bd52bbec",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libssh-config-0.9.6-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e9e954ba21bac58e3aebaf52bf824758fe4c2ad09d75171b3009a214bd52bbec",
    ],
)

rpm(
    name = "libssh-config-0__0.9.6-3.el8.x86_64",
    sha256 = "e9e954ba21bac58e3aebaf52bf824758fe4c2ad09d75171b3009a214bd52bbec",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libssh-config-0.9.6-3.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e9e954ba21bac58e3aebaf52bf824758fe4c2ad09d75171b3009a214bd52bbec",
    ],
)

rpm(
    name = "libsss_idmap-0__2.5.2-2.el8_5.1.aarch64",
    sha256 = "3f771df57c4fc8d05604531b46f0196f177265b14969c0a284e6a0318d0e54e0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.5.2-2.el8_5.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3f771df57c4fc8d05604531b46f0196f177265b14969c0a284e6a0318d0e54e0",
    ],
)

rpm(
    name = "libsss_idmap-0__2.5.2-2.el8_5.1.x86_64",
    sha256 = "a21a4ff8158ee80e7353aeca04eb812f4c9ed9394ebb6c3e605cc5edba8edfc6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.5.2-2.el8_5.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a21a4ff8158ee80e7353aeca04eb812f4c9ed9394ebb6c3e605cc5edba8edfc6",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.5.2-2.el8_5.1.aarch64",
    sha256 = "6521d0b915181c5631bce1ce8a30cf1483442b6b2b77af95c639cbb106b0493b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.5.2-2.el8_5.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6521d0b915181c5631bce1ce8a30cf1483442b6b2b77af95c639cbb106b0493b",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.5.2-2.el8_5.1.x86_64",
    sha256 = "f352791eb3799100c12aae6c58aade243b252667eadcb3d81609b12ff75f645b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.5.2-2.el8_5.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f352791eb3799100c12aae6c58aade243b252667eadcb3d81609b12ff75f645b",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-3.el8.aarch64",
    sha256 = "62b1ecd40aa76506162253dd1453f3ecd70994ae82fa86a972c2118793cb1d34",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libstdc++-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/62b1ecd40aa76506162253dd1453f3ecd70994ae82fa86a972c2118793cb1d34",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-3.el8.x86_64",
    sha256 = "e204a911cf409a4da2a9c92841f2a27af38d1a249dadaff77df4bfd072345d1b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libstdc++-8.5.0-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e204a911cf409a4da2a9c92841f2a27af38d1a249dadaff77df4bfd072345d1b",
    ],
)

rpm(
    name = "libtasn1-0__4.13-3.el8.aarch64",
    sha256 = "3401ccfb7fd08c12578b6257b4dac7e94ba5f4cd70fc6a234fd90bb99d1bb108",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libtasn1-4.13-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3401ccfb7fd08c12578b6257b4dac7e94ba5f4cd70fc6a234fd90bb99d1bb108",
    ],
)

rpm(
    name = "libtasn1-0__4.13-3.el8.x86_64",
    sha256 = "e8d9697a8914226a2d3ed5a4523b85e8e70ac09cf90aae05395e6faee9858534",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libtasn1-4.13-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e8d9697a8914226a2d3ed5a4523b85e8e70ac09cf90aae05395e6faee9858534",
    ],
)

rpm(
    name = "libtirpc-0__1.1.4-5.el8.aarch64",
    sha256 = "c378aad0473ca944ce881d3d45bd76429e365216634e63213e0bdc19738d25db",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libtirpc-1.1.4-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c378aad0473ca944ce881d3d45bd76429e365216634e63213e0bdc19738d25db",
    ],
)

rpm(
    name = "libtirpc-0__1.1.4-5.el8.x86_64",
    sha256 = "71f2babdefc7c063cce7541f3f132d3fed6f5a1df94f360850d4dc3d95a7bf28",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libtirpc-1.1.4-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/71f2babdefc7c063cce7541f3f132d3fed6f5a1df94f360850d4dc3d95a7bf28",
    ],
)

rpm(
    name = "libtpms-0__0.7.4-6.20201106git2452a24dab.module_el8.6.0__plus__983__plus__a7505f3f.aarch64",
    sha256 = "0ae491b89a5601c7a3f180b425d4cf3107cc494f798dfdb1ec30329fe458df20",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libtpms-0.7.4-6.20201106git2452a24dab.module_el8.6.0+983+a7505f3f.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0ae491b89a5601c7a3f180b425d4cf3107cc494f798dfdb1ec30329fe458df20",
    ],
)

rpm(
    name = "libtpms-0__0.7.4-6.20201106git2452a24dab.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "6c95ff51281fbed9019e959c7aed143b3e070e8a326e7903ead86b218d8dc1cc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libtpms-0.7.4-6.20201106git2452a24dab.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6c95ff51281fbed9019e959c7aed143b3e070e8a326e7903ead86b218d8dc1cc",
    ],
)

rpm(
    name = "libubsan-0__8.5.0-3.el8.aarch64",
    sha256 = "b7df5f809e95f49b198ff660d3b8a1771d49564b1d7ff479f5ddefa4d590bba3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libubsan-8.5.0-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b7df5f809e95f49b198ff660d3b8a1771d49564b1d7ff479f5ddefa4d590bba3",
    ],
)

rpm(
    name = "libunistring-0__0.9.9-3.el8.aarch64",
    sha256 = "707429ccb3223628d55097a162cd0d3de1bd00b48800677c1099931b0f019e80",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libunistring-0.9.9-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/707429ccb3223628d55097a162cd0d3de1bd00b48800677c1099931b0f019e80",
    ],
)

rpm(
    name = "libunistring-0__0.9.9-3.el8.x86_64",
    sha256 = "20bb189228afa589141d9c9d4ed457729d13c11608305387602d0b00ed0a3093",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libunistring-0.9.9-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20bb189228afa589141d9c9d4ed457729d13c11608305387602d0b00ed0a3093",
    ],
)

rpm(
    name = "libusal-0__1.1.11-39.el8.x86_64",
    sha256 = "0b2b79d9f8cd01090816386ad89852662b5489bbd43fbd04760f0e57c28bce4c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libusal-1.1.11-39.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0b2b79d9f8cd01090816386ad89852662b5489bbd43fbd04760f0e57c28bce4c",
    ],
)

rpm(
    name = "libusbx-0__1.0.23-4.el8.aarch64",
    sha256 = "ae797d004f3cafb89773fcc8a3f0d6d046546b7cb3f9741be200d095c637706f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libusbx-1.0.23-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ae797d004f3cafb89773fcc8a3f0d6d046546b7cb3f9741be200d095c637706f",
    ],
)

rpm(
    name = "libusbx-0__1.0.23-4.el8.x86_64",
    sha256 = "7e704756a93f07feec345a9748204e78994ce06a4667a2ef35b44964ff754306",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libusbx-1.0.23-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e704756a93f07feec345a9748204e78994ce06a4667a2ef35b44964ff754306",
    ],
)

rpm(
    name = "libutempter-0__1.1.6-14.el8.aarch64",
    sha256 = "8f6d9839a758fdacfdb4b4b0731e8023b8bbb0b633bd32dbf21c2ce85a933a8a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libutempter-1.1.6-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8f6d9839a758fdacfdb4b4b0731e8023b8bbb0b633bd32dbf21c2ce85a933a8a",
    ],
)

rpm(
    name = "libutempter-0__1.1.6-14.el8.x86_64",
    sha256 = "c8c54c56bff9ca416c3ba6bccac483fb66c81a53d93a19420088715018ed5169",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libutempter-1.1.6-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c8c54c56bff9ca416c3ba6bccac483fb66c81a53d93a19420088715018ed5169",
    ],
)

rpm(
    name = "libuuid-0__2.32.1-28.el8.aarch64",
    sha256 = "3976c3648ef9503a771a8f2466bb854b68c1569d363018db9cc63e097ecff41b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libuuid-2.32.1-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3976c3648ef9503a771a8f2466bb854b68c1569d363018db9cc63e097ecff41b",
    ],
)

rpm(
    name = "libuuid-0__2.32.1-28.el8.x86_64",
    sha256 = "4338967a50af54b0392b45193be52959157e67122eef9ebf12aa2b0b3661b16c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libuuid-2.32.1-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4338967a50af54b0392b45193be52959157e67122eef9ebf12aa2b0b3661b16c",
    ],
)

rpm(
    name = "libverto-0__0.3.0-5.el8.aarch64",
    sha256 = "446f45706d78e80d4057d9d55dda32ce1cb823b2ca4dfe50f0ca5b515238130d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libverto-0.3.0-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/446f45706d78e80d4057d9d55dda32ce1cb823b2ca4dfe50f0ca5b515238130d",
    ],
)

rpm(
    name = "libverto-0__0.3.0-5.el8.x86_64",
    sha256 = "f95f673fc9236dc712270a343807cdac06297d847001e78cd707482c751b2d0d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libverto-0.3.0-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f95f673fc9236dc712270a343807cdac06297d847001e78cd707482c751b2d0d",
    ],
)

rpm(
    name = "libvirt-client-0__7.6.0-6.el8s.aarch64",
    sha256 = "b3bbb74465f8cf00e7f8f5889ef952b3849ac5392c032cd2eec423b04d89c179",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-client-7.6.0-6.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-client-0__7.6.0-6.el8s.x86_64",
    sha256 = "9eea0d26a51c00032f31da04b29ece265b82250f011a1ef0cdad75ec1a547b0d",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-client-7.6.0-6.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-0__7.6.0-6.el8s.aarch64",
    sha256 = "c6e4128f38aa16a27d79681716865f0eb903421d1097bda3e30effc29e9a129e",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-daemon-7.6.0-6.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-daemon-0__7.6.0-6.el8s.x86_64",
    sha256 = "6aa8e6f8d7dc8c419a6bc01773900402facce418cad5eb8b43781fb603e1460c",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-7.6.0-6.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.6.0-6.el8s.aarch64",
    sha256 = "021cd9f5005bbd9db238b0840cc1725366671599987ad8622eb399ed20fc7395",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-daemon-driver-qemu-7.6.0-6.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.6.0-6.el8s.x86_64",
    sha256 = "9e7dbb1b6f81c6100e7517663d63cb8ac9eab5758a979a8ba86d5fcd8072aded",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-driver-qemu-7.6.0-6.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-devel-0__7.6.0-6.el8s.aarch64",
    sha256 = "f6d8f8edd459aed9a0ef14e23afd0beedc299c8ad2133d52d61e919d5bd4fd0f",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-devel-7.6.0-6.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-devel-0__7.6.0-6.el8s.x86_64",
    sha256 = "9d0f5740c176cb7630e4d4add1b839a4648a9ad723be754abd324eb9a65c504f",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-devel-7.6.0-6.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-libs-0__7.6.0-6.el8s.aarch64",
    sha256 = "79071ce936c963eca72fea4280008a6df494a594eb5346bcdf4810080eb09bd1",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-libs-7.6.0-6.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-libs-0__7.6.0-6.el8s.x86_64",
    sha256 = "d0ced5e0ffae07a2aa17da0509e01a3efd251901eca47b4078bf89c476e801b1",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-libs-7.6.0-6.el8s.x86_64.rpm"],
)

rpm(
    name = "libxcrypt-0__4.1.1-6.el8.aarch64",
    sha256 = "4948420ee35381c71c619fab4b8deabfa93c04e7c5729620b02e4382a50550ad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libxcrypt-4.1.1-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4948420ee35381c71c619fab4b8deabfa93c04e7c5729620b02e4382a50550ad",
    ],
)

rpm(
    name = "libxcrypt-0__4.1.1-6.el8.x86_64",
    sha256 = "645853feb85c921d979cb9cf9109663528429eda63cf5a1e31fe578d3d7e713a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libxcrypt-4.1.1-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/645853feb85c921d979cb9cf9109663528429eda63cf5a1e31fe578d3d7e713a",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.1.1-6.el8.aarch64",
    sha256 = "c561c433a3c295f5d7a49e79a43e4cc96094ed15bcc2fa271bf31f5a6deeacd1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libxcrypt-devel-4.1.1-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c561c433a3c295f5d7a49e79a43e4cc96094ed15bcc2fa271bf31f5a6deeacd1",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.1.1-6.el8.x86_64",
    sha256 = "6d84082741a4b7f1a98872a7ee8f12efca835b3dbcb15401aa1b5eccfc674bd4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libxcrypt-devel-4.1.1-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6d84082741a4b7f1a98872a7ee8f12efca835b3dbcb15401aa1b5eccfc674bd4",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.1.1-6.el8.aarch64",
    sha256 = "a8268856b30e6700f0f67651a6a43449b1e5fccaff512a95280d305468e44dfc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/PowerTools/aarch64/os/Packages/libxcrypt-static-4.1.1-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a8268856b30e6700f0f67651a6a43449b1e5fccaff512a95280d305468e44dfc",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.1.1-6.el8.x86_64",
    sha256 = "599cded5497aa6155c409321f3bb88b7a820341e1d502eac80bf17447283a29b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/PowerTools/x86_64/os/Packages/libxcrypt-static-4.1.1-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/599cded5497aa6155c409321f3bb88b7a820341e1d502eac80bf17447283a29b",
    ],
)

rpm(
    name = "libxkbcommon-0__0.9.1-1.el8.aarch64",
    sha256 = "3aca03c788af2ecf8ef39421f246769d7ef7f37260ee9421fc68c1d1cc913600",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libxkbcommon-0.9.1-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3aca03c788af2ecf8ef39421f246769d7ef7f37260ee9421fc68c1d1cc913600",
    ],
)

rpm(
    name = "libxkbcommon-0__0.9.1-1.el8.x86_64",
    sha256 = "e03d462995326a4477dcebc8c12eae3c1776ce2f095617ace253c0c492c89082",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libxkbcommon-0.9.1-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e03d462995326a4477dcebc8c12eae3c1776ce2f095617ace253c0c492c89082",
    ],
)

rpm(
    name = "libxml2-0__2.9.7-11.el8.aarch64",
    sha256 = "3514c1fa9f0ff57538e74e9b66991e4911e5176e250d49cd6fe079d4a9a3ba04",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.7-11.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3514c1fa9f0ff57538e74e9b66991e4911e5176e250d49cd6fe079d4a9a3ba04",
    ],
)

rpm(
    name = "libxml2-0__2.9.7-11.el8.x86_64",
    sha256 = "d13a830e42506b9ada2b719521e020e4857bc49aacef3c9a66368485690443da",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.7-11.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d13a830e42506b9ada2b719521e020e4857bc49aacef3c9a66368485690443da",
    ],
)

rpm(
    name = "libzstd-0__1.4.4-1.el8.aarch64",
    sha256 = "b560a8a185100a7c80e6c32f69ba65ce17004156f7218cf183249b15c13295cc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libzstd-1.4.4-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b560a8a185100a7c80e6c32f69ba65ce17004156f7218cf183249b15c13295cc",
    ],
)

rpm(
    name = "libzstd-0__1.4.4-1.el8.x86_64",
    sha256 = "7c2dc6044f13fe4ae04a4c1620da822a6be591b5129bf68ba98a3d8e9092f83b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libzstd-1.4.4-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7c2dc6044f13fe4ae04a4c1620da822a6be591b5129bf68ba98a3d8e9092f83b",
    ],
)

rpm(
    name = "lsscsi-0__0.32-3.el8.x86_64",
    sha256 = "863628671d0164392b4977b2e674a18531fadd6269d97e4f5485b36c01aef5a7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lsscsi-0.32-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/863628671d0164392b4977b2e674a18531fadd6269d97e4f5485b36c01aef5a7",
    ],
)

rpm(
    name = "lua-libs-0__5.3.4-12.el8.aarch64",
    sha256 = "2ef9801e4453de316429be284d4f6cb12f4d7662e7c6224dbf2341e3cfc5fab6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lua-libs-5.3.4-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2ef9801e4453de316429be284d4f6cb12f4d7662e7c6224dbf2341e3cfc5fab6",
    ],
)

rpm(
    name = "lua-libs-0__5.3.4-12.el8.x86_64",
    sha256 = "0268af0ee5754fb90fcf71b00fb737f1bf5b3c54c9ff312f13df8c2201311cfe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lua-libs-5.3.4-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0268af0ee5754fb90fcf71b00fb737f1bf5b3c54c9ff312f13df8c2201311cfe",
    ],
)

rpm(
    name = "lvm2-8__2.03.14-1.el8.x86_64",
    sha256 = "020d59d3269a73d6a5cb91bfe3910b8dbbd3fc99c378fb1bc72a37fc17528c82",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-2.03.14-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/020d59d3269a73d6a5cb91bfe3910b8dbbd3fc99c378fb1bc72a37fc17528c82",
    ],
)

rpm(
    name = "lvm2-libs-8__2.03.14-1.el8.x86_64",
    sha256 = "b7873f99a7e76627e0443b4ba342d79f72d35661246a934577b99fe5f3cff188",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-libs-2.03.14-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7873f99a7e76627e0443b4ba342d79f72d35661246a934577b99fe5f3cff188",
    ],
)

rpm(
    name = "lz4-libs-0__1.8.3-3.el8_4.aarch64",
    sha256 = "db9075646bed11355faf8b425c655a40a55436715a9f401f60e205ddd66edfeb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lz4-libs-1.8.3-3.el8_4.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/db9075646bed11355faf8b425c655a40a55436715a9f401f60e205ddd66edfeb",
    ],
)

rpm(
    name = "lz4-libs-0__1.8.3-3.el8_4.x86_64",
    sha256 = "8ecac05bb0ec99f91026f2361f7443b9be3272582193a7836884ec473bf8f423",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lz4-libs-1.8.3-3.el8_4.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ecac05bb0ec99f91026f2361f7443b9be3272582193a7836884ec473bf8f423",
    ],
)

rpm(
    name = "lzo-0__2.08-14.el8.aarch64",
    sha256 = "6809839757bd05082ca1b8d23eac617898eda3ce34844a0d31b0a030c8cc6653",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lzo-2.08-14.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6809839757bd05082ca1b8d23eac617898eda3ce34844a0d31b0a030c8cc6653",
    ],
)

rpm(
    name = "lzo-0__2.08-14.el8.x86_64",
    sha256 = "5c68635cb03533a38d4a42f6547c21a1d5f9952351bb01f3cf865d2621a6e634",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lzo-2.08-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5c68635cb03533a38d4a42f6547c21a1d5f9952351bb01f3cf865d2621a6e634",
    ],
)

rpm(
    name = "lzop-0__1.03-20.el8.aarch64",
    sha256 = "003b309833a1ed94ad97ed62f04c2fcda4a20fb8b7b5933c36459974f4e4986c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lzop-1.03-20.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/003b309833a1ed94ad97ed62f04c2fcda4a20fb8b7b5933c36459974f4e4986c",
    ],
)

rpm(
    name = "lzop-0__1.03-20.el8.x86_64",
    sha256 = "04eae61018a5be7656be832797016f97cd7b6e19d56f58cb658cd3969dedf2b0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lzop-1.03-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/04eae61018a5be7656be832797016f97cd7b6e19d56f58cb658cd3969dedf2b0",
    ],
)

rpm(
    name = "mdadm-0__4.2-rc2.el8.x86_64",
    sha256 = "08ed63795716f7da0aabaa5c250d64d0ab86c1545553d2635094c8efb48c6be0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mdadm-4.2-rc2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/08ed63795716f7da0aabaa5c250d64d0ab86c1545553d2635094c8efb48c6be0",
    ],
)

rpm(
    name = "mpfr-0__3.1.6-1.el8.aarch64",
    sha256 = "97a998a1b93c21bf070f9a9a1dbb525234b00fccedfe67de8967cd9ec7132eb1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/mpfr-3.1.6-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/97a998a1b93c21bf070f9a9a1dbb525234b00fccedfe67de8967cd9ec7132eb1",
    ],
)

rpm(
    name = "mpfr-0__3.1.6-1.el8.x86_64",
    sha256 = "e7f0c34f83c1ec2abb22951779e84d51e234c4ba0a05252e4ffd8917461891a5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mpfr-3.1.6-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e7f0c34f83c1ec2abb22951779e84d51e234c4ba0a05252e4ffd8917461891a5",
    ],
)

rpm(
    name = "mtools-0__4.0.18-14.el8.x86_64",
    sha256 = "f726efa5063fdb4b0bff847b20087a3286f9c069ce62f75561a6d1adee0dad5a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mtools-4.0.18-14.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f726efa5063fdb4b0bff847b20087a3286f9c069ce62f75561a6d1adee0dad5a",
    ],
)

rpm(
    name = "ncurses-base-0__6.1-9.20180224.el8.aarch64",
    sha256 = "41716536ea16798238ac89fbc3041b3f9dc80f9a64ea4b19d6e67ad2c909269a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.1-9.20180224.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/41716536ea16798238ac89fbc3041b3f9dc80f9a64ea4b19d6e67ad2c909269a",
    ],
)

rpm(
    name = "ncurses-base-0__6.1-9.20180224.el8.x86_64",
    sha256 = "41716536ea16798238ac89fbc3041b3f9dc80f9a64ea4b19d6e67ad2c909269a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.1-9.20180224.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/41716536ea16798238ac89fbc3041b3f9dc80f9a64ea4b19d6e67ad2c909269a",
    ],
)

rpm(
    name = "ncurses-libs-0__6.1-9.20180224.el8.aarch64",
    sha256 = "b938a6facc8d8a3de12b369871738bb531c822b1ec5212501b06bcaaf6cd25fa",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.1-9.20180224.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b938a6facc8d8a3de12b369871738bb531c822b1ec5212501b06bcaaf6cd25fa",
    ],
)

rpm(
    name = "ncurses-libs-0__6.1-9.20180224.el8.x86_64",
    sha256 = "54609dd070a57a14a6103f0c06bea99bb0a4e568d1fbc6a22b8ba67c954d90bf",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.1-9.20180224.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54609dd070a57a14a6103f0c06bea99bb0a4e568d1fbc6a22b8ba67c954d90bf",
    ],
)

rpm(
    name = "ndctl-libs-0__71.1-2.el8.x86_64",
    sha256 = "7e8fa8aa5971b39e329905feb378545d8dad32a46e4d25e9a9daf5eb19e6c593",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ndctl-libs-71.1-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e8fa8aa5971b39e329905feb378545d8dad32a46e4d25e9a9daf5eb19e6c593",
    ],
)

rpm(
    name = "nettle-0__3.4.1-7.el8.aarch64",
    sha256 = "5441222132ae52cd31063e9b9e3bb40f2e5711dfb0c84315b4aec2907278a075",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/nettle-3.4.1-7.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5441222132ae52cd31063e9b9e3bb40f2e5711dfb0c84315b4aec2907278a075",
    ],
)

rpm(
    name = "nettle-0__3.4.1-7.el8.x86_64",
    sha256 = "fe9a848502c595e0b7acc699d69c24b9c5ad0ac58a0b3933cd228f3633de31cb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/nettle-3.4.1-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe9a848502c595e0b7acc699d69c24b9c5ad0ac58a0b3933cd228f3633de31cb",
    ],
)

rpm(
    name = "nftables-1__0.9.3-23.el8.aarch64",
    sha256 = "0c42fc6f392b74d1ad5984dbd665ca2c13b1d2c17fc7ab548faa7990445d9603",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/nftables-0.9.3-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0c42fc6f392b74d1ad5984dbd665ca2c13b1d2c17fc7ab548faa7990445d9603",
    ],
)

rpm(
    name = "nftables-1__0.9.3-23.el8.x86_64",
    sha256 = "0a8cfe673ffe01ed333726b0dcf097f2a89f2e7046412f11d9ae15faef568335",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/nftables-0.9.3-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0a8cfe673ffe01ed333726b0dcf097f2a89f2e7046412f11d9ae15faef568335",
    ],
)

rpm(
    name = "nmap-ncat-2__7.70-6.el8.aarch64",
    sha256 = "541ddb604ddf8405ae552528ec05ac559f963fe5628de2b11354cbc8d7ce1ed0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.70-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/541ddb604ddf8405ae552528ec05ac559f963fe5628de2b11354cbc8d7ce1ed0",
    ],
)

rpm(
    name = "nmap-ncat-2__7.70-6.el8.x86_64",
    sha256 = "1397e8c7ef1a7b3680cd8119b1e231db1a5ee0a5202e6e557f2e9082a92761ca",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.70-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1397e8c7ef1a7b3680cd8119b1e231db1a5ee0a5202e6e557f2e9082a92761ca",
    ],
)

rpm(
    name = "npth-0__1.5-4.el8.x86_64",
    sha256 = "168ab5dbc86b836b8742b2e63eee51d074f1d790728e3d30b0c59fff93cf1d8d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/npth-1.5-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/168ab5dbc86b836b8742b2e63eee51d074f1d790728e3d30b0c59fff93cf1d8d",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.12-13.el8.aarch64",
    sha256 = "5f2d7a8db99ad318df35e60d43e5e7f462294c00ffa3d7c24207c16bfd3a6619",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.12-13.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5f2d7a8db99ad318df35e60d43e5e7f462294c00ffa3d7c24207c16bfd3a6619",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.12-13.el8.x86_64",
    sha256 = "b7b71ba34b3af893dc0acbb9d2228a2307da849d38e1c0007bd3d64f456640af",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.12-13.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7b71ba34b3af893dc0acbb9d2228a2307da849d38e1c0007bd3d64f456640af",
    ],
)

rpm(
    name = "numad-0__0.5-26.20150602git.el8.aarch64",
    sha256 = "5b580f1a1c2193384a7c4c5171200d1e6f4ca6a19e6a01a327a75d03db916484",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/numad-0.5-26.20150602git.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5b580f1a1c2193384a7c4c5171200d1e6f4ca6a19e6a01a327a75d03db916484",
    ],
)

rpm(
    name = "numad-0__0.5-26.20150602git.el8.x86_64",
    sha256 = "5d975c08273b1629683275c32f16e52ca8e37e6836598e211092c915d38878bf",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/numad-0.5-26.20150602git.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5d975c08273b1629683275c32f16e52ca8e37e6836598e211092c915d38878bf",
    ],
)

rpm(
    name = "openldap-0__2.4.46-18.el8.aarch64",
    sha256 = "254200cc7c35fefbeab3de24c36f94dec10f913ea2199b6d6c769f0fc8a10546",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/openldap-2.4.46-18.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/254200cc7c35fefbeab3de24c36f94dec10f913ea2199b6d6c769f0fc8a10546",
    ],
)

rpm(
    name = "openldap-0__2.4.46-18.el8.x86_64",
    sha256 = "95327d6c83a370a12c125767403496435d20a94b70ee395eabfc356270d2ada9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/openldap-2.4.46-18.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95327d6c83a370a12c125767403496435d20a94b70ee395eabfc356270d2ada9",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-4.el8.aarch64",
    sha256 = "032e8c0576f2743234369ed3a9d682e1b4467e27587a43fd427d2b5b5949e08a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/openssl-libs-1.1.1k-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/032e8c0576f2743234369ed3a9d682e1b4467e27587a43fd427d2b5b5949e08a",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-4.el8.x86_64",
    sha256 = "7bf8f1f49b71019e960f5d5e93f8b0a069ec2743b05ed6e1344bdb4af2dade39",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/openssl-libs-1.1.1k-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7bf8f1f49b71019e960f5d5e93f8b0a069ec2743b05ed6e1344bdb4af2dade39",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-1.el8.aarch64",
    sha256 = "cfee10a5ca5613896a4e84716aa393094fd97c09f2c585c9aa921e6063783867",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/p11-kit-0.23.22-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cfee10a5ca5613896a4e84716aa393094fd97c09f2c585c9aa921e6063783867",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-1.el8.x86_64",
    sha256 = "6a67c8721fe24af25ec56c6aae956a190d8463e46efed45adfbbd800086550c7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/p11-kit-0.23.22-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a67c8721fe24af25ec56c6aae956a190d8463e46efed45adfbbd800086550c7",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-1.el8.aarch64",
    sha256 = "3fc181bf0f076fef283fdb63d36e7b84930c8822fa67dff6e1ccea9987d6dbf3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.23.22-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3fc181bf0f076fef283fdb63d36e7b84930c8822fa67dff6e1ccea9987d6dbf3",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-1.el8.x86_64",
    sha256 = "d218619a4859e002fe677703bc1767986314cd196ae2ac397ed057f3bec36516",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.23.22-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d218619a4859e002fe677703bc1767986314cd196ae2ac397ed057f3bec36516",
    ],
)

rpm(
    name = "pam-0__1.3.1-15.el8.aarch64",
    sha256 = "a33349c435ef9b8348864e5b8f09ed050d0b7a79fb2db5a88b2f7a5d869231d7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pam-1.3.1-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a33349c435ef9b8348864e5b8f09ed050d0b7a79fb2db5a88b2f7a5d869231d7",
    ],
)

rpm(
    name = "pam-0__1.3.1-15.el8.x86_64",
    sha256 = "a0096af833462f915fe6474f7f85324992f641ea7ebeccca1f666815a4afad19",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pam-1.3.1-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0096af833462f915fe6474f7f85324992f641ea7ebeccca1f666815a4afad19",
    ],
)

rpm(
    name = "parted-0__3.2-39.el8.x86_64",
    sha256 = "2a9f8558c6c640d8f035004f3a9e607f6941e028785da562f01b61a142b5e282",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/parted-3.2-39.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2a9f8558c6c640d8f035004f3a9e607f6941e028785da562f01b61a142b5e282",
    ],
)

rpm(
    name = "pcre-0__8.42-6.el8.aarch64",
    sha256 = "5591faa4f51dc97067292938883b771d75ec2b3a749ec956eddc0408e689c369",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pcre-8.42-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5591faa4f51dc97067292938883b771d75ec2b3a749ec956eddc0408e689c369",
    ],
)

rpm(
    name = "pcre-0__8.42-6.el8.x86_64",
    sha256 = "876e9e99b0e50cb2752499045bafa903dd29e5c491d112daacef1ae16f614dad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pcre-8.42-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/876e9e99b0e50cb2752499045bafa903dd29e5c491d112daacef1ae16f614dad",
    ],
)

rpm(
    name = "pcre2-0__10.32-2.el8.aarch64",
    sha256 = "3a386eca4550def1fef05213ddc8fe082e589a2fe2898f634265fbe8fe828296",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pcre2-10.32-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3a386eca4550def1fef05213ddc8fe082e589a2fe2898f634265fbe8fe828296",
    ],
)

rpm(
    name = "pcre2-0__10.32-2.el8.x86_64",
    sha256 = "fb29d2bd46a98affd617bbb243bb117ebbb3d074a6455036abb2aa5b507cce62",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pcre2-10.32-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb29d2bd46a98affd617bbb243bb117ebbb3d074a6455036abb2aa5b507cce62",
    ],
)

rpm(
    name = "pixman-0__0.38.4-1.el8.aarch64",
    sha256 = "9886953d4bc5b03f26b5c3164ce5b5fd86e9f80cf6358b91dd00f870f86052fe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/pixman-0.38.4-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9886953d4bc5b03f26b5c3164ce5b5fd86e9f80cf6358b91dd00f870f86052fe",
    ],
)

rpm(
    name = "pixman-0__0.38.4-1.el8.x86_64",
    sha256 = "ddbbf3a8191dbc1a9fcb67ccf9cea0d34dbe9bbb74780e1359933cd03ee24451",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/pixman-0.38.4-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ddbbf3a8191dbc1a9fcb67ccf9cea0d34dbe9bbb74780e1359933cd03ee24451",
    ],
)

rpm(
    name = "pkgconf-0__1.4.2-1.el8.aarch64",
    sha256 = "9a2c046a45d46e681f417f3b438d4bb5a21e1b93deacb59d906b8aa08a7535ad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pkgconf-1.4.2-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9a2c046a45d46e681f417f3b438d4bb5a21e1b93deacb59d906b8aa08a7535ad",
    ],
)

rpm(
    name = "pkgconf-0__1.4.2-1.el8.x86_64",
    sha256 = "dd08de48d25573f0a8492cf858ce8c37abb10eb560975d9df0e45a7f91b3b41d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pkgconf-1.4.2-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd08de48d25573f0a8492cf858ce8c37abb10eb560975d9df0e45a7f91b3b41d",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.4.2-1.el8.aarch64",
    sha256 = "56187f25e8ae7c2a5ce228d13c6e93b9c6a701960d61dff8ad720a8879b6059e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pkgconf-m4-1.4.2-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/56187f25e8ae7c2a5ce228d13c6e93b9c6a701960d61dff8ad720a8879b6059e",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.4.2-1.el8.x86_64",
    sha256 = "56187f25e8ae7c2a5ce228d13c6e93b9c6a701960d61dff8ad720a8879b6059e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pkgconf-m4-1.4.2-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/56187f25e8ae7c2a5ce228d13c6e93b9c6a701960d61dff8ad720a8879b6059e",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.4.2-1.el8.aarch64",
    sha256 = "aadca7b635ac2b30c3463a4edfe38eaee2c6064181cb090694619186747f3950",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pkgconf-pkg-config-1.4.2-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aadca7b635ac2b30c3463a4edfe38eaee2c6064181cb090694619186747f3950",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.4.2-1.el8.x86_64",
    sha256 = "bf5319e42dbe96c24cd64c974b17f422847cc658c4461d9d61cfe76ad76e9c67",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pkgconf-pkg-config-1.4.2-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bf5319e42dbe96c24cd64c974b17f422847cc658c4461d9d61cfe76ad76e9c67",
    ],
)

rpm(
    name = "platform-python-0__3.6.8-44.el8.aarch64",
    sha256 = "2e17d00c66afe63e8c60d6879163e1856ce02b158f964ae56f3f57c5e12ddbbc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-3.6.8-44.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2e17d00c66afe63e8c60d6879163e1856ce02b158f964ae56f3f57c5e12ddbbc",
    ],
)

rpm(
    name = "platform-python-0__3.6.8-44.el8.x86_64",
    sha256 = "929f041d830b153b1ed38c08d8eaa698a2034705e065c1dc9e3a9b95de36670a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-3.6.8-44.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/929f041d830b153b1ed38c08d8eaa698a2034705e065c1dc9e3a9b95de36670a",
    ],
)

rpm(
    name = "platform-python-pip-0__9.0.3-22.el8.aarch64",
    sha256 = "f66c6d22a96febc3907247a6350097cceeaf77abcb628574052dfdb1a4411607",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-pip-9.0.3-22.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f66c6d22a96febc3907247a6350097cceeaf77abcb628574052dfdb1a4411607",
    ],
)

rpm(
    name = "platform-python-pip-0__9.0.3-22.el8.x86_64",
    sha256 = "f66c6d22a96febc3907247a6350097cceeaf77abcb628574052dfdb1a4411607",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-pip-9.0.3-22.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f66c6d22a96febc3907247a6350097cceeaf77abcb628574052dfdb1a4411607",
    ],
)

rpm(
    name = "platform-python-setuptools-0__39.2.0-6.el8.aarch64",
    sha256 = "946ba273a3a3b6fdf140f3c03112918c0a556a5871c477f5dbbb98600e6ca557",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-setuptools-39.2.0-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/946ba273a3a3b6fdf140f3c03112918c0a556a5871c477f5dbbb98600e6ca557",
    ],
)

rpm(
    name = "platform-python-setuptools-0__39.2.0-6.el8.x86_64",
    sha256 = "946ba273a3a3b6fdf140f3c03112918c0a556a5871c477f5dbbb98600e6ca557",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-setuptools-39.2.0-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/946ba273a3a3b6fdf140f3c03112918c0a556a5871c477f5dbbb98600e6ca557",
    ],
)

rpm(
    name = "policycoreutils-0__2.9-16.el8.aarch64",
    sha256 = "d724e022491864e8416e900b271031660a54c855d53021e99dc984f9b6c92e5f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/policycoreutils-2.9-16.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d724e022491864e8416e900b271031660a54c855d53021e99dc984f9b6c92e5f",
    ],
)

rpm(
    name = "policycoreutils-0__2.9-16.el8.x86_64",
    sha256 = "28ccebf9ca45069bafa539a0cf662b38164a22c52e9b99846a7fe3d4f3e9a8bd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/policycoreutils-2.9-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/28ccebf9ca45069bafa539a0cf662b38164a22c52e9b99846a7fe3d4f3e9a8bd",
    ],
)

rpm(
    name = "polkit-0__0.115-12.el8.aarch64",
    sha256 = "cbd709de63c28a95b78bb32e8da27cf062a2008a47c8d799a8d8bb82a00a33e3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-0.115-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cbd709de63c28a95b78bb32e8da27cf062a2008a47c8d799a8d8bb82a00a33e3",
    ],
)

rpm(
    name = "polkit-0__0.115-12.el8.x86_64",
    sha256 = "df82da310e172a5b40116b47b87e39cdf15b9a68b2f86d5b251203569c5d3c10",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-0.115-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df82da310e172a5b40116b47b87e39cdf15b9a68b2f86d5b251203569c5d3c10",
    ],
)

rpm(
    name = "polkit-libs-0__0.115-12.el8.aarch64",
    sha256 = "5de1ed82200ffe2d2fe91b0bf8362a6a7ff12d2f703db4eb63f6f162e510263b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.115-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5de1ed82200ffe2d2fe91b0bf8362a6a7ff12d2f703db4eb63f6f162e510263b",
    ],
)

rpm(
    name = "polkit-libs-0__0.115-12.el8.x86_64",
    sha256 = "07fbc8d163a0c526f5a6a4851c17dbc440011c25b164fe022f391ebeacfc2ebe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.115-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/07fbc8d163a0c526f5a6a4851c17dbc440011c25b164fe022f391ebeacfc2ebe",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-12.el8.aarch64",
    sha256 = "d25d562fe77f391458903ebf0d9078b6d38af6d9ced39d902b9afc7e717d2234",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-pkla-compat-0.1-12.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d25d562fe77f391458903ebf0d9078b6d38af6d9ced39d902b9afc7e717d2234",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-12.el8.x86_64",
    sha256 = "e7ee4b6d6456cb7da0332f5a6fb8a7c47df977bcf616f12f0455413765367e89",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-pkla-compat-0.1-12.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e7ee4b6d6456cb7da0332f5a6fb8a7c47df977bcf616f12f0455413765367e89",
    ],
)

rpm(
    name = "popt-0__1.18-1.el8.aarch64",
    sha256 = "2596d6cba62bf9594e4fbb07df31e2459eb6fca8e479fd0be2b32c7561e9ad95",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/popt-1.18-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2596d6cba62bf9594e4fbb07df31e2459eb6fca8e479fd0be2b32c7561e9ad95",
    ],
)

rpm(
    name = "popt-0__1.18-1.el8.x86_64",
    sha256 = "3fc009f00388e66befab79be548ff3c7aa80ca70bd7f183d22f59137d8e2c2ae",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/popt-1.18-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3fc009f00388e66befab79be548ff3c7aa80ca70bd7f183d22f59137d8e2c2ae",
    ],
)

rpm(
    name = "procps-ng-0__3.3.15-6.el8.aarch64",
    sha256 = "dda0f9ad611135e6bee3459f183292cb1364b6c09795ead62cfe402426482212",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.15-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dda0f9ad611135e6bee3459f183292cb1364b6c09795ead62cfe402426482212",
    ],
)

rpm(
    name = "procps-ng-0__3.3.15-6.el8.x86_64",
    sha256 = "f5e5f477118224715f12a7151a5effcb6eda892898b5a176e1bde98b03ba7b77",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.15-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f5e5f477118224715f12a7151a5effcb6eda892898b5a176e1bde98b03ba7b77",
    ],
)

rpm(
    name = "psmisc-0__23.1-5.el8.x86_64",
    sha256 = "9d433d8c058e59c891c0852b95b3b87795ea30a85889c77ba0b12f965517d626",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/psmisc-23.1-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d433d8c058e59c891c0852b95b3b87795ea30a85889c77ba0b12f965517d626",
    ],
)

rpm(
    name = "python3-libs-0__3.6.8-44.el8.aarch64",
    sha256 = "8309837ef6f969aa176435335d10675ef367ea5a39684131d4e6e53b9d4d22ad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-libs-3.6.8-44.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8309837ef6f969aa176435335d10675ef367ea5a39684131d4e6e53b9d4d22ad",
    ],
)

rpm(
    name = "python3-libs-0__3.6.8-44.el8.x86_64",
    sha256 = "77adb45ccd96618ac86dbbf61d9f910910a7ac31a16a1ecb20d901e9388438b6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-libs-3.6.8-44.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/77adb45ccd96618ac86dbbf61d9f910910a7ac31a16a1ecb20d901e9388438b6",
    ],
)

rpm(
    name = "python3-pip-0__9.0.3-22.el8.aarch64",
    sha256 = "ba83ca7667c98d265da7334a3ef7f786fbb48c85e32cdec11979778594750953",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/python3-pip-9.0.3-22.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ba83ca7667c98d265da7334a3ef7f786fbb48c85e32cdec11979778594750953",
    ],
)

rpm(
    name = "python3-pip-0__9.0.3-22.el8.x86_64",
    sha256 = "ba83ca7667c98d265da7334a3ef7f786fbb48c85e32cdec11979778594750953",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/python3-pip-9.0.3-22.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ba83ca7667c98d265da7334a3ef7f786fbb48c85e32cdec11979778594750953",
    ],
)

rpm(
    name = "python3-pip-wheel-0__9.0.3-22.el8.aarch64",
    sha256 = "772093492e290af496c3c8d4cf1d83d3288af49c4f0eb550f9c2489f96ecd89d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-9.0.3-22.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/772093492e290af496c3c8d4cf1d83d3288af49c4f0eb550f9c2489f96ecd89d",
    ],
)

rpm(
    name = "python3-pip-wheel-0__9.0.3-22.el8.x86_64",
    sha256 = "772093492e290af496c3c8d4cf1d83d3288af49c4f0eb550f9c2489f96ecd89d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-9.0.3-22.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/772093492e290af496c3c8d4cf1d83d3288af49c4f0eb550f9c2489f96ecd89d",
    ],
)

rpm(
    name = "python3-setuptools-0__39.2.0-6.el8.aarch64",
    sha256 = "c6f27b6e01d80e756408e3c1451e4af00e7d02da0aa24402644c0785118753fe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-setuptools-39.2.0-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c6f27b6e01d80e756408e3c1451e4af00e7d02da0aa24402644c0785118753fe",
    ],
)

rpm(
    name = "python3-setuptools-0__39.2.0-6.el8.x86_64",
    sha256 = "c6f27b6e01d80e756408e3c1451e4af00e7d02da0aa24402644c0785118753fe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-setuptools-39.2.0-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c6f27b6e01d80e756408e3c1451e4af00e7d02da0aa24402644c0785118753fe",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__39.2.0-6.el8.aarch64",
    sha256 = "b19bd4f106ce301ee21c860183cc1c2ef9c09bdf495059bdf16e8d8ccc71bbe8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-39.2.0-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b19bd4f106ce301ee21c860183cc1c2ef9c09bdf495059bdf16e8d8ccc71bbe8",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__39.2.0-6.el8.x86_64",
    sha256 = "b19bd4f106ce301ee21c860183cc1c2ef9c09bdf495059bdf16e8d8ccc71bbe8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-39.2.0-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b19bd4f106ce301ee21c860183cc1c2ef9c09bdf495059bdf16e8d8ccc71bbe8",
    ],
)

rpm(
    name = "python36-0__3.6.8-38.module_el8.5.0__plus__895__plus__a459eca8.aarch64",
    sha256 = "ab1d26bddf3f97decf17ac4a12c545add80be07bba1d7a1519481df24151e390",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/python36-3.6.8-38.module_el8.5.0+895+a459eca8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab1d26bddf3f97decf17ac4a12c545add80be07bba1d7a1519481df24151e390",
    ],
)

rpm(
    name = "python36-0__3.6.8-38.module_el8.5.0__plus__895__plus__a459eca8.x86_64",
    sha256 = "002b3672de2744c3f97ad8776d012952c058f9213a0cf8e01f7f9b8651b3e6af",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/python36-3.6.8-38.module_el8.5.0+895+a459eca8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/002b3672de2744c3f97ad8776d012952c058f9213a0cf8e01f7f9b8651b3e6af",
    ],
)

rpm(
    name = "qemu-img-15__6.0.0-33.el8s.aarch64",
    sha256 = "96f1a3e08859469530db354b751eda12023e0be631e8ed77bf2924ed4f673d5f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/q/qemu-img-6.0.0-33.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/96f1a3e08859469530db354b751eda12023e0be631e8ed77bf2924ed4f673d5f",
    ],
)

rpm(
    name = "qemu-img-15__6.0.0-33.el8s.x86_64",
    sha256 = "be9ca3aee35feb794292a81f98e33130ed33711b52c5971d7d4e3b1205ca2d76",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-img-6.0.0-33.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be9ca3aee35feb794292a81f98e33130ed33711b52c5971d7d4e3b1205ca2d76",
    ],
)

rpm(
    name = "qemu-kvm-common-15__6.0.0-33.el8s.aarch64",
    sha256 = "c6b93ad2788e16d780a856f29bd88704af3ec1103096cddcf2a21ba05aba3b1c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/q/qemu-kvm-common-6.0.0-33.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c6b93ad2788e16d780a856f29bd88704af3ec1103096cddcf2a21ba05aba3b1c",
    ],
)

rpm(
    name = "qemu-kvm-common-15__6.0.0-33.el8s.x86_64",
    sha256 = "98e9958db16c59f39225960f36273b31eadf17e2e3cd65be546ca33ab3612a83",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-kvm-common-6.0.0-33.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/98e9958db16c59f39225960f36273b31eadf17e2e3cd65be546ca33ab3612a83",
    ],
)

rpm(
    name = "qemu-kvm-core-15__6.0.0-33.el8s.aarch64",
    sha256 = "6b385d5a34812c2e788fb80991fb361604a90079cb7196a6e32de6a3a37daad1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/q/qemu-kvm-core-6.0.0-33.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6b385d5a34812c2e788fb80991fb361604a90079cb7196a6e32de6a3a37daad1",
    ],
)

rpm(
    name = "qemu-kvm-core-15__6.0.0-33.el8s.x86_64",
    sha256 = "88c261aa775f838b64fc9cd8b75c00db3d9a3cac48e65bb84b39cdbf0d9b837a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-kvm-core-6.0.0-33.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/88c261aa775f838b64fc9cd8b75c00db3d9a3cac48e65bb84b39cdbf0d9b837a",
    ],
)

rpm(
    name = "qemu-kvm-hw-usbredir-15__6.0.0-33.el8s.x86_64",
    sha256 = "3fa572098bab53d3f3b3887e8406d89dede24e1d0f836d10872a007aaac9a18a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-kvm-hw-usbredir-6.0.0-33.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3fa572098bab53d3f3b3887e8406d89dede24e1d0f836d10872a007aaac9a18a",
    ],
)

rpm(
    name = "readline-0__7.0-10.el8.aarch64",
    sha256 = "ef74f2c65ed0e38dd021177d6e59fcdf7fb8de8929b7544b7a6f0709eff6562c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/readline-7.0-10.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ef74f2c65ed0e38dd021177d6e59fcdf7fb8de8929b7544b7a6f0709eff6562c",
    ],
)

rpm(
    name = "readline-0__7.0-10.el8.x86_64",
    sha256 = "fea868a7d82a7b6f392260ed4afb472dc4428fd71eab1456319f423a845b5084",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/readline-7.0-10.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fea868a7d82a7b6f392260ed4afb472dc4428fd71eab1456319f423a845b5084",
    ],
)

rpm(
    name = "rpm-0__4.14.3-19.el8.aarch64",
    sha256 = "cb738068451c94107910ce4c49e81dea7333cc2fff9c65ae2936bb8c3d6119aa",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-4.14.3-19.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb738068451c94107910ce4c49e81dea7333cc2fff9c65ae2936bb8c3d6119aa",
    ],
)

rpm(
    name = "rpm-0__4.14.3-19.el8.x86_64",
    sha256 = "3f10ff18b8d74f301ef888f2f3c5bbeca3753ca5b63a1c06346b342c1d337d43",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-4.14.3-19.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f10ff18b8d74f301ef888f2f3c5bbeca3753ca5b63a1c06346b342c1d337d43",
    ],
)

rpm(
    name = "rpm-libs-0__4.14.3-19.el8.aarch64",
    sha256 = "7a081cebd2477835d68e88bebf1f08635526d2700e340dd019785396d3c0afa4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.14.3-19.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7a081cebd2477835d68e88bebf1f08635526d2700e340dd019785396d3c0afa4",
    ],
)

rpm(
    name = "rpm-libs-0__4.14.3-19.el8.x86_64",
    sha256 = "8e47e605b8a904523a619578367bbd53ff8db79e4c41bbeabbeaf9c7a5f0e002",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.14.3-19.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e47e605b8a904523a619578367bbd53ff8db79e4c41bbeabbeaf9c7a5f0e002",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-19.el8.aarch64",
    sha256 = "90df8e203f38e5306543e7e437c71bbd5726fbb230560db808d05928f016b96c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.14.3-19.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/90df8e203f38e5306543e7e437c71bbd5726fbb230560db808d05928f016b96c",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-19.el8.x86_64",
    sha256 = "2301bc2b7aff1b8c35343ce9cbb2fed94d095a0ebf5337886ec0b03d3005776f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.14.3-19.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2301bc2b7aff1b8c35343ce9cbb2fed94d095a0ebf5337886ec0b03d3005776f",
    ],
)

rpm(
    name = "scrub-0__2.5.2-16.el8.x86_64",
    sha256 = "3d269d1d609637a1fcd72b3e789191292aea31aac7af48be986a0c42fd7e2f14",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/scrub-2.5.2-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3d269d1d609637a1fcd72b3e789191292aea31aac7af48be986a0c42fd7e2f14",
    ],
)

rpm(
    name = "seabios-0__1.14.0-1.el8s.x86_64",
    sha256 = "468be89248e2b4cf655832f7b156e8ce90d726f0203d7c729293ce708a16cc7f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/seabios-1.14.0-1.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/468be89248e2b4cf655832f7b156e8ce90d726f0203d7c729293ce708a16cc7f",
    ],
)

rpm(
    name = "seabios-bin-0__1.14.0-1.el8s.x86_64",
    sha256 = "89033ae80928e60ff0377703208e08fb57c31b4e6d31697936817a3125180abb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/seabios-bin-1.14.0-1.el8s.noarch.rpm",
        "https://storage.googleapis.com/builddeps/89033ae80928e60ff0377703208e08fb57c31b4e6d31697936817a3125180abb",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.14.0-1.el8s.x86_64",
    sha256 = "558868cab91b079c7a33e15602b715d38b31966a6f12f953b2b67ec8cad9ccf1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/seavgabios-bin-1.14.0-1.el8s.noarch.rpm",
        "https://storage.googleapis.com/builddeps/558868cab91b079c7a33e15602b715d38b31966a6f12f953b2b67ec8cad9ccf1",
    ],
)

rpm(
    name = "sed-0__4.5-5.el8.aarch64",
    sha256 = "806550c684c46a58a455953223fafbacc343e35e488d436bf963844944a33861",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sed-4.5-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/806550c684c46a58a455953223fafbacc343e35e488d436bf963844944a33861",
    ],
)

rpm(
    name = "sed-0__4.5-5.el8.x86_64",
    sha256 = "5a09d6d967d12580c7e6ab92db35bcafd3426d6121ec60c78f54e3cd4961cd26",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sed-4.5-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a09d6d967d12580c7e6ab92db35bcafd3426d6121ec60c78f54e3cd4961cd26",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.3-83.el8.aarch64",
    sha256 = "1669e0faaab0220137ce44f69961a8bd640016a4d89176d77f616a176c9d73a6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-3.14.3-83.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1669e0faaab0220137ce44f69961a8bd640016a4d89176d77f616a176c9d73a6",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.3-83.el8.x86_64",
    sha256 = "1669e0faaab0220137ce44f69961a8bd640016a4d89176d77f616a176c9d73a6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-3.14.3-83.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1669e0faaab0220137ce44f69961a8bd640016a4d89176d77f616a176c9d73a6",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-83.el8.aarch64",
    sha256 = "a024999718a55804cc47bb47427ea0070a9cb093ccc19a0337c3b1b8d0772464",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-3.14.3-83.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a024999718a55804cc47bb47427ea0070a9cb093ccc19a0337c3b1b8d0772464",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-83.el8.x86_64",
    sha256 = "a024999718a55804cc47bb47427ea0070a9cb093ccc19a0337c3b1b8d0772464",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-3.14.3-83.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a024999718a55804cc47bb47427ea0070a9cb093ccc19a0337c3b1b8d0772464",
    ],
)

rpm(
    name = "setup-0__2.12.2-6.el8.aarch64",
    sha256 = "9e540fe1fcf866ba1e738e012eef5459d34cca30385df73973e6fc7c6eadb55f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/setup-2.12.2-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9e540fe1fcf866ba1e738e012eef5459d34cca30385df73973e6fc7c6eadb55f",
    ],
)

rpm(
    name = "setup-0__2.12.2-6.el8.x86_64",
    sha256 = "9e540fe1fcf866ba1e738e012eef5459d34cca30385df73973e6fc7c6eadb55f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/setup-2.12.2-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9e540fe1fcf866ba1e738e012eef5459d34cca30385df73973e6fc7c6eadb55f",
    ],
)

rpm(
    name = "sgabios-bin-1__0.20170427git-3.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "79675eae8221b4abd2ef195328fc9b2c27b7f6e901ed65ac11b93f0637033b2f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/sgabios-bin-0.20170427git-3.module_el8.6.0+983+a7505f3f.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79675eae8221b4abd2ef195328fc9b2c27b7f6e901ed65ac11b93f0637033b2f",
    ],
)

rpm(
    name = "shadow-utils-2__4.6-15.el8.aarch64",
    sha256 = "362ea1e91564c4d9beb80f948926273fbff05dd9766d6593116390e75bcf92f1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.6-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/362ea1e91564c4d9beb80f948926273fbff05dd9766d6593116390e75bcf92f1",
    ],
)

rpm(
    name = "shadow-utils-2__4.6-15.el8.x86_64",
    sha256 = "297be55a53f3fc1bead0e12a7ac44a7970aaf61feedb5a4488e416aa8367f341",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.6-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/297be55a53f3fc1bead0e12a7ac44a7970aaf61feedb5a4488e416aa8367f341",
    ],
)

rpm(
    name = "snappy-0__1.1.8-3.el8.aarch64",
    sha256 = "4731985b22fc7b733ff89be6c1423396f27c94a78bb09fc89be5c2200bee893c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/snappy-1.1.8-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4731985b22fc7b733ff89be6c1423396f27c94a78bb09fc89be5c2200bee893c",
    ],
)

rpm(
    name = "snappy-0__1.1.8-3.el8.x86_64",
    sha256 = "839c62cd7fc7e152decded6f28c80b5f7b8f34a5e319057867b38b26512cee67",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/snappy-1.1.8-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/839c62cd7fc7e152decded6f28c80b5f7b8f34a5e319057867b38b26512cee67",
    ],
)

rpm(
    name = "sqlite-libs-0__3.26.0-15.el8.aarch64",
    sha256 = "b3a0c27117c927795b1a3a1ef2c08c857a88199bcfad5603cd2303c9519671a4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.26.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b3a0c27117c927795b1a3a1ef2c08c857a88199bcfad5603cd2303c9519671a4",
    ],
)

rpm(
    name = "sqlite-libs-0__3.26.0-15.el8.x86_64",
    sha256 = "46d01b59aba3aaccaf32731ada7323f62ae848fe17ff2bd020589f282b3ccac3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.26.0-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46d01b59aba3aaccaf32731ada7323f62ae848fe17ff2bd020589f282b3ccac3",
    ],
)

rpm(
    name = "squashfs-tools-0__4.3-20.el8.x86_64",
    sha256 = "956da9a94f3f2331df649b8351ebeb0c102702486ff447e8252e7af3b96ab414",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/squashfs-tools-4.3-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/956da9a94f3f2331df649b8351ebeb0c102702486ff447e8252e7af3b96ab414",
    ],
)

rpm(
    name = "sssd-client-0__2.5.2-2.el8_5.1.aarch64",
    sha256 = "f65ac248e7d480221f082758e8d00a8fad79ac3262b1e097f557c4bce17b64f3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sssd-client-2.5.2-2.el8_5.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f65ac248e7d480221f082758e8d00a8fad79ac3262b1e097f557c4bce17b64f3",
    ],
)

rpm(
    name = "sssd-client-0__2.5.2-2.el8_5.1.x86_64",
    sha256 = "9e375182ef23ab5696778944f5d450d261aa15b52dfd1748f204d38b3fe00206",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sssd-client-2.5.2-2.el8_5.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9e375182ef23ab5696778944f5d450d261aa15b52dfd1748f204d38b3fe00206",
    ],
)

rpm(
    name = "supermin-0__5.2.1-1.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "33f3a33c9ecef931b68ae943bfd5939db6d79d6813f18d05f780b87647b40f1a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/supermin-5.2.1-1.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/33f3a33c9ecef931b68ae943bfd5939db6d79d6813f18d05f780b87647b40f1a",
    ],
)

rpm(
    name = "swtpm-0__0.6.0-2.20210607gitea627b3.module_el8.6.0__plus__983__plus__a7505f3f.aarch64",
    sha256 = "3b050da52c5e16b9052b1978cb4a1b6af83008cb37897ed20ec34ab1a1f545ef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-0.6.0-2.20210607gitea627b3.module_el8.6.0+983+a7505f3f.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3b050da52c5e16b9052b1978cb4a1b6af83008cb37897ed20ec34ab1a1f545ef",
    ],
)

rpm(
    name = "swtpm-0__0.6.0-2.20210607gitea627b3.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "f6b92bef838ad97ef33ab42fbe69772344cf8bcd286a342fcd890da6da902657",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-0.6.0-2.20210607gitea627b3.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f6b92bef838ad97ef33ab42fbe69772344cf8bcd286a342fcd890da6da902657",
    ],
)

rpm(
    name = "swtpm-libs-0__0.6.0-2.20210607gitea627b3.module_el8.6.0__plus__983__plus__a7505f3f.aarch64",
    sha256 = "60c9b37b804ae32c79a7f3760471ffbf517d8a4cee6868ab1e85c233ceca3221",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.6.0-2.20210607gitea627b3.module_el8.6.0+983+a7505f3f.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/60c9b37b804ae32c79a7f3760471ffbf517d8a4cee6868ab1e85c233ceca3221",
    ],
)

rpm(
    name = "swtpm-libs-0__0.6.0-2.20210607gitea627b3.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "70be15d1d5e62b3337197972559e1bc3f6a37401803e66aad8c806ee6c8f3ccc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.6.0-2.20210607gitea627b3.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/70be15d1d5e62b3337197972559e1bc3f6a37401803e66aad8c806ee6c8f3ccc",
    ],
)

rpm(
    name = "swtpm-tools-0__0.6.0-2.20210607gitea627b3.module_el8.6.0__plus__983__plus__a7505f3f.aarch64",
    sha256 = "18dfa4b1282bee5e96a29eee73e23cff89416a87dc8ab2e2d5197a0445fe9bb6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.6.0-2.20210607gitea627b3.module_el8.6.0+983+a7505f3f.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/18dfa4b1282bee5e96a29eee73e23cff89416a87dc8ab2e2d5197a0445fe9bb6",
    ],
)

rpm(
    name = "swtpm-tools-0__0.6.0-2.20210607gitea627b3.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "0d56e59ea3a81e0528351e2bdd389c197fd35f6849f4b4973f417c2aa648d1d1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.6.0-2.20210607gitea627b3.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0d56e59ea3a81e0528351e2bdd389c197fd35f6849f4b4973f417c2aa648d1d1",
    ],
)

rpm(
    name = "syslinux-0__6.04-5.el8.x86_64",
    sha256 = "33996f2476ed82d68353ac5c6d22c204db4ee76821eefc4c0cc2dafcf44ae16b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-6.04-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/33996f2476ed82d68353ac5c6d22c204db4ee76821eefc4c0cc2dafcf44ae16b",
    ],
)

rpm(
    name = "syslinux-extlinux-0__6.04-5.el8.x86_64",
    sha256 = "28ede201bc3e0a3aae01dfb96d84260c933372b710be37c25d923504ee43acea",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-extlinux-6.04-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/28ede201bc3e0a3aae01dfb96d84260c933372b710be37c25d923504ee43acea",
    ],
)

rpm(
    name = "syslinux-extlinux-nonlinux-0__6.04-5.el8.x86_64",
    sha256 = "32b57460a7ce649954f813033d44a8feba1ab30cd1cf99c0a64f5826d2448167",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-extlinux-nonlinux-6.04-5.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/32b57460a7ce649954f813033d44a8feba1ab30cd1cf99c0a64f5826d2448167",
    ],
)

rpm(
    name = "syslinux-nonlinux-0__6.04-5.el8.x86_64",
    sha256 = "89f2d9a00712110d283de570cd3212c204fdcf78a32cd71e0d6ee660e412941c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-nonlinux-6.04-5.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/89f2d9a00712110d283de570cd3212c204fdcf78a32cd71e0d6ee660e412941c",
    ],
)

rpm(
    name = "systemd-0__239-51.el8.aarch64",
    sha256 = "2610fab54bafd98bc72452f6c2f8e6cab618cda5514b6957881e14b24feab53e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-239-51.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2610fab54bafd98bc72452f6c2f8e6cab618cda5514b6957881e14b24feab53e",
    ],
)

rpm(
    name = "systemd-0__239-51.el8.x86_64",
    sha256 = "91bf1d1b1dc959ee07d22f2d9d9deb27cc2a4dba9c0ffaa9d2e19b6c39922673",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-239-51.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/91bf1d1b1dc959ee07d22f2d9d9deb27cc2a4dba9c0ffaa9d2e19b6c39922673",
    ],
)

rpm(
    name = "systemd-container-0__239-51.el8.aarch64",
    sha256 = "89a21619a80e5b642ebde93f672d61272789e133dbe1c98df027f34bbf4d7468",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-container-239-51.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/89a21619a80e5b642ebde93f672d61272789e133dbe1c98df027f34bbf4d7468",
    ],
)

rpm(
    name = "systemd-container-0__239-51.el8.x86_64",
    sha256 = "b18210020ed8da397ea54f88380e57964ad0ddf08007ed11afe55171aaeefd7a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-container-239-51.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b18210020ed8da397ea54f88380e57964ad0ddf08007ed11afe55171aaeefd7a",
    ],
)

rpm(
    name = "systemd-libs-0__239-51.el8.aarch64",
    sha256 = "6a90be7f128e1d0633ee0229404c47667eb518e990a377a8584b630a0ef75966",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-libs-239-51.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a90be7f128e1d0633ee0229404c47667eb518e990a377a8584b630a0ef75966",
    ],
)

rpm(
    name = "systemd-libs-0__239-51.el8.x86_64",
    sha256 = "95a0cb747a79ebb1552f73f2f4b5f612b7a8c0b3fc56a7496854f2ca491d3c3a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-libs-239-51.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95a0cb747a79ebb1552f73f2f4b5f612b7a8c0b3fc56a7496854f2ca491d3c3a",
    ],
)

rpm(
    name = "systemd-pam-0__239-51.el8.aarch64",
    sha256 = "f67fc47495b7d6ede23f76b2bd24c89f200761355b8c8b748ff1399ad3ecfaac",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-pam-239-51.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f67fc47495b7d6ede23f76b2bd24c89f200761355b8c8b748ff1399ad3ecfaac",
    ],
)

rpm(
    name = "systemd-pam-0__239-51.el8.x86_64",
    sha256 = "0207c8a5fafaddf9deebc9e25262927a1a36aa4077f477960df1e695b8a656a3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-pam-239-51.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0207c8a5fafaddf9deebc9e25262927a1a36aa4077f477960df1e695b8a656a3",
    ],
)

rpm(
    name = "systemd-udev-0__239-51.el8.x86_64",
    sha256 = "4dd7fcf5111a9d51766a06204a5c3e0d63503fa2eeff6b2f2c059c4d37c4ca34",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-udev-239-51.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4dd7fcf5111a9d51766a06204a5c3e0d63503fa2eeff6b2f2c059c4d37c4ca34",
    ],
)

rpm(
    name = "tar-2__1.30-5.el8.aarch64",
    sha256 = "3d527d861793fe3a74b6254540068e8b846e6df20d75754df39904e67f1e569f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/tar-1.30-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3d527d861793fe3a74b6254540068e8b846e6df20d75754df39904e67f1e569f",
    ],
)

rpm(
    name = "tar-2__1.30-5.el8.x86_64",
    sha256 = "ed1f7ab0225df75734034cb2aea426c48c089f2bd476ec66b66af879437c5393",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/tar-1.30-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed1f7ab0225df75734034cb2aea426c48c089f2bd476ec66b66af879437c5393",
    ],
)

rpm(
    name = "tzdata-0__2021e-1.el8.aarch64",
    sha256 = "6ae03d640e42eb1057d2438374025587c108a5a5eef91aa0fbca48c530140b78",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/tzdata-2021e-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6ae03d640e42eb1057d2438374025587c108a5a5eef91aa0fbca48c530140b78",
    ],
)

rpm(
    name = "tzdata-0__2021e-1.el8.x86_64",
    sha256 = "6ae03d640e42eb1057d2438374025587c108a5a5eef91aa0fbca48c530140b78",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/tzdata-2021e-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6ae03d640e42eb1057d2438374025587c108a5a5eef91aa0fbca48c530140b78",
    ],
)

rpm(
    name = "unbound-libs-0__1.7.3-17.el8.aarch64",
    sha256 = "406140d0a2d6fe921875898b24b91376870fb9ab1b1baf7778cff060bbbe0d72",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/unbound-libs-1.7.3-17.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/406140d0a2d6fe921875898b24b91376870fb9ab1b1baf7778cff060bbbe0d72",
    ],
)

rpm(
    name = "unbound-libs-0__1.7.3-17.el8.x86_64",
    sha256 = "9a5380195d24327a8a2e059395d7902f9bc3b771275afe1533702998dc5be364",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/unbound-libs-1.7.3-17.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a5380195d24327a8a2e059395d7902f9bc3b771275afe1533702998dc5be364",
    ],
)

rpm(
    name = "usbredir-0__0.8.0-1.el8.x86_64",
    sha256 = "359290c30476453554d970c0f5360b6039e8b92fb72018a65b7e56b38f260bda",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/usbredir-0.8.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/359290c30476453554d970c0f5360b6039e8b92fb72018a65b7e56b38f260bda",
    ],
)

rpm(
    name = "userspace-rcu-0__0.10.1-4.el8.aarch64",
    sha256 = "c4b53c8f1121938c2c5ae3fabd48b9d8f77c7d26f47a76f5c0eab3fd7f0a6cfc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/userspace-rcu-0.10.1-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c4b53c8f1121938c2c5ae3fabd48b9d8f77c7d26f47a76f5c0eab3fd7f0a6cfc",
    ],
)

rpm(
    name = "userspace-rcu-0__0.10.1-4.el8.x86_64",
    sha256 = "4025900345c5125fd6c10c1780275139f56b63be2bfac10be83628758c225dd0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/userspace-rcu-0.10.1-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4025900345c5125fd6c10c1780275139f56b63be2bfac10be83628758c225dd0",
    ],
)

rpm(
    name = "util-linux-0__2.32.1-28.el8.aarch64",
    sha256 = "1600cd7372ca2682c9bd58de6e783092d6bdb6412c9888e3dd767db92c9b5239",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/util-linux-2.32.1-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1600cd7372ca2682c9bd58de6e783092d6bdb6412c9888e3dd767db92c9b5239",
    ],
)

rpm(
    name = "util-linux-0__2.32.1-28.el8.x86_64",
    sha256 = "8213aa26dbe71291c0cd5969256f3f0bce13ee8e8cd76e3b949c1f6752142471",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/util-linux-2.32.1-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8213aa26dbe71291c0cd5969256f3f0bce13ee8e8cd76e3b949c1f6752142471",
    ],
)

rpm(
    name = "vim-minimal-2__8.0.1763-16.el8_5.2.aarch64",
    sha256 = "c6331cb155e845a1a3a3f804b7d0e80b10629b4e951b6eab8856cf48faba7157",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.0.1763-16.el8_5.2.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c6331cb155e845a1a3a3f804b7d0e80b10629b4e951b6eab8856cf48faba7157",
    ],
)

rpm(
    name = "vim-minimal-2__8.0.1763-16.el8_5.2.x86_64",
    sha256 = "b2e246b9e83c2f7daceef97fe2946e832b6ce396bba90fe7aaba96b585ec7dc9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.0.1763-16.el8_5.2.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2e246b9e83c2f7daceef97fe2946e832b6ce396bba90fe7aaba96b585ec7dc9",
    ],
)

rpm(
    name = "which-0__2.21-16.el8.aarch64",
    sha256 = "81a1147f174921fabcba53f773cc714a4937ae9371fa3687988b145512e51193",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/which-2.21-16.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/81a1147f174921fabcba53f773cc714a4937ae9371fa3687988b145512e51193",
    ],
)

rpm(
    name = "which-0__2.21-16.el8.x86_64",
    sha256 = "0a4bd60fba20ec837a384a52cc56c852f6d01b3b6ec810e3a3d538a42442b937",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/which-2.21-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0a4bd60fba20ec837a384a52cc56c852f6d01b3b6ec810e3a3d538a42442b937",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.28-1.el8.aarch64",
    sha256 = "a2aeabb3962859069a78acc288bc3bffb35485428e162caafec8134f5ce6ca67",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/xkeyboard-config-2.28-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a2aeabb3962859069a78acc288bc3bffb35485428e162caafec8134f5ce6ca67",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.28-1.el8.x86_64",
    sha256 = "a2aeabb3962859069a78acc288bc3bffb35485428e162caafec8134f5ce6ca67",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/xkeyboard-config-2.28-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a2aeabb3962859069a78acc288bc3bffb35485428e162caafec8134f5ce6ca67",
    ],
)

rpm(
    name = "xorriso-0__1.4.8-4.el8.aarch64",
    sha256 = "4280064ab658525b486d7b8c2ca5f87aeef90002361a0925f2819fd7a7909500",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/xorriso-1.4.8-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4280064ab658525b486d7b8c2ca5f87aeef90002361a0925f2819fd7a7909500",
    ],
)

rpm(
    name = "xorriso-0__1.4.8-4.el8.x86_64",
    sha256 = "3a232d848da1ace286efef6c8c9cf0fcfab2c47dd58968ddb6a24718629a6220",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/xorriso-1.4.8-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a232d848da1ace286efef6c8c9cf0fcfab2c47dd58968ddb6a24718629a6220",
    ],
)

rpm(
    name = "xz-0__5.2.4-3.el8.aarch64",
    sha256 = "b9a899e715019e7002600005bcb2a9dd7b089eaef9c55c3764c326d745ad681f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/xz-5.2.4-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b9a899e715019e7002600005bcb2a9dd7b089eaef9c55c3764c326d745ad681f",
    ],
)

rpm(
    name = "xz-0__5.2.4-3.el8.x86_64",
    sha256 = "02f10beaf61212427e0cd57140d050948eea0b533cf432d7bc4c10266c8b33db",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/xz-5.2.4-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/02f10beaf61212427e0cd57140d050948eea0b533cf432d7bc4c10266c8b33db",
    ],
)

rpm(
    name = "xz-libs-0__5.2.4-3.el8.aarch64",
    sha256 = "8f141db26834b1ec60028790b130d00b14b7fda256db0df1e51b7ba8d3d40c7b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/xz-libs-5.2.4-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8f141db26834b1ec60028790b130d00b14b7fda256db0df1e51b7ba8d3d40c7b",
    ],
)

rpm(
    name = "xz-libs-0__5.2.4-3.el8.x86_64",
    sha256 = "61553db2c5d1da168da53ec285de14d00ce91bb02dd902a1688725cf37a7b1a2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/xz-libs-5.2.4-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61553db2c5d1da168da53ec285de14d00ce91bb02dd902a1688725cf37a7b1a2",
    ],
)

rpm(
    name = "yajl-0__2.1.0-10.el8.aarch64",
    sha256 = "255e74b387f5e9b517d82cd00f3b62af88b32054095be91a63b3e5eb5db34939",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/yajl-2.1.0-10.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/255e74b387f5e9b517d82cd00f3b62af88b32054095be91a63b3e5eb5db34939",
    ],
)

rpm(
    name = "yajl-0__2.1.0-10.el8.x86_64",
    sha256 = "a7797aa70d6a35116ec3253523dc91d1b08df44bad7442b94af07bb6c0a661f0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/yajl-2.1.0-10.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a7797aa70d6a35116ec3253523dc91d1b08df44bad7442b94af07bb6c0a661f0",
    ],
)

rpm(
    name = "zlib-0__1.2.11-17.el8.aarch64",
    sha256 = "19223c1996366de6f38c38f5d0163368fbff9c29149bb925ffe8d2eba79b239c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/zlib-1.2.11-17.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19223c1996366de6f38c38f5d0163368fbff9c29149bb925ffe8d2eba79b239c",
    ],
)

rpm(
    name = "zlib-0__1.2.11-17.el8.x86_64",
    sha256 = "a604ffec838794e53b7721e4f113dbd780b5a0765f200df6c41ea19018fa7ea6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/zlib-1.2.11-17.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a604ffec838794e53b7721e4f113dbd780b5a0765f200df6c41ea19018fa7ea6",
    ],
)
