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
    digest = "sha256:d5f7dca58e3db53d1de502bd1a747ecb1110cf6b0773af129f951ee11e2e3ed4",
    image = "gcr.io/distroless/base-debian12",
)

oci_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:ba2aeab48a1dadbd47ac4ce37b7f6084043a8f59172f8b73a3ede3c3e1a71be4",
    image = "gcr.io/distroless/base-debian12",
)

oci_pull(
    name = "go_image_base_s390x",
    digest = "sha256:214b82df32d6dfe855715b7ce56dfe72a777da2c1e0b9fe47efb8cbc5cce5484",
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
    digest = "sha256:8c8e8bb6cd81c75e492c678abb3e5f186d52eba2174ebabc328316250acfea58",
    image = "quay.io/kubevirtci/alpine-with-test-tooling-container-disk",
)

oci_pull(
    name = "alpine_with_test_tooling_s390x",
    digest = "sha256:1a52903133c00507607e8a82308a34923e89288d852762b9f4d5da227767e965",
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
    name = "s390x-guestless-kernel",
    digest = "sha256:3bf6fc355fc9718c088c4c881b2d35a073ea274f6b16dc42236ef5e29db2215d",
    image = "quay.io/kubevirt/s390x-guestless-kernel",
)

oci_pull(
    name = "alpine-ext-kernel-boot-demo-container-base",
    digest = "sha256:bccd990554f55623d96fa70bc7efc553dd617523ebca76919b917ad3ee616c1d",
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

load("//images/virt-template:deps.bzl", "virt_template_images")

virt_template_images()

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
    name = "audit-libs-0__3.1.5-8.el9.aarch64",
    sha256 = "83af8b9a4dd0539f10ffda2ee09fe4a93eaf45fb12a3fc4aaea5899025f12cac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/audit-libs-3.1.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/83af8b9a4dd0539f10ffda2ee09fe4a93eaf45fb12a3fc4aaea5899025f12cac",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-8.el9.s390x",
    sha256 = "267f9e2528d2ca70c83abd80002aab8284ea93da3f2d87be0d13a0ec7efb13c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/audit-libs-3.1.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/267f9e2528d2ca70c83abd80002aab8284ea93da3f2d87be0d13a0ec7efb13c9",
    ],
)

rpm(
    name = "audit-libs-0__3.1.5-8.el9.x86_64",
    sha256 = "f970ce7fc0589c0a7b37784c6fc602a35a771db811f8061b8b8af2f4e9b46349",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/audit-libs-3.1.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f970ce7fc0589c0a7b37784c6fc602a35a771db811f8061b8b8af2f4e9b46349",
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
    name = "binutils-0__2.35.2-69.el9.aarch64",
    sha256 = "5276381ae395c0d5cf7414cb7bfd3bc14ab93f83233238f1e2b0aa8703eb159b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-69.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5276381ae395c0d5cf7414cb7bfd3bc14ab93f83233238f1e2b0aa8703eb159b",
    ],
)

rpm(
    name = "binutils-0__2.35.2-69.el9.s390x",
    sha256 = "2312fba1a49412188587995f932f0a8cd18232ccb148a4cb8b5784edb5eddc09",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-2.35.2-69.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2312fba1a49412188587995f932f0a8cd18232ccb148a4cb8b5784edb5eddc09",
    ],
)

rpm(
    name = "binutils-0__2.35.2-69.el9.x86_64",
    sha256 = "8f5d12203960d696a941987548e6085642eaed291d4858a308c8be247570bf27",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-69.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8f5d12203960d696a941987548e6085642eaed291d4858a308c8be247570bf27",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-69.el9.aarch64",
    sha256 = "b3e833b049af21ad1467586c5975e494ba99a7072320a8f7d81a24edbb7071b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-69.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b3e833b049af21ad1467586c5975e494ba99a7072320a8f7d81a24edbb7071b2",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-69.el9.s390x",
    sha256 = "df03b983980a4ae5a236dcb480be7a85c3979fc28bb534b2f11dec0cb21e5ebb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-gold-2.35.2-69.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/df03b983980a4ae5a236dcb480be7a85c3979fc28bb534b2f11dec0cb21e5ebb",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-69.el9.x86_64",
    sha256 = "2bdfd486a24ec343d34085524a9a78a174b6beb746a3ee78ea49d70fddde8f20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-69.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2bdfd486a24ec343d34085524a9a78a174b6beb746a3ee78ea49d70fddde8f20",
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
    name = "bzip2-0__1.0.8-11.el9.aarch64",
    sha256 = "b10f34223776bbf7a1b433a92ba6f87a4a58d893acae3a56d38654c3790b5c03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.8-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b10f34223776bbf7a1b433a92ba6f87a4a58d893acae3a56d38654c3790b5c03",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-11.el9.s390x",
    sha256 = "e94ba526c13a81e1046ff33234dd37bce3a2a9b1e1a15b5cd33434b6b0d399b4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-1.0.8-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e94ba526c13a81e1046ff33234dd37bce3a2a9b1e1a15b5cd33434b6b0d399b4",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-11.el9.x86_64",
    sha256 = "27de96d7fb8285910bdd240cf6c6a842863b9a01d20b25874f5d121a16239441",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.8-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/27de96d7fb8285910bdd240cf6c6a842863b9a01d20b25874f5d121a16239441",
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
    name = "bzip2-libs-0__1.0.8-11.el9.aarch64",
    sha256 = "fafc0f2b7632774d4c07264c73eebbe52f815b4c81056bd44b944e5255cb20bb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.8-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fafc0f2b7632774d4c07264c73eebbe52f815b4c81056bd44b944e5255cb20bb",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-11.el9.s390x",
    sha256 = "e9746e7bd442b4104b726e239cf3b7b87400824c7094de6d11f356da4c27593f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/bzip2-libs-1.0.8-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e9746e7bd442b4104b726e239cf3b7b87400824c7094de6d11f356da4c27593f",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-11.el9.x86_64",
    sha256 = "e1f4ca1a16276a6ede5f67cab8d8d2920b98531419af7498f5fded85835e0fca",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.8-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e1f4ca1a16276a6ede5f67cab8d8d2920b98531419af7498f5fded85835e0fca",
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
    name = "ca-certificates-0__2025.2.80_v9.0.305-91.el9.aarch64",
    sha256 = "489fdf258344892412ff2f10d0c1c849c45d5a15c4628abda33f325a42dd1bb0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ca-certificates-2025.2.80_v9.0.305-91.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/489fdf258344892412ff2f10d0c1c849c45d5a15c4628abda33f325a42dd1bb0",
    ],
)

rpm(
    name = "ca-certificates-0__2025.2.80_v9.0.305-91.el9.s390x",
    sha256 = "489fdf258344892412ff2f10d0c1c849c45d5a15c4628abda33f325a42dd1bb0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ca-certificates-2025.2.80_v9.0.305-91.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/489fdf258344892412ff2f10d0c1c849c45d5a15c4628abda33f325a42dd1bb0",
    ],
)

rpm(
    name = "ca-certificates-0__2025.2.80_v9.0.305-91.el9.x86_64",
    sha256 = "489fdf258344892412ff2f10d0c1c849c45d5a15c4628abda33f325a42dd1bb0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ca-certificates-2025.2.80_v9.0.305-91.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/489fdf258344892412ff2f10d0c1c849c45d5a15c4628abda33f325a42dd1bb0",
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
    urls = ["https://storage.googleapis.com/builddeps/8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca"],
)

rpm(
    name = "centos-gpg-keys-0__9.0-35.el9.aarch64",
    sha256 = "77e4a14370a63fc7b42d5dd7953654d9ae791a8a41e2388788559d65182da8fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/77e4a14370a63fc7b42d5dd7953654d9ae791a8a41e2388788559d65182da8fb",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-35.el9.s390x",
    sha256 = "77e4a14370a63fc7b42d5dd7953654d9ae791a8a41e2388788559d65182da8fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-gpg-keys-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/77e4a14370a63fc7b42d5dd7953654d9ae791a8a41e2388788559d65182da8fb",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-35.el9.x86_64",
    sha256 = "77e4a14370a63fc7b42d5dd7953654d9ae791a8a41e2388788559d65182da8fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/77e4a14370a63fc7b42d5dd7953654d9ae791a8a41e2388788559d65182da8fb",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-26.el9.x86_64",
    sha256 = "3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    urls = ["https://storage.googleapis.com/builddeps/3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3"],
)

rpm(
    name = "centos-stream-release-0__9.0-35.el9.aarch64",
    sha256 = "1c9986cabdf106cae20bc548d11aec1af6446ed670c6226b38a2b0383493c184",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-release-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1c9986cabdf106cae20bc548d11aec1af6446ed670c6226b38a2b0383493c184",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-35.el9.s390x",
    sha256 = "1c9986cabdf106cae20bc548d11aec1af6446ed670c6226b38a2b0383493c184",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-release-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1c9986cabdf106cae20bc548d11aec1af6446ed670c6226b38a2b0383493c184",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-35.el9.x86_64",
    sha256 = "1c9986cabdf106cae20bc548d11aec1af6446ed670c6226b38a2b0383493c184",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-release-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1c9986cabdf106cae20bc548d11aec1af6446ed670c6226b38a2b0383493c184",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-26.el9.x86_64",
    sha256 = "eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    urls = ["https://storage.googleapis.com/builddeps/eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035"],
)

rpm(
    name = "centos-stream-repos-0__9.0-35.el9.aarch64",
    sha256 = "23f3d6d63dd948cf2b0b4ebb5562ccc0facca73bed907db9056fd3d42fdefa29",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/23f3d6d63dd948cf2b0b4ebb5562ccc0facca73bed907db9056fd3d42fdefa29",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-35.el9.s390x",
    sha256 = "23f3d6d63dd948cf2b0b4ebb5562ccc0facca73bed907db9056fd3d42fdefa29",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-repos-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/23f3d6d63dd948cf2b0b4ebb5562ccc0facca73bed907db9056fd3d42fdefa29",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-35.el9.x86_64",
    sha256 = "23f3d6d63dd948cf2b0b4ebb5562ccc0facca73bed907db9056fd3d42fdefa29",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-35.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/23f3d6d63dd948cf2b0b4ebb5562ccc0facca73bed907db9056fd3d42fdefa29",
    ],
)

rpm(
    name = "checkpolicy-0__3.6-1.el9.aarch64",
    sha256 = "a96ddf25086443769d5febc9c737cede5ea8c790f33961aa512726e4f2072404",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/checkpolicy-3.6-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "checkpolicy-0__3.6-1.el9.s390x",
    sha256 = "898219a26e80f88feb6038cb99b6ed8202d6d7b5f1e1620eab9f91eb1cfc95f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/checkpolicy-3.6-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "checkpolicy-0__3.6-1.el9.x86_64",
    sha256 = "df808c446c615b5841124a15c0e1e1383dee2e7c520979636bdc4cfb9b9b8dcf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/checkpolicy-3.6-1.el9.x86_64.rpm",
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
    name = "cpp-0__11.5.0-14.el9.aarch64",
    sha256 = "6c14ab2a1cfa7fcaa55e1a6a1d35220c817010c89321c7e8654855cc9582b381",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6c14ab2a1cfa7fcaa55e1a6a1d35220c817010c89321c7e8654855cc9582b381",
    ],
)

rpm(
    name = "cpp-0__11.5.0-14.el9.s390x",
    sha256 = "168ccf4f3a4dad3eeccaaf2614cdae24a77c3a0ec1ee409d71e9fbc3dca12a23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/cpp-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/168ccf4f3a4dad3eeccaaf2614cdae24a77c3a0ec1ee409d71e9fbc3dca12a23",
    ],
)

rpm(
    name = "cpp-0__11.5.0-14.el9.x86_64",
    sha256 = "b2792e076b41cd6d044d341ae756575a51e0ab85a6dae375f1cb1d59cf47d921",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.5.0-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2792e076b41cd6d044d341ae756575a51e0ab85a6dae375f1cb1d59cf47d921",
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
    name = "cracklib-0__2.9.6-28.el9.aarch64",
    sha256 = "78dbd83e4de7c011dedc8071af056989dece25dae7605eb60703b219ebbeadc1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cracklib-2.9.6-28.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/78dbd83e4de7c011dedc8071af056989dece25dae7605eb60703b219ebbeadc1",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-28.el9.s390x",
    sha256 = "14006fd9132581ca7ab86b87eb4751efd25279bc60df48aced985002e401112d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cracklib-2.9.6-28.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/14006fd9132581ca7ab86b87eb4751efd25279bc60df48aced985002e401112d",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-28.el9.x86_64",
    sha256 = "aa659fc5fc1f40d9301850411e1e4cfb9351175e1879a1d404292cbd909982f0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cracklib-2.9.6-28.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aa659fc5fc1f40d9301850411e1e4cfb9351175e1879a1d404292cbd909982f0",
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
    name = "cracklib-dicts-0__2.9.6-28.el9.aarch64",
    sha256 = "3b449db83d1a649b93eff386e098ab01f24028b106827d9fef899abc99818b15",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/cracklib-dicts-2.9.6-28.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3b449db83d1a649b93eff386e098ab01f24028b106827d9fef899abc99818b15",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-28.el9.s390x",
    sha256 = "a0ac88ff592620ae37ea0826d59874f0f5a08828c02fcd514473302d15cf6c03",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/cracklib-dicts-2.9.6-28.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a0ac88ff592620ae37ea0826d59874f0f5a08828c02fcd514473302d15cf6c03",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-28.el9.x86_64",
    sha256 = "b0e372c09e6eb01d2de1316b7e59c79178c0eaee6d713004d7fe5fbc7e718603",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/cracklib-dicts-2.9.6-28.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b0e372c09e6eb01d2de1316b7e59c79178c0eaee6d713004d7fe5fbc7e718603",
    ],
)

rpm(
    name = "crypto-policies-0__20250128-1.git5269e22.el9.x86_64",
    sha256 = "f811d2c848f6f93a188f2d74d4ccd172e1dc88fa7919e8e203cf1df3d93571e1",
    urls = ["https://storage.googleapis.com/builddeps/f811d2c848f6f93a188f2d74d4ccd172e1dc88fa7919e8e203cf1df3d93571e1"],
)

rpm(
    name = "crypto-policies-0__20251126-1.gite9c4db2.el9.aarch64",
    sha256 = "38c1e40b477795017996db0683b72004a4810d88a320ae0554e6736b118c5c9a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20251126-1.gite9c4db2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/38c1e40b477795017996db0683b72004a4810d88a320ae0554e6736b118c5c9a",
    ],
)

rpm(
    name = "crypto-policies-0__20251126-1.gite9c4db2.el9.s390x",
    sha256 = "38c1e40b477795017996db0683b72004a4810d88a320ae0554e6736b118c5c9a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/crypto-policies-20251126-1.gite9c4db2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/38c1e40b477795017996db0683b72004a4810d88a320ae0554e6736b118c5c9a",
    ],
)

rpm(
    name = "crypto-policies-0__20251126-1.gite9c4db2.el9.x86_64",
    sha256 = "38c1e40b477795017996db0683b72004a4810d88a320ae0554e6736b118c5c9a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20251126-1.gite9c4db2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/38c1e40b477795017996db0683b72004a4810d88a320ae0554e6736b118c5c9a",
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
    name = "curl-minimal-0__7.76.1-40.el9.aarch64",
    sha256 = "a3de170776a05462a04ab6bfd8c66f4a032d70f53d34018013751eb2e0392657",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/curl-minimal-7.76.1-40.el9.aarch64.rpm",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-40.el9.s390x",
    sha256 = "fb571ec63ecacabed30de75cb81b048e4e01f3a0b521d91190fcd2412df7b6ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/curl-minimal-7.76.1-40.el9.s390x.rpm",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-40.el9.x86_64",
    sha256 = "94c55a702411b0bc2c6c9c1bb0ab794105785af58c5ad22597cc68536709a092",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/curl-minimal-7.76.1-40.el9.x86_64.rpm",
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
    name = "daxctl-libs-0__82-1.el9.x86_64",
    sha256 = "f3650df75436eebe1fd14369f161a7b15c8ab9f4ed6333b8a83e1be70dc185a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/daxctl-libs-82-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f3650df75436eebe1fd14369f161a7b15c8ab9f4ed6333b8a83e1be70dc185a3",
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
    name = "device-mapper-multipath-libs-0__0.8.7-44.el9.aarch64",
    sha256 = "5345a2ebb787fd142fc056675bd657c15f124733c85958f40dac9512aebdd80a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.7-44.el9.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-44.el9.x86_64",
    sha256 = "6b0c5ee67467eb2a0bb4cd3969878b6add9fe5f0e9dbfea4963e7d8ca239b17b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.7-44.el9.x86_64.rpm",
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
    name = "dmidecode-1__3.6-2.el9.aarch64",
    sha256 = "800b4e874e52f2d181eccb438ac2ef82185b938f67c80e65e72e1004df9fa575",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dmidecode-3.6-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/800b4e874e52f2d181eccb438ac2ef82185b938f67c80e65e72e1004df9fa575",
    ],
)

rpm(
    name = "dmidecode-1__3.6-2.el9.x86_64",
    sha256 = "11bdceab038ffa793efd223db04703bbac90d65c057831e83bc5380e6a16e959",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dmidecode-3.6-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11bdceab038ffa793efd223db04703bbac90d65c057831e83bc5380e6a16e959",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-8.el9.aarch64",
    sha256 = "e764033c6a78fba5f7f5a2cfe59d627aa2b6ff4962dee494b33ed7de9ef0ef51",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.46.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e764033c6a78fba5f7f5a2cfe59d627aa2b6ff4962dee494b33ed7de9ef0ef51",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-8.el9.s390x",
    sha256 = "b9048f417885369956a1668e204ca2499e2d46b2c61f62a34cb8caa7797e2ff1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/e2fsprogs-1.46.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b9048f417885369956a1668e204ca2499e2d46b2c61f62a34cb8caa7797e2ff1",
    ],
)

rpm(
    name = "e2fsprogs-0__1.46.5-8.el9.x86_64",
    sha256 = "a0b4500adf6c74516aeeb6aa2bf08a5a20508fc7ad4d241c7e686110abe17dbe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.46.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0b4500adf6c74516aeeb6aa2bf08a5a20508fc7ad4d241c7e686110abe17dbe",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-8.el9.aarch64",
    sha256 = "f8ec39d902f629559a263ff7238192887b8f7cc16815af5a4577b86627599919",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.46.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f8ec39d902f629559a263ff7238192887b8f7cc16815af5a4577b86627599919",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-8.el9.s390x",
    sha256 = "d279fad6453b9b5f90fc14181727594fb4aaf980b641976d049448ae978f27c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/e2fsprogs-libs-1.46.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d279fad6453b9b5f90fc14181727594fb4aaf980b641976d049448ae978f27c2",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.46.5-8.el9.x86_64",
    sha256 = "28841ef6789b99559061c236b30e680bd045650bd22180133a3815cceb65cc46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.46.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/28841ef6789b99559061c236b30e680bd045650bd22180133a3815cceb65cc46",
    ],
)

rpm(
    name = "edk2-aarch64-0__20241117-8.el9.aarch64",
    sha256 = "4d92929c0c6a83146955894e0a7da7b626d7872c2e339301f82cf4ecc24f21a0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20241117-8.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4d92929c0c6a83146955894e0a7da7b626d7872c2e339301f82cf4ecc24f21a0",
    ],
)

rpm(
    name = "edk2-ovmf-0__20241117-2.el9.x86_64",
    sha256 = "a64ed00fed189c823f533a013ce8f044a439066524fbb628b266fd898fe23172",
    urls = ["https://storage.googleapis.com/builddeps/a64ed00fed189c823f533a013ce8f044a439066524fbb628b266fd898fe23172"],
)

rpm(
    name = "edk2-ovmf-0__20241117-8.el9.s390x",
    sha256 = "6275c2657d403be296076d7a42d62e9253d42cde571f955126db728e339bab23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/edk2-ovmf-20241117-8.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6275c2657d403be296076d7a42d62e9253d42cde571f955126db728e339bab23",
    ],
)

rpm(
    name = "edk2-ovmf-0__20241117-8.el9.x86_64",
    sha256 = "6275c2657d403be296076d7a42d62e9253d42cde571f955126db728e339bab23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20241117-8.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6275c2657d403be296076d7a42d62e9253d42cde571f955126db728e339bab23",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.194-1.el9.aarch64",
    sha256 = "745a3f1dec43e34bad7ac3677472a57dd98a293bb5ed6d42c2a423163ae78b9f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.194-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/745a3f1dec43e34bad7ac3677472a57dd98a293bb5ed6d42c2a423163ae78b9f",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.194-1.el9.s390x",
    sha256 = "fb52d62ad9e8477833a5912bba1cccd9d9972ee7fa88c31b4d9867e324000699",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.194-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fb52d62ad9e8477833a5912bba1cccd9d9972ee7fa88c31b4d9867e324000699",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.194-1.el9.x86_64",
    sha256 = "40de0a46e149c1ed6bb79a19191f7279ebf429f05ee9693f6185ed8b56370caf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.194-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/40de0a46e149c1ed6bb79a19191f7279ebf429f05ee9693f6185ed8b56370caf",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.194-1.el9.aarch64",
    sha256 = "6d94e5a11b829a2e7aa57e28fc3bfd727a77e750e043583236b20f07544e5e3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.194-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6d94e5a11b829a2e7aa57e28fc3bfd727a77e750e043583236b20f07544e5e3a",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.194-1.el9.s390x",
    sha256 = "6d94e5a11b829a2e7aa57e28fc3bfd727a77e750e043583236b20f07544e5e3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.194-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6d94e5a11b829a2e7aa57e28fc3bfd727a77e750e043583236b20f07544e5e3a",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.194-1.el9.x86_64",
    sha256 = "6d94e5a11b829a2e7aa57e28fc3bfd727a77e750e043583236b20f07544e5e3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.194-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6d94e5a11b829a2e7aa57e28fc3bfd727a77e750e043583236b20f07544e5e3a",
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
    name = "elfutils-libelf-0__0.194-1.el9.aarch64",
    sha256 = "ac9cc272659364f6b60f3754b25fedb2e9aa1f8a3fd91eebde5f4e75ecc8510e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.194-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ac9cc272659364f6b60f3754b25fedb2e9aa1f8a3fd91eebde5f4e75ecc8510e",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.194-1.el9.s390x",
    sha256 = "d1ba973a8569fff460f72ae253a3c05d7a6592a9025bf998a5388eefeb7cf2b5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.194-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d1ba973a8569fff460f72ae253a3c05d7a6592a9025bf998a5388eefeb7cf2b5",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.194-1.el9.x86_64",
    sha256 = "c59294fcfe3a216267078a010f4cf7e0d191fd1a222f19bf3036ba1b0ce40e1f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.194-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c59294fcfe3a216267078a010f4cf7e0d191fd1a222f19bf3036ba1b0ce40e1f",
    ],
)

rpm(
    name = "elfutils-libs-0__0.194-1.el9.aarch64",
    sha256 = "78b614ff56d76403679094de597ce56f27a776ab8ed40ef399a0d022976a35b2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.194-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/78b614ff56d76403679094de597ce56f27a776ab8ed40ef399a0d022976a35b2",
    ],
)

rpm(
    name = "elfutils-libs-0__0.194-1.el9.s390x",
    sha256 = "a7bcd615b80395cd0b88db9406cd063273945972033ebccff541e1fa2d95699b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.194-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a7bcd615b80395cd0b88db9406cd063273945972033ebccff541e1fa2d95699b",
    ],
)

rpm(
    name = "elfutils-libs-0__0.194-1.el9.x86_64",
    sha256 = "432d99395a7f57c13a61b3cd987205714a5f83eb95cec3c9e344e1abab5a196e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.194-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/432d99395a7f57c13a61b3cd987205714a5f83eb95cec3c9e344e1abab5a196e",
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
    name = "expat-0__2.5.0-6.el9.aarch64",
    sha256 = "01f1ff2194173775ebbc1d00934152585a259c9a852e987e672d1810384e4786",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/expat-2.5.0-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/01f1ff2194173775ebbc1d00934152585a259c9a852e987e672d1810384e4786",
    ],
)

rpm(
    name = "expat-0__2.5.0-6.el9.s390x",
    sha256 = "6e85c05c7eacb3d964af391a67898919239b973d8094c442b917ea450391d25d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/expat-2.5.0-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6e85c05c7eacb3d964af391a67898919239b973d8094c442b917ea450391d25d",
    ],
)

rpm(
    name = "expat-0__2.5.0-6.el9.x86_64",
    sha256 = "39cffc5a3a75ccd06d4214f99e3d3a89dd79bee3532175ae38d37c14aad529fc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/expat-2.5.0-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/39cffc5a3a75ccd06d4214f99e3d3a89dd79bee3532175ae38d37c14aad529fc",
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
    name = "gcc-0__11.5.0-14.el9.aarch64",
    sha256 = "ab3bb73a4443fdef60969ae4d57cce670a88e4c73d8a758111bf713037eef286",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab3bb73a4443fdef60969ae4d57cce670a88e4c73d8a758111bf713037eef286",
    ],
)

rpm(
    name = "gcc-0__11.5.0-14.el9.s390x",
    sha256 = "0c816944b06c65f19d8ed958416554eee0f128f38bfda9f0951926917fafd8de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gcc-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0c816944b06c65f19d8ed958416554eee0f128f38bfda9f0951926917fafd8de",
    ],
)

rpm(
    name = "gcc-0__11.5.0-14.el9.x86_64",
    sha256 = "c0d0eb5639d870197ccb4cee6fbbb8bfc8e0038983285a9660369dc9651f0089",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.5.0-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c0d0eb5639d870197ccb4cee6fbbb8bfc8e0038983285a9660369dc9651f0089",
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
    name = "glib2-0__2.68.4-16.el9.x86_64",
    sha256 = "793cbb8b6f5885a3b8a501dd5e4c0fe19141c34beeb4410fbc680424ae02ed2d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/793cbb8b6f5885a3b8a501dd5e4c0fe19141c34beeb4410fbc680424ae02ed2d",
    ],
)

rpm(
    name = "glib2-0__2.68.4-19.el9.aarch64",
    sha256 = "5fc2f7510779708b553a13fc5f0de31fcfe384ce318295a8b9d3cc496b99905c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glib2-2.68.4-19.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5fc2f7510779708b553a13fc5f0de31fcfe384ce318295a8b9d3cc496b99905c",
    ],
)

rpm(
    name = "glib2-0__2.68.4-19.el9.s390x",
    sha256 = "eae096d247448db793b42cada5accb2176444752b1fe17560c8fc9135626de05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glib2-2.68.4-19.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eae096d247448db793b42cada5accb2176444752b1fe17560c8fc9135626de05",
    ],
)

rpm(
    name = "glib2-0__2.68.4-19.el9.x86_64",
    sha256 = "3128523dc47f5fdea4633b0166544de8fbda27b03165265ec0b6d360a056b169",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-19.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3128523dc47f5fdea4633b0166544de8fbda27b03165265ec0b6d360a056b169",
    ],
)

rpm(
    name = "glibc-0__2.34-168.el9.x86_64",
    sha256 = "e06212b1cac1d9fd9857a00ddefefe9fb9f406199cb84fdd1153589c15e16289",
    urls = ["https://storage.googleapis.com/builddeps/e06212b1cac1d9fd9857a00ddefefe9fb9f406199cb84fdd1153589c15e16289"],
)

rpm(
    name = "glibc-0__2.34-245.el9.aarch64",
    sha256 = "bf6a0f12d012662ae124aa18ed00d4f270709cda83bd60b16f6a93138bcd1e50",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-245.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bf6a0f12d012662ae124aa18ed00d4f270709cda83bd60b16f6a93138bcd1e50",
    ],
)

rpm(
    name = "glibc-0__2.34-245.el9.s390x",
    sha256 = "b0682a36512f5e61724aba0e1af041481454ee23d290e5978f3e5016615ef962",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-2.34-245.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b0682a36512f5e61724aba0e1af041481454ee23d290e5978f3e5016615ef962",
    ],
)

rpm(
    name = "glibc-0__2.34-245.el9.x86_64",
    sha256 = "b4a8c93a28d59a4070ee7a3f7f517d0ce4e9c11886477e6d1b5707718892817e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-245.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4a8c93a28d59a4070ee7a3f7f517d0ce4e9c11886477e6d1b5707718892817e",
    ],
)

rpm(
    name = "glibc-common-0__2.34-168.el9.x86_64",
    sha256 = "531650744909efd0284bf6c16a45dbaf455b214c0cac4197cf6d43e8c7d83af8",
    urls = ["https://storage.googleapis.com/builddeps/531650744909efd0284bf6c16a45dbaf455b214c0cac4197cf6d43e8c7d83af8"],
)

rpm(
    name = "glibc-common-0__2.34-245.el9.aarch64",
    sha256 = "4645238ef9126cd91fd4ff37fc681693479779de7cf14fae852605caf9423156",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-245.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4645238ef9126cd91fd4ff37fc681693479779de7cf14fae852605caf9423156",
    ],
)

rpm(
    name = "glibc-common-0__2.34-245.el9.s390x",
    sha256 = "2d58050c2282d9961bdf565a86e175fc14161369dc4a0631d491a23250b22990",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-common-2.34-245.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2d58050c2282d9961bdf565a86e175fc14161369dc4a0631d491a23250b22990",
    ],
)

rpm(
    name = "glibc-common-0__2.34-245.el9.x86_64",
    sha256 = "747ea3dbcdbe9b3b8c26ac70ea8e117a79e62f50df0b8754a645c2ef68e05be2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-245.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/747ea3dbcdbe9b3b8c26ac70ea8e117a79e62f50df0b8754a645c2ef68e05be2",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-245.el9.aarch64",
    sha256 = "35609ad1fd287efb4f5766467d5c19340606c1d60521371f82f6afb21de87b4e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-245.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/35609ad1fd287efb4f5766467d5c19340606c1d60521371f82f6afb21de87b4e",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-245.el9.s390x",
    sha256 = "eef697217b220b2ec5b4a5bf36082c7e2ca6796de72a82ac46e9c7a573674c4b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-devel-2.34-245.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eef697217b220b2ec5b4a5bf36082c7e2ca6796de72a82ac46e9c7a573674c4b",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-245.el9.x86_64",
    sha256 = "90abad549b44b8846cb6d3fbbaaeae5541ecf6d10a675bc4856d2fcbf1006a23",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-245.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/90abad549b44b8846cb6d3fbbaaeae5541ecf6d10a675bc4856d2fcbf1006a23",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-245.el9.s390x",
    sha256 = "8f8bf5112ff902be0dc02d55cf12d13bb3110434b6dd6c98ce9faa96e17cbce7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-headers-2.34-245.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8f8bf5112ff902be0dc02d55cf12d13bb3110434b6dd6c98ce9faa96e17cbce7",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-245.el9.x86_64",
    sha256 = "411bc52f3bc71684ffa69a0e3026ee3226a6162e16ae55f10bb1024dc877dda7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-245.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/411bc52f3bc71684ffa69a0e3026ee3226a6162e16ae55f10bb1024dc877dda7",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-168.el9.x86_64",
    sha256 = "991b6d7370b237a3d576536a517d01a1ccc997959f4ea30ba07bd779641f79e8",
    urls = ["https://storage.googleapis.com/builddeps/991b6d7370b237a3d576536a517d01a1ccc997959f4ea30ba07bd779641f79e8"],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-245.el9.aarch64",
    sha256 = "0b208faa1cefdafb7a71544746f0d858696d3c2e60861858c145098836ff3aa3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-245.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0b208faa1cefdafb7a71544746f0d858696d3c2e60861858c145098836ff3aa3",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-245.el9.s390x",
    sha256 = "a325deb26609c8f9746e9143f8a3a1d993cd5acf707ba5870e75353fedc87d15",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.34-245.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a325deb26609c8f9746e9143f8a3a1d993cd5acf707ba5870e75353fedc87d15",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-245.el9.x86_64",
    sha256 = "fe736927c6e91ef4d6ba4f1b467f373e6ab8a47c490c2a1e1d5508ae402f45fa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-245.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe736927c6e91ef4d6ba4f1b467f373e6ab8a47c490c2a1e1d5508ae402f45fa",
    ],
)

rpm(
    name = "glibc-static-0__2.34-245.el9.aarch64",
    sha256 = "0d31fb6e5f8f9dc8e3495ecc98d231764d205e4462b406b63055d8bef97cf085",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-245.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0d31fb6e5f8f9dc8e3495ecc98d231764d205e4462b406b63055d8bef97cf085",
    ],
)

rpm(
    name = "glibc-static-0__2.34-245.el9.s390x",
    sha256 = "28dd7aec003faae255ec84cd5781915082245caa5369c88466901cdba627f307",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/glibc-static-2.34-245.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/28dd7aec003faae255ec84cd5781915082245caa5369c88466901cdba627f307",
    ],
)

rpm(
    name = "glibc-static-0__2.34-245.el9.x86_64",
    sha256 = "033feeb7051d46932e7697a50798d48d5fa168f3f05284219346daddf3b5044f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-245.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/033feeb7051d46932e7697a50798d48d5fa168f3f05284219346daddf3b5044f",
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
    name = "gnupg2-0__2.3.3-5.el9.s390x",
    sha256 = "9cbb342b46df96e85e55919bee459b2fd5023642494eeb2466344b765c1802d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gnupg2-2.3.3-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9cbb342b46df96e85e55919bee459b2fd5023642494eeb2466344b765c1802d3",
    ],
)

rpm(
    name = "gnupg2-0__2.3.3-5.el9.x86_64",
    sha256 = "5628444d9a62a7b6b46951c5187ccf43cb4d9254a45ae225808c6ef7d28c027f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnupg2-2.3.3-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5628444d9a62a7b6b46951c5187ccf43cb4d9254a45ae225808c6ef7d28c027f",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-3.el9.aarch64",
    sha256 = "4b9e4757f999a9995f53e49577b9b3f3a5e0d683a227015c357e6d5603a87982",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gnutls-3.8.10-3.el9.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-3.el9.s390x",
    sha256 = "3d2808eadf410398ae827793fcc526af2bcc1e24a2551d33e61e1293786f1fb6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gnutls-3.8.10-3.el9.s390x.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-3.el9.x86_64",
    sha256 = "95b8f11db2d075d96817bad2ca9b131b78ad2836edc5a75f99f967c30249b1e7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnutls-3.8.10-3.el9.x86_64.rpm",
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
    name = "gnutls-dane-0__3.8.10-3.el9.aarch64",
    sha256 = "a41c588fc3f4d6c3686f872f493ba32b61a46cfd0cb8f0c7458d9f62e7b7c22a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.8.10-3.el9.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-3.el9.s390x",
    sha256 = "8e8793bc2a500256747f4745bed2d7b494854153d52e064a51aeb252c214dd26",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-dane-3.8.10-3.el9.s390x.rpm",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-3.el9.x86_64",
    sha256 = "9d8148e86c8013335fe8747bc113eda9ba6ad59921c057ab9715f6f9beb17288",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.8.10-3.el9.x86_64.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-3.el9.aarch64",
    sha256 = "d88cfcc208c54f0fb9182785356382295d188d680e190340a2c187aa15e49faa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.8.10-3.el9.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-3.el9.s390x",
    sha256 = "8938b2f992f04bd4a120d1052db7865c58dc42f88e35e917f3628ee00d5e091d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-utils-3.8.10-3.el9.s390x.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-3.el9.x86_64",
    sha256 = "e9e4aff0e53b81327f6bfd5c0a07de4be8900fd8e466c301c2b74c5af74fb508",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.8.10-3.el9.x86_64.rpm",
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
    name = "gsettings-desktop-schemas-0__40.0-8.el9.s390x",
    sha256 = "2de739236b8adb578b1dff03269c977b8ba9ad1ae6581793acf4614a70705638",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gsettings-desktop-schemas-40.0-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2de739236b8adb578b1dff03269c977b8ba9ad1ae6581793acf4614a70705638",
    ],
)

rpm(
    name = "gsettings-desktop-schemas-0__40.0-8.el9.x86_64",
    sha256 = "06b0fd5bd1b106371aa42cd47d8784f248c7df3962165cab4a58ff67d35512be",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gsettings-desktop-schemas-40.0-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/06b0fd5bd1b106371aa42cd47d8784f248c7df3962165cab4a58ff67d35512be",
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
    name = "hwdata-0__0.348-9.18.el9.x86_64",
    sha256 = "b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    ],
)

rpm(
    name = "hwdata-0__0.348-9.21.el9.s390x",
    sha256 = "3087aee4e6b637e0ea1d745931bf332ef27b9b9b24346544fbe2132bfdc21d49",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/hwdata-0.348-9.21.el9.noarch.rpm",
    ],
)

rpm(
    name = "hwdata-0__0.348-9.21.el9.x86_64",
    sha256 = "3087aee4e6b637e0ea1d745931bf332ef27b9b9b24346544fbe2132bfdc21d49",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.21.el9.noarch.rpm",
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
    name = "iproute-0__6.17.0-2.el9.aarch64",
    sha256 = "d388c9ac3e1ab6fdc97227710f35243bf63de4a4aa818222aa8f3fe241e8a9ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-6.17.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d388c9ac3e1ab6fdc97227710f35243bf63de4a4aa818222aa8f3fe241e8a9ae",
    ],
)

rpm(
    name = "iproute-0__6.17.0-2.el9.s390x",
    sha256 = "e838d01d8318ab25f322225b6cdb7dd906c5f1d35b95fa3c89cb8e8ae1450791",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iproute-6.17.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e838d01d8318ab25f322225b6cdb7dd906c5f1d35b95fa3c89cb8e8ae1450791",
    ],
)

rpm(
    name = "iproute-0__6.17.0-2.el9.x86_64",
    sha256 = "95f9c5dc7e6bc06ce26c1f905bb1c9114aee2df223a20d0806396664bee2cc67",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-6.17.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95f9c5dc7e6bc06ce26c1f905bb1c9114aee2df223a20d0806396664bee2cc67",
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
    name = "iproute-tc-0__6.17.0-2.el9.aarch64",
    sha256 = "906641c6eb987b2d1ae0e66ee81d50c88b1268894daa83884126a5005389f418",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iproute-tc-6.17.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/906641c6eb987b2d1ae0e66ee81d50c88b1268894daa83884126a5005389f418",
    ],
)

rpm(
    name = "iproute-tc-0__6.17.0-2.el9.s390x",
    sha256 = "9ee321e9f865810eb4098788af2231918716c1a7c2b8b34b103b5cc636b1f3d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iproute-tc-6.17.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9ee321e9f865810eb4098788af2231918716c1a7c2b8b34b103b5cc636b1f3d7",
    ],
)

rpm(
    name = "iproute-tc-0__6.17.0-2.el9.x86_64",
    sha256 = "8813251157fade8cfb191f476d568a8d8a86dcdd1537eeb5d6dfb636c630afcc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.17.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8813251157fade8cfb191f476d568a8d8a86dcdd1537eeb5d6dfb636c630afcc",
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
    name = "iputils-0__20210202-15.el9.aarch64",
    sha256 = "834bb57dc3ce91c41050e1fab4d2808a10a2b410b7c4a07cce3199c57181fe4d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/iputils-20210202-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/834bb57dc3ce91c41050e1fab4d2808a10a2b410b7c4a07cce3199c57181fe4d",
    ],
)

rpm(
    name = "iputils-0__20210202-15.el9.s390x",
    sha256 = "ed9f65c5d621415497674b76c248d79da0128d8fb1ee1838060129889f947360",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/iputils-20210202-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ed9f65c5d621415497674b76c248d79da0128d8fb1ee1838060129889f947360",
    ],
)

rpm(
    name = "iputils-0__20210202-15.el9.x86_64",
    sha256 = "225a2b191c1d5f9070dacb71ba3aed71f7247f0912a9ba18c6c08379587e4c37",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/iputils-20210202-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/225a2b191c1d5f9070dacb71ba3aed71f7247f0912a9ba18c6c08379587e4c37",
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
    name = "kernel-headers-0__5.14.0-681.el9.aarch64",
    sha256 = "a8788404793f2e7b262bc45a239ccd87adb19b7af5411f135a7ad7c5bf09c104",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/kernel-headers-5.14.0-681.el9.aarch64.rpm",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-681.el9.s390x",
    sha256 = "7acd194a781653c47a8d44b0ef0837f3ce140e56dff6db49d89e45e6e71924ef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/kernel-headers-5.14.0-681.el9.s390x.rpm",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-681.el9.x86_64",
    sha256 = "f79949070758e99d7a24ad886423c3d30c0c3ae178ec8f9431ce328a531b6859",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/kernel-headers-5.14.0-681.el9.x86_64.rpm",
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
    name = "kmod-0__28-11.el9.aarch64",
    sha256 = "857f2f75fc01e8228750d5aef82040c00520c0a91670c951620a4161a1ac2d77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/kmod-28-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/857f2f75fc01e8228750d5aef82040c00520c0a91670c951620a4161a1ac2d77",
    ],
)

rpm(
    name = "kmod-0__28-11.el9.s390x",
    sha256 = "3021be113a18631fb55f76bc2491733b3d9887fa563be60004569e2f36caab6c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/kmod-28-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3021be113a18631fb55f76bc2491733b3d9887fa563be60004569e2f36caab6c",
    ],
)

rpm(
    name = "kmod-0__28-11.el9.x86_64",
    sha256 = "c2cd4a2dcdc7cc55fbaf97f5641487091dbd13a5ea7974087ef8215d18fe2ffb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-28-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c2cd4a2dcdc7cc55fbaf97f5641487091dbd13a5ea7974087ef8215d18fe2ffb",
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
    name = "kmod-libs-0__28-11.el9.aarch64",
    sha256 = "68bd119a65b2d37388623c0e4a0a717b74787e1243244c8ffa0a448f42718ee4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/kmod-libs-28-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/68bd119a65b2d37388623c0e4a0a717b74787e1243244c8ffa0a448f42718ee4",
    ],
)

rpm(
    name = "kmod-libs-0__28-11.el9.s390x",
    sha256 = "e04b90f099224b2cb1dd28df4ff45aaa1982d26b2e2f04cb7bdcdf9b5a1306c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/kmod-libs-28-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e04b90f099224b2cb1dd28df4ff45aaa1982d26b2e2f04cb7bdcdf9b5a1306c4",
    ],
)

rpm(
    name = "kmod-libs-0__28-11.el9.x86_64",
    sha256 = "29d2fd267498f3e12d420a3d867483d32ce97d544327de983872f8ee89ec02b3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/kmod-libs-28-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/29d2fd267498f3e12d420a3d867483d32ce97d544327de983872f8ee89ec02b3",
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
    name = "less-0__590-5.el9.x86_64",
    sha256 = "46e11dfacb75a8d03047d82f44ae46b11d95da31e0ec1b3a8cc37a132b1c7cae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/less-590-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46e11dfacb75a8d03047d82f44ae46b11d95da31e0ec1b3a8cc37a132b1c7cae",
    ],
)

rpm(
    name = "less-0__590-6.el9.s390x",
    sha256 = "0f83cc72b28298a902ac58ab5e0a0dfb092a9a218a00f5baaa45f07ec0fbc09d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/less-590-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0f83cc72b28298a902ac58ab5e0a0dfb092a9a218a00f5baaa45f07ec0fbc09d",
    ],
)

rpm(
    name = "less-0__590-6.el9.x86_64",
    sha256 = "c6d44d94d48746cb6a909c5a929f69c86db83aef59635cbabfc5a4a176a388a0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/less-590-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c6d44d94d48746cb6a909c5a929f69c86db83aef59635cbabfc5a4a176a388a0",
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
    name = "libarchive-0__3.5.3-6.el9.aarch64",
    sha256 = "737b5493221fbebb7d6578de00ccf9ee29a2685ef6cb6bc4705a4680384b63ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libarchive-3.5.3-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/737b5493221fbebb7d6578de00ccf9ee29a2685ef6cb6bc4705a4680384b63ae",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-6.el9.s390x",
    sha256 = "343b64738033626728e908dc8c3e186c64fb9eb1752f35a2cf5fefa5156fa593",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libarchive-3.5.3-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/343b64738033626728e908dc8c3e186c64fb9eb1752f35a2cf5fefa5156fa593",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-6.el9.x86_64",
    sha256 = "a8f609bd9ce84d600fd4b6843c93025569cec32e3006d065a5e4323d2ff5696a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libarchive-3.5.3-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a8f609bd9ce84d600fd4b6843c93025569cec32e3006d065a5e4323d2ff5696a",
    ],
)

rpm(
    name = "libasan-0__11.5.0-14.el9.aarch64",
    sha256 = "e2c312abf632bae493826b22f82b80809c307b40e7993226655bae9c9bafa228",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e2c312abf632bae493826b22f82b80809c307b40e7993226655bae9c9bafa228",
    ],
)

rpm(
    name = "libasan-0__11.5.0-14.el9.s390x",
    sha256 = "d341357ba1a1add96ff4d59a34005e8618cca59c5d378f58148b9310d1401682",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libasan-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d341357ba1a1add96ff4d59a34005e8618cca59c5d378f58148b9310d1401682",
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
    name = "libatomic-0__11.5.0-14.el9.aarch64",
    sha256 = "9111ad5dcd16ac04ee06dbedbc730bdf438d58f1f16af2de5cd3cdb3e346efbe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9111ad5dcd16ac04ee06dbedbc730bdf438d58f1f16af2de5cd3cdb3e346efbe",
    ],
)

rpm(
    name = "libatomic-0__11.5.0-14.el9.s390x",
    sha256 = "b071b407128db07af4859894ee99fc7f1106be0a325bdb6242ca5d92752a3c5d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libatomic-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b071b407128db07af4859894ee99fc7f1106be0a325bdb6242ca5d92752a3c5d",
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
    name = "libbpf-2__1.5.0-1.el9.x86_64",
    sha256 = "5280ff8fe8a1f217b54abeed8b357c4c3fd47fbcfa928483806effac723accba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-1.5.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5280ff8fe8a1f217b54abeed8b357c4c3fd47fbcfa928483806effac723accba",
    ],
)

rpm(
    name = "libbpf-2__1.5.0-3.el9.aarch64",
    sha256 = "1aee4e44b6e5b6b0b888c2d994367efaa4c729521c925baef2e9f96c29813fa1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libbpf-1.5.0-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1aee4e44b6e5b6b0b888c2d994367efaa4c729521c925baef2e9f96c29813fa1",
    ],
)

rpm(
    name = "libbpf-2__1.5.0-3.el9.s390x",
    sha256 = "9087b5e9cc04fbffeaaa824b5fc050d41ad59cb7e0b3e2657810d4455308282b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libbpf-1.5.0-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9087b5e9cc04fbffeaaa824b5fc050d41ad59cb7e0b3e2657810d4455308282b",
    ],
)

rpm(
    name = "libbpf-2__1.5.0-3.el9.x86_64",
    sha256 = "27859976d0cc4123f8f717dbb8e9b4031f685ac648851eb290b54b069d51c868",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbpf-1.5.0-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/27859976d0cc4123f8f717dbb8e9b4031f685ac648851eb290b54b069d51c868",
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
    name = "libbrotli-0__1.0.9-9.el9.s390x",
    sha256 = "aae2d2967058c0d6da0697788b37338a83dbb7c0a83c42e526051e2b12ee2c3f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libbrotli-1.0.9-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/aae2d2967058c0d6da0697788b37338a83dbb7c0a83c42e526051e2b12ee2c3f",
    ],
)

rpm(
    name = "libbrotli-0__1.0.9-9.el9.x86_64",
    sha256 = "b9add1a745e5a7edae289502896a3ac075fda4d097eb73c4cee592363cd976ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libbrotli-1.0.9-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b9add1a745e5a7edae289502896a3ac075fda4d097eb73c4cee592363cd976ed",
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
    name = "libcap-0__2.48-10.el9.aarch64",
    sha256 = "7159fe4c1e6be9c8324632bfabcbc86ad8b7cb5105acb0b8a5c35774c93470f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcap-2.48-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7159fe4c1e6be9c8324632bfabcbc86ad8b7cb5105acb0b8a5c35774c93470f2",
    ],
)

rpm(
    name = "libcap-0__2.48-10.el9.s390x",
    sha256 = "2883f350016ef87b8f6aa33966023cb0f3c789bdcb36374037fc94096ee61bf7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcap-2.48-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2883f350016ef87b8f6aa33966023cb0f3c789bdcb36374037fc94096ee61bf7",
    ],
)

rpm(
    name = "libcap-0__2.48-10.el9.x86_64",
    sha256 = "bda5d981249ac16603228a4f544a15a140e1eed105ab1206da6bef9705cddee7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcap-2.48-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bda5d981249ac16603228a4f544a15a140e1eed105ab1206da6bef9705cddee7",
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
    name = "libcom_err-0__1.46.5-7.el9.x86_64",
    sha256 = "d11e18dbfddd56538c476851d98bd96795f34045e14ebe7b3285f225c4b4b189",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcom_err-1.46.5-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d11e18dbfddd56538c476851d98bd96795f34045e14ebe7b3285f225c4b4b189",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-8.el9.aarch64",
    sha256 = "7bf194e4f69e548566ff21b178ae1f47d5e00f064bfa492616e4dd42f812f2a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcom_err-1.46.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7bf194e4f69e548566ff21b178ae1f47d5e00f064bfa492616e4dd42f812f2a7",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-8.el9.s390x",
    sha256 = "b8aa8922757718f85c31dfc7c333434e576a52f9425e91f51db8fb082661c3ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcom_err-1.46.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b8aa8922757718f85c31dfc7c333434e576a52f9425e91f51db8fb082661c3ff",
    ],
)

rpm(
    name = "libcom_err-0__1.46.5-8.el9.x86_64",
    sha256 = "ef43794f39d49b69e12506722e432a497e7f96038e26cab2c34476aad4b3d413",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcom_err-1.46.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ef43794f39d49b69e12506722e432a497e7f96038e26cab2c34476aad4b3d413",
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
    name = "libcurl-minimal-0__7.76.1-31.el9.x86_64",
    sha256 = "6438485e38465ee944e25abedcf4a1761564fe5202f05a02c71e4c880255b539",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-31.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6438485e38465ee944e25abedcf4a1761564fe5202f05a02c71e4c880255b539",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-40.el9.aarch64",
    sha256 = "888e91e92f094f60d88b501ae583ec1e440fb3f9ba5faa41ba520f8c11623ea2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.76.1-40.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-40.el9.s390x",
    sha256 = "e9bb1bbb6e5239f02eded93f1774e535d0d215c91b59149c76a3101a26422f33",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcurl-minimal-7.76.1-40.el9.s390x.rpm",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-40.el9.x86_64",
    sha256 = "e2153e0e701347fd2cbf7660905b21ad9707342b4c1a715a8d49aaf555471d51",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-40.el9.x86_64.rpm",
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
    name = "libeconf-0__0.4.1-4.el9.x86_64",
    sha256 = "ed519cc2e9031e2bf03275b28c7cca6520ae916d0a7edbbc69f327c1b70ed6cc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libeconf-0.4.1-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed519cc2e9031e2bf03275b28c7cca6520ae916d0a7edbbc69f327c1b70ed6cc",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-5.el9.aarch64",
    sha256 = "40675233785bf5ffc0e97cd559efd2e9f4b8b8c392013bb6c9441c6e01b332d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libeconf-0.4.1-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/40675233785bf5ffc0e97cd559efd2e9f4b8b8c392013bb6c9441c6e01b332d6",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-5.el9.s390x",
    sha256 = "025025697e1f9d222fb6224be5f6a463ad630971c67f4a55ff8d70fc25780443",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libeconf-0.4.1-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/025025697e1f9d222fb6224be5f6a463ad630971c67f4a55ff8d70fc25780443",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-5.el9.x86_64",
    sha256 = "aba2474e57729f395e1918638270e7aa7cf8de8f3fc31b81f9412888320459e8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libeconf-0.4.1-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aba2474e57729f395e1918638270e7aa7cf8de8f3fc31b81f9412888320459e8",
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
    name = "libgcc-0__11.5.0-14.el9.aarch64",
    sha256 = "ed0598c9cb4f10406c662d17ac2367eeba1e207683953410146927bba3d92c46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ed0598c9cb4f10406c662d17ac2367eeba1e207683953410146927bba3d92c46",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-14.el9.s390x",
    sha256 = "6ccddf8ec532ddc49d7b857ad46cb5404efc30a1ba2d4af575db77c402efdb8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcc-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6ccddf8ec532ddc49d7b857ad46cb5404efc30a1ba2d4af575db77c402efdb8e",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-14.el9.x86_64",
    sha256 = "8e9b2f611466e02703348bfd7fbdc40035898c804dcc417b920d6ad77bf077e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.5.0-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e9b2f611466e02703348bfd7fbdc40035898c804dcc417b920d6ad77bf077e9",
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
    name = "libgomp-0__11.5.0-14.el9.aarch64",
    sha256 = "24d684550fda70ed7eaf592393f78249fb8b0f4879793cdcd36e08c7b9af4ff5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/24d684550fda70ed7eaf592393f78249fb8b0f4879793cdcd36e08c7b9af4ff5",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-14.el9.s390x",
    sha256 = "39a911cdfa8dfbef686c9f2ba74f17b0abafe30ef99f93cae00e3cd1d8d0571a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgomp-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/39a911cdfa8dfbef686c9f2ba74f17b0abafe30ef99f93cae00e3cd1d8d0571a",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-14.el9.x86_64",
    sha256 = "a2b750f8588cfb3d4caefea5f25a8585ed732775ab20dd243531ec136c21476d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.5.0-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a2b750f8588cfb3d4caefea5f25a8585ed732775ab20dd243531ec136c21476d",
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
    name = "libibverbs-0__54.0-1.el9.x86_64",
    sha256 = "b57effbc14e02e546a6e94bf8247f2dbfe24b5da0b95691ba3979b3652002b77",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libibverbs-54.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b57effbc14e02e546a6e94bf8247f2dbfe24b5da0b95691ba3979b3652002b77",
    ],
)

rpm(
    name = "libibverbs-0__61.0-2.el9.aarch64",
    sha256 = "ef89a4e7bb3dcaf8967b83b97b8f8f2820a7ab9d6e1a042c3babe316b10f91ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libibverbs-61.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ef89a4e7bb3dcaf8967b83b97b8f8f2820a7ab9d6e1a042c3babe316b10f91ee",
    ],
)

rpm(
    name = "libibverbs-0__61.0-2.el9.s390x",
    sha256 = "a515a7d9cea9d8d82ef327460795cd7fe41299684ae1e514e7fb3015db0f8ac8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libibverbs-61.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a515a7d9cea9d8d82ef327460795cd7fe41299684ae1e514e7fb3015db0f8ac8",
    ],
)

rpm(
    name = "libibverbs-0__61.0-2.el9.x86_64",
    sha256 = "f28e91c9e94d8a9dc0e418be86422c261d1018d6bae88ae66b51f1cce57c23c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libibverbs-61.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f28e91c9e94d8a9dc0e418be86422c261d1018d6bae88ae66b51f1cce57c23c4",
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
    name = "libpng-2__1.6.37-12.el9.x86_64",
    sha256 = "b3f3a689918dc50a9bc41c33abf1a36bdb8e4a707daac77a91e0814407b07ae3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpng-1.6.37-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3f3a689918dc50a9bc41c33abf1a36bdb8e4a707daac77a91e0814407b07ae3",
    ],
)

rpm(
    name = "libpng-2__1.6.37-13.el9.aarch64",
    sha256 = "ae248d590f2303b3e4eb8c8c14c9eb53a0a8cb353fb3c4e0321f944f66a4ad8b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libpng-1.6.37-13.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libpng-2__1.6.37-13.el9.s390x",
    sha256 = "270195bc1262e327312250c669ef24a97f2f7c6f41674e2e2a1239742e174d91",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libpng-1.6.37-13.el9.s390x.rpm",
    ],
)

rpm(
    name = "libpng-2__1.6.37-13.el9.x86_64",
    sha256 = "77f87e9a9ea407ea95204d07a39e1d6af50f6575f4bfa5c2352e983bb550650a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpng-1.6.37-13.el9.x86_64.rpm",
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
    name = "librdmacm-0__54.0-1.el9.x86_64",
    sha256 = "82d2d2eecace0a17f97e44e42d766a0ef5cf67f5c42e139c58e18406dfc38f4d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/librdmacm-54.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82d2d2eecace0a17f97e44e42d766a0ef5cf67f5c42e139c58e18406dfc38f4d",
    ],
)

rpm(
    name = "librdmacm-0__61.0-2.el9.aarch64",
    sha256 = "f02fbb25e313058137f32a3d20b7da2421c1cbc6cb35991158b0c400b970db3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/librdmacm-61.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f02fbb25e313058137f32a3d20b7da2421c1cbc6cb35991158b0c400b970db3a",
    ],
)

rpm(
    name = "librdmacm-0__61.0-2.el9.x86_64",
    sha256 = "b3ec91b7db56ab47cbdd3a8bd80a19e450a84f098aef29bedb082f198f4e17c1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/librdmacm-61.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3ec91b7db56ab47cbdd3a8bd80a19e450a84f098aef29bedb082f198f4e17c1",
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
    name = "libss-0__1.46.5-8.el9.aarch64",
    sha256 = "52ba79e72eb6feef925cd94e1989b879750a33a5f926cc48f576368211799796",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libss-1.46.5-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/52ba79e72eb6feef925cd94e1989b879750a33a5f926cc48f576368211799796",
    ],
)

rpm(
    name = "libss-0__1.46.5-8.el9.s390x",
    sha256 = "95a10e09d72daebb2cdae054cadd8e9ef3c3689dade4236723e8e69fdd674d3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libss-1.46.5-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/95a10e09d72daebb2cdae054cadd8e9ef3c3689dade4236723e8e69fdd674d3a",
    ],
)

rpm(
    name = "libss-0__1.46.5-8.el9.x86_64",
    sha256 = "095ae726757b2e9a0c17f4391b9667210c84f4fa72dbd65f006db78e47f3915d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libss-1.46.5-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/095ae726757b2e9a0c17f4391b9667210c84f4fa72dbd65f006db78e47f3915d",
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
    name = "libssh-0__0.10.4-17.el9.aarch64",
    sha256 = "420be5bba5c7c331c5c93d0c9b5a5bc26f7fcee99156e0e2ad0fbd21556c325f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-0.10.4-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/420be5bba5c7c331c5c93d0c9b5a5bc26f7fcee99156e0e2ad0fbd21556c325f",
    ],
)

rpm(
    name = "libssh-0__0.10.4-17.el9.s390x",
    sha256 = "6e1fb62c5a61b432f7e18255677155a9de94241c18bff17786a001c8776aec1c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libssh-0.10.4-17.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6e1fb62c5a61b432f7e18255677155a9de94241c18bff17786a001c8776aec1c",
    ],
)

rpm(
    name = "libssh-0__0.10.4-17.el9.x86_64",
    sha256 = "5bcf6ec9ec3cd108791fcb93d95c71209e1080598b5e6f45b9371a43e0b4519f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-0.10.4-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5bcf6ec9ec3cd108791fcb93d95c71209e1080598b5e6f45b9371a43e0b4519f",
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
    name = "libssh-config-0__0.10.4-17.el9.aarch64",
    sha256 = "ab4182e8ef3ffaf47951fe96b620d1d616d11d7f9c99fab4d8b6ba3dbf5d5bde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-config-0.10.4-17.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ab4182e8ef3ffaf47951fe96b620d1d616d11d7f9c99fab4d8b6ba3dbf5d5bde",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-17.el9.s390x",
    sha256 = "ab4182e8ef3ffaf47951fe96b620d1d616d11d7f9c99fab4d8b6ba3dbf5d5bde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libssh-config-0.10.4-17.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ab4182e8ef3ffaf47951fe96b620d1d616d11d7f9c99fab4d8b6ba3dbf5d5bde",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-17.el9.x86_64",
    sha256 = "ab4182e8ef3ffaf47951fe96b620d1d616d11d7f9c99fab4d8b6ba3dbf5d5bde",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-config-0.10.4-17.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ab4182e8ef3ffaf47951fe96b620d1d616d11d7f9c99fab4d8b6ba3dbf5d5bde",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.8-1.el9.aarch64",
    sha256 = "638b417e5c726de5fc158c03e0d67c9573a862d307660ec2954fb96285b86075",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.9.8-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/638b417e5c726de5fc158c03e0d67c9573a862d307660ec2954fb96285b86075",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.8-1.el9.s390x",
    sha256 = "8999405bd3fb1901922bcec02c450852c2e2b1ebe305e57c5afba15015408232",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.9.8-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8999405bd3fb1901922bcec02c450852c2e2b1ebe305e57c5afba15015408232",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.8-1.el9.x86_64",
    sha256 = "9a29f5bfe5f444071eda063ad9de94b00b6e5a9e3227505ef1b8ea7d11970d6a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.9.8-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a29f5bfe5f444071eda063ad9de94b00b6e5a9e3227505ef1b8ea7d11970d6a",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.8-1.el9.aarch64",
    sha256 = "53a48e97f56f0ffc1e2536494d5e4d18b4b904e6fda3e2969ac377549f5ec202",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.9.8-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/53a48e97f56f0ffc1e2536494d5e4d18b4b904e6fda3e2969ac377549f5ec202",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.8-1.el9.s390x",
    sha256 = "29baa933285a2b20e7571e346ab7dc1e2a0cbf74045646ad984e8569a9a53e31",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.9.8-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29baa933285a2b20e7571e346ab7dc1e2a0cbf74045646ad984e8569a9a53e31",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.8-1.el9.x86_64",
    sha256 = "d80957b223b2e6489d9da5148aada13a5e745765633fe07c3e3ce9e0ba7c7801",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.9.8-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d80957b223b2e6489d9da5148aada13a5e745765633fe07c3e3ce9e0ba7c7801",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-14.el9.aarch64",
    sha256 = "ec5482f096781a16d55762e96be3f6b21ee2f714bc8e45327ea978ae87951cc0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ec5482f096781a16d55762e96be3f6b21ee2f714bc8e45327ea978ae87951cc0",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-14.el9.s390x",
    sha256 = "e31be1174ae46e9e9cc6bce09d4cfd47eb280f96ef68488d4f0acefb2661a7df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libstdc++-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e31be1174ae46e9e9cc6bce09d4cfd47eb280f96ef68488d4f0acefb2661a7df",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-14.el9.x86_64",
    sha256 = "5b9119d93375d19b8ab140c359f9623de0fde1487fc1e930bfa29f54962ec448",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.5.0-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5b9119d93375d19b8ab140c359f9623de0fde1487fc1e930bfa29f54962ec448",
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
    name = "libtpms-0__0.9.6-11.el9.aarch64",
    sha256 = "299e63f64347c8738e7d86d0c3410362d98866d68617b0f6f9247295347d89f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libtpms-0.9.6-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/299e63f64347c8738e7d86d0c3410362d98866d68617b0f6f9247295347d89f2",
    ],
)

rpm(
    name = "libtpms-0__0.9.6-11.el9.s390x",
    sha256 = "0ec765f3f074c28652b8aabb010d4e64b4d2b1cb254c25e893be2af828c98687",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libtpms-0.9.6-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0ec765f3f074c28652b8aabb010d4e64b4d2b1cb254c25e893be2af828c98687",
    ],
)

rpm(
    name = "libtpms-0__0.9.6-11.el9.x86_64",
    sha256 = "e5863ebf4aad8570d7cd58a65b854de53df4ca366244bb65521c9fb96099f1da",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libtpms-0.9.6-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e5863ebf4aad8570d7cd58a65b854de53df4ca366244bb65521c9fb96099f1da",
    ],
)

rpm(
    name = "libubsan-0__11.5.0-14.el9.aarch64",
    sha256 = "a4f09558c0e26c82b5ae7c51d5d97d1b5ef9f0f50159c43a946a9958842af31e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.5.0-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a4f09558c0e26c82b5ae7c51d5d97d1b5ef9f0f50159c43a946a9958842af31e",
    ],
)

rpm(
    name = "libubsan-0__11.5.0-14.el9.s390x",
    sha256 = "248f8ad998b8bd25e583f6e6437d7f6a9ca9b9d0c8be2e212ce98fa9d3e85f12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libubsan-11.5.0-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/248f8ad998b8bd25e583f6e6437d7f6a9ca9b9d0c8be2e212ce98fa9d3e85f12",
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
    name = "liburing-0__2.12-1.el9.aarch64",
    sha256 = "7b99b8c28e8cf9a7d355231207e6151cc3b98cd722682359fff41737744d35d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/liburing-2.12-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7b99b8c28e8cf9a7d355231207e6151cc3b98cd722682359fff41737744d35d0",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el9.s390x",
    sha256 = "b259bcadc7623840495a33d9dabec62511a0f2133b731d070b59c5df60e8f7c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/liburing-2.12-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b259bcadc7623840495a33d9dabec62511a0f2133b731d070b59c5df60e8f7c6",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el9.x86_64",
    sha256 = "49b44a2192b8a3f3184d0ca80c318aa9852dddda391b66e7c38c53f900a08ce4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/liburing-2.12-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/49b44a2192b8a3f3184d0ca80c318aa9852dddda391b66e7c38c53f900a08ce4",
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
    name = "libvirt-client-0__11.10.0-4.el9.aarch64",
    sha256 = "34a8bbcaa0481943d04f3b6db333eb27662fe063d0afa0f7d825cb3e70e60f00",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-11.10.0-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-client-0__11.10.0-4.el9.s390x",
    sha256 = "0884d691b6fa0638eedaa5fe36a38b815d5e25e1b4aaf5a2257b1f7c2c30810a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-client-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-client-0__11.10.0-4.el9.x86_64",
    sha256 = "2a90cef3b9b70a1451cd968deab71fbf0ad6d8a4599105760327487f863cee12",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-7.el9.x86_64",
    sha256 = "ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af",
    urls = ["https://storage.googleapis.com/builddeps/ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af"],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-4.el9.aarch64",
    sha256 = "ff38205cd56f43b15129cc2a166f84584273237bdc76897b866b33ef076a334e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-11.10.0-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-4.el9.s390x",
    sha256 = "26fc50336037279e4fae9e71b4fbdd1bb8b2749d83d1fe3a33b4137262511340",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-4.el9.x86_64",
    sha256 = "b1f06f3575ce236bcfae407c2ccfe6df23989e75c2f6651a1af2c1fca2095742",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-7.el9.x86_64",
    sha256 = "13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1",
    urls = ["https://storage.googleapis.com/builddeps/13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-4.el9.aarch64",
    sha256 = "5fb18bfc6f2e2e34578d784a8bd69cb74d5ae54892c0fb6ff099fc692719f24a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-11.10.0-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-4.el9.s390x",
    sha256 = "cc0701102071e0f3393f198226cd593dd5987660b883fa34471b2249e9693943",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-4.el9.x86_64",
    sha256 = "a63f736b3efd00ff5cd7eeb7fafe1f99fca3a22fdef4f3ffc5849a413b7b6de3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.10.0-7.el9.x86_64",
    sha256 = "8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3",
    urls = ["https://storage.googleapis.com/builddeps/8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3"],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.10.0-4.el9.s390x",
    sha256 = "d311bbdfda7044bde671dda51143c3456d1b5b8a6ad199035f3577b021c2d31f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-secret-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.10.0-4.el9.x86_64",
    sha256 = "2c015af453a96afe868b6916835805185eb889ccb70823cd653d129420ece2d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.10.0-7.el9.x86_64",
    sha256 = "a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067",
    urls = ["https://storage.googleapis.com/builddeps/a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067"],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.10.0-4.el9.s390x",
    sha256 = "e609b1d5c9579a7ecab8f99310bc39552deef59d77d2216af88470d2e40df309",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-storage-core-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.10.0-4.el9.x86_64",
    sha256 = "ef7cbab8fcebcdef4b15ec190ba666c2a6723b58a448896393655a6334ed9c1f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-7.el9.x86_64",
    sha256 = "7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff",
    urls = ["https://storage.googleapis.com/builddeps/7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff"],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-4.el9.aarch64",
    sha256 = "0b78ef986c28c1c4775188db5493a12555dedb32eec9fe70851de784dd1999e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-11.10.0-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-4.el9.s390x",
    sha256 = "efcfef501c60475ac8437902100e4960333037946acdc449c41884a9c67ee015",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-4.el9.x86_64",
    sha256 = "7ccfeea4bfad99ba587b0a1df2ef6a573830647a5fcbcf0760a5d88c22a3e38c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-4.el9.aarch64",
    sha256 = "ec692e19eac06857116a191d42a3e25742dd9567d620ba25b5b32c7f6d479094",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-11.10.0-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-4.el9.s390x",
    sha256 = "116469cffef7ed6b67a5f49b7537654a207a30d7692fb42c89f4ec4826b97c37",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libvirt-devel-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-4.el9.x86_64",
    sha256 = "36259c57ed5c4fb31312caa8cc12a1ab35a709e914c670ecc8edeb867e27990c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-11.10.0-4.el9.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-7.el9.x86_64",
    sha256 = "72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39",
    urls = ["https://storage.googleapis.com/builddeps/72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39"],
)

rpm(
    name = "libvirt-libs-0__11.10.0-4.el9.aarch64",
    sha256 = "7b6bd2de1d0b44e90cc3b9de0ac00d12ef319ffc7990574f53467462f410277a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-11.10.0-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-libs-0__11.10.0-4.el9.s390x",
    sha256 = "bc599b1814514e2d21ad72902947a03e6d9ee0ef23be451a7748469e52fc2c72",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-libs-11.10.0-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-libs-0__11.10.0-4.el9.x86_64",
    sha256 = "af74243fac27151abf16017572202f7a1b6d82ca946abbd04ffe2153744bc70e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-11.10.0-4.el9.x86_64.rpm",
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
    name = "libxml2-0__2.9.13-14.el9.aarch64",
    sha256 = "f62d552977c2b1d53cc4f6d4e9ea91fa7c0351dcd3a5bec8ceb7f91bc1157aaf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.13-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f62d552977c2b1d53cc4f6d4e9ea91fa7c0351dcd3a5bec8ceb7f91bc1157aaf",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-14.el9.s390x",
    sha256 = "78256fb046360c848f4f4d2a0419705ce747e87de18bc0d3994c6d5865656992",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libxml2-2.9.13-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/78256fb046360c848f4f4d2a0419705ce747e87de18bc0d3994c6d5865656992",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-14.el9.x86_64",
    sha256 = "6e3e385bd23d1f0ecf30859d65eaaaa9583c814a9afb8e04379b1eeca21a54c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.13-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6e3e385bd23d1f0ecf30859d65eaaaa9583c814a9afb8e04379b1eeca21a54c3",
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
    name = "mpfr-0__4.1.0-7.el9.x86_64",
    sha256 = "179760104aa5a31ca463c586d0f21f380ba4d0eed212eee91bd1ca513e5d7a8d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/mpfr-4.1.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/179760104aa5a31ca463c586d0f21f380ba4d0eed212eee91bd1ca513e5d7a8d",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-8.el9.aarch64",
    sha256 = "d2e205e6a97983668a7228316a32c9ef0246ee267efa65d2399a05bab5315c86",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/mpfr-4.1.0-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d2e205e6a97983668a7228316a32c9ef0246ee267efa65d2399a05bab5315c86",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-8.el9.s390x",
    sha256 = "984e6d532c23be3cba9468d5e5786413a3d24167143db3cec75dfac8a7fe1f5d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/mpfr-4.1.0-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/984e6d532c23be3cba9468d5e5786413a3d24167143db3cec75dfac8a7fe1f5d",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-8.el9.x86_64",
    sha256 = "1944e0ee71e7e5eb0cd0772b78f78e04f5c5b1d5b9aecd3caac3d40c245e080c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/mpfr-4.1.0-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1944e0ee71e7e5eb0cd0772b78f78e04f5c5b1d5b9aecd3caac3d40c245e080c",
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
    name = "ncurses-base-0__6.2-12.20210508.el9.aarch64",
    sha256 = "49f6470fa7dd1b3ba81ccdd0547b29953af2835e067de915eeca3c45d5faa339",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.2-12.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/49f6470fa7dd1b3ba81ccdd0547b29953af2835e067de915eeca3c45d5faa339",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-12.20210508.el9.s390x",
    sha256 = "49f6470fa7dd1b3ba81ccdd0547b29953af2835e067de915eeca3c45d5faa339",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ncurses-base-6.2-12.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/49f6470fa7dd1b3ba81ccdd0547b29953af2835e067de915eeca3c45d5faa339",
    ],
)

rpm(
    name = "ncurses-base-0__6.2-12.20210508.el9.x86_64",
    sha256 = "49f6470fa7dd1b3ba81ccdd0547b29953af2835e067de915eeca3c45d5faa339",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.2-12.20210508.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/49f6470fa7dd1b3ba81ccdd0547b29953af2835e067de915eeca3c45d5faa339",
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
    name = "ncurses-libs-0__6.2-12.20210508.el9.aarch64",
    sha256 = "7b61d1dab8d4113a6ad015c083ac3053ec9db1f2503527d547ba7c741d54e57a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.2-12.20210508.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7b61d1dab8d4113a6ad015c083ac3053ec9db1f2503527d547ba7c741d54e57a",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-12.20210508.el9.s390x",
    sha256 = "d2a6307a398b9cde8f0a83fff92c3b31f5f6c4c15f911f64ff84168a7cd060a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/ncurses-libs-6.2-12.20210508.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d2a6307a398b9cde8f0a83fff92c3b31f5f6c4c15f911f64ff84168a7cd060a4",
    ],
)

rpm(
    name = "ncurses-libs-0__6.2-12.20210508.el9.x86_64",
    sha256 = "7b396883232158d4f9a6977bcd72b5e6f7fa6bc34a51030379833d4c0d24ab6f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.2-12.20210508.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7b396883232158d4f9a6977bcd72b5e6f7fa6bc34a51030379833d4c0d24ab6f",
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
    name = "ndctl-libs-0__82-1.el9.x86_64",
    sha256 = "a0e0c0618946ac1cf3ede937f9a43787d7cb40fe2626ccbf29cc8267b75d48f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/ndctl-libs-82-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0e0c0618946ac1cf3ede937f9a43787d7cb40fe2626ccbf29cc8267b75d48f5",
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
    name = "nftables-1__1.0.9-6.el9.aarch64",
    sha256 = "961be61924e53a0e1f6c7cb70c2687eb8dbe131cacbef3c4d335c0df46f402a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.9-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/961be61924e53a0e1f6c7cb70c2687eb8dbe131cacbef3c4d335c0df46f402a4",
    ],
)

rpm(
    name = "nftables-1__1.0.9-6.el9.s390x",
    sha256 = "e7672f3f0f6bce29e909386370325bf3de635bd34b318c5597aa92bd23331ce0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nftables-1.0.9-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e7672f3f0f6bce29e909386370325bf3de635bd34b318c5597aa92bd23331ce0",
    ],
)

rpm(
    name = "nftables-1__1.0.9-6.el9.x86_64",
    sha256 = "0c4a7846656c9da0e2f7c5be3173219fbe4ddda695d6609803487092dc505f9f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.9-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c4a7846656c9da0e2f7c5be3173219fbe4ddda695d6609803487092dc505f9f",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-4.el9.aarch64",
    sha256 = "c14582e4b22d11a93b981efa4734f5edbbb542e110c0261de8c4e7263b5b9286",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.92-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c14582e4b22d11a93b981efa4734f5edbbb542e110c0261de8c4e7263b5b9286",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-4.el9.s390x",
    sha256 = "13f00e7add0073a12f78cf2a1d7bd66f04e1dd58db1a0020ead30d4ec6026554",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/nmap-ncat-7.92-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/13f00e7add0073a12f78cf2a1d7bd66f04e1dd58db1a0020ead30d4ec6026554",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-4.el9.x86_64",
    sha256 = "7d3d8657b927479d811f80c3a6d76ca32c4595f684deb3a90ec0e098f8501f98",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.92-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d3d8657b927479d811f80c3a6d76ca32c4595f684deb3a90ec0e098f8501f98",
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
    name = "numactl-libs-0__2.0.19-1.el9.x86_64",
    sha256 = "3abe41a330364e2e1bf905458de8c0314f0ac3082e6cc475149bd9b2ffdeb428",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.19-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3abe41a330364e2e1bf905458de8c0314f0ac3082e6cc475149bd9b2ffdeb428",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el9.aarch64",
    sha256 = "ff63cef9b42cbc82149a6bc6970c20c9e781016dbb3eadd03effa330cb3b2bdd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.19-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ff63cef9b42cbc82149a6bc6970c20c9e781016dbb3eadd03effa330cb3b2bdd",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el9.s390x",
    sha256 = "43de4c5b609d2914a7fd27937b3baf15a988be1c891ee17e1395cae2232a06c4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/numactl-libs-2.0.19-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/43de4c5b609d2914a7fd27937b3baf15a988be1c891ee17e1395cae2232a06c4",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el9.x86_64",
    sha256 = "ad52833edf28b5bf2053bd96d96b211de4c6b11376978379dae211460c4596d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.19-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ad52833edf28b5bf2053bd96d96b211de4c6b11376978379dae211460c4596d8",
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
    name = "openssl-1__3.5.5-1.el9.aarch64",
    sha256 = "9614ff011d01a53615fc9a05ab29de503af3576ea8aaadfb6f7405a1c60b5755",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.5.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9614ff011d01a53615fc9a05ab29de503af3576ea8aaadfb6f7405a1c60b5755",
    ],
)

rpm(
    name = "openssl-1__3.5.5-1.el9.s390x",
    sha256 = "37de2ce39ecacbb0105a0aaaf99d186d05814f34df79663ec73ae5e33a523dfc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-3.5.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/37de2ce39ecacbb0105a0aaaf99d186d05814f34df79663ec73ae5e33a523dfc",
    ],
)

rpm(
    name = "openssl-1__3.5.5-1.el9.x86_64",
    sha256 = "a847741effa30135a514d585303dd53e5d45f7be672accdb602e48162344e396",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.5.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a847741effa30135a514d585303dd53e5d45f7be672accdb602e48162344e396",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-1.el9.aarch64",
    sha256 = "e59ea0883ea57fe2b6c5d2ea52cafa8183a95fc83e0be7bd72ab4e4517f41588",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-fips-provider-3.5.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e59ea0883ea57fe2b6c5d2ea52cafa8183a95fc83e0be7bd72ab4e4517f41588",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-1.el9.s390x",
    sha256 = "1845bf99e7825235a6a0d5ae2990bd5bda0a9e889aa6a735c1e8f459b504195e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-fips-provider-3.5.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1845bf99e7825235a6a0d5ae2990bd5bda0a9e889aa6a735c1e8f459b504195e",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-1.el9.x86_64",
    sha256 = "5aec51f1d460ff8da23044b8462e3569ea5e87e531d0ecaf67639b039fbbb896",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-fips-provider-3.5.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5aec51f1d460ff8da23044b8462e3569ea5e87e531d0ecaf67639b039fbbb896",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-6.el9.x86_64",
    sha256 = "4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738",
    urls = ["https://storage.googleapis.com/builddeps/4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738"],
)

rpm(
    name = "openssl-libs-1__3.5.5-1.el9.aarch64",
    sha256 = "2814ed29cf7d61164f9a0fdfa02ecf6f3d98b1341146effca29441ec2b0341bd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.5.5-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2814ed29cf7d61164f9a0fdfa02ecf6f3d98b1341146effca29441ec2b0341bd",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.5-1.el9.s390x",
    sha256 = "fdf91f4b3f4635964b6a04f76d18fdfdc7de47830a591171e95167047441df15",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-libs-3.5.5-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fdf91f4b3f4635964b6a04f76d18fdfdc7de47830a591171e95167047441df15",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.5-1.el9.x86_64",
    sha256 = "93be0a9dbce02f827ffe1f1dd538fd5f61b189f63d0d0b3b98b662d524321b30",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.5.5-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/93be0a9dbce02f827ffe1f1dd538fd5f61b189f63d0d0b3b98b662d524321b30",
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
    name = "p11-kit-0__0.25.3-3.el9.x86_64",
    sha256 = "2d02f32cdb62fac32563c70fad44c7252f0173552ccabc58d2b5161207c291a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-0.25.3-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d02f32cdb62fac32563c70fad44c7252f0173552ccabc58d2b5161207c291a3",
    ],
)

rpm(
    name = "p11-kit-0__0.26.2-1.el9.aarch64",
    sha256 = "078862b28f0e95c1464b8c8b85fd23a05351823acd3b60185af21a6ab5104271",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-0.26.2-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "p11-kit-0__0.26.2-1.el9.s390x",
    sha256 = "6743449ac49200da5f9ba3fcc8ef8f95880fbf8364ca67ccca5117dd9a126a0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-0.26.2-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "p11-kit-0__0.26.2-1.el9.x86_64",
    sha256 = "4e2f216f57ba90659679cb6cedcae7b38fb335a9d301c890ea7744b769ac15d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-0.26.2-1.el9.x86_64.rpm",
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
    name = "p11-kit-trust-0__0.26.2-1.el9.aarch64",
    sha256 = "3db76997186c82a6c7b2ecf514b8098bfecf8db5358ebafdbed02b51b67465f6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.26.2-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.2-1.el9.s390x",
    sha256 = "30854a67c6e2bcc1584210f0991704c64323d9a367ea1a98429e9a6a2d25b9b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-trust-0.26.2-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.2-1.el9.x86_64",
    sha256 = "d8dcb0fb0302e74bc2276e78d1bdcc2a512bcfaee86fe8b1d01e491bea6b250a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.26.2-1.el9.x86_64.rpm",
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
    name = "pam-0__1.5.1-28.el9.aarch64",
    sha256 = "598477ca76dadefb1c80d4322c2b074aac54d9ecf3d717353641b939147d8caa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-28.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/598477ca76dadefb1c80d4322c2b074aac54d9ecf3d717353641b939147d8caa",
    ],
)

rpm(
    name = "pam-0__1.5.1-28.el9.s390x",
    sha256 = "f823ac185f1a0c966e608301f09e22a645e37c447f4dd49b6b6930ff67fd8156",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pam-1.5.1-28.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f823ac185f1a0c966e608301f09e22a645e37c447f4dd49b6b6930ff67fd8156",
    ],
)

rpm(
    name = "pam-0__1.5.1-28.el9.x86_64",
    sha256 = "3c92fd1347d78fc3621cd5ae62f7a159588d86a8a0c22ba4a754dcd51926b6b7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-28.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3c92fd1347d78fc3621cd5ae62f7a159588d86a8a0c22ba4a754dcd51926b6b7",
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
    name = "passt-0__0__caret__20250512.g8ec1341-2.el9.aarch64",
    sha256 = "b25fd3d6395c66279b57f2c6cafa51d487b40cf7b3caa16915c7e702c5495679",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/passt-0%5E20250512.g8ec1341-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b25fd3d6395c66279b57f2c6cafa51d487b40cf7b3caa16915c7e702c5495679",
    ],
)

rpm(
    name = "passt-0__0__caret__20250512.g8ec1341-2.el9.s390x",
    sha256 = "ee4e4896389277f09db8e3a112e83e83be80d1401b43a9993137c12c004bdafc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/passt-0%5E20250512.g8ec1341-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ee4e4896389277f09db8e3a112e83e83be80d1401b43a9993137c12c004bdafc",
    ],
)

rpm(
    name = "passt-0__0__caret__20250512.g8ec1341-2.el9.x86_64",
    sha256 = "27818a1904c1d1e39890abe239ea156a29c3e548e83941d7696e0cd3113abfdd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/passt-0%5E20250512.g8ec1341-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/27818a1904c1d1e39890abe239ea156a29c3e548e83941d7696e0cd3113abfdd",
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
    name = "policycoreutils-0__3.6-5.el9.aarch64",
    sha256 = "98f6b034b67a76f18a7e44a8b1b22fd5fd293d326c833418cbb2313abffc57d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.6-5.el9.aarch64.rpm",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-5.el9.s390x",
    sha256 = "659c4c2fe612c6a3e9af23a90993d2772e771f5f83f3cf195e8117a06962c6a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/policycoreutils-3.6-5.el9.s390x.rpm",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-5.el9.x86_64",
    sha256 = "436dd0b9df40d54a5ff97051738e67bf080d2059d9e767ceac477f9155ab4ca9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.6-5.el9.x86_64.rpm",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.6-5.el9.aarch64",
    sha256 = "81ae128e56421df1941477b698d275002e8d1f95ae1ad04033e516ecaf2488a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/policycoreutils-python-utils-3.6-5.el9.noarch.rpm",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.6-5.el9.s390x",
    sha256 = "81ae128e56421df1941477b698d275002e8d1f95ae1ad04033e516ecaf2488a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/policycoreutils-python-utils-3.6-5.el9.noarch.rpm",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.6-5.el9.x86_64",
    sha256 = "81ae128e56421df1941477b698d275002e8d1f95ae1ad04033e516ecaf2488a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/policycoreutils-python-utils-3.6-5.el9.noarch.rpm",
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
    name = "polkit-0__0.117-14.el9.aarch64",
    sha256 = "9b756ae0148672fd428182ecbfae0e6d4fe249ad41cddb5f99ef2868b1b07c27",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-0.117-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9b756ae0148672fd428182ecbfae0e6d4fe249ad41cddb5f99ef2868b1b07c27",
    ],
)

rpm(
    name = "polkit-0__0.117-14.el9.s390x",
    sha256 = "8d97919e210f66f10b8f4faad916eb21b9e33c3cdb088638bc6a1dd5efa75b9d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/polkit-0.117-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8d97919e210f66f10b8f4faad916eb21b9e33c3cdb088638bc6a1dd5efa75b9d",
    ],
)

rpm(
    name = "polkit-0__0.117-14.el9.x86_64",
    sha256 = "30f3a75427f33cd136ffbacbb01d4570a62a15b6da4ecb005a0e1da25f0ca57a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-0.117-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/30f3a75427f33cd136ffbacbb01d4570a62a15b6da4ecb005a0e1da25f0ca57a",
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
    name = "polkit-libs-0__0.117-14.el9.aarch64",
    sha256 = "a2021169f907a5cf2ac57193ff1d32d9df514db03a12aa5842ae7d358d66c20f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.117-14.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a2021169f907a5cf2ac57193ff1d32d9df514db03a12aa5842ae7d358d66c20f",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-14.el9.s390x",
    sha256 = "475daa3f2f4890182b4aac23ca2fa0c071c536fbe5c7a1f17b5bf555d71eda26",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/polkit-libs-0.117-14.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/475daa3f2f4890182b4aac23ca2fa0c071c536fbe5c7a1f17b5bf555d71eda26",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-14.el9.x86_64",
    sha256 = "b2acb122a1cbadef39c62cdbd34781aa094a07ee9fafc61f987dd43ea09f182f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.117-14.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2acb122a1cbadef39c62cdbd34781aa094a07ee9fafc61f987dd43ea09f182f",
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
    name = "python3-0__3.9.25-3.el9.aarch64",
    sha256 = "6bcc49ccb04386015b21e0f9a97e4e74fddc3b2aaeb11a4de890662bea8116d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.25-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6bcc49ccb04386015b21e0f9a97e4e74fddc3b2aaeb11a4de890662bea8116d0",
    ],
)

rpm(
    name = "python3-0__3.9.25-3.el9.s390x",
    sha256 = "8d256069560f02e70dab77d81fb1f8427adcdabc8f55ac1d64e9c8975b0eab6e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-3.9.25-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8d256069560f02e70dab77d81fb1f8427adcdabc8f55ac1d64e9c8975b0eab6e",
    ],
)

rpm(
    name = "python3-0__3.9.25-3.el9.x86_64",
    sha256 = "80665300816d833df3f3ed808022c53dda3c3687901bfaf802780bc0b7899842",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.25-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/80665300816d833df3f3ed808022c53dda3c3687901bfaf802780bc0b7899842",
    ],
)

rpm(
    name = "python3-audit-0__3.1.5-8.el9.aarch64",
    sha256 = "78b0b2315438c22ea52bd5ce7bac251a9e8145a391e62e77f2b7d1765e224c9a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-audit-3.1.5-8.el9.aarch64.rpm",
    ],
)

rpm(
    name = "python3-audit-0__3.1.5-8.el9.s390x",
    sha256 = "c3060d2aaf74b636a10dc8205bdfce064c5c8fbd841833217dfda83846ecef4b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-audit-3.1.5-8.el9.s390x.rpm",
    ],
)

rpm(
    name = "python3-audit-0__3.1.5-8.el9.x86_64",
    sha256 = "4145f9a7d78dc8469ba1cfbcadeb3bd9284fd07bfa9d80c023c69d96281a065b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-audit-3.1.5-8.el9.x86_64.rpm",
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
    name = "python3-distro-0__1.5.0-7.el9.aarch64",
    sha256 = "370ab59bdcfc5657002bb12c6344a90338e4ccc735de9967575f06c5cf3c65a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-distro-1.5.0-7.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-distro-0__1.5.0-7.el9.s390x",
    sha256 = "370ab59bdcfc5657002bb12c6344a90338e4ccc735de9967575f06c5cf3c65a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-distro-1.5.0-7.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-distro-0__1.5.0-7.el9.x86_64",
    sha256 = "370ab59bdcfc5657002bb12c6344a90338e4ccc735de9967575f06c5cf3c65a7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-distro-1.5.0-7.el9.noarch.rpm",
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
    name = "python3-libs-0__3.9.25-3.el9.aarch64",
    sha256 = "a7382c2bf7a19df0250ca2f28a7e25a294a19b42e54a5a63c5c499b8c6c6c685",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.25-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a7382c2bf7a19df0250ca2f28a7e25a294a19b42e54a5a63c5c499b8c6c6c685",
    ],
)

rpm(
    name = "python3-libs-0__3.9.25-3.el9.s390x",
    sha256 = "48ddc59124e89c00be67d752f6d2c4f6e65c8681a454bc69b39392f569fe8518",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-libs-3.9.25-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/48ddc59124e89c00be67d752f6d2c4f6e65c8681a454bc69b39392f569fe8518",
    ],
)

rpm(
    name = "python3-libs-0__3.9.25-3.el9.x86_64",
    sha256 = "8929e0c6a72abb2f4890897b80ad9ad28c2cab6c7aeae8b145400c0f21443ace",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.25-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8929e0c6a72abb2f4890897b80ad9ad28c2cab6c7aeae8b145400c0f21443ace",
    ],
)

rpm(
    name = "python3-libselinux-0__3.6-3.el9.aarch64",
    sha256 = "857e354d3ff4d6c9a3a3966aeeacd4fd6d48563a3d6e9b6ba935ebe1e60f489e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-libselinux-3.6-3.el9.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libselinux-0__3.6-3.el9.s390x",
    sha256 = "80e1befb404caf22cbd770d917970022999a1b6cd2269ab67ee7a86029adbb1b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-libselinux-3.6-3.el9.s390x.rpm",
    ],
)

rpm(
    name = "python3-libselinux-0__3.6-3.el9.x86_64",
    sha256 = "664b918f9e59b4ce2fba4a1c2ae4775b03af4f704e6c49de8490d76fbc8bcc70",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-libselinux-3.6-3.el9.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.6-5.el9.aarch64",
    sha256 = "db6bcca0fd0892995ab15308c991dda2bff1cb50e8f04f3bdb4eeabeb250be19",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-libsemanage-3.6-5.el9.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.6-5.el9.s390x",
    sha256 = "c0b702e9cbbc9059b5218a9f0dbcdb0c25298bce5022890a3c442bb9367b37e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-libsemanage-3.6-5.el9.s390x.rpm",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.6-5.el9.x86_64",
    sha256 = "af719e0ec7dc0ff84c412a4894fb6f6bac875416239590a173f30b2ff07f65d4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-libsemanage-3.6-5.el9.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libvirt-0__11.10.0-1.el9.aarch64",
    sha256 = "1ab8074474f286050cca15f15d4de376b8e529f8259243d0e21cd5033fe0eea9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-libvirt-11.10.0-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libvirt-0__11.10.0-1.el9.s390x",
    sha256 = "5f89c1935a98e00a15f554b548625a564e0430d7671aad414c2314a57c0389f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-libvirt-11.10.0-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "python3-libvirt-0__11.10.0-1.el9.x86_64",
    sha256 = "ad7eea851a012f871e109ee4ff7c571f26a403f3976384907bf34ce7a078455f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-libvirt-11.10.0-1.el9.x86_64.rpm",
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
    name = "python3-policycoreutils-0__3.6-5.el9.aarch64",
    sha256 = "ca57c6f4279e5a2111c2fa6f44d655a342ffb808efd16d6af319102e42730f79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/python3-policycoreutils-3.6-5.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-policycoreutils-0__3.6-5.el9.s390x",
    sha256 = "ca57c6f4279e5a2111c2fa6f44d655a342ffb808efd16d6af319102e42730f79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/python3-policycoreutils-3.6-5.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-policycoreutils-0__3.6-5.el9.x86_64",
    sha256 = "ca57c6f4279e5a2111c2fa6f44d655a342ffb808efd16d6af319102e42730f79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/python3-policycoreutils-3.6-5.el9.noarch.rpm",
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
    name = "python3-setools-0__4.4.4-1.el9.aarch64",
    sha256 = "4b379e66446331c2f2d70a25efd59629389d71f1e7457d79194277b1040c1d67",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setools-4.4.4-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "python3-setools-0__4.4.4-1.el9.s390x",
    sha256 = "e312deb18d01a902c4bd94d6c035a710fb240bcb65da83a8982c593524da474e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-setools-4.4.4-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "python3-setools-0__4.4.4-1.el9.x86_64",
    sha256 = "4df7313e3cc57f8dd1ca29a0a7ece9b3c85a4643688b171d5848be1ba501508b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setools-4.4.4-1.el9.x86_64.rpm",
    ],
)

rpm(
    name = "python3-setuptools-0__53.0.0-15.el9.aarch64",
    sha256 = "83f013e1faa85969156e574373fb70c1b97ab70b1db27def32deacbd970da578",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setuptools-53.0.0-15.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-setuptools-0__53.0.0-15.el9.s390x",
    sha256 = "83f013e1faa85969156e574373fb70c1b97ab70b1db27def32deacbd970da578",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-setuptools-53.0.0-15.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-setuptools-0__53.0.0-15.el9.x86_64",
    sha256 = "83f013e1faa85969156e574373fb70c1b97ab70b1db27def32deacbd970da578",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setuptools-53.0.0-15.el9.noarch.rpm",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-15.el9.aarch64",
    sha256 = "4d61c666c3862bd18caebac2295c088627b47612f3367cd636fcaec9a021bbac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-53.0.0-15.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4d61c666c3862bd18caebac2295c088627b47612f3367cd636fcaec9a021bbac",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-15.el9.s390x",
    sha256 = "4d61c666c3862bd18caebac2295c088627b47612f3367cd636fcaec9a021bbac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-setuptools-wheel-53.0.0-15.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4d61c666c3862bd18caebac2295c088627b47612f3367cd636fcaec9a021bbac",
    ],
)

rpm(
    name = "python3-setuptools-wheel-0__53.0.0-15.el9.x86_64",
    sha256 = "4d61c666c3862bd18caebac2295c088627b47612f3367cd636fcaec9a021bbac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-53.0.0-15.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4d61c666c3862bd18caebac2295c088627b47612f3367cd636fcaec9a021bbac",
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
    name = "qemu-img-17__10.1.0-10.el9.aarch64",
    sha256 = "b3c671a12649fcba7d3874ac0e0c0afc270a3faeef9c1a790c97befd38a30872",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b3c671a12649fcba7d3874ac0e0c0afc270a3faeef9c1a790c97befd38a30872",
    ],
)

rpm(
    name = "qemu-img-17__10.1.0-10.el9.s390x",
    sha256 = "f70342c7ed802c44ae8bde3519d4b9f502f843f248846955960a39a2a1ece164",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-img-10.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f70342c7ed802c44ae8bde3519d4b9f502f843f248846955960a39a2a1ece164",
    ],
)

rpm(
    name = "qemu-img-17__10.1.0-10.el9.x86_64",
    sha256 = "020472c00f3160f8bcc5b13cfdacc46d5e65676b40e5758a6b7f64de168fac79",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/020472c00f3160f8bcc5b13cfdacc46d5e65676b40e5758a6b7f64de168fac79",
    ],
)

rpm(
    name = "qemu-img-17__9.1.0-15.el9.x86_64",
    sha256 = "6149224d6968142db7c12330dd4d9f6882af2ad73a97e591214a3869603b663f",
    urls = ["https://storage.googleapis.com/builddeps/6149224d6968142db7c12330dd4d9f6882af2ad73a97e591214a3869603b663f"],
)

rpm(
    name = "qemu-kvm-common-17__10.1.0-10.el9.aarch64",
    sha256 = "7850e9a4e039659e3fd94e1b45a46eab5b069c78832a6f1520eb6e60c0681b47",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7850e9a4e039659e3fd94e1b45a46eab5b069c78832a6f1520eb6e60c0681b47",
    ],
)

rpm(
    name = "qemu-kvm-common-17__10.1.0-10.el9.s390x",
    sha256 = "0fe7718edf8c464fdc6c346bbd65bd2468f3266c34960426f1fe02303183d2c3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-common-10.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0fe7718edf8c464fdc6c346bbd65bd2468f3266c34960426f1fe02303183d2c3",
    ],
)

rpm(
    name = "qemu-kvm-common-17__10.1.0-10.el9.x86_64",
    sha256 = "8ef35ad940fedae158346385fe825c0309f8ad6495df73a552b13c3383b9d4b7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ef35ad940fedae158346385fe825c0309f8ad6495df73a552b13c3383b9d4b7",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.1.0-15.el9.x86_64",
    sha256 = "345b3dae626a756f160321e025774d3d3e193a767388e69ffc832ea75988b166",
    urls = ["https://storage.googleapis.com/builddeps/345b3dae626a756f160321e025774d3d3e193a767388e69ffc832ea75988b166"],
)

rpm(
    name = "qemu-kvm-core-17__10.1.0-10.el9.aarch64",
    sha256 = "2b27d861d1f8019b3db06ba1ddfc6e4278ec5df2e6eca20251367465e39258f7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2b27d861d1f8019b3db06ba1ddfc6e4278ec5df2e6eca20251367465e39258f7",
    ],
)

rpm(
    name = "qemu-kvm-core-17__10.1.0-10.el9.s390x",
    sha256 = "34181ed8209aed15c3039287631e63958cecfd79e62f60f67aa9b7c4899b8ba1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-core-10.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/34181ed8209aed15c3039287631e63958cecfd79e62f60f67aa9b7c4899b8ba1",
    ],
)

rpm(
    name = "qemu-kvm-core-17__10.1.0-10.el9.x86_64",
    sha256 = "e24d2e3154b71618031cdd3314e7de9d6d68b032531a2098bf3d6cbcc0d9f4ef",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e24d2e3154b71618031cdd3314e7de9d6d68b032531a2098bf3d6cbcc0d9f4ef",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.1.0-15.el9.x86_64",
    sha256 = "aa36521b947a78d2d06d90e1a8f5d74bab5ffbbb6d8ca8d939497477c4878565",
    urls = ["https://storage.googleapis.com/builddeps/aa36521b947a78d2d06d90e1a8f5d74bab5ffbbb6d8ca8d939497477c4878565"],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__10.1.0-10.el9.aarch64",
    sha256 = "57f01d67fb90e28f8a0651b495f25a3f10e87ac96eef2227d3c3af01791e6e71",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/57f01d67fb90e28f8a0651b495f25a3f10e87ac96eef2227d3c3af01791e6e71",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__10.1.0-10.el9.s390x",
    sha256 = "c93b176d87f21c58ee277f3abaa04eebba2af8465356bc34a94b23207d0f4bd3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c93b176d87f21c58ee277f3abaa04eebba2af8465356bc34a94b23207d0f4bd3",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__10.1.0-10.el9.x86_64",
    sha256 = "a851e72c1f1684006e1bd104297beb94ac486dd931930ad59a754bdbaf1955ac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a851e72c1f1684006e1bd104297beb94ac486dd931930ad59a754bdbaf1955ac",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-ccw-17__10.1.0-10.el9.s390x",
    sha256 = "f1915ceafea5f2ad0ce567d25bb9d904c30139ab2924bb6ee43759959e68278f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-10.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f1915ceafea5f2ad0ce567d25bb9d904c30139ab2924bb6ee43759959e68278f",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__10.1.0-10.el9.aarch64",
    sha256 = "f3adcc8def322ea354c6a1a03cb984fdb983bfcb570d9a5c2d22c356edb068c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f3adcc8def322ea354c6a1a03cb984fdb983bfcb570d9a5c2d22c356edb068c2",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__10.1.0-10.el9.x86_64",
    sha256 = "abda4ef1719f3dd33951b9a6b95056ba370d34c72440c84f1a04c652a77f066d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/abda4ef1719f3dd33951b9a6b95056ba370d34c72440c84f1a04c652a77f066d",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-vga-17__10.1.0-10.el9.x86_64",
    sha256 = "b11c166261824350ea29f31dbc4327e6bf445fe340e2f91cd4f2c26b44332b6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-vga-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b11c166261824350ea29f31dbc4327e6bf445fe340e2f91cd4f2c26b44332b6b",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__10.1.0-10.el9.aarch64",
    sha256 = "82663b34760d96ffa404d63d386d8d886fe8ad25cf8e4856fe8ad3dec131742b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/82663b34760d96ffa404d63d386d8d886fe8ad25cf8e4856fe8ad3dec131742b",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__10.1.0-10.el9.s390x",
    sha256 = "0eafa1d32cb0eb8fe9397093b4afc7bcbc0c3a2dc64f0accba099697c5ab30ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-10.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0eafa1d32cb0eb8fe9397093b4afc7bcbc0c3a2dc64f0accba099697c5ab30ed",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__10.1.0-10.el9.x86_64",
    sha256 = "2575227bcaed72dedc44c71b8fb893a58fe7322abfc3ca8f707073cedc6858ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2575227bcaed72dedc44c71b8fb893a58fe7322abfc3ca8f707073cedc6858ce",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__10.1.0-10.el9.aarch64",
    sha256 = "98897947fe9bce3726f43d6f1c594253a31e908a31e50d3d763d7c94db4de8a1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98897947fe9bce3726f43d6f1c594253a31e908a31e50d3d763d7c94db4de8a1",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__10.1.0-10.el9.x86_64",
    sha256 = "54740022c454f0eb9fadd8f7eaa9f94e24115523fb473b034d46016806bc5c1f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54740022c454f0eb9fadd8f7eaa9f94e24115523fb473b034d46016806bc5c1f",
    ],
)

rpm(
    name = "qemu-pr-helper-17__10.1.0-13.el9.aarch64",
    sha256 = "09256a8b3bf6c10be7ef9de5a418db108bc55c909de170f9afd0e7f910587f68",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-10.1.0-13.el9.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-pr-helper-17__10.1.0-13.el9.x86_64",
    sha256 = "eede0e06a4fe83d039c611b3f57ee5df2beca0d0f858a1ee0f07ecba85165746",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-10.1.0-13.el9.x86_64.rpm",
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
    name = "rpm-0__4.16.1.3-40.el9.aarch64",
    sha256 = "75e4d04e5712c50d717088642fd23adb9ea62c9662a159c740375510c8f30a47",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-4.16.1.3-40.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/75e4d04e5712c50d717088642fd23adb9ea62c9662a159c740375510c8f30a47",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-40.el9.s390x",
    sha256 = "9363a8f161f6965bbfc4bf591bbbfd9dfbdbafbd72a480f58dcacfddd0344048",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-4.16.1.3-40.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9363a8f161f6965bbfc4bf591bbbfd9dfbdbafbd72a480f58dcacfddd0344048",
    ],
)

rpm(
    name = "rpm-0__4.16.1.3-40.el9.x86_64",
    sha256 = "46e39d6ce74f21c388ac74db702d74813253e0b79096905bc9683be4ff0323fe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-4.16.1.3-40.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46e39d6ce74f21c388ac74db702d74813253e0b79096905bc9683be4ff0323fe",
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
    name = "rpm-libs-0__4.16.1.3-40.el9.aarch64",
    sha256 = "aeddcc0609d5a2faa1206148be2514becc210a5422e7eb6be7a9a159c1e910ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.16.1.3-40.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aeddcc0609d5a2faa1206148be2514becc210a5422e7eb6be7a9a159c1e910ff",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-40.el9.s390x",
    sha256 = "699c13851921821ce338801d19d59cf0ba91584bd7c483a3cbae7466ef6779db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-libs-4.16.1.3-40.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/699c13851921821ce338801d19d59cf0ba91584bd7c483a3cbae7466ef6779db",
    ],
)

rpm(
    name = "rpm-libs-0__4.16.1.3-40.el9.x86_64",
    sha256 = "5a141739706737beb8c89678cc9a9c7d5adc35f9d893b3ecf919b42c83775dae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.16.1.3-40.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a141739706737beb8c89678cc9a9c7d5adc35f9d893b3ecf919b42c83775dae",
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
    name = "rpm-plugin-selinux-0__4.16.1.3-40.el9.aarch64",
    sha256 = "19ce6961dfec6e9f818d633e3730c0990b14411757eeb17139d2a1ee71e1c785",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.16.1.3-40.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19ce6961dfec6e9f818d633e3730c0990b14411757eeb17139d2a1ee71e1c785",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-40.el9.s390x",
    sha256 = "42758d8484ace9e5df8edab294d1e31cf30c7f0c641ac5b9150baf1f6efc25e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/rpm-plugin-selinux-4.16.1.3-40.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/42758d8484ace9e5df8edab294d1e31cf30c7f0c641ac5b9150baf1f6efc25e9",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.16.1.3-40.el9.x86_64",
    sha256 = "9b9ed99dde1c72fbb7860370d5b80c8fe83c54519887fe62dbf74acee9b23eb7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.16.1.3-40.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9b9ed99dde1c72fbb7860370d5b80c8fe83c54519887fe62dbf74acee9b23eb7",
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
    name = "selinux-policy-0__38.1.73-1.el9.aarch64",
    sha256 = "8797960a78b46b3c3c173fb54c14f8a0c5c8c3cbfd950a5a627309544bdd90fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-38.1.73-1.el9.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.73-1.el9.s390x",
    sha256 = "8797960a78b46b3c3c173fb54c14f8a0c5c8c3cbfd950a5a627309544bdd90fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-38.1.73-1.el9.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.73-1.el9.x86_64",
    sha256 = "8797960a78b46b3c3c173fb54c14f8a0c5c8c3cbfd950a5a627309544bdd90fb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-38.1.73-1.el9.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.53-2.el9.x86_64",
    sha256 = "b9f921bdc764af3b8c5c8580fc9db4f75b0fb3b2c0a3ea1f541536de091664b1",
    urls = ["https://storage.googleapis.com/builddeps/b9f921bdc764af3b8c5c8580fc9db4f75b0fb3b2c0a3ea1f541536de091664b1"],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.73-1.el9.aarch64",
    sha256 = "eaf9db99b4759a86ea95d6fc8e6e163392707c70b7cc05b0aea9eec5765436ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-38.1.73-1.el9.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.73-1.el9.s390x",
    sha256 = "eaf9db99b4759a86ea95d6fc8e6e163392707c70b7cc05b0aea9eec5765436ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-38.1.73-1.el9.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.73-1.el9.x86_64",
    sha256 = "eaf9db99b4759a86ea95d6fc8e6e163392707c70b7cc05b0aea9eec5765436ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-38.1.73-1.el9.noarch.rpm",
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
    name = "shadow-utils-2__4.9-16.el9.aarch64",
    sha256 = "085a4d0d20ee46e72564939e92533fbf4c049658c58d4e7cc075d5da5baa7098",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.9-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/085a4d0d20ee46e72564939e92533fbf4c049658c58d4e7cc075d5da5baa7098",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-16.el9.s390x",
    sha256 = "18c43c994a4c8f6c97c195f2bf30ffad338b3cf5ee08e3d813731dbcedf51b4e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/shadow-utils-4.9-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/18c43c994a4c8f6c97c195f2bf30ffad338b3cf5ee08e3d813731dbcedf51b4e",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-16.el9.x86_64",
    sha256 = "f82dcf66ba99287eaebe3225cb01d252eea40202b0b263a2b2619f87d98918fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f82dcf66ba99287eaebe3225cb01d252eea40202b0b263a2b2619f87d98918fd",
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
    name = "sqlite-libs-0__3.34.1-9.el9.aarch64",
    sha256 = "115fbe01cb007257cbd277bb83e63eb211dd244f10e150e5e71b3edca68181c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.34.1-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/115fbe01cb007257cbd277bb83e63eb211dd244f10e150e5e71b3edca68181c6",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-9.el9.s390x",
    sha256 = "759932a92153b14492308c8330aa398fcabec87f2b42635a231b455431032f3b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sqlite-libs-3.34.1-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/759932a92153b14492308c8330aa398fcabec87f2b42635a231b455431032f3b",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-9.el9.x86_64",
    sha256 = "88abee2ce36a11d707d610e796afcd2919a7adc3bb5ba4e92b08ef06046e6970",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/88abee2ce36a11d707d610e796afcd2919a7adc3bb5ba4e92b08ef06046e6970",
    ],
)

rpm(
    name = "sssd-client-0__2.9.8-1.el9.aarch64",
    sha256 = "09956d57874696e6e40d78536bb25326d1d798e206e468048d4a122aca8d193d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.9.8-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/09956d57874696e6e40d78536bb25326d1d798e206e468048d4a122aca8d193d",
    ],
)

rpm(
    name = "sssd-client-0__2.9.8-1.el9.s390x",
    sha256 = "6051a62aa35b330b7dc76f425317d06e10646cd6446682940258302697b1ce42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sssd-client-2.9.8-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6051a62aa35b330b7dc76f425317d06e10646cd6446682940258302697b1ce42",
    ],
)

rpm(
    name = "sssd-client-0__2.9.8-1.el9.x86_64",
    sha256 = "d14fb246e49b57d5170d6eb6f916458b119d34a810d66c9b522da12fbcb35406",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.9.8-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d14fb246e49b57d5170d6eb6f916458b119d34a810d66c9b522da12fbcb35406",
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
    urls = ["https://storage.googleapis.com/builddeps/c5e5ae6f65f085c9f811a2a7950920eecb0c7ddf3d82c3f63b5662231cfc5de0"],
)

rpm(
    name = "systemd-0__252-64.el9.aarch64",
    sha256 = "5468eb07e3c5aa6f81e53a67a2cfaa76bb9eaa91a2054149c3eb47d27751c843",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-252-64.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5468eb07e3c5aa6f81e53a67a2cfaa76bb9eaa91a2054149c3eb47d27751c843",
    ],
)

rpm(
    name = "systemd-0__252-64.el9.s390x",
    sha256 = "5d61fb611b20b166e17d6f239535957939b5a021d9508f22d92cda060dd82fed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-252-64.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5d61fb611b20b166e17d6f239535957939b5a021d9508f22d92cda060dd82fed",
    ],
)

rpm(
    name = "systemd-0__252-64.el9.x86_64",
    sha256 = "845a3bc3e6bc8ca9e9d0f464b720aaf877860e013037fb410d872bc1d6d537d7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-64.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/845a3bc3e6bc8ca9e9d0f464b720aaf877860e013037fb410d872bc1d6d537d7",
    ],
)

rpm(
    name = "systemd-container-0__252-51.el9.x86_64",
    sha256 = "653fcd14047fb557e3a3f5da47c83d6ceb2194169f3ef42a27566bb4e2102dde",
    urls = ["https://storage.googleapis.com/builddeps/653fcd14047fb557e3a3f5da47c83d6ceb2194169f3ef42a27566bb4e2102dde"],
)

rpm(
    name = "systemd-container-0__252-64.el9.aarch64",
    sha256 = "a85db64c78dac1e04435f0769c89a34084cf107e66e107267a94bd3c7532c294",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-252-64.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a85db64c78dac1e04435f0769c89a34084cf107e66e107267a94bd3c7532c294",
    ],
)

rpm(
    name = "systemd-container-0__252-64.el9.s390x",
    sha256 = "889bfeac74479f096d25e72d6d06b6715abc4b96f3969634f3256164673ed42b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-container-252-64.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/889bfeac74479f096d25e72d6d06b6715abc4b96f3969634f3256164673ed42b",
    ],
)

rpm(
    name = "systemd-container-0__252-64.el9.x86_64",
    sha256 = "730f7fbf6729e3cf4d9b0b56ca54b4ac960b0b297481cbccb4ae9287319c9615",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-64.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/730f7fbf6729e3cf4d9b0b56ca54b4ac960b0b297481cbccb4ae9287319c9615",
    ],
)

rpm(
    name = "systemd-libs-0__252-51.el9.x86_64",
    sha256 = "a9d02a16bbc778ad3a2b46b8740fa821df065cdacd6ba8570c3301dacad79f0f",
    urls = ["https://storage.googleapis.com/builddeps/a9d02a16bbc778ad3a2b46b8740fa821df065cdacd6ba8570c3301dacad79f0f"],
)

rpm(
    name = "systemd-libs-0__252-64.el9.aarch64",
    sha256 = "e48bfbd29f6a1412de603e0cb5d9ac6d46515360841c1b8421c8e2b4871e3d04",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-252-64.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e48bfbd29f6a1412de603e0cb5d9ac6d46515360841c1b8421c8e2b4871e3d04",
    ],
)

rpm(
    name = "systemd-libs-0__252-64.el9.s390x",
    sha256 = "8f232bee21b9fda85b9b9c69a14d348306620745f9b70261efc14a85e73f10f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-libs-252-64.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8f232bee21b9fda85b9b9c69a14d348306620745f9b70261efc14a85e73f10f5",
    ],
)

rpm(
    name = "systemd-libs-0__252-64.el9.x86_64",
    sha256 = "df842cca567614bf20891234df566b3de3f008450a25a6e4b6031ac183e7d17d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-64.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df842cca567614bf20891234df566b3de3f008450a25a6e4b6031ac183e7d17d",
    ],
)

rpm(
    name = "systemd-pam-0__252-51.el9.x86_64",
    sha256 = "26014995c59a6d43c7cc0ba55b829cc14513491bc901fe60faf5a10b43c8fb03",
    urls = ["https://storage.googleapis.com/builddeps/26014995c59a6d43c7cc0ba55b829cc14513491bc901fe60faf5a10b43c8fb03"],
)

rpm(
    name = "systemd-pam-0__252-64.el9.aarch64",
    sha256 = "6f35935fafcc60a6eba0859f165ec411ee4b0b8ea6a8dd63100a2199165ca665",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-252-64.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6f35935fafcc60a6eba0859f165ec411ee4b0b8ea6a8dd63100a2199165ca665",
    ],
)

rpm(
    name = "systemd-pam-0__252-64.el9.s390x",
    sha256 = "d2d89235bf69a9c840ad71d36895a8454721356c22b9fefb405198075c0628e9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-pam-252-64.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d2d89235bf69a9c840ad71d36895a8454721356c22b9fefb405198075c0628e9",
    ],
)

rpm(
    name = "systemd-pam-0__252-64.el9.x86_64",
    sha256 = "1cf1fb13d6b5016b1d6c94de9e2149c7ea35fbaac9122dd1ac70f8ff24f9fd9b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-64.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cf1fb13d6b5016b1d6c94de9e2149c7ea35fbaac9122dd1ac70f8ff24f9fd9b",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-51.el9.x86_64",
    sha256 = "afa84ccbac79bb3950cca69bbfa9868429ed3aa464c96f5b2a15405a9c49f56c",
    urls = ["https://storage.googleapis.com/builddeps/afa84ccbac79bb3950cca69bbfa9868429ed3aa464c96f5b2a15405a9c49f56c"],
)

rpm(
    name = "systemd-rpm-macros-0__252-64.el9.aarch64",
    sha256 = "3a6d7b3d25e5faf5c1514ff4bfdadac927a8c33c159d08707e5e631ee330ee0e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-252-64.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3a6d7b3d25e5faf5c1514ff4bfdadac927a8c33c159d08707e5e631ee330ee0e",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-64.el9.s390x",
    sha256 = "3a6d7b3d25e5faf5c1514ff4bfdadac927a8c33c159d08707e5e631ee330ee0e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-rpm-macros-252-64.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3a6d7b3d25e5faf5c1514ff4bfdadac927a8c33c159d08707e5e631ee330ee0e",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-64.el9.x86_64",
    sha256 = "3a6d7b3d25e5faf5c1514ff4bfdadac927a8c33c159d08707e5e631ee330ee0e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-64.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3a6d7b3d25e5faf5c1514ff4bfdadac927a8c33c159d08707e5e631ee330ee0e",
    ],
)

rpm(
    name = "tar-2__1.34-10.el9.aarch64",
    sha256 = "84831858ad6cef9cbbb5aa63b383e399e71b7309ddd3e0bbba42b80738f9822e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tar-1.34-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/84831858ad6cef9cbbb5aa63b383e399e71b7309ddd3e0bbba42b80738f9822e",
    ],
)

rpm(
    name = "tar-2__1.34-10.el9.s390x",
    sha256 = "9f6c5294ffcbaeb67c6c2d0e4436118fadc7437c5396aa802cadd822d6284759",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tar-1.34-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9f6c5294ffcbaeb67c6c2d0e4436118fadc7437c5396aa802cadd822d6284759",
    ],
)

rpm(
    name = "tar-2__1.34-10.el9.x86_64",
    sha256 = "449213355bb8fe15f9a9f8c29c0ec87458fc8959fb42bdcf678a63f32fee2505",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tar-1.34-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/449213355bb8fe15f9a9f8c29c0ec87458fc8959fb42bdcf678a63f32fee2505",
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
    name = "targetcli-0__2.1.57-3.el9.aarch64",
    sha256 = "71aee4574ecf55ca3dec350e1dee3f1188909b2b5430f23bb7352baad322e21f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/targetcli-2.1.57-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/71aee4574ecf55ca3dec350e1dee3f1188909b2b5430f23bb7352baad322e21f",
    ],
)

rpm(
    name = "targetcli-0__2.1.57-3.el9.s390x",
    sha256 = "71aee4574ecf55ca3dec350e1dee3f1188909b2b5430f23bb7352baad322e21f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/targetcli-2.1.57-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/71aee4574ecf55ca3dec350e1dee3f1188909b2b5430f23bb7352baad322e21f",
    ],
)

rpm(
    name = "targetcli-0__2.1.57-3.el9.x86_64",
    sha256 = "71aee4574ecf55ca3dec350e1dee3f1188909b2b5430f23bb7352baad322e21f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/targetcli-2.1.57-3.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/71aee4574ecf55ca3dec350e1dee3f1188909b2b5430f23bb7352baad322e21f",
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
    name = "tzdata-0__2025c-1.el9.aarch64",
    sha256 = "a7a70f4e8aa1473153235900a76753aef8f43a4c21eb869012bf4b065cc8b932",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2025c-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a7a70f4e8aa1473153235900a76753aef8f43a4c21eb869012bf4b065cc8b932",
    ],
)

rpm(
    name = "tzdata-0__2025c-1.el9.s390x",
    sha256 = "a7a70f4e8aa1473153235900a76753aef8f43a4c21eb869012bf4b065cc8b932",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tzdata-2025c-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a7a70f4e8aa1473153235900a76753aef8f43a4c21eb869012bf4b065cc8b932",
    ],
)

rpm(
    name = "tzdata-0__2025c-1.el9.x86_64",
    sha256 = "a7a70f4e8aa1473153235900a76753aef8f43a4c21eb869012bf4b065cc8b932",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2025c-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a7a70f4e8aa1473153235900a76753aef8f43a4c21eb869012bf4b065cc8b932",
    ],
)

rpm(
    name = "unbound-libs-0__1.24.2-2.el9.aarch64",
    sha256 = "90dbe1a0feb24693819fd890953bfafd24324a3b52722e782ae3d938f2a0a911",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/unbound-libs-1.24.2-2.el9.aarch64.rpm",
    ],
)

rpm(
    name = "unbound-libs-0__1.24.2-2.el9.s390x",
    sha256 = "70e50eee29cf83a811f4ae1861a2fe2398caf01fa2efcc3a2cfc46cc01bebddc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/unbound-libs-1.24.2-2.el9.s390x.rpm",
    ],
)

rpm(
    name = "unbound-libs-0__1.24.2-2.el9.x86_64",
    sha256 = "ba15a24c05f917d88acf555f77ac8fb5bd1de466d980fd58c4571809af53b1da",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/unbound-libs-1.24.2-2.el9.x86_64.rpm",
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
    name = "vim-minimal-2__8.2.2637-25.el9.aarch64",
    sha256 = "d0aae74ac54fc5234c436d42899cd5c2be1ea9646b9fb58f84a35c71f5196339",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.2.2637-25.el9.aarch64.rpm",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-25.el9.s390x",
    sha256 = "31583b0d5c3a1b7d50497f1aae5bc679a1b57a1f76cf54b554cb20f1e7792a6a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/vim-minimal-8.2.2637-25.el9.s390x.rpm",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-25.el9.x86_64",
    sha256 = "a7b4f3621e2e64fae37ae18e78930ae13d3ff78f40fd18598e914bf2eed0c443",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-25.el9.x86_64.rpm",
    ],
)

rpm(
    name = "virt-lint-0__0.0.1-1.el9.aarch64",
    sha256 = "6b1ba3053844b46e7cd1f200ccc022f6c747eae90bc79b257a183971e9c25a22",
    urls = [
        "https://virt-lint.k8r.cz/aarch64/virt-lint-0.0.1-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "virt-lint-0__0.0.1-1.el9.s390x",
    sha256 = "c98ee84c976d90eb87dfe4653a0981e3e7ba2d08f5069cfffb99bc93849e76da",
    urls = [
        "https://virt-lint.k8r.cz/s390x/virt-lint-0.0.1-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "virt-lint-0__0.0.1-1.el9.x86_64",
    sha256 = "f18208e9a9f705b1078d63e2d4e12725362633235410efd438892d89e37e1c0f",
    urls = [
        "https://virt-lint.k8r.cz/x86_64/virt-lint-0.0.1-1.el9.x86_64.rpm",
    ],
)

rpm(
    name = "virt-lint-devel-0__0.0.1-1.el9.aarch64",
    sha256 = "7a39022f953e288a4bbe1e544f15e067b44e38ff83d3755b7f800e9fc500c3f4",
    urls = [
        "https://virt-lint.k8r.cz/aarch64/virt-lint-devel-0.0.1-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "virt-lint-devel-0__0.0.1-1.el9.s390x",
    sha256 = "29d1935522ade1e994059b966ed4549b72d1938464c6a1885bd7503c8addf38b",
    urls = [
        "https://virt-lint.k8r.cz/s390x/virt-lint-devel-0.0.1-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "virt-lint-devel-0__0.0.1-1.el9.x86_64",
    sha256 = "309258fbd747d189db327612436b8ca57e637151a77ddfc3aeac6db172728f65",
    urls = [
        "https://virt-lint.k8r.cz/x86_64/virt-lint-devel-0.0.1-1.el9.x86_64.rpm",
    ],
)

rpm(
    name = "virt-lint-validators-python-0__0.0.1-1.el9.aarch64",
    sha256 = "8669e04fbaefee772b9b4aac84c5d0f4a05b167d5376fd226b10c02ce3fd2293",
    urls = [
        "https://virt-lint.k8r.cz/aarch64/virt-lint-validators-python-0.0.1-1.el9.aarch64.rpm",
    ],
)

rpm(
    name = "virt-lint-validators-python-0__0.0.1-1.el9.s390x",
    sha256 = "a4a5b31f41b757c4278e8a1d3937647d0a5e66d5ea31f8f0f035f208cfd4e8ec",
    urls = [
        "https://virt-lint.k8r.cz/s390x/virt-lint-validators-python-0.0.1-1.el9.s390x.rpm",
    ],
)

rpm(
    name = "virt-lint-validators-python-0__0.0.1-1.el9.x86_64",
    sha256 = "a9d2fd1283dc2d9275256a687007a3fb8122100c8458ae62edbad10b44626abf",
    urls = [
        "https://virt-lint.k8r.cz/x86_64/virt-lint-validators-python-0.0.1-1.el9.x86_64.rpm",
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
    name = "which-0__2.21-29.el9.x86_64",
    sha256 = "c69af7b876363091bbeb99b4adfbab743f91da3c45478bb7a055c441e395174d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/which-2.21-29.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c69af7b876363091bbeb99b4adfbab743f91da3c45478bb7a055c441e395174d",
    ],
)

rpm(
    name = "which-0__2.21-30.el9.aarch64",
    sha256 = "e31074fa3d7bbfb387f7d6d2c97988069594a6cd6c0bd7c798036a4f8ed7ab48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/which-2.21-30.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e31074fa3d7bbfb387f7d6d2c97988069594a6cd6c0bd7c798036a4f8ed7ab48",
    ],
)

rpm(
    name = "which-0__2.21-30.el9.s390x",
    sha256 = "671cebde29b96ff50fd35e09b867505721144b4507c1c81ee60276f7c14f8723",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/which-2.21-30.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/671cebde29b96ff50fd35e09b867505721144b4507c1c81ee60276f7c14f8723",
    ],
)

rpm(
    name = "which-0__2.21-30.el9.x86_64",
    sha256 = "d602350243d5950c473624788e78e783461e5db242cc0a2d7a988e1b9a3e079b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/which-2.21-30.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d602350243d5950c473624788e78e783461e5db242cc0a2d7a988e1b9a3e079b",
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
