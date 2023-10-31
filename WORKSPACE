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
        "https://storage.googleapis.com/builddeps/099a9fb96a376ccbbb7d291ed4ecbdfd42f6bc822ab77ae6f1b5cb9e914e94fa",
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
    sha256 = "9dfd5ab882cae4ff9e2a7c1352c05949fa0c175af6b4103b19db48657e6da8b8",
    urls = [
        "https://github.com/rmohr/bazeldnf/releases/download/v0.5.6/bazeldnf-v0.5.6.tar.gz",
        "https://storage.googleapis.com/builddeps/9dfd5ab882cae4ff9e2a7c1352c05949fa0c175af6b4103b19db48657e6da8b8",
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
    go_version = "1.19.9",
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
    digest = "sha256:6c871aa3c9019984dfd7f520635bd658d740ad20c6268a82faa433f69dfc9a0b",
    registry = "gcr.io",
    repository = "distroless/base",
)

container_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:4f81adb2fa054fd2ea49a918e2eb025325992b1235733da5ba51ab75bf9bd386",
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
    digest = "sha256:85f7bb99624422dc45b2c203d520a06bfce7a760ef831aafbf0e2bf2b92ebcf4",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
)

container_pull(
    name = "alpine_with_test_tooling",
    digest = "sha256:abd71660edffc355520239e8910debfa7491516ee35240f23bba378d9095410c",
    registry = "quay.io",
    repository = "kubevirtci/alpine-with-test-tooling-container-disk",
    tag = "2211021552-8cca8c0",
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
    name = "alternatives-0__1.24-1.el9.x86_64",
    sha256 = "b58e7ea30c27ecb321d9a279b95b62aef59d92173714fce859bfb359ee231ff3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/alternatives-1.24-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b58e7ea30c27ecb321d9a279b95b62aef59d92173714fce859bfb359ee231ff3",
    ],
)

rpm(
    name = "audit-libs-0__3.0.7-104.el9.aarch64",
    sha256 = "959ba1d41b5d28d7121c2f8b64f931d886f2d41f77fbd1034d55fbb33328d9f8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/audit-libs-3.0.7-104.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/959ba1d41b5d28d7121c2f8b64f931d886f2d41f77fbd1034d55fbb33328d9f8",
    ],
)

rpm(
    name = "audit-libs-0__3.0.7-104.el9.x86_64",
    sha256 = "22d335e369ffb90f4b9ab496aa5f47c1a0913679db7ff4e706397be3bba4e3ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.0.7-104.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/22d335e369ffb90f4b9ab496aa5f47c1a0913679db7ff4e706397be3bba4e3ca",
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
    name = "basesystem-0__11-13.el9.x86_64",
    sha256 = "a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/basesystem-11-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a7a687ef39dd28d01d34fab18ea7e3e87f649f6c202dded82260b7ea625b9973",
    ],
)

rpm(
    name = "bash-0__5.1.8-6.el9.aarch64",
    sha256 = "adbea9afe78b2f67de854fdf5440326dda5383763797eb9ac486969edeecaef0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bash-5.1.8-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/adbea9afe78b2f67de854fdf5440326dda5383763797eb9ac486969edeecaef0",
    ],
)

rpm(
    name = "bash-0__5.1.8-6.el9.x86_64",
    sha256 = "09f700a94e187a74f6f4a5f750082732e193d41392a85f042bdeb0bcbabe0a1f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bash-5.1.8-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09f700a94e187a74f6f4a5f750082732e193d41392a85f042bdeb0bcbabe0a1f",
    ],
)

rpm(
    name = "binutils-0__2.35.2-42.el9.aarch64",
    sha256 = "11682cf535642ab99699f7c42c030b368a6b5919a0a1b1a2b08a86412201f044",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-42.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/11682cf535642ab99699f7c42c030b368a6b5919a0a1b1a2b08a86412201f044",
    ],
)

rpm(
    name = "binutils-0__2.35.2-42.el9.x86_64",
    sha256 = "24a456337ef4e6d346a095483cd545accff9ff35a4fc59aa89abfcf234568a2d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-42.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/24a456337ef4e6d346a095483cd545accff9ff35a4fc59aa89abfcf234568a2d",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-42.el9.aarch64",
    sha256 = "32bf8d1f2b72efda4903a6c022283caaec6502041cbc270a08f21ea22cdacf76",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-42.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/32bf8d1f2b72efda4903a6c022283caaec6502041cbc270a08f21ea22cdacf76",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-42.el9.x86_64",
    sha256 = "a68329816fa0be3cf4ec251866ef0d9666c9de63603fa152cbb6493a0fd3000d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-42.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a68329816fa0be3cf4ec251866ef0d9666c9de63603fa152cbb6493a0fd3000d",
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
    name = "ca-certificates-0__2023.2.60_v7.0.306-90.1.el9.aarch64",
    sha256 = "76d996300aeaf56a06191f8ea2df8387813f4fa8100c6f4c1000073633e1147f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ca-certificates-2023.2.60_v7.0.306-90.1.el9.noarch.rpm",
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
    name = "capstone-0__4.0.2-10.el9.x86_64",
    sha256 = "f6a9fdc6bcb5da1b2ce44ca7ed6289759c37add7adbb19916dd36d5bb4624a41",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/capstone-4.0.2-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f6a9fdc6bcb5da1b2ce44ca7ed6289759c37add7adbb19916dd36d5bb4624a41",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-23.el9.aarch64",
    sha256 = "23aaff377dffc4a7e82eec56feabe1d80616d818e2160f30b744cd3cde1af17e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-9.0-23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/23aaff377dffc4a7e82eec56feabe1d80616d818e2160f30b744cd3cde1af17e",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-23.el9.x86_64",
    sha256 = "23aaff377dffc4a7e82eec56feabe1d80616d818e2160f30b744cd3cde1af17e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/23aaff377dffc4a7e82eec56feabe1d80616d818e2160f30b744cd3cde1af17e",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-23.el9.aarch64",
    sha256 = "3077d913caf3ede40e2dec2873492347d363917659d9fc6182f7cf9ae656eb25",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-release-9.0-23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3077d913caf3ede40e2dec2873492347d363917659d9fc6182f7cf9ae656eb25",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-23.el9.x86_64",
    sha256 = "3077d913caf3ede40e2dec2873492347d363917659d9fc6182f7cf9ae656eb25",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-release-9.0-23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3077d913caf3ede40e2dec2873492347d363917659d9fc6182f7cf9ae656eb25",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-23.el9.aarch64",
    sha256 = "b269292cbdd24f177b4f1c61e75a69a8bb0266aa3ea9c00df6246b8f4e2f6970",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-9.0-23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b269292cbdd24f177b4f1c61e75a69a8bb0266aa3ea9c00df6246b8f4e2f6970",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-23.el9.x86_64",
    sha256 = "b269292cbdd24f177b4f1c61e75a69a8bb0266aa3ea9c00df6246b8f4e2f6970",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b269292cbdd24f177b4f1c61e75a69a8bb0266aa3ea9c00df6246b8f4e2f6970",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-34.el9.aarch64",
    sha256 = "9ab931a79d42f2cf38ef98283603792abbef8c99d7cc112e04c69d0a66fb074c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.32-34.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9ab931a79d42f2cf38ef98283603792abbef8c99d7cc112e04c69d0a66fb074c",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-34.el9.x86_64",
    sha256 = "fd6001340bdba2e7b49b6dee004dc7e54e5b2393bdb0c9de9ca2e8801e39e671",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.32-34.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd6001340bdba2e7b49b6dee004dc7e54e5b2393bdb0c9de9ca2e8801e39e671",
    ],
)

rpm(
    name = "cpp-0__11.4.1-2.3.el9.aarch64",
    sha256 = "fd944d59239d5d86ee51efc390f3d0fd90f59a7be8aaae7fe2cfe2784d39b0e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fd944d59239d5d86ee51efc390f3d0fd90f59a7be8aaae7fe2cfe2784d39b0e3",
    ],
)

rpm(
    name = "cpp-0__11.4.1-2.3.el9.x86_64",
    sha256 = "f19408aab5b0c6cbb3d13d4d0df4d8705bd54b2974f87ae314c42d790ef263de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.4.1-2.3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f19408aab5b0c6cbb3d13d4d0df4d8705bd54b2974f87ae314c42d790ef263de",
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
    name = "crypto-policies-0__20230920-1.git8dcf74d.el9.aarch64",
    sha256 = "f912ea5bfecfa396ed812aafc7ed47e4e55e438d37cd86a370970ebe85f7d8bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20230920-1.git8dcf74d.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f912ea5bfecfa396ed812aafc7ed47e4e55e438d37cd86a370970ebe85f7d8bb",
    ],
)

rpm(
    name = "crypto-policies-0__20230920-1.git8dcf74d.el9.x86_64",
    sha256 = "f912ea5bfecfa396ed812aafc7ed47e4e55e438d37cd86a370970ebe85f7d8bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20230920-1.git8dcf74d.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f912ea5bfecfa396ed812aafc7ed47e4e55e438d37cd86a370970ebe85f7d8bb",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-28.el9.aarch64",
    sha256 = "3db2ce489d5eec9587f24e18054d6e56aab123a02b1464bac1fc3df7f4bf45b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/curl-minimal-7.76.1-28.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3db2ce489d5eec9587f24e18054d6e56aab123a02b1464bac1fc3df7f4bf45b8",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-28.el9.x86_64",
    sha256 = "c35bb84e51f54d506c255a04e2bdb062ebdd9181771e48c4f90aa7e5e078c1c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/curl-minimal-7.76.1-28.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c35bb84e51f54d506c255a04e2bdb062ebdd9181771e48c4f90aa7e5e078c1c6",
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
    name = "dbus-libs-1__1.12.20-8.el9.x86_64",
    sha256 = "2d46aaa0b1e8032d10156b040a5226b5a90ef000d8d85d40fd5671379a5bc904",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.20-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d46aaa0b1e8032d10156b040a5226b5a90ef000d8d85d40fd5671379a5bc904",
    ],
)

rpm(
    name = "device-mapper-9__1.02.195-3.el9.aarch64",
    sha256 = "e696b10edfe98a7c4e6f87358353035618e20a18e8e733c00bfe22078263fcd3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.195-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e696b10edfe98a7c4e6f87358353035618e20a18e8e733c00bfe22078263fcd3",
    ],
)

rpm(
    name = "device-mapper-9__1.02.195-3.el9.x86_64",
    sha256 = "441482e8244d2753fff19c2cfc0e17b2655fb7542bde27db540c9cfac2b76e3d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.195-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/441482e8244d2753fff19c2cfc0e17b2655fb7542bde27db540c9cfac2b76e3d",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.195-3.el9.aarch64",
    sha256 = "1409398162b3129e33044270537388112011d2b7a90dc2746354dfc177480599",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.195-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1409398162b3129e33044270537388112011d2b7a90dc2746354dfc177480599",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.195-3.el9.x86_64",
    sha256 = "a5b4509bbd31c4cad65c8460fa2acf4c1505e6b27f38dc7b56fdab125a5fa0b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.195-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a5b4509bbd31c4cad65c8460fa2acf4c1505e6b27f38dc7b56fdab125a5fa0b8",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-22.el9.aarch64",
    sha256 = "63520410c680dad15f88197bec9067f087d02692a661f041b2392427b06c9c94",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.7-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/63520410c680dad15f88197bec9067f087d02692a661f041b2392427b06c9c94",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-22.el9.x86_64",
    sha256 = "ca6d7bebafdfb3f8a3e6f541f06004d3a21c41c1b3e9e95e42dc97ece1d88dce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.7-22.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ca6d7bebafdfb3f8a3e6f541f06004d3a21c41c1b3e9e95e42dc97ece1d88dce",
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
    name = "edk2-aarch64-0__20230524-3.el9.aarch64",
    sha256 = "893e8f9a58deb7270c8e4a3a8e9e002392ff1c1b124e56b1bc59e6e0bd377da4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20230524-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/893e8f9a58deb7270c8e4a3a8e9e002392ff1c1b124e56b1bc59e6e0bd377da4",
    ],
)

rpm(
    name = "edk2-ovmf-0__20230524-3.el9.x86_64",
    sha256 = "865e123047cff07ce8d9b51ba36b3f6a3c914267e4422249e4a0d9c897f86946",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20230524-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/865e123047cff07ce8d9b51ba36b3f6a3c914267e4422249e4a0d9c897f86946",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.189-3.el9.aarch64",
    sha256 = "f217f18028c124b914b45a174f8ead2682ecfc43bd7d7e9236d97c012ca4684b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.189-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f217f18028c124b914b45a174f8ead2682ecfc43bd7d7e9236d97c012ca4684b",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.189-3.el9.x86_64",
    sha256 = "83531115c33351ead363a4865c6f6395fcfae89f45488da2e1b68ca82f9fc5a6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.189-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/83531115c33351ead363a4865c6f6395fcfae89f45488da2e1b68ca82f9fc5a6",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.189-3.el9.aarch64",
    sha256 = "d971cb78f763cce00cb91a47eac4ccfc0f33f82304c3a71743cc19e4583cec54",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.189-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d971cb78f763cce00cb91a47eac4ccfc0f33f82304c3a71743cc19e4583cec54",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.189-3.el9.x86_64",
    sha256 = "d971cb78f763cce00cb91a47eac4ccfc0f33f82304c3a71743cc19e4583cec54",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.189-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d971cb78f763cce00cb91a47eac4ccfc0f33f82304c3a71743cc19e4583cec54",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.189-3.el9.aarch64",
    sha256 = "4dc92746c3ae09bfbb99f044bdc2ddf3d8d8f175a28f8cb406523bc2a495ffa2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.189-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4dc92746c3ae09bfbb99f044bdc2ddf3d8d8f175a28f8cb406523bc2a495ffa2",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.189-3.el9.x86_64",
    sha256 = "4868aa3b7329c8b4664afee46eba0199ccd17ec867f53c4ecb42945f1f854b5b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.189-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4868aa3b7329c8b4664afee46eba0199ccd17ec867f53c4ecb42945f1f854b5b",
    ],
)

rpm(
    name = "elfutils-libs-0__0.189-3.el9.aarch64",
    sha256 = "b453fb87b50bd63b4b89135cb38e315c14ce0da79bebabda9d118ac7fdd0f60a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.189-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b453fb87b50bd63b4b89135cb38e315c14ce0da79bebabda9d118ac7fdd0f60a",
    ],
)

rpm(
    name = "elfutils-libs-0__0.189-3.el9.x86_64",
    sha256 = "4065bc4c40514aa234fdf56692c98ee5798f021d7a14e15d0ad865996e84b2d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.189-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4065bc4c40514aa234fdf56692c98ee5798f021d7a14e15d0ad865996e84b2d4",
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
    name = "ethtool-2__6.2-1.el9.x86_64",
    sha256 = "bc4b58cbda4ce3eb8795aa35db56c6e7d52a53f48b3c197d0ee83911f5d3eadc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ethtool-6.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc4b58cbda4ce3eb8795aa35db56c6e7d52a53f48b3c197d0ee83911f5d3eadc",
    ],
)

rpm(
    name = "expat-0__2.5.0-1.el9.aarch64",
    sha256 = "2163792c7a297e441d7c3c0cbef7a6da0695e44e0b16fbb796cd90ab91dfe0cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/expat-2.5.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2163792c7a297e441d7c3c0cbef7a6da0695e44e0b16fbb796cd90ab91dfe0cb",
    ],
)

rpm(
    name = "expat-0__2.5.0-1.el9.x86_64",
    sha256 = "b5092845377c3505cd072a896c443abe5da21d3c6c6cb23d917db159905178a6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.5.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b5092845377c3505cd072a896c443abe5da21d3c6c6cb23d917db159905178a6",
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
    name = "findutils-1__4.8.0-6.el9.aarch64",
    sha256 = "c552d76e062fdb098b57eef1f8f6566a72354d441a101e6238e4b8ba43dfb77e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/findutils-4.8.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c552d76e062fdb098b57eef1f8f6566a72354d441a101e6238e4b8ba43dfb77e",
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
    name = "fuse-common-0__3.10.2-6.el9.x86_64",
    sha256 = "17f0e60e894c860d5019e6a40e072b23921c4f5726717ec55e3563a6b5a5d3b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-common-3.10.2-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/17f0e60e894c860d5019e6a40e072b23921c4f5726717ec55e3563a6b5a5d3b3",
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
    name = "gcc-0__11.4.1-2.3.el9.aarch64",
    sha256 = "d018eafb2e2cd578cf97dda5e98612e6b596853b42f1e1ca4e49fc143f780554",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d018eafb2e2cd578cf97dda5e98612e6b596853b42f1e1ca4e49fc143f780554",
    ],
)

rpm(
    name = "gcc-0__11.4.1-2.3.el9.x86_64",
    sha256 = "c61d8f6c05e46046da99822c7469bac7a7c79d3469333664b7c9ae41b048a12a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.4.1-2.3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c61d8f6c05e46046da99822c7469bac7a7c79d3469333664b7c9ae41b048a12a",
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
    name = "gettext-0__0.21-8.el9.aarch64",
    sha256 = "66387c45fa58eea0120e0cdfa27ffb2ca4eda1cb9f157be7af23503f4b42fdab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gettext-0.21-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/66387c45fa58eea0120e0cdfa27ffb2ca4eda1cb9f157be7af23503f4b42fdab",
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
    name = "glib2-0__2.68.4-11.el9.aarch64",
    sha256 = "37eaf1d4446bdb5f65c64f4bffb7e131cb55628151b47d859b9e5de909a32d9f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glib2-2.68.4-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/37eaf1d4446bdb5f65c64f4bffb7e131cb55628151b47d859b9e5de909a32d9f",
    ],
)

rpm(
    name = "glib2-0__2.68.4-11.el9.x86_64",
    sha256 = "4f75096fde19d60137eccade0dc9451cb8baa9306c84d2aae1bd7faa8b8cff05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4f75096fde19d60137eccade0dc9451cb8baa9306c84d2aae1bd7faa8b8cff05",
    ],
)

rpm(
    name = "glibc-0__2.34-83.el9.7.aarch64",
    sha256 = "41243fe82d29e35302d434a96b1545021a60acd6cdcadbfb15cef8b9e4a2062e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-83.el9.7.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/41243fe82d29e35302d434a96b1545021a60acd6cdcadbfb15cef8b9e4a2062e",
    ],
)

rpm(
    name = "glibc-0__2.34-83.el9.7.x86_64",
    sha256 = "b8782a8b5c896b73a91db59efe2a6d32b73f361a432efb6385fc65c3f11f01ec",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-83.el9.7.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b8782a8b5c896b73a91db59efe2a6d32b73f361a432efb6385fc65c3f11f01ec",
    ],
)

rpm(
    name = "glibc-common-0__2.34-83.el9.7.aarch64",
    sha256 = "1dca757b405704cb4ed2a65d042deee9a99698c6e16d32c6355c9654fba791bf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-83.el9.7.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1dca757b405704cb4ed2a65d042deee9a99698c6e16d32c6355c9654fba791bf",
    ],
)

rpm(
    name = "glibc-common-0__2.34-83.el9.7.x86_64",
    sha256 = "1cb8d470cc3a31b326f77915ac57496012c2effad05ff069a7282fe1e0a2a32a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-83.el9.7.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cb8d470cc3a31b326f77915ac57496012c2effad05ff069a7282fe1e0a2a32a",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-83.el9.7.aarch64",
    sha256 = "1f03c53397967290e5d06f9dfd38baa441680f3a382f457957c915f393a109f7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-83.el9.7.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1f03c53397967290e5d06f9dfd38baa441680f3a382f457957c915f393a109f7",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-83.el9.7.x86_64",
    sha256 = "b61a61370e932cee7b1ca19a4edd09b9580a045d910c8d5101956513067fa4d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-83.el9.7.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b61a61370e932cee7b1ca19a4edd09b9580a045d910c8d5101956513067fa4d9",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-83.el9.7.x86_64",
    sha256 = "ea3fca0116694456200556fbbbd41775ce58aa89d92281be9c30d8e9a1310544",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-83.el9.7.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ea3fca0116694456200556fbbbd41775ce58aa89d92281be9c30d8e9a1310544",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-83.el9.7.aarch64",
    sha256 = "62bbf3aabb2db57c8ef6e6ea1d85650e5eec522561fa45dca9d4eb20fda6baea",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-83.el9.7.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/62bbf3aabb2db57c8ef6e6ea1d85650e5eec522561fa45dca9d4eb20fda6baea",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-83.el9.7.x86_64",
    sha256 = "20efb20986a8150102abcc8a089e1e0a952a65e17e606b85c1ff921e97fd1b48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-83.el9.7.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20efb20986a8150102abcc8a089e1e0a952a65e17e606b85c1ff921e97fd1b48",
    ],
)

rpm(
    name = "glibc-static-0__2.34-83.el9.7.aarch64",
    sha256 = "92ddfa8ba04645dd45e4d56d445e049fcf61df41f3ec97bf5e8b4b3eb5b5c7a9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-83.el9.7.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/92ddfa8ba04645dd45e4d56d445e049fcf61df41f3ec97bf5e8b4b3eb5b5c7a9",
    ],
)

rpm(
    name = "glibc-static-0__2.34-83.el9.7.x86_64",
    sha256 = "5232ba36529f3522024355b9971b6aef2bf6565e63d3cb9bd4e7aff03b8bb5ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-83.el9.7.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5232ba36529f3522024355b9971b6aef2bf6565e63d3cb9bd4e7aff03b8bb5ae",
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
    name = "gnutls-0__3.7.6-23.el9.aarch64",
    sha256 = "0be14e945956d83deb956d47cc7684509b533b4c6f57961f404d005ebb42a113",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gnutls-3.7.6-23.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0be14e945956d83deb956d47cc7684509b533b4c6f57961f404d005ebb42a113",
    ],
)

rpm(
    name = "gnutls-0__3.7.6-23.el9.x86_64",
    sha256 = "6fd8a96f4632f2d08efc211d04ebf7591fecd213af1374f4ffe57b614b76552b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnutls-3.7.6-23.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6fd8a96f4632f2d08efc211d04ebf7591fecd213af1374f4ffe57b614b76552b",
    ],
)

rpm(
    name = "gnutls-dane-0__3.7.6-23.el9.aarch64",
    sha256 = "bafcbc6cb7f35ddac759e1420879c544eb7a77d07db2df31d55fd699428bd6cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.7.6-23.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bafcbc6cb7f35ddac759e1420879c544eb7a77d07db2df31d55fd699428bd6cb",
    ],
)

rpm(
    name = "gnutls-dane-0__3.7.6-23.el9.x86_64",
    sha256 = "901866032a72c13f61069ca9c6d44b2023aa39f5d3d60d8f0d85042d926182e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.7.6-23.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/901866032a72c13f61069ca9c6d44b2023aa39f5d3d60d8f0d85042d926182e0",
    ],
)

rpm(
    name = "gnutls-utils-0__3.7.6-23.el9.aarch64",
    sha256 = "2552c95cec352cd9a586953a43e8c3432daf719973cd59fbc5e9b3a328d0b568",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.7.6-23.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2552c95cec352cd9a586953a43e8c3432daf719973cd59fbc5e9b3a328d0b568",
    ],
)

rpm(
    name = "gnutls-utils-0__3.7.6-23.el9.x86_64",
    sha256 = "3ddc45e32cc788ff874bdfc21c1260032a61d903858b0b47b7975cf8300a27db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.7.6-23.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3ddc45e32cc788ff874bdfc21c1260032a61d903858b0b47b7975cf8300a27db",
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
    name = "guestfs-tools-0__1.50.1-3.el9.x86_64",
    sha256 = "9cff429bf3d24ace4083e3bae739557f04db1629fc0bc296936dc94b1f793dc6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.50.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9cff429bf3d24ace4083e3bae739557f04db1629fc0bc296936dc94b1f793dc6",
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
    name = "hwdata-0__0.348-9.11.el9.x86_64",
    sha256 = "3794c5c7bef966008bd42baff1262a7715b5b419b0fb92cd9e704acbb8f2e919",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3794c5c7bef966008bd42baff1262a7715b5b419b0fb92cd9e704acbb8f2e919",
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
    name = "iproute-tc-0__6.2.0-5.el9.x86_64",
    sha256 = "f142c4f355288c3583b2d0fabe71ab447e8c4f03a4b3754c411b21e4e26c8f2b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.2.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f142c4f355288c3583b2d0fabe71ab447e8c4f03a4b3754c411b21e4e26c8f2b",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.8-6.el9.aarch64",
    sha256 = "a0572f3b2eddcc18370801fd86bf6e5ed729702b63fadfc032c9855661090639",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.8-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a0572f3b2eddcc18370801fd86bf6e5ed729702b63fadfc032c9855661090639",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.8-6.el9.x86_64",
    sha256 = "c1e4ebce15d824604e777993f46b94706239044c81bc5240e9541b1ae93485a5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.8-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c1e4ebce15d824604e777993f46b94706239044c81bc5240e9541b1ae93485a5",
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
    name = "json-glib-0__1.6.6-1.el9.x86_64",
    sha256 = "d850cb45d31fe84cb50cb1fa26eb5418633aae1f0dcab8b7ebadd3bd3e340956",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/json-glib-1.6.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d850cb45d31fe84cb50cb1fa26eb5418633aae1f0dcab8b7ebadd3bd3e340956",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-375.el9.aarch64",
    sha256 = "ec54c5d1325e8deeb362274efa5695f0f69444e709d8df8f12341379e77b40b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/kernel-headers-5.14.0-375.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ec54c5d1325e8deeb362274efa5695f0f69444e709d8df8f12341379e77b40b4",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-375.el9.x86_64",
    sha256 = "000f893ba989fa999d6cd75d8063425e73d27196bd9331ee16a79386b3f9a242",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/kernel-headers-5.14.0-375.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/000f893ba989fa999d6cd75d8063425e73d27196bd9331ee16a79386b3f9a242",
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
    name = "kmod-libs-0__28-9.el9.x86_64",
    sha256 = "319957f8f3abe9b05b4aca442a3c633b36c8974e2dbd87f31ec66885f66e1b88",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-libs-28-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/319957f8f3abe9b05b4aca442a3c633b36c8974e2dbd87f31ec66885f66e1b88",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-1.el9.aarch64",
    sha256 = "348c8b97edf3ec258e3b5281af48ac22369bba8b747e0a52de1258578e91c36e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.1-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/348c8b97edf3ec258e3b5281af48ac22369bba8b747e0a52de1258578e91c36e",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-1.el9.x86_64",
    sha256 = "3ef93138174dc618bbf4680b5df11d27cd6afb361cd02efad8bcbb5bf0769c2e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3ef93138174dc618bbf4680b5df11d27cd6afb361cd02efad8bcbb5bf0769c2e",
    ],
)

rpm(
    name = "less-0__590-2.el9.x86_64",
    sha256 = "de1c6723f43ef77ae0992726210de296731b8f440d74d28ea276e9fd3a1289d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/less-590-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de1c6723f43ef77ae0992726210de296731b8f440d74d28ea276e9fd3a1289d6",
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
    name = "libarchive-0__3.5.3-4.el9.aarch64",
    sha256 = "c043954972a8dea0b6cf5d3092c1eee90bb48b3fcb7cedf30aa861dc1d3f402c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libarchive-3.5.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c043954972a8dea0b6cf5d3092c1eee90bb48b3fcb7cedf30aa861dc1d3f402c",
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
    name = "libasan-0__11.4.1-2.3.el9.aarch64",
    sha256 = "de0b570c99b5371176a8157d1a7a84a82b5f7f5e0bcd93190d3ebc92994e7cd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de0b570c99b5371176a8157d1a7a84a82b5f7f5e0bcd93190d3ebc92994e7cd1",
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
    name = "libatomic-0__11.4.1-2.3.el9.aarch64",
    sha256 = "98d89991440bc23aba0696d304109fd474326eb5c17e95ddc124d645f6def2ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98d89991440bc23aba0696d304109fd474326eb5c17e95ddc124d645f6def2ad",
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
    name = "libblkid-0__2.37.4-15.el9.aarch64",
    sha256 = "e7f4f1a30f71feea534a79b3be03d26ae3001c6ececb7b2a5a54371e66deef78",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libblkid-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e7f4f1a30f71feea534a79b3be03d26ae3001c6ececb7b2a5a54371e66deef78",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-15.el9.x86_64",
    sha256 = "519a372517cb0d466878808afceda7afd95be9278aa67b2dd311a8a886783b77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libblkid-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/519a372517cb0d466878808afceda7afd95be9278aa67b2dd311a8a886783b77",
    ],
)

rpm(
    name = "libbpf-2__1.2.0-1.el9.aarch64",
    sha256 = "5016490cb170cd073f702a827435a84c1c56faeeabc3c1b273a94e7040f8191d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libbpf-1.2.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5016490cb170cd073f702a827435a84c1c56faeeabc3c1b273a94e7040f8191d",
    ],
)

rpm(
    name = "libbpf-2__1.2.0-1.el9.x86_64",
    sha256 = "fcd9d737d25864206da7fd048ebb7e7e011914e7bfda3ae5e8bfa1097d387852",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-1.2.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fcd9d737d25864206da7fd048ebb7e7e011914e7bfda3ae5e8bfa1097d387852",
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
    name = "libcurl-minimal-0__7.76.1-28.el9.aarch64",
    sha256 = "1b32cdc10a287a9cc77655f5a4ba25a3ddc5ca9635640cc1a5644bd91c1152a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.76.1-28.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1b32cdc10a287a9cc77655f5a4ba25a3ddc5ca9635640cc1a5644bd91c1152a1",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-28.el9.x86_64",
    sha256 = "c3b6c5681713e3475f2b652e631cc2f6d9893004219121bb6a571bd04fc48906",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-28.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c3b6c5681713e3475f2b652e631cc2f6d9893004219121bb6a571bd04fc48906",
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
    name = "libeconf-0__0.4.1-3.el9.aarch64",
    sha256 = "f2a26663f33189999b437c769bcd3069a3e919b4590c62edaac706fdb32654f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libeconf-0.4.1-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f2a26663f33189999b437c769bcd3069a3e919b4590c62edaac706fdb32654f5",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-3.el9.x86_64",
    sha256 = "841f2f5822dafc227f1eb70f4549fb382b326440fd22dc655dcbb37c843b1320",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libeconf-0.4.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/841f2f5822dafc227f1eb70f4549fb382b326440fd22dc655dcbb37c843b1320",
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
    name = "libfdisk-0__2.37.4-15.el9.aarch64",
    sha256 = "471af108df2956f60540f1b87f2f274906f564e9871b625940549a1b25d3adef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libfdisk-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/471af108df2956f60540f1b87f2f274906f564e9871b625940549a1b25d3adef",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-15.el9.x86_64",
    sha256 = "56ce4611076497f779467d631ec855a465a559c243635f9531eb365810e2eff5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56ce4611076497f779467d631ec855a465a559c243635f9531eb365810e2eff5",
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
    name = "libffi-0__3.4.2-8.el9.aarch64",
    sha256 = "da6d3f1b21c23a97e61c35fde044aca5bc9f1097ffdcb387759f544c61548301",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libffi-3.4.2-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da6d3f1b21c23a97e61c35fde044aca5bc9f1097ffdcb387759f544c61548301",
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
    name = "libgcc-0__11.4.1-2.3.el9.aarch64",
    sha256 = "ba345e53a3ffaa517bdf01654e3c835fcda21513aca95e04b5d830f36b2b4210",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ba345e53a3ffaa517bdf01654e3c835fcda21513aca95e04b5d830f36b2b4210",
    ],
)

rpm(
    name = "libgcc-0__11.4.1-2.3.el9.x86_64",
    sha256 = "f73077d3a4de0a4899bc3b4cc0389b5cd040cc0e39bd7357b960d5858f4111a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.4.1-2.3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f73077d3a4de0a4899bc3b4cc0389b5cd040cc0e39bd7357b960d5858f4111a4",
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
    name = "libgcrypt-0__1.10.0-10.el9.x86_64",
    sha256 = "186ae69a1f72d3992f2f65a4cc91da856a54475f4762a69f3b5ca5d350e7edb3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.10.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/186ae69a1f72d3992f2f65a4cc91da856a54475f4762a69f3b5ca5d350e7edb3",
    ],
)

rpm(
    name = "libgomp-0__11.4.1-2.3.el9.aarch64",
    sha256 = "7d52d4d5dc1cf88dae7aadbca77847b37f4da22a9864a9f365a79378320e0fde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7d52d4d5dc1cf88dae7aadbca77847b37f4da22a9864a9f365a79378320e0fde",
    ],
)

rpm(
    name = "libgomp-0__11.4.1-2.3.el9.x86_64",
    sha256 = "40d9cb2ff571d2fb7198d9daef43ac24a3785c848e472d5fdca5fb48d879559d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.4.1-2.3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/40d9cb2ff571d2fb7198d9daef43ac24a3785c848e472d5fdca5fb48d879559d",
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
    name = "libguestfs-1__1.50.1-6.el9.x86_64",
    sha256 = "ac7bec669c7bdf784282859c375ca52b53ae64305b1128a8d199928cb029e3da",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libguestfs-1.50.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ac7bec669c7bdf784282859c375ca52b53ae64305b1128a8d199928cb029e3da",
    ],
)

rpm(
    name = "libibverbs-0__46.0-1.el9.aarch64",
    sha256 = "503d39f5db45aeaa5eb6a3d559af7a40cec54c424b03ed1653904160c858976a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libibverbs-46.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/503d39f5db45aeaa5eb6a3d559af7a40cec54c424b03ed1653904160c858976a",
    ],
)

rpm(
    name = "libibverbs-0__46.0-1.el9.x86_64",
    sha256 = "ca7eae95bf6bf989574f10b0fb3cdd82e3d5c871faeb57f6271233cbdbac5cfe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libibverbs-46.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ca7eae95bf6bf989574f10b0fb3cdd82e3d5c871faeb57f6271233cbdbac5cfe",
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
    name = "libksba-0__1.5.1-6.el9.x86_64",
    sha256 = "ff76d9798e2f040fed715968a9e67f6d5cfef59671e07575fc8d6510126b5340",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libksba-1.5.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff76d9798e2f040fed715968a9e67f6d5cfef59671e07575fc8d6510126b5340",
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
    name = "libmount-0__2.37.4-15.el9.aarch64",
    sha256 = "09cb345eba67be22dd50a87b5232c41ed6da5b7a13bff57f8d951a69d9dc4196",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmount-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/09cb345eba67be22dd50a87b5232c41ed6da5b7a13bff57f8d951a69d9dc4196",
    ],
)

rpm(
    name = "libmount-0__2.37.4-15.el9.x86_64",
    sha256 = "b615d8c2fe67089c91025f48c700986c5cb62298e3ed58dbfc3c188af4e77a96",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmount-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b615d8c2fe67089c91025f48c700986c5cb62298e3ed58dbfc3c188af4e77a96",
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
    name = "libnetfilter_conntrack-0__1.0.9-1.el9.aarch64",
    sha256 = "6871a3371b5a9a8239606efd453b59b274040e9d8d8f0c18bdffa7264db64264",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnetfilter_conntrack-1.0.9-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6871a3371b5a9a8239606efd453b59b274040e9d8d8f0c18bdffa7264db64264",
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
    name = "libnfnetlink-0__1.0.1-21.el9.x86_64",
    sha256 = "64f54f412cc0ee6fe82be7557f471a06f6bf1f5bba1d6fe0ad1879e5a62d7c95",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.1-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/64f54f412cc0ee6fe82be7557f471a06f6bf1f5bba1d6fe0ad1879e5a62d7c95",
    ],
)

rpm(
    name = "libnfsidmap-1__2.5.4-20.el9.x86_64",
    sha256 = "bf64d740ff8bbc0c7a9a703425e1d77abe3715211b7517ce537b7b8b04369dcd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfsidmap-2.5.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bf64d740ff8bbc0c7a9a703425e1d77abe3715211b7517ce537b7b8b04369dcd",
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
    name = "libpwquality-0__1.4.4-8.el9.x86_64",
    sha256 = "93f00e5efac1e3f1ecbc0d6a4c068772cb12912cd20c9ea58716d6c0cd004886",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.4-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/93f00e5efac1e3f1ecbc0d6a4c068772cb12912cd20c9ea58716d6c0cd004886",
    ],
)

rpm(
    name = "librdmacm-0__46.0-1.el9.aarch64",
    sha256 = "adf318e099bb6244b519e3c3a556f3b0b3e6f873b7023208783c52901d77e624",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/librdmacm-46.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/adf318e099bb6244b519e3c3a556f3b0b3e6f873b7023208783c52901d77e624",
    ],
)

rpm(
    name = "librdmacm-0__46.0-1.el9.x86_64",
    sha256 = "1d864f77037d8ffc277854e8975440b93f4f55fe4d3c5d0c6097eab81a600318",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/librdmacm-46.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1d864f77037d8ffc277854e8975440b93f4f55fe4d3c5d0c6097eab81a600318",
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
    name = "libselinux-0__3.5-1.el9.aarch64",
    sha256 = "1968d3199e772d0476df14b54b5f85a23329befc1ff7597f45d457b8dc9b0ddd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-3.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1968d3199e772d0476df14b54b5f85a23329befc1ff7597f45d457b8dc9b0ddd",
    ],
)

rpm(
    name = "libselinux-0__3.5-1.el9.x86_64",
    sha256 = "7e7309502af6056593e4c247f1829fd46cc7480ed46da020446ea6c2f1553bd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-3.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e7309502af6056593e4c247f1829fd46cc7480ed46da020446ea6c2f1553bd1",
    ],
)

rpm(
    name = "libselinux-utils-0__3.5-1.el9.aarch64",
    sha256 = "94546ba5175d89231654f3dd12fea21b967bb58c8edbee605dec09c69e260ff7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/94546ba5175d89231654f3dd12fea21b967bb58c8edbee605dec09c69e260ff7",
    ],
)

rpm(
    name = "libselinux-utils-0__3.5-1.el9.x86_64",
    sha256 = "ab57f8b616e7a29cdb72d040b19fa001e68a29f0f8b17e151f0652d6111c66a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ab57f8b616e7a29cdb72d040b19fa001e68a29f0f8b17e151f0652d6111c66a7",
    ],
)

rpm(
    name = "libsemanage-0__3.5-2.el9.aarch64",
    sha256 = "216f1393639dbd62fafb993478db286e7cd8ccf0a411afc46510b80e5cecba68",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsemanage-3.5-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/216f1393639dbd62fafb993478db286e7cd8ccf0a411afc46510b80e5cecba68",
    ],
)

rpm(
    name = "libsemanage-0__3.5-2.el9.x86_64",
    sha256 = "918842b65f93e5a4fe3582178777a3e73591acc58e127a5a0048b62eebc3e10d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsemanage-3.5-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/918842b65f93e5a4fe3582178777a3e73591acc58e127a5a0048b62eebc3e10d",
    ],
)

rpm(
    name = "libsepol-0__3.5-1.el9.aarch64",
    sha256 = "70e6cb0c9d177d512431a1a18ecb7f0bced1e08940df5463961de59fc243ab62",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsepol-3.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/70e6cb0c9d177d512431a1a18ecb7f0bced1e08940df5463961de59fc243ab62",
    ],
)

rpm(
    name = "libsepol-0__3.5-1.el9.x86_64",
    sha256 = "90428114387b69b45fcd7014b219a44ffd89cfecb3bb47c94ca29ab7dce5b940",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsepol-3.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/90428114387b69b45fcd7014b219a44ffd89cfecb3bb47c94ca29ab7dce5b940",
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
    name = "libslirp-0__4.4.0-7.el9.aarch64",
    sha256 = "321ef98abb278174e60823b5f032ef8f5bee45d67a0e2b0a56e08e6ae8a7381b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libslirp-4.4.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/321ef98abb278174e60823b5f032ef8f5bee45d67a0e2b0a56e08e6ae8a7381b",
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
    name = "libsmartcols-0__2.37.4-15.el9.aarch64",
    sha256 = "461a1ed6f3ba43e2f2372eadf18148638b576eca97e6b913fbc283e08e765d4e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/461a1ed6f3ba43e2f2372eadf18148638b576eca97e6b913fbc283e08e765d4e",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-15.el9.x86_64",
    sha256 = "e538fc0ff411e62612f1430d4a97e112a0e77a7059db108c84b972aac6788462",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e538fc0ff411e62612f1430d4a97e112a0e77a7059db108c84b972aac6788462",
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
    name = "libssh-0__0.10.4-11.el9.aarch64",
    sha256 = "78fd484c96b97264c725993874e9891b8a788eb4e19c2dda87996f1cca96cfcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-0.10.4-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/78fd484c96b97264c725993874e9891b8a788eb4e19c2dda87996f1cca96cfcc",
    ],
)

rpm(
    name = "libssh-0__0.10.4-11.el9.x86_64",
    sha256 = "fc9d5d5911e7e5710faff7ee8d86127a97bb820764824fe3308a017a832439b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-0.10.4-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fc9d5d5911e7e5710faff7ee8d86127a97bb820764824fe3308a017a832439b0",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-11.el9.aarch64",
    sha256 = "395ceff19b1fae8a0fcaf4448ebdcd0fa3ed5cbb80b721ab78317c4eebd262b9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-config-0.10.4-11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/395ceff19b1fae8a0fcaf4448ebdcd0fa3ed5cbb80b721ab78317c4eebd262b9",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-11.el9.x86_64",
    sha256 = "395ceff19b1fae8a0fcaf4448ebdcd0fa3ed5cbb80b721ab78317c4eebd262b9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-config-0.10.4-11.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/395ceff19b1fae8a0fcaf4448ebdcd0fa3ed5cbb80b721ab78317c4eebd262b9",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.2-2.el9.aarch64",
    sha256 = "f9195e7e3a5bbb0c75e438bebef1d69711b7d787e176831f0eed1f1fd862593d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.9.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f9195e7e3a5bbb0c75e438bebef1d69711b7d787e176831f0eed1f1fd862593d",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.2-2.el9.x86_64",
    sha256 = "95f315c5b4ebed2a513eabef1b859755e3ba92fb1261904702cdb3d48b2aee48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.9.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95f315c5b4ebed2a513eabef1b859755e3ba92fb1261904702cdb3d48b2aee48",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.2-2.el9.aarch64",
    sha256 = "ae87061b8c581ad899c4d5d5302becf10f4183c2eea1f0f1be6dba6f2baf9047",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.9.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ae87061b8c581ad899c4d5d5302becf10f4183c2eea1f0f1be6dba6f2baf9047",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.2-2.el9.x86_64",
    sha256 = "a4708732a75f75cd7f96d7ef5a1152e223ac622f080a702374318ed6a1ac7c1a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.9.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a4708732a75f75cd7f96d7ef5a1152e223ac622f080a702374318ed6a1ac7c1a",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.4.1-2.3.el9.aarch64",
    sha256 = "ecdbc2771fc50be776f846a6f17d34e2b8e9614f4b89b09930f73fda544b089a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ecdbc2771fc50be776f846a6f17d34e2b8e9614f4b89b09930f73fda544b089a",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.4.1-2.3.el9.x86_64",
    sha256 = "7ef5095ccb164e1b8df68cda4bade0f3c1888536ef8e6724f6a57cb9367e14ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.4.1-2.3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7ef5095ccb164e1b8df68cda4bade0f3c1888536ef8e6724f6a57cb9367e14ff",
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
    name = "libtasn1-0__4.16.0-8.el9.x86_64",
    sha256 = "c8b13c9e1292de474e76ab80f230f86cce2e8f5f53592e168bdcaa604ed1b37d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtasn1-4.16.0-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c8b13c9e1292de474e76ab80f230f86cce2e8f5f53592e168bdcaa604ed1b37d",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-2.el9.aarch64",
    sha256 = "6133ac4ce7568deef7edc2226ee73fb35f3d8102569ce529a4440e3c21501535",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtirpc-1.3.3-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6133ac4ce7568deef7edc2226ee73fb35f3d8102569ce529a4440e3c21501535",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-2.el9.x86_64",
    sha256 = "218b7c8d5d4fbafd404b5059305c420e20f3457dfad981f125c7882832d1b38c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtirpc-1.3.3-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/218b7c8d5d4fbafd404b5059305c420e20f3457dfad981f125c7882832d1b38c",
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
    name = "libtpms-0__0.9.1-3.20211126git1ff6fe1f43.el9.x86_64",
    sha256 = "4ed3052085b118c19f44c3ce29749895627acc590e45acc0722da5d53582afe7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libtpms-0.9.1-3.20211126git1ff6fe1f43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4ed3052085b118c19f44c3ce29749895627acc590e45acc0722da5d53582afe7",
    ],
)

rpm(
    name = "libubsan-0__11.4.1-2.3.el9.aarch64",
    sha256 = "8e4429e26ed9d5f3f70c8e5ede70403aaf9ed1293d480a7c0a19308810a2ca47",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.4.1-2.3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8e4429e26ed9d5f3f70c8e5ede70403aaf9ed1293d480a7c0a19308810a2ca47",
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
    name = "liburing-0__2.3-2.el9.aarch64",
    sha256 = "433ecb131e763b7150b40f11f0a94a7252943dfb3827ca40bd0021884b2a5dc4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/liburing-2.3-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/433ecb131e763b7150b40f11f0a94a7252943dfb3827ca40bd0021884b2a5dc4",
    ],
)

rpm(
    name = "liburing-0__2.3-2.el9.x86_64",
    sha256 = "7d04b63a1d183515a2471ad5813409586fc746ca7bcdc3f8b474543cfc325c9c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/liburing-2.3-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d04b63a1d183515a2471ad5813409586fc746ca7bcdc3f8b474543cfc325c9c",
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
    name = "libuuid-0__2.37.4-15.el9.aarch64",
    sha256 = "6486b8a7e56ca99a40a080dea4f5b71ce4dfefe2692a53c49f565dc6aff9c474",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libuuid-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6486b8a7e56ca99a40a080dea4f5b71ce4dfefe2692a53c49f565dc6aff9c474",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-15.el9.x86_64",
    sha256 = "383cde88a366f8d6bfb9fc29a33f8dc835cdaaf4302dc6fc90d87742a0ab0c00",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/383cde88a366f8d6bfb9fc29a33f8dc835cdaaf4302dc6fc90d87742a0ab0c00",
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
    name = "libvirt-client-0__9.5.0-6.el9.aarch64",
    sha256 = "0237c86ab4ad229c5dd400122ef8d41b20c094d5ed344d048614a2aa564ef5e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-9.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0237c86ab4ad229c5dd400122ef8d41b20c094d5ed344d048614a2aa564ef5e9",
    ],
)

rpm(
    name = "libvirt-client-0__9.5.0-6.el9.x86_64",
    sha256 = "3dbb02da7f4a3343b9085f2ffa455779ae3d8e19b4e6423222eb60c65b48a51b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3dbb02da7f4a3343b9085f2ffa455779ae3d8e19b4e6423222eb60c65b48a51b",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__9.5.0-6.el9.aarch64",
    sha256 = "915e7a996182557b6ecd060efad9ffc0bf0f506b25b4f0f93bf020f7c2119828",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-9.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/915e7a996182557b6ecd060efad9ffc0bf0f506b25b4f0f93bf020f7c2119828",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__9.5.0-6.el9.x86_64",
    sha256 = "506af6eaa7968688ad77e9a37f1bd766cb89b28a79d3feca3ef6cce9f4bb3c77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/506af6eaa7968688ad77e9a37f1bd766cb89b28a79d3feca3ef6cce9f4bb3c77",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__9.5.0-6.el9.aarch64",
    sha256 = "ff7b490069c2a59ea1da42886b70fb8498dcc9858ea4d57d70f381d298e42739",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-9.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ff7b490069c2a59ea1da42886b70fb8498dcc9858ea4d57d70f381d298e42739",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__9.5.0-6.el9.x86_64",
    sha256 = "de16e915037b7295ba20469e4301a6d719d1250e6dd46ba851afac7cb8aef746",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de16e915037b7295ba20469e4301a6d719d1250e6dd46ba851afac7cb8aef746",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__9.5.0-6.el9.x86_64",
    sha256 = "813cd25b5015d7879de39443a5f0795243f53f444ccd281f3393b940c276548a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/813cd25b5015d7879de39443a5f0795243f53f444ccd281f3393b940c276548a",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__9.5.0-6.el9.x86_64",
    sha256 = "e4d877ded56a7a3624f278ed55482d5de68194076d49b3a65060ea87d686ae54",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e4d877ded56a7a3624f278ed55482d5de68194076d49b3a65060ea87d686ae54",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__9.5.0-6.el9.aarch64",
    sha256 = "3c3afeeb6a2a8288db23f88929a2f0183e8c462e6762730c2642ab55be15b11a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-9.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3c3afeeb6a2a8288db23f88929a2f0183e8c462e6762730c2642ab55be15b11a",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__9.5.0-6.el9.x86_64",
    sha256 = "8ce3b735c02503d3093209a359ecf07e6de1da449338df3a1c8bc6d2d0cfedcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ce3b735c02503d3093209a359ecf07e6de1da449338df3a1c8bc6d2d0cfedcc",
    ],
)

rpm(
    name = "libvirt-devel-0__9.5.0-6.el9.aarch64",
    sha256 = "cdaba8bfc69b2c798f455b6c4a902043396d07e6eee25820d4aa1a035767e8d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-9.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cdaba8bfc69b2c798f455b6c4a902043396d07e6eee25820d4aa1a035767e8d8",
    ],
)

rpm(
    name = "libvirt-devel-0__9.5.0-6.el9.x86_64",
    sha256 = "caca5fbbfabe75cf56e24343bf93596f78a91fa341eacb668b15b505245922e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/caca5fbbfabe75cf56e24343bf93596f78a91fa341eacb668b15b505245922e9",
    ],
)

rpm(
    name = "libvirt-libs-0__9.5.0-6.el9.aarch64",
    sha256 = "340b6cdbfefd836dd0f37d6f77ce08f4cbd0c907cc7ea62b1ceddbac54cf76ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-9.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/340b6cdbfefd836dd0f37d6f77ce08f4cbd0c907cc7ea62b1ceddbac54cf76ce",
    ],
)

rpm(
    name = "libvirt-libs-0__9.5.0-6.el9.x86_64",
    sha256 = "3c393f116df3ec9bc02a4dd2fe3d11b19a7188087693dce42f1582ad004a0f9d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-9.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3c393f116df3ec9bc02a4dd2fe3d11b19a7188087693dce42f1582ad004a0f9d",
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
    name = "libxml2-0__2.9.13-4.el9.aarch64",
    sha256 = "a007525b4b82ca2d62cec26e750ee546a4165635dbf2cb39a6e1b579bbf9c035",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.13-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a007525b4b82ca2d62cec26e750ee546a4165635dbf2cb39a6e1b579bbf9c035",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-4.el9.x86_64",
    sha256 = "ee1a3c25255ad5821bd4a7bec9fdc45c77ae2a4671ea3ea96235305e19efec11",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.13-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ee1a3c25255ad5821bd4a7bec9fdc45c77ae2a4671ea3ea96235305e19efec11",
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
    name = "ncurses-base-0__6.2-10.20210508.el9.aarch64",
    sha256 = "00ba56b28a3a85c3c03387bb7abeca92597c8a5fac7f53d48410ca2a20fd8065",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.2-10.20210508.el9.noarch.rpm",
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
    name = "nfs-utils-1__2.5.4-20.el9.x86_64",
    sha256 = "23038d6f6e125dc7f74a4b53f7a9e77bc93cce42d208738676aea8f238a7afb5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nfs-utils-2.5.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23038d6f6e125dc7f74a4b53f7a9e77bc93cce42d208738676aea8f238a7afb5",
    ],
)

rpm(
    name = "nftables-1__1.0.4-11.el9.aarch64",
    sha256 = "1921fa3fb26c4d1249d387783144ebe2be56967c2c838ab7cdd352c4c96aeeb5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.4-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1921fa3fb26c4d1249d387783144ebe2be56967c2c838ab7cdd352c4c96aeeb5",
    ],
)

rpm(
    name = "nftables-1__1.0.4-11.el9.x86_64",
    sha256 = "2d9c555041853486216e5885115b0a8a4c9a346670dce13583ed0a64ceb1f811",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.4-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d9c555041853486216e5885115b0a8a4c9a346670dce13583ed0a64ceb1f811",
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
    name = "numactl-libs-0__2.0.16-3.el9.x86_64",
    sha256 = "56167ea50d70d737d28da028f279f42ac4b624f95ef8f5cce05944cb804230af",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.16-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56167ea50d70d737d28da028f279f42ac4b624f95ef8f5cce05944cb804230af",
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
    name = "openldap-0__2.6.3-1.el9.aarch64",
    sha256 = "2e6e4097eb6b282c94511ee5b96d97523e0f06610570abad918836e8d784050c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openldap-2.6.3-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2e6e4097eb6b282c94511ee5b96d97523e0f06610570abad918836e8d784050c",
    ],
)

rpm(
    name = "openldap-0__2.6.3-1.el9.x86_64",
    sha256 = "847286e28d64a2e52ff858cf09fcc659f2d2d025313249da8d0f6cbd702e51cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openldap-2.6.3-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/847286e28d64a2e52ff858cf09fcc659f2d2d025313249da8d0f6cbd702e51cb",
    ],
)

rpm(
    name = "openssl-1__3.0.7-24.el9.aarch64",
    sha256 = "783a074fc19e611f191c680e3721dc66f2538d5bcf1b0dfbb0d2395c5cf9c521",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.0.7-24.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/783a074fc19e611f191c680e3721dc66f2538d5bcf1b0dfbb0d2395c5cf9c521",
    ],
)

rpm(
    name = "openssl-1__3.0.7-24.el9.x86_64",
    sha256 = "a63f38ebf62f3ee60db2939ebf554797c4a67dde9e58c38d79ca3d6c3b946593",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.0.7-24.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a63f38ebf62f3ee60db2939ebf554797c4a67dde9e58c38d79ca3d6c3b946593",
    ],
)

rpm(
    name = "openssl-libs-1__3.0.7-24.el9.aarch64",
    sha256 = "724ebdfe345be2457c4793cc354d11b926deb2bf27672f758fb6f1c71d1b68d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.0.7-24.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/724ebdfe345be2457c4793cc354d11b926deb2bf27672f758fb6f1c71d1b68d2",
    ],
)

rpm(
    name = "openssl-libs-1__3.0.7-24.el9.x86_64",
    sha256 = "c61b626a5dd42c1df532a51febc1fd24029bf89046be6a5755e47e244e5481d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.0.7-24.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c61b626a5dd42c1df532a51febc1fd24029bf89046be6a5755e47e244e5481d4",
    ],
)

rpm(
    name = "osinfo-db-0__20230518-1.el9.x86_64",
    sha256 = "8f70c46a5dccad0e61fe0500b054f3af5a6ebb5371ec667fd8ee338e13e19afc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/osinfo-db-20230518-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8f70c46a5dccad0e61fe0500b054f3af5a6ebb5371ec667fd8ee338e13e19afc",
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
    name = "pam-0__1.5.1-15.el9.aarch64",
    sha256 = "64e3412816678c0491d22f44adeb764b681b6cac4139480a20125754034f47cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/64e3412816678c0491d22f44adeb764b681b6cac4139480a20125754034f47cb",
    ],
)

rpm(
    name = "pam-0__1.5.1-15.el9.x86_64",
    sha256 = "d2215206d4f6a18e5d485fe312ea891eb8c6669e4079e36676b6e4c2dd7b8bc5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d2215206d4f6a18e5d485fe312ea891eb8c6669e4079e36676b6e4c2dd7b8bc5",
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
    name = "passt-0__0__caret__20230818.g0af928e-4.el9.aarch64",
    sha256 = "d3edd23371bd9363aa2eb0eb4c72026356b45efcbda6e979d71ec6441f014214",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/passt-0%5E20230818.g0af928e-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d3edd23371bd9363aa2eb0eb4c72026356b45efcbda6e979d71ec6441f014214",
    ],
)

rpm(
    name = "passt-0__0__caret__20230818.g0af928e-4.el9.x86_64",
    sha256 = "f5ccbf38ca9dd821fc39508f565b7e8cead55b9eb698058e7bdb8ea50f6e3ad0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/passt-0%5E20230818.g0af928e-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f5ccbf38ca9dd821fc39508f565b7e8cead55b9eb698058e7bdb8ea50f6e3ad0",
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
    name = "pixman-0__0.40.0-6.el9.aarch64",
    sha256 = "949aaa9855119b3372bb4be01b7b2ab87ba9b6c949cad37f411f71553968248f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/pixman-0.40.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/949aaa9855119b3372bb4be01b7b2ab87ba9b6c949cad37f411f71553968248f",
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
    name = "policycoreutils-0__3.5-2.el9.aarch64",
    sha256 = "6176eb53385f6ecddbb7eb4306874b755cecd3acf59683ea739095b001911077",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.5-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6176eb53385f6ecddbb7eb4306874b755cecd3acf59683ea739095b001911077",
    ],
)

rpm(
    name = "policycoreutils-0__3.5-2.el9.x86_64",
    sha256 = "f741f63481fc2f737028b0cf303a6b3916fc6bcf86d22414743f523af0d59dce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.5-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f741f63481fc2f737028b0cf303a6b3916fc6bcf86d22414743f523af0d59dce",
    ],
)

rpm(
    name = "polkit-0__0.117-11.el9.aarch64",
    sha256 = "89396fbd7ee82b5e275b4710e97c66a82e6f8ea3f06801583255c5afb4fbb1a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-0.117-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/89396fbd7ee82b5e275b4710e97c66a82e6f8ea3f06801583255c5afb4fbb1a7",
    ],
)

rpm(
    name = "polkit-0__0.117-11.el9.x86_64",
    sha256 = "aa1dba30bfe853f39cb75bb83de6b4aebf81bed3b743b29ca325558320834b0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-0.117-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aa1dba30bfe853f39cb75bb83de6b4aebf81bed3b743b29ca325558320834b0d",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-11.el9.aarch64",
    sha256 = "80b4bec3572e4be6c17deab334b45b31667b00fec587b339607239429c67b278",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.117-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/80b4bec3572e4be6c17deab334b45b31667b00fec587b339607239429c67b278",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-11.el9.x86_64",
    sha256 = "975049fa240c818d49dfc22716e270de7c193fe526f93f091fc0f91f361921d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.117-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/975049fa240c818d49dfc22716e270de7c193fe526f93f091fc0f91f361921d8",
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
    name = "procps-ng-0__3.3.17-13.el9.aarch64",
    sha256 = "d8220f6f5fe307815c22803c3db6af1e83ef8b79112e8f2c911c9fba1092eee8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.17-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d8220f6f5fe307815c22803c3db6af1e83ef8b79112e8f2c911c9fba1092eee8",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-13.el9.x86_64",
    sha256 = "20932dc3e2818d562acb5fb60e4b5c222fc2f5cca77988cd7e9be4a81647b339",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.17-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20932dc3e2818d562acb5fb60e4b5c222fc2f5cca77988cd7e9be4a81647b339",
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
    name = "python3-0__3.9.17-2.el9.aarch64",
    sha256 = "733c7d872287738bc4c412b4c1e4208e199ac8fdcfdde58bc9b25cfa12105f60",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.17-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/733c7d872287738bc4c412b4c1e4208e199ac8fdcfdde58bc9b25cfa12105f60",
    ],
)

rpm(
    name = "python3-0__3.9.17-2.el9.x86_64",
    sha256 = "0c908646997be2a982c6f971230ca467c1039ad5a6cb961ee86d2c0ed1586983",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.17-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c908646997be2a982c6f971230ca467c1039ad5a6cb961ee86d2c0ed1586983",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.28-7.el9.aarch64",
    sha256 = "39d28be696eb7b915c2d0be2da6a4f98ed8888ad03acd5c30f863387adfc5386",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-configshell-1.1.28-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/39d28be696eb7b915c2d0be2da6a4f98ed8888ad03acd5c30f863387adfc5386",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.28-7.el9.x86_64",
    sha256 = "39d28be696eb7b915c2d0be2da6a4f98ed8888ad03acd5c30f863387adfc5386",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-configshell-1.1.28-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/39d28be696eb7b915c2d0be2da6a4f98ed8888ad03acd5c30f863387adfc5386",
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
    name = "python3-kmod-0__0.9-32.el9.x86_64",
    sha256 = "e0b0ae0d507496349b667e0281b4d72ac3f7b7fa65c633c56afa3f328855a2d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-kmod-0.9-32.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e0b0ae0d507496349b667e0281b4d72ac3f7b7fa65c633c56afa3f328855a2d9",
    ],
)

rpm(
    name = "python3-libs-0__3.9.17-2.el9.aarch64",
    sha256 = "dd9cfbb8dc9714960ff65b0ba49102fd1b7b8132bc1917a2c32aa33702f913dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.17-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dd9cfbb8dc9714960ff65b0ba49102fd1b7b8132bc1917a2c32aa33702f913dc",
    ],
)

rpm(
    name = "python3-libs-0__3.9.17-2.el9.x86_64",
    sha256 = "ceeede42edd33b2ac964dd2fce66d77363be4530cce206185ae168186696bee4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.17-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ceeede42edd33b2ac964dd2fce66d77363be4530cce206185ae168186696bee4",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.2.3-7.el9.aarch64",
    sha256 = "629fa0b71150272bfc7d4f348b04d2588a70fcdd5c3b52e176ea3a7d3de71e89",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-21.2.3-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/629fa0b71150272bfc7d4f348b04d2588a70fcdd5c3b52e176ea3a7d3de71e89",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.2.3-7.el9.x86_64",
    sha256 = "629fa0b71150272bfc7d4f348b04d2588a70fcdd5c3b52e176ea3a7d3de71e89",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-21.2.3-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/629fa0b71150272bfc7d4f348b04d2588a70fcdd5c3b52e176ea3a7d3de71e89",
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
    name = "python3-pyudev-0__0.22.0-6.el9.x86_64",
    sha256 = "db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pyudev-0.22.0-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db815d76afabb8dd7eca6ca5a5bf838304f82824c41e4f06b6d25b5eb63c65c6",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.75-1.el9.aarch64",
    sha256 = "ddeac3075ca78cb0c61a229aec9d534aa99b9a6bb2b242a39e28baf6ef4c2f64",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-rtslib-2.1.75-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ddeac3075ca78cb0c61a229aec9d534aa99b9a6bb2b242a39e28baf6ef4c2f64",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.75-1.el9.x86_64",
    sha256 = "ddeac3075ca78cb0c61a229aec9d534aa99b9a6bb2b242a39e28baf6ef4c2f64",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-rtslib-2.1.75-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ddeac3075ca78cb0c61a229aec9d534aa99b9a6bb2b242a39e28baf6ef4c2f64",
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
    name = "python3-urwid-0__2.1.2-4.el9.x86_64",
    sha256 = "b4e4915a49904035e0e8d8ed15a545f2d7191e9d760c438343980fbf0b66abf4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-urwid-2.1.2-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4e4915a49904035e0e8d8ed15a545f2d7191e9d760c438343980fbf0b66abf4",
    ],
)

rpm(
    name = "qemu-img-17__8.0.0-13.el9.aarch64",
    sha256 = "4e5e088c57310147f64a0dc8b7096a0fe9daac8cee6149c42687a0679923fc39",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-8.0.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4e5e088c57310147f64a0dc8b7096a0fe9daac8cee6149c42687a0679923fc39",
    ],
)

rpm(
    name = "qemu-img-17__8.0.0-13.el9.x86_64",
    sha256 = "2f2b859728eaacf2034ec755f5b1b8a35650f46bb11ec8005d59400ceaefaefa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-8.0.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2f2b859728eaacf2034ec755f5b1b8a35650f46bb11ec8005d59400ceaefaefa",
    ],
)

rpm(
    name = "qemu-kvm-common-17__8.0.0-13.el9.aarch64",
    sha256 = "cd563f5f7e11852e878dad78f0c80de9c60d89ebab0eff336a27582c3315635c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-8.0.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cd563f5f7e11852e878dad78f0c80de9c60d89ebab0eff336a27582c3315635c",
    ],
)

rpm(
    name = "qemu-kvm-common-17__8.0.0-13.el9.x86_64",
    sha256 = "a5c40e85781ccfc10c98975df9582651199893b513521c2b79b81a50f48b2127",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-8.0.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a5c40e85781ccfc10c98975df9582651199893b513521c2b79b81a50f48b2127",
    ],
)

rpm(
    name = "qemu-kvm-core-17__8.0.0-13.el9.aarch64",
    sha256 = "1616a8d2fb11c963cb2de8c6cabf151840c939b928869a0f3633dddd245b66c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-8.0.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1616a8d2fb11c963cb2de8c6cabf151840c939b928869a0f3633dddd245b66c2",
    ],
)

rpm(
    name = "qemu-kvm-core-17__8.0.0-13.el9.x86_64",
    sha256 = "7887c4bf06b08b057af9c96352c8007f952836a5940a9c8f8a76a9388fd7e338",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-8.0.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7887c4bf06b08b057af9c96352c8007f952836a5940a9c8f8a76a9388fd7e338",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__8.0.0-13.el9.aarch64",
    sha256 = "8098025cb36fe31938efc76c6ec61537ae353b638472f9ccfe6c1d0d5a630634",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-8.0.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8098025cb36fe31938efc76c6ec61537ae353b638472f9ccfe6c1d0d5a630634",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__8.0.0-13.el9.aarch64",
    sha256 = "a5c1663351101ed443925e05150d8e91cda71a4a73deeb0b6abc6ea17726b2d5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-8.0.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a5c1663351101ed443925e05150d8e91cda71a4a73deeb0b6abc6ea17726b2d5",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__8.0.0-13.el9.aarch64",
    sha256 = "6766a8164e99c82cd985bd96f2653381564dada2ecb373ff8fc462733ce75ee8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-8.0.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6766a8164e99c82cd985bd96f2653381564dada2ecb373ff8fc462733ce75ee8",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__8.0.0-13.el9.x86_64",
    sha256 = "b42fbf8706a056377b87bd82c298d81261a89200e2768e90be84b3773491e59e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-8.0.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b42fbf8706a056377b87bd82c298d81261a89200e2768e90be84b3773491e59e",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__8.0.0-13.el9.x86_64",
    sha256 = "6de8f93673068504e702035ea064070663a3c8e7e91366e9b39a4ca8c8829597",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-8.0.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6de8f93673068504e702035ea064070663a3c8e7e91366e9b39a4ca8c8829597",
    ],
)

rpm(
    name = "qemu-pr-helper-17__8.1.0-2.el9.aarch64",
    sha256 = "4eafe344bedd2ff1153974dae35676de4d40913843b6b6d7b44432dde5303830",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-8.1.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4eafe344bedd2ff1153974dae35676de4d40913843b6b6d7b44432dde5303830",
    ],
)

rpm(
    name = "qemu-pr-helper-17__8.1.0-2.el9.x86_64",
    sha256 = "7f0c8eff52df80b286e4def800f494af10d394a1420ac1f78658e7e17c5e7434",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-8.1.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f0c8eff52df80b286e4def800f494af10d394a1420ac1f78658e7e17c5e7434",
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
    name = "rpm-0__4.16.1.3-25.el9.aarch64",
    sha256 = "adf72ac7d0717f39699f9efa99c07fdd4ad173537d92458cba831a80330ab972",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-4.16.1.3-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/adf72ac7d0717f39699f9efa99c07fdd4ad173537d92458cba831a80330ab972",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-25.el9.x86_64",
    sha256 = "6096fbadb206559bf8e046a1eb849a62980998bad3424422495d0def3f304141",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6096fbadb206559bf8e046a1eb849a62980998bad3424422495d0def3f304141",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-25.el9.aarch64",
    sha256 = "a3fc2420f987f51949e04ecd7cc3d0268d2b0f93859ec650c390afa0b152419c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.16.1.3-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a3fc2420f987f51949e04ecd7cc3d0268d2b0f93859ec650c390afa0b152419c",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-25.el9.x86_64",
    sha256 = "314a24e8e0b51bfd355cc4bbc8bc4981a9822a4aabde75f1227b9ab63fc4d15d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/314a24e8e0b51bfd355cc4bbc8bc4981a9822a4aabde75f1227b9ab63fc4d15d",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-25.el9.aarch64",
    sha256 = "dce5e366b0a534b22c1d6f4b9d1d600c2f1c0d83ddc6b9ac5f820a6ccf286258",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.16.1.3-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dce5e366b0a534b22c1d6f4b9d1d600c2f1c0d83ddc6b9ac5f820a6ccf286258",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-25.el9.x86_64",
    sha256 = "9f0b8c3e8e7af9d08e28282907cbea17d1eca5b109525fe223f741016620f3e2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f0b8c3e8e7af9d08e28282907cbea17d1eca5b109525fe223f741016620f3e2",
    ],
)

rpm(
    name = "seabios-0__1.16.1-1.el9.x86_64",
    sha256 = "eec7c900965dfee16c97a341374c4592d5d194694f9a4af998be3f1e56f6bbad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-1.16.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eec7c900965dfee16c97a341374c4592d5d194694f9a4af998be3f1e56f6bbad",
    ],
)

rpm(
    name = "seabios-bin-0__1.16.1-1.el9.x86_64",
    sha256 = "bc66dda921365d3e1c99a989c4e7344bb1bebf7da34af910741dff599a2a950c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-bin-1.16.1-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bc66dda921365d3e1c99a989c4e7344bb1bebf7da34af910741dff599a2a950c",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.16.1-1.el9.x86_64",
    sha256 = "3032204d68939ad64b7f245adf578c75c9d7f8ed579cf2f06a77d4d97e57a966",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seavgabios-bin-1.16.1-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3032204d68939ad64b7f245adf578c75c9d7f8ed579cf2f06a77d4d97e57a966",
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
    name = "selinux-policy-0__38.1.24-1.el9.aarch64",
    sha256 = "d3fc153588a9b76885666aec4fb1bb5a844504a3c4b3685d7f55fcc5cfefbebe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-38.1.24-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d3fc153588a9b76885666aec4fb1bb5a844504a3c4b3685d7f55fcc5cfefbebe",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.24-1.el9.x86_64",
    sha256 = "d3fc153588a9b76885666aec4fb1bb5a844504a3c4b3685d7f55fcc5cfefbebe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-38.1.24-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d3fc153588a9b76885666aec4fb1bb5a844504a3c4b3685d7f55fcc5cfefbebe",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.24-1.el9.aarch64",
    sha256 = "2b7a1e5589b113cfef927c4fd2a5c0dc5de30b3e8e5708b5d9fe8ae5d4fe1fc9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-38.1.24-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2b7a1e5589b113cfef927c4fd2a5c0dc5de30b3e8e5708b5d9fe8ae5d4fe1fc9",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.24-1.el9.x86_64",
    sha256 = "2b7a1e5589b113cfef927c4fd2a5c0dc5de30b3e8e5708b5d9fe8ae5d4fe1fc9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-38.1.24-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2b7a1e5589b113cfef927c4fd2a5c0dc5de30b3e8e5708b5d9fe8ae5d4fe1fc9",
    ],
)

rpm(
    name = "setup-0__2.13.7-9.el9.aarch64",
    sha256 = "e1b7458eff8a50015cdfaef129aeebf663ffd70a5b94f4e3318a7603023de8ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/setup-2.13.7-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e1b7458eff8a50015cdfaef129aeebf663ffd70a5b94f4e3318a7603023de8ae",
    ],
)

rpm(
    name = "setup-0__2.13.7-9.el9.x86_64",
    sha256 = "e1b7458eff8a50015cdfaef129aeebf663ffd70a5b94f4e3318a7603023de8ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/setup-2.13.7-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e1b7458eff8a50015cdfaef129aeebf663ffd70a5b94f4e3318a7603023de8ae",
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
    name = "snappy-0__1.1.8-8.el9.x86_64",
    sha256 = "10facee86b64af91b06292ca9892fd94fe5fc08c068b0baed6a0927d6a64955a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/snappy-1.1.8-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10facee86b64af91b06292ca9892fd94fe5fc08c068b0baed6a0927d6a64955a",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-6.el9.aarch64",
    sha256 = "14ebed56d97af9a87504d2bf4c1c52f68e514cba6fb308ef559a0ed18e51d77f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.34.1-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/14ebed56d97af9a87504d2bf4c1c52f68e514cba6fb308ef559a0ed18e51d77f",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-6.el9.x86_64",
    sha256 = "440da6dd7ad99e29e540626efe09650add959846d00a9759f0c4a417161d911e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/440da6dd7ad99e29e540626efe09650add959846d00a9759f0c4a417161d911e",
    ],
)

rpm(
    name = "sssd-client-0__2.9.2-2.el9.aarch64",
    sha256 = "3858e8ed3fb22c176aa9f8e4035c7503dbbb16f7ffc6546767f6a8c280969960",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.9.2-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3858e8ed3fb22c176aa9f8e4035c7503dbbb16f7ffc6546767f6a8c280969960",
    ],
)

rpm(
    name = "sssd-client-0__2.9.2-2.el9.x86_64",
    sha256 = "edea13ebef3838a1d322c8a5e67cd77eada96f170deff9d43109a53362523c23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.9.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/edea13ebef3838a1d322c8a5e67cd77eada96f170deff9d43109a53362523c23",
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
    name = "swtpm-tools-0__0.8.0-1.el9.x86_64",
    sha256 = "b3531cb8d1d4408945ae6ca072852d6a9cf82300061a263b11bd841cf51ea037",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.8.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3531cb8d1d4408945ae6ca072852d6a9cf82300061a263b11bd841cf51ea037",
    ],
)

rpm(
    name = "systemd-0__252-18.el9.aarch64",
    sha256 = "b227048150dea6866efbcdb67f3f1a4f6fc89531fc3827bf11263857f442546a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-252-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b227048150dea6866efbcdb67f3f1a4f6fc89531fc3827bf11263857f442546a",
    ],
)

rpm(
    name = "systemd-0__252-18.el9.x86_64",
    sha256 = "9c0c1834dcc3db548f97cdfd6a6ac65a261fd4d8224edf46e30bb88b2b2c63f1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9c0c1834dcc3db548f97cdfd6a6ac65a261fd4d8224edf46e30bb88b2b2c63f1",
    ],
)

rpm(
    name = "systemd-container-0__252-18.el9.aarch64",
    sha256 = "4cebef3662f6c0de51bcd967debc8b8898f4cb112e1745772d2027f1f5588eee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-252-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4cebef3662f6c0de51bcd967debc8b8898f4cb112e1745772d2027f1f5588eee",
    ],
)

rpm(
    name = "systemd-container-0__252-18.el9.x86_64",
    sha256 = "69591d4e37541f3acd73156c1766c8acb84cc5d5d174cb140f1fd3275bcf2c5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/69591d4e37541f3acd73156c1766c8acb84cc5d5d174cb140f1fd3275bcf2c5f",
    ],
)

rpm(
    name = "systemd-libs-0__252-18.el9.aarch64",
    sha256 = "f71ca1cfedf8f65a7b7212fb9dc9fd1a8b510bd740d5e79d46acece46f933d05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-252-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f71ca1cfedf8f65a7b7212fb9dc9fd1a8b510bd740d5e79d46acece46f933d05",
    ],
)

rpm(
    name = "systemd-libs-0__252-18.el9.x86_64",
    sha256 = "a572727265b31de5d74c8065f452e785484158941da4c960148e1618ab0b64dd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a572727265b31de5d74c8065f452e785484158941da4c960148e1618ab0b64dd",
    ],
)

rpm(
    name = "systemd-pam-0__252-18.el9.aarch64",
    sha256 = "d246ba6dac02d263d2f220321dd672f2b21cfafc9983f744efd10eacaf100979",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-252-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d246ba6dac02d263d2f220321dd672f2b21cfafc9983f744efd10eacaf100979",
    ],
)

rpm(
    name = "systemd-pam-0__252-18.el9.x86_64",
    sha256 = "fdafbf55660897e5bf33c9cf39c73b3c7a24e4e5677272490855427053a6799b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fdafbf55660897e5bf33c9cf39c73b3c7a24e4e5677272490855427053a6799b",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-18.el9.aarch64",
    sha256 = "8e3fef50d8c964af111bddd370675deabbc2b3e3c68a9b9c0492463e2b51ef08",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-252-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8e3fef50d8c964af111bddd370675deabbc2b3e3c68a9b9c0492463e2b51ef08",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-18.el9.x86_64",
    sha256 = "8e3fef50d8c964af111bddd370675deabbc2b3e3c68a9b9c0492463e2b51ef08",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8e3fef50d8c964af111bddd370675deabbc2b3e3c68a9b9c0492463e2b51ef08",
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
    name = "tar-2__1.34-6.el9.x86_64",
    sha256 = "9f6adb2da035d5123587a2bb401487521bd6543497003ffc6e66386d898133f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tar-1.34-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f6adb2da035d5123587a2bb401487521bd6543497003ffc6e66386d898133f3",
    ],
)

rpm(
    name = "target-restore-0__2.1.75-1.el9.aarch64",
    sha256 = "b8cd2a141abaf56b14e1792ecacf5e60f43bb769c9a014cd518a528a46d7fe28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/target-restore-2.1.75-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b8cd2a141abaf56b14e1792ecacf5e60f43bb769c9a014cd518a528a46d7fe28",
    ],
)

rpm(
    name = "target-restore-0__2.1.75-1.el9.x86_64",
    sha256 = "b8cd2a141abaf56b14e1792ecacf5e60f43bb769c9a014cd518a528a46d7fe28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/target-restore-2.1.75-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b8cd2a141abaf56b14e1792ecacf5e60f43bb769c9a014cd518a528a46d7fe28",
    ],
)

rpm(
    name = "targetcli-0__2.1.53-7.el9.aarch64",
    sha256 = "a3a04950e17fa74236978efa7ff167ad8830fac3b74abc22af825cfcdeddde13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/targetcli-2.1.53-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a3a04950e17fa74236978efa7ff167ad8830fac3b74abc22af825cfcdeddde13",
    ],
)

rpm(
    name = "targetcli-0__2.1.53-7.el9.x86_64",
    sha256 = "a3a04950e17fa74236978efa7ff167ad8830fac3b74abc22af825cfcdeddde13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/targetcli-2.1.53-7.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a3a04950e17fa74236978efa7ff167ad8830fac3b74abc22af825cfcdeddde13",
    ],
)

rpm(
    name = "tzdata-0__2023c-1.el9.aarch64",
    sha256 = "6990005a7665404476ca1a274a5e195ca3afbb5763b51720ce2c3127cc5e6114",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2023c-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6990005a7665404476ca1a274a5e195ca3afbb5763b51720ce2c3127cc5e6114",
    ],
)

rpm(
    name = "tzdata-0__2023c-1.el9.x86_64",
    sha256 = "6990005a7665404476ca1a274a5e195ca3afbb5763b51720ce2c3127cc5e6114",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2023c-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6990005a7665404476ca1a274a5e195ca3afbb5763b51720ce2c3127cc5e6114",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-3.el9.aarch64",
    sha256 = "a8919b80ffaeae3d7a2247062bac1fbaf9de7c7e9aab5bd36181b758043ee2d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/unbound-libs-1.16.2-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a8919b80ffaeae3d7a2247062bac1fbaf9de7c7e9aab5bd36181b758043ee2d0",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-3.el9.x86_64",
    sha256 = "db922b8fc89c38939f879d25909f6881ef736580925642fd3f4fbf8e93a7d139",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/unbound-libs-1.16.2-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/db922b8fc89c38939f879d25909f6881ef736580925642fd3f4fbf8e93a7d139",
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
    name = "util-linux-0__2.37.4-15.el9.aarch64",
    sha256 = "6b4a34b132ab405fcede4b2f027998640acc1285ab91ae3146e9a7dab0d6e271",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6b4a34b132ab405fcede4b2f027998640acc1285ab91ae3146e9a7dab0d6e271",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-15.el9.x86_64",
    sha256 = "b7a2259bfb358029dadb493d5d746462f1c7c0244e450ccdc7a8681ac263b9ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7a2259bfb358029dadb493d5d746462f1c7c0244e450ccdc7a8681ac263b9ed",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-15.el9.aarch64",
    sha256 = "8a4e235a2ead88eb2b9e5b45333e60061b6c8062abcd5c3d5c388a745541046f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.37.4-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8a4e235a2ead88eb2b9e5b45333e60061b6c8062abcd5c3d5c388a745541046f",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-15.el9.x86_64",
    sha256 = "027d0fa1342fb2118784f910c390d33a5e145237ca7e2bceec0342dd28598232",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.37.4-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/027d0fa1342fb2118784f910c390d33a5e145237ca7e2bceec0342dd28598232",
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
    name = "vim-minimal-2__8.2.2637-20.el9.x86_64",
    sha256 = "5bef7d6b66ece8820a758a6b1fb99a4512dd3bdcac0774723b630bfd5144ee62",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5bef7d6b66ece8820a758a6b1fb99a4512dd3bdcac0774723b630bfd5144ee62",
    ],
)

rpm(
    name = "virtiofsd-0__1.7.2-1.el9.aarch64",
    sha256 = "ee73bcdc283d8a086f2f4af34e8d10bb5a706ac10407fa14e8374ae7347748f9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/virtiofsd-1.7.2-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ee73bcdc283d8a086f2f4af34e8d10bb5a706ac10407fa14e8374ae7347748f9",
    ],
)

rpm(
    name = "virtiofsd-0__1.7.2-1.el9.x86_64",
    sha256 = "6b8d0956739b53a7e99f0867623f05a5278aa13406dfbd8c45e6fecaa3ba463b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/virtiofsd-1.7.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6b8d0956739b53a7e99f0867623f05a5278aa13406dfbd8c45e6fecaa3ba463b",
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
    name = "yajl-0__2.1.0-22.el9.aarch64",
    sha256 = "5f099ce8836377f6aba662e5835cc500b2e8f29cd8c9b56b22df7c564f7d209c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/yajl-2.1.0-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5f099ce8836377f6aba662e5835cc500b2e8f29cd8c9b56b22df7c564f7d209c",
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
    name = "zlib-0__1.2.11-40.el9.aarch64",
    sha256 = "dfba73a51e7d01bf239d6bc58270814da76081c9666a2ae0ce6d28d0a479e766",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/zlib-1.2.11-40.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dfba73a51e7d01bf239d6bc58270814da76081c9666a2ae0ce6d28d0a479e766",
    ],
)

rpm(
    name = "zlib-0__1.2.11-40.el9.x86_64",
    sha256 = "8a9f51eac4658d4d05c883cbef15ae7b08acf274a46b4c4d9d28a3e2ae9f5b47",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/zlib-1.2.11-40.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a9f51eac4658d4d05c883cbef15ae7b08acf274a46b4c4d9d28a3e2ae9f5b47",
    ],
)
