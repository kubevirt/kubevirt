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
    name = "acl-0__2.3.2-4.el10.aarch64",
    sha256 = "e5c1d6460330fabe5ef57fb4b13d46ab0840f93556d898b5179f1b267f34455f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/acl-2.3.2-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "acl-0__2.3.2-4.el10.s390x",
    sha256 = "295d62b3d46571e5327671616bff8d1872af066f41719e09d5e0554d00001e49",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/acl-2.3.2-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "acl-0__2.3.2-4.el10.x86_64",
    sha256 = "fd89f3c793d09fe633bf7721da719d29d599d01f65aaaa355b1b308a6fa580f2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/acl-2.3.2-4.el10.x86_64.rpm",
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
    name = "alternatives-0__1.30-2.el10.aarch64",
    sha256 = "13d1cae28aecbc13bee2cf23391ec2ee41d39c51c9bb47f466fbad133d38f5c9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/alternatives-1.30-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "alternatives-0__1.30-2.el10.s390x",
    sha256 = "ab4f800759f602c25f483681b126b4eced6ba81331c9b613dd47a229379c71e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/alternatives-1.30-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "alternatives-0__1.30-2.el10.x86_64",
    sha256 = "1c8b83bf3dd0fa8d998a3c801986f50ea3661c2f8a21c60971c0391c381919c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/alternatives-1.30-2.el10.x86_64.rpm",
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
    name = "audit-libs-0__4.0.3-5.el10.aarch64",
    sha256 = "f45973727e2dea77b2209bc9795c890abac187383a596b3cb81ab066b11ddb90",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/audit-libs-4.0.3-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "audit-libs-0__4.0.3-5.el10.s390x",
    sha256 = "1d7617a754258f58b0986c6f944621819381543eee344f60f348fc44bc2274c1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/audit-libs-4.0.3-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "audit-libs-0__4.0.3-5.el10.x86_64",
    sha256 = "a2be49cd9497b28aa9688b6e58bce216797c868559d249e3a08034e22d1e86f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/audit-libs-4.0.3-5.el10.x86_64.rpm",
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
    name = "augeas-libs-0__1.14.2-0.9.20260120gitf4135e3.el10.s390x",
    sha256 = "03bb6de04d7f5c64cf30fd0fe26301508b8fad5520c25fd1fc291132b01e0c7a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/augeas-libs-1.14.2-0.9.20260120gitf4135e3.el10.s390x.rpm",
    ],
)

rpm(
    name = "augeas-libs-0__1.14.2-0.9.20260120gitf4135e3.el10.x86_64",
    sha256 = "479f9bb17e1ede3ae449e0cc47474a935b519febdb65c214eedccc1e836aeb8b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/augeas-libs-1.14.2-0.9.20260120gitf4135e3.el10.x86_64.rpm",
    ],
)

rpm(
    name = "authselect-0__1.5.0-8.el10.aarch64",
    sha256 = "6806edc3ab06e45d1077f5d89865ff94d6939004acf365f9eaaf407e02642666",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/authselect-1.5.0-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "authselect-0__1.5.0-8.el10.s390x",
    sha256 = "dad241106db112ab5cb7dcc45164af6a38c614739672d9f0136f4e85b5907d3e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/authselect-1.5.0-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "authselect-0__1.5.0-8.el10.x86_64",
    sha256 = "2a16d12c77181f77189fac10b4a3f76c2d0dd97e230d9074f7d24d2e4967ab35",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/authselect-1.5.0-8.el10.x86_64.rpm",
    ],
)

rpm(
    name = "authselect-libs-0__1.5.0-8.el10.aarch64",
    sha256 = "cc6557c5707792705ffe41b0deae2c76a30382a86e35d3cc812f7d872e9f5871",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/authselect-libs-1.5.0-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "authselect-libs-0__1.5.0-8.el10.s390x",
    sha256 = "890def3a93b6204476966957eee6a893adc9ef76f0212873fa240279575f4ad3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/authselect-libs-1.5.0-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "authselect-libs-0__1.5.0-8.el10.x86_64",
    sha256 = "c3cb5c662f1225e0c1f90c406c2ab3bfad8191a4d1b46614b49dfd298f33c53a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/authselect-libs-1.5.0-8.el10.x86_64.rpm",
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
    name = "basesystem-0__11-22.el10.aarch64",
    sha256 = "76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/basesystem-11-22.el10.noarch.rpm",
    ],
)

rpm(
    name = "basesystem-0__11-22.el10.s390x",
    sha256 = "76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/basesystem-11-22.el10.noarch.rpm",
    ],
)

rpm(
    name = "basesystem-0__11-22.el10.x86_64",
    sha256 = "76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/basesystem-11-22.el10.noarch.rpm",
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
    name = "bash-0__5.2.26-6.el10.aarch64",
    sha256 = "3f42c3de9fddc6e6c08f7c603ce29ed96d8d66f4425ce1c27bcb0d7d0e0490b5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/bash-5.2.26-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "bash-0__5.2.26-6.el10.s390x",
    sha256 = "07261872bd05c23366da7c2529b776dccfdf1a33c99d784370ebfde32d8909d7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/bash-5.2.26-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "bash-0__5.2.26-6.el10.x86_64",
    sha256 = "31eaf885847a6671a93e2b6e0d48e937ae5520f0442265aae19f4294260b5618",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/bash-5.2.26-6.el10.x86_64.rpm",
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
    name = "binutils-0__2.41-60.el10.aarch64",
    sha256 = "829fe311199f54f58c0b0e5a8297b6b9c89ba7cce31e51b9a575ddd5f8aaf80f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/binutils-2.41-60.el10.aarch64.rpm",
    ],
)

rpm(
    name = "binutils-0__2.41-60.el10.s390x",
    sha256 = "cfb7608108550dc979945bf8bbbc99dc201933e81252020f662799c0fbe5c8a3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/binutils-2.41-60.el10.s390x.rpm",
    ],
)

rpm(
    name = "binutils-0__2.41-60.el10.x86_64",
    sha256 = "a948054c9e555dfd17d22864d345a42b948a6a60bdaea75f1324e48ec6aa285e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/binutils-2.41-60.el10.x86_64.rpm",
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
    name = "binutils-gold-0__2.41-60.el10.aarch64",
    sha256 = "eb1c2c492ecbf6affd5d22d29b4b773fae418aa875b85af7c7187d5f975b58b4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.41-60.el10.aarch64.rpm",
    ],
)

rpm(
    name = "binutils-gold-0__2.41-60.el10.s390x",
    sha256 = "0e9c9469fa781dfada84de48b56996aa185a72bcc207eb12cd4d97dff322fd4b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/binutils-gold-2.41-60.el10.s390x.rpm",
    ],
)

rpm(
    name = "binutils-gold-0__2.41-60.el10.x86_64",
    sha256 = "2ec08ee46033739b426b8275e6c0d271e833b471a8ef5658f57372963df1f44f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.41-60.el10.x86_64.rpm",
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
    name = "bzip2-0__1.0.8-25.el10.aarch64",
    sha256 = "30fd7d37e3f06d0b06b6f3e6fda58fd9d54582b0e497795719d81dc68ac88ba7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.8-25.el10.aarch64.rpm",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-25.el10.s390x",
    sha256 = "c9208b97a6a3e2cb7fc84a7bea4e330399cc6ef892c3f0abe20d5df10797eade",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/bzip2-1.0.8-25.el10.s390x.rpm",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-25.el10.x86_64",
    sha256 = "ff7f8e9c3cc936d35033ec40545ee4a836db27c30c240d3aa39be4c8b0fda631",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.8-25.el10.x86_64.rpm",
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
    name = "bzip2-libs-0__1.0.8-25.el10.aarch64",
    sha256 = "ac836c2c133077d0e71092f2c21e69d3985ace8458af527440e13b7edf165beb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.8-25.el10.aarch64.rpm",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-25.el10.s390x",
    sha256 = "219adea56b92ecf22cb63fad38638e16115df270b78ea1fbd3cc1b183caf69a4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/bzip2-libs-1.0.8-25.el10.s390x.rpm",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-25.el10.x86_64",
    sha256 = "309c7dbb857254655c51c4ab02d8038137c1363058542d8701c9272609f5b433",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.8-25.el10.x86_64.rpm",
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
    name = "ca-certificates-0__2025.2.80_v9.0.305-102.el10.aarch64",
    sha256 = "a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/ca-certificates-2025.2.80_v9.0.305-102.el10.noarch.rpm",
    ],
)

rpm(
    name = "ca-certificates-0__2025.2.80_v9.0.305-102.el10.s390x",
    sha256 = "a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/ca-certificates-2025.2.80_v9.0.305-102.el10.noarch.rpm",
    ],
)

rpm(
    name = "ca-certificates-0__2025.2.80_v9.0.305-102.el10.x86_64",
    sha256 = "a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/ca-certificates-2025.2.80_v9.0.305-102.el10.noarch.rpm",
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
    name = "capstone-0__5.0.1-6.el10.aarch64",
    sha256 = "be12ff671fc1244c69b39284b61f4a7e825570d11176dcd83e8476010157db92",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/capstone-5.0.1-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "capstone-0__5.0.1-6.el10.s390x",
    sha256 = "f94850c0dedde1efd687de604a99f6461ec2cb394184f76e3d2d17af0654f0d0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/capstone-5.0.1-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "capstone-0__5.0.1-6.el10.x86_64",
    sha256 = "aa46343e831205d94b08f3d692f88b3a84a16f35b260152684ea10183d972160",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/capstone-5.0.1-6.el10.x86_64.rpm",
    ],
)

rpm(
    name = "centos-gpg-keys-0__10.0-19.el10.aarch64",
    sha256 = "9de24d7bd3ee5b686170e6f27bd99b6550d02a8d4df5d00a7c6a83750f4d4b0a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-10.0-19.el10.noarch.rpm",
    ],
)

rpm(
    name = "centos-gpg-keys-0__10.0-19.el10.s390x",
    sha256 = "9de24d7bd3ee5b686170e6f27bd99b6550d02a8d4df5d00a7c6a83750f4d4b0a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/centos-gpg-keys-10.0-19.el10.noarch.rpm",
    ],
)

rpm(
    name = "centos-gpg-keys-0__10.0-19.el10.x86_64",
    sha256 = "9de24d7bd3ee5b686170e6f27bd99b6550d02a8d4df5d00a7c6a83750f4d4b0a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-10.0-19.el10.noarch.rpm",
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
    name = "centos-stream-release-0__10.0-19.el10.aarch64",
    sha256 = "b47742c7d0ee92454c15b97bca9240b61de31547d7de039f67ba498703623188",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/centos-stream-release-10.0-19.el10.noarch.rpm",
    ],
)

rpm(
    name = "centos-stream-release-0__10.0-19.el10.s390x",
    sha256 = "b47742c7d0ee92454c15b97bca9240b61de31547d7de039f67ba498703623188",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/centos-stream-release-10.0-19.el10.noarch.rpm",
    ],
)

rpm(
    name = "centos-stream-release-0__10.0-19.el10.x86_64",
    sha256 = "b47742c7d0ee92454c15b97bca9240b61de31547d7de039f67ba498703623188",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/centos-stream-release-10.0-19.el10.noarch.rpm",
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
    name = "centos-stream-repos-0__10.0-19.el10.aarch64",
    sha256 = "5fa429468121be8530982d8776e69e7cf91f2c4f159d5152169898a451baf676",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-10.0-19.el10.noarch.rpm",
    ],
)

rpm(
    name = "centos-stream-repos-0__10.0-19.el10.s390x",
    sha256 = "5fa429468121be8530982d8776e69e7cf91f2c4f159d5152169898a451baf676",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/centos-stream-repos-10.0-19.el10.noarch.rpm",
    ],
)

rpm(
    name = "centos-stream-repos-0__10.0-19.el10.x86_64",
    sha256 = "5fa429468121be8530982d8776e69e7cf91f2c4f159d5152169898a451baf676",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-10.0-19.el10.noarch.rpm",
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
    name = "coreutils-single-0__9.5-6.el10.aarch64",
    sha256 = "d1cfc460e243e2fc1934b8b0d173d2f2b37bb69b1eedef2cfdc93619cfe6998a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/coreutils-single-9.5-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "coreutils-single-0__9.5-6.el10.s390x",
    sha256 = "d6ba2511cf43ebd40110b9b1786923da409ff73ae90aac67a12121e9259beb49",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/coreutils-single-9.5-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "coreutils-single-0__9.5-6.el10.x86_64",
    sha256 = "b1f91efb9d930b8b021d3648610029f78433a65b39b578e69e575c9767be61d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/coreutils-single-9.5-6.el10.x86_64.rpm",
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
    name = "cpp-0__14.3.1-4.3.el10.aarch64",
    sha256 = "d88d1b7c37bc90ffaae0e729a8314cee7a2d3d3b6d24279fbf01c63c2c307408",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/cpp-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "cpp-0__14.3.1-4.3.el10.s390x",
    sha256 = "d627790816fdaf878c633887d4f35ab3aeee8e703057db981f299246f286fba7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/cpp-14.3.1-4.3.el10.s390x.rpm",
    ],
)

rpm(
    name = "cpp-0__14.3.1-4.3.el10.x86_64",
    sha256 = "d173162b43fbf0948354cc90e68bbc37e31b026943f0ac1502a0367b535b7bb2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/cpp-14.3.1-4.3.el10.x86_64.rpm",
    ],
)

rpm(
    name = "cracklib-0__2.9.11-8.el10.aarch64",
    sha256 = "04112224e2f1b7027ef15ee4cb9ede5bb89426b29f150692778d8f7ca155eea9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/cracklib-2.9.11-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "cracklib-0__2.9.11-8.el10.s390x",
    sha256 = "2e0c0ba830f1a497461b1a7f6e76f5d409c9bf87d2c4a6874957abe3fdb74be3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cracklib-2.9.11-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "cracklib-0__2.9.11-8.el10.x86_64",
    sha256 = "4d648a415fe67550a22ff50befdaf9a33ccb55dbc9a2e3d4121ddfbe2ee843f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cracklib-2.9.11-8.el10.x86_64.rpm",
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
    name = "cracklib-dicts-0__2.9.11-8.el10.aarch64",
    sha256 = "51210426186039c77239cbb3c710acbc9f7778ca44292204ffa2ecf1448e2c1e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/cracklib-dicts-2.9.11-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.11-8.el10.s390x",
    sha256 = "45cf94fabce8c9c035df7db91b19fefec5cfef5cee54505cabebce1822e3099d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cracklib-dicts-2.9.11-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.11-8.el10.x86_64",
    sha256 = "79dd2684b0ae0cbc47739c0e292f17243eb448b92f74bae893cf1eb4aba14703",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cracklib-dicts-2.9.11-8.el10.x86_64.rpm",
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
    name = "crypto-policies-0__20251127-1.git27c2902.el10.aarch64",
    sha256 = "84f438e426f45ecf1ce51fc71a1bb4c1a1a1b5ee63faf793273ee5d6aaaecb33",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/crypto-policies-20251127-1.git27c2902.el10.noarch.rpm",
    ],
)

rpm(
    name = "crypto-policies-0__20251127-1.git27c2902.el10.s390x",
    sha256 = "84f438e426f45ecf1ce51fc71a1bb4c1a1a1b5ee63faf793273ee5d6aaaecb33",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/crypto-policies-20251127-1.git27c2902.el10.noarch.rpm",
    ],
)

rpm(
    name = "crypto-policies-0__20251127-1.git27c2902.el10.x86_64",
    sha256 = "84f438e426f45ecf1ce51fc71a1bb4c1a1a1b5ee63faf793273ee5d6aaaecb33",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/crypto-policies-20251127-1.git27c2902.el10.noarch.rpm",
    ],
)

rpm(
    name = "curl-0__8.12.1-4.el10.aarch64",
    sha256 = "7fe56b8ad3db9141cd721455717109785447e79358f4541d27bec012230db8c4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/curl-8.12.1-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "curl-0__8.12.1-4.el10.s390x",
    sha256 = "2aa147ae00c5fc1a0264f785127771e6ced0f4ee3d82a9bd6c48d1f240e44c7c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/curl-8.12.1-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "curl-0__8.12.1-4.el10.x86_64",
    sha256 = "30b38c7b64e1a33c6b69634fcb4b9d9f1714f9bd6530ee0175fc3be149f23d9b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/curl-8.12.1-4.el10.x86_64.rpm",
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
    name = "cyrus-sasl-gssapi-0__2.1.28-27.el10.aarch64",
    sha256 = "f030977f59727e389143e1813c5fc848799abbea48ed60aca460dc2eb1a79637",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-gssapi-2.1.28-27.el10.aarch64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.28-27.el10.s390x",
    sha256 = "28c75a50cf3f092920ac56fb65805e9c875fc95d4e76bce0e1cc6b6d21e3fba3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cyrus-sasl-gssapi-2.1.28-27.el10.s390x.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.28-27.el10.x86_64",
    sha256 = "f9ab02ca832fe4d5c1e1ee3abd7ff3db3815d164561350316032a82b44d68b6c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-gssapi-2.1.28-27.el10.x86_64.rpm",
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
    name = "cyrus-sasl-lib-0__2.1.28-27.el10.aarch64",
    sha256 = "917d6b8d2eff0dd71b55646c758b938ac7b9f0a298f2dffae5948c9865215067",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-lib-2.1.28-27.el10.aarch64.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.28-27.el10.s390x",
    sha256 = "b40557a0d21461db27adf093fe6a72ec17a243f6743a3d1e26c32601753e97ee",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cyrus-sasl-lib-2.1.28-27.el10.s390x.rpm",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.28-27.el10.x86_64",
    sha256 = "ea78a83980b03f3709266f5e4c96b41699fe8d5f7003fb9503c3a7529c6ca46a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-lib-2.1.28-27.el10.x86_64.rpm",
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
    name = "dbus-1__1.14.10-5.el10.aarch64",
    sha256 = "2f00025969ff8b32c254ec38919908120f83847e98285413c718d1ad0b2a8766",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-1.14.10-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "dbus-1__1.14.10-5.el10.s390x",
    sha256 = "2a746bab9a5c03b6bc2f680ad3be8ecf935404c17f6488de44e77ab61bdfedb8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-1.14.10-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "dbus-1__1.14.10-5.el10.x86_64",
    sha256 = "c71f38667ecebd3ba0adf415ccf181209330bb0e2ca9ad0bf4de9828b370b9e4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-1.14.10-5.el10.x86_64.rpm",
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
    name = "dbus-broker-0__36-4.el10.aarch64",
    sha256 = "3716b1d4daa23c6fd965175473464ddfa91ea5651a68298a2e0b139021e23035",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-broker-36-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "dbus-broker-0__36-4.el10.s390x",
    sha256 = "3d1ec31218c8925602bb7fcd88150c628a0e24ab5cc4e7c63b85785202756283",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-broker-36-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "dbus-broker-0__36-4.el10.x86_64",
    sha256 = "a0778052571fe74351500a06e765219fcf53c0ca2eeb4969a2682a36ee9f9c10",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-broker-36-4.el10.x86_64.rpm",
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
    name = "dbus-common-1__1.14.10-5.el10.aarch64",
    sha256 = "1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-common-1.14.10-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "dbus-common-1__1.14.10-5.el10.s390x",
    sha256 = "1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-common-1.14.10-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "dbus-common-1__1.14.10-5.el10.x86_64",
    sha256 = "1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-common-1.14.10-5.el10.noarch.rpm",
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
    name = "dbus-libs-1__1.14.10-5.el10.aarch64",
    sha256 = "976a662683dc4f8235303cd6065f589c4d4728671116827b2002ac1fd4a74a72",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.14.10-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "dbus-libs-1__1.14.10-5.el10.s390x",
    sha256 = "261a5aee8fd8417bdb0b629b7ae4141cec92de79d32b45982c66cc82878f3175",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-libs-1.14.10-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "dbus-libs-1__1.14.10-5.el10.x86_64",
    sha256 = "7cd5d99568a89ef7100ae60d44aa270cbf5882e95cbc8f43497696f81c664284",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.14.10-5.el10.x86_64.rpm",
    ],
)

rpm(
    name = "device-mapper-10__1.02.206-3.el10.aarch64",
    sha256 = "185e448e20167139d421ce9177c63deaa75cd1a875110b4c0f2dd05481214141",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.206-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-10__1.02.206-3.el10.s390x",
    sha256 = "0da51b07ab865cae27da5d651513fb93358503e5160c1f034cf4db9e7393708e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/device-mapper-1.02.206-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "device-mapper-10__1.02.206-3.el10.x86_64",
    sha256 = "30fc452056e3b1117f3d48b88a7a9a1633690a9cb37b6c4c663ddd8b44dcf159",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.206-3.el10.x86_64.rpm",
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
    name = "device-mapper-libs-10__1.02.206-3.el10.aarch64",
    sha256 = "8a6d569a6a478c816a4a596dde65e299e1072443e78237c5c44428a9557ed6bd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.206-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-libs-10__1.02.206-3.el10.s390x",
    sha256 = "aa3c0b549b37445d7e5f754a09efd474ad385580ff4c4c612ca2eed3b2528fdd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/device-mapper-libs-1.02.206-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "device-mapper-libs-10__1.02.206-3.el10.x86_64",
    sha256 = "9342a7c107577149f860e71e4dd09002da1da5cf82e08d85a361758a6eaa6fee",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.206-3.el10.x86_64.rpm",
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
    name = "device-mapper-multipath-libs-0__0.9.9-15.el10.aarch64",
    sha256 = "14a7c7b2affa61a420527a77beba4b9968269b42671ef5bd0a690f11dd3241c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.9.9-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.9.9-15.el10.x86_64",
    sha256 = "e88082ce08b8067cd35fd27daadddd11bd35d55be78f4e3985eb3074c52ef464",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.9.9-15.el10.x86_64.rpm",
    ],
)

rpm(
    name = "diffutils-0__3.10-8.el10.aarch64",
    sha256 = "d06031d2cd612618343d29186bc873cafd52c9e71efae6d04dcb494de2b53b58",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/diffutils-3.10-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "diffutils-0__3.10-8.el10.s390x",
    sha256 = "4668ee01492723f3a4fd094ff49ef2485ab3f17d1e30b19103a70e4b24a7c3e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/diffutils-3.10-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "diffutils-0__3.10-8.el10.x86_64",
    sha256 = "96882ec03cfc01ae557f0ec547fb8d346179eb705c899bec0533eafda7c1bd80",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/diffutils-3.10-8.el10.x86_64.rpm",
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
    name = "dmidecode-1__3.6-5.el10.aarch64",
    sha256 = "381d5765cc5b1346f47dea4818c013bc308eb2cd9a76a9a3c4046a6982910956",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dmidecode-3.6-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "dmidecode-1__3.6-5.el10.x86_64",
    sha256 = "332cfc77ea06aab27c93c1cf2382e50bf62ddad534c526795083a98ec10668c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dmidecode-3.6-5.el10.x86_64.rpm",
    ],
)

rpm(
    name = "duktape-0__2.7.0-10.el10.aarch64",
    sha256 = "c390a43273231fec4a25199690e0106268e3eb46a1592d4cd68cf56909efce5e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/duktape-2.7.0-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "duktape-0__2.7.0-10.el10.s390x",
    sha256 = "7cafae00eb1aa432b96c9fb9a6df9789d3ccf03515b7714c16ff8dcbaa7210d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/duktape-2.7.0-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "duktape-0__2.7.0-10.el10.x86_64",
    sha256 = "23b7d2905723ed7adabe3362c54d54f0745c908029ec3be79bd881770d2c591a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/duktape-2.7.0-10.el10.x86_64.rpm",
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
    name = "e2fsprogs-0__1.47.1-5.el10.aarch64",
    sha256 = "fd5592fb0e7c1ae9ae023eafb55c7ae3ac71c94c44e1f498f1eb56c1940f3c40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/e2fsprogs-1.47.1-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "e2fsprogs-0__1.47.1-5.el10.s390x",
    sha256 = "23803262e02ed5ad895284267c828bee4620aa498326a36c659a36dcd12bce9e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/e2fsprogs-1.47.1-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "e2fsprogs-0__1.47.1-5.el10.x86_64",
    sha256 = "736291b66f30c8ad543f5bed5375c92bc8a2e3bce1704a77f5b727ee844fb0dd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.47.1-5.el10.x86_64.rpm",
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
    name = "e2fsprogs-libs-0__1.47.1-5.el10.aarch64",
    sha256 = "e8b7d03d574363beaebef73048b8fe8461ed7b1206152b81eb0852f5c01d533b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/e2fsprogs-libs-1.47.1-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.47.1-5.el10.s390x",
    sha256 = "25bb41764aefa735e891df10d2846b4c86f00f8eaabaf9a66acf08ebf290b700",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/e2fsprogs-libs-1.47.1-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.47.1-5.el10.x86_64",
    sha256 = "d73c79a7bda1ce465707d82fa6b9777fcd2776301a6f6722ca323b4c9337c64b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.47.1-5.el10.x86_64.rpm",
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
    name = "edk2-aarch64-0__20251114-2.el10.aarch64",
    sha256 = "14b8a283058af0f4fb30f4e7c2235945b6420143a441ed31a3c1e976505ba2b8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20251114-2.el10.noarch.rpm",
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
    name = "edk2-ovmf-0__20251114-2.el10.s390x",
    sha256 = "7568e5b29bb3644cceab0faf2d59578ec8de88f0dafad4808b91bc2680dee682",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/edk2-ovmf-20251114-2.el10.noarch.rpm",
    ],
)

rpm(
    name = "edk2-ovmf-0__20251114-2.el10.x86_64",
    sha256 = "7568e5b29bb3644cceab0faf2d59578ec8de88f0dafad4808b91bc2680dee682",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20251114-2.el10.noarch.rpm",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.194-1.el10.aarch64",
    sha256 = "280b20ad99ef6a5097776c729d7b7ccc679d9eb4c977d32ee92af4641a8e745d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.194-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.194-1.el10.s390x",
    sha256 = "70da3d5d468afd29b27733d38e61b79ebeee2de0e75c5f11b9edbd5e151aa4fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.194-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.194-1.el10.x86_64",
    sha256 = "5ac0c4084d431eda2da1db7698d10d62195ec03f44e25755f4d6b8133d6606e6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.194-1.el10.x86_64.rpm",
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
    name = "elfutils-default-yama-scope-0__0.194-1.el10.aarch64",
    sha256 = "35f822daa4ecdce5dc624e6875d3b55491f8b5e0696d070672d2678036ad2ad0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.194-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.194-1.el10.s390x",
    sha256 = "35f822daa4ecdce5dc624e6875d3b55491f8b5e0696d070672d2678036ad2ad0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.194-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.194-1.el10.x86_64",
    sha256 = "35f822daa4ecdce5dc624e6875d3b55491f8b5e0696d070672d2678036ad2ad0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.194-1.el10.noarch.rpm",
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
    name = "elfutils-libelf-0__0.194-1.el10.aarch64",
    sha256 = "97c0ad3cb708215214b2c79fce3e840eeb023e751a679c8da23b0ac24c9286b4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.194-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.194-1.el10.s390x",
    sha256 = "01795d511317f3717a7f837bf9e0ac92d5db4da33eb1fd5b93987313f6638fcf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.194-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.194-1.el10.x86_64",
    sha256 = "1bfacc8e5b007821e21f82b50aa1ab3f1a2959fd4f3361c277e75db43bd69284",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.194-1.el10.x86_64.rpm",
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
    name = "elfutils-libs-0__0.194-1.el10.aarch64",
    sha256 = "ca36cc469aae95470c33e08087f5176615ebe42453e06695c8897da87c8e6185",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.194-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "elfutils-libs-0__0.194-1.el10.s390x",
    sha256 = "f421c5e17662e93a3f0ba2d4511a206c4ee08c7bf2f7ee40e59d8c803c7c6097",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.194-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "elfutils-libs-0__0.194-1.el10.x86_64",
    sha256 = "6a6cce578a25f607ab0c593d889c9c52487f6c9019d3f9b4c3ebc2edb5dbbc89",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.194-1.el10.x86_64.rpm",
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
    name = "expat-0__2.7.3-1.el10.aarch64",
    sha256 = "9d093b8a289a4fbac304097d8d628744fa0ea88f3a50a64c4ee1c657cb42a5c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/expat-2.7.3-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "expat-0__2.7.3-1.el10.s390x",
    sha256 = "5fce4ab3c8a5e188f560bdbac6f780e36af2e71210f765153ee2c9328b8a2a5f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/expat-2.7.3-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "expat-0__2.7.3-1.el10.x86_64",
    sha256 = "e00c0876574daba5e70a3e2c86e21823fae1269b7a123d08ff5493a59dde3f36",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/expat-2.7.3-1.el10.x86_64.rpm",
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
    name = "filesystem-0__3.18-17.el10.aarch64",
    sha256 = "6c4d8ecaf8b45c8d7d588c6ebe368a77805ed84830d0bc3b38e4c8e499514aba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/filesystem-3.18-17.el10.aarch64.rpm",
    ],
)

rpm(
    name = "filesystem-0__3.18-17.el10.s390x",
    sha256 = "087e8def18ded2dd2a96f7a4292a3654704807d05f4424c43c0f5c873d7f9cb5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/filesystem-3.18-17.el10.s390x.rpm",
    ],
)

rpm(
    name = "filesystem-0__3.18-17.el10.x86_64",
    sha256 = "bcfb13f67c813d645f47e0a56d4bb76c0863deaf64ba93be8e0c30eecdc1e45e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/filesystem-3.18-17.el10.x86_64.rpm",
    ],
)

rpm(
    name = "findutils-1__4.10.0-5.el10.aarch64",
    sha256 = "f0e4db5b6e713c75e097e80218c592de4e6cb85d353f0933f64714df11b178b2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/findutils-4.10.0-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "findutils-1__4.10.0-5.el10.s390x",
    sha256 = "da20bdfeb9053ac3a1689d2ee2281298ee119175a8d486e4bb3eed1bc2857a94",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/findutils-4.10.0-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "findutils-1__4.10.0-5.el10.x86_64",
    sha256 = "c646c7c108a007d62792aa66e0bc9326312089a0f8bc1c9e9300b301fd2e4276",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/findutils-4.10.0-5.el10.x86_64.rpm",
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
    name = "fips-provider-next-0__1.2.0-3.el10.aarch64",
    sha256 = "8b1a3f9bcf30fa7850ff5f068bb01b0c0b07385135a0155a50257f014f3156bd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/fips-provider-next-1.2.0-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "fips-provider-next-0__1.2.0-3.el10.s390x",
    sha256 = "f2d281204b6118905a9668f2cbc5ee832aa65cca751c3cfd6df5f94fd880c8ae",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/fips-provider-next-1.2.0-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "fips-provider-next-0__1.2.0-3.el10.x86_64",
    sha256 = "d1314bd57fd4e4bb2030519cd79ab562f8ce64866d51827cd4e0f73c190a6c9c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/fips-provider-next-1.2.0-3.el10.x86_64.rpm",
    ],
)

rpm(
    name = "fips-provider-next-0__1.2.0-7.el9.s390x",
    sha256 = "0f1f863a00b32b0517c785c0b68f839426cae4c087fde39c8c5d91ab3cdf2ee4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/fips-provider-next-1.2.0-7.el9.s390x.rpm",
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
    name = "fuse-0__2.9.9-25.el10.s390x",
    sha256 = "6d0dd7c5dc828fc93d96ff215d90324f8efd9e88a9512081f4cf6d6323387a2f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/fuse-2.9.9-25.el10.s390x.rpm",
    ],
)

rpm(
    name = "fuse-0__2.9.9-25.el10.x86_64",
    sha256 = "0707885f1d8074b5d36d85b4c60a68a10867894b379225302a94f3d54b6d4934",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse-2.9.9-25.el10.x86_64.rpm",
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
    name = "fuse-common-0__3.16.2-5.el10.s390x",
    sha256 = "86983857ec56f535e57283f302d9f344a348b55a9dc5e6e81ef388b397a14e2a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/fuse-common-3.16.2-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "fuse-common-0__3.16.2-5.el10.x86_64",
    sha256 = "eecc51472bf7713a97821ae02898b6811752aa513aa40dc5d380459fce590a40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse-common-3.16.2-5.el10.x86_64.rpm",
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
    name = "fuse-libs-0__2.9.9-25.el10.s390x",
    sha256 = "65b86c79a139100f7d61acbef829a0a345c70316988cd7eb0f573f0c57e98647",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/fuse-libs-2.9.9-25.el10.s390x.rpm",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-25.el10.x86_64",
    sha256 = "a8b094d60b9a7f83a84d8c7b0cdeed565be044dc2ecd170965b2c55ee4fa40f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.9-25.el10.x86_64.rpm",
    ],
)

rpm(
    name = "fuse3-libs-0__3.16.2-5.el10.aarch64",
    sha256 = "919f632731bc755d7c9c81d6faebb3bb703d7460ed72fdd65c453541d3999a72",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/fuse3-libs-3.16.2-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "fuse3-libs-0__3.16.2-5.el10.s390x",
    sha256 = "68501eaef0f538ca7e3731a4968f308ced9ae9c2a1b3b4d310890dd86b1843c5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/fuse3-libs-3.16.2-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "fuse3-libs-0__3.16.2-5.el10.x86_64",
    sha256 = "3482d8de135a306e94f7a35c1f8315b4e6acb699c1871ef28ddb02dc0fbdf7d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse3-libs-3.16.2-5.el10.x86_64.rpm",
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
    name = "gawk-0__5.3.0-6.el10.aarch64",
    sha256 = "16d7b639936dd4c8c977cd5b2ee3f5a02d3235954f67aa7485765a6b146683de",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gawk-5.3.0-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gawk-0__5.3.0-6.el10.s390x",
    sha256 = "0c918acd6aed7bbe461611db414bed4c1871b9ee9e4e5369460e016eb0c6bcbb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gawk-5.3.0-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "gawk-0__5.3.0-6.el10.x86_64",
    sha256 = "ba59a3a4ee8741ed4e0c2517086164a76dc85309947f8b5ca7884f05c08ed959",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gawk-5.3.0-6.el10.x86_64.rpm",
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
    name = "gcc-0__14.3.1-4.3.el10.aarch64",
    sha256 = "0b3278b287510da35e3e4c01e91b8dd369c26ceec1666a056baf638a7a985169",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/gcc-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gcc-0__14.3.1-4.3.el10.s390x",
    sha256 = "0d0a820dcc592e30a947859e38c5a285d33aa8567afde50908fe7b5cf556f30c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gcc-14.3.1-4.3.el10.s390x.rpm",
    ],
)

rpm(
    name = "gcc-0__14.3.1-4.3.el10.x86_64",
    sha256 = "33e41378d8e45c67021c7a10d7a6ecf69836d68a1eab0412a8a0bb95475a0094",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/gcc-14.3.1-4.3.el10.x86_64.rpm",
    ],
)

rpm(
    name = "gdbm-1__1.23-14.el10.aarch64",
    sha256 = "0db16e24bf3d297cc3543842d63143f583de6ee157806b0a3dc51b5740a2722f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gdbm-1.23-14.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gdbm-1__1.23-14.el10.s390x",
    sha256 = "95c556f933af240938736727df962465928b7a556a8586b01e90c647facc2839",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gdbm-1.23-14.el10.s390x.rpm",
    ],
)

rpm(
    name = "gdbm-1__1.23-14.el10.x86_64",
    sha256 = "159a6f1affc65d960c11a8726472699f693cec90a54a0862ad8340d0968f4838",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gdbm-1.23-14.el10.x86_64.rpm",
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
    name = "gdbm-libs-1__1.23-14.el10.aarch64",
    sha256 = "b46628d13eba77191aad6905de11fff87d6f45e52168e5b5365cb1f62078fd4d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gdbm-libs-1.23-14.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-14.el10.s390x",
    sha256 = "38f1f8006c38c8fffa7f298bf3a143943f8611acaee7aad8200edc6bcde534aa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gdbm-libs-1.23-14.el10.s390x.rpm",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-14.el10.x86_64",
    sha256 = "b5f678293062eb1fcba572501d62e215dccfd222c26f5b76d9424f3c188cedee",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.23-14.el10.x86_64.rpm",
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
    name = "gettext-0__0.22.5-6.el10.aarch64",
    sha256 = "27cba50dbb800aaf7f46bffa04003338c797472b334b08344b3633a60e0f1755",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gettext-0.22.5-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gettext-0__0.22.5-6.el10.s390x",
    sha256 = "02ab0b35769a517e0a2c255c4e4f23cfb9f661179355f81687a2d5b5198289d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-0.22.5-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "gettext-0__0.22.5-6.el10.x86_64",
    sha256 = "19430ae2b77a7e4637bfcb70501748a27011f6c1e144a195b7046ecd9e6a96b4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-0.22.5-6.el10.x86_64.rpm",
    ],
)

rpm(
    name = "gettext-envsubst-0__0.22.5-6.el10.aarch64",
    sha256 = "ae3a179fff748702f7ad12bc2d8e58910d724a1d42f4cafb22af8ddfaf2eb216",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gettext-envsubst-0.22.5-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gettext-envsubst-0__0.22.5-6.el10.s390x",
    sha256 = "a9f2345a5875671c4d3a14ae491ea02b535d52ecb6ff65813aba392660d96065",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-envsubst-0.22.5-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "gettext-envsubst-0__0.22.5-6.el10.x86_64",
    sha256 = "f7b90e29f350fd67a2425a9d06c404371f1bbcdc43727452bf27c6c855d9eccf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-envsubst-0.22.5-6.el10.x86_64.rpm",
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
    name = "gettext-libs-0__0.22.5-6.el10.aarch64",
    sha256 = "460e9216dbdd5a5a42bcd49162639e5515020a1caf9a246734ab7c19d5747b8e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gettext-libs-0.22.5-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gettext-libs-0__0.22.5-6.el10.s390x",
    sha256 = "aebeafee7bc7b3513b5210039214447304f4734e8ba4e590cbf74cfe0fb04393",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-libs-0.22.5-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "gettext-libs-0__0.22.5-6.el10.x86_64",
    sha256 = "de538283e9cc0281d53e05c235905a9e5c64ad1ac2533afb915ba75052f540a3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-libs-0.22.5-6.el10.x86_64.rpm",
    ],
)

rpm(
    name = "gettext-runtime-0__0.22.5-6.el10.aarch64",
    sha256 = "76d58cbcdddca202c4eecc30df7692d5f6e847f0ac233227349942b6f860a5da",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gettext-runtime-0.22.5-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gettext-runtime-0__0.22.5-6.el10.s390x",
    sha256 = "59a0988b7180c5b0c78c02b40c60f902f60d363beb3acc379c5d1ffd8fa6dfeb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-runtime-0.22.5-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "gettext-runtime-0__0.22.5-6.el10.x86_64",
    sha256 = "aec2ce3c3805190c65667c617e1ed100b65c251d16896819b0bc933ec3084ebf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-runtime-0.22.5-6.el10.x86_64.rpm",
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
    name = "glib2-0__2.80.4-11.el10.aarch64",
    sha256 = "34720321f5c846b69a1f9b36a928596dcadcf5e11c4d5298cc358c3fa184341d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glib2-2.80.4-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "glib2-0__2.80.4-11.el10.s390x",
    sha256 = "8a5a41fd388e3f2b056effbdb5a88cc361650852679dd9e1942a85ff9e4d87d9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glib2-2.80.4-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "glib2-0__2.80.4-11.el10.x86_64",
    sha256 = "8de6a8233b09fff4cd3acd29017ec6750af1bf5670772dcfa2b2bb80f6f85885",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glib2-2.80.4-11.el10.x86_64.rpm",
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
    name = "glibc-0__2.39-99.el10.aarch64",
    sha256 = "1812456204a03ba8f95f2a57d2dd2c150358b8a7702a4fbfb9065a91c6d150f1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-2.39-99.el10.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-0__2.39-99.el10.s390x",
    sha256 = "dc8579cd4111606b3ee378b8bfbae86588f30ee083797cb60c843fd19829496d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-2.39-99.el10.s390x.rpm",
    ],
)

rpm(
    name = "glibc-0__2.39-99.el10.x86_64",
    sha256 = "cae4b2718a3a864f504056f35159be691f2091f3fa9a11b5a6e14ec1ed66f068",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-2.39-99.el10.x86_64.rpm",
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
    name = "glibc-common-0__2.39-99.el10.aarch64",
    sha256 = "4cb977fcb1ee23b92bbc646a0a76370aa8ed6d2ec791f674be069e769b7e6b21",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-common-2.39-99.el10.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-common-0__2.39-99.el10.s390x",
    sha256 = "644f0a1c2bc83994674bfe3f116799679e729c4d8484dd282c79270bc58ec9dd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-common-2.39-99.el10.s390x.rpm",
    ],
)

rpm(
    name = "glibc-common-0__2.39-99.el10.x86_64",
    sha256 = "48946309a086d9dd6f2dd6cf5ba0cba6560c6f29d6611990b609c043dcee2a0b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-common-2.39-99.el10.x86_64.rpm",
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
    name = "glibc-devel-0__2.39-99.el10.aarch64",
    sha256 = "544ecb6fc79f28a455c809beb222bdca33c7806198885302ae529d66ff100818",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/glibc-devel-2.39-99.el10.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-devel-0__2.39-99.el10.s390x",
    sha256 = "84c11de51a0e0ca7df13ae839b8f5cc9ab5c28ee151f1e78c8646b426e348d9a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/glibc-devel-2.39-99.el10.s390x.rpm",
    ],
)

rpm(
    name = "glibc-devel-0__2.39-99.el10.x86_64",
    sha256 = "b8b686c9cc6115aa8d0b658a1cc1676ebac0f225e49c4f2276b81874519f060b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/glibc-devel-2.39-99.el10.x86_64.rpm",
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
    name = "glibc-langpack-el-0__2.34-245.el9.x86_64",
    sha256 = "7fe311b4bb2e516d1b86667a449ad77b0d056f7d3b43f97069cb33b5fbf8787e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-langpack-el-2.34-245.el9.x86_64.rpm",
    ],
)

rpm(
    name = "glibc-langpack-eu-0__2.34-245.el9.aarch64",
    sha256 = "aa5b6ae892a991354aef90356507e91f926d97eb5444df87248a54947b36d226",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-langpack-eu-2.34-245.el9.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-langpack-eu-0__2.34-245.el9.s390x",
    sha256 = "1033dcf8c4c92f71212fe7258a27e4dfa316960fe3bf2cd330cdba2d0ba4d56b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-langpack-eu-2.34-245.el9.s390x.rpm",
    ],
)

rpm(
    name = "glibc-langpack-fo-0__2.39-99.el10.s390x",
    sha256 = "e78a7b207a4d9f16e33146a9a82b6c8a03307435bcc8c3f389d73671f7d0130d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-langpack-fo-2.39-99.el10.s390x.rpm",
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
    name = "glibc-minimal-langpack-0__2.39-99.el10.aarch64",
    sha256 = "04ef96e01570aab3aecc92c52a33e74042b76e5a531475b7e79bfc38723316e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.39-99.el10.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.39-99.el10.s390x",
    sha256 = "aa21d067dc02b52ed3a580711bc153da4399d95b5953683011c7e6be29f6f5ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.39-99.el10.s390x.rpm",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.39-99.el10.x86_64",
    sha256 = "0d0d2e2e8f233c4dd39a69ccdbfd8224ffa95fd44d3bc5a90328cbe2f5727b26",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.39-99.el10.x86_64.rpm",
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
    name = "glibc-static-0__2.39-99.el10.aarch64",
    sha256 = "f35d8cde12d5c3f2ccda56905011442ce4c649543bb640332115ec4b7eb9f516",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/aarch64/os/Packages/glibc-static-2.39-99.el10.aarch64.rpm",
    ],
)

rpm(
    name = "glibc-static-0__2.39-99.el10.s390x",
    sha256 = "50c5864f59e62b246491eb1986e204f16355b068de80f8ecd416e0fc837e19d1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/glibc-static-2.39-99.el10.s390x.rpm",
    ],
)

rpm(
    name = "glibc-static-0__2.39-99.el10.x86_64",
    sha256 = "ee90381f83bf6d7804c7c0423e8c4a9c83e351f5754eac95d23ec0d4ce2fe9e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/glibc-static-2.39-99.el10.x86_64.rpm",
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
    name = "gmp-1__6.2.1-12.el10.aarch64",
    sha256 = "9bbe58df2a29320daf9b4c36305fcc7f781ab0bdd486736c6d8c685838141a41",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gmp-6.2.1-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gmp-1__6.2.1-12.el10.s390x",
    sha256 = "54d437788539933aa6de0963c6b1303e50b07f17db9ea847a71e19d1b4ef6a66",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gmp-6.2.1-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "gmp-1__6.2.1-12.el10.x86_64",
    sha256 = "6678824b5d45f9b66e8bfeb8f32736e0d710e3b38531a85548f55702d96b63a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gmp-6.2.1-12.el10.x86_64.rpm",
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
    name = "gnupg2-0__2.4.5-3.el10.s390x",
    sha256 = "d1488ea4e7128106cbe7685ab5e608dc70d79fafde3c892b66506d038a5d0a67",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gnupg2-2.4.5-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "gnupg2-0__2.4.5-3.el10.x86_64",
    sha256 = "32438ad3bc18d0d6d146ee6d2d97f34707e1391aa22d8f0d2eec203d67b082cb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gnupg2-2.4.5-3.el10.x86_64.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-2.el10.aarch64",
    sha256 = "34023920a6a73834417f61a1169fe8a3edda3acd1f0d780db037e8be01b3866f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gnutls-3.8.10-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-2.el10.s390x",
    sha256 = "08f1f6d00fd7513d03dc79f41595c3baad7f24171b3b6b8c671754e236999d85",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gnutls-3.8.10-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-2.el10.x86_64",
    sha256 = "0226b47f6900316b131298753165e14869c8e4eedce1d1819ae7a5e5b8bd9fac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gnutls-3.8.10-2.el10.x86_64.rpm",
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
    name = "gnutls-dane-0__3.8.10-2.el10.aarch64",
    sha256 = "04cc91a776f6e19b6861f54d2d27e1ef856467eecde4658f8ac54d0aeb1bcb3c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.8.10-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-2.el10.s390x",
    sha256 = "a66242f54e66cf76af97646047832eece06a47e80ef4f033c31d5f20053a3364",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gnutls-dane-3.8.10-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-2.el10.x86_64",
    sha256 = "349be5a7e270c1d7a9e4c43d0a4c60d3408196d50a32261c0d1dd664a5a10954",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.8.10-2.el10.x86_64.rpm",
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
    name = "gnutls-utils-0__3.8.10-2.el10.aarch64",
    sha256 = "bf3239d5a05fe33a62da3aa79e027b10343be365040535c589c6d3c1a7e59dc0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.8.10-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-2.el10.s390x",
    sha256 = "5c99377ef43bd1a5b641a78305b2f8cf10ccf27331562bb965dd8686637191c4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gnutls-utils-3.8.10-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-2.el10.x86_64",
    sha256 = "c425c9f618245d385afd651d6cb72d84001a8e056bcb12ca3775f287b6349697",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.8.10-2.el10.x86_64.rpm",
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
    name = "gobject-introspection-0__1.79.1-6.el10.aarch64",
    sha256 = "a3bd85b169c321602bafe23ca724dfa2b897379a89384dfd453cbb3a03d25e66",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gobject-introspection-1.79.1-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gobject-introspection-0__1.79.1-6.el10.s390x",
    sha256 = "440ef891180126b7d295bca67df47a23bdf05dff3a43d535826b8aa82ad26bb3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gobject-introspection-1.79.1-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "gobject-introspection-0__1.79.1-6.el10.x86_64",
    sha256 = "80913f97462db46c9962d539f325cef09bf85ab4c415a2c47b445fe96bba84b6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gobject-introspection-1.79.1-6.el10.x86_64.rpm",
    ],
)

rpm(
    name = "grep-0__3.11-10.el10.aarch64",
    sha256 = "d797740f7c738e5e7729949bde3d82274c5c6422242a82c1058fbe71ea0c37e9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/grep-3.11-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "grep-0__3.11-10.el10.s390x",
    sha256 = "d30a1ab1991131978b67f26d6c119f97bb5408a4bebac0294f2ac5417fe12276",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/grep-3.11-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "grep-0__3.11-10.el10.x86_64",
    sha256 = "a0eb701c640cd0a0c9195493a8fc9206fff62174d958ba4af2d92527191f803f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/grep-3.11-10.el10.x86_64.rpm",
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
    name = "guestfs-tools-0__1.54.0-7.el10.s390x",
    sha256 = "a8afff6d24bfb91072d2a0c98ad7d574ac5840da0f5a97e92b717862f8f28492",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/guestfs-tools-1.54.0-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "guestfs-tools-0__1.54.0-7.el10.x86_64",
    sha256 = "8e050750f08fbb8a0fd2b0600e8d3acd966e74155bccafe0cd6eb0aaf9d087a3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.54.0-7.el10.x86_64.rpm",
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
    name = "gzip-0__1.13-3.el10.aarch64",
    sha256 = "9b276d61a13e3c996f059a095881630fab9ec5a4a56a07ddc711e4db0a3362d4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gzip-1.13-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "gzip-0__1.13-3.el10.s390x",
    sha256 = "d76be88d032b4f7525f5414d11081fb930fe338f108830b01b24f8501de3c2d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gzip-1.13-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "gzip-0__1.13-3.el10.x86_64",
    sha256 = "b7117230deceaba8bcd1341f0528df5855e54997cea04379fd3cc2c7c1e07ba8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gzip-1.13-3.el10.x86_64.rpm",
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
    name = "hexedit-0__1.6-8.el10.s390x",
    sha256 = "3b3fa64ec84f359ff667cf1c9f0c66e5300d08284d6a944e295fefbd3fd1e720",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/hexedit-1.6-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "hexedit-0__1.6-8.el10.x86_64",
    sha256 = "b4e61671ac71d0dc721f67b6a7d5ff28e3ec9c8e1b8251104bcd34d6f0611ce3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/hexedit-1.6-8.el10.x86_64.rpm",
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
    name = "hivex-libs-0__1.3.24-2.el10.s390x",
    sha256 = "967989dfab46ed23e33b59c956d27ad881051582f1c59995824b93249c5ff004",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/hivex-libs-1.3.24-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "hivex-libs-0__1.3.24-2.el10.x86_64",
    sha256 = "ecf17b83680af8d8a3cef0a632ec3a7163d01c3dddd2364c9b47a4ba79e23150",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/hivex-libs-1.3.24-2.el10.x86_64.rpm",
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
    name = "hwdata-0__0.379-10.6.el10.s390x",
    sha256 = "99183d83a278795a010aabf072072e4734ebfd27f33af0587b707e07017c54d4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/hwdata-0.379-10.6.el10.noarch.rpm",
    ],
)

rpm(
    name = "hwdata-0__0.379-10.6.el10.x86_64",
    sha256 = "99183d83a278795a010aabf072072e4734ebfd27f33af0587b707e07017c54d4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/hwdata-0.379-10.6.el10.noarch.rpm",
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
    name = "iproute-0__6.17.0-1.el10.aarch64",
    sha256 = "44ddded795dfa336e7c553ee68f70d2ccfbe5f954849cfba4975d1a914565398",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iproute-6.17.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "iproute-0__6.17.0-1.el10.s390x",
    sha256 = "902537a96edca3984a114f83b7a927b83316104ae3ba34c9606b4a964dea7aba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iproute-6.17.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "iproute-0__6.17.0-1.el10.x86_64",
    sha256 = "6ebcdb339cc28036f2dc26b8be2f38d628b7206c715c484dca6031631c304e88",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iproute-6.17.0-1.el10.x86_64.rpm",
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
    name = "iproute-tc-0__6.17.0-1.el10.aarch64",
    sha256 = "46b8ce7f1acdae7878bf667f40c63c3497576749ced449f51ba5b951ca9c39c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iproute-tc-6.17.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "iproute-tc-0__6.17.0-1.el10.s390x",
    sha256 = "50a10556d2351456e5e4cb6b5a17b608f731352c06046e4e5ce2bd7a6a387490",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iproute-tc-6.17.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "iproute-tc-0__6.17.0-1.el10.x86_64",
    sha256 = "1956f5939d423ba743e5d30ca9310ed7e38c5b6cd671c966d0ed093edfa29ba9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.17.0-1.el10.x86_64.rpm",
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
    name = "iptables-libs-0__1.8.11-12.el10.aarch64",
    sha256 = "4bf894764ed0f9e7e92228587b8ec02962197b6ff87db3c0562081daf54efb40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.11-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.11-12.el10.s390x",
    sha256 = "c94759e8d3245cfbe43eb96964ee9a4031585e6874535d564a00596cd56a4d76",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iptables-libs-1.8.11-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.11-12.el10.x86_64",
    sha256 = "450dfc1d463564d4955c3a244cf190bd4544cf56288b9703e2d39af238494f6c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.11-12.el10.x86_64.rpm",
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
    name = "iputils-0__20240905-5.el10.aarch64",
    sha256 = "3ed67cca3fbb5f60f14f85ea712b1822f2a80c58287e744795ef995ebebc3761",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iputils-20240905-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "iputils-0__20240905-5.el10.s390x",
    sha256 = "bf09f778f68c47515f0763e7c4aa952ed32dea57608a9473cb8edd50742e8a6a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iputils-20240905-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "iputils-0__20240905-5.el10.x86_64",
    sha256 = "adfa1b26bf1cd23d0998c85da06ad787f0fa745bfd232f7acec225c1e88b05d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iputils-20240905-5.el10.x86_64.rpm",
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
    name = "ipxe-roms-qemu-0__20240119-5.gitde8a0821.el10.x86_64",
    sha256 = "0b834df444ffe592d164f1dd5a2ce690417e459b8cd6d6c69b2075bbb9c8b4cb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/ipxe-roms-qemu-20240119-5.gitde8a0821.el10.noarch.rpm",
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
    name = "jansson-0__2.14-3.el10.aarch64",
    sha256 = "a838d217420f9f10eb80a221b6cda50ff65e729c15be94f33cbb420f206ee880",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/jansson-2.14-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "jansson-0__2.14-3.el10.s390x",
    sha256 = "cc054f4efd4b779ec708061759de28acd9eb9df0ad8f3b32f9fe4752b1dcb06c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/jansson-2.14-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "jansson-0__2.14-3.el10.x86_64",
    sha256 = "25d2ef852d5941b27ae105ec780aa367605a6f8b86e6c6a13abdee1c1065979f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/jansson-2.14-3.el10.x86_64.rpm",
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
    name = "json-c-0__0.18-3.el10.aarch64",
    sha256 = "d3ecfebff7515c94e971c9584b0815202712cc2642526ee4fe5e424ec8ff2fae",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/json-c-0.18-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "json-c-0__0.18-3.el10.s390x",
    sha256 = "d4bc7597af6496e70ffa04858c8d2418267b302959db9da3f89c6edfc723ccd4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/json-c-0.18-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "json-c-0__0.18-3.el10.x86_64",
    sha256 = "e73ae01d509fb9bef1bbd675be1c0003b0ee942a4187e9b14ef43e56e508245b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/json-c-0.18-3.el10.x86_64.rpm",
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
    name = "json-glib-0__1.8.0-5.el10.aarch64",
    sha256 = "41de435cef6d704c1bb85066b9711e44f60b1dbff3574997094e5ed166e2b95e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/json-glib-1.8.0-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "json-glib-0__1.8.0-5.el10.s390x",
    sha256 = "851e663120bae993deed48bd36f06a84d9082b51b35c87244e7a8d3735bf422f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/json-glib-1.8.0-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "json-glib-0__1.8.0-5.el10.x86_64",
    sha256 = "156fddb0053ab256ec6ecbe7818c0ec8e957228eb2ed1d7cd244ecda85e1197e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/json-glib-1.8.0-5.el10.x86_64.rpm",
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
    name = "kernel-headers-0__6.12.0-192.el10.aarch64",
    sha256 = "f69696aec37a229e1e427fad77d099027f44c8719c8c173b5ad99f7dc0cb7299",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/kernel-headers-6.12.0-192.el10.aarch64.rpm",
    ],
)

rpm(
    name = "kernel-headers-0__6.12.0-192.el10.s390x",
    sha256 = "155a3424d5df56088745abbfbecea8eccd2cd4ed99e2c3ce821bbb6eecfda661",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/kernel-headers-6.12.0-192.el10.s390x.rpm",
    ],
)

rpm(
    name = "kernel-headers-0__6.12.0-192.el10.x86_64",
    sha256 = "427895c58448d116f16c47b287fe5bd0de5984763390e7776bbd89431a1da1b2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/kernel-headers-6.12.0-192.el10.x86_64.rpm",
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
    name = "keyutils-libs-0__1.6.3-5.el10.aarch64",
    sha256 = "a6ff394736256d5c2317ab5503a056d0f60155a92090853179506358bfd2333f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/keyutils-libs-1.6.3-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-5.el10.s390x",
    sha256 = "f2ed690c8ec6ef2a0b912ba324268d354d282f437b30044a44304543e78d9238",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/keyutils-libs-1.6.3-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-5.el10.x86_64",
    sha256 = "312e0bf42841bb330f7721012d1ee5816e5ea223e54fc5dfd1a95c6f1d7516b0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/keyutils-libs-1.6.3-5.el10.x86_64.rpm",
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
    name = "kmod-0__31-12.el10.aarch64",
    sha256 = "2aa38be351be2c1b4efd0932928ebdc74217d23851f6b2aa0b86ce4fb3df2c12",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/kmod-31-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "kmod-0__31-12.el10.s390x",
    sha256 = "0273efd3eedba40d92e26ab6e72ff0dd27eba674c0f22ecf2e37a9d2dca1ab14",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/kmod-31-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "kmod-0__31-12.el10.x86_64",
    sha256 = "9c08b22962d94f0d96cf156ec6e8624a537ab3821b52b4c0f725a7c059380d4e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/kmod-31-12.el10.x86_64.rpm",
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
    name = "kmod-libs-0__31-12.el10.aarch64",
    sha256 = "3776b62bbebcd5862814723a738123b8903f2933c899e6221711e968600fc8f8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/kmod-libs-31-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "kmod-libs-0__31-12.el10.s390x",
    sha256 = "598b1bcbdd2a5806066fbd19eefc62efe53051acd09b530b5913f5dd2f151dcc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/kmod-libs-31-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "kmod-libs-0__31-12.el10.x86_64",
    sha256 = "c1f5cd1f7bc9148f88ddac82d2c70b01a6ba871b7ef5cf81825d96c9ea2360de",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/kmod-libs-31-12.el10.x86_64.rpm",
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
    name = "krb5-libs-0__1.21.3-8.el10.aarch64",
    sha256 = "ae0332e7dc9a151a1f86f44e8cd75148f8ced6aeb54d1d9671af752fd863b0c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.3-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.3-8.el10.s390x",
    sha256 = "415688256b2cea553669441e078c28bcb9dc2227c4bbbd29c46c505cec994d6e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/krb5-libs-1.21.3-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.3-8.el10.x86_64",
    sha256 = "c19429221a54c4de8c9d88d6c6e0f929d1c4199828300c6f94b667c6f7ff00f4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.3-8.el10.x86_64.rpm",
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
    name = "less-0__661-3.el10.s390x",
    sha256 = "6eb8705527ce26aa64dd692e6103e6b2197fb134a9e4f03e5db78d4c45035ddf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/less-661-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "less-0__661-3.el10.x86_64",
    sha256 = "1cf4afdf660772f65668cc702722facd7ed79d849150e14cab623cffd4167516",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/less-661-3.el10.x86_64.rpm",
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
    name = "libacl-0__2.3.2-4.el10.aarch64",
    sha256 = "20f3eeb53bf86dd2c7152fcdc33df3efd60777edd11f31f633739c0fdc0bdbf5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libacl-2.3.2-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libacl-0__2.3.2-4.el10.s390x",
    sha256 = "5f26a314b6e88e87516979610e9c0bda3dc55c67c9339bf79739574162eb1fa6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libacl-2.3.2-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "libacl-0__2.3.2-4.el10.x86_64",
    sha256 = "dd06cfe883fcdf7cb14b749180abfd9fe9924723341a8644a9f65c086febc647",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libacl-2.3.2-4.el10.x86_64.rpm",
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
    name = "libaio-0__0.3.111-22.el10.aarch64",
    sha256 = "99660f7b25fdb5503e0414e263ad91d0c1b61f1dc4e106721c0d1380b239d17f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libaio-0.3.111-22.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libaio-0__0.3.111-22.el10.s390x",
    sha256 = "5ee6ef6f4625016ae0746586e62fcf1c70596f8b557d8e6ad54cc67a4ae26690",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libaio-0.3.111-22.el10.s390x.rpm",
    ],
)

rpm(
    name = "libaio-0__0.3.111-22.el10.x86_64",
    sha256 = "ea807b22c77a37a766e62ad533dc3f9b80fd5b260016487cecea55b095092446",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libaio-0.3.111-22.el10.x86_64.rpm",
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
    name = "libarchive-0__3.7.7-4.el10.aarch64",
    sha256 = "c1a8850b22bb37325ee675db6c312615a4f3944777cd0cc3f24f72b13abc1ecb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libarchive-3.7.7-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libarchive-0__3.7.7-4.el10.s390x",
    sha256 = "6d1dedaf9597a0df9e485662dda1f971639912c40af3c7fbc1150dab8622900a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libarchive-3.7.7-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "libarchive-0__3.7.7-4.el10.x86_64",
    sha256 = "604bd62429638f12bd4699692e75f1bc0c2be2558c3e8abdf85d03974c443194",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libarchive-3.7.7-4.el10.x86_64.rpm",
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
    name = "libasan-0__14.3.1-4.3.el10.aarch64",
    sha256 = "a04297d8b681a9237a1034600a62e5fda0c394c496c2b3d88cdb0838a5788126",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libasan-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libasan-0__14.3.1-4.3.el10.s390x",
    sha256 = "3384d53721a49340c07e02ca23a5bf138830c8c2bf1c6f8d144272abaf88c526",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libasan-14.3.1-4.3.el10.s390x.rpm",
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
    name = "libassuan-0__2.5.6-6.el10.s390x",
    sha256 = "d31b659dd6036b990ea71c10c426df13f2f395685ea8674fcca49d6a5fbeb580",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libassuan-2.5.6-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libassuan-0__2.5.6-6.el10.x86_64",
    sha256 = "5cb1eff4efadf906bb8060bb41c205bf77eabe73142599053ae892ee7d85e9c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libassuan-2.5.6-6.el10.x86_64.rpm",
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
    name = "libatomic-0__14.3.1-4.3.el10.aarch64",
    sha256 = "d069d8b89a5cf93b1ee185f3fb76a109084db9538b7fd0a90eaa5414699f43f6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libatomic-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libatomic-0__14.3.1-4.3.el10.s390x",
    sha256 = "0596f69781d1d98bd986e6ed0f34ce2164789694fc590613817a83bf435e5b58",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libatomic-14.3.1-4.3.el10.s390x.rpm",
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
    name = "libattr-0__2.5.2-5.el10.aarch64",
    sha256 = "37a06ff130ff4112ca431839607e4d7c583ec4b0191431aa9913bba754880040",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libattr-2.5.2-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libattr-0__2.5.2-5.el10.s390x",
    sha256 = "583ef53e42b6928c6a707baee521a3161f0d00d094db96ce05b39ae4409c73f8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libattr-2.5.2-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "libattr-0__2.5.2-5.el10.x86_64",
    sha256 = "2ec3c5ba70aaae97db5226f07476c3fd0adfeed15d7cce3b676288273c829274",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libattr-2.5.2-5.el10.x86_64.rpm",
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
    name = "libblkid-0__2.40.2-15.el10.aarch64",
    sha256 = "255304ac6a0462e6cd059d128680f6c15dcd88cfa95cc7272f7387b17624a0c2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libblkid-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libblkid-0__2.40.2-15.el10.s390x",
    sha256 = "1cf3f37342b1b0cc20dace8837592749bc5cad8baaceb7f2b6480fb1ede7d895",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libblkid-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "libblkid-0__2.40.2-15.el10.x86_64",
    sha256 = "00cec7dfaf08b5ab015ab88bf41f263bab25d416993f271da6990f998eb7569c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libblkid-2.40.2-15.el10.x86_64.rpm",
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
    name = "libbpf-2__1.7.0-1.el10.aarch64",
    sha256 = "f89a67afcbc8eacc5c8c40e7c30ec5a5aaa78e89bca1dd1032b89c8634bbc605",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libbpf-1.7.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libbpf-2__1.7.0-1.el10.s390x",
    sha256 = "0d45b7988fb7d679b8f4f9e88439b086c0c086a1ba7853bfd376b5061ac3c12b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libbpf-1.7.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libbpf-2__1.7.0-1.el10.x86_64",
    sha256 = "1379b88512429975bbbdd65c8737cbc793664bef2f0e8c2e04c1481b939c85ca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libbpf-1.7.0-1.el10.x86_64.rpm",
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
    name = "libbrotli-0__1.1.0-7.el10.s390x",
    sha256 = "6deea1eafedaa040d5c7c4af870f2ae2edb6742f02cd56cdf07789cf2acd1359",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libbrotli-1.1.0-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "libbrotli-0__1.1.0-7.el10.x86_64",
    sha256 = "9b397443a3ffe381380af22b55b0cac0f02412b859f92888138fba5e1df5e15d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libbrotli-1.1.0-7.el10.x86_64.rpm",
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
    name = "libburn-0__1.5.6-6.el10.aarch64",
    sha256 = "221ced92933bca63eb94d1ed60699f364e5d0b0b9915a0fba39d8b98d513d887",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libburn-1.5.6-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libburn-0__1.5.6-6.el10.s390x",
    sha256 = "6ac13f60b7e3bee622332411d4ec87c77dbd23f5710edf73209d29bf85904e4d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libburn-1.5.6-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libburn-0__1.5.6-6.el10.x86_64",
    sha256 = "83ba66223a60bf93d13710b632ecc8c057c294a17f36ba31a18e16e7d4b97819",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libburn-1.5.6-6.el10.x86_64.rpm",
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
    name = "libcap-0__2.69-7.el10.aarch64",
    sha256 = "38c8ab1a8883b39bf46006ed39b7834cfa0df2ca0a4825908f7da4ed631c8fc6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcap-2.69-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libcap-0__2.69-7.el10.s390x",
    sha256 = "3f7016928e759a177e4103d6c31dc0833ec43eae5836d871bc5ea540fd3d0c7b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcap-2.69-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "libcap-0__2.69-7.el10.x86_64",
    sha256 = "54c14cb5c8dc3536f43d632d766ed302a8ecad2ad8efd6aa2d079dafc11d1cd9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcap-2.69-7.el10.x86_64.rpm",
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
    name = "libcap-ng-0__0.8.4-6.el10.aarch64",
    sha256 = "993a88b692dbb7a73ec214c464d8c267155c87ebdb18fb3ecb8782a2f777ce31",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcap-ng-0.8.4-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libcap-ng-0__0.8.4-6.el10.s390x",
    sha256 = "8256ca0cc7612b3bc6d2f86039c914f90c349fda27513cb42868242cb4949542",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcap-ng-0.8.4-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libcap-ng-0__0.8.4-6.el10.x86_64",
    sha256 = "38b2ce6018bc0c73cdbc79f5cb2bad63045d84c308a56085a9de4adfd3250add",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcap-ng-0.8.4-6.el10.x86_64.rpm",
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
    name = "libcom_err-0__1.47.1-5.el10.aarch64",
    sha256 = "97380e5fe0fce42be70418333bae2d5d9044c5f7fbda30c9b28f7776718e76a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcom_err-1.47.1-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libcom_err-0__1.47.1-5.el10.s390x",
    sha256 = "09f3032ebafe4e8d93152c940f160b68df98b1a1c978fae86b8524a8d7713467",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcom_err-1.47.1-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "libcom_err-0__1.47.1-5.el10.x86_64",
    sha256 = "37b036aa4cb44adade9c4206f2eea389035387082be0d2eda0249d9fd07fb842",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcom_err-1.47.1-5.el10.x86_64.rpm",
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
    name = "libconfig-0__1.7.3-10.el10.s390x",
    sha256 = "2df21ddec6f917d330835cee60c9c71a604fa3f988aabe5f1deeebd76582dc5a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libconfig-1.7.3-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libconfig-0__1.7.3-10.el10.x86_64",
    sha256 = "5bee52a5f0599fc6a59df28222e6e831c441887471412a3d18e6d13ddfaaa881",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libconfig-1.7.3-10.el10.x86_64.rpm",
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
    name = "libcurl-minimal-0__8.12.1-4.el10.aarch64",
    sha256 = "17a02035eee04463ae727468fb756a7088e83fe3db93138b8a1e5a6d8a7b2904",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-8.12.1-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libcurl-minimal-0__8.12.1-4.el10.s390x",
    sha256 = "4acfbb95ccb0ba29bd9598472942f3df90a6e3bbe7e4ec7893ec92dd97636a66",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcurl-minimal-8.12.1-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "libcurl-minimal-0__8.12.1-4.el10.x86_64",
    sha256 = "053872b16ba35bdac16d7e3b3bed01fde3143face1ac9292dde9ac9b44a96758",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-8.12.1-4.el10.x86_64.rpm",
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
    name = "libeconf-0__0.6.2-4.el10.aarch64",
    sha256 = "1bb73420b4f72fb200ccc560224107e1ae62b8a7156051a88c9239c9def47983",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libeconf-0.6.2-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libeconf-0__0.6.2-4.el10.s390x",
    sha256 = "eb314cc56ffe80a641f97664b4d5a7313e3be7f1bef79e6636bcaab5751b350f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libeconf-0.6.2-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "libeconf-0__0.6.2-4.el10.x86_64",
    sha256 = "1cdb8e5bf4d7680e41ebb2b76da3aab34c1ece4bba2fed952d8f49da69117dfa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libeconf-0.6.2-4.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libevent-0__2.1.12-16.el10.aarch64",
    sha256 = "275530b6896bec203e5cdf0cb427c78da43f5b01d3d26b0a2b239f2ad49fcda2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libevent-2.1.12-16.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libevent-0__2.1.12-16.el10.s390x",
    sha256 = "11e041d07e9f2f30736efa420ed437e11676bc796914f22c7b3fc322f08e997a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libevent-2.1.12-16.el10.s390x.rpm",
    ],
)

rpm(
    name = "libevent-0__2.1.12-16.el10.x86_64",
    sha256 = "f8f5c3946bbd53590978e9aeca3064d81ab580492c4ff8044c48797870276f47",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libevent-2.1.12-16.el10.x86_64.rpm",
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
    name = "libfdisk-0__2.40.2-15.el10.aarch64",
    sha256 = "946bd21e2a3b4dca038342bc02ced6e161415f3654bd117a6725510b5e44741c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libfdisk-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libfdisk-0__2.40.2-15.el10.s390x",
    sha256 = "db4c70cc90d26af06f4253703e436fe232e0ef0c954728eaa3501134f009cfb6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libfdisk-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "libfdisk-0__2.40.2-15.el10.x86_64",
    sha256 = "31aff408f90ee0628690b836477de4e5bf4ea3bc249fc4032085539a81480dbb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libfdisk-2.40.2-15.el10.x86_64.rpm",
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
    name = "libfdt-0__1.7.0-12.el10.aarch64",
    sha256 = "2a568810d2b8fbd8425eeb64738491a514feaec766f1b90d320caf320e543134",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libfdt-1.7.0-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libfdt-0__1.7.0-12.el10.s390x",
    sha256 = "bd49a0dac4411aca4319b6dc670e0fab0fb9672eaedd83aab40452878fbe46ac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libfdt-1.7.0-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "libfdt-0__1.7.0-12.el10.x86_64",
    sha256 = "9c519693ffe97be0dda3e06ea9708446b2692b5fd72b8cfb93f0b78ddc6418d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libfdt-1.7.0-12.el10.x86_64.rpm",
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
    name = "libffi-0__3.4.4-10.el10.aarch64",
    sha256 = "87b620ad4069f0a9623913acc568a2659bcee3695293b275a9f09b809437bf6e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libffi-3.4.4-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libffi-0__3.4.4-10.el10.s390x",
    sha256 = "592be60a3f4ee70236cc254894f587012ce533a6f4fc74031bf6674792782338",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libffi-3.4.4-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libffi-0__3.4.4-10.el10.x86_64",
    sha256 = "72aff2f3b4291f5418491e612be4f92d65a9239224a4906c0c63dbc4fc668e73",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libffi-3.4.4-10.el10.x86_64.rpm",
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
    name = "libgcc-0__14.3.1-4.3.el10.aarch64",
    sha256 = "f86c3466afd9a017bd0c9f7f26120b4ecc3e77eef0c0f6da7843765050012bc5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libgcc-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libgcc-0__14.3.1-4.3.el10.s390x",
    sha256 = "e61bef933045b06a0b851d28c42556fdfedaed37aa704ee7b937ae890fe6627c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgcc-14.3.1-4.3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libgcc-0__14.3.1-4.3.el10.x86_64",
    sha256 = "20a4555ff333952c625e39d5d0384161371a1f8fb037f0f1c8800ed3778abd14",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgcc-14.3.1-4.3.el10.x86_64.rpm",
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
    name = "libgcrypt-0__1.11.0-6.el10.s390x",
    sha256 = "b4383e8187d076c47b732487866eeebc0e40c1474e0dd6d927dfca6172cb274f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgcrypt-1.11.0-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libgcrypt-0__1.11.0-6.el10.x86_64",
    sha256 = "1be7cfbc9f69f9e2b3d3f0621e14ded96e27d1c334decb5c88d1e396edf825e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.11.0-6.el10.x86_64.rpm",
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
    name = "libgomp-0__14.3.1-4.3.el10.aarch64",
    sha256 = "741baaea475d1bbd2078e5366be928caa51f476e4cb2b2ca7d45ea7523ca1d8f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libgomp-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libgomp-0__14.3.1-4.3.el10.s390x",
    sha256 = "9f561bcd63e706eb8b6cc6253223ef66ac2d995d640413b76de2080dad05dddf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgomp-14.3.1-4.3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libgomp-0__14.3.1-4.3.el10.x86_64",
    sha256 = "bfdcba2fd598203e366fb8379c8d76442e9dd5763a4c60ce42af0bf712a6df8c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgomp-14.3.1-4.3.el10.x86_64.rpm",
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
    name = "libgpg-error-0__1.50-2.el10.s390x",
    sha256 = "d2ae277868010dcf3003a59234b70743f2ff0846e0f8aba6cf349de19d6c2173",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgpg-error-1.50-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libgpg-error-0__1.50-2.el10.x86_64",
    sha256 = "b7d74c79f82abf581fdb5b9fbd0b3792640c26780652036be284347b7b339fff",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgpg-error-1.50-2.el10.x86_64.rpm",
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
    name = "libguestfs-1__1.58.1-2.el10.s390x",
    sha256 = "5546a48cbc10fe6dac94e219d9458be67cfb047a60482e4429987461a4f7ee71",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libguestfs-1.58.1-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libguestfs-1__1.58.1-2.el10.x86_64",
    sha256 = "9e258c41aeb346fd4e37d13a3346f89c9a5d1354586da91567a948e520da78bd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libguestfs-1.58.1-2.el10.x86_64.rpm",
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
    name = "libibverbs-0__57.0-2.el10.aarch64",
    sha256 = "c542dc9c95c8a74c8521b64c971fa2a9415ee78becb5ec22dd8fc991be9d36f1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libibverbs-57.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libibverbs-0__57.0-2.el10.s390x",
    sha256 = "ace62b9368b6f207ecc03642baadac205ab4fc4247da37530dbf3e3b77a1216f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libibverbs-57.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libibverbs-0__57.0-2.el10.x86_64",
    sha256 = "370284438b7e09f12e250dfb345c3d6a57263404009ebab56bc8271d0650ff19",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libibverbs-57.0-2.el10.x86_64.rpm",
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
    name = "libidn2-0__2.3.7-3.el10.aarch64",
    sha256 = "947248aeedd08f88d9490f3020dee6416595cf8d25e15738c306d55e9cae8bfc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libidn2-2.3.7-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libidn2-0__2.3.7-3.el10.s390x",
    sha256 = "7d75542211c9b7b8e53aaf6aee0dc430f091414fb2ca755baadc35f844bded75",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libidn2-2.3.7-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libidn2-0__2.3.7-3.el10.x86_64",
    sha256 = "04ae61bbe2cc0db7581d6f96a562b9b87a8a4dba714a0cf2c73bba6306e94c27",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libidn2-2.3.7-3.el10.x86_64.rpm",
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
    name = "libisoburn-0__1.5.6-6.el10.aarch64",
    sha256 = "038aa1a45c117b4c4abbeedc3f67faa01a66570bc554f0e53955ce83a26e7281",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libisoburn-1.5.6-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libisoburn-0__1.5.6-6.el10.s390x",
    sha256 = "52fa5b55814330bf00ce834de3fa86c0dff29691feb242800f04cc13c44f2f3a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libisoburn-1.5.6-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libisoburn-0__1.5.6-6.el10.x86_64",
    sha256 = "6f41c5e8e0d9dedf3c0b07f2ddc0748870cb18ed15a23dc22038a022c1d38e74",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libisoburn-1.5.6-6.el10.x86_64.rpm",
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
    name = "libisofs-0__1.5.6-6.el10.aarch64",
    sha256 = "cacea726d7dd126a364ec2431f49417b884287c84664042d51d59011acac34d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libisofs-1.5.6-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libisofs-0__1.5.6-6.el10.s390x",
    sha256 = "655af71fa634d1bb1c86b6c4810c452c98c1772dc9d55da3e3e9f2413bafc293",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libisofs-1.5.6-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libisofs-0__1.5.6-6.el10.x86_64",
    sha256 = "a61eda57352e86657ca99fc774d0d74c0fbd67dad5bcd02139ff6de466a038ac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libisofs-1.5.6-6.el10.x86_64.rpm",
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
    name = "libksba-0__1.6.7-2.el10.s390x",
    sha256 = "f4a0e294968ce54ad30e4de5baabcdebd7a9db7900266e4feb7cadecd7f18cba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libksba-1.6.7-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libksba-0__1.6.7-2.el10.x86_64",
    sha256 = "2bfa8330ad9c63eaecd2bd1d0989625e812d853a85c505cb759a2d1c06750607",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libksba-1.6.7-2.el10.x86_64.rpm",
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
    name = "libmnl-0__1.0.5-7.el10.aarch64",
    sha256 = "786a9caea9de8f4529e5ec07ac24c9cfccd50ee5f4b045c37dbd4eb074b34f34",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libmnl-1.0.5-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libmnl-0__1.0.5-7.el10.s390x",
    sha256 = "bb6229f5c62e69828f94cd16f0086115e17ea9c9ab831ad14781f9b9807cd3d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libmnl-1.0.5-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "libmnl-0__1.0.5-7.el10.x86_64",
    sha256 = "1e1d36725d958fc3f9016cc85b238bf5462a43ae7be144d0a4a72be0982d68ce",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libmnl-1.0.5-7.el10.x86_64.rpm",
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
    name = "libmount-0__2.40.2-15.el10.aarch64",
    sha256 = "5d5edf37b93295b534e1ce55f4a7370e08f6831b1b9ea3d18085a889ea7dd111",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libmount-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libmount-0__2.40.2-15.el10.s390x",
    sha256 = "652be9673bbc71ea2c71e0c7295dfbba5f39c3e222a0412cdf8b07d66b09ce6d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libmount-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "libmount-0__2.40.2-15.el10.x86_64",
    sha256 = "857a25a9634578ee103810bf684d3ec0c881c258d9e74240ff77962a70a2e6a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libmount-2.40.2-15.el10.x86_64.rpm",
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
    name = "libmpc-0__1.3.1-7.el10.aarch64",
    sha256 = "bb46a7465559a26c085bf1c02f0764332430a6c1b8fb3f08c8cee184e3d1f02a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libmpc-1.3.1-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libmpc-0__1.3.1-7.el10.s390x",
    sha256 = "ad956e3c217ba500101acb4219f4e07390ca5ac8a14f99ca9cca85220b525da1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libmpc-1.3.1-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "libmpc-0__1.3.1-7.el10.x86_64",
    sha256 = "daaa73a35dfe21a8201581e333b79ccd296ae87a93f9796ba522e58edc23777c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libmpc-1.3.1-7.el10.x86_64.rpm",
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
    name = "libnbd-0__1.24.0-1.el10.aarch64",
    sha256 = "393fb9a22ff850b0e2b2523bc7ac553af21c11e8c1c2f9c25ac5230526fdf490",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libnbd-1.24.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libnbd-0__1.24.0-1.el10.s390x",
    sha256 = "77090bbbd3ea707fce4ebbe8949f4bf594608de81675630ee51b7e1199c82cd8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libnbd-1.24.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libnbd-0__1.24.0-1.el10.x86_64",
    sha256 = "0b44c10ebfb4be2c12275c1edf9063ee45624edea98d01c76fc1a0ec61723ffd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libnbd-1.24.0-1.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libnbd-devel-0__1.20.3-4.el9.aarch64",
    sha256 = "cb38d0fc674e15a84caa1606fdb7430ba3c0e61bfc2a7dd3a9719ff82dfa920f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libnbd-devel-1.20.3-4.el9.aarch64.rpm",
    ],
)

rpm(
    name = "libnbd-devel-0__1.20.3-4.el9.s390x",
    sha256 = "3289f3e0c7d6f290c79faca35a80b3b3170941bdd478468d339c692c3e35c882",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libnbd-devel-1.20.3-4.el9.s390x.rpm",
    ],
)

rpm(
    name = "libnbd-devel-0__1.20.3-4.el9.x86_64",
    sha256 = "88dbe3a521391c371075dc20f70de464004ea8d3bca120f5fa8fdd66ef244847",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libnbd-devel-1.20.3-4.el9.x86_64.rpm",
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
    name = "libnetfilter_conntrack-0__1.0.9-12.el10.aarch64",
    sha256 = "53c1b45e66ef040f6486052395cfda198d9b8b3058834ae7e8b7864b04f9c766",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnetfilter_conntrack-1.0.9-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-12.el10.s390x",
    sha256 = "f7f15dc88c380db9cbf6e62c1a04b872867d7f5a4f2cc0b2f9988db963d8401d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnetfilter_conntrack-1.0.9-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-12.el10.x86_64",
    sha256 = "71af0b9fb8b790e3d471a74ef463dc3cbb0267c9bdbaf876160fa9821f63200f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnetfilter_conntrack-1.0.9-12.el10.x86_64.rpm",
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
    name = "libnfnetlink-0__1.0.2-3.el10.aarch64",
    sha256 = "0ac9c6ebd2c5652bea632435fd73bfb36785d2f15eb840ca53049d6bb3abe639",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnfnetlink-1.0.2-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.2-3.el10.s390x",
    sha256 = "433915ab5525daf404ae37d47f1f735d257a33ff8a2565741e3449d5b39b0cab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnfnetlink-1.0.2-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.2-3.el10.x86_64",
    sha256 = "2988f90762058160e4071b79b96523901a2170bc6242488878ce51fbc0d871ca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.2-3.el10.x86_64.rpm",
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
    name = "libnftnl-0__1.3.0-2.el10.aarch64",
    sha256 = "c420a00ab70913201a953ed39ea017980c0b9a92132b93700a53bfe7c8d04bfe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnftnl-1.3.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libnftnl-0__1.3.0-2.el10.s390x",
    sha256 = "f2a520590d440b4aa79cbf03b42afa60b7918751917df8b9f55cd77924637f86",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnftnl-1.3.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libnftnl-0__1.3.0-2.el10.x86_64",
    sha256 = "3733a952b42041ea2f83eb8ce39000ab4ea713008a9ac1a22e2f13ef10eab93e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnftnl-1.3.0-2.el10.x86_64.rpm",
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
    name = "libnghttp2-0__1.64.0-2.el10.aarch64",
    sha256 = "3bb538521491eaf5ee76c9bce5e0668345f6c03c5eb8610375ea62fc13252cf5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.64.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libnghttp2-0__1.64.0-2.el10.s390x",
    sha256 = "2731569708badb4582ed74326175366aca1e1a86a90fcec460acaefdb6c4543f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnghttp2-1.64.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libnghttp2-0__1.64.0-2.el10.x86_64",
    sha256 = "087a6ea4e234b3a6b12326f5da756c8010efdda58a7a8ebcbf4f4da32247a566",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.64.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el10.aarch64",
    sha256 = "b27497d441cd6ae6fbf6a077913eff334c64ace7b6030d6c11a221316a8c8d92",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnl3-3.11.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el10.s390x",
    sha256 = "4171ef398ec29504e828c461581fa44e6afe6b36ff84da66f902edd932271b34",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnl3-3.11.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el10.x86_64",
    sha256 = "886324f9d4b8c95a46d5c77c0f5cd90051c83462e78ca752c70cc709f5a01d90",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnl3-3.11.0-1.el10.x86_64.rpm",
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
    name = "libosinfo-0__1.11.0-8.el10.s390x",
    sha256 = "3ae1d93313b6b63d6cf2887307a539e9a371d64134ad7a363d5b31c64ee2734d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libosinfo-1.11.0-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "libosinfo-0__1.11.0-8.el10.x86_64",
    sha256 = "e632610473056869e88ddaa33511e043721d89f8c2216590a274e638f64e98fa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libosinfo-1.11.0-8.el10.x86_64.rpm",
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
    name = "libpcap-14__1.10.4-7.el10.aarch64",
    sha256 = "f15a71822ba8269643911bdd5455e2c24c2489927217d3512c9453a6ff8af5bf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libpcap-1.10.4-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libpcap-14__1.10.4-7.el10.s390x",
    sha256 = "98f721f0b8b731a77d27bcb3178c7297e84372095a452ab52a98192c76953d3f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpcap-1.10.4-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "libpcap-14__1.10.4-7.el10.x86_64",
    sha256 = "a933eb7fba1535c9df52f7e44504535b25f8b6fb79c5cf68a0b6e80eb4b9dbf8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpcap-1.10.4-7.el10.x86_64.rpm",
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
    name = "libpkgconf-0__2.1.0-3.el10.aarch64",
    sha256 = "fc2db71a801f4cd03425463d0aea745da36837f25d8cc2042eb747c8a336f989",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libpkgconf-2.1.0-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libpkgconf-0__2.1.0-3.el10.s390x",
    sha256 = "7d503fcdd8154231531ec1e076ac2552b9d0a5fe096fb50d3a9ff0ebce07d92d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpkgconf-2.1.0-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libpkgconf-0__2.1.0-3.el10.x86_64",
    sha256 = "813f59114413d5e14fc566262ee3d4b56b621beacbe40eda6f28d31f464de1a6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpkgconf-2.1.0-3.el10.x86_64.rpm",
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
    name = "libpng-2__1.6.40-8.el10.aarch64",
    sha256 = "449443264f27154b3453af2deb2bb91ab184994b8e2cc94f45d93aa56381c081",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libpng-1.6.40-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libpng-2__1.6.40-8.el10.s390x",
    sha256 = "434d4e41e3409a7907219176b8b366f3f9c9c761a152af011a4d794c4972e1ab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libpng-1.6.40-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "libpng-2__1.6.40-8.el10.x86_64",
    sha256 = "215a0ac1a843c31f8d77b77c03719ecf96d5033ec83e95000c8f5f669bfbca95",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpng-1.6.40-8.el10.x86_64.rpm",
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
    name = "libpsl-0__0.21.5-6.el10.s390x",
    sha256 = "ef89d923a60bc9658e5524a80960a865d805aa136b7dd3761a162d58b2aff46d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpsl-0.21.5-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "libpsl-0__0.21.5-6.el10.x86_64",
    sha256 = "1dca94a85aabd9730bc731fa8a6abb138fec28b75c6a39694d862135c2ade0f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpsl-0.21.5-6.el10.x86_64.rpm",
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
    name = "libpwquality-0__1.4.5-12.el10.aarch64",
    sha256 = "0d0d6a0e741f94889796b551935f72cf551587067f0c9b64531b5c34b03ab1d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libpwquality-1.4.5-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libpwquality-0__1.4.5-12.el10.s390x",
    sha256 = "bff94322487bd0bd36640c27e56d7b0167187772cab630758bc56aada0038aea",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpwquality-1.4.5-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "libpwquality-0__1.4.5-12.el10.x86_64",
    sha256 = "eda9e6acc99c2c9fa058a9db428da1b0c7441f2be174b9aa7f1628359e36e6ab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.5-12.el10.x86_64.rpm",
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
    name = "libseccomp-0__2.5.6-1.el10.aarch64",
    sha256 = "322aa4ea140a63645c7f086b58a08346617eea2efee8044287b76373d633b65f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libseccomp-2.5.6-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el10.s390x",
    sha256 = "e652de14f8c0d52480c2bba779daf5da7b7fd66e65090e1781f41d3bee250840",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libseccomp-2.5.6-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el10.x86_64",
    sha256 = "654051862cc301ed43501ab36b687ed5adeb3ca57689f54a80bf760ad9686e54",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libseccomp-2.5.6-1.el10.x86_64.rpm",
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
    name = "libselinux-0__3.9-3.el10.aarch64",
    sha256 = "9df151baa8c60a2bc4998a636b7f50b40bf86d64420578de7501a9daf1d25a31",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libselinux-3.9-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libselinux-0__3.9-3.el10.s390x",
    sha256 = "9b95b6d120c78041499f330aeb8092bea16fd01536f518445618034599741182",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libselinux-3.9-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libselinux-0__3.9-3.el10.x86_64",
    sha256 = "9030f7855d93e37b4d2d9e1d7a05522e059f6f90f328ad9433125eedd21f35fa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libselinux-3.9-3.el10.x86_64.rpm",
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
    name = "libselinux-utils-0__3.9-3.el10.aarch64",
    sha256 = "971a828ba86b404b4d689dcae7725c62a08bfc7f4c460e5e9a80ff2c3428ea78",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.9-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libselinux-utils-0__3.9-3.el10.s390x",
    sha256 = "b61a38bf43fe870be78dabd06af7b99f68f450a14fbdc7590ecfcd706d435617",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libselinux-utils-3.9-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libselinux-utils-0__3.9-3.el10.x86_64",
    sha256 = "794445213c267c95ed897828e76671463e6dd8fe2b4003e6ad5f4ef64437a5e6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.9-3.el10.x86_64.rpm",
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
    name = "libsemanage-0__3.9-2.el10.aarch64",
    sha256 = "cc0aab6cace2891a9991af486461d5507c26b352e7f335e5558404aa3a2d23ae",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsemanage-3.9-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libsemanage-0__3.9-2.el10.s390x",
    sha256 = "773fce11f06bb13d5e7c821ef80d478e07ef277ab08272e0498b05a3f380c53c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsemanage-3.9-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libsemanage-0__3.9-2.el10.x86_64",
    sha256 = "20d1fd8d8e72ab3efd493fd6c24cf134bdcb351d9f235a6a479f10097bc6f516",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsemanage-3.9-2.el10.x86_64.rpm",
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
    name = "libsepol-0__3.9-1.el10.aarch64",
    sha256 = "7aadc40ad02a2d595d01e6322a005bd68278b905b95aa8b517cff89a8a3cbf21",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsepol-3.9-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libsepol-0__3.9-1.el10.s390x",
    sha256 = "436ef10d6dc7d82c06620e43ae78edb153a1d9bd16da55786be538a0560669b5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsepol-3.9-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libsepol-0__3.9-1.el10.x86_64",
    sha256 = "3bd100da5da32dc933544f2206043a0fc34c94a5d249a3129dc9305e1844df0c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsepol-3.9-1.el10.x86_64.rpm",
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
    name = "libslirp-0__4.7.0-10.el10.aarch64",
    sha256 = "077e56fc67d139c2569bdd6b920777df742773a155425b11401406aac39f4e7b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libslirp-4.7.0-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libslirp-0__4.7.0-10.el10.s390x",
    sha256 = "392067b3525f2d603a121d6a2b7e5683c7a903a7be677ffc94fe2f1b278d3a11",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libslirp-4.7.0-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libslirp-0__4.7.0-10.el10.x86_64",
    sha256 = "bc98bf4c15d226b809474c1237700e4e3158d77c4b7488611599672ac0b570af",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libslirp-4.7.0-10.el10.x86_64.rpm",
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
    name = "libsmartcols-0__2.40.2-15.el10.aarch64",
    sha256 = "803ad2645c105834a9ce3b89cd74653ba9278ac28cf25e3a6fd0bb60d08cdd11",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libsmartcols-0__2.40.2-15.el10.s390x",
    sha256 = "ea2a641dc725f5fa0ac7d5a1fc5f564f41e7f806214941175e7ceec14f36bb8a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsmartcols-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "libsmartcols-0__2.40.2-15.el10.x86_64",
    sha256 = "18ae7ebcafe3fa1f0d7b1bc9290a8cf6e087da36a82c951031d00e0457d46859",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.40.2-15.el10.x86_64.rpm",
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
    name = "libsoup3-0__3.6.5-5.el10.s390x",
    sha256 = "eb1082bb8403619c3ce9352feab119dd27eaf3e10417d7968c0342c0269d6272",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libsoup3-3.6.5-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "libsoup3-0__3.6.5-5.el10.x86_64",
    sha256 = "38b4b20b159ae75afc780d10b1f212a2337a750e42c9183b071f5164a6cdfeac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libsoup3-3.6.5-5.el10.x86_64.rpm",
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
    name = "libss-0__1.47.1-5.el10.aarch64",
    sha256 = "edb0a7af06913af8ce5a72ab62de780b8b08ad7a7db6cabc4bae9b95d4253607",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libss-1.47.1-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libss-0__1.47.1-5.el10.s390x",
    sha256 = "1a38ebb62511e39bae10e1cbde35152562e90d8014f243696bed5c3deef6bf9f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libss-1.47.1-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "libss-0__1.47.1-5.el10.x86_64",
    sha256 = "2dea843b06f0bd161807d8cfd7e3ef05f5944f5ea57332e5b58700bd93262765",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libss-1.47.1-5.el10.x86_64.rpm",
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
    name = "libssh-0__0.11.1-3.el10.aarch64",
    sha256 = "95144fadc027f326413a7960e36ca57fd72c94eb76692632a3cb1184eea19e81",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libssh-0.11.1-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libssh-0__0.11.1-3.el10.s390x",
    sha256 = "7fc02fabb13f1c1fe5322e99bf27f9b683361b1159de6f83cec27662524e51ab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libssh-0.11.1-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libssh-0__0.11.1-3.el10.x86_64",
    sha256 = "ef979981656e3422ce8ab958097db269e235f5f17eb9d58aeb5c9c41a6844d97",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libssh-0.11.1-3.el10.x86_64.rpm",
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
    name = "libssh-config-0__0.11.1-3.el10.aarch64",
    sha256 = "a894c61e02dfb1d9630deb0cada5c1508bdfe63ea0bd906fab9c8bb0f5a0418d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libssh-config-0.11.1-3.el10.noarch.rpm",
    ],
)

rpm(
    name = "libssh-config-0__0.11.1-3.el10.s390x",
    sha256 = "a894c61e02dfb1d9630deb0cada5c1508bdfe63ea0bd906fab9c8bb0f5a0418d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libssh-config-0.11.1-3.el10.noarch.rpm",
    ],
)

rpm(
    name = "libssh-config-0__0.11.1-3.el10.x86_64",
    sha256 = "a894c61e02dfb1d9630deb0cada5c1508bdfe63ea0bd906fab9c8bb0f5a0418d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libssh-config-0.11.1-3.el10.noarch.rpm",
    ],
)

rpm(
    name = "libsss_idmap-0__2.12.0-1.el10.aarch64",
    sha256 = "5d56064a3fe65a8eb6b9e02b239587bc9a3e132a947d836c5edf4f02d538fd03",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.12.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libsss_idmap-0__2.12.0-1.el10.s390x",
    sha256 = "573803676a86b4f7d8ed234ce70c8a40412249a4bdbd8256c4fac09fb52eb2c9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.12.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libsss_idmap-0__2.12.0-1.el10.x86_64",
    sha256 = "fbfda0b5f6b73eb4b75196370ea91361afb736e08212f291f6c5328ecab401de",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.12.0-1.el10.x86_64.rpm",
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
    name = "libsss_nss_idmap-0__2.12.0-1.el10.aarch64",
    sha256 = "fff9355f09c72d9e9f6eddf6a9b5ae5d91d03d34c9e09acd4790af60ef7e2fb5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.12.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.12.0-1.el10.s390x",
    sha256 = "cd8c9d8edca4e2658839230d7afd66c8e5c52c782169eb1cd1354b8683f9d606",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.12.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.12.0-1.el10.x86_64",
    sha256 = "690e064654f7f3ede0eed89565408970258329a37580c61ee04872a64c489b32",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.12.0-1.el10.x86_64.rpm",
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
    name = "libstdc__plus____plus__-0__14.3.1-4.3.el10.aarch64",
    sha256 = "150b07a914aaf05801ebb3a9cb539c8291003738a5b0a86cb44ad07a8a99032c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libstdc++-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__14.3.1-4.3.el10.s390x",
    sha256 = "5d0f9667179fce3828c81c8fe2e2806de7841222d25251dd4371acb72966ec77",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libstdc++-14.3.1-4.3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__14.3.1-4.3.el10.x86_64",
    sha256 = "c58029947e33553509b21496b23445b7c85c9ca2dd3013bd7f2c8a25a32de12b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libstdc++-14.3.1-4.3.el10.x86_64.rpm",
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
    name = "libtasn1-0__4.20.0-1.el10.aarch64",
    sha256 = "f46e93f5bff81ef89c872c2ad91ddde57c9ee0025d162647618ba5e764520854",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libtasn1-4.20.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libtasn1-0__4.20.0-1.el10.s390x",
    sha256 = "1c866239a4d6d0198fb9916c5ae132f19ccc576cb389101270291e7c1b5e3f1d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libtasn1-4.20.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libtasn1-0__4.20.0-1.el10.x86_64",
    sha256 = "6f88995a1e9181e8d99b77b8cc60681f79ba424382a17e590c3d813f300adf65",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libtasn1-4.20.0-1.el10.x86_64.rpm",
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
    name = "libtirpc-0__1.3.5-1.el10.aarch64",
    sha256 = "6e0345c38ef8c15d2f1743892063241f0273a6b14c1844ab127ad0c085b510e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libtirpc-1.3.5-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libtirpc-0__1.3.5-1.el10.s390x",
    sha256 = "5f6160af1ea75ef4df15281225e50e13d7de69aad58a7bc08d558ffb88c86086",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libtirpc-1.3.5-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "libtirpc-0__1.3.5-1.el10.x86_64",
    sha256 = "8692d388ed8b7fa6ffe56c9403576ea7b49d55c305e9a64dd44a59fb592fe295",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libtirpc-1.3.5-1.el10.x86_64.rpm",
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
    name = "libtpms-0__0.9.6-11.el10.aarch64",
    sha256 = "3c666376aabf7fa14a76232e8709a390587bcebfebb24897de6c7c693703fed0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libtpms-0.9.6-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libtpms-0__0.9.6-11.el10.s390x",
    sha256 = "55e810e2e6a3c8b166c1fa48a38e2430a2f3b1587014009f48b88fc17ba2f5c4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libtpms-0.9.6-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "libtpms-0__0.9.6-11.el10.x86_64",
    sha256 = "595a554e74b9e9515d2d615b05ed13118f4a9e0c21f1afc74bf5b4e677b56c9a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libtpms-0.9.6-11.el10.x86_64.rpm",
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
    name = "libubsan-0__14.3.1-4.3.el10.aarch64",
    sha256 = "36fb1bf1480e59bfe4a5ed8aedf5c10323bc8c0410b7ffd36721ec6ebafbebb7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libubsan-14.3.1-4.3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libubsan-0__14.3.1-4.3.el10.s390x",
    sha256 = "f44ed900f63cc5709b5798190a644045def5d44464db671176005ebcd6618372",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libubsan-14.3.1-4.3.el10.s390x.rpm",
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
    name = "libunistring-0__1.1-10.el10.aarch64",
    sha256 = "aa793b61f51cb8727c37520bc4b261845831b9a5789649a798c4e8a2cc207f4f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libunistring-1.1-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libunistring-0__1.1-10.el10.s390x",
    sha256 = "9c45ddec6ffa51201a570b9881e9277b8f22c8eb40ee62b45d3c2b86bdc8eeac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libunistring-1.1-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libunistring-0__1.1-10.el10.x86_64",
    sha256 = "603c06593a43f5766a53588d9ba18855ddb7b238963b8e09d8a328a17959b774",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libunistring-1.1-10.el10.x86_64.rpm",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el10.aarch64",
    sha256 = "29f16a2950ef7ddaae31d7806d98961bf1f7d1772623782ae45cc687a3980c62",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/liburing-2.12-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el10.s390x",
    sha256 = "53f3015878c4044a13caf8060a4afa6c10aff93452aa4b8de84cd2374d456a51",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/liburing-2.12-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el10.x86_64",
    sha256 = "133309fc854ab7859713d7944e5a14e8cbc3f3916bbcd9f9e6af4d4850424c15",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/liburing-2.12-1.el10.x86_64.rpm",
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
    name = "libusb1-0__1.0.29-3.el10.aarch64",
    sha256 = "e0d9019535c50ac90e39e158bcf1b4ef7796d30aff42be9183e7177165b40f90",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libusb1-1.0.29-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libusb1-0__1.0.29-3.el10.s390x",
    sha256 = "9071987b92f299616adeffc5e6b6e0fe4ec0cd1d7cd293cedeb5e06f17ee6a9a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libusb1-1.0.29-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "libusb1-0__1.0.29-3.el10.x86_64",
    sha256 = "60b40b436504fe0b046ea990512936c916c7500f08aaf82e8ea886cb06ca5f53",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libusb1-1.0.29-3.el10.x86_64.rpm",
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
    name = "libutempter-0__1.2.1-15.el10.aarch64",
    sha256 = "6444bf715fdd137bd1bd096d9903e29516c609d41113139756df2e9316825d6a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libutempter-1.2.1-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-15.el10.s390x",
    sha256 = "1314f6b74597ad5a5a85b51f5243f3802d2f722a5ed5a41a3ebca827eb7d6a6f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libutempter-1.2.1-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-15.el10.x86_64",
    sha256 = "db498c4b6ce6f223597f8ea955fe4e286f4fc5838e81579de877ca0e80c2d6eb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libutempter-1.2.1-15.el10.x86_64.rpm",
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
    name = "libuuid-0__2.40.2-15.el10.aarch64",
    sha256 = "f165963e4e47c6a79848df3a775c10607083ca1664f86e3c554e186c7c4b5d3b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libuuid-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libuuid-0__2.40.2-15.el10.s390x",
    sha256 = "37093b4261f004e73d0edd0e2cf36fe68bf947bd6984a8c2a73508ab92169749",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libuuid-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "libuuid-0__2.40.2-15.el10.x86_64",
    sha256 = "df0b13144e4bc5c7b7607d66ac067d411d2f6ef362c9541d20bbda08ba65e371",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libuuid-2.40.2-15.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libverto-0__0.3.2-10.el10.aarch64",
    sha256 = "0583db7823a8f33a1e09db1e4aa389c10bc98b58de3bd985b6f67be5351d814a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libverto-0.3.2-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libverto-0__0.3.2-10.el10.s390x",
    sha256 = "80757eae2999d4dbc8975747eb4d8fdfb64b144826ba58215672a0f34d313228",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libverto-0.3.2-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libverto-0__0.3.2-10.el10.x86_64",
    sha256 = "52777e532dc2351c83b72b5033c40df20494afb6504100f7413a65f74368c284",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libverto-0.3.2-10.el10.x86_64.rpm",
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
    name = "libvirt-client-0__11.10.0-2.el10.aarch64",
    sha256 = "df833b65e4ee6047544d26fb099c8dd977f1c5037f18b26769185c98b6a94870",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-client-11.10.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-client-0__11.10.0-2.el10.s390x",
    sha256 = "b078aac179d3c9a77109bb0c517bbc951098951e7ba2e2f429ff6c3abbd958d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-client-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-client-0__11.10.0-2.el10.x86_64",
    sha256 = "22d5b09ad651276b13a01d6a70527f7e01fe38ea4aacc4e2c4ecaef56263b638",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-client-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-client-0__11.9.0-1.el9.aarch64",
    sha256 = "cbdd67bf9c4cc0c3d962c56b352d228901e8a833934f82eae835d9f2a8a49398",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-11.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cbdd67bf9c4cc0c3d962c56b352d228901e8a833934f82eae835d9f2a8a49398",
    ],
)

rpm(
    name = "libvirt-client-0__11.9.0-1.el9.s390x",
    sha256 = "99b00a3251310318b35e01ee502a1ddb24d1a3e8fc777b51d5b45fa0f7130a75",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-client-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/99b00a3251310318b35e01ee502a1ddb24d1a3e8fc777b51d5b45fa0f7130a75",
    ],
)

rpm(
    name = "libvirt-client-0__11.9.0-1.el9.x86_64",
    sha256 = "86a5c27f9b4f446760465836ed354761de714a6b08a75d2ea7a7c706a331ab86",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86a5c27f9b4f446760465836ed354761de714a6b08a75d2ea7a7c706a331ab86",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-7.el9.x86_64",
    sha256 = "ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af",
    urls = ["https://storage.googleapis.com/builddeps/ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af"],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-2.el10.aarch64",
    sha256 = "bf4cb56a6935f66e96ff7db2e69c1caadeeda686e36cc1e94101558d5efbb885",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-11.10.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-2.el10.s390x",
    sha256 = "b5f611d5be71e57fbcefc6cbd93c0975f8f34b5b4399791a810e7007548057df",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-2.el10.x86_64",
    sha256 = "7692c66b827c2115742678e945a6e2439b4172fe36928f8d342af8840d7ed6bc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.9.0-1.el9.aarch64",
    sha256 = "e55036abe63905c336e4ab2a81ecb49c98ce83397dab4ed3a177487ba13e7fb4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-11.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e55036abe63905c336e4ab2a81ecb49c98ce83397dab4ed3a177487ba13e7fb4",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.9.0-1.el9.s390x",
    sha256 = "957cdeb3a3e0708cc01912ce5c2eaadb01970b43f4fb29bed70768d79eb159a4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/957cdeb3a3e0708cc01912ce5c2eaadb01970b43f4fb29bed70768d79eb159a4",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.9.0-1.el9.x86_64",
    sha256 = "bc470fd7f29bb48410c65f12500681d27ee3adbc73e59cc7dd9fbb45a3ead2de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc470fd7f29bb48410c65f12500681d27ee3adbc73e59cc7dd9fbb45a3ead2de",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-7.el9.x86_64",
    sha256 = "13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1",
    urls = ["https://storage.googleapis.com/builddeps/13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-2.el10.aarch64",
    sha256 = "1f7475430f454794794889d5d5e827c07ec842719a74844ea3e797eaaf30e6a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-11.10.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-2.el10.s390x",
    sha256 = "ed3b8d512b2156297ce05fa5d974b77e2304ce36db5acc9f45e3aeced83f49a5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-2.el10.x86_64",
    sha256 = "77b0d98d943bcf6d09637e8b1888b17b09e451f31a56866d722052e6bd2aa558",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.9.0-1.el9.aarch64",
    sha256 = "19fa15b5fd2bec4cfd6470e9595320be9c2c01ea5320a4dc4f2b3df1be87c5b0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-11.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19fa15b5fd2bec4cfd6470e9595320be9c2c01ea5320a4dc4f2b3df1be87c5b0",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.9.0-1.el9.s390x",
    sha256 = "431c2264ae9b0cb79110f04586f7004222235050cf22989330a619dbefa7d71b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/431c2264ae9b0cb79110f04586f7004222235050cf22989330a619dbefa7d71b",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.9.0-1.el9.x86_64",
    sha256 = "fc357ad81e321752e136a0252bf93fcc4c51705fdad5fe152e8b458c1bd7c804",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fc357ad81e321752e136a0252bf93fcc4c51705fdad5fe152e8b458c1bd7c804",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.10.0-7.el9.x86_64",
    sha256 = "8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3",
    urls = ["https://storage.googleapis.com/builddeps/8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3"],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.10.0-2.el10.s390x",
    sha256 = "2e9b6c638e06624e6e78e316121f91e530d1d5dbf34a27b62fe71758bb42ea3e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-secret-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.10.0-2.el10.x86_64",
    sha256 = "be3b620e72f39b4782184492396675f0ed95de3d6d82632bb432e127c0a8106f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.9.0-1.el9.s390x",
    sha256 = "63920895b3ff60a8203e684f1bb1c0eab39f18c97ccd88b12e1de800f5894ca6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-secret-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/63920895b3ff60a8203e684f1bb1c0eab39f18c97ccd88b12e1de800f5894ca6",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.9.0-1.el9.x86_64",
    sha256 = "9f7ad76f1918935dd25157f6f37930706b81449392e6bb720db1b4e7f9df9619",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f7ad76f1918935dd25157f6f37930706b81449392e6bb720db1b4e7f9df9619",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.10.0-7.el9.x86_64",
    sha256 = "a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067",
    urls = ["https://storage.googleapis.com/builddeps/a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067"],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.10.0-2.el10.s390x",
    sha256 = "583dfcaa660df1ae76d49215b2c3b3983e6a8015ba156694b27c81c85a7c87da",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-storage-core-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.10.0-2.el10.x86_64",
    sha256 = "2fe6c3f8c80b180f59cf4d12b5bd94af01506a90f9667dda480698c1aa0a5cdf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.9.0-1.el9.s390x",
    sha256 = "30e35bd1454accdcd90c255d660ce1e7e724329e70a424dee078de3d66547ebe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-storage-core-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/30e35bd1454accdcd90c255d660ce1e7e724329e70a424dee078de3d66547ebe",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.9.0-1.el9.x86_64",
    sha256 = "9066ad1b758b950809a2761abc99ee8d96aab03d9150f24c3038f61e76180e32",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9066ad1b758b950809a2761abc99ee8d96aab03d9150f24c3038f61e76180e32",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-7.el9.x86_64",
    sha256 = "7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff",
    urls = ["https://storage.googleapis.com/builddeps/7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff"],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-2.el10.aarch64",
    sha256 = "4862db80ab75e5306a3443a385926e1f5078f464e31ff36bb8230848e043a441",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-11.10.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-2.el10.s390x",
    sha256 = "3da1b34dfa7726533953bf97ddf7f5a1d055c1497bbe746147ee1666594646fd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-2.el10.x86_64",
    sha256 = "847390056c804637424c0c6d8a62a6e9b4b5ded8369e299b6af6a5a3156f098b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.9.0-1.el9.aarch64",
    sha256 = "a4fd6a282bf09969d6e11f2a7a68577d9c87fc529e5af76c64a94c0683f88bb7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-11.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a4fd6a282bf09969d6e11f2a7a68577d9c87fc529e5af76c64a94c0683f88bb7",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.9.0-1.el9.s390x",
    sha256 = "49f189f5704b1e2b02333bb72954854659b00753eb1941931f672d3d647b9b35",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/49f189f5704b1e2b02333bb72954854659b00753eb1941931f672d3d647b9b35",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.9.0-1.el9.x86_64",
    sha256 = "4e499cd2d4331c986b80249cabbbfa5c991b2a98ae301ff539a5ca201301400b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4e499cd2d4331c986b80249cabbbfa5c991b2a98ae301ff539a5ca201301400b",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-2.el10.aarch64",
    sha256 = "1d8309c2ee9c50bfb28602f1e03861348f88cb9258474fe7f968f00e9b717854",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/aarch64/os/Packages/libvirt-devel-11.10.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-2.el10.s390x",
    sha256 = "b04793934320541758e5d8322e0d15a434dbc3c31f45d81ec8103693011ebbae",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/libvirt-devel-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-2.el10.x86_64",
    sha256 = "b0738e63829bef9e10c4e76cc2b041aa467026a31f3ac3e52f9d0ee340c8e878",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/libvirt-devel-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-devel-0__11.9.0-1.el9.aarch64",
    sha256 = "b4e5f496902c96280d2717b215e2365759c5e535210af6a8c0b7847e1c912be7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-11.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b4e5f496902c96280d2717b215e2365759c5e535210af6a8c0b7847e1c912be7",
    ],
)

rpm(
    name = "libvirt-devel-0__11.9.0-1.el9.s390x",
    sha256 = "3043bd36232c9625aba61c805307d557f7c05c19bb25169b1798aa09ef97f351",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libvirt-devel-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3043bd36232c9625aba61c805307d557f7c05c19bb25169b1798aa09ef97f351",
    ],
)

rpm(
    name = "libvirt-devel-0__11.9.0-1.el9.x86_64",
    sha256 = "37a170292e19a69bbbfcae3197a226dce84be287f4b5a53037880835b03c0d85",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/37a170292e19a69bbbfcae3197a226dce84be287f4b5a53037880835b03c0d85",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-7.el9.x86_64",
    sha256 = "72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39",
    urls = ["https://storage.googleapis.com/builddeps/72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39"],
)

rpm(
    name = "libvirt-libs-0__11.10.0-2.el10.aarch64",
    sha256 = "f6d4042b82116a37c4cf8af8611cef3dc91345b41b20e570391890986c851156",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-libs-11.10.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libvirt-libs-0__11.10.0-2.el10.s390x",
    sha256 = "5c53fff93f7ac5bf69ca0c94ac9e910f0b35bd4d4d2ca2c8fc63a203b9404c6f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-libs-11.10.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "libvirt-libs-0__11.10.0-2.el10.x86_64",
    sha256 = "9cc0acc8b4c1d3a7bac6e11e6bad344097f1223760cf4c8306a2e4bfd178504a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-libs-11.10.0-2.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libvirt-libs-0__11.9.0-1.el9.aarch64",
    sha256 = "5c189b32115edc2a71f73ab70e1964272ec6ee08d3c259c5104486683bc6d301",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-11.9.0-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5c189b32115edc2a71f73ab70e1964272ec6ee08d3c259c5104486683bc6d301",
    ],
)

rpm(
    name = "libvirt-libs-0__11.9.0-1.el9.s390x",
    sha256 = "f87768c7ac75945faddeefdbcd3c560eb4f72cc26ed09720df2b0438d20e3770",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-libs-11.9.0-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f87768c7ac75945faddeefdbcd3c560eb4f72cc26ed09720df2b0438d20e3770",
    ],
)

rpm(
    name = "libvirt-libs-0__11.9.0-1.el9.x86_64",
    sha256 = "00e7d11d7515ad4343e73bfe2b01c3917bda350ba072fda3b0ee8ee5cddddb43",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-11.9.0-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/00e7d11d7515ad4343e73bfe2b01c3917bda350ba072fda3b0ee8ee5cddddb43",
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
    name = "libxcrypt-0__4.4.36-10.el10.aarch64",
    sha256 = "465ade16c8f369b5abc1a39671f882bc645ac90b1aeaa29cdfc3958e57640144",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libxcrypt-4.4.36-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.36-10.el10.s390x",
    sha256 = "d14c5523dd6c7f233277acbbb11fb2644f26e91da18e6184ae6ad445e3835a36",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libxcrypt-4.4.36-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.36-10.el10.x86_64",
    sha256 = "503a29c4c767637d810c7e89ed4355fe0b588381cb360517585fb56a2cf5ee46",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libxcrypt-4.4.36-10.el10.x86_64.rpm",
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
    name = "libxcrypt-devel-0__4.4.36-10.el10.aarch64",
    sha256 = "2f86c95726f3c3efdcb2d97f5d0020e86d254defebb084df7c13a5fa51442b5a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libxcrypt-devel-4.4.36-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.4.36-10.el10.s390x",
    sha256 = "a3f57faa74cefedf8baddec91311a5e0cafe73878e83c3447335495c7ed7934b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libxcrypt-devel-4.4.36-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.4.36-10.el10.x86_64",
    sha256 = "ccee1b09985e24bfed47cf7b5c965d7e0e869862ac55f3b7f783cdcec93716f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libxcrypt-devel-4.4.36-10.el10.x86_64.rpm",
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
    name = "libxcrypt-static-0__4.4.36-10.el10.aarch64",
    sha256 = "a2d13d4bb5d7ca66384346f0b90801862e9e58e016870565929a699bb1c15feb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/aarch64/os/Packages/libxcrypt-static-4.4.36-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.4.36-10.el10.s390x",
    sha256 = "42d6494724ad9eb96949ac0632b9658b488c5db84d7aa2f9db3d20198a29f9fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/libxcrypt-static-4.4.36-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.4.36-10.el10.x86_64",
    sha256 = "a4b6f28908fa252bac1c366f91bb37117c0b56ebce1743933dcf2029f10f86c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/libxcrypt-static-4.4.36-10.el10.x86_64.rpm",
    ],
)

rpm(
    name = "libxml2-0__2.12.5-9.el10.aarch64",
    sha256 = "4cfe820e11a4e226235d10f4db732bda5c3c44a62e7162766e70819c0ad2aa7d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libxml2-2.12.5-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libxml2-0__2.12.5-9.el10.s390x",
    sha256 = "ecfe86d6e6523469d104f42480eddad00802fefeb6bb758f98cd60a9e3c63472",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libxml2-2.12.5-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "libxml2-0__2.12.5-9.el10.x86_64",
    sha256 = "5ae5a9056e911b43ef89472929139ac06fb5fe4a88b03614a221faf6145640e3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libxml2-2.12.5-9.el10.x86_64.rpm",
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
    name = "libxslt-0__1.1.39-8.el10.s390x",
    sha256 = "1e6ec8eb0dac9858a45d3b42ac6755ce77182581ca77af4eec34af9256aa874e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libxslt-1.1.39-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "libxslt-0__1.1.39-8.el10.x86_64",
    sha256 = "394d4f76d3a0ed6283ecc2e840520958361f9764402d616014687f92ad750d81",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libxslt-1.1.39-8.el10.x86_64.rpm",
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
    name = "libzstd-0__1.5.5-9.el10.aarch64",
    sha256 = "474a4497b7901176be4a59895cd02bba744300fd673668ef068bd1dfc5e129c7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libzstd-1.5.5-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "libzstd-0__1.5.5-9.el10.s390x",
    sha256 = "59d29a77a5792bbc4ce42b3ac700a1df776ace058e040f391374f011d39f0eef",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libzstd-1.5.5-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "libzstd-0__1.5.5-9.el10.x86_64",
    sha256 = "86f3cb406d56283119c45ec8c1f4689aa37ff6c04cf44f6608c10cfdcccdb2c1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libzstd-1.5.5-9.el10.x86_64.rpm",
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
    name = "lua-libs-0__5.4.6-7.el10.aarch64",
    sha256 = "f8e353910af43a3d81e92ed6355e7d85b64e6946c7af48ac3900bd107e9d91cc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/lua-libs-5.4.6-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "lua-libs-0__5.4.6-7.el10.s390x",
    sha256 = "e0676e298166577f2766305025f4c99fe774473f084fe13fcdf8937b4b0e5eab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lua-libs-5.4.6-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "lua-libs-0__5.4.6-7.el10.x86_64",
    sha256 = "cb9268a17c06928ffb0805ff43d733b0c67171ff3a969cc16b40c7f3e59d64f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lua-libs-5.4.6-7.el10.x86_64.rpm",
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
    name = "lz4-libs-0__1.9.4-8.el10.aarch64",
    sha256 = "7db176282f02ed0243d66b9136e1269e4db85da61157392ecc0febeac418ec85",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/lz4-libs-1.9.4-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.4-8.el10.s390x",
    sha256 = "bd0ba485141caa931c930540a150a55a89ab3dfc6bba448aa592e5b9551dee2e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lz4-libs-1.9.4-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.4-8.el10.x86_64",
    sha256 = "de360e857e8465c4b38990375e9435efc78e20d022afe42dbf2986d11fc2c759",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lz4-libs-1.9.4-8.el10.x86_64.rpm",
    ],
)

rpm(
    name = "lzo-0__2.10-14.el10.aarch64",
    sha256 = "677b7730dfa8e554a8ddd22940c5c6288b0d51cb09e9547c150905e856fb0575",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/lzo-2.10-14.el10.aarch64.rpm",
    ],
)

rpm(
    name = "lzo-0__2.10-14.el10.s390x",
    sha256 = "32bde43a3a00f4b5d078b2c831270f8cc195664e0e3ab5b1c5bcc6dc802e33d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lzo-2.10-14.el10.s390x.rpm",
    ],
)

rpm(
    name = "lzo-0__2.10-14.el10.x86_64",
    sha256 = "9e4f4e6dc19d15eb865805a43f5834b0ce3a405dcc6df0fba72f0b73f59685a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lzo-2.10-14.el10.x86_64.rpm",
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
    name = "lzop-0__1.04-16.el10.aarch64",
    sha256 = "e463088918132202d22ada263686b6b723af02b6a49066fd6f9d48cf191cb25e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/lzop-1.04-16.el10.aarch64.rpm",
    ],
)

rpm(
    name = "lzop-0__1.04-16.el10.s390x",
    sha256 = "5eeeda50a19223224ac6de6428853904e6210b0c11223e71aa39848e613bcb0b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lzop-1.04-16.el10.s390x.rpm",
    ],
)

rpm(
    name = "lzop-0__1.04-16.el10.x86_64",
    sha256 = "925d4dfbf179f00032be3a3a1ec7cf8ed8f9b9b2cd8ea87c2a4da1e97fcfd180",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lzop-1.04-16.el10.x86_64.rpm",
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
    name = "make-1__4.4.1-9.el10.aarch64",
    sha256 = "4cd069f5132c87ad16d02ff648b6389e3e303b41661362252134519993afc45c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/make-4.4.1-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "make-1__4.4.1-9.el10.s390x",
    sha256 = "aa138cd7a41f8b054dbecd74462e796b47d58cd9058ad8b56734c0cf242dcd80",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/make-4.4.1-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "make-1__4.4.1-9.el10.x86_64",
    sha256 = "7d0b52fe16c826f8b08656abd70509987e69e2b8a9f0c42fda803d41a9e7c74e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/make-4.4.1-9.el10.x86_64.rpm",
    ],
)

rpm(
    name = "mpdecimal-0__2.5.1-12.el10.aarch64",
    sha256 = "f7755f98208b3f400c950ba46acf568f113029893fede5770d19eedadfa0b3ea",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/mpdecimal-2.5.1-12.el10.aarch64.rpm",
    ],
)

rpm(
    name = "mpdecimal-0__2.5.1-12.el10.s390x",
    sha256 = "2dd0dbab48a3481fab6bcb4554b0854cb66c8d142ef28b8e97b7dfc96d4c2c93",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/mpdecimal-2.5.1-12.el10.s390x.rpm",
    ],
)

rpm(
    name = "mpdecimal-0__2.5.1-12.el10.x86_64",
    sha256 = "7d1762e4770170efa93ff4f7e07cf523f62d3e3378f50d87d7b307cd8a73ee77",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/mpdecimal-2.5.1-12.el10.x86_64.rpm",
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
    name = "mpfr-0__4.2.1-6.el10.aarch64",
    sha256 = "ff42c0656eb7659b733cf29dcec9db96216ce1725bb4f633df360fde860a5b47",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/mpfr-4.2.1-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "mpfr-0__4.2.1-6.el10.s390x",
    sha256 = "0a428bc21172ed27583705623cef0fca8f0ff487ca251b268cb6f0cd8ef95cb9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/mpfr-4.2.1-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "mpfr-0__4.2.1-6.el10.x86_64",
    sha256 = "1285b14028ab77959841b214f3d36800df49c35ce5922f1bb44fe72e34da74f4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/mpfr-4.2.1-6.el10.x86_64.rpm",
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
    name = "ncurses-base-0__6.4-14.20240127.el10.aarch64",
    sha256 = "6e439dd9afd65b489675c37f03bdcd950353ed6b822c31ee620fe370642db042",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.4-14.20240127.el10.noarch.rpm",
    ],
)

rpm(
    name = "ncurses-base-0__6.4-14.20240127.el10.s390x",
    sha256 = "6e439dd9afd65b489675c37f03bdcd950353ed6b822c31ee620fe370642db042",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/ncurses-base-6.4-14.20240127.el10.noarch.rpm",
    ],
)

rpm(
    name = "ncurses-base-0__6.4-14.20240127.el10.x86_64",
    sha256 = "6e439dd9afd65b489675c37f03bdcd950353ed6b822c31ee620fe370642db042",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.4-14.20240127.el10.noarch.rpm",
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
    name = "ncurses-libs-0__6.4-14.20240127.el10.aarch64",
    sha256 = "d781030401acc90746bf50c039bac36bcb812b33bb76965d6e6cfed43787a45b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.4-14.20240127.el10.aarch64.rpm",
    ],
)

rpm(
    name = "ncurses-libs-0__6.4-14.20240127.el10.s390x",
    sha256 = "9dccdd3dc565eb6a75ff4e9d359a7de762e11bf834ebfedf03a815188da8d429",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/ncurses-libs-6.4-14.20240127.el10.s390x.rpm",
    ],
)

rpm(
    name = "ncurses-libs-0__6.4-14.20240127.el10.x86_64",
    sha256 = "80256770f1fb9639ea2e1cc744ba6cdbc6b65850d74c6e66a64fc9bcbb4837f4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.4-14.20240127.el10.x86_64.rpm",
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
    name = "nftables-1__1.1.5-3.el10.aarch64",
    sha256 = "be675a4acfbf9cf768d95dc25d5390eda069cf2ee8ac774bb81e701cd5ae3135",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/nftables-1.1.5-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "nftables-1__1.1.5-3.el10.s390x",
    sha256 = "ce456b58b2f65c45c8ecaf147eb6936e106a9bc2169e9ba39f5ab760479e42a0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/nftables-1.1.5-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "nftables-1__1.1.5-3.el10.x86_64",
    sha256 = "b04022e2f5e38f6600bf9d3f8cad167704e19bd1191cdbf0dd62db3f15b16a1c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/nftables-1.1.5-3.el10.x86_64.rpm",
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
    name = "nmap-ncat-4__7.92-5.el10.aarch64",
    sha256 = "7975edb3d4e9c583a41707bdd6f4d21dee67e571f7b7338352d75cf09130612e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.92-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "nmap-ncat-4__7.92-5.el10.s390x",
    sha256 = "a0dd5969f49cf59a2448a75498a25007ea270fa25eed6b9d70c1148b88e37a96",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/nmap-ncat-7.92-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "nmap-ncat-4__7.92-5.el10.x86_64",
    sha256 = "30fcce0936e6fad42a6cca2d6999758fd637ad6d7057bd5c155eb8f49af53157",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.92-5.el10.x86_64.rpm",
    ],
)

rpm(
    name = "npth-0__1.6-21.el10.s390x",
    sha256 = "47f1f79ad844c4d845591871bc752bf8677fb257fa2cc4d58778fab215965bf1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/npth-1.6-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "npth-0__1.6-21.el10.x86_64",
    sha256 = "9d5de697dd346d3eeac85008ab93fbfce90ea49342418402959eda90829578d0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/npth-1.6-21.el10.x86_64.rpm",
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
    name = "numactl-libs-0__2.0.19-3.el10.aarch64",
    sha256 = "81016ab56c83cb8c221216794571ae58bb914e21dd3794c242f7ce8a8d8fbf8f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.19-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el10.s390x",
    sha256 = "184bd0085cc03d74e317a2aad472fa7638fffc35e4e1a314700e31b00398a6cc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/numactl-libs-2.0.19-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el10.x86_64",
    sha256 = "263ee2cba1d57996778f70045fbc4657067f73edafd6c6b04f4599c3eb12fbfd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.19-3.el10.x86_64.rpm",
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
    name = "numad-0__0.5-50.20251104git.el10.aarch64",
    sha256 = "942f7db59b047cc56e6c53c5bb9a2a84ba4715088021e85526e973d9485bc8fa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/numad-0.5-50.20251104git.el10.aarch64.rpm",
    ],
)

rpm(
    name = "numad-0__0.5-50.20251104git.el10.x86_64",
    sha256 = "91def9a46ee7b6ee35a11276987c7eace5b18e97d6d967fe09139e0a01cb0731",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/numad-0.5-50.20251104git.el10.x86_64.rpm",
    ],
)

rpm(
    name = "openldap-0__2.6.10-1.el10.s390x",
    sha256 = "8ccbbb3c19df87e02214012a0fb7eed53455db552b6548e482734f65039b6057",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/openldap-2.6.10-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "openldap-0__2.6.10-1.el10.x86_64",
    sha256 = "c9b225c90d849b679e4ecdc4108703b54749bed23af829d3238f1551dd88fd27",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/openldap-2.6.10-1.el10.x86_64.rpm",
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
    name = "openssl-fips-provider-1__3.5.5-1.el10.aarch64",
    sha256 = "8508a817efb181727d4cbed2ef81ddbde1f1da487709f0a6aaaea4ac265acea2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/openssl-fips-provider-3.5.5-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-1.el10.s390x",
    sha256 = "d9844964d1c85617b43d0a9a9cc92a98d50008925abfc6766edecda075085047",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/openssl-fips-provider-3.5.5-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-1.el10.x86_64",
    sha256 = "8cd0358b2f324315431e075aa4f96aba00f2be2fbea15929808d77b38c450b7b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/openssl-fips-provider-3.5.5-1.el10.x86_64.rpm",
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
    name = "openssl-libs-1__3.5.5-1.el10.aarch64",
    sha256 = "b57aea518dd32913ada2d53def2aa2b67ddd97f9826c95392a9ff8942c0bc992",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.5.5-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.5-1.el10.s390x",
    sha256 = "63346962f9adf622da34b1727f33e5abddbedd9ad47dfea2526f2f28ca0dcd47",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/openssl-libs-3.5.5-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.5-1.el10.x86_64",
    sha256 = "f398762ae421f1aa605af7e3e8770da8b3fa6fbb0afcf74a2d82945dc4670c39",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.5.5-1.el10.x86_64.rpm",
    ],
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
    name = "osinfo-db-0__20250606-1.el10.s390x",
    sha256 = "44f126f2f67319b5c84345c63150dd5d87ed468cd63b18b4320593505b90b4d1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/osinfo-db-20250606-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "osinfo-db-0__20250606-1.el10.x86_64",
    sha256 = "44f126f2f67319b5c84345c63150dd5d87ed468cd63b18b4320593505b90b4d1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/osinfo-db-20250606-1.el10.noarch.rpm",
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
    name = "osinfo-db-tools-0__1.11.0-8.el10.s390x",
    sha256 = "fed0a8870fb28338db4b8b2bb6f57d44fcbfcaafe88187d787e3bf6cd5f911f0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/osinfo-db-tools-1.11.0-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "osinfo-db-tools-0__1.11.0-8.el10.x86_64",
    sha256 = "31d38586cdd723e3de145e9b03dd1f4eaa2e63681323a7d0bcce3a47cc2e1d62",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/osinfo-db-tools-1.11.0-8.el10.x86_64.rpm",
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
    name = "p11-kit-0__0.26.1-1.el10.aarch64",
    sha256 = "e1aad342a866ae8c7b44333c84a82f1830827fad7d1543a6cfccfe9c95b4e6e6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/p11-kit-0.26.1-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "p11-kit-0__0.26.1-1.el10.s390x",
    sha256 = "6ba3c0dac20f1f19e2adf1d345556042196205bc68f98f34b0ff6bdadad2be00",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/p11-kit-0.26.1-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "p11-kit-0__0.26.1-1.el10.x86_64",
    sha256 = "7cd91bff6dca8f9b5620ab55588549a42f09f83668b6313ce38968965c59dad7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/p11-kit-0.26.1-1.el10.x86_64.rpm",
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
    name = "p11-kit-trust-0__0.26.1-1.el10.aarch64",
    sha256 = "73ff16cefdcac061ee9c3c5528935989350240aec659a331ec255cc9b09c9f44",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.26.1-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.1-1.el10.s390x",
    sha256 = "3d9823071c8cd27c08591be53cbc1dbfeef7c70bfdb95bf40bd9e591a7e21bea",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/p11-kit-trust-0.26.1-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.1-1.el10.x86_64",
    sha256 = "f11e6e6f177fbb0d28083213edbf5a2dd91a9af897ea3d81d10ca41fae50db37",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.26.1-1.el10.x86_64.rpm",
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
    name = "pam-0__1.6.1-9.el10.aarch64",
    sha256 = "48762bdea0227ff022ec0740b1241147e76d40898a041628e61d20fe8aea344c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pam-1.6.1-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "pam-0__1.6.1-9.el10.s390x",
    sha256 = "565a9b5f35ff92d0f91330974e0da3e7322bb87b398a081cef94258e1db17c76",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pam-1.6.1-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "pam-0__1.6.1-9.el10.x86_64",
    sha256 = "0709423d4705d5f06c4cd6005d205a1a26fb3c4a9d08bbe197fa103924506157",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pam-1.6.1-9.el10.x86_64.rpm",
    ],
)

rpm(
    name = "pam-libs-0__1.6.1-9.el10.aarch64",
    sha256 = "6d055fc43ae94b745214a675bc077261395a9c9475a66138912b77314fb064fd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pam-libs-1.6.1-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "pam-libs-0__1.6.1-9.el10.s390x",
    sha256 = "8927066863a1128bb08750c8e01a26f57b7393454a604436233b1779d55ae655",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pam-libs-1.6.1-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "pam-libs-0__1.6.1-9.el10.x86_64",
    sha256 = "4f5115114c1bf4882ce2ed5e7629f07a8d0b6e6a93412e9b3a00ab21d8f49973",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pam-libs-1.6.1-9.el10.x86_64.rpm",
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
    name = "parted-0__3.6-7.el10.s390x",
    sha256 = "bba170cbf71b85ebc299c466f4a15c14ce80fae6a138ec87014b772bab377f02",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/parted-3.6-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "parted-0__3.6-7.el10.x86_64",
    sha256 = "bb339dd10bc7951376243b5a9ae11e18f3ecd235db576739da006a147c6bf412",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/parted-3.6-7.el10.x86_64.rpm",
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
    name = "passt-0__0__caret__20251210.gd04c480-2.el10.aarch64",
    sha256 = "b13beb46385431f9d715217c1712c50bd0c96f25d8658d531495954486c04efb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/passt-0%5E20251210.gd04c480-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "passt-0__0__caret__20251210.gd04c480-2.el10.s390x",
    sha256 = "795615af5477338e3835946b9befca2a259a6c33523b4c9fed07c5ac53742a2d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/passt-0%5E20251210.gd04c480-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "passt-0__0__caret__20251210.gd04c480-2.el10.x86_64",
    sha256 = "d7c62dd6032ecc02fd020f1673d79106190531d3c4ed7c8843abd0cf78c2ebbf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/passt-0%5E20251210.gd04c480-2.el10.x86_64.rpm",
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
    name = "pcre2-0__10.44-1.el10.3.aarch64",
    sha256 = "23f2a34aa9bc9c8c6662e93d184e07d7e01d45d0fb1b554fd3ed92c03ba2ae3c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pcre2-10.44-1.el10.3.aarch64.rpm",
    ],
)

rpm(
    name = "pcre2-0__10.44-1.el10.3.s390x",
    sha256 = "7577ec5ef81b0aa96e340c6d292c4b828e957508962ec1ed68c1f048dff3998e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pcre2-10.44-1.el10.3.s390x.rpm",
    ],
)

rpm(
    name = "pcre2-0__10.44-1.el10.3.x86_64",
    sha256 = "773781e3aa9994fa8d6105ddc0b3d00fdd735bd589a5d9fe40fe96be6a7d89a7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pcre2-10.44-1.el10.3.x86_64.rpm",
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
    name = "pcre2-syntax-0__10.44-1.el10.3.aarch64",
    sha256 = "71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pcre2-syntax-10.44-1.el10.3.noarch.rpm",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.44-1.el10.3.s390x",
    sha256 = "71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pcre2-syntax-10.44-1.el10.3.noarch.rpm",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.44-1.el10.3.x86_64",
    sha256 = "71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pcre2-syntax-10.44-1.el10.3.noarch.rpm",
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
    name = "pixman-0__0.43.4-2.el10.aarch64",
    sha256 = "dc2c0f98c210e8209690b1d2a4fffa348b6ad22062461f4b3ebc7d7f6dd0246e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/pixman-0.43.4-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "pixman-0__0.43.4-2.el10.s390x",
    sha256 = "269fcda361ff485d379f8a773e47752758b8c58e288f78196f169149570af637",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/pixman-0.43.4-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "pixman-0__0.43.4-2.el10.x86_64",
    sha256 = "c91d0077a917e843a009c69b63793de4b8d2f9a81414f63e319ac31bbf6a08cb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/pixman-0.43.4-2.el10.x86_64.rpm",
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
    name = "pkgconf-0__2.1.0-3.el10.aarch64",
    sha256 = "5bd76130128a85e6275c6e56f7e519532425cd2a5d2db7a795a4d1d15f7d0d57",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pkgconf-2.1.0-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "pkgconf-0__2.1.0-3.el10.s390x",
    sha256 = "010973bdd551e8489eb97446701fdf3100b8dd0b1ea7efd650412d8869d8181a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pkgconf-2.1.0-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "pkgconf-0__2.1.0-3.el10.x86_64",
    sha256 = "ced8f494b664667d52245ff94ce6c0b2cad135586a36a9ff7f81281d1533f178",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pkgconf-2.1.0-3.el10.x86_64.rpm",
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
    name = "pkgconf-m4-0__2.1.0-3.el10.aarch64",
    sha256 = "4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pkgconf-m4-2.1.0-3.el10.noarch.rpm",
    ],
)

rpm(
    name = "pkgconf-m4-0__2.1.0-3.el10.s390x",
    sha256 = "4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pkgconf-m4-2.1.0-3.el10.noarch.rpm",
    ],
)

rpm(
    name = "pkgconf-m4-0__2.1.0-3.el10.x86_64",
    sha256 = "4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pkgconf-m4-2.1.0-3.el10.noarch.rpm",
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
    name = "pkgconf-pkg-config-0__2.1.0-3.el10.aarch64",
    sha256 = "3d646b74ccc730b097ceba50c5a054a6017b61e354b0e8731c66b5c266d55e40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pkgconf-pkg-config-2.1.0-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__2.1.0-3.el10.s390x",
    sha256 = "f895f22efbaa5a4f978600b3df480eabab7bb2eea7f0f8e90897ab4d76ae2102",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pkgconf-pkg-config-2.1.0-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__2.1.0-3.el10.x86_64",
    sha256 = "4f5231ffccc59b5f1c42d85cc0dafea9b6901107660931071cf1e65f99af1e0b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pkgconf-pkg-config-2.1.0-3.el10.x86_64.rpm",
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
    name = "policycoreutils-0__3.9-2.el10.aarch64",
    sha256 = "6efb426864c00bad20c92e898916a9bc4a217934b5dcf5e77e08a1e10bb1b88f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.9-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "policycoreutils-0__3.9-2.el10.s390x",
    sha256 = "ca50814a174dc418970e111a10d1c3524075c0df8647b18121bfbd07e26d829a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/policycoreutils-3.9-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "policycoreutils-0__3.9-2.el10.x86_64",
    sha256 = "0dc8d2b0aeb70502b00f3697c80a71e7d8e03a281fa1e77d720d6a1f71d9a9db",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.9-2.el10.x86_64.rpm",
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
    name = "polkit-0__125-4.el10.aarch64",
    sha256 = "c7e294ea2b01e7d3f52f3e88d9520f1c57ed7577a220fc1c25092f05c2c2be09",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/polkit-125-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "polkit-0__125-4.el10.s390x",
    sha256 = "746bed9c6883ac33d60f35f0233289dc8e73137cc2e2028a6308b805385fb1e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/polkit-125-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "polkit-0__125-4.el10.x86_64",
    sha256 = "e4965cc1a34a64e8b5cc6c8738de6c6c0cf2f08f2dced23f9d88427575c3a386",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/polkit-125-4.el10.x86_64.rpm",
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
    name = "polkit-libs-0__125-4.el10.aarch64",
    sha256 = "0b7197e7b1c5c394aeb3254f5678c7ec80053566cb367fd6c2d79a11e9a22f37",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/polkit-libs-125-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "polkit-libs-0__125-4.el10.s390x",
    sha256 = "b8715d596f893869f60408286325995dbeeb570286fd397c3c611d86c30a7014",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/polkit-libs-125-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "polkit-libs-0__125-4.el10.x86_64",
    sha256 = "dea631790902108c8e2932c272cb24a5949c3c329009b96cf2f9c8fa5aaee29a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/polkit-libs-125-4.el10.x86_64.rpm",
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
    name = "popt-0__1.19-8.el10.aarch64",
    sha256 = "4c727d11de14d8bf1bc0df2be55c75cb0200a685c2737c740e636dedd3edbb0c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/popt-1.19-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "popt-0__1.19-8.el10.s390x",
    sha256 = "3dca46c310266fc9cce48d39651984d726cf727e12f446b70db78cd6f96e3515",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/popt-1.19-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "popt-0__1.19-8.el10.x86_64",
    sha256 = "bd15d2816600655a5241bc3efe6e1ac386061ba6ff2d05e53c70683db8761e5f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/popt-1.19-8.el10.x86_64.rpm",
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
    name = "procps-ng-0__4.0.4-9.el10.aarch64",
    sha256 = "41441a1d9724db5a3327dd3fd0dec9a1f5a19345677f346bd4ea295b1657bfc8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/procps-ng-4.0.4-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "procps-ng-0__4.0.4-9.el10.s390x",
    sha256 = "76e7f6323ac9f9a769a9b27d39673bf2a4cbbc4cf740d186d3b38e3d3f98238e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/procps-ng-4.0.4-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "procps-ng-0__4.0.4-9.el10.x86_64",
    sha256 = "8c618a494766c8c85fee4acd5fa730722bfab04694f72d01219324ec7adb43fb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/procps-ng-4.0.4-9.el10.x86_64.rpm",
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
    name = "protobuf-c-0__1.5.0-6.el10.aarch64",
    sha256 = "3e17d0b103c0444852bbb952e39a64bb18b36f797cf5d4fd1bea57d9bc2c4cbe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/protobuf-c-1.5.0-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "protobuf-c-0__1.5.0-6.el10.s390x",
    sha256 = "70c17f3805e9ecb6eaef0c13ae83b889405c22bdf09eafc74d3f0ba26e0882c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/protobuf-c-1.5.0-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "protobuf-c-0__1.5.0-6.el10.x86_64",
    sha256 = "a7e792e5ed4f89d5f48eb60453619a6e0ad5cd34469526952c1748f1a99ce3ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/protobuf-c-1.5.0-6.el10.x86_64.rpm",
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
    name = "psmisc-0__23.6-8.el10.aarch64",
    sha256 = "cca0153b72dfb9c9e2f9f8386514ff9591b6166e0746ff32d5cda0eeb9adbaba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/psmisc-23.6-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "psmisc-0__23.6-8.el10.s390x",
    sha256 = "3c0f3724b4040c7a6c07f5405850ed172f906daa9a7904cd78c8e52c680d4611",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/psmisc-23.6-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "psmisc-0__23.6-8.el10.x86_64",
    sha256 = "9fea410c82d95565a4cbb178da9557ec9cef3512d573efd4dd940c9f2c4219cf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/psmisc-23.6-8.el10.x86_64.rpm",
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
    name = "publicsuffix-list-dafsa-0__20240107-5.el10.s390x",
    sha256 = "440cb6e03187dfd68f62abf1dd751ace84ec8e2179d7de45dde348cf2e7dba11",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/publicsuffix-list-dafsa-20240107-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "publicsuffix-list-dafsa-0__20240107-5.el10.x86_64",
    sha256 = "440cb6e03187dfd68f62abf1dd751ace84ec8e2179d7de45dde348cf2e7dba11",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/publicsuffix-list-dafsa-20240107-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-0__3.12.12-3.el10.aarch64",
    sha256 = "21775dabf6661090390f8c62c21d436de108d0487e1cc2530c0523b869bd2b45",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-3.12.12-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "python3-0__3.12.12-3.el10.s390x",
    sha256 = "a6bd9d3a578d948bb4deca00b9f0dbc9c6f3d44a318a97ebd469a8d89ca3d39e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-3.12.12-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "python3-0__3.12.12-3.el10.x86_64",
    sha256 = "cd420c83445309ee7ded2740579742da574ab210ed2209d11472a34307fc1b5c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-3.12.12-3.el10.x86_64.rpm",
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
    name = "python3-configshell-1__1.1.30-9.el10.aarch64",
    sha256 = "bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-configshell-1.1.30-9.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-9.el10.s390x",
    sha256 = "bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-configshell-1.1.30-9.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-9.el10.x86_64",
    sha256 = "bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-configshell-1.1.30-9.el10.noarch.rpm",
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
    name = "python3-dbus-0__1.3.2-8.el10.aarch64",
    sha256 = "6508de73c7fb8966b0d05f631af1002cb5238237791b0bd1b085384b9d6e15fd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-dbus-1.3.2-8.el10.aarch64.rpm",
    ],
)

rpm(
    name = "python3-dbus-0__1.3.2-8.el10.s390x",
    sha256 = "98bf40e2dadd95cc0640650673b54d2dc7ebfa48bbba64a7857aede9c42550a5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-dbus-1.3.2-8.el10.s390x.rpm",
    ],
)

rpm(
    name = "python3-dbus-0__1.3.2-8.el10.x86_64",
    sha256 = "95455a0bc5c76704ba2e46f5dd68b8bd47027b83ae551d62f02b15415a78a164",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-dbus-1.3.2-8.el10.x86_64.rpm",
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
    name = "python3-gobject-base-0__3.46.0-7.el10.aarch64",
    sha256 = "66267f4d40ef4d29b7084c60752688856a78949de1347292d4fe25be501e024b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-gobject-base-3.46.0-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.46.0-7.el10.s390x",
    sha256 = "19a92f5cebbd47d89e69c63172d504154790e0ef013967d872ceb7ab0bc4b3f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-gobject-base-3.46.0-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.46.0-7.el10.x86_64",
    sha256 = "dd8582c736f50481252e556960f885c63af2d6a64888027d9345e35ec9bc0e27",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-gobject-base-3.46.0-7.el10.x86_64.rpm",
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
    name = "python3-gobject-base-noarch-0__3.46.0-7.el10.aarch64",
    sha256 = "2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-gobject-base-noarch-3.46.0-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.46.0-7.el10.s390x",
    sha256 = "2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-gobject-base-noarch-3.46.0-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.46.0-7.el10.x86_64",
    sha256 = "2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-gobject-base-noarch-3.46.0-7.el10.noarch.rpm",
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
    name = "python3-kmod-0__0.9.2-6.el10.aarch64",
    sha256 = "75e70ceef21104220cd13f4de2cd23669c5660c23a07d15e471b39fa61418e8e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-kmod-0.9.2-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "python3-kmod-0__0.9.2-6.el10.s390x",
    sha256 = "cc4bf90e94d8a7ac36762f21899fc174fe734aae502af314933685c0a342e832",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-kmod-0.9.2-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "python3-kmod-0__0.9.2-6.el10.x86_64",
    sha256 = "c9f50b595ee5a45bb14d571974662ed42c84bcb8660c3a78da891038847da651",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-kmod-0.9.2-6.el10.x86_64.rpm",
    ],
)

rpm(
    name = "python3-libs-0__3.12.12-3.el10.aarch64",
    sha256 = "9f56d7dd4675899a19162cce8b778a232afc8f4513b6c69c2e66dd6a4fe0bf5e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-libs-3.12.12-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "python3-libs-0__3.12.12-3.el10.s390x",
    sha256 = "0103b4031a7fb202d5361241ddfcee0d96e0ec33bc7161ca617a9a019d6f30f8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-libs-3.12.12-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "python3-libs-0__3.12.12-3.el10.x86_64",
    sha256 = "d20b719ab3bd3456197544ea9f8e14d0c566987153480ca264ce1e15056b6da8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-libs-3.12.12-3.el10.x86_64.rpm",
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
    name = "python3-pip-wheel-0__23.3.2-7.el10.aarch64",
    sha256 = "19b2ce4f91ed680267712a2d2158e679267f9163db71f57e9db5f5c684ac15d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-23.3.2-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-pip-wheel-0__23.3.2-7.el10.s390x",
    sha256 = "19b2ce4f91ed680267712a2d2158e679267f9163db71f57e9db5f5c684ac15d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-pip-wheel-23.3.2-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-pip-wheel-0__23.3.2-7.el10.x86_64",
    sha256 = "19b2ce4f91ed680267712a2d2158e679267f9163db71f57e9db5f5c684ac15d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-23.3.2-7.el10.noarch.rpm",
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
    name = "python3-pyparsing-0__3.1.1-7.el10.aarch64",
    sha256 = "8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-pyparsing-3.1.1-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-pyparsing-0__3.1.1-7.el10.s390x",
    sha256 = "8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-pyparsing-3.1.1-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-pyparsing-0__3.1.1-7.el10.x86_64",
    sha256 = "8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-pyparsing-3.1.1-7.el10.noarch.rpm",
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
    name = "python3-pyudev-0__0.24.1-10.el10.aarch64",
    sha256 = "69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-pyudev-0.24.1-10.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-pyudev-0__0.24.1-10.el10.s390x",
    sha256 = "69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-pyudev-0.24.1-10.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-pyudev-0__0.24.1-10.el10.x86_64",
    sha256 = "69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-pyudev-0.24.1-10.el10.noarch.rpm",
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
    name = "python3-rtslib-0__2.1.76-12.el10.aarch64",
    sha256 = "4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/python3-rtslib-2.1.76-12.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-12.el10.s390x",
    sha256 = "4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/python3-rtslib-2.1.76-12.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-12.el10.x86_64",
    sha256 = "4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/python3-rtslib-2.1.76-12.el10.noarch.rpm",
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
    name = "python3-six-0__1.16.0-16.el10.aarch64",
    sha256 = "587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-six-1.16.0-16.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-six-0__1.16.0-16.el10.s390x",
    sha256 = "587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-six-1.16.0-16.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-six-0__1.16.0-16.el10.x86_64",
    sha256 = "587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-six-1.16.0-16.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-typing-extensions-0__4.9.0-6.el10.aarch64",
    sha256 = "d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-typing-extensions-4.9.0-6.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-typing-extensions-0__4.9.0-6.el10.s390x",
    sha256 = "d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-typing-extensions-4.9.0-6.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-typing-extensions-0__4.9.0-6.el10.x86_64",
    sha256 = "d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-typing-extensions-4.9.0-6.el10.noarch.rpm",
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
    name = "python3-urwid-0__2.5.3-4.el10.aarch64",
    sha256 = "b754bb3fe723d716e43404b58f706b506b14123fbfd3cad0fa016da10ce7aaf0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-urwid-2.5.3-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "python3-urwid-0__2.5.3-4.el10.s390x",
    sha256 = "e73588c982971858102bb9a6804391709e72e5dd765034f5e4b1d7bea8c59332",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-urwid-2.5.3-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "python3-urwid-0__2.5.3-4.el10.x86_64",
    sha256 = "8ccc08409b227ee8b2cc6879a3b2f84a0e9cc792b720f3b5b7dd1381109a51cd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-urwid-2.5.3-4.el10.x86_64.rpm",
    ],
)

rpm(
    name = "python3-wcwidth-0__0.2.6-6.el10.aarch64",
    sha256 = "0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-wcwidth-0.2.6-6.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-wcwidth-0__0.2.6-6.el10.s390x",
    sha256 = "0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-wcwidth-0.2.6-6.el10.noarch.rpm",
    ],
)

rpm(
    name = "python3-wcwidth-0__0.2.6-6.el10.x86_64",
    sha256 = "0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-wcwidth-0.2.6-6.el10.noarch.rpm",
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
    name = "qemu-img-18__10.1.0-11.el10.aarch64",
    sha256 = "a569f70d8f5c2e1c0693b7310b13c033a303832e908ed1aca0ade126d51e7be2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-img-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-img-18__10.1.0-11.el10.s390x",
    sha256 = "2b1d9a00f8deb15450df7ffeba8b2d3852a9a22b2599971223d197a3deb81b90",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-img-10.1.0-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "qemu-img-18__10.1.0-11.el10.x86_64",
    sha256 = "49c32346581296f9c9341aef0b7ee7bcc948218e336b88f4a6bb30216613d823",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-img-10.1.0-11.el10.x86_64.rpm",
    ],
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
    name = "qemu-kvm-common-18__10.1.0-11.el10.aarch64",
    sha256 = "28f2c6c7a45643e8af4cafe028d2b771c0ff6a8033a5629cd2bdf55f99832ba9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-common-18__10.1.0-11.el10.s390x",
    sha256 = "833d96b773e0d69215ab2b975a37af667c9ad1c1f0faa9ea65c684f2e3a0f6dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-common-10.1.0-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "qemu-kvm-common-18__10.1.0-11.el10.x86_64",
    sha256 = "12c7b1d1ae4ed4d32900b6d445812f686017b26860505f01a72f169a56fb36bf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-10.1.0-11.el10.x86_64.rpm",
    ],
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
    name = "qemu-kvm-core-18__10.1.0-11.el10.aarch64",
    sha256 = "a9c162bc2167ebe69056eea04bf99ce3e8ab5660db5d5fc8b3c360585eac83e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-core-18__10.1.0-11.el10.s390x",
    sha256 = "c31b2474238c901dd63d5ed453764bae4d616ad3f7ea86a8fea32d0d8110ca2f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-core-10.1.0-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "qemu-kvm-core-18__10.1.0-11.el10.x86_64",
    sha256 = "86194391f63a26d0421541da474365563adb2019d07e52f4fcc1c1a96dd734bc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-10.1.0-11.el10.x86_64.rpm",
    ],
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
    name = "qemu-kvm-device-display-virtio-gpu-18__10.1.0-11.el10.aarch64",
    sha256 = "3308a97062d4b394298f304222d51ada4eaeceb880b6d08e65b916ac928ae164",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-18__10.1.0-11.el10.s390x",
    sha256 = "727b89e3828fbc7bbfaefad0659e5370ea4673284d2afcf0862bbd4f41e8664f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-18__10.1.0-11.el10.x86_64",
    sha256 = "2fefdf73cbb5824bc5364ce0affbad48faf2a9d57b6cca784b0b26700a3b8e9b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-11.el10.x86_64.rpm",
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
    name = "qemu-kvm-device-display-virtio-gpu-ccw-18__10.1.0-11.el10.s390x",
    sha256 = "d7e9e48ba36616ec8dbfbcc0fc7b62de04200c7a8935c7e311ffc410bf31f42d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-10.1.0-11.el10.s390x.rpm",
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
    name = "qemu-kvm-device-display-virtio-gpu-pci-18__10.1.0-11.el10.aarch64",
    sha256 = "460690efde08449e27d99c7a6ec8ed54fed47e5e146e252e7c3975679cab8cab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-18__10.1.0-11.el10.x86_64",
    sha256 = "bc7ad5145e50a5d53d3dcaca9c0e3febe9ef4995b284017b201e0ab59467d060",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-11.el10.x86_64.rpm",
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
    name = "qemu-kvm-device-display-virtio-vga-18__10.1.0-11.el10.x86_64",
    sha256 = "eb39072537c28199bd05552e04af2a0a9ea5861e30c392fbb4d426567f06c6ac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-vga-10.1.0-11.el10.x86_64.rpm",
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
    name = "qemu-kvm-device-usb-host-18__10.1.0-11.el10.aarch64",
    sha256 = "7adce6b58164570ffe743bc42e0a40872e8b0a2275401862ecb3c599e8488b9e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-18__10.1.0-11.el10.s390x",
    sha256 = "d3917c4122fef0a85c1395661b6fd318e76debb5d753f2a0b6782d42e50ea7f9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-10.1.0-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-18__10.1.0-11.el10.x86_64",
    sha256 = "fd948b93fd42a6de232bd55c838908e44f2185ea137f87843d5c153c4eda88c3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-10.1.0-11.el10.x86_64.rpm",
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
    name = "qemu-kvm-device-usb-redirect-18__10.1.0-11.el10.aarch64",
    sha256 = "b25636328ec64b0192150761f8d619618c43038b5e56422549f4c2b5f3f0104e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-18__10.1.0-11.el10.x86_64",
    sha256 = "363e8c6ab90bb303261e00ebd847fc590078f123a68cfc6f34e8e0700103a5d1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-11.el10.x86_64.rpm",
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
    name = "qemu-pr-helper-18__10.1.0-11.el10.aarch64",
    sha256 = "a4c5a19a7264828f44413d02d2aa140c10ef9ad4c7e4d0f26759660e05b23992",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-10.1.0-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-pr-helper-18__10.1.0-11.el10.x86_64",
    sha256 = "86a0b76c0d861db465d13e53e49b18fcce07593fcc5cd6a126a2af52dd557c9a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-10.1.0-11.el10.x86_64.rpm",
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
    name = "readline-0__8.2-11.el10.aarch64",
    sha256 = "a1f1fe411d40cb802c7a3e3b105faffe05c2376563bec2d59c71ed28778684cd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/readline-8.2-11.el10.aarch64.rpm",
    ],
)

rpm(
    name = "readline-0__8.2-11.el10.s390x",
    sha256 = "fd8cc3c7dd19bf773afb0488b2521d1710dfc1d8337fbff55c9f8d84572a4f9f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/readline-8.2-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "readline-0__8.2-11.el10.x86_64",
    sha256 = "d8e2d7c011d0e5c56b6875919ce036605862db02d59d6983d470bc5757021783",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/readline-8.2-11.el10.x86_64.rpm",
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
    name = "rpm-0__4.19.1.1-21.el10.aarch64",
    sha256 = "a77920742b41f7215c8bd21f9df7101ebb685bd00cced62b5d2e6fda8c9ca23f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/rpm-4.19.1.1-21.el10.aarch64.rpm",
    ],
)

rpm(
    name = "rpm-0__4.19.1.1-21.el10.s390x",
    sha256 = "dfec4a45376597db60e956a387810a6b93b1538354f38a2fea04f9289dfde19d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/rpm-4.19.1.1-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "rpm-0__4.19.1.1-21.el10.x86_64",
    sha256 = "ef6bd8b6b43a4704e6f14dddd0900f94d111d5fc35a69de78f93999de26c3ed2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/rpm-4.19.1.1-21.el10.x86_64.rpm",
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
    name = "rpm-libs-0__4.19.1.1-21.el10.aarch64",
    sha256 = "753191e6f82ed6e841c8ae435ec2d0b67b8daf31c0e40843cf14645ce5f718c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.19.1.1-21.el10.aarch64.rpm",
    ],
)

rpm(
    name = "rpm-libs-0__4.19.1.1-21.el10.s390x",
    sha256 = "65f0e8add7fa5e01c16babd21f66cf6b08f429c21f91e4fa1efe3d72932bb674",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/rpm-libs-4.19.1.1-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "rpm-libs-0__4.19.1.1-21.el10.x86_64",
    sha256 = "130985a89e230b91fa829d790de35042843355ad25d41cfa24808296d87047fd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.19.1.1-21.el10.x86_64.rpm",
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
    name = "rpm-sequoia-0__1.9.0.3-1.el10.aarch64",
    sha256 = "493981060d42eb43b76084339119d2ad32c631a69316ccf9cac06e2c01685c17",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/rpm-sequoia-1.9.0.3-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "rpm-sequoia-0__1.9.0.3-1.el10.s390x",
    sha256 = "263f4ca38f6593d00bea74ab4d2ccd7e5a7a478f45b59a8e85c5b0277c83b953",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/rpm-sequoia-1.9.0.3-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "rpm-sequoia-0__1.9.0.3-1.el10.x86_64",
    sha256 = "1e503067eec855e443b262b21b6cda63867a5362814331b8bff3b4f0a0c46b1d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/rpm-sequoia-1.9.0.3-1.el10.x86_64.rpm",
    ],
)

rpm(
    name = "scrub-0__2.6.1-11.el10.s390x",
    sha256 = "527e3cf6d20579cbc13efd1b13c639c243e538b6e6121a22efd46ec13d2fa557",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/scrub-2.6.1-11.el10.s390x.rpm",
    ],
)

rpm(
    name = "scrub-0__2.6.1-11.el10.x86_64",
    sha256 = "935258a3ef8ada2d8cba193df349c9d1dc62d38018b9613aab6f727f93655f85",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/scrub-2.6.1-11.el10.x86_64.rpm",
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
    name = "seabios-0__1.17.0-1.el10.x86_64",
    sha256 = "18044b16fa0f0256167f42ba6ab1f8b5ac338747e150d3c9aead064cd28255c9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/seabios-1.17.0-1.el10.x86_64.rpm",
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
    name = "seabios-bin-0__1.17.0-1.el10.x86_64",
    sha256 = "5edf7ad5039c74faab0fe3bc7f9741db6153c6f9ebe3367d20701d4e659d930d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/seabios-bin-1.17.0-1.el10.noarch.rpm",
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
    name = "seavgabios-bin-0__1.17.0-1.el10.x86_64",
    sha256 = "5ed6563e3d13189aa28fe86d0fef8540d61539aa44dd2d5558ca068e79df4ea2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/seavgabios-bin-1.17.0-1.el10.noarch.rpm",
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
    name = "sed-0__4.9-3.el10.aarch64",
    sha256 = "ffa5a588c4b731f4d0f53095e1c26f8aed9cc7c1e40538908b8429dde3405597",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/sed-4.9-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "sed-0__4.9-3.el10.s390x",
    sha256 = "9d736fb53a44b453a669da46a0f99150cfd207b12d466b11891c5838d317abca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/sed-4.9-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "sed-0__4.9-3.el10.x86_64",
    sha256 = "e0f382e42cee7264161ae86a9f063aed753b2a64cfea78d61ba8c35b6a980995",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/sed-4.9-3.el10.x86_64.rpm",
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
    name = "selinux-policy-0__42.1.15-1.el10.aarch64",
    sha256 = "b1f6f0846abc65c7ef73c7b86138a09c7e491c0c2225bb12e1628ece8c7c4eda",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/selinux-policy-42.1.15-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-0__42.1.15-1.el10.s390x",
    sha256 = "b1f6f0846abc65c7ef73c7b86138a09c7e491c0c2225bb12e1628ece8c7c4eda",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/selinux-policy-42.1.15-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-0__42.1.15-1.el10.x86_64",
    sha256 = "b1f6f0846abc65c7ef73c7b86138a09c7e491c0c2225bb12e1628ece8c7c4eda",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/selinux-policy-42.1.15-1.el10.noarch.rpm",
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
    name = "selinux-policy-targeted-0__42.1.15-1.el10.aarch64",
    sha256 = "03963934c85fa312233e75132d7ee664b766cda13f8ea1501d19551afb947598",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-42.1.15-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__42.1.15-1.el10.s390x",
    sha256 = "03963934c85fa312233e75132d7ee664b766cda13f8ea1501d19551afb947598",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-42.1.15-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__42.1.15-1.el10.x86_64",
    sha256 = "03963934c85fa312233e75132d7ee664b766cda13f8ea1501d19551afb947598",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-42.1.15-1.el10.noarch.rpm",
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
    name = "setup-0__2.14.5-7.el10.aarch64",
    sha256 = "bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/setup-2.14.5-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "setup-0__2.14.5-7.el10.s390x",
    sha256 = "bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/setup-2.14.5-7.el10.noarch.rpm",
    ],
)

rpm(
    name = "setup-0__2.14.5-7.el10.x86_64",
    sha256 = "bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/setup-2.14.5-7.el10.noarch.rpm",
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
    name = "sevctl-0__0.4.3-3.el10.x86_64",
    sha256 = "790b23bb704c9c42b9478b859f909186751771d9c8e864b5b1eb7a0158e690ca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/sevctl-0.4.3-3.el10.x86_64.rpm",
    ],
)

rpm(
    name = "shadow-utils-2__4.15.0-9.el10.aarch64",
    sha256 = "57e87032b85ab8275629c60274b46ca29d425d2feaf5df9131fe05eb914813ed",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.15.0-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "shadow-utils-2__4.15.0-9.el10.s390x",
    sha256 = "cadac7aec232378077b4858a673566f425d90ef1a04cc69f1fd5dd259c20571e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/shadow-utils-4.15.0-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "shadow-utils-2__4.15.0-9.el10.x86_64",
    sha256 = "ec463d422dc9f65451543bff9b321093f9c1d37ae85735cf71e79317a7e1faa6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.15.0-9.el10.x86_64.rpm",
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
    name = "snappy-0__1.1.10-7.el10.aarch64",
    sha256 = "cc7bc94dc673d8d6d5b4559036648410e790e2c59e2254bc6acd1578fb5e6781",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/snappy-1.1.10-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "snappy-0__1.1.10-7.el10.s390x",
    sha256 = "03a4ac68f64e146332224557a251a9d051dada615815dd6eb2f4bb22b73826e0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/snappy-1.1.10-7.el10.s390x.rpm",
    ],
)

rpm(
    name = "snappy-0__1.1.10-7.el10.x86_64",
    sha256 = "952dcfbe66d93bece4a4f3753ce721594acbd2af82cd5ca02bf9028375c136b3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/snappy-1.1.10-7.el10.x86_64.rpm",
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
    name = "sqlite-libs-0__3.46.1-5.el10.aarch64",
    sha256 = "217f00d515ac790fd028f0fd70a195a288258d0e1157ce6293ab65d29a965cf1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.46.1-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "sqlite-libs-0__3.46.1-5.el10.s390x",
    sha256 = "34d97be1a2df9d53a327cec2ca15887897168b880a19a4b7af2b860ad80b35fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/sqlite-libs-3.46.1-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "sqlite-libs-0__3.46.1-5.el10.x86_64",
    sha256 = "fa8bd71adaf88ff1b893731fd5f49c949cf3f618332c9b80390113237699f8e7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.46.1-5.el10.x86_64.rpm",
    ],
)

rpm(
    name = "sssd-client-0__2.12.0-1.el10.aarch64",
    sha256 = "38b0c51913fee7468a2e585c2e9ae811cbcd682ac266f3681da43dcd223d7718",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/sssd-client-2.12.0-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "sssd-client-0__2.12.0-1.el10.s390x",
    sha256 = "a7455c34c936bd13af3360275301ad413fc246e65a969cc32b3fb1b25c7870c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/sssd-client-2.12.0-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "sssd-client-0__2.12.0-1.el10.x86_64",
    sha256 = "83b0d541eaed2737ab42036d731df94094458b6f0fa39db2e23391b947d59ff0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/sssd-client-2.12.0-1.el10.x86_64.rpm",
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
    name = "swtpm-0__0.9.0-2.el10.aarch64",
    sha256 = "2ab32944b56a5d288754d90a3758d667cdc3703631488a2c2f4ac357880bff0b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/swtpm-0.9.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "swtpm-0__0.9.0-2.el10.s390x",
    sha256 = "4fc33cbd8611b571b7952968cd67999ca3d457f7d331290ed5850f70d292f89d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/swtpm-0.9.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "swtpm-0__0.9.0-2.el10.x86_64",
    sha256 = "2754a70eda7d481964e28e610d493f0c705ae966e75dda10ee901a6cf2ef5919",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/swtpm-0.9.0-2.el10.x86_64.rpm",
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
    name = "swtpm-libs-0__0.9.0-2.el10.aarch64",
    sha256 = "e4233f1d21b64737a8c42fecaa652b2388b897a8748b416cf4bd599f30dd7fe2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/swtpm-libs-0.9.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "swtpm-libs-0__0.9.0-2.el10.s390x",
    sha256 = "2cd497257c5a03b6e579f3ada2bef874350a0fbb0aad78cff5d38e247af20c0f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/swtpm-libs-0.9.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "swtpm-libs-0__0.9.0-2.el10.x86_64",
    sha256 = "57b1c9b2ab6540e9504f32e1aa58331fc98ad9476e47b82907f42ab17ab5288a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.9.0-2.el10.x86_64.rpm",
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
    name = "swtpm-tools-0__0.9.0-2.el10.aarch64",
    sha256 = "7da6702303b52d8724152e1235f9e3a1eca4dbb7e2dc2ce51f0b32ac6e04aef9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/swtpm-tools-0.9.0-2.el10.aarch64.rpm",
    ],
)

rpm(
    name = "swtpm-tools-0__0.9.0-2.el10.s390x",
    sha256 = "764ead866ad117155085de09e2d0cf5c483ef9aeb134421ae5cc00892aed146e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/swtpm-tools-0.9.0-2.el10.s390x.rpm",
    ],
)

rpm(
    name = "swtpm-tools-0__0.9.0-2.el10.x86_64",
    sha256 = "4c16e59ac5ef48d0e83d6e0a83ba1838c25c6531fb3ef3564bf98643e1e70503",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.9.0-2.el10.x86_64.rpm",
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
    name = "systemd-0__257-21.el10.aarch64",
    sha256 = "cda7a62d115e0f2b3cfe2aa6379930073a826e7799b8256fd13feebdee3b16ad",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-257-21.el10.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-0__257-21.el10.s390x",
    sha256 = "446b6ca90a23a4f2211ca381f58e66878a54bdaeec436452c0863f4738a716ef",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-257-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "systemd-0__257-21.el10.x86_64",
    sha256 = "a6e8c4fb89da61ab8291f1ae72ae65fbf96d08d54bc86733d4ebd37ccc7952f8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-257-21.el10.x86_64.rpm",
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
    name = "systemd-container-0__257-21.el10.aarch64",
    sha256 = "903869e3c0890ad2dee569aaac34ac17f9ec4f397a016f94311cfe733280db14",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-container-257-21.el10.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-container-0__257-21.el10.s390x",
    sha256 = "a6f0304a97d995b86d2d99bf723d10c962650e89e25a33a6e1a38a6b97c353ec",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-container-257-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "systemd-container-0__257-21.el10.x86_64",
    sha256 = "1f8d942392327786cc13036635573c3f7578e1fbd60e7c7746fb25697058e7ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-container-257-21.el10.x86_64.rpm",
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
    name = "systemd-libs-0__257-21.el10.aarch64",
    sha256 = "dd8aad1f044a279282bbaaabce515c8c3f485093be43b36c49c348ce31944655",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-libs-257-21.el10.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-libs-0__257-21.el10.s390x",
    sha256 = "abeafe36af17ba108485db15560f8983e6bd9f87355b22678891eef109624c9f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-libs-257-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "systemd-libs-0__257-21.el10.x86_64",
    sha256 = "250ae420e98f000e61b517b22cfc33d292f9cad17f21fd256412060dd8f88dde",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-libs-257-21.el10.x86_64.rpm",
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
    name = "systemd-pam-0__257-21.el10.aarch64",
    sha256 = "d7bac21680e6cb8ea830bb51a813185a49d94fb01453efd0eb62042349175b0e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-pam-257-21.el10.aarch64.rpm",
    ],
)

rpm(
    name = "systemd-pam-0__257-21.el10.s390x",
    sha256 = "82861e6b471fd026c08b12d1a93556cb696247bdc2ada50ab7da75a9401533b6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-pam-257-21.el10.s390x.rpm",
    ],
)

rpm(
    name = "systemd-pam-0__257-21.el10.x86_64",
    sha256 = "ce3c5eeb84b99b51f818b6b3084cc8eb1cf5f4252f3d9426ff518ee713b63fdd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-pam-257-21.el10.x86_64.rpm",
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
    name = "tar-2__1.35-10.el10.aarch64",
    sha256 = "7e8eff09bd7f39b2121c8420e5d91109341321bd63ca17311c36fe5f3b42ecf5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/tar-1.35-10.el10.aarch64.rpm",
    ],
)

rpm(
    name = "tar-2__1.35-10.el10.s390x",
    sha256 = "efb27d3706cbc79b151a5337af23ba2184e5b58f14a2ef0c059b369e4a62d12f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/tar-1.35-10.el10.s390x.rpm",
    ],
)

rpm(
    name = "tar-2__1.35-10.el10.x86_64",
    sha256 = "b3201e691a366ff630dd112a02b6f012d7b796ae87c458e5a9341a02c1fed4a9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/tar-1.35-10.el10.x86_64.rpm",
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
    name = "target-restore-0__2.1.76-12.el10.aarch64",
    sha256 = "aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/target-restore-2.1.76-12.el10.noarch.rpm",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-12.el10.s390x",
    sha256 = "aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/target-restore-2.1.76-12.el10.noarch.rpm",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-12.el10.x86_64",
    sha256 = "aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/target-restore-2.1.76-12.el10.noarch.rpm",
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
    name = "targetcli-0__2.1.58-5.el10.aarch64",
    sha256 = "687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/targetcli-2.1.58-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "targetcli-0__2.1.58-5.el10.s390x",
    sha256 = "687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/targetcli-2.1.58-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "targetcli-0__2.1.58-5.el10.x86_64",
    sha256 = "687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/targetcli-2.1.58-5.el10.noarch.rpm",
    ],
)

rpm(
    name = "tpm2-tss-0__4.1.3-5.el10.aarch64",
    sha256 = "ed8e49307084dbe71709e5528003811194fd9e00982222a89d289924e7accaea",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/tpm2-tss-4.1.3-5.el10.aarch64.rpm",
    ],
)

rpm(
    name = "tpm2-tss-0__4.1.3-5.el10.s390x",
    sha256 = "a18c9a2026d523cb1ec472a68212db1b3198a64e8a306548f0341c69bec6cbfa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/tpm2-tss-4.1.3-5.el10.s390x.rpm",
    ],
)

rpm(
    name = "tpm2-tss-0__4.1.3-5.el10.x86_64",
    sha256 = "863c36c073642a98f267bd3503fb2a505412341f2ae3c828f80e8188b4dc6d97",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/tpm2-tss-4.1.3-5.el10.x86_64.rpm",
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
    name = "tzdata-0__2025c-1.el10.aarch64",
    sha256 = "f42431990a112a5a422eae042de7d28bd2e9d9a971d9082771962b18a2951846",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/tzdata-2025c-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "tzdata-0__2025c-1.el10.s390x",
    sha256 = "f42431990a112a5a422eae042de7d28bd2e9d9a971d9082771962b18a2951846",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/tzdata-2025c-1.el10.noarch.rpm",
    ],
)

rpm(
    name = "tzdata-0__2025c-1.el10.x86_64",
    sha256 = "f42431990a112a5a422eae042de7d28bd2e9d9a971d9082771962b18a2951846",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/tzdata-2025c-1.el10.noarch.rpm",
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
    name = "unbound-libs-0__1.20.0-15.el10.aarch64",
    sha256 = "e3b8abdb07e727487f1a39e6de652c621e87c5279c361a26d852fc89d5430d86",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/unbound-libs-1.20.0-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "unbound-libs-0__1.20.0-15.el10.s390x",
    sha256 = "5dce54a982a4f63e22ac639fc17f027480d81af0916053f4608042b22125bc49",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/unbound-libs-1.20.0-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "unbound-libs-0__1.20.0-15.el10.x86_64",
    sha256 = "95b37ceb9b0a300e1a2894bb18c621db0a9aa12581cefea1eb4086095b0cb7d7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/unbound-libs-1.20.0-15.el10.x86_64.rpm",
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
    name = "usbredir-0__0.13.0-6.el10.aarch64",
    sha256 = "12a672104464f85388819600c8b2e4eee38cdb67342107485183bc2076b54fe2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/usbredir-0.13.0-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "usbredir-0__0.13.0-6.el10.x86_64",
    sha256 = "11551f45b3e60a80530431dfcd5a1d29c5624d34aeaf86dcfd6bcd0a4f87337f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/usbredir-0.13.0-6.el10.x86_64.rpm",
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
    name = "userspace-rcu-0__0.14.0-7.el10.aarch64",
    sha256 = "4c68e72d9cf6b3ae7b001c181998eeff4514058621e7517ebff26f315757c11d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/userspace-rcu-0.14.0-7.el10.aarch64.rpm",
    ],
)

rpm(
    name = "userspace-rcu-0__0.14.0-7.el10.x86_64",
    sha256 = "2ff9144b446e979b4d014fac8912e7ea9f9dbc2ebbe913c715629bf82aa34082",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/userspace-rcu-0.14.0-7.el10.x86_64.rpm",
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
    name = "util-linux-0__2.40.2-15.el10.aarch64",
    sha256 = "25ba1628a53deba99c20c5149678b950e2ebc60fb8612d2c22d7f6d05686b62f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/util-linux-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "util-linux-0__2.40.2-15.el10.s390x",
    sha256 = "9c838c66bb698a62102c9ee144069450c3482990ebcbbe5dccff249d8726d1bc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/util-linux-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "util-linux-0__2.40.2-15.el10.x86_64",
    sha256 = "8e486e9240aaede3947025a502a3f1b43d4aac04ab7400395d923251274ea76c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/util-linux-2.40.2-15.el10.x86_64.rpm",
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
    name = "util-linux-core-0__2.40.2-15.el10.aarch64",
    sha256 = "fea3374abdfaa5967ff585a7e8a06dd4612e6dc20ea3909658e04d95547e647e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.40.2-15.el10.aarch64.rpm",
    ],
)

rpm(
    name = "util-linux-core-0__2.40.2-15.el10.s390x",
    sha256 = "7e0133dd073238098b41fd6571da2ad71141c5e58eae5ba682b21b01d099e3a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/util-linux-core-2.40.2-15.el10.s390x.rpm",
    ],
)

rpm(
    name = "util-linux-core-0__2.40.2-15.el10.x86_64",
    sha256 = "d5192dee9734c527a4527b9cba6d754fd18b38f1e85d9ca0f2237e647409f3d3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.40.2-15.el10.x86_64.rpm",
    ],
)

rpm(
    name = "vim-data-2__9.1.083-6.el10.aarch64",
    sha256 = "09b0f20af6272c4f242bce1d67b15c743f625adb78a78ae20425fc877045ae83",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/vim-data-9.1.083-6.el10.noarch.rpm",
    ],
)

rpm(
    name = "vim-data-2__9.1.083-6.el10.s390x",
    sha256 = "09b0f20af6272c4f242bce1d67b15c743f625adb78a78ae20425fc877045ae83",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/vim-data-9.1.083-6.el10.noarch.rpm",
    ],
)

rpm(
    name = "vim-data-2__9.1.083-6.el10.x86_64",
    sha256 = "09b0f20af6272c4f242bce1d67b15c743f625adb78a78ae20425fc877045ae83",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/vim-data-9.1.083-6.el10.noarch.rpm",
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
    name = "vim-minimal-2__9.1.083-6.el10.aarch64",
    sha256 = "b542c48e8187bb909aca7deefc8f1358205744b3faa765df22890c77d5b4467d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/vim-minimal-9.1.083-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "vim-minimal-2__9.1.083-6.el10.s390x",
    sha256 = "520b48c16edcfe4a1ad88d7481001fc7c03f8c428c4ebd12579a8bb99459ec27",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/vim-minimal-9.1.083-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "vim-minimal-2__9.1.083-6.el10.x86_64",
    sha256 = "9e39250b2c331a51c63c47b319fa85d802d00e8d57e7d68992d28e98ebb09e6c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/vim-minimal-9.1.083-6.el10.x86_64.rpm",
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
    name = "virtiofsd-0__1.13.3-1.el10.aarch64",
    sha256 = "ad5eec8ff18d9610a2eff000406908876ea1b1af69cbf97c396b74efc65d54dd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/virtiofsd-1.13.3-1.el10.aarch64.rpm",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.3-1.el10.s390x",
    sha256 = "a9bd279fd632f35e33ba50593ab833ec2673878ec8e54a45d7e8e20c61fa7ed4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/virtiofsd-1.13.3-1.el10.s390x.rpm",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.3-1.el10.x86_64",
    sha256 = "fa45976edcd696c9fcda96b0c47b1e91d7c59ae83f5616bd047a0bad6b0be0ae",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/virtiofsd-1.13.3-1.el10.x86_64.rpm",
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
    name = "which-0__2.21-44.el10.aarch64",
    sha256 = "369a215b68f7dd87ce2b0c7be20425b63a19ba8b18a74775b474a717524388fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/which-2.21-44.el10.aarch64.rpm",
    ],
)

rpm(
    name = "which-0__2.21-44.el10.s390x",
    sha256 = "93c0edc58db280e4bcc7a7568fc9eb935a27666fea4659c0de6aa93e1017d0d7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/which-2.21-44.el10.s390x.rpm",
    ],
)

rpm(
    name = "which-0__2.21-44.el10.x86_64",
    sha256 = "8817b5d8ce0a8a07e38daa93a72d0cca53934e1631322b990d389ccb34376e1c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/which-2.21-44.el10.x86_64.rpm",
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
    name = "xorriso-0__1.5.6-6.el10.aarch64",
    sha256 = "8e152db322abfb8b173f703a0af4be1ef294abeb3dd78da974f800b074a06530",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/xorriso-1.5.6-6.el10.aarch64.rpm",
    ],
)

rpm(
    name = "xorriso-0__1.5.6-6.el10.s390x",
    sha256 = "ab7d3d43d22e8a4920453c73de0e60dbc3df69ab28042cad271df8a51cfa0b4b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/xorriso-1.5.6-6.el10.s390x.rpm",
    ],
)

rpm(
    name = "xorriso-0__1.5.6-6.el10.x86_64",
    sha256 = "2077b91e476836bec242f0fbf4a83384bfc785e9531bedb81ee825936105b017",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/xorriso-1.5.6-6.el10.x86_64.rpm",
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
    name = "xz-1__5.6.2-4.el10.aarch64",
    sha256 = "7bf62608392ae9fd5dd59add39723086f5a052c2064e0498c1641c572cd46460",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/xz-5.6.2-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "xz-1__5.6.2-4.el10.s390x",
    sha256 = "37e1052ce13b55ef1f4e33a8997728963f51c76223d165e2534f0cd6e8f9ba59",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/xz-5.6.2-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "xz-1__5.6.2-4.el10.x86_64",
    sha256 = "dc71c8e5b558c9f9fdea14a7d38819fc12ad8bdcb6834989188b225ca191eded",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/xz-5.6.2-4.el10.x86_64.rpm",
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
    name = "xz-libs-1__5.6.2-4.el10.aarch64",
    sha256 = "fcf207b0e6fe443fafe62fa43fc44ce16c8c118dd5e69491b3ad4b9eda72cc61",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/xz-libs-5.6.2-4.el10.aarch64.rpm",
    ],
)

rpm(
    name = "xz-libs-1__5.6.2-4.el10.s390x",
    sha256 = "7edd13c2a8dfb66b1e8c8a0d1d9259a1ff5cfb4891d568cc8f990664a02f7e32",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/xz-libs-5.6.2-4.el10.s390x.rpm",
    ],
)

rpm(
    name = "xz-libs-1__5.6.2-4.el10.x86_64",
    sha256 = "21733e8b6bf26b20633618adb074706972479080527f3f7a51246e83b3d4342e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/xz-libs-5.6.2-4.el10.x86_64.rpm",
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
    name = "zlib-ng-compat-0__2.2.3-3.el10.aarch64",
    sha256 = "a7870bf73b68086ae1fdd3e2fb6191bf79dff1ab5ae16b907efbb0befe590dca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/zlib-ng-compat-2.2.3-3.el10.aarch64.rpm",
    ],
)

rpm(
    name = "zlib-ng-compat-0__2.2.3-3.el10.s390x",
    sha256 = "89c8decb9febd474ba2f3fbb38c37577dd7098b349a9e766267723fb94f25962",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/zlib-ng-compat-2.2.3-3.el10.s390x.rpm",
    ],
)

rpm(
    name = "zlib-ng-compat-0__2.2.3-3.el10.x86_64",
    sha256 = "8fe3c2d5203810828fa3e4a5d84ae53172ffd27f4f0eec9d192b42b187795c09",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/zlib-ng-compat-2.2.3-3.el10.x86_64.rpm",
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

rpm(
    name = "zstd-0__1.5.5-9.el10.aarch64",
    sha256 = "b45bf236f2a5a034295eb933b3c056b302785fa122ecba44b98fec6d2d8b39a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/zstd-1.5.5-9.el10.aarch64.rpm",
    ],
)

rpm(
    name = "zstd-0__1.5.5-9.el10.s390x",
    sha256 = "def19135b3b6f01e46d9ee17e69ae1227e9addaf1fcd231596836000917fe393",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/zstd-1.5.5-9.el10.s390x.rpm",
    ],
)

rpm(
    name = "zstd-0__1.5.5-9.el10.x86_64",
    sha256 = "4ef415b98ddbe28f836b86699f4cec6002817ea20fb47499d3c6bb0814db6d4b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/zstd-1.5.5-9.el10.x86_64.rpm",
    ],
)
