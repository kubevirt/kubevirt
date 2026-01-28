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

http_archive(
    name = "rules_oci",
    sha256 = "acbf8f40e062f707f8754e914dcb0013803c6e5e3679d3e05b571a9f5c7e0b43",
    strip_prefix = "rules_oci-2.0.1",
    urls = [
        "https://github.com/bazel-contrib/rules_oci/releases/download/v2.0.1/rules_oci-v2.0.1.tar.gz",
        "https://storage.googleapis.com/builddeps/acbf8f40e062f707f8754e914dcb0013803c6e5e3679d3e05b571a9f5c7e0b43",
    ],
)

load("@rules_oci//oci:dependencies.bzl", "rules_oci_dependencies")

rules_oci_dependencies()

load("@rules_oci//oci:repositories.bzl", "oci_register_toolchains")

oci_register_toolchains(
    name = "oci",
)

load("@rules_oci//oci:pull.bzl", "oci_pull")

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
    name = "platforms",
    sha256 = "3384eb1c30762704fbe38e440204e114154086c8fc8a8c2e3e28441028c019a8",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/platforms/releases/download/1.0.0/platforms-1.0.0.tar.gz",
        "https://github.com/bazelbuild/platforms/releases/download/1.0.0/platforms-1.0.0.tar.gz",
        "https://storage.googleapis.com/builddeps/3384eb1c30762704fbe38e440204e114154086c8fc8a8c2e3e28441028c019a8",
    ],
)

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "130739704540caa14e77c54810b9f01d6d9ae897d53eedceb40fd6b75efc3c23",
    urls = [
        "https://github.com/bazel-contrib/rules_go/releases/download/v0.54.1/rules_go-v0.54.1.zip",
        "https://storage.googleapis.com/builddeps/130739704540caa14e77c54810b9f01d6d9ae897d53eedceb40fd6b75efc3c23",
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
    sha256 = "b760f7fe75173886007f7c2e616a21241208f3d90e8657dc65d36a771e916b6a",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.39.1/bazel-gazelle-v0.39.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.39.1/bazel-gazelle-v0.39.1.tar.gz",
        "https://storage.googleapis.com/builddeps/b760f7fe75173886007f7c2e616a21241208f3d90e8657dc65d36a771e916b6a",
    ],
)

http_archive(
    name = "rules_pkg",
    sha256 = "d20c951960ed77cb7b341c2a59488534e494d5ad1d30c4818c736d57772a9fef",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_pkg/releases/download/1.0.1/rules_pkg-1.0.1.tar.gz",
        "https://github.com/bazelbuild/rules_pkg/releases/download/1.0.1/rules_pkg-1.0.1.tar.gz",
        "https://storage.googleapis.com/builddeps/d20c951960ed77cb7b341c2a59488534e494d5ad1d30c4818c736d57772a9fef",
    ],
)

load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")

rules_pkg_dependencies()

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
    sha256 = "0a4b9740da1839ded674c8f5012c069b235b101f1eaa2552a4721287808541af",
    strip_prefix = "bazeldnf-v0.5.9-2",
    urls = [
        "https://github.com/brianmcarey/bazeldnf/releases/download/v0.5.9-2/bazeldnf-v0.5.9-2.tar.gz",
        "https://storage.googleapis.com/builddeps/0a4b9740da1839ded674c8f5012c069b235b101f1eaa2552a4721287808541af",
    ],
)

load("@bazeldnf//bazeldnf:defs.bzl", "rpm")
load(
    "@bazeldnf//bazeldnf:repositories.bzl",
    "bazeldnf_dependencies",
    "bazeldnf_register_toolchains",
)
load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

bazeldnf_dependencies()

bazeldnf_register_toolchains(
    name = "bazeldnf_prebuilt",
)

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.24.9",
    nogo = "@//:nogo_vet",
)

load("@com_github_ash2k_bazel_tools//goimports:deps.bzl", "goimports_dependencies")

goimports_dependencies()

load(
    "@bazel_gazelle//:deps.bzl",
    "gazelle_dependencies",
    "go_repository",
)

gazelle_dependencies(go_sdk = "go_sdk")

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

# Pull go_image_base
oci_pull(
    name = "go_image_base",
    digest = "sha256:0ba6aa6b538aeae3d0f716ea8837703eb147173cd673241662e89adb794da829",
    image = "gcr.io/distroless/base-debian12",
)

oci_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:9ee08ca352647dad1511153afb18f4a6dbb4f56bafc7d618d0082c16a14cfdf1",
    image = "gcr.io/distroless/base-debian12",
)

oci_pull(
    name = "go_image_base_s390x",
    digest = "sha256:6e2e356c462d69668a0313bf45ed3de614e9d4e0b9c03fa081d3bcae143a58ba",
    image = "gcr.io/distroless/base-debian12",
)

# Pull fedora container-disk preconfigured with ci tooling
# like stress and qemu guest agent pre-configured
# TODO build fedora_with_test_tooling for multi-arch
oci_pull(
    name = "fedora_with_test_tooling",
    digest = "sha256:897af945d1c58366086d5933ae4f341a5f1413b88e6c7f2b659436adc5d0f522",
    image = "quay.io/kubevirtci/fedora-with-test-tooling",
)

oci_pull(
    name = "alpine_with_test_tooling",
    digest = "sha256:882450d4a0141d29422d049390cae59176c1a9ddf75bd9f3ecdd5a9081c7d95b",
    image = "quay.io/kubevirtci/alpine-with-test-tooling-container-disk",
)

oci_pull(
    name = "fedora_with_test_tooling_aarch64",
    digest = "sha256:3d5a2a95f7f9382dc6730073fe19a6b1bc668b424c362339c88c6a13dff2ef49",
    image = "quay.io/kubevirtci/fedora-with-test-tooling",
)

oci_pull(
    name = "fedora_with_test_tooling_s390x",
    digest = "sha256:3d9f468750d90845a81608ea13c85237ea295c6295c911a99dc5e0504c8bc05b",
    image = "quay.io/kubevirtci/fedora-with-test-tooling",
)

oci_pull(
    name = "alpine-ext-kernel-boot-demo-container-base",
    digest = "sha256:de4bc8de772ff7570e6dda871ea9cdd502feeeff1973f16f84bfbd60ff8f4149",
    image = "quay.io/kubevirt/alpine-ext-kernel-boot-demo",
)

# TODO build fedora_realtime for multi-arch
oci_pull(
    name = "fedora_realtime",
    digest = "sha256:f91379d202a5493aba9ce06870b5d1ada2c112f314530c9820a9ad07426aa565",
    image = "quay.io/kubevirt/fedora-realtime-container-disk",
)

oci_pull(
    name = "busybox",
    digest = "sha256:545e6a6310a27636260920bc07b994a299b6708a1b26910cfefd335fdfb60d2b",
    image = "registry.k8s.io/busybox",
)

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

