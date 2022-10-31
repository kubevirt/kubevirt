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
    sha256 = "099a9fb96a376ccbbb7d291ed4ecbdfd42f6bc822ab77ae6f1b5cb9e914e94fa",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.35.0/rules_go-v0.35.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.35.0/rules_go-v0.35.0.zip",
    ],
)

# XXX: For now stick with 0.24. The resultion mode changed from 'external' to 'static' which causes some import troubles.
# See https://github.com/bazelbuild/bazel-gazelle/pull/1264#issuecomment-1288680264 for details. Can be resolved at some point.
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
    sha256 = "46fdbc00930c8dc9d84690b5bd94db6b4683b061199967d2cda1cfbda8f02c49",
    strip_prefix = "bazel-tools-19b174803c0db1a01e77f10fa2079c35f54eed6e",
    urls = [
        "https://github.com/ash2k/bazel-tools/archive/19b174803c0db1a01e77f10fa2079c35f54eed6e.zip",
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
    sha256 = "404fc34e6bd3b568a7ca6fbcde70267d43830d0171d3192e3ecd83c14c320cfc",
    strip_prefix = "bazeldnf-0.5.4",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.5.4.tar.gz",
        "https://storage.googleapis.com/builddeps/404fc34e6bd3b568a7ca6fbcde70267d43830d0171d3192e3ecd83c14c320cfc",
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
    go_version = "1.19.2",
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

bazeldnf_dependencies()

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
    sha256 = "59fe17973fdaf4d969203b66b1446d855d406aea0736d06ee1cd624100942c8f",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/appliance-1.48.4-linux-5.14.0-176-centos9.tar.xz",
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

rpm(
    name = "acl-0__2.3.1-3.el9.aarch64",
    sha256 = "151d6542a39243b5f65698b31edfe2d9c59e2fd71a7dcaa237442fc5d1d9de1e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/acl-2.3.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/151d6542a39243b5f65698b31edfe2d9c59e2fd71a7dcaa237442fc5d1d9de1e",
    ],
)

rpm(
    name = "acl-0__2.3.1-3.el9.x86_64",
    sha256 = "986044c3837eddbc9231d7be5e5fc517e245296978b988a803bc9f9172fe84ea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/acl-2.3.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/986044c3837eddbc9231d7be5e5fc517e245296978b988a803bc9f9172fe84ea",
    ],
)

rpm(
    name = "alternatives-0__1.20-2.el9.aarch64",
    sha256 = "4d9055232088f1ab181e4741358aa188749b8195f184817c04a61447606cdfb5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/alternatives-1.20-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4d9055232088f1ab181e4741358aa188749b8195f184817c04a61447606cdfb5",
    ],
)

rpm(
    name = "alternatives-0__1.20-2.el9.x86_64",
    sha256 = "1851d5f64ebaeac67c5c2d9e4adc1e73aa6433b44a167268a3510c3d056062db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/alternatives-1.20-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1851d5f64ebaeac67c5c2d9e4adc1e73aa6433b44a167268a3510c3d056062db",
    ],
)

rpm(
    name = "audit-libs-0__3.0.7-103.el9.aarch64",
    sha256 = "d76fb317d2c119de235f079463163dc5a6ed8df8073aa747463697cb667ca604",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/audit-libs-3.0.7-103.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d76fb317d2c119de235f079463163dc5a6ed8df8073aa747463697cb667ca604",
    ],
)

rpm(
    name = "audit-libs-0__3.0.7-103.el9.x86_64",
    sha256 = "cdd16764f76df434a731a331577fb03a51f19d0a8249ae782506e5ac12dabb0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.0.7-103.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cdd16764f76df434a731a331577fb03a51f19d0a8249ae782506e5ac12dabb0a",
    ],
)

rpm(
    name = "augeas-libs-0__1.13.0-3.el9.x86_64",
    sha256 = "f15b57d9629d67b29072782d540eb9ca4f89cac4f49de517afd8a0bb4f7ae025",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/augeas-libs-1.13.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f15b57d9629d67b29072782d540eb9ca4f89cac4f49de517afd8a0bb4f7ae025",
    ],
)

rpm(
    name = "basesystem-0__11-13.el9.aarch64",
    sha256 = "a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/basesystem-11-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    ],
)

rpm(
    name = "basesystem-0__11-13.el9.x86_64",
    sha256 = "a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/basesystem-11-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    ],
)

rpm(
    name = "bash-0__5.1.8-5.el9.aarch64",
    sha256 = "a420094f613ae39a964b7fc2accebe9a95abae8f85a3b126950df9a86e419ca8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bash-5.1.8-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a420094f613ae39a964b7fc2accebe9a95abae8f85a3b126950df9a86e419ca8",
    ],
)

rpm(
    name = "bash-0__5.1.8-5.el9.x86_64",
    sha256 = "fa0cd3cdd8500592a4d2f9749edd29732ad5cd3261d2b772ea6ffc6e94e2247d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bash-5.1.8-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fa0cd3cdd8500592a4d2f9749edd29732ad5cd3261d2b772ea6ffc6e94e2247d",
    ],
)

rpm(
    name = "binutils-0__2.35.2-24.el9.aarch64",
    sha256 = "2d3bf7f4a6777a143a54a571d0d8e3e744f0e2bdf42acdc3ecb6f669299046d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-24.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2d3bf7f4a6777a143a54a571d0d8e3e744f0e2bdf42acdc3ecb6f669299046d7",
    ],
)

rpm(
    name = "binutils-0__2.35.2-24.el9.x86_64",
    sha256 = "48f0daaa35a2e885d4eebcb90538f5c7703946a73e7e815f99272d022bf6c3a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-24.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/48f0daaa35a2e885d4eebcb90538f5c7703946a73e7e815f99272d022bf6c3a4",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-24.el9.aarch64",
    sha256 = "56ef60ed5ed96b8bceee6c1cad8f2d35905253c22d16482c5027021be69674d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-24.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56ef60ed5ed96b8bceee6c1cad8f2d35905253c22d16482c5027021be69674d7",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-24.el9.x86_64",
    sha256 = "bafe22650ec8524dd11a29b69c40c8189c85cdbad190485913518dbab878de52",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-24.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bafe22650ec8524dd11a29b69c40c8189c85cdbad190485913518dbab878de52",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-8.el9.aarch64",
    sha256 = "d89b742bc5327741c3ff26a7fb17a518552bbad74c0c299f147af57a0a208b93",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.8-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d89b742bc5327741c3ff26a7fb17a518552bbad74c0c299f147af57a0a208b93",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-8.el9.x86_64",
    sha256 = "90aeb088fad0093b1ca531387d38e1c32ad64efd56f2306eacc0edbc4c37e205",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.8-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/90aeb088fad0093b1ca531387d38e1c32ad64efd56f2306eacc0edbc4c37e205",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-8.el9.aarch64",
    sha256 = "6c20f6f13c274fa2487f95f1e3dddcee9b931ce222abebd2f1d9b3f7eb69fcde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.8-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6c20f6f13c274fa2487f95f1e3dddcee9b931ce222abebd2f1d9b3f7eb69fcde",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-8.el9.x86_64",
    sha256 = "fabd6b5c065c2b9d4a8d39a938ae577d801de2ddc73c8cdf6f7803db29c28d0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.8-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fabd6b5c065c2b9d4a8d39a938ae577d801de2ddc73c8cdf6f7803db29c28d0a",
    ],
)

rpm(
    name = "ca-certificates-0__2022.2.54-90.2.el9.aarch64",
    sha256 = "24978e8dd3e054583da86036657ab16e93da97a0bafc148ec28d871d8c15257c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ca-certificates-2022.2.54-90.2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/24978e8dd3e054583da86036657ab16e93da97a0bafc148ec28d871d8c15257c",
    ],
)

rpm(
    name = "ca-certificates-0__2022.2.54-90.2.el9.x86_64",
    sha256 = "24978e8dd3e054583da86036657ab16e93da97a0bafc148ec28d871d8c15257c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ca-certificates-2022.2.54-90.2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/24978e8dd3e054583da86036657ab16e93da97a0bafc148ec28d871d8c15257c",
    ],
)

rpm(
    name = "capstone-0__4.0.2-10.el9.aarch64",
    sha256 = "fe07aa69a9e6b70d0324e702b825ad55f330225ecb2af504f7026917e0ff197e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/capstone-4.0.2-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fe07aa69a9e6b70d0324e702b825ad55f330225ecb2af504f7026917e0ff197e",
    ],
)

rpm(
    name = "capstone-0__4.0.2-10.el9.x86_64",
    sha256 = "f6a9fdc6bcb5da1b2ce44ca7ed6289759c37add7adbb19916dd36d5bb4624a41",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/capstone-4.0.2-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f6a9fdc6bcb5da1b2ce44ca7ed6289759c37add7adbb19916dd36d5bb4624a41",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-18.el9.aarch64",
    sha256 = "4d08c97e3852712e5a46d37e1abf4fe234fbdbdfad0c3c047fe6f3f14881bd81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-9.0-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4d08c97e3852712e5a46d37e1abf4fe234fbdbdfad0c3c047fe6f3f14881bd81",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-18.el9.x86_64",
    sha256 = "4d08c97e3852712e5a46d37e1abf4fe234fbdbdfad0c3c047fe6f3f14881bd81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4d08c97e3852712e5a46d37e1abf4fe234fbdbdfad0c3c047fe6f3f14881bd81",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-18.el9.aarch64",
    sha256 = "7078f8a58d6749b6d755cf375291c283318b5fc1b81ba550513e4570d77b961d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-release-9.0-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7078f8a58d6749b6d755cf375291c283318b5fc1b81ba550513e4570d77b961d",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-18.el9.x86_64",
    sha256 = "7078f8a58d6749b6d755cf375291c283318b5fc1b81ba550513e4570d77b961d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-release-9.0-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7078f8a58d6749b6d755cf375291c283318b5fc1b81ba550513e4570d77b961d",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-18.el9.aarch64",
    sha256 = "447442d183d82e93a9516258dc272ba87207c9d5e755ca8e37d562dea92f1875",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-9.0-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/447442d183d82e93a9516258dc272ba87207c9d5e755ca8e37d562dea92f1875",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-18.el9.x86_64",
    sha256 = "447442d183d82e93a9516258dc272ba87207c9d5e755ca8e37d562dea92f1875",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/447442d183d82e93a9516258dc272ba87207c9d5e755ca8e37d562dea92f1875",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-33.el9.aarch64",
    sha256 = "688f21e86d8632a8f3c378e2d5b9cccc01b66d929b5eb8fb5234c3eec764b630",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.32-33.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/688f21e86d8632a8f3c378e2d5b9cccc01b66d929b5eb8fb5234c3eec764b630",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-33.el9.x86_64",
    sha256 = "ada0b3dfc46e2944206ca4af18a87067cc2a3d2f802ac7b49c627e3a46f1dd16",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.32-33.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ada0b3dfc46e2944206ca4af18a87067cc2a3d2f802ac7b49c627e3a46f1dd16",
    ],
)

rpm(
    name = "cpp-0__11.3.1-2.1.el9.aarch64",
    sha256 = "69e1a478bdabf2e2910ce66d5d728a2fd1e0e88e89dd9126c2c9173de487fdcb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/69e1a478bdabf2e2910ce66d5d728a2fd1e0e88e89dd9126c2c9173de487fdcb",
    ],
)

rpm(
    name = "cpp-0__11.3.1-2.1.el9.x86_64",
    sha256 = "cb3d2eab5287be321f142d6a450bdfafc1352c4bea6bfb065a8106e5b365f5b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.3.1-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cb3d2eab5287be321f142d6a450bdfafc1352c4bea6bfb065a8106e5b365f5b3",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-27.el9.aarch64",
    sha256 = "d92900088b558cd3c96c63db24b048a0f3ea575a0f8bfe66c26df4acfcb2f811",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cracklib-2.9.6-27.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d92900088b558cd3c96c63db24b048a0f3ea575a0f8bfe66c26df4acfcb2f811",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-27.el9.x86_64",
    sha256 = "be9deb2efd06b4b2c1c130acae94c687161d04830119e65a989d904ba9fd1864",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cracklib-2.9.6-27.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be9deb2efd06b4b2c1c130acae94c687161d04830119e65a989d904ba9fd1864",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-27.el9.aarch64",
    sha256 = "bfd16ac0aebb165d43d3139448ab8eac66d4d67c9eac506c3f3bef799f1352c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cracklib-dicts-2.9.6-27.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bfd16ac0aebb165d43d3139448ab8eac66d4d67c9eac506c3f3bef799f1352c2",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-27.el9.x86_64",
    sha256 = "01df2a72fcdf988132e82764ce1a22a5a9513fa253b54e17d23058bdb53c2d85",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cracklib-dicts-2.9.6-27.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/01df2a72fcdf988132e82764ce1a22a5a9513fa253b54e17d23058bdb53c2d85",
    ],
)

rpm(
    name = "crypto-policies-0__20221003-1.git04dee29.el9.aarch64",
    sha256 = "14f3359e27f31564af576a0d1f128d150aee38ae9d80ea9d71d24bf92296e152",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20221003-1.git04dee29.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/14f3359e27f31564af576a0d1f128d150aee38ae9d80ea9d71d24bf92296e152",
    ],
)

rpm(
    name = "crypto-policies-0__20221003-1.git04dee29.el9.x86_64",
    sha256 = "14f3359e27f31564af576a0d1f128d150aee38ae9d80ea9d71d24bf92296e152",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20221003-1.git04dee29.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/14f3359e27f31564af576a0d1f128d150aee38ae9d80ea9d71d24bf92296e152",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.4.3-5.el9.aarch64",
    sha256 = "f4967c4e3c8e728973e57d7a92f2071800e1d7759f331afec49bea5324d18f63",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cryptsetup-libs-2.4.3-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f4967c4e3c8e728973e57d7a92f2071800e1d7759f331afec49bea5324d18f63",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.4.3-5.el9.x86_64",
    sha256 = "af64f15f679b7cbb83821afb71e6878a668af7e3a7c5fe0e91033ed444afde21",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cryptsetup-libs-2.4.3-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/af64f15f679b7cbb83821afb71e6878a668af7e3a7c5fe0e91033ed444afde21",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-20.el9.aarch64",
    sha256 = "6a125a561c46ac5987780e8f1721392c40b01b845b89b79eeee101b0ce9d8510",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/curl-minimal-7.76.1-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a125a561c46ac5987780e8f1721392c40b01b845b89b79eeee101b0ce9d8510",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-20.el9.x86_64",
    sha256 = "bf13c7e7e2bc27832583526b4de3fec7153541597c574853d86bbdd11be92d0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/curl-minimal-7.76.1-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bf13c7e7e2bc27832583526b4de3fec7153541597c574853d86bbdd11be92d0a",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-21.el9.aarch64",
    sha256 = "3ad176b2bb0f6b89e8b9951cd207415d40859f1f46e9d60b79a6516a7783ff7c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-2.1.27-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3ad176b2bb0f6b89e8b9951cd207415d40859f1f46e9d60b79a6516a7783ff7c",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-21.el9.x86_64",
    sha256 = "b919e98a1da12adaf63056e4b3fe068541fdcaea5b891ac32c50f70074e7a682",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-2.1.27-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b919e98a1da12adaf63056e4b3fe068541fdcaea5b891ac32c50f70074e7a682",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-21.el9.aarch64",
    sha256 = "12e292b4e05934f8fc8ecc557b2b57c2844335a559f720140bb7810ef249c043",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-gssapi-2.1.27-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/12e292b4e05934f8fc8ecc557b2b57c2844335a559f720140bb7810ef249c043",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-21.el9.x86_64",
    sha256 = "c7cba5ec41adada2d95348705d91a5ef7b4bca2f82ca22440e881ad28d2d27d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-gssapi-2.1.27-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c7cba5ec41adada2d95348705d91a5ef7b4bca2f82ca22440e881ad28d2d27d0",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-21.el9.aarch64",
    sha256 = "898d7094964022ca527a6596550b8d46499b3274f8c6a1ee632a98961012d80c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-lib-2.1.27-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/898d7094964022ca527a6596550b8d46499b3274f8c6a1ee632a98961012d80c",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-21.el9.x86_64",
    sha256 = "fd4292a29759f9531bbc876d1818e7a83ccac76907234002f598671d7b338469",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-lib-2.1.27-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd4292a29759f9531bbc876d1818e7a83ccac76907234002f598671d7b338469",
    ],
)

rpm(
    name = "daxctl-libs-0__71.1-7.el9.x86_64",
    sha256 = "d5b01e4e24933d2d17ec94d83542b6d19f911ffa7d2af2d9ecc57e3ce551af6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/daxctl-libs-71.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5b01e4e24933d2d17ec94d83542b6d19f911ffa7d2af2d9ecc57e3ce551af6b",
    ],
)

rpm(
    name = "dbus-1__1.12.20-6.el9.aarch64",
    sha256 = "012ce5d17be2ccb6fc027f7e680c6d6ceee76bcf0a72aada226b4cffcced6f33",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-1.12.20-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/012ce5d17be2ccb6fc027f7e680c6d6ceee76bcf0a72aada226b4cffcced6f33",
    ],
)

rpm(
    name = "dbus-1__1.12.20-6.el9.x86_64",
    sha256 = "51569a0d4e000c77218f213b413c09eea06f5f14acf8ce57ca259151980de0f1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-1.12.20-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/51569a0d4e000c77218f213b413c09eea06f5f14acf8ce57ca259151980de0f1",
    ],
)

rpm(
    name = "dbus-broker-0__28-7.el9.aarch64",
    sha256 = "28a7abe52040dcda6e5d941206ef6e5c47478fcc06a9f05c2ab7dacc2afa9f42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-broker-28-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/28a7abe52040dcda6e5d941206ef6e5c47478fcc06a9f05c2ab7dacc2afa9f42",
    ],
)

rpm(
    name = "dbus-broker-0__28-7.el9.x86_64",
    sha256 = "dd65bddd728ed08dcdba5d06b5a5af9f958e5718e8cab938783241bd8f4d1131",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-broker-28-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd65bddd728ed08dcdba5d06b5a5af9f958e5718e8cab938783241bd8f4d1131",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-6.el9.aarch64",
    sha256 = "3da731122540ae37d50a5f434e3b0dc376fb9568e391553ece29e8438868a87a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.20-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3da731122540ae37d50a5f434e3b0dc376fb9568e391553ece29e8438868a87a",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-6.el9.x86_64",
    sha256 = "3da731122540ae37d50a5f434e3b0dc376fb9568e391553ece29e8438868a87a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.20-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3da731122540ae37d50a5f434e3b0dc376fb9568e391553ece29e8438868a87a",
    ],
)

rpm(
    name = "device-mapper-9__1.02.185-3.el9.aarch64",
    sha256 = "2ea771954009edfe99fa060e1c4d02d3b17f3cf08bb8fea6c5f54389cb95cb5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.185-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2ea771954009edfe99fa060e1c4d02d3b17f3cf08bb8fea6c5f54389cb95cb5f",
    ],
)

rpm(
    name = "device-mapper-9__1.02.185-3.el9.x86_64",
    sha256 = "099ff858d4785f80447f5207da8dba56e3910d33302a168fdedbf25b93f43e0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.185-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/099ff858d4785f80447f5207da8dba56e3910d33302a168fdedbf25b93f43e0d",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.185-3.el9.aarch64",
    sha256 = "cb93b4a6dbb5a16b5b972dc858e4750749caa4946c80c9a90b922f7cbb7c757b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.185-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb93b4a6dbb5a16b5b972dc858e4750749caa4946c80c9a90b922f7cbb7c757b",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.185-3.el9.x86_64",
    sha256 = "4c3791fcd753a4d81826e14b7a587b68e2a3a536e050dd6b235f7d5bae7c4979",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.185-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c3791fcd753a4d81826e14b7a587b68e2a3a536e050dd6b235f7d5bae7c4979",
    ],
)

rpm(
    name = "diffutils-0__3.7-12.el9.aarch64",
    sha256 = "4fea2be2558981a55a569cc7b93f17afce86bba830ebce32a0aa320e4759293e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/diffutils-3.7-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4fea2be2558981a55a569cc7b93f17afce86bba830ebce32a0aa320e4759293e",
    ],
)

rpm(
    name = "diffutils-0__3.7-12.el9.x86_64",
    sha256 = "fdebefc46badf2e700e00582041a0e5f5183dd4fdc04badfe47c91f030cea0ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/diffutils-3.7-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fdebefc46badf2e700e00582041a0e5f5183dd4fdc04badfe47c91f030cea0ce",
    ],
)

rpm(
    name = "dmidecode-1__3.3-7.el9.x86_64",
    sha256 = "2afb32bf0c30908817d57d221dbded83917aa8a88d2586e98ce548bad4f86e3d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dmidecode-3.3-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2afb32bf0c30908817d57d221dbded83917aa8a88d2586e98ce548bad4f86e3d",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-3.el9.aarch64",
    sha256 = "e2eea5a568a705df9ad4afa5b15bdaa1c7e8febef9ce0f51ea394f0145efb224",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.46.5-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e2eea5a568a705df9ad4afa5b15bdaa1c7e8febef9ce0f51ea394f0145efb224",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-3.el9.x86_64",
    sha256 = "7fbb88e8ad5b578e157f383da089c9af4d5361dec8d23495b00f7f1102d6e4de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.46.5-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7fbb88e8ad5b578e157f383da089c9af4d5361dec8d23495b00f7f1102d6e4de",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-3.el9.aarch64",
    sha256 = "efc31e2b3a79208457bafe9fc61f6bee935dd021216cfe18567936b95487fdd0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.46.5-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/efc31e2b3a79208457bafe9fc61f6bee935dd021216cfe18567936b95487fdd0",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-3.el9.x86_64",
    sha256 = "0626ca08ef0d4ddafbb7679eb3915c61f0496038f92263529715681952854d20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.46.5-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0626ca08ef0d4ddafbb7679eb3915c61f0496038f92263529715681952854d20",
    ],
)

rpm(
    name = "edk2-aarch64-0__20220826gitba0e0e4c6a-1.el9.aarch64",
    sha256 = "3c951f911063eada4e22c725b755c5912f6f600422315f50c41e61f326fdd3fa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20220826gitba0e0e4c6a-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c951f911063eada4e22c725b755c5912f6f600422315f50c41e61f326fdd3fa",
    ],
)

rpm(
    name = "edk2-ovmf-0__20220826gitba0e0e4c6a-1.el9.x86_64",
    sha256 = "e8c2564191a8793d0b8ea8ede095167790316f2c02d4ff53be944bd2882c7e99",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20220826gitba0e0e4c6a-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e8c2564191a8793d0b8ea8ede095167790316f2c02d4ff53be944bd2882c7e99",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.187-6.el9.aarch64",
    sha256 = "766f1ed91093291eda86cd836d1a2b2f0290780781b9aa5271572b95fd5d2087",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.187-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/766f1ed91093291eda86cd836d1a2b2f0290780781b9aa5271572b95fd5d2087",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.187-6.el9.x86_64",
    sha256 = "86f699ca0f75b7d2f122412fdaa70dbb83523fb19c2d2baf627b51230f4bb733",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.187-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86f699ca0f75b7d2f122412fdaa70dbb83523fb19c2d2baf627b51230f4bb733",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.187-6.el9.aarch64",
    sha256 = "94554707290f5e6dc77a345a1b958b9d919ba40d9824d10fddc206d0b3951c9f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.187-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/94554707290f5e6dc77a345a1b958b9d919ba40d9824d10fddc206d0b3951c9f",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.187-6.el9.x86_64",
    sha256 = "94554707290f5e6dc77a345a1b958b9d919ba40d9824d10fddc206d0b3951c9f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.187-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/94554707290f5e6dc77a345a1b958b9d919ba40d9824d10fddc206d0b3951c9f",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.187-6.el9.aarch64",
    sha256 = "9e5d400222c5dc2ae216d1f1a6eb4358354da86f6781412cfffe645fcd2be9b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.187-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9e5d400222c5dc2ae216d1f1a6eb4358354da86f6781412cfffe645fcd2be9b3",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.187-6.el9.x86_64",
    sha256 = "1518eddb479bc684b160778bfa238f637f5b4d51daabe4dcc75714607fae5187",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.187-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1518eddb479bc684b160778bfa238f637f5b4d51daabe4dcc75714607fae5187",
    ],
)

rpm(
    name = "elfutils-libs-0__0.187-6.el9.aarch64",
    sha256 = "bc585a28b98b45dcb905cf887de78a1f4679856b9bd95b16ec63adaa31bab86a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.187-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bc585a28b98b45dcb905cf887de78a1f4679856b9bd95b16ec63adaa31bab86a",
    ],
)

rpm(
    name = "elfutils-libs-0__0.187-6.el9.x86_64",
    sha256 = "9606fa58f2cb2c2c36b0206a4a12fcb1e45dfa3b6b855a9b84b30f768efcdc90",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.187-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9606fa58f2cb2c2c36b0206a4a12fcb1e45dfa3b6b855a9b84b30f768efcdc90",
    ],
)

rpm(
    name = "ethtool-2__5.10-4.el9.aarch64",
    sha256 = "997f0540541faec55f0407c9e92791ad44c68e6b428e9cd355b7790e3c8800d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ethtool-5.10-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/997f0540541faec55f0407c9e92791ad44c68e6b428e9cd355b7790e3c8800d6",
    ],
)

rpm(
    name = "ethtool-2__5.10-4.el9.x86_64",
    sha256 = "496b68212d637810fab7321d7de729cd28cffa0093a69133dbf786c10f20b005",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ethtool-5.10-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/496b68212d637810fab7321d7de729cd28cffa0093a69133dbf786c10f20b005",
    ],
)

rpm(
    name = "expat-0__2.4.9-1.el9.aarch64",
    sha256 = "94dd3c7cc615241c19be13ab9bfe524769520694cc281515d383a3918c07f84b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/expat-2.4.9-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/94dd3c7cc615241c19be13ab9bfe524769520694cc281515d383a3918c07f84b",
    ],
)

rpm(
    name = "expat-0__2.4.9-1.el9.x86_64",
    sha256 = "a29c66d2daa8026c6781fb1435e976244f13203c30dd3882afccce945dd5c5d1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.4.9-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a29c66d2daa8026c6781fb1435e976244f13203c30dd3882afccce945dd5c5d1",
    ],
)

rpm(
    name = "file-0__5.39-10.el9.x86_64",
    sha256 = "5127d8fba1f3b07e2982a4f21a2e4fa0f7dfb089681b2f10e267f5908735d625",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/file-5.39-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5127d8fba1f3b07e2982a4f21a2e4fa0f7dfb089681b2f10e267f5908735d625",
    ],
)

rpm(
    name = "file-libs-0__5.39-10.el9.x86_64",
    sha256 = "da4dcbcf8f49bc84db988884a208f823cf1876fa5db79a05a66f5f0a30f67a01",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/file-libs-5.39-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da4dcbcf8f49bc84db988884a208f823cf1876fa5db79a05a66f5f0a30f67a01",
    ],
)

rpm(
    name = "filesystem-0__3.16-2.el9.aarch64",
    sha256 = "0afb1f7582830fa9c8c58a6679ab3b4ccf8bbdf1c0c76908fea1429eec8b8a53",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/filesystem-3.16-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0afb1f7582830fa9c8c58a6679ab3b4ccf8bbdf1c0c76908fea1429eec8b8a53",
    ],
)

rpm(
    name = "filesystem-0__3.16-2.el9.x86_64",
    sha256 = "b69a472751268a1b9acd566dc7aa486fc1d6c8cb6d23f36d6a6dfead62e71475",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/filesystem-3.16-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b69a472751268a1b9acd566dc7aa486fc1d6c8cb6d23f36d6a6dfead62e71475",
    ],
)

rpm(
    name = "findutils-1__4.8.0-5.el9.aarch64",
    sha256 = "3ae59472d5b58715c452f74cb865dbe4058d155ed30d3908a96c51ccccaf4e82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/findutils-4.8.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3ae59472d5b58715c452f74cb865dbe4058d155ed30d3908a96c51ccccaf4e82",
    ],
)

rpm(
    name = "findutils-1__4.8.0-5.el9.x86_64",
    sha256 = "552548e6d6f9623ccd9d31bb185bba3a66730da6e9d02296b417d501356c3848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/findutils-4.8.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/552548e6d6f9623ccd9d31bb185bba3a66730da6e9d02296b417d501356c3848",
    ],
)

rpm(
    name = "fuse-0__2.9.9-15.el9.x86_64",
    sha256 = "f0f8b58029ffddf73c5147c67c8e5f90f60e0e315f195c25695ceb0e9fec9d4b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-2.9.9-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f0f8b58029ffddf73c5147c67c8e5f90f60e0e315f195c25695ceb0e9fec9d4b",
    ],
)

rpm(
    name = "fuse-common-0__3.10.2-5.el9.x86_64",
    sha256 = "a156d82484b61b6323524631d80c8d184042e8819a6d86a1b9b3076f3b5f3612",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-common-3.10.2-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a156d82484b61b6323524631d80c8d184042e8819a6d86a1b9b3076f3b5f3612",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-15.el9.aarch64",
    sha256 = "d82ebf3bcfe85eae2b34c4dbd507d8a0b0ac05d4df4c9ee16fee7555f36c7873",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/fuse-libs-2.9.9-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d82ebf3bcfe85eae2b34c4dbd507d8a0b0ac05d4df4c9ee16fee7555f36c7873",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-15.el9.x86_64",
    sha256 = "610c601daea8fa587c3ee43f2af06c25c506caf4588bf214e04de7eb960b95fa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.9-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/610c601daea8fa587c3ee43f2af06c25c506caf4588bf214e04de7eb960b95fa",
    ],
)

rpm(
    name = "gawk-0__5.1.0-6.el9.aarch64",
    sha256 = "656d23c583b0705eaad75cffbe880f2ec39c7d5b7a756c6a8853c2977eec331b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gawk-5.1.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/656d23c583b0705eaad75cffbe880f2ec39c7d5b7a756c6a8853c2977eec331b",
    ],
)

rpm(
    name = "gawk-0__5.1.0-6.el9.x86_64",
    sha256 = "6e6d77b76b1e89fe6f012cdc16111bea35eb4ceedac5040e5d81b5a066429af8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gawk-5.1.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6e6d77b76b1e89fe6f012cdc16111bea35eb4ceedac5040e5d81b5a066429af8",
    ],
)

rpm(
    name = "gcc-0__11.3.1-2.1.el9.aarch64",
    sha256 = "6eb900d99e97859ae8ff3cadd0dc17cb7590bcb83f36a696b4b2c0ff0d493f77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6eb900d99e97859ae8ff3cadd0dc17cb7590bcb83f36a696b4b2c0ff0d493f77",
    ],
)

rpm(
    name = "gcc-0__11.3.1-2.1.el9.x86_64",
    sha256 = "ba86de67c23d55ffb29a4086635317145152518d19a60e753cb831fbbb73fa05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.3.1-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ba86de67c23d55ffb29a4086635317145152518d19a60e753cb831fbbb73fa05",
    ],
)

rpm(
    name = "gdbm-libs-1__1.19-4.el9.aarch64",
    sha256 = "4fc723b43287c971507ec7899a1517dcc91abab962707febc7fdd9c1d865ace8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gdbm-libs-1.19-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4fc723b43287c971507ec7899a1517dcc91abab962707febc7fdd9c1d865ace8",
    ],
)

rpm(
    name = "gdbm-libs-1__1.19-4.el9.x86_64",
    sha256 = "8cd5a78cab8783dd241c52c4fcda28fb111c443887dd6d0fe38385e8383c98b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.19-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8cd5a78cab8783dd241c52c4fcda28fb111c443887dd6d0fe38385e8383c98b3",
    ],
)

rpm(
    name = "gettext-0__0.21-7.el9.aarch64",
    sha256 = "ba6583dd3960106d255266c1428d7b1f2383e75e14101a4f46e61053237c2920",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gettext-0.21-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ba6583dd3960106d255266c1428d7b1f2383e75e14101a4f46e61053237c2920",
    ],
)

rpm(
    name = "gettext-0__0.21-7.el9.x86_64",
    sha256 = "386905ddacb2614d519ec5dbaf038d40dbc44307b0edaa0bd3e6a5baa405a7b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gettext-0.21-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/386905ddacb2614d519ec5dbaf038d40dbc44307b0edaa0bd3e6a5baa405a7b8",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-7.el9.aarch64",
    sha256 = "3114d6de7dadd0dae0b520f776ef7da581fe39fde3880ec9dbedf58bafc6b1c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gettext-libs-0.21-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3114d6de7dadd0dae0b520f776ef7da581fe39fde3880ec9dbedf58bafc6b1c9",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-7.el9.x86_64",
    sha256 = "1388fca61334c67cac638edba2459b362cc401c8ff5ab8d7d5ca387b0ffc8786",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gettext-libs-0.21-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1388fca61334c67cac638edba2459b362cc401c8ff5ab8d7d5ca387b0ffc8786",
    ],
)

rpm(
    name = "glib2-0__2.68.4-5.el9.aarch64",
    sha256 = "fa9e25b82015b5d2023d9f71582e2dc0ed13ce7fc70c29ee49797713a88b46db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glib2-2.68.4-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fa9e25b82015b5d2023d9f71582e2dc0ed13ce7fc70c29ee49797713a88b46db",
    ],
)

rpm(
    name = "glib2-0__2.68.4-5.el9.x86_64",
    sha256 = "34bc8c6f001daa8dba60aee15956d7ac124e71bd7c5c99039245a4bf6e61a8f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/34bc8c6f001daa8dba60aee15956d7ac124e71bd7c5c99039245a4bf6e61a8f5",
    ],
)

rpm(
    name = "glibc-0__2.34-47.el9.aarch64",
    sha256 = "30ce099f2f0fea66b0fa996394941bcd664f2f9f7db809356f7831210151b080",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-47.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/30ce099f2f0fea66b0fa996394941bcd664f2f9f7db809356f7831210151b080",
    ],
)

rpm(
    name = "glibc-0__2.34-47.el9.x86_64",
    sha256 = "b25a2aebb1469137f5cbf9528b2279dc8028ac19271f87ce6e52196469c7acf5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b25a2aebb1469137f5cbf9528b2279dc8028ac19271f87ce6e52196469c7acf5",
    ],
)

rpm(
    name = "glibc-common-0__2.34-47.el9.aarch64",
    sha256 = "d5e0cd68eff3b3bf08b583f5bf8aa8624927ea1f4344e911bd0b5b92e68c4280",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-47.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d5e0cd68eff3b3bf08b583f5bf8aa8624927ea1f4344e911bd0b5b92e68c4280",
    ],
)

rpm(
    name = "glibc-common-0__2.34-47.el9.x86_64",
    sha256 = "ad41ad9bba2f0dd64d9bc59ce9efa640094e288a8336709e83c531ea5e0e8fb6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ad41ad9bba2f0dd64d9bc59ce9efa640094e288a8336709e83c531ea5e0e8fb6",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-47.el9.aarch64",
    sha256 = "336127c9bf1f1944647b823bb849d5019b6d6ec67975688d666c77d9e3d87482",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-47.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/336127c9bf1f1944647b823bb849d5019b6d6ec67975688d666c77d9e3d87482",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-47.el9.x86_64",
    sha256 = "f7395e8148135566bb10d4a7d64d396efea46eb9dab557d63537609e6277cf71",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7395e8148135566bb10d4a7d64d396efea46eb9dab557d63537609e6277cf71",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-47.el9.x86_64",
    sha256 = "a3884864fbcdc88641820b4d93ba60f9d8be4efd89dad3fc27889ebc0d9167b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a3884864fbcdc88641820b4d93ba60f9d8be4efd89dad3fc27889ebc0d9167b8",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-47.el9.aarch64",
    sha256 = "f9dc080198a47b4f4d4564de7721aa12d1ba416fa6e05bbf4c223f64b64f4b4a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-47.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f9dc080198a47b4f4d4564de7721aa12d1ba416fa6e05bbf4c223f64b64f4b4a",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-47.el9.x86_64",
    sha256 = "e8dd0542f1956e3433e12f7c16b3809031dac33ae6207ffa816140c40bb549f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e8dd0542f1956e3433e12f7c16b3809031dac33ae6207ffa816140c40bb549f3",
    ],
)

rpm(
    name = "glibc-static-0__2.34-47.el9.aarch64",
    sha256 = "6a1c504df6c86a2084664a029b8f47c7478b569ae9d8253309d55beb19017fed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-47.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a1c504df6c86a2084664a029b8f47c7478b569ae9d8253309d55beb19017fed",
    ],
)

rpm(
    name = "glibc-static-0__2.34-47.el9.x86_64",
    sha256 = "6592c7beb32729740bc5c6e5c5bca23d31840bbe6e8a2ed887e031651e039e0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6592c7beb32729740bc5c6e5c5bca23d31840bbe6e8a2ed887e031651e039e0a",
    ],
)

rpm(
    name = "gmp-1__6.2.0-10.el9.aarch64",
    sha256 = "1fe837ca20f20f8291a32c0f4673ea2560f94d75d25ab5131f6ae271694a4b44",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gmp-6.2.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1fe837ca20f20f8291a32c0f4673ea2560f94d75d25ab5131f6ae271694a4b44",
    ],
)

rpm(
    name = "gmp-1__6.2.0-10.el9.x86_64",
    sha256 = "1a6ededc80029ef258288ddbf24bcce7c6228647841416950c88e3f14b7258a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gmp-6.2.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1a6ededc80029ef258288ddbf24bcce7c6228647841416950c88e3f14b7258a2",
    ],
)

rpm(
    name = "gnupg2-0__2.3.3-2.el9.x86_64",
    sha256 = "d537e48c6947c6086d1af21b81b2619931b0ff708606d7545e388bbea05dcf32",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnupg2-2.3.3-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d537e48c6947c6086d1af21b81b2619931b0ff708606d7545e388bbea05dcf32",
    ],
)

rpm(
    name = "gnutls-0__3.7.6-12.el9.aarch64",
    sha256 = "ef571980c4864a3ec3fb1c3a662598fceb5aea77a6622d0fdacdab84c07f6660",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gnutls-3.7.6-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ef571980c4864a3ec3fb1c3a662598fceb5aea77a6622d0fdacdab84c07f6660",
    ],
)

rpm(
    name = "gnutls-0__3.7.6-12.el9.x86_64",
    sha256 = "9f0a12e96695045bf83425dcaa7a2840c7db9f2f5f1bf24f25f055d6119750cd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnutls-3.7.6-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f0a12e96695045bf83425dcaa7a2840c7db9f2f5f1bf24f25f055d6119750cd",
    ],
)

rpm(
    name = "gnutls-dane-0__3.7.6-12.el9.aarch64",
    sha256 = "2469b54282c4a9b4d6dd74c14e70f799f5ccedfd45c0ec7e102db14ecee334e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.7.6-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2469b54282c4a9b4d6dd74c14e70f799f5ccedfd45c0ec7e102db14ecee334e9",
    ],
)

rpm(
    name = "gnutls-dane-0__3.7.6-12.el9.x86_64",
    sha256 = "f92c0c5bb5b183adc2c6c59556b5ed90f11073caeb35873641f0eeacd649b9cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.7.6-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f92c0c5bb5b183adc2c6c59556b5ed90f11073caeb35873641f0eeacd649b9cb",
    ],
)

rpm(
    name = "gnutls-utils-0__3.7.6-12.el9.aarch64",
    sha256 = "fab5deb295e4734c1b649ef2c59a384303685e674e0c5b0537b7bf80f67f9238",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.7.6-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fab5deb295e4734c1b649ef2c59a384303685e674e0c5b0537b7bf80f67f9238",
    ],
)

rpm(
    name = "gnutls-utils-0__3.7.6-12.el9.x86_64",
    sha256 = "df67764cebd9055df1bfa083122e3decfa76c922c7ccd2891a95f43841f0b527",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.7.6-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df67764cebd9055df1bfa083122e3decfa76c922c7ccd2891a95f43841f0b527",
    ],
)

rpm(
    name = "grep-0__3.6-5.el9.aarch64",
    sha256 = "33bdf571a62cb8b7d659617e9278e46043aa936f8e963202750d19463a805f60",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/grep-3.6-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/33bdf571a62cb8b7d659617e9278e46043aa936f8e963202750d19463a805f60",
    ],
)

rpm(
    name = "grep-0__3.6-5.el9.x86_64",
    sha256 = "10a41b66b1fbd6eb055178e22c37199e5b49b4852e77c806f7af7211044a4a55",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/grep-3.6-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10a41b66b1fbd6eb055178e22c37199e5b49b4852e77c806f7af7211044a4a55",
    ],
)

rpm(
    name = "gssproxy-0__0.8.4-4.el9.x86_64",
    sha256 = "bc7b37a4bc3342ca7884f0166b4124d68b51b75ead9f8e996ddbd0125ab571d5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gssproxy-0.8.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc7b37a4bc3342ca7884f0166b4124d68b51b75ead9f8e996ddbd0125ab571d5",
    ],
)

rpm(
    name = "guestfs-tools-0__1.48.2-7.el9.x86_64",
    sha256 = "96d4308cdc6da7a85448f6353f0cbe0830c9faebcdb637b935e51005fffcc4b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.48.2-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/96d4308cdc6da7a85448f6353f0cbe0830c9faebcdb637b935e51005fffcc4b2",
    ],
)

rpm(
    name = "gzip-0__1.12-1.el9.aarch64",
    sha256 = "5a39a441dad01ccc8af601f1cca5bed46ac231fbdbe39ea3202bd54cf9390d81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gzip-1.12-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5a39a441dad01ccc8af601f1cca5bed46ac231fbdbe39ea3202bd54cf9390d81",
    ],
)

rpm(
    name = "gzip-0__1.12-1.el9.x86_64",
    sha256 = "e8d7783c666a58ab870246b04eb0ea22965123fe284697d2c0e1e6dbf10ea861",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gzip-1.12-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e8d7783c666a58ab870246b04eb0ea22965123fe284697d2c0e1e6dbf10ea861",
    ],
)

rpm(
    name = "hexedit-0__1.6-1.el9.x86_64",
    sha256 = "8c0781f044f9e45329cfc0f4c7d7acd65c9f779b34816c205279f977919e856f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/hexedit-1.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c0781f044f9e45329cfc0f4c7d7acd65c9f779b34816c205279f977919e856f",
    ],
)

rpm(
    name = "hivex-libs-0__1.3.21-3.el9.x86_64",
    sha256 = "3b0b567737f8a78e9264a07f935b25098f505d2b46653dba944919da85020ef7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/hivex-libs-1.3.21-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b0b567737f8a78e9264a07f935b25098f505d2b46653dba944919da85020ef7",
    ],
)

rpm(
    name = "iproute-0__5.18.0-1.el9.aarch64",
    sha256 = "c9d8c9099467bf1a8ef992606ed741d4468e44266ce6b419414a34b88e4aa213",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-5.18.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c9d8c9099467bf1a8ef992606ed741d4468e44266ce6b419414a34b88e4aa213",
    ],
)

rpm(
    name = "iproute-0__5.18.0-1.el9.x86_64",
    sha256 = "7396e9caf6a3b98de2fe82bad2d8b3607c1e0abf1dcbd9c86c8fb378d605e92a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-5.18.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7396e9caf6a3b98de2fe82bad2d8b3607c1e0abf1dcbd9c86c8fb378d605e92a",
    ],
)

rpm(
    name = "iproute-tc-0__5.18.0-1.el9.aarch64",
    sha256 = "455e0071f0021e7de822cb02c23ed4449c8a635ad4e711c8dd023ab0817e4d65",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-tc-5.18.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/455e0071f0021e7de822cb02c23ed4449c8a635ad4e711c8dd023ab0817e4d65",
    ],
)

rpm(
    name = "iproute-tc-0__5.18.0-1.el9.x86_64",
    sha256 = "de2e6ab190d0515cd0190c8341379efc7fdf631d464bddac1227ab45efa4693b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-tc-5.18.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de2e6ab190d0515cd0190c8341379efc7fdf631d464bddac1227ab45efa4693b",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.8-4.el9.aarch64",
    sha256 = "f418e2aad2d98b363f007f412865a7dd9ec3e39bd18a1008c2f4abe04a426423",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.8-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f418e2aad2d98b363f007f412865a7dd9ec3e39bd18a1008c2f4abe04a426423",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.8-4.el9.x86_64",
    sha256 = "cec6683679a617cbfd3f2b17b746ea3b53e068e41f761b0c5ec3524e3aa6862d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.8-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cec6683679a617cbfd3f2b17b746ea3b53e068e41f761b0c5ec3524e3aa6862d",
    ],
)

rpm(
    name = "iptables-nft-0__1.8.8-4.el9.aarch64",
    sha256 = "19ec852d0da591c94e09b88f0aad5da05c66aacccdaac936fec4e48c987482a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iptables-nft-1.8.8-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19ec852d0da591c94e09b88f0aad5da05c66aacccdaac936fec4e48c987482a8",
    ],
)

rpm(
    name = "iptables-nft-0__1.8.8-4.el9.x86_64",
    sha256 = "161ace77d8aa79c6a55a2c306bbf68a4b466501b49b5f322297c4797b1874573",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iptables-nft-1.8.8-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/161ace77d8aa79c6a55a2c306bbf68a4b466501b49b5f322297c4797b1874573",
    ],
)

rpm(
    name = "iputils-0__20210202-7.el9.aarch64",
    sha256 = "f9504344a47cbdb2710e3e39df729f26c35814795462fb5e3e4c85bb62cfdb15",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iputils-20210202-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f9504344a47cbdb2710e3e39df729f26c35814795462fb5e3e4c85bb62cfdb15",
    ],
)

rpm(
    name = "iputils-0__20210202-7.el9.x86_64",
    sha256 = "ec7784303ac30b70348b742a8338ee283350006983cdcbb44e24d05a54facc8b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iputils-20210202-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ec7784303ac30b70348b742a8338ee283350006983cdcbb44e24d05a54facc8b",
    ],
)

rpm(
    name = "ipxe-roms-qemu-0__20200823-9.git4bd064de.el9.x86_64",
    sha256 = "fa304f6cffa4a84a8aae1e0d2dd10606ffb51b88d9568b7da92ffd63acb14851",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/ipxe-roms-qemu-20200823-9.git4bd064de.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/fa304f6cffa4a84a8aae1e0d2dd10606ffb51b88d9568b7da92ffd63acb14851",
    ],
)

rpm(
    name = "jansson-0__2.14-1.el9.aarch64",
    sha256 = "23a8033dae909a6b87db199e04ecbc9798820b1b939e12d51733fed4554b9279",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/jansson-2.14-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/23a8033dae909a6b87db199e04ecbc9798820b1b939e12d51733fed4554b9279",
    ],
)

rpm(
    name = "jansson-0__2.14-1.el9.x86_64",
    sha256 = "c3fb9f8020f978f9b392709996e62e4ddb6cb19074635af3338487195b688f66",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/jansson-2.14-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c3fb9f8020f978f9b392709996e62e4ddb6cb19074635af3338487195b688f66",
    ],
)

rpm(
    name = "json-c-0__0.14-11.el9.aarch64",
    sha256 = "65a68a23f33540b4d7cd2d9227a63d7eda1a7ab7cdd52457fee9662c06731cfa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/json-c-0.14-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/65a68a23f33540b4d7cd2d9227a63d7eda1a7ab7cdd52457fee9662c06731cfa",
    ],
)

rpm(
    name = "json-c-0__0.14-11.el9.x86_64",
    sha256 = "1a75404c6bc8c1369914077dc99480e73bf13a40f15fd1cd8afc792b8600adf8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/json-c-0.14-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1a75404c6bc8c1369914077dc99480e73bf13a40f15fd1cd8afc792b8600adf8",
    ],
)

rpm(
    name = "json-glib-0__1.6.6-1.el9.aarch64",
    sha256 = "04a7348a546a972f275a4de34373ad7a937a5a93f4c868dffa47daa31a226243",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/json-glib-1.6.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/04a7348a546a972f275a4de34373ad7a937a5a93f4c868dffa47daa31a226243",
    ],
)

rpm(
    name = "json-glib-0__1.6.6-1.el9.x86_64",
    sha256 = "d850cb45d31fe84cb50cb1fa26eb5418633aae1f0dcab8b7ebadd3bd3e340956",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/json-glib-1.6.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d850cb45d31fe84cb50cb1fa26eb5418633aae1f0dcab8b7ebadd3bd3e340956",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-176.el9.aarch64",
    sha256 = "865bbd061b50c1d083f42ad3559f179c4aa5a1fd4b65a906b9ec18bfe246045f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/kernel-headers-5.14.0-176.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/865bbd061b50c1d083f42ad3559f179c4aa5a1fd4b65a906b9ec18bfe246045f",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-176.el9.x86_64",
    sha256 = "18d70f8a3396b8b7386fd197605c2a50c1a280bd284ef5d65e509223c43357d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/kernel-headers-5.14.0-176.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/18d70f8a3396b8b7386fd197605c2a50c1a280bd284ef5d65e509223c43357d6",
    ],
)

rpm(
    name = "keyutils-0__1.6.1-4.el9.x86_64",
    sha256 = "a507c5e173ac4b9af08ebb0a8cecfdfac80b3ee18bd23171d09872b27385a108",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/keyutils-1.6.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a507c5e173ac4b9af08ebb0a8cecfdfac80b3ee18bd23171d09872b27385a108",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.1-4.el9.aarch64",
    sha256 = "bb0cc6cde590e58d76610c5d0d0811f20603758f63a604f10289a170bcde4e0f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/keyutils-libs-1.6.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bb0cc6cde590e58d76610c5d0d0811f20603758f63a604f10289a170bcde4e0f",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.1-4.el9.x86_64",
    sha256 = "56c94b7b30b5e5b1411b0053fd62edf408d59fc2260d7d31883a97a667342d6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/keyutils-libs-1.6.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56c94b7b30b5e5b1411b0053fd62edf408d59fc2260d7d31883a97a667342d6f",
    ],
)

rpm(
    name = "kmod-0__28-7.el9.aarch64",
    sha256 = "9a3293fa629de83a0ce0e6326122a8ec8d66132f43dda4abe135a61ceeba739b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/kmod-28-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9a3293fa629de83a0ce0e6326122a8ec8d66132f43dda4abe135a61ceeba739b",
    ],
)

rpm(
    name = "kmod-0__28-7.el9.x86_64",
    sha256 = "3d4bc7935959a109a10020d0d19a5e059719ae4c99c5f32d3020ff6da47d53ea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-28-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3d4bc7935959a109a10020d0d19a5e059719ae4c99c5f32d3020ff6da47d53ea",
    ],
)

rpm(
    name = "kmod-libs-0__28-7.el9.aarch64",
    sha256 = "e7be371f66f08c54bc464cf376fce0f09ba07d1e10843c7f9bc35abbb3c38085",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/kmod-libs-28-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e7be371f66f08c54bc464cf376fce0f09ba07d1e10843c7f9bc35abbb3c38085",
    ],
)

rpm(
    name = "kmod-libs-0__28-7.el9.x86_64",
    sha256 = "0727ff3131223446158aaec88cbf8f894a9e3592e73f231a1802629518eeb64b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-libs-28-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0727ff3131223446158aaec88cbf8f894a9e3592e73f231a1802629518eeb64b",
    ],
)

rpm(
    name = "krb5-libs-0__1.19.1-22.el9.aarch64",
    sha256 = "24b7ef009af4066c4477edc2e0b145517d8d4469d050fa922d4cfb68d728fde2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.19.1-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/24b7ef009af4066c4477edc2e0b145517d8d4469d050fa922d4cfb68d728fde2",
    ],
)

rpm(
    name = "krb5-libs-0__1.19.1-22.el9.x86_64",
    sha256 = "81195fcb28dca19447c75d1eff6b62c0b4f849e6b492c992e890bb65aca55734",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.19.1-22.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/81195fcb28dca19447c75d1eff6b62c0b4f849e6b492c992e890bb65aca55734",
    ],
)

rpm(
    name = "less-0__590-1.el9.x86_64",
    sha256 = "75ec2628be5ebe149a79b00f2160b9653297651d5ea291022e053719d2ff07f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/less-590-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/75ec2628be5ebe149a79b00f2160b9653297651d5ea291022e053719d2ff07f5",
    ],
)

rpm(
    name = "libacl-0__2.3.1-3.el9.aarch64",
    sha256 = "4975593414dfa1e822cd108e988d18453c2ff036b03e4cdbf38db0afb45e0c92",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libacl-2.3.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4975593414dfa1e822cd108e988d18453c2ff036b03e4cdbf38db0afb45e0c92",
    ],
)

rpm(
    name = "libacl-0__2.3.1-3.el9.x86_64",
    sha256 = "fd829e9a03f6d321313002d6fcb37ee0434f548aa75fcd3ecdbdd891115de6a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libacl-2.3.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd829e9a03f6d321313002d6fcb37ee0434f548aa75fcd3ecdbdd891115de6a7",
    ],
)

rpm(
    name = "libaio-0__0.3.111-13.el9.aarch64",
    sha256 = "1730d732818fa2471b5cd461175ceda18e909410db8a32185d8db2aa7461130c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libaio-0.3.111-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1730d732818fa2471b5cd461175ceda18e909410db8a32185d8db2aa7461130c",
    ],
)

rpm(
    name = "libaio-0__0.3.111-13.el9.x86_64",
    sha256 = "7d9d4d37e86ba94bb941e2dad40c90a157aaa0602f02f3f90e76086515f439be",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libaio-0.3.111-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d9d4d37e86ba94bb941e2dad40c90a157aaa0602f02f3f90e76086515f439be",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-3.el9.aarch64",
    sha256 = "273867588e3fb645b630bd00584394db07f02418bd2792cc479a8f41022f6b90",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libarchive-3.5.3-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/273867588e3fb645b630bd00584394db07f02418bd2792cc479a8f41022f6b90",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-3.el9.x86_64",
    sha256 = "82806a2dd17fc65512636d6437f07f5299e05c7f8279d9bb811739efb7d2b972",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libarchive-3.5.3-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82806a2dd17fc65512636d6437f07f5299e05c7f8279d9bb811739efb7d2b972",
    ],
)

rpm(
    name = "libasan-0__11.3.1-2.1.el9.aarch64",
    sha256 = "e41eeaa3f5b7b2b0249e742ed8e8db53b13b949e9e001752fcd09c8880e43791",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e41eeaa3f5b7b2b0249e742ed8e8db53b13b949e9e001752fcd09c8880e43791",
    ],
)

rpm(
    name = "libassuan-0__2.5.5-3.el9.x86_64",
    sha256 = "3f7ab80145768029619033b31406a9aeef8c8f0d42a0c94ad464d8a3405e12b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libassuan-2.5.5-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f7ab80145768029619033b31406a9aeef8c8f0d42a0c94ad464d8a3405e12b0",
    ],
)

rpm(
    name = "libatomic-0__11.3.1-2.1.el9.aarch64",
    sha256 = "d77ccddef8c520c356b592ffe115fa1b099f51c7f771e99db356d6f6a6ae7071",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d77ccddef8c520c356b592ffe115fa1b099f51c7f771e99db356d6f6a6ae7071",
    ],
)

rpm(
    name = "libattr-0__2.5.1-3.el9.aarch64",
    sha256 = "a0101ccea66aef376f4067c1002ebdfb5dbeeecd334047459b3855eff17a6fda",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libattr-2.5.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a0101ccea66aef376f4067c1002ebdfb5dbeeecd334047459b3855eff17a6fda",
    ],
)

rpm(
    name = "libattr-0__2.5.1-3.el9.x86_64",
    sha256 = "d4db095a015e84065f27a642ee7829cd1690041ba8c51501f908cc34760c9409",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libattr-2.5.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d4db095a015e84065f27a642ee7829cd1690041ba8c51501f908cc34760c9409",
    ],
)

rpm(
    name = "libbasicobjects-0__0.1.1-53.el9.x86_64",
    sha256 = "14ce3dd811d88dddc4009c12094cd0e52bbcabe0f2463bdfcc4124c620fb13d5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbasicobjects-0.1.1-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/14ce3dd811d88dddc4009c12094cd0e52bbcabe0f2463bdfcc4124c620fb13d5",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-9.el9.aarch64",
    sha256 = "30f9f288185a85f20447994c0d2dba80665bf5ccce089d8c16fd8a9c8495c6a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libblkid-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/30f9f288185a85f20447994c0d2dba80665bf5ccce089d8c16fd8a9c8495c6a2",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-9.el9.x86_64",
    sha256 = "cb09fe87839c17ae2726459d4d5f3e2a7396071b03cda70201a6d1e9db5e7504",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libblkid-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cb09fe87839c17ae2726459d4d5f3e2a7396071b03cda70201a6d1e9db5e7504",
    ],
)

rpm(
    name = "libbpf-2__0.6.0-1.el9.aarch64",
    sha256 = "d9640d83e2d4b39800dad59e05f9f5fe51da03a6d3e89d6dbb666f5374635b44",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libbpf-0.6.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d9640d83e2d4b39800dad59e05f9f5fe51da03a6d3e89d6dbb666f5374635b44",
    ],
)

rpm(
    name = "libbpf-2__0.6.0-1.el9.x86_64",
    sha256 = "b31c8fdfaa2d39e71fcf925c6dd5b7f9beb0f4fc425edecdc15ef445ea1bc587",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-0.6.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b31c8fdfaa2d39e71fcf925c6dd5b7f9beb0f4fc425edecdc15ef445ea1bc587",
    ],
)

rpm(
    name = "libburn-0__1.5.4-4.el9.aarch64",
    sha256 = "9a38b538faba01eb3aa59abdbfa05f37705ed6e21cea74fa8cca97f4fe7dad20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libburn-1.5.4-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9a38b538faba01eb3aa59abdbfa05f37705ed6e21cea74fa8cca97f4fe7dad20",
    ],
)

rpm(
    name = "libburn-0__1.5.4-4.el9.x86_64",
    sha256 = "c6bbbaa269d37d0cd29bdd329bf91096112cff6aa623112d6b3c9b3bb365ebaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libburn-1.5.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c6bbbaa269d37d0cd29bdd329bf91096112cff6aa623112d6b3c9b3bb365ebaa",
    ],
)

rpm(
    name = "libcap-0__2.48-8.el9.aarch64",
    sha256 = "881d4e7729633ce71b1a6bab3a84c1f79d5e7c49ef3ffdc1bc703cdd7ae3cd81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcap-2.48-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/881d4e7729633ce71b1a6bab3a84c1f79d5e7c49ef3ffdc1bc703cdd7ae3cd81",
    ],
)

rpm(
    name = "libcap-0__2.48-8.el9.x86_64",
    sha256 = "c41f91075ee8ca480c2631a485bcc74876b9317b4dc9bd66566da32313621bd7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcap-2.48-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c41f91075ee8ca480c2631a485bcc74876b9317b4dc9bd66566da32313621bd7",
    ],
)

rpm(
    name = "libcap-ng-0__0.8.2-7.el9.aarch64",
    sha256 = "1dfa7208abe1af5522523cabdabb73783ed1df4424dc8846eab8a570d010deaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcap-ng-0.8.2-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1dfa7208abe1af5522523cabdabb73783ed1df4424dc8846eab8a570d010deaa",
    ],
)

rpm(
    name = "libcap-ng-0__0.8.2-7.el9.x86_64",
    sha256 = "62429b788acfb40dbc9da9951690c11e907e230879c790d139f73d0e85dd76f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcap-ng-0.8.2-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/62429b788acfb40dbc9da9951690c11e907e230879c790d139f73d0e85dd76f4",
    ],
)

rpm(
    name = "libcollection-0__0.7.0-53.el9.x86_64",
    sha256 = "07c24fc00d1fd088a7f2b16b6cf70b781aed6ed682f11c4bce3ab76cf56707fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcollection-0.7.0-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/07c24fc00d1fd088a7f2b16b6cf70b781aed6ed682f11c4bce3ab76cf56707fd",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-3.el9.aarch64",
    sha256 = "a735b91094a13612830db66fd2021c9ec86c92697e526068e8b3919111cc2ba8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcom_err-1.46.5-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a735b91094a13612830db66fd2021c9ec86c92697e526068e8b3919111cc2ba8",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-3.el9.x86_64",
    sha256 = "ef9db384c8fbfc0b8676aec1896070dc308cfc0c7b515ebbe556e0fea68318d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcom_err-1.46.5-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ef9db384c8fbfc0b8676aec1896070dc308cfc0c7b515ebbe556e0fea68318d0",
    ],
)

rpm(
    name = "libconfig-0__1.7.2-9.el9.x86_64",
    sha256 = "e0d4d2cf8215404750c3975a19e2b7cd2c9e9e1e5c539d3fd93532775fd2ed16",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libconfig-1.7.2-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e0d4d2cf8215404750c3975a19e2b7cd2c9e9e1e5c539d3fd93532775fd2ed16",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-20.el9.aarch64",
    sha256 = "5d3371d2150d67f7c7c4c1e329d979ccad20b7e4066b7c6f59b0a78ea9f040e2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.76.1-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5d3371d2150d67f7c7c4c1e329d979ccad20b7e4066b7c6f59b0a78ea9f040e2",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-20.el9.x86_64",
    sha256 = "747e5ee33ad88d676d38e05cd04d86fc68264c505e4766e73f92ba3a7ce8e70f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/747e5ee33ad88d676d38e05cd04d86fc68264c505e4766e73f92ba3a7ce8e70f",
    ],
)

rpm(
    name = "libdb-0__5.3.28-53.el9.aarch64",
    sha256 = "65a5743728c6c331dd8aadc9b51f261f90ffa47ffd0cfb448da8bdf28af6dd77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libdb-5.3.28-53.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/65a5743728c6c331dd8aadc9b51f261f90ffa47ffd0cfb448da8bdf28af6dd77",
    ],
)

rpm(
    name = "libdb-0__5.3.28-53.el9.x86_64",
    sha256 = "3a44d15d695944bde4e7290800b815f98bfd9cd6f6f868cec3e8991606f556d5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libdb-5.3.28-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a44d15d695944bde4e7290800b815f98bfd9cd6f6f868cec3e8991606f556d5",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-2.el9.aarch64",
    sha256 = "082dff130121fcdb7cb3fd432de482075b5003e0d95ff4ab6d8ba02404b69d6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libeconf-0.4.1-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/082dff130121fcdb7cb3fd432de482075b5003e0d95ff4ab6d8ba02404b69d6b",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-2.el9.x86_64",
    sha256 = "1d6fe169e74daff38ad5b0d6424c4d1b14545d5974c39e4421d20838a68f5892",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libeconf-0.4.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1d6fe169e74daff38ad5b0d6424c4d1b14545d5974c39e4421d20838a68f5892",
    ],
)

rpm(
    name = "libev-0__4.33-5.el9.x86_64",
    sha256 = "9ee87c7d34e341bc7b136125ef5f1429a0b5fadaffcf888ab896b2c62c2b4e8d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libev-4.33-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ee87c7d34e341bc7b136125ef5f1429a0b5fadaffcf888ab896b2c62c2b4e8d",
    ],
)

rpm(
    name = "libevent-0__2.1.12-6.el9.aarch64",
    sha256 = "5ff00c047204190e3b2ee19f81d644c8f82ea7e8d1f36fdaaf6483f0fa3b3339",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libevent-2.1.12-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5ff00c047204190e3b2ee19f81d644c8f82ea7e8d1f36fdaaf6483f0fa3b3339",
    ],
)

rpm(
    name = "libevent-0__2.1.12-6.el9.x86_64",
    sha256 = "82179f6f214ddf523e143c16c3474ccf8832551c6305faf89edfbd83b3424d48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libevent-2.1.12-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82179f6f214ddf523e143c16c3474ccf8832551c6305faf89edfbd83b3424d48",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-9.el9.aarch64",
    sha256 = "dad6efc735b4078ec086bbd2c4981b2be5b0686b85207c282b25348c97a64306",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libfdisk-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dad6efc735b4078ec086bbd2c4981b2be5b0686b85207c282b25348c97a64306",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-9.el9.x86_64",
    sha256 = "1e75c0e916ce41ca3fc04322f414aa295ccc2cb4ed9cc4f512d656f8726230ab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e75c0e916ce41ca3fc04322f414aa295ccc2cb4ed9cc4f512d656f8726230ab",
    ],
)

rpm(
    name = "libfdt-0__1.6.0-7.el9.aarch64",
    sha256 = "19cd82e2bbdd6254169b267e645564acd0911e02fafaf6e3ad9893cd1f9d3d67",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libfdt-1.6.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19cd82e2bbdd6254169b267e645564acd0911e02fafaf6e3ad9893cd1f9d3d67",
    ],
)

rpm(
    name = "libfdt-0__1.6.0-7.el9.x86_64",
    sha256 = "a071b9d517505a2ff8642de7ac094faa689b96122c0a3e9ce86933aa1dea525f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libfdt-1.6.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a071b9d517505a2ff8642de7ac094faa689b96122c0a3e9ce86933aa1dea525f",
    ],
)

rpm(
    name = "libffi-0__3.4.2-7.el9.aarch64",
    sha256 = "6a42002c0b63a3c4d1e8da5cdf4822f442a7b458d80e69673715715d38ea977d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libffi-3.4.2-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a42002c0b63a3c4d1e8da5cdf4822f442a7b458d80e69673715715d38ea977d",
    ],
)

rpm(
    name = "libffi-0__3.4.2-7.el9.x86_64",
    sha256 = "f0ac4b6454d4018833dd10e3f437d8271c7c6a628d99b37e75b83af890b86bc4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libffi-3.4.2-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f0ac4b6454d4018833dd10e3f437d8271c7c6a628d99b37e75b83af890b86bc4",
    ],
)

rpm(
    name = "libgcc-0__11.3.1-2.1.el9.aarch64",
    sha256 = "9a99bd50285ee01446e6580eba3adbd8a402262d1d72e8474ac5b8f06cdc874f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9a99bd50285ee01446e6580eba3adbd8a402262d1d72e8474ac5b8f06cdc874f",
    ],
)

rpm(
    name = "libgcc-0__11.3.1-2.1.el9.x86_64",
    sha256 = "61fe577de2c8f93bf6d49973da196efedf7733795194af05c95e60892259f076",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.3.1-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61fe577de2c8f93bf6d49973da196efedf7733795194af05c95e60892259f076",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-7.el9.aarch64",
    sha256 = "e03e7cca559ba37941762911ed24ed2c4e15da5fe3e73fb617ce060ecb1e5c58",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.10.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e03e7cca559ba37941762911ed24ed2c4e15da5fe3e73fb617ce060ecb1e5c58",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-7.el9.x86_64",
    sha256 = "1a881fc225c44aefb8a5373a400c4387e00f2e01481b2dbe89c9779a419510f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.10.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1a881fc225c44aefb8a5373a400c4387e00f2e01481b2dbe89c9779a419510f3",
    ],
)

rpm(
    name = "libgomp-0__11.3.1-2.1.el9.aarch64",
    sha256 = "7a8491ef16677ed09d2ebef5f51c158d04bc6b53985b813832defafb24b42215",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7a8491ef16677ed09d2ebef5f51c158d04bc6b53985b813832defafb24b42215",
    ],
)

rpm(
    name = "libgomp-0__11.3.1-2.1.el9.x86_64",
    sha256 = "20051ac14a8977148a1663afffac421a2b6655d62f69e98bc0d7638ed261f77c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.3.1-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20051ac14a8977148a1663afffac421a2b6655d62f69e98bc0d7638ed261f77c",
    ],
)

rpm(
    name = "libgpg-error-0__1.42-5.el9.aarch64",
    sha256 = "ffeb04823b5317c7e016542c8ecc5180c7824f8b59a180f2434fd096a34a9105",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgpg-error-1.42-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ffeb04823b5317c7e016542c8ecc5180c7824f8b59a180f2434fd096a34a9105",
    ],
)

rpm(
    name = "libgpg-error-0__1.42-5.el9.x86_64",
    sha256 = "a1883804c376f737109f4dff06077d1912b90150a732d11be7bc5b3b67e512fe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgpg-error-1.42-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a1883804c376f737109f4dff06077d1912b90150a732d11be7bc5b3b67e512fe",
    ],
)

rpm(
    name = "libguestfs-1__1.48.4-2.el9.x86_64",
    sha256 = "210ed16cf83b9c9b224698f9fb9c2b29797da3076f70aface2ee515a9b875178",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libguestfs-1.48.4-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/210ed16cf83b9c9b224698f9fb9c2b29797da3076f70aface2ee515a9b875178",
    ],
)

rpm(
    name = "libibverbs-0__41.0-3.el9.aarch64",
    sha256 = "118839cf6dfbd3dfdb30fa8838580f8c32291ca9f918dff7f4df92a5da4bc663",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libibverbs-41.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/118839cf6dfbd3dfdb30fa8838580f8c32291ca9f918dff7f4df92a5da4bc663",
    ],
)

rpm(
    name = "libibverbs-0__41.0-3.el9.x86_64",
    sha256 = "b7b3673aa94b178533d0934bf9b30cc28caa071646add99171baef5a5882d92a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libibverbs-41.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7b3673aa94b178533d0934bf9b30cc28caa071646add99171baef5a5882d92a",
    ],
)

rpm(
    name = "libidn2-0__2.3.0-7.el9.aarch64",
    sha256 = "6ed96112059449aa37b99d4d4e3b5d089c34afefbd9b618691bed8c206c4d441",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libidn2-2.3.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6ed96112059449aa37b99d4d4e3b5d089c34afefbd9b618691bed8c206c4d441",
    ],
)

rpm(
    name = "libidn2-0__2.3.0-7.el9.x86_64",
    sha256 = "f7fa1ad2fcd86beea5d4d965994c21dc98f47871faff14f73940190c754ab244",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libidn2-2.3.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7fa1ad2fcd86beea5d4d965994c21dc98f47871faff14f73940190c754ab244",
    ],
)

rpm(
    name = "libini_config-0__1.3.1-53.el9.x86_64",
    sha256 = "fb7dbaeb7c172663cab3029c4efaf80230bcba4abf1604cc6cc00993b5d9659e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libini_config-1.3.1-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb7dbaeb7c172663cab3029c4efaf80230bcba4abf1604cc6cc00993b5d9659e",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-4.el9.aarch64",
    sha256 = "73b53499b7cd070a899a202385ca3da1ae37e7bb65123767e6d74dbb432f7a08",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libisoburn-1.5.4-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/73b53499b7cd070a899a202385ca3da1ae37e7bb65123767e6d74dbb432f7a08",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-4.el9.x86_64",
    sha256 = "922f3d45899dfdcebf4cc9b5b82f31be240f76098ffbdbcb3291b4a77dcbbab7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libisoburn-1.5.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/922f3d45899dfdcebf4cc9b5b82f31be240f76098ffbdbcb3291b4a77dcbbab7",
    ],
)

rpm(
    name = "libisofs-0__1.5.4-4.el9.aarch64",
    sha256 = "0f4c8376add266f01328ea001c580ef9258c0ce39c26906226871c934a159e88",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libisofs-1.5.4-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0f4c8376add266f01328ea001c580ef9258c0ce39c26906226871c934a159e88",
    ],
)

rpm(
    name = "libisofs-0__1.5.4-4.el9.x86_64",
    sha256 = "78abca0dc6134189106ff550986cc059dc0edea129e572a742d2cc0b934c2d13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libisofs-1.5.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/78abca0dc6134189106ff550986cc059dc0edea129e572a742d2cc0b934c2d13",
    ],
)

rpm(
    name = "libksba-0__1.5.1-4.el9.x86_64",
    sha256 = "d8aab5c91906bbccb3f45eb873004bbc7dc58d5ca8d338778c444fb298fff30a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libksba-1.5.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d8aab5c91906bbccb3f45eb873004bbc7dc58d5ca8d338778c444fb298fff30a",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-15.el9.aarch64",
    sha256 = "a3e80b22d57f0e2843e37eee0440a9bae92e4a0cbe75b13520be7616afd70e78",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmnl-1.0.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a3e80b22d57f0e2843e37eee0440a9bae92e4a0cbe75b13520be7616afd70e78",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-15.el9.x86_64",
    sha256 = "a70fdda85cd771ef5bf5b17c2996e4ff4d21c2e5b1eece1764a87f12e720ab68",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmnl-1.0.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a70fdda85cd771ef5bf5b17c2996e4ff4d21c2e5b1eece1764a87f12e720ab68",
    ],
)

rpm(
    name = "libmount-0__2.37.4-9.el9.aarch64",
    sha256 = "36143d654f53c13d409a2be19970465dd61f4bf294f808a7dc9954e7a4414272",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmount-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/36143d654f53c13d409a2be19970465dd61f4bf294f808a7dc9954e7a4414272",
    ],
)

rpm(
    name = "libmount-0__2.37.4-9.el9.x86_64",
    sha256 = "10fefd21b2d0e3b4c48e87fc29303eb493589e68d4b5edccd43ced8154904874",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmount-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10fefd21b2d0e3b4c48e87fc29303eb493589e68d4b5edccd43ced8154904874",
    ],
)

rpm(
    name = "libmpc-0__1.2.1-4.el9.aarch64",
    sha256 = "489bd89037b1a77d696e391315c740f185e6447aacdb1d7fe84b411491c34b88",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libmpc-1.2.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/489bd89037b1a77d696e391315c740f185e6447aacdb1d7fe84b411491c34b88",
    ],
)

rpm(
    name = "libmpc-0__1.2.1-4.el9.x86_64",
    sha256 = "207e758fadd4779cb11b91a78446f098d0a95b782f30a24c0e998fe08e2561df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libmpc-1.2.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/207e758fadd4779cb11b91a78446f098d0a95b782f30a24c0e998fe08e2561df",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.8-4.el9.aarch64",
    sha256 = "ccdb8b18ad62387cc6ddc6fc7d61bbbcf5148cfef8843713cc1843fab523c93f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnetfilter_conntrack-1.0.8-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ccdb8b18ad62387cc6ddc6fc7d61bbbcf5148cfef8843713cc1843fab523c93f",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.8-4.el9.x86_64",
    sha256 = "8073d0ec79490fc14a1dc0fff813520e339ce89a6ebb53897c47d22483bf78e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnetfilter_conntrack-1.0.8-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8073d0ec79490fc14a1dc0fff813520e339ce89a6ebb53897c47d22483bf78e3",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-21.el9.aarch64",
    sha256 = "682c4cca565ce483ff0749dbb39b154bc080ac531c418d05890e454114c11821",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnfnetlink-1.0.1-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/682c4cca565ce483ff0749dbb39b154bc080ac531c418d05890e454114c11821",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-21.el9.x86_64",
    sha256 = "64f54f412cc0ee6fe82be7557f471a06f6bf1f5bba1d6fe0ad1879e5a62d7c95",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.1-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/64f54f412cc0ee6fe82be7557f471a06f6bf1f5bba1d6fe0ad1879e5a62d7c95",
    ],
)

rpm(
    name = "libnfsidmap-1__2.5.4-15.el9.x86_64",
    sha256 = "a5d3be4903e8afba326c5b6b28ac51c8ab58d47894ce0c1f8f6ebdf0bb90605d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfsidmap-2.5.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a5d3be4903e8afba326c5b6b28ac51c8ab58d47894ce0c1f8f6ebdf0bb90605d",
    ],
)

rpm(
    name = "libnftnl-0__1.2.2-1.el9.aarch64",
    sha256 = "6e2dac1414db86b13f0efbca18bd0128a122ba2b814faed1bce309200304cc86",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnftnl-1.2.2-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6e2dac1414db86b13f0efbca18bd0128a122ba2b814faed1bce309200304cc86",
    ],
)

rpm(
    name = "libnftnl-0__1.2.2-1.el9.x86_64",
    sha256 = "fd75863a6dd1be0e7f1b7eed3e5f13a0efead33ba9bb05b0f8430574aa804783",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnftnl-1.2.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd75863a6dd1be0e7f1b7eed3e5f13a0efead33ba9bb05b0f8430574aa804783",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-5.el9.aarch64",
    sha256 = "702abf0c5b1574b828132e4dbea17ad7099034db18f47fd1ac84b4d9534dcfea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.43.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/702abf0c5b1574b828132e4dbea17ad7099034db18f47fd1ac84b4d9534dcfea",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-5.el9.x86_64",
    sha256 = "58c5d589ee370951b98e908ac05a5a6154d52dbb8cf2067583ccdd10cdf099bf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.43.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/58c5d589ee370951b98e908ac05a5a6154d52dbb8cf2067583ccdd10cdf099bf",
    ],
)

rpm(
    name = "libnl3-0__3.7.0-1.el9.aarch64",
    sha256 = "5f8ede2ff552132a369b43e7babfd5e08e0dc46b5c659a665f188dc497cb0415",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnl3-3.7.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5f8ede2ff552132a369b43e7babfd5e08e0dc46b5c659a665f188dc497cb0415",
    ],
)

rpm(
    name = "libnl3-0__3.7.0-1.el9.x86_64",
    sha256 = "8abf9bf3f62df66aeed157fc9f9494a2ea792eb11eb221caa17ce7f97330a2f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnl3-3.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8abf9bf3f62df66aeed157fc9f9494a2ea792eb11eb221caa17ce7f97330a2f3",
    ],
)

rpm(
    name = "libpath_utils-0__0.2.1-53.el9.x86_64",
    sha256 = "0a2519647ef22df7c975fa2851da713e67361ff33f2bff05f91cb588b2722772",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpath_utils-0.2.1-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0a2519647ef22df7c975fa2851da713e67361ff33f2bff05f91cb588b2722772",
    ],
)

rpm(
    name = "libpcap-14__1.10.0-4.el9.aarch64",
    sha256 = "c1827185bde78c34817a75c79522963c76cd07585eeeb6961e58c6ddadc69333",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libpcap-1.10.0-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c1827185bde78c34817a75c79522963c76cd07585eeeb6961e58c6ddadc69333",
    ],
)

rpm(
    name = "libpcap-14__1.10.0-4.el9.x86_64",
    sha256 = "c76c9887f6b9d218300b24f1adee1b0d9104d25152df3fcd005002d12e12399e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpcap-1.10.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c76c9887f6b9d218300b24f1adee1b0d9104d25152df3fcd005002d12e12399e",
    ],
)

rpm(
    name = "libpkgconf-0__1.7.3-10.el9.aarch64",
    sha256 = "ad86227404ab0df04f1b98f74921a77c4068251da74067d3633cc1c43fee4a9b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libpkgconf-1.7.3-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad86227404ab0df04f1b98f74921a77c4068251da74067d3633cc1c43fee4a9b",
    ],
)

rpm(
    name = "libpkgconf-0__1.7.3-10.el9.x86_64",
    sha256 = "2dc8b201f4e24ca65fe6389fec8901eb84d48519cc44a6b0e474d7859370f389",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpkgconf-1.7.3-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2dc8b201f4e24ca65fe6389fec8901eb84d48519cc44a6b0e474d7859370f389",
    ],
)

rpm(
    name = "libpmem-0__1.10.1-2.el9.x86_64",
    sha256 = "627964323b9a5a93ea8175d5f1b070d0019243d5d1e048c1abed28059ecf8525",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libpmem-1.10.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/627964323b9a5a93ea8175d5f1b070d0019243d5d1e048c1abed28059ecf8525",
    ],
)

rpm(
    name = "libpng-2__1.6.37-12.el9.aarch64",
    sha256 = "99f9eca159e41e315b9fe48ec6c6d1d7a944bd5d8fc0b308aba779a6608b3777",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libpng-1.6.37-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/99f9eca159e41e315b9fe48ec6c6d1d7a944bd5d8fc0b308aba779a6608b3777",
    ],
)

rpm(
    name = "libpng-2__1.6.37-12.el9.x86_64",
    sha256 = "b3f3a689918dc50a9bc41c33abf1a36bdb8e4a707daac77a91e0814407b07ae3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpng-1.6.37-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3f3a689918dc50a9bc41c33abf1a36bdb8e4a707daac77a91e0814407b07ae3",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-8.el9.aarch64",
    sha256 = "3c22a268ce022cb4722aa2d35a95c1174778f424fbf29e98990801651d468aeb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libpwquality-1.4.4-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3c22a268ce022cb4722aa2d35a95c1174778f424fbf29e98990801651d468aeb",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-8.el9.x86_64",
    sha256 = "93f00e5efac1e3f1ecbc0d6a4c068772cb12912cd20c9ea58716d6c0cd004886",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.4-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/93f00e5efac1e3f1ecbc0d6a4c068772cb12912cd20c9ea58716d6c0cd004886",
    ],
)

rpm(
    name = "librdmacm-0__41.0-3.el9.aarch64",
    sha256 = "7cbe23a203af600bf38bb16884caa6cec670afb0801c6bd8a25b08a8c96f9330",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/librdmacm-41.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7cbe23a203af600bf38bb16884caa6cec670afb0801c6bd8a25b08a8c96f9330",
    ],
)

rpm(
    name = "librdmacm-0__41.0-3.el9.x86_64",
    sha256 = "62661a80fc924f55f81a0746cd428668e3d00103550c9d67aca953b5eb9eb33f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/librdmacm-41.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/62661a80fc924f55f81a0746cd428668e3d00103550c9d67aca953b5eb9eb33f",
    ],
)

rpm(
    name = "libref_array-0__0.1.5-53.el9.x86_64",
    sha256 = "7a7eaf030a25e866148daa6b38ac6f49afeba63b66f11040cc7b5b5522977d1e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libref_array-0.1.5-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7a7eaf030a25e866148daa6b38ac6f49afeba63b66f11040cc7b5b5522977d1e",
    ],
)

rpm(
    name = "libseccomp-0__2.5.2-2.el9.aarch64",
    sha256 = "ee31abd3d1325b05c5ba336158ba3b235a718a99ad5cec5e6ab498ca99b688b5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libseccomp-2.5.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ee31abd3d1325b05c5ba336158ba3b235a718a99ad5cec5e6ab498ca99b688b5",
    ],
)

rpm(
    name = "libseccomp-0__2.5.2-2.el9.x86_64",
    sha256 = "d5c1c4473ebf5fd9c605eb866118d7428cdec9b188db18e45545801cc2a689c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libseccomp-2.5.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5c1c4473ebf5fd9c605eb866118d7428cdec9b188db18e45545801cc2a689c3",
    ],
)

rpm(
    name = "libselinux-0__3.4-3.el9.aarch64",
    sha256 = "bf2e3f00871a2d969a9cac61855c224fdcf127284f890a25564872d97b1043e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-3.4-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bf2e3f00871a2d969a9cac61855c224fdcf127284f890a25564872d97b1043e9",
    ],
)

rpm(
    name = "libselinux-0__3.4-3.el9.x86_64",
    sha256 = "9be03d8382bf156d9cda703e453d213bde9f53389ec6841fb4cb900f13e22d99",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-3.4-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9be03d8382bf156d9cda703e453d213bde9f53389ec6841fb4cb900f13e22d99",
    ],
)

rpm(
    name = "libselinux-utils-0__3.4-3.el9.aarch64",
    sha256 = "da8240aa81ebc72b7dca93937887b19613fc2d45a4e1b84a58d3393fc81ae4cc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.4-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da8240aa81ebc72b7dca93937887b19613fc2d45a4e1b84a58d3393fc81ae4cc",
    ],
)

rpm(
    name = "libselinux-utils-0__3.4-3.el9.x86_64",
    sha256 = "fde4963b3512e33efd007a47f4adf893e5bd11b9a6fc4d41c329c67a98132204",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.4-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fde4963b3512e33efd007a47f4adf893e5bd11b9a6fc4d41c329c67a98132204",
    ],
)

rpm(
    name = "libsemanage-0__3.4-2.el9.aarch64",
    sha256 = "df0712f9a86e48674eae975751de4dfd858a0b2744576db5cdd7878db26d5dc4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsemanage-3.4-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/df0712f9a86e48674eae975751de4dfd858a0b2744576db5cdd7878db26d5dc4",
    ],
)

rpm(
    name = "libsemanage-0__3.4-2.el9.x86_64",
    sha256 = "f2a78bfe03b84b3722e5b0f17cb8b21e5b258e4221b3c0130dcd3e6ed00f43b7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsemanage-3.4-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f2a78bfe03b84b3722e5b0f17cb8b21e5b258e4221b3c0130dcd3e6ed00f43b7",
    ],
)

rpm(
    name = "libsepol-0__3.4-2.el9.aarch64",
    sha256 = "f39f41924402f7fa979bd9258b165a1322adeb33c7c9bab01f849faac200f8b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsepol-3.4-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f39f41924402f7fa979bd9258b165a1322adeb33c7c9bab01f849faac200f8b2",
    ],
)

rpm(
    name = "libsepol-0__3.4-2.el9.x86_64",
    sha256 = "eaa79d397630170ec53c5fd9744add41af4ab2cf5965324752e902dfbc568d46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsepol-3.4-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eaa79d397630170ec53c5fd9744add41af4ab2cf5965324752e902dfbc568d46",
    ],
)

rpm(
    name = "libsigsegv-0__2.13-4.el9.aarch64",
    sha256 = "097399718ae50fb03fde85fa151c060c50445a1a5af185052cac6b92d6fdcdae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsigsegv-2.13-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/097399718ae50fb03fde85fa151c060c50445a1a5af185052cac6b92d6fdcdae",
    ],
)

rpm(
    name = "libsigsegv-0__2.13-4.el9.x86_64",
    sha256 = "931bd0ec7050e8c3b37a9bfb489e30af32486a3c77203f1e9113eeceaa3b0a3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsigsegv-2.13-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/931bd0ec7050e8c3b37a9bfb489e30af32486a3c77203f1e9113eeceaa3b0a3a",
    ],
)

rpm(
    name = "libslirp-0__4.4.0-4.el9.aarch64",
    sha256 = "b3d35a0f47c1666bbed175469443617a6b2366bd198d8fe2e7b9fb047e058cff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libslirp-4.4.0-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b3d35a0f47c1666bbed175469443617a6b2366bd198d8fe2e7b9fb047e058cff",
    ],
)

rpm(
    name = "libslirp-0__4.4.0-4.el9.x86_64",
    sha256 = "06a12c4b78f60bd866ea91e648b86f1d52369f1981b5f18b6d2880ab8a951f81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libslirp-4.4.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/06a12c4b78f60bd866ea91e648b86f1d52369f1981b5f18b6d2880ab8a951f81",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-9.el9.aarch64",
    sha256 = "db518be8ad99feee87bcbccbd5c1c740f8cbe610f3a1d59bd70637053c37fba8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/db518be8ad99feee87bcbccbd5c1c740f8cbe610f3a1d59bd70637053c37fba8",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-9.el9.x86_64",
    sha256 = "ef59bdcffeaab46c8151ad3f36251d56d6b3aae7706f864c502965e6be099733",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ef59bdcffeaab46c8151ad3f36251d56d6b3aae7706f864c502965e6be099733",
    ],
)

rpm(
    name = "libss-0__1.46.5-3.el9.aarch64",
    sha256 = "aa55b0ee8bedeb59bd02534f4906bc8f01d79eacad833a278c25bf8841547bb0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libss-1.46.5-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aa55b0ee8bedeb59bd02534f4906bc8f01d79eacad833a278c25bf8841547bb0",
    ],
)

rpm(
    name = "libss-0__1.46.5-3.el9.x86_64",
    sha256 = "bf0e9aee3f87c9c9e660a03b879f958ef41c3e94d110af2d97b42cac2a1bde56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libss-1.46.5-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bf0e9aee3f87c9c9e660a03b879f958ef41c3e94d110af2d97b42cac2a1bde56",
    ],
)

rpm(
    name = "libssh-0__0.10.4-3.el9.aarch64",
    sha256 = "dcd4d7f38cb5379ff2ddcb91d7e01f89ceda7ecde0e69ec3b192656712a73ab6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-0.10.4-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dcd4d7f38cb5379ff2ddcb91d7e01f89ceda7ecde0e69ec3b192656712a73ab6",
    ],
)

rpm(
    name = "libssh-0__0.10.4-3.el9.x86_64",
    sha256 = "11c8e1ac10f4c02037c58a9fe4042badcdc86c497948430fe28d734de40e35b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-0.10.4-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11c8e1ac10f4c02037c58a9fe4042badcdc86c497948430fe28d734de40e35b0",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-3.el9.aarch64",
    sha256 = "c74b239eb2040ca414b32492fde653439e10f1072cfc6ff557a4c5d054b39261",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-config-0.10.4-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c74b239eb2040ca414b32492fde653439e10f1072cfc6ff557a4c5d054b39261",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-3.el9.x86_64",
    sha256 = "c74b239eb2040ca414b32492fde653439e10f1072cfc6ff557a4c5d054b39261",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-config-0.10.4-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c74b239eb2040ca414b32492fde653439e10f1072cfc6ff557a4c5d054b39261",
    ],
)

rpm(
    name = "libsss_idmap-0__2.7.3-4.el9.aarch64",
    sha256 = "8299f9860a52ffac1e56edffa6ccf8935835d1a115346ab65085e3f710963fc8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.7.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8299f9860a52ffac1e56edffa6ccf8935835d1a115346ab65085e3f710963fc8",
    ],
)

rpm(
    name = "libsss_idmap-0__2.7.3-4.el9.x86_64",
    sha256 = "8a247052585816eab83625f9fd48e51ea0d3ec9d729eb8c52fe8fc14b7ae49b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.7.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a247052585816eab83625f9fd48e51ea0d3ec9d729eb8c52fe8fc14b7ae49b2",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.7.3-4.el9.aarch64",
    sha256 = "bcc9b933cc29062dccbafd4d09cc34fd6e54acde2d22d809bfd3fdd32856de42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.7.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bcc9b933cc29062dccbafd4d09cc34fd6e54acde2d22d809bfd3fdd32856de42",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.7.3-4.el9.x86_64",
    sha256 = "a278da162cd0aacbfcd789e6a518942328a1bd3e40134a56db30d1dd340db8be",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.7.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a278da162cd0aacbfcd789e6a518942328a1bd3e40134a56db30d1dd340db8be",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.3.1-2.1.el9.aarch64",
    sha256 = "496c3bca33472135bd2c54072f139634c4631efa579d99734e9a1ace8ca4deb5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/496c3bca33472135bd2c54072f139634c4631efa579d99734e9a1ace8ca4deb5",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.3.1-2.1.el9.x86_64",
    sha256 = "1e7a79405156d04cceba673f607e0962bcb656ab09eb961eb2b11508b2de6f4d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.3.1-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e7a79405156d04cceba673f607e0962bcb656ab09eb961eb2b11508b2de6f4d",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-7.el9.aarch64",
    sha256 = "4eaa01b044d688793eb928170f3937bc8618b76d702d49a8843aa89461e43fa8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtasn1-4.16.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4eaa01b044d688793eb928170f3937bc8618b76d702d49a8843aa89461e43fa8",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-7.el9.x86_64",
    sha256 = "656031558c53da4a5b3ccfd883bd6d55996037891323152b1f07e8d1d5377406",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtasn1-4.16.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/656031558c53da4a5b3ccfd883bd6d55996037891323152b1f07e8d1d5377406",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-0.el9.aarch64",
    sha256 = "34ec67f125034e9cd6562b79ba13f8155bba3bcfe1f71ddd3def862af1b6f6b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtirpc-1.3.3-0.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/34ec67f125034e9cd6562b79ba13f8155bba3bcfe1f71ddd3def862af1b6f6b0",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-0.el9.x86_64",
    sha256 = "fb0e5cda7aa1aabe26a70c9b2e1ca64ea5658f05de2d5977e198dfa3f8bb1645",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtirpc-1.3.3-0.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb0e5cda7aa1aabe26a70c9b2e1ca64ea5658f05de2d5977e198dfa3f8bb1645",
    ],
)

rpm(
    name = "libtpms-0__0.8.2-0.20210301git729fc6a4ca.el9.6.aarch64",
    sha256 = "825da63dd42e049f3d5bc037e2c7e1a193823a1999d84434765b3c4d4e8785bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libtpms-0.8.2-0.20210301git729fc6a4ca.el9.6.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/825da63dd42e049f3d5bc037e2c7e1a193823a1999d84434765b3c4d4e8785bd",
    ],
)

rpm(
    name = "libtpms-0__0.8.2-0.20210301git729fc6a4ca.el9.6.x86_64",
    sha256 = "0f20d5977b5eb078a892231d83ee0b2ce74734216502371e276d8a1c5615679d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libtpms-0.8.2-0.20210301git729fc6a4ca.el9.6.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0f20d5977b5eb078a892231d83ee0b2ce74734216502371e276d8a1c5615679d",
    ],
)

rpm(
    name = "libubsan-0__11.3.1-2.1.el9.aarch64",
    sha256 = "71b84ff036b26e01a22b16f7271f458be2f2c7cdda40107aa2e5133224c4f184",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.3.1-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/71b84ff036b26e01a22b16f7271f458be2f2c7cdda40107aa2e5133224c4f184",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-15.el9.aarch64",
    sha256 = "09381b23c9d2343592b8b565dcbb23d055999ab1e521aa802b6d40a682b80e42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libunistring-0.9.10-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/09381b23c9d2343592b8b565dcbb23d055999ab1e521aa802b6d40a682b80e42",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-15.el9.x86_64",
    sha256 = "11e736e44265d2d0ca0afa4c11cfe0856553c4124e534fb616e6ab61c9b59e46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libunistring-0.9.10-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11e736e44265d2d0ca0afa4c11cfe0856553c4124e534fb616e6ab61c9b59e46",
    ],
)

rpm(
    name = "libusbx-0__1.0.26-1.el9.aarch64",
    sha256 = "f008b954b622f27dbc5b0c8f3633589c844b5428a1dfe84ca96d42a72dae707c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libusbx-1.0.26-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f008b954b622f27dbc5b0c8f3633589c844b5428a1dfe84ca96d42a72dae707c",
    ],
)

rpm(
    name = "libusbx-0__1.0.26-1.el9.x86_64",
    sha256 = "bfc8e2bfbcc0e6aaa4e4e665e52ebdc93fb84f7bf00be4640df0fa6df9cbf042",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libusbx-1.0.26-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bfc8e2bfbcc0e6aaa4e4e665e52ebdc93fb84f7bf00be4640df0fa6df9cbf042",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-6.el9.aarch64",
    sha256 = "65cd8c3813afc69dd2ea9eeb6e2fc7db4a7d626b51efe376b8000dfdaa10402a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libutempter-1.2.1-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/65cd8c3813afc69dd2ea9eeb6e2fc7db4a7d626b51efe376b8000dfdaa10402a",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-6.el9.x86_64",
    sha256 = "fab361a9cba04490fd8b5664049983d1e57ebf7c1080804726ba600708524125",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libutempter-1.2.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fab361a9cba04490fd8b5664049983d1e57ebf7c1080804726ba600708524125",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-9.el9.aarch64",
    sha256 = "1de486c59df5317dc192a194df9ca548c4fb2cc3d3d1a4dde20bef6eaad62f1f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libuuid-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1de486c59df5317dc192a194df9ca548c4fb2cc3d3d1a4dde20bef6eaad62f1f",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-9.el9.x86_64",
    sha256 = "73b06bf582fb3e0161e55714040e9e0c44d81099dc17485bacaf8c30d3fab4e7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/73b06bf582fb3e0161e55714040e9e0c44d81099dc17485bacaf8c30d3fab4e7",
    ],
)

rpm(
    name = "libverto-0__0.3.2-3.el9.aarch64",
    sha256 = "1190ea8310b0dab3ebbade3180b4c2cf7064e90c894e5415711d7751e709be8a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libverto-0.3.2-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1190ea8310b0dab3ebbade3180b4c2cf7064e90c894e5415711d7751e709be8a",
    ],
)

rpm(
    name = "libverto-0__0.3.2-3.el9.x86_64",
    sha256 = "c55578b84f169c4ed79b2d50ea03fd1817007e35062c9fe7a58e6cad025f3b24",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libverto-0.3.2-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c55578b84f169c4ed79b2d50ea03fd1817007e35062c9fe7a58e6cad025f3b24",
    ],
)

rpm(
    name = "libverto-libev-0__0.3.2-3.el9.x86_64",
    sha256 = "7d4423bc582773e23bf08f1f73d99275838a45fa188971a2f20448811e524a50",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libverto-libev-0.3.2-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d4423bc582773e23bf08f1f73d99275838a45fa188971a2f20448811e524a50",
    ],
)

rpm(
    name = "libvirt-client-0__8.7.0-1.el9.aarch64",
    sha256 = "f396538fc60b19b9447d77a46f4a5dbea863cdd0de02266bdd93cdfd2e9b4f8b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-8.7.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f396538fc60b19b9447d77a46f4a5dbea863cdd0de02266bdd93cdfd2e9b4f8b",
    ],
)

rpm(
    name = "libvirt-client-0__8.7.0-1.el9.x86_64",
    sha256 = "19e3ac35fb5bcb04cbec05274e19e20d43777628530938c22763c7276d63483c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19e3ac35fb5bcb04cbec05274e19e20d43777628530938c22763c7276d63483c",
    ],
)

rpm(
    name = "libvirt-daemon-0__8.7.0-1.el9.aarch64",
    sha256 = "da0811366d40d01f3d22b716c0afa781df2fc662969cee9351b72c8f360ea07b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-8.7.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da0811366d40d01f3d22b716c0afa781df2fc662969cee9351b72c8f360ea07b",
    ],
)

rpm(
    name = "libvirt-daemon-0__8.7.0-1.el9.x86_64",
    sha256 = "d934e6b436846b7af4724c368a2a4ba55253111b45ff80f9e5051b63cb71ed30",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d934e6b436846b7af4724c368a2a4ba55253111b45ff80f9e5051b63cb71ed30",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__8.7.0-1.el9.aarch64",
    sha256 = "cb060c4e33b19256bd6c502e42df57d7f7aea55e26c5e3a8a3499b7cbad805d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-8.7.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb060c4e33b19256bd6c502e42df57d7f7aea55e26c5e3a8a3499b7cbad805d2",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__8.7.0-1.el9.x86_64",
    sha256 = "8c95f0b94d0e58960713b8d2212055c77a57311997b6c0a48448659e3897eb03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c95f0b94d0e58960713b8d2212055c77a57311997b6c0a48448659e3897eb03",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__8.7.0-1.el9.x86_64",
    sha256 = "660aa18ed1a6df440699a347cdf01793acb0dd9b55cd143dd42e8fb7d7420893",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/660aa18ed1a6df440699a347cdf01793acb0dd9b55cd143dd42e8fb7d7420893",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__8.7.0-1.el9.x86_64",
    sha256 = "e5af6782881294d5e5801ac77f9b68f707cff79044cb256f7a95537e950e5b87",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e5af6782881294d5e5801ac77f9b68f707cff79044cb256f7a95537e950e5b87",
    ],
)

rpm(
    name = "libvirt-devel-0__8.7.0-1.el9.aarch64",
    sha256 = "bb7df619b153dc1f0744e8098d8420cd9077ce6e22dcc2706eee0f2d35f15cb9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-8.7.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bb7df619b153dc1f0744e8098d8420cd9077ce6e22dcc2706eee0f2d35f15cb9",
    ],
)

rpm(
    name = "libvirt-devel-0__8.7.0-1.el9.x86_64",
    sha256 = "463b5b4b4dc614e2b631de0ddb183e0431113986c817462e3a3e135cbc6a7220",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/463b5b4b4dc614e2b631de0ddb183e0431113986c817462e3a3e135cbc6a7220",
    ],
)

rpm(
    name = "libvirt-libs-0__8.7.0-1.el9.aarch64",
    sha256 = "1d35f5e79e5a68dbc9882c6090e321398c672c55b3c2c425ed2e3df59b0bdeb2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-8.7.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1d35f5e79e5a68dbc9882c6090e321398c672c55b3c2c425ed2e3df59b0bdeb2",
    ],
)

rpm(
    name = "libvirt-libs-0__8.7.0-1.el9.x86_64",
    sha256 = "ec9300092c6f20fd11d4083299534a067c9d20337e34f0ed752570fe74520279",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-8.7.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ec9300092c6f20fd11d4083299534a067c9d20337e34f0ed752570fe74520279",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.18-3.el9.aarch64",
    sha256 = "f697d91abb19e9be9b69b8836a802711d2cf7989af27a4e1ba261f35ce53b8b5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxcrypt-4.4.18-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f697d91abb19e9be9b69b8836a802711d2cf7989af27a4e1ba261f35ce53b8b5",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.18-3.el9.x86_64",
    sha256 = "97e88678b420f619a44608fff30062086aa1dd6931ecbd54f21bba005ff1de1a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxcrypt-4.4.18-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/97e88678b420f619a44608fff30062086aa1dd6931ecbd54f21bba005ff1de1a",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.4.18-3.el9.aarch64",
    sha256 = "4d6085cd4068264576d023784ceddf0d9e19eb7633d87c31efd9444dab0c3420",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libxcrypt-devel-4.4.18-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4d6085cd4068264576d023784ceddf0d9e19eb7633d87c31efd9444dab0c3420",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.4.18-3.el9.x86_64",
    sha256 = "162461e5f31f94907c91815370b545844cc9d33b1311e0063e23ae427241d1e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libxcrypt-devel-4.4.18-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/162461e5f31f94907c91815370b545844cc9d33b1311e0063e23ae427241d1e0",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.4.18-3.el9.aarch64",
    sha256 = "34033b8a089eac80956a9542a77b4c5e8c32a27ab0e8cf61728a9fbac970d5ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libxcrypt-static-4.4.18-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/34033b8a089eac80956a9542a77b4c5e8c32a27ab0e8cf61728a9fbac970d5ad",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.4.18-3.el9.x86_64",
    sha256 = "251a45a42a342459303bb1b928359eed1ea88bcd12605a9fe084f24fac020869",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libxcrypt-static-4.4.18-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/251a45a42a342459303bb1b928359eed1ea88bcd12605a9fe084f24fac020869",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-2.el9.aarch64",
    sha256 = "306a7da40dad31e0307870d71b6502d1c46aab94171b5148745b4a31e39fcd96",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.13-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/306a7da40dad31e0307870d71b6502d1c46aab94171b5148745b4a31e39fcd96",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-2.el9.x86_64",
    sha256 = "2ada7b1a4c330ba634e58c5720e0761bc649ff71bf3b4fb9a9b6d330a5c337c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.13-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ada7b1a4c330ba634e58c5720e0761bc649ff71bf3b4fb9a9b6d330a5c337c4",
    ],
)

rpm(
    name = "libzstd-0__1.5.1-2.el9.aarch64",
    sha256 = "68101e014106305c840611b64d71311600edb30a34e09514c169c9eef6090d42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libzstd-1.5.1-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/68101e014106305c840611b64d71311600edb30a34e09514c169c9eef6090d42",
    ],
)

rpm(
    name = "libzstd-0__1.5.1-2.el9.x86_64",
    sha256 = "0840678cb3c1b418286f55da6973df9468c4cf500192de82d05ef28e6b4215a0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libzstd-1.5.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0840678cb3c1b418286f55da6973df9468c4cf500192de82d05ef28e6b4215a0",
    ],
)

rpm(
    name = "lua-libs-0__5.4.2-4.el9.aarch64",
    sha256 = "a5924e224aa5941e9bb54bf00b40200e790454d78a4772906844bd45b823ccda",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/lua-libs-5.4.2-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a5924e224aa5941e9bb54bf00b40200e790454d78a4772906844bd45b823ccda",
    ],
)

rpm(
    name = "lua-libs-0__5.4.2-4.el9.x86_64",
    sha256 = "59342315ee1c9589ae36bde722a66d718cc7cb5b750521aa31dc7704e2d0c0f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/lua-libs-5.4.2-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/59342315ee1c9589ae36bde722a66d718cc7cb5b750521aa31dc7704e2d0c0f4",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.3-5.el9.aarch64",
    sha256 = "9aa14d26393dd46c0a390cf04f939f7f759a33165bdb506f8bee0653f3b70f45",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/lz4-libs-1.9.3-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9aa14d26393dd46c0a390cf04f939f7f759a33165bdb506f8bee0653f3b70f45",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.3-5.el9.x86_64",
    sha256 = "cba6a63054d070956a182e33269ee245bcfbe87e3e605c27816519db762a66ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/lz4-libs-1.9.3-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cba6a63054d070956a182e33269ee245bcfbe87e3e605c27816519db762a66ad",
    ],
)

rpm(
    name = "lzo-0__2.10-7.el9.aarch64",
    sha256 = "eb10493cb600631bc42b0c0bad707f9b79da912750fa9b9e5d8a9978a98babdf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/lzo-2.10-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eb10493cb600631bc42b0c0bad707f9b79da912750fa9b9e5d8a9978a98babdf",
    ],
)

rpm(
    name = "lzo-0__2.10-7.el9.x86_64",
    sha256 = "7bee77c82bd6c183bba7a4b4fdd3ecb99d0a089a25c735ebbabc44e0c51e4b2e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/lzo-2.10-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7bee77c82bd6c183bba7a4b4fdd3ecb99d0a089a25c735ebbabc44e0c51e4b2e",
    ],
)

rpm(
    name = "lzop-0__1.04-8.el9.aarch64",
    sha256 = "ae5bdeee08c76f6ce902c70e16b373160e1c595dd1718f2f1db3a37ec5d63703",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/lzop-1.04-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ae5bdeee08c76f6ce902c70e16b373160e1c595dd1718f2f1db3a37ec5d63703",
    ],
)

rpm(
    name = "lzop-0__1.04-8.el9.x86_64",
    sha256 = "ad84787d14a62195822ea89cec0fcf475f09b425f0822ce34d858d2d8bbd9466",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/lzop-1.04-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ad84787d14a62195822ea89cec0fcf475f09b425f0822ce34d858d2d8bbd9466",
    ],
)

rpm(
    name = "make-1__4.3-7.el9.aarch64",
    sha256 = "63386f5e0f71bfff56bc73906e70e0a3a9d6679f5e272381a7b1f954d2a27367",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/make-4.3-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/63386f5e0f71bfff56bc73906e70e0a3a9d6679f5e272381a7b1f954d2a27367",
    ],
)

rpm(
    name = "make-1__4.3-7.el9.x86_64",
    sha256 = "d2c768e50950964bfdcefb9f1a36b268ae695fdea2bfd24daf8587def885e55d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/make-4.3-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d2c768e50950964bfdcefb9f1a36b268ae695fdea2bfd24daf8587def885e55d",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-7.el9.aarch64",
    sha256 = "f3bd8510505a53450abe05dc34edbc5313fe89a6f88d0252624205dc7bb884c7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/mpfr-4.1.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f3bd8510505a53450abe05dc34edbc5313fe89a6f88d0252624205dc7bb884c7",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-7.el9.x86_64",
    sha256 = "179760104aa5a31ca463c586d0f21f380ba4d0eed212eee91bd1ca513e5d7a8d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/mpfr-4.1.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/179760104aa5a31ca463c586d0f21f380ba4d0eed212eee91bd1ca513e5d7a8d",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-8.20210508.el9.aarch64",
    sha256 = "e4cc4a4a479b8c27776debba5c20e8ef21dc4b513da62a25ed09f88386ac08a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.2-8.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e4cc4a4a479b8c27776debba5c20e8ef21dc4b513da62a25ed09f88386ac08a8",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-8.20210508.el9.x86_64",
    sha256 = "e4cc4a4a479b8c27776debba5c20e8ef21dc4b513da62a25ed09f88386ac08a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.2-8.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e4cc4a4a479b8c27776debba5c20e8ef21dc4b513da62a25ed09f88386ac08a8",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-8.20210508.el9.aarch64",
    sha256 = "26a21395b0bb4f7b60ab89bacaa8fc210c9921f1aba90ec950b91b3ee9e25dcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.2-8.20210508.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/26a21395b0bb4f7b60ab89bacaa8fc210c9921f1aba90ec950b91b3ee9e25dcc",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-8.20210508.el9.x86_64",
    sha256 = "328f4d50e66b00f24344ebe239817204fda8e68b1d988c6943abb3c36231beaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.2-8.20210508.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/328f4d50e66b00f24344ebe239817204fda8e68b1d988c6943abb3c36231beaa",
    ],
)

rpm(
    name = "ndctl-libs-0__71.1-7.el9.x86_64",
    sha256 = "de5096ba7c6600452485207200b695bf0b7d3b9924c18ae36c605f67ef11c3c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ndctl-libs-71.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de5096ba7c6600452485207200b695bf0b7d3b9924c18ae36c605f67ef11c3c2",
    ],
)

rpm(
    name = "nettle-0__3.8-3.el9.aarch64",
    sha256 = "94386170c99bb195481806f20ae034f246e863fc02a1eeaddf88212ae545f826",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nettle-3.8-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/94386170c99bb195481806f20ae034f246e863fc02a1eeaddf88212ae545f826",
    ],
)

rpm(
    name = "nettle-0__3.8-3.el9.x86_64",
    sha256 = "ed956f9e018ab00d6ddf567487dd6bbcdc634d27dd69b485b416c6cf40026b82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nettle-3.8-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed956f9e018ab00d6ddf567487dd6bbcdc634d27dd69b485b416c6cf40026b82",
    ],
)

rpm(
    name = "nfs-utils-1__2.5.4-15.el9.x86_64",
    sha256 = "6fc102aa1648103af63099688639835dcd434915dc1d0a3ea41b3ee37b355861",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nfs-utils-2.5.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6fc102aa1648103af63099688639835dcd434915dc1d0a3ea41b3ee37b355861",
    ],
)

rpm(
    name = "nftables-1__1.0.4-2.el9.aarch64",
    sha256 = "303ffe5f65156d1ca673be719170968f94f33a0c27fb820454872abcabc29b26",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.4-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/303ffe5f65156d1ca673be719170968f94f33a0c27fb820454872abcabc29b26",
    ],
)

rpm(
    name = "nftables-1__1.0.4-2.el9.x86_64",
    sha256 = "5906b0c1252f7a929881de4dac54faaa521c72ab45a09737f136bb34945140d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.4-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5906b0c1252f7a929881de4dac54faaa521c72ab45a09737f136bb34945140d8",
    ],
)

rpm(
    name = "nmap-ncat-3__7.91-10.el9.aarch64",
    sha256 = "1b0aea22fd4028782d54b1a11fb77f8394c958f9d47799022c8d357527d77444",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.91-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1b0aea22fd4028782d54b1a11fb77f8394c958f9d47799022c8d357527d77444",
    ],
)

rpm(
    name = "nmap-ncat-3__7.91-10.el9.x86_64",
    sha256 = "7151569bbd4890f4548ff4571d3df8dc312e4fa24d245565dbdbcc0545b17d90",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.91-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7151569bbd4890f4548ff4571d3df8dc312e4fa24d245565dbdbcc0545b17d90",
    ],
)

rpm(
    name = "npth-0__1.6-8.el9.x86_64",
    sha256 = "a7da4ef003bc60045bc60dae299b703e7f1db326f25208fb922ce1b79e2882da",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/npth-1.6-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a7da4ef003bc60045bc60dae299b703e7f1db326f25208fb922ce1b79e2882da",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.14-7.el9.aarch64",
    sha256 = "288250e514a6d1e4299656c1b68d49653cc92060a35024631c80fc0f206cf433",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.14-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/288250e514a6d1e4299656c1b68d49653cc92060a35024631c80fc0f206cf433",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.14-7.el9.x86_64",
    sha256 = "7a3bc16b3fee48c53e0f54a7cb4cd3857eb1be3984d58da3bdf2c297d6b55af1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.14-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7a3bc16b3fee48c53e0f54a7cb4cd3857eb1be3984d58da3bdf2c297d6b55af1",
    ],
)

rpm(
    name = "numad-0__0.5-36.20150602git.el9.aarch64",
    sha256 = "c7cd5aa2f682cfa56f9c35f4c0b28a2e75c23dee993b54e4c39c21931392f6bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numad-0.5-36.20150602git.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7cd5aa2f682cfa56f9c35f4c0b28a2e75c23dee993b54e4c39c21931392f6bb",
    ],
)

rpm(
    name = "numad-0__0.5-36.20150602git.el9.x86_64",
    sha256 = "1b4242cdefa165b70926aee4dd4606b0f5ecdf4a436812746e9fe1c417724d23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numad-0.5-36.20150602git.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1b4242cdefa165b70926aee4dd4606b0f5ecdf4a436812746e9fe1c417724d23",
    ],
)

rpm(
    name = "openldap-0__2.6.2-3.el9.aarch64",
    sha256 = "492daf98d77aa62021d3956e0a0727c66bd13c2322267c8e6556bfbb68c06fa5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openldap-2.6.2-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/492daf98d77aa62021d3956e0a0727c66bd13c2322267c8e6556bfbb68c06fa5",
    ],
)

rpm(
    name = "openldap-0__2.6.2-3.el9.x86_64",
    sha256 = "8ce2a645dfc4444c698d8c2a644df93fd53b9a00ef887e138528aa473ee76456",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openldap-2.6.2-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ce2a645dfc4444c698d8c2a644df93fd53b9a00ef887e138528aa473ee76456",
    ],
)

rpm(
    name = "openssl-1__3.0.1-41.el9.aarch64",
    sha256 = "259c086886ca32fc72e87af8de3f70a3f5cb4da9edc778b47181a6008588ee61",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.0.1-41.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/259c086886ca32fc72e87af8de3f70a3f5cb4da9edc778b47181a6008588ee61",
    ],
)

rpm(
    name = "openssl-1__3.0.1-41.el9.x86_64",
    sha256 = "73b43cfcc4b2c0be24480d8b979dde170d6a71a4023bbd35f16e2b050f07018f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.0.1-41.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/73b43cfcc4b2c0be24480d8b979dde170d6a71a4023bbd35f16e2b050f07018f",
    ],
)

rpm(
    name = "openssl-libs-1__3.0.1-41.el9.aarch64",
    sha256 = "c5ddd3de059392ede313bdf18f9fe10c51d5f23e479c0bea16999b38fe556a38",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.0.1-41.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c5ddd3de059392ede313bdf18f9fe10c51d5f23e479c0bea16999b38fe556a38",
    ],
)

rpm(
    name = "openssl-libs-1__3.0.1-41.el9.x86_64",
    sha256 = "6891a18063e05c0ec43f83ee1fa05315501ac9c6f01774a7d627c1f72c6812a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.0.1-41.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6891a18063e05c0ec43f83ee1fa05315501ac9c6f01774a7d627c1f72c6812a8",
    ],
)

rpm(
    name = "p11-kit-0__0.24.1-2.el9.aarch64",
    sha256 = "98e7f00d012549fa8fbaba21626388a0b07731f3f25a5801418247d66a5a985f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-0.24.1-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98e7f00d012549fa8fbaba21626388a0b07731f3f25a5801418247d66a5a985f",
    ],
)

rpm(
    name = "p11-kit-0__0.24.1-2.el9.x86_64",
    sha256 = "da167e41efd19cf25fd1c708b6f123d0203824324b14dd32401d49f2aa0ef0a6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-0.24.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da167e41efd19cf25fd1c708b6f123d0203824324b14dd32401d49f2aa0ef0a6",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.24.1-2.el9.aarch64",
    sha256 = "80e288a5b62f20f7794674c6fdf2f0765a322cd0e81df9359e37582fe950289c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.24.1-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/80e288a5b62f20f7794674c6fdf2f0765a322cd0e81df9359e37582fe950289c",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.24.1-2.el9.x86_64",
    sha256 = "ae9a633c58980328bef6358c6aa3c9ce0a65130c66fbfa4249922ddf5a3e2bb1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.24.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ae9a633c58980328bef6358c6aa3c9ce0a65130c66fbfa4249922ddf5a3e2bb1",
    ],
)

rpm(
    name = "pam-0__1.5.1-13.el9.aarch64",
    sha256 = "eb7071e8799762e82fdc64e677d313ab3fe1679477d43e062fdac87b643df067",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eb7071e8799762e82fdc64e677d313ab3fe1679477d43e062fdac87b643df067",
    ],
)

rpm(
    name = "pam-0__1.5.1-13.el9.x86_64",
    sha256 = "85eb2a199415a7f0872ad93f31e839eed37879ab7bd98d22f2715471e8fd524f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/85eb2a199415a7f0872ad93f31e839eed37879ab7bd98d22f2715471e8fd524f",
    ],
)

rpm(
    name = "parted-0__3.5-2.el9.x86_64",
    sha256 = "ab6500203b5f0b3bd551c026ca60e5aec51170bdc62978a2702d386d2a645b5e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/parted-3.5-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ab6500203b5f0b3bd551c026ca60e5aec51170bdc62978a2702d386d2a645b5e",
    ],
)

rpm(
    name = "passt-0__0__caret__20221026.gf212044-1.el9.aarch64",
    sha256 = "9e8a111d88ac449a345fd4fbe4f98e49c74b13ac9d5e09d00e43d1df8a6e3f35",
    urls = [
        "https://passt.top/builds/copr/0%5E20221026.gf212044/centos-stream-9-aarch64/04992985-passt/passt-0^20221026.gf212044-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9e8a111d88ac449a345fd4fbe4f98e49c74b13ac9d5e09d00e43d1df8a6e3f35",
    ],
)

rpm(
    name = "passt-0__0__caret__20221026.gf212044-1.el9.x86_64",
    sha256 = "a3774617cbc3d17ff85bcfbd4c33fb41ce5048e8b13a362d5f262897ad06fa92",
    urls = [
        "https://passt.top/builds/copr/0%5E20221026.gf212044/centos-stream-9-x86_64/04992985-passt/passt-0^20221026.gf212044-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a3774617cbc3d17ff85bcfbd4c33fb41ce5048e8b13a362d5f262897ad06fa92",
    ],
)

rpm(
    name = "pcre-0__8.44-3.el9.3.aarch64",
    sha256 = "0331efd537704e75e26324ba6bb1568762d01bafe7fbce5b981ff0ee0d3ea80c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre-8.44-3.el9.3.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0331efd537704e75e26324ba6bb1568762d01bafe7fbce5b981ff0ee0d3ea80c",
    ],
)

rpm(
    name = "pcre-0__8.44-3.el9.3.x86_64",
    sha256 = "4a3cb61eb08c4f24e44756b6cb329812fe48d5c65c1fba546fadfa975045a8c5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre-8.44-3.el9.3.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a3cb61eb08c4f24e44756b6cb329812fe48d5c65c1fba546fadfa975045a8c5",
    ],
)

rpm(
    name = "pcre2-0__10.40-2.el9.aarch64",
    sha256 = "8879da4bf6f8ec1a17105a3d54130d77afad48021c7280d8edb3f63fed80c4a5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre2-10.40-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8879da4bf6f8ec1a17105a3d54130d77afad48021c7280d8edb3f63fed80c4a5",
    ],
)

rpm(
    name = "pcre2-0__10.40-2.el9.x86_64",
    sha256 = "8cc83f9f130e6ef50d54d75eb4050ce879d8acaf5bb616b398ad92c1ad2b3d21",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre2-10.40-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8cc83f9f130e6ef50d54d75eb4050ce879d8acaf5bb616b398ad92c1ad2b3d21",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-2.el9.aarch64",
    sha256 = "4dad144194fe6794c7621c38b6a7f917a81ceaeb3f2be25833b9b0af1181ebe2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre2-syntax-10.40-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4dad144194fe6794c7621c38b6a7f917a81ceaeb3f2be25833b9b0af1181ebe2",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-2.el9.x86_64",
    sha256 = "4dad144194fe6794c7621c38b6a7f917a81ceaeb3f2be25833b9b0af1181ebe2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre2-syntax-10.40-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4dad144194fe6794c7621c38b6a7f917a81ceaeb3f2be25833b9b0af1181ebe2",
    ],
)

rpm(
    name = "pixman-0__0.40.0-5.el9.aarch64",
    sha256 = "0cb4f93b6307d5c1cbc6738b187eaac08ce571297fb4a0bd0f8f2a9c843db83b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/pixman-0.40.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0cb4f93b6307d5c1cbc6738b187eaac08ce571297fb4a0bd0f8f2a9c843db83b",
    ],
)

rpm(
    name = "pixman-0__0.40.0-5.el9.x86_64",
    sha256 = "8673872772fec90180fa9688363b4d808c5d01bd9951afaddfa7e64bb7274aba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/pixman-0.40.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8673872772fec90180fa9688363b4d808c5d01bd9951afaddfa7e64bb7274aba",
    ],
)

rpm(
    name = "pkgconf-0__1.7.3-10.el9.aarch64",
    sha256 = "94f174c9829d44f345bb8a734147f379ba95fb47d04befdb20a17e8b158b3710",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pkgconf-1.7.3-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/94f174c9829d44f345bb8a734147f379ba95fb47d04befdb20a17e8b158b3710",
    ],
)

rpm(
    name = "pkgconf-0__1.7.3-10.el9.x86_64",
    sha256 = "2ff8b131570687e4eca9877feaa9058ef7c0772cff507c019f6c26aff126d065",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pkgconf-1.7.3-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ff8b131570687e4eca9877feaa9058ef7c0772cff507c019f6c26aff126d065",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.7.3-10.el9.aarch64",
    sha256 = "de4946454f110a9b12ab50c9c3dfaa68633b4ae3cb4e5278b23d491eb3edc27a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pkgconf-m4-1.7.3-10.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de4946454f110a9b12ab50c9c3dfaa68633b4ae3cb4e5278b23d491eb3edc27a",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.7.3-10.el9.x86_64",
    sha256 = "de4946454f110a9b12ab50c9c3dfaa68633b4ae3cb4e5278b23d491eb3edc27a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pkgconf-m4-1.7.3-10.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de4946454f110a9b12ab50c9c3dfaa68633b4ae3cb4e5278b23d491eb3edc27a",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.7.3-10.el9.aarch64",
    sha256 = "d36ff5361c4b31273b15ff34f0fec5ae5316d6555270b3d051d97c85c12defac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pkgconf-pkg-config-1.7.3-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d36ff5361c4b31273b15ff34f0fec5ae5316d6555270b3d051d97c85c12defac",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.7.3-10.el9.x86_64",
    sha256 = "e308e84f06756bf3c14bc426fb2519008ad8423925c4662bb379ea87aced19d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pkgconf-pkg-config-1.7.3-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e308e84f06756bf3c14bc426fb2519008ad8423925c4662bb379ea87aced19d9",
    ],
)

rpm(
    name = "policycoreutils-0__3.4-4.el9.aarch64",
    sha256 = "0f50b567d538990d025a59227768ecc8f5359cd747a1eddfcceaeab82bdc4064",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.4-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0f50b567d538990d025a59227768ecc8f5359cd747a1eddfcceaeab82bdc4064",
    ],
)

rpm(
    name = "policycoreutils-0__3.4-4.el9.x86_64",
    sha256 = "8a43d0f8c24f1c746acae28c18232d132da6f988b022ef08d7d734f95e76b27b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a43d0f8c24f1c746acae28c18232d132da6f988b022ef08d7d734f95e76b27b",
    ],
)

rpm(
    name = "polkit-0__0.117-10.el9.aarch64",
    sha256 = "56e8df687e647f0a7415d2c27698ee8181c3737efe1ca33f01a3fd42732fbf2a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-0.117-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56e8df687e647f0a7415d2c27698ee8181c3737efe1ca33f01a3fd42732fbf2a",
    ],
)

rpm(
    name = "polkit-0__0.117-10.el9.x86_64",
    sha256 = "93d7128562762cf4046b849e8da6bbd65f0a31ba00c7db336976ff88d203f04f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-0.117-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/93d7128562762cf4046b849e8da6bbd65f0a31ba00c7db336976ff88d203f04f",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-10.el9.aarch64",
    sha256 = "afe7854cecb2d59429e4435d75cfae647af0a49f31a063d184bfb991ceef74b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.117-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/afe7854cecb2d59429e4435d75cfae647af0a49f31a063d184bfb991ceef74b2",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-10.el9.x86_64",
    sha256 = "bedb4e439852632b74834a58cdc10313dd2b0737b551ca39b7e8485ef0b02350",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.117-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bedb4e439852632b74834a58cdc10313dd2b0737b551ca39b7e8485ef0b02350",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-21.el9.aarch64",
    sha256 = "c22bfa6ebfb7c8803cd115e750f29408a00d73475ec8b77d409b7eabd2aeb61a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-pkla-compat-0.1-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c22bfa6ebfb7c8803cd115e750f29408a00d73475ec8b77d409b7eabd2aeb61a",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-21.el9.x86_64",
    sha256 = "ffb4cc04548f24cf7cd62da9747d3839af7676b29b60cfd3da59c6ec31ebdf99",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-pkla-compat-0.1-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffb4cc04548f24cf7cd62da9747d3839af7676b29b60cfd3da59c6ec31ebdf99",
    ],
)

rpm(
    name = "popt-0__1.18-8.el9.aarch64",
    sha256 = "032427adaa37d2a1c6d2f3cab42ccbdce2c6d9b3c1f3cd91c05a92c99198babb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/popt-1.18-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/032427adaa37d2a1c6d2f3cab42ccbdce2c6d9b3c1f3cd91c05a92c99198babb",
    ],
)

rpm(
    name = "popt-0__1.18-8.el9.x86_64",
    sha256 = "d864419035e99f8bb06f5d1c767608ed81f942cb128a98b590c1dbc4afbd54d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/popt-1.18-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d864419035e99f8bb06f5d1c767608ed81f942cb128a98b590c1dbc4afbd54d4",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-8.el9.aarch64",
    sha256 = "530b2c7b32c0664dbaf19afb07af2f05796eadda02d0b45e75aa1b1c0fa3ee76",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.17-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/530b2c7b32c0664dbaf19afb07af2f05796eadda02d0b45e75aa1b1c0fa3ee76",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-8.el9.x86_64",
    sha256 = "c6814ec3b0c64dacc7b4dc799ece741a0ce111dc245be5ead64580253499df5c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.17-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c6814ec3b0c64dacc7b4dc799ece741a0ce111dc245be5ead64580253499df5c",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-12.el9.aarch64",
    sha256 = "f6096a23837dcf6755968c54d9a16875f3e9a86388f936ab1e2ff28e381f3cfa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/protobuf-c-1.3.3-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f6096a23837dcf6755968c54d9a16875f3e9a86388f936ab1e2ff28e381f3cfa",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-12.el9.x86_64",
    sha256 = "5d1091426fc81321e00c805fff53b2da159de91d6d219d20f3defdfde41bf1d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/protobuf-c-1.3.3-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5d1091426fc81321e00c805fff53b2da159de91d6d219d20f3defdfde41bf1d4",
    ],
)

rpm(
    name = "psmisc-0__23.4-3.el9.aarch64",
    sha256 = "4ad245b41ebf13cbabbb2962fad8d4aa0db7c75eb2171a4235252ad48e81a680",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/psmisc-23.4-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4ad245b41ebf13cbabbb2962fad8d4aa0db7c75eb2171a4235252ad48e81a680",
    ],
)

rpm(
    name = "psmisc-0__23.4-3.el9.x86_64",
    sha256 = "e02fc28d42912689b006fcc1e98bdb5b0eefba538eb024c4e00ec9adc348449d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/psmisc-23.4-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e02fc28d42912689b006fcc1e98bdb5b0eefba538eb024c4e00ec9adc348449d",
    ],
)

rpm(
    name = "python3-0__3.9.14-1.el9.aarch64",
    sha256 = "67453e7c4fef744a161127ff0ad799105239fa9ab4c899ccfa007cd23a9d8946",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.14-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/67453e7c4fef744a161127ff0ad799105239fa9ab4c899ccfa007cd23a9d8946",
    ],
)

rpm(
    name = "python3-0__3.9.14-1.el9.x86_64",
    sha256 = "b74d7479e9a96c30c9277d2787e2dad7dafde36362581feee700cdcb51fe2067",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.14-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b74d7479e9a96c30c9277d2787e2dad7dafde36362581feee700cdcb51fe2067",
    ],
)

rpm(
    name = "python3-libs-0__3.9.14-1.el9.aarch64",
    sha256 = "172c02d6f478069a59a5b71b88d6bb61edd0188c63571e9b00d24bc9ae155942",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.14-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/172c02d6f478069a59a5b71b88d6bb61edd0188c63571e9b00d24bc9ae155942",
    ],
)

rpm(
    name = "python3-libs-0__3.9.14-1.el9.x86_64",
    sha256 = "a20b887d181faa782f1513e15e8522ce299fa13856b83fc194947c8f41e5978d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.14-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a20b887d181faa782f1513e15e8522ce299fa13856b83fc194947c8f41e5978d",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.2.3-6.el9.aarch64",
    sha256 = "8e9e72535944204b48dbcb9cb34007b4991bdb4b5223e4c5874b07c6c122c1ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-21.2.3-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8e9e72535944204b48dbcb9cb34007b4991bdb4b5223e4c5874b07c6c122c1ff",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.2.3-6.el9.x86_64",
    sha256 = "8e9e72535944204b48dbcb9cb34007b4991bdb4b5223e4c5874b07c6c122c1ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-21.2.3-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8e9e72535944204b48dbcb9cb34007b4991bdb4b5223e4c5874b07c6c122c1ff",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-11.el9.aarch64",
    sha256 = "b923161167a7bab6fc9f235ebe4ae0f0344df9db6f1879dc9a52fd2c1efe2af5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-53.0.0-11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b923161167a7bab6fc9f235ebe4ae0f0344df9db6f1879dc9a52fd2c1efe2af5",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-11.el9.x86_64",
    sha256 = "b923161167a7bab6fc9f235ebe4ae0f0344df9db6f1879dc9a52fd2c1efe2af5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-53.0.0-11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b923161167a7bab6fc9f235ebe4ae0f0344df9db6f1879dc9a52fd2c1efe2af5",
    ],
)

rpm(
    name = "qemu-img-17__7.1.0-3.el9.aarch64",
    sha256 = "e98ab5f45a2f58c2c3d48cc6be4e72b59e0e0c8d5ef53c524c5cb1d086418c5e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-7.1.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e98ab5f45a2f58c2c3d48cc6be4e72b59e0e0c8d5ef53c524c5cb1d086418c5e",
    ],
)

rpm(
    name = "qemu-img-17__7.1.0-3.el9.x86_64",
    sha256 = "608889c274982d54128b57e18b2a769872d90c7edbfd7e425ff807e5af5ec798",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-7.1.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/608889c274982d54128b57e18b2a769872d90c7edbfd7e425ff807e5af5ec798",
    ],
)

rpm(
    name = "qemu-kvm-common-17__7.1.0-3.el9.aarch64",
    sha256 = "da349487c76a0d58604ee6e10c5cb37a19aabd667e7f27f0088eae1ca8b81383",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-7.1.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da349487c76a0d58604ee6e10c5cb37a19aabd667e7f27f0088eae1ca8b81383",
    ],
)

rpm(
    name = "qemu-kvm-common-17__7.1.0-3.el9.x86_64",
    sha256 = "d13c353b22151a34e2d023bcfcb22c37a3f008390769211190b1d511c05b21b6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-7.1.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d13c353b22151a34e2d023bcfcb22c37a3f008390769211190b1d511c05b21b6",
    ],
)

rpm(
    name = "qemu-kvm-core-17__7.1.0-3.el9.aarch64",
    sha256 = "135ad5fd4be10669cae6b61a4e11838f14c5a2b8851963dbe24cd8372770ed34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-7.1.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/135ad5fd4be10669cae6b61a4e11838f14c5a2b8851963dbe24cd8372770ed34",
    ],
)

rpm(
    name = "qemu-kvm-core-17__7.1.0-3.el9.x86_64",
    sha256 = "36b2384e8053b31a4d6d855f4d2547f0e14552dd10e213c844a2c036a4209120",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-7.1.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/36b2384e8053b31a4d6d855f4d2547f0e14552dd10e213c844a2c036a4209120",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__7.1.0-3.el9.x86_64",
    sha256 = "3daebfa32f5e15a5b36b3fc8886a34346a9b487a4971aaa6180394026c403198",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-7.1.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3daebfa32f5e15a5b36b3fc8886a34346a9b487a4971aaa6180394026c403198",
    ],
)

rpm(
    name = "quota-1__4.06-6.el9.x86_64",
    sha256 = "b4827d71208202beeecc6e661584b3cf008f2ee22ddd7250089dd94ff22be31e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/quota-4.06-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4827d71208202beeecc6e661584b3cf008f2ee22ddd7250089dd94ff22be31e",
    ],
)

rpm(
    name = "quota-nls-1__4.06-6.el9.x86_64",
    sha256 = "7a63c4fcc7166563de95bfffb23b54db2b17c8cef178f5c0887ac8f5ab8ec1e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/quota-nls-4.06-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7a63c4fcc7166563de95bfffb23b54db2b17c8cef178f5c0887ac8f5ab8ec1e3",
    ],
)

rpm(
    name = "readline-0__8.1-4.el9.aarch64",
    sha256 = "2ecec47a882ff434cc869b691a7e1e8d7639bc1af44bcb214ff4921f675776aa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/readline-8.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2ecec47a882ff434cc869b691a7e1e8d7639bc1af44bcb214ff4921f675776aa",
    ],
)

rpm(
    name = "readline-0__8.1-4.el9.x86_64",
    sha256 = "49945472925286ad89b0575657b43f9224777e36b442f0c88df67f0b61e26aee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/readline-8.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/49945472925286ad89b0575657b43f9224777e36b442f0c88df67f0b61e26aee",
    ],
)

rpm(
    name = "rpcbind-0__1.2.6-5.el9.x86_64",
    sha256 = "9ff0aa1299bb78f3c494620283cd34bcc9a1aa9f03fc902f21ba4c4c854b1e22",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpcbind-1.2.6-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ff0aa1299bb78f3c494620283cd34bcc9a1aa9f03fc902f21ba4c4c854b1e22",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-17.el9.aarch64",
    sha256 = "3f687889e490fb4afa71739e00de5a1f5a95ead799eeb6c23ba7a7ce65a9e84f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-4.16.1.3-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3f687889e490fb4afa71739e00de5a1f5a95ead799eeb6c23ba7a7ce65a9e84f",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-17.el9.x86_64",
    sha256 = "e43037bb16ee3b385ccee0c68d7a79698c8c1c732d27ccab36e5a7509070a230",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e43037bb16ee3b385ccee0c68d7a79698c8c1c732d27ccab36e5a7509070a230",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-17.el9.aarch64",
    sha256 = "1364c81db62f93d77d2763f4ebaaf390e9829e4a3a6d8eef0504fb670206a0fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.16.1.3-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1364c81db62f93d77d2763f4ebaaf390e9829e4a3a6d8eef0504fb670206a0fd",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-17.el9.x86_64",
    sha256 = "8d4603de9cf069d6a607621bbf797f3d92ead5479b0705502b552013e168c621",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8d4603de9cf069d6a607621bbf797f3d92ead5479b0705502b552013e168c621",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-17.el9.aarch64",
    sha256 = "eafc0e9bbb9f36ae802812ff33127b1e1162bd61f2750a49a334910899a27054",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.16.1.3-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eafc0e9bbb9f36ae802812ff33127b1e1162bd61f2750a49a334910899a27054",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-17.el9.x86_64",
    sha256 = "1e305142fe0d5f2aaff5a63215d569beb712e33ad41c1eed9092982c19fe2ee6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e305142fe0d5f2aaff5a63215d569beb712e33ad41c1eed9092982c19fe2ee6",
    ],
)

rpm(
    name = "seabios-0__1.16.0-4.el9.x86_64",
    sha256 = "bdd77d6d92f67506788389fa81d49f426273df69393b2e94e0bf0ea3652ccb92",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-1.16.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bdd77d6d92f67506788389fa81d49f426273df69393b2e94e0bf0ea3652ccb92",
    ],
)

rpm(
    name = "seabios-bin-0__1.16.0-4.el9.x86_64",
    sha256 = "a94713f14a127bef763c7ba3e411a759434bd683ef041a4f3727866283ec6207",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-bin-1.16.0-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a94713f14a127bef763c7ba3e411a759434bd683ef041a4f3727866283ec6207",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.16.0-4.el9.x86_64",
    sha256 = "6b2d1f32e1723581cea44232f69039b31f4d7bc80c915ff1878944d26f05d5d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seavgabios-bin-1.16.0-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6b2d1f32e1723581cea44232f69039b31f4d7bc80c915ff1878944d26f05d5d0",
    ],
)

rpm(
    name = "sed-0__4.8-9.el9.aarch64",
    sha256 = "cfdec0f026af984c11277ae613f16af7a86ea6170aac3da495a027599fdc8e3d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sed-4.8-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cfdec0f026af984c11277ae613f16af7a86ea6170aac3da495a027599fdc8e3d",
    ],
)

rpm(
    name = "sed-0__4.8-9.el9.x86_64",
    sha256 = "a2c5d9a7f569abb5a592df1c3aaff0441bf827c9d0e2df0ab42b6c443dbc475f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sed-4.8-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a2c5d9a7f569abb5a592df1c3aaff0441bf827c9d0e2df0ab42b6c443dbc475f",
    ],
)

rpm(
    name = "selinux-policy-0__34.1.44-1.el9.aarch64",
    sha256 = "fa987962c9caeb6cb8a4ee4e0098a58875a18ec05ecf19ebdf44d383865c3368",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-34.1.44-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/fa987962c9caeb6cb8a4ee4e0098a58875a18ec05ecf19ebdf44d383865c3368",
    ],
)

rpm(
    name = "selinux-policy-0__34.1.44-1.el9.x86_64",
    sha256 = "fa987962c9caeb6cb8a4ee4e0098a58875a18ec05ecf19ebdf44d383865c3368",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-34.1.44-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/fa987962c9caeb6cb8a4ee4e0098a58875a18ec05ecf19ebdf44d383865c3368",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__34.1.44-1.el9.aarch64",
    sha256 = "64b4f0e24bb6efe7d1636d4c229a654d85bab5baca31d64fadc5a95a44d33a7f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-34.1.44-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/64b4f0e24bb6efe7d1636d4c229a654d85bab5baca31d64fadc5a95a44d33a7f",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__34.1.44-1.el9.x86_64",
    sha256 = "64b4f0e24bb6efe7d1636d4c229a654d85bab5baca31d64fadc5a95a44d33a7f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-34.1.44-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/64b4f0e24bb6efe7d1636d4c229a654d85bab5baca31d64fadc5a95a44d33a7f",
    ],
)

rpm(
    name = "setup-0__2.13.7-7.el9.aarch64",
    sha256 = "ae0994cdf4ae34de6acb668f5672c77eaaa99be9b630cbc2dbe26c756b87790b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/setup-2.13.7-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ae0994cdf4ae34de6acb668f5672c77eaaa99be9b630cbc2dbe26c756b87790b",
    ],
)

rpm(
    name = "setup-0__2.13.7-7.el9.x86_64",
    sha256 = "ae0994cdf4ae34de6acb668f5672c77eaaa99be9b630cbc2dbe26c756b87790b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/setup-2.13.7-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ae0994cdf4ae34de6acb668f5672c77eaaa99be9b630cbc2dbe26c756b87790b",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-6.el9.aarch64",
    sha256 = "8d04bd9c627fdcfae1ff8a2ee8e4e75a6eb2391566a7cdfe89f6380c1cec06bf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.9-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8d04bd9c627fdcfae1ff8a2ee8e4e75a6eb2391566a7cdfe89f6380c1cec06bf",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-6.el9.x86_64",
    sha256 = "21eec2a59ddfe9976c24f8e5dcf8f8ffb4d565f4214325b88f32af935399bb93",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21eec2a59ddfe9976c24f8e5dcf8f8ffb4d565f4214325b88f32af935399bb93",
    ],
)

rpm(
    name = "snappy-0__1.1.8-8.el9.aarch64",
    sha256 = "02e5739b35acb3874546e98a8c182e1281f5a80604a550f05de2094c38c5e0d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/snappy-1.1.8-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/02e5739b35acb3874546e98a8c182e1281f5a80604a550f05de2094c38c5e0d7",
    ],
)

rpm(
    name = "snappy-0__1.1.8-8.el9.x86_64",
    sha256 = "10facee86b64af91b06292ca9892fd94fe5fc08c068b0baed6a0927d6a64955a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/snappy-1.1.8-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10facee86b64af91b06292ca9892fd94fe5fc08c068b0baed6a0927d6a64955a",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-5.el9.aarch64",
    sha256 = "7ebb88f2eb86ae915c1fc83f6983b8dca6348a6264c68166ee14541b99bfbcdc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.34.1-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7ebb88f2eb86ae915c1fc83f6983b8dca6348a6264c68166ee14541b99bfbcdc",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-5.el9.x86_64",
    sha256 = "420bf785149b4a852ae1b3259a4bc1fd22055998af26b042ad2f06deeb345ba3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/420bf785149b4a852ae1b3259a4bc1fd22055998af26b042ad2f06deeb345ba3",
    ],
)

rpm(
    name = "sssd-client-0__2.7.3-4.el9.aarch64",
    sha256 = "d1015a4d701bfb85c7273c2b81431cca597c8a51de5bcdda6db06d50eaf22c76",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.7.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d1015a4d701bfb85c7273c2b81431cca597c8a51de5bcdda6db06d50eaf22c76",
    ],
)

rpm(
    name = "sssd-client-0__2.7.3-4.el9.x86_64",
    sha256 = "9953d28bc9e80004a49d0c8cb150fd54f2fe5ee135da9d364990e49f21d103e4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.7.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9953d28bc9e80004a49d0c8cb150fd54f2fe5ee135da9d364990e49f21d103e4",
    ],
)

rpm(
    name = "swtpm-0__0.7.0-2.20211109gitb79fd91.el9.aarch64",
    sha256 = "629ce51263e1fc1499931161fe528b31bb636facfac3257e4f9e521de3592213",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-0.7.0-2.20211109gitb79fd91.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/629ce51263e1fc1499931161fe528b31bb636facfac3257e4f9e521de3592213",
    ],
)

rpm(
    name = "swtpm-0__0.7.0-2.20211109gitb79fd91.el9.x86_64",
    sha256 = "58e618362f6fd9b5efdfa27c1f5bb14b4a0c498f3751d5eb9f0153bcbc671024",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-0.7.0-2.20211109gitb79fd91.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/58e618362f6fd9b5efdfa27c1f5bb14b4a0c498f3751d5eb9f0153bcbc671024",
    ],
)

rpm(
    name = "swtpm-libs-0__0.7.0-2.20211109gitb79fd91.el9.aarch64",
    sha256 = "9985ed026c12bad4478d7db085581fee23058c5d71ff822c2f1cdefe4127f0e6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.7.0-2.20211109gitb79fd91.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9985ed026c12bad4478d7db085581fee23058c5d71ff822c2f1cdefe4127f0e6",
    ],
)

rpm(
    name = "swtpm-libs-0__0.7.0-2.20211109gitb79fd91.el9.x86_64",
    sha256 = "2d72d6e18a3feb7c66caa6c5296279ba9492111620839899b9342348d2eb4acb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.7.0-2.20211109gitb79fd91.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d72d6e18a3feb7c66caa6c5296279ba9492111620839899b9342348d2eb4acb",
    ],
)

rpm(
    name = "swtpm-tools-0__0.7.0-2.20211109gitb79fd91.el9.aarch64",
    sha256 = "2983e78edf61aac1a624d31a6867cb3ab04bfaed4e31da4e1aac9f14d704cfa8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.7.0-2.20211109gitb79fd91.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2983e78edf61aac1a624d31a6867cb3ab04bfaed4e31da4e1aac9f14d704cfa8",
    ],
)

rpm(
    name = "swtpm-tools-0__0.7.0-2.20211109gitb79fd91.el9.x86_64",
    sha256 = "607d390e8078b7d3fb2f65be7ea835708471c27ed320fce4cf7cca2de7174807",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.7.0-2.20211109gitb79fd91.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/607d390e8078b7d3fb2f65be7ea835708471c27ed320fce4cf7cca2de7174807",
    ],
)

rpm(
    name = "systemd-0__250-11.el9.aarch64",
    sha256 = "812fbb20eadb23bd6248e71bbfaaae3e9e30660587ed55b9bdffe10df760274d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-250-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/812fbb20eadb23bd6248e71bbfaaae3e9e30660587ed55b9bdffe10df760274d",
    ],
)

rpm(
    name = "systemd-0__250-11.el9.x86_64",
    sha256 = "c54925a4589445877aaa2f2fc7cd2ddd744951abf35b0e1a2f175dc7a9652455",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-250-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c54925a4589445877aaa2f2fc7cd2ddd744951abf35b0e1a2f175dc7a9652455",
    ],
)

rpm(
    name = "systemd-container-0__250-11.el9.aarch64",
    sha256 = "ca97c8787fb57525db49907127bb3b454bd5dba97beb81d4979f83561fc46bbe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-250-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ca97c8787fb57525db49907127bb3b454bd5dba97beb81d4979f83561fc46bbe",
    ],
)

rpm(
    name = "systemd-container-0__250-11.el9.x86_64",
    sha256 = "2fad2cf53c114121530885a19ca7cd0420ee8989f5c29686e5d106b695dead7d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-250-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2fad2cf53c114121530885a19ca7cd0420ee8989f5c29686e5d106b695dead7d",
    ],
)

rpm(
    name = "systemd-libs-0__250-11.el9.aarch64",
    sha256 = "729291dddd0a27b741e954116b9ab143e27c3535b08f625ea9925ff488d6d126",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-250-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/729291dddd0a27b741e954116b9ab143e27c3535b08f625ea9925ff488d6d126",
    ],
)

rpm(
    name = "systemd-libs-0__250-11.el9.x86_64",
    sha256 = "7327e0f30646679a55fce51cad27357bb8af4c217725441fa46e0f68df3649c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-250-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7327e0f30646679a55fce51cad27357bb8af4c217725441fa46e0f68df3649c3",
    ],
)

rpm(
    name = "systemd-pam-0__250-11.el9.aarch64",
    sha256 = "133fc234ac61d9d3efe417a66bdab02747441c9af3b88216ba7da889d929c1dd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-250-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/133fc234ac61d9d3efe417a66bdab02747441c9af3b88216ba7da889d929c1dd",
    ],
)

rpm(
    name = "systemd-pam-0__250-11.el9.x86_64",
    sha256 = "d672dd479a75b359ef65282a5559eb66a51288e62d9e5be094670bbe3ad4ab8d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-250-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d672dd479a75b359ef65282a5559eb66a51288e62d9e5be094670bbe3ad4ab8d",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__250-11.el9.aarch64",
    sha256 = "de785470787458bf69cc884d1fe39acf7fbba30f95c725ae48d9645ff19c9936",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-250-11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de785470787458bf69cc884d1fe39acf7fbba30f95c725ae48d9645ff19c9936",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__250-11.el9.x86_64",
    sha256 = "de785470787458bf69cc884d1fe39acf7fbba30f95c725ae48d9645ff19c9936",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-250-11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de785470787458bf69cc884d1fe39acf7fbba30f95c725ae48d9645ff19c9936",
    ],
)

rpm(
    name = "tar-2__1.34-5.el9.aarch64",
    sha256 = "d4fd778156c539f96cf2aacaaea9faf84a9bd8763bf89b6c54bfc37d1bb469c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tar-1.34-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d4fd778156c539f96cf2aacaaea9faf84a9bd8763bf89b6c54bfc37d1bb469c9",
    ],
)

rpm(
    name = "tar-2__1.34-5.el9.x86_64",
    sha256 = "b907cafd5fefcab9569d5e3c807ee00b0b2beea10d08260a951fdf537edf5c2f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tar-1.34-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b907cafd5fefcab9569d5e3c807ee00b0b2beea10d08260a951fdf537edf5c2f",
    ],
)

rpm(
    name = "tzdata-0__2022d-1.el9.aarch64",
    sha256 = "c2cff488ae306ec74e4201bd18257534b2b9214c9df6f96fa5e30e7494b23848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2022d-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c2cff488ae306ec74e4201bd18257534b2b9214c9df6f96fa5e30e7494b23848",
    ],
)

rpm(
    name = "tzdata-0__2022d-1.el9.x86_64",
    sha256 = "c2cff488ae306ec74e4201bd18257534b2b9214c9df6f96fa5e30e7494b23848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2022d-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c2cff488ae306ec74e4201bd18257534b2b9214c9df6f96fa5e30e7494b23848",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-2.el9.aarch64",
    sha256 = "2dbfa28c1818a193141b774c011db4b1b52ce672aece5f58fb0fb41c4a5fad8b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/unbound-libs-1.16.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2dbfa28c1818a193141b774c011db4b1b52ce672aece5f58fb0fb41c4a5fad8b",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-2.el9.x86_64",
    sha256 = "7b6dd4c3d907b3f2d2f5ab08ed76ee97638a2c2ebfb3a8abe4a905cb1092f23d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/unbound-libs-1.16.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7b6dd4c3d907b3f2d2f5ab08ed76ee97638a2c2ebfb3a8abe4a905cb1092f23d",
    ],
)

rpm(
    name = "usbredir-0__0.12.0-3.el9.x86_64",
    sha256 = "1edb414d18d8aa2cc0ee19a6d2712dd1ac98c19064e5b5e148ac45d190ddb8e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/usbredir-0.12.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1edb414d18d8aa2cc0ee19a6d2712dd1ac98c19064e5b5e148ac45d190ddb8e0",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-9.el9.aarch64",
    sha256 = "b731fe83b08f43336d436a2f6400aa6251171d1d9261fcca6ff5734460571729",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b731fe83b08f43336d436a2f6400aa6251171d1d9261fcca6ff5734460571729",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-9.el9.x86_64",
    sha256 = "3b3ae5007cbd3b14f3b9689a9a0d51752df9699c2f94b1cdf44a68d3621d8e05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b3ae5007cbd3b14f3b9689a9a0d51752df9699c2f94b1cdf44a68d3621d8e05",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-9.el9.aarch64",
    sha256 = "d5a6df418f446d3d983845d6155f137eca79cc55017faa8e9278fd2235a4ab12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.37.4-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d5a6df418f446d3d983845d6155f137eca79cc55017faa8e9278fd2235a4ab12",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-9.el9.x86_64",
    sha256 = "f426eee17734e73378b9326cd06f9d9ac14808b96078ea709da2abb632bf4c0c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.37.4-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f426eee17734e73378b9326cd06f9d9ac14808b96078ea709da2abb632bf4c0c",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-16.el9.aarch64",
    sha256 = "624e97558ee98889660ff17be2ea63d6246e275d4e0b0b1976627d87bd002d4e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.2.2637-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/624e97558ee98889660ff17be2ea63d6246e275d4e0b0b1976627d87bd002d4e",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-16.el9.x86_64",
    sha256 = "9fba13d288a8aa748f407e75ff610f6ac9e78295347f75284c849c44ab67bf44",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9fba13d288a8aa748f407e75ff610f6ac9e78295347f75284c849c44ab67bf44",
    ],
)

rpm(
    name = "virtiofsd-0__1.4.0-1.el9.aarch64",
    sha256 = "fb382cf3a872c10ed6fae781944d301b17702eb1a900f49b66a60807b38d7b78",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/virtiofsd-1.4.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fb382cf3a872c10ed6fae781944d301b17702eb1a900f49b66a60807b38d7b78",
    ],
)

rpm(
    name = "virtiofsd-0__1.4.0-1.el9.x86_64",
    sha256 = "75b5efa1ee4498aa24bcdae3bb5752fec4f4148a59947a0bd9d267bdf7a1fb0f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/virtiofsd-1.4.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/75b5efa1ee4498aa24bcdae3bb5752fec4f4148a59947a0bd9d267bdf7a1fb0f",
    ],
)

rpm(
    name = "which-0__2.21-28.el9.aarch64",
    sha256 = "cb0673e18b104ea7f039235c664e8357d1a667f4fdceff97874374e574a59fe2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/which-2.21-28.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb0673e18b104ea7f039235c664e8357d1a667f4fdceff97874374e574a59fe2",
    ],
)

rpm(
    name = "which-0__2.21-28.el9.x86_64",
    sha256 = "26730943b9a2550b0df8f17ef155efc3c3d966a711f2d5df0e351a5962369d82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/which-2.21-28.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/26730943b9a2550b0df8f17ef155efc3c3d966a711f2d5df0e351a5962369d82",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-4.el9.aarch64",
    sha256 = "6b51217704ff76b372e5405a21206c646d0aa28ef12b46a8bdb933ebcaf4b7f9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/xorriso-1.5.4-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6b51217704ff76b372e5405a21206c646d0aa28ef12b46a8bdb933ebcaf4b7f9",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-4.el9.x86_64",
    sha256 = "f5f6e99d32dbe9d2db413ef294083a59b0161710cd1fc2623bb5e94f0abc2062",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/xorriso-1.5.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f5f6e99d32dbe9d2db413ef294083a59b0161710cd1fc2623bb5e94f0abc2062",
    ],
)

rpm(
    name = "xz-0__5.2.5-8.el9.aarch64",
    sha256 = "c543b995056f118a141b499548ad00e566cc2062da2c36b2fc1e1b058c81dec1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/xz-5.2.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c543b995056f118a141b499548ad00e566cc2062da2c36b2fc1e1b058c81dec1",
    ],
)

rpm(
    name = "xz-0__5.2.5-8.el9.x86_64",
    sha256 = "159f0d11b5a78efa493b478b0c2df7ef42a54a9710b32dba9f94dd73eb333481",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/xz-5.2.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/159f0d11b5a78efa493b478b0c2df7ef42a54a9710b32dba9f94dd73eb333481",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-8.el9.aarch64",
    sha256 = "99784163a31515239be42e68608478b8337fd168cdb12bcba31de9dd78e35a25",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/xz-libs-5.2.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/99784163a31515239be42e68608478b8337fd168cdb12bcba31de9dd78e35a25",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-8.el9.x86_64",
    sha256 = "ff3c88297d75c51a5f8e9d2d69f8ad1eaf8347e20920b4335a3e0fc53269ad28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/xz-libs-5.2.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff3c88297d75c51a5f8e9d2d69f8ad1eaf8347e20920b4335a3e0fc53269ad28",
    ],
)

rpm(
    name = "yajl-0__2.1.0-21.el9.aarch64",
    sha256 = "e40aede8c85585cf816078ddca50d0678ace4d326c99fa4d5a96413173fe652a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/yajl-2.1.0-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e40aede8c85585cf816078ddca50d0678ace4d326c99fa4d5a96413173fe652a",
    ],
)

rpm(
    name = "yajl-0__2.1.0-21.el9.x86_64",
    sha256 = "d159334f408022942e77f67322288d13c1d575a3af54512d4310310709b644d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/yajl-2.1.0-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d159334f408022942e77f67322288d13c1d575a3af54512d4310310709b644d9",
    ],
)

rpm(
    name = "zlib-0__1.2.11-34.el9.aarch64",
    sha256 = "e36eec819a4e2e66b0e18727acbcbaa190957eab8741f6106a0cb46f8b46b1ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/zlib-1.2.11-34.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e36eec819a4e2e66b0e18727acbcbaa190957eab8741f6106a0cb46f8b46b1ae",
    ],
)

rpm(
    name = "zlib-0__1.2.11-34.el9.x86_64",
    sha256 = "648dc96d298cb7934620dbaedf6ea04d3f96d3b5fa398e96a1bd481be85f00a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/zlib-1.2.11-34.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/648dc96d298cb7934620dbaedf6ea04d3f96d3b5fa398e96a1bd481be85f00a3",
    ],
)
