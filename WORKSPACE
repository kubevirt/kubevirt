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

# Bazel buildtools prebuilt binaries
http_archive(
    name = "buildifier_prebuilt",
    sha256 = "8ada9d88e51ebf5a1fdff37d75ed41d51f5e677cdbeafb0a22dda54747d6e07e",
    strip_prefix = "buildifier-prebuilt-6.4.0",
    urls = [
        "http://github.com/keith/buildifier-prebuilt/archive/6.4.0.tar.gz",
        "https://storage.googleapis.com/builddeps/8ada9d88e51ebf5a1fdff37d75ed41d51f5e677cdbeafb0a22dda54747d6e07e",
    ],
)

load("@buildifier_prebuilt//:deps.bzl", "buildifier_prebuilt_deps")

buildifier_prebuilt_deps()

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

load("@buildifier_prebuilt//:defs.bzl", "buildifier_prebuilt_register_toolchains")

buildifier_prebuilt_register_toolchains()

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
    sha256 = "f87a0fd3ab0e65d2a84acd5dad5f8b6afce51cb465f65dd6f8a3810a3723b6e4",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/x86_64/alpine-virt-3.20.1-x86_64.iso",
        "https://storage.googleapis.com/builddeps/f87a0fd3ab0e65d2a84acd5dad5f8b6afce51cb465f65dd6f8a3810a3723b6e4",
    ],
)

http_file(
    name = "alpine_image_aarch64",
    sha256 = "ca2f0e8aa7a1d7917bce7b9e7bd413772b64ec529a1938d20352558f90a5035a",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.20/releases/aarch64/alpine-virt-3.20.1-aarch64.iso",
        "https://storage.googleapis.com/builddeps/ca2f0e8aa7a1d7917bce7b9e7bd413772b64ec529a1938d20352558f90a5035a",
    ],
)

http_file(
    name = "alpine_image_s390x",
    sha256 = "4ca1462252246d53e4949523b87fcea088e8b4992dbd6df792818c5875069b16",
    urls = [
        "https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/s390x/alpine-standard-3.18.8-s390x.iso",
        "https://storage.googleapis.com/builddeps/4ca1462252246d53e4949523b87fcea088e8b4992dbd6df792818c5875069b16",
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
    sha256 = "57b0f6dc8dc92dc2ae8621f8b1bfbd8a873de9bedc788c4c4b305ea28acc77cd",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.266-1/virtio-win-0.1.266.iso",
        "https://storage.googleapis.com/builddeps/57b0f6dc8dc92dc2ae8621f8b1bfbd8a873de9bedc788c4c4b305ea28acc77cd",
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
    go_version = "1.22.8",
    nogo = "@//:nogo_vet",
)

load("@com_github_ash2k_bazel_tools//goimports:deps.bzl", "goimports_dependencies")

goimports_dependencies()

load(
    "@bazel_gazelle//:deps.bzl",
    "gazelle_dependencies",
    "go_repository",
)

go_repository(
    name = "org_golang_google_grpc",
    build_file_proto_mode = "disable",
    importpath = "google.golang.org/grpc",
    sum = "h1:BjnpXut1btbtgN/6sp+brB2Kbm2LjNXnidYujAVbSoQ=",
    version = "v1.58.3",
)

go_repository(
    name = "org_golang_google_genproto_googleapis_rpc",
    build_file_proto_mode = "disable_global",
    importpath = "google.golang.org/genproto/googleapis/rpc",
    sum = "h1:uvYuEyMHKNt+lT4K3bN6fGswmK8qSvcreM3BwjDh+y4=",
    version = "v0.0.0-20230822172742-b8732ec3820d",
)

gazelle_dependencies()

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
    digest = "sha256:a7af3ef5d69f6534ba0492cc7d6b8fbcffddcb02511b45becc2fac752f907584",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
)

container_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:198302a46cd40ab2e24ee54d39ba0919a431e59289fd7b87f798b62e2076c62a",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
)

container_pull(
    name = "go_image_base_s390x",
    digest = "sha256:642791d0afe3d071e365923e65203074f30bad4ca621309d2eab52bf2d32077e",
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
    digest = "sha256:ffcfed26f1784535ec5a2fed49ed80ccfd774aa09c665f95835c3d3bf3ec37aa",
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
    digest = "sha256:bae2ed95318223e6bd367efbd2839952c698b938f855684dc48eff29a2bbc9af",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
)

container_pull(
    name = "fedora_with_test_tooling_s390x",
    digest = "sha256:43eb8c7942c98e5380a7ec816a2072617184a1d3ec2bcf225539db412d56ea3e",
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
    name = "audit-libs-0__3.1.5-1.el9.aarch64",
    sha256 = "ce97ff90c24105c48d6ef29b0643021f366048f10c79c7f3d81e3f0f9483d5e6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/audit-libs-3.1.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ce97ff90c24105c48d6ef29b0643021f366048f10c79c7f3d81e3f0f9483d5e6",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-1.el9.s390x",
    sha256 = "090ef1e4057d3235a050ad72728f40752faa6958a7f3ee6ebd0cd43e5f97d026",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/audit-libs-3.1.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/090ef1e4057d3235a050ad72728f40752faa6958a7f3ee6ebd0cd43e5f97d026",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-1.el9.x86_64",
    sha256 = "e1998c3847956ad86d846f8b857e5382897ef2f444b4a2ef8e82a0cb8b1aa1ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.1.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e1998c3847956ad86d846f8b857e5382897ef2f444b4a2ef8e82a0cb8b1aa1ad",
    ],
)

rpm(
    name = "augeas-libs-0__1.14.1-2.el9.x86_64",
    sha256 = "f391f8f22e87442cb03e2f822e1b869f49af4b8a6587cdfb05a18eb368eece7b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/augeas-libs-1.14.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f391f8f22e87442cb03e2f822e1b869f49af4b8a6587cdfb05a18eb368eece7b",
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
    name = "binutils-0__2.35.2-56.el9.aarch64",
    sha256 = "cabe60a60bab83454a15d5e0956b3fd9b84440085db0f9901cf078985aaeb99f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-56.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cabe60a60bab83454a15d5e0956b3fd9b84440085db0f9901cf078985aaeb99f",
    ],
)

rpm(
    name = "binutils-0__2.35.2-56.el9.s390x",
    sha256 = "23afc801a3b0a86ac1755f6574667c569498ef2c9f496c31933d1d46542c7a11",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-2.35.2-56.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/23afc801a3b0a86ac1755f6574667c569498ef2c9f496c31933d1d46542c7a11",
    ],
)

rpm(
    name = "binutils-0__2.35.2-56.el9.x86_64",
    sha256 = "8e167e6bb501c0ab5f4e74523de54283498a715f9068f0fc5d3115ea01476415",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-56.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e167e6bb501c0ab5f4e74523de54283498a715f9068f0fc5d3115ea01476415",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-56.el9.aarch64",
    sha256 = "de47fcb9c36903ae418034b28cb6351765cd7b1811a6e7ca235c1c120cb5b54c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-56.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de47fcb9c36903ae418034b28cb6351765cd7b1811a6e7ca235c1c120cb5b54c",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-56.el9.s390x",
    sha256 = "200921a1fb254a667bee33d12a1bad63baa9c8b98a4b27921b7991f71e5a6a49",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-gold-2.35.2-56.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/200921a1fb254a667bee33d12a1bad63baa9c8b98a4b27921b7991f71e5a6a49",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-56.el9.x86_64",
    sha256 = "124fdda54c36c2ce5cef3b5ebb8db896a3b45a0908679abdaeffef928a5193c1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-56.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/124fdda54c36c2ce5cef3b5ebb8db896a3b45a0908679abdaeffef928a5193c1",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-9.el9.aarch64",
    sha256 = "a2724258cca82162a124c1555bccfa6346c1b7ebfc7e57a6ee55bb2e9e267fa7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.8-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a2724258cca82162a124c1555bccfa6346c1b7ebfc7e57a6ee55bb2e9e267fa7",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-9.el9.s390x",
    sha256 = "ce76e8b78599de8c744974a1fc611e20b4ff01e304942f691f31085ec876453c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-1.0.8-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ce76e8b78599de8c744974a1fc611e20b4ff01e304942f691f31085ec876453c",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-9.el9.x86_64",
    sha256 = "4d27d2d3bf09a52183a1299949d63cbc3072d24b8cfe3144dba74976ae21ed14",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.8-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d27d2d3bf09a52183a1299949d63cbc3072d24b8cfe3144dba74976ae21ed14",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-9.el9.aarch64",
    sha256 = "ab156bc96f02f4f9b8f5017bf0a002de021b2d0bd9467b5d6b9ccf31def005bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.8-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab156bc96f02f4f9b8f5017bf0a002de021b2d0bd9467b5d6b9ccf31def005bb",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-9.el9.s390x",
    sha256 = "fc412de11ef77e37c935d00370d37af32cda8a9f2bd40462523b1a181efe659a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-libs-1.0.8-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fc412de11ef77e37c935d00370d37af32cda8a9f2bd40462523b1a181efe659a",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-9.el9.x86_64",
    sha256 = "9a16a6163b4819dd56936bccf9ddb870934d5dd037124cd6be65d78f5bd94690",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.8-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a16a6163b4819dd56936bccf9ddb870934d5dd037124cd6be65d78f5bd94690",
    ],
)

rpm(
    name = "ca-certificates-0__2024.2.69_v8.0.303-91.4.el9.aarch64",
    sha256 = "d18c1b9763c22dc93da804f96ad3d92b3157195c9eff6e923c33e9011df3e246",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ca-certificates-2024.2.69_v8.0.303-91.4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d18c1b9763c22dc93da804f96ad3d92b3157195c9eff6e923c33e9011df3e246",
    ],
)

rpm(
    name = "ca-certificates-0__2024.2.69_v8.0.303-91.4.el9.s390x",
    sha256 = "d18c1b9763c22dc93da804f96ad3d92b3157195c9eff6e923c33e9011df3e246",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ca-certificates-2024.2.69_v8.0.303-91.4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d18c1b9763c22dc93da804f96ad3d92b3157195c9eff6e923c33e9011df3e246",
    ],
)

rpm(
    name = "ca-certificates-0__2024.2.69_v8.0.303-91.4.el9.x86_64",
    sha256 = "d18c1b9763c22dc93da804f96ad3d92b3157195c9eff6e923c33e9011df3e246",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ca-certificates-2024.2.69_v8.0.303-91.4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d18c1b9763c22dc93da804f96ad3d92b3157195c9eff6e923c33e9011df3e246",
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
    name = "coreutils-single-0__8.32-37.el9.aarch64",
    sha256 = "022083ab78d191e4c09e0f33c9355fcc8d6ae4d28214bc4b6379bfb119e47518",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.32-37.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/022083ab78d191e4c09e0f33c9355fcc8d6ae4d28214bc4b6379bfb119e47518",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-37.el9.s390x",
    sha256 = "a1a938da348fbd2ebc6a7430b11a5e2f460464df0828a59b59d520836ea12c27",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/coreutils-single-8.32-37.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a1a938da348fbd2ebc6a7430b11a5e2f460464df0828a59b59d520836ea12c27",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-37.el9.x86_64",
    sha256 = "e022a66324a8961f8f987c78fe2cf860701b022ecdb79a40c1eabbdd43c7ec42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.32-37.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e022a66324a8961f8f987c78fe2cf860701b022ecdb79a40c1eabbdd43c7ec42",
    ],
)

rpm(
    name = "cpp-0__11.5.0-2.el9.aarch64",
    sha256 = "037e69247a7d158ab860cd230a28f83c96a1d53ff6e8b8ff4db95f7a929d786d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/037e69247a7d158ab860cd230a28f83c96a1d53ff6e8b8ff4db95f7a929d786d",
    ],
)

rpm(
    name = "cpp-0__11.5.0-2.el9.s390x",
    sha256 = "3a34c92a8f01110f71ef411209f4eb5dd860eec5cf4826953847f1fbd8d37f5c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/cpp-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3a34c92a8f01110f71ef411209f4eb5dd860eec5cf4826953847f1fbd8d37f5c",
    ],
)

rpm(
    name = "cpp-0__11.5.0-2.el9.x86_64",
    sha256 = "6db7dfd1925305b0a24d19d10116608439ea06e73512d27b03c2057eb3631382",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.5.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6db7dfd1925305b0a24d19d10116608439ea06e73512d27b03c2057eb3631382",
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
    name = "crypto-policies-0__20240828-2.git626aa59.el9.aarch64",
    sha256 = "3479b2aedc8b1bc5d5a0567f7117cf90702012b88fe7956775a4df58a4bcf65c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20240828-2.git626aa59.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3479b2aedc8b1bc5d5a0567f7117cf90702012b88fe7956775a4df58a4bcf65c",
    ],
)

rpm(
    name = "crypto-policies-0__20240828-2.git626aa59.el9.s390x",
    sha256 = "3479b2aedc8b1bc5d5a0567f7117cf90702012b88fe7956775a4df58a4bcf65c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/crypto-policies-20240828-2.git626aa59.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3479b2aedc8b1bc5d5a0567f7117cf90702012b88fe7956775a4df58a4bcf65c",
    ],
)

rpm(
    name = "crypto-policies-0__20240828-2.git626aa59.el9.x86_64",
    sha256 = "3479b2aedc8b1bc5d5a0567f7117cf90702012b88fe7956775a4df58a4bcf65c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20240828-2.git626aa59.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3479b2aedc8b1bc5d5a0567f7117cf90702012b88fe7956775a4df58a4bcf65c",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-31.el9.aarch64",
    sha256 = "7cbda5bca46c13e80bd28391e998b8695e93fb450c40c99ffb52e3b3a74a2ac2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/curl-minimal-7.76.1-31.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7cbda5bca46c13e80bd28391e998b8695e93fb450c40c99ffb52e3b3a74a2ac2",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-31.el9.s390x",
    sha256 = "1f43a0fc561b1055e1302964f64e042f27e5fa8cfc56f368736cf76c39a3ee6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/curl-minimal-7.76.1-31.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1f43a0fc561b1055e1302964f64e042f27e5fa8cfc56f368736cf76c39a3ee6b",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-31.el9.x86_64",
    sha256 = "be145eb1684cb38553b6611bca6c0fb562ff8485902c49131c5ed0b9ac0733f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/curl-minimal-7.76.1-31.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be145eb1684cb38553b6611bca6c0fb562ff8485902c49131c5ed0b9ac0733f4",
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
    name = "daxctl-libs-0__78-2.el9.x86_64",
    sha256 = "1db2937a9c93ecbf3de5bd8da49475156fcf2d082c93008d786b3ce8ece43829",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/daxctl-libs-78-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1db2937a9c93ecbf3de5bd8da49475156fcf2d082c93008d786b3ce8ece43829",
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
    name = "device-mapper-9__1.02.202-1.el9.aarch64",
    sha256 = "2a7e9b94dfe69e84aa32d7157b8d9ddd37d180780b2d9f1bcf413cfcdffa4db0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.202-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2a7e9b94dfe69e84aa32d7157b8d9ddd37d180780b2d9f1bcf413cfcdffa4db0",
    ],
)

rpm(
    name = "device-mapper-9__1.02.202-1.el9.s390x",
    sha256 = "ca1838213c3faacf0f274a86ec34911016709c7c07fb4911c4290a91447836bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-1.02.202-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ca1838213c3faacf0f274a86ec34911016709c7c07fb4911c4290a91447836bd",
    ],
)

rpm(
    name = "device-mapper-9__1.02.202-1.el9.x86_64",
    sha256 = "365a589ea38c3b53d8353c10f0a01fe8aba4d4ec94bdbfb15db4796488a50a90",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.202-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/365a589ea38c3b53d8353c10f0a01fe8aba4d4ec94bdbfb15db4796488a50a90",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.202-1.el9.aarch64",
    sha256 = "af649e8929af4465bd678801f9cd7cb4609a87d26b78f969f1d87db25490af7d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.202-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/af649e8929af4465bd678801f9cd7cb4609a87d26b78f969f1d87db25490af7d",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.202-1.el9.s390x",
    sha256 = "6a591b7e6cc250215a137488ecceeea85048f9cc68e5db80c5f4b8a9f40b703c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-libs-1.02.202-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6a591b7e6cc250215a137488ecceeea85048f9cc68e5db80c5f4b8a9f40b703c",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.202-1.el9.x86_64",
    sha256 = "8e0ab57246c773b4fccdeaea20f407916128c4da6346ce42c5aa4d962ebeb86f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.202-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e0ab57246c773b4fccdeaea20f407916128c4da6346ce42c5aa4d962ebeb86f",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-34.el9.aarch64",
    sha256 = "28504f5e83e12fd88cb45aef83b31d4a37c9732f32ba675ebe031e98dbe01676",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.7-34.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/28504f5e83e12fd88cb45aef83b31d4a37c9732f32ba675ebe031e98dbe01676",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-34.el9.x86_64",
    sha256 = "20c82057c387f04ad995a14ee2106642b5fb291d3a97fb49ebe538b5af41d060",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.7-34.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20c82057c387f04ad995a14ee2106642b5fb291d3a97fb49ebe538b5af41d060",
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
    name = "dmidecode-1__3.6-1.el9.x86_64",
    sha256 = "e06daab6e4f008799ac56a8ff51e51e2333d070bb253fc4506cd106e14657a87",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dmidecode-3.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e06daab6e4f008799ac56a8ff51e51e2333d070bb253fc4506cd106e14657a87",
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
    name = "edk2-aarch64-0__20240524-6.el9.aarch64",
    sha256 = "f168002c5bc3eda85bf49a432a6c0cf3fda00cdcf2619ebafed376174b9c7cd2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20240524-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f168002c5bc3eda85bf49a432a6c0cf3fda00cdcf2619ebafed376174b9c7cd2",
    ],
)

rpm(
    name = "edk2-ovmf-0__20240524-6.el9.x86_64",
    sha256 = "e7ae86321446374996ad9c635f4777098e72340ffc2f2925adf476d55dad1d81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20240524-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e7ae86321446374996ad9c635f4777098e72340ffc2f2925adf476d55dad1d81",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.192-2.el9.aarch64",
    sha256 = "0b9d18b426af693c42a509d500c12ae4ef646e34f84fec2a9e906c74467d7d6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.192-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0b9d18b426af693c42a509d500c12ae4ef646e34f84fec2a9e906c74467d7d6f",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.192-2.el9.s390x",
    sha256 = "58a796427b81861ae9e5727116010e4467523f8e4608c3d3ad31fa55bb83ea4b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.192-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/58a796427b81861ae9e5727116010e4467523f8e4608c3d3ad31fa55bb83ea4b",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.192-2.el9.x86_64",
    sha256 = "5206db8ae4fb3fc0603e1b264ece02785d660f489a6936251aaad50bed8b66b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.192-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5206db8ae4fb3fc0603e1b264ece02785d660f489a6936251aaad50bed8b66b8",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.192-2.el9.aarch64",
    sha256 = "f9802d0cb59395ab14d435df2e481c607912da7803436f16e6dd43d208336c75",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.192-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f9802d0cb59395ab14d435df2e481c607912da7803436f16e6dd43d208336c75",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.192-2.el9.s390x",
    sha256 = "f9802d0cb59395ab14d435df2e481c607912da7803436f16e6dd43d208336c75",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.192-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f9802d0cb59395ab14d435df2e481c607912da7803436f16e6dd43d208336c75",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.192-2.el9.x86_64",
    sha256 = "f9802d0cb59395ab14d435df2e481c607912da7803436f16e6dd43d208336c75",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.192-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f9802d0cb59395ab14d435df2e481c607912da7803436f16e6dd43d208336c75",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.192-2.el9.aarch64",
    sha256 = "811b2165a63e53335cf101f46840734c1e87d028272ac9883d434cfd2006d0fe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.192-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/811b2165a63e53335cf101f46840734c1e87d028272ac9883d434cfd2006d0fe",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.192-2.el9.s390x",
    sha256 = "767546c58380759c77f0b7704384b5d82d72f1a255255ff978562ec074f0b0f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.192-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/767546c58380759c77f0b7704384b5d82d72f1a255255ff978562ec074f0b0f5",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.192-2.el9.x86_64",
    sha256 = "51cf83b63b954c086b93ebe4d0f001b275a562bac01aa4702ce1f8f1ec373a2f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.192-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/51cf83b63b954c086b93ebe4d0f001b275a562bac01aa4702ce1f8f1ec373a2f",
    ],
)

rpm(
    name = "elfutils-libs-0__0.192-2.el9.aarch64",
    sha256 = "b1f67dae4e4faaa057a84bc641e00cd4580137fe5d6ff3d7c9147e4cf4b6fdbe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.192-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b1f67dae4e4faaa057a84bc641e00cd4580137fe5d6ff3d7c9147e4cf4b6fdbe",
    ],
)

rpm(
    name = "elfutils-libs-0__0.192-2.el9.s390x",
    sha256 = "2abfe1c7311b6821a68b057077e46272c9b3949ce647505d95e258f3e5a7c77e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.192-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2abfe1c7311b6821a68b057077e46272c9b3949ce647505d95e258f3e5a7c77e",
    ],
)

rpm(
    name = "elfutils-libs-0__0.192-2.el9.x86_64",
    sha256 = "0c803f9c0613083df5d36d9ffcb7a5faae16e9af5c0e45984ace424c8b247d82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.192-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c803f9c0613083df5d36d9ffcb7a5faae16e9af5c0e45984ace424c8b247d82",
    ],
)

rpm(
    name = "ethtool-2__6.11-1.el9.aarch64",
    sha256 = "25086c6105c5502599a99d3be128aee8cacd181298b342d29d1cdc204de009ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ethtool-6.11-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/25086c6105c5502599a99d3be128aee8cacd181298b342d29d1cdc204de009ce",
    ],
)

rpm(
    name = "ethtool-2__6.11-1.el9.s390x",
    sha256 = "ea7fab5579e130e6a1dd6b486f594e2120ac60df5a0f194e92e859c0cf79e5ab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ethtool-6.11-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ea7fab5579e130e6a1dd6b486f594e2120ac60df5a0f194e92e859c0cf79e5ab",
    ],
)

rpm(
    name = "ethtool-2__6.11-1.el9.x86_64",
    sha256 = "41bfba2ca8a62d6b4bc4dd17bda915af208030c86f4ecc295d88e06d54b0c4ab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ethtool-6.11-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/41bfba2ca8a62d6b4bc4dd17bda915af208030c86f4ecc295d88e06d54b0c4ab",
    ],
)

rpm(
    name = "expat-0__2.5.0-4.el9.aarch64",
    sha256 = "e071ad9e4ac5e4b21adc19304c62b32ac61f0b4dfd17092939eb3eb393f912f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/expat-2.5.0-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e071ad9e4ac5e4b21adc19304c62b32ac61f0b4dfd17092939eb3eb393f912f2",
    ],
)

rpm(
    name = "expat-0__2.5.0-4.el9.s390x",
    sha256 = "4a074438af9dcba19b3a7918d5877a6463a12a42ab0d885c120269b219723ee8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/expat-2.5.0-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4a074438af9dcba19b3a7918d5877a6463a12a42ab0d885c120269b219723ee8",
    ],
)

rpm(
    name = "expat-0__2.5.0-4.el9.x86_64",
    sha256 = "360ed994ea2af5b3a7f37694dfdf2249d97e5e5ec2492c9223a2aec72ff8f480",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.5.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/360ed994ea2af5b3a7f37694dfdf2249d97e5e5ec2492c9223a2aec72ff8f480",
    ],
)

rpm(
    name = "filesystem-0__3.16-5.el9.aarch64",
    sha256 = "c20f1ab9760a8ba5f2d9cb37d7e8fa27f49f91a21a46fe7ad648ff6caf237013",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/filesystem-3.16-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c20f1ab9760a8ba5f2d9cb37d7e8fa27f49f91a21a46fe7ad648ff6caf237013",
    ],
)

rpm(
    name = "filesystem-0__3.16-5.el9.s390x",
    sha256 = "67a733fe124cda9da89f6946757800c0fe73b918a477adcf67dfbef15c995729",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/filesystem-3.16-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/67a733fe124cda9da89f6946757800c0fe73b918a477adcf67dfbef15c995729",
    ],
)

rpm(
    name = "filesystem-0__3.16-5.el9.x86_64",
    sha256 = "da7750fc31248ecc606016391c3f570e1abe7422f812b29a49d830c71884e6dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/filesystem-3.16-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da7750fc31248ecc606016391c3f570e1abe7422f812b29a49d830c71884e6dc",
    ],
)

rpm(
    name = "findutils-1__4.8.0-7.el9.aarch64",
    sha256 = "de9914a265a46cc629f7423ef5f53deefc7044a9c46acb941d9ca0dc6bfc73f8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/findutils-4.8.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de9914a265a46cc629f7423ef5f53deefc7044a9c46acb941d9ca0dc6bfc73f8",
    ],
)

rpm(
    name = "findutils-1__4.8.0-7.el9.s390x",
    sha256 = "627204a8e5a95bde190b1755dacfd72ffe66862438a6e9878d0d0fec90cf5097",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/findutils-4.8.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/627204a8e5a95bde190b1755dacfd72ffe66862438a6e9878d0d0fec90cf5097",
    ],
)

rpm(
    name = "findutils-1__4.8.0-7.el9.x86_64",
    sha256 = "393fc651dddb826521d528d78819515c09b93e551701cafb62b672c2c4701d04",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/findutils-4.8.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/393fc651dddb826521d528d78819515c09b93e551701cafb62b672c2c4701d04",
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
    name = "fuse-0__2.9.9-16.el9.x86_64",
    sha256 = "e83129905e59e8008a74a9f2354dc757b6e40f01a41ea07857755da7989b2b8d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-2.9.9-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e83129905e59e8008a74a9f2354dc757b6e40f01a41ea07857755da7989b2b8d",
    ],
)

rpm(
    name = "fuse-common-0__3.10.2-9.el9.x86_64",
    sha256 = "ad4960b97840017eb3996e150d59a7fe4158da8bb88c178bc2acc08c35772431",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-common-3.10.2-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ad4960b97840017eb3996e150d59a7fe4158da8bb88c178bc2acc08c35772431",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-16.el9.aarch64",
    sha256 = "5bbf1b9e2891fce2fc573ed3f32bb234f796d0eb3893da8fb604c13eeff4a8e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/fuse-libs-2.9.9-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5bbf1b9e2891fce2fc573ed3f32bb234f796d0eb3893da8fb604c13eeff4a8e9",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-16.el9.s390x",
    sha256 = "dec44821c2f1097207ce73554f5b4c3d1a94c8d0ac4fce54bdecfeec9de71f34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/fuse-libs-2.9.9-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/dec44821c2f1097207ce73554f5b4c3d1a94c8d0ac4fce54bdecfeec9de71f34",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-16.el9.x86_64",
    sha256 = "784fa314b778d7dece2ab1bdf30218fdab382d34ec51d6358b8834c866b20ab7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.9-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/784fa314b778d7dece2ab1bdf30218fdab382d34ec51d6358b8834c866b20ab7",
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
    name = "gcc-0__11.5.0-2.el9.aarch64",
    sha256 = "4a7ed3b194085bbf80e03288996b8357c0a339feda1b4a656fdf6fefd889c32a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4a7ed3b194085bbf80e03288996b8357c0a339feda1b4a656fdf6fefd889c32a",
    ],
)

rpm(
    name = "gcc-0__11.5.0-2.el9.s390x",
    sha256 = "e796076ae3a523a9e7ffee1928419ea71e9914608d326704999005aa15bd86ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gcc-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e796076ae3a523a9e7ffee1928419ea71e9914608d326704999005aa15bd86ed",
    ],
)

rpm(
    name = "gcc-0__11.5.0-2.el9.x86_64",
    sha256 = "f0b8a7e5796f28b8fdae2d31f86521f24b0e45b228f765fea6f7bcde157fef41",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.5.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f0b8a7e5796f28b8fdae2d31f86521f24b0e45b228f765fea6f7bcde157fef41",
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
    name = "glib2-0__2.68.4-16.el9.aarch64",
    sha256 = "6d47f73da8f765a536e2647b611017afc13ea5da440efcd9d8d92820e51320b9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glib2-2.68.4-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6d47f73da8f765a536e2647b611017afc13ea5da440efcd9d8d92820e51320b9",
    ],
)

rpm(
    name = "glib2-0__2.68.4-16.el9.s390x",
    sha256 = "4199c2ee05b0e4338d43903665b7f5f02bc04d3fbdf8a5cdcc33ee7ca2ef5d11",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glib2-2.68.4-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4199c2ee05b0e4338d43903665b7f5f02bc04d3fbdf8a5cdcc33ee7ca2ef5d11",
    ],
)

rpm(
    name = "glib2-0__2.68.4-16.el9.x86_64",
    sha256 = "793cbb8b6f5885a3b8a501dd5e4c0fe19141c34beeb4410fbc680424ae02ed2d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/793cbb8b6f5885a3b8a501dd5e4c0fe19141c34beeb4410fbc680424ae02ed2d",
    ],
)

rpm(
    name = "glibc-0__2.34-140.el9.aarch64",
    sha256 = "4448e09f56ab4d0769d19c2fb450751f50521fb7b47932e8b0712a71f0696b4c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-140.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4448e09f56ab4d0769d19c2fb450751f50521fb7b47932e8b0712a71f0696b4c",
    ],
)

rpm(
    name = "glibc-0__2.34-140.el9.s390x",
    sha256 = "ad5d37f6a537603ce90f1a3942dfd681097d7da47315794281045bb37b88dad8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-2.34-140.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ad5d37f6a537603ce90f1a3942dfd681097d7da47315794281045bb37b88dad8",
    ],
)

rpm(
    name = "glibc-0__2.34-140.el9.x86_64",
    sha256 = "1cda18c8251d182c2c1fcb84edcf2dbca244d273f0545ca565cef4b745ef21b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-140.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cda18c8251d182c2c1fcb84edcf2dbca244d273f0545ca565cef4b745ef21b1",
    ],
)

rpm(
    name = "glibc-common-0__2.34-140.el9.aarch64",
    sha256 = "5deadffce9dda02768a8c931bacab476fb75717e3c0f5c2ab8f7d06b95e6824d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-140.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5deadffce9dda02768a8c931bacab476fb75717e3c0f5c2ab8f7d06b95e6824d",
    ],
)

rpm(
    name = "glibc-common-0__2.34-140.el9.s390x",
    sha256 = "7c0df95b55d18b92f96b427b23867cf719d87a816ec123b587f2adea8e1226a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-common-2.34-140.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7c0df95b55d18b92f96b427b23867cf719d87a816ec123b587f2adea8e1226a4",
    ],
)

rpm(
    name = "glibc-common-0__2.34-140.el9.x86_64",
    sha256 = "b52a961176481f95f4c14eace946de8e69cb57b49e491bb45df3f4edac053843",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-140.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b52a961176481f95f4c14eace946de8e69cb57b49e491bb45df3f4edac053843",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-140.el9.aarch64",
    sha256 = "007dfdd600da4cef01a8b8bfe90618d1509514c8162eb08d89e3d34e91aadc23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-140.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/007dfdd600da4cef01a8b8bfe90618d1509514c8162eb08d89e3d34e91aadc23",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-140.el9.s390x",
    sha256 = "31fbe6eb7a4c6c9a42d88612bedf16e88648c988a413d6e05be5b1f06038e4db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-devel-2.34-140.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/31fbe6eb7a4c6c9a42d88612bedf16e88648c988a413d6e05be5b1f06038e4db",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-140.el9.x86_64",
    sha256 = "52af7a597756594caf174fc5742a157aa7c08402a5442aa7400fe2e72a6cf1cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-140.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/52af7a597756594caf174fc5742a157aa7c08402a5442aa7400fe2e72a6cf1cb",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-140.el9.s390x",
    sha256 = "6ca58c754b4fadc41988830d3f10dd5b2f0875c3f1571e448074fc0d6944d2b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-headers-2.34-140.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6ca58c754b4fadc41988830d3f10dd5b2f0875c3f1571e448074fc0d6944d2b0",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-140.el9.x86_64",
    sha256 = "1edb5e89520881d39aae2a87591a9268f277d6748f5c49634d8d73efa8dc216a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-140.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1edb5e89520881d39aae2a87591a9268f277d6748f5c49634d8d73efa8dc216a",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-140.el9.aarch64",
    sha256 = "e02ad5c9690d49c1a17750d2ca90d80ddf91b653c21384ebb3b4010971a53128",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-140.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e02ad5c9690d49c1a17750d2ca90d80ddf91b653c21384ebb3b4010971a53128",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-140.el9.s390x",
    sha256 = "308c718f508333c6291925e14a72cad9b62f7e92d0c39263d147bd7616eb37a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.34-140.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/308c718f508333c6291925e14a72cad9b62f7e92d0c39263d147bd7616eb37a1",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-140.el9.x86_64",
    sha256 = "16323939cbda9b1d756927154dd1c16c7d47451301c007dd62e0be25bcf0c638",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-140.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/16323939cbda9b1d756927154dd1c16c7d47451301c007dd62e0be25bcf0c638",
    ],
)

rpm(
    name = "glibc-static-0__2.34-140.el9.aarch64",
    sha256 = "2b8472ba666fb63cc46a9a4c079b0150be699fd8c61cd13cc4395209be3c53ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-140.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2b8472ba666fb63cc46a9a4c079b0150be699fd8c61cd13cc4395209be3c53ff",
    ],
)

rpm(
    name = "glibc-static-0__2.34-140.el9.s390x",
    sha256 = "5ce8b6f627aec6e66d2c5e9e68efd81556e184997f8c013b0acd93481ce750b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/glibc-static-2.34-140.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5ce8b6f627aec6e66d2c5e9e68efd81556e184997f8c013b0acd93481ce750b0",
    ],
)

rpm(
    name = "glibc-static-0__2.34-140.el9.x86_64",
    sha256 = "ed886db3512e6b0560faf51e00f41e3d95f8f6fd441d32ea45b1326186d0fc25",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-140.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed886db3512e6b0560faf51e00f41e3d95f8f6fd441d32ea45b1326186d0fc25",
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
    name = "gssproxy-0__0.8.4-7.el9.x86_64",
    sha256 = "45b8f4390ad72f21a7976b40ee132b6b80fcc81954b0a59a76b0c08494dcccf0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gssproxy-0.8.4-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/45b8f4390ad72f21a7976b40ee132b6b80fcc81954b0a59a76b0c08494dcccf0",
    ],
)

rpm(
    name = "guestfs-tools-0__1.51.6-5.el9.x86_64",
    sha256 = "992ac23f4f30f229cbb2d5cdbd63e44ee94069a33c6b753aa633ac2e71eeaaa7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.51.6-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/992ac23f4f30f229cbb2d5cdbd63e44ee94069a33c6b753aa633ac2e71eeaaa7",
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
    name = "hivex-libs-0__1.3.24-1.el9.x86_64",
    sha256 = "f757c1720320e62ebc874dd169dea4540f145d5a0132afb4263c640cae87af46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/hivex-libs-1.3.24-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f757c1720320e62ebc874dd169dea4540f145d5a0132afb4263c640cae87af46",
    ],
)

rpm(
    name = "hwdata-0__0.348-9.15.el9.x86_64",
    sha256 = "e0a8c0a065c32a433925a0ee89c74837f49505002fda4059193d3edf5c83e19a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.15.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e0a8c0a065c32a433925a0ee89c74837f49505002fda4059193d3edf5c83e19a",
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
    name = "iptables-libs-0__1.8.10-5.el9.aarch64",
    sha256 = "0a1a62e87beefb172561f8c399ffd227a2200d9e75da6ee34e573e5535b21782",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.10-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0a1a62e87beefb172561f8c399ffd227a2200d9e75da6ee34e573e5535b21782",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-5.el9.s390x",
    sha256 = "f70904187dba41332b65d6678d680ed6761c811c2c620662d6ee185716951f93",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iptables-libs-1.8.10-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f70904187dba41332b65d6678d680ed6761c811c2c620662d6ee185716951f93",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-5.el9.x86_64",
    sha256 = "36823d15bd684acf2df31039914c186cb513bf2b9ad08603d6890ce785b96661",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.10-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/36823d15bd684acf2df31039914c186cb513bf2b9ad08603d6890ce785b96661",
    ],
)

rpm(
    name = "iputils-0__20210202-11.el9.aarch64",
    sha256 = "6539781b8a4ca6dd0c55c8b33b6f86868a1ec61f4b0b80079ab79b9a318b6068",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iputils-20210202-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6539781b8a4ca6dd0c55c8b33b6f86868a1ec61f4b0b80079ab79b9a318b6068",
    ],
)

rpm(
    name = "iputils-0__20210202-11.el9.s390x",
    sha256 = "8fd05a83334e0167429c9cabcbd90415e2c3ace5dd0ab8cc4b2f8a938657c39f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iputils-20210202-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8fd05a83334e0167429c9cabcbd90415e2c3ace5dd0ab8cc4b2f8a938657c39f",
    ],
)

rpm(
    name = "iputils-0__20210202-11.el9.x86_64",
    sha256 = "c71055f2a1a3bdb732fc5c05eea7c5ee1cba3dc72f884b1dd728b18fc730a87e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iputils-20210202-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c71055f2a1a3bdb732fc5c05eea7c5ee1cba3dc72f884b1dd728b18fc730a87e",
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
    name = "json-c-0__0.14-11.el9.aarch64",
    sha256 = "65a68a23f33540b4d7cd2d9227a63d7eda1a7ab7cdd52457fee9662c06731cfa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/json-c-0.14-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/65a68a23f33540b4d7cd2d9227a63d7eda1a7ab7cdd52457fee9662c06731cfa",
    ],
)

rpm(
    name = "json-c-0__0.14-11.el9.s390x",
    sha256 = "224d820ba796088e5742a550fe7add8accf6bae309f154b4589bc11628edbcc4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/json-c-0.14-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/224d820ba796088e5742a550fe7add8accf6bae309f154b4589bc11628edbcc4",
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
    name = "kernel-headers-0__5.14.0-529.el9.aarch64",
    sha256 = "b9b7c4fa40d69c5e44bc8949fc7f53934dd3911a3845388a2a2bbb27e0368a2d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/kernel-headers-5.14.0-529.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b9b7c4fa40d69c5e44bc8949fc7f53934dd3911a3845388a2a2bbb27e0368a2d",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-529.el9.s390x",
    sha256 = "31a94cb5078168be041421cc2eb5e56cccfd0df4843742b3b192961281372e19",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/kernel-headers-5.14.0-529.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/31a94cb5078168be041421cc2eb5e56cccfd0df4843742b3b192961281372e19",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-529.el9.x86_64",
    sha256 = "7f0b73fb1fb30d9c296080fbefcb8bab3436f65249dca700f39524f03664dfb9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/kernel-headers-5.14.0-529.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f0b73fb1fb30d9c296080fbefcb8bab3436f65249dca700f39524f03664dfb9",
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
    name = "kmod-0__28-10.el9.x86_64",
    sha256 = "a06c4530e21608365adad811f10deb11ed53a75f7c7f415e448d5afedc2592b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-28-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a06c4530e21608365adad811f10deb11ed53a75f7c7f415e448d5afedc2592b0",
    ],
)

rpm(
    name = "kmod-libs-0__28-10.el9.aarch64",
    sha256 = "5da40af25f9af3e6ce1ff8dd751da596073dd0adf15dcf44c393330ff0346355",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/kmod-libs-28-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5da40af25f9af3e6ce1ff8dd751da596073dd0adf15dcf44c393330ff0346355",
    ],
)

rpm(
    name = "kmod-libs-0__28-10.el9.s390x",
    sha256 = "7011810fca95064c8d78e55071716ec1dd5bc7b9836f662c195a282f4f4e5d0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/kmod-libs-28-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7011810fca95064c8d78e55071716ec1dd5bc7b9836f662c195a282f4f4e5d0a",
    ],
)

rpm(
    name = "kmod-libs-0__28-10.el9.x86_64",
    sha256 = "79deb68a50b02b69df260fdb6e5c29f1b992290968ac6b07e7b249b2bdbc8ced",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-libs-28-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/79deb68a50b02b69df260fdb6e5c29f1b992290968ac6b07e7b249b2bdbc8ced",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-4.el9.aarch64",
    sha256 = "ec9f42b46e94ac39c2aea842f8d72d0748509022aba8306125d25658a610699c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.1-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ec9f42b46e94ac39c2aea842f8d72d0748509022aba8306125d25658a610699c",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-4.el9.s390x",
    sha256 = "fea8a1c82acad5a706dfc66e2ab324c9f51d4d1bb4d95b8590240f5063c2cd3b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/krb5-libs-1.21.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fea8a1c82acad5a706dfc66e2ab324c9f51d4d1bb4d95b8590240f5063c2cd3b",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-4.el9.x86_64",
    sha256 = "cf5acbb17ccf4c77f9283360c29d04cccffa8e18f3fc66a23a12742a2dfdcb73",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cf5acbb17ccf4c77f9283360c29d04cccffa8e18f3fc66a23a12742a2dfdcb73",
    ],
)

rpm(
    name = "less-0__590-5.el9.x86_64",
    sha256 = "46e11dfacb75a8d03047d82f44ae46b11d95da31e0ec1b3a8cc37a132b1c7cae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/less-590-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46e11dfacb75a8d03047d82f44ae46b11d95da31e0ec1b3a8cc37a132b1c7cae",
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
    name = "libasan-0__11.5.0-2.el9.aarch64",
    sha256 = "bad0d39fdc7baa33ab31e819a65b047cdd36ecd7a020722295bdc7b33c629f4a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bad0d39fdc7baa33ab31e819a65b047cdd36ecd7a020722295bdc7b33c629f4a",
    ],
)

rpm(
    name = "libasan-0__11.5.0-2.el9.s390x",
    sha256 = "777128e90cd1942bb71075174a3f4a7736022e8b8fc5b4c5b99476a3a19cf4cf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libasan-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/777128e90cd1942bb71075174a3f4a7736022e8b8fc5b4c5b99476a3a19cf4cf",
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
    name = "libatomic-0__11.5.0-2.el9.aarch64",
    sha256 = "4d7e0f0b57511419d1680b979a9ef4ff3ff2a0eea9003992b03ff691b203a479",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4d7e0f0b57511419d1680b979a9ef4ff3ff2a0eea9003992b03ff691b203a479",
    ],
)

rpm(
    name = "libatomic-0__11.5.0-2.el9.s390x",
    sha256 = "d222f085da399a443cc87985ffcb09fa679689a1f61de3f41652e430e4bbeaed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libatomic-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d222f085da399a443cc87985ffcb09fa679689a1f61de3f41652e430e4bbeaed",
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
    name = "libblkid-0__2.37.4-20.el9.aarch64",
    sha256 = "cebd26c399911e618eb2fa326cd0fd09ac8eb11884e9e4835aec01af79e18105",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libblkid-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cebd26c399911e618eb2fa326cd0fd09ac8eb11884e9e4835aec01af79e18105",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-20.el9.s390x",
    sha256 = "25e49a656a3eba08ef3041b90f18da2abfbc55f6e67257c192ccde9f4009cb56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libblkid-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/25e49a656a3eba08ef3041b90f18da2abfbc55f6e67257c192ccde9f4009cb56",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-20.el9.x86_64",
    sha256 = "5fa87671fdc5bb3e4e6c2b8e2253ac8fcf4add8ce44bf216864f952f10cdeeaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libblkid-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5fa87671fdc5bb3e4e6c2b8e2253ac8fcf4add8ce44bf216864f952f10cdeeaa",
    ],
)

rpm(
    name = "libbpf-2__1.4.0-1.el9.aarch64",
    sha256 = "693f1ea0b46ede7bce112562e58fc33532af307ff217baf5b280d2949a78ddde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libbpf-1.4.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/693f1ea0b46ede7bce112562e58fc33532af307ff217baf5b280d2949a78ddde",
    ],
)

rpm(
    name = "libbpf-2__1.4.0-1.el9.s390x",
    sha256 = "01c9fb8c866e82ab0a6067e2a0034abcc577c06604a4160ffd46c51bece58c6d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libbpf-1.4.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/01c9fb8c866e82ab0a6067e2a0034abcc577c06604a4160ffd46c51bece58c6d",
    ],
)

rpm(
    name = "libbpf-2__1.4.0-1.el9.x86_64",
    sha256 = "ea0d65618dba3830d6044bf5441a0013302a6ee9ee85d8292a7a7a5094c6c851",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-1.4.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ea0d65618dba3830d6044bf5441a0013302a6ee9ee85d8292a7a7a5094c6c851",
    ],
)

rpm(
    name = "libbrotli-0__1.0.9-7.el9.x86_64",
    sha256 = "5eb0c43339cf40cc8b668c9f2803b80aff8f149798002660947edf8d5a75de1a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbrotli-1.0.9-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5eb0c43339cf40cc8b668c9f2803b80aff8f149798002660947edf8d5a75de1a",
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
    name = "libcurl-minimal-0__7.76.1-31.el9.aarch64",
    sha256 = "9c0ec87af11f82ac5a2a4e6be45617b80737435a89c2be6a90a0e4b380e63053",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.76.1-31.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9c0ec87af11f82ac5a2a4e6be45617b80737435a89c2be6a90a0e4b380e63053",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-31.el9.s390x",
    sha256 = "ece81fe8aa2bfd5ff0c98cfdafe110a5e023184101ace9196d38a49665639b6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcurl-minimal-7.76.1-31.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ece81fe8aa2bfd5ff0c98cfdafe110a5e023184101ace9196d38a49665639b6f",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-31.el9.x86_64",
    sha256 = "6438485e38465ee944e25abedcf4a1761564fe5202f05a02c71e4c880255b539",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-31.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6438485e38465ee944e25abedcf4a1761564fe5202f05a02c71e4c880255b539",
    ],
)

rpm(
    name = "libdb-0__5.3.28-55.el9.aarch64",
    sha256 = "e8d47189a01859ac933f767d52c2f4042b884f78e896ed6f42e45db23c4579df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libdb-5.3.28-55.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e8d47189a01859ac933f767d52c2f4042b884f78e896ed6f42e45db23c4579df",
    ],
)

rpm(
    name = "libdb-0__5.3.28-55.el9.s390x",
    sha256 = "c69d2091b590fb864f51cb78709bd26004d8afd0322d7c202ca70e340c084606",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libdb-5.3.28-55.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c69d2091b590fb864f51cb78709bd26004d8afd0322d7c202ca70e340c084606",
    ],
)

rpm(
    name = "libdb-0__5.3.28-55.el9.x86_64",
    sha256 = "e28608db5eaa3ee38e8bc0d6be1831048da1e638920a6f16a8084e72e2ebf6c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libdb-5.3.28-55.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e28608db5eaa3ee38e8bc0d6be1831048da1e638920a6f16a8084e72e2ebf6c9",
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
    name = "libev-0__4.33-6.el9.x86_64",
    sha256 = "e41a990a750f3798e4f4ff5a2a1046586e5979215c7e5a805f3a7be387e9edc1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libev-4.33-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e41a990a750f3798e4f4ff5a2a1046586e5979215c7e5a805f3a7be387e9edc1",
    ],
)

rpm(
    name = "libevent-0__2.1.12-8.el9.aarch64",
    sha256 = "abea343484ceb42612ce394cf7cf0a191ae7d6ea93391fa32721ff7e04b0bb28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libevent-2.1.12-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/abea343484ceb42612ce394cf7cf0a191ae7d6ea93391fa32721ff7e04b0bb28",
    ],
)

rpm(
    name = "libevent-0__2.1.12-8.el9.s390x",
    sha256 = "5c1bdffe7f5dfc8175e2b06acbb4154b272205c40d3c19b88a0d1fde095728b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libevent-2.1.12-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c1bdffe7f5dfc8175e2b06acbb4154b272205c40d3c19b88a0d1fde095728b0",
    ],
)

rpm(
    name = "libevent-0__2.1.12-8.el9.x86_64",
    sha256 = "5683f51c9b02d5f4a3324dc6dacb3a84f0c3710cdc46fa7f04df64b60d38a62b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libevent-2.1.12-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5683f51c9b02d5f4a3324dc6dacb3a84f0c3710cdc46fa7f04df64b60d38a62b",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-20.el9.aarch64",
    sha256 = "c61bf4906bdd46399d50b453b557533060c5a3c344ac1bb0a9bb94ce41246e6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libfdisk-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c61bf4906bdd46399d50b453b557533060c5a3c344ac1bb0a9bb94ce41246e6f",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-20.el9.s390x",
    sha256 = "bf3c3200f0a1e1b1b2fcd0e53b65226d562aee9762cabedd2471bdf2a402b454",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libfdisk-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bf3c3200f0a1e1b1b2fcd0e53b65226d562aee9762cabedd2471bdf2a402b454",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-20.el9.x86_64",
    sha256 = "d1fcceb55185b4d898c8df3d0b9177126be0144b8829f908f40d2b58d44ad268",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d1fcceb55185b4d898c8df3d0b9177126be0144b8829f908f40d2b58d44ad268",
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
    name = "libgcc-0__11.5.0-2.el9.aarch64",
    sha256 = "f668e90e60502b349c33996abff84694c407c87e004b74df020f07ad030b846d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f668e90e60502b349c33996abff84694c407c87e004b74df020f07ad030b846d",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-2.el9.s390x",
    sha256 = "ac6c003d9fe74072a7b3b34fc33fe649b5ec98cb0d8f08efb5239002cbd578c8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcc-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ac6c003d9fe74072a7b3b34fc33fe649b5ec98cb0d8f08efb5239002cbd578c8",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-2.el9.x86_64",
    sha256 = "ff344c9aaf0ef773230411b64e58d35d372314641b69113229afa6c539aa270a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.5.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff344c9aaf0ef773230411b64e58d35d372314641b69113229afa6c539aa270a",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-11.el9.aarch64",
    sha256 = "932bfe51b207e2ad8a0bd2b89e2fb33df73f3993586aaa4cc60576f57795e4db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.10.0-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/932bfe51b207e2ad8a0bd2b89e2fb33df73f3993586aaa4cc60576f57795e4db",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-11.el9.s390x",
    sha256 = "cf30c86fc1a18f504d639d3cbcf9e431af1ea639e6a5e7db1f6d30b763dd51a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcrypt-1.10.0-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cf30c86fc1a18f504d639d3cbcf9e431af1ea639e6a5e7db1f6d30b763dd51a8",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-11.el9.x86_64",
    sha256 = "0323a74a5ad27bc3dc4ac4e9565825f37dc58b2a4800adbf33f767fa7a267c35",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.10.0-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0323a74a5ad27bc3dc4ac4e9565825f37dc58b2a4800adbf33f767fa7a267c35",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-2.el9.aarch64",
    sha256 = "2d6035aa3dceeefda811195fb49a0868c649737e582120f935d524819030c237",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2d6035aa3dceeefda811195fb49a0868c649737e582120f935d524819030c237",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-2.el9.s390x",
    sha256 = "0231d314b3c28c536c17c66b35e82c312fca7a1e59375c4c026d1848bb9ea905",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgomp-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0231d314b3c28c536c17c66b35e82c312fca7a1e59375c4c026d1848bb9ea905",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-2.el9.x86_64",
    sha256 = "924750386c78b20adedc26687e109426029e628c533009ef7b28af6d5f64e50a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.5.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/924750386c78b20adedc26687e109426029e628c533009ef7b28af6d5f64e50a",
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
    name = "libguestfs-1__1.50.2-1.el9.x86_64",
    sha256 = "d2f966b0e56b21150839c864f4669056d85a2f56af711f3c214fe031427ecc26",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libguestfs-1.50.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d2f966b0e56b21150839c864f4669056d85a2f56af711f3c214fe031427ecc26",
    ],
)

rpm(
    name = "libibverbs-0__54.0-1.el9.aarch64",
    sha256 = "32c421ec7c2d0abd2c17f9f9f07e48d39a8e886c43d9ef82f8692c6e68ad1ca5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libibverbs-54.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/32c421ec7c2d0abd2c17f9f9f07e48d39a8e886c43d9ef82f8692c6e68ad1ca5",
    ],
)

rpm(
    name = "libibverbs-0__54.0-1.el9.s390x",
    sha256 = "a539e21965c8b8d05c0af0cd9615a3486476d2c2169251ae5e27c32b840ed45d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libibverbs-54.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a539e21965c8b8d05c0af0cd9615a3486476d2c2169251ae5e27c32b840ed45d",
    ],
)

rpm(
    name = "libibverbs-0__54.0-1.el9.x86_64",
    sha256 = "b57effbc14e02e546a6e94bf8247f2dbfe24b5da0b95691ba3979b3652002b77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libibverbs-54.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b57effbc14e02e546a6e94bf8247f2dbfe24b5da0b95691ba3979b3652002b77",
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
    name = "libksba-0__1.5.1-7.el9.x86_64",
    sha256 = "8c2a4312f0a700286e1c3630f62dba6d06e7a4c07a17182ca97f2d40d0b4c6a0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libksba-1.5.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c2a4312f0a700286e1c3630f62dba6d06e7a4c07a17182ca97f2d40d0b4c6a0",
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
    name = "libmount-0__2.37.4-20.el9.aarch64",
    sha256 = "84f9ee04bb2f3957e927dceaa9c36b3d3e009892b08741e1b45817b6eb6ca30c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmount-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/84f9ee04bb2f3957e927dceaa9c36b3d3e009892b08741e1b45817b6eb6ca30c",
    ],
)

rpm(
    name = "libmount-0__2.37.4-20.el9.s390x",
    sha256 = "a917e4342e7934d4a6d361734e69e42694e59bca82d617305bd8f6aed9c2d7d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libmount-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a917e4342e7934d4a6d361734e69e42694e59bca82d617305bd8f6aed9c2d7d4",
    ],
)

rpm(
    name = "libmount-0__2.37.4-20.el9.x86_64",
    sha256 = "f602bea553bf92e512a39af33c3e8ee289dd9584e37d2ca02b69cb51b64dc623",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmount-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f602bea553bf92e512a39af33c3e8ee289dd9584e37d2ca02b69cb51b64dc623",
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
    name = "libnbd-0__1.20.3-1.el9.aarch64",
    sha256 = "ffcb07a14e2435c3ae087a62072c620345e1d2d25d64ff50f1123efc488deb81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libnbd-1.20.3-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ffcb07a14e2435c3ae087a62072c620345e1d2d25d64ff50f1123efc488deb81",
    ],
)

rpm(
    name = "libnbd-0__1.20.3-1.el9.s390x",
    sha256 = "6382945507c5912a627b65cd41dadc912295cbfcb737fe39fd07f42f39342a6a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libnbd-1.20.3-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6382945507c5912a627b65cd41dadc912295cbfcb737fe39fd07f42f39342a6a",
    ],
)

rpm(
    name = "libnbd-0__1.20.3-1.el9.x86_64",
    sha256 = "4d39fdb30ac2f05b0cb67e296f0fd553b7fe2092ba8b9f3940f2d90e0146a835",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libnbd-1.20.3-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d39fdb30ac2f05b0cb67e296f0fd553b7fe2092ba8b9f3940f2d90e0146a835",
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
    name = "libnfsidmap-1__2.5.4-27.el9.x86_64",
    sha256 = "1b98f68d8817699137925685afe277d375112b18a8e3a44c25538b639169b3d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfsidmap-2.5.4-27.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1b98f68d8817699137925685afe277d375112b18a8e3a44c25538b639169b3d9",
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
    name = "libnl3-0__3.11.0-1.el9.aarch64",
    sha256 = "931603d3bd38323504f5650a51eb18e8f0ff042a8e9d55deaa55d9ed8c1b0371",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnl3-3.11.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/931603d3bd38323504f5650a51eb18e8f0ff042a8e9d55deaa55d9ed8c1b0371",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el9.s390x",
    sha256 = "af754f7cc0670de1449a2a2a5ef353aa21187593f7f7e48389fb3a43724903cc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnl3-3.11.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/af754f7cc0670de1449a2a2a5ef353aa21187593f7f7e48389fb3a43724903cc",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el9.x86_64",
    sha256 = "8988a2e97b63bfe07568a1a85fa8ca9fe6a1b940320f6f72e63d908c54b78a2a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnl3-3.11.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8988a2e97b63bfe07568a1a85fa8ca9fe6a1b940320f6f72e63d908c54b78a2a",
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
    name = "librdmacm-0__54.0-1.el9.aarch64",
    sha256 = "d2bd7ddf591482c48dfd4db3590dbdaa33dcd6d5b6ca36666efea6aab58169d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/librdmacm-54.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d2bd7ddf591482c48dfd4db3590dbdaa33dcd6d5b6ca36666efea6aab58169d8",
    ],
)

rpm(
    name = "librdmacm-0__54.0-1.el9.x86_64",
    sha256 = "82d2d2eecace0a17f97e44e42d766a0ef5cf67f5c42e139c58e18406dfc38f4d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/librdmacm-54.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82d2d2eecace0a17f97e44e42d766a0ef5cf67f5c42e139c58e18406dfc38f4d",
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
    name = "libselinux-0__3.6-2.el9.aarch64",
    sha256 = "a3286f9e68923cc7acf33297b90cf39b4ead485f044cc97b0d1dc8daa9aed086",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-3.6-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a3286f9e68923cc7acf33297b90cf39b4ead485f044cc97b0d1dc8daa9aed086",
    ],
)

rpm(
    name = "libselinux-0__3.6-2.el9.s390x",
    sha256 = "c9db29eceb5f4c5aae0e823ebe99729512434260b71426bc6ccdc1177d0958d5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-3.6-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c9db29eceb5f4c5aae0e823ebe99729512434260b71426bc6ccdc1177d0958d5",
    ],
)

rpm(
    name = "libselinux-0__3.6-2.el9.x86_64",
    sha256 = "25730cb1b020298f50c681249479b418edd54fb68732e765012ab90e67b77479",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-3.6-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/25730cb1b020298f50c681249479b418edd54fb68732e765012ab90e67b77479",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-2.el9.aarch64",
    sha256 = "84d2614f351ad674d64fed4600bcbf4129ebfe2b098a64e1f9772f3daf0af32d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.6-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/84d2614f351ad674d64fed4600bcbf4129ebfe2b098a64e1f9772f3daf0af32d",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-2.el9.s390x",
    sha256 = "a32d36fcff35315c74192d7b0c8410f81c8d8e6ff698009180704039b932286f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-utils-3.6-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a32d36fcff35315c74192d7b0c8410f81c8d8e6ff698009180704039b932286f",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-2.el9.x86_64",
    sha256 = "f7bd1cd6202c47cb1a7299d8de08199ec991f07a21560446de06d1d6a7cb1615",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.6-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7bd1cd6202c47cb1a7299d8de08199ec991f07a21560446de06d1d6a7cb1615",
    ],
)

rpm(
    name = "libsemanage-0__3.6-3.el9.aarch64",
    sha256 = "45b1840615672dd6d811b4dab1213da5cd96cacc55f0d9ff4f6ec026d2be4d4c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsemanage-3.6-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/45b1840615672dd6d811b4dab1213da5cd96cacc55f0d9ff4f6ec026d2be4d4c",
    ],
)

rpm(
    name = "libsemanage-0__3.6-3.el9.s390x",
    sha256 = "deac76881bc0c223aa8488a114620d31a87d1cde5b9531511b6f11517e48c0ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsemanage-3.6-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/deac76881bc0c223aa8488a114620d31a87d1cde5b9531511b6f11517e48c0ca",
    ],
)

rpm(
    name = "libsemanage-0__3.6-3.el9.x86_64",
    sha256 = "90eb1d419b8de11a092935ff242d51e4b8a2fe26b905ebe9e5cfd838d0c973b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsemanage-3.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/90eb1d419b8de11a092935ff242d51e4b8a2fe26b905ebe9e5cfd838d0c973b3",
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
    name = "libslirp-0__4.4.0-8.el9.aarch64",
    sha256 = "52a73957cdbce4484adc9755e42393aeb31443e199fbcdcf3ae867dee82145bf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libslirp-4.4.0-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/52a73957cdbce4484adc9755e42393aeb31443e199fbcdcf3ae867dee82145bf",
    ],
)

rpm(
    name = "libslirp-0__4.4.0-8.el9.s390x",
    sha256 = "d47be3b8520589ff857b0264075f98b0483863762a0d3b0ebb1fba7c870edba6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libslirp-4.4.0-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d47be3b8520589ff857b0264075f98b0483863762a0d3b0ebb1fba7c870edba6",
    ],
)

rpm(
    name = "libslirp-0__4.4.0-8.el9.x86_64",
    sha256 = "aa5c4568ef12b3324e28e2353a97e5d531892e9e0682a035a5669819c7fd6dc3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libslirp-4.4.0-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aa5c4568ef12b3324e28e2353a97e5d531892e9e0682a035a5669819c7fd6dc3",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-20.el9.aarch64",
    sha256 = "e81543e1ac16943bf49fb9a74526ffa6f0cee41e902f93282b9d8787154ba08b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e81543e1ac16943bf49fb9a74526ffa6f0cee41e902f93282b9d8787154ba08b",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-20.el9.s390x",
    sha256 = "afc481221d6f3adc1727289ca543ee40bb410a9c564fba75d356c8a51131ece0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsmartcols-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/afc481221d6f3adc1727289ca543ee40bb410a9c564fba75d356c8a51131ece0",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-20.el9.x86_64",
    sha256 = "e51f3a4fac42fe95d4a7fb1128afd99d9cb7cfdb6ab2ec5e68089bbb72af13ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e51f3a4fac42fe95d4a7fb1128afd99d9cb7cfdb6ab2ec5e68089bbb72af13ca",
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
    name = "libsss_idmap-0__2.9.5-5.el9.aarch64",
    sha256 = "27f994175fb7e57e23d1320c04707ed8d39460c75f9b31c980e168ba31597306",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.9.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/27f994175fb7e57e23d1320c04707ed8d39460c75f9b31c980e168ba31597306",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.5-5.el9.s390x",
    sha256 = "f0226392b662902300b56c338f615526aa42ef4709948b59e0fba3a7eae11a34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.9.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f0226392b662902300b56c338f615526aa42ef4709948b59e0fba3a7eae11a34",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.5-5.el9.x86_64",
    sha256 = "15bca8f2fd97454cbc88763428a8d882d3cbef3b4f01d5d6173231be97bf5e20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.9.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/15bca8f2fd97454cbc88763428a8d882d3cbef3b4f01d5d6173231be97bf5e20",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.5-5.el9.aarch64",
    sha256 = "46823b9c43058e266d7fcee53b793234a04033ddeba284e5c7e1535e20bf7e38",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.9.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/46823b9c43058e266d7fcee53b793234a04033ddeba284e5c7e1535e20bf7e38",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.5-5.el9.s390x",
    sha256 = "1422da4af0892eb13c63bfdea9d9a40ff25c5716de6ca6cb91d34cf107f6bd59",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.9.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1422da4af0892eb13c63bfdea9d9a40ff25c5716de6ca6cb91d34cf107f6bd59",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.5-5.el9.x86_64",
    sha256 = "7e364aec77c0a3228b06c0195ccd0a34a1495250892a209f8f7f1f18d2df4dc2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.9.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e364aec77c0a3228b06c0195ccd0a34a1495250892a209f8f7f1f18d2df4dc2",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-2.el9.aarch64",
    sha256 = "fff7f00d26008ab09b566c3b14d446b4b0b3df08bedbeee29142d62278568c82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fff7f00d26008ab09b566c3b14d446b4b0b3df08bedbeee29142d62278568c82",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-2.el9.s390x",
    sha256 = "3644bcebe706602976b4d1596eedefcb0af0cfdb74141ca9084e0b34e8d22890",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libstdc++-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3644bcebe706602976b4d1596eedefcb0af0cfdb74141ca9084e0b34e8d22890",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-2.el9.x86_64",
    sha256 = "dcd7090c2a37f13b2d4a1a2bc2d1fedc514c745efc4f2619783bbd1979b5e82f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.5.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dcd7090c2a37f13b2d4a1a2bc2d1fedc514c745efc4f2619783bbd1979b5e82f",
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
    name = "libtirpc-0__1.3.3-9.el9.aarch64",
    sha256 = "a5e098dea257c3a423f46377624d5317c9484709ad293292b415574312988780",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtirpc-1.3.3-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a5e098dea257c3a423f46377624d5317c9484709ad293292b415574312988780",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-9.el9.s390x",
    sha256 = "59c54c89a7f6ffff9dd2e064b607992b2f0339a0fb6512596145b7e0ac931837",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libtirpc-1.3.3-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/59c54c89a7f6ffff9dd2e064b607992b2f0339a0fb6512596145b7e0ac931837",
    ],
)

rpm(
    name = "libtirpc-0__1.3.3-9.el9.x86_64",
    sha256 = "b0c69260f1a74faec97109c6b13de120f38903573e863892abc79b96b0a46f7f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtirpc-1.3.3-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b0c69260f1a74faec97109c6b13de120f38903573e863892abc79b96b0a46f7f",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-4.20211126git1ff6fe1f43.el9.aarch64",
    sha256 = "d499d04b1c4893e701c5d44fe4129993ef0f20c9b94fea1057367b72aa6ee4f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libtpms-0.9.1-4.20211126git1ff6fe1f43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d499d04b1c4893e701c5d44fe4129993ef0f20c9b94fea1057367b72aa6ee4f5",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-4.20211126git1ff6fe1f43.el9.s390x",
    sha256 = "bfd53e294938a568c972fada1152445c233bf5709f024c346a5a3f8f6ca0ac58",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libtpms-0.9.1-4.20211126git1ff6fe1f43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bfd53e294938a568c972fada1152445c233bf5709f024c346a5a3f8f6ca0ac58",
    ],
)

rpm(
    name = "libtpms-0__0.9.1-4.20211126git1ff6fe1f43.el9.x86_64",
    sha256 = "43083395bf6131abe2df8c9e0f27f6046aa47e6c8cf0a9092900e72527a0e21b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libtpms-0.9.1-4.20211126git1ff6fe1f43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/43083395bf6131abe2df8c9e0f27f6046aa47e6c8cf0a9092900e72527a0e21b",
    ],
)

rpm(
    name = "libubsan-0__11.5.0-2.el9.aarch64",
    sha256 = "46c551306d1bb8afc86efb95ff6154db82769ab9799a7b6fc8ba41188e78447e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.5.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/46c551306d1bb8afc86efb95ff6154db82769ab9799a7b6fc8ba41188e78447e",
    ],
)

rpm(
    name = "libubsan-0__11.5.0-2.el9.s390x",
    sha256 = "c8952e97c7cd103c1f5c8d17e3d43d88ed5e1a7c1b3bb21c092c08c46a20d4aa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libubsan-11.5.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c8952e97c7cd103c1f5c8d17e3d43d88ed5e1a7c1b3bb21c092c08c46a20d4aa",
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
    name = "libuuid-0__2.37.4-20.el9.aarch64",
    sha256 = "f1c54eeed0c892cb9cc3bea42e8c09b5a4b515381eb5d0fe6e5eb84346c51839",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libuuid-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f1c54eeed0c892cb9cc3bea42e8c09b5a4b515381eb5d0fe6e5eb84346c51839",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-20.el9.s390x",
    sha256 = "6021fe138b00f88d32a7745efac96331e7302e11c41aa302e04dd7283df8ab36",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libuuid-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6021fe138b00f88d32a7745efac96331e7302e11c41aa302e04dd7283df8ab36",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-20.el9.x86_64",
    sha256 = "10754bbddc76e88458ae6e9fd7b00cd6e5102c9e493eb2df73372b8f1d88dc1b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10754bbddc76e88458ae6e9fd7b00cd6e5102c9e493eb2df73372b8f1d88dc1b",
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
    name = "libvirt-client-0__10.5.0-5.el9.aarch64",
    sha256 = "9379da9163956436c5b9180a0d2a3b58e2bc4ae8ca43a0fdf7a6390db7e8c0d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-10.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9379da9163956436c5b9180a0d2a3b58e2bc4ae8ca43a0fdf7a6390db7e8c0d4",
    ],
)

rpm(
    name = "libvirt-client-0__10.5.0-5.el9.s390x",
    sha256 = "703e6bcafb4307469b4897fade8f8cba669d615a3f6198c66a9a503f4ffa7fd4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-client-10.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/703e6bcafb4307469b4897fade8f8cba669d615a3f6198c66a9a503f4ffa7fd4",
    ],
)

rpm(
    name = "libvirt-client-0__10.5.0-5.el9.x86_64",
    sha256 = "5f683496e89a986affe717019c67a470f4f86b2027c18256ee981fa54219e10a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5f683496e89a986affe717019c67a470f4f86b2027c18256ee981fa54219e10a",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.5.0-5.el9.aarch64",
    sha256 = "138ed7925f282abd54948b3bafca81a4cf2e59187bdad4b3a078d13624b73331",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-10.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/138ed7925f282abd54948b3bafca81a4cf2e59187bdad4b3a078d13624b73331",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.5.0-5.el9.s390x",
    sha256 = "420e17fbb2b553fcbd574b38e0d85b8a35bbde2cbd07989bcd94c1a3736dec7f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-10.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/420e17fbb2b553fcbd574b38e0d85b8a35bbde2cbd07989bcd94c1a3736dec7f",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.5.0-5.el9.x86_64",
    sha256 = "b59eedef3b227b54630ec4ed3fa8dbcfbcbc35eb51dc6b41da4275e1dbf23d48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b59eedef3b227b54630ec4ed3fa8dbcfbcbc35eb51dc6b41da4275e1dbf23d48",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.5.0-5.el9.aarch64",
    sha256 = "cd6c50a82335929d324bf5a886435c28b52462c559fe08a87ad62e203cb38f70",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-10.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cd6c50a82335929d324bf5a886435c28b52462c559fe08a87ad62e203cb38f70",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.5.0-5.el9.s390x",
    sha256 = "0beeb7a24c5f0534827f2b9dad42ecd9a273a186abc4b4dc199ef4f10183f6d1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-10.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0beeb7a24c5f0534827f2b9dad42ecd9a273a186abc4b4dc199ef4f10183f6d1",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.5.0-5.el9.x86_64",
    sha256 = "71cbc1a63a89fef444adafcf613f5547acb67f446329ed5438c406483e82ab5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/71cbc1a63a89fef444adafcf613f5547acb67f446329ed5438c406483e82ab5f",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.5.0-5.el9.x86_64",
    sha256 = "26b24a5648bc99e8021bbfcf0929981543adc0ad963595f8355908a80c2ec4ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/26b24a5648bc99e8021bbfcf0929981543adc0ad963595f8355908a80c2ec4ff",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.5.0-5.el9.x86_64",
    sha256 = "ccad3aabaf272d2c93a06cbc1ff409d9b97add1568270fa926dbf2cb89cfadeb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ccad3aabaf272d2c93a06cbc1ff409d9b97add1568270fa926dbf2cb89cfadeb",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.5.0-5.el9.aarch64",
    sha256 = "205022cc0fe9835f499889f55d03dd1d8dc16159a94adb03e3c001d82c6c1ec0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-10.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/205022cc0fe9835f499889f55d03dd1d8dc16159a94adb03e3c001d82c6c1ec0",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.5.0-5.el9.s390x",
    sha256 = "3b22e0de0a73464b1c4eeb865a3290f0402dce585bdc736859cf1a5de4c1d3f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-10.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3b22e0de0a73464b1c4eeb865a3290f0402dce585bdc736859cf1a5de4c1d3f5",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.5.0-5.el9.x86_64",
    sha256 = "fe541617dcd77bb94fabbe00b3f29fe94bae78701c1bd7f7fbfaaab7aea883f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe541617dcd77bb94fabbe00b3f29fe94bae78701c1bd7f7fbfaaab7aea883f2",
    ],
)

rpm(
    name = "libvirt-devel-0__10.5.0-5.el9.aarch64",
    sha256 = "c7bf84f133214d2f3ff1003937e539b13a72a342f5830f564feea53c59677fef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-10.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7bf84f133214d2f3ff1003937e539b13a72a342f5830f564feea53c59677fef",
    ],
)

rpm(
    name = "libvirt-devel-0__10.5.0-5.el9.s390x",
    sha256 = "0d32ab80ef663fb1d32e987f962524ba882a8ef0ebef5635bd1dfe56a12a64dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libvirt-devel-10.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0d32ab80ef663fb1d32e987f962524ba882a8ef0ebef5635bd1dfe56a12a64dc",
    ],
)

rpm(
    name = "libvirt-devel-0__10.5.0-5.el9.x86_64",
    sha256 = "ec117a3e47c2e22ab0a8b815c919b46152d43b8b1679d079afb5f02b6a7b8a45",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ec117a3e47c2e22ab0a8b815c919b46152d43b8b1679d079afb5f02b6a7b8a45",
    ],
)

rpm(
    name = "libvirt-libs-0__10.5.0-5.el9.aarch64",
    sha256 = "e92e0acc69865651c12664b2c484d4ae1a00c38b80dc1ca339daddb59cc3f28f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-10.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e92e0acc69865651c12664b2c484d4ae1a00c38b80dc1ca339daddb59cc3f28f",
    ],
)

rpm(
    name = "libvirt-libs-0__10.5.0-5.el9.s390x",
    sha256 = "a3f17f9a837a4e69247794253c1ae8c0a8f333fb9b81671c9bba6f1fd945d580",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-libs-10.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a3f17f9a837a4e69247794253c1ae8c0a8f333fb9b81671c9bba6f1fd945d580",
    ],
)

rpm(
    name = "libvirt-libs-0__10.5.0-5.el9.x86_64",
    sha256 = "5f8954367bb4125e2fdf647504b39963bf65405d77d693e5574fd62df80e6933",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-10.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5f8954367bb4125e2fdf647504b39963bf65405d77d693e5574fd62df80e6933",
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
    name = "ndctl-libs-0__78-2.el9.x86_64",
    sha256 = "6e8464b63dd264c7ade60f733747606dc214bbe3edf5e826836348c16fd3a970",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ndctl-libs-78-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6e8464b63dd264c7ade60f733747606dc214bbe3edf5e826836348c16fd3a970",
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
    name = "nfs-utils-1__2.5.4-27.el9.x86_64",
    sha256 = "5076c1a6a145e9cdce6005ea42e5064d20965bbf00af89900650f13ca655d976",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nfs-utils-2.5.4-27.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5076c1a6a145e9cdce6005ea42e5064d20965bbf00af89900650f13ca655d976",
    ],
)

rpm(
    name = "nftables-1__1.0.9-3.el9.aarch64",
    sha256 = "979faab3c0c318f4f1df5edd8b06efb20898461003237af3838f937d63b12d98",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.9-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/979faab3c0c318f4f1df5edd8b06efb20898461003237af3838f937d63b12d98",
    ],
)

rpm(
    name = "nftables-1__1.0.9-3.el9.s390x",
    sha256 = "a8d9bd2a045a06a50756af71d41a3d4d15677d120bb1cf833907db2e990adad0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nftables-1.0.9-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a8d9bd2a045a06a50756af71d41a3d4d15677d120bb1cf833907db2e990adad0",
    ],
)

rpm(
    name = "nftables-1__1.0.9-3.el9.x86_64",
    sha256 = "3f72eee1c40da5fa1f2eb59a77723f781ff27c53411b2aca1aee8bd6a577915b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.9-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f72eee1c40da5fa1f2eb59a77723f781ff27c53411b2aca1aee8bd6a577915b",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-3.el9.aarch64",
    sha256 = "8501b68c6a67b0a34f36b9c125fe5a1a71b6500460c160c1acc02131eaa280b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.92-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8501b68c6a67b0a34f36b9c125fe5a1a71b6500460c160c1acc02131eaa280b3",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-3.el9.s390x",
    sha256 = "0e3b53ac9a6711647f92c744038eca3ababd57253eb397d24fc8af2bebdd3e32",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/nmap-ncat-7.92-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0e3b53ac9a6711647f92c744038eca3ababd57253eb397d24fc8af2bebdd3e32",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-3.el9.x86_64",
    sha256 = "6126169e5ba3c3acc5fe8c458b425adb7beeeadf21177a5e3204931b9333b2ef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.92-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6126169e5ba3c3acc5fe8c458b425adb7beeeadf21177a5e3204931b9333b2ef",
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
    name = "numactl-libs-0__2.0.18-2.el9.aarch64",
    sha256 = "5907095ac70b01b5fd6baeffc983a5fa4ddf1f65fe1785b1a350b28c1d09bba7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.18-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5907095ac70b01b5fd6baeffc983a5fa4ddf1f65fe1785b1a350b28c1d09bba7",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.18-2.el9.s390x",
    sha256 = "dbdc42d76a55894ac794e524358b895c0f4484de01dced9ee36c26ab04db6baa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/numactl-libs-2.0.18-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/dbdc42d76a55894ac794e524358b895c0f4484de01dced9ee36c26ab04db6baa",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.18-2.el9.x86_64",
    sha256 = "9e7a56eb4196f4948a1ffeb164d078868ad08bb4ebc472634d95be855effdabc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.18-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9e7a56eb4196f4948a1ffeb164d078868ad08bb4ebc472634d95be855effdabc",
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
    name = "openldap-0__2.6.6-3.el9.x86_64",
    sha256 = "da4c54a99c4556ab6c95f91ac0f472e8e96509fd97a59f45e196c0f613a1dbab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openldap-2.6.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da4c54a99c4556ab6c95f91ac0f472e8e96509fd97a59f45e196c0f613a1dbab",
    ],
)

rpm(
    name = "openssl-1__3.2.2-6.el9.aarch64",
    sha256 = "6cad4a9668d8f7c1e813ebed24bf536cc087b529bf2287dacf99b4087fb88e7c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.2.2-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6cad4a9668d8f7c1e813ebed24bf536cc087b529bf2287dacf99b4087fb88e7c",
    ],
)

rpm(
    name = "openssl-1__3.2.2-6.el9.s390x",
    sha256 = "78c30e1672240edd49aead126b5e7a252edbc5b759602dd3f5f24ea285399b36",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-3.2.2-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/78c30e1672240edd49aead126b5e7a252edbc5b759602dd3f5f24ea285399b36",
    ],
)

rpm(
    name = "openssl-1__3.2.2-6.el9.x86_64",
    sha256 = "3018c5d2901213b6bdbe62301ef894008ec52b1122e270190eabb62ad282a46a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.2.2-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3018c5d2901213b6bdbe62301ef894008ec52b1122e270190eabb62ad282a46a",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-6.el9.aarch64",
    sha256 = "87c5306a737dd0d6e048aa13b8d87d65e22b58f05b086ce1d84d236d32ab83d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.2.2-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/87c5306a737dd0d6e048aa13b8d87d65e22b58f05b086ce1d84d236d32ab83d3",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-6.el9.s390x",
    sha256 = "3c57bf4b0a4f527797052d881da42bff488e4ffdaa365d95ea16d7e874299073",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-libs-3.2.2-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3c57bf4b0a4f527797052d881da42bff488e4ffdaa365d95ea16d7e874299073",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-6.el9.x86_64",
    sha256 = "4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.2.2-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738",
    ],
)

rpm(
    name = "osinfo-db-0__20240701-3.el9.x86_64",
    sha256 = "cb5ee4fa514502a1e731749cf6d77475f274e4bec88449bfc50daa8e32b9a0ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/osinfo-db-20240701-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/cb5ee4fa514502a1e731749cf6d77475f274e4bec88449bfc50daa8e32b9a0ca",
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
    name = "p11-kit-0__0.25.3-3.el9.aarch64",
    sha256 = "6a255062581be1ba36a33d1b22b46f129fc42d20e0e300c0e8f57639f2951266",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-0.25.3-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6a255062581be1ba36a33d1b22b46f129fc42d20e0e300c0e8f57639f2951266",
    ],
)

rpm(
    name = "p11-kit-0__0.25.3-3.el9.s390x",
    sha256 = "11c4d8edac3f3944104ba989ee9460efc1ba6a8bf79a3bae49b395c0cc7fd5dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-0.25.3-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/11c4d8edac3f3944104ba989ee9460efc1ba6a8bf79a3bae49b395c0cc7fd5dc",
    ],
)

rpm(
    name = "p11-kit-0__0.25.3-3.el9.x86_64",
    sha256 = "2d02f32cdb62fac32563c70fad44c7252f0173552ccabc58d2b5161207c291a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-0.25.3-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d02f32cdb62fac32563c70fad44c7252f0173552ccabc58d2b5161207c291a3",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.25.3-3.el9.aarch64",
    sha256 = "6edd6a98ba08cf62dfe98fca9d16337808321504c778165cf8aff055b63dcd06",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.25.3-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6edd6a98ba08cf62dfe98fca9d16337808321504c778165cf8aff055b63dcd06",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.25.3-3.el9.s390x",
    sha256 = "7aa852f515edcc3056bbf35e208bd8f57f68577bff047e54337fa87e0505057a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-trust-0.25.3-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7aa852f515edcc3056bbf35e208bd8f57f68577bff047e54337fa87e0505057a",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.25.3-3.el9.x86_64",
    sha256 = "f3b18cc69d79899e17d7c7514a4e350bdd6166a37e979fee5dcfbdc7921a02fa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.25.3-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f3b18cc69d79899e17d7c7514a4e350bdd6166a37e979fee5dcfbdc7921a02fa",
    ],
)

rpm(
    name = "pam-0__1.5.1-22.el9.aarch64",
    sha256 = "551494003d266647324f45f5a16308af451ce7d0fd80fe73149d2cea33ca9517",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/551494003d266647324f45f5a16308af451ce7d0fd80fe73149d2cea33ca9517",
    ],
)

rpm(
    name = "pam-0__1.5.1-22.el9.s390x",
    sha256 = "da0b4a4705170e99d0e6b8773d146b6b1a58b8df0f6b76b0bf13a30b499e5465",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pam-1.5.1-22.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/da0b4a4705170e99d0e6b8773d146b6b1a58b8df0f6b76b0bf13a30b499e5465",
    ],
)

rpm(
    name = "pam-0__1.5.1-22.el9.x86_64",
    sha256 = "ac2b01bccf551d181bef0566b6160fddef3bfd268ff9f4b6e1360187e1573d37",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-22.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ac2b01bccf551d181bef0566b6160fddef3bfd268ff9f4b6e1360187e1573d37",
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
    name = "passt-0__0__caret__20240806.gee36266-2.el9.aarch64",
    sha256 = "f494976aa0fe23d037bb1739f5e55e217c6b936c60c8bf7d4bbb0c0d061b4e5f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/passt-0%5E20240806.gee36266-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f494976aa0fe23d037bb1739f5e55e217c6b936c60c8bf7d4bbb0c0d061b4e5f",
    ],
)

rpm(
    name = "passt-0__0__caret__20240806.gee36266-2.el9.s390x",
    sha256 = "0377e4d064dda2ffe7b0af454fbf17477f937d8358130947b89b32ec9194cc79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/passt-0%5E20240806.gee36266-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0377e4d064dda2ffe7b0af454fbf17477f937d8358130947b89b32ec9194cc79",
    ],
)

rpm(
    name = "passt-0__0__caret__20240806.gee36266-2.el9.x86_64",
    sha256 = "40afca646a77fb61983e922ddc119bdab7459ed7a586ab97f38f2f9538fbfb7a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/passt-0%5E20240806.gee36266-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/40afca646a77fb61983e922ddc119bdab7459ed7a586ab97f38f2f9538fbfb7a",
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
    name = "pcre2-0__10.40-6.el9.aarch64",
    sha256 = "c13e323c383bd5bbe3415701aa21a56b3fefc32d96e081e91c012ef692c78599",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre2-10.40-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c13e323c383bd5bbe3415701aa21a56b3fefc32d96e081e91c012ef692c78599",
    ],
)

rpm(
    name = "pcre2-0__10.40-6.el9.s390x",
    sha256 = "f7c2df461b8fe6a9617a1c1089fc88576e4df16f6ff9aea83b05413d2e15b4d5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pcre2-10.40-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f7c2df461b8fe6a9617a1c1089fc88576e4df16f6ff9aea83b05413d2e15b4d5",
    ],
)

rpm(
    name = "pcre2-0__10.40-6.el9.x86_64",
    sha256 = "bc1012f5417aab8393836d78ac8c5472b1a2d84a2f9fa2b00fff5f8ad3a5ec26",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre2-10.40-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc1012f5417aab8393836d78ac8c5472b1a2d84a2f9fa2b00fff5f8ad3a5ec26",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-6.el9.aarch64",
    sha256 = "be36a84f6e311a59190664d61a466471391ab01fb77bd1d2348e9a76414aded4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pcre2-syntax-10.40-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/be36a84f6e311a59190664d61a466471391ab01fb77bd1d2348e9a76414aded4",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-6.el9.s390x",
    sha256 = "be36a84f6e311a59190664d61a466471391ab01fb77bd1d2348e9a76414aded4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pcre2-syntax-10.40-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/be36a84f6e311a59190664d61a466471391ab01fb77bd1d2348e9a76414aded4",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.40-6.el9.x86_64",
    sha256 = "be36a84f6e311a59190664d61a466471391ab01fb77bd1d2348e9a76414aded4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pcre2-syntax-10.40-6.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/be36a84f6e311a59190664d61a466471391ab01fb77bd1d2348e9a76414aded4",
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
    name = "python3-0__3.9.20-1.el9.aarch64",
    sha256 = "c89035c2f2e620ec2416dc34d7789ab3a5b35394e411d32302d14b2ddeb01949",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.20-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c89035c2f2e620ec2416dc34d7789ab3a5b35394e411d32302d14b2ddeb01949",
    ],
)

rpm(
    name = "python3-0__3.9.20-1.el9.s390x",
    sha256 = "b4e1b50e218955f29c8893319d9da8b99dfaa58c605bd42b5d9a1b6e18355617",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-3.9.20-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b4e1b50e218955f29c8893319d9da8b99dfaa58c605bd42b5d9a1b6e18355617",
    ],
)

rpm(
    name = "python3-0__3.9.20-1.el9.x86_64",
    sha256 = "c953b63b9c98d58dfe03317a19422e7e90520afc959f9b91bfb64a1e02058d9a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.20-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c953b63b9c98d58dfe03317a19422e7e90520afc959f9b91bfb64a1e02058d9a",
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
    name = "python3-libs-0__3.9.20-1.el9.aarch64",
    sha256 = "966500975377eb81a9ccc373b162ea837879bcd353d0cfa377a660a8a3139ec9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.20-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/966500975377eb81a9ccc373b162ea837879bcd353d0cfa377a660a8a3139ec9",
    ],
)

rpm(
    name = "python3-libs-0__3.9.20-1.el9.s390x",
    sha256 = "9d415c11db24986607de1d35a0b19cc32698c85e4969b6cb6f8f572be8b22897",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-libs-3.9.20-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9d415c11db24986607de1d35a0b19cc32698c85e4969b6cb6f8f572be8b22897",
    ],
)

rpm(
    name = "python3-libs-0__3.9.20-1.el9.x86_64",
    sha256 = "b0f2d674195e28e3c9b4a116f87ef221edda432ca3a04ab5be70b3062c883c01",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.20-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b0f2d674195e28e3c9b4a116f87ef221edda432ca3a04ab5be70b3062c883c01",
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
    name = "python3-setuptools-wheel-0__53.0.0-13.el9.aarch64",
    sha256 = "a4dfbc2c514f58839d7704acc046eb0fc54cfb670413decebd9641b4d76439e8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-53.0.0-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a4dfbc2c514f58839d7704acc046eb0fc54cfb670413decebd9641b4d76439e8",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-13.el9.s390x",
    sha256 = "a4dfbc2c514f58839d7704acc046eb0fc54cfb670413decebd9641b4d76439e8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-setuptools-wheel-53.0.0-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a4dfbc2c514f58839d7704acc046eb0fc54cfb670413decebd9641b4d76439e8",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-13.el9.x86_64",
    sha256 = "a4dfbc2c514f58839d7704acc046eb0fc54cfb670413decebd9641b4d76439e8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-53.0.0-13.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a4dfbc2c514f58839d7704acc046eb0fc54cfb670413decebd9641b4d76439e8",
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
    name = "qemu-img-17__9.0.0-10.el9.aarch64",
    sha256 = "2860419c864609d35bbd376a7f7fdf3ef44052464c6e8616f2d654062c1c48fc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2860419c864609d35bbd376a7f7fdf3ef44052464c6e8616f2d654062c1c48fc",
    ],
)

rpm(
    name = "qemu-img-17__9.0.0-10.el9.s390x",
    sha256 = "db4b57641bc142f65f246e6f239a373a01563b9a2d37f591193945c48888bbe2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-img-9.0.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/db4b57641bc142f65f246e6f239a373a01563b9a2d37f591193945c48888bbe2",
    ],
)

rpm(
    name = "qemu-img-17__9.0.0-10.el9.x86_64",
    sha256 = "11433926242da50d0327b4df19ae3e573359db208a093e514c26bdbabf2b8269",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-9.0.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11433926242da50d0327b4df19ae3e573359db208a093e514c26bdbabf2b8269",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.0.0-10.el9.aarch64",
    sha256 = "5c52ac09c816352d1f69330923d07bfa2c3900f1b4bd6d701dec6b004cdbc2b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5c52ac09c816352d1f69330923d07bfa2c3900f1b4bd6d701dec6b004cdbc2b1",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.0.0-10.el9.s390x",
    sha256 = "f5b1b1a41d9f251ac30d41ed36b854386e55566c1b581151f38ced8e3042e95a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-common-9.0.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f5b1b1a41d9f251ac30d41ed36b854386e55566c1b581151f38ced8e3042e95a",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.0.0-10.el9.x86_64",
    sha256 = "8e28839fb742b45c4473ba32aa9ff5c30e7d94a629b3bed933a1d5dbab4f9e23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-9.0.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e28839fb742b45c4473ba32aa9ff5c30e7d94a629b3bed933a1d5dbab4f9e23",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.0.0-10.el9.aarch64",
    sha256 = "ffc7e54e1f73d7af78f516736e2f0be5116193cfe2c4818d04b3277122395ab2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ffc7e54e1f73d7af78f516736e2f0be5116193cfe2c4818d04b3277122395ab2",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.0.0-10.el9.s390x",
    sha256 = "3c787c8135493b0ce6164650b8341421866e0b4b042bb21967b6f5fa8106e89d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-core-9.0.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3c787c8135493b0ce6164650b8341421866e0b4b042bb21967b6f5fa8106e89d",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.0.0-10.el9.x86_64",
    sha256 = "e9ae1c91789375489de8275d9c38a55fbfc85f56c6069f5e7a29d15ff033c637",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-9.0.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e9ae1c91789375489de8275d9c38a55fbfc85f56c6069f5e7a29d15ff033c637",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__9.0.0-10.el9.aarch64",
    sha256 = "dd5f3b2aa4504ec7308fcd415c6d01f3448d4b032ef710990faf4a3b617188e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dd5f3b2aa4504ec7308fcd415c6d01f3448d4b032ef710990faf4a3b617188e9",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__9.0.0-10.el9.s390x",
    sha256 = "17a1c4fafa35972b5c53f7b5c602b202196b63b6ee18364a169744d25e88873d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-9.0.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/17a1c4fafa35972b5c53f7b5c602b202196b63b6ee18364a169744d25e88873d",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-ccw-17__9.0.0-10.el9.s390x",
    sha256 = "62d283feeaef8867099921894cbdf628e4c7849678e3e02a6ef1a9d4913532a9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-9.0.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/62d283feeaef8867099921894cbdf628e4c7849678e3e02a6ef1a9d4913532a9",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__9.0.0-10.el9.aarch64",
    sha256 = "f4e94839c87c011a286190351b1fa1b917b1276d90fea7c9daca6b8702a9df45",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f4e94839c87c011a286190351b1fa1b917b1276d90fea7c9daca6b8702a9df45",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__9.0.0-10.el9.aarch64",
    sha256 = "8d2acdbf16e75d50a548ed15796c37453c7ad7943e0d2708aacda08851c103b6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8d2acdbf16e75d50a548ed15796c37453c7ad7943e0d2708aacda08851c103b6",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__9.0.0-10.el9.s390x",
    sha256 = "0a1a28784a172d423a5c36965a66d04106653c3e177efd07fed52cc11bf26ed1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-9.0.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0a1a28784a172d423a5c36965a66d04106653c3e177efd07fed52cc11bf26ed1",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__9.0.0-10.el9.x86_64",
    sha256 = "8f0bf1a843a0584f1e6ed2f086a0d7843284a7aa39ebf058fa08597c69f0e0b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-9.0.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8f0bf1a843a0584f1e6ed2f086a0d7843284a7aa39ebf058fa08597c69f0e0b2",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__9.0.0-10.el9.aarch64",
    sha256 = "8218b7d416097d274695398a252c43eaac0cd29c3639147ed43c7530c4f714e4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-9.0.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8218b7d416097d274695398a252c43eaac0cd29c3639147ed43c7530c4f714e4",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__9.0.0-10.el9.x86_64",
    sha256 = "486ad75b29cd7f52587b58a0a94ffdc76eba6f3600f5a3e8b7598fb0bb80d242",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-9.0.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/486ad75b29cd7f52587b58a0a94ffdc76eba6f3600f5a3e8b7598fb0bb80d242",
    ],
)

rpm(
    name = "qemu-pr-helper-17__9.1.0-1.el9.aarch64",
    sha256 = "8650db3dc0d7fc2d4949e7e2e10f8c373e3a82fcfab1263e38ff09f521e994e2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-9.1.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8650db3dc0d7fc2d4949e7e2e10f8c373e3a82fcfab1263e38ff09f521e994e2",
    ],
)

rpm(
    name = "qemu-pr-helper-17__9.1.0-1.el9.x86_64",
    sha256 = "c5ad0bed2114a857aae25efa204043b7eee7a98e2aff6c70e33e8b72bb0c0840",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-9.1.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c5ad0bed2114a857aae25efa204043b7eee7a98e2aff6c70e33e8b72bb0c0840",
    ],
)

rpm(
    name = "quota-1__4.09-4.el9.x86_64",
    sha256 = "3e2302251e7c3d5088bce6e7d4fc35d0b84aecf93cb4f061492a7bdfb586850d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/quota-4.09-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3e2302251e7c3d5088bce6e7d4fc35d0b84aecf93cb4f061492a7bdfb586850d",
    ],
)

rpm(
    name = "quota-nls-1__4.09-4.el9.x86_64",
    sha256 = "1f94f8c3e304567967087e1494096b06a5a1acbe94bc8e542ecd4cf9e7e9e4a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/quota-nls-4.09-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1f94f8c3e304567967087e1494096b06a5a1acbe94bc8e542ecd4cf9e7e9e4a4",
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
    name = "rpm-0__4.16.1.3-36.el9.aarch64",
    sha256 = "68974d8d9153d1d3a01844b29d5804f1ff8b362303d81d80bcb2dad666f5306d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-4.16.1.3-36.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/68974d8d9153d1d3a01844b29d5804f1ff8b362303d81d80bcb2dad666f5306d",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-36.el9.s390x",
    sha256 = "992e8faf8de845fad8c818763c2b36e4f421e9e7d1cce55b60929c1b9fb7d024",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-4.16.1.3-36.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/992e8faf8de845fad8c818763c2b36e4f421e9e7d1cce55b60929c1b9fb7d024",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-36.el9.x86_64",
    sha256 = "510047c59adc4f6ec2272feaeb4f707d7cab71485c916a0b3ed438fa2be084cf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-36.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/510047c59adc4f6ec2272feaeb4f707d7cab71485c916a0b3ed438fa2be084cf",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-36.el9.aarch64",
    sha256 = "31f90b50268f22c59215fdedd60310ab82f13c80d09c6f138677ac5b07f168ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.16.1.3-36.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/31f90b50268f22c59215fdedd60310ab82f13c80d09c6f138677ac5b07f168ee",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-36.el9.s390x",
    sha256 = "30b123fe376babacf3cbc58a70ffc619dec4c8c36da6ed0218ac3380cbec12ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-libs-4.16.1.3-36.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/30b123fe376babacf3cbc58a70ffc619dec4c8c36da6ed0218ac3380cbec12ce",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-36.el9.x86_64",
    sha256 = "e78e9d692bf9fa1a5113ef89d124cf5707224e15e141a09b67cf819ff25cb875",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-36.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e78e9d692bf9fa1a5113ef89d124cf5707224e15e141a09b67cf819ff25cb875",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-36.el9.aarch64",
    sha256 = "f95f5a95cec3e85cd4ad269ae6e55cc598796daf85c770f000a83d3b046a7a52",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.16.1.3-36.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f95f5a95cec3e85cd4ad269ae6e55cc598796daf85c770f000a83d3b046a7a52",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-36.el9.s390x",
    sha256 = "36acbafeac8af6f150cd9a7f402187d5d6505611701f42ff842b27cfa3bbf475",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-plugin-selinux-4.16.1.3-36.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/36acbafeac8af6f150cd9a7f402187d5d6505611701f42ff842b27cfa3bbf475",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-36.el9.x86_64",
    sha256 = "3c1d0b39a9a6a0d5a24a13db81c75823295977aac1afdd76068b3497b2e12ac8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-36.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3c1d0b39a9a6a0d5a24a13db81c75823295977aac1afdd76068b3497b2e12ac8",
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
    name = "selinux-policy-0__38.1.48-1.el9.aarch64",
    sha256 = "e9a9887573f5bd240c02136e21a30e1347da3f2ad33c32505fe03a86070b2fdb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-38.1.48-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e9a9887573f5bd240c02136e21a30e1347da3f2ad33c32505fe03a86070b2fdb",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.48-1.el9.s390x",
    sha256 = "e9a9887573f5bd240c02136e21a30e1347da3f2ad33c32505fe03a86070b2fdb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-38.1.48-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e9a9887573f5bd240c02136e21a30e1347da3f2ad33c32505fe03a86070b2fdb",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.48-1.el9.x86_64",
    sha256 = "e9a9887573f5bd240c02136e21a30e1347da3f2ad33c32505fe03a86070b2fdb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-38.1.48-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e9a9887573f5bd240c02136e21a30e1347da3f2ad33c32505fe03a86070b2fdb",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.48-1.el9.aarch64",
    sha256 = "d5c54ab49a832db799b92119d6fccee76fdfead6bd50d2ec5d97d6ff5a01e90b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-38.1.48-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d5c54ab49a832db799b92119d6fccee76fdfead6bd50d2ec5d97d6ff5a01e90b",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.48-1.el9.s390x",
    sha256 = "d5c54ab49a832db799b92119d6fccee76fdfead6bd50d2ec5d97d6ff5a01e90b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-38.1.48-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d5c54ab49a832db799b92119d6fccee76fdfead6bd50d2ec5d97d6ff5a01e90b",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.48-1.el9.x86_64",
    sha256 = "d5c54ab49a832db799b92119d6fccee76fdfead6bd50d2ec5d97d6ff5a01e90b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-38.1.48-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d5c54ab49a832db799b92119d6fccee76fdfead6bd50d2ec5d97d6ff5a01e90b",
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
    name = "shadow-utils-2__4.9-12.el9.aarch64",
    sha256 = "37f2e7bbe372bcceaa50f9d36bdc821e6ec13092a580f22c2e15d08a5c5c46ac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.9-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/37f2e7bbe372bcceaa50f9d36bdc821e6ec13092a580f22c2e15d08a5c5c46ac",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-12.el9.s390x",
    sha256 = "be7591a5fc1954e2328195a50c113c7ceb07d5bdc563dcd5d02956993ed65f6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/shadow-utils-4.9-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/be7591a5fc1954e2328195a50c113c7ceb07d5bdc563dcd5d02956993ed65f6f",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-12.el9.x86_64",
    sha256 = "23f14143a188cf9bf8a0315f930fbeeb0ad34c58357007a52d112c5f8b6029e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23f14143a188cf9bf8a0315f930fbeeb0ad34c58357007a52d112c5f8b6029e0",
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
    name = "sssd-client-0__2.9.5-5.el9.aarch64",
    sha256 = "e54023971610562b976fd11bb97051c1fe8f308999498528552856eefa7df921",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.9.5-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e54023971610562b976fd11bb97051c1fe8f308999498528552856eefa7df921",
    ],
)

rpm(
    name = "sssd-client-0__2.9.5-5.el9.s390x",
    sha256 = "c1a27d4aa4adccdaca4464e9b50d9309bacaee87e2343b43250eae54b92be8c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sssd-client-2.9.5-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c1a27d4aa4adccdaca4464e9b50d9309bacaee87e2343b43250eae54b92be8c3",
    ],
)

rpm(
    name = "sssd-client-0__2.9.5-5.el9.x86_64",
    sha256 = "56bc2403d1121c55d8c06f2ee33ea0edf8b250b0830eaaffd16107adf39ae076",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.9.5-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56bc2403d1121c55d8c06f2ee33ea0edf8b250b0830eaaffd16107adf39ae076",
    ],
)

rpm(
    name = "swtpm-0__0.8.0-2.el9.aarch64",
    sha256 = "54ab5545703dbce2156675bda5719e530beff7b62970824db3cc6db96648c3a5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-0.8.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/54ab5545703dbce2156675bda5719e530beff7b62970824db3cc6db96648c3a5",
    ],
)

rpm(
    name = "swtpm-0__0.8.0-2.el9.s390x",
    sha256 = "2eb083281ba5e1d44cea3325c50549202c44b8c1331a92fc0056625e54b6be74",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/swtpm-0.8.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2eb083281ba5e1d44cea3325c50549202c44b8c1331a92fc0056625e54b6be74",
    ],
)

rpm(
    name = "swtpm-0__0.8.0-2.el9.x86_64",
    sha256 = "e09635dac83f4f3d75b5b61bbe4879d013e38066c6cc07ab2b38bd355ff915ba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-0.8.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e09635dac83f4f3d75b5b61bbe4879d013e38066c6cc07ab2b38bd355ff915ba",
    ],
)

rpm(
    name = "swtpm-libs-0__0.8.0-2.el9.aarch64",
    sha256 = "da68ca794b6517e3af94f9edfa815269b4a25446f39751a0d4abe7528a465fd5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.8.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da68ca794b6517e3af94f9edfa815269b4a25446f39751a0d4abe7528a465fd5",
    ],
)

rpm(
    name = "swtpm-libs-0__0.8.0-2.el9.s390x",
    sha256 = "2b6024dcaa008808c7c1b4b3409194db3d1813655aaaf399fe27ec6690c3f6c5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/swtpm-libs-0.8.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2b6024dcaa008808c7c1b4b3409194db3d1813655aaaf399fe27ec6690c3f6c5",
    ],
)

rpm(
    name = "swtpm-libs-0__0.8.0-2.el9.x86_64",
    sha256 = "732895c380d3474aebda2c8fa3e2de1f5219fce246a188b936ed7f9a9e6077d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.8.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/732895c380d3474aebda2c8fa3e2de1f5219fce246a188b936ed7f9a9e6077d3",
    ],
)

rpm(
    name = "swtpm-tools-0__0.8.0-2.el9.aarch64",
    sha256 = "35d142d4a3fbf02732a0ed0edaccd71399e34a19286ced7b00c0f5d79d4d3685",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.8.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/35d142d4a3fbf02732a0ed0edaccd71399e34a19286ced7b00c0f5d79d4d3685",
    ],
)

rpm(
    name = "swtpm-tools-0__0.8.0-2.el9.s390x",
    sha256 = "81e3af9e0ba27e5fc782df6a9177e84e8ee032f0b70bb68c97de8a3376cb91f1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/swtpm-tools-0.8.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/81e3af9e0ba27e5fc782df6a9177e84e8ee032f0b70bb68c97de8a3376cb91f1",
    ],
)

rpm(
    name = "swtpm-tools-0__0.8.0-2.el9.x86_64",
    sha256 = "8bb8baa44595a786df5d7309f03c309c4dd9ae288f0d444f371eaca42560ab97",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.8.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8bb8baa44595a786df5d7309f03c309c4dd9ae288f0d444f371eaca42560ab97",
    ],
)

rpm(
    name = "systemd-0__252-48.el9.aarch64",
    sha256 = "bb906860c1659e84b7885b5d2c8066cf5a07b2c18cbfb896fa2e27debf757fcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-252-48.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bb906860c1659e84b7885b5d2c8066cf5a07b2c18cbfb896fa2e27debf757fcc",
    ],
)

rpm(
    name = "systemd-0__252-48.el9.s390x",
    sha256 = "2a459f3e8e38e32dfea165d7c218898a28491dca42946a54856a95488629227d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-252-48.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2a459f3e8e38e32dfea165d7c218898a28491dca42946a54856a95488629227d",
    ],
)

rpm(
    name = "systemd-0__252-48.el9.x86_64",
    sha256 = "abd114db84ba553cd5197c2c88b2d64595ba508c25829db3a73e79aa35ca3562",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-48.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/abd114db84ba553cd5197c2c88b2d64595ba508c25829db3a73e79aa35ca3562",
    ],
)

rpm(
    name = "systemd-container-0__252-48.el9.aarch64",
    sha256 = "5ce8a1cdb2c2387824bf5e9859be253796400b7dc78fb132f96655035da69b98",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-252-48.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5ce8a1cdb2c2387824bf5e9859be253796400b7dc78fb132f96655035da69b98",
    ],
)

rpm(
    name = "systemd-container-0__252-48.el9.s390x",
    sha256 = "97680886bdce9eead4b31cb3eea3e956afde3da97b7d917eb97c28962b2428c5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-container-252-48.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/97680886bdce9eead4b31cb3eea3e956afde3da97b7d917eb97c28962b2428c5",
    ],
)

rpm(
    name = "systemd-container-0__252-48.el9.x86_64",
    sha256 = "3cd25e68c916d26cd190a9e937342f09d83dd35448541c3e42b2184fd6cf0285",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-48.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3cd25e68c916d26cd190a9e937342f09d83dd35448541c3e42b2184fd6cf0285",
    ],
)

rpm(
    name = "systemd-libs-0__252-48.el9.aarch64",
    sha256 = "5c839f31ec5f5ec13a2acaf517f5f020176273963f4a449b5530e2c8dd0ba005",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-252-48.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5c839f31ec5f5ec13a2acaf517f5f020176273963f4a449b5530e2c8dd0ba005",
    ],
)

rpm(
    name = "systemd-libs-0__252-48.el9.s390x",
    sha256 = "82d99989fd1418b3471720aa15bc3d9d9b928256e69b999ad8c2125cf611ae41",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-libs-252-48.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/82d99989fd1418b3471720aa15bc3d9d9b928256e69b999ad8c2125cf611ae41",
    ],
)

rpm(
    name = "systemd-libs-0__252-48.el9.x86_64",
    sha256 = "efa67de893a41b571e9aa812e2a5b81329d1d18679d3136cf1076bb23ce7fb3f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-48.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/efa67de893a41b571e9aa812e2a5b81329d1d18679d3136cf1076bb23ce7fb3f",
    ],
)

rpm(
    name = "systemd-pam-0__252-48.el9.aarch64",
    sha256 = "4756637ff1e21177ac0311bb3149785bd42f69356d9c0eca5947e45bfb362205",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-252-48.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4756637ff1e21177ac0311bb3149785bd42f69356d9c0eca5947e45bfb362205",
    ],
)

rpm(
    name = "systemd-pam-0__252-48.el9.s390x",
    sha256 = "9e86c59650bcce6c5bbaba0bbf27b2d2ca50df22aa3bace4da57dd40df8f03d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-pam-252-48.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9e86c59650bcce6c5bbaba0bbf27b2d2ca50df22aa3bace4da57dd40df8f03d7",
    ],
)

rpm(
    name = "systemd-pam-0__252-48.el9.x86_64",
    sha256 = "3f0fc54a9c6562656be20f9f2366582412698491ceb6d67cae0d3dea6c01d56b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-48.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f0fc54a9c6562656be20f9f2366582412698491ceb6d67cae0d3dea6c01d56b",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-48.el9.aarch64",
    sha256 = "db949d3c631f08638881f5ebef5a5ce8587f90cd2138bf971a8e41664a3adc8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-252-48.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db949d3c631f08638881f5ebef5a5ce8587f90cd2138bf971a8e41664a3adc8e",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-48.el9.s390x",
    sha256 = "db949d3c631f08638881f5ebef5a5ce8587f90cd2138bf971a8e41664a3adc8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-rpm-macros-252-48.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db949d3c631f08638881f5ebef5a5ce8587f90cd2138bf971a8e41664a3adc8e",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-48.el9.x86_64",
    sha256 = "db949d3c631f08638881f5ebef5a5ce8587f90cd2138bf971a8e41664a3adc8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-48.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/db949d3c631f08638881f5ebef5a5ce8587f90cd2138bf971a8e41664a3adc8e",
    ],
)

rpm(
    name = "tar-2__1.34-7.el9.aarch64",
    sha256 = "e3ee12a44a68c84627e43c2512ad8904a4778a82b274d0e8147ca46645f4a1fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tar-1.34-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e3ee12a44a68c84627e43c2512ad8904a4778a82b274d0e8147ca46645f4a1fb",
    ],
)

rpm(
    name = "tar-2__1.34-7.el9.s390x",
    sha256 = "304bca9dd546a39a59bd50b8ec5fb3f42898138f92e49945be09cab503cdf1a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tar-1.34-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/304bca9dd546a39a59bd50b8ec5fb3f42898138f92e49945be09cab503cdf1a2",
    ],
)

rpm(
    name = "tar-2__1.34-7.el9.x86_64",
    sha256 = "b90b0e6f70433d3935b1dd45a3c10a40768950b5c9121545034179bd7b55159f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tar-1.34-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b90b0e6f70433d3935b1dd45a3c10a40768950b5c9121545034179bd7b55159f",
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
    name = "tzdata-0__2024b-2.el9.aarch64",
    sha256 = "909bc0b9ad6e9c76a11cb737b8911a7ea4a1e2374c7a4eb39c9f718739c6dfff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2024b-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/909bc0b9ad6e9c76a11cb737b8911a7ea4a1e2374c7a4eb39c9f718739c6dfff",
    ],
)

rpm(
    name = "tzdata-0__2024b-2.el9.s390x",
    sha256 = "909bc0b9ad6e9c76a11cb737b8911a7ea4a1e2374c7a4eb39c9f718739c6dfff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tzdata-2024b-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/909bc0b9ad6e9c76a11cb737b8911a7ea4a1e2374c7a4eb39c9f718739c6dfff",
    ],
)

rpm(
    name = "tzdata-0__2024b-2.el9.x86_64",
    sha256 = "909bc0b9ad6e9c76a11cb737b8911a7ea4a1e2374c7a4eb39c9f718739c6dfff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2024b-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/909bc0b9ad6e9c76a11cb737b8911a7ea4a1e2374c7a4eb39c9f718739c6dfff",
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
    name = "util-linux-0__2.37.4-20.el9.aarch64",
    sha256 = "76ae6df88815700e14674fd1acd5d2162fd023374c98dc53c000e0f7b574288a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76ae6df88815700e14674fd1acd5d2162fd023374c98dc53c000e0f7b574288a",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-20.el9.s390x",
    sha256 = "fd814b3b94ffe1f905a49308c8d5863b13d865ba48dcca68d6d2b2d09677d610",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fd814b3b94ffe1f905a49308c8d5863b13d865ba48dcca68d6d2b2d09677d610",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-20.el9.x86_64",
    sha256 = "5011faf8c26d7402f1f0438687e3393b1d6a64eaa2ac7f30c1dcf472e8635ef5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5011faf8c26d7402f1f0438687e3393b1d6a64eaa2ac7f30c1dcf472e8635ef5",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-20.el9.aarch64",
    sha256 = "7f452299af4a3e656fc3aa59a3ce91f61ce1a57e9753a5fbbc5886db5e5fe36a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.37.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7f452299af4a3e656fc3aa59a3ce91f61ce1a57e9753a5fbbc5886db5e5fe36a",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-20.el9.s390x",
    sha256 = "5c751a55026449698454e4de778bfbb5acb5d890e8fdace4a0d9826ad9423108",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-core-2.37.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c751a55026449698454e4de778bfbb5acb5d890e8fdace4a0d9826ad9423108",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-20.el9.x86_64",
    sha256 = "e4df98c254564404ae8750d6105290dedf18593ce53654b66ed9cb170bbfbcc7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.37.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e4df98c254564404ae8750d6105290dedf18593ce53654b66ed9cb170bbfbcc7",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-21.el9.aarch64",
    sha256 = "2a06e6863cc4d8c699b727424f2e0a06c75f5c8265cb2bc576242054d1bff444",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.2.2637-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2a06e6863cc4d8c699b727424f2e0a06c75f5c8265cb2bc576242054d1bff444",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-21.el9.s390x",
    sha256 = "a04988c53eea9735bb2eb5106e7e2215f5a355af2c33dfbe20c643c811b9176f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/vim-minimal-8.2.2637-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a04988c53eea9735bb2eb5106e7e2215f5a355af2c33dfbe20c643c811b9176f",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-21.el9.x86_64",
    sha256 = "1b15304790e4b2e7d4ff378b7bf0363b6ecb1c852fc42f984267296538de0c16",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1b15304790e4b2e7d4ff378b7bf0363b6ecb1c852fc42f984267296538de0c16",
    ],
)

rpm(
    name = "virtiofsd-0__1.11.1-1.el9.aarch64",
    sha256 = "0ba3ac4fee86f207f25ed0c5223be3cd7e88af6d97cb6c4904d35b1f240a6861",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/virtiofsd-1.11.1-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0ba3ac4fee86f207f25ed0c5223be3cd7e88af6d97cb6c4904d35b1f240a6861",
    ],
)

rpm(
    name = "virtiofsd-0__1.11.1-1.el9.s390x",
    sha256 = "241c88bc2b4d1370cf215872006862af6ac086d59be787dc315b8d841dff2640",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/virtiofsd-1.11.1-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/241c88bc2b4d1370cf215872006862af6ac086d59be787dc315b8d841dff2640",
    ],
)

rpm(
    name = "virtiofsd-0__1.11.1-1.el9.x86_64",
    sha256 = "5e2d3fddb1d18192a4d7bddeaef8b57cae16e53ac79b508eaf0958cbf42a8dce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/virtiofsd-1.11.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5e2d3fddb1d18192a4d7bddeaef8b57cae16e53ac79b508eaf0958cbf42a8dce",
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

rpm(
    name = "zstd-0__1.5.1-2.el9.aarch64",
    sha256 = "9434bdf20865fd2539b02c3813e94e46a70c7bc1fdb2e1cd8dfe112d16aad770",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/zstd-1.5.1-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9434bdf20865fd2539b02c3813e94e46a70c7bc1fdb2e1cd8dfe112d16aad770",
    ],
)

rpm(
    name = "zstd-0__1.5.1-2.el9.s390x",
    sha256 = "5dc369105f1d0c212356b8c1501830304acb81bd03969d2f400a1e1502eab279",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/zstd-1.5.1-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5dc369105f1d0c212356b8c1501830304acb81bd03969d2f400a1e1502eab279",
    ],
)

rpm(
    name = "zstd-0__1.5.1-2.el9.x86_64",
    sha256 = "a658684608dc353b9b5b3749315391ad7abc17b15f53c5576663ea8515691266",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/zstd-1.5.1-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a658684608dc353b9b5b3749315391ad7abc17b15f53c5576663ea8515691266",
    ],
)
