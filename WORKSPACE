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
    sha256 = "e97eaedb3bff39a081d1d7e67629d5c0e8fb39677d6a9dd1eaf2752e39061e02",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.15/releases/x86_64/alpine-virt-3.15.0-x86_64.iso",
        "https://storage.googleapis.com/builddeps/e97eaedb3bff39a081d1d7e67629d5c0e8fb39677d6a9dd1eaf2752e39061e02",
    ],
)

http_file(
    name = "alpine_image_aarch64",
    sha256 = "f302cf1b2dbbd0661b8f53b167f24131c781b86ab3ae059654db05cd62d3c39c",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.15/releases/aarch64/alpine-virt-3.15.0-aarch64.iso",
        "https://storage.googleapis.com/builddeps/f302cf1b2dbbd0661b8f53b167f24131c781b86ab3ae059654db05cd62d3c39c",
    ],
)

http_file(
    name = "cirros_image",
    sha256 = "932fcae93574e242dc3d772d5235061747dfe537668443a1f0567d893614b464",
    urls = [
        "https://download.cirros-cloud.net/0.5.2/cirros-0.5.2-x86_64-disk.img",
        "https://storage.googleapis.com/builddeps/932fcae93574e242dc3d772d5235061747dfe537668443a1f0567d893614b464",
    ],
)

http_file(
    name = "cirros_image_aarch64",
    sha256 = "889c1117647b3b16cfc47957931c6573bf8e755fc9098fdcad13727b6c9f2629",
    urls = [
        "https://download.cirros-cloud.net/0.5.2/cirros-0.5.2-aarch64-disk.img",
        "https://storage.googleapis.com/builddeps/889c1117647b3b16cfc47957931c6573bf8e755fc9098fdcad13727b6c9f2629",
    ],
)

http_file(
    name = "virtio_win_image",
    sha256 = "b8a4bc66835c43091a85d35a10b59bd8b1b62b55ea9f02ec754f68bd32e82c0e",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.217-1/virtio-win-0.1.217.iso",
        "https://storage.googleapis.com/builddeps/b8a4bc66835c43091a85d35a10b59bd8b1b62b55ea9f02ec754f68bd32e82c0e",
    ],
)

http_archive(
    name = "bazeldnf",
    sha256 = "c37709d05ad7eae4d32d7a525f098fd026483ada5e11cdf84d47028222796605",
    strip_prefix = "bazeldnf-0.5.2",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.5.2.tar.gz",
        "https://storage.googleapis.com/builddeps/c37709d05ad7eae4d32d7a525f098fd026483ada5e11cdf84d47028222796605",
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
    go_version = "1.17.8",
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

go_repository(
    name = "org_golang_x_sys",
    commit = "2964e1e4b1dbd55a8ac69a4c9e3004a8038515b6",
    importpath = "golang.org/x/sys",
)

# Winrmcli dependencies
go_repository(
    name = "com_github_masterzen_winrmcli",
    commit = "c85a68ee8b6e3ac95af2a5fd62d2f41c9e9c5f32",
    importpath = "github.com/masterzen/winrm-cli",
)

# Compress Dependency
go_repository(
    name = "com_github_klauspost_compress",
    commit = "67a538e2b4df11f8ec7139388838a13bce84b5d5",
    importpath = "github.com/klauspost/compress",
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
    digest = "sha256:da6c118dbd9ac643713c1737cbaa43dcc7386b269b4beb0984413168f3a5f2d3",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
)

container_pull(
    name = "alpine_with_test_tooling",
    digest = "sha256:d1dab23ed46af711acb33e54b1dd2a7c6dfaab24227346a487748057e2c81d11",
    registry = "quay.io",
    repository = "kubevirtci/alpine-with-test-tooling-container-disk",
    tag = "2206291207-35b9c64",
)

container_pull(
    name = "fedora_with_test_tooling_aarch64",
    digest = "sha256:9b1371260c05086a24ac9effdbedca9759c885ea8db93de7f0339df3bcd5a5c3",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
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
    name = "audit-libs-0__3.0.7-4.el8.aarch64",
    sha256 = "2b05f70005d024a2b540a56afd9e05729c07c9dee120ff01100a21e21781f017",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/audit-libs-3.0.7-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2b05f70005d024a2b540a56afd9e05729c07c9dee120ff01100a21e21781f017",
    ],
)

rpm(
    name = "audit-libs-0__3.0.7-4.el8.x86_64",
    sha256 = "b37099679b46f9a15d20b7c54fdd993388a8b84105f76869494c1be17140b512",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/audit-libs-3.0.7-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b37099679b46f9a15d20b7c54fdd993388a8b84105f76869494c1be17140b512",
    ],
)

rpm(
    name = "augeas-libs-0__1.12.0-7.el8.x86_64",
    sha256 = "672cf6c97f6aa00a0d5a39d20372501d6c6f40ac431083a499d89b7b25c84ba4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/augeas-libs-1.12.0-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/672cf6c97f6aa00a0d5a39d20372501d6c6f40ac431083a499d89b7b25c84ba4",
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
    name = "bash-0__4.4.20-4.el8.aarch64",
    sha256 = "cb47111790ede91e0f1fb34817a27123a97e0304e7f7b6df06731fd391859f45",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bash-4.4.20-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb47111790ede91e0f1fb34817a27123a97e0304e7f7b6df06731fd391859f45",
    ],
)

rpm(
    name = "bash-0__4.4.20-4.el8.x86_64",
    sha256 = "a104837b8aea5214122cf09c2de436db8f528812c1361c39f2d7471343dc509b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bash-4.4.20-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a104837b8aea5214122cf09c2de436db8f528812c1361c39f2d7471343dc509b",
    ],
)

rpm(
    name = "binutils-0__2.30-117.el8.aarch64",
    sha256 = "10cc7e5ae3939eb78ef345127f05428eb003482c91dff1506121bde6228ed55f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/binutils-2.30-117.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/10cc7e5ae3939eb78ef345127f05428eb003482c91dff1506121bde6228ed55f",
    ],
)

rpm(
    name = "binutils-0__2.30-117.el8.x86_64",
    sha256 = "d5c059ff1e586a5c7f581f916529f715b24d89bdf77e831f930306957f8870ed",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/binutils-2.30-117.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5c059ff1e586a5c7f581f916529f715b24d89bdf77e831f930306957f8870ed",
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
    name = "ca-certificates-0__2022.2.54-80.2.el8.aarch64",
    sha256 = "3200d42d5585afa93a94600614a82b6e804139b06fff151576a53effd221e12b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ca-certificates-2022.2.54-80.2.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3200d42d5585afa93a94600614a82b6e804139b06fff151576a53effd221e12b",
    ],
)

rpm(
    name = "ca-certificates-0__2022.2.54-80.2.el8.x86_64",
    sha256 = "3200d42d5585afa93a94600614a82b6e804139b06fff151576a53effd221e12b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ca-certificates-2022.2.54-80.2.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3200d42d5585afa93a94600614a82b6e804139b06fff151576a53effd221e12b",
    ],
)

rpm(
    name = "centos-gpg-keys-1__8-6.el8.aarch64",
    sha256 = "567dd699e703dc6f5fa6ddb5548bf0dbd3bda08a0a6b1d10b32fa19012409cd0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-8-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/567dd699e703dc6f5fa6ddb5548bf0dbd3bda08a0a6b1d10b32fa19012409cd0",
    ],
)

rpm(
    name = "centos-gpg-keys-1__8-6.el8.x86_64",
    sha256 = "567dd699e703dc6f5fa6ddb5548bf0dbd3bda08a0a6b1d10b32fa19012409cd0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-8-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/567dd699e703dc6f5fa6ddb5548bf0dbd3bda08a0a6b1d10b32fa19012409cd0",
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
    name = "centos-stream-repos-0__8-6.el8.aarch64",
    sha256 = "ff0a2d1fb5b00e9a26b05a82675d0dcdf0378ee5476f9ae765b32399c2ee561f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-8-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ff0a2d1fb5b00e9a26b05a82675d0dcdf0378ee5476f9ae765b32399c2ee561f",
    ],
)

rpm(
    name = "centos-stream-repos-0__8-6.el8.x86_64",
    sha256 = "ff0a2d1fb5b00e9a26b05a82675d0dcdf0378ee5476f9ae765b32399c2ee561f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-8-6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ff0a2d1fb5b00e9a26b05a82675d0dcdf0378ee5476f9ae765b32399c2ee561f",
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
    name = "coreutils-single-0__8.30-13.el8.aarch64",
    sha256 = "0f560179f5b79ee62e0d71efb8d67f0d8eca9b31b752064a507c1052985e1251",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.30-13.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0f560179f5b79ee62e0d71efb8d67f0d8eca9b31b752064a507c1052985e1251",
    ],
)

rpm(
    name = "coreutils-single-0__8.30-13.el8.x86_64",
    sha256 = "8a8a3a45697389d029d439711c65969408ebbf4ba4d7c573d6dbe6f2b26b439d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.30-13.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a8a3a45697389d029d439711c65969408ebbf4ba4d7c573d6dbe6f2b26b439d",
    ],
)

rpm(
    name = "cpp-0__8.5.0-15.el8.aarch64",
    sha256 = "36bb703e9305764b2075c56d79f98d4ff86a8a9dbcb59c2ce2a8eef37b4b98a2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/cpp-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/36bb703e9305764b2075c56d79f98d4ff86a8a9dbcb59c2ce2a8eef37b4b98a2",
    ],
)

rpm(
    name = "cpp-0__8.5.0-15.el8.x86_64",
    sha256 = "1484662ef1bc1e6770c2aa8be9753e73bac8a5623c3841b6f27809c1b53989b5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/cpp-8.5.0-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1484662ef1bc1e6770c2aa8be9753e73bac8a5623c3841b6f27809c1b53989b5",
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
    name = "crypto-policies-0__20211116-1.gitae470d6.el8.aarch64",
    sha256 = "8fb69892af346bacf18e8f8e7e8098e09c6ef9547abab9c39f7e729db06c3d1e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/crypto-policies-20211116-1.gitae470d6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8fb69892af346bacf18e8f8e7e8098e09c6ef9547abab9c39f7e729db06c3d1e",
    ],
)

rpm(
    name = "crypto-policies-0__20211116-1.gitae470d6.el8.x86_64",
    sha256 = "8fb69892af346bacf18e8f8e7e8098e09c6ef9547abab9c39f7e729db06c3d1e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/crypto-policies-20211116-1.gitae470d6.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8fb69892af346bacf18e8f8e7e8098e09c6ef9547abab9c39f7e729db06c3d1e",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.7-2.el8.aarch64",
    sha256 = "15a9d91ba7f5c192bee3e0d511e9b501c109a53c68120987e3f79ed88b1f69b5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cryptsetup-libs-2.3.7-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/15a9d91ba7f5c192bee3e0d511e9b501c109a53c68120987e3f79ed88b1f69b5",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.7-2.el8.x86_64",
    sha256 = "6fe218c49155d7b22cd97156583b98d08abfbbffb61c32fe1965a0683ab7ed9e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-libs-2.3.7-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6fe218c49155d7b22cd97156583b98d08abfbbffb61c32fe1965a0683ab7ed9e",
    ],
)

rpm(
    name = "curl-0__7.61.1-25.el8.aarch64",
    sha256 = "56d7d77a32456f4c6b84ae4c6251d7ddfe2fb7097f9ecf8ba5e5834f7b7611c7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/curl-7.61.1-25.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56d7d77a32456f4c6b84ae4c6251d7ddfe2fb7097f9ecf8ba5e5834f7b7611c7",
    ],
)

rpm(
    name = "curl-0__7.61.1-25.el8.x86_64",
    sha256 = "6d5a740367b807f9cb102f9f3868ddd102c330944654a2903a016f651a6c25ed",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/curl-7.61.1-25.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6d5a740367b807f9cb102f9f3868ddd102c330944654a2903a016f651a6c25ed",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-6.el8_5.aarch64",
    sha256 = "e7acd635ac3d42260807c3fd6eab8713e3177b88bceadd79fe10d0719bfbff00",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-2.1.27-6.el8_5.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e7acd635ac3d42260807c3fd6eab8713e3177b88bceadd79fe10d0719bfbff00",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-6.el8_5.x86_64",
    sha256 = "65a62affe9c99e597aabf117b8439a363761686c496723bc492dbfdcb6f60692",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-2.1.27-6.el8_5.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65a62affe9c99e597aabf117b8439a363761686c496723bc492dbfdcb6f60692",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-6.el8_5.aarch64",
    sha256 = "9fac42ea86802ebaf480d7373155a019d0a85dfd8093189d17194334af466a15",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-gssapi-2.1.27-6.el8_5.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9fac42ea86802ebaf480d7373155a019d0a85dfd8093189d17194334af466a15",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-6.el8_5.x86_64",
    sha256 = "6c9a8d9adc93d1be7db41fe7327c4dcce144cefad3008e580f5e9cadb6155eb4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-gssapi-2.1.27-6.el8_5.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6c9a8d9adc93d1be7db41fe7327c4dcce144cefad3008e580f5e9cadb6155eb4",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-6.el8_5.aarch64",
    sha256 = "984998500ff0d60cb8756fee9eaeb82a001b7323b1130955770f2fa824f8a937",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-lib-2.1.27-6.el8_5.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/984998500ff0d60cb8756fee9eaeb82a001b7323b1130955770f2fa824f8a937",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-6.el8_5.x86_64",
    sha256 = "5bd6e1201d8b10c6f01f500c43f63204f1d2ec8a4d8ce53c741e611c81ffb404",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-lib-2.1.27-6.el8_5.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5bd6e1201d8b10c6f01f500c43f63204f1d2ec8a4d8ce53c741e611c81ffb404",
    ],
)

rpm(
    name = "daxctl-libs-0__71.1-4.el8.x86_64",
    sha256 = "332af3c063fdb03d95632dc5010712c4e9ca7416f3049c901558c5aa0c6e445b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/daxctl-libs-71.1-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/332af3c063fdb03d95632dc5010712c4e9ca7416f3049c901558c5aa0c6e445b",
    ],
)

rpm(
    name = "dbus-1__1.12.8-23.el8.aarch64",
    sha256 = "687dc9e92456cf34d3caf73b37b9a9ae5acc075aba6dbbbecc74a31bd2c6eab1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-1.12.8-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/687dc9e92456cf34d3caf73b37b9a9ae5acc075aba6dbbbecc74a31bd2c6eab1",
    ],
)

rpm(
    name = "dbus-1__1.12.8-23.el8.x86_64",
    sha256 = "72745a2e75f4d7f7c85dd5a92c57a95ef9850cd9286e98aa52bd7e629e7487bb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-1.12.8-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/72745a2e75f4d7f7c85dd5a92c57a95ef9850cd9286e98aa52bd7e629e7487bb",
    ],
)

rpm(
    name = "dbus-common-1__1.12.8-23.el8.aarch64",
    sha256 = "3f5a3dbca29172f117e43d2551f0b80507ca29eed07c5d35b0374b6a5feff657",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.8-23.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3f5a3dbca29172f117e43d2551f0b80507ca29eed07c5d35b0374b6a5feff657",
    ],
)

rpm(
    name = "dbus-common-1__1.12.8-23.el8.x86_64",
    sha256 = "3f5a3dbca29172f117e43d2551f0b80507ca29eed07c5d35b0374b6a5feff657",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.8-23.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3f5a3dbca29172f117e43d2551f0b80507ca29eed07c5d35b0374b6a5feff657",
    ],
)

rpm(
    name = "dbus-daemon-1__1.12.8-23.el8.aarch64",
    sha256 = "0b0a27298b5cd803e0344ce7e4a55ab157ecb6e7e9197e826d5b40c0d92649a8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-daemon-1.12.8-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0b0a27298b5cd803e0344ce7e4a55ab157ecb6e7e9197e826d5b40c0d92649a8",
    ],
)

rpm(
    name = "dbus-daemon-1__1.12.8-23.el8.x86_64",
    sha256 = "125870bc9d4a010c4d105dea4a5e2efb4344b5a5d43aeb5fdb80436e7bf00a08",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-daemon-1.12.8-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/125870bc9d4a010c4d105dea4a5e2efb4344b5a5d43aeb5fdb80436e7bf00a08",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.8-23.el8.aarch64",
    sha256 = "31cb3418fc47087230b4b6bbba65a81e34e690f25b716e8604f883de1953a5c5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.12.8-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/31cb3418fc47087230b4b6bbba65a81e34e690f25b716e8604f883de1953a5c5",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.8-23.el8.x86_64",
    sha256 = "7739e3a34748a97adde197cbbafe1b111fd0577aa3eb58b0e997e65e1fbbe970",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.8-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7739e3a34748a97adde197cbbafe1b111fd0577aa3eb58b0e997e65e1fbbe970",
    ],
)

rpm(
    name = "dbus-tools-1__1.12.8-23.el8.aarch64",
    sha256 = "a5697ac626a89e0623fc131db9b0ae07d885d410f29fce2443df1df5ce9be8ef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-tools-1.12.8-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a5697ac626a89e0623fc131db9b0ae07d885d410f29fce2443df1df5ce9be8ef",
    ],
)

rpm(
    name = "dbus-tools-1__1.12.8-23.el8.x86_64",
    sha256 = "b0c78431c478695ed5d4f14b08af7d89d79804638815eccb7bd8116a482aba88",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-tools-1.12.8-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b0c78431c478695ed5d4f14b08af7d89d79804638815eccb7bd8116a482aba88",
    ],
)

rpm(
    name = "device-mapper-8__1.02.181-6.el8.aarch64",
    sha256 = "05ca821f4cef038bb994d59b1bbd7feebcba7ed6089aab0debf79ba759768a47",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.181-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/05ca821f4cef038bb994d59b1bbd7feebcba7ed6089aab0debf79ba759768a47",
    ],
)

rpm(
    name = "device-mapper-8__1.02.181-6.el8.x86_64",
    sha256 = "8e89a7c9e0b011917c8c360e625c3a2bfa3a81b82e9ac961977aa09a98b9da27",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.181-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e89a7c9e0b011917c8c360e625c3a2bfa3a81b82e9ac961977aa09a98b9da27",
    ],
)

rpm(
    name = "device-mapper-event-8__1.02.181-6.el8.x86_64",
    sha256 = "a89244d4420b679d3d567e09578e68ff4159b7eec3968a3ecc39a9c0521c5ec3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-1.02.181-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a89244d4420b679d3d567e09578e68ff4159b7eec3968a3ecc39a9c0521c5ec3",
    ],
)

rpm(
    name = "device-mapper-event-libs-8__1.02.181-6.el8.x86_64",
    sha256 = "e759885e81245b164bf418caf38f2358eebe56f68094d848f915ba957cf04a47",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-libs-1.02.181-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e759885e81245b164bf418caf38f2358eebe56f68094d848f915ba957cf04a47",
    ],
)

rpm(
    name = "device-mapper-libs-8__1.02.181-6.el8.aarch64",
    sha256 = "53d03a64bcbb33297eaa744b61d3bfddf001c0bcfdf263729236f3fec85a1b3c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.181-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/53d03a64bcbb33297eaa744b61d3bfddf001c0bcfdf263729236f3fec85a1b3c",
    ],
)

rpm(
    name = "device-mapper-libs-8__1.02.181-6.el8.x86_64",
    sha256 = "bdd83ee5a458034908007edae7c55aa7ebc1138f0306877356f9d2f1215dd065",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.181-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bdd83ee5a458034908007edae7c55aa7ebc1138f0306877356f9d2f1215dd065",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-28.el8.aarch64",
    sha256 = "92aafe5d2c90d6b265284e30a7df557a103ebdd6b56106450830382979569fd1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.4-28.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/92aafe5d2c90d6b265284e30a7df557a103ebdd6b56106450830382979569fd1",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-28.el8.x86_64",
    sha256 = "83d7f1a1df87d5b22d6f9e640ec5054b3b2afcb36396fa9c5c4d00e3de42280a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.4-28.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/83d7f1a1df87d5b22d6f9e640ec5054b3b2afcb36396fa9c5c4d00e3de42280a",
    ],
)

rpm(
    name = "device-mapper-persistent-data-0__0.9.0-7.el8.x86_64",
    sha256 = "609c2bf12ce2994a0753177e334cde294a96750903c24d8583e7a0674c80485e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-persistent-data-0.9.0-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/609c2bf12ce2994a0753177e334cde294a96750903c24d8583e7a0674c80485e",
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
    name = "dmidecode-1__3.3-4.el8.x86_64",
    sha256 = "c1347fe2d5621a249ea230e9e8ff2774e538031070a225245154a75428ec67a5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dmidecode-3.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c1347fe2d5621a249ea230e9e8ff2774e538031070a225245154a75428ec67a5",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.6-5.el8.aarch64",
    sha256 = "b916de2e7ea8fc3b0b381e0afe4353ab401b82885cea5afec0551232beb30fe2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.45.6-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b916de2e7ea8fc3b0b381e0afe4353ab401b82885cea5afec0551232beb30fe2",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.6-5.el8.x86_64",
    sha256 = "baa1ec089da85bf196f6e1e135727bb540f27ee7fe39d08bb17b712e59f4db8a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.45.6-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/baa1ec089da85bf196f6e1e135727bb540f27ee7fe39d08bb17b712e59f4db8a",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-5.el8.aarch64",
    sha256 = "0ec196d820abc43432cfa52c887c880b27b63619c6785dc30daed0e091c5bb76",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.45.6-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0ec196d820abc43432cfa52c887c880b27b63619c6785dc30daed0e091c5bb76",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.6-5.el8.x86_64",
    sha256 = "035c5ed68339e632907c3f952098cdc9181ab9138239473903000e6a50446d98",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.45.6-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/035c5ed68339e632907c3f952098cdc9181ab9138239473903000e6a50446d98",
    ],
)

rpm(
    name = "edk2-aarch64-0__20220126gitbb1bba3d77-2.el8.aarch64",
    sha256 = "0985ef697fbe90b66dbb0f70bfb4d0022f97255a36479e8d9ae4dd0489afd01a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20220126gitbb1bba3d77-2.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0985ef697fbe90b66dbb0f70bfb4d0022f97255a36479e8d9ae4dd0489afd01a",
    ],
)

rpm(
    name = "edk2-ovmf-0__20220126gitbb1bba3d77-2.el8.x86_64",
    sha256 = "a360d8e0ac13460ebab244e3063d6a9e2fb4d3a6bc2eb501534e5bfe9d0cff1e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20220126gitbb1bba3d77-2.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a360d8e0ac13460ebab244e3063d6a9e2fb4d3a6bc2eb501534e5bfe9d0cff1e",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.187-4.el8.aarch64",
    sha256 = "3c89377bb7409293f0dc8ada62071fe2e3cf042ae2b5ca7cf09faf77394b5187",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.187-4.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c89377bb7409293f0dc8ada62071fe2e3cf042ae2b5ca7cf09faf77394b5187",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.187-4.el8.x86_64",
    sha256 = "3c89377bb7409293f0dc8ada62071fe2e3cf042ae2b5ca7cf09faf77394b5187",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.187-4.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c89377bb7409293f0dc8ada62071fe2e3cf042ae2b5ca7cf09faf77394b5187",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.187-4.el8.aarch64",
    sha256 = "bfdfc37f2dd1052d4067937724a6ef6a9858a9c1b3c1aacf1e9085a83e99e1b4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.187-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bfdfc37f2dd1052d4067937724a6ef6a9858a9c1b3c1aacf1e9085a83e99e1b4",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.187-4.el8.x86_64",
    sha256 = "39d8cbfb137ca9044c258b5fa2129d2a953cc180cab225e843fd46a9267ee8a3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.187-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/39d8cbfb137ca9044c258b5fa2129d2a953cc180cab225e843fd46a9267ee8a3",
    ],
)

rpm(
    name = "elfutils-libs-0__0.187-4.el8.aarch64",
    sha256 = "682c1b9f11d68cdec87ea746ea0d5861f3afcf2159aa732854625bfa180bbaee",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.187-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/682c1b9f11d68cdec87ea746ea0d5861f3afcf2159aa732854625bfa180bbaee",
    ],
)

rpm(
    name = "elfutils-libs-0__0.187-4.el8.x86_64",
    sha256 = "ab96131314dbe1ed50f6a2086c0103ceb2e981e71f644ef95d3334a624723a22",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.187-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ab96131314dbe1ed50f6a2086c0103ceb2e981e71f644ef95d3334a624723a22",
    ],
)

rpm(
    name = "ethtool-2__5.13-2.el8.aarch64",
    sha256 = "5bdb69b9c4161ba3d4846082686ee8edce640b7c6ff0bbf1c1eae12084661c24",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ethtool-5.13-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5bdb69b9c4161ba3d4846082686ee8edce640b7c6ff0bbf1c1eae12084661c24",
    ],
)

rpm(
    name = "ethtool-2__5.13-2.el8.x86_64",
    sha256 = "f1af67b33961ddf98360e5ce855910d2dee534bffe953068f27ad96b846a2fb7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ethtool-5.13-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1af67b33961ddf98360e5ce855910d2dee534bffe953068f27ad96b846a2fb7",
    ],
)

rpm(
    name = "expat-0__2.2.5-9.el8.aarch64",
    sha256 = "4ca97fb015687a8f2ac442f581d1c42154662b4336e0f34c71be2659cb716fc8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/expat-2.2.5-9.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4ca97fb015687a8f2ac442f581d1c42154662b4336e0f34c71be2659cb716fc8",
    ],
)

rpm(
    name = "expat-0__2.2.5-9.el8.x86_64",
    sha256 = "a24088d02bfc25fb2efc1cc8c92e716ead35b38c8a96e69d08a9c78a5782f0e8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/expat-2.2.5-9.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a24088d02bfc25fb2efc1cc8c92e716ead35b38c8a96e69d08a9c78a5782f0e8",
    ],
)

rpm(
    name = "file-0__5.33-21.el8.x86_64",
    sha256 = "202e8164df8a6110d58692fa25eaf1d1078a988372943ae73536333237dc3818",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/file-5.33-21.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/202e8164df8a6110d58692fa25eaf1d1078a988372943ae73536333237dc3818",
    ],
)

rpm(
    name = "file-libs-0__5.33-21.el8.x86_64",
    sha256 = "9a51006d0e557e456eb9fc03ff7ed236633d32823dbd46984aca96f379e09f21",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/file-libs-5.33-21.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a51006d0e557e456eb9fc03ff7ed236633d32823dbd46984aca96f379e09f21",
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
    name = "fuse-0__2.9.7-16.el8.x86_64",
    sha256 = "c208aa2f2f216a2172b1d9fa82bcad1b201e62f9a3101f4d52fb3de54ed28596",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-2.9.7-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c208aa2f2f216a2172b1d9fa82bcad1b201e62f9a3101f4d52fb3de54ed28596",
    ],
)

rpm(
    name = "fuse-common-0__3.3.0-16.el8.x86_64",
    sha256 = "d637dfd117080f52f1a60444b6c09aaf65a535844cacce05945d1d691b8d7043",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-common-3.3.0-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d637dfd117080f52f1a60444b6c09aaf65a535844cacce05945d1d691b8d7043",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.7-16.el8.aarch64",
    sha256 = "6970abceb1e040a2a37a13faeaf2a4204c79a57d5bc8273ed276b385be813afb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/fuse-libs-2.9.7-16.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6970abceb1e040a2a37a13faeaf2a4204c79a57d5bc8273ed276b385be813afb",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.7-16.el8.x86_64",
    sha256 = "77fff0f92a55307b7df2334bc9cc2998c024586abd96286a251919b0509f0473",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.7-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/77fff0f92a55307b7df2334bc9cc2998c024586abd96286a251919b0509f0473",
    ],
)

rpm(
    name = "gawk-0__4.2.1-4.el8.aarch64",
    sha256 = "75594a09076ad901d5afb1027c74aae945f77e0e357e7d4f46148cbcbd1d0ae4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gawk-4.2.1-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/75594a09076ad901d5afb1027c74aae945f77e0e357e7d4f46148cbcbd1d0ae4",
    ],
)

rpm(
    name = "gawk-0__4.2.1-4.el8.x86_64",
    sha256 = "ff4438c2dff5bf933d7874fd55f131ca6ee067f8fb4324c89719d63e60b40aba",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gawk-4.2.1-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff4438c2dff5bf933d7874fd55f131ca6ee067f8fb4324c89719d63e60b40aba",
    ],
)

rpm(
    name = "gcc-0__8.5.0-15.el8.aarch64",
    sha256 = "347dbe82b51689eda62164b0ffdabb2dadf26f170c7430c32936d3ee87a67693",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gcc-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/347dbe82b51689eda62164b0ffdabb2dadf26f170c7430c32936d3ee87a67693",
    ],
)

rpm(
    name = "gcc-0__8.5.0-15.el8.x86_64",
    sha256 = "3ff2903895a5b75d737de8926ddfb31d01e05be07ab60b11ad168b761b14e9fc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gcc-8.5.0-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3ff2903895a5b75d737de8926ddfb31d01e05be07ab60b11ad168b761b14e9fc",
    ],
)

rpm(
    name = "gdbm-1__1.18-2.el8.aarch64",
    sha256 = "c032e3863180bb2247ddc0e02cd54be72099137af21452e2dc25ddd03f9a5395",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gdbm-1.18-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c032e3863180bb2247ddc0e02cd54be72099137af21452e2dc25ddd03f9a5395",
    ],
)

rpm(
    name = "gdbm-1__1.18-2.el8.x86_64",
    sha256 = "fa1751b26519b9637cf3f0a25ea1874eb2df005dde1e1371a3f13d0c9a38b9ca",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdbm-1.18-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fa1751b26519b9637cf3f0a25ea1874eb2df005dde1e1371a3f13d0c9a38b9ca",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18-2.el8.aarch64",
    sha256 = "bdb64aec2a4ea8a2c70652cd57e5f88353079042402e7662e0e89934d3737562",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gdbm-libs-1.18-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bdb64aec2a4ea8a2c70652cd57e5f88353079042402e7662e0e89934d3737562",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18-2.el8.x86_64",
    sha256 = "eddcea96342c8cfaa60b79fc2c66cb8c5b0038c3b11855abe55e659b2cad6199",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.18-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eddcea96342c8cfaa60b79fc2c66cb8c5b0038c3b11855abe55e659b2cad6199",
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
    name = "glib2-0__2.56.4-159.el8.aarch64",
    sha256 = "daac37a432b09faa6dd1e330c3595f6a70c53bff23a71fbce8df33c72e9fde24",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glib2-2.56.4-159.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/daac37a432b09faa6dd1e330c3595f6a70c53bff23a71fbce8df33c72e9fde24",
    ],
)

rpm(
    name = "glib2-0__2.56.4-159.el8.x86_64",
    sha256 = "d4b34f328efd6f144c8c1bcb61b6faa1318c367302b9f95d5db84078ca96a730",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glib2-2.56.4-159.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d4b34f328efd6f144c8c1bcb61b6faa1318c367302b9f95d5db84078ca96a730",
    ],
)

rpm(
    name = "glibc-0__2.28-211.el8.aarch64",
    sha256 = "7adf1cf7941e41077fdb294568638fe4ccefe685f7e767be7a82768709af0916",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-2.28-211.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7adf1cf7941e41077fdb294568638fe4ccefe685f7e767be7a82768709af0916",
    ],
)

rpm(
    name = "glibc-0__2.28-211.el8.x86_64",
    sha256 = "af5414cd755e6efd1f6ff7242c53bde389dd2c80f62a0e7fc03340c2b4036adc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-2.28-211.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/af5414cd755e6efd1f6ff7242c53bde389dd2c80f62a0e7fc03340c2b4036adc",
    ],
)

rpm(
    name = "glibc-common-0__2.28-211.el8.aarch64",
    sha256 = "2b5dec4d1cd079511561525828d5ce782269fd5b6e5bd3d2f630b2dd9dd5386c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-common-2.28-211.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2b5dec4d1cd079511561525828d5ce782269fd5b6e5bd3d2f630b2dd9dd5386c",
    ],
)

rpm(
    name = "glibc-common-0__2.28-211.el8.x86_64",
    sha256 = "0c3ab4d5ead2eacf9d3d313889dd0b4549824627d451c8156e2af74d2a30acbb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-common-2.28-211.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c3ab4d5ead2eacf9d3d313889dd0b4549824627d451c8156e2af74d2a30acbb",
    ],
)

rpm(
    name = "glibc-devel-0__2.28-211.el8.aarch64",
    sha256 = "76f98c8a73275625506863434abb0630e988ec67d74c29c9327e6ab9c69fd367",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-devel-2.28-211.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76f98c8a73275625506863434abb0630e988ec67d74c29c9327e6ab9c69fd367",
    ],
)

rpm(
    name = "glibc-devel-0__2.28-211.el8.x86_64",
    sha256 = "272d8ead57ef88b56833714dde7f366382344107999866d259d7404ebe811308",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-devel-2.28-211.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/272d8ead57ef88b56833714dde7f366382344107999866d259d7404ebe811308",
    ],
)

rpm(
    name = "glibc-headers-0__2.28-211.el8.aarch64",
    sha256 = "b1316336d7cce30779121338562d21e4514f720bd17686e8f5cb2177895d9fdb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-headers-2.28-211.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b1316336d7cce30779121338562d21e4514f720bd17686e8f5cb2177895d9fdb",
    ],
)

rpm(
    name = "glibc-headers-0__2.28-211.el8.x86_64",
    sha256 = "0cf90ea194bd6ac9140bd2cfc9a41a028592fad300c4b8bd73ca2fa0e7f8d749",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-headers-2.28-211.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0cf90ea194bd6ac9140bd2cfc9a41a028592fad300c4b8bd73ca2fa0e7f8d749",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-211.el8.aarch64",
    sha256 = "3607d6a967633522a885ee242911f21d59a1773c05ee06aa850151b5b923e197",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.28-211.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3607d6a967633522a885ee242911f21d59a1773c05ee06aa850151b5b923e197",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-211.el8.x86_64",
    sha256 = "7c5457b19f950a2afb834a106a9c9023564e53a62424d083e90cdba0d042a66e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.28-211.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7c5457b19f950a2afb834a106a9c9023564e53a62424d083e90cdba0d042a66e",
    ],
)

rpm(
    name = "glibc-static-0__2.28-211.el8.aarch64",
    sha256 = "03d8ff6274c07605abfc765e9205bd9f2ea141e10e805828c128f0834fec3282",
    urls = [
        "http://mirror.centos.org/centos/8-stream/PowerTools/aarch64/os/Packages/glibc-static-2.28-211.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/03d8ff6274c07605abfc765e9205bd9f2ea141e10e805828c128f0834fec3282",
    ],
)

rpm(
    name = "glibc-static-0__2.28-211.el8.x86_64",
    sha256 = "6b8d33d052d5b21897d828f381d18f36bb6d956d7c8630ee14d7554ca7daebb8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/PowerTools/x86_64/os/Packages/glibc-static-2.28-211.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6b8d33d052d5b21897d828f381d18f36bb6d956d7c8630ee14d7554ca7daebb8",
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
    name = "gnupg2-0__2.2.20-3.el8.x86_64",
    sha256 = "8c44c980dd9a6a42ccb93578d7e6e1940d36d2da0a5a99d783189c43b2ad6d5f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gnupg2-2.2.20-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c44c980dd9a6a42ccb93578d7e6e1940d36d2da0a5a99d783189c43b2ad6d5f",
    ],
)

rpm(
    name = "gnutls-0__3.6.16-5.el8.aarch64",
    sha256 = "6116c9afcae8723b1c985df5be06a2ce729eff8231800bd61d03758f9b249463",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gnutls-3.6.16-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6116c9afcae8723b1c985df5be06a2ce729eff8231800bd61d03758f9b249463",
    ],
)

rpm(
    name = "gnutls-0__3.6.16-5.el8.x86_64",
    sha256 = "4bd8fc9616f01f02cf1b17cccf4ae4d072f5adbd0c159b04203c87e8fb74b013",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gnutls-3.6.16-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4bd8fc9616f01f02cf1b17cccf4ae4d072f5adbd0c159b04203c87e8fb74b013",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.16-5.el8.aarch64",
    sha256 = "a768b99f8d974c192e1429a6822da3c79e866edd9d56c39cd787235cf6b110de",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.6.16-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a768b99f8d974c192e1429a6822da3c79e866edd9d56c39cd787235cf6b110de",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.16-5.el8.x86_64",
    sha256 = "bb65a2fb02d9d77c983ae1ecd2a64b211c96804d25fdc8e5b6575a8a19d8c59e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.6.16-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb65a2fb02d9d77c983ae1ecd2a64b211c96804d25fdc8e5b6575a8a19d8c59e",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.16-5.el8.aarch64",
    sha256 = "b925f5665d796db4f9a18e8df9dd911035fd49705b3a0b75b274bd8e83b4a2b0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.6.16-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b925f5665d796db4f9a18e8df9dd911035fd49705b3a0b75b274bd8e83b4a2b0",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.16-5.el8.x86_64",
    sha256 = "fc7abd04a01d77c7f0207b4ffd1edbd9a5ebdfb2e5154351abb481a11fdaf534",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.6.16-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fc7abd04a01d77c7f0207b4ffd1edbd9a5ebdfb2e5154351abb481a11fdaf534",
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
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/groff-base-1.22.3-18.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b00855013100d3796e9ed6d82b1ab2d4dc7f4a3a3fa2e186f6de8523577974a0",
    ],
)

rpm(
    name = "gzip-0__1.9-13.el8.aarch64",
    sha256 = "80ee79fb497c43c06d3c54bf432e6391c5ae19ae43241111f3be4113ea49fa96",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gzip-1.9-13.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/80ee79fb497c43c06d3c54bf432e6391c5ae19ae43241111f3be4113ea49fa96",
    ],
)

rpm(
    name = "gzip-0__1.9-13.el8.x86_64",
    sha256 = "1cc189e4991fc6b3526f7eebc9f798b8922e70d60a12ba499b6e0329eb473cea",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gzip-1.9-13.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cc189e4991fc6b3526f7eebc9f798b8922e70d60a12ba499b6e0329eb473cea",
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
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/info-6.5-7.el8_5.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/24a7e6f02ac095d965832203d0c8a9ee13aea301ef8572bb1ecdace435c796be",
    ],
)

rpm(
    name = "info-0__6.5-7.el8_5.x86_64",
    sha256 = "63f03261cc8109b2fb61002ca50c93e52acb9cfd8382d139e8de6623394051e8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/info-6.5-7.el8_5.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/63f03261cc8109b2fb61002ca50c93e52acb9cfd8382d139e8de6623394051e8",
    ],
)

rpm(
    name = "iproute-0__5.18.0-1.el8.aarch64",
    sha256 = "7ec84f47ebaed2388e48e27d9566a43609c7c384bbfbc3f0497c6bc314f618a5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-5.18.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7ec84f47ebaed2388e48e27d9566a43609c7c384bbfbc3f0497c6bc314f618a5",
    ],
)

rpm(
    name = "iproute-0__5.18.0-1.el8.x86_64",
    sha256 = "7ae4b834f060d111db19fa3cf6f6266d4c6fb56992b0347145799d7ff9f03d3c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-5.18.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7ae4b834f060d111db19fa3cf6f6266d4c6fb56992b0347145799d7ff9f03d3c",
    ],
)

rpm(
    name = "iproute-tc-0__5.18.0-1.el8.aarch64",
    sha256 = "8696d818b8ead9df0a2d66cf8e1fe03affd19899dd86e451267603faade5a161",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-tc-5.18.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8696d818b8ead9df0a2d66cf8e1fe03affd19899dd86e451267603faade5a161",
    ],
)

rpm(
    name = "iproute-tc-0__5.18.0-1.el8.x86_64",
    sha256 = "bca80255b377f2a715c1fa2023485cd8fd03f2bab2a873faa0e5879082bca1c9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-tc-5.18.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bca80255b377f2a715c1fa2023485cd8fd03f2bab2a873faa0e5879082bca1c9",
    ],
)

rpm(
    name = "iptables-0__1.8.4-23.el8.aarch64",
    sha256 = "09f12f3637e229c11481e965306dc056664904663a28983e2a06f6a987ccde96",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-1.8.4-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/09f12f3637e229c11481e965306dc056664904663a28983e2a06f6a987ccde96",
    ],
)

rpm(
    name = "iptables-0__1.8.4-23.el8.x86_64",
    sha256 = "edcfa2553fd55051814fb8c0806ae61eda042c49f3f8c5c7b91ce91d567c6170",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-1.8.4-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/edcfa2553fd55051814fb8c0806ae61eda042c49f3f8c5c7b91ce91d567c6170",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-23.el8.aarch64",
    sha256 = "f16feb8722435e81f025ba4a05d8e3b970cb0adbc1d1da6ba399d7f3a6d5b6f8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.4-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f16feb8722435e81f025ba4a05d8e3b970cb0adbc1d1da6ba399d7f3a6d5b6f8",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-23.el8.x86_64",
    sha256 = "32bcd075b3f1d5a4fe363097d33227885243c75348aec4171fd06552e245d4c8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.4-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/32bcd075b3f1d5a4fe363097d33227885243c75348aec4171fd06552e245d4c8",
    ],
)

rpm(
    name = "iputils-0__20180629-10.el8.aarch64",
    sha256 = "7a40254a162ab0117a106ed2a08b824a2f2186b14e56257a5e848ae070cee0f1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iputils-20180629-10.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7a40254a162ab0117a106ed2a08b824a2f2186b14e56257a5e848ae070cee0f1",
    ],
)

rpm(
    name = "iputils-0__20180629-10.el8.x86_64",
    sha256 = "66358ff76f9f26f6dbc403e479ab9389326d56233c5114daef316f589990c941",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iputils-20180629-10.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/66358ff76f9f26f6dbc403e479ab9389326d56233c5114daef316f589990c941",
    ],
)

rpm(
    name = "ipxe-roms-qemu-0__20181214-9.git133f4c47.el8.x86_64",
    sha256 = "73679ab2ab87aef03d9a0c0a071a4697cf3fef70e0fd3a05f1cb5b74319c70be",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/ipxe-roms-qemu-20181214-9.git133f4c47.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/73679ab2ab87aef03d9a0c0a071a4697cf3fef70e0fd3a05f1cb5b74319c70be",
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
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/jansson-2.14-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/69b4dd56ca16ed4ac5840e0d39a29d2e0b050905a349e1aceae4ec511a11b792",
    ],
)

rpm(
    name = "jansson-0__2.14-1.el8.x86_64",
    sha256 = "f825b85b4506a740fb2f85b9a577c51264f3cfe792dd8b2bf8963059cc77c3c4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/jansson-2.14-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f825b85b4506a740fb2f85b9a577c51264f3cfe792dd8b2bf8963059cc77c3c4",
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
    name = "kernel-headers-0__4.18.0-408.el8.aarch64",
    sha256 = "208e7b141b8ad93ee6bd748f5c4117ed5a947b4ff48071d4fcdb826670aad76a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kernel-headers-4.18.0-408.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/208e7b141b8ad93ee6bd748f5c4117ed5a947b4ff48071d4fcdb826670aad76a",
    ],
)

rpm(
    name = "kernel-headers-0__4.18.0-408.el8.x86_64",
    sha256 = "9f8784bf9b19f7e10f404bad73adc1ab520df781760ee7f9fbbf1192d8bff0c4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kernel-headers-4.18.0-408.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f8784bf9b19f7e10f404bad73adc1ab520df781760ee7f9fbbf1192d8bff0c4",
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
    name = "kmod-0__25-19.el8.aarch64",
    sha256 = "056e83e9da3c6a582e83634b66c3ead78f1729f4b9dbd9970dbf3bfdc45edb54",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-25-19.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/056e83e9da3c6a582e83634b66c3ead78f1729f4b9dbd9970dbf3bfdc45edb54",
    ],
)

rpm(
    name = "kmod-0__25-19.el8.x86_64",
    sha256 = "37c299fdaa42efb0d653ba5e22c83bd20833af1244b66ed6ea880e75c1672dd2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-25-19.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/37c299fdaa42efb0d653ba5e22c83bd20833af1244b66ed6ea880e75c1672dd2",
    ],
)

rpm(
    name = "kmod-libs-0__25-19.el8.aarch64",
    sha256 = "053b443be1bb0cbbc6da3314775391950350106462cc1dae01c7aed4358bf852",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-libs-25-19.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/053b443be1bb0cbbc6da3314775391950350106462cc1dae01c7aed4358bf852",
    ],
)

rpm(
    name = "kmod-libs-0__25-19.el8.x86_64",
    sha256 = "46a2ddc6067ed12089f04f2255c57117992807d707e280fc002f3ce786fc2abf",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-libs-25-19.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46a2ddc6067ed12089f04f2255c57117992807d707e280fc002f3ce786fc2abf",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-21.el8.aarch64",
    sha256 = "30f23e30b9e0de1c62a6b1d9f7031f7d5b263b458ad43c43915ea41a34711a92",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.18.2-21.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/30f23e30b9e0de1c62a6b1d9f7031f7d5b263b458ad43c43915ea41a34711a92",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-21.el8.x86_64",
    sha256 = "b02dcbdc99f85926d6595bc3f7e24ba535b0e22ae7932e61a4ea8ab8fb4b35d9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.18.2-21.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b02dcbdc99f85926d6595bc3f7e24ba535b0e22ae7932e61a4ea8ab8fb4b35d9",
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
    name = "libarchive-0__3.3.3-4.el8.aarch64",
    sha256 = "0dd36d8de0c8f40cbb01d9d1fc072eebf28967302b1eed287d7ad958aa383673",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libarchive-3.3.3-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0dd36d8de0c8f40cbb01d9d1fc072eebf28967302b1eed287d7ad958aa383673",
    ],
)

rpm(
    name = "libarchive-0__3.3.3-4.el8.x86_64",
    sha256 = "498b81c8c4f7fb75eccf6228776f0956c0f8c958cc3c6b45c61fdbf53ae6f039",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libarchive-3.3.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/498b81c8c4f7fb75eccf6228776f0956c0f8c958cc3c6b45c61fdbf53ae6f039",
    ],
)

rpm(
    name = "libasan-0__8.5.0-15.el8.aarch64",
    sha256 = "34e627e042580439b22395344a15dbfb7fe0ce7a93530217ce38134278084c60",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libasan-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/34e627e042580439b22395344a15dbfb7fe0ce7a93530217ce38134278084c60",
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
    name = "libatomic-0__8.5.0-15.el8.aarch64",
    sha256 = "58ea796ac4166da751068de1e250378e83b016586e08e2b2fb85d5903387f3b4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libatomic-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/58ea796ac4166da751068de1e250378e83b016586e08e2b2fb85d5903387f3b4",
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
    name = "libblkid-0__2.32.1-38.el8.aarch64",
    sha256 = "9337f86080be4696747646024137295f472e17f56bba764348c74201fcfa694a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libblkid-2.32.1-38.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9337f86080be4696747646024137295f472e17f56bba764348c74201fcfa694a",
    ],
)

rpm(
    name = "libblkid-0__2.32.1-38.el8.x86_64",
    sha256 = "74d1f0453b300e01c78a84314082d377a843bc19e8e6fe98ce6140b2028e64bb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libblkid-2.32.1-38.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/74d1f0453b300e01c78a84314082d377a843bc19e8e6fe98ce6140b2028e64bb",
    ],
)

rpm(
    name = "libbpf-0__0.5.0-1.el8.aarch64",
    sha256 = "1ecce335e1821b021b9fcfc8ffe1093a75f474249503510cf2bc499c61848cbb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libbpf-0.5.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1ecce335e1821b021b9fcfc8ffe1093a75f474249503510cf2bc499c61848cbb",
    ],
)

rpm(
    name = "libbpf-0__0.5.0-1.el8.x86_64",
    sha256 = "4d25308c27041d8a88a3340be12591e9bd46c9aebbe4195ee5d2f712d63ce033",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libbpf-0.5.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d25308c27041d8a88a3340be12591e9bd46c9aebbe4195ee5d2f712d63ce033",
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
    name = "libcap-0__2.48-4.el8.aarch64",
    sha256 = "f1fb5fe3b85ce5016a7882ccd9640b80f8fd6fbad1c44dc02076a8cdf33fc33d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcap-2.48-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f1fb5fe3b85ce5016a7882ccd9640b80f8fd6fbad1c44dc02076a8cdf33fc33d",
    ],
)

rpm(
    name = "libcap-0__2.48-4.el8.x86_64",
    sha256 = "34f69bed9ae0f5ba314a62172e8cfd9cf6795cb0c3bd29f15d174fc2a0acbb5b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcap-2.48-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/34f69bed9ae0f5ba314a62172e8cfd9cf6795cb0c3bd29f15d174fc2a0acbb5b",
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
    name = "libcom_err-0__1.45.6-5.el8.aarch64",
    sha256 = "bdd5ab69772a43725e1f8397e8142094bdd28b21b65ff02da74a8fc986424f3c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcom_err-1.45.6-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bdd5ab69772a43725e1f8397e8142094bdd28b21b65ff02da74a8fc986424f3c",
    ],
)

rpm(
    name = "libcom_err-0__1.45.6-5.el8.x86_64",
    sha256 = "4e4f13acac0477f0a121812107a9939ea2164eebab052813f1618d5b7df5d87a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcom_err-1.45.6-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4e4f13acac0477f0a121812107a9939ea2164eebab052813f1618d5b7df5d87a",
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
    name = "libcurl-minimal-0__7.61.1-25.el8.aarch64",
    sha256 = "2852cffc539a2178e52304b24c83ded856a7da3dbc76c0f21c7db522c72b03b1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.61.1-25.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2852cffc539a2178e52304b24c83ded856a7da3dbc76c0f21c7db522c72b03b1",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.61.1-25.el8.x86_64",
    sha256 = "06783b8a7201001f657e6800e4b0c646025e1963e0f806fed6f2d2e6234824b1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.61.1-25.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/06783b8a7201001f657e6800e4b0c646025e1963e0f806fed6f2d2e6234824b1",
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
    name = "libfdisk-0__2.32.1-38.el8.aarch64",
    sha256 = "6b34849e8d42cfa88a1a7d4862fcbb56dfa4477d8bc8c8415a801aa41261b2d6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libfdisk-2.32.1-38.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6b34849e8d42cfa88a1a7d4862fcbb56dfa4477d8bc8c8415a801aa41261b2d6",
    ],
)

rpm(
    name = "libfdisk-0__2.32.1-38.el8.x86_64",
    sha256 = "d5a9931230789cb11ed52ff2927f85c045e922ce31b22194545f7d70962d0fbd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libfdisk-2.32.1-38.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5a9931230789cb11ed52ff2927f85c045e922ce31b22194545f7d70962d0fbd",
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
    name = "libfdt-0__1.6.0-1.el8.x86_64",
    sha256 = "1788b4786715c45a1ac90ca9f413ef51f2cdd03170a981e0ef13eab204f44429",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libfdt-1.6.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1788b4786715c45a1ac90ca9f413ef51f2cdd03170a981e0ef13eab204f44429",
    ],
)

rpm(
    name = "libffi-0__3.1-23.el8.aarch64",
    sha256 = "ba34d0bb067722c37dd4367534d82aa18c659facbfd17952f8d826e8662cb7c1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libffi-3.1-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ba34d0bb067722c37dd4367534d82aa18c659facbfd17952f8d826e8662cb7c1",
    ],
)

rpm(
    name = "libffi-0__3.1-23.el8.x86_64",
    sha256 = "643d1b969c7fbcd55c523f779089f3f2fe8b105c719fd49c7edd1f142dfc2143",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libffi-3.1-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/643d1b969c7fbcd55c523f779089f3f2fe8b105c719fd49c7edd1f142dfc2143",
    ],
)

rpm(
    name = "libgcc-0__8.5.0-15.el8.aarch64",
    sha256 = "f62a7bd6b2ce584a9ee3561513053372db492efd867333b27f7ba9a3844ff553",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcc-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f62a7bd6b2ce584a9ee3561513053372db492efd867333b27f7ba9a3844ff553",
    ],
)

rpm(
    name = "libgcc-0__8.5.0-15.el8.x86_64",
    sha256 = "e020248e0906263fc12ca404974d1ae7e23357ef2f73881e7f874f57290ac4d4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcc-8.5.0-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e020248e0906263fc12ca404974d1ae7e23357ef2f73881e7f874f57290ac4d4",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-7.el8.aarch64",
    sha256 = "88a32029615cc5986884cbab1b5c137e455b9ef08b23c6219b9ec9b42079be88",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.8.5-7.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/88a32029615cc5986884cbab1b5c137e455b9ef08b23c6219b9ec9b42079be88",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-7.el8.x86_64",
    sha256 = "01541f1263532f80114111a44f797d6a8eed75744db997e85fddd021e636c5bb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.8.5-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/01541f1263532f80114111a44f797d6a8eed75744db997e85fddd021e636c5bb",
    ],
)

rpm(
    name = "libgomp-0__8.5.0-15.el8.aarch64",
    sha256 = "edb71029b4d451240f53399652c872035ebab3237bfa4d416e010be58bc8a056",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgomp-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/edb71029b4d451240f53399652c872035ebab3237bfa4d416e010be58bc8a056",
    ],
)

rpm(
    name = "libgomp-0__8.5.0-15.el8.x86_64",
    sha256 = "9d17f906c5d6412344615999f23fec33e4b2232bf7c1b0871f3bec12f96ce897",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgomp-8.5.0-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d17f906c5d6412344615999f23fec33e4b2232bf7c1b0871f3bec12f96ce897",
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
    name = "libguestfs-1__1.44.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "a0cbdc5c27f1d45480b2c4b28caac267a9a879de19091efa057119705611cbef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libguestfs-1.44.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0cbdc5c27f1d45480b2c4b28caac267a9a879de19091efa057119705611cbef",
    ],
)

rpm(
    name = "libguestfs-tools-1__1.44.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "fb8f81a46a30e7254f614f5b0376af1fef45c9082b2e6f88061e61cc046de99f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libguestfs-tools-1.44.0-5.module_el8.6.0+1087+b42c8331.noarch.rpm",
        "https://storage.googleapis.com/builddeps/fb8f81a46a30e7254f614f5b0376af1fef45c9082b2e6f88061e61cc046de99f",
    ],
)

rpm(
    name = "libguestfs-tools-c-1__1.44.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "61bb7c563c80a44fcce4bf9c1004539cf33165700f94a3ee384483345f60edc2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libguestfs-tools-c-1.44.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61bb7c563c80a44fcce4bf9c1004539cf33165700f94a3ee384483345f60edc2",
    ],
)

rpm(
    name = "libibverbs-0__41.0-1.el8.aarch64",
    sha256 = "64304bd0d2e426b705f798fda9441fd20efcd71e7b99e536ba27636c73d1dcba",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libibverbs-41.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/64304bd0d2e426b705f798fda9441fd20efcd71e7b99e536ba27636c73d1dcba",
    ],
)

rpm(
    name = "libibverbs-0__41.0-1.el8.x86_64",
    sha256 = "888b1ce059dfaf1b8277cac3529970114ba1cadc75fbcf9410f3031451ab7e30",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libibverbs-41.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/888b1ce059dfaf1b8277cac3529970114ba1cadc75fbcf9410f3031451ab7e30",
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
    name = "libmount-0__2.32.1-38.el8.aarch64",
    sha256 = "0fc8a00a2fb09a3d9d47e01bdf2ee5392fc7d2702ec27882dad466ae9a43b4af",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmount-2.32.1-38.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0fc8a00a2fb09a3d9d47e01bdf2ee5392fc7d2702ec27882dad466ae9a43b4af",
    ],
)

rpm(
    name = "libmount-0__2.32.1-38.el8.x86_64",
    sha256 = "ad9351168c138eca10d2173dc74832be90786a4d181951bf86fceb0c8693ab95",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmount-2.32.1-38.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ad9351168c138eca10d2173dc74832be90786a4d181951bf86fceb0c8693ab95",
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
    name = "libnftnl-0__1.1.5-5.el8.aarch64",
    sha256 = "00522e43ce63cf63468052e627a429ededac0815212c644f4eadda88b990c3ee",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnftnl-1.1.5-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/00522e43ce63cf63468052e627a429ededac0815212c644f4eadda88b990c3ee",
    ],
)

rpm(
    name = "libnftnl-0__1.1.5-5.el8.x86_64",
    sha256 = "293e1f0f44a9c1d5dedbe831dff3049fad9e88c5f0e281d889f427603ac51fa6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnftnl-1.1.5-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/293e1f0f44a9c1d5dedbe831dff3049fad9e88c5f0e281d889f427603ac51fa6",
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
    name = "libnl3-0__3.7.0-1.el8.aarch64",
    sha256 = "8c8dd63daf7ad4c6322a4316fceb256f1cfd2d8244bce515bbae539b4314a643",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnl3-3.7.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8c8dd63daf7ad4c6322a4316fceb256f1cfd2d8244bce515bbae539b4314a643",
    ],
)

rpm(
    name = "libnl3-0__3.7.0-1.el8.x86_64",
    sha256 = "9ce7aa4d7bd810448d9fb3aa85a66cca00950f7c2c59bc9721ced3e4f3ad2885",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnl3-3.7.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ce7aa4d7bd810448d9fb3aa85a66cca00950f7c2c59bc9721ced3e4f3ad2885",
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
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libpmem-1.11.1-1.module_el8.6.0+1088+6891f51c.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/924d405a5a7b2de6405cd277f72f2c4af20f9162e8484d9142bd7a56f546a894",
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
    name = "libpwquality-0__1.4.4-5.el8.aarch64",
    sha256 = "01d7a24f607279d3ceddbee4bc1de275cbe5e496c3ebc8765d8c81acae45904c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpwquality-1.4.4-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01d7a24f607279d3ceddbee4bc1de275cbe5e496c3ebc8765d8c81acae45904c",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-5.el8.x86_64",
    sha256 = "4a7159ebfb7914f23f009981a38fcbec8368b243b20dfed6326a6dade95cf3a2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.4-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a7159ebfb7914f23f009981a38fcbec8368b243b20dfed6326a6dade95cf3a2",
    ],
)

rpm(
    name = "librdmacm-0__41.0-1.el8.aarch64",
    sha256 = "1e7580eca85aa66b7989d632fafb4a9d3f7aeb9c2294b699d37249bf8f5f5cad",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/librdmacm-41.0-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1e7580eca85aa66b7989d632fafb4a9d3f7aeb9c2294b699d37249bf8f5f5cad",
    ],
)

rpm(
    name = "librdmacm-0__41.0-1.el8.x86_64",
    sha256 = "caf52cd9c97677b5684730ad61f8abe464cfc41d332b3f4d4887fb2e8ea87916",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/librdmacm-41.0-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/caf52cd9c97677b5684730ad61f8abe464cfc41d332b3f4d4887fb2e8ea87916",
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
    name = "libselinux-0__2.9-6.el8.aarch64",
    sha256 = "f08e19d08afef99a50b1945a8562e65c84ebdbd9327f1cabdf5fe324dcb5550e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libselinux-2.9-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f08e19d08afef99a50b1945a8562e65c84ebdbd9327f1cabdf5fe324dcb5550e",
    ],
)

rpm(
    name = "libselinux-0__2.9-6.el8.x86_64",
    sha256 = "5c5af2edd462b42dcf37f9188ab1b7810e21d814b9f81419d82504a49d2a4cd3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libselinux-2.9-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5c5af2edd462b42dcf37f9188ab1b7810e21d814b9f81419d82504a49d2a4cd3",
    ],
)

rpm(
    name = "libselinux-utils-0__2.9-6.el8.aarch64",
    sha256 = "984094cc5b9d5854d4f96691ce81518fba3a28df1e82cfcab4df79dffb78cccd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libselinux-utils-2.9-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/984094cc5b9d5854d4f96691ce81518fba3a28df1e82cfcab4df79dffb78cccd",
    ],
)

rpm(
    name = "libselinux-utils-0__2.9-6.el8.x86_64",
    sha256 = "ce2a912a22aa86ea4847621043f48e4231fd2cb7d3a718b550aba04b88310ad0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libselinux-utils-2.9-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce2a912a22aa86ea4847621043f48e4231fd2cb7d3a718b550aba04b88310ad0",
    ],
)

rpm(
    name = "libsemanage-0__2.9-9.el8.aarch64",
    sha256 = "95da090dc1010ed9dec6ee352ddb5293825d47844441ad908fca1c4852bb51e7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsemanage-2.9-9.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/95da090dc1010ed9dec6ee352ddb5293825d47844441ad908fca1c4852bb51e7",
    ],
)

rpm(
    name = "libsemanage-0__2.9-9.el8.x86_64",
    sha256 = "7b8293193b1dda6c408c04074c4b501faf37ff9e4a4b6cd1ca2cce81d5bb67bf",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsemanage-2.9-9.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7b8293193b1dda6c408c04074c4b501faf37ff9e4a4b6cd1ca2cce81d5bb67bf",
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
    name = "libsmartcols-0__2.32.1-38.el8.aarch64",
    sha256 = "9ac2b7da9ef39ad0ea119ff0f68f44bf1b6025aca227cc10d6df29e59b6fbe24",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.32.1-38.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9ac2b7da9ef39ad0ea119ff0f68f44bf1b6025aca227cc10d6df29e59b6fbe24",
    ],
)

rpm(
    name = "libsmartcols-0__2.32.1-38.el8.x86_64",
    sha256 = "3eb6ea68ac85b7a13d9b55a7d1f38107979d023ffcd5901b9265b038d0833973",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.32.1-38.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3eb6ea68ac85b7a13d9b55a7d1f38107979d023ffcd5901b9265b038d0833973",
    ],
)

rpm(
    name = "libss-0__1.45.6-5.el8.aarch64",
    sha256 = "68b0f490ced8811f8b25423c7bd2d81b26301317e4445705c4b280283a50b8e9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libss-1.45.6-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/68b0f490ced8811f8b25423c7bd2d81b26301317e4445705c4b280283a50b8e9",
    ],
)

rpm(
    name = "libss-0__1.45.6-5.el8.x86_64",
    sha256 = "f489f5eaaddbdedae046e4ddfe93947cdd636533ca8d35820bf5c92ae5dd3037",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libss-1.45.6-5.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f489f5eaaddbdedae046e4ddfe93947cdd636533ca8d35820bf5c92ae5dd3037",
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
    name = "libsss_idmap-0__2.7.3-4.el8.aarch64",
    sha256 = "1b349a7f62cca5b60f634920b57c7770c0ae86e137f36ccf4ae1f9f95cd533b9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.7.3-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1b349a7f62cca5b60f634920b57c7770c0ae86e137f36ccf4ae1f9f95cd533b9",
    ],
)

rpm(
    name = "libsss_idmap-0__2.7.3-4.el8.x86_64",
    sha256 = "ef32621f014358ec62bdcf4ee1e61ce1e5ec77237cde9bb26c37a38a453e1044",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.7.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ef32621f014358ec62bdcf4ee1e61ce1e5ec77237cde9bb26c37a38a453e1044",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.7.3-4.el8.aarch64",
    sha256 = "a5f7a789034c78edee700bc61c96c495bd67f8d403464dfeed681d29a28d1443",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.7.3-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a5f7a789034c78edee700bc61c96c495bd67f8d403464dfeed681d29a28d1443",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.7.3-4.el8.x86_64",
    sha256 = "0b512d29c433eb87392d5172fa3ff63c8a6408a27273dc4b2f7b54231b367dcb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.7.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0b512d29c433eb87392d5172fa3ff63c8a6408a27273dc4b2f7b54231b367dcb",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-15.el8.aarch64",
    sha256 = "91d6f78ddeab3c6df90479eeca76e77450605983619a54c01faaa8ede3767214",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libstdc++-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/91d6f78ddeab3c6df90479eeca76e77450605983619a54c01faaa8ede3767214",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-15.el8.x86_64",
    sha256 = "298bab1223dfa678e3fc567792e14dc8329b50bbf1d93a66bd287e7005da9fb0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libstdc++-8.5.0-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/298bab1223dfa678e3fc567792e14dc8329b50bbf1d93a66bd287e7005da9fb0",
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
    name = "libtirpc-0__1.1.4-8.el8.aarch64",
    sha256 = "95a8f001c48779fcd1ef52d7d633bb3f6abb27684c71dfeaa421e58ebb38ad33",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libtirpc-1.1.4-8.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/95a8f001c48779fcd1ef52d7d633bb3f6abb27684c71dfeaa421e58ebb38ad33",
    ],
)

rpm(
    name = "libtirpc-0__1.1.4-8.el8.x86_64",
    sha256 = "bcade31f01063824b3a3e77218caaedd16532413282978c437c82b81c2991e4e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libtirpc-1.1.4-8.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bcade31f01063824b3a3e77218caaedd16532413282978c437c82b81c2991e4e",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-1.20211126git1ff6fe1f43.module_el8.7.0__plus__1218__plus__f626c2ff.aarch64",
    sha256 = "3acd4597c1f45e6c9968da8b3b47f18dae4829b94814f61c64d1764696762fbd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libtpms-0.9.1-1.20211126git1ff6fe1f43.module_el8.7.0+1218+f626c2ff.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3acd4597c1f45e6c9968da8b3b47f18dae4829b94814f61c64d1764696762fbd",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-1.20211126git1ff6fe1f43.module_el8.7.0__plus__1218__plus__f626c2ff.x86_64",
    sha256 = "22948530ccb9782fb07a6fadbe1904e7c8d9863d6f097d3fb210a7b63d4843fd",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libtpms-0.9.1-1.20211126git1ff6fe1f43.module_el8.7.0+1218+f626c2ff.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/22948530ccb9782fb07a6fadbe1904e7c8d9863d6f097d3fb210a7b63d4843fd",
    ],
)

rpm(
    name = "libubsan-0__8.5.0-15.el8.aarch64",
    sha256 = "f17b6540d94e217baf503abe38e9ff08132872c7d35c15048e8891fe0cefedb1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libubsan-8.5.0-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f17b6540d94e217baf503abe38e9ff08132872c7d35c15048e8891fe0cefedb1",
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
    name = "libuuid-0__2.32.1-38.el8.aarch64",
    sha256 = "acc2cea2e85fabdee5ce88ec9ce46aa03b1e7651940705dae89fe076428e7193",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libuuid-2.32.1-38.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/acc2cea2e85fabdee5ce88ec9ce46aa03b1e7651940705dae89fe076428e7193",
    ],
)

rpm(
    name = "libuuid-0__2.32.1-38.el8.x86_64",
    sha256 = "a91c928ef63cee299d3ffeb3880d8747d4915068f15e460f30aa2e90dae50602",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libuuid-2.32.1-38.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a91c928ef63cee299d3ffeb3880d8747d4915068f15e460f30aa2e90dae50602",
    ],
)

rpm(
    name = "libverto-0__0.3.2-2.el8.aarch64",
    sha256 = "1a8478fe342782d95f29253a2845bdb3e88ced25b5e6b029cecc52a43df1932b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libverto-0.3.2-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1a8478fe342782d95f29253a2845bdb3e88ced25b5e6b029cecc52a43df1932b",
    ],
)

rpm(
    name = "libverto-0__0.3.2-2.el8.x86_64",
    sha256 = "96b8ea32c5e9b3275788525ecbf35fd6ac1ae137754a2857503776512d4db58a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libverto-0.3.2-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/96b8ea32c5e9b3275788525ecbf35fd6ac1ae137754a2857503776512d4db58a",
    ],
)

rpm(
    name = "libvirt-client-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "fd736b99c4910c52e7bffd34532ece859819ea1e4ad2dc616a554fe630eb8d3a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libvirt-client-8.0.0-2.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fd736b99c4910c52e7bffd34532ece859819ea1e4ad2dc616a554fe630eb8d3a",
    ],
)

rpm(
    name = "libvirt-client-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "722f30f8e4a8240662ec03c4bfc1320de88908738ca77fa4fa05e87627821bb1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libvirt-client-8.0.0-2.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/722f30f8e4a8240662ec03c4bfc1320de88908738ca77fa4fa05e87627821bb1",
    ],
)

rpm(
    name = "libvirt-daemon-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "734437ae41c5c705ab1da476ee9521a57124261727c16f398c8a1bdd8be44922",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libvirt-daemon-8.0.0-2.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/734437ae41c5c705ab1da476ee9521a57124261727c16f398c8a1bdd8be44922",
    ],
)

rpm(
    name = "libvirt-daemon-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "0429a9e9d8eb98c5ebd689993a3ca8f14949ae45be5a290fce8bbe9c4ad68850",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libvirt-daemon-8.0.0-2.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0429a9e9d8eb98c5ebd689993a3ca8f14949ae45be5a290fce8bbe9c4ad68850",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "77a8a98da56eeaf7cdfe11bdc6b01e42f9eea16b0c04f1abfe7fbafe216a4a66",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-8.0.0-2.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/77a8a98da56eeaf7cdfe11bdc6b01e42f9eea16b0c04f1abfe7fbafe216a4a66",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "d34af964ae21ad21c7e8f97f7f05e2b362744e77270930d8e41a98bced9d91e7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-8.0.0-2.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d34af964ae21ad21c7e8f97f7f05e2b362744e77270930d8e41a98bced9d91e7",
    ],
)

rpm(
    name = "libvirt-devel-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "aa47408e4c1499bc03442a6873444ea7d4cd3b62bf59118ff30da2e9db29369f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libvirt-devel-8.0.0-2.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aa47408e4c1499bc03442a6873444ea7d4cd3b62bf59118ff30da2e9db29369f",
    ],
)

rpm(
    name = "libvirt-devel-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "b14d075708f66875be58adb67be1f2ba3b7c1c1c89c87b3656c07b3b6ee03ded",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libvirt-devel-8.0.0-2.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b14d075708f66875be58adb67be1f2ba3b7c1c1c89c87b3656c07b3b6ee03ded",
    ],
)

rpm(
    name = "libvirt-libs-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "7feb59b591f71783999b5ec9256ef61da19e5e3cdaae46bec162781cdab4b074",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libvirt-libs-8.0.0-2.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7feb59b591f71783999b5ec9256ef61da19e5e3cdaae46bec162781cdab4b074",
    ],
)

rpm(
    name = "libvirt-libs-0__8.0.0-2.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "ba3daa6361d8b7a0f673840088f81f8aa994f811de1cc95c8c6e1c4baf31ebed",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libvirt-libs-8.0.0-2.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ba3daa6361d8b7a0f673840088f81f8aa994f811de1cc95c8c6e1c4baf31ebed",
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
    name = "libxml2-0__2.9.7-15.el8.aarch64",
    sha256 = "8e1f021974ac791a367b10b8bf196d43eec3978ed3cc24f75b6f7abfc7089054",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.7-15.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8e1f021974ac791a367b10b8bf196d43eec3978ed3cc24f75b6f7abfc7089054",
    ],
)

rpm(
    name = "libxml2-0__2.9.7-15.el8.x86_64",
    sha256 = "fd99e5a3ef51c11b1380bb3ea1d906a9677032dd80fe3a5fc274e1e9407a8efb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.7-15.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd99e5a3ef51c11b1380bb3ea1d906a9677032dd80fe3a5fc274e1e9407a8efb",
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
    name = "lvm2-8__2.03.14-6.el8.x86_64",
    sha256 = "d66449a34c08cf0d22fae47507c032fa4f51401d4ea6aafc70fa606f3a548019",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-2.03.14-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d66449a34c08cf0d22fae47507c032fa4f51401d4ea6aafc70fa606f3a548019",
    ],
)

rpm(
    name = "lvm2-libs-8__2.03.14-6.el8.x86_64",
    sha256 = "dce1d014dd3107351c1c6918ffd4de4a88fbaebed210c00a4a3f0c1966c3aabf",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-libs-2.03.14-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dce1d014dd3107351c1c6918ffd4de4a88fbaebed210c00a4a3f0c1966c3aabf",
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
    name = "ncurses-0__6.1-9.20180224.el8.x86_64",
    sha256 = "fc22ce73243e2f926e72967c28de57beabfa3720e51248b9a39e40207fbc6c8a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ncurses-6.1-9.20180224.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fc22ce73243e2f926e72967c28de57beabfa3720e51248b9a39e40207fbc6c8a",
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
    name = "ndctl-libs-0__71.1-4.el8.x86_64",
    sha256 = "d1518d8f29a72c8c9501f67929258405cf25fd4be365fd905acc57b846d49c8a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ndctl-libs-71.1-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d1518d8f29a72c8c9501f67929258405cf25fd4be365fd905acc57b846d49c8a",
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
    name = "nftables-1__0.9.3-26.el8.aarch64",
    sha256 = "22cacdb52fb6a31659789b5190f8e6db27ca1dddd9b67f3c6b2c1db917ef882f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/nftables-0.9.3-26.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/22cacdb52fb6a31659789b5190f8e6db27ca1dddd9b67f3c6b2c1db917ef882f",
    ],
)

rpm(
    name = "nftables-1__0.9.3-26.el8.x86_64",
    sha256 = "813d7c361e77b394f6f05fb29983c3ee6c2dd2e8fe8b857e2bdb6b9914e0c129",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/nftables-0.9.3-26.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/813d7c361e77b394f6f05fb29983c3ee6c2dd2e8fe8b857e2bdb6b9914e0c129",
    ],
)

rpm(
    name = "nmap-ncat-2__7.70-8.el8.aarch64",
    sha256 = "dc83ec9685aa03079f7348f56b616f112a6c8829e7fdcf88f8355065e72c187d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.70-8.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dc83ec9685aa03079f7348f56b616f112a6c8829e7fdcf88f8355065e72c187d",
    ],
)

rpm(
    name = "nmap-ncat-2__7.70-8.el8.x86_64",
    sha256 = "01f8398a2bcb3b258bc51f219ec7d3fb9c408c91170659919f136edea2b1cc32",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.70-8.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/01f8398a2bcb3b258bc51f219ec7d3fb9c408c91170659919f136edea2b1cc32",
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
    name = "openssl-libs-1__1.1.1k-7.el8.aarch64",
    sha256 = "7fe60edf1f59b0eb61c5bf3f298cb247be14ddb713291fec770914f7df6ec17d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/openssl-libs-1.1.1k-7.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7fe60edf1f59b0eb61c5bf3f298cb247be14ddb713291fec770914f7df6ec17d",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-7.el8.x86_64",
    sha256 = "7b42ba3855f29955fe204ad7c189a832a5b1423a32abcda079d8ef2f787c8e73",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/openssl-libs-1.1.1k-7.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7b42ba3855f29955fe204ad7c189a832a5b1423a32abcda079d8ef2f787c8e73",
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
    name = "pam-0__1.3.1-22.el8.aarch64",
    sha256 = "b900edf1f702460be4a6b2e402e02887068fe9172b88256660b8c20b89a772d5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pam-1.3.1-22.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b900edf1f702460be4a6b2e402e02887068fe9172b88256660b8c20b89a772d5",
    ],
)

rpm(
    name = "pam-0__1.3.1-22.el8.x86_64",
    sha256 = "435bf0de1d95994530d596a93905394d066b8f0df0da360edce7dbe466ab3101",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pam-1.3.1-22.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/435bf0de1d95994530d596a93905394d066b8f0df0da360edce7dbe466ab3101",
    ],
)

rpm(
    name = "passt-0__0.git.2022_08_29.60ffc5b-1.el8.aarch64",
    sha256 = "909bb0b287fb9e29cee43e8703302bf118763b5cee85a4c21085a34efbd48e37",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/sbrivio/passt/centos-stream-8-aarch64/04776284-passt/passt-0.git.2022_08_29.60ffc5b-1.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/909bb0b287fb9e29cee43e8703302bf118763b5cee85a4c21085a34efbd48e37",
    ],
)

rpm(
    name = "passt-0__0.git.2022_08_29.60ffc5b-1.el8.x86_64",
    sha256 = "e87c53d771dfa8c46a034c895706d4db09c40e594e0fc363005e652be2201bbd",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/sbrivio/passt/centos-stream-8-x86_64/04776284-passt/passt-0.git.2022_08_29.60ffc5b-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e87c53d771dfa8c46a034c895706d4db09c40e594e0fc363005e652be2201bbd",
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
    name = "pcre2-0__10.32-3.el8.aarch64",
    sha256 = "b8e4367f28a53ec70a6b8a329a5bda886374eddde5f55c9467e1783d4158b5d1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pcre2-10.32-3.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b8e4367f28a53ec70a6b8a329a5bda886374eddde5f55c9467e1783d4158b5d1",
    ],
)

rpm(
    name = "pcre2-0__10.32-3.el8.x86_64",
    sha256 = "2f865747024d26b91d5a9f2f35dd1b04e1039d64e772d0371b437145cd7beceb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pcre2-10.32-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2f865747024d26b91d5a9f2f35dd1b04e1039d64e772d0371b437145cd7beceb",
    ],
)

rpm(
    name = "perl-Carp-0__1.42-396.el8.x86_64",
    sha256 = "d03b9f4b9848e3a88d62bcf6e536d659c325b2dc03b2136be7342b5fe5e2b6a9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Carp-1.42-396.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d03b9f4b9848e3a88d62bcf6e536d659c325b2dc03b2136be7342b5fe5e2b6a9",
    ],
)

rpm(
    name = "perl-Encode-4__2.97-3.el8.x86_64",
    sha256 = "d2b0e4b28a5aac754f6caa119d5479a64816f93c059e0ac564e46391264e2234",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Encode-2.97-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d2b0e4b28a5aac754f6caa119d5479a64816f93c059e0ac564e46391264e2234",
    ],
)

rpm(
    name = "perl-Errno-0__1.28-421.el8.x86_64",
    sha256 = "8d9b26f17e427dc497032b1897b9296c4ca37fa1b96d9c459b42516d72ef06a1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Errno-1.28-421.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8d9b26f17e427dc497032b1897b9296c4ca37fa1b96d9c459b42516d72ef06a1",
    ],
)

rpm(
    name = "perl-Exporter-0__5.72-396.el8.x86_64",
    sha256 = "7edc503f5a919c489b651757095d8031982d530cc88088fdaeb743188364e9b0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Exporter-5.72-396.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7edc503f5a919c489b651757095d8031982d530cc88088fdaeb743188364e9b0",
    ],
)

rpm(
    name = "perl-File-Path-0__2.15-2.el8.x86_64",
    sha256 = "e83928bd4552ecdf8e71d283e2358c7eccd006d284ba31fbc9c89e407989fd60",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-File-Path-2.15-2.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e83928bd4552ecdf8e71d283e2358c7eccd006d284ba31fbc9c89e407989fd60",
    ],
)

rpm(
    name = "perl-File-Temp-0__0.230.600-1.el8.x86_64",
    sha256 = "e269f7d33abbb790311ffa95fa7df9766cac8bf31ace24fce6ed732ba0db19ae",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-File-Temp-0.230.600-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e269f7d33abbb790311ffa95fa7df9766cac8bf31ace24fce6ed732ba0db19ae",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.50-4.el8.x86_64",
    sha256 = "da4c6daa0d5406bc967cc89b02a69689491f42c543aceea1a31136f0f1a8d991",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Getopt-Long-2.50-4.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/da4c6daa0d5406bc967cc89b02a69689491f42c543aceea1a31136f0f1a8d991",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.074-1.el8.x86_64",
    sha256 = "a1af93a1b62e8ca05b7597d5749a2b3d28735a86928f0432064fec61db1ff844",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-HTTP-Tiny-0.074-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a1af93a1b62e8ca05b7597d5749a2b3d28735a86928f0432064fec61db1ff844",
    ],
)

rpm(
    name = "perl-IO-0__1.38-421.el8.x86_64",
    sha256 = "7ff911df218c38953660d4a09f9864364e2433b9aaf8283db8b7d5214411e28a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-IO-1.38-421.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7ff911df218c38953660d4a09f9864364e2433b9aaf8283db8b7d5214411e28a",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.15-396.el8.x86_64",
    sha256 = "5642297bf32bb174173917dd10fd2a3a2ef7277c599f76c0669c5c448f10bdaf",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-MIME-Base64-3.15-396.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5642297bf32bb174173917dd10fd2a3a2ef7277c599f76c0669c5c448f10bdaf",
    ],
)

rpm(
    name = "perl-PathTools-0__3.74-1.el8.x86_64",
    sha256 = "512245f7741790b36b03562469b9262f4dedfb8862dfa2d42e64598bb205d4c9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-PathTools-3.74-1.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/512245f7741790b36b03562469b9262f4dedfb8862dfa2d42e64598bb205d4c9",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-395.el8.x86_64",
    sha256 = "545cd23ad8e4f71a5109551093668fd4b5e1a50d6a60364ce0f04f64eecd99d1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Escapes-1.07-395.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/545cd23ad8e4f71a5109551093668fd4b5e1a50d6a60364ce0f04f64eecd99d1",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28-396.el8.x86_64",
    sha256 = "0225dc3999e3d7b1bb57186a2fc93c98bd1e4e08e062fb51c966e1f2a2c91bb4",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Perldoc-3.28-396.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0225dc3999e3d7b1bb57186a2fc93c98bd1e4e08e062fb51c966e1f2a2c91bb4",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.35-395.el8.x86_64",
    sha256 = "51c3ee5d824bdde0a8faa10c99841c2590c0c26edfb17125aa97945a688c83ed",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Simple-3.35-395.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/51c3ee5d824bdde0a8faa10c99841c2590c0c26edfb17125aa97945a688c83ed",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__1.69-395.el8.x86_64",
    sha256 = "794f970f498af07b37f914c19ad5dedc6b6c2f89d343af9dd1768d17232555de",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Pod-Usage-1.69-395.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/794f970f498af07b37f914c19ad5dedc6b6c2f89d343af9dd1768d17232555de",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-3__1.49-2.el8.x86_64",
    sha256 = "3db0d05ca5ba00981312f3a3ddcbabf466c2f1fc639cbf29482bb2cd952df456",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Scalar-List-Utils-1.49-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3db0d05ca5ba00981312f3a3ddcbabf466c2f1fc639cbf29482bb2cd952df456",
    ],
)

rpm(
    name = "perl-Socket-4__2.027-3.el8.x86_64",
    sha256 = "de138a9614191af63b9603cf0912d4ffd9bd9e5b122c2d0a78ae0eac009a602f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Socket-2.027-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de138a9614191af63b9603cf0912d4ffd9bd9e5b122c2d0a78ae0eac009a602f",
    ],
)

rpm(
    name = "perl-Storable-1__3.11-3.el8.x86_64",
    sha256 = "0c3007b68a37325866aaade4ae076232bca15e268f66c3d3b3a6d236bb85e1e9",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Storable-3.11-3.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c3007b68a37325866aaade4ae076232bca15e268f66c3d3b3a6d236bb85e1e9",
    ],
)

rpm(
    name = "perl-Sys-Guestfs-1__1.44.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "8e01d8cca7a1297980a36db1b56835cce506c08450d12b7b21e11bfa58ad22bb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/perl-Sys-Guestfs-1.44.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e01d8cca7a1297980a36db1b56835cce506c08450d12b7b21e11bfa58ad22bb",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__4.06-396.el8.x86_64",
    sha256 = "f4e3607f242bbca7ec2379822ca961860e6d9c276da51c6e2dfd17a29469ec78",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Term-ANSIColor-4.06-396.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f4e3607f242bbca7ec2379822ca961860e6d9c276da51c6e2dfd17a29469ec78",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-395.el8.x86_64",
    sha256 = "6bbb721dd2c411c85c75f7477b14c54c776d78ee9b93557615e919ef47577440",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Term-Cap-1.17-395.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6bbb721dd2c411c85c75f7477b14c54c776d78ee9b93557615e919ef47577440",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-395.el8.x86_64",
    sha256 = "2975de6545b4ca7907ae368a1716c531764e4afccbf27fb0a694d90e983c38e2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Text-ParseWords-3.30-395.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2975de6545b4ca7907ae368a1716c531764e4afccbf27fb0a694d90e983c38e2",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-395.el8.x86_64",
    sha256 = "7e50a5d0f2fbd8c95375f72f5772c7731186e999a447121b8247f448b065a4ef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Text-Tabs+Wrap-2013.0523-395.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7e50a5d0f2fbd8c95375f72f5772c7731186e999a447121b8247f448b065a4ef",
    ],
)

rpm(
    name = "perl-Time-Local-1__1.280-1.el8.x86_64",
    sha256 = "1edcf2b441ddf21417ef2b33e1ab2a30900758819335d7fabafe3b16bb3eab62",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Time-Local-1.280-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1edcf2b441ddf21417ef2b33e1ab2a30900758819335d7fabafe3b16bb3eab62",
    ],
)

rpm(
    name = "perl-Unicode-Normalize-0__1.25-396.el8.x86_64",
    sha256 = "99678a57c35343d8b2e2a502efcccc17bde3e40d97d7d2c5f988af8d3aa166d0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-Unicode-Normalize-1.25-396.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/99678a57c35343d8b2e2a502efcccc17bde3e40d97d7d2c5f988af8d3aa166d0",
    ],
)

rpm(
    name = "perl-constant-0__1.33-396.el8.x86_64",
    sha256 = "7559c097998db5e5d14dab1a7a1637a5749e9dab234ca68d17c9c21f8cfbf8d6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-constant-1.33-396.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7559c097998db5e5d14dab1a7a1637a5749e9dab234ca68d17c9c21f8cfbf8d6",
    ],
)

rpm(
    name = "perl-hivex-0__1.3.18-23.module_el8.6.0__plus__983__plus__a7505f3f.x86_64",
    sha256 = "42db01e9df5ba75147ad2a0cfb37f5f6c37ae980260d218dc93a0ead8cab7983",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/perl-hivex-1.3.18-23.module_el8.6.0+983+a7505f3f.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/42db01e9df5ba75147ad2a0cfb37f5f6c37ae980260d218dc93a0ead8cab7983",
    ],
)

rpm(
    name = "perl-interpreter-4__5.26.3-421.el8.x86_64",
    sha256 = "4618427acf4bcfa66ec91cccf995d938e1ed0f87b1088d7d948a9993a6d15b29",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-interpreter-5.26.3-421.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4618427acf4bcfa66ec91cccf995d938e1ed0f87b1088d7d948a9993a6d15b29",
    ],
)

rpm(
    name = "perl-libintl-perl-0__1.29-2.el8.x86_64",
    sha256 = "8b8c1ce375e1d8dd73f905e99bd452243ec194dd707a36fa5bdea7a252165c60",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/perl-libintl-perl-1.29-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8b8c1ce375e1d8dd73f905e99bd452243ec194dd707a36fa5bdea7a252165c60",
    ],
)

rpm(
    name = "perl-libs-4__5.26.3-421.el8.x86_64",
    sha256 = "d3a5510385cd4b2d53d70942e4fb4c149917aac2ce2df881c28ae2afdcd26619",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-libs-5.26.3-421.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d3a5510385cd4b2d53d70942e4fb4c149917aac2ce2df881c28ae2afdcd26619",
    ],
)

rpm(
    name = "perl-macros-4__5.26.3-421.el8.x86_64",
    sha256 = "5969bb5bd8b28a6cead135cfbdae89ac60f649b29f88a1daac3016eea47dc45b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-macros-5.26.3-421.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5969bb5bd8b28a6cead135cfbdae89ac60f649b29f88a1daac3016eea47dc45b",
    ],
)

rpm(
    name = "perl-parent-1__0.237-1.el8.x86_64",
    sha256 = "f5e73bbd776a2426a796971d8d38664f2e94898479fb76947dccdd28cf9fe1d0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-parent-0.237-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f5e73bbd776a2426a796971d8d38664f2e94898479fb76947dccdd28cf9fe1d0",
    ],
)

rpm(
    name = "perl-podlators-0__4.11-1.el8.x86_64",
    sha256 = "78d17ed089151e7fa3d1a3cdbbac8ca3b1b5c484fae5ba025642cc9107991037",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-podlators-4.11-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/78d17ed089151e7fa3d1a3cdbbac8ca3b1b5c484fae5ba025642cc9107991037",
    ],
)

rpm(
    name = "perl-threads-1__2.21-2.el8.x86_64",
    sha256 = "2e3da17b1c1685edea9c52bdaa0d77c019d6144c765fc6b3b1c783d98f634f96",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-threads-2.21-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2e3da17b1c1685edea9c52bdaa0d77c019d6144c765fc6b3b1c783d98f634f96",
    ],
)

rpm(
    name = "perl-threads-shared-0__1.58-2.el8.x86_64",
    sha256 = "b4a14dc0e3550da946d7ca65e54d19fc805e30c6c3dbf5ef3fc077d1d94e6d71",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/perl-threads-shared-1.58-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4a14dc0e3550da946d7ca65e54d19fc805e30c6c3dbf5ef3fc077d1d94e6d71",
    ],
)

rpm(
    name = "pixman-0__0.38.4-2.el8.aarch64",
    sha256 = "038eba8224034c5090cd08184c68a25ff8037dee804ad3eae0109a1cf4096078",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/pixman-0.38.4-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/038eba8224034c5090cd08184c68a25ff8037dee804ad3eae0109a1cf4096078",
    ],
)

rpm(
    name = "pixman-0__0.38.4-2.el8.x86_64",
    sha256 = "e496740940bd0b4d6f6537feaaffff57580624f6629c736c7f5e415259dc6cbe",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/pixman-0.38.4-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e496740940bd0b4d6f6537feaaffff57580624f6629c736c7f5e415259dc6cbe",
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
    name = "platform-python-0__3.6.8-47.el8.aarch64",
    sha256 = "43ffa547514ccad75bc69b6fdc402cc133234b33da4a62ddacc3c51ebf738fd0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-3.6.8-47.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/43ffa547514ccad75bc69b6fdc402cc133234b33da4a62ddacc3c51ebf738fd0",
    ],
)

rpm(
    name = "platform-python-0__3.6.8-47.el8.x86_64",
    sha256 = "ead951c74984ba09c297c7286533b4b4ce2fcc18fa60102c760016e761a85a73",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-3.6.8-47.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ead951c74984ba09c297c7286533b4b4ce2fcc18fa60102c760016e761a85a73",
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
    name = "policycoreutils-0__2.9-20.el8.aarch64",
    sha256 = "c9b9b0ebb76076878a19bda6c762ae165c5ce7b2d5109b5be391c60015d8a7dc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/policycoreutils-2.9-20.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c9b9b0ebb76076878a19bda6c762ae165c5ce7b2d5109b5be391c60015d8a7dc",
    ],
)

rpm(
    name = "policycoreutils-0__2.9-20.el8.x86_64",
    sha256 = "341b432d82c58ba14392e38d0ab9aa1e9686a18d0be72491832e6dc697400b17",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/policycoreutils-2.9-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/341b432d82c58ba14392e38d0ab9aa1e9686a18d0be72491832e6dc697400b17",
    ],
)

rpm(
    name = "polkit-0__0.115-13.0.1.el8.2.aarch64",
    sha256 = "eef4d3b177ff36c7f1781fcb456bef44169484a29f5931f268486f15933e4b24",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-0.115-13.0.1.el8.2.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eef4d3b177ff36c7f1781fcb456bef44169484a29f5931f268486f15933e4b24",
    ],
)

rpm(
    name = "polkit-0__0.115-13.0.1.el8.2.x86_64",
    sha256 = "8bfccf9235747eb132c1d10c2f26b5544a0db078019eb7911b88522131e16dc8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-0.115-13.0.1.el8.2.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8bfccf9235747eb132c1d10c2f26b5544a0db078019eb7911b88522131e16dc8",
    ],
)

rpm(
    name = "polkit-libs-0__0.115-13.0.1.el8.2.aarch64",
    sha256 = "dc74d77dfeb155b2708820c9a1d5cbb2c4c29de2c3a1cb76d0987e6bbbf40c9a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.115-13.0.1.el8.2.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dc74d77dfeb155b2708820c9a1d5cbb2c4c29de2c3a1cb76d0987e6bbbf40c9a",
    ],
)

rpm(
    name = "polkit-libs-0__0.115-13.0.1.el8.2.x86_64",
    sha256 = "d957da6b452f7b15830ad9a73176d4f04d9c3e26e119b7f3f4f4060087bb9082",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.115-13.0.1.el8.2.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d957da6b452f7b15830ad9a73176d4f04d9c3e26e119b7f3f4f4060087bb9082",
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
    name = "procps-ng-0__3.3.15-9.el8.aarch64",
    sha256 = "9811ac732f8266ec4ff97b314abb403279805e735740ec039c57d37cd4b82333",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.15-9.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9811ac732f8266ec4ff97b314abb403279805e735740ec039c57d37cd4b82333",
    ],
)

rpm(
    name = "procps-ng-0__3.3.15-9.el8.x86_64",
    sha256 = "8b518929d9973761aa2551766bc0be5d1b1c8d06be8ca294aeb1b23ecceb8451",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.15-9.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8b518929d9973761aa2551766bc0be5d1b1c8d06be8ca294aeb1b23ecceb8451",
    ],
)

rpm(
    name = "psmisc-0__23.1-5.el8.aarch64",
    sha256 = "e6852f9e715174c037c57ef9ee45a6318775968322c244185fc51f40a10dbdcc",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/psmisc-23.1-5.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e6852f9e715174c037c57ef9ee45a6318775968322c244185fc51f40a10dbdcc",
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
    name = "python3-libs-0__3.6.8-47.el8.aarch64",
    sha256 = "1ec95b8b8d4e226558d193bd46d3e928c143e41e5c0403a8868f872f7a7d2ad1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-libs-3.6.8-47.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1ec95b8b8d4e226558d193bd46d3e928c143e41e5c0403a8868f872f7a7d2ad1",
    ],
)

rpm(
    name = "python3-libs-0__3.6.8-47.el8.x86_64",
    sha256 = "279a02854cd438f33d624c86cfa2b3c266f04eda7cb8a81d1d70970f8c6c90fa",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-libs-3.6.8-47.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/279a02854cd438f33d624c86cfa2b3c266f04eda7cb8a81d1d70970f8c6c90fa",
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
    name = "qemu-img-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "af3133d3653a921ca543317bc1bc327fc3c853abfe71d7c8343af4bd8885cfaa",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/qemu-img-6.2.0-5.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/af3133d3653a921ca543317bc1bc327fc3c853abfe71d7c8343af4bd8885cfaa",
    ],
)

rpm(
    name = "qemu-img-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "dc7da9491c187c7002447c9041aabf4277a1e312ccf8acbab074cf77d0dcc9a8",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/qemu-img-6.2.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dc7da9491c187c7002447c9041aabf4277a1e312ccf8acbab074cf77d0dcc9a8",
    ],
)

rpm(
    name = "qemu-kvm-common-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "e51be1ba77f9e5436483e748bea7dd141c26f5557764cbebbece8f175034a2ab",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-6.2.0-5.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e51be1ba77f9e5436483e748bea7dd141c26f5557764cbebbece8f175034a2ab",
    ],
)

rpm(
    name = "qemu-kvm-common-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "f64ef0a04bc8e2448070b0bffe26b67c81bbcb505c45050d2d6f628510fb7960",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-6.2.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f64ef0a04bc8e2448070b0bffe26b67c81bbcb505c45050d2d6f628510fb7960",
    ],
)

rpm(
    name = "qemu-kvm-core-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.aarch64",
    sha256 = "28b10ff340e60d70ded17ce0b06dfe19962cf9f5c8e0c04d50bbd0becaeb99f2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-6.2.0-5.module_el8.6.0+1087+b42c8331.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/28b10ff340e60d70ded17ce0b06dfe19962cf9f5c8e0c04d50bbd0becaeb99f2",
    ],
)

rpm(
    name = "qemu-kvm-core-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "98f1aadc4858c7aa6f7aa052f494e8fbfc46dc4bdd278fb6195a35918775c9c3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-6.2.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/98f1aadc4858c7aa6f7aa052f494e8fbfc46dc4bdd278fb6195a35918775c9c3",
    ],
)

rpm(
    name = "qemu-kvm-hw-usbredir-15__6.2.0-5.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "c14bb507dd173802c5e8aee7264071a70fe5a0ac3de3e93cc3996e35f1c1bac1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/qemu-kvm-hw-usbredir-6.2.0-5.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c14bb507dd173802c5e8aee7264071a70fe5a0ac3de3e93cc3996e35f1c1bac1",
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
    name = "rpm-0__4.14.3-23.el8.aarch64",
    sha256 = "d803f082920abc401f44b7220ce96f6f2b070b06dcfe6b5c34573b8c7bcc5267",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-4.14.3-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d803f082920abc401f44b7220ce96f6f2b070b06dcfe6b5c34573b8c7bcc5267",
    ],
)

rpm(
    name = "rpm-0__4.14.3-23.el8.x86_64",
    sha256 = "4fa7a471aeba9b03daad1306a727fa12edb4b633f96a3da627495b24d6a4f185",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-4.14.3-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4fa7a471aeba9b03daad1306a727fa12edb4b633f96a3da627495b24d6a4f185",
    ],
)

rpm(
    name = "rpm-libs-0__4.14.3-23.el8.aarch64",
    sha256 = "26fdda368fc8c50c774cebd9ddf4786ced58d8ee9b12e5ce57113205d147f0a1",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.14.3-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/26fdda368fc8c50c774cebd9ddf4786ced58d8ee9b12e5ce57113205d147f0a1",
    ],
)

rpm(
    name = "rpm-libs-0__4.14.3-23.el8.x86_64",
    sha256 = "59cdcaac989655b450a369c41282b2dc312a1e5b24f5be0233d15035a3682400",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.14.3-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/59cdcaac989655b450a369c41282b2dc312a1e5b24f5be0233d15035a3682400",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-23.el8.aarch64",
    sha256 = "66c8e46bde5c784c083c7e674f72edb493394c9dedf59e7b40600968f083ca5c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.14.3-23.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/66c8e46bde5c784c083c7e674f72edb493394c9dedf59e7b40600968f083ca5c",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-23.el8.x86_64",
    sha256 = "2f55d15cb498f2613ebaf6a59bc0303579ae5b80f6edfc3c0c226125b2d2ca30",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.14.3-23.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2f55d15cb498f2613ebaf6a59bc0303579ae5b80f6edfc3c0c226125b2d2ca30",
    ],
)

rpm(
    name = "seabios-0__1.15.0-1.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "4d421d4139e7ad6e5a2ec8be8f347bc16a871571525d6b8d2ae251436d4bd89f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/seabios-1.15.0-1.module_el8.6.0+1087+b42c8331.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d421d4139e7ad6e5a2ec8be8f347bc16a871571525d6b8d2ae251436d4bd89f",
    ],
)

rpm(
    name = "seabios-bin-0__1.15.0-1.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "3c8d058cabbdad4e9780aab2f3770c8162bfc28f837dd6036690497b82101d3f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/seabios-bin-1.15.0-1.module_el8.6.0+1087+b42c8331.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c8d058cabbdad4e9780aab2f3770c8162bfc28f837dd6036690497b82101d3f",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.15.0-1.module_el8.6.0__plus__1087__plus__b42c8331.x86_64",
    sha256 = "34d9c5e00e88a00e8be874470dc2f1460f7957335fd0081936e8a17fcf66605c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/seavgabios-bin-1.15.0-1.module_el8.6.0+1087+b42c8331.noarch.rpm",
        "https://storage.googleapis.com/builddeps/34d9c5e00e88a00e8be874470dc2f1460f7957335fd0081936e8a17fcf66605c",
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
    name = "selinux-policy-0__3.14.3-108.el8.aarch64",
    sha256 = "84b49fd4b40c26b7dcfd05fcfe9b249af48798c45749e6b25dd6e2017eb1547b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-3.14.3-108.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/84b49fd4b40c26b7dcfd05fcfe9b249af48798c45749e6b25dd6e2017eb1547b",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.3-108.el8.x86_64",
    sha256 = "84b49fd4b40c26b7dcfd05fcfe9b249af48798c45749e6b25dd6e2017eb1547b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-3.14.3-108.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/84b49fd4b40c26b7dcfd05fcfe9b249af48798c45749e6b25dd6e2017eb1547b",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-108.el8.aarch64",
    sha256 = "f41687f7f44a1f7bb0bfa60325e9cf9036970dc74cf650f94c8fb6b3baf3036a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-3.14.3-108.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f41687f7f44a1f7bb0bfa60325e9cf9036970dc74cf650f94c8fb6b3baf3036a",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-108.el8.x86_64",
    sha256 = "f41687f7f44a1f7bb0bfa60325e9cf9036970dc74cf650f94c8fb6b3baf3036a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-3.14.3-108.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f41687f7f44a1f7bb0bfa60325e9cf9036970dc74cf650f94c8fb6b3baf3036a",
    ],
)

rpm(
    name = "setup-0__2.12.2-7.el8.aarch64",
    sha256 = "0e5bdfebabb44848a9f37d2cc02a8a6a099b1c4c1644f4940718e55ce5b95464",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/setup-2.12.2-7.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0e5bdfebabb44848a9f37d2cc02a8a6a099b1c4c1644f4940718e55ce5b95464",
    ],
)

rpm(
    name = "setup-0__2.12.2-7.el8.x86_64",
    sha256 = "0e5bdfebabb44848a9f37d2cc02a8a6a099b1c4c1644f4940718e55ce5b95464",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/setup-2.12.2-7.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0e5bdfebabb44848a9f37d2cc02a8a6a099b1c4c1644f4940718e55ce5b95464",
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
    name = "shadow-utils-2__4.6-17.el8.aarch64",
    sha256 = "c2ed285e2a2495b33e926c57e1917114c7898f2f4536866d643f206780a699af",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.6-17.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c2ed285e2a2495b33e926c57e1917114c7898f2f4536866d643f206780a699af",
    ],
)

rpm(
    name = "shadow-utils-2__4.6-17.el8.x86_64",
    sha256 = "fb3c71778fc23c4d3c91911c49e0a0d14c8a5192c431fc9ba07f2a14c938a172",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.6-17.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb3c71778fc23c4d3c91911c49e0a0d14c8a5192c431fc9ba07f2a14c938a172",
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
    name = "sqlite-libs-0__3.26.0-16.el8.aarch64",
    sha256 = "dd9b9c781a443d2712cdc5268dfa54116dfcf6c659df4e6da593f65e17ea0f60",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.26.0-16.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dd9b9c781a443d2712cdc5268dfa54116dfcf6c659df4e6da593f65e17ea0f60",
    ],
)

rpm(
    name = "sqlite-libs-0__3.26.0-16.el8.x86_64",
    sha256 = "1edb5a767311032bf7a35acd1db8c09cb86cb37c700ab3c412e8809bf962938e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.26.0-16.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1edb5a767311032bf7a35acd1db8c09cb86cb37c700ab3c412e8809bf962938e",
    ],
)

rpm(
    name = "sssd-client-0__2.7.3-4.el8.aarch64",
    sha256 = "3be860b5a9682f3374fad6a70597e023b9c7198434765d17cd76cd61639bb997",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sssd-client-2.7.3-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3be860b5a9682f3374fad6a70597e023b9c7198434765d17cd76cd61639bb997",
    ],
)

rpm(
    name = "sssd-client-0__2.7.3-4.el8.x86_64",
    sha256 = "23e5d6088f4bb34716ded62f4bcf4c084abada94f1b8cd69e2597ebda008deda",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sssd-client-2.7.3-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23e5d6088f4bb34716ded62f4bcf4c084abada94f1b8cd69e2597ebda008deda",
    ],
)

rpm(
    name = "swtpm-0__0.7.0-4.20211109gitb79fd91.module_el8.7.0__plus__1218__plus__f626c2ff.aarch64",
    sha256 = "c4acfc0bd76c4dd286887e20aac02c9bdb83b2357012641b32605207ad619ff6",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-0.7.0-4.20211109gitb79fd91.module_el8.7.0+1218+f626c2ff.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c4acfc0bd76c4dd286887e20aac02c9bdb83b2357012641b32605207ad619ff6",
    ],
)

rpm(
    name = "swtpm-0__0.7.0-4.20211109gitb79fd91.module_el8.7.0__plus__1218__plus__f626c2ff.x86_64",
    sha256 = "2125c4d6cb910e47daf45fbef10d75f93b5d30e64908b42dfc77aeee201feb60",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-0.7.0-4.20211109gitb79fd91.module_el8.7.0+1218+f626c2ff.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2125c4d6cb910e47daf45fbef10d75f93b5d30e64908b42dfc77aeee201feb60",
    ],
)

rpm(
    name = "swtpm-libs-0__0.7.0-4.20211109gitb79fd91.module_el8.7.0__plus__1218__plus__f626c2ff.aarch64",
    sha256 = "31d1177c9161063114580f26614cd64d90b3dc1f9163b317f52cb9d99dc128d5",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.7.0-4.20211109gitb79fd91.module_el8.7.0+1218+f626c2ff.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/31d1177c9161063114580f26614cd64d90b3dc1f9163b317f52cb9d99dc128d5",
    ],
)

rpm(
    name = "swtpm-libs-0__0.7.0-4.20211109gitb79fd91.module_el8.7.0__plus__1218__plus__f626c2ff.x86_64",
    sha256 = "f29e2f9e3f3c4ba3cddbe4af4dc7db2e7ad0088db6e955da86dacb40d4e75466",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.7.0-4.20211109gitb79fd91.module_el8.7.0+1218+f626c2ff.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f29e2f9e3f3c4ba3cddbe4af4dc7db2e7ad0088db6e955da86dacb40d4e75466",
    ],
)

rpm(
    name = "swtpm-tools-0__0.7.0-4.20211109gitb79fd91.module_el8.7.0__plus__1218__plus__f626c2ff.aarch64",
    sha256 = "9677ebd929255a1a4b10a8ea834f7b4771ba3243ed12c37c1ee5e0cf7ab82938",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.7.0-4.20211109gitb79fd91.module_el8.7.0+1218+f626c2ff.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9677ebd929255a1a4b10a8ea834f7b4771ba3243ed12c37c1ee5e0cf7ab82938",
    ],
)

rpm(
    name = "swtpm-tools-0__0.7.0-4.20211109gitb79fd91.module_el8.7.0__plus__1218__plus__f626c2ff.x86_64",
    sha256 = "bb88081e4d8978aaea3e902252be225211fc496f053ac721757a8b005c3ad86d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.7.0-4.20211109gitb79fd91.module_el8.7.0+1218+f626c2ff.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb88081e4d8978aaea3e902252be225211fc496f053ac721757a8b005c3ad86d",
    ],
)

rpm(
    name = "systemd-0__239-67.el8.aarch64",
    sha256 = "5a2637fb3502c931e9c48402773df76912d7cb8bf0c9b6a55531f3db3ac9842e",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-239-67.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5a2637fb3502c931e9c48402773df76912d7cb8bf0c9b6a55531f3db3ac9842e",
    ],
)

rpm(
    name = "systemd-0__239-67.el8.x86_64",
    sha256 = "153f5ff023d68681d0ca0220739e6c7c764ad23e7f784ca340613fca8aa468d0",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-239-67.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/153f5ff023d68681d0ca0220739e6c7c764ad23e7f784ca340613fca8aa468d0",
    ],
)

rpm(
    name = "systemd-container-0__239-67.el8.aarch64",
    sha256 = "01a0553b2ae7a013752bc3d4e59c8e4d5d3912dfd9090abdc25ff0de76f4590b",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-container-239-67.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01a0553b2ae7a013752bc3d4e59c8e4d5d3912dfd9090abdc25ff0de76f4590b",
    ],
)

rpm(
    name = "systemd-container-0__239-67.el8.x86_64",
    sha256 = "92864c6c250432c75119c96e4599177df21c4f922f7b87b33bbe0b408f2aa5e7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-container-239-67.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/92864c6c250432c75119c96e4599177df21c4f922f7b87b33bbe0b408f2aa5e7",
    ],
)

rpm(
    name = "systemd-libs-0__239-67.el8.aarch64",
    sha256 = "fbdcddaee9f71505cbf46cd61a4ed31e55ce3eca47db334f44886d73813b57ef",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-libs-239-67.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fbdcddaee9f71505cbf46cd61a4ed31e55ce3eca47db334f44886d73813b57ef",
    ],
)

rpm(
    name = "systemd-libs-0__239-67.el8.x86_64",
    sha256 = "f77c3d5836af005071ef92358742a28c517e65d0da74f1313966f4e86edcec0d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-libs-239-67.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f77c3d5836af005071ef92358742a28c517e65d0da74f1313966f4e86edcec0d",
    ],
)

rpm(
    name = "systemd-pam-0__239-67.el8.aarch64",
    sha256 = "32a772300fd1c6fbec40cf5b319e29de76c76916f2b00280a27204f9c41e3b01",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-pam-239-67.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/32a772300fd1c6fbec40cf5b319e29de76c76916f2b00280a27204f9c41e3b01",
    ],
)

rpm(
    name = "systemd-pam-0__239-67.el8.x86_64",
    sha256 = "888a7bddd28e89b74e432200d8dde7e7d3401d60944b082c4e1dfaba3e31f50a",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-pam-239-67.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/888a7bddd28e89b74e432200d8dde7e7d3401d60944b082c4e1dfaba3e31f50a",
    ],
)

rpm(
    name = "tar-2__1.30-6.el8.aarch64",
    sha256 = "ef568db2a1acf8da0aa45c2378fd517150d3c878b025c0c5e030471ddb548772",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/tar-1.30-6.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ef568db2a1acf8da0aa45c2378fd517150d3c878b025c0c5e030471ddb548772",
    ],
)

rpm(
    name = "tar-2__1.30-6.el8.x86_64",
    sha256 = "3c58fd72932efeccda39578fd55a37d9544a1f64c0ffeebad1c2741fba55fda2",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/tar-1.30-6.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3c58fd72932efeccda39578fd55a37d9544a1f64c0ffeebad1c2741fba55fda2",
    ],
)

rpm(
    name = "tzdata-0__2022c-1.el8.aarch64",
    sha256 = "aec23ed2a3c13dece3c9afbcb455feb399849478042d02b9c2ce29f5bcaef552",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/tzdata-2022c-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aec23ed2a3c13dece3c9afbcb455feb399849478042d02b9c2ce29f5bcaef552",
    ],
)

rpm(
    name = "tzdata-0__2022c-1.el8.x86_64",
    sha256 = "aec23ed2a3c13dece3c9afbcb455feb399849478042d02b9c2ce29f5bcaef552",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/tzdata-2022c-1.el8.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aec23ed2a3c13dece3c9afbcb455feb399849478042d02b9c2ce29f5bcaef552",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-2.el8.aarch64",
    sha256 = "39fe6d556d9456718922c74b639dfea8991a49cf109b6dff9872826f02e33934",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/unbound-libs-1.16.2-2.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/39fe6d556d9456718922c74b639dfea8991a49cf109b6dff9872826f02e33934",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-2.el8.x86_64",
    sha256 = "9d1fd4ba858e6788c32a6dd3adaa8db51fc4c1ae34366fe62ef136dbfa64a9b7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/unbound-libs-1.16.2-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d1fd4ba858e6788c32a6dd3adaa8db51fc4c1ae34366fe62ef136dbfa64a9b7",
    ],
)

rpm(
    name = "usbredir-0__0.12.0-2.el8.x86_64",
    sha256 = "0b6e50e9e9c68d0dbacc39e81c4a3a3a7ccf3afaddf40afb06ca86424a46ba23",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/usbredir-0.12.0-2.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0b6e50e9e9c68d0dbacc39e81c4a3a3a7ccf3afaddf40afb06ca86424a46ba23",
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
    name = "util-linux-0__2.32.1-38.el8.aarch64",
    sha256 = "590e73677f9c23b838bb78dce0ae886366b7b946252dc70c757db004175233bb",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/util-linux-2.32.1-38.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/590e73677f9c23b838bb78dce0ae886366b7b946252dc70c757db004175233bb",
    ],
)

rpm(
    name = "util-linux-0__2.32.1-38.el8.x86_64",
    sha256 = "c1bba56c2968815d47112cca2ac99f3054097897a7a82855ced2511aee50bb63",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/util-linux-2.32.1-38.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c1bba56c2968815d47112cca2ac99f3054097897a7a82855ced2511aee50bb63",
    ],
)

rpm(
    name = "vim-minimal-2__8.0.1763-19.el8.4.aarch64",
    sha256 = "4a921c33ca497386a80d4f6ace2ec54bc8e568c83f6197daa9a0f29b8a97fe1d",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.0.1763-19.el8.4.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4a921c33ca497386a80d4f6ace2ec54bc8e568c83f6197daa9a0f29b8a97fe1d",
    ],
)

rpm(
    name = "vim-minimal-2__8.0.1763-19.el8.4.x86_64",
    sha256 = "8d1659cf14095e2a82da7b2b7c21e5b62fda058590ea66b9e3d33a6794449e2c",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.0.1763-19.el8.4.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8d1659cf14095e2a82da7b2b7c21e5b62fda058590ea66b9e3d33a6794449e2c",
    ],
)

rpm(
    name = "which-0__2.21-18.el8.aarch64",
    sha256 = "c27e749065a42c812467155241ee9eedfcaae0f08f4cec952aa65194e98723d7",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/which-2.21-18.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c27e749065a42c812467155241ee9eedfcaae0f08f4cec952aa65194e98723d7",
    ],
)

rpm(
    name = "which-0__2.21-18.el8.x86_64",
    sha256 = "0e4d5ee4cbea952903ee4febb1450caf92bf3c2d6ecac9d0dd8ac8611e9ff4db",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/which-2.21-18.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0e4d5ee4cbea952903ee4febb1450caf92bf3c2d6ecac9d0dd8ac8611e9ff4db",
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
    name = "xz-0__5.2.4-4.el8.aarch64",
    sha256 = "c30b066af6b844602964858ef77b995e944ffbdd7a153a9c5c7fc30fd802b926",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/xz-5.2.4-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c30b066af6b844602964858ef77b995e944ffbdd7a153a9c5c7fc30fd802b926",
    ],
)

rpm(
    name = "xz-0__5.2.4-4.el8.x86_64",
    sha256 = "99d7d4bfee1d5b55e08ee27c6869186531939f399d6c3ea33db191cae7e53f70",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/xz-5.2.4-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/99d7d4bfee1d5b55e08ee27c6869186531939f399d6c3ea33db191cae7e53f70",
    ],
)

rpm(
    name = "xz-libs-0__5.2.4-4.el8.aarch64",
    sha256 = "9498f961afe361c5f9e0eea0ce64f11071b1cb1afe30636cb888d109737ea16f",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/xz-libs-5.2.4-4.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9498f961afe361c5f9e0eea0ce64f11071b1cb1afe30636cb888d109737ea16f",
    ],
)

rpm(
    name = "xz-libs-0__5.2.4-4.el8.x86_64",
    sha256 = "69d67ea8b4bd532f750ff0592f0098ace60470da0fd0e4056188fda37a268d42",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/xz-libs-5.2.4-4.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/69d67ea8b4bd532f750ff0592f0098ace60470da0fd0e4056188fda37a268d42",
    ],
)

rpm(
    name = "yajl-0__2.1.0-11.el8.aarch64",
    sha256 = "3ae671d2c8bfd1f53ea706e3969dd2dafd5a2960371e8b6f6083fb345985a491",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/yajl-2.1.0-11.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3ae671d2c8bfd1f53ea706e3969dd2dafd5a2960371e8b6f6083fb345985a491",
    ],
)

rpm(
    name = "yajl-0__2.1.0-11.el8.x86_64",
    sha256 = "55a094ffe9f378ef465619bf6f60e9f26b672f67236883565fb893de7675c163",
    urls = [
        "http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/yajl-2.1.0-11.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/55a094ffe9f378ef465619bf6f60e9f26b672f67236883565fb893de7675c163",
    ],
)

rpm(
    name = "zlib-0__1.2.11-20.el8.aarch64",
    sha256 = "c6dbfad47ac76904024403eecfe97dd2a84d51ef29709c6e89572fae922adce3",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/zlib-1.2.11-20.el8.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c6dbfad47ac76904024403eecfe97dd2a84d51ef29709c6e89572fae922adce3",
    ],
)

rpm(
    name = "zlib-0__1.2.11-20.el8.x86_64",
    sha256 = "f28062598508e566a453f26398b7165a565b706800498b77f8a8249821ac2674",
    urls = [
        "http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/zlib-1.2.11-20.el8.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f28062598508e566a453f26398b7165a565b706800498b77f8a8249821ac2674",
    ],
)
