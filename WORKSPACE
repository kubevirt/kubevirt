workspace(name = "kubevirt")

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
    sha256 = "80a98277ad1311dacd837f9b16db62887702e9f1d1c4c9f796d0121a46c8e184",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.46.0/rules_go-v0.46.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.46.0/rules_go-v0.46.0.zip",
        "https://storage.googleapis.com/builddeps/80a98277ad1311dacd837f9b16db62887702e9f1d1c4c9f796d0121a46c8e184",
    ],
)

# FIXME: added to make update of gazelle possible, see https://github.com/bazelbuild/bazel-gazelle/issues/1290#issuecomment-1312809060
http_archive(
    name = "bazel_skylib",
    sha256 = "c6966ec828da198c5d9adbaa94c05e3a1c7f21bd012a0b29ba8ddbccb2c93b0d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.1.1/bazel-skylib-1.1.1.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.1.1/bazel-skylib-1.1.1.tar.gz",
        "https://storage.googleapis.com/builddeps/c6966ec828da198c5d9adbaa94c05e3a1c7f21bd012a0b29ba8ddbccb2c93b0d",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "d3fa66a39028e97d76f9e2db8f1b0c11c099e8e01bf363a923074784e451f809",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.33.0/bazel-gazelle-v0.33.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.33.0/bazel-gazelle-v0.33.0.tar.gz",
        "https://storage.googleapis.com/builddeps/d3fa66a39028e97d76f9e2db8f1b0c11c099e8e01bf363a923074784e451f809",
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
        "https://storage.googleapis.com/builddeps/46fdbc00930c8dc9d84690b5bd94db6b4683b061199967d2cda1cfbda8f02c49",
    ],
)

# Disk images
http_file(
    name = "alpine_image",
    sha256 = "a90150589e493d5b7e87297056b6e124d8af1b91fa2eb92bab61a839839e287b",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.16/releases/x86_64/alpine-virt-3.16.3-x86_64.iso",
        "https://storage.googleapis.com/builddeps/a90150589e493d5b7e87297056b6e124d8af1b91fa2eb92bab61a839839e287b",
    ],
)

http_file(
    name = "alpine_image_aarch64",
    sha256 = "f3510fa675a6480a5f86b3325e97ca764368a8138d95fc4ba2efaebb41f8e325",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.16/releases/aarch64/alpine-virt-3.16.3-aarch64.iso",
        "https://storage.googleapis.com/builddeps/f3510fa675a6480a5f86b3325e97ca764368a8138d95fc4ba2efaebb41f8e325",
    ],
)

http_file(
    name = "alpine_image_s390x",
    sha256 = "6844e37c5f5bdbf34bc80cac6d504e1c450e799539a218fb4f1d625ad4afff54",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.16/releases/s390x/alpine-standard-3.16.3-s390x.iso",
        "https://storage.googleapis.com/builddeps/6844e37c5f5bdbf34bc80cac6d504e1c450e799539a218fb4f1d625ad4afff54",
    ],
)

http_file(
    name = "cirros_image",
    sha256 = "07e44a73e54c94d988028515403c1ed762055e01b83a767edf3c2b387f78ce00",
    urls = [
        "https://github.com/cirros-dev/cirros/releases/download/0.6.2/cirros-0.6.2-x86_64-disk.img",
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
    sha256 = "d5b5739cf297f0538d263e30678d5a09bba470a7c6bcbd8dff74e44153f16549",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.248-1/virtio-win-0.1.248.iso",
        "https://storage.googleapis.com/builddeps/d5b5739cf297f0538d263e30678d5a09bba470a7c6bcbd8dff74e44153f16549",
    ],
)

http_archive(
    name = "bazeldnf",
    sha256 = "fb24d80ad9edad0f7bd3000e8cffcfbba89cc07e495c47a7d3b1f803bd527a40",
    urls = [
        "https://github.com/rmohr/bazeldnf/releases/download/v0.5.9/bazeldnf-v0.5.9.tar.gz",
        "https://storage.googleapis.com/builddeps/fb24d80ad9edad0f7bd3000e8cffcfbba89cc07e495c47a7d3b1f803bd527a40",
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
    go_version = "1.22.2",
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
# rules_docker 0.16 uses 0.1.4, let's grab by commit
go_repository(
    name = "com_github_google_go_containerregistry",
    commit = "8a2841911ffee4f6892ca0083e89752fb46c48dd",  # v0.1.4
    importpath = "github.com/google/go-containerregistry",
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

# Pull go_image_base
container_pull(
    name = "go_image_base",
    digest = "sha256:1f957e6333e5df53de1b470837789ffa8c6e4c375ffbd5caddd40b22a226b68d",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
)

container_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:02e08b836ad99a1de187da19278c5058dc6bf2c62b857d313b6076e0c68c5099",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
)

container_pull(
    name = "go_image_base_s390x",
    digest = "sha256:b1b4713bcd4b9a6d6bf15a8039e87dc8ed3792b158feeb6eab0281539f3e2d73",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
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
    digest = "sha256:85f7bb99624422dc45b2c203d520a06bfce7a760ef831aafbf0e2bf2b92ebcf4",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
)

container_pull(
    name = "alpine_with_test_tooling",
    digest = "sha256:4a6c258a75cff2190d768ab06e57dbf375bedb260ce4ba79dd249f077e769dc5",
    registry = "quay.io",
    repository = "kubevirtci/alpine-with-test-tooling-container-disk",
    tag = "2404181910-1c58677",
)

container_pull(
    name = "fedora_with_test_tooling_aarch64",
    digest = "sha256:f5bcb56c8c3ce6f0801aa897db4691950235e7676d1ae22c64b088def4196701",
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

http_archive(
    name = "libguestfs-appliance",
    sha256 = "124d6325a799e958843be4818ef2c32661755be1c56e519665779948861b04f6",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz",
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
    name = "acl-0__2.3.1-4.el9.aarch64",
    sha256 = "a0a9b302d252d32c0da8100a0ad762852c22eeac4ccad0aaf72ad68a2bbd7a93",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/acl-2.3.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a0a9b302d252d32c0da8100a0ad762852c22eeac4ccad0aaf72ad68a2bbd7a93",
    ],
)

rpm(
    name = "acl-0__2.3.1-4.el9.s390x",
    sha256 = "5d12a3e157b07244a7c0546905af864148730e982ac7ceaa4b0bf287dd7ae669",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/acl-2.3.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5d12a3e157b07244a7c0546905af864148730e982ac7ceaa4b0bf287dd7ae669",
    ],
)

rpm(
    name = "acl-0__2.3.1-4.el9.x86_64",
    sha256 = "dd11bab2ea0abdfa310362eace871422a003340bf223135626500f8f5a985f6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/acl-2.3.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd11bab2ea0abdfa310362eace871422a003340bf223135626500f8f5a985f6b",
    ],
)

rpm(
    name = "adobe-source-code-pro-fonts-0__2.030.1.050-12.el9.1.x86_64",
    sha256 = "9e6aa0c60204bb4b152ce541ca3a9f5c28b020ed551dd417d3936a8b2153f0df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/adobe-source-code-pro-fonts-2.030.1.050-12.el9.1.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9e6aa0c60204bb4b152ce541ca3a9f5c28b020ed551dd417d3936a8b2153f0df",
    ],
)

rpm(
    name = "alternatives-0__1.24-1.el9.aarch64",
    sha256 = "a9bba5fd3731426733609e996881cddb0775e979091fab91a3878178a63c7656",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/alternatives-1.24-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a9bba5fd3731426733609e996881cddb0775e979091fab91a3878178a63c7656",
    ],
)

rpm(
    name = "alternatives-0__1.24-1.el9.s390x",
    sha256 = "009eeff2a85e9682beb3d576e2a2359c83efa71371464e6021e9b4e92f32af36",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/alternatives-1.24-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/009eeff2a85e9682beb3d576e2a2359c83efa71371464e6021e9b4e92f32af36",
    ],
)

rpm(
    name = "alternatives-0__1.24-1.el9.x86_64",
    sha256 = "b58e7ea30c27ecb321d9a279b95b62aef59d92173714fce859bfb359ee231ff3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/alternatives-1.24-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b58e7ea30c27ecb321d9a279b95b62aef59d92173714fce859bfb359ee231ff3",
    ],
)

rpm(
    name = "audit-libs-0__3.1.2-2.el9.aarch64",
    sha256 = "4f8080dd299fb57f78bb4093367766a1b19d9121723f1f942f3593515f76cbba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/audit-libs-3.1.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4f8080dd299fb57f78bb4093367766a1b19d9121723f1f942f3593515f76cbba",
    ],
)

rpm(
    name = "audit-libs-0__3.1.2-2.el9.s390x",
    sha256 = "ed1effa0c36fa606f88769f7c36839eeeee51e6ef5116859f031e6cde4d509c8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/audit-libs-3.1.2-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ed1effa0c36fa606f88769f7c36839eeeee51e6ef5116859f031e6cde4d509c8",
    ],
)

rpm(
    name = "audit-libs-0__3.1.2-2.el9.x86_64",
    sha256 = "de7efacf4bf377b96e9420b0f05905dae407c3f52e18cd2acc5879919b4c401e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.1.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de7efacf4bf377b96e9420b0f05905dae407c3f52e18cd2acc5879919b4c401e",
    ],
)

rpm(
    name = "augeas-libs-0__1.13.0-5.el9.x86_64",
    sha256 = "d169ce09a7637372981eb379e619a1824181ca7201a768cb14e208895db2a22e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/augeas-libs-1.13.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d169ce09a7637372981eb379e619a1824181ca7201a768cb14e208895db2a22e",
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
    name = "basesystem-0__11-13.el9.s390x",
    sha256 = "a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/basesystem-11-13.el9.noarch.rpm",
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
    name = "bash-0__5.1.8-9.el9.aarch64",
    sha256 = "acb782e8dacd2f3efb25d0b8b1b64c59b8a60a84fc86a4fca88ede1affc68f4c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bash-5.1.8-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/acb782e8dacd2f3efb25d0b8b1b64c59b8a60a84fc86a4fca88ede1affc68f4c",
    ],
)

rpm(
    name = "bash-0__5.1.8-9.el9.s390x",
    sha256 = "7f69429a343d53be5f3390e0e6032869c33cf1e9e344ee1448a4ec2998dc9d9e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bash-5.1.8-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7f69429a343d53be5f3390e0e6032869c33cf1e9e344ee1448a4ec2998dc9d9e",
    ],
)

rpm(
    name = "bash-0__5.1.8-9.el9.x86_64",
    sha256 = "823859a9e8fad83004fa0d9f698ff223f6f7d38fd8e7629509d98b5ba6764c03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bash-5.1.8-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/823859a9e8fad83004fa0d9f698ff223f6f7d38fd8e7629509d98b5ba6764c03",
    ],
)

rpm(
    name = "binutils-0__2.35.2-43.el9.aarch64",
    sha256 = "6814f37a59d417143be795b8435df1127ed0a71e86d74d00120007f694f085a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6814f37a59d417143be795b8435df1127ed0a71e86d74d00120007f694f085a2",
    ],
)

rpm(
    name = "binutils-0__2.35.2-43.el9.s390x",
    sha256 = "20249b3a49f8b3d039b40ff2dcda4a873aaf7b5a836b4d45016a4babd29f5af2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-2.35.2-43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/20249b3a49f8b3d039b40ff2dcda4a873aaf7b5a836b4d45016a4babd29f5af2",
    ],
)

rpm(
    name = "binutils-0__2.35.2-43.el9.x86_64",
    sha256 = "a19fd351796769822371877393ff2179296530d83d0ddb9df663bdb9cacba725",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a19fd351796769822371877393ff2179296530d83d0ddb9df663bdb9cacba725",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-43.el9.aarch64",
    sha256 = "3a9bf2079f8dcac20e45e24afdb065761c09ef7eaf7c8005f55f6211180f23a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3a9bf2079f8dcac20e45e24afdb065761c09ef7eaf7c8005f55f6211180f23a3",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-43.el9.s390x",
    sha256 = "9ab6b33796b9e757ce3c771ee006b1a507a325b8822f8bb2fc1338b96f323a25",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-gold-2.35.2-43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9ab6b33796b9e757ce3c771ee006b1a507a325b8822f8bb2fc1338b96f323a25",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-43.el9.x86_64",
    sha256 = "00545632eebd19e1399fde103a44bbc5c8cabc25e1cb8ba1259230a7ae0d528a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/00545632eebd19e1399fde103a44bbc5c8cabc25e1cb8ba1259230a7ae0d528a",
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
    name = "bzip2-0__1.0.8-8.el9.s390x",
    sha256 = "1c41489cf61cbf6ba54994da5cd7d6bb796d4f3b91fcc83c89692ddcffda3f0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-1.0.8-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1c41489cf61cbf6ba54994da5cd7d6bb796d4f3b91fcc83c89692ddcffda3f0a",
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
    name = "bzip2-libs-0__1.0.8-8.el9.s390x",
    sha256 = "187c9275d53ddd209339a5ae7ae7af4d2c80647a054197f7abe39c020a66262d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-libs-1.0.8-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/187c9275d53ddd209339a5ae7ae7af4d2c80647a054197f7abe39c020a66262d",
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
    name = "ca-certificates-0__2023.2.60_v7.0.306-90.1.el9.aarch64",
    sha256 = "76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ca-certificates-2023.2.60_v7.0.306-90.1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
    ],
)

rpm(
    name = "ca-certificates-0__2023.2.60_v7.0.306-90.1.el9.s390x",
    sha256 = "76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ca-certificates-2023.2.60_v7.0.306-90.1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
    ],
)

rpm(
    name = "ca-certificates-0__2023.2.60_v7.0.306-90.1.el9.x86_64",
    sha256 = "76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ca-certificates-2023.2.60_v7.0.306-90.1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
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
    name = "capstone-0__4.0.2-10.el9.s390x",
    sha256 = "1110f472053cbfaa31ff98c2722c147ac2d9f006fded91d1987ea8d114f3ce0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/capstone-4.0.2-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1110f472053cbfaa31ff98c2722c147ac2d9f006fded91d1987ea8d114f3ce0a",
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
    name = "centos-gpg-keys-0__9.0-26.el9.aarch64",
    sha256 = "8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-26.el9.s390x",
    sha256 = "8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-gpg-keys-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-26.el9.x86_64",
    sha256 = "8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-26.el9.aarch64",
    sha256 = "3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-release-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-26.el9.s390x",
    sha256 = "3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-release-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-26.el9.x86_64",
    sha256 = "3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-release-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-26.el9.aarch64",
    sha256 = "eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-26.el9.s390x",
    sha256 = "eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-repos-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-26.el9.x86_64",
    sha256 = "eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-35.el9.aarch64",
    sha256 = "bb97628ca49734e508c72631251b9a6737f47d85961a75f99c7ddebdf121443f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.32-35.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bb97628ca49734e508c72631251b9a6737f47d85961a75f99c7ddebdf121443f",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-35.el9.s390x",
    sha256 = "4751affb09bc84155f12caa36692a6f5c3733f10dd6bea40d59bfa2f892226e8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/coreutils-single-8.32-35.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4751affb09bc84155f12caa36692a6f5c3733f10dd6bea40d59bfa2f892226e8",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-35.el9.x86_64",
    sha256 = "09c1828de7a4a5c787eaa43204cdaa8d5ec52d7b11a29f911c2e960c554c19b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.32-35.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09c1828de7a4a5c787eaa43204cdaa8d5ec52d7b11a29f911c2e960c554c19b4",
    ],
)

rpm(
    name = "cpp-0__11.4.1-3.el9.aarch64",
    sha256 = "1865699638f72cde70d12c96159db6b32e78faf2601247c667e428a39cb12606",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1865699638f72cde70d12c96159db6b32e78faf2601247c667e428a39cb12606",
    ],
)

rpm(
    name = "cpp-0__11.4.1-3.el9.s390x",
    sha256 = "065ff09e92b5e99f051081ebefa419e1f4ad5bd71bd301d1cee275873ddd3cf7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/cpp-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/065ff09e92b5e99f051081ebefa419e1f4ad5bd71bd301d1cee275873ddd3cf7",
    ],
)

rpm(
    name = "cpp-0__11.4.1-3.el9.x86_64",
    sha256 = "5915741a629c68c912b9b77fc5449e5e49fd2f16af816c8d80d5c4d17f171f01",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.4.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5915741a629c68c912b9b77fc5449e5e49fd2f16af816c8d80d5c4d17f171f01",
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
    name = "cracklib-0__2.9.6-27.el9.s390x",
    sha256 = "f090c83e4fa8e5d170aaf13fe5c7795213d9d2ac0af16f92c60d6425a7b23253",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cracklib-2.9.6-27.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f090c83e4fa8e5d170aaf13fe5c7795213d9d2ac0af16f92c60d6425a7b23253",
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
    name = "cracklib-dicts-0__2.9.6-27.el9.s390x",
    sha256 = "bac458a7a96be0b856d6c3294c5675fa159694d111fae63819f0a70dc3c6ccf0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cracklib-dicts-2.9.6-27.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bac458a7a96be0b856d6c3294c5675fa159694d111fae63819f0a70dc3c6ccf0",
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
    name = "crypto-policies-0__20240304-1.gitb1c706d.el9.aarch64",
    sha256 = "6f9a5fd1f60651b62471dd44b4870c409d20b8090280c5b7f231f00029b25b2c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20240304-1.gitb1c706d.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6f9a5fd1f60651b62471dd44b4870c409d20b8090280c5b7f231f00029b25b2c",
    ],
)

rpm(
    name = "crypto-policies-0__20240304-1.gitb1c706d.el9.s390x",
    sha256 = "6f9a5fd1f60651b62471dd44b4870c409d20b8090280c5b7f231f00029b25b2c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/crypto-policies-20240304-1.gitb1c706d.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6f9a5fd1f60651b62471dd44b4870c409d20b8090280c5b7f231f00029b25b2c",
    ],
)

rpm(
    name = "crypto-policies-0__20240304-1.gitb1c706d.el9.x86_64",
    sha256 = "6f9a5fd1f60651b62471dd44b4870c409d20b8090280c5b7f231f00029b25b2c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20240304-1.gitb1c706d.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6f9a5fd1f60651b62471dd44b4870c409d20b8090280c5b7f231f00029b25b2c",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-29.el9.aarch64",
    sha256 = "231cf035ef2ee6565e39bf84c4f6875b87efb84fa24079612a1d94ccb10d45bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/curl-minimal-7.76.1-29.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/231cf035ef2ee6565e39bf84c4f6875b87efb84fa24079612a1d94ccb10d45bd",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-29.el9.s390x",
    sha256 = "80e821b9f6ea0f5dfdf5bdb54c56913231803e1fd7d20904a4ed34bb1fa04d55",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/curl-minimal-7.76.1-29.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/80e821b9f6ea0f5dfdf5bdb54c56913231803e1fd7d20904a4ed34bb1fa04d55",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-29.el9.x86_64",
    sha256 = "7f3dfe4424246d072b294410ab5c240a3084419bfaa0d385cb40aa06ff7eef07",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/curl-minimal-7.76.1-29.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f3dfe4424246d072b294410ab5c240a3084419bfaa0d385cb40aa06ff7eef07",
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
    name = "cyrus-sasl-0__2.1.27-21.el9.s390x",
    sha256 = "ac2ea6ebbf2af219770155a193e044c02dfcc8ee738497858834af63e2087999",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cyrus-sasl-2.1.27-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ac2ea6ebbf2af219770155a193e044c02dfcc8ee738497858834af63e2087999",
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
    name = "cyrus-sasl-gssapi-0__2.1.27-21.el9.s390x",
    sha256 = "0c9badb44b1c126966382c2016fb3a28e93c79046992656b643b59ff628b306d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cyrus-sasl-gssapi-2.1.27-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0c9badb44b1c126966382c2016fb3a28e93c79046992656b643b59ff628b306d",
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
    name = "cyrus-sasl-lib-0__2.1.27-21.el9.s390x",
    sha256 = "e8954c3d19fc3aa905d09488c111df37bd5b9fe9c1eeec314420b3be2e75a74f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cyrus-sasl-lib-2.1.27-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e8954c3d19fc3aa905d09488c111df37bd5b9fe9c1eeec314420b3be2e75a74f",
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
    name = "daxctl-libs-0__71.1-8.el9.x86_64",
    sha256 = "95bbf4ffb69cebc022fe3a2b35b828978d47e5b016747197ed5be34a57712432",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/daxctl-libs-71.1-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95bbf4ffb69cebc022fe3a2b35b828978d47e5b016747197ed5be34a57712432",
    ],
)

rpm(
    name = "dbus-1__1.12.20-8.el9.aarch64",
    sha256 = "29c244f31d9f3ae910a6b95d4d5534cdf1ea4870fc277e29876a10cf3bd193ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-1.12.20-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/29c244f31d9f3ae910a6b95d4d5534cdf1ea4870fc277e29876a10cf3bd193ae",
    ],
)

rpm(
    name = "dbus-1__1.12.20-8.el9.s390x",
    sha256 = "a99d278716899bb35100d4c9c26a66a795d309555d8d71ef6d1739e2f44cf44d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-1.12.20-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a99d278716899bb35100d4c9c26a66a795d309555d8d71ef6d1739e2f44cf44d",
    ],
)

rpm(
    name = "dbus-1__1.12.20-8.el9.x86_64",
    sha256 = "d13d52df79bb9a0a1795530a5ce1134c9c92a2a7c401dfc3827ee8bf02f60018",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-1.12.20-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d13d52df79bb9a0a1795530a5ce1134c9c92a2a7c401dfc3827ee8bf02f60018",
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
    name = "dbus-broker-0__28-7.el9.s390x",
    sha256 = "d38a5ae851f9006000c3cd7a37310f901a02864e0272d7284c4f2db1efcd61ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-broker-28-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d38a5ae851f9006000c3cd7a37310f901a02864e0272d7284c4f2db1efcd61ff",
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
    name = "dbus-common-1__1.12.20-8.el9.aarch64",
    sha256 = "ff91286d9413256c50886a0c96b3d5d0773bd25284b9a94b28b98a5215f09a56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.20-8.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ff91286d9413256c50886a0c96b3d5d0773bd25284b9a94b28b98a5215f09a56",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-8.el9.s390x",
    sha256 = "ff91286d9413256c50886a0c96b3d5d0773bd25284b9a94b28b98a5215f09a56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-common-1.12.20-8.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ff91286d9413256c50886a0c96b3d5d0773bd25284b9a94b28b98a5215f09a56",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-8.el9.x86_64",
    sha256 = "ff91286d9413256c50886a0c96b3d5d0773bd25284b9a94b28b98a5215f09a56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.20-8.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ff91286d9413256c50886a0c96b3d5d0773bd25284b9a94b28b98a5215f09a56",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-8.el9.aarch64",
    sha256 = "4f9a0d0712363aaee565b9883560de7b0afd7f8ffdc5f8584afadc1623ff1897",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.12.20-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4f9a0d0712363aaee565b9883560de7b0afd7f8ffdc5f8584afadc1623ff1897",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-8.el9.s390x",
    sha256 = "03174ea3bd7d525a263d23fbd5c797acff256d3f01ca75d58b2558c561a2e472",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-libs-1.12.20-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/03174ea3bd7d525a263d23fbd5c797acff256d3f01ca75d58b2558c561a2e472",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-8.el9.x86_64",
    sha256 = "2d46aaa0b1e8032d10156b040a5226b5a90ef000d8d85d40fd5671379a5bc904",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.20-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d46aaa0b1e8032d10156b040a5226b5a90ef000d8d85d40fd5671379a5bc904",
    ],
)

rpm(
    name = "device-mapper-9__1.02.197-2.el9.aarch64",
    sha256 = "41d78b6baddb35ed9c4f3a587e22fa36fa5a4ab2caf84af5b5522395e4ecc56f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.197-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/41d78b6baddb35ed9c4f3a587e22fa36fa5a4ab2caf84af5b5522395e4ecc56f",
    ],
)

rpm(
    name = "device-mapper-9__1.02.197-2.el9.s390x",
    sha256 = "cccd913bccae53bd584f731add538528929005025680cbc13a996a323939f795",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-1.02.197-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cccd913bccae53bd584f731add538528929005025680cbc13a996a323939f795",
    ],
)

rpm(
    name = "device-mapper-9__1.02.197-2.el9.x86_64",
    sha256 = "22c465b0c6f812f1fd8234fdda187c8cc72f63dcd310f07e82ea0f6a7ae19fff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.197-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/22c465b0c6f812f1fd8234fdda187c8cc72f63dcd310f07e82ea0f6a7ae19fff",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.197-2.el9.aarch64",
    sha256 = "bf0ed60ed623fd8aacdca50ada4bc0a77b430edbe00157810dac0717c0b11183",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.197-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bf0ed60ed623fd8aacdca50ada4bc0a77b430edbe00157810dac0717c0b11183",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.197-2.el9.s390x",
    sha256 = "29d1bcb2c2c2ec57c5c6757401cac262a4b5dfa7395aba658ace056390666703",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-libs-1.02.197-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29d1bcb2c2c2ec57c5c6757401cac262a4b5dfa7395aba658ace056390666703",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.197-2.el9.x86_64",
    sha256 = "86cac18e0b3dbb6fec783d4bee8f1d4ca1074412442a0f89fe5622d2650c01ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.197-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86cac18e0b3dbb6fec783d4bee8f1d4ca1074412442a0f89fe5622d2650c01ca",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-29.el9.aarch64",
    sha256 = "65b58d990e5eafa9cee8a788fd3abb548c4d500901ea8b7e9f8ca092c97afe5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.7-29.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/65b58d990e5eafa9cee8a788fd3abb548c4d500901ea8b7e9f8ca092c97afe5f",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-29.el9.x86_64",
    sha256 = "f365b0dd5b55cdae4af746c87e7d16a4d87d6031bb22e6ca17627d1a976ff7fc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.7-29.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f365b0dd5b55cdae4af746c87e7d16a4d87d6031bb22e6ca17627d1a976ff7fc",
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
    name = "diffutils-0__3.7-12.el9.s390x",
    sha256 = "e0f62f72c6d24e0507fa16c23bb74ece2704aabfb902c3649c57dad090f0c1ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/diffutils-3.7-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e0f62f72c6d24e0507fa16c23bb74ece2704aabfb902c3649c57dad090f0c1ae",
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
    name = "e2fsprogs-0__1.46.5-5.el9.aarch64",
    sha256 = "2c1d0878dbe3725b1c9a2769955b39774a4eaaa2d635e4ac940fea2f1fd8c8b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.46.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2c1d0878dbe3725b1c9a2769955b39774a4eaaa2d635e4ac940fea2f1fd8c8b1",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-5.el9.s390x",
    sha256 = "60aa9e7ed851d1e4649f26c61581829c8a492f41056aa9def47428ce2a6022c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/e2fsprogs-1.46.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/60aa9e7ed851d1e4649f26c61581829c8a492f41056aa9def47428ce2a6022c4",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-5.el9.x86_64",
    sha256 = "4a17a32ec4efaecead6a6bcea2a189510faad44bf2a189191cd62f2922d3f888",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.46.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a17a32ec4efaecead6a6bcea2a189510faad44bf2a189191cd62f2922d3f888",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-5.el9.aarch64",
    sha256 = "2ce6e23afee9c67bf29067d47a4257280c7d466614a3d466e050e27ca86b769d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.46.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2ce6e23afee9c67bf29067d47a4257280c7d466614a3d466e050e27ca86b769d",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-5.el9.s390x",
    sha256 = "ef696a3e6ae1a50a6cd26a8fbb6acd0728ab6798cf5665b3729f8ba73382f2f8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/e2fsprogs-libs-1.46.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ef696a3e6ae1a50a6cd26a8fbb6acd0728ab6798cf5665b3729f8ba73382f2f8",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-5.el9.x86_64",
    sha256 = "0ef6a9f568898557765b2e65e099c31308bdafcb5f2d8712054975b24858ad5a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.46.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0ef6a9f568898557765b2e65e099c31308bdafcb5f2d8712054975b24858ad5a",
    ],
)

rpm(
    name = "edk2-aarch64-0__20231122-6.el9.aarch64",
    sha256 = "1b7bdee31471b63891434bb8e0662c2bf39659aa2502be7fd3be612bc66f81bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20231122-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1b7bdee31471b63891434bb8e0662c2bf39659aa2502be7fd3be612bc66f81bd",
    ],
)

rpm(
    name = "edk2-ovmf-0__20231122-6.el9.x86_64",
    sha256 = "e3964f8009223b6e23b3054a7c969fe51cd175b12a6ca27119dca322c8fecd09",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20231122-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e3964f8009223b6e23b3054a7c969fe51cd175b12a6ca27119dca322c8fecd09",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.191-4.el9.aarch64",
    sha256 = "3967a0107ec76487b868f9d2bed81467c0b4cb4bd079cd82118f039cb585aa9e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.191-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3967a0107ec76487b868f9d2bed81467c0b4cb4bd079cd82118f039cb585aa9e",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.191-4.el9.s390x",
    sha256 = "fbae2a1221489d5924bc6c78343fce8dd18ed4c92ecc939e978068382395a344",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.191-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fbae2a1221489d5924bc6c78343fce8dd18ed4c92ecc939e978068382395a344",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.191-4.el9.x86_64",
    sha256 = "6b69fb1c9ecac14f1e7fa27f5f520beb32e32a4f89fc07918b9d27e9e3f114e2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.191-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6b69fb1c9ecac14f1e7fa27f5f520beb32e32a4f89fc07918b9d27e9e3f114e2",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.191-4.el9.aarch64",
    sha256 = "4eb8d70db7988a546afee6f315fa582f6becd38468aa1e2a6dd2eb3bca1316f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.191-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4eb8d70db7988a546afee6f315fa582f6becd38468aa1e2a6dd2eb3bca1316f4",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.191-4.el9.s390x",
    sha256 = "4eb8d70db7988a546afee6f315fa582f6becd38468aa1e2a6dd2eb3bca1316f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.191-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4eb8d70db7988a546afee6f315fa582f6becd38468aa1e2a6dd2eb3bca1316f4",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.191-4.el9.x86_64",
    sha256 = "4eb8d70db7988a546afee6f315fa582f6becd38468aa1e2a6dd2eb3bca1316f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.191-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4eb8d70db7988a546afee6f315fa582f6becd38468aa1e2a6dd2eb3bca1316f4",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.191-4.el9.aarch64",
    sha256 = "ca9026bf4ead9ad6de1679c94969a17e557dcd75ca9d54fa2364883ec63b8279",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.191-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ca9026bf4ead9ad6de1679c94969a17e557dcd75ca9d54fa2364883ec63b8279",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.191-4.el9.s390x",
    sha256 = "0fa5ca4f46c139eb5c5e3b985c87ebe3df39e2db08d8723fe7f98260676bb46c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.191-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0fa5ca4f46c139eb5c5e3b985c87ebe3df39e2db08d8723fe7f98260676bb46c",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.191-4.el9.x86_64",
    sha256 = "31be9f30d161d8bfab5730e3f1878bd60b77848c943748affd3fca45480ecf33",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.191-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31be9f30d161d8bfab5730e3f1878bd60b77848c943748affd3fca45480ecf33",
    ],
)

rpm(
    name = "elfutils-libs-0__0.191-4.el9.aarch64",
    sha256 = "a8c816daf1b5027e1f041bc9c04ae595d960a8b06f7dd88518cd8f0afee3f1d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.191-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a8c816daf1b5027e1f041bc9c04ae595d960a8b06f7dd88518cd8f0afee3f1d0",
    ],
)

rpm(
    name = "elfutils-libs-0__0.191-4.el9.s390x",
    sha256 = "4ffe4beb0bb0b86602d0f88ef79a4b6dc21b861b4e60d12b728f4489a84fbd0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.191-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4ffe4beb0bb0b86602d0f88ef79a4b6dc21b861b4e60d12b728f4489a84fbd0d",
    ],
)

rpm(
    name = "elfutils-libs-0__0.191-4.el9.x86_64",
    sha256 = "54997a36ad27d92c2f27b88a79a3edeb65a1fcdd4cb5dd4c25a1e4f1e53adff7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.191-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54997a36ad27d92c2f27b88a79a3edeb65a1fcdd4cb5dd4c25a1e4f1e53adff7",
    ],
)

rpm(
    name = "ethtool-2__6.2-1.el9.aarch64",
    sha256 = "9f086c7b6796d5749f5f93f727cbe380c9d04f54c968b5555db2763bace23e6a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ethtool-6.2-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9f086c7b6796d5749f5f93f727cbe380c9d04f54c968b5555db2763bace23e6a",
    ],
)

rpm(
    name = "ethtool-2__6.2-1.el9.s390x",
    sha256 = "dc269dcde0a48bb19d8d5fe30a5716ce2e43b2602aaa3a8c2fec5513f9da1a13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ethtool-6.2-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/dc269dcde0a48bb19d8d5fe30a5716ce2e43b2602aaa3a8c2fec5513f9da1a13",
    ],
)

rpm(
    name = "ethtool-2__6.2-1.el9.x86_64",
    sha256 = "bc4b58cbda4ce3eb8795aa35db56c6e7d52a53f48b3c197d0ee83911f5d3eadc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ethtool-6.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc4b58cbda4ce3eb8795aa35db56c6e7d52a53f48b3c197d0ee83911f5d3eadc",
    ],
)

rpm(
    name = "expat-0__2.5.0-2.el9.aarch64",
    sha256 = "cc115aa6a973d7d7c5516c45a961a96ed149bf37349e51ab5d0d9a87b935b890",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/expat-2.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cc115aa6a973d7d7c5516c45a961a96ed149bf37349e51ab5d0d9a87b935b890",
    ],
)

rpm(
    name = "expat-0__2.5.0-2.el9.s390x",
    sha256 = "30a6753dc782e0044c547ae9da6a3be7db12e23e147508dffdeea7180e3ffe02",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/expat-2.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/30a6753dc782e0044c547ae9da6a3be7db12e23e147508dffdeea7180e3ffe02",
    ],
)

rpm(
    name = "expat-0__2.5.0-2.el9.x86_64",
    sha256 = "00580404a4c3f59a32d7b9ae513ff48d7bbaa65d3a9bf9193dc33c9668dae6b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.5.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/00580404a4c3f59a32d7b9ae513ff48d7bbaa65d3a9bf9193dc33c9668dae6b4",
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
    name = "filesystem-0__3.16-2.el9.s390x",
    sha256 = "d7964cb337adb860a737828759fbd1c64a2ab2d2eb62411aa66525ece2ed62af",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/filesystem-3.16-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d7964cb337adb860a737828759fbd1c64a2ab2d2eb62411aa66525ece2ed62af",
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
    name = "findutils-1__4.8.0-6.el9.aarch64",
    sha256 = "c552d76e062fdb098b57eef1f8f6566a72354d441a101e6238e4b8ba43dfb77e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/findutils-4.8.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c552d76e062fdb098b57eef1f8f6566a72354d441a101e6238e4b8ba43dfb77e",
    ],
)

rpm(
    name = "findutils-1__4.8.0-6.el9.s390x",
    sha256 = "d2e41410f2fc2fb1369e46e487524e8dfc07adaa189e6f849c3586fca1c1b822",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/findutils-4.8.0-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d2e41410f2fc2fb1369e46e487524e8dfc07adaa189e6f849c3586fca1c1b822",
    ],
)

rpm(
    name = "findutils-1__4.8.0-6.el9.x86_64",
    sha256 = "2634379fd4f1c42d0ea733e3006dd4f76c2a9144fce69c8e99e4d50b71c4fb13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/findutils-4.8.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2634379fd4f1c42d0ea733e3006dd4f76c2a9144fce69c8e99e4d50b71c4fb13",
    ],
)

rpm(
    name = "fonts-filesystem-1__2.0.5-7.el9.1.x86_64",
    sha256 = "c79fa96aa7fb447975497dd50c94002ee73d01171343f8ee14032d06adb58a92",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fonts-filesystem-2.0.5-7.el9.1.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c79fa96aa7fb447975497dd50c94002ee73d01171343f8ee14032d06adb58a92",
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
    name = "fuse-common-0__3.10.2-8.el9.x86_64",
    sha256 = "a370c1adf4718f5172a812b72471cc5fcaa69825441e443193f30db31ecde54b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-common-3.10.2-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a370c1adf4718f5172a812b72471cc5fcaa69825441e443193f30db31ecde54b",
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
    name = "fuse-libs-0__2.9.9-15.el9.s390x",
    sha256 = "c72647810ee5826d0523f597af1f17b64905918f40b6ec20cabc57eed8ca6dc6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/fuse-libs-2.9.9-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c72647810ee5826d0523f597af1f17b64905918f40b6ec20cabc57eed8ca6dc6",
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
    name = "gawk-0__5.1.0-6.el9.s390x",
    sha256 = "acad833571094a674d4073b4e747e15d373e3a8b06a7e7e8aecfec6fd4860c0e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gawk-5.1.0-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/acad833571094a674d4073b4e747e15d373e3a8b06a7e7e8aecfec6fd4860c0e",
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
    name = "gcc-0__11.4.1-3.el9.aarch64",
    sha256 = "8e5a4214b660d082b1df639d8f0f54f6f20aed9e33f6606f3e02150a9e55567f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8e5a4214b660d082b1df639d8f0f54f6f20aed9e33f6606f3e02150a9e55567f",
    ],
)

rpm(
    name = "gcc-0__11.4.1-3.el9.s390x",
    sha256 = "c13d594abaab48fe182b8af0446f4b1d3e4848d21b7f5ce8e4d1a7521467bd35",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gcc-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c13d594abaab48fe182b8af0446f4b1d3e4848d21b7f5ce8e4d1a7521467bd35",
    ],
)

rpm(
    name = "gcc-0__11.4.1-3.el9.x86_64",
    sha256 = "5c53248130a7ce726f4cd15b98f394aff37bf4ec4e5be310de8fb2157d5ac372",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.4.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5c53248130a7ce726f4cd15b98f394aff37bf4ec4e5be310de8fb2157d5ac372",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-1.el9.aarch64",
    sha256 = "69754627d810b252c6202f2ef8765ca39b9c8a0b0fd6da0325a9e492dbf88f96",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gdbm-libs-1.23-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/69754627d810b252c6202f2ef8765ca39b9c8a0b0fd6da0325a9e492dbf88f96",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-1.el9.s390x",
    sha256 = "29c9ab72536be72b9c78285ef12117633cf3e2dfd18757bcf7587cd94eb9e055",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gdbm-libs-1.23-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29c9ab72536be72b9c78285ef12117633cf3e2dfd18757bcf7587cd94eb9e055",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-1.el9.x86_64",
    sha256 = "cada66331cc07a4f8a0701fc1ad13c346913a0d6f913e35c0257a68b6a1e6ce0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.23-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cada66331cc07a4f8a0701fc1ad13c346913a0d6f913e35c0257a68b6a1e6ce0",
    ],
)

rpm(
    name = "gettext-0__0.21-8.el9.aarch64",
    sha256 = "66387c45fa58eea0120e0cdfa27ffb2ca4eda1cb9f157be7af23503f4b42fdab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gettext-0.21-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/66387c45fa58eea0120e0cdfa27ffb2ca4eda1cb9f157be7af23503f4b42fdab",
    ],
)

rpm(
    name = "gettext-0__0.21-8.el9.s390x",
    sha256 = "369ef71c5a7c3337079cf9a25647dc1835a35a99ed3bbb3a028dbd49366db910",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gettext-0.21-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/369ef71c5a7c3337079cf9a25647dc1835a35a99ed3bbb3a028dbd49366db910",
    ],
)

rpm(
    name = "gettext-0__0.21-8.el9.x86_64",
    sha256 = "1f1f79d426dd3d6c3c39a45fa9af8bbf37e2547a50136b7c30b76c1bfe5a487f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gettext-0.21-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f1f79d426dd3d6c3c39a45fa9af8bbf37e2547a50136b7c30b76c1bfe5a487f",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-8.el9.aarch64",
    sha256 = "f979fa61b8cb97a3f26dec4844a3ad978cf85a85e9ccccac8f0698c04c7849dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gettext-libs-0.21-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f979fa61b8cb97a3f26dec4844a3ad978cf85a85e9ccccac8f0698c04c7849dc",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-8.el9.s390x",
    sha256 = "d55003d65db061381fa5ab04e16049451ead0d15ec5b19ac87269c453c50987f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gettext-libs-0.21-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d55003d65db061381fa5ab04e16049451ead0d15ec5b19ac87269c453c50987f",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-8.el9.x86_64",
    sha256 = "5a1780e9d485c014b95802531aecd7bf8593daa0af24646a74ab335cddfb40fa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gettext-libs-0.21-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a1780e9d485c014b95802531aecd7bf8593daa0af24646a74ab335cddfb40fa",
    ],
)

rpm(
    name = "glib-networking-0__2.68.3-3.el9.x86_64",
    sha256 = "ea106ccc142daf5016626cfe5c4f0a2d97e700ae7ad4780835e899897b63317f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib-networking-2.68.3-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ea106ccc142daf5016626cfe5c4f0a2d97e700ae7ad4780835e899897b63317f",
    ],
)

rpm(
    name = "glib2-0__2.68.4-15.el9.aarch64",
    sha256 = "a2baecdb746f3312a5b37d4bc139182195b31eb4a1cb16b9b84db00a1e640042",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glib2-2.68.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a2baecdb746f3312a5b37d4bc139182195b31eb4a1cb16b9b84db00a1e640042",
    ],
)

rpm(
    name = "glib2-0__2.68.4-15.el9.s390x",
    sha256 = "34a706adcd651397a5f6681c4cb4973aac48df23cf00c9457946d1f264376960",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glib2-2.68.4-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/34a706adcd651397a5f6681c4cb4973aac48df23cf00c9457946d1f264376960",
    ],
)

rpm(
    name = "glib2-0__2.68.4-15.el9.x86_64",
    sha256 = "85f10ac062e7c1dc1c4becd0f5287f7e5e0f17267e139a59cfb5eabf4632713e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/85f10ac062e7c1dc1c4becd0f5287f7e5e0f17267e139a59cfb5eabf4632713e",
    ],
)

rpm(
    name = "glibc-0__2.34-110.el9.aarch64",
    sha256 = "441bd24f0530064f0d3e5c512b171c62adf210db99a5a37ab3e7109bcd61400b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-110.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/441bd24f0530064f0d3e5c512b171c62adf210db99a5a37ab3e7109bcd61400b",
    ],
)

rpm(
    name = "glibc-0__2.34-110.el9.s390x",
    sha256 = "cf798e89071fc59b8e700f52b7421f33250d550b8e718c0f405366b85cc9139f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-2.34-110.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cf798e89071fc59b8e700f52b7421f33250d550b8e718c0f405366b85cc9139f",
    ],
)

rpm(
    name = "glibc-0__2.34-110.el9.x86_64",
    sha256 = "ed5dc6cc8809b4b58b99846663abcf1fd6a8dcee244bea35c8344b7a6f3a9454",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-110.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed5dc6cc8809b4b58b99846663abcf1fd6a8dcee244bea35c8344b7a6f3a9454",
    ],
)

rpm(
    name = "glibc-common-0__2.34-110.el9.aarch64",
    sha256 = "7f8a0b655f608337ee9f8ab76b39129a0f9d78b069f144a55ceae0e504189e5e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-110.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7f8a0b655f608337ee9f8ab76b39129a0f9d78b069f144a55ceae0e504189e5e",
    ],
)

rpm(
    name = "glibc-common-0__2.34-110.el9.s390x",
    sha256 = "db9560eadbb065d6de4a32d8605e7ea426a599ad3b64b24bf1821673345c5d88",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-common-2.34-110.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/db9560eadbb065d6de4a32d8605e7ea426a599ad3b64b24bf1821673345c5d88",
    ],
)

rpm(
    name = "glibc-common-0__2.34-110.el9.x86_64",
    sha256 = "5b58f16d8467584f254b37b975ce09af99f691738bef8a4dc16afb9d2903ff03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-110.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5b58f16d8467584f254b37b975ce09af99f691738bef8a4dc16afb9d2903ff03",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-110.el9.aarch64",
    sha256 = "74e6a8535c5d45e5d7d56268d0d3b7be2c5d7510b712980ade31dc47021e5f63",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-110.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/74e6a8535c5d45e5d7d56268d0d3b7be2c5d7510b712980ade31dc47021e5f63",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-110.el9.s390x",
    sha256 = "01f921a96e909002795231fbde1b192cab5a8cbc4e8913d9164c771e100e0c2c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-devel-2.34-110.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/01f921a96e909002795231fbde1b192cab5a8cbc4e8913d9164c771e100e0c2c",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-110.el9.x86_64",
    sha256 = "9ad5e484481f85d504830495cf3d51026d35e25315728b31585ff4e4dd84de5e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-110.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ad5e484481f85d504830495cf3d51026d35e25315728b31585ff4e4dd84de5e",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-110.el9.s390x",
    sha256 = "c71f05dbcb1236d90bdbe9f07326777956984f916f01f0f5ffc63c53ad7d6a58",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-headers-2.34-110.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c71f05dbcb1236d90bdbe9f07326777956984f916f01f0f5ffc63c53ad7d6a58",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-110.el9.x86_64",
    sha256 = "5a5fc3a35f63aa6d79ae5f62549783a5cc4f81971889d0f27de6d2af30d1fd6c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-110.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a5fc3a35f63aa6d79ae5f62549783a5cc4f81971889d0f27de6d2af30d1fd6c",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-110.el9.aarch64",
    sha256 = "405a09cb1a981b128c7abc73d37d7eac6d5100f0e4161d95ace2053cef34f139",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-110.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/405a09cb1a981b128c7abc73d37d7eac6d5100f0e4161d95ace2053cef34f139",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-110.el9.s390x",
    sha256 = "b5acbda9c7cf69265826685e4e73b2ecd5f52570163d361135b923cd9ddcd8eb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.34-110.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b5acbda9c7cf69265826685e4e73b2ecd5f52570163d361135b923cd9ddcd8eb",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-110.el9.x86_64",
    sha256 = "e821c4e960f0f8f6282b928a8ef7be4732e797ee45d467e38e550ca1e6ea38ea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-110.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e821c4e960f0f8f6282b928a8ef7be4732e797ee45d467e38e550ca1e6ea38ea",
    ],
)

rpm(
    name = "glibc-static-0__2.34-110.el9.aarch64",
    sha256 = "314009a74e56726fc4dc4098092ae1f9c1488da82170c38b89236a379cc6a0aa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-110.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/314009a74e56726fc4dc4098092ae1f9c1488da82170c38b89236a379cc6a0aa",
    ],
)

rpm(
    name = "glibc-static-0__2.34-110.el9.s390x",
    sha256 = "fb0af58800123714cd7fe0fc15ae5010d8068b673f32948165893466dd7935a9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/glibc-static-2.34-110.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fb0af58800123714cd7fe0fc15ae5010d8068b673f32948165893466dd7935a9",
    ],
)

rpm(
    name = "glibc-static-0__2.34-110.el9.x86_64",
    sha256 = "92babc44c7d8305a3ea98ac70e475b57058438b07c4cb4161a8205a082b8c43e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-110.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/92babc44c7d8305a3ea98ac70e475b57058438b07c4cb4161a8205a082b8c43e",
    ],
)

rpm(
    name = "gmp-1__6.2.0-13.el9.aarch64",
    sha256 = "01716c2de2af5ddce80cfc2f81fbcabe50670583f8d3ebf8af1058982edb9c70",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gmp-6.2.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01716c2de2af5ddce80cfc2f81fbcabe50670583f8d3ebf8af1058982edb9c70",
    ],
)

rpm(
    name = "gmp-1__6.2.0-13.el9.s390x",
    sha256 = "c26b4f2d1e2c6a9a3b683d1909df8f788a261fcc8e766ded00a96681e5dc62d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gmp-6.2.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c26b4f2d1e2c6a9a3b683d1909df8f788a261fcc8e766ded00a96681e5dc62d2",
    ],
)

rpm(
    name = "gmp-1__6.2.0-13.el9.x86_64",
    sha256 = "b6d592895ccc0fcad6106cd41800cd9d68e5384c418e53a2c3ff2ac8c8b15a33",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gmp-6.2.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b6d592895ccc0fcad6106cd41800cd9d68e5384c418e53a2c3ff2ac8c8b15a33",
    ],
)

rpm(
    name = "gnupg2-0__2.3.3-4.el9.x86_64",
    sha256 = "03e7697ffc0ae9301c30adccfe28d3b100063e5d2c7c5f87dc21f1c56af4052f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnupg2-2.3.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/03e7697ffc0ae9301c30adccfe28d3b100063e5d2c7c5f87dc21f1c56af4052f",
    ],
)

rpm(
    name = "gnutls-0__3.8.3-4.el9.aarch64",
    sha256 = "c7c658c2f2364f4fcbc056f3059c3a4f8a8fa5db3a34a56bbab8386e9f1a9ac5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gnutls-3.8.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7c658c2f2364f4fcbc056f3059c3a4f8a8fa5db3a34a56bbab8386e9f1a9ac5",
    ],
)

rpm(
    name = "gnutls-0__3.8.3-4.el9.s390x",
    sha256 = "f71b6727e720d44781702bb37815cddbe7f0aab173174bbc7f555d88ca00160e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gnutls-3.8.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f71b6727e720d44781702bb37815cddbe7f0aab173174bbc7f555d88ca00160e",
    ],
)

rpm(
    name = "gnutls-0__3.8.3-4.el9.x86_64",
    sha256 = "91e1e46e6f315445e715184237a69f4152359efa1a9ae54cc0524b9616d0741f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnutls-3.8.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/91e1e46e6f315445e715184237a69f4152359efa1a9ae54cc0524b9616d0741f",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.3-4.el9.aarch64",
    sha256 = "1da81d1b7550757a406c75ea2812a27dcedf8ddcfcec2064e5d493ec579c137b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.8.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1da81d1b7550757a406c75ea2812a27dcedf8ddcfcec2064e5d493ec579c137b",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.3-4.el9.s390x",
    sha256 = "bc1efead7fc64a01cf80fccbaab0022ca646c4f27fad84fcf5fb0d93937d24a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-dane-3.8.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bc1efead7fc64a01cf80fccbaab0022ca646c4f27fad84fcf5fb0d93937d24a3",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.3-4.el9.x86_64",
    sha256 = "de16c064ac6f4650a90038aa63e1a84244345fecc22f2dbdf6d4645b055348eb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.8.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de16c064ac6f4650a90038aa63e1a84244345fecc22f2dbdf6d4645b055348eb",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.3-4.el9.aarch64",
    sha256 = "818c74ec8584b3df3c4d8ee191e03c9dcb39651a39afed2a53d728d6efec9e47",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.8.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/818c74ec8584b3df3c4d8ee191e03c9dcb39651a39afed2a53d728d6efec9e47",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.3-4.el9.s390x",
    sha256 = "2001bb380a91122ccfd6bdcf694fa0cff5ed4b537f13a2ca48acdd9f7967b481",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-utils-3.8.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2001bb380a91122ccfd6bdcf694fa0cff5ed4b537f13a2ca48acdd9f7967b481",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.3-4.el9.x86_64",
    sha256 = "84a2279a7e01190c3c0bcdf7f98b8268ed8c744a4c9d3af728dc84ef4f0e2f9c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.8.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84a2279a7e01190c3c0bcdf7f98b8268ed8c744a4c9d3af728dc84ef4f0e2f9c",
    ],
)

rpm(
    name = "gobject-introspection-0__1.68.0-11.el9.aarch64",
    sha256 = "bcb5e3ab1d0ee579a11ec1449585196c0d13b552f73bbea3e2ada642b5313fbd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gobject-introspection-1.68.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bcb5e3ab1d0ee579a11ec1449585196c0d13b552f73bbea3e2ada642b5313fbd",
    ],
)

rpm(
    name = "gobject-introspection-0__1.68.0-11.el9.s390x",
    sha256 = "27ff550b5596d6a8ae414c20b42c20aba8f37794372fd19ddce5270a6e0d0328",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gobject-introspection-1.68.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/27ff550b5596d6a8ae414c20b42c20aba8f37794372fd19ddce5270a6e0d0328",
    ],
)

rpm(
    name = "gobject-introspection-0__1.68.0-11.el9.x86_64",
    sha256 = "d75cc220f9b5978bb1755cf5e4de30244ff8e7ad7f98dfbdfe897f41442e4587",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gobject-introspection-1.68.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d75cc220f9b5978bb1755cf5e4de30244ff8e7ad7f98dfbdfe897f41442e4587",
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
    name = "grep-0__3.6-5.el9.s390x",
    sha256 = "b6b83738fc6afb9ba28d0c2c57eaf17cdbe5b26ff89a8da17812dd261045df3e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/grep-3.6-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b6b83738fc6afb9ba28d0c2c57eaf17cdbe5b26ff89a8da17812dd261045df3e",
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
    name = "gsettings-desktop-schemas-0__40.0-6.el9.x86_64",
    sha256 = "9935991dc0dfb2eda15db01d388d4a018ee3aaf0c5f8ffa4ca1297f05d62db33",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gsettings-desktop-schemas-40.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9935991dc0dfb2eda15db01d388d4a018ee3aaf0c5f8ffa4ca1297f05d62db33",
    ],
)

rpm(
    name = "gssproxy-0__0.8.4-6.el9.x86_64",
    sha256 = "19dfec5fba0c719f8768adfeb63c9cdca5856264a237d9235172cef2aa8eeebc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gssproxy-0.8.4-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19dfec5fba0c719f8768adfeb63c9cdca5856264a237d9235172cef2aa8eeebc",
    ],
)

rpm(
    name = "guestfs-tools-0__1.51.6-2.el9.x86_64",
    sha256 = "89d9baf05d69c29b88934ceec40ec7655120be802ffe23367153830a4759a6c7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.51.6-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89d9baf05d69c29b88934ceec40ec7655120be802ffe23367153830a4759a6c7",
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
    name = "gzip-0__1.12-1.el9.s390x",
    sha256 = "72b8b818027d9d716be069743c03431f057ce5af62b38273c249990890cbc504",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gzip-1.12-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/72b8b818027d9d716be069743c03431f057ce5af62b38273c249990890cbc504",
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
    name = "hwdata-0__0.348-9.13.el9.x86_64",
    sha256 = "bc60591db3be73ae7119e05bd065b86e639e59848f6d71f31e13a94c822a9743",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bc60591db3be73ae7119e05bd065b86e639e59848f6d71f31e13a94c822a9743",
    ],
)

rpm(
    name = "iproute-0__6.2.0-5.el9.aarch64",
    sha256 = "56f200e0a4edcc6865573c6c043be407001d6fe6dfbc43c5053ee9c7d2e5d9da",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-6.2.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56f200e0a4edcc6865573c6c043be407001d6fe6dfbc43c5053ee9c7d2e5d9da",
    ],
)

rpm(
    name = "iproute-0__6.2.0-5.el9.s390x",
    sha256 = "06f7215c5e30a8f2e32732205338528ecac2e4ada17f2beb282eb09e17f2a2b7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iproute-6.2.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/06f7215c5e30a8f2e32732205338528ecac2e4ada17f2beb282eb09e17f2a2b7",
    ],
)

rpm(
    name = "iproute-0__6.2.0-5.el9.x86_64",
    sha256 = "d7656f7f1694cec4ea6747bb745a4949cd2a24b037663ce02f5027454163b215",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-6.2.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d7656f7f1694cec4ea6747bb745a4949cd2a24b037663ce02f5027454163b215",
    ],
)

rpm(
    name = "iproute-tc-0__6.2.0-5.el9.aarch64",
    sha256 = "7b44f9e06333070bb339a2af71e26ec3f9c257ff363b4efcb41a4ec3bbeffaad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-tc-6.2.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7b44f9e06333070bb339a2af71e26ec3f9c257ff363b4efcb41a4ec3bbeffaad",
    ],
)

rpm(
    name = "iproute-tc-0__6.2.0-5.el9.s390x",
    sha256 = "e4bcf382404383419959656c579a5efc86c5cb18622ec6629ca08279ad81a494",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iproute-tc-6.2.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e4bcf382404383419959656c579a5efc86c5cb18622ec6629ca08279ad81a494",
    ],
)

rpm(
    name = "iproute-tc-0__6.2.0-5.el9.x86_64",
    sha256 = "f142c4f355288c3583b2d0fabe71ab447e8c4f03a4b3754c411b21e4e26c8f2b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.2.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f142c4f355288c3583b2d0fabe71ab447e8c4f03a4b3754c411b21e4e26c8f2b",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-2.el9.aarch64",
    sha256 = "4a21b9b6cbb5143ba88554acf05ba11dfc95a8e0dc4d665f76da2b2452722eaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.10-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4a21b9b6cbb5143ba88554acf05ba11dfc95a8e0dc4d665f76da2b2452722eaa",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-2.el9.s390x",
    sha256 = "1061d7da311f2c6ba29f97cf5b742e48c9778a96790a1fd09c4d2e6e603ab547",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iptables-libs-1.8.10-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1061d7da311f2c6ba29f97cf5b742e48c9778a96790a1fd09c4d2e6e603ab547",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-2.el9.x86_64",
    sha256 = "a6ed138f68e6083633a5d7ecfcfaa21f7dbdb0afd4c16e5752f423dfcb9497d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.10-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a6ed138f68e6083633a5d7ecfcfaa21f7dbdb0afd4c16e5752f423dfcb9497d4",
    ],
)

rpm(
    name = "iputils-0__20210202-9.el9.aarch64",
    sha256 = "8adc7b856fb98c268e9f50c572e27dc687fe41d1ea4ff47b1788d2fea90e2b7d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iputils-20210202-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8adc7b856fb98c268e9f50c572e27dc687fe41d1ea4ff47b1788d2fea90e2b7d",
    ],
)

rpm(
    name = "iputils-0__20210202-9.el9.s390x",
    sha256 = "5c8e673794918dfcf335b61f23b1ac772855bd4cd344233164f0401e86cb58b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iputils-20210202-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c8e673794918dfcf335b61f23b1ac772855bd4cd344233164f0401e86cb58b1",
    ],
)

rpm(
    name = "iputils-0__20210202-9.el9.x86_64",
    sha256 = "9712d85fb1bebbf74dfda41c81f670bf7eb3d548569ce394b8cb46ca0a18cfd9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iputils-20210202-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9712d85fb1bebbf74dfda41c81f670bf7eb3d548569ce394b8cb46ca0a18cfd9",
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
    name = "jansson-0__2.14-1.el9.s390x",
    sha256 = "ec1863fd2bd9672ecb0bd4f77d929dad04f253330a41307300f485ae13d017e5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/jansson-2.14-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ec1863fd2bd9672ecb0bd4f77d929dad04f253330a41307300f485ae13d017e5",
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
    name = "json-glib-0__1.6.6-1.el9.aarch64",
    sha256 = "04a7348a546a972f275a4de34373ad7a937a5a93f4c868dffa47daa31a226243",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/json-glib-1.6.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/04a7348a546a972f275a4de34373ad7a937a5a93f4c868dffa47daa31a226243",
    ],
)

rpm(
    name = "json-glib-0__1.6.6-1.el9.s390x",
    sha256 = "5cdd9c06afe511d378bcfad5624ec79ae27b154ca2de67f1073404381891fc79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/json-glib-1.6.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5cdd9c06afe511d378bcfad5624ec79ae27b154ca2de67f1073404381891fc79",
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
    name = "kernel-headers-0__5.14.0-460.el9.aarch64",
    sha256 = "2ab5fdd663dcb1ff40c584ad4d81a86b601400a7bd23cc86dc30c0a8ad6fe753",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/kernel-headers-5.14.0-460.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2ab5fdd663dcb1ff40c584ad4d81a86b601400a7bd23cc86dc30c0a8ad6fe753",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-460.el9.s390x",
    sha256 = "de7f359fdc2c38397ad371f62caa3c738eca85c4a12d76146718556ac71bd555",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/kernel-headers-5.14.0-460.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/de7f359fdc2c38397ad371f62caa3c738eca85c4a12d76146718556ac71bd555",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-460.el9.x86_64",
    sha256 = "be4631dc828074019eb5437d909fcc16d134ff5a0e93389212c71f2379e90068",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/kernel-headers-5.14.0-460.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be4631dc828074019eb5437d909fcc16d134ff5a0e93389212c71f2379e90068",
    ],
)

rpm(
    name = "keyutils-0__1.6.3-1.el9.x86_64",
    sha256 = "bc9b6262006e7722b7936e3d1e5079d7281f96e161bcd0aa93328564a32984bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/keyutils-1.6.3-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc9b6262006e7722b7936e3d1e5079d7281f96e161bcd0aa93328564a32984bb",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-1.el9.aarch64",
    sha256 = "5d97ee3ed28533eb2ea01a6be97696fbbbc72f8178dcf7f1acf30e674a298a6e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/keyutils-libs-1.6.3-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5d97ee3ed28533eb2ea01a6be97696fbbbc72f8178dcf7f1acf30e674a298a6e",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-1.el9.s390x",
    sha256 = "954b22cc636f29363edc7a29c24cb05039929ca71780174b8ec4dc495af314ef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/keyutils-libs-1.6.3-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/954b22cc636f29363edc7a29c24cb05039929ca71780174b8ec4dc495af314ef",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-1.el9.x86_64",
    sha256 = "aef982501694486a27411c68698886d76ec70c5cd10bfe619501e7e4c36f50a9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/keyutils-libs-1.6.3-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aef982501694486a27411c68698886d76ec70c5cd10bfe619501e7e4c36f50a9",
    ],
)

rpm(
    name = "kmod-0__28-9.el9.x86_64",
    sha256 = "0c3073304639c87da92a7217fc61a778595a72c24ea999f941e3cb608f70aad5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-28-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c3073304639c87da92a7217fc61a778595a72c24ea999f941e3cb608f70aad5",
    ],
)

rpm(
    name = "kmod-libs-0__28-9.el9.aarch64",
    sha256 = "0e51fa74611d31585fb4e665fc4b24b0ff300821d109b3e0116ccdfc54c04789",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/kmod-libs-28-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0e51fa74611d31585fb4e665fc4b24b0ff300821d109b3e0116ccdfc54c04789",
    ],
)

rpm(
    name = "kmod-libs-0__28-9.el9.s390x",
    sha256 = "88bc8e78b7d093f24177d5939384878142c3b4bcd9c0983e7f8b493f5d3a87ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/kmod-libs-28-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/88bc8e78b7d093f24177d5939384878142c3b4bcd9c0983e7f8b493f5d3a87ee",
    ],
)

rpm(
    name = "kmod-libs-0__28-9.el9.x86_64",
    sha256 = "319957f8f3abe9b05b4aca442a3c633b36c8974e2dbd87f31ec66885f66e1b88",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-libs-28-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/319957f8f3abe9b05b4aca442a3c633b36c8974e2dbd87f31ec66885f66e1b88",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-2.el9.aarch64",
    sha256 = "a804b46c2957a2038690fed34500b034864e2532867ca92ef1b5087abb15e370",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.1-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a804b46c2957a2038690fed34500b034864e2532867ca92ef1b5087abb15e370",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-2.el9.s390x",
    sha256 = "1ff7d6becee47e3e1609ed9e0069c291ca2994ed26c2d41148640b54a01e36e4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/krb5-libs-1.21.1-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1ff7d6becee47e3e1609ed9e0069c291ca2994ed26c2d41148640b54a01e36e4",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-2.el9.x86_64",
    sha256 = "657fc4530c69dd4b54a94eefeabb64549245bed1fad446bf095338c7dceabc8b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/657fc4530c69dd4b54a94eefeabb64549245bed1fad446bf095338c7dceabc8b",
    ],
)

rpm(
    name = "less-0__590-3.el9.x86_64",
    sha256 = "e2734a6cdb7851fba2641895439c540ff21662367930a1da05332b3d63bb8687",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/less-590-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e2734a6cdb7851fba2641895439c540ff21662367930a1da05332b3d63bb8687",
    ],
)

rpm(
    name = "libacl-0__2.3.1-4.el9.aarch64",
    sha256 = "90e4392e312cd793eeba4cd68bd12836a882ac37356c784806d67a0cd1d48c25",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libacl-2.3.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/90e4392e312cd793eeba4cd68bd12836a882ac37356c784806d67a0cd1d48c25",
    ],
)

rpm(
    name = "libacl-0__2.3.1-4.el9.s390x",
    sha256 = "bfdd2316c1742032df9b15d1a91ff2e3674faeae1e27e4a851165e5c6bb666f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libacl-2.3.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bfdd2316c1742032df9b15d1a91ff2e3674faeae1e27e4a851165e5c6bb666f5",
    ],
)

rpm(
    name = "libacl-0__2.3.1-4.el9.x86_64",
    sha256 = "60a3affaa1c387fd6f72dd65aa7ad619a1830947823abb4b29e7b9fcb4c9d27c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libacl-2.3.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/60a3affaa1c387fd6f72dd65aa7ad619a1830947823abb4b29e7b9fcb4c9d27c",
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
    name = "libaio-0__0.3.111-13.el9.s390x",
    sha256 = "b4adecd95273b4ae7590b84ecbed5a7b4a1795066bab430d15f04eb82bb9dc1c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libaio-0.3.111-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b4adecd95273b4ae7590b84ecbed5a7b4a1795066bab430d15f04eb82bb9dc1c",
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
    name = "libarchive-0__3.5.3-4.el9.aarch64",
    sha256 = "c043954972a8dea0b6cf5d3092c1eee90bb48b3fcb7cedf30aa861dc1d3f402c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libarchive-3.5.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c043954972a8dea0b6cf5d3092c1eee90bb48b3fcb7cedf30aa861dc1d3f402c",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-4.el9.s390x",
    sha256 = "f95a05acd33d6f63a43ac2b065c45a3d2c9ef1923ec80d3a33946501dde0e751",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libarchive-3.5.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f95a05acd33d6f63a43ac2b065c45a3d2c9ef1923ec80d3a33946501dde0e751",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-4.el9.x86_64",
    sha256 = "4c53176eafd8c449aef704b8fbc2d5401bb7d2ea0a67961956f318f2e9a2c7a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libarchive-3.5.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c53176eafd8c449aef704b8fbc2d5401bb7d2ea0a67961956f318f2e9a2c7a4",
    ],
)

rpm(
    name = "libasan-0__11.4.1-3.el9.aarch64",
    sha256 = "b723559a50f76c340558db7ba9fa7c3e966b161175309f1fdc073fff8b8dae3e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b723559a50f76c340558db7ba9fa7c3e966b161175309f1fdc073fff8b8dae3e",
    ],
)

rpm(
    name = "libasan-0__11.4.1-3.el9.s390x",
    sha256 = "2c8ade930bf0f7b07cf388afa18f3a19e6fc800585aba6998eead2940f127fcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libasan-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2c8ade930bf0f7b07cf388afa18f3a19e6fc800585aba6998eead2940f127fcc",
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
    name = "libatomic-0__11.4.1-3.el9.aarch64",
    sha256 = "ca737169033f199ad497fa886d9b43784740d6f4b475da7ed8e2be8a71cfe812",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ca737169033f199ad497fa886d9b43784740d6f4b475da7ed8e2be8a71cfe812",
    ],
)

rpm(
    name = "libatomic-0__11.4.1-3.el9.s390x",
    sha256 = "a46fd032b758c28e215ef2c4d5ab944731736c760f58fbbdc770e34edce971df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libatomic-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a46fd032b758c28e215ef2c4d5ab944731736c760f58fbbdc770e34edce971df",
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
    name = "libattr-0__2.5.1-3.el9.s390x",
    sha256 = "c37335be62aaca9f21f2b0b0312d3800e245f6e70fa8b57d03ab89cce863f2be",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libattr-2.5.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c37335be62aaca9f21f2b0b0312d3800e245f6e70fa8b57d03ab89cce863f2be",
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
    name = "libblkid-0__2.37.4-18.el9.aarch64",
    sha256 = "bb4cd8f1748f2ecf837017dada4c52ef60dc896dc504aef3378e016d5cab57b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libblkid-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bb4cd8f1748f2ecf837017dada4c52ef60dc896dc504aef3378e016d5cab57b4",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-18.el9.s390x",
    sha256 = "eac8c0829e2a7b496f368891840a71b38ddd175e870b2d1d8b056e3d0381b21a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libblkid-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eac8c0829e2a7b496f368891840a71b38ddd175e870b2d1d8b056e3d0381b21a",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-18.el9.x86_64",
    sha256 = "f6dcef2625cb6910451ba7c7a8034ea0f73d06a9d7741e0373fe088fe6cf72dd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libblkid-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f6dcef2625cb6910451ba7c7a8034ea0f73d06a9d7741e0373fe088fe6cf72dd",
    ],
)

rpm(
    name = "libbpf-2__1.3.0-2.el9.aarch64",
    sha256 = "ff1b8056026a3b31ac4e292d578ff60dfb81c33fe3492bd3fed704ae1afc8038",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libbpf-1.3.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ff1b8056026a3b31ac4e292d578ff60dfb81c33fe3492bd3fed704ae1afc8038",
    ],
)

rpm(
    name = "libbpf-2__1.3.0-2.el9.s390x",
    sha256 = "4479a28eb752f05acb4dc2be2ae3fdffe714518bbe5cda0644f899928bffbc28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libbpf-1.3.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4479a28eb752f05acb4dc2be2ae3fdffe714518bbe5cda0644f899928bffbc28",
    ],
)

rpm(
    name = "libbpf-2__1.3.0-2.el9.x86_64",
    sha256 = "eed649a3d12a1fe2f3186b4b5885e16a5eeff498adf281af82fc2cb0cc49ce3e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-1.3.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eed649a3d12a1fe2f3186b4b5885e16a5eeff498adf281af82fc2cb0cc49ce3e",
    ],
)

rpm(
    name = "libbrotli-0__1.0.9-6.el9.x86_64",
    sha256 = "10b93bc07c62f31b96cbd4141a645880e76a2bc7d7163306ce2cc61a49616202",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbrotli-1.0.9-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10b93bc07c62f31b96cbd4141a645880e76a2bc7d7163306ce2cc61a49616202",
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
    name = "libburn-0__1.5.4-4.el9.s390x",
    sha256 = "21f50193372d1a50c1ad2ec1e51607b6cd480768fe9d4737929f0cef8486c157",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libburn-1.5.4-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/21f50193372d1a50c1ad2ec1e51607b6cd480768fe9d4737929f0cef8486c157",
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
    name = "libcap-0__2.48-9.el9.aarch64",
    sha256 = "2d78c324f8f8d9a14042995ab6e4c063c7d0a6acec1be07ac0d0d2c1a6de0ca5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcap-2.48-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2d78c324f8f8d9a14042995ab6e4c063c7d0a6acec1be07ac0d0d2c1a6de0ca5",
    ],
)

rpm(
    name = "libcap-0__2.48-9.el9.s390x",
    sha256 = "5c0d3fa01feeda3389847de7c0cd8d2631c26f0e929f609f176cbb661e09a8a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcap-2.48-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c0d3fa01feeda3389847de7c0cd8d2631c26f0e929f609f176cbb661e09a8a2",
    ],
)

rpm(
    name = "libcap-0__2.48-9.el9.x86_64",
    sha256 = "7d07ec8a6a0975d84c66adf21c885c41a5571ecb631055959265c60fda314111",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcap-2.48-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d07ec8a6a0975d84c66adf21c885c41a5571ecb631055959265c60fda314111",
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
    name = "libcap-ng-0__0.8.2-7.el9.s390x",
    sha256 = "9b68fda78e685d347ae1b9e937613125d01d7c8cdb06226e3c57e6cb08b9f306",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcap-ng-0.8.2-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9b68fda78e685d347ae1b9e937613125d01d7c8cdb06226e3c57e6cb08b9f306",
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
    name = "libcom_err-0__1.46.5-5.el9.aarch64",
    sha256 = "cd8b9b439b0434543cf0988567159bf9e6a329b7cbe8d9991a43375f88cc01d1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcom_err-1.46.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cd8b9b439b0434543cf0988567159bf9e6a329b7cbe8d9991a43375f88cc01d1",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-5.el9.s390x",
    sha256 = "3cca2a8ed3e319760a5935faf3d269288f0cea2cf2db2a5291e8996fc1ce7832",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcom_err-1.46.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3cca2a8ed3e319760a5935faf3d269288f0cea2cf2db2a5291e8996fc1ce7832",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-5.el9.x86_64",
    sha256 = "db2e675293b91b0f9b659cec0cad82c9c1b4af2112b6727e851d98a28ac83ed2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcom_err-1.46.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/db2e675293b91b0f9b659cec0cad82c9c1b4af2112b6727e851d98a28ac83ed2",
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
    name = "libcurl-minimal-0__7.76.1-29.el9.aarch64",
    sha256 = "ab067d463a4f534d6180ec03dd597da0eeda8cd6a1a3882c2d8c3fb31be10533",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.76.1-29.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab067d463a4f534d6180ec03dd597da0eeda8cd6a1a3882c2d8c3fb31be10533",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-29.el9.s390x",
    sha256 = "de8e5a00fed0d7ac020cf3c0ec2a6ea8e61f147ddf6b75824e8cfb8673aba250",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcurl-minimal-7.76.1-29.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/de8e5a00fed0d7ac020cf3c0ec2a6ea8e61f147ddf6b75824e8cfb8673aba250",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-29.el9.x86_64",
    sha256 = "c304626e349ed7953b4962705bf2903cd96bf5d3feb5eb5d14a68c4b7e2ff093",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-29.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c304626e349ed7953b4962705bf2903cd96bf5d3feb5eb5d14a68c4b7e2ff093",
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
    name = "libdb-0__5.3.28-53.el9.s390x",
    sha256 = "f29eec1adacc743d47d02539e4e8c735e49a6428c5edd08f89e39b0872e3ce89",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libdb-5.3.28-53.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f29eec1adacc743d47d02539e4e8c735e49a6428c5edd08f89e39b0872e3ce89",
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
    name = "libeconf-0__0.4.1-4.el9.aarch64",
    sha256 = "c221c71bfd8f6692e305a4e0c0025c4789ab04661c11a1a18c34c3f873f1276f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libeconf-0.4.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c221c71bfd8f6692e305a4e0c0025c4789ab04661c11a1a18c34c3f873f1276f",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-4.el9.s390x",
    sha256 = "1ee2d8e7b48a5e9616c1f7a5b019e0aa054a80b5962d972104d78d095b2e926d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libeconf-0.4.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1ee2d8e7b48a5e9616c1f7a5b019e0aa054a80b5962d972104d78d095b2e926d",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-4.el9.x86_64",
    sha256 = "ed519cc2e9031e2bf03275b28c7cca6520ae916d0a7edbbc69f327c1b70ed6cc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libeconf-0.4.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed519cc2e9031e2bf03275b28c7cca6520ae916d0a7edbbc69f327c1b70ed6cc",
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
    name = "libevent-0__2.1.12-6.el9.s390x",
    sha256 = "3a93804c6662e4ed7a4f8fffe66c1f1a0148e8f2269713affd710e78ba4ba680",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libevent-2.1.12-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3a93804c6662e4ed7a4f8fffe66c1f1a0148e8f2269713affd710e78ba4ba680",
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
    name = "libfdisk-0__2.37.4-18.el9.aarch64",
    sha256 = "691c82ee8a6fcebd52ed5ae3538cf997bfa3002260ef076c3bef3ff0289d72cf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libfdisk-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/691c82ee8a6fcebd52ed5ae3538cf997bfa3002260ef076c3bef3ff0289d72cf",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-18.el9.s390x",
    sha256 = "c121b308138c0ca6667f56d65eea86bdfdf917f496ee6ad57cedb8653abd1662",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libfdisk-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c121b308138c0ca6667f56d65eea86bdfdf917f496ee6ad57cedb8653abd1662",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-18.el9.x86_64",
    sha256 = "051951603ec09ab292198cb96d3377fb6353534d6572f3a21053de0f48ab9430",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/051951603ec09ab292198cb96d3377fb6353534d6572f3a21053de0f48ab9430",
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
    name = "libfdt-0__1.6.0-7.el9.s390x",
    sha256 = "fd91a54a5655e7727f059dd3a4c942cf81137e1eb30581a6d00bcc360aedacc4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libfdt-1.6.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fd91a54a5655e7727f059dd3a4c942cf81137e1eb30581a6d00bcc360aedacc4",
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
    name = "libffi-0__3.4.2-8.el9.aarch64",
    sha256 = "da6d3f1b21c23a97e61c35fde044aca5bc9f1097ffdcb387759f544c61548301",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libffi-3.4.2-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da6d3f1b21c23a97e61c35fde044aca5bc9f1097ffdcb387759f544c61548301",
    ],
)

rpm(
    name = "libffi-0__3.4.2-8.el9.s390x",
    sha256 = "25556c4a1bdb85f426595faa76996616a45986c93cac4361c2371f2e9b737304",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libffi-3.4.2-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/25556c4a1bdb85f426595faa76996616a45986c93cac4361c2371f2e9b737304",
    ],
)

rpm(
    name = "libffi-0__3.4.2-8.el9.x86_64",
    sha256 = "110d5008364a65b38b832949970886fdccb97762b0cdb257571cc0c84182d7d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libffi-3.4.2-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/110d5008364a65b38b832949970886fdccb97762b0cdb257571cc0c84182d7d0",
    ],
)

rpm(
    name = "libgcc-0__11.4.1-3.el9.aarch64",
    sha256 = "a43e0518c18884b805346ae15b093a83e88fafaa7650a0029a6bcfde92fb9fd0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a43e0518c18884b805346ae15b093a83e88fafaa7650a0029a6bcfde92fb9fd0",
    ],
)

rpm(
    name = "libgcc-0__11.4.1-3.el9.s390x",
    sha256 = "d520751ab1ae48e6383cc2ca0623510dbcfbfedfe43d50b63a5e1dec804717dd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcc-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d520751ab1ae48e6383cc2ca0623510dbcfbfedfe43d50b63a5e1dec804717dd",
    ],
)

rpm(
    name = "libgcc-0__11.4.1-3.el9.x86_64",
    sha256 = "2d396a7a02f751b420e62c60db854350cf1dc06d1ac2b05cd45952868e2ff46e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.4.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d396a7a02f751b420e62c60db854350cf1dc06d1ac2b05cd45952868e2ff46e",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-10.el9.aarch64",
    sha256 = "b5a90cb5a86ee956da8439362d8547342f240e71674e4703d87f27736dbede14",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.10.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b5a90cb5a86ee956da8439362d8547342f240e71674e4703d87f27736dbede14",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-10.el9.s390x",
    sha256 = "18a46fd543356843b5458b2bc347178ad89bfc21986f0f8bae0baf0350dbc5b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcrypt-1.10.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/18a46fd543356843b5458b2bc347178ad89bfc21986f0f8bae0baf0350dbc5b4",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-10.el9.x86_64",
    sha256 = "186ae69a1f72d3992f2f65a4cc91da856a54475f4762a69f3b5ca5d350e7edb3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.10.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/186ae69a1f72d3992f2f65a4cc91da856a54475f4762a69f3b5ca5d350e7edb3",
    ],
)

rpm(
    name = "libgomp-0__11.4.1-3.el9.aarch64",
    sha256 = "4c49fcdfda2926200b85e6185573a7ef8587ff257d5a20e67a04342d966230b9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4c49fcdfda2926200b85e6185573a7ef8587ff257d5a20e67a04342d966230b9",
    ],
)

rpm(
    name = "libgomp-0__11.4.1-3.el9.s390x",
    sha256 = "d32b5f3fdda0902edfe7658d9e12d5dc0b111ae01126383e05b907c5ac8d7ae2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgomp-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d32b5f3fdda0902edfe7658d9e12d5dc0b111ae01126383e05b907c5ac8d7ae2",
    ],
)

rpm(
    name = "libgomp-0__11.4.1-3.el9.x86_64",
    sha256 = "11e28d0e1fa5016fe8036656aec1a3f39741316363d276334e4a671bad18f241",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.4.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11e28d0e1fa5016fe8036656aec1a3f39741316363d276334e4a671bad18f241",
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
    name = "libgpg-error-0__1.42-5.el9.s390x",
    sha256 = "655367cd72f1908dbc2e42fee35974447d33eae7ec07249d3df098a6512d4601",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgpg-error-1.42-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/655367cd72f1908dbc2e42fee35974447d33eae7ec07249d3df098a6512d4601",
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
    name = "libguestfs-1__1.50.1-7.el9.x86_64",
    sha256 = "55ce206bede84732f6b29ce7c3c0e95583f66d351aacd61ed6165d1ecefabb22",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libguestfs-1.50.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/55ce206bede84732f6b29ce7c3c0e95583f66d351aacd61ed6165d1ecefabb22",
    ],
)

rpm(
    name = "libibverbs-0__51.0-1.el9.aarch64",
    sha256 = "c112f36234fd88d35c763fde0acc5a4c710fa04150b042ca1f9a9b9fb021b213",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libibverbs-51.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c112f36234fd88d35c763fde0acc5a4c710fa04150b042ca1f9a9b9fb021b213",
    ],
)

rpm(
    name = "libibverbs-0__51.0-1.el9.s390x",
    sha256 = "fa1a2b918636c9b77387b4f66aec6e14fb30dcfb30505d945d3dfd636f7561f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libibverbs-51.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fa1a2b918636c9b77387b4f66aec6e14fb30dcfb30505d945d3dfd636f7561f4",
    ],
)

rpm(
    name = "libibverbs-0__51.0-1.el9.x86_64",
    sha256 = "27a4ca698ffee208c042c203a9ab83b3d9ff98795d9ab26d039354e1e6f6b4d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libibverbs-51.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/27a4ca698ffee208c042c203a9ab83b3d9ff98795d9ab26d039354e1e6f6b4d9",
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
    name = "libidn2-0__2.3.0-7.el9.s390x",
    sha256 = "716716b688d4b702cee523a82d4ee035675f01ee404eb7dd7f2ef63d3389bb66",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libidn2-2.3.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/716716b688d4b702cee523a82d4ee035675f01ee404eb7dd7f2ef63d3389bb66",
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
    name = "libisoburn-0__1.5.4-4.el9.s390x",
    sha256 = "6ad23b4931165493e772413b7c8806f5a1e5a7b60f5c9393d23b74494c48a72e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libisoburn-1.5.4-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6ad23b4931165493e772413b7c8806f5a1e5a7b60f5c9393d23b74494c48a72e",
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
    name = "libisofs-0__1.5.4-4.el9.s390x",
    sha256 = "d6b426b1fc4c4343c66bc6aac25e18d643caa35bd1a103f710eef1c528bef299",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libisofs-1.5.4-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d6b426b1fc4c4343c66bc6aac25e18d643caa35bd1a103f710eef1c528bef299",
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
    name = "libksba-0__1.5.1-6.el9.x86_64",
    sha256 = "ff76d9798e2f040fed715968a9e67f6d5cfef59671e07575fc8d6510126b5340",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libksba-1.5.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff76d9798e2f040fed715968a9e67f6d5cfef59671e07575fc8d6510126b5340",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-16.el9.aarch64",
    sha256 = "c4d87c6439aa762891b024c0213df47af50e5b0683ffd827013bd02882d7d9b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmnl-1.0.4-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c4d87c6439aa762891b024c0213df47af50e5b0683ffd827013bd02882d7d9b3",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-16.el9.s390x",
    sha256 = "344f21dedaaad1ddc5279e31a4dafd9354662a61f010249d86a424c903c4415a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libmnl-1.0.4-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/344f21dedaaad1ddc5279e31a4dafd9354662a61f010249d86a424c903c4415a",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-16.el9.x86_64",
    sha256 = "e60f3be453b44ea04bb596594963be1e1b3f4377f87b4ff923d612eae15740ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmnl-1.0.4-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e60f3be453b44ea04bb596594963be1e1b3f4377f87b4ff923d612eae15740ce",
    ],
)

rpm(
    name = "libmount-0__2.37.4-18.el9.aarch64",
    sha256 = "47ff34986c31df37b782c62c0621d35954be2334bfd92a90376467b4376119fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmount-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/47ff34986c31df37b782c62c0621d35954be2334bfd92a90376467b4376119fd",
    ],
)

rpm(
    name = "libmount-0__2.37.4-18.el9.s390x",
    sha256 = "1e87191e999ea6ef99a7cf8eb65c6ddf6657c66a68eab46e0cd26a2cf463c28b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libmount-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1e87191e999ea6ef99a7cf8eb65c6ddf6657c66a68eab46e0cd26a2cf463c28b",
    ],
)

rpm(
    name = "libmount-0__2.37.4-18.el9.x86_64",
    sha256 = "54237ed9c05e3e307f0eb94a4ccd27c6790e8e05e39d3eafd76fd42a344aaa1d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmount-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54237ed9c05e3e307f0eb94a4ccd27c6790e8e05e39d3eafd76fd42a344aaa1d",
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
    name = "libmpc-0__1.2.1-4.el9.s390x",
    sha256 = "3d2a320348dd3d396005a0c2a75001fb1177fc35190ff009a1dd2cd370f6c629",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libmpc-1.2.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3d2a320348dd3d396005a0c2a75001fb1177fc35190ff009a1dd2cd370f6c629",
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
    name = "libnbd-0__1.20.0-1.el9.aarch64",
    sha256 = "00bd0691601823d680a3948788fa71d94d318d2fd2491664795a30f7a55cfe54",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libnbd-1.20.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/00bd0691601823d680a3948788fa71d94d318d2fd2491664795a30f7a55cfe54",
    ],
)

rpm(
    name = "libnbd-0__1.20.0-1.el9.s390x",
    sha256 = "397e085e942fb6db5f815038236749c4114f162d9c72f76bf8d208ed185964d1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libnbd-1.20.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/397e085e942fb6db5f815038236749c4114f162d9c72f76bf8d208ed185964d1",
    ],
)

rpm(
    name = "libnbd-0__1.20.0-1.el9.x86_64",
    sha256 = "3a75f8c6d3896c520c920eee09f2892e822a18986ffa0b87a7fdec6755389875",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libnbd-1.20.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a75f8c6d3896c520c920eee09f2892e822a18986ffa0b87a7fdec6755389875",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-1.el9.aarch64",
    sha256 = "6871a3371b5a9a8239606efd453b59b274040e9d8d8f0c18bdffa7264db64264",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnetfilter_conntrack-1.0.9-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6871a3371b5a9a8239606efd453b59b274040e9d8d8f0c18bdffa7264db64264",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-1.el9.s390x",
    sha256 = "803ecb7d6e42554735836a113b61e8501e952a715c754b76cec90631926e4830",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnetfilter_conntrack-1.0.9-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/803ecb7d6e42554735836a113b61e8501e952a715c754b76cec90631926e4830",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-1.el9.x86_64",
    sha256 = "f81a0188964268ae9e1d53d99dba3ef96a65fe2fb00bc8fe6c39cedfdd364f44",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnetfilter_conntrack-1.0.9-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f81a0188964268ae9e1d53d99dba3ef96a65fe2fb00bc8fe6c39cedfdd364f44",
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
    name = "libnfnetlink-0__1.0.1-21.el9.s390x",
    sha256 = "30dc6e1a8e1a026ff5a59759cf1cf8456f478c81fa11bc44aa69b9e80d7c3b5b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnfnetlink-1.0.1-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/30dc6e1a8e1a026ff5a59759cf1cf8456f478c81fa11bc44aa69b9e80d7c3b5b",
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
    name = "libnfsidmap-1__2.5.4-25.el9.x86_64",
    sha256 = "cdbc352c71574dccbec935613e7d70fd64b0aafc7aa0f62b35aba1aa2a9e0908",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfsidmap-2.5.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cdbc352c71574dccbec935613e7d70fd64b0aafc7aa0f62b35aba1aa2a9e0908",
    ],
)

rpm(
    name = "libnftnl-0__1.2.6-4.el9.aarch64",
    sha256 = "59f6d922f5540479c088120d411d2ca3cdb4e5ddf6fe8fc05dbd796b9e36ecd3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnftnl-1.2.6-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/59f6d922f5540479c088120d411d2ca3cdb4e5ddf6fe8fc05dbd796b9e36ecd3",
    ],
)

rpm(
    name = "libnftnl-0__1.2.6-4.el9.s390x",
    sha256 = "1a717d2a04f257e452753ba29cc6c0848cd51a226bf5d000b89863fa7aad5250",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnftnl-1.2.6-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1a717d2a04f257e452753ba29cc6c0848cd51a226bf5d000b89863fa7aad5250",
    ],
)

rpm(
    name = "libnftnl-0__1.2.6-4.el9.x86_64",
    sha256 = "45d7325859bdfbddd9f24235695fc55138549fdccbe509484e9f905c5f1b466b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnftnl-1.2.6-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/45d7325859bdfbddd9f24235695fc55138549fdccbe509484e9f905c5f1b466b",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-6.el9.aarch64",
    sha256 = "b9c3685701dc2ad11adac83055811bb8c4909bd73469f31953ef7d534c747b83",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.43.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b9c3685701dc2ad11adac83055811bb8c4909bd73469f31953ef7d534c747b83",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-6.el9.s390x",
    sha256 = "6d9ea7820d952bb492ff575b87fd46c606acf12bd368a5b4c8df3efc6a054c57",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnghttp2-1.43.0-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6d9ea7820d952bb492ff575b87fd46c606acf12bd368a5b4c8df3efc6a054c57",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-6.el9.x86_64",
    sha256 = "fc1cadbc6cf37cbea60112b7ae6f92fabfd5a7f76fa526bb5a1ea82746455ec7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.43.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fc1cadbc6cf37cbea60112b7ae6f92fabfd5a7f76fa526bb5a1ea82746455ec7",
    ],
)

rpm(
    name = "libnl3-0__3.9.0-1.el9.aarch64",
    sha256 = "c25c5717cbc4d00a47864de3c94aa68632a761f09167330aa91ef37b417b5439",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnl3-3.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c25c5717cbc4d00a47864de3c94aa68632a761f09167330aa91ef37b417b5439",
    ],
)

rpm(
    name = "libnl3-0__3.9.0-1.el9.s390x",
    sha256 = "393bf3ec56b39d80865914e60bbc7e71397d8787dfe66bfd48afabbec9d4ece4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnl3-3.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/393bf3ec56b39d80865914e60bbc7e71397d8787dfe66bfd48afabbec9d4ece4",
    ],
)

rpm(
    name = "libnl3-0__3.9.0-1.el9.x86_64",
    sha256 = "12218884f64bc18062a56315ab5f9f30a6b1c576be7f2ae841d47455b4da36bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnl3-3.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/12218884f64bc18062a56315ab5f9f30a6b1c576be7f2ae841d47455b4da36bd",
    ],
)

rpm(
    name = "libosinfo-0__1.10.0-1.el9.x86_64",
    sha256 = "ace3a92175ee1be1f5c3a1d31bd702c49076eea7f4d6e859fc301832424d3dc9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libosinfo-1.10.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ace3a92175ee1be1f5c3a1d31bd702c49076eea7f4d6e859fc301832424d3dc9",
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
    name = "libpcap-14__1.10.0-4.el9.s390x",
    sha256 = "ca99e77dd39751b9e769fbec73af47704b30ebac2b3fd0a5f3b4e3b6dca7ebc2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libpcap-1.10.0-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ca99e77dd39751b9e769fbec73af47704b30ebac2b3fd0a5f3b4e3b6dca7ebc2",
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
    name = "libpkgconf-0__1.7.3-10.el9.s390x",
    sha256 = "56221e0aeef5537804b6362a5336c5b1673b14c18b4dea09f42916fa9f976bc9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libpkgconf-1.7.3-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/56221e0aeef5537804b6362a5336c5b1673b14c18b4dea09f42916fa9f976bc9",
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
    name = "libpmem-0__1.12.1-1.el9.x86_64",
    sha256 = "5377dcb3b4ca48eb056a998d3a684eb68e8d059e2a26844cda8535d8f125fc83",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libpmem-1.12.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5377dcb3b4ca48eb056a998d3a684eb68e8d059e2a26844cda8535d8f125fc83",
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
    name = "libpng-2__1.6.37-12.el9.s390x",
    sha256 = "add58062b5ed4b22af0c1d6d5702260b2bc7c27cd08f298e908ac40a9df2f3f7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libpng-1.6.37-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/add58062b5ed4b22af0c1d6d5702260b2bc7c27cd08f298e908ac40a9df2f3f7",
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
    name = "libproxy-0__0.4.15-35.el9.x86_64",
    sha256 = "0042c2dd5a88f7f1db096426bb1f6557e7d790eabca01a086afd832e47217ee1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libproxy-0.4.15-35.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0042c2dd5a88f7f1db096426bb1f6557e7d790eabca01a086afd832e47217ee1",
    ],
)

rpm(
    name = "libpsl-0__0.21.1-5.el9.x86_64",
    sha256 = "42bd5fb4b34c993c103ea2d47fc69a0fcc231fcfb88646ed55403519868caa94",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpsl-0.21.1-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/42bd5fb4b34c993c103ea2d47fc69a0fcc231fcfb88646ed55403519868caa94",
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
    name = "libpwquality-0__1.4.4-8.el9.s390x",
    sha256 = "b8b5178474a0a53bc6463e817e0bca8a3568e333bcae9eda3dabbe84a1e24941",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libpwquality-1.4.4-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b8b5178474a0a53bc6463e817e0bca8a3568e333bcae9eda3dabbe84a1e24941",
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
    name = "librdmacm-0__51.0-1.el9.aarch64",
    sha256 = "56a1ec32fb075d18116f39629eff5a5dfd3cdba00345477a9fb594a3be01be8a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/librdmacm-51.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56a1ec32fb075d18116f39629eff5a5dfd3cdba00345477a9fb594a3be01be8a",
    ],
)

rpm(
    name = "librdmacm-0__51.0-1.el9.x86_64",
    sha256 = "add13f4e4c78a0480555f136a9bde1b7eed9f81ac5cd8b5278791d17c4ee41ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/librdmacm-51.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/add13f4e4c78a0480555f136a9bde1b7eed9f81ac5cd8b5278791d17c4ee41ad",
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
    name = "libseccomp-0__2.5.2-2.el9.s390x",
    sha256 = "1479993c13970d0a69826051948a080ea216fb74f0717d8718801065edf1a1de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libseccomp-2.5.2-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1479993c13970d0a69826051948a080ea216fb74f0717d8718801065edf1a1de",
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
    name = "libselinux-0__3.6-1.el9.aarch64",
    sha256 = "c7de4c5829488c9d1f0bb815f9b29dc3919533fb914f3408c7a276db2ee6c297",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-3.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7de4c5829488c9d1f0bb815f9b29dc3919533fb914f3408c7a276db2ee6c297",
    ],
)

rpm(
    name = "libselinux-0__3.6-1.el9.s390x",
    sha256 = "820f7c6a2321cc162df878504de505e26cdcd1be854bebbbedbbeb4b3cf95fc1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-3.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/820f7c6a2321cc162df878504de505e26cdcd1be854bebbbedbbeb4b3cf95fc1",
    ],
)

rpm(
    name = "libselinux-0__3.6-1.el9.x86_64",
    sha256 = "274be34f74f9a5adab8428cd4761df8c50ff85b2c1bad832d90a3b3bf3efd174",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-3.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/274be34f74f9a5adab8428cd4761df8c50ff85b2c1bad832d90a3b3bf3efd174",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-1.el9.aarch64",
    sha256 = "da8f16409e08c5d959002f37c84bde486160de078869bc95a4f373b6100c1636",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da8f16409e08c5d959002f37c84bde486160de078869bc95a4f373b6100c1636",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-1.el9.s390x",
    sha256 = "b3af5a9ac657dde5261e50c7bc77ebff2f66f77f0bbc8313e7b977c57a76d335",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-utils-3.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b3af5a9ac657dde5261e50c7bc77ebff2f66f77f0bbc8313e7b977c57a76d335",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-1.el9.x86_64",
    sha256 = "5ccd4affb097e4cb8f88917f8997f21d548272192ba8ab86b592a0de3b39305f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5ccd4affb097e4cb8f88917f8997f21d548272192ba8ab86b592a0de3b39305f",
    ],
)

rpm(
    name = "libsemanage-0__3.6-1.el9.aarch64",
    sha256 = "f82d438c8af25926a67f75e9f758a9d39419b6f540118d41a1c57aa2c902d1e8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsemanage-3.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f82d438c8af25926a67f75e9f758a9d39419b6f540118d41a1c57aa2c902d1e8",
    ],
)

rpm(
    name = "libsemanage-0__3.6-1.el9.s390x",
    sha256 = "15eb6e6dfc06fa5351eca930a611803783991464209994d9a1402dfc76884f5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsemanage-3.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/15eb6e6dfc06fa5351eca930a611803783991464209994d9a1402dfc76884f5f",
    ],
)

rpm(
    name = "libsemanage-0__3.6-1.el9.x86_64",
    sha256 = "7f5bce18f75287a1911028cb38a5e8103787038026f6d1fda4ef6afa1cb00efd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsemanage-3.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f5bce18f75287a1911028cb38a5e8103787038026f6d1fda4ef6afa1cb00efd",
    ],
)

rpm(
    name = "libsepol-0__3.6-1.el9.aarch64",
    sha256 = "d5fbf72e47423eadf245d8cf8ecc3fb8bec2725ea0504c2cec8d68120603783a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsepol-3.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d5fbf72e47423eadf245d8cf8ecc3fb8bec2725ea0504c2cec8d68120603783a",
    ],
)

rpm(
    name = "libsepol-0__3.6-1.el9.s390x",
    sha256 = "58df3e6e550cded42d31f51140e7d0adc427bc4efbb6737e8efe3b6a30680369",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsepol-3.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/58df3e6e550cded42d31f51140e7d0adc427bc4efbb6737e8efe3b6a30680369",
    ],
)

rpm(
    name = "libsepol-0__3.6-1.el9.x86_64",
    sha256 = "834f9dd59bf8bd0cf5047c672b1d610b722a0981f53c15dd36cc3daffaba0230",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsepol-3.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/834f9dd59bf8bd0cf5047c672b1d610b722a0981f53c15dd36cc3daffaba0230",
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
    name = "libsigsegv-0__2.13-4.el9.s390x",
    sha256 = "730c827d66bd292fccdb6f8ac4c29176e7f06283489be41b67f4bf55deeb3ffb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsigsegv-2.13-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/730c827d66bd292fccdb6f8ac4c29176e7f06283489be41b67f4bf55deeb3ffb",
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
    name = "libslirp-0__4.4.0-7.el9.aarch64",
    sha256 = "321ef98abb278174e60823b5f032ef8f5bee45d67a0e2b0a56e08e6ae8a7381b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libslirp-4.4.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/321ef98abb278174e60823b5f032ef8f5bee45d67a0e2b0a56e08e6ae8a7381b",
    ],
)

rpm(
    name = "libslirp-0__4.4.0-7.el9.s390x",
    sha256 = "480a6c0f545a5b1d5c9f24467b5fe6cb01b2ebe9e4b0a1469339dcd48710c740",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libslirp-4.4.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/480a6c0f545a5b1d5c9f24467b5fe6cb01b2ebe9e4b0a1469339dcd48710c740",
    ],
)

rpm(
    name = "libslirp-0__4.4.0-7.el9.x86_64",
    sha256 = "4d7383a18c393e909d037f64c35a8d5d01c559032a3bd760a77844986d57062a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libslirp-4.4.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d7383a18c393e909d037f64c35a8d5d01c559032a3bd760a77844986d57062a",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-18.el9.aarch64",
    sha256 = "c2126c4ef442a5b76e12a7e26fe5d2c9c342aadeef567ff7ee6c445e9a50bd48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c2126c4ef442a5b76e12a7e26fe5d2c9c342aadeef567ff7ee6c445e9a50bd48",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-18.el9.s390x",
    sha256 = "0e5adad760d3cc133a48f426a79fcba0fefd6d5e67aa0c9a64e15ed52afa806b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsmartcols-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0e5adad760d3cc133a48f426a79fcba0fefd6d5e67aa0c9a64e15ed52afa806b",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-18.el9.x86_64",
    sha256 = "9bfa25bb3cc5308b29c9eb466a40a9c35e9c7e3c4f31d71487a91f9dacaa1870",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9bfa25bb3cc5308b29c9eb466a40a9c35e9c7e3c4f31d71487a91f9dacaa1870",
    ],
)

rpm(
    name = "libsoup-0__2.72.0-8.el9.x86_64",
    sha256 = "f28214b594a46422e75a946a491de3f8cf29289c33c26ecab60cce82fcff6d68",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libsoup-2.72.0-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f28214b594a46422e75a946a491de3f8cf29289c33c26ecab60cce82fcff6d68",
    ],
)

rpm(
    name = "libss-0__1.46.5-5.el9.aarch64",
    sha256 = "2daf0795387601ae55a2892e26e3fe924e5671753f1a47825699ff593c51b053",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libss-1.46.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2daf0795387601ae55a2892e26e3fe924e5671753f1a47825699ff593c51b053",
    ],
)

rpm(
    name = "libss-0__1.46.5-5.el9.s390x",
    sha256 = "927a618410f07c005520c26bd1326afd14d8c5547dfda888916e70c31f232771",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libss-1.46.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/927a618410f07c005520c26bd1326afd14d8c5547dfda888916e70c31f232771",
    ],
)

rpm(
    name = "libss-0__1.46.5-5.el9.x86_64",
    sha256 = "2c0548dd2ba1272fbf81ffed0dcab629f4ada97a17ca839aba2cdbf6e11948b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libss-1.46.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c0548dd2ba1272fbf81ffed0dcab629f4ada97a17ca839aba2cdbf6e11948b4",
    ],
)

rpm(
    name = "libssh-0__0.10.4-13.el9.aarch64",
    sha256 = "81d3f1d8489d3330065c24604c0e994cb9f8aa653e6859512beeabaa64b8f6c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-0.10.4-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/81d3f1d8489d3330065c24604c0e994cb9f8aa653e6859512beeabaa64b8f6c3",
    ],
)

rpm(
    name = "libssh-0__0.10.4-13.el9.s390x",
    sha256 = "dac9ce73baa7946783ef8a277a8a28e6d8f6ef3375126d47cf57bcffd29e77a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libssh-0.10.4-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/dac9ce73baa7946783ef8a277a8a28e6d8f6ef3375126d47cf57bcffd29e77a4",
    ],
)

rpm(
    name = "libssh-0__0.10.4-13.el9.x86_64",
    sha256 = "08f4dd4a9a61fb4dc05b30523cbd6a6bb698e634c8c87e884f78db2cfc658499",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-0.10.4-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/08f4dd4a9a61fb4dc05b30523cbd6a6bb698e634c8c87e884f78db2cfc658499",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-13.el9.aarch64",
    sha256 = "bd86ac0962a7f517dd0ab4b963e08b6b7a2af1821df374746223ab9781ee9a20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-config-0.10.4-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd86ac0962a7f517dd0ab4b963e08b6b7a2af1821df374746223ab9781ee9a20",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-13.el9.s390x",
    sha256 = "bd86ac0962a7f517dd0ab4b963e08b6b7a2af1821df374746223ab9781ee9a20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libssh-config-0.10.4-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd86ac0962a7f517dd0ab4b963e08b6b7a2af1821df374746223ab9781ee9a20",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-13.el9.x86_64",
    sha256 = "bd86ac0962a7f517dd0ab4b963e08b6b7a2af1821df374746223ab9781ee9a20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-config-0.10.4-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd86ac0962a7f517dd0ab4b963e08b6b7a2af1821df374746223ab9781ee9a20",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.5-1.el9.aarch64",
    sha256 = "14fd9263a1897171cdce82aaba3a2aa51dae75df7be2d47b82567439d4ae9a32",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.9.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/14fd9263a1897171cdce82aaba3a2aa51dae75df7be2d47b82567439d4ae9a32",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.5-1.el9.s390x",
    sha256 = "15b02745cddd4f1977b5bac81d5d57c13bda5f9ab7c0ef8aa5274b38081b0367",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.9.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/15b02745cddd4f1977b5bac81d5d57c13bda5f9ab7c0ef8aa5274b38081b0367",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.5-1.el9.x86_64",
    sha256 = "52d1cea7f64f465de3a6713276e6c26b6920ae833f2f48c1f3e3fd395807bc10",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.9.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/52d1cea7f64f465de3a6713276e6c26b6920ae833f2f48c1f3e3fd395807bc10",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.5-1.el9.aarch64",
    sha256 = "3a89a375b6e33f1944c517d8eeb1c634aff96ae5a2342f869b7ff01ad90d2a5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.9.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3a89a375b6e33f1944c517d8eeb1c634aff96ae5a2342f869b7ff01ad90d2a5f",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.5-1.el9.s390x",
    sha256 = "494e77dfdf0934e006dc1ba54e012413f44c2c88df999dc625bd90c0c690886e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.9.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/494e77dfdf0934e006dc1ba54e012413f44c2c88df999dc625bd90c0c690886e",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.5-1.el9.x86_64",
    sha256 = "c5535c56ff23e1e9380e25e40e1d90e3d617402121222658c331b8fcc943160f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.9.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c5535c56ff23e1e9380e25e40e1d90e3d617402121222658c331b8fcc943160f",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.4.1-3.el9.aarch64",
    sha256 = "58a628cb93581f0683934487d6d37092dbfd0ee7eb244e18d16e54b56e2481b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/58a628cb93581f0683934487d6d37092dbfd0ee7eb244e18d16e54b56e2481b2",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.4.1-3.el9.s390x",
    sha256 = "f826e1082213bdf7f588d262b2a05a267bd616e7c33b8b52db6cb84e988c53bc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libstdc++-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f826e1082213bdf7f588d262b2a05a267bd616e7c33b8b52db6cb84e988c53bc",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.4.1-3.el9.x86_64",
    sha256 = "1a390ff013dd608de0dfe63dde620ebb5ca388d7e9c7600135eb61bbd287d7a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.4.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1a390ff013dd608de0dfe63dde620ebb5ca388d7e9c7600135eb61bbd287d7a8",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-8.el9.aarch64",
    sha256 = "1046c07821506ef6a84291b093de0d62dcc9873142e1ac2c66aaa72abd08532c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtasn1-4.16.0-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1046c07821506ef6a84291b093de0d62dcc9873142e1ac2c66aaa72abd08532c",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-8.el9.s390x",
    sha256 = "1a03374dd2825e0cc9dacddb31c9537835138b0c12713faed4d38890bb1a3882",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libtasn1-4.16.0-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1a03374dd2825e0cc9dacddb31c9537835138b0c12713faed4d38890bb1a3882",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-8.el9.x86_64",
    sha256 = "c8b13c9e1292de474e76ab80f230f86cce2e8f5f53592e168bdcaa604ed1b37d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtasn1-4.16.0-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c8b13c9e1292de474e76ab80f230f86cce2e8f5f53592e168bdcaa604ed1b37d",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-6.el9.aarch64",
    sha256 = "3d3b2a02844fa74bef7f900f7ba7515641549865bbb91737222aa7ec3fba7ccd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtirpc-1.3.3-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3d3b2a02844fa74bef7f900f7ba7515641549865bbb91737222aa7ec3fba7ccd",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-6.el9.s390x",
    sha256 = "ced0b2acf8959fc5516a2578880aaa0eb885eeb6f190872b4dd3ea4032bf0abd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libtirpc-1.3.3-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ced0b2acf8959fc5516a2578880aaa0eb885eeb6f190872b4dd3ea4032bf0abd",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-6.el9.x86_64",
    sha256 = "bb04a824e4a15f067a72f4f38a42763fe91d1de8766a51633e231a1dbe35fcf2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtirpc-1.3.3-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb04a824e4a15f067a72f4f38a42763fe91d1de8766a51633e231a1dbe35fcf2",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-3.20211126git1ff6fe1f43.el9.aarch64",
    sha256 = "7f3313bf113fce33ece6b942942a8126713289a545da9eafbb508e9ff6008be2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libtpms-0.9.1-3.20211126git1ff6fe1f43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7f3313bf113fce33ece6b942942a8126713289a545da9eafbb508e9ff6008be2",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-3.20211126git1ff6fe1f43.el9.s390x",
    sha256 = "ebfe9894efc7e773d9bcbf1dc737ce3d6919077119dc24a9d7ab2a86b8c088c1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libtpms-0.9.1-3.20211126git1ff6fe1f43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ebfe9894efc7e773d9bcbf1dc737ce3d6919077119dc24a9d7ab2a86b8c088c1",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-3.20211126git1ff6fe1f43.el9.x86_64",
    sha256 = "4ed3052085b118c19f44c3ce29749895627acc590e45acc0722da5d53582afe7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libtpms-0.9.1-3.20211126git1ff6fe1f43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4ed3052085b118c19f44c3ce29749895627acc590e45acc0722da5d53582afe7",
    ],
)

rpm(
    name = "libubsan-0__11.4.1-3.el9.aarch64",
    sha256 = "e28c48794e2259b89618749941454900a32ad737a458d5681fdd8169ae789c00",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e28c48794e2259b89618749941454900a32ad737a458d5681fdd8169ae789c00",
    ],
)

rpm(
    name = "libubsan-0__11.4.1-3.el9.s390x",
    sha256 = "010d4cdb2d4e4d6091a3959cc9ca080ac6e664e0eea5755c323bfbd92ac5828f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libubsan-11.4.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/010d4cdb2d4e4d6091a3959cc9ca080ac6e664e0eea5755c323bfbd92ac5828f",
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
    name = "libunistring-0__0.9.10-15.el9.s390x",
    sha256 = "029cedc9f79dcc145f59e2bbf2121d406b3853765d56345a75bc987760d5d2d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libunistring-0.9.10-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/029cedc9f79dcc145f59e2bbf2121d406b3853765d56345a75bc987760d5d2d2",
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
    name = "liburing-0__2.5-1.el9.aarch64",
    sha256 = "12f91bd14e1eb7e2b37783561c1a0658d85c7ee2a9259391ed15e01bf4186649",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/liburing-2.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/12f91bd14e1eb7e2b37783561c1a0658d85c7ee2a9259391ed15e01bf4186649",
    ],
)

rpm(
    name = "liburing-0__2.5-1.el9.s390x",
    sha256 = "f45d4fcccfd217d5aa394a317d4d2645b79edb50cd7ad01dc14ad0d1b1bdb2f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/liburing-2.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f45d4fcccfd217d5aa394a317d4d2645b79edb50cd7ad01dc14ad0d1b1bdb2f0",
    ],
)

rpm(
    name = "liburing-0__2.5-1.el9.x86_64",
    sha256 = "12558038d4226495da372e5f4369d02c144c759a621d27116299ce0a794e849f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/liburing-2.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/12558038d4226495da372e5f4369d02c144c759a621d27116299ce0a794e849f",
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
    name = "libusbx-0__1.0.26-1.el9.s390x",
    sha256 = "d590301604a0636520462079997fa6fab7839084c77985a8a7fe16f1126d1b9b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libusbx-1.0.26-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d590301604a0636520462079997fa6fab7839084c77985a8a7fe16f1126d1b9b",
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
    name = "libutempter-0__1.2.1-6.el9.s390x",
    sha256 = "6c000dac4305215beb37c8931a85ee137806f06547ecfb9a23e1915f01a3baa2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libutempter-1.2.1-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6c000dac4305215beb37c8931a85ee137806f06547ecfb9a23e1915f01a3baa2",
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
    name = "libuuid-0__2.37.4-18.el9.aarch64",
    sha256 = "39003e00883594e490723b13754537b0b53ebc320b91ed58fe4715f852dee8ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libuuid-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/39003e00883594e490723b13754537b0b53ebc320b91ed58fe4715f852dee8ee",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-18.el9.s390x",
    sha256 = "7fc04c6e2b4159bbb2b3c5aa761359205d44447aab3ba5799de69c167cf234fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libuuid-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7fc04c6e2b4159bbb2b3c5aa761359205d44447aab3ba5799de69c167cf234fb",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-18.el9.x86_64",
    sha256 = "226514aadd153d4065cdb1d008f1dbab1d8ee8653e319331c9000e52ebbefe67",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/226514aadd153d4065cdb1d008f1dbab1d8ee8653e319331c9000e52ebbefe67",
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
    name = "libverto-0__0.3.2-3.el9.s390x",
    sha256 = "3d794c924cc3611f1b37033d6835c4af71a555fcba053618bd6d48ad79547ab0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libverto-0.3.2-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3d794c924cc3611f1b37033d6835c4af71a555fcba053618bd6d48ad79547ab0",
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
    name = "libvirt-client-0__10.0.0-7.el9.aarch64",
    sha256 = "9976fde502df1b979d0f47d540e45e62cc01e12a44a4cfd05b5b5fbdfa93ec63",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-10.0.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9976fde502df1b979d0f47d540e45e62cc01e12a44a4cfd05b5b5fbdfa93ec63",
    ],
)

rpm(
    name = "libvirt-client-0__10.0.0-7.el9.s390x",
    sha256 = "eab9b8ee70e4aac5f2dfa57e33d29a6f44bc344f85ad6ae340436b978e97557b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-client-10.0.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eab9b8ee70e4aac5f2dfa57e33d29a6f44bc344f85ad6ae340436b978e97557b",
    ],
)

rpm(
    name = "libvirt-client-0__10.0.0-7.el9.x86_64",
    sha256 = "f2281936ca15a92e33164d9091c5e50679a8f50b964ef47583002f8f44697cb2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f2281936ca15a92e33164d9091c5e50679a8f50b964ef47583002f8f44697cb2",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.0.0-7.el9.aarch64",
    sha256 = "ac53dc4e5df99c30ad24f0c0b2f00d50487a76efca7cd250d1c74f79126a149b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-10.0.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ac53dc4e5df99c30ad24f0c0b2f00d50487a76efca7cd250d1c74f79126a149b",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.0.0-7.el9.s390x",
    sha256 = "6f8bc563da1172beeaff374f378a09aa9f0fbbf5936f337f1b981c2d8541f395",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-10.0.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6f8bc563da1172beeaff374f378a09aa9f0fbbf5936f337f1b981c2d8541f395",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.0.0-7.el9.x86_64",
    sha256 = "20e7cde192df5206951b6e7dfaa851030fd7c1280b60400462a24c6938da5fee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20e7cde192df5206951b6e7dfaa851030fd7c1280b60400462a24c6938da5fee",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.0.0-7.el9.aarch64",
    sha256 = "e6352e1073435f27fed6ce61a59d895afbf8eaa8696a6fd6fb24580557a9a62d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-10.0.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e6352e1073435f27fed6ce61a59d895afbf8eaa8696a6fd6fb24580557a9a62d",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.0.0-7.el9.s390x",
    sha256 = "9a16542f2724599c4f63930ae0b3596f48331f70c49deab3136dffba4111a7f6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-10.0.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9a16542f2724599c4f63930ae0b3596f48331f70c49deab3136dffba4111a7f6",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.0.0-7.el9.x86_64",
    sha256 = "ebaff385d85e95befd7d1c0eac733ce58896097ee301e2f28ebb2f2c081cb6ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ebaff385d85e95befd7d1c0eac733ce58896097ee301e2f28ebb2f2c081cb6ed",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.0.0-7.el9.x86_64",
    sha256 = "abd8727aa126ad0305024e5712d6a91d287c61a9c95f77e1ef744ddf4e7bbbc2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/abd8727aa126ad0305024e5712d6a91d287c61a9c95f77e1ef744ddf4e7bbbc2",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.0.0-7.el9.x86_64",
    sha256 = "e7e73e35feadf063256527c4f03552e4349c1038f883db815244755fef26e9f9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e7e73e35feadf063256527c4f03552e4349c1038f883db815244755fef26e9f9",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.0.0-7.el9.aarch64",
    sha256 = "63597238d55f265c98db56ebc0b30e4413f40fd043c85e36d6ad9d5f9e308e97",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-10.0.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/63597238d55f265c98db56ebc0b30e4413f40fd043c85e36d6ad9d5f9e308e97",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.0.0-7.el9.s390x",
    sha256 = "94e7232b41df3865b927d7f4f6a60a9651e53209d3eae6d236502136953d5f82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-10.0.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/94e7232b41df3865b927d7f4f6a60a9651e53209d3eae6d236502136953d5f82",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.0.0-7.el9.x86_64",
    sha256 = "52a9b231699b93b070e7283cef3dc74fa1dda84d0f3d0838b059db5e492e0330",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/52a9b231699b93b070e7283cef3dc74fa1dda84d0f3d0838b059db5e492e0330",
    ],
)

rpm(
    name = "libvirt-devel-0__10.0.0-7.el9.aarch64",
    sha256 = "2b709fc0db1acac46ae07c58b1681929ea8c8dd7775b82a4557bfb3f7dd30caf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-10.0.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2b709fc0db1acac46ae07c58b1681929ea8c8dd7775b82a4557bfb3f7dd30caf",
    ],
)

rpm(
    name = "libvirt-devel-0__10.0.0-7.el9.s390x",
    sha256 = "8854ff7474865dc1049e5031f762919b94cbdcb61eda87faf369f3d7de2c9779",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libvirt-devel-10.0.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8854ff7474865dc1049e5031f762919b94cbdcb61eda87faf369f3d7de2c9779",
    ],
)

rpm(
    name = "libvirt-devel-0__10.0.0-7.el9.x86_64",
    sha256 = "b2435e80b394e29fafef980b90e82d4997ce01f9e52f1ef20f1cbfdd192b7f10",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2435e80b394e29fafef980b90e82d4997ce01f9e52f1ef20f1cbfdd192b7f10",
    ],
)

rpm(
    name = "libvirt-libs-0__10.0.0-7.el9.aarch64",
    sha256 = "aed08c7461fda651e44968ed4a4dfd194706b1ec472c56030ddde9bdf48fcec4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-10.0.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aed08c7461fda651e44968ed4a4dfd194706b1ec472c56030ddde9bdf48fcec4",
    ],
)

rpm(
    name = "libvirt-libs-0__10.0.0-7.el9.s390x",
    sha256 = "c8b09dbd36adc9420ed347f4d0bc204408abb303e6bd5801b6fefae9ea94d26a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-libs-10.0.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c8b09dbd36adc9420ed347f4d0bc204408abb303e6bd5801b6fefae9ea94d26a",
    ],
)

rpm(
    name = "libvirt-libs-0__10.0.0-7.el9.x86_64",
    sha256 = "557a216b934016003eed62979e5a8945d786a3a1f02dc65876da4bb7f445c245",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-10.0.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/557a216b934016003eed62979e5a8945d786a3a1f02dc65876da4bb7f445c245",
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
    name = "libxcrypt-0__4.4.18-3.el9.s390x",
    sha256 = "dd9d51f68ae799b41cbe4cc00945280c65ed0c098b72f79d8d39a5c462b37074",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libxcrypt-4.4.18-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/dd9d51f68ae799b41cbe4cc00945280c65ed0c098b72f79d8d39a5c462b37074",
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
    name = "libxcrypt-devel-0__4.4.18-3.el9.s390x",
    sha256 = "bc088a5a60f086756b5be929fd420d5bfe56a77740a2b68be3c14601537244ac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libxcrypt-devel-4.4.18-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bc088a5a60f086756b5be929fd420d5bfe56a77740a2b68be3c14601537244ac",
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
    name = "libxcrypt-static-0__4.4.18-3.el9.s390x",
    sha256 = "32de43720371755ee2fffd5c5421cdd3c66a6470ce8ce1a5b8d2d975c4c19c99",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libxcrypt-static-4.4.18-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/32de43720371755ee2fffd5c5421cdd3c66a6470ce8ce1a5b8d2d975c4c19c99",
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
    name = "libxml2-0__2.9.13-6.el9.aarch64",
    sha256 = "d567f4bcf953cffe949be6d11d5597bf1a8c806c89c999e7943c240da40122b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.13-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d567f4bcf953cffe949be6d11d5597bf1a8c806c89c999e7943c240da40122b8",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-6.el9.s390x",
    sha256 = "2ba167d1c5fe690868d32c2f09645a080297ca7f731c9793c9ac89ff8043455d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libxml2-2.9.13-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2ba167d1c5fe690868d32c2f09645a080297ca7f731c9793c9ac89ff8043455d",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-6.el9.x86_64",
    sha256 = "7b23a9ca73db2ec13ee983594d4d0f4a85160ef8d05484f65c247801cb808a29",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.13-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7b23a9ca73db2ec13ee983594d4d0f4a85160ef8d05484f65c247801cb808a29",
    ],
)

rpm(
    name = "libxslt-0__1.1.34-9.el9.x86_64",
    sha256 = "576a1d36454a155d109ba1d0bb89b3a90b932d0b539fcd6392a67054bebc0015",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libxslt-1.1.34-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/576a1d36454a155d109ba1d0bb89b3a90b932d0b539fcd6392a67054bebc0015",
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
    name = "libzstd-0__1.5.1-2.el9.s390x",
    sha256 = "a84659a6861d44aaa063e69d58c1a582c34431b2e168965ac9e717ce7efb5b4a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libzstd-1.5.1-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a84659a6861d44aaa063e69d58c1a582c34431b2e168965ac9e717ce7efb5b4a",
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
    name = "lua-libs-0__5.4.4-4.el9.aarch64",
    sha256 = "bd72283eb56206de91a71b1b7dbdcca1201fdaea4a08faf7b92d8ef9a600a88a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/lua-libs-5.4.4-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bd72283eb56206de91a71b1b7dbdcca1201fdaea4a08faf7b92d8ef9a600a88a",
    ],
)

rpm(
    name = "lua-libs-0__5.4.4-4.el9.s390x",
    sha256 = "616111e91869993d6db2fec066d5b5b29b2c17bfbce87748a51ed772dbc4d4ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/lua-libs-5.4.4-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/616111e91869993d6db2fec066d5b5b29b2c17bfbce87748a51ed772dbc4d4ca",
    ],
)

rpm(
    name = "lua-libs-0__5.4.4-4.el9.x86_64",
    sha256 = "a24f7e08163b012cdbbdaba70788331050c2b7bdb9bc2fdc261c5c1f3cd3960d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/lua-libs-5.4.4-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a24f7e08163b012cdbbdaba70788331050c2b7bdb9bc2fdc261c5c1f3cd3960d",
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
    name = "lz4-libs-0__1.9.3-5.el9.s390x",
    sha256 = "358c7c19e9ec8778874066342c591b71877c3324f0727357342dffb4e1ec3498",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/lz4-libs-1.9.3-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/358c7c19e9ec8778874066342c591b71877c3324f0727357342dffb4e1ec3498",
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
    name = "lzo-0__2.10-7.el9.s390x",
    sha256 = "d35dc772b6fe7070ddc15aef9d37550ae638f304bf9b9a5c15bff0f5730cd43a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/lzo-2.10-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d35dc772b6fe7070ddc15aef9d37550ae638f304bf9b9a5c15bff0f5730cd43a",
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
    name = "lzop-0__1.04-8.el9.s390x",
    sha256 = "f1b48c30e04aa4734302a24f965647b20b784f5ed73debed74920d2e68eb6bda",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/lzop-1.04-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f1b48c30e04aa4734302a24f965647b20b784f5ed73debed74920d2e68eb6bda",
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
    name = "make-1__4.3-8.el9.aarch64",
    sha256 = "65fbb428870eea959bb831c11a7e0eaa249071b7185b5d8d16ad84b124280ae8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/make-4.3-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/65fbb428870eea959bb831c11a7e0eaa249071b7185b5d8d16ad84b124280ae8",
    ],
)

rpm(
    name = "make-1__4.3-8.el9.s390x",
    sha256 = "b3ad9b83ee1419b3f614c5bae44f5f3502bc4cf67ca8f1d9664186eb169dc262",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/make-4.3-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b3ad9b83ee1419b3f614c5bae44f5f3502bc4cf67ca8f1d9664186eb169dc262",
    ],
)

rpm(
    name = "make-1__4.3-8.el9.x86_64",
    sha256 = "3f6a7886f17d9bf4266d507e8f93a3e6164cb3444429517da6cfcacf041a08a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/make-4.3-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f6a7886f17d9bf4266d507e8f93a3e6164cb3444429517da6cfcacf041a08a4",
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
    name = "mpfr-0__4.1.0-7.el9.s390x",
    sha256 = "7297fc0b6869453925eed12b13c17ed76379352f63e0303644bef64386b034f1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/mpfr-4.1.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7297fc0b6869453925eed12b13c17ed76379352f63e0303644bef64386b034f1",
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
    name = "ncurses-base-0__6.2-10.20210508.el9.aarch64",
    sha256 = "00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.2-10.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-10.20210508.el9.s390x",
    sha256 = "00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ncurses-base-6.2-10.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-10.20210508.el9.x86_64",
    sha256 = "00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.2-10.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-10.20210508.el9.aarch64",
    sha256 = "0ccfc9eeb99be404367bf6157db2d1a6fb9ed479247f578501594e08e8f7080c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.2-10.20210508.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0ccfc9eeb99be404367bf6157db2d1a6fb9ed479247f578501594e08e8f7080c",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-10.20210508.el9.s390x",
    sha256 = "6ff5f715d02fa044b431b4766e13a424961faa04795f3189b05bf5c58b13dee2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ncurses-libs-6.2-10.20210508.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6ff5f715d02fa044b431b4766e13a424961faa04795f3189b05bf5c58b13dee2",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-10.20210508.el9.x86_64",
    sha256 = "f4ead70a508051ed338499b35605b5b2b5bccde19c9e83f7e4b948f171b542ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.2-10.20210508.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f4ead70a508051ed338499b35605b5b2b5bccde19c9e83f7e4b948f171b542ff",
    ],
)

rpm(
    name = "ndctl-libs-0__71.1-8.el9.x86_64",
    sha256 = "69d469e5106559ca5a156a2191f85e89fd44f7866701bfb35e197e5133413098",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ndctl-libs-71.1-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/69d469e5106559ca5a156a2191f85e89fd44f7866701bfb35e197e5133413098",
    ],
)

rpm(
    name = "nettle-0__3.9.1-1.el9.aarch64",
    sha256 = "991294c5c3f1544172cbc0c3bf27540036e0d09f42c161ef8bdf231c97d9ced0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nettle-3.9.1-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/991294c5c3f1544172cbc0c3bf27540036e0d09f42c161ef8bdf231c97d9ced0",
    ],
)

rpm(
    name = "nettle-0__3.9.1-1.el9.s390x",
    sha256 = "3b13fd8975ebb5bf3eff89eeb0d5ec0dc6f65d8bd8776b1dae8d2c8ce99b54bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nettle-3.9.1-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3b13fd8975ebb5bf3eff89eeb0d5ec0dc6f65d8bd8776b1dae8d2c8ce99b54bb",
    ],
)

rpm(
    name = "nettle-0__3.9.1-1.el9.x86_64",
    sha256 = "ffeeab0a6b0caaf457ad77a64bb1dfac6c1144343f1057de64a89b5ae4b58bf5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nettle-3.9.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffeeab0a6b0caaf457ad77a64bb1dfac6c1144343f1057de64a89b5ae4b58bf5",
    ],
)

rpm(
    name = "nfs-utils-1__2.5.4-25.el9.x86_64",
    sha256 = "985879156810b0df16ce8207a9753e5264e7430f6403a585b7efa2b256f68b45",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nfs-utils-2.5.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/985879156810b0df16ce8207a9753e5264e7430f6403a585b7efa2b256f68b45",
    ],
)

rpm(
    name = "nftables-1__1.0.9-1.el9.aarch64",
    sha256 = "95a6440657d2fa3624b6ab161ee1ad25c0345d434cf803e8d1408f3ab91a3ba8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.9-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/95a6440657d2fa3624b6ab161ee1ad25c0345d434cf803e8d1408f3ab91a3ba8",
    ],
)

rpm(
    name = "nftables-1__1.0.9-1.el9.s390x",
    sha256 = "79671afaa8a5d2196ce08668b278f56bffec0a8e484f608ce79093b4249d841e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nftables-1.0.9-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/79671afaa8a5d2196ce08668b278f56bffec0a8e484f608ce79093b4249d841e",
    ],
)

rpm(
    name = "nftables-1__1.0.9-1.el9.x86_64",
    sha256 = "22056e3f92582cdae0c39311e37ad95a693959a82f584faab1ecf82dc0b5fea8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.9-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/22056e3f92582cdae0c39311e37ad95a693959a82f584faab1ecf82dc0b5fea8",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-1.el9.aarch64",
    sha256 = "521d708d7679c793d5fc63d7c51800bfaf39090d36f9eebee63cbf874f2c993f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.92-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/521d708d7679c793d5fc63d7c51800bfaf39090d36f9eebee63cbf874f2c993f",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-1.el9.s390x",
    sha256 = "0c47f84da1ed99168250fc76be7da9484b0f070d8e94eddfb5ea33cd0f4d0d69",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/nmap-ncat-7.92-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0c47f84da1ed99168250fc76be7da9484b0f070d8e94eddfb5ea33cd0f4d0d69",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-1.el9.x86_64",
    sha256 = "59e5378a2a0188793559e27feb1cdf66f195c3b3e1280b5b182a66d7c3803962",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.92-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/59e5378a2a0188793559e27feb1cdf66f195c3b3e1280b5b182a66d7c3803962",
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
    name = "numactl-libs-0__2.0.16-3.el9.aarch64",
    sha256 = "018b1f427fd576c1acd7ba2dd79f74a49ee8afab5670a2519241260ef1466562",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.16-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/018b1f427fd576c1acd7ba2dd79f74a49ee8afab5670a2519241260ef1466562",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.16-3.el9.s390x",
    sha256 = "8b0d263318265a9c172bb7f3f9bccafa9f2f832abe954bcdd78a64d2e4e843f1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/numactl-libs-2.0.16-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8b0d263318265a9c172bb7f3f9bccafa9f2f832abe954bcdd78a64d2e4e843f1",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.16-3.el9.x86_64",
    sha256 = "56167ea50d70d737d28da028f279f42ac4b624f95ef8f5cce05944cb804230af",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.16-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56167ea50d70d737d28da028f279f42ac4b624f95ef8f5cce05944cb804230af",
    ],
)

rpm(
    name = "numad-0__0.5-37.20150602git.el9.aarch64",
    sha256 = "c7f9e7e2d37c5d8ae263e8789142ba6956337a12a139a9661efff6ebfd3758c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numad-0.5-37.20150602git.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7f9e7e2d37c5d8ae263e8789142ba6956337a12a139a9661efff6ebfd3758c4",
    ],
)

rpm(
    name = "numad-0__0.5-37.20150602git.el9.x86_64",
    sha256 = "82e83efcc0528646c0cfdaa846e45e89b6e347b78664b5528bbfdf919d57bd46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numad-0.5-37.20150602git.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82e83efcc0528646c0cfdaa846e45e89b6e347b78664b5528bbfdf919d57bd46",
    ],
)

rpm(
    name = "openldap-0__2.6.6-3.el9.aarch64",
    sha256 = "3553d2a8ef3901444aae018a0f1a17f41d3ffba77af79735e8422c37f6ac57d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openldap-2.6.6-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3553d2a8ef3901444aae018a0f1a17f41d3ffba77af79735e8422c37f6ac57d8",
    ],
)

rpm(
    name = "openldap-0__2.6.6-3.el9.s390x",
    sha256 = "3c675112c68a0aadab98d1f33ca35eaa182a473000433590d5a86b9c55fdd6cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openldap-2.6.6-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3c675112c68a0aadab98d1f33ca35eaa182a473000433590d5a86b9c55fdd6cb",
    ],
)

rpm(
    name = "openldap-0__2.6.6-3.el9.x86_64",
    sha256 = "da4c54a99c4556ab6c95f91ac0f472e8e96509fd97a59f45e196c0f613a1dbab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openldap-2.6.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da4c54a99c4556ab6c95f91ac0f472e8e96509fd97a59f45e196c0f613a1dbab",
    ],
)

rpm(
    name = "openssl-1__3.2.2-2.el9.aarch64",
    sha256 = "01b335a1c3380c7f458404bf036bed73ffeefb179a38b2eae3f432454fdb3797",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.2.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01b335a1c3380c7f458404bf036bed73ffeefb179a38b2eae3f432454fdb3797",
    ],
)

rpm(
    name = "openssl-1__3.2.2-2.el9.s390x",
    sha256 = "34c6413378eb8ac27a108daa8c4ab575e9b49b89ef971fbf942e75f297192bb2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-3.2.2-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/34c6413378eb8ac27a108daa8c4ab575e9b49b89ef971fbf942e75f297192bb2",
    ],
)

rpm(
    name = "openssl-1__3.2.2-2.el9.x86_64",
    sha256 = "220bc122a55b081c8f05094b773875cc47de713a2425edf284d9048373431428",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.2.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/220bc122a55b081c8f05094b773875cc47de713a2425edf284d9048373431428",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-2.el9.aarch64",
    sha256 = "29de503dcbd1c3bfdacb137c6798d76e68abcd76d9f164d819c1bc1f1f78c05b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.2.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/29de503dcbd1c3bfdacb137c6798d76e68abcd76d9f164d819c1bc1f1f78c05b",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-2.el9.s390x",
    sha256 = "9cd15f4aa6968428b952f2feac5b965864086825686fbd9f90a0141d19d17440",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-libs-3.2.2-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9cd15f4aa6968428b952f2feac5b965864086825686fbd9f90a0141d19d17440",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-2.el9.x86_64",
    sha256 = "b060c100f9920714d6923427ed9426ae742c715bc0badfa041c64972ccc5ca45",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.2.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b060c100f9920714d6923427ed9426ae742c715bc0badfa041c64972ccc5ca45",
    ],
)

rpm(
    name = "osinfo-db-0__20231215-1.el9.x86_64",
    sha256 = "88316f64c0d24de55f2291060e0bc4cc8e958f5a846d4bd2c3fd28a9fe29ac22",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/osinfo-db-20231215-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/88316f64c0d24de55f2291060e0bc4cc8e958f5a846d4bd2c3fd28a9fe29ac22",
    ],
)

rpm(
    name = "osinfo-db-tools-0__1.10.0-1.el9.x86_64",
    sha256 = "2681f49bf19314e44e7189852d6fbfc22fc3ed428240df9f3936a5200c14ddd0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/osinfo-db-tools-1.10.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2681f49bf19314e44e7189852d6fbfc22fc3ed428240df9f3936a5200c14ddd0",
    ],
)

rpm(
    name = "p11-kit-0__0.25.3-2.el9.aarch64",
    sha256 = "bdd4c7f279730c079b5f766a5c9f1297ee02120840bd12f084ab6f22e50c2203",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-0.25.3-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bdd4c7f279730c079b5f766a5c9f1297ee02120840bd12f084ab6f22e50c2203",
    ],
)

rpm(
    name = "p11-kit-0__0.25.3-2.el9.s390x",
    sha256 = "fc238839eff36402fdc4fabb7b1a2298f512c384e62c2de5f931f62943a8d595",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-0.25.3-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fc238839eff36402fdc4fabb7b1a2298f512c384e62c2de5f931f62943a8d595",
    ],
)

rpm(
    name = "p11-kit-0__0.25.3-2.el9.x86_64",
    sha256 = "0839ee9854251e66f3109b6c685f1e6b3cce6d2a1415e9f71a03d71f03eeb708",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-0.25.3-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0839ee9854251e66f3109b6c685f1e6b3cce6d2a1415e9f71a03d71f03eeb708",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.25.3-2.el9.aarch64",
    sha256 = "f49208d939702ade5ff36a42af67be05ca5c3125665c23275520880a07f5d16a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.25.3-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f49208d939702ade5ff36a42af67be05ca5c3125665c23275520880a07f5d16a",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.25.3-2.el9.s390x",
    sha256 = "3e506a1cd02aa89b7de59cfb2b1370a8130fa45dd9034f426d6d18bfee42943c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-trust-0.25.3-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3e506a1cd02aa89b7de59cfb2b1370a8130fa45dd9034f426d6d18bfee42943c",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.25.3-2.el9.x86_64",
    sha256 = "177b963e62a19a2539138c1e5828a331bdf04c3675829a0dc88699765a4e0e63",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.25.3-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/177b963e62a19a2539138c1e5828a331bdf04c3675829a0dc88699765a4e0e63",
    ],
)

rpm(
    name = "pam-0__1.5.1-19.el9.aarch64",
    sha256 = "f9b9f35ef6862dc0a4201bb263b37e7c894f8cc97aa8a5e495acf4fab80b5cfa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f9b9f35ef6862dc0a4201bb263b37e7c894f8cc97aa8a5e495acf4fab80b5cfa",
    ],
)

rpm(
    name = "pam-0__1.5.1-19.el9.s390x",
    sha256 = "b18863ffee5e0f9d14dd60add77ba2ca21e2e33a4ccda5edd70023ea3076aeca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pam-1.5.1-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b18863ffee5e0f9d14dd60add77ba2ca21e2e33a4ccda5edd70023ea3076aeca",
    ],
)

rpm(
    name = "pam-0__1.5.1-19.el9.x86_64",
    sha256 = "31dd4a1a7dc7b02d64831c8e2f85c0e96674da48f5d87770959445b1f47abbd4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31dd4a1a7dc7b02d64831c8e2f85c0e96674da48f5d87770959445b1f47abbd4",
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
    name = "passt-0__0__caret__20231204.gb86afe3-1.el9.aarch64",
    sha256 = "5d05fe3e9a9b281512a16a3bf59e2bfa67a2213cb7cbaf3dd946257c37e3e641",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/passt-0%5E20231204.gb86afe3-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5d05fe3e9a9b281512a16a3bf59e2bfa67a2213cb7cbaf3dd946257c37e3e641",
    ],
)

rpm(
    name = "passt-0__0__caret__20231204.gb86afe3-1.el9.s390x",
    sha256 = "d3e11327eb43fa871df67df2d5ad6c8f9a0a9329f578fbed06e4fb40640e99ba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/passt-0%5E20231204.gb86afe3-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d3e11327eb43fa871df67df2d5ad6c8f9a0a9329f578fbed06e4fb40640e99ba",
    ],
)

rpm(
    name = "passt-0__0__caret__20231204.gb86afe3-1.el9.x86_64",
    sha256 = "7033b72eddb6ee05cead546bb3ad326018446c4823cb77c641ce5991fba493b6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/passt-0%5E20231204.gb86afe3-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7033b72eddb6ee05cead546bb3ad326018446c4823cb77c641ce5991fba493b6",
    ],
)

rpm(
    name = "pcre-0__8.44-4.el9.aarch64",
    sha256 = "dc5d71786a68cfa15f49aecd12e90de7af7489a2d0a4d102be38a9faf0c99ae8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre-8.44-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dc5d71786a68cfa15f49aecd12e90de7af7489a2d0a4d102be38a9faf0c99ae8",
    ],
)

rpm(
    name = "pcre-0__8.44-4.el9.s390x",
    sha256 = "e42ebd2b71ed4d5ee34a5fbba116396c22ed4deb7d7ab6189f048a3f603e5dbb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pcre-8.44-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e42ebd2b71ed4d5ee34a5fbba116396c22ed4deb7d7ab6189f048a3f603e5dbb",
    ],
)

rpm(
    name = "pcre-0__8.44-4.el9.x86_64",
    sha256 = "7d6be1d41cb4d0b159a764bfc7c8efecc0353224b46e5286cbbea7092b700690",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre-8.44-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d6be1d41cb4d0b159a764bfc7c8efecc0353224b46e5286cbbea7092b700690",
    ],
)

rpm(
    name = "pcre2-0__10.40-5.el9.aarch64",
    sha256 = "74afde46575a6d9efeb26a3ba63d9204fa92d1fd21aaadb46e4c4f6bd4427dc8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre2-10.40-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/74afde46575a6d9efeb26a3ba63d9204fa92d1fd21aaadb46e4c4f6bd4427dc8",
    ],
)

rpm(
    name = "pcre2-0__10.40-5.el9.s390x",
    sha256 = "b16dfd9ed0221e5dbff8998bd1fe0b6dad0e82019af39c46f57e10d5c6326179",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pcre2-10.40-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b16dfd9ed0221e5dbff8998bd1fe0b6dad0e82019af39c46f57e10d5c6326179",
    ],
)

rpm(
    name = "pcre2-0__10.40-5.el9.x86_64",
    sha256 = "f48e389d6b22577760673f207acc0e95cdf6327e5cfa32e522647d471b37a7c0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre2-10.40-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f48e389d6b22577760673f207acc0e95cdf6327e5cfa32e522647d471b37a7c0",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-5.el9.aarch64",
    sha256 = "9a52a1a58ef7b5feb5720f8cd789da918a0bf70ac42859c6af2dec6856a8b3f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre2-syntax-10.40-5.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9a52a1a58ef7b5feb5720f8cd789da918a0bf70ac42859c6af2dec6856a8b3f0",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-5.el9.s390x",
    sha256 = "9a52a1a58ef7b5feb5720f8cd789da918a0bf70ac42859c6af2dec6856a8b3f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pcre2-syntax-10.40-5.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9a52a1a58ef7b5feb5720f8cd789da918a0bf70ac42859c6af2dec6856a8b3f0",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-5.el9.x86_64",
    sha256 = "9a52a1a58ef7b5feb5720f8cd789da918a0bf70ac42859c6af2dec6856a8b3f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre2-syntax-10.40-5.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9a52a1a58ef7b5feb5720f8cd789da918a0bf70ac42859c6af2dec6856a8b3f0",
    ],
)

rpm(
    name = "pixman-0__0.40.0-6.el9.aarch64",
    sha256 = "949aaa9855119b3372bb4be01b7b2ab87ba9b6c949cad37f411f71553968248f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/pixman-0.40.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/949aaa9855119b3372bb4be01b7b2ab87ba9b6c949cad37f411f71553968248f",
    ],
)

rpm(
    name = "pixman-0__0.40.0-6.el9.s390x",
    sha256 = "8ee2116bc324edfac404192338cfd469373ffba64b1a5c2bfb199d551e922563",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/pixman-0.40.0-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8ee2116bc324edfac404192338cfd469373ffba64b1a5c2bfb199d551e922563",
    ],
)

rpm(
    name = "pixman-0__0.40.0-6.el9.x86_64",
    sha256 = "e5f710c9d8ab38f2286070877560e99a28d3067ac117231e68c9e8cfb5c617de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/pixman-0.40.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e5f710c9d8ab38f2286070877560e99a28d3067ac117231e68c9e8cfb5c617de",
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
    name = "pkgconf-0__1.7.3-10.el9.s390x",
    sha256 = "18b95c0969e2a47a4db32976707227f1d2204f498e904a69c15ae642229f2684",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pkgconf-1.7.3-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/18b95c0969e2a47a4db32976707227f1d2204f498e904a69c15ae642229f2684",
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
    name = "pkgconf-m4-0__1.7.3-10.el9.s390x",
    sha256 = "de4946454f110a9b12ab50c9c3dfaa68633b4ae3cb4e5278b23d491eb3edc27a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pkgconf-m4-1.7.3-10.el9.noarch.rpm",
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
    name = "pkgconf-pkg-config-0__1.7.3-10.el9.s390x",
    sha256 = "d2683075e4d5f2222ae8f9e5c36f1bd5637c07bb9bc9c5fb3aa48914a901f5fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pkgconf-pkg-config-1.7.3-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d2683075e4d5f2222ae8f9e5c36f1bd5637c07bb9bc9c5fb3aa48914a901f5fd",
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
    name = "policycoreutils-0__3.6-2.1.el9.aarch64",
    sha256 = "93270211cc317bdd44706c3a216ebc8155942e349510a3906f26df0d10328d78",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.6-2.1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/93270211cc317bdd44706c3a216ebc8155942e349510a3906f26df0d10328d78",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-2.1.el9.s390x",
    sha256 = "7ccadb5f8c3ecea0e24447211179c90abbb56cb8d52b97e811137a4588d9ce79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/policycoreutils-3.6-2.1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7ccadb5f8c3ecea0e24447211179c90abbb56cb8d52b97e811137a4588d9ce79",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-2.1.el9.x86_64",
    sha256 = "a87874363af6432b1c96b40f8b79b90616df22bff3bd4f9aa39da24f5bddd3e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.6-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a87874363af6432b1c96b40f8b79b90616df22bff3bd4f9aa39da24f5bddd3e9",
    ],
)

rpm(
    name = "polkit-0__0.117-13.el9.aarch64",
    sha256 = "f80beec26bc1ccd464ccfc1c692f9bb0ff04fee6000b4f10948e88b36de1149d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-0.117-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f80beec26bc1ccd464ccfc1c692f9bb0ff04fee6000b4f10948e88b36de1149d",
    ],
)

rpm(
    name = "polkit-0__0.117-13.el9.s390x",
    sha256 = "b80af496f6394f3758d7636f406473ecf8352186488c5cd7626d8b6b3f445c3c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/polkit-0.117-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b80af496f6394f3758d7636f406473ecf8352186488c5cd7626d8b6b3f445c3c",
    ],
)

rpm(
    name = "polkit-0__0.117-13.el9.x86_64",
    sha256 = "81090043c437cb6e6a73b4f72a6d9d5980d99fbb8a176ca36647a8d5f1cd4db4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-0.117-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/81090043c437cb6e6a73b4f72a6d9d5980d99fbb8a176ca36647a8d5f1cd4db4",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-13.el9.aarch64",
    sha256 = "d8bbf2c31e641fdec12dc572497dc7756a6b1fe0c5f24133ada81a6ebf89b556",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.117-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d8bbf2c31e641fdec12dc572497dc7756a6b1fe0c5f24133ada81a6ebf89b556",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-13.el9.s390x",
    sha256 = "a4e49dcaa9ec165ea4d67faa1bd96fac28bfc705b3fa1a9f5fc7cce0388ecb34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/polkit-libs-0.117-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a4e49dcaa9ec165ea4d67faa1bd96fac28bfc705b3fa1a9f5fc7cce0388ecb34",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-13.el9.x86_64",
    sha256 = "127d13c1e41ca8f5e82bb8d453351aa3c48376e00c4d659b5d0de414dcfd4fd4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.117-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/127d13c1e41ca8f5e82bb8d453351aa3c48376e00c4d659b5d0de414dcfd4fd4",
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
    name = "popt-0__1.18-8.el9.s390x",
    sha256 = "b2bc4dbd78a6c3b9458cbc022e80d860fb2c6022fa308604f553289b62cb9511",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/popt-1.18-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b2bc4dbd78a6c3b9458cbc022e80d860fb2c6022fa308604f553289b62cb9511",
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
    name = "procps-ng-0__3.3.17-14.el9.aarch64",
    sha256 = "a79af64966d8bf303d3bd14396df577826f082679f3acdfeaf1bb9a9048be6fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.17-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a79af64966d8bf303d3bd14396df577826f082679f3acdfeaf1bb9a9048be6fb",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-14.el9.s390x",
    sha256 = "3eaf08992132ad2a4b7c924593a8f3ab871967374a96734764941cb9aae7f191",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/procps-ng-3.3.17-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3eaf08992132ad2a4b7c924593a8f3ab871967374a96734764941cb9aae7f191",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-14.el9.x86_64",
    sha256 = "e2ab525ae66c31122005fc8e6eb836d7eb3336280e8ccfff2ca98165a11a482b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.17-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e2ab525ae66c31122005fc8e6eb836d7eb3336280e8ccfff2ca98165a11a482b",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-13.el9.aarch64",
    sha256 = "7293996e2cbb1fabb43c5c156fa37c22558a73125ebdfe036e2338ca18a319c8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/protobuf-c-1.3.3-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7293996e2cbb1fabb43c5c156fa37c22558a73125ebdfe036e2338ca18a319c8",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-13.el9.s390x",
    sha256 = "a34d3241e8c90dc1122056fce571bc3042f08a4fc12a0b58b3303c1973c38488",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/protobuf-c-1.3.3-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a34d3241e8c90dc1122056fce571bc3042f08a4fc12a0b58b3303c1973c38488",
    ],
)

rpm(
    name = "protobuf-c-0__1.3.3-13.el9.x86_64",
    sha256 = "3a4af8395499f19ebebc1cd928cd01fb96e05173e3a5d03d8e981c04b0042409",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/protobuf-c-1.3.3-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a4af8395499f19ebebc1cd928cd01fb96e05173e3a5d03d8e981c04b0042409",
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
    name = "psmisc-0__23.4-3.el9.s390x",
    sha256 = "2d538437d62a278205126b5c4808feae4fdf6cb873519b68f4cfa6657686579f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/psmisc-23.4-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2d538437d62a278205126b5c4808feae4fdf6cb873519b68f4cfa6657686579f",
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
    name = "publicsuffix-list-dafsa-0__20210518-3.el9.x86_64",
    sha256 = "992c17312bf5f144ec17b3c9733ab180c6c3641323d2deaf7c13e6bd1971f7a6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/publicsuffix-list-dafsa-20210518-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/992c17312bf5f144ec17b3c9733ab180c6c3641323d2deaf7c13e6bd1971f7a6",
    ],
)

rpm(
    name = "python3-0__3.9.19-1.el9.aarch64",
    sha256 = "2eccdf4dd18ae2987629b9216fb229d69fa7cd3e2a458d5e4feb54b97b0a6595",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.19-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2eccdf4dd18ae2987629b9216fb229d69fa7cd3e2a458d5e4feb54b97b0a6595",
    ],
)

rpm(
    name = "python3-0__3.9.19-1.el9.s390x",
    sha256 = "8207458fa78cd27cfe2ef9f960eff3041567b0ddcd0c8237dead9d2127d0b3bc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-3.9.19-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8207458fa78cd27cfe2ef9f960eff3041567b0ddcd0c8237dead9d2127d0b3bc",
    ],
)

rpm(
    name = "python3-0__3.9.19-1.el9.x86_64",
    sha256 = "794d3ec7ca74b2b3a150864f81b0ae81c860d7f4cebe9386fabb0c57779fde42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.19-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/794d3ec7ca74b2b3a150864f81b0ae81c860d7f4cebe9386fabb0c57779fde42",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-1.el9.aarch64",
    sha256 = "15575ccf52609db52e8535ebdd52f64f2fc9f599a2b4f0ac79d2c3f49aa32cd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-configshell-1.1.30-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/15575ccf52609db52e8535ebdd52f64f2fc9f599a2b4f0ac79d2c3f49aa32cd1",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-1.el9.s390x",
    sha256 = "15575ccf52609db52e8535ebdd52f64f2fc9f599a2b4f0ac79d2c3f49aa32cd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-configshell-1.1.30-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/15575ccf52609db52e8535ebdd52f64f2fc9f599a2b4f0ac79d2c3f49aa32cd1",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-1.el9.x86_64",
    sha256 = "15575ccf52609db52e8535ebdd52f64f2fc9f599a2b4f0ac79d2c3f49aa32cd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-configshell-1.1.30-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/15575ccf52609db52e8535ebdd52f64f2fc9f599a2b4f0ac79d2c3f49aa32cd1",
    ],
)

rpm(
    name = "python3-dbus-0__1.2.18-2.el9.aarch64",
    sha256 = "ce454fa8f9a2d015face9e9ae64f6730f2ba104d0556c91b93fca2006f132bf9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-dbus-1.2.18-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ce454fa8f9a2d015face9e9ae64f6730f2ba104d0556c91b93fca2006f132bf9",
    ],
)

rpm(
    name = "python3-dbus-0__1.2.18-2.el9.s390x",
    sha256 = "6285fd8cbd484311a0e9f6b4fef4c8b0892b468f3633e49b3c93061fb6a0b360",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-dbus-1.2.18-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6285fd8cbd484311a0e9f6b4fef4c8b0892b468f3633e49b3c93061fb6a0b360",
    ],
)

rpm(
    name = "python3-dbus-0__1.2.18-2.el9.x86_64",
    sha256 = "8e42f3e54292bfc76ab52ee3f91f850fb0cca63c9a49692938381ca93460a686",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-dbus-1.2.18-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e42f3e54292bfc76ab52ee3f91f850fb0cca63c9a49692938381ca93460a686",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.40.1-6.el9.aarch64",
    sha256 = "815369710e8d7c6f7473380210283f9e6dfdc0c6cc553c4ea9cb709835937adb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-gobject-base-3.40.1-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/815369710e8d7c6f7473380210283f9e6dfdc0c6cc553c4ea9cb709835937adb",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.40.1-6.el9.s390x",
    sha256 = "7a4cfa43d12f5afd3035e4c92395acae04b4e8c397f188dee4f6fa4c933db263",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-gobject-base-3.40.1-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7a4cfa43d12f5afd3035e4c92395acae04b4e8c397f188dee4f6fa4c933db263",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.40.1-6.el9.x86_64",
    sha256 = "bb795c9ba439bd1a0329e3534001432c95c5c454ccc61029f68501006f539a51",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-gobject-base-3.40.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb795c9ba439bd1a0329e3534001432c95c5c454ccc61029f68501006f539a51",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.40.1-6.el9.aarch64",
    sha256 = "57ae14f5296ed26cabd264a2b88a015b05f962b65c9633eb328da029a0372b01",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-gobject-base-noarch-3.40.1-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/57ae14f5296ed26cabd264a2b88a015b05f962b65c9633eb328da029a0372b01",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.40.1-6.el9.s390x",
    sha256 = "57ae14f5296ed26cabd264a2b88a015b05f962b65c9633eb328da029a0372b01",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-gobject-base-noarch-3.40.1-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/57ae14f5296ed26cabd264a2b88a015b05f962b65c9633eb328da029a0372b01",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.40.1-6.el9.x86_64",
    sha256 = "57ae14f5296ed26cabd264a2b88a015b05f962b65c9633eb328da029a0372b01",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-gobject-base-noarch-3.40.1-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/57ae14f5296ed26cabd264a2b88a015b05f962b65c9633eb328da029a0372b01",
    ],
)

rpm(
    name = "python3-kmod-0__0.9-32.el9.aarch64",
    sha256 = "600b42a5b139ea5f8b246561294581c09237d88ec9bbcce823f56213d7e2652f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-kmod-0.9-32.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/600b42a5b139ea5f8b246561294581c09237d88ec9bbcce823f56213d7e2652f",
    ],
)

rpm(
    name = "python3-kmod-0__0.9-32.el9.s390x",
    sha256 = "d26e61644fa735dc2e63b8793f9bc549d4476a07c77dc587457e86487e0363d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-kmod-0.9-32.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d26e61644fa735dc2e63b8793f9bc549d4476a07c77dc587457e86487e0363d4",
    ],
)

rpm(
    name = "python3-kmod-0__0.9-32.el9.x86_64",
    sha256 = "e0b0ae0d507496349b667e0281b4d72ac3f7b7fa65c633c56afa3f328855a2d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-kmod-0.9-32.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e0b0ae0d507496349b667e0281b4d72ac3f7b7fa65c633c56afa3f328855a2d9",
    ],
)

rpm(
    name = "python3-libs-0__3.9.19-1.el9.aarch64",
    sha256 = "f9420cac72d33e7620d257dc98dc3e8915a1fe2cb6e005afae9bcd67002a23a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.19-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f9420cac72d33e7620d257dc98dc3e8915a1fe2cb6e005afae9bcd67002a23a8",
    ],
)

rpm(
    name = "python3-libs-0__3.9.19-1.el9.s390x",
    sha256 = "f9fd4b76d65766c2d7829519aa87a4a2e01a0cd79bd1c0b25cfe0733b9af4f50",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-libs-3.9.19-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f9fd4b76d65766c2d7829519aa87a4a2e01a0cd79bd1c0b25cfe0733b9af4f50",
    ],
)

rpm(
    name = "python3-libs-0__3.9.19-1.el9.x86_64",
    sha256 = "388f9e5a1167ab2c480295bae60b4bad1854b6e6ec17d253b7a8a0647fc2fe1d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.19-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/388f9e5a1167ab2c480295bae60b4bad1854b6e6ec17d253b7a8a0647fc2fe1d",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.3.1-1.el9.aarch64",
    sha256 = "1c8096f1dd57c5d6db4d1391cafb15326431923ba139f3119015773a307f80d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-21.3.1-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1c8096f1dd57c5d6db4d1391cafb15326431923ba139f3119015773a307f80d9",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.3.1-1.el9.s390x",
    sha256 = "1c8096f1dd57c5d6db4d1391cafb15326431923ba139f3119015773a307f80d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-pip-wheel-21.3.1-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1c8096f1dd57c5d6db4d1391cafb15326431923ba139f3119015773a307f80d9",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.3.1-1.el9.x86_64",
    sha256 = "1c8096f1dd57c5d6db4d1391cafb15326431923ba139f3119015773a307f80d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-21.3.1-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1c8096f1dd57c5d6db4d1391cafb15326431923ba139f3119015773a307f80d9",
    ],
)

rpm(
    name = "python3-pyparsing-0__2.4.7-9.el9.aarch64",
    sha256 = "ee20a60fb835392fc76c1a1a3e9befa0e4b3d27bdcfbfb0aab90fcddf3c60439",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-pyparsing-2.4.7-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ee20a60fb835392fc76c1a1a3e9befa0e4b3d27bdcfbfb0aab90fcddf3c60439",
    ],
)

rpm(
    name = "python3-pyparsing-0__2.4.7-9.el9.s390x",
    sha256 = "ee20a60fb835392fc76c1a1a3e9befa0e4b3d27bdcfbfb0aab90fcddf3c60439",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-pyparsing-2.4.7-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ee20a60fb835392fc76c1a1a3e9befa0e4b3d27bdcfbfb0aab90fcddf3c60439",
    ],
)

rpm(
    name = "python3-pyparsing-0__2.4.7-9.el9.x86_64",
    sha256 = "ee20a60fb835392fc76c1a1a3e9befa0e4b3d27bdcfbfb0aab90fcddf3c60439",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pyparsing-2.4.7-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ee20a60fb835392fc76c1a1a3e9befa0e4b3d27bdcfbfb0aab90fcddf3c60439",
    ],
)

rpm(
    name = "python3-pyudev-0__0.22.0-6.el9.aarch64",
    sha256 = "db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-pyudev-0.22.0-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    ],
)

rpm(
    name = "python3-pyudev-0__0.22.0-6.el9.s390x",
    sha256 = "db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-pyudev-0.22.0-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    ],
)

rpm(
    name = "python3-pyudev-0__0.22.0-6.el9.x86_64",
    sha256 = "db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pyudev-0.22.0-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-1.el9.aarch64",
    sha256 = "2cc7a615005d44835de5d211208722d9ebc7e5d36ad62632c5773a301ef6f0d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-rtslib-2.1.76-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2cc7a615005d44835de5d211208722d9ebc7e5d36ad62632c5773a301ef6f0d2",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-1.el9.s390x",
    sha256 = "2cc7a615005d44835de5d211208722d9ebc7e5d36ad62632c5773a301ef6f0d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-rtslib-2.1.76-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2cc7a615005d44835de5d211208722d9ebc7e5d36ad62632c5773a301ef6f0d2",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-1.el9.x86_64",
    sha256 = "2cc7a615005d44835de5d211208722d9ebc7e5d36ad62632c5773a301ef6f0d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-rtslib-2.1.76-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2cc7a615005d44835de5d211208722d9ebc7e5d36ad62632c5773a301ef6f0d2",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-12.el9.aarch64",
    sha256 = "de1a05afcb6087cf6fc6e38b952485239a72ae719538bd255e14789e606ab2ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-53.0.0-12.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de1a05afcb6087cf6fc6e38b952485239a72ae719538bd255e14789e606ab2ca",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-12.el9.s390x",
    sha256 = "de1a05afcb6087cf6fc6e38b952485239a72ae719538bd255e14789e606ab2ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-setuptools-wheel-53.0.0-12.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de1a05afcb6087cf6fc6e38b952485239a72ae719538bd255e14789e606ab2ca",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-12.el9.x86_64",
    sha256 = "de1a05afcb6087cf6fc6e38b952485239a72ae719538bd255e14789e606ab2ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-53.0.0-12.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de1a05afcb6087cf6fc6e38b952485239a72ae719538bd255e14789e606ab2ca",
    ],
)

rpm(
    name = "python3-six-0__1.15.0-9.el9.aarch64",
    sha256 = "efecffed29602079a1ea1d41c819271ec705a97a68891b43e1d626b2fa0ea8a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-six-1.15.0-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/efecffed29602079a1ea1d41c819271ec705a97a68891b43e1d626b2fa0ea8a1",
    ],
)

rpm(
    name = "python3-six-0__1.15.0-9.el9.s390x",
    sha256 = "efecffed29602079a1ea1d41c819271ec705a97a68891b43e1d626b2fa0ea8a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-six-1.15.0-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/efecffed29602079a1ea1d41c819271ec705a97a68891b43e1d626b2fa0ea8a1",
    ],
)

rpm(
    name = "python3-six-0__1.15.0-9.el9.x86_64",
    sha256 = "efecffed29602079a1ea1d41c819271ec705a97a68891b43e1d626b2fa0ea8a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-six-1.15.0-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/efecffed29602079a1ea1d41c819271ec705a97a68891b43e1d626b2fa0ea8a1",
    ],
)

rpm(
    name = "python3-urwid-0__2.1.2-4.el9.aarch64",
    sha256 = "a91fcc1b5b01aeb0830d04f562cb843489f38d2606d8ab480a876207f4335990",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-urwid-2.1.2-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a91fcc1b5b01aeb0830d04f562cb843489f38d2606d8ab480a876207f4335990",
    ],
)

rpm(
    name = "python3-urwid-0__2.1.2-4.el9.s390x",
    sha256 = "8c2347f24774578aee45917782ca5e535cdb5eb0bc12a8bbf301a8cc71174ab7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-urwid-2.1.2-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8c2347f24774578aee45917782ca5e535cdb5eb0bc12a8bbf301a8cc71174ab7",
    ],
)

rpm(
    name = "python3-urwid-0__2.1.2-4.el9.x86_64",
    sha256 = "b4e4915a49904035e0e8d8ed15a545f2d7191e9d760c438343980fbf0b66abf4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-urwid-2.1.2-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4e4915a49904035e0e8d8ed15a545f2d7191e9d760c438343980fbf0b66abf4",
    ],
)

rpm(
    name = "qemu-img-17__8.2.0-11.el9.aarch64",
    sha256 = "1524eca552f738a1c90a410c1fb1fc93dd892916619c7e7617d69598bd4f1faf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1524eca552f738a1c90a410c1fb1fc93dd892916619c7e7617d69598bd4f1faf",
    ],
)

rpm(
    name = "qemu-img-17__8.2.0-11.el9.s390x",
    sha256 = "64b0eba3638b09bce02291a34b30095408af10b560107df8abfc16e078a7690a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-img-8.2.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/64b0eba3638b09bce02291a34b30095408af10b560107df8abfc16e078a7690a",
    ],
)

rpm(
    name = "qemu-img-17__8.2.0-11.el9.x86_64",
    sha256 = "0f6353565bd7ad18e6a26130b940ed6cc397f5a6b022f25ca5766b7fb412b9ba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-8.2.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0f6353565bd7ad18e6a26130b940ed6cc397f5a6b022f25ca5766b7fb412b9ba",
    ],
)

rpm(
    name = "qemu-kvm-common-17__8.2.0-11.el9.aarch64",
    sha256 = "f3dff9b5df26de74953d0fab08c58c93c8619795d46eba93586acf84c6bd8039",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f3dff9b5df26de74953d0fab08c58c93c8619795d46eba93586acf84c6bd8039",
    ],
)

rpm(
    name = "qemu-kvm-common-17__8.2.0-11.el9.s390x",
    sha256 = "d3633036c1622e62ba1ccb26546fca1652c1299c1f6d09259e5781f74351fecb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-common-8.2.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d3633036c1622e62ba1ccb26546fca1652c1299c1f6d09259e5781f74351fecb",
    ],
)

rpm(
    name = "qemu-kvm-common-17__8.2.0-11.el9.x86_64",
    sha256 = "c12bc9f1c56fa8667ef2f39b3fdc3930c2874b449ca98d229107044a42ae4239",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-8.2.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c12bc9f1c56fa8667ef2f39b3fdc3930c2874b449ca98d229107044a42ae4239",
    ],
)

rpm(
    name = "qemu-kvm-core-17__8.2.0-11.el9.aarch64",
    sha256 = "7d612942debf569c757ed8adda60365195b9130848b9730b1d646c5b766fd1f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7d612942debf569c757ed8adda60365195b9130848b9730b1d646c5b766fd1f4",
    ],
)

rpm(
    name = "qemu-kvm-core-17__8.2.0-11.el9.s390x",
    sha256 = "756ddb8c4e910194357a8a5413faf9ea7bb1febe1a0a63c1798d32e652c40220",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-core-8.2.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/756ddb8c4e910194357a8a5413faf9ea7bb1febe1a0a63c1798d32e652c40220",
    ],
)

rpm(
    name = "qemu-kvm-core-17__8.2.0-11.el9.x86_64",
    sha256 = "4ecc58865c06728717a0b7c4d13b757eac0be05c6e20c3c05a50a411691dec63",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-8.2.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4ecc58865c06728717a0b7c4d13b757eac0be05c6e20c3c05a50a411691dec63",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__8.2.0-11.el9.aarch64",
    sha256 = "c8fe5bc44160e3de6216268b15e3804d5ce98c3378987ce8a37fe2688aa7fc24",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c8fe5bc44160e3de6216268b15e3804d5ce98c3378987ce8a37fe2688aa7fc24",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__8.2.0-11.el9.s390x",
    sha256 = "29ed2f75cb0d1c9813e9b605c256c2a5b54374b31dc3ec01a496fe704cf01871",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-8.2.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29ed2f75cb0d1c9813e9b605c256c2a5b54374b31dc3ec01a496fe704cf01871",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-ccw-17__8.2.0-11.el9.s390x",
    sha256 = "e36616cb1c963fb09bc58687b66db4a47769c5bbbddbbd33fb26e7e95a223f76",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-8.2.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e36616cb1c963fb09bc58687b66db4a47769c5bbbddbbd33fb26e7e95a223f76",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__8.2.0-11.el9.aarch64",
    sha256 = "76b2101c08d901a14d868c0d22aa9b198a58ad65cecdbfc03df56245859b456f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76b2101c08d901a14d868c0d22aa9b198a58ad65cecdbfc03df56245859b456f",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__8.2.0-11.el9.aarch64",
    sha256 = "9f775f9c4ec7a5251cc4d3c3af6a694f7e977b2ff089a5930793f557ff4b84f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9f775f9c4ec7a5251cc4d3c3af6a694f7e977b2ff089a5930793f557ff4b84f0",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__8.2.0-11.el9.s390x",
    sha256 = "eb1f6adb1a70f6bd74c01c0d347b161b3cac760b0b9fdf90f395e86ec4373511",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-8.2.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eb1f6adb1a70f6bd74c01c0d347b161b3cac760b0b9fdf90f395e86ec4373511",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__8.2.0-11.el9.x86_64",
    sha256 = "820f0085504707fa5b7bcacc6e8fad587f427d759a0d041c45fe38bf0533beb7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-8.2.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/820f0085504707fa5b7bcacc6e8fad587f427d759a0d041c45fe38bf0533beb7",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__8.2.0-11.el9.aarch64",
    sha256 = "a7f7224ea096e1b5ddb18409725f210946005078b86d2f007347b03ee488f2af",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-8.2.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a7f7224ea096e1b5ddb18409725f210946005078b86d2f007347b03ee488f2af",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__8.2.0-11.el9.x86_64",
    sha256 = "498adf3f4c9bd72dea7146a55dadf06075111caa57ee5528d2a72ceb8cda7a0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-8.2.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/498adf3f4c9bd72dea7146a55dadf06075111caa57ee5528d2a72ceb8cda7a0a",
    ],
)

rpm(
    name = "qemu-pr-helper-17__9.0.0-3.el9.aarch64",
    sha256 = "038b0510d6dad108df4a04af9bfdb2b561687dd4b9274963338f78c78c74b982",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-9.0.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/038b0510d6dad108df4a04af9bfdb2b561687dd4b9274963338f78c78c74b982",
    ],
)

rpm(
    name = "qemu-pr-helper-17__9.0.0-3.el9.x86_64",
    sha256 = "1a179cd578b4d21e095da85db2c35c537e34dc248d10182259a62fb12d8043c7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-9.0.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1a179cd578b4d21e095da85db2c35c537e34dc248d10182259a62fb12d8043c7",
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
    name = "readline-0__8.1-4.el9.s390x",
    sha256 = "7b4b6f641f65d99d33ccbefaf4fbfe25a146d80213d359940779be4ad29569a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/readline-8.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7b4b6f641f65d99d33ccbefaf4fbfe25a146d80213d359940779be4ad29569a8",
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
    name = "rpcbind-0__1.2.6-7.el9.x86_64",
    sha256 = "d5ac02c7b98f72c3c06ce9aff0efa220faf0f36f86db83a6fb16eaccf7427b07",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpcbind-1.2.6-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5ac02c7b98f72c3c06ce9aff0efa220faf0f36f86db83a6fb16eaccf7427b07",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-30.el9.aarch64",
    sha256 = "1519fb4dde7aa054c4bc15f301d4d955fc632e831144ef7d29babe58481eb6fc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-4.16.1.3-30.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1519fb4dde7aa054c4bc15f301d4d955fc632e831144ef7d29babe58481eb6fc",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-30.el9.s390x",
    sha256 = "15dc28a83b9c3feee68a0769a41fbb79d0c88cd2d6e7037e7e7268a9ea04f801",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-4.16.1.3-30.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/15dc28a83b9c3feee68a0769a41fbb79d0c88cd2d6e7037e7e7268a9ea04f801",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-30.el9.x86_64",
    sha256 = "65afdf1b03a74026cc0fe115cc28ce5f6c306603355871f184fc6e7c20dfb1b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-30.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65afdf1b03a74026cc0fe115cc28ce5f6c306603355871f184fc6e7c20dfb1b2",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-30.el9.aarch64",
    sha256 = "cfd83091f22d8cdf79cf01d4d7a58b7d14b1731aadce90ceb80c6d7cd9ca41ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.16.1.3-30.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cfd83091f22d8cdf79cf01d4d7a58b7d14b1731aadce90ceb80c6d7cd9ca41ad",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-30.el9.s390x",
    sha256 = "de653b525035939ac6ea8c7d2987cffdd17f98dd9518873ef8faf6a5979e26bc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-libs-4.16.1.3-30.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/de653b525035939ac6ea8c7d2987cffdd17f98dd9518873ef8faf6a5979e26bc",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-30.el9.x86_64",
    sha256 = "b228ef91f74e47c2fbb419689a4c93e155ef9da2eb9c9dc0d773831b9928738c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-30.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b228ef91f74e47c2fbb419689a4c93e155ef9da2eb9c9dc0d773831b9928738c",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-30.el9.aarch64",
    sha256 = "6b35d8a01d114580321305aadd97b83758450495a0e7ed72a37479d800e1b24a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.16.1.3-30.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6b35d8a01d114580321305aadd97b83758450495a0e7ed72a37479d800e1b24a",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-30.el9.s390x",
    sha256 = "b0842cad58f484f311d0555fd3fa977526a130b6a980049f5b124339c1dae884",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-plugin-selinux-4.16.1.3-30.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b0842cad58f484f311d0555fd3fa977526a130b6a980049f5b124339c1dae884",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-30.el9.x86_64",
    sha256 = "00afc99b8030a46ed349c4b93149a33553571bb337a00cddc9dc42fa5d715b32",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-30.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/00afc99b8030a46ed349c4b93149a33553571bb337a00cddc9dc42fa5d715b32",
    ],
)

rpm(
    name = "scrub-0__2.6.1-4.el9.x86_64",
    sha256 = "cda882a3418a7dec3ab58fa7d96084bdf27067997d5dd23023a52d25c5a9f7f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/scrub-2.6.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cda882a3418a7dec3ab58fa7d96084bdf27067997d5dd23023a52d25c5a9f7f3",
    ],
)

rpm(
    name = "seabios-0__1.16.3-2.el9.x86_64",
    sha256 = "dab0195d5ab91240336ee5299a968e97c876dc85a2ca0f1df9d0bc6041ebc271",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-1.16.3-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dab0195d5ab91240336ee5299a968e97c876dc85a2ca0f1df9d0bc6041ebc271",
    ],
)

rpm(
    name = "seabios-bin-0__1.16.3-2.el9.x86_64",
    sha256 = "0af9794667eb8dbcd1960b10e3313902395f1ef0fde90f67cbce6271d3ee3321",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-bin-1.16.3-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0af9794667eb8dbcd1960b10e3313902395f1ef0fde90f67cbce6271d3ee3321",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.16.3-2.el9.x86_64",
    sha256 = "95d4fc40e4510f9f08abf206d8d5815cc5e7da866273d6eae2575e444ef86562",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seavgabios-bin-1.16.3-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/95d4fc40e4510f9f08abf206d8d5815cc5e7da866273d6eae2575e444ef86562",
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
    name = "sed-0__4.8-9.el9.s390x",
    sha256 = "7185b39912949fe56bc0a9bd6463b1c2dc1206efa00dadecfd6e37c9028e1575",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sed-4.8-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7185b39912949fe56bc0a9bd6463b1c2dc1206efa00dadecfd6e37c9028e1575",
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
    name = "selinux-policy-0__38.1.39-1.el9.aarch64",
    sha256 = "0cbe7b931a90a8f07be10d35f691a0debe07ca468b0e6f7fae2bcecf667eece8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-38.1.39-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0cbe7b931a90a8f07be10d35f691a0debe07ca468b0e6f7fae2bcecf667eece8",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.39-1.el9.s390x",
    sha256 = "0cbe7b931a90a8f07be10d35f691a0debe07ca468b0e6f7fae2bcecf667eece8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-38.1.39-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0cbe7b931a90a8f07be10d35f691a0debe07ca468b0e6f7fae2bcecf667eece8",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.39-1.el9.x86_64",
    sha256 = "0cbe7b931a90a8f07be10d35f691a0debe07ca468b0e6f7fae2bcecf667eece8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-38.1.39-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0cbe7b931a90a8f07be10d35f691a0debe07ca468b0e6f7fae2bcecf667eece8",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.39-1.el9.aarch64",
    sha256 = "2faf7ba9cc662b38aaf3fcd712d1d2d97f7a266a7be1ef78a0aa5932985b13b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-38.1.39-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2faf7ba9cc662b38aaf3fcd712d1d2d97f7a266a7be1ef78a0aa5932985b13b1",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.39-1.el9.s390x",
    sha256 = "2faf7ba9cc662b38aaf3fcd712d1d2d97f7a266a7be1ef78a0aa5932985b13b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-38.1.39-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2faf7ba9cc662b38aaf3fcd712d1d2d97f7a266a7be1ef78a0aa5932985b13b1",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.39-1.el9.x86_64",
    sha256 = "2faf7ba9cc662b38aaf3fcd712d1d2d97f7a266a7be1ef78a0aa5932985b13b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-38.1.39-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2faf7ba9cc662b38aaf3fcd712d1d2d97f7a266a7be1ef78a0aa5932985b13b1",
    ],
)

rpm(
    name = "setup-0__2.13.7-10.el9.aarch64",
    sha256 = "42a1c5a415c44e3b55551f49595c087e2ba55f0fd9ece8056b791983601b76d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/setup-2.13.7-10.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/42a1c5a415c44e3b55551f49595c087e2ba55f0fd9ece8056b791983601b76d2",
    ],
)

rpm(
    name = "setup-0__2.13.7-10.el9.s390x",
    sha256 = "42a1c5a415c44e3b55551f49595c087e2ba55f0fd9ece8056b791983601b76d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/setup-2.13.7-10.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/42a1c5a415c44e3b55551f49595c087e2ba55f0fd9ece8056b791983601b76d2",
    ],
)

rpm(
    name = "setup-0__2.13.7-10.el9.x86_64",
    sha256 = "42a1c5a415c44e3b55551f49595c087e2ba55f0fd9ece8056b791983601b76d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/setup-2.13.7-10.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/42a1c5a415c44e3b55551f49595c087e2ba55f0fd9ece8056b791983601b76d2",
    ],
)

rpm(
    name = "sevctl-0__0.1.0-4.el9.aarch64",
    sha256 = "10a9ace255a5b84c2e89b413c08e24894470bfec6f6c790ea073b6fa3df7ee7a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/sevctl-0.1.0-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/10a9ace255a5b84c2e89b413c08e24894470bfec6f6c790ea073b6fa3df7ee7a",
    ],
)

rpm(
    name = "sevctl-0__0.1.0-4.el9.s390x",
    sha256 = "1f9c055b710e3840ea16027f68699587459cf4132e3509aff5db0c4dd7af10dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/sevctl-0.1.0-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1f9c055b710e3840ea16027f68699587459cf4132e3509aff5db0c4dd7af10dc",
    ],
)

rpm(
    name = "sevctl-0__0.4.2-1.el9.x86_64",
    sha256 = "3a365631679a0ebf367ba1701235019c6d04e2a92233035409b8ee84b0b54297",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/sevctl-0.4.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a365631679a0ebf367ba1701235019c6d04e2a92233035409b8ee84b0b54297",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-8.el9.aarch64",
    sha256 = "e425a9b6b5ba059e0d633f9193b83db4e0bef7f9c4f5b8dbeef41bbb153d6162",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.9-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e425a9b6b5ba059e0d633f9193b83db4e0bef7f9c4f5b8dbeef41bbb153d6162",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-8.el9.s390x",
    sha256 = "69cecc2c4887d641d88abed1921085279364de02e7b090f87a6bb7721071f501",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/shadow-utils-4.9-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/69cecc2c4887d641d88abed1921085279364de02e7b090f87a6bb7721071f501",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-8.el9.x86_64",
    sha256 = "d656b38df69084201a459e9d7084e3653a58b238a7c947e465b8db6c31104261",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d656b38df69084201a459e9d7084e3653a58b238a7c947e465b8db6c31104261",
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
    name = "snappy-0__1.1.8-8.el9.s390x",
    sha256 = "e048f5d0966c06eeffb85bc0c26823e1f9af7b7659365e216839e41c2cb1dcaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/snappy-1.1.8-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e048f5d0966c06eeffb85bc0c26823e1f9af7b7659365e216839e41c2cb1dcaa",
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
    name = "sqlite-libs-0__3.34.1-7.el9.aarch64",
    sha256 = "f8ffaf1f7ca932f6565754d4c6327f58f41ff4fa7239394b6ad593641dd6ce74",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.34.1-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f8ffaf1f7ca932f6565754d4c6327f58f41ff4fa7239394b6ad593641dd6ce74",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-7.el9.s390x",
    sha256 = "00136bb1b209b112853b5e2217966276c1cf24c115028afa99f5eb1389984790",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sqlite-libs-3.34.1-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/00136bb1b209b112853b5e2217966276c1cf24c115028afa99f5eb1389984790",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-7.el9.x86_64",
    sha256 = "eddc9570ff3c2f672034888a57eac371e166671fee8300c3c4976324d502a00f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eddc9570ff3c2f672034888a57eac371e166671fee8300c3c4976324d502a00f",
    ],
)

rpm(
    name = "sssd-client-0__2.9.5-1.el9.aarch64",
    sha256 = "ddfd06ffa5a8b655cc1d3e4d24192cbfb777b3b80b1e6d5e4032d9dee525a989",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.9.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ddfd06ffa5a8b655cc1d3e4d24192cbfb777b3b80b1e6d5e4032d9dee525a989",
    ],
)

rpm(
    name = "sssd-client-0__2.9.5-1.el9.s390x",
    sha256 = "fca331e61f8bd4ea2b4b35cb24565b70c3bb3d395502557c6726f7f6777dd2b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sssd-client-2.9.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fca331e61f8bd4ea2b4b35cb24565b70c3bb3d395502557c6726f7f6777dd2b8",
    ],
)

rpm(
    name = "sssd-client-0__2.9.5-1.el9.x86_64",
    sha256 = "216776c4529f838959c8cd0a53224f466349fd975cc9a95ec464580e8cf3cbfa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.9.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/216776c4529f838959c8cd0a53224f466349fd975cc9a95ec464580e8cf3cbfa",
    ],
)

rpm(
    name = "swtpm-0__0.8.0-1.el9.aarch64",
    sha256 = "7ccdddbd8dab7287094dddfee27e1791ffa9b8593611d90a68f9b5e3827389c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-0.8.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7ccdddbd8dab7287094dddfee27e1791ffa9b8593611d90a68f9b5e3827389c3",
    ],
)

rpm(
    name = "swtpm-0__0.8.0-1.el9.s390x",
    sha256 = "f1223458c51d36d75071384dec6801ee68f5de7ee0b9976cb8a21be900df5794",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/swtpm-0.8.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f1223458c51d36d75071384dec6801ee68f5de7ee0b9976cb8a21be900df5794",
    ],
)

rpm(
    name = "swtpm-0__0.8.0-1.el9.x86_64",
    sha256 = "1828ddb3d6e0155b004eb2bc07c778b020509429cd3fae79ed20b533e06066c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-0.8.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1828ddb3d6e0155b004eb2bc07c778b020509429cd3fae79ed20b533e06066c6",
    ],
)

rpm(
    name = "swtpm-libs-0__0.8.0-1.el9.aarch64",
    sha256 = "27521263b75a6d3a69898b5300781ddd502b82b191f99a7bd450604e3def6db9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.8.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/27521263b75a6d3a69898b5300781ddd502b82b191f99a7bd450604e3def6db9",
    ],
)

rpm(
    name = "swtpm-libs-0__0.8.0-1.el9.s390x",
    sha256 = "fb6a5d884e462a7a0268c7b0067a89e9bd459f4184b43823ffb8a70e99a8a29a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/swtpm-libs-0.8.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fb6a5d884e462a7a0268c7b0067a89e9bd459f4184b43823ffb8a70e99a8a29a",
    ],
)

rpm(
    name = "swtpm-libs-0__0.8.0-1.el9.x86_64",
    sha256 = "9afc462a3f5d36db313d3ebc4b09fc34f83b4e34419a1a8f738bcce03489d683",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.8.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9afc462a3f5d36db313d3ebc4b09fc34f83b4e34419a1a8f738bcce03489d683",
    ],
)

rpm(
    name = "swtpm-tools-0__0.8.0-1.el9.aarch64",
    sha256 = "3f3624dbf8520a3bd46e2254b67b997eefbe8e2781753cd75b97876f0e396255",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.8.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3f3624dbf8520a3bd46e2254b67b997eefbe8e2781753cd75b97876f0e396255",
    ],
)

rpm(
    name = "swtpm-tools-0__0.8.0-1.el9.s390x",
    sha256 = "efb0080e071bb370955632dbdbcc0c9fee92456b48b438c5b04b0a8a2f18782b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/swtpm-tools-0.8.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/efb0080e071bb370955632dbdbcc0c9fee92456b48b438c5b04b0a8a2f18782b",
    ],
)

rpm(
    name = "swtpm-tools-0__0.8.0-1.el9.x86_64",
    sha256 = "b3531cb8d1d4408945ae6ca072852d6a9cf82300061a263b11bd841cf51ea037",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.8.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3531cb8d1d4408945ae6ca072852d6a9cf82300061a263b11bd841cf51ea037",
    ],
)

rpm(
    name = "systemd-0__252-35.el9.aarch64",
    sha256 = "2b2a80d92bd9a90600cc38651abccc051537ef059875700bc95650f7b92cba28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-252-35.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2b2a80d92bd9a90600cc38651abccc051537ef059875700bc95650f7b92cba28",
    ],
)

rpm(
    name = "systemd-0__252-35.el9.s390x",
    sha256 = "d0bac6f7d96c70b9bb3ee97b1c689107c136b1a8f02e0315c832cb2629c2d12b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-252-35.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d0bac6f7d96c70b9bb3ee97b1c689107c136b1a8f02e0315c832cb2629c2d12b",
    ],
)

rpm(
    name = "systemd-0__252-35.el9.x86_64",
    sha256 = "3117a3563c6becfa0b2fa153d21452a39321bc449a83d753d29ced40e82df04d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-35.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3117a3563c6becfa0b2fa153d21452a39321bc449a83d753d29ced40e82df04d",
    ],
)

rpm(
    name = "systemd-container-0__252-35.el9.aarch64",
    sha256 = "de667dbcf785dca8cd5b9b2b6b757086fa6465859f96464a9f26da9a2cd634af",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-252-35.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de667dbcf785dca8cd5b9b2b6b757086fa6465859f96464a9f26da9a2cd634af",
    ],
)

rpm(
    name = "systemd-container-0__252-35.el9.s390x",
    sha256 = "6f041e3aaf34b424ab352fdb91d81ea0bed5b3e71da7f417d1aa3505119cfa5e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-container-252-35.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6f041e3aaf34b424ab352fdb91d81ea0bed5b3e71da7f417d1aa3505119cfa5e",
    ],
)

rpm(
    name = "systemd-container-0__252-35.el9.x86_64",
    sha256 = "2350b9324a1858a4c8df8e81698eea863beb213053ca9278c33da9ea952ce973",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-35.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2350b9324a1858a4c8df8e81698eea863beb213053ca9278c33da9ea952ce973",
    ],
)

rpm(
    name = "systemd-libs-0__252-35.el9.aarch64",
    sha256 = "a3e7fd01e2d9e9a77ee9a79c864cd8e07aa905d35730ea692f32c3fe23db2d7c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-252-35.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a3e7fd01e2d9e9a77ee9a79c864cd8e07aa905d35730ea692f32c3fe23db2d7c",
    ],
)

rpm(
    name = "systemd-libs-0__252-35.el9.s390x",
    sha256 = "8c15df5581695b4f0d2abb1afab853457ec2e0472669ae7d7b36055fd2009d12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-libs-252-35.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8c15df5581695b4f0d2abb1afab853457ec2e0472669ae7d7b36055fd2009d12",
    ],
)

rpm(
    name = "systemd-libs-0__252-35.el9.x86_64",
    sha256 = "7391cedacd079a0e4b5855f2c0e7caabb0fd0577023997ddee3a1eabd3c2d065",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-35.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7391cedacd079a0e4b5855f2c0e7caabb0fd0577023997ddee3a1eabd3c2d065",
    ],
)

rpm(
    name = "systemd-pam-0__252-35.el9.aarch64",
    sha256 = "6dacc19945938cc3afc635db4a472b2f13b678a49762ff4a29502e250d1dbde4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-252-35.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6dacc19945938cc3afc635db4a472b2f13b678a49762ff4a29502e250d1dbde4",
    ],
)

rpm(
    name = "systemd-pam-0__252-35.el9.s390x",
    sha256 = "ce3ac46ad6df41d5165a4bdd4a283b63b2ad8785e95ff716c20173deb3e2bf61",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-pam-252-35.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ce3ac46ad6df41d5165a4bdd4a283b63b2ad8785e95ff716c20173deb3e2bf61",
    ],
)

rpm(
    name = "systemd-pam-0__252-35.el9.x86_64",
    sha256 = "127b1fa922372e179c2d0e38a98dd8ab103b7cf719a786fd697289f932702038",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-35.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/127b1fa922372e179c2d0e38a98dd8ab103b7cf719a786fd697289f932702038",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-35.el9.aarch64",
    sha256 = "b4674459d3b88f41caaf5a9b0e3fc9a6b9c4c0b483d942113f1bad0023a4966f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-252-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b4674459d3b88f41caaf5a9b0e3fc9a6b9c4c0b483d942113f1bad0023a4966f",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-35.el9.s390x",
    sha256 = "b4674459d3b88f41caaf5a9b0e3fc9a6b9c4c0b483d942113f1bad0023a4966f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-rpm-macros-252-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b4674459d3b88f41caaf5a9b0e3fc9a6b9c4c0b483d942113f1bad0023a4966f",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-35.el9.x86_64",
    sha256 = "b4674459d3b88f41caaf5a9b0e3fc9a6b9c4c0b483d942113f1bad0023a4966f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b4674459d3b88f41caaf5a9b0e3fc9a6b9c4c0b483d942113f1bad0023a4966f",
    ],
)

rpm(
    name = "tar-2__1.34-6.el9.aarch64",
    sha256 = "98a9ca5a25c6aa73b5183b3333abad062a8f82d8b9390d2b2fbdc1eea5b4fb9b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tar-1.34-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98a9ca5a25c6aa73b5183b3333abad062a8f82d8b9390d2b2fbdc1eea5b4fb9b",
    ],
)

rpm(
    name = "tar-2__1.34-6.el9.s390x",
    sha256 = "584142eceda9fb5685310dab712603636c0f66efe7c24c9b8150d8fc07c0fef2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tar-1.34-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/584142eceda9fb5685310dab712603636c0f66efe7c24c9b8150d8fc07c0fef2",
    ],
)

rpm(
    name = "tar-2__1.34-6.el9.x86_64",
    sha256 = "9f6adb2da035d5123587a2bb401487521bd6543497003ffc6e66386d898133f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tar-1.34-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f6adb2da035d5123587a2bb401487521bd6543497003ffc6e66386d898133f3",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-1.el9.aarch64",
    sha256 = "506d34ce73d61becd6190c0a86954e183c30e2d4efb873a4e258229812670ce3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/target-restore-2.1.76-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/506d34ce73d61becd6190c0a86954e183c30e2d4efb873a4e258229812670ce3",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-1.el9.s390x",
    sha256 = "506d34ce73d61becd6190c0a86954e183c30e2d4efb873a4e258229812670ce3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/target-restore-2.1.76-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/506d34ce73d61becd6190c0a86954e183c30e2d4efb873a4e258229812670ce3",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-1.el9.x86_64",
    sha256 = "506d34ce73d61becd6190c0a86954e183c30e2d4efb873a4e258229812670ce3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/target-restore-2.1.76-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/506d34ce73d61becd6190c0a86954e183c30e2d4efb873a4e258229812670ce3",
    ],
)

rpm(
    name = "targetcli-0__2.1.57-2.el9.aarch64",
    sha256 = "41b6ff3fc0e6ce313a60b07e5f80c5217cb6177c414a97439d78cf8c94bf547b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/targetcli-2.1.57-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/41b6ff3fc0e6ce313a60b07e5f80c5217cb6177c414a97439d78cf8c94bf547b",
    ],
)

rpm(
    name = "targetcli-0__2.1.57-2.el9.s390x",
    sha256 = "41b6ff3fc0e6ce313a60b07e5f80c5217cb6177c414a97439d78cf8c94bf547b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/targetcli-2.1.57-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/41b6ff3fc0e6ce313a60b07e5f80c5217cb6177c414a97439d78cf8c94bf547b",
    ],
)

rpm(
    name = "targetcli-0__2.1.57-2.el9.x86_64",
    sha256 = "41b6ff3fc0e6ce313a60b07e5f80c5217cb6177c414a97439d78cf8c94bf547b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/targetcli-2.1.57-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/41b6ff3fc0e6ce313a60b07e5f80c5217cb6177c414a97439d78cf8c94bf547b",
    ],
)

rpm(
    name = "tzdata-0__2024a-2.el9.aarch64",
    sha256 = "5f289c6c263a42f354051c3d7875d12f90eba6842a89f1cb2b0ed79c9956ab0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2024a-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5f289c6c263a42f354051c3d7875d12f90eba6842a89f1cb2b0ed79c9956ab0d",
    ],
)

rpm(
    name = "tzdata-0__2024a-2.el9.s390x",
    sha256 = "5f289c6c263a42f354051c3d7875d12f90eba6842a89f1cb2b0ed79c9956ab0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tzdata-2024a-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5f289c6c263a42f354051c3d7875d12f90eba6842a89f1cb2b0ed79c9956ab0d",
    ],
)

rpm(
    name = "tzdata-0__2024a-2.el9.x86_64",
    sha256 = "5f289c6c263a42f354051c3d7875d12f90eba6842a89f1cb2b0ed79c9956ab0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2024a-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5f289c6c263a42f354051c3d7875d12f90eba6842a89f1cb2b0ed79c9956ab0d",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-8.el9.aarch64",
    sha256 = "ea440356d7a11b3b291fd010f82d6afc6ba1eed3b181cb19363b01b290b18866",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/unbound-libs-1.16.2-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ea440356d7a11b3b291fd010f82d6afc6ba1eed3b181cb19363b01b290b18866",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-8.el9.s390x",
    sha256 = "7919a1178433bdf1c9668c73f624082d96901feda72397233c5213593e62cc8b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/unbound-libs-1.16.2-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7919a1178433bdf1c9668c73f624082d96901feda72397233c5213593e62cc8b",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-8.el9.x86_64",
    sha256 = "7e7836a8c710f7d10a594086ba7f3c6eb4a8402bb811a525c66407427262b947",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/unbound-libs-1.16.2-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e7836a8c710f7d10a594086ba7f3c6eb4a8402bb811a525c66407427262b947",
    ],
)

rpm(
    name = "usbredir-0__0.13.0-2.el9.aarch64",
    sha256 = "3cbb5cb71c942e2f0a5780cba9f8ca69741b1b877c0835ed7ddfca85f9b3ddda",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/usbredir-0.13.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3cbb5cb71c942e2f0a5780cba9f8ca69741b1b877c0835ed7ddfca85f9b3ddda",
    ],
)

rpm(
    name = "usbredir-0__0.13.0-2.el9.x86_64",
    sha256 = "7b6cec071b2d7437b70f8af875c127c00bd9b2e9d516ece64a9c30c96245394d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/usbredir-0.13.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7b6cec071b2d7437b70f8af875c127c00bd9b2e9d516ece64a9c30c96245394d",
    ],
)

rpm(
    name = "userspace-rcu-0__0.12.1-6.el9.aarch64",
    sha256 = "5ab924e8c35535d0101a5e1cb732e63940ef7b4b35a5cd0b422bf53809876b56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/userspace-rcu-0.12.1-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5ab924e8c35535d0101a5e1cb732e63940ef7b4b35a5cd0b422bf53809876b56",
    ],
)

rpm(
    name = "userspace-rcu-0__0.12.1-6.el9.x86_64",
    sha256 = "119e159428dda0e194c6428da57fae87ef75cce5c7271d347fe84283a7374c03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/userspace-rcu-0.12.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/119e159428dda0e194c6428da57fae87ef75cce5c7271d347fe84283a7374c03",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-18.el9.aarch64",
    sha256 = "b3d4b1559f81d72e2112e9f5acf1c9e0bdadaa366cbcb8df7d2e603db0e689e4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b3d4b1559f81d72e2112e9f5acf1c9e0bdadaa366cbcb8df7d2e603db0e689e4",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-18.el9.s390x",
    sha256 = "ab31c060229398e6e3e33034505b5e4254bab1e56d0e426efa521292628ac695",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ab31c060229398e6e3e33034505b5e4254bab1e56d0e426efa521292628ac695",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-18.el9.x86_64",
    sha256 = "9f9b09b5d78cbeb3df6b9d6d5901b6a0b2b9668bf541d93e1a18e68cb33a8fb5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f9b09b5d78cbeb3df6b9d6d5901b6a0b2b9668bf541d93e1a18e68cb33a8fb5",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-18.el9.aarch64",
    sha256 = "d0145bfa0348ccc4e23e2414533b9517ae02911645e312468d27cddb32575790",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.37.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d0145bfa0348ccc4e23e2414533b9517ae02911645e312468d27cddb32575790",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-18.el9.s390x",
    sha256 = "a181dbfe6385d7564f766f07785490542edce4bf086b6669f09cffb9d9871c35",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-core-2.37.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a181dbfe6385d7564f766f07785490542edce4bf086b6669f09cffb9d9871c35",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-18.el9.x86_64",
    sha256 = "85a2802812f1ae05fa54e713183c9577afe767f47928cf66be746b12fa81edea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.37.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/85a2802812f1ae05fa54e713183c9577afe767f47928cf66be746b12fa81edea",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-20.el9.aarch64",
    sha256 = "b142f0b4f853c0560a17f118cbffadd89d16296cac85287cd14d35bf8b0847f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.2.2637-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b142f0b4f853c0560a17f118cbffadd89d16296cac85287cd14d35bf8b0847f2",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-20.el9.s390x",
    sha256 = "cee1fbebaf32f85e824872aa675626070aba60e9ba06d7dd97e0a70b59050c5d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/vim-minimal-8.2.2637-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cee1fbebaf32f85e824872aa675626070aba60e9ba06d7dd97e0a70b59050c5d",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-20.el9.x86_64",
    sha256 = "5bef7d6b66ece8820a758a6b1fb99a4512dd3bdcac0774723b630bfd5144ee62",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5bef7d6b66ece8820a758a6b1fb99a4512dd3bdcac0774723b630bfd5144ee62",
    ],
)

rpm(
    name = "virtiofsd-0__1.10.1-1.el9.aarch64",
    sha256 = "342b9704a63f440c693f408c7898cf25fe1849fe9cb8e656c0f403c492192480",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/virtiofsd-1.10.1-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/342b9704a63f440c693f408c7898cf25fe1849fe9cb8e656c0f403c492192480",
    ],
)

rpm(
    name = "virtiofsd-0__1.10.1-1.el9.s390x",
    sha256 = "dbcdb264998b91b91727b3b1d7fe1151071155e4a26ee4501aef258ff8c32628",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/virtiofsd-1.10.1-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/dbcdb264998b91b91727b3b1d7fe1151071155e4a26ee4501aef258ff8c32628",
    ],
)

rpm(
    name = "virtiofsd-0__1.10.1-1.el9.x86_64",
    sha256 = "8a0534ec0ce796488f55c18388ca0ca3d0ace4a4e696159989c836f9db51cfe9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/virtiofsd-1.10.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a0534ec0ce796488f55c18388ca0ca3d0ace4a4e696159989c836f9db51cfe9",
    ],
)

rpm(
    name = "which-0__2.21-29.el9.aarch64",
    sha256 = "2edd6b710ebd483724d0c0c1dff3d5922ce3082ea1bd10865d9f4f0bbf4bb050",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/which-2.21-29.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2edd6b710ebd483724d0c0c1dff3d5922ce3082ea1bd10865d9f4f0bbf4bb050",
    ],
)

rpm(
    name = "which-0__2.21-29.el9.s390x",
    sha256 = "e76d002db39aa53a485a47b97d92378b9a1221de6dedd89a50b070d98e3a4d48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/which-2.21-29.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e76d002db39aa53a485a47b97d92378b9a1221de6dedd89a50b070d98e3a4d48",
    ],
)

rpm(
    name = "which-0__2.21-29.el9.x86_64",
    sha256 = "c69af7b876363091bbeb99b4adfbab743f91da3c45478bb7a055c441e395174d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/which-2.21-29.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c69af7b876363091bbeb99b4adfbab743f91da3c45478bb7a055c441e395174d",
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
    name = "xorriso-0__1.5.4-4.el9.s390x",
    sha256 = "0e2c98057f72060a8de367f6c26e42b0818356a20d0c54b20e1c31ab47f271ea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/xorriso-1.5.4-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0e2c98057f72060a8de367f6c26e42b0818356a20d0c54b20e1c31ab47f271ea",
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
    name = "xz-0__5.2.5-8.el9.s390x",
    sha256 = "e3bbe47e750775943bace76db54b52b08ed2a572ec3fe2aac200661fc54dd001",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/xz-5.2.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e3bbe47e750775943bace76db54b52b08ed2a572ec3fe2aac200661fc54dd001",
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
    name = "xz-libs-0__5.2.5-8.el9.s390x",
    sha256 = "f5df58b242361ae5aaf97d1149c4331cc762394cadb5ebd054db089a6e10ae24",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/xz-libs-5.2.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f5df58b242361ae5aaf97d1149c4331cc762394cadb5ebd054db089a6e10ae24",
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
    name = "yajl-0__2.1.0-22.el9.aarch64",
    sha256 = "5f099ce8836377f6aba662e5835cc500b2e8f29cd8c9b56b22df7c564f7d209c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/yajl-2.1.0-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5f099ce8836377f6aba662e5835cc500b2e8f29cd8c9b56b22df7c564f7d209c",
    ],
)

rpm(
    name = "yajl-0__2.1.0-22.el9.s390x",
    sha256 = "45c55fec973903149868133e4416265694f4589643337639211cac0db239db42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/yajl-2.1.0-22.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/45c55fec973903149868133e4416265694f4589643337639211cac0db239db42",
    ],
)

rpm(
    name = "yajl-0__2.1.0-22.el9.x86_64",
    sha256 = "907156eb13e2120402287396f92b7589515ab0cba802b99c3835dd36f6a12cdf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/yajl-2.1.0-22.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/907156eb13e2120402287396f92b7589515ab0cba802b99c3835dd36f6a12cdf",
    ],
)

rpm(
    name = "zlib-0__1.2.11-41.el9.aarch64",
    sha256 = "c50e107cdd35460294852d99c954296e0e833d37852a1be1e2aaea2f1b48f9d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/zlib-1.2.11-41.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c50e107cdd35460294852d99c954296e0e833d37852a1be1e2aaea2f1b48f9d2",
    ],
)

rpm(
    name = "zlib-0__1.2.11-41.el9.s390x",
    sha256 = "bbe95dadf7383694d5b13ea8ae89b76697ed7009b4be889220d4a7d23db28759",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/zlib-1.2.11-41.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bbe95dadf7383694d5b13ea8ae89b76697ed7009b4be889220d4a7d23db28759",
    ],
)

rpm(
    name = "zlib-0__1.2.11-41.el9.x86_64",
    sha256 = "370951ea635bc16313f21ac2823ec815147ed1124b74865a34c54e94e4db9602",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/zlib-1.2.11-41.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/370951ea635bc16313f21ac2823ec815147ed1124b74865a34c54e94e4db9602",
    ],
)
