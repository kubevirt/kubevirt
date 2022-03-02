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
    sha256 = "2b1641428dff9018f9e85c0384f03ec6c10660d935b750e3fa1492a281a53b0f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
        "https://storage.googleapis.com/builddeps/2b1641428dff9018f9e85c0384f03ec6c10660d935b750e3fa1492a281a53b0f",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "de69a09dc70417580aabf20a28619bb3ef60d038470c7cf8442fafcf627c21cb",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
        "https://storage.googleapis.com/builddeps/de69a09dc70417580aabf20a28619bb3ef60d038470c7cf8442fafcf627c21cb",
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
    go_version = "1.17.5",
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
    name = "audit-libs-0__3.0.7-1.el8.aarch64",
    sha256 = "498e52ac86e6131ef31c5dde72cf6f442695444f55da33ef9a99401bf3b353ff",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/audit-libs-3.0.7-1.el8.aarch64.rpm"],
)

rpm(
    name = "audit-libs-0__3.0.7-1.el8.x86_64",
    sha256 = "18df4898df9e64a144245c9a180fecf5f87e378afffe90d9488758e1263b2e7b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/audit-libs-3.0.7-1.el8.x86_64.rpm"],
)

rpm(
    name = "augeas-libs-0__1.12.0-7.el8.x86_64",
    sha256 = "672cf6c97f6aa00a0d5a39d20372501d6c6f40ac431083a499d89b7b25c84ba4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/augeas-libs-1.12.0-7.el8.x86_64.rpm"],
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
    name = "bind-export-libs-32__9.11.36-2.el8.x86_64",
    sha256 = "bdc472abbb50eb652e78290b26756ad06ac2ea52e36f8acb3c4fc09bc7a58067",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bind-export-libs-9.11.36-2.el8.x86_64.rpm"],
)

rpm(
    name = "binutils-0__2.30-113.el8.aarch64",
    sha256 = "c4fcd3a97572a10f58a11fbe30d48736feb706dc5f4aa5645203cd5d1a85921a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/binutils-2.30-113.el8.aarch64.rpm"],
)

rpm(
    name = "binutils-0__2.30-113.el8.x86_64",
    sha256 = "7ea573580f796cf4ca5e8a7e1a1850dbf7249c2fcfce999f9794e7b8583407d7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/binutils-2.30-113.el8.x86_64.rpm"],
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
    name = "centos-gpg-keys-1__8-4.el8.aarch64",
    sha256 = "03e8d06d84b6c983524bb4e2c83559f7a28f13a668a3ff97782015a3c2ad36e5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-8-4.el8.noarch.rpm"],
)

rpm(
    name = "centos-gpg-keys-1__8-4.el8.x86_64",
    sha256 = "03e8d06d84b6c983524bb4e2c83559f7a28f13a668a3ff97782015a3c2ad36e5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-8-4.el8.noarch.rpm"],
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
    name = "centos-stream-repos-0__8-4.el8.aarch64",
    sha256 = "6421059eda903b4494c4b01689193d81a3161b5e7f04697f810e96cb901b970e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-8-4.el8.noarch.rpm"],
)

rpm(
    name = "centos-stream-repos-0__8-4.el8.x86_64",
    sha256 = "6421059eda903b4494c4b01689193d81a3161b5e7f04697f810e96cb901b970e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-8-4.el8.noarch.rpm"],
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
    name = "cpp-0__8.5.0-10.el8.aarch64",
    sha256 = "3ec614e10d86b81e246bea2e7fdf0fc204d4509380588ac32efe47a108138c0d",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/cpp-8.5.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "cpp-0__8.5.0-10.el8.x86_64",
    sha256 = "85a491bdcbe175ca8e7d98bca8f6f7b200f8e04ad9dcbda150aa0743d3534a3b",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/cpp-8.5.0-10.el8.x86_64.rpm"],
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
    name = "crypto-policies-0__20211116-1.gitae470d6.el8.aarch64",
    sha256 = "8fb69892af346bacf18e8f8e7e8098e09c6ef9547abab9c39f7e729db06c3d1e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/crypto-policies-20211116-1.gitae470d6.el8.noarch.rpm"],
)

rpm(
    name = "crypto-policies-0__20211116-1.gitae470d6.el8.x86_64",
    sha256 = "8fb69892af346bacf18e8f8e7e8098e09c6ef9547abab9c39f7e729db06c3d1e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/crypto-policies-20211116-1.gitae470d6.el8.noarch.rpm"],
)

rpm(
    name = "cryptsetup-0__2.3.7-1.el8.x86_64",
    sha256 = "b7d6da68eb7d5fbb19574f1565f9ddfe0e2d8818015aa4ef233a32f81600319f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-2.3.7-1.el8.x86_64.rpm"],
)

rpm(
    name = "cryptsetup-libs-0__2.3.7-1.el8.aarch64",
    sha256 = "c8cee9719acdcaed39753c546fceda1e93fdf07f67290985f65f501b6ce03e9e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cryptsetup-libs-2.3.7-1.el8.aarch64.rpm"],
)

rpm(
    name = "cryptsetup-libs-0__2.3.7-1.el8.x86_64",
    sha256 = "e81e3682cbc99cc9d42d070f73ded56cf0ea95666bae82c38d3b36afed7af12c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-libs-2.3.7-1.el8.x86_64.rpm"],
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
    name = "daxctl-libs-0__71.1-3.el8.x86_64",
    sha256 = "772d44fa92d450ce1733a27efc88b05d002662b5e29ded358ee9275ac6c2e99b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/daxctl-libs-71.1-3.el8.x86_64.rpm"],
)

rpm(
    name = "dbus-1__1.12.8-18.el8.aarch64",
    sha256 = "aaea0456f0340129d8ca09489bc1508d1bcdd6cf9af8fedaed419f014a92b621",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-1.12.8-18.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-1__1.12.8-18.el8.x86_64",
    sha256 = "4e53bf1c32bb4a2d2162fa34095e13ad6db7b61be4c4b2a260e72b9e090e50d4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-1.12.8-18.el8.x86_64.rpm"],
)

rpm(
    name = "dbus-common-1__1.12.8-18.el8.aarch64",
    sha256 = "8d707fd0fd2c7152148d7cf058f03c8ddaac6b458c77ca046fe860b0c7a83ec1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.8-18.el8.noarch.rpm"],
)

rpm(
    name = "dbus-common-1__1.12.8-18.el8.x86_64",
    sha256 = "8d707fd0fd2c7152148d7cf058f03c8ddaac6b458c77ca046fe860b0c7a83ec1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.8-18.el8.noarch.rpm"],
)

rpm(
    name = "dbus-daemon-1__1.12.8-18.el8.aarch64",
    sha256 = "98456fd3ab85f77190a759639ed342d11499cd695f518d551001fef381c61451",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-daemon-1.12.8-18.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-daemon-1__1.12.8-18.el8.x86_64",
    sha256 = "ca1217252fae51339f8f30f753763332aaae528cd719484921d21130274b6fa6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-daemon-1.12.8-18.el8.x86_64.rpm"],
)

rpm(
    name = "dbus-libs-1__1.12.8-18.el8.aarch64",
    sha256 = "60a44cc0f7f0592cf43e62d4682ac9166b095031ec541de26746c421ead8f865",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.12.8-18.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-libs-1__1.12.8-18.el8.x86_64",
    sha256 = "f060a852fe69fd60727d8f13054dee778f40803a6a945bf0c4da1eb4322aae0e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.8-18.el8.x86_64.rpm"],
)

rpm(
    name = "dbus-tools-1__1.12.8-18.el8.aarch64",
    sha256 = "eeda2f4870339df12e346fd51871f5fc68d0fdd6c8c9d54018bef86377e09bd3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-tools-1.12.8-18.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-tools-1__1.12.8-18.el8.x86_64",
    sha256 = "e9d060063165b110317d6450e20d563b3f1f9fff3026396b19ea37cf119b9709",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-tools-1.12.8-18.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-8__1.02.181-3.el8.aarch64",
    sha256 = "a8f80c18fec48fd84a0e573e3698f36184c489814a2e2c9a944a5361d99a0a9c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.181-3.el8.aarch64.rpm"],
)

rpm(
    name = "device-mapper-8__1.02.181-3.el8.x86_64",
    sha256 = "b17f66c328688e7fc41c4833f26e7427afd76fe22abc245ef3bc00e51e950868",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.181-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-event-8__1.02.181-3.el8.x86_64",
    sha256 = "0b49adba21bed320605b4c6eee6e624bd202c5048a10a682d586a0a78e83ea7f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-1.02.181-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-event-libs-8__1.02.181-3.el8.x86_64",
    sha256 = "30c703f6317c75856917578cf9f1f48635cd63e982cb48fddb227fbfabb36c12",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-libs-1.02.181-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-libs-8__1.02.181-3.el8.aarch64",
    sha256 = "6caee334370e41d5b836663ceb6167c12f6278063a3b4d663a0fe43abbaa4dbd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.181-3.el8.aarch64.rpm"],
)

rpm(
    name = "device-mapper-libs-8__1.02.181-3.el8.x86_64",
    sha256 = "4b3118cfe0038768edbe25d4ad4322ed41bd63c7915edb67609cd6e34b577b4d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.181-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-22.el8.aarch64",
    sha256 = "7708a81c9f31f336db959467d20b9fbe84011e272981f7ddd1d3dc6a753bae8f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.4-22.el8.aarch64.rpm"],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-22.el8.x86_64",
    sha256 = "c8bdf23638bd01c0d890a6b7bbbea006f00838489566aafb8791441f9169c90d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.4-22.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-persistent-data-0__0.9.0-6.el8.x86_64",
    sha256 = "09d84aeab48bdc6091b30f578a4d5d88e80467aac5dc1b1ac8454e4db77a8e58",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-persistent-data-0.9.0-6.el8.x86_64.rpm"],
)

rpm(
    name = "dhcp-client-12__4.3.6-47.el8.0.1.x86_64",
    sha256 = "9423e73b59c427ec6258be16d7c4e2014187ee81539652dc8bee95677d5e3f6e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-client-4.3.6-47.el8.0.1.x86_64.rpm"],
)

rpm(
    name = "dhcp-common-12__4.3.6-47.el8.0.1.x86_64",
    sha256 = "92af3ab99545b691391158526294da044ff7dd51460db6ec8de81f99fc329375",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-common-4.3.6-47.el8.0.1.noarch.rpm"],
)

rpm(
    name = "dhcp-libs-12__4.3.6-47.el8.0.1.x86_64",
    sha256 = "fdbfcb0183272d9c7f898ae529ea92b69b4fb0c16c20446a3c973bcbe1e5abef",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-libs-4.3.6-47.el8.0.1.x86_64.rpm"],
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
    name = "dmidecode-1__3.3-1.el8.x86_64",
    sha256 = "6c3c36a5536f1bcc12aa5c6b4755fd3717f723baa18d12a426ff1628c7c73b7d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dmidecode-3.3-1.el8.x86_64.rpm"],
)

rpm(
    name = "dnf-0__4.7.0-7.el8.x86_64",
    sha256 = "c5f76d4032450cc3fed58ec54604c64fb102a2442eaea44929863dba3f07a727",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dnf-4.7.0-7.el8.noarch.rpm"],
)

rpm(
    name = "dnf-plugins-core-0__4.0.21-10.el8.x86_64",
    sha256 = "f22de28b828e5c7e8ef53d4fad5d32762e0237f32ff101124f745e649b896703",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dnf-plugins-core-4.0.21-10.el8.noarch.rpm"],
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
    name = "e2fsprogs-0__1.45.6-3.el8.aarch64",
    sha256 = "b3f37fdfc42760affc52b83952ea4d97504a446f93ec459b3d529aed981d68a7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.45.6-3.el8.aarch64.rpm"],
)

rpm(
    name = "e2fsprogs-0__1.45.6-3.el8.x86_64",
    sha256 = "735b17184c4dfc319c0f733952366c12d53f7ae7dca0c4db068dccfbbf6560bc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.45.6-3.el8.x86_64.rpm"],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-3.el8.aarch64",
    sha256 = "f28e1ba48d059774537f7a13a8a87643693828a9aa68ce2fb786eab9635d79af",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.45.6-3.el8.aarch64.rpm"],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-3.el8.x86_64",
    sha256 = "0ff9e879fc05de4714ff375ae810ccd0a86f23a18459a7fa8af4b365c3ccae6c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.45.6-3.el8.x86_64.rpm"],
)

rpm(
    name = "edk2-aarch64-0__20210527gite1999b264f1f-1.el8.aarch64",
    sha256 = "774b59c8042bbb4abb948c6a2eaeb07ef81ca3d4c18260599186dc39913ffa48",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20210527gite1999b264f1f-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/774b59c8042bbb4abb948c6a2eaeb07ef81ca3d4c18260599186dc39913ffa48",
    ],
)

rpm(
    name = "edk2-ovmf-0__20210527gite1999b264f1f-1.el8.x86_64",
    sha256 = "a03af8838431f72ae3301144ae90fd01a58a52e6bd94d5f2ac869471a4e0362d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20210527gite1999b264f1f-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a03af8838431f72ae3301144ae90fd01a58a52e6bd94d5f2ac869471a4e0362d",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.186-1.el8.aarch64",
    sha256 = "69d582121cd34ab51b1557543d0beaea62bcc5e624ea1c68a2c5bb0351955314",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.186-1.el8.noarch.rpm"],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.186-1.el8.x86_64",
    sha256 = "69d582121cd34ab51b1557543d0beaea62bcc5e624ea1c68a2c5bb0351955314",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.186-1.el8.noarch.rpm"],
)

rpm(
    name = "elfutils-libelf-0__0.186-1.el8.aarch64",
    sha256 = "8a9bc73377accf3b11ef38cad26a12f47c247f76994be3919c09c64211d1b207",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.186-1.el8.aarch64.rpm"],
)

rpm(
    name = "elfutils-libelf-0__0.186-1.el8.x86_64",
    sha256 = "4901abdc87d933d5000724e40dd0c6f196a23eaf2792b82dff33b07be740e53c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.186-1.el8.x86_64.rpm"],
)

rpm(
    name = "elfutils-libs-0__0.186-1.el8.aarch64",
    sha256 = "19c4c1075d093e905f1dcc6b04f0d471d1eb9f2e959a1398e12cf80a5b548402",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.186-1.el8.aarch64.rpm"],
)

rpm(
    name = "elfutils-libs-0__0.186-1.el8.x86_64",
    sha256 = "a913dec09e866e860fb5d67cbcab73ae861ec44a0758ec4c0810f975dfebdfd5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.186-1.el8.x86_64.rpm"],
)

rpm(
    name = "ethtool-2__5.13-1.el8.aarch64",
    sha256 = "c00518a0294c75119072266aaa855ad95e4e1f065a4b22f3f7f65707ee097eec",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ethtool-5.13-1.el8.aarch64.rpm"],
)

rpm(
    name = "ethtool-2__5.13-1.el8.x86_64",
    sha256 = "9e0d3a7f05fc84bf4be7972c3614816088fd560da52e512708943808c4fee48d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ethtool-5.13-1.el8.x86_64.rpm"],
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
    name = "fuse-0__2.9.7-14.el8.x86_64",
    sha256 = "7e2bd1d1053468be827a616d9a38709b6559a36e8280eddfd641a5ea93d04867",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-2.9.7-14.el8.x86_64.rpm"],
)

rpm(
    name = "fuse-common-0__3.3.0-14.el8.x86_64",
    sha256 = "70eb3c336ecdab5fd416d71ff793ad08ce79d0a0bb1f15885731b6880f360a77",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-common-3.3.0-14.el8.x86_64.rpm"],
)

rpm(
    name = "fuse-libs-0__2.9.7-14.el8.aarch64",
    sha256 = "cf633d5fbcd926edcbbc80c1308b7a3bd18ad69185f6e277c4cb74aedffac7ff",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/fuse-libs-2.9.7-14.el8.aarch64.rpm"],
)

rpm(
    name = "fuse-libs-0__2.9.7-14.el8.x86_64",
    sha256 = "77a8a518f033728705d0f6121c2a863c5b38d63af3ebd92e9cc5fbd589022bc7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.7-14.el8.x86_64.rpm"],
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
    name = "gcc-0__8.5.0-10.el8.aarch64",
    sha256 = "a1406b454d5467c15f891f96bb92c5802c43da4098029fab1c0cc94490f7a6ee",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gcc-8.5.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "gcc-0__8.5.0-10.el8.x86_64",
    sha256 = "d9f477f9c2fcd6ed9e0e274dcd2eb9342d2f24d8cd29a782496a8542b62c12a1",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gcc-8.5.0-10.el8.x86_64.rpm"],
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
    name = "glib2-0__2.56.4-158.el8.aarch64",
    sha256 = "1a0d575a73b98550a79497c8bd43f78e58fd6fcb441697eb907c6d02d21762fe",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glib2-2.56.4-158.el8.aarch64.rpm"],
)

rpm(
    name = "glib2-0__2.56.4-158.el8.x86_64",
    sha256 = "4543844c3ebdc85d8c0facaeb9de9e0cda324995fd62e084ffdbf238204fa529",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glib2-2.56.4-158.el8.x86_64.rpm"],
)

rpm(
    name = "glibc-0__2.28-189.el8.aarch64",
    sha256 = "0b0efd2a42aea6e33d588c5eb13e33afc312f284c4eac60246247f7a18f17ab3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-2.28-189.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-0__2.28-189.el8.x86_64",
    sha256 = "0337020f46cba670130c8bf50d3d88b90bc5b76e65f9326cc2806c66c49bbf5a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-2.28-189.el8.x86_64.rpm"],
)

rpm(
    name = "glibc-common-0__2.28-189.el8.aarch64",
    sha256 = "ba9b85c0f600006ee76ad797376e8908292df111628c873b56c6d37f2124f933",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-common-2.28-189.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-common-0__2.28-189.el8.x86_64",
    sha256 = "50f9fe79b42e27421e2709cae89d2329d16b3b31804b846ba1d0b727743c21f9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-common-2.28-189.el8.x86_64.rpm"],
)

rpm(
    name = "glibc-devel-0__2.28-189.el8.aarch64",
    sha256 = "5de92bf4b68189409c818d45e3f633d8b41b21a33aaa6770cf8509d31bda7d1c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-devel-2.28-189.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-devel-0__2.28-189.el8.x86_64",
    sha256 = "c29d7a95e83da8013028887ed2cf087e61f49a98e86f999b936745d859a8789c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-devel-2.28-189.el8.x86_64.rpm"],
)

rpm(
    name = "glibc-headers-0__2.28-189.el8.aarch64",
    sha256 = "1f33629bed515802f4bed8fc2107b31ce7847171c90ece051fc882201406590c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-headers-2.28-189.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-headers-0__2.28-189.el8.x86_64",
    sha256 = "91974a639df6e79a19def97a8c15e98f4ea5bcf4e12d48763ce156ba0c85be1a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-headers-2.28-189.el8.x86_64.rpm"],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-189.el8.aarch64",
    sha256 = "fb4892bb07dfe4d0058603cfc6b86e5e53f62c864522c01ca0b51577a719ed6d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.28-189.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-189.el8.x86_64",
    sha256 = "67d69e6656b6d526d1f1292e357f9a0b26c02cd6644b8dd4fb6b05bcc8817631",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.28-189.el8.x86_64.rpm"],
)

rpm(
    name = "glibc-static-0__2.28-189.el8.aarch64",
    sha256 = "06db5860b07b5b1cca6b160efb8e97dd3a09a2b396b1f23bc6cf48e136a27b4c",
    urls = ["http://mirror.centos.org/centos/8-stream/PowerTools/aarch64/os/Packages/glibc-static-2.28-189.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-static-0__2.28-189.el8.x86_64",
    sha256 = "363f19ce5048770e2db4680b57f12053a781dce6665b537d73472a5f79dabbd2",
    urls = ["http://mirror.centos.org/centos/8-stream/PowerTools/x86_64/os/Packages/glibc-static-2.28-189.el8.x86_64.rpm"],
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
    name = "groff-base-0__1.22.3-18.el8.x86_64",
    sha256 = "b00855013100d3796e9ed6d82b1ab2d4dc7f4a3a3fa2e186f6de8523577974a0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/groff-base-1.22.3-18.el8.x86_64.rpm"],
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
    name = "info-0__6.5-7.el8_5.aarch64",
    sha256 = "24a7e6f02ac095d965832203d0c8a9ee13aea301ef8572bb1ecdace435c796be",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/info-6.5-7.el8_5.aarch64.rpm"],
)

rpm(
    name = "info-0__6.5-7.el8_5.x86_64",
    sha256 = "63f03261cc8109b2fb61002ca50c93e52acb9cfd8382d139e8de6623394051e8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/info-6.5-7.el8_5.x86_64.rpm"],
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
    name = "iproute-0__5.15.0-3.el8.aarch64",
    sha256 = "70b31d692996af069fbf2d234b92373ed503a678568e1b1a780aa0cb7d05b820",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-5.15.0-3.el8.aarch64.rpm"],
)

rpm(
    name = "iproute-0__5.15.0-3.el8.x86_64",
    sha256 = "eccaa5603ea1d41752f20affa37c366b5a8ebe528f52dc28c01c410f9aea38bb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-5.15.0-3.el8.x86_64.rpm"],
)

rpm(
    name = "iproute-tc-0__5.15.0-3.el8.aarch64",
    sha256 = "acd893375be6e9af3c9cfa0a63d7d47f5851e3440a3bee5a326ad2754a8d14f3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-tc-5.15.0-3.el8.aarch64.rpm"],
)

rpm(
    name = "iproute-tc-0__5.15.0-3.el8.x86_64",
    sha256 = "dd79fa1e306bbc7d534f803d881d4388201219c264f77db3aa272b1ce671f348",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-tc-5.15.0-3.el8.x86_64.rpm"],
)

rpm(
    name = "iptables-0__1.8.4-22.el8.aarch64",
    sha256 = "ddc326783f717ba44a9872faf2c82e2ea0fde7de021b2f144417889f84ce614e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-1.8.4-22.el8.aarch64.rpm"],
)

rpm(
    name = "iptables-0__1.8.4-22.el8.x86_64",
    sha256 = "8993bf1f075984412a57cf5b2c0110984a4dd1125d60048872c85f9c496f9e66",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-1.8.4-22.el8.x86_64.rpm"],
)

rpm(
    name = "iptables-libs-0__1.8.4-22.el8.aarch64",
    sha256 = "2c99183b888b75ae3b1d7665836757fb7a1ba130b03454177331f9efe33ca630",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.4-22.el8.aarch64.rpm"],
)

rpm(
    name = "iptables-libs-0__1.8.4-22.el8.x86_64",
    sha256 = "84cef50494317f6e968c8a27134e6f442a1898b137715bd69573d6d72e7b6fb1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.4-22.el8.x86_64.rpm"],
)

rpm(
    name = "iputils-0__20180629-9.el8.aarch64",
    sha256 = "5fe55a880041e5f8e1193639e3caab23ab4b18533a56a87c56f27b43dbc34531",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iputils-20180629-9.el8.aarch64.rpm"],
)

rpm(
    name = "iputils-0__20180629-9.el8.x86_64",
    sha256 = "45a993bc309492f10e3d45edc22adf125341a2b4ffb992f7e8087086fa704bff",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iputils-20180629-9.el8.x86_64.rpm"],
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
    name = "jansson-0__2.14-1.el8.aarch64",
    sha256 = "69b4dd56ca16ed4ac5840e0d39a29d2e0b050905a349e1aceae4ec511a11b792",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/jansson-2.14-1.el8.aarch64.rpm"],
)

rpm(
    name = "jansson-0__2.14-1.el8.x86_64",
    sha256 = "f825b85b4506a740fb2f85b9a577c51264f3cfe792dd8b2bf8963059cc77c3c4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/jansson-2.14-1.el8.x86_64.rpm"],
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
    name = "kernel-headers-0__4.18.0-365.el8.aarch64",
    sha256 = "caa815c0c67c3070312647809c91b510d1f20f9f9530eb853d33831430b9165f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kernel-headers-4.18.0-365.el8.aarch64.rpm"],
)

rpm(
    name = "kernel-headers-0__4.18.0-365.el8.x86_64",
    sha256 = "8a5b4e112c916f578907e595475ebc02f1c62aaa243c913d30429b7aff476ea5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kernel-headers-4.18.0-365.el8.x86_64.rpm"],
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
    name = "kmod-0__25-19.el8.aarch64",
    sha256 = "056e83e9da3c6a582e83634b66c3ead78f1729f4b9dbd9970dbf3bfdc45edb54",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-25-19.el8.aarch64.rpm"],
)

rpm(
    name = "kmod-0__25-19.el8.x86_64",
    sha256 = "37c299fdaa42efb0d653ba5e22c83bd20833af1244b66ed6ea880e75c1672dd2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-25-19.el8.x86_64.rpm"],
)

rpm(
    name = "kmod-libs-0__25-19.el8.aarch64",
    sha256 = "053b443be1bb0cbbc6da3314775391950350106462cc1dae01c7aed4358bf852",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-libs-25-19.el8.aarch64.rpm"],
)

rpm(
    name = "kmod-libs-0__25-19.el8.x86_64",
    sha256 = "46a2ddc6067ed12089f04f2255c57117992807d707e280fc002f3ce786fc2abf",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-libs-25-19.el8.x86_64.rpm"],
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
    name = "libarchive-0__3.3.3-3.el8_5.aarch64",
    sha256 = "01a00cc3d5239857ae9cbfa2005581c850a6f47e343cf523b7961d951e7df0bd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libarchive-3.3.3-3.el8_5.aarch64.rpm"],
)

rpm(
    name = "libarchive-0__3.3.3-3.el8_5.x86_64",
    sha256 = "461f8d0def774d7b3b99cdc3ad2e9b8c6216f34f3b70d38fb9af75607da6e1f3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libarchive-3.3.3-3.el8_5.x86_64.rpm"],
)

rpm(
    name = "libasan-0__8.5.0-10.el8.aarch64",
    sha256 = "472418043a744026d32844adde3b9e798f098430b4f4a88cf3d9e4285c3a1a71",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libasan-8.5.0-10.el8.aarch64.rpm"],
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
    name = "libatomic-0__8.5.0-10.el8.aarch64",
    sha256 = "370b70040c3f15e4ca6a75bef0fe841c9dd419eb7100ca84cdb722a079755956",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libatomic-8.5.0-10.el8.aarch64.rpm"],
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
    name = "libblkid-0__2.32.1-32.el8.aarch64",
    sha256 = "07fa2fb3ffde8797b45ebfe3123270d28d9d09b3ebb5289c3f5f563240b89f83",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libblkid-2.32.1-32.el8.aarch64.rpm"],
)

rpm(
    name = "libblkid-0__2.32.1-32.el8.x86_64",
    sha256 = "3617554c78c7d072bf381d0590c15d45afef2096af36d14753278cd72a2d3391",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libblkid-2.32.1-32.el8.x86_64.rpm"],
)

rpm(
    name = "libbpf-0__0.4.0-3.el8.aarch64",
    sha256 = "d1b96b4e5da6bd14ac8354c9ff70e910108313dc46fcad406e63695a956eec3e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libbpf-0.4.0-3.el8.aarch64.rpm"],
)

rpm(
    name = "libbpf-0__0.4.0-3.el8.x86_64",
    sha256 = "186eb41dd1bdf2955287e36e9a20bd254e03d19d7457e1dc3c503d89b08bbb44",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libbpf-0.4.0-3.el8.x86_64.rpm"],
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
    name = "libcap-0__2.48-2.el8.aarch64",
    sha256 = "5d2c94c1d54cfdd4fb02287e3c93be0a7ccef6ac0dc1e36a0d8782044a8d5014",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcap-2.48-2.el8.aarch64.rpm"],
)

rpm(
    name = "libcap-0__2.48-2.el8.x86_64",
    sha256 = "187b54e8368f1687f53ec15580b78f085f080690d17ee4f566967e74d257840e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcap-2.48-2.el8.x86_64.rpm"],
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
    name = "libcom_err-0__1.45.6-3.el8.aarch64",
    sha256 = "52e7f6004fa7adca50000b6ed02b617c0b072c5aa1572c13c05fa0037edc8a10",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcom_err-1.45.6-3.el8.aarch64.rpm"],
)

rpm(
    name = "libcom_err-0__1.45.6-3.el8.x86_64",
    sha256 = "79e1c8b506bc6b41616c86e10cb8204a71000daf938a354b960b925f525cfe24",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcom_err-1.45.6-3.el8.x86_64.rpm"],
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
    name = "libfdisk-0__2.32.1-32.el8.aarch64",
    sha256 = "419dfa974b23b5a0c93ff2355803e75ccd41f977399d4e144b5a63cb5e5cc88d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libfdisk-2.32.1-32.el8.aarch64.rpm"],
)

rpm(
    name = "libfdisk-0__2.32.1-32.el8.x86_64",
    sha256 = "0bd547870e69b6a2f48779eb78560a2c3c97b006ed2ca5d76d3c24f04f0d0892",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libfdisk-2.32.1-32.el8.x86_64.rpm"],
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
    name = "libffi-0__3.1-23.el8.aarch64",
    sha256 = "ba34d0bb067722c37dd4367534d82aa18c659facbfd17952f8d826e8662cb7c1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libffi-3.1-23.el8.aarch64.rpm"],
)

rpm(
    name = "libffi-0__3.1-23.el8.x86_64",
    sha256 = "643d1b969c7fbcd55c523f779089f3f2fe8b105c719fd49c7edd1f142dfc2143",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libffi-3.1-23.el8.x86_64.rpm"],
)

rpm(
    name = "libgcc-0__8.5.0-10.el8.aarch64",
    sha256 = "efafc69402d1f9b195c7946e20e0c285da44ddb151f2a67e58275b228e077094",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcc-8.5.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "libgcc-0__8.5.0-10.el8.x86_64",
    sha256 = "8b92c47d3fee8eeb1314e96096605f46bcc2985292b9210d0e9947dbe3ab9867",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcc-8.5.0-10.el8.x86_64.rpm"],
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
    name = "libgomp-0__8.5.0-10.el8.aarch64",
    sha256 = "51097c11b92a117bdfad4c9de90e6bc16437c15e024347f1ca2c44c6e18a6d70",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgomp-8.5.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "libgomp-0__8.5.0-10.el8.x86_64",
    sha256 = "7df9487d1f8cef4e33663656cfa20ef0963838a52f107ddf5037e83e209ceb46",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgomp-8.5.0-10.el8.x86_64.rpm"],
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
    name = "libibverbs-0__37.2-1.el8.aarch64",
    sha256 = "588f7bddcde691b08b8cf652de96f5e3334231ff4d3697353fbd53d25eadad03",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libibverbs-37.2-1.el8.aarch64.rpm"],
)

rpm(
    name = "libibverbs-0__37.2-1.el8.x86_64",
    sha256 = "146616aaa85d8b44dd554db70468e4237d88dd5f4335b40e3c28209554d86102",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libibverbs-37.2-1.el8.x86_64.rpm"],
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
    name = "libmount-0__2.32.1-32.el8.aarch64",
    sha256 = "3f8d49bcacd9b81ac8aed37673d4f066a6edb98169b61a323e0960f40d2c337b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmount-2.32.1-32.el8.aarch64.rpm"],
)

rpm(
    name = "libmount-0__2.32.1-32.el8.x86_64",
    sha256 = "109a30abb8912cf8b9e455acdb485a5c7766a2ae01857a98ee7520bf89faebc0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmount-2.32.1-32.el8.x86_64.rpm"],
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
    name = "libnftnl-0__1.1.5-5.el8.aarch64",
    sha256 = "00522e43ce63cf63468052e627a429ededac0815212c644f4eadda88b990c3ee",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnftnl-1.1.5-5.el8.aarch64.rpm"],
)

rpm(
    name = "libnftnl-0__1.1.5-5.el8.x86_64",
    sha256 = "293e1f0f44a9c1d5dedbe831dff3049fad9e88c5f0e281d889f427603ac51fa6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnftnl-1.1.5-5.el8.x86_64.rpm"],
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
    name = "libpmem-0__1.11.1-1.module_el8.6.0__plus__1088__plus__6891f51c.x86_64",
    sha256 = "924d405a5a7b2de6405cd277f72f2c4af20f9162e8484d9142bd7a56f546a894",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libpmem-1.11.1-1.module_el8.6.0+1088+6891f51c.x86_64.rpm"],
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
    name = "librdmacm-0__37.2-1.el8.aarch64",
    sha256 = "b2a6c52cb312817b7c4010d7af46df4cf378a680bb611770e85dae2f259fa96c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/librdmacm-37.2-1.el8.aarch64.rpm"],
)

rpm(
    name = "librdmacm-0__37.2-1.el8.x86_64",
    sha256 = "ebee4a85de6732065dc0cfe2967613188964a26bcd276498e5f02a5c62797979",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/librdmacm-37.2-1.el8.x86_64.rpm"],
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
    name = "libsmartcols-0__2.32.1-32.el8.aarch64",
    sha256 = "742078bf76069fe1363f917ddff19b4e1a53814377951c884bdd923c11340cb0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.32.1-32.el8.aarch64.rpm"],
)

rpm(
    name = "libsmartcols-0__2.32.1-32.el8.x86_64",
    sha256 = "8eb37b78baefc93f5d524fb08040af6448bd7151b3a0564381f28efb78c6b9c4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.32.1-32.el8.x86_64.rpm"],
)

rpm(
    name = "libss-0__1.45.6-3.el8.aarch64",
    sha256 = "63556df1f4ccac289c156d73cbd0a999724fa9302aa07f49ce164a66aab31998",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libss-1.45.6-3.el8.aarch64.rpm"],
)

rpm(
    name = "libss-0__1.45.6-3.el8.x86_64",
    sha256 = "09f91e371deb818112b4688896020b0dacf921a30087e9f26e2a17ae2f52e269",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libss-1.45.6-3.el8.x86_64.rpm"],
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
    name = "libsss_idmap-0__2.6.1-2.el8.aarch64",
    sha256 = "a5991f698619f4b8f197dfc7b3037ccb0ec2fe688c17bff777702e45a245d342",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.6.1-2.el8.aarch64.rpm"],
)

rpm(
    name = "libsss_idmap-0__2.6.1-2.el8.x86_64",
    sha256 = "b15dc77b71e99cd06da153455a5b705fe982e1d8d32fb5633eb4b9a1b3315477",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.6.1-2.el8.x86_64.rpm"],
)

rpm(
    name = "libsss_nss_idmap-0__2.6.1-2.el8.aarch64",
    sha256 = "d7e08a889c01ce82a1791f8f06f70b81e841276d7be300fb81cfcf092130bb2e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.6.1-2.el8.aarch64.rpm"],
)

rpm(
    name = "libsss_nss_idmap-0__2.6.1-2.el8.x86_64",
    sha256 = "893119e9cc3b3ca13f8e55bfd5705f69973bafcdd2aba7b28c44c354628de3c5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.6.1-2.el8.x86_64.rpm"],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-10.el8.aarch64",
    sha256 = "eec79512e887485aa60a55f5efed8ee84691f3464fff5467efac3b16a31cfe4b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libstdc++-8.5.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-10.el8.x86_64",
    sha256 = "353c910c4a489cde1b2d7db91fc02e9847188600a2841c70934676a4358341e9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libstdc++-8.5.0-10.el8.x86_64.rpm"],
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
    name = "libtirpc-0__1.1.4-6.el8.aarch64",
    sha256 = "34e53a718ae7b0fed25eb90780853d08dd1d2b8095545bad557e974d5e3c1498",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libtirpc-1.1.4-6.el8.aarch64.rpm"],
)

rpm(
    name = "libtirpc-0__1.1.4-6.el8.x86_64",
    sha256 = "aebbff9c7bce81a83874da0d90529a6cdb4db841790165c510c1260abeb382d4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libtirpc-1.1.4-6.el8.x86_64.rpm"],
)

rpm(
    name = "libtpms-0__0.9.1-0.20211126git1ff6fe1f43.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "6e7acc637f2b2697a18419655b9a0d4b9d06b1a7a380c4546de8064c1be035d5",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libtpms-0.9.1-0.20211126git1ff6fe1f43.module_el8.6.0+1087+b42c8331.aarch64.rpm"],
)

rpm(
    name = "libtpms-0__0.9.1-0.20211126git1ff6fe1f43.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "cbe249e006aa70ba12707e0b7151164f4e4a50060d97e2be5eb139033e889b74",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libtpms-0.9.1-0.20211126git1ff6fe1f43.module_el8.6.0+1087+b42c8331.x86_64.rpm"],
)

rpm(
    name = "libubsan-0__8.5.0-10.el8.aarch64",
    sha256 = "27a453616cae4817ab7ed53182de132ba100dcfbfdc1a2cd46751d0b37516b1f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libubsan-8.5.0-10.el8.aarch64.rpm"],
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
    name = "libuuid-0__2.32.1-32.el8.aarch64",
    sha256 = "267217612404193c0d421be4c74b5c7bf3734b1d6d602da691786d1ddb43a168",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libuuid-2.32.1-32.el8.aarch64.rpm"],
)

rpm(
    name = "libuuid-0__2.32.1-32.el8.x86_64",
    sha256 = "d1a3d4b89e2e6ced2bb1312eb832256967bfe53cd1106d0d8ef3911802be7dff",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libuuid-2.32.1-32.el8.x86_64.rpm"],
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
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-client-7.6.0-6.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b3bbb74465f8cf00e7f8f5889ef952b3849ac5392c032cd2eec423b04d89c179",
    ],
)

rpm(
    name = "libvirt-client-0__7.6.0-6.el8s.x86_64",
    sha256 = "9eea0d26a51c00032f31da04b29ece265b82250f011a1ef0cdad75ec1a547b0d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-client-7.6.0-6.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9eea0d26a51c00032f31da04b29ece265b82250f011a1ef0cdad75ec1a547b0d",
    ],
)

rpm(
    name = "libvirt-daemon-0__7.6.0-6.el8s.aarch64",
    sha256 = "c6e4128f38aa16a27d79681716865f0eb903421d1097bda3e30effc29e9a129e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-daemon-7.6.0-6.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c6e4128f38aa16a27d79681716865f0eb903421d1097bda3e30effc29e9a129e",
    ],
)

rpm(
    name = "libvirt-daemon-0__7.6.0-6.el8s.x86_64",
    sha256 = "6aa8e6f8d7dc8c419a6bc01773900402facce418cad5eb8b43781fb603e1460c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-7.6.0-6.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6aa8e6f8d7dc8c419a6bc01773900402facce418cad5eb8b43781fb603e1460c",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.6.0-6.el8s.aarch64",
    sha256 = "021cd9f5005bbd9db238b0840cc1725366671599987ad8622eb399ed20fc7395",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-daemon-driver-qemu-7.6.0-6.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/021cd9f5005bbd9db238b0840cc1725366671599987ad8622eb399ed20fc7395",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.6.0-6.el8s.x86_64",
    sha256 = "9e7dbb1b6f81c6100e7517663d63cb8ac9eab5758a979a8ba86d5fcd8072aded",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-driver-qemu-7.6.0-6.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9e7dbb1b6f81c6100e7517663d63cb8ac9eab5758a979a8ba86d5fcd8072aded",
    ],
)

rpm(
    name = "libvirt-devel-0__7.6.0-6.el8s.aarch64",
    sha256 = "f6d8f8edd459aed9a0ef14e23afd0beedc299c8ad2133d52d61e919d5bd4fd0f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-devel-7.6.0-6.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f6d8f8edd459aed9a0ef14e23afd0beedc299c8ad2133d52d61e919d5bd4fd0f",
    ],
)

rpm(
    name = "libvirt-devel-0__7.6.0-6.el8s.x86_64",
    sha256 = "9d0f5740c176cb7630e4d4add1b839a4648a9ad723be754abd324eb9a65c504f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-devel-7.6.0-6.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d0f5740c176cb7630e4d4add1b839a4648a9ad723be754abd324eb9a65c504f",
    ],
)

rpm(
    name = "libvirt-libs-0__7.6.0-6.el8s.aarch64",
    sha256 = "79071ce936c963eca72fea4280008a6df494a594eb5346bcdf4810080eb09bd1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-libs-7.6.0-6.el8s.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/79071ce936c963eca72fea4280008a6df494a594eb5346bcdf4810080eb09bd1",
    ],
)

rpm(
    name = "libvirt-libs-0__7.6.0-6.el8s.x86_64",
    sha256 = "d0ced5e0ffae07a2aa17da0509e01a3efd251901eca47b4078bf89c476e801b1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-libs-7.6.0-6.el8s.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d0ced5e0ffae07a2aa17da0509e01a3efd251901eca47b4078bf89c476e801b1",
    ],
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
    name = "lvm2-8__2.03.14-3.el8.x86_64",
    sha256 = "51146493afbdd053e51e8f930f85572bbfc4f4cd12aad2ba19dec06df6b0ec07",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-2.03.14-3.el8.x86_64.rpm"],
)

rpm(
    name = "lvm2-libs-8__2.03.14-3.el8.x86_64",
    sha256 = "7b3b5723d789421fe8af20e5dd6a0ba1344eb5fbfddd02ffeed6ea0b3f8d0111",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-libs-2.03.14-3.el8.x86_64.rpm"],
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
    name = "mdadm-0__4.2-rc3.el8.x86_64",
    sha256 = "0e66d6845fe5c7304f4fd64575c48a528a8cdd20ffbda650ba9b37fadebdc622",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mdadm-4.2-rc3.el8.x86_64.rpm"],
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
    name = "ncurses-0__6.1-9.20180224.el8.x86_64",
    sha256 = "fc22ce73243e2f926e72967c28de57beabfa3720e51248b9a39e40207fbc6c8a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ncurses-6.1-9.20180224.el8.x86_64.rpm"],
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
    name = "ndctl-libs-0__71.1-3.el8.x86_64",
    sha256 = "a2d43111debbe48a2e2cd85a6df94cd861abe24ac4d2da3058e9946252909f43",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ndctl-libs-71.1-3.el8.x86_64.rpm"],
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
    name = "nftables-1__0.9.3-24.el8.aarch64",
    sha256 = "8e957d3783c2e9683746813324a2ac906209e3a9c580ed1a713a7804a0682598",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/nftables-0.9.3-24.el8.aarch64.rpm"],
)

rpm(
    name = "nftables-1__0.9.3-24.el8.x86_64",
    sha256 = "5962d711c48ffa228cf236f2316ac28610b56259a1dc55ab5a41ea09344f0f7e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/nftables-0.9.3-24.el8.x86_64.rpm"],
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
    name = "openssl-libs-1__1.1.1k-5.el8_5.aarch64",
    sha256 = "5649769840389235bd09b1777db024fa1c84e9213321c73a972151608aa802f4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/openssl-libs-1.1.1k-5.el8_5.aarch64.rpm"],
)

rpm(
    name = "openssl-libs-1__1.1.1k-5.el8_5.x86_64",
    sha256 = "fcb20a662cb237a35396f0738b46f0a648273b836d44ca6e5620a09e3b5d2cdb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/openssl-libs-1.1.1k-5.el8_5.x86_64.rpm"],
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
    name = "pam-0__1.3.1-16.el8.aarch64",
    sha256 = "e40de16b3a574426a6220e81e4bc522a06e2093d7b742c39bf2bca745eff5d96",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pam-1.3.1-16.el8.aarch64.rpm"],
)

rpm(
    name = "pam-0__1.3.1-16.el8.x86_64",
    sha256 = "b38d198bb4a224f7c79b1e66a1576998bea774f7266349c5178eb0c086da1722",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pam-1.3.1-16.el8.x86_64.rpm"],
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
    name = "perl-Carp-0__1.42-396.el8.x86_64",
    sha256 = "d03b9f4b9848e3a88d62bcf6e536d659c325b2dc03b2136be7342b5fe5e2b6a9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Carp-1.42-396.el8.noarch.rpm"],
)

rpm(
    name = "perl-Encode-4__2.97-3.el8.x86_64",
    sha256 = "d2b0e4b28a5aac754f6caa119d5479a64816f93c059e0ac564e46391264e2234",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Encode-2.97-3.el8.x86_64.rpm"],
)

rpm(
    name = "perl-Errno-0__1.28-421.el8.x86_64",
    sha256 = "8d9b26f17e427dc497032b1897b9296c4ca37fa1b96d9c459b42516d72ef06a1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Errno-1.28-421.el8.x86_64.rpm"],
)

rpm(
    name = "perl-Exporter-0__5.72-396.el8.x86_64",
    sha256 = "7edc503f5a919c489b651757095d8031982d530cc88088fdaeb743188364e9b0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Exporter-5.72-396.el8.noarch.rpm"],
)

rpm(
    name = "perl-File-Path-0__2.15-2.el8.x86_64",
    sha256 = "e83928bd4552ecdf8e71d283e2358c7eccd006d284ba31fbc9c89e407989fd60",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-File-Path-2.15-2.el8.noarch.rpm"],
)

rpm(
    name = "perl-File-Temp-0__0.230.600-1.el8.x86_64",
    sha256 = "e269f7d33abbb790311ffa95fa7df9766cac8bf31ace24fce6ed732ba0db19ae",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-File-Temp-0.230.600-1.el8.noarch.rpm"],
)

rpm(
    name = "perl-Getopt-Long-1__2.50-4.el8.x86_64",
    sha256 = "da4c6daa0d5406bc967cc89b02a69689491f42c543aceea1a31136f0f1a8d991",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Getopt-Long-2.50-4.el8.noarch.rpm"],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.074-1.el8.x86_64",
    sha256 = "a1af93a1b62e8ca05b7597d5749a2b3d28735a86928f0432064fec61db1ff844",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-HTTP-Tiny-0.074-1.el8.noarch.rpm"],
)

rpm(
    name = "perl-IO-0__1.38-421.el8.x86_64",
    sha256 = "7ff911df218c38953660d4a09f9864364e2433b9aaf8283db8b7d5214411e28a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-IO-1.38-421.el8.x86_64.rpm"],
)

rpm(
    name = "perl-MIME-Base64-0__3.15-396.el8.x86_64",
    sha256 = "5642297bf32bb174173917dd10fd2a3a2ef7277c599f76c0669c5c448f10bdaf",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-MIME-Base64-3.15-396.el8.x86_64.rpm"],
)

rpm(
    name = "perl-PathTools-0__3.74-1.el8.x86_64",
    sha256 = "512245f7741790b36b03562469b9262f4dedfb8862dfa2d42e64598bb205d4c9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-PathTools-3.74-1.el8.x86_64.rpm"],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-395.el8.x86_64",
    sha256 = "545cd23ad8e4f71a5109551093668fd4b5e1a50d6a60364ce0f04f64eecd99d1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Escapes-1.07-395.el8.noarch.rpm"],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28-396.el8.x86_64",
    sha256 = "0225dc3999e3d7b1bb57186a2fc93c98bd1e4e08e062fb51c966e1f2a2c91bb4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Perldoc-3.28-396.el8.noarch.rpm"],
)

rpm(
    name = "perl-Pod-Simple-1__3.35-395.el8.x86_64",
    sha256 = "51c3ee5d824bdde0a8faa10c99841c2590c0c26edfb17125aa97945a688c83ed",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Simple-3.35-395.el8.noarch.rpm"],
)

rpm(
    name = "perl-Pod-Usage-4__1.69-395.el8.x86_64",
    sha256 = "794f970f498af07b37f914c19ad5dedc6b6c2f89d343af9dd1768d17232555de",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Usage-1.69-395.el8.noarch.rpm"],
)

rpm(
    name = "perl-Scalar-List-Utils-3__1.49-2.el8.x86_64",
    sha256 = "3db0d05ca5ba00981312f3a3ddcbabf466c2f1fc639cbf29482bb2cd952df456",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Scalar-List-Utils-1.49-2.el8.x86_64.rpm"],
)

rpm(
    name = "perl-Socket-4__2.027-3.el8.x86_64",
    sha256 = "de138a9614191af63b9603cf0912d4ffd9bd9e5b122c2d0a78ae0eac009a602f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Socket-2.027-3.el8.x86_64.rpm"],
)

rpm(
    name = "perl-Storable-1__3.11-3.el8.x86_64",
    sha256 = "0c3007b68a37325866aaade4ae076232bca15e268f66c3d3b3a6d236bb85e1e9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Storable-3.11-3.el8.x86_64.rpm"],
)

rpm(
    name = "perl-Sys-Guestfs-1__1.44.0-3.el8s.x86_64",
    sha256 = "d79e373e1bc85627b68af853afcfca7fabaec4aa28dc8fbb93015564f41a7caa",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/p/perl-Sys-Guestfs-1.44.0-3.el8s.x86_64.rpm"],
)

rpm(
    name = "perl-Term-ANSIColor-0__4.06-396.el8.x86_64",
    sha256 = "f4e3607f242bbca7ec2379822ca961860e6d9c276da51c6e2dfd17a29469ec78",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Term-ANSIColor-4.06-396.el8.noarch.rpm"],
)

rpm(
    name = "perl-Term-Cap-0__1.17-395.el8.x86_64",
    sha256 = "6bbb721dd2c411c85c75f7477b14c54c776d78ee9b93557615e919ef47577440",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Term-Cap-1.17-395.el8.noarch.rpm"],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-395.el8.x86_64",
    sha256 = "2975de6545b4ca7907ae368a1716c531764e4afccbf27fb0a694d90e983c38e2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Text-ParseWords-3.30-395.el8.noarch.rpm"],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-395.el8.x86_64",
    sha256 = "7e50a5d0f2fbd8c95375f72f5772c7731186e999a447121b8247f448b065a4ef",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Text-Tabs+Wrap-2013.0523-395.el8.noarch.rpm"],
)

rpm(
    name = "perl-Time-Local-1__1.280-1.el8.x86_64",
    sha256 = "1edcf2b441ddf21417ef2b33e1ab2a30900758819335d7fabafe3b16bb3eab62",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Time-Local-1.280-1.el8.noarch.rpm"],
)

rpm(
    name = "perl-Unicode-Normalize-0__1.25-396.el8.x86_64",
    sha256 = "99678a57c35343d8b2e2a502efcccc17bde3e40d97d7d2c5f988af8d3aa166d0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Unicode-Normalize-1.25-396.el8.x86_64.rpm"],
)

rpm(
    name = "perl-constant-0__1.33-396.el8.x86_64",
    sha256 = "7559c097998db5e5d14dab1a7a1637a5749e9dab234ca68d17c9c21f8cfbf8d6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-constant-1.33-396.el8.noarch.rpm"],
)

rpm(
    name = "perl-hivex-0__1.3.18-23.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "42db01e9df5ba75147ad2a0cfb37f5f6c37ae980260d218dc93a0ead8cab7983",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/perl-hivex-1.3.18-23.module_el8.6.0+983+a7505f3f.x86_64.rpm"],
)

rpm(
    name = "perl-interpreter-4__5.26.3-421.el8.x86_64",
    sha256 = "4618427acf4bcfa66ec91cccf995d938e1ed0f87b1088d7d948a9993a6d15b29",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-interpreter-5.26.3-421.el8.x86_64.rpm"],
)

rpm(
    name = "perl-libintl-perl-0__1.29-2.el8.x86_64",
    sha256 = "8b8c1ce375e1d8dd73f905e99bd452243ec194dd707a36fa5bdea7a252165c60",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/perl-libintl-perl-1.29-2.el8.x86_64.rpm"],
)

rpm(
    name = "perl-libs-4__5.26.3-421.el8.x86_64",
    sha256 = "d3a5510385cd4b2d53d70942e4fb4c149917aac2ce2df881c28ae2afdcd26619",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-libs-5.26.3-421.el8.x86_64.rpm"],
)

rpm(
    name = "perl-macros-4__5.26.3-421.el8.x86_64",
    sha256 = "5969bb5bd8b28a6cead135cfbdae89ac60f649b29f88a1daac3016eea47dc45b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-macros-5.26.3-421.el8.x86_64.rpm"],
)

rpm(
    name = "perl-parent-1__0.237-1.el8.x86_64",
    sha256 = "f5e73bbd776a2426a796971d8d38664f2e94898479fb76947dccdd28cf9fe1d0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-parent-0.237-1.el8.noarch.rpm"],
)

rpm(
    name = "perl-podlators-0__4.11-1.el8.x86_64",
    sha256 = "78d17ed089151e7fa3d1a3cdbbac8ca3b1b5c484fae5ba025642cc9107991037",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-podlators-4.11-1.el8.noarch.rpm"],
)

rpm(
    name = "perl-threads-1__2.21-2.el8.x86_64",
    sha256 = "2e3da17b1c1685edea9c52bdaa0d77c019d6144c765fc6b3b1c783d98f634f96",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-threads-2.21-2.el8.x86_64.rpm"],
)

rpm(
    name = "perl-threads-shared-0__1.58-2.el8.x86_64",
    sha256 = "b4a14dc0e3550da946d7ca65e54d19fc805e30c6c3dbf5ef3fc077d1d94e6d71",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-threads-shared-1.58-2.el8.x86_64.rpm"],
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
    name = "platform-python-0__3.6.8-45.el8.aarch64",
    sha256 = "b66dd2f640bfb0ddbb5c84e5ff9530b2f0eb38c9e18061a4c1f3a5af0a8d66ad",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-3.6.8-45.el8.aarch64.rpm"],
)

rpm(
    name = "platform-python-0__3.6.8-45.el8.x86_64",
    sha256 = "0b09850c7ecaef3749c2632c12b8cdc7c60936dbea0c210a666bd31adb94c323",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-3.6.8-45.el8.x86_64.rpm"],
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
    name = "policycoreutils-0__2.9-18.el8.aarch64",
    sha256 = "6a2fe0aaca7614157468df9ee9ad3cafe8953178a090c442f568fb2eadaf966a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/policycoreutils-2.9-18.el8.aarch64.rpm"],
)

rpm(
    name = "policycoreutils-0__2.9-18.el8.x86_64",
    sha256 = "50576501b1eefc7d9dfd479857966d4ca8c7baea68465906aa35261a7bf0fb74",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/policycoreutils-2.9-18.el8.x86_64.rpm"],
)

rpm(
    name = "polkit-0__0.115-13.el8_5.1.aarch64",
    sha256 = "aabd3fa1c44fec3a95d5a9fe4b880a84a42a47723f440102ec098dc119e3f41e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-0.115-13.el8_5.1.aarch64.rpm"],
)

rpm(
    name = "polkit-0__0.115-13.el8_5.1.x86_64",
    sha256 = "cd873179894c7fc92ca30493f11bc00ca3505111bba32be0c54f0d4a49d5100d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-0.115-13.el8_5.1.x86_64.rpm"],
)

rpm(
    name = "polkit-libs-0__0.115-13.el8_5.1.aarch64",
    sha256 = "2ffe101b6f52d38686ce9ec5ed61f5952f7a9191c601c7617292cc717bbc497d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.115-13.el8_5.1.aarch64.rpm"],
)

rpm(
    name = "polkit-libs-0__0.115-13.el8_5.1.x86_64",
    sha256 = "5ad7045e373ec2bf1b32f287fd62019f6a319b79da71da789b83e376f8ba92f6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.115-13.el8_5.1.x86_64.rpm"],
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
    name = "psmisc-0__23.1-5.el8.aarch64",
    sha256 = "e6852f9e715174c037c57ef9ee45a6318775968322c244185fc51f40a10dbdcc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/psmisc-23.1-5.el8.aarch64.rpm"],
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
    name = "python3-libs-0__3.6.8-45.el8.aarch64",
    sha256 = "5c5cceb3bf66b2bd1c9cab3ae1f4c91efe4981caca30d6242c9227ce23f6b934",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-libs-3.6.8-45.el8.aarch64.rpm"],
)

rpm(
    name = "python3-libs-0__3.6.8-45.el8.x86_64",
    sha256 = "603aad60c3d0e7134a2dbbfe85ddb672441d87ae4469e4f1497125b671a7b13e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-libs-3.6.8-45.el8.x86_64.rpm"],
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
    name = "rpm-0__4.14.3-21.el8.aarch64",
    sha256 = "974a32aaa741a24f75f4b3964f4fad9b1e43d291c57ad681847506e788bcd40d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-4.14.3-21.el8.aarch64.rpm"],
)

rpm(
    name = "rpm-0__4.14.3-21.el8.x86_64",
    sha256 = "a80fa936288828b1b3050f56698cca2bea0df2cf3bfce6afa86f7a131192d916",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-4.14.3-21.el8.x86_64.rpm"],
)

rpm(
    name = "rpm-libs-0__4.14.3-21.el8.aarch64",
    sha256 = "4bd083cd5637ac04f24b139094aec54e8987c1924025d7ebe6d0c5d8a2783fd7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.14.3-21.el8.aarch64.rpm"],
)

rpm(
    name = "rpm-libs-0__4.14.3-21.el8.x86_64",
    sha256 = "cb17e2ad56305c99d1589a9965e34b62e342f7ee0348cdb17c852f65db24221e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.14.3-21.el8.x86_64.rpm"],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-21.el8.aarch64",
    sha256 = "d47fa2f66647f4d4da0a19d8b56377aff1d027f06590d6362a9172c04dc4a3dc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.14.3-21.el8.aarch64.rpm"],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-21.el8.x86_64",
    sha256 = "81e8dfbb428e82fb461f1e3de7aa429dfe031cad2109eef3d9bf4fa66ecf278d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.14.3-21.el8.x86_64.rpm"],
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
    name = "selinux-policy-0__3.14.3-91.el8.aarch64",
    sha256 = "ba891e3bf2fab177281bfe32d2de5239f34d9a3099e77162ae8464e31501e19a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-3.14.3-91.el8.noarch.rpm"],
)

rpm(
    name = "selinux-policy-0__3.14.3-91.el8.x86_64",
    sha256 = "ba891e3bf2fab177281bfe32d2de5239f34d9a3099e77162ae8464e31501e19a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-3.14.3-91.el8.noarch.rpm"],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-91.el8.aarch64",
    sha256 = "029f91ad75586cd5b49f8b743fce2c09af6ebc0bbf3d2dddf403a65225e1be07",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-3.14.3-91.el8.noarch.rpm"],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-91.el8.x86_64",
    sha256 = "029f91ad75586cd5b49f8b743fce2c09af6ebc0bbf3d2dddf403a65225e1be07",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-3.14.3-91.el8.noarch.rpm"],
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
    name = "shadow-utils-2__4.6-16.el8.aarch64",
    sha256 = "6873ddc4ebf0f2c35a7cf46c237f8b9068ac035e43b16e66c1247eac83373295",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.6-16.el8.aarch64.rpm"],
)

rpm(
    name = "shadow-utils-2__4.6-16.el8.x86_64",
    sha256 = "be4f208e508cc5277eed1c3644b9570defc7c99160e31eca0be0c293066fa535",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.6-16.el8.x86_64.rpm"],
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
    name = "sssd-client-0__2.6.1-2.el8.aarch64",
    sha256 = "7ada70443cabcd3524c2897d397168337b414cac9043b730c24c01e5a3652fba",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sssd-client-2.6.1-2.el8.aarch64.rpm"],
)

rpm(
    name = "sssd-client-0__2.6.1-2.el8.x86_64",
    sha256 = "a73f58c980440f764ed032f701da9d45e7d4f3c20f00f47a05153a29e22a5b20",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sssd-client-2.6.1-2.el8.x86_64.rpm"],
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
    name = "swtpm-0__0.7.0-1.20211109gitb79fd91.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "e004d8c0bf81bedece3539bea0e12961740ae7dcc69d39f3774d7a738902e063",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-0.7.0-1.20211109gitb79fd91.module_el8.6.0+1087+b42c8331.aarch64.rpm"],
)

rpm(
    name = "swtpm-0__0.7.0-1.20211109gitb79fd91.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "d062f5c3420aae57429124c44b9c204eea85d9e336f82b6a0b6b9b5b4043aa88",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-0.7.0-1.20211109gitb79fd91.module_el8.6.0+1087+b42c8331.x86_64.rpm"],
)

rpm(
    name = "swtpm-libs-0__0.7.0-1.20211109gitb79fd91.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "df014e3d5b857c279e5aac2c8147889710010e52abad3f0cef7945865533ec21",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.7.0-1.20211109gitb79fd91.module_el8.6.0+1087+b42c8331.aarch64.rpm"],
)

rpm(
    name = "swtpm-libs-0__0.7.0-1.20211109gitb79fd91.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "75a44e4cb73b21d811c93a610dcc67c1f03215d5d8fdb6eff83c0e162b63f569",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.7.0-1.20211109gitb79fd91.module_el8.6.0+1087+b42c8331.x86_64.rpm"],
)

rpm(
    name = "swtpm-tools-0__0.7.0-1.20211109gitb79fd91.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "dd923963bfa8c404d69dbc8d70014791355a08335930c3802d403a13b341838c",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.7.0-1.20211109gitb79fd91.module_el8.6.0+1087+b42c8331.aarch64.rpm"],
)

rpm(
    name = "swtpm-tools-0__0.7.0-1.20211109gitb79fd91.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "ac29006dfc30e4129a2ee0c11a08e32cec1e423bea26a473448507516d874c4b",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.7.0-1.20211109gitb79fd91.module_el8.6.0+1087+b42c8331.x86_64.rpm"],
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
    name = "systemd-0__239-58.el8.aarch64",
    sha256 = "d3388e47444ce1e624930ea6c1e2aef04c7571045ac459d0ee317163c20d6cbe",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-239-58.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-0__239-58.el8.x86_64",
    sha256 = "0a373ef0d76dc8cfa755ebd390f9897e563970e840de337248f61840731d42cc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-239-58.el8.x86_64.rpm"],
)

rpm(
    name = "systemd-container-0__239-58.el8.aarch64",
    sha256 = "f878467aa636e18793bf88aa2522896c1cc2c673102aed5a5f1c064d9c4c0dbd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-container-239-58.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-container-0__239-58.el8.x86_64",
    sha256 = "6ad9bf6369f0f7df0331440f900f1e5ce2f302ef3619240375144a030e49e2d2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-container-239-58.el8.x86_64.rpm"],
)

rpm(
    name = "systemd-libs-0__239-58.el8.aarch64",
    sha256 = "2e016e9b5cd5f61142f48c834e792e93ef6cbee3c1f4865a361ceeac9f53bf63",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-libs-239-58.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-libs-0__239-58.el8.x86_64",
    sha256 = "a48d2d7b308723d997d69397dc26cdc552caa2dc1a368a5c079827d2ee17ee2c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-libs-239-58.el8.x86_64.rpm"],
)

rpm(
    name = "systemd-pam-0__239-58.el8.aarch64",
    sha256 = "8f46c794df6d67ec5e70c7e7bd38e58018601305497a7bc3e450f2253b04b3dd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-pam-239-58.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-pam-0__239-58.el8.x86_64",
    sha256 = "32c0f886890f175632aae1fd16a88346362de8fe6d7c3d8b55f74edaa05caf0c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-pam-239-58.el8.x86_64.rpm"],
)

rpm(
    name = "systemd-udev-0__239-58.el8.x86_64",
    sha256 = "7ef0684b02c89ad411d3b7a1b8f4a5e2d17aa9612b601a8d1c850b58b0dd869d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-udev-239-58.el8.x86_64.rpm"],
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
    name = "usbredir-0__0.12.0-1.el8.x86_64",
    sha256 = "d1c88979f605d07c1750a0f4d29b8ad35cd8deaced40472e00fe380bf6b1ba57",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/usbredir-0.12.0-1.el8.x86_64.rpm"],
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
    name = "util-linux-0__2.32.1-32.el8.aarch64",
    sha256 = "6a6c95b7e6759b9e8b55830565fce6af31caedaf4c3f36f921d83531c4f8fe66",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/util-linux-2.32.1-32.el8.aarch64.rpm"],
)

rpm(
    name = "util-linux-0__2.32.1-32.el8.x86_64",
    sha256 = "b819f75e03f0d33d0e06c04a2624ff9436bdc694b59918fab0c736acc6119904",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/util-linux-2.32.1-32.el8.x86_64.rpm"],
)

rpm(
    name = "vim-minimal-2__8.0.1763-16.el8_5.7.aarch64",
    sha256 = "238590c5d546cc509a55383745fe22cf762c1fbcd454484d2c1b4149e0a71a4d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.0.1763-16.el8_5.7.aarch64.rpm"],
)

rpm(
    name = "vim-minimal-2__8.0.1763-16.el8_5.7.x86_64",
    sha256 = "d06ec97d21fd7e263b40649e78e1b3777460ec2a4ae9ec9cd2bae124d31e1409",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.0.1763-16.el8_5.7.x86_64.rpm"],
)

rpm(
    name = "which-0__2.21-17.el8.aarch64",
    sha256 = "35332bb903317e2300555f439b0eddd412f8731ce04e74a9272cfd7232c84f81",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/which-2.21-17.el8.aarch64.rpm"],
)

rpm(
    name = "which-0__2.21-17.el8.x86_64",
    sha256 = "0d761d50ec6ab263dde637c37b876b994f4a5394f6e2b3036e84563b66b75a3d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/which-2.21-17.el8.x86_64.rpm"],
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
