workspace(name = "kubevirt")

# register crosscompiler toolchains
load("//bazel/toolchain:toolchain.bzl", "register_all_toolchains")

register_all_toolchains()

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)

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
    sha256 = "7f85b688a4b558e2d9099340cfb510ba7179f829454fba842370bccffb67d6cc",
    strip_prefix = "buildifier-prebuilt-7.3.1",
    urls = [
        "http://github.com/keith/buildifier-prebuilt/archive/7.3.1.tar.gz",
        "https://storage.googleapis.com/builddeps/7f85b688a4b558e2d9099340cfb510ba7179f829454fba842370bccffb67d6cc",
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

load("@buildifier_prebuilt//:defs.bzl", "buildifier_prebuilt_register_toolchains", "buildtools_assets")

buildifier_prebuilt_register_toolchains(
    assets = buildtools_assets(
        arches = [
            "amd64",
            "arm64",
            "s390x",
        ],
        names = [
            "buildifier",
            "buildozer",
        ],
        platforms = [
            "darwin",
            "linux",
            "windows",
        ],
        sha256_values = {
            "buildifier_darwin_amd64": "375f823103d01620aaec20a0c29c6cbca99f4fd0725ae30b93655c6704f44d71",
            "buildifier_darwin_arm64": "5a6afc6ac7a09f5455ba0b89bd99d5ae23b4174dc5dc9d6c0ed5ce8caac3f813",
            "buildifier_linux_amd64": "5474cc5128a74e806783d54081f581662c4be8ae65022f557e9281ed5dc88009",
            "buildifier_linux_arm64": "0bf86c4bfffaf4f08eed77bde5b2082e4ae5039a11e2e8b03984c173c34a561c",
            "buildifier_linux_s390x": "e2d79ff5885d45274f76531f1adbc7b73a129f59e767f777e8fbde633d9d4e2e",
            "buildifier_windows_amd64": "370cd576075ad29930a82f5de132f1a1de4084c784a82514bd4da80c85acf4a8",
            "buildozer_darwin_amd64": "854c9583efc166602276802658cef3f224d60898cfaa60630b33d328db3b0de2",
            "buildozer_darwin_arm64": "31b1bfe20d7d5444be217af78f94c5c43799cdf847c6ce69794b7bf3319c5364",
            "buildozer_linux_amd64": "3305e287b3fcc68b9a35fd8515ee617452cd4e018f9e6886b6c7cdbcba8710d4",
            "buildozer_linux_arm64": "0b5a2a717ac4fc911e1fec8d92af71dbb4fe95b10e5213da0cc3d56cea64a328",
            "buildozer_linux_s390x": "7e28da8722656e800424989f5cdbc095cb29b2d398d33e6b3d04e0f50bc0bb10",
            "buildozer_windows_amd64": "58d41ce53257c5594c9bc86d769f580909269f68de114297f46284fbb9023dcf",
        },
        version = "v7.3.1",
    ),
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

load("@bazeldnf//:deps.bzl", "bazeldnf_dependencies", "rpm")
load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.23.9",
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
    sum = "h1:bs/cUb4lp1G5iImFFd3u5ixQzweKizoZJAwBNLR42lc=",
    version = "v1.65.0",
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
    digest = "sha256:0ba6aa6b538aeae3d0f716ea8837703eb147173cd673241662e89adb794da829",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
)

container_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:9ee08ca352647dad1511153afb18f4a6dbb4f56bafc7d618d0082c16a14cfdf1",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
)

container_pull(
    name = "go_image_base_s390x",
    digest = "sha256:6e2e356c462d69668a0313bf45ed3de614e9d4e0b9c03fa081d3bcae143a58ba",
    registry = "gcr.io",
    repository = "distroless/base-debian12",
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

container_pull(
    name = "busybox",
    digest = "sha256:545e6a6310a27636260920bc07b994a299b6708a1b26910cfefd335fdfb60d2b",
    registry = "registry.k8s.io",
    repository = "busybox",
    tag = "1.27",
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
    name = "libguestfs-appliance-x86_64",
    sha256 = "fb6da700eeae24da89aae6516091f7c5f46958b0b7812d2b122dc11dca1ab26a",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/libguestfs-appliance-1.54.0-qcow2-linux-5.14.0-575-centos9-amd64.tar.xz",
    ],
)

http_archive(
    name = "libguestfs-appliance-s390x",
    sha256 = "532cb951d4245265da645c8cce14033c19ea8f0d163c01e88f4153dae44e0f95",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/libguestfs-appliance-1.54.0-qcow2-linux-5.14.0-575-centos9-s390x.tar.xz",
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
    name = "adobe-source-code-pro-fonts-0__2.030.1.050-12.el9.1.s390x",
    sha256 = "9e6aa0c60204bb4b152ce541ca3a9f5c28b020ed551dd417d3936a8b2153f0df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/adobe-source-code-pro-fonts-2.030.1.050-12.el9.1.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9e6aa0c60204bb4b152ce541ca3a9f5c28b020ed551dd417d3936a8b2153f0df",
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
    name = "alternatives-0__1.24-2.el9.aarch64",
    sha256 = "3b8d0d6154ccc1047474072afc94cc1f72b7c234d8cd4e50734c67ca67da4161",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/alternatives-1.24-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3b8d0d6154ccc1047474072afc94cc1f72b7c234d8cd4e50734c67ca67da4161",
    ],
)

rpm(
    name = "alternatives-0__1.24-2.el9.s390x",
    sha256 = "8eb7ef117114059c44818eec88c4ed06c271a1185be1b1178ad096adcc934f11",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/alternatives-1.24-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8eb7ef117114059c44818eec88c4ed06c271a1185be1b1178ad096adcc934f11",
    ],
)

rpm(
    name = "alternatives-0__1.24-2.el9.x86_64",
    sha256 = "1e9effe6f59312207b55f87eaded01e8f238622ad14018ffd33ef49e9ce8d4c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/alternatives-1.24-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e9effe6f59312207b55f87eaded01e8f238622ad14018ffd33ef49e9ce8d4c6",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-4.el9.x86_64",
    sha256 = "d1482f65e84e761f0282e9e2c2a7111f0638dc889d6f34e4cde160e465855d1e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.1.5-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d1482f65e84e761f0282e9e2c2a7111f0638dc889d6f34e4cde160e465855d1e",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-7.el9.aarch64",
    sha256 = "0687c4b4d23ad6219bb3d557266b0adad1e9a24c8061e2579bcc285cc53cf106",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/audit-libs-3.1.5-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0687c4b4d23ad6219bb3d557266b0adad1e9a24c8061e2579bcc285cc53cf106",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-7.el9.s390x",
    sha256 = "441a308a28b5abdb631284dd849a2a1101d7b1e5034c50db1587e302157cbe48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/audit-libs-3.1.5-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/441a308a28b5abdb631284dd849a2a1101d7b1e5034c50db1587e302157cbe48",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-7.el9.x86_64",
    sha256 = "7d503abfc0f88258b39d518ae9a1e8c25af4c77370c784f37553a2dd18c222e5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.1.5-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d503abfc0f88258b39d518ae9a1e8c25af4c77370c784f37553a2dd18c222e5",
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
    name = "augeas-libs-0__1.14.1-3.el9.s390x",
    sha256 = "79f80d96c84bcfc0379638eb82862d6217052e8e72cd188b3544c2f7fc059bcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/augeas-libs-1.14.1-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/79f80d96c84bcfc0379638eb82862d6217052e8e72cd188b3544c2f7fc059bcc",
    ],
)

rpm(
    name = "augeas-libs-0__1.14.1-3.el9.x86_64",
    sha256 = "3db7a360240d905fa0dda490ac8f00f28553299087dc31a18c9e671616889553",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/augeas-libs-1.14.1-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3db7a360240d905fa0dda490ac8f00f28553299087dc31a18c9e671616889553",
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
    name = "binutils-0__2.35.2-65.el9.aarch64",
    sha256 = "4fc2c69ba884ae4104c3fa8ac5a5b44257d5297c4a8e2e4715ea0b3dda9777fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-65.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4fc2c69ba884ae4104c3fa8ac5a5b44257d5297c4a8e2e4715ea0b3dda9777fd",
    ],
)

rpm(
    name = "binutils-0__2.35.2-65.el9.s390x",
    sha256 = "10dc06f47e492aeb588a63ccccfbffc097030a6f0e4b34eeac666e81c52dd384",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-2.35.2-65.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/10dc06f47e492aeb588a63ccccfbffc097030a6f0e4b34eeac666e81c52dd384",
    ],
)

rpm(
    name = "binutils-0__2.35.2-65.el9.x86_64",
    sha256 = "78b3a17cfdb9aefdef998231c37d6c67d190d0282e3c681b24f3028ca14c245b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-65.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/78b3a17cfdb9aefdef998231c37d6c67d190d0282e3c681b24f3028ca14c245b",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-65.el9.aarch64",
    sha256 = "8515d890626ac5311d3a264c3e8b52bf936b1bf3acb08c88f4f1f203f3cf416c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-65.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8515d890626ac5311d3a264c3e8b52bf936b1bf3acb08c88f4f1f203f3cf416c",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-65.el9.s390x",
    sha256 = "417a93b84a961a990cdbc3b4939c2fdd63b7307c8d13b2477a30d58ce292fd12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-gold-2.35.2-65.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/417a93b84a961a990cdbc3b4939c2fdd63b7307c8d13b2477a30d58ce292fd12",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-65.el9.x86_64",
    sha256 = "14f6b09d6610f015fd636aac241e5f1385ce64054d3487d5b23aef2587f1a456",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-65.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/14f6b09d6610f015fd636aac241e5f1385ce64054d3487d5b23aef2587f1a456",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-10.el9.aarch64",
    sha256 = "79f097e912369d002db05995f4ba7b47f83a4fd2c9b5d6b6640066e1961f0f83",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.8-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/79f097e912369d002db05995f4ba7b47f83a4fd2c9b5d6b6640066e1961f0f83",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-10.el9.s390x",
    sha256 = "affd546407a3872a8db4fb0bc98c6c7aa46f59277be6f6bc8097f1709f8ec3d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-1.0.8-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/affd546407a3872a8db4fb0bc98c6c7aa46f59277be6f6bc8097f1709f8ec3d0",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-10.el9.x86_64",
    sha256 = "930b323ac8a0fc2357baecddc71d0fa1ea6cbae19d2ac61667aef19ed25d088e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.8-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/930b323ac8a0fc2357baecddc71d0fa1ea6cbae19d2ac61667aef19ed25d088e",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-10.el9.aarch64",
    sha256 = "065787a932991bd8e7a705d8a977658cafab06f78cf2e405b68978a02718998e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.8-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/065787a932991bd8e7a705d8a977658cafab06f78cf2e405b68978a02718998e",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-10.el9.s390x",
    sha256 = "26d36d213959fba230d4c8550410d66e04b279ac8ccee7b8600680a87dde2d73",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-libs-1.0.8-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/26d36d213959fba230d4c8550410d66e04b279ac8ccee7b8600680a87dde2d73",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-10.el9.x86_64",
    sha256 = "84392815cc1a8f01c651edd17f570aa449ef6f397ae48d773d655606ea7b4c96",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.8-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84392815cc1a8f01c651edd17f570aa449ef6f397ae48d773d655606ea7b4c96",
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
    name = "centos-gpg-keys-0__9.0-26.el9.x86_64",
    sha256 = "8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-26.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-28.el9.aarch64",
    sha256 = "f5e8879dfa34b2d2ec4146d9f4b55889e3d51af2195474dec520ca68c578266e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f5e8879dfa34b2d2ec4146d9f4b55889e3d51af2195474dec520ca68c578266e",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-28.el9.s390x",
    sha256 = "f5e8879dfa34b2d2ec4146d9f4b55889e3d51af2195474dec520ca68c578266e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-gpg-keys-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f5e8879dfa34b2d2ec4146d9f4b55889e3d51af2195474dec520ca68c578266e",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-28.el9.x86_64",
    sha256 = "f5e8879dfa34b2d2ec4146d9f4b55889e3d51af2195474dec520ca68c578266e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f5e8879dfa34b2d2ec4146d9f4b55889e3d51af2195474dec520ca68c578266e",
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
    name = "centos-stream-release-0__9.0-28.el9.aarch64",
    sha256 = "634026902cc4bd1dddfed59084897bc462afb46fb070d5cc0ed7985cce905a34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-release-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/634026902cc4bd1dddfed59084897bc462afb46fb070d5cc0ed7985cce905a34",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-28.el9.s390x",
    sha256 = "634026902cc4bd1dddfed59084897bc462afb46fb070d5cc0ed7985cce905a34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-release-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/634026902cc4bd1dddfed59084897bc462afb46fb070d5cc0ed7985cce905a34",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-28.el9.x86_64",
    sha256 = "634026902cc4bd1dddfed59084897bc462afb46fb070d5cc0ed7985cce905a34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-release-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/634026902cc4bd1dddfed59084897bc462afb46fb070d5cc0ed7985cce905a34",
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
    name = "centos-stream-repos-0__9.0-28.el9.aarch64",
    sha256 = "36f90791ba4dc3b160ec9129dbc141b9705081b88aee20a1f1cb883e22d0a584",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/36f90791ba4dc3b160ec9129dbc141b9705081b88aee20a1f1cb883e22d0a584",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-28.el9.s390x",
    sha256 = "36f90791ba4dc3b160ec9129dbc141b9705081b88aee20a1f1cb883e22d0a584",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-repos-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/36f90791ba4dc3b160ec9129dbc141b9705081b88aee20a1f1cb883e22d0a584",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-28.el9.x86_64",
    sha256 = "36f90791ba4dc3b160ec9129dbc141b9705081b88aee20a1f1cb883e22d0a584",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-28.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/36f90791ba4dc3b160ec9129dbc141b9705081b88aee20a1f1cb883e22d0a584",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-39.el9.aarch64",
    sha256 = "ff8039cbb4fc624462abb4f556535fff128c99685834f6137db564d1b5a24c95",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.32-39.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ff8039cbb4fc624462abb4f556535fff128c99685834f6137db564d1b5a24c95",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-39.el9.s390x",
    sha256 = "33f20a9d1a8dcbe9b6e587bda728a91b5b014cd0b0c979c7908135c5fed23115",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/coreutils-single-8.32-39.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/33f20a9d1a8dcbe9b6e587bda728a91b5b014cd0b0c979c7908135c5fed23115",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-39.el9.x86_64",
    sha256 = "09f7d8250c478a2931678063068adb8fccd2048d29fe9df31ca4e12c68f2ec7a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.32-39.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09f7d8250c478a2931678063068adb8fccd2048d29fe9df31ca4e12c68f2ec7a",
    ],
)

rpm(
    name = "cpp-0__11.5.0-7.el9.aarch64",
    sha256 = "8ce69c9990eada0af6c37884c95a2902dba420ebd5140fcdf160ee4fdc32a57a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8ce69c9990eada0af6c37884c95a2902dba420ebd5140fcdf160ee4fdc32a57a",
    ],
)

rpm(
    name = "cpp-0__11.5.0-7.el9.s390x",
    sha256 = "085230f3cb1dac40967680ca1ce61155b81288bf4560a8feebe21ada01959199",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/cpp-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/085230f3cb1dac40967680ca1ce61155b81288bf4560a8feebe21ada01959199",
    ],
)

rpm(
    name = "cpp-0__11.5.0-7.el9.x86_64",
    sha256 = "642e6ddea2f3d317a18d1dc03262f05947e19ed14cbb416b25ae55b378187779",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.5.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/642e6ddea2f3d317a18d1dc03262f05947e19ed14cbb416b25ae55b378187779",
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
    name = "crypto-policies-0__20250128-1.git5269e22.el9.x86_64",
    sha256 = "f811d2c848f6f93a188f2d74d4ccd172e1dc88fa7919e8e203cf1df3d93571e1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20250128-1.git5269e22.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f811d2c848f6f93a188f2d74d4ccd172e1dc88fa7919e8e203cf1df3d93571e1",
    ],
)

rpm(
    name = "crypto-policies-0__20250602-1.gita839241.el9.aarch64",
    sha256 = "e8b48dd9a5763141aa14e1d04132935964adbf779cc877beef164e1e6890c848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20250602-1.gita839241.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e8b48dd9a5763141aa14e1d04132935964adbf779cc877beef164e1e6890c848",
    ],
)

rpm(
    name = "crypto-policies-0__20250602-1.gita839241.el9.s390x",
    sha256 = "e8b48dd9a5763141aa14e1d04132935964adbf779cc877beef164e1e6890c848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/crypto-policies-20250602-1.gita839241.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e8b48dd9a5763141aa14e1d04132935964adbf779cc877beef164e1e6890c848",
    ],
)

rpm(
    name = "crypto-policies-0__20250602-1.gita839241.el9.x86_64",
    sha256 = "e8b48dd9a5763141aa14e1d04132935964adbf779cc877beef164e1e6890c848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20250602-1.gita839241.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e8b48dd9a5763141aa14e1d04132935964adbf779cc877beef164e1e6890c848",
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
    name = "device-mapper-9__1.02.202-6.el9.x86_64",
    sha256 = "0bf0cd224f72b8c6f3747d8c8d053418b13ff819601cc1293233b31e3a01998b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.202-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0bf0cd224f72b8c6f3747d8c8d053418b13ff819601cc1293233b31e3a01998b",
    ],
)

rpm(
    name = "device-mapper-9__1.02.206-2.el9.aarch64",
    sha256 = "3e2ab355c84e3c552f0c801bcf2abfa52d303c8362e64ee1a1b8349ff5de4e58",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.206-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3e2ab355c84e3c552f0c801bcf2abfa52d303c8362e64ee1a1b8349ff5de4e58",
    ],
)

rpm(
    name = "device-mapper-9__1.02.206-2.el9.s390x",
    sha256 = "ba1c1a6f529a1700b2c7257baf88ab421770ea2c760889bc74d1e0372ecf6ea8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-1.02.206-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ba1c1a6f529a1700b2c7257baf88ab421770ea2c760889bc74d1e0372ecf6ea8",
    ],
)

rpm(
    name = "device-mapper-9__1.02.206-2.el9.x86_64",
    sha256 = "dca7b6ad60c556111c6a2ab198fe7fa802cafb2733183cce831135fedad8a7e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.206-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dca7b6ad60c556111c6a2ab198fe7fa802cafb2733183cce831135fedad8a7e0",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.202-6.el9.x86_64",
    sha256 = "8efb6c63cb8dfa44329e6f47cc1f4d97f727ea1b21c619a8cc1244769e692af9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.202-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8efb6c63cb8dfa44329e6f47cc1f4d97f727ea1b21c619a8cc1244769e692af9",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.206-2.el9.aarch64",
    sha256 = "935550e46fcdabb578ad5373e12f81ed0cadde85b5d7522b1e0d7171eb73c9de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.206-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/935550e46fcdabb578ad5373e12f81ed0cadde85b5d7522b1e0d7171eb73c9de",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.206-2.el9.s390x",
    sha256 = "61827b1539c7db88538c58fba2a0b95c8cc5ed0f1a696ab26b6acab4cf5c4e67",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-libs-1.02.206-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/61827b1539c7db88538c58fba2a0b95c8cc5ed0f1a696ab26b6acab4cf5c4e67",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.206-2.el9.x86_64",
    sha256 = "2b6ab8f98bed43e79f71d4ac2aeb2ff4d303497f04addbfff74bd2dd6ad71796",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.206-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2b6ab8f98bed43e79f71d4ac2aeb2ff4d303497f04addbfff74bd2dd6ad71796",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-38.el9.aarch64",
    sha256 = "7554581d114dbac6e465822cb1b4b90a400ca10f4d7f2d6e1c8580ac46483cf3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.7-38.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7554581d114dbac6e465822cb1b4b90a400ca10f4d7f2d6e1c8580ac46483cf3",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-38.el9.x86_64",
    sha256 = "c12f768f16c6e2a0c840cda64a6e751f9ecebcde65d8bc2e2fc71712e94ec4c8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.7-38.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c12f768f16c6e2a0c840cda64a6e751f9ecebcde65d8bc2e2fc71712e94ec4c8",
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
    name = "dmidecode-1__3.6-1.el9.aarch64",
    sha256 = "6cacf42907aaa5bbad69c2ff24eff8b09a1d007a1e630f4b670edb97bbc29bf0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dmidecode-3.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6cacf42907aaa5bbad69c2ff24eff8b09a1d007a1e630f4b670edb97bbc29bf0",
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
    name = "e2fsprogs-0__1.46.5-7.el9.aarch64",
    sha256 = "5299eeb33ca510c243919c17865703cf139ce92a90c6fc21face49754a529ca1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.46.5-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5299eeb33ca510c243919c17865703cf139ce92a90c6fc21face49754a529ca1",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-7.el9.s390x",
    sha256 = "b6175e4a475e31020b5f07de98898c24ca98174e298c65c75837ebd8b33908b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/e2fsprogs-1.46.5-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b6175e4a475e31020b5f07de98898c24ca98174e298c65c75837ebd8b33908b1",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-7.el9.x86_64",
    sha256 = "b9f41fb4f5f208f772cd8c3b8769b11ca1d6c17dfe39da3fba656b49f4e8ebe3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.46.5-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b9f41fb4f5f208f772cd8c3b8769b11ca1d6c17dfe39da3fba656b49f4e8ebe3",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-7.el9.aarch64",
    sha256 = "fd899d06c72a77b984e149c1ecc2a78335773766e303053b1d36ebd090e25ccf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.46.5-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fd899d06c72a77b984e149c1ecc2a78335773766e303053b1d36ebd090e25ccf",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-7.el9.s390x",
    sha256 = "cd513f65144483060aae16ca09f4faef36e53729e79d701c30b9267463fb66fe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/e2fsprogs-libs-1.46.5-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cd513f65144483060aae16ca09f4faef36e53729e79d701c30b9267463fb66fe",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-7.el9.x86_64",
    sha256 = "19171a686f1faa6be6a61301a79dff144a74fcd0e5596b17c48f0e4ab081a601",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.46.5-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19171a686f1faa6be6a61301a79dff144a74fcd0e5596b17c48f0e4ab081a601",
    ],
)

rpm(
    name = "edk2-aarch64-0__20241117-3.el9.aarch64",
    sha256 = "5f4f2cf0a8c1271bf32d6534cacaa036d770953a797c82f4a3ab6aae14350a13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20241117-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5f4f2cf0a8c1271bf32d6534cacaa036d770953a797c82f4a3ab6aae14350a13",
    ],
)

rpm(
    name = "edk2-ovmf-0__20241117-2.el9.x86_64",
    sha256 = "a64ed00fed189c823f533a013ce8f044a439066524fbb628b266fd898fe23172",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20241117-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a64ed00fed189c823f533a013ce8f044a439066524fbb628b266fd898fe23172",
    ],
)

rpm(
    name = "edk2-ovmf-0__20241117-3.el9.s390x",
    sha256 = "312b99d64803997220502e652575eceae37bae23dfa42b92dfa393c018b60676",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/edk2-ovmf-20241117-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/312b99d64803997220502e652575eceae37bae23dfa42b92dfa393c018b60676",
    ],
)

rpm(
    name = "edk2-ovmf-0__20241117-3.el9.x86_64",
    sha256 = "312b99d64803997220502e652575eceae37bae23dfa42b92dfa393c018b60676",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20241117-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/312b99d64803997220502e652575eceae37bae23dfa42b92dfa393c018b60676",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.193-1.el9.aarch64",
    sha256 = "e72173a25681d4133d15770c263d1bbfcff8ae08a0b606e4af71ed0fb936d678",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.193-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e72173a25681d4133d15770c263d1bbfcff8ae08a0b606e4af71ed0fb936d678",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.193-1.el9.s390x",
    sha256 = "04a806ba027541ff665fa3bd39ec10ee0dedf8c85f3bd76f927450574fa9ca18",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.193-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/04a806ba027541ff665fa3bd39ec10ee0dedf8c85f3bd76f927450574fa9ca18",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.193-1.el9.x86_64",
    sha256 = "ed09caa612b06b4f839ec0650d43724ce1f41cbe0a19a289bc91e4d4571c1a10",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.193-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed09caa612b06b4f839ec0650d43724ce1f41cbe0a19a289bc91e4d4571c1a10",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.193-1.el9.aarch64",
    sha256 = "0d8e80edd33e4029c2d8bdfa451ddf49854cc127fa1d59bb158b6b8314a59b6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.193-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0d8e80edd33e4029c2d8bdfa451ddf49854cc127fa1d59bb158b6b8314a59b6f",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.193-1.el9.s390x",
    sha256 = "0d8e80edd33e4029c2d8bdfa451ddf49854cc127fa1d59bb158b6b8314a59b6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.193-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0d8e80edd33e4029c2d8bdfa451ddf49854cc127fa1d59bb158b6b8314a59b6f",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.193-1.el9.x86_64",
    sha256 = "0d8e80edd33e4029c2d8bdfa451ddf49854cc127fa1d59bb158b6b8314a59b6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.193-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0d8e80edd33e4029c2d8bdfa451ddf49854cc127fa1d59bb158b6b8314a59b6f",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.192-5.el9.x86_64",
    sha256 = "be527a162e856c28841d407aa2b4845ef1095f6730f71602da3782009f956ba5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.192-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be527a162e856c28841d407aa2b4845ef1095f6730f71602da3782009f956ba5",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.193-1.el9.aarch64",
    sha256 = "80458e8bb299d3f73ce8078bfb4f16cf147e69bf59662d24a2f0d15a27f11f4e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.193-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/80458e8bb299d3f73ce8078bfb4f16cf147e69bf59662d24a2f0d15a27f11f4e",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.193-1.el9.s390x",
    sha256 = "7a68c24aec7075d3d260470a4a4e9aa09a684d16a029f5591a66a223adcdd794",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.193-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7a68c24aec7075d3d260470a4a4e9aa09a684d16a029f5591a66a223adcdd794",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.193-1.el9.x86_64",
    sha256 = "9f5169441d8203e1db66199c64a7fbc06b37fb4cc9d5356117262d292f933d12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.193-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f5169441d8203e1db66199c64a7fbc06b37fb4cc9d5356117262d292f933d12",
    ],
)

rpm(
    name = "elfutils-libs-0__0.193-1.el9.aarch64",
    sha256 = "955844015d2f660b66eba54ad4dcf47b848eb6e57a1e60f96cb721be4d1faeb9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.193-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/955844015d2f660b66eba54ad4dcf47b848eb6e57a1e60f96cb721be4d1faeb9",
    ],
)

rpm(
    name = "elfutils-libs-0__0.193-1.el9.s390x",
    sha256 = "772735b1002acc39f008f21b99886ce45ea0918b3bdb42b40312df54779385c5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.193-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/772735b1002acc39f008f21b99886ce45ea0918b3bdb42b40312df54779385c5",
    ],
)

rpm(
    name = "elfutils-libs-0__0.193-1.el9.x86_64",
    sha256 = "83577daeda825a338d8b1285048b0b55dd99a4a3d4f935c866e9ffc1cb8639dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.193-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/83577daeda825a338d8b1285048b0b55dd99a4a3d4f935c866e9ffc1cb8639dc",
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
    name = "expat-0__2.5.0-4.el9.x86_64",
    sha256 = "360ed994ea2af5b3a7f37694dfdf2249d97e5e5ec2492c9223a2aec72ff8f480",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.5.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/360ed994ea2af5b3a7f37694dfdf2249d97e5e5ec2492c9223a2aec72ff8f480",
    ],
)

rpm(
    name = "expat-0__2.5.0-5.el9.aarch64",
    sha256 = "8d533bfa2656c45a8d159314e9447492442db5705a1cd73033b8dad720f33b46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/expat-2.5.0-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8d533bfa2656c45a8d159314e9447492442db5705a1cd73033b8dad720f33b46",
    ],
)

rpm(
    name = "expat-0__2.5.0-5.el9.s390x",
    sha256 = "3ea924e01856fa990e48e54665947f51630411ff1b71edd7f65a198e1a9d3f1f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/expat-2.5.0-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3ea924e01856fa990e48e54665947f51630411ff1b71edd7f65a198e1a9d3f1f",
    ],
)

rpm(
    name = "expat-0__2.5.0-5.el9.x86_64",
    sha256 = "fe45d00a4d532178c552cd62b49e9d56514afc6ef29403eb1625340ab173d163",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe45d00a4d532178c552cd62b49e9d56514afc6ef29403eb1625340ab173d163",
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
    name = "fonts-filesystem-1__2.0.5-7.el9.1.s390x",
    sha256 = "c79fa96aa7fb447975497dd50c94002ee73d01171343f8ee14032d06adb58a92",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/fonts-filesystem-2.0.5-7.el9.1.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c79fa96aa7fb447975497dd50c94002ee73d01171343f8ee14032d06adb58a92",
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
    name = "fuse-0__2.9.9-17.el9.s390x",
    sha256 = "4f5532023b6272eb79706c080fce40a5f083398820bfbbfaa7116243c6a93bc0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/fuse-2.9.9-17.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4f5532023b6272eb79706c080fce40a5f083398820bfbbfaa7116243c6a93bc0",
    ],
)

rpm(
    name = "fuse-0__2.9.9-17.el9.x86_64",
    sha256 = "8cb98fe8a2bd6f4c39661c12f0daccae258acadcf3d444136c517fe2f46c421c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-2.9.9-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8cb98fe8a2bd6f4c39661c12f0daccae258acadcf3d444136c517fe2f46c421c",
    ],
)

rpm(
    name = "fuse-common-0__3.10.2-9.el9.s390x",
    sha256 = "18de6b2985152ae3b3f1e72d90591543362c09e71ccb749a3adb63099c37496e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/fuse-common-3.10.2-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/18de6b2985152ae3b3f1e72d90591543362c09e71ccb749a3adb63099c37496e",
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
    name = "fuse-libs-0__2.9.9-17.el9.aarch64",
    sha256 = "5cfdb796cb825686e224aec5ab1752cccd7416b5078f860246e7210cdee0e57a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/fuse-libs-2.9.9-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5cfdb796cb825686e224aec5ab1752cccd7416b5078f860246e7210cdee0e57a",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-17.el9.s390x",
    sha256 = "89b568150669f246789540bb83b24db22821a1b5d761881e591a67643c2aaeaa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/fuse-libs-2.9.9-17.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/89b568150669f246789540bb83b24db22821a1b5d761881e591a67643c2aaeaa",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-17.el9.x86_64",
    sha256 = "a164f06f802c04e6d3091d57150362b26a5ec3ab85ac612fba5dc9a068e77ac5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.9-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a164f06f802c04e6d3091d57150362b26a5ec3ab85ac612fba5dc9a068e77ac5",
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
    name = "gcc-0__11.5.0-7.el9.aarch64",
    sha256 = "b5d8c5ae4d480c1617c5d44a8d3e6544ef2d0f5f33b6c62196b40939478fa0db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b5d8c5ae4d480c1617c5d44a8d3e6544ef2d0f5f33b6c62196b40939478fa0db",
    ],
)

rpm(
    name = "gcc-0__11.5.0-7.el9.s390x",
    sha256 = "7b7fb21f2676d83161d61ef1c2fd7011e1dd666ff3f2043f9556d4fba47ed509",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gcc-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7b7fb21f2676d83161d61ef1c2fd7011e1dd666ff3f2043f9556d4fba47ed509",
    ],
)

rpm(
    name = "gcc-0__11.5.0-7.el9.x86_64",
    sha256 = "2baecd53063896e5794ebe506d23b22581a8b110b78c397bc4ea51e1846f2a8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.5.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2baecd53063896e5794ebe506d23b22581a8b110b78c397bc4ea51e1846f2a8e",
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
    name = "glib-networking-0__2.68.3-3.el9.s390x",
    sha256 = "f5d013624d04c2f1ec232a59e46342b4c52688c29c2a43304e52456a63408667",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glib-networking-2.68.3-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f5d013624d04c2f1ec232a59e46342b4c52688c29c2a43304e52456a63408667",
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
    name = "glibc-0__2.34-168.el9.x86_64",
    sha256 = "e06212b1cac1d9fd9857a00ddefefe9fb9f406199cb84fdd1153589c15e16289",
    urls = ["https://storage.googleapis.com/builddeps/e06212b1cac1d9fd9857a00ddefefe9fb9f406199cb84fdd1153589c15e16289"],
)

rpm(
    name = "glibc-0__2.34-203.el9.aarch64",
    sha256 = "173e6f4c511ff7c296f788052d78245c6c358c89935860d165e88bb10607ac06",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-203.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/173e6f4c511ff7c296f788052d78245c6c358c89935860d165e88bb10607ac06",
    ],
)

rpm(
    name = "glibc-0__2.34-203.el9.s390x",
    sha256 = "0ee47ea03628e41f05bd7b1b085468911460aa1852b83adb7e7c73fc0d8fbbaf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-2.34-203.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0ee47ea03628e41f05bd7b1b085468911460aa1852b83adb7e7c73fc0d8fbbaf",
    ],
)

rpm(
    name = "glibc-0__2.34-203.el9.x86_64",
    sha256 = "ca78283ae06b9c3df2ebb2e0b1c46bfa00019526890944429cf81b8d7199ed29",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-203.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ca78283ae06b9c3df2ebb2e0b1c46bfa00019526890944429cf81b8d7199ed29",
    ],
)

rpm(
    name = "glibc-common-0__2.34-168.el9.x86_64",
    sha256 = "531650744909efd0284bf6c16a45dbaf455b214c0cac4197cf6d43e8c7d83af8",
    urls = ["https://storage.googleapis.com/builddeps/531650744909efd0284bf6c16a45dbaf455b214c0cac4197cf6d43e8c7d83af8"],
)

rpm(
    name = "glibc-common-0__2.34-203.el9.aarch64",
    sha256 = "373409395084a742bd48567bde085b91f771355a97651b2097d4ab6850e1ed0b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-203.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/373409395084a742bd48567bde085b91f771355a97651b2097d4ab6850e1ed0b",
    ],
)

rpm(
    name = "glibc-common-0__2.34-203.el9.s390x",
    sha256 = "fc9bf3b70ce13f597bb9a477d002d803e68d38d4334b859bd119d794565ed9fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-common-2.34-203.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fc9bf3b70ce13f597bb9a477d002d803e68d38d4334b859bd119d794565ed9fd",
    ],
)

rpm(
    name = "glibc-common-0__2.34-203.el9.x86_64",
    sha256 = "d03c40e93f30c7d44d3da3bf1d9833a3bccb533f5cfc4e53597c6521a6b8fd1e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-203.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d03c40e93f30c7d44d3da3bf1d9833a3bccb533f5cfc4e53597c6521a6b8fd1e",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-203.el9.aarch64",
    sha256 = "0bdf242ec7ae9c33a21e79bd74714416a923610d94f4746d1337b56ce216c2d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-203.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0bdf242ec7ae9c33a21e79bd74714416a923610d94f4746d1337b56ce216c2d2",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-203.el9.s390x",
    sha256 = "f27e35ad0418d514a7baf5644e4c4228957da6b81dc400fcba6d07f3c1045ef5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-devel-2.34-203.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f27e35ad0418d514a7baf5644e4c4228957da6b81dc400fcba6d07f3c1045ef5",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-203.el9.x86_64",
    sha256 = "9444f02bed16c6edac805c9e4732f8cffaa8d6f160ffb14bf095676f86fd53c1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-203.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9444f02bed16c6edac805c9e4732f8cffaa8d6f160ffb14bf095676f86fd53c1",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-203.el9.s390x",
    sha256 = "c325ff399d9b3132e7e51ce40d7f1d0deeb1c27ccb84b0022af5e9d97c64b40b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-headers-2.34-203.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c325ff399d9b3132e7e51ce40d7f1d0deeb1c27ccb84b0022af5e9d97c64b40b",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-203.el9.x86_64",
    sha256 = "0afa5ba83992f077033d463faa5ecaa56e6da7b58bdefa5195788a22462d6372",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-203.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0afa5ba83992f077033d463faa5ecaa56e6da7b58bdefa5195788a22462d6372",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-168.el9.x86_64",
    sha256 = "991b6d7370b237a3d576536a517d01a1ccc997959f4ea30ba07bd779641f79e8",
    urls = ["https://storage.googleapis.com/builddeps/991b6d7370b237a3d576536a517d01a1ccc997959f4ea30ba07bd779641f79e8"],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-203.el9.aarch64",
    sha256 = "7a12dde0886645daf0e1e89b135583dd4eb52c8f294eed1276a843330ef64a97",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-203.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7a12dde0886645daf0e1e89b135583dd4eb52c8f294eed1276a843330ef64a97",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-203.el9.s390x",
    sha256 = "be2737f724e55e20d4f7d007496b8ed5fcf74f1e29ecc1658df55d8830ff11dd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.34-203.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/be2737f724e55e20d4f7d007496b8ed5fcf74f1e29ecc1658df55d8830ff11dd",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-203.el9.x86_64",
    sha256 = "bb1501ed57c473aec53df74e8887a03c1e00cbdce75b7005d4a8b8d719f29afb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-203.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb1501ed57c473aec53df74e8887a03c1e00cbdce75b7005d4a8b8d719f29afb",
    ],
)

rpm(
    name = "glibc-static-0__2.34-203.el9.aarch64",
    sha256 = "ad3cc75e66ea8357d1e499cc647cfa6ab4fa51fd05e1ea58f26e89c805d747b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-203.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad3cc75e66ea8357d1e499cc647cfa6ab4fa51fd05e1ea58f26e89c805d747b4",
    ],
)

rpm(
    name = "glibc-static-0__2.34-203.el9.s390x",
    sha256 = "0df289255a83a49cb0b2066c9913607504f3e52d951ddd60187d35421bef5ddc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/glibc-static-2.34-203.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0df289255a83a49cb0b2066c9913607504f3e52d951ddd60187d35421bef5ddc",
    ],
)

rpm(
    name = "glibc-static-0__2.34-203.el9.x86_64",
    sha256 = "a8f4f75dad0142d5e687a61fd30f4e737ff82e9cc5079c28f4eebc45eb834ef8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-203.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a8f4f75dad0142d5e687a61fd30f4e737ff82e9cc5079c28f4eebc45eb834ef8",
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
    name = "gnupg2-0__2.3.3-4.el9.s390x",
    sha256 = "79f4d4ce2953babbca0f5ba558b633e8e2a03bb5745f6e9340dfe83c7181c782",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gnupg2-2.3.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/79f4d4ce2953babbca0f5ba558b633e8e2a03bb5745f6e9340dfe83c7181c782",
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
    name = "gnutls-0__3.8.3-6.el9.aarch64",
    sha256 = "98953b8f2b13767a37b90f5e055df8b26a73c98f6040d5f1b8985f3fbfe9e609",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gnutls-3.8.3-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98953b8f2b13767a37b90f5e055df8b26a73c98f6040d5f1b8985f3fbfe9e609",
    ],
)

rpm(
    name = "gnutls-0__3.8.3-6.el9.s390x",
    sha256 = "76b9181f48730d9abe4108991a5bbe3ad91025673a21adae930a564834063bad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gnutls-3.8.3-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/76b9181f48730d9abe4108991a5bbe3ad91025673a21adae930a564834063bad",
    ],
)

rpm(
    name = "gnutls-0__3.8.3-6.el9.x86_64",
    sha256 = "97364bd099856650cdbcc18448e85a3cc6a3cebc9513190a1b4d7016132920d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnutls-3.8.3-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/97364bd099856650cdbcc18448e85a3cc6a3cebc9513190a1b4d7016132920d9",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.3-6.el9.aarch64",
    sha256 = "bc002c311bc364a43159f3bdab6280f5ab108ed41fe23fdf4719f2ce6518df34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.8.3-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bc002c311bc364a43159f3bdab6280f5ab108ed41fe23fdf4719f2ce6518df34",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.3-6.el9.s390x",
    sha256 = "408bae554e7f9f0fa3d7e28df042961d66184075dcf5b1b3eed2d1ae445da4bf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-dane-3.8.3-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/408bae554e7f9f0fa3d7e28df042961d66184075dcf5b1b3eed2d1ae445da4bf",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.3-6.el9.x86_64",
    sha256 = "b56d21fc34a1e90ec841e1ce8f48cffde4bf94d5ad4944476ce0082e9dff1870",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.8.3-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b56d21fc34a1e90ec841e1ce8f48cffde4bf94d5ad4944476ce0082e9dff1870",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.3-6.el9.aarch64",
    sha256 = "ea3b210554cfc75d1d496a103f25695b77b7148db71f24cebb77c6ac07108815",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.8.3-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ea3b210554cfc75d1d496a103f25695b77b7148db71f24cebb77c6ac07108815",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.3-6.el9.s390x",
    sha256 = "38724642a093afc4c5bd031db3a31e2a4fe4396f3dc9ce2ca32b91f47ad23e38",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-utils-3.8.3-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/38724642a093afc4c5bd031db3a31e2a4fe4396f3dc9ce2ca32b91f47ad23e38",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.3-6.el9.x86_64",
    sha256 = "cf75261ee612e1abe9bb0463639e3aad4086649069e6ff9c15b955541279f95d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.8.3-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cf75261ee612e1abe9bb0463639e3aad4086649069e6ff9c15b955541279f95d",
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
    name = "gsettings-desktop-schemas-0__40.0-6.el9.s390x",
    sha256 = "1f45d7ed5d0853d202496cabf534de539879bb33e511ca3e8fc71044059e2847",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gsettings-desktop-schemas-40.0-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1f45d7ed5d0853d202496cabf534de539879bb33e511ca3e8fc71044059e2847",
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
    name = "guestfs-tools-0__1.52.2-2.el9.x86_64",
    sha256 = "2c3cfa7e3de1e97ff1a6465d2c776311bffded28d451d38983919c1a6cca8dd8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.52.2-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c3cfa7e3de1e97ff1a6465d2c776311bffded28d451d38983919c1a6cca8dd8",
    ],
)

rpm(
    name = "guestfs-tools-0__1.52.2-5.el9.s390x",
    sha256 = "09f07a0309459e9aeaa0d4004496a4bbbd10dc1ebf2ea0b97187a8a27020c08e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/guestfs-tools-1.52.2-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/09f07a0309459e9aeaa0d4004496a4bbbd10dc1ebf2ea0b97187a8a27020c08e",
    ],
)

rpm(
    name = "guestfs-tools-0__1.52.2-5.el9.x86_64",
    sha256 = "0d4ff23abb20c16245b8452201cd693aa864ba609f4fea36ba23612011c2a9bf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.52.2-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0d4ff23abb20c16245b8452201cd693aa864ba609f4fea36ba23612011c2a9bf",
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
    name = "hexedit-0__1.6-1.el9.s390x",
    sha256 = "d6a58dc3d17cad456ea20945a93ccca5bae70620a1d16ac059c9c8554337b33a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/hexedit-1.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d6a58dc3d17cad456ea20945a93ccca5bae70620a1d16ac059c9c8554337b33a",
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
    name = "hivex-libs-0__1.3.24-1.el9.s390x",
    sha256 = "1976d559fb2ad7ae330a7265d2ff03d59ca1726637db13a54bceba2be08d9920",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/hivex-libs-1.3.24-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1976d559fb2ad7ae330a7265d2ff03d59ca1726637db13a54bceba2be08d9920",
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
    name = "hwdata-0__0.348-9.18.el9.s390x",
    sha256 = "b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/hwdata-0.348-9.18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    ],
)

rpm(
    name = "hwdata-0__0.348-9.18.el9.x86_64",
    sha256 = "b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    ],
)

rpm(
    name = "iproute-0__6.11.0-1.el9.x86_64",
    sha256 = "3780635befbf4a3c3b8a1a52e6b9eb666b64574189be3b9b13624355dae4a8a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-6.11.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3780635befbf4a3c3b8a1a52e6b9eb666b64574189be3b9b13624355dae4a8a8",
    ],
)

rpm(
    name = "iproute-0__6.14.0-1.el9.aarch64",
    sha256 = "93d293ed98fd75878c6dfe2ffb8559abfe59d201b4f25eebbe073cf7bab3e7f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-6.14.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/93d293ed98fd75878c6dfe2ffb8559abfe59d201b4f25eebbe073cf7bab3e7f5",
    ],
)

rpm(
    name = "iproute-0__6.14.0-1.el9.s390x",
    sha256 = "514116754442a7730a4bd563cc0b60b5f232ce2fea601f9cdd21c20b27be0c17",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iproute-6.14.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/514116754442a7730a4bd563cc0b60b5f232ce2fea601f9cdd21c20b27be0c17",
    ],
)

rpm(
    name = "iproute-0__6.14.0-1.el9.x86_64",
    sha256 = "ee307ce7058e0fe5b7f3e810e203525a99e19b2e020f7a4b86271699685c2460",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-6.14.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ee307ce7058e0fe5b7f3e810e203525a99e19b2e020f7a4b86271699685c2460",
    ],
)

rpm(
    name = "iproute-tc-0__6.11.0-1.el9.x86_64",
    sha256 = "0dd645d098e02a1ebc31cbddc8d1cd6f36a3bd92190bb496b2cfc1e9849958ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.11.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0dd645d098e02a1ebc31cbddc8d1cd6f36a3bd92190bb496b2cfc1e9849958ed",
    ],
)

rpm(
    name = "iproute-tc-0__6.14.0-1.el9.aarch64",
    sha256 = "c1d71d57c395112bcb9c0d96b755bec86ce4698b14931d657f49e627718f9d72",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-tc-6.14.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c1d71d57c395112bcb9c0d96b755bec86ce4698b14931d657f49e627718f9d72",
    ],
)

rpm(
    name = "iproute-tc-0__6.14.0-1.el9.s390x",
    sha256 = "0bb6c2edfdf3907f08ab15b8263edfa1b4b8eef93bd5638a5e1b4d77b68e74e2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iproute-tc-6.14.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0bb6c2edfdf3907f08ab15b8263edfa1b4b8eef93bd5638a5e1b4d77b68e74e2",
    ],
)

rpm(
    name = "iproute-tc-0__6.14.0-1.el9.x86_64",
    sha256 = "fb42d0cea6be8f7767b20d9f792f9ed567e94198852f45071d1979ffe99aa177",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.14.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb42d0cea6be8f7767b20d9f792f9ed567e94198852f45071d1979ffe99aa177",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-11.el9.aarch64",
    sha256 = "097df125f6836f5dbdce2f3e961a649cd2e15b5f2a8164267c7c98b281ab60e4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.10-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/097df125f6836f5dbdce2f3e961a649cd2e15b5f2a8164267c7c98b281ab60e4",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-11.el9.s390x",
    sha256 = "469bd3ae07fb31f648a81d8ffa6b5053ee647b4c5dffcbcfbf11081921231715",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iptables-libs-1.8.10-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/469bd3ae07fb31f648a81d8ffa6b5053ee647b4c5dffcbcfbf11081921231715",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.10-11.el9.x86_64",
    sha256 = "7ffd51ff29c86e31d36ff9518dead9fd403034824e874b069a24c6587d4e1084",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.10-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7ffd51ff29c86e31d36ff9518dead9fd403034824e874b069a24c6587d4e1084",
    ],
)

rpm(
    name = "iputils-0__20210202-12.el9.aarch64",
    sha256 = "fd517c54fa048e9fd3614329e7c4fc1a31473bef72981afdaac8796feeafb98a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iputils-20210202-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fd517c54fa048e9fd3614329e7c4fc1a31473bef72981afdaac8796feeafb98a",
    ],
)

rpm(
    name = "iputils-0__20210202-12.el9.s390x",
    sha256 = "56b2186c787471ec5326a968709d3ee7d04ec329f172f082190614000858ff85",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iputils-20210202-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/56b2186c787471ec5326a968709d3ee7d04ec329f172f082190614000858ff85",
    ],
)

rpm(
    name = "iputils-0__20210202-12.el9.x86_64",
    sha256 = "53ef31b72b36cbc621bf78c2fbdb387e089bf26fea7bac6c50a59f365ec8fdb4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iputils-20210202-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/53ef31b72b36cbc621bf78c2fbdb387e089bf26fea7bac6c50a59f365ec8fdb4",
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
    name = "kernel-headers-0__5.14.0-592.el9.aarch64",
    sha256 = "2e31a848091e1d41dfb08cb6eb9ac5b2c7b34153f54d13e29ef8beaeecc586f2",
    urls = ["https://storage.googleapis.com/builddeps/2e31a848091e1d41dfb08cb6eb9ac5b2c7b34153f54d13e29ef8beaeecc586f2"],
)

rpm(
    name = "kernel-headers-0__5.14.0-592.el9.s390x",
    sha256 = "6d23d0c6671b20e3a89743bfe99ed7df184a9fc1acf752a02ad8a38981ff8de9",
    urls = ["https://storage.googleapis.com/builddeps/6d23d0c6671b20e3a89743bfe99ed7df184a9fc1acf752a02ad8a38981ff8de9"],
)

rpm(
    name = "kernel-headers-0__5.14.0-592.el9.x86_64",
    sha256 = "1730dd32b1fcf5fd51174ddb1d2c295bbdd9309b9ac2d310a533365fee0791cd",
    urls = ["https://storage.googleapis.com/builddeps/1730dd32b1fcf5fd51174ddb1d2c295bbdd9309b9ac2d310a533365fee0791cd"],
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
    name = "krb5-libs-0__1.21.1-6.el9.x86_64",
    sha256 = "50edf4089d0480048aeba2bfd736b93aa89dc25735cd02e80bad57e562e1e001",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.1-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/50edf4089d0480048aeba2bfd736b93aa89dc25735cd02e80bad57e562e1e001",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-8.el9.aarch64",
    sha256 = "7671147fe79cb1fa2fd011cee451be325a8e4aca6290ff67eb9e4d01f0c1edfd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.1-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7671147fe79cb1fa2fd011cee451be325a8e4aca6290ff67eb9e4d01f0c1edfd",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-8.el9.s390x",
    sha256 = "ea2d05d119e5b9072dd4c58d74079accfd24bae196d012e0de105ab3e0bcc892",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/krb5-libs-1.21.1-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ea2d05d119e5b9072dd4c58d74079accfd24bae196d012e0de105ab3e0bcc892",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-8.el9.x86_64",
    sha256 = "d5d3473637d3453c3216b1f14fef500e10bd8158c99bbadb48b788b88d92786f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.1-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d5d3473637d3453c3216b1f14fef500e10bd8158c99bbadb48b788b88d92786f",
    ],
)

rpm(
    name = "less-0__590-5.el9.s390x",
    sha256 = "a33ec6f39ae3101351c6bf7e04cb43cfdd6da38e86f1f68db3693c0b42e2a418",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/less-590-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a33ec6f39ae3101351c6bf7e04cb43cfdd6da38e86f1f68db3693c0b42e2a418",
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
    name = "libarchive-0__3.5.3-4.el9.x86_64",
    sha256 = "4c53176eafd8c449aef704b8fbc2d5401bb7d2ea0a67961956f318f2e9a2c7a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libarchive-3.5.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c53176eafd8c449aef704b8fbc2d5401bb7d2ea0a67961956f318f2e9a2c7a4",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-5.el9.aarch64",
    sha256 = "d142c76ed90b5ca7e9be57b65eee877b2368846aab6c74bae98743876e1190f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libarchive-3.5.3-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d142c76ed90b5ca7e9be57b65eee877b2368846aab6c74bae98743876e1190f4",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-5.el9.s390x",
    sha256 = "c4052b247e24e438cd3db7d9b57120f63dbd5454cb8bf1baa83e095addee057f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libarchive-3.5.3-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c4052b247e24e438cd3db7d9b57120f63dbd5454cb8bf1baa83e095addee057f",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-5.el9.x86_64",
    sha256 = "118578dbc675a5271b75329751c8ca4b314e1dc52f24c3008efa62a74e111e09",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libarchive-3.5.3-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/118578dbc675a5271b75329751c8ca4b314e1dc52f24c3008efa62a74e111e09",
    ],
)

rpm(
    name = "libasan-0__11.5.0-7.el9.aarch64",
    sha256 = "991573af3ec0f38cb6e21c8b7c8382dbff789be7068e3b61b17ccef2a1d95478",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/991573af3ec0f38cb6e21c8b7c8382dbff789be7068e3b61b17ccef2a1d95478",
    ],
)

rpm(
    name = "libasan-0__11.5.0-7.el9.s390x",
    sha256 = "2cf7ced5c2b752b723d53063e0ce6c5ce41f3010105c40a6c93b1194cd272c88",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libasan-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2cf7ced5c2b752b723d53063e0ce6c5ce41f3010105c40a6c93b1194cd272c88",
    ],
)

rpm(
    name = "libassuan-0__2.5.5-3.el9.s390x",
    sha256 = "56a2e5e9e6c2fde071486b174eeecec2631d3b40a6bfc036019e5cd6e590a49c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libassuan-2.5.5-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/56a2e5e9e6c2fde071486b174eeecec2631d3b40a6bfc036019e5cd6e590a49c",
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
    name = "libatomic-0__11.5.0-7.el9.aarch64",
    sha256 = "ef1bfb325b6ac8b9ba05a0f6253ff431772177e0b758f73de6cc4148e8c53027",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ef1bfb325b6ac8b9ba05a0f6253ff431772177e0b758f73de6cc4148e8c53027",
    ],
)

rpm(
    name = "libatomic-0__11.5.0-7.el9.s390x",
    sha256 = "2094974e8abc8132fd13a1ba67b4a2bda4f484c19484b7dfa04d719533b0c039",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libatomic-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2094974e8abc8132fd13a1ba67b4a2bda4f484c19484b7dfa04d719533b0c039",
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
    name = "libblkid-0__2.37.4-21.el9.aarch64",
    sha256 = "ddbde77138e33d01fe88e275f351f8f31754eb26ac5665190437049e867f2b17",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libblkid-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ddbde77138e33d01fe88e275f351f8f31754eb26ac5665190437049e867f2b17",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-21.el9.s390x",
    sha256 = "28b355ddec6b299b6526c80032cd3f9a74fbf7f965bb3c425352a7e616cab287",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libblkid-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/28b355ddec6b299b6526c80032cd3f9a74fbf7f965bb3c425352a7e616cab287",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-21.el9.x86_64",
    sha256 = "2433f8829f894c7c5ba0639eb37a18a92632d4f9383551c901434b4353f96fc4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libblkid-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2433f8829f894c7c5ba0639eb37a18a92632d4f9383551c901434b4353f96fc4",
    ],
)

rpm(
    name = "libbpf-2__1.5.0-1.el9.aarch64",
    sha256 = "ac426fd03d585b90aafd2ab146feb03f28c899985c35d284263429c6d09e7615",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libbpf-1.5.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ac426fd03d585b90aafd2ab146feb03f28c899985c35d284263429c6d09e7615",
    ],
)

rpm(
    name = "libbpf-2__1.5.0-1.el9.s390x",
    sha256 = "26253820a9924054eb6fa27ebde6c248b4575bf578604e181aa5efbe4bf3bd12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libbpf-1.5.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/26253820a9924054eb6fa27ebde6c248b4575bf578604e181aa5efbe4bf3bd12",
    ],
)

rpm(
    name = "libbpf-2__1.5.0-1.el9.x86_64",
    sha256 = "5280ff8fe8a1f217b54abeed8b357c4c3fd47fbcfa928483806effac723accba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-1.5.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5280ff8fe8a1f217b54abeed8b357c4c3fd47fbcfa928483806effac723accba",
    ],
)

rpm(
    name = "libbrotli-0__1.0.9-7.el9.s390x",
    sha256 = "72b4b9ce9df8c2e4fa515af25a8ea3f3cf8ee92572226551c2852b03392c5b63",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libbrotli-1.0.9-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/72b4b9ce9df8c2e4fa515af25a8ea3f3cf8ee92572226551c2852b03392c5b63",
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
    name = "libburn-0__1.5.4-5.el9.aarch64",
    sha256 = "acaf7cc4d8f4926e8a7560af50c0e3248c75dea0ce95d8f0fe48d62088375695",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libburn-1.5.4-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/acaf7cc4d8f4926e8a7560af50c0e3248c75dea0ce95d8f0fe48d62088375695",
    ],
)

rpm(
    name = "libburn-0__1.5.4-5.el9.s390x",
    sha256 = "a663e91009f1daaf573f9f054918983b1a48cada23913a099795f3643471b7c1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libburn-1.5.4-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a663e91009f1daaf573f9f054918983b1a48cada23913a099795f3643471b7c1",
    ],
)

rpm(
    name = "libburn-0__1.5.4-5.el9.x86_64",
    sha256 = "356d18d9694992d402013cf1eea1d5755e70ee57ab95a7dd9a9b1bca0ab57111",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libburn-1.5.4-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/356d18d9694992d402013cf1eea1d5755e70ee57ab95a7dd9a9b1bca0ab57111",
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
    name = "libcom_err-0__1.46.5-7.el9.aarch64",
    sha256 = "5186fb058bd91d4a1a0b00a8aa80b1adafaf9b1f10d9cf2037ebe92d092e2ecf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcom_err-1.46.5-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5186fb058bd91d4a1a0b00a8aa80b1adafaf9b1f10d9cf2037ebe92d092e2ecf",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-7.el9.s390x",
    sha256 = "ebcf2edc91d15da5dd74104d6df8cf7ae5219635cb6c4288ac80545ee2746dad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcom_err-1.46.5-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ebcf2edc91d15da5dd74104d6df8cf7ae5219635cb6c4288ac80545ee2746dad",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-7.el9.x86_64",
    sha256 = "d11e18dbfddd56538c476851d98bd96795f34045e14ebe7b3285f225c4b4b189",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcom_err-1.46.5-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d11e18dbfddd56538c476851d98bd96795f34045e14ebe7b3285f225c4b4b189",
    ],
)

rpm(
    name = "libconfig-0__1.7.2-9.el9.s390x",
    sha256 = "6000e568152331fe40da9b244e579aa22c404bed59b4f33b08de2c962f5849b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libconfig-1.7.2-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6000e568152331fe40da9b244e579aa22c404bed59b4f33b08de2c962f5849b4",
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
    name = "libdb-0__5.3.28-55.el9.x86_64",
    sha256 = "e28608db5eaa3ee38e8bc0d6be1831048da1e638920a6f16a8084e72e2ebf6c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libdb-5.3.28-55.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e28608db5eaa3ee38e8bc0d6be1831048da1e638920a6f16a8084e72e2ebf6c9",
    ],
)

rpm(
    name = "libdb-0__5.3.28-57.el9.aarch64",
    sha256 = "32cfcb3dbd040c206ead6aae6bb3378246af95ab2c7ba18a9db7ec0cec649f34",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libdb-5.3.28-57.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/32cfcb3dbd040c206ead6aae6bb3378246af95ab2c7ba18a9db7ec0cec649f34",
    ],
)

rpm(
    name = "libdb-0__5.3.28-57.el9.s390x",
    sha256 = "5bae96e362fb4731b841f84d22b8ec876eeca2519404625afc51b5ae9fcd6326",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libdb-5.3.28-57.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5bae96e362fb4731b841f84d22b8ec876eeca2519404625afc51b5ae9fcd6326",
    ],
)

rpm(
    name = "libdb-0__5.3.28-57.el9.x86_64",
    sha256 = "17f7fd8c15436826da5ac9d0428ecb83feec18c01b6c5057ab9b85ab97314c96",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libdb-5.3.28-57.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/17f7fd8c15436826da5ac9d0428ecb83feec18c01b6c5057ab9b85ab97314c96",
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
    name = "libfdisk-0__2.37.4-21.el9.aarch64",
    sha256 = "e8c908711f60ab15256f368ee1a5d9b4570e2c19139ec52bb73876d379f98f7a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libfdisk-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e8c908711f60ab15256f368ee1a5d9b4570e2c19139ec52bb73876d379f98f7a",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-21.el9.s390x",
    sha256 = "67f5d8d46714139b75cd84941201e0fa0e1eebd0b26762accfb11d6b4c8b51bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libfdisk-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/67f5d8d46714139b75cd84941201e0fa0e1eebd0b26762accfb11d6b4c8b51bb",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-21.el9.x86_64",
    sha256 = "9a594c51e3bf09cb5016485ee2f143de6db960ff1c7e135c0097f59fa51b2edb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a594c51e3bf09cb5016485ee2f143de6db960ff1c7e135c0097f59fa51b2edb",
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
    name = "libgcc-0__11.5.0-5.el9.x86_64",
    sha256 = "442c065a815212ac21760ff9f0bd93e9f5d5972925d9e987a421cbf6ebba41d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/442c065a815212ac21760ff9f0bd93e9f5d5972925d9e987a421cbf6ebba41d2",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-7.el9.aarch64",
    sha256 = "5daa6ab9cc6e21ca32163dff29a79095a87eb953c842d40527f22b034b2850ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5daa6ab9cc6e21ca32163dff29a79095a87eb953c842d40527f22b034b2850ee",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-7.el9.s390x",
    sha256 = "f101ac7c66efa6c866c781291d7cdd070399aa097ad07f812875468d52212832",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcc-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f101ac7c66efa6c866c781291d7cdd070399aa097ad07f812875468d52212832",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-7.el9.x86_64",
    sha256 = "b4d22e27f51f5152d71f975cd1d732ff3849afccc820df2475ae66d71c96f7cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.5.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4d22e27f51f5152d71f975cd1d732ff3849afccc820df2475ae66d71c96f7cb",
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
    name = "libgomp-0__11.5.0-5.el9.x86_64",
    sha256 = "0158d5640d1f4b3841b681fa26a17361c56d7b1231e64eb163e3d75155913053",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0158d5640d1f4b3841b681fa26a17361c56d7b1231e64eb163e3d75155913053",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-7.el9.aarch64",
    sha256 = "9ed20a7b70027e18f4661c681caf87d1b0f636b96cdac9bf98273923e19a974b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9ed20a7b70027e18f4661c681caf87d1b0f636b96cdac9bf98273923e19a974b",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-7.el9.s390x",
    sha256 = "10656b47bff774837c7746a964cc21f518d2217ff8c2745cf9080f8847fbe67c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgomp-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/10656b47bff774837c7746a964cc21f518d2217ff8c2745cf9080f8847fbe67c",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-7.el9.x86_64",
    sha256 = "268f03661ca54880d2a552676f19c0d74aa2b3f9524c0856804fc0a2b2e278a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.5.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/268f03661ca54880d2a552676f19c0d74aa2b3f9524c0856804fc0a2b2e278a1",
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
    name = "libguestfs-1__1.54.0-3.el9.x86_64",
    sha256 = "1a42f3efec4fba203e3dff85ef391d1ca3573d3d6896601afe9bd17fb15dbd86",
    urls = ["https://storage.googleapis.com/builddeps/1a42f3efec4fba203e3dff85ef391d1ca3573d3d6896601afe9bd17fb15dbd86"],
)

rpm(
    name = "libguestfs-1__1.54.0-9.el9.s390x",
    sha256 = "612deb549f07c32689421856afa195f003ec9fb2ac9161dfa8b4f631348cdaef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libguestfs-1.54.0-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/612deb549f07c32689421856afa195f003ec9fb2ac9161dfa8b4f631348cdaef",
    ],
)

rpm(
    name = "libguestfs-1__1.54.0-9.el9.x86_64",
    sha256 = "262f8449b469e08285e008ac903a02da4587560e3c3c11f695f0abc4870ee627",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libguestfs-1.54.0-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/262f8449b469e08285e008ac903a02da4587560e3c3c11f695f0abc4870ee627",
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
    name = "libisoburn-0__1.5.4-5.el9.aarch64",
    sha256 = "1a81eca953e8c268f4c7e9fe41b81589c056888649924d9717215fefefe2f4d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libisoburn-1.5.4-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1a81eca953e8c268f4c7e9fe41b81589c056888649924d9717215fefefe2f4d6",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-5.el9.s390x",
    sha256 = "0e137123b209360ec496522cf6da2b6eed11eade27d1b61685e6d4387b984464",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libisoburn-1.5.4-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0e137123b209360ec496522cf6da2b6eed11eade27d1b61685e6d4387b984464",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-5.el9.x86_64",
    sha256 = "ef66466bb16b1955cf65715240f371d6bc1aa018a73f9c8c1b28ba0ce3bc5f41",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libisoburn-1.5.4-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ef66466bb16b1955cf65715240f371d6bc1aa018a73f9c8c1b28ba0ce3bc5f41",
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
    name = "libksba-0__1.5.1-7.el9.s390x",
    sha256 = "10e17f1f886f90259f915e855389f3e3852fddd52be35110ebe0d0f4b9b4f51a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libksba-1.5.1-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/10e17f1f886f90259f915e855389f3e3852fddd52be35110ebe0d0f4b9b4f51a",
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
    name = "libmount-0__2.37.4-21.el9.aarch64",
    sha256 = "84c61be8eee5f148ece4b17cab7b9774bd4e5e51377ff459c278dadc99d733d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmount-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/84c61be8eee5f148ece4b17cab7b9774bd4e5e51377ff459c278dadc99d733d6",
    ],
)

rpm(
    name = "libmount-0__2.37.4-21.el9.s390x",
    sha256 = "d7f982570169708f7ff3ba976dd3df4c64cf1d9a197f5da98d7dae2e9f5f2cb2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libmount-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d7f982570169708f7ff3ba976dd3df4c64cf1d9a197f5da98d7dae2e9f5f2cb2",
    ],
)

rpm(
    name = "libmount-0__2.37.4-21.el9.x86_64",
    sha256 = "d8bfc70d1a9a594569c8c95bda682804a20bb4ee602db3efa7b6e76d289ecc66",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmount-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d8bfc70d1a9a594569c8c95bda682804a20bb4ee602db3efa7b6e76d289ecc66",
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
    name = "libnbd-0__1.20.3-1.el9.x86_64",
    sha256 = "4d39fdb30ac2f05b0cb67e296f0fd553b7fe2092ba8b9f3940f2d90e0146a835",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libnbd-1.20.3-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d39fdb30ac2f05b0cb67e296f0fd553b7fe2092ba8b9f3940f2d90e0146a835",
    ],
)

rpm(
    name = "libnbd-0__1.20.3-4.el9.aarch64",
    sha256 = "7c9bb6872b93d95b2a2bf729793b50848cde216089293010197471146d23d9a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libnbd-1.20.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7c9bb6872b93d95b2a2bf729793b50848cde216089293010197471146d23d9a4",
    ],
)

rpm(
    name = "libnbd-0__1.20.3-4.el9.s390x",
    sha256 = "d73945914b3ea835369f64416cf111fcf527775d70e35109f2a270763328e6ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libnbd-1.20.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d73945914b3ea835369f64416cf111fcf527775d70e35109f2a270763328e6ce",
    ],
)

rpm(
    name = "libnbd-0__1.20.3-4.el9.x86_64",
    sha256 = "d74d51b389dcf44bd2e10e76085dc41db925debee2ce33b721c554a9dd1f40af",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libnbd-1.20.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d74d51b389dcf44bd2e10e76085dc41db925debee2ce33b721c554a9dd1f40af",
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
    name = "libnfnetlink-0__1.0.1-23.el9.aarch64",
    sha256 = "8b261a1555fd3b299c8b16d7c1159c726ec17dbd78d5217dbc6e69099f01c6cb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnfnetlink-1.0.1-23.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8b261a1555fd3b299c8b16d7c1159c726ec17dbd78d5217dbc6e69099f01c6cb",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-23.el9.s390x",
    sha256 = "1d092de5c4fde5b75011185bda315959d01994c162009b63373e901e72e42769",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnfnetlink-1.0.1-23.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1d092de5c4fde5b75011185bda315959d01994c162009b63373e901e72e42769",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-23.el9.x86_64",
    sha256 = "c920598cb4dab7c5b6b00af9f09c21f89b23c4e12729016fd892d6d7e1291615",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.1-23.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c920598cb4dab7c5b6b00af9f09c21f89b23c4e12729016fd892d6d7e1291615",
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
    name = "libosinfo-0__1.10.0-1.el9.s390x",
    sha256 = "f7704e01f4ab1315cf32f5e2f8d2bb33411e403fdbd4398ea1d76eb4f90550a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libosinfo-1.10.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f7704e01f4ab1315cf32f5e2f8d2bb33411e403fdbd4398ea1d76eb4f90550a1",
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
    name = "libproxy-0__0.4.15-35.el9.s390x",
    sha256 = "1ce49253ec771fbeefb6fa26ae07707fe0039e7f393cf24845d88f2453b7c116",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libproxy-0.4.15-35.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1ce49253ec771fbeefb6fa26ae07707fe0039e7f393cf24845d88f2453b7c116",
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
    name = "libpsl-0__0.21.1-5.el9.s390x",
    sha256 = "d54f8e3050d403352fe6afcf6aa34838017b2d56026625ea29f5307fc2ce173c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libpsl-0.21.1-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d54f8e3050d403352fe6afcf6aa34838017b2d56026625ea29f5307fc2ce173c",
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
    name = "libselinux-0__3.6-3.el9.aarch64",
    sha256 = "42b6190d9e4ea6019059991f50965ac6267012343241f0cc64fd24c6e20aaa2a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-3.6-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/42b6190d9e4ea6019059991f50965ac6267012343241f0cc64fd24c6e20aaa2a",
    ],
)

rpm(
    name = "libselinux-0__3.6-3.el9.s390x",
    sha256 = "16b3c0c73dcfff8b54a5554a4bcbd639603508d8502857c05ff9aa2360690094",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-3.6-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/16b3c0c73dcfff8b54a5554a4bcbd639603508d8502857c05ff9aa2360690094",
    ],
)

rpm(
    name = "libselinux-0__3.6-3.el9.x86_64",
    sha256 = "79abe72ea8dccb4134286fd1aae79827f10bde0cc1c35224886e93b293d282d1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-3.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/79abe72ea8dccb4134286fd1aae79827f10bde0cc1c35224886e93b293d282d1",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-3.el9.aarch64",
    sha256 = "5e028899301316df30d03631e7d317c3236fea0f5138c799b055560676f991eb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.6-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5e028899301316df30d03631e7d317c3236fea0f5138c799b055560676f991eb",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-3.el9.s390x",
    sha256 = "05a8b056b7df62d0f6fde665fb98302fb9b1c0b18a40d68528270e275748891e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-utils-3.6-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/05a8b056b7df62d0f6fde665fb98302fb9b1c0b18a40d68528270e275748891e",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-3.el9.x86_64",
    sha256 = "f78d42cbd9cc6220b44631787ba17faf4ad44befa8ebddfdf504d4654eb2dfe0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f78d42cbd9cc6220b44631787ba17faf4ad44befa8ebddfdf504d4654eb2dfe0",
    ],
)

rpm(
    name = "libsemanage-0__3.6-5.el9.aarch64",
    sha256 = "f5402c7056dc92ea2e52ad436c6eece8c18040ac77141e5f0ffe01eea209dfe7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsemanage-3.6-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f5402c7056dc92ea2e52ad436c6eece8c18040ac77141e5f0ffe01eea209dfe7",
    ],
)

rpm(
    name = "libsemanage-0__3.6-5.el9.s390x",
    sha256 = "888a4ef687c43c03324bfe3c5815810d48322478cd966b4bcb1d237a16b3a0b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsemanage-3.6-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/888a4ef687c43c03324bfe3c5815810d48322478cd966b4bcb1d237a16b3a0b0",
    ],
)

rpm(
    name = "libsemanage-0__3.6-5.el9.x86_64",
    sha256 = "3dcf6e7f2779434d9dc7aef0065c3a2977792170264a60d4324f6625bb9cd69a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsemanage-3.6-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3dcf6e7f2779434d9dc7aef0065c3a2977792170264a60d4324f6625bb9cd69a",
    ],
)

rpm(
    name = "libsepol-0__3.6-2.el9.x86_64",
    sha256 = "7a1c10a4512624dfc1b76da45b7a0d15f8ecdddf20c9738b10ca12df7f488ae1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsepol-3.6-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7a1c10a4512624dfc1b76da45b7a0d15f8ecdddf20c9738b10ca12df7f488ae1",
    ],
)

rpm(
    name = "libsepol-0__3.6-3.el9.aarch64",
    sha256 = "2cd63ed497af8a202c79790b04362ba224b50ec7c377abb21901160e4000e07d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsepol-3.6-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2cd63ed497af8a202c79790b04362ba224b50ec7c377abb21901160e4000e07d",
    ],
)

rpm(
    name = "libsepol-0__3.6-3.el9.s390x",
    sha256 = "c1246f8553c2aec3ca86721f8bd77fab4f4fcd22527bb6a6e494b4046ee17461",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsepol-3.6-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c1246f8553c2aec3ca86721f8bd77fab4f4fcd22527bb6a6e494b4046ee17461",
    ],
)

rpm(
    name = "libsepol-0__3.6-3.el9.x86_64",
    sha256 = "6d3d16c3121ccf989f8a123812e524cb1fc098fb01ec9f1c6327544e85aaf84d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsepol-3.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6d3d16c3121ccf989f8a123812e524cb1fc098fb01ec9f1c6327544e85aaf84d",
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
    name = "libsmartcols-0__2.37.4-21.el9.aarch64",
    sha256 = "022b23b2500972666db40ab9d5278ca704c54600e8443dea8e327910afa5b411",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/022b23b2500972666db40ab9d5278ca704c54600e8443dea8e327910afa5b411",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-21.el9.s390x",
    sha256 = "94929db8ee7027fca21e8ac17519dfcf9366a5a7ba8c8ba627dcb5976b199036",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsmartcols-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/94929db8ee7027fca21e8ac17519dfcf9366a5a7ba8c8ba627dcb5976b199036",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-21.el9.x86_64",
    sha256 = "30e2a071ad6f1939f14fc89c827d61ccb28a6cbf6e443db39e8019a18c7e18d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/30e2a071ad6f1939f14fc89c827d61ccb28a6cbf6e443db39e8019a18c7e18d4",
    ],
)

rpm(
    name = "libsoup-0__2.72.0-10.el9.s390x",
    sha256 = "5c157ebfd258aaa833701f0e48d2a8e54064d5e9fbcdae9f4e426ef84efa1a75",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libsoup-2.72.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c157ebfd258aaa833701f0e48d2a8e54064d5e9fbcdae9f4e426ef84efa1a75",
    ],
)

rpm(
    name = "libsoup-0__2.72.0-10.el9.x86_64",
    sha256 = "e7dc6b485f95e65f22d7a91575dd6cfaae6d9cfbeaacd612e7fa4bbccaa9211d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libsoup-2.72.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e7dc6b485f95e65f22d7a91575dd6cfaae6d9cfbeaacd612e7fa4bbccaa9211d",
    ],
)

rpm(
    name = "libss-0__1.46.5-7.el9.aarch64",
    sha256 = "007f768d0143d2a4ba9553ce0e104cf2ffa27be15b8d652786ff0b18fc4492ef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libss-1.46.5-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/007f768d0143d2a4ba9553ce0e104cf2ffa27be15b8d652786ff0b18fc4492ef",
    ],
)

rpm(
    name = "libss-0__1.46.5-7.el9.s390x",
    sha256 = "1ddfb93a4d5fe0825da2b329bae5a02a481e9c9e3fa2f27a0f1070334b856369",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libss-1.46.5-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1ddfb93a4d5fe0825da2b329bae5a02a481e9c9e3fa2f27a0f1070334b856369",
    ],
)

rpm(
    name = "libss-0__1.46.5-7.el9.x86_64",
    sha256 = "4556a06adc4bf034fc39118a6da3ed64727c8bf369d58649e70922ec4aa5beb4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libss-1.46.5-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4556a06adc4bf034fc39118a6da3ed64727c8bf369d58649e70922ec4aa5beb4",
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
    name = "libsss_idmap-0__2.9.7-1.el9.aarch64",
    sha256 = "10314e595620639d8c940ba72f8097d7c1eb3e18dd720efb86a1b44ffc5ad9ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.9.7-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/10314e595620639d8c940ba72f8097d7c1eb3e18dd720efb86a1b44ffc5ad9ed",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.7-1.el9.s390x",
    sha256 = "78f73deae0a633526c00b42be5416d8da24f07db9d592e4d372c26b1b61df707",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.9.7-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/78f73deae0a633526c00b42be5416d8da24f07db9d592e4d372c26b1b61df707",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.7-1.el9.x86_64",
    sha256 = "ad8245828ee742c1a737a1427fa59155871b53eaa689cdbe4a9bffe893c91988",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.9.7-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ad8245828ee742c1a737a1427fa59155871b53eaa689cdbe4a9bffe893c91988",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.7-1.el9.aarch64",
    sha256 = "c2268a621f64d6e40b74e6548a3264646cc191e458a395eb665b06e70103a907",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.9.7-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c2268a621f64d6e40b74e6548a3264646cc191e458a395eb665b06e70103a907",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.7-1.el9.s390x",
    sha256 = "e15e3b90b9909a98f5a2d41a43f13eebd62d661330092a1f290468c5f0d609c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.9.7-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e15e3b90b9909a98f5a2d41a43f13eebd62d661330092a1f290468c5f0d609c4",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.7-1.el9.x86_64",
    sha256 = "aecbe43028872c3b0101581b2f35e53aa0ea16b561fda4da7c6e43f22a26c482",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.9.7-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aecbe43028872c3b0101581b2f35e53aa0ea16b561fda4da7c6e43f22a26c482",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-5.el9.x86_64",
    sha256 = "6628a0027a113c8687d0cd52ed5725ee6cb1ee2a02897349289d683fc6453223",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.5.0-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6628a0027a113c8687d0cd52ed5725ee6cb1ee2a02897349289d683fc6453223",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-7.el9.aarch64",
    sha256 = "359737f9cdc0fede1528cc7808538a3317ded34648588dc51ba2eb411acf12c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/359737f9cdc0fede1528cc7808538a3317ded34648588dc51ba2eb411acf12c6",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-7.el9.s390x",
    sha256 = "6a22072211b8838a5ad7e0630a2eabea08ae86e909338320d64ab23db36109a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libstdc++-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6a22072211b8838a5ad7e0630a2eabea08ae86e909338320d64ab23db36109a1",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-7.el9.x86_64",
    sha256 = "10e843cb5d33800f3290344d8efd3bb0d129898097b8aec6a5575ea8f019e7ef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.5.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10e843cb5d33800f3290344d8efd3bb0d129898097b8aec6a5575ea8f019e7ef",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-9.el9.aarch64",
    sha256 = "7b99e8f1081ba2c511021b666b9f8176abb31168920e86c392cd45299f400b59",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtasn1-4.16.0-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7b99e8f1081ba2c511021b666b9f8176abb31168920e86c392cd45299f400b59",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-9.el9.s390x",
    sha256 = "0ebbc12c3ae3f270efef2965bb77d6e806733eb07505ec7a33468f0fd72360bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libtasn1-4.16.0-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0ebbc12c3ae3f270efef2965bb77d6e806733eb07505ec7a33468f0fd72360bd",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-9.el9.x86_64",
    sha256 = "addd155d4abc41529d7e8588f442e50a87db3a1314bd2162fbb4950d898a2e28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtasn1-4.16.0-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/addd155d4abc41529d7e8588f442e50a87db3a1314bd2162fbb4950d898a2e28",
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
    name = "libtool-ltdl-0__2.4.6-46.el9.s390x",
    sha256 = "548a2de100fb988854c4e3e814314eb03c8645f7a6e9f658b61adbed81c8251e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libtool-ltdl-2.4.6-46.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/548a2de100fb988854c4e3e814314eb03c8645f7a6e9f658b61adbed81c8251e",
    ],
)

rpm(
    name = "libtool-ltdl-0__2.4.6-46.el9.x86_64",
    sha256 = "a04d5a4ccd83b8903e2d7fe76208f57636a6ed07f20e0d350a2b1075c15a2147",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtool-ltdl-2.4.6-46.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a04d5a4ccd83b8903e2d7fe76208f57636a6ed07f20e0d350a2b1075c15a2147",
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
    name = "libubsan-0__11.5.0-7.el9.aarch64",
    sha256 = "db5514afb8bae266efda272d1b9eb00e35023d8f80b1d82b79d1743f93efae42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.5.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/db5514afb8bae266efda272d1b9eb00e35023d8f80b1d82b79d1743f93efae42",
    ],
)

rpm(
    name = "libubsan-0__11.5.0-7.el9.s390x",
    sha256 = "97f357b5ca51a07826621885b5241f47a5b91cd3d34894bbcd2594bc2bbfc3d2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libubsan-11.5.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/97f357b5ca51a07826621885b5241f47a5b91cd3d34894bbcd2594bc2bbfc3d2",
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
    name = "libuuid-0__2.37.4-21.el9.aarch64",
    sha256 = "9ef80033bf2bbba7aca3a8f789a48a7597f2113271c6da9d9f71bca213aae259",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libuuid-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9ef80033bf2bbba7aca3a8f789a48a7597f2113271c6da9d9f71bca213aae259",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-21.el9.s390x",
    sha256 = "b604d6ef2f49c29d93601f6cad1ac04d83f14ec719ba0f561d6912948d6f2b56",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libuuid-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b604d6ef2f49c29d93601f6cad1ac04d83f14ec719ba0f561d6912948d6f2b56",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-21.el9.x86_64",
    sha256 = "be4793be5af11772206abe023746ec4021a8b7bc124fdc7e7cdb92b57c46d125",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be4793be5af11772206abe023746ec4021a8b7bc124fdc7e7cdb92b57c46d125",
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
    name = "libvirt-client-0__10.10.0-13.el9.aarch64",
    sha256 = "40ef751232331f32f16dd2905767ff0b32cca0ebfb47813ea0371a3c1ddf2699",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-10.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/40ef751232331f32f16dd2905767ff0b32cca0ebfb47813ea0371a3c1ddf2699",
    ],
)

rpm(
    name = "libvirt-client-0__10.10.0-13.el9.s390x",
    sha256 = "04d94d28abda653680b8fdc608212dabd3e1b4c68d11796515ebb185c5e990d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-client-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/04d94d28abda653680b8fdc608212dabd3e1b4c68d11796515ebb185c5e990d6",
    ],
)

rpm(
    name = "libvirt-client-0__10.10.0-13.el9.x86_64",
    sha256 = "56b926345002ba7b51c5217e00a144f0fe768675f7dc93a5d98814a529d0f949",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56b926345002ba7b51c5217e00a144f0fe768675f7dc93a5d98814a529d0f949",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-13.el9.aarch64",
    sha256 = "6e131f3c777a0c1bf89e687cb549249a1b8243eee3739ad425b6d33dd9302622",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-10.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6e131f3c777a0c1bf89e687cb549249a1b8243eee3739ad425b6d33dd9302622",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-13.el9.s390x",
    sha256 = "8a364e6ffef1cba91da795fa38761eebb7b8d56ea5ce8ba84a28cd0aec686197",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8a364e6ffef1cba91da795fa38761eebb7b8d56ea5ce8ba84a28cd0aec686197",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-13.el9.x86_64",
    sha256 = "70c6ba85fca29812dcae142efa5ee2007cadcc72d9d065bba0f8b25c0d4b9227",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/70c6ba85fca29812dcae142efa5ee2007cadcc72d9d065bba0f8b25c0d4b9227",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-7.el9.x86_64",
    sha256 = "ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af",
    urls = ["https://storage.googleapis.com/builddeps/ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-13.el9.aarch64",
    sha256 = "7b12ad04be34565a2d01763b8d8101c7b7cce81b0a74dbb5359e4f74f3defddb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-10.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7b12ad04be34565a2d01763b8d8101c7b7cce81b0a74dbb5359e4f74f3defddb",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-13.el9.s390x",
    sha256 = "8b0a0da052c3ca3d17b9f9ab05fa91746719789ef188c4dbf13887780aa426cf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8b0a0da052c3ca3d17b9f9ab05fa91746719789ef188c4dbf13887780aa426cf",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-13.el9.x86_64",
    sha256 = "576df107c4f55d316d857333147937cbcfe26c402bca415947be36fc1dc2cdf6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/576df107c4f55d316d857333147937cbcfe26c402bca415947be36fc1dc2cdf6",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-7.el9.x86_64",
    sha256 = "13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1",
    urls = ["https://storage.googleapis.com/builddeps/13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1"],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.10.0-13.el9.s390x",
    sha256 = "8dd19607375a4b92aacbf05fe03f6fdc4443ed58b1d2b7c6e47c8da47ebe5741",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-secret-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8dd19607375a4b92aacbf05fe03f6fdc4443ed58b1d2b7c6e47c8da47ebe5741",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.10.0-13.el9.x86_64",
    sha256 = "675391f529e4552d77ad0fca655ab84bb622d2935ae53552a0db055e2d06f6a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/675391f529e4552d77ad0fca655ab84bb622d2935ae53552a0db055e2d06f6a1",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.10.0-7.el9.x86_64",
    sha256 = "8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3",
    urls = ["https://storage.googleapis.com/builddeps/8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3"],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.10.0-13.el9.s390x",
    sha256 = "bd2f997a9832a9979631a949914a81f69d9e76ce2a596be1bcaf8b9a8b513b30",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-storage-core-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bd2f997a9832a9979631a949914a81f69d9e76ce2a596be1bcaf8b9a8b513b30",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.10.0-13.el9.x86_64",
    sha256 = "dd93b98f8f66eeab93d7cd5527b0f253471174db51f74106099187b85514fbf0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd93b98f8f66eeab93d7cd5527b0f253471174db51f74106099187b85514fbf0",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.10.0-7.el9.x86_64",
    sha256 = "a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067",
    urls = ["https://storage.googleapis.com/builddeps/a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067"],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-13.el9.aarch64",
    sha256 = "95bb112d12bdb61fc1693610c4148699c154b2601f5dda61491a1f242f749ba2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-10.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/95bb112d12bdb61fc1693610c4148699c154b2601f5dda61491a1f242f749ba2",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-13.el9.s390x",
    sha256 = "8096e3a5998474b4082f668a9858fbf8193d4638d8fc5137bbfe1bceace7e0a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8096e3a5998474b4082f668a9858fbf8193d4638d8fc5137bbfe1bceace7e0a7",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-13.el9.x86_64",
    sha256 = "641753baa06e7894b58e95f6a6036e2f243edf4c8c25cf67fbfc43013b2bea3c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/641753baa06e7894b58e95f6a6036e2f243edf4c8c25cf67fbfc43013b2bea3c",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-7.el9.x86_64",
    sha256 = "7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff",
    urls = ["https://storage.googleapis.com/builddeps/7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff"],
)

rpm(
    name = "libvirt-devel-0__10.10.0-13.el9.aarch64",
    sha256 = "eacbe6b2bf1642fe798f6916fd1cbd1c371b09bf229d59e68fc1fe8806117e12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-10.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eacbe6b2bf1642fe798f6916fd1cbd1c371b09bf229d59e68fc1fe8806117e12",
    ],
)

rpm(
    name = "libvirt-devel-0__10.10.0-13.el9.s390x",
    sha256 = "c2ee97220ea918c34ada85dfddc7535cb296fc3e2697c649cc764d593d281fb4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libvirt-devel-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c2ee97220ea918c34ada85dfddc7535cb296fc3e2697c649cc764d593d281fb4",
    ],
)

rpm(
    name = "libvirt-devel-0__10.10.0-13.el9.x86_64",
    sha256 = "6d29b2fa42d3f07704c0556b3f0ed72253ddb273e183b7e8778d640e66e89f55",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6d29b2fa42d3f07704c0556b3f0ed72253ddb273e183b7e8778d640e66e89f55",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-13.el9.aarch64",
    sha256 = "ab827dbe53e0cb0ae5bad881cd8164f2d95aa23bdd8498760d3c1d08cb912144",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-10.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab827dbe53e0cb0ae5bad881cd8164f2d95aa23bdd8498760d3c1d08cb912144",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-13.el9.s390x",
    sha256 = "7d169dbc46a99050ea85bb68133e37dc2aad07e8bfd4018f666a4477b8ecacf1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-libs-10.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7d169dbc46a99050ea85bb68133e37dc2aad07e8bfd4018f666a4477b8ecacf1",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-13.el9.x86_64",
    sha256 = "16bfc41c7df521e374cdf140aec9e0a6b3fdb49b085c8c71f5333e6427f968cd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-10.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/16bfc41c7df521e374cdf140aec9e0a6b3fdb49b085c8c71f5333e6427f968cd",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-7.el9.x86_64",
    sha256 = "72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39",
    urls = ["https://storage.googleapis.com/builddeps/72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39"],
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
    name = "libxml2-0__2.9.13-9.el9.aarch64",
    sha256 = "70ffc89d054d4c98d125e628f1ea5208e66baea96ddc170f83e20e0fcd08e731",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.13-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/70ffc89d054d4c98d125e628f1ea5208e66baea96ddc170f83e20e0fcd08e731",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-9.el9.s390x",
    sha256 = "a0fac775bc308c9c2fc9c88cb1f5a14bfbeb1aee206bfe5f7bda2aa136940c5c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libxml2-2.9.13-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a0fac775bc308c9c2fc9c88cb1f5a14bfbeb1aee206bfe5f7bda2aa136940c5c",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-9.el9.x86_64",
    sha256 = "70b74fdfab02d40caad350cf83bc676a782de69b25beb3d37dc193aaf381d9e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.13-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/70b74fdfab02d40caad350cf83bc676a782de69b25beb3d37dc193aaf381d9e0",
    ],
)

rpm(
    name = "libxslt-0__1.1.34-12.el9.s390x",
    sha256 = "d2a72b102141ce337c5dab51985071e29bc2e1a00008f866c1cfd265c49c5d65",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libxslt-1.1.34-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d2a72b102141ce337c5dab51985071e29bc2e1a00008f866c1cfd265c49c5d65",
    ],
)

rpm(
    name = "libxslt-0__1.1.34-12.el9.x86_64",
    sha256 = "d14a14cb0ab0be6864c26d26d2e0c580fdf50b534cf79f23cc7677d51ddb2adc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libxslt-1.1.34-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d14a14cb0ab0be6864c26d26d2e0c580fdf50b534cf79f23cc7677d51ddb2adc",
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
    name = "libzstd-0__1.5.5-1.el9.aarch64",
    sha256 = "49fb3a1052d9f50abb9ad3f0ab4ed186b2c0bb51fcb04883702fbc362d116108",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libzstd-1.5.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/49fb3a1052d9f50abb9ad3f0ab4ed186b2c0bb51fcb04883702fbc362d116108",
    ],
)

rpm(
    name = "libzstd-0__1.5.5-1.el9.s390x",
    sha256 = "720ce927a447b6c9fd2479ecb924112d450ec9b4f927090b36ef34b10ad4b163",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libzstd-1.5.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/720ce927a447b6c9fd2479ecb924112d450ec9b4f927090b36ef34b10ad4b163",
    ],
)

rpm(
    name = "libzstd-0__1.5.5-1.el9.x86_64",
    sha256 = "3439a7437a4b47ef4b6efbcd8c5862180fb281dd956d70a4ffe3764fd8d997dd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libzstd-1.5.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3439a7437a4b47ef4b6efbcd8c5862180fb281dd956d70a4ffe3764fd8d997dd",
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
    name = "nettle-0__3.10.1-1.el9.aarch64",
    sha256 = "caf6dda4eaf3c7e3061ec335d45176ebfcaa72ed583df59c32c9dffc00a24ad9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nettle-3.10.1-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/caf6dda4eaf3c7e3061ec335d45176ebfcaa72ed583df59c32c9dffc00a24ad9",
    ],
)

rpm(
    name = "nettle-0__3.10.1-1.el9.s390x",
    sha256 = "d05a33e0b673bc34580c443f7d7c28b50f8b4fd77ad87ed3cef30f991d7cbf09",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nettle-3.10.1-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d05a33e0b673bc34580c443f7d7c28b50f8b4fd77ad87ed3cef30f991d7cbf09",
    ],
)

rpm(
    name = "nettle-0__3.10.1-1.el9.x86_64",
    sha256 = "aa28996450c98399099cfcc0fb722723b5821edff27cff53288e1c0298a98190",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nettle-3.10.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aa28996450c98399099cfcc0fb722723b5821edff27cff53288e1c0298a98190",
    ],
)

rpm(
    name = "nftables-1__1.0.9-4.el9.aarch64",
    sha256 = "98d8d68631ccbf041d7069fa6c1f8ac500aae1bfdc5c899b555a684049b6ed36",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.9-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98d8d68631ccbf041d7069fa6c1f8ac500aae1bfdc5c899b555a684049b6ed36",
    ],
)

rpm(
    name = "nftables-1__1.0.9-4.el9.s390x",
    sha256 = "a836fbc79366f335ff13b6bdbee0e119e1bc64f910cdd64aa0d44b395e9b03c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nftables-1.0.9-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a836fbc79366f335ff13b6bdbee0e119e1bc64f910cdd64aa0d44b395e9b03c9",
    ],
)

rpm(
    name = "nftables-1__1.0.9-4.el9.x86_64",
    sha256 = "c75b14d92af48c07b167e3b81eab5393748f23afc497e780d8509661ab139d0b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.9-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c75b14d92af48c07b167e3b81eab5393748f23afc497e780d8509661ab139d0b",
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
    name = "npth-0__1.6-8.el9.s390x",
    sha256 = "f66f12068208409067e6c342e6c0f4f0646fe527dbb7d5bc3d41adb4d9802b52",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/npth-1.6-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f66f12068208409067e6c342e6c0f4f0646fe527dbb7d5bc3d41adb4d9802b52",
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
    name = "numactl-libs-0__2.0.19-1.el9.aarch64",
    sha256 = "40f7ef0c0fd573c86227a56b25352c731f41d75acd25d7c7a10c2167bb9aaf74",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.19-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/40f7ef0c0fd573c86227a56b25352c731f41d75acd25d7c7a10c2167bb9aaf74",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-1.el9.s390x",
    sha256 = "47d0a1240e2223c10dc00b6f303e41c638e916b4d2f7a780c8e690ecd2731931",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/numactl-libs-2.0.19-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/47d0a1240e2223c10dc00b6f303e41c638e916b4d2f7a780c8e690ecd2731931",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-1.el9.x86_64",
    sha256 = "3abe41a330364e2e1bf905458de8c0314f0ac3082e6cc475149bd9b2ffdeb428",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.19-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3abe41a330364e2e1bf905458de8c0314f0ac3082e6cc475149bd9b2ffdeb428",
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
    name = "openldap-0__2.6.8-4.el9.s390x",
    sha256 = "67a9f53b14250f7934faebcc7961b2f9c06875b9c8fb16d4ac19c0e110d0e446",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openldap-2.6.8-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/67a9f53b14250f7934faebcc7961b2f9c06875b9c8fb16d4ac19c0e110d0e446",
    ],
)

rpm(
    name = "openldap-0__2.6.8-4.el9.x86_64",
    sha256 = "606794339fa964e9c68a94fe756566e0651c52da47ebe7df5fb66f700c6a0421",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openldap-2.6.8-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/606794339fa964e9c68a94fe756566e0651c52da47ebe7df5fb66f700c6a0421",
    ],
)

rpm(
    name = "openssl-1__3.2.2-6.el9.x86_64",
    sha256 = "3018c5d2901213b6bdbe62301ef894008ec52b1122e270190eabb62ad282a46a",
    urls = ["https://storage.googleapis.com/builddeps/3018c5d2901213b6bdbe62301ef894008ec52b1122e270190eabb62ad282a46a"],
)

rpm(
    name = "openssl-1__3.5.0-4.el9.aarch64",
    sha256 = "49cb6a9da7909eeccdc56cd8cfe4c4bb7d78d34ba98596dcbc376802cd0f8cb1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.5.0-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/49cb6a9da7909eeccdc56cd8cfe4c4bb7d78d34ba98596dcbc376802cd0f8cb1",
    ],
)

rpm(
    name = "openssl-1__3.5.0-4.el9.s390x",
    sha256 = "5750362b25d55cee629a447728652e04294e30c5e603fca931983e1c36170c21",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-3.5.0-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5750362b25d55cee629a447728652e04294e30c5e603fca931983e1c36170c21",
    ],
)

rpm(
    name = "openssl-1__3.5.0-4.el9.x86_64",
    sha256 = "f2675a074aaf4ed88cf3cbc6aeb872eafdf9de90ba54b80a3a5352c5d6d75ccd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.5.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f2675a074aaf4ed88cf3cbc6aeb872eafdf9de90ba54b80a3a5352c5d6d75ccd",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-6.el9.x86_64",
    sha256 = "4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738",
    urls = ["https://storage.googleapis.com/builddeps/4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738"],
)

rpm(
    name = "openssl-libs-1__3.5.0-4.el9.aarch64",
    sha256 = "0c61b16e16ba0ac8471ffa2d0a0af9e5daf2ed08121f74b5f4b022ce05b68319",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.5.0-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0c61b16e16ba0ac8471ffa2d0a0af9e5daf2ed08121f74b5f4b022ce05b68319",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.0-4.el9.s390x",
    sha256 = "bfb6dc2142c255f382fa6eeab468f0966e39fa92d0a8b857a083e4fd499c6c31",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-libs-3.5.0-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bfb6dc2142c255f382fa6eeab468f0966e39fa92d0a8b857a083e4fd499c6c31",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.0-4.el9.x86_64",
    sha256 = "f66719b4fed80a139cfcbc3f5ce57ece33fd37ec52ca3e45ea6e146a50fac3ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.5.0-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f66719b4fed80a139cfcbc3f5ce57ece33fd37ec52ca3e45ea6e146a50fac3ae",
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
    name = "osinfo-db-0__20250606-1.el9.s390x",
    sha256 = "78097993d3459be72266cc576d14f7e51c6df7d7deeb14453b3697216344fe44",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/osinfo-db-20250606-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/78097993d3459be72266cc576d14f7e51c6df7d7deeb14453b3697216344fe44",
    ],
)

rpm(
    name = "osinfo-db-0__20250606-1.el9.x86_64",
    sha256 = "78097993d3459be72266cc576d14f7e51c6df7d7deeb14453b3697216344fe44",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/osinfo-db-20250606-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/78097993d3459be72266cc576d14f7e51c6df7d7deeb14453b3697216344fe44",
    ],
)

rpm(
    name = "osinfo-db-tools-0__1.10.0-1.el9.s390x",
    sha256 = "2df9abce1b172c03c768b4f7ea7befe0c2e4d6647447d5b729caa061fdbf3803",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/osinfo-db-tools-1.10.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2df9abce1b172c03c768b4f7ea7befe0c2e4d6647447d5b729caa061fdbf3803",
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
    name = "pam-0__1.5.1-23.el9.x86_64",
    sha256 = "fba392096cbf59204549bca23d4060cdf8aaaa9ce35ade8194c111f519033e10",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-23.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fba392096cbf59204549bca23d4060cdf8aaaa9ce35ade8194c111f519033e10",
    ],
)

rpm(
    name = "pam-0__1.5.1-24.el9.aarch64",
    sha256 = "c7b7358a0797b47af1e404d577850fbd85bec5142e35e4c360ca6e6dd6973f31",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-24.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7b7358a0797b47af1e404d577850fbd85bec5142e35e4c360ca6e6dd6973f31",
    ],
)

rpm(
    name = "pam-0__1.5.1-24.el9.s390x",
    sha256 = "d3be7090c028966cd0eb0338c94510f15591d503101feea2d3aa7d94393bc145",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pam-1.5.1-24.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d3be7090c028966cd0eb0338c94510f15591d503101feea2d3aa7d94393bc145",
    ],
)

rpm(
    name = "pam-0__1.5.1-24.el9.x86_64",
    sha256 = "022453a8847c6bfc4a8360f0c073cf9a3a9decb4afca94688af33f10ba11bc86",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-24.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/022453a8847c6bfc4a8360f0c073cf9a3a9decb4afca94688af33f10ba11bc86",
    ],
)

rpm(
    name = "parted-0__3.5-3.el9.s390x",
    sha256 = "e328d103fa4e64fa6558998d79b42bb37882d068b9251fc3301b1001b7f01936",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/parted-3.5-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e328d103fa4e64fa6558998d79b42bb37882d068b9251fc3301b1001b7f01936",
    ],
)

rpm(
    name = "parted-0__3.5-3.el9.x86_64",
    sha256 = "77255654a5fa5d0b45a7bdaee26e50e9935a657eaa49949ae9313e2500136213",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/parted-3.5-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/77255654a5fa5d0b45a7bdaee26e50e9935a657eaa49949ae9313e2500136213",
    ],
)

rpm(
    name = "passt-0__0__caret__20250415.g8ec1341-1.el9.aarch64",
    sha256 = "5476728540a2c925918362024905081680f55885e0c5d01658cd36a5a2321085",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/passt-0%5E20250415.g8ec1341-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5476728540a2c925918362024905081680f55885e0c5d01658cd36a5a2321085",
    ],
)

rpm(
    name = "passt-0__0__caret__20250415.g8ec1341-1.el9.s390x",
    sha256 = "efe849c9884788bb8bb509bc030a8ec546475bfb32a4f365a4277a938096efd0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/passt-0%5E20250415.g8ec1341-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/efe849c9884788bb8bb509bc030a8ec546475bfb32a4f365a4277a938096efd0",
    ],
)

rpm(
    name = "passt-0__0__caret__20250415.g8ec1341-1.el9.x86_64",
    sha256 = "171bf1df52e67027b9a250c44d2cafa46e0707521861be8a9e1839389ed06a14",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/passt-0%5E20250415.g8ec1341-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/171bf1df52e67027b9a250c44d2cafa46e0707521861be8a9e1839389ed06a14",
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
    name = "policycoreutils-0__3.6-2.1.el9.x86_64",
    sha256 = "a87874363af6432b1c96b40f8b79b90616df22bff3bd4f9aa39da24f5bddd3e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.6-2.1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a87874363af6432b1c96b40f8b79b90616df22bff3bd4f9aa39da24f5bddd3e9",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-3.el9.aarch64",
    sha256 = "e9f13425f0a7f4900d971ffaa2363bb6353b1990dac260bbbf6c2bbda5d6e83d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.6-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e9f13425f0a7f4900d971ffaa2363bb6353b1990dac260bbbf6c2bbda5d6e83d",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-3.el9.s390x",
    sha256 = "02dd6d0f44100d9c5f73a8102a0b3dbdc90c0fa142b78ab10a9dd8eb93cfd725",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/policycoreutils-3.6-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/02dd6d0f44100d9c5f73a8102a0b3dbdc90c0fa142b78ab10a9dd8eb93cfd725",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-3.el9.x86_64",
    sha256 = "6a108397ed0aa3a7ad3be130c11e48c4f234316c64a97061a665480eb210cc04",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.6-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a108397ed0aa3a7ad3be130c11e48c4f234316c64a97061a665480eb210cc04",
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
    name = "publicsuffix-list-dafsa-0__20210518-3.el9.s390x",
    sha256 = "992c17312bf5f144ec17b3c9733ab180c6c3641323d2deaf7c13e6bd1971f7a6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/publicsuffix-list-dafsa-20210518-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/992c17312bf5f144ec17b3c9733ab180c6c3641323d2deaf7c13e6bd1971f7a6",
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
    name = "python3-0__3.9.21-2.el9.aarch64",
    sha256 = "91e1a87884c23995332a26133d50b133c1a001195ba0381de8149bf7e15bfbf8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.21-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/91e1a87884c23995332a26133d50b133c1a001195ba0381de8149bf7e15bfbf8",
    ],
)

rpm(
    name = "python3-0__3.9.21-2.el9.s390x",
    sha256 = "f769a241663f94179b28c139714dbba4d9366f77f15cb57f1d40d6df86c7a767",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-3.9.21-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f769a241663f94179b28c139714dbba4d9366f77f15cb57f1d40d6df86c7a767",
    ],
)

rpm(
    name = "python3-0__3.9.21-2.el9.x86_64",
    sha256 = "86c570c2497d8499be137b5051ad245100b6a19031e32e7ce14c3f1e891aa875",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.21-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86c570c2497d8499be137b5051ad245100b6a19031e32e7ce14c3f1e891aa875",
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
    name = "python3-libs-0__3.9.21-2.el9.aarch64",
    sha256 = "3519ee6de773b698816952a236d23371ae3dfdec9d50c8e8481b963fd2c0f92e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.21-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3519ee6de773b698816952a236d23371ae3dfdec9d50c8e8481b963fd2c0f92e",
    ],
)

rpm(
    name = "python3-libs-0__3.9.21-2.el9.s390x",
    sha256 = "82e326bd86d263ed0f33d026486e73b935f76f538d1ddd07b827f82d638830df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-libs-3.9.21-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/82e326bd86d263ed0f33d026486e73b935f76f538d1ddd07b827f82d638830df",
    ],
)

rpm(
    name = "python3-libs-0__3.9.21-2.el9.x86_64",
    sha256 = "768f9b7a1eab353288efa7e6f283a5f44020927a6790173471681fdc0d53a362",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.21-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/768f9b7a1eab353288efa7e6f283a5f44020927a6790173471681fdc0d53a362",
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
    name = "python3-setuptools-wheel-0__53.0.0-14.el9.aarch64",
    sha256 = "f48fc3928f427b3675bc52a9259010b64d6c3f2db6ae5885c13b5ecc7f69ef81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-53.0.0-14.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f48fc3928f427b3675bc52a9259010b64d6c3f2db6ae5885c13b5ecc7f69ef81",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-14.el9.s390x",
    sha256 = "f48fc3928f427b3675bc52a9259010b64d6c3f2db6ae5885c13b5ecc7f69ef81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-setuptools-wheel-53.0.0-14.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f48fc3928f427b3675bc52a9259010b64d6c3f2db6ae5885c13b5ecc7f69ef81",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-14.el9.x86_64",
    sha256 = "f48fc3928f427b3675bc52a9259010b64d6c3f2db6ae5885c13b5ecc7f69ef81",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-53.0.0-14.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f48fc3928f427b3675bc52a9259010b64d6c3f2db6ae5885c13b5ecc7f69ef81",
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
    name = "qemu-img-17__9.1.0-15.el9.x86_64",
    sha256 = "6149224d6968142db7c12330dd4d9f6882af2ad73a97e591214a3869603b663f",
    urls = ["https://storage.googleapis.com/builddeps/6149224d6968142db7c12330dd4d9f6882af2ad73a97e591214a3869603b663f"],
)

rpm(
    name = "qemu-img-17__9.1.0-19.el9.aarch64",
    sha256 = "eecb36dd4c18c7d60b6a24b6583d31cc068aa9639f8a3a2860276794963dd0a9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eecb36dd4c18c7d60b6a24b6583d31cc068aa9639f8a3a2860276794963dd0a9",
    ],
)

rpm(
    name = "qemu-img-17__9.1.0-19.el9.s390x",
    sha256 = "81d0d805a6e43207691e2bc6042706c00a997edcf63b40159fdf69c9b9450dd0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-img-9.1.0-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/81d0d805a6e43207691e2bc6042706c00a997edcf63b40159fdf69c9b9450dd0",
    ],
)

rpm(
    name = "qemu-img-17__9.1.0-19.el9.x86_64",
    sha256 = "ce34587d1bd0d97788291595e8f768f287c91f173a40578d1ecf8b2f95d66db5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce34587d1bd0d97788291595e8f768f287c91f173a40578d1ecf8b2f95d66db5",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.1.0-15.el9.x86_64",
    sha256 = "345b3dae626a756f160321e025774d3d3e193a767388e69ffc832ea75988b166",
    urls = ["https://storage.googleapis.com/builddeps/345b3dae626a756f160321e025774d3d3e193a767388e69ffc832ea75988b166"],
)

rpm(
    name = "qemu-kvm-common-17__9.1.0-19.el9.aarch64",
    sha256 = "01554c96ea43856036e47fc6895787ec30cea6978a9d47d2d6c7e851f107e194",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01554c96ea43856036e47fc6895787ec30cea6978a9d47d2d6c7e851f107e194",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.1.0-19.el9.s390x",
    sha256 = "8e3500ea2aff72d73e11d8d7b49db73de3deb9dbd892320341697bd4adeaf7b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-common-9.1.0-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8e3500ea2aff72d73e11d8d7b49db73de3deb9dbd892320341697bd4adeaf7b1",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.1.0-19.el9.x86_64",
    sha256 = "eaafbd7066d377311b2c82223e2571296e60052ef5ec5eaa07e2b916726406e4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eaafbd7066d377311b2c82223e2571296e60052ef5ec5eaa07e2b916726406e4",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.1.0-15.el9.x86_64",
    sha256 = "aa36521b947a78d2d06d90e1a8f5d74bab5ffbbb6d8ca8d939497477c4878565",
    urls = ["https://storage.googleapis.com/builddeps/aa36521b947a78d2d06d90e1a8f5d74bab5ffbbb6d8ca8d939497477c4878565"],
)

rpm(
    name = "qemu-kvm-core-17__9.1.0-19.el9.aarch64",
    sha256 = "89a8de4b8406bf95c77aaa1e48769af68a5e4646ac5958a35394395a66c44837",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/89a8de4b8406bf95c77aaa1e48769af68a5e4646ac5958a35394395a66c44837",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.1.0-19.el9.s390x",
    sha256 = "b33497718f0a645d33e9cd0cb80d901cb92041a83dafe35315d81422a26e5845",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-core-9.1.0-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b33497718f0a645d33e9cd0cb80d901cb92041a83dafe35315d81422a26e5845",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.1.0-19.el9.x86_64",
    sha256 = "7853218eb84ee0b36055efd4043901af21cc9f18e3ee3960e96ff6594ba55a9c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7853218eb84ee0b36055efd4043901af21cc9f18e3ee3960e96ff6594ba55a9c",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__9.1.0-19.el9.aarch64",
    sha256 = "5621548934b7bf9e97815aeac73f2c91a3fcdea9c073d3cbd295ecd2de9289fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5621548934b7bf9e97815aeac73f2c91a3fcdea9c073d3cbd295ecd2de9289fb",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__9.1.0-19.el9.s390x",
    sha256 = "f70781a51889ac1e3a573de6050ce5cb009020f3e61769ec0476224835acd62c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-9.1.0-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f70781a51889ac1e3a573de6050ce5cb009020f3e61769ec0476224835acd62c",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__9.1.0-19.el9.x86_64",
    sha256 = "28fb96790e97670f095ef18aca963ba3e78cfbf0566cc05d458c17d3d98d3d0c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/28fb96790e97670f095ef18aca963ba3e78cfbf0566cc05d458c17d3d98d3d0c",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-ccw-17__9.1.0-19.el9.s390x",
    sha256 = "a1febc5db65cbb730cfc93a793a618386272bf7a6f5eec608962ef6d16074e99",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-9.1.0-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a1febc5db65cbb730cfc93a793a618386272bf7a6f5eec608962ef6d16074e99",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__9.1.0-19.el9.aarch64",
    sha256 = "346a927a8957dae8e28628e2b98f65e81db2ea80a1d79a5797ac50377be9be5a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/346a927a8957dae8e28628e2b98f65e81db2ea80a1d79a5797ac50377be9be5a",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__9.1.0-19.el9.x86_64",
    sha256 = "9aa591852507c5a6d2f99802ca22b13952198b9704fb2e1c8d813140995afd6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9aa591852507c5a6d2f99802ca22b13952198b9704fb2e1c8d813140995afd6b",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-vga-17__9.1.0-19.el9.x86_64",
    sha256 = "aff5dd2d60bfc1f670430b627a46e6273fe4307eb3efa73152f2ddf3e2d999f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-vga-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aff5dd2d60bfc1f670430b627a46e6273fe4307eb3efa73152f2ddf3e2d999f3",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__9.1.0-19.el9.aarch64",
    sha256 = "e4ca3610d6ac9511c93e84f300d23a4b14c3190c4b40573f4c2aa56694fd6e71",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e4ca3610d6ac9511c93e84f300d23a4b14c3190c4b40573f4c2aa56694fd6e71",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__9.1.0-19.el9.s390x",
    sha256 = "415673ecafed2255b795bbf74404598f63ad4e83efe69e8f958a707c99d1baa5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-9.1.0-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/415673ecafed2255b795bbf74404598f63ad4e83efe69e8f958a707c99d1baa5",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__9.1.0-19.el9.x86_64",
    sha256 = "cabaa65fcd35fd06ab6dec41b4da47ec57859cc7547bb7d817896e907fa70313",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cabaa65fcd35fd06ab6dec41b4da47ec57859cc7547bb7d817896e907fa70313",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__9.1.0-19.el9.aarch64",
    sha256 = "21d2239d7e5696f812eaf9e7ffec858a81cb483aa1a55a3cb87b26861d367359",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-9.1.0-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/21d2239d7e5696f812eaf9e7ffec858a81cb483aa1a55a3cb87b26861d367359",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__9.1.0-19.el9.x86_64",
    sha256 = "d969c650d212338d4550b88cd451d321392b035256dca4859d11cf7cecce6786",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-9.1.0-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d969c650d212338d4550b88cd451d321392b035256dca4859d11cf7cecce6786",
    ],
)

rpm(
    name = "qemu-pr-helper-17__9.1.0-23.el9.aarch64",
    sha256 = "33fa7c11e6bf10d0a962109fad138706fdb27a0591801afa47288676c662183e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-9.1.0-23.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/33fa7c11e6bf10d0a962109fad138706fdb27a0591801afa47288676c662183e",
    ],
)

rpm(
    name = "qemu-pr-helper-17__9.1.0-23.el9.x86_64",
    sha256 = "e8a8ffe71bd37191c7f31111f389767e6731b06e57510a2392812b459ca93106",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-9.1.0-23.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e8a8ffe71bd37191c7f31111f389767e6731b06e57510a2392812b459ca93106",
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
    name = "rpm-0__4.16.1.3-37.el9.x86_64",
    sha256 = "84caf776cfb5175fbe960dd8bb4bd10d799c45c3c0fd9d6b01bdf4d0c254d40d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-37.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84caf776cfb5175fbe960dd8bb4bd10d799c45c3c0fd9d6b01bdf4d0c254d40d",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-38.el9.aarch64",
    sha256 = "1f74e063aebc04c24c8385995d620af5bd13473f01fd39c62d785fb7d8e4fefc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-4.16.1.3-38.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1f74e063aebc04c24c8385995d620af5bd13473f01fd39c62d785fb7d8e4fefc",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-38.el9.s390x",
    sha256 = "786004c72803bb818d5ee9d86cb0b4a601cafcbcc861e34d5ef65840b1f2ecee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-4.16.1.3-38.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/786004c72803bb818d5ee9d86cb0b4a601cafcbcc861e34d5ef65840b1f2ecee",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-38.el9.x86_64",
    sha256 = "d1c1ebb6ec19eb9a128e4c469ff072324bf42b1775b408e8bf39dd349cbf94cf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-38.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d1c1ebb6ec19eb9a128e4c469ff072324bf42b1775b408e8bf39dd349cbf94cf",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-37.el9.x86_64",
    sha256 = "ff504743e1b532c3825d1c6d4d72109a998de862f3d8e4896b49aecd3f33d3ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-37.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff504743e1b532c3825d1c6d4d72109a998de862f3d8e4896b49aecd3f33d3ed",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-38.el9.aarch64",
    sha256 = "f62d0ade3ed29931ca3fa062f11e5d69084a814652a4f60f19e289f5ccb2816e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.16.1.3-38.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f62d0ade3ed29931ca3fa062f11e5d69084a814652a4f60f19e289f5ccb2816e",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-38.el9.s390x",
    sha256 = "5f53b6865dc6a3859984aec2f910fb7de22c05c69f6dc6f92dcec4eaa90308b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-libs-4.16.1.3-38.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5f53b6865dc6a3859984aec2f910fb7de22c05c69f6dc6f92dcec4eaa90308b1",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-38.el9.x86_64",
    sha256 = "65c9bedb1431d48741af288dca623f56ab17564633e8581a004f9c8c1ee6cb7f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-38.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65c9bedb1431d48741af288dca623f56ab17564633e8581a004f9c8c1ee6cb7f",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-37.el9.x86_64",
    sha256 = "0abb8313e99600887e851d249d914968a0b5623aead736831077f7d53be25837",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-37.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0abb8313e99600887e851d249d914968a0b5623aead736831077f7d53be25837",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-38.el9.aarch64",
    sha256 = "5b38c8da112be1f8c058c9050f7c655aaa9823b686e4787b08425466902013f1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.16.1.3-38.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5b38c8da112be1f8c058c9050f7c655aaa9823b686e4787b08425466902013f1",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-38.el9.s390x",
    sha256 = "ba4781c136c37bdc66927b90718673d85b47db99bc32f393ce57c51724b86746",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-plugin-selinux-4.16.1.3-38.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ba4781c136c37bdc66927b90718673d85b47db99bc32f393ce57c51724b86746",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-38.el9.x86_64",
    sha256 = "704ec34409878f6c328e4a700e9ec5e905a8305f6028741b80f35ad397b5d816",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-38.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/704ec34409878f6c328e4a700e9ec5e905a8305f6028741b80f35ad397b5d816",
    ],
)

rpm(
    name = "scrub-0__2.6.1-4.el9.s390x",
    sha256 = "eadebb92f9a6955e7f3391ea9964c1e66f84afeeff1abd23b1c4137fdc21625c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/scrub-2.6.1-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eadebb92f9a6955e7f3391ea9964c1e66f84afeeff1abd23b1c4137fdc21625c",
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
    name = "seabios-0__1.16.3-4.el9.x86_64",
    sha256 = "017b84c1189a9ec40b029d4a3ea5add67bceb0a48f1b3d9d135e1cc0fe465002",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-1.16.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/017b84c1189a9ec40b029d4a3ea5add67bceb0a48f1b3d9d135e1cc0fe465002",
    ],
)

rpm(
    name = "seabios-bin-0__1.16.3-4.el9.x86_64",
    sha256 = "95b4f37519a9c83f493b0109be461fbdf7205ca0eb3b572bec6ce10c2f5f6d00",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seabios-bin-1.16.3-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/95b4f37519a9c83f493b0109be461fbdf7205ca0eb3b572bec6ce10c2f5f6d00",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.16.3-4.el9.x86_64",
    sha256 = "8bdae1cc5c6ea4ed2347180d9f94dabe9891264a612e3afed2fb4ad86686eb43",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/seavgabios-bin-1.16.3-4.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8bdae1cc5c6ea4ed2347180d9f94dabe9891264a612e3afed2fb4ad86686eb43",
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
    name = "selinux-policy-0__38.1.53-2.el9.x86_64",
    sha256 = "6840efbf87f7f4782c332e0e0a3e3567075a804c070b1d501ff7e7a44a09448c",
    urls = ["https://storage.googleapis.com/builddeps/6840efbf87f7f4782c332e0e0a3e3567075a804c070b1d501ff7e7a44a09448c"],
)

rpm(
    name = "selinux-policy-0__38.1.58-1.el9.aarch64",
    sha256 = "2d90febb2a69188bf707a5f9cdd79f7825d9cb8de65450c1334c1541e87e2a89",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-38.1.58-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2d90febb2a69188bf707a5f9cdd79f7825d9cb8de65450c1334c1541e87e2a89",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.58-1.el9.s390x",
    sha256 = "2d90febb2a69188bf707a5f9cdd79f7825d9cb8de65450c1334c1541e87e2a89",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-38.1.58-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2d90febb2a69188bf707a5f9cdd79f7825d9cb8de65450c1334c1541e87e2a89",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.58-1.el9.x86_64",
    sha256 = "2d90febb2a69188bf707a5f9cdd79f7825d9cb8de65450c1334c1541e87e2a89",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-38.1.58-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2d90febb2a69188bf707a5f9cdd79f7825d9cb8de65450c1334c1541e87e2a89",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.53-2.el9.x86_64",
    sha256 = "b9f921bdc764af3b8c5c8580fc9db4f75b0fb3b2c0a3ea1f541536de091664b1",
    urls = ["https://storage.googleapis.com/builddeps/b9f921bdc764af3b8c5c8580fc9db4f75b0fb3b2c0a3ea1f541536de091664b1"],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.58-1.el9.aarch64",
    sha256 = "155501311c71a3f2be12e90b6efb21c0760e7749797f33d562cbb8b06c34ddd7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-38.1.58-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/155501311c71a3f2be12e90b6efb21c0760e7749797f33d562cbb8b06c34ddd7",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.58-1.el9.s390x",
    sha256 = "155501311c71a3f2be12e90b6efb21c0760e7749797f33d562cbb8b06c34ddd7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-38.1.58-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/155501311c71a3f2be12e90b6efb21c0760e7749797f33d562cbb8b06c34ddd7",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.58-1.el9.x86_64",
    sha256 = "155501311c71a3f2be12e90b6efb21c0760e7749797f33d562cbb8b06c34ddd7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-38.1.58-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/155501311c71a3f2be12e90b6efb21c0760e7749797f33d562cbb8b06c34ddd7",
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
    name = "shadow-utils-2__4.9-12.el9.x86_64",
    sha256 = "23f14143a188cf9bf8a0315f930fbeeb0ad34c58357007a52d112c5f8b6029e0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23f14143a188cf9bf8a0315f930fbeeb0ad34c58357007a52d112c5f8b6029e0",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-13.el9.aarch64",
    sha256 = "a0e8a4165126cf7ae9bb5dc0fa3e856df19ab2f5ab7e054574c976b54d7fe3f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.9-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a0e8a4165126cf7ae9bb5dc0fa3e856df19ab2f5ab7e054574c976b54d7fe3f0",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-13.el9.s390x",
    sha256 = "7bf498ff31f8546dad8681ad18acd019dcb44b72ed9554a5ed2eae06ba3695db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/shadow-utils-4.9-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7bf498ff31f8546dad8681ad18acd019dcb44b72ed9554a5ed2eae06ba3695db",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-13.el9.x86_64",
    sha256 = "fefa596d747ab8b17a99b054f29eec1e7e54ba09a7e26c7e81ae230022797ba3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fefa596d747ab8b17a99b054f29eec1e7e54ba09a7e26c7e81ae230022797ba3",
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
    name = "sqlite-libs-0__3.34.1-7.el9.x86_64",
    sha256 = "eddc9570ff3c2f672034888a57eac371e166671fee8300c3c4976324d502a00f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eddc9570ff3c2f672034888a57eac371e166671fee8300c3c4976324d502a00f",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-8.el9.aarch64",
    sha256 = "9fe7fee63ca05f390da0efc7d80d7395322b646e67fa20cd74bc09a930a05103",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.34.1-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9fe7fee63ca05f390da0efc7d80d7395322b646e67fa20cd74bc09a930a05103",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-8.el9.s390x",
    sha256 = "ecd0187ec3eb76967cedd6a6fb18d1ea2f8871fcb8eb0bab571daebb25d008b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sqlite-libs-3.34.1-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ecd0187ec3eb76967cedd6a6fb18d1ea2f8871fcb8eb0bab571daebb25d008b8",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-8.el9.x86_64",
    sha256 = "c5f5c7b797740bcc5252bbc49ce90b283e2eb8be56fa28b875068d03aec4cfb2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c5f5c7b797740bcc5252bbc49ce90b283e2eb8be56fa28b875068d03aec4cfb2",
    ],
)

rpm(
    name = "sssd-client-0__2.9.7-1.el9.aarch64",
    sha256 = "7beecc5aaffbaf263207fe8fe3253abe0ba76dbeae8f9e80a4e1b2c0091664ab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.9.7-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7beecc5aaffbaf263207fe8fe3253abe0ba76dbeae8f9e80a4e1b2c0091664ab",
    ],
)

rpm(
    name = "sssd-client-0__2.9.7-1.el9.s390x",
    sha256 = "5f4780bbd942795530d09bb0756b85bc326140a2b5dcd586575b421e8fa2de27",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sssd-client-2.9.7-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5f4780bbd942795530d09bb0756b85bc326140a2b5dcd586575b421e8fa2de27",
    ],
)

rpm(
    name = "sssd-client-0__2.9.7-1.el9.x86_64",
    sha256 = "bb2fdab3f1397cde1511c76bc1769a32b74d69c2e4c34bfac2e27c8a516f84b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.9.7-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb2fdab3f1397cde1511c76bc1769a32b74d69c2e4c34bfac2e27c8a516f84b4",
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
    name = "systemd-0__252-51.el9.x86_64",
    sha256 = "c5e5ae6f65f085c9f811a2a7950920eecb0c7ddf3d82c3f63b5662231cfc5de0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-51.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c5e5ae6f65f085c9f811a2a7950920eecb0c7ddf3d82c3f63b5662231cfc5de0",
    ],
)

rpm(
    name = "systemd-0__252-53.el9.aarch64",
    sha256 = "522e7b48e01bd5b97db036327c2e02f246a362cf95a9cca780c704f2bc995d55",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-252-53.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/522e7b48e01bd5b97db036327c2e02f246a362cf95a9cca780c704f2bc995d55",
    ],
)

rpm(
    name = "systemd-0__252-53.el9.s390x",
    sha256 = "ad328594663e92545aa5d83391610f5de3e7d85067b7ebe9ed760d97ae94aa9f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-252-53.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ad328594663e92545aa5d83391610f5de3e7d85067b7ebe9ed760d97ae94aa9f",
    ],
)

rpm(
    name = "systemd-0__252-53.el9.x86_64",
    sha256 = "914c148a1b6f9bba358beb3b6a8158c3227e030e4bda10daaa29150c8a4dec14",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/914c148a1b6f9bba358beb3b6a8158c3227e030e4bda10daaa29150c8a4dec14",
    ],
)

rpm(
    name = "systemd-container-0__252-51.el9.x86_64",
    sha256 = "653fcd14047fb557e3a3f5da47c83d6ceb2194169f3ef42a27566bb4e2102dde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-51.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/653fcd14047fb557e3a3f5da47c83d6ceb2194169f3ef42a27566bb4e2102dde",
    ],
)

rpm(
    name = "systemd-container-0__252-53.el9.aarch64",
    sha256 = "0df7250cbec391d05ed7cfa8f78813dfd88bf1b22e730fdfa1adddced1be9755",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-252-53.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0df7250cbec391d05ed7cfa8f78813dfd88bf1b22e730fdfa1adddced1be9755",
    ],
)

rpm(
    name = "systemd-container-0__252-53.el9.s390x",
    sha256 = "cf147ee5bc6b9046ef863fd776443e492e753e1e7e2fdf2ffbb8b7750ca1b67f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-container-252-53.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cf147ee5bc6b9046ef863fd776443e492e753e1e7e2fdf2ffbb8b7750ca1b67f",
    ],
)

rpm(
    name = "systemd-container-0__252-53.el9.x86_64",
    sha256 = "bc325ad6c4e3ebaf7d80b5dd3223142e5253e0ef23e74cd9b421b1193823f4d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc325ad6c4e3ebaf7d80b5dd3223142e5253e0ef23e74cd9b421b1193823f4d8",
    ],
)

rpm(
    name = "systemd-libs-0__252-51.el9.x86_64",
    sha256 = "a9d02a16bbc778ad3a2b46b8740fa821df065cdacd6ba8570c3301dacad79f0f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-51.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a9d02a16bbc778ad3a2b46b8740fa821df065cdacd6ba8570c3301dacad79f0f",
    ],
)

rpm(
    name = "systemd-libs-0__252-53.el9.aarch64",
    sha256 = "9bb3ecf74dfd855357be53e0cddadf2d776e305dce80fe83d6627e07f981baf6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-252-53.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9bb3ecf74dfd855357be53e0cddadf2d776e305dce80fe83d6627e07f981baf6",
    ],
)

rpm(
    name = "systemd-libs-0__252-53.el9.s390x",
    sha256 = "aa4fe4903d3642160c8f4e04e2301bc1334b8c105f240e752523063cf066fb82",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-libs-252-53.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/aa4fe4903d3642160c8f4e04e2301bc1334b8c105f240e752523063cf066fb82",
    ],
)

rpm(
    name = "systemd-libs-0__252-53.el9.x86_64",
    sha256 = "0080a5aa77131cdf69316cd5fe416cad5dbae98b612dd957bd16f328f69301b5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0080a5aa77131cdf69316cd5fe416cad5dbae98b612dd957bd16f328f69301b5",
    ],
)

rpm(
    name = "systemd-pam-0__252-51.el9.x86_64",
    sha256 = "26014995c59a6d43c7cc0ba55b829cc14513491bc901fe60faf5a10b43c8fb03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-51.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/26014995c59a6d43c7cc0ba55b829cc14513491bc901fe60faf5a10b43c8fb03",
    ],
)

rpm(
    name = "systemd-pam-0__252-53.el9.aarch64",
    sha256 = "978eedb2909378caeae810043129a73bb2ae66c9f57ce0167bb636007ebaaf37",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-252-53.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/978eedb2909378caeae810043129a73bb2ae66c9f57ce0167bb636007ebaaf37",
    ],
)

rpm(
    name = "systemd-pam-0__252-53.el9.s390x",
    sha256 = "0ecb8cb190584cd689ca7f67ffdf67f469d217f0c4dc36b85bca7176355253d9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-pam-252-53.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0ecb8cb190584cd689ca7f67ffdf67f469d217f0c4dc36b85bca7176355253d9",
    ],
)

rpm(
    name = "systemd-pam-0__252-53.el9.x86_64",
    sha256 = "9dd7d630e51b4d644d37cd2d472acb153b9ac3ac8b1d5adffdf724a3c3775008",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-53.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9dd7d630e51b4d644d37cd2d472acb153b9ac3ac8b1d5adffdf724a3c3775008",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-51.el9.x86_64",
    sha256 = "afa84ccbac79bb3950cca69bbfa9868429ed3aa464c96f5b2a15405a9c49f56c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-51.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/afa84ccbac79bb3950cca69bbfa9868429ed3aa464c96f5b2a15405a9c49f56c",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-53.el9.aarch64",
    sha256 = "dbcc4bfc5e8b4580ad294c3774328f8daad6d1258173b522ef118654e9e1003e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-252-53.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/dbcc4bfc5e8b4580ad294c3774328f8daad6d1258173b522ef118654e9e1003e",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-53.el9.s390x",
    sha256 = "dbcc4bfc5e8b4580ad294c3774328f8daad6d1258173b522ef118654e9e1003e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-rpm-macros-252-53.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/dbcc4bfc5e8b4580ad294c3774328f8daad6d1258173b522ef118654e9e1003e",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-53.el9.x86_64",
    sha256 = "dbcc4bfc5e8b4580ad294c3774328f8daad6d1258173b522ef118654e9e1003e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-53.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/dbcc4bfc5e8b4580ad294c3774328f8daad6d1258173b522ef118654e9e1003e",
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
    name = "tzdata-0__2025a-1.el9.x86_64",
    sha256 = "655945e6a0e95b960a422828bc1cb3bac2232fe9b76590e35ad00069097f087a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2025a-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/655945e6a0e95b960a422828bc1cb3bac2232fe9b76590e35ad00069097f087a",
    ],
)

rpm(
    name = "tzdata-0__2025b-1.el9.aarch64",
    sha256 = "6d98a9bab563c68269f9c6f8f85e886d7f80b2053b10e5956ab0a9de161ae98e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2025b-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6d98a9bab563c68269f9c6f8f85e886d7f80b2053b10e5956ab0a9de161ae98e",
    ],
)

rpm(
    name = "tzdata-0__2025b-1.el9.s390x",
    sha256 = "6d98a9bab563c68269f9c6f8f85e886d7f80b2053b10e5956ab0a9de161ae98e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tzdata-2025b-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6d98a9bab563c68269f9c6f8f85e886d7f80b2053b10e5956ab0a9de161ae98e",
    ],
)

rpm(
    name = "tzdata-0__2025b-1.el9.x86_64",
    sha256 = "6d98a9bab563c68269f9c6f8f85e886d7f80b2053b10e5956ab0a9de161ae98e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2025b-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6d98a9bab563c68269f9c6f8f85e886d7f80b2053b10e5956ab0a9de161ae98e",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-18.el9.aarch64",
    sha256 = "abe2556799289a4bc132a3e92c0d692fc2d7069622a9ce87169ccc3d3998d35d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/unbound-libs-1.16.2-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/abe2556799289a4bc132a3e92c0d692fc2d7069622a9ce87169ccc3d3998d35d",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-18.el9.s390x",
    sha256 = "010fa3346f205f7136709208c75994bbd1456f8ef0aec8831d65638748d48f8a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/unbound-libs-1.16.2-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/010fa3346f205f7136709208c75994bbd1456f8ef0aec8831d65638748d48f8a",
    ],
)

rpm(
    name = "unbound-libs-0__1.16.2-18.el9.x86_64",
    sha256 = "6f8e07616232f482dfbdc8523959ea258bd211700c29b0f0a14ffbee31a80e35",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/unbound-libs-1.16.2-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6f8e07616232f482dfbdc8523959ea258bd211700c29b0f0a14ffbee31a80e35",
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
    name = "util-linux-0__2.37.4-21.el9.aarch64",
    sha256 = "434fe9c8a283246524ced5b6637ef2f95ad0e1ab20cbeabc582592b30f69b0b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/434fe9c8a283246524ced5b6637ef2f95ad0e1ab20cbeabc582592b30f69b0b8",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-21.el9.s390x",
    sha256 = "bc5501f5a26586828a30cc97870f2f907d44c6107d7974c620ff920718b4d6bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bc5501f5a26586828a30cc97870f2f907d44c6107d7974c620ff920718b4d6bd",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-21.el9.x86_64",
    sha256 = "77f5aa59c85c1231bde7f64a7e348bb7b4675a04e385e219275abbd748037075",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/77f5aa59c85c1231bde7f64a7e348bb7b4675a04e385e219275abbd748037075",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-21.el9.aarch64",
    sha256 = "1066ec56c02d69030fb95b0749f8d21e7748449ffa95bba4fb0e12ec938ca1d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.37.4-21.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1066ec56c02d69030fb95b0749f8d21e7748449ffa95bba4fb0e12ec938ca1d8",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-21.el9.s390x",
    sha256 = "62553c85f62156441de1b4e5da924182b3bafabef47fe803539396fd44fadb43",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-core-2.37.4-21.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/62553c85f62156441de1b4e5da924182b3bafabef47fe803539396fd44fadb43",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-21.el9.x86_64",
    sha256 = "1858fbea657a9edce414fd98b8260b37ef521769f06830fccc7831094ec04154",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1858fbea657a9edce414fd98b8260b37ef521769f06830fccc7831094ec04154",
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
    name = "vim-minimal-2__8.2.2637-22.el9.aarch64",
    sha256 = "321cbfe7e8a5354bdc99c56b3fed34dca626c7d0337bb5505c1d7302f87f8677",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.2.2637-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/321cbfe7e8a5354bdc99c56b3fed34dca626c7d0337bb5505c1d7302f87f8677",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-22.el9.s390x",
    sha256 = "96950333e54291cb96add201bd335581722e753221b4c1eee447a02838041984",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/vim-minimal-8.2.2637-22.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/96950333e54291cb96add201bd335581722e753221b4c1eee447a02838041984",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-22.el9.x86_64",
    sha256 = "471e89c0410ca9204cabe323435b7fc787fffc8ef28734e324130f969caca462",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-22.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/471e89c0410ca9204cabe323435b7fc787fffc8ef28734e324130f969caca462",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.0-1.el9.aarch64",
    sha256 = "7b1503b55bc88dd4af2a2dd7d44d0d36a7f80ae4765baf353fa2a03bb9482b12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/virtiofsd-1.13.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7b1503b55bc88dd4af2a2dd7d44d0d36a7f80ae4765baf353fa2a03bb9482b12",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.0-1.el9.s390x",
    sha256 = "4b1912675a305a39f0ffa047a1d6745a7fff22304b08d064979f014771b64bbe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/virtiofsd-1.13.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4b1912675a305a39f0ffa047a1d6745a7fff22304b08d064979f014771b64bbe",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.0-1.el9.x86_64",
    sha256 = "531c66110a700566b703da037abda2b32a1860a7fa615c54ef645dcfffeaf9bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/virtiofsd-1.13.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/531c66110a700566b703da037abda2b32a1860a7fa615c54ef645dcfffeaf9bd",
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
    name = "xorriso-0__1.5.4-5.el9.aarch64",
    sha256 = "e9413affb36cac66415d4a3c6ab0a787f96c0ab2ebeac84d5336a98f286156ba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/xorriso-1.5.4-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e9413affb36cac66415d4a3c6ab0a787f96c0ab2ebeac84d5336a98f286156ba",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-5.el9.s390x",
    sha256 = "35a558dc2a2e221e46c5e2a9f04886f9f77cb42d6f8116834760926648c9a70d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/xorriso-1.5.4-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/35a558dc2a2e221e46c5e2a9f04886f9f77cb42d6f8116834760926648c9a70d",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-5.el9.x86_64",
    sha256 = "15e4269000f4f3dc15046fca6a4d80077ba8a2f5e74c095b9d6e0007aa78c251",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/xorriso-1.5.4-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/15e4269000f4f3dc15046fca6a4d80077ba8a2f5e74c095b9d6e0007aa78c251",
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
    name = "zstd-0__1.5.5-1.el9.aarch64",
    sha256 = "bdb442cb624d05b2da828d0894ed8440d53baa3e1523cc37e2598a3dda0193bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/zstd-1.5.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bdb442cb624d05b2da828d0894ed8440d53baa3e1523cc37e2598a3dda0193bd",
    ],
)

rpm(
    name = "zstd-0__1.5.5-1.el9.s390x",
    sha256 = "09c2cb5f2226cf3e8d084d68cd99b14989e3fabf0860959e71823cb72cf75b13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/zstd-1.5.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/09c2cb5f2226cf3e8d084d68cd99b14989e3fabf0860959e71823cb72cf75b13",
    ],
)

rpm(
    name = "zstd-0__1.5.5-1.el9.x86_64",
    sha256 = "6635550f3a87a734a069b3598e33a16174d14dca3ca52b9ef4bff78ea6f91c16",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/zstd-1.5.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6635550f3a87a734a069b3598e33a16174d14dca3ca52b9ef4bff78ea6f91c16",
    ],
)
