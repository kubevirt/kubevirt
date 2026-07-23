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
    name = "package_metadata",
    sha256 = "5bd0cc7594ea528fd28f98d82457f157827d48cc20e07bcfdbb56072f35c8f67",
    strip_prefix = "supply-chain-0.0.6/metadata",
    urls = [
        "https://github.com/bazel-contrib/supply-chain/releases/download/v0.0.6/supply-chain-v0.0.6.tar.gz",
        "https://storage.googleapis.com/builddeps/5bd0cc7594ea528fd28f98d82457f157827d48cc20e07bcfdbb56072f35c8f67",
    ],
)

http_archive(
    name = "rules_oci",
    sha256 = "e987cab7a35475cb9c9060fc3f338a1fc8896c240295a3272968b217acefd0cb",
    strip_prefix = "rules_oci-2.3.0",
    urls = [
        "https://github.com/bazel-contrib/rules_oci/releases/download/v2.3.0/rules_oci-v2.3.0.tar.gz",
        "https://storage.googleapis.com/builddeps/e987cab7a35475cb9c9060fc3f338a1fc8896c240295a3272968b217acefd0cb",
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
    sha256 = "86d3dc8f59d253524f933aaf2f3c05896cb0b605fc35b460c0b4b039996124c6",
    urls = [
        "https://mirror.bazel.build/github.com/bazel-contrib/rules_go/releases/download/v0.60.0/rules_go-v0.60.0.zip",
        "https://github.com/bazel-contrib/rules_go/releases/download/v0.60.0/rules_go-v0.60.0.zip",
        "https://storage.googleapis.com/builddeps/86d3dc8f59d253524f933aaf2f3c05896cb0b605fc35b460c0b4b039996124c6",
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
    sha256 = "92329a7dbb26d0beacc43da669211546ea6627582793f4dd5f28837fde3a5c08",
    urls = [
        "https://github.com/bazel-contrib/bazel-gazelle/releases/download/v0.50.0/bazel-gazelle-v0.50.0.tar.gz",
        "https://storage.googleapis.com/builddeps/92329a7dbb26d0beacc43da669211546ea6627582793f4dd5f28837fde3a5c08",
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
    nogo = "@//:nogo_vet",
    version = "1.26.4",
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
    digest = "sha256:6f23ea6c3ce8b2a62c92e3b59c076118a366fb470af3fbf7d149bb51b98234ed",
    image = "gcr.io/distroless/base-debian12",
)

oci_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:a09192f27226e9e704bf7b4d0d69368c4fa0df54b21bcaabbaf5cc54dec4d77c",
    image = "gcr.io/distroless/base-debian12",
)

oci_pull(
    name = "go_image_base_s390x",
    digest = "sha256:7da65d894be967ee6cb28519bb0ae269c5521e22b88b2ce17b13aee259f3fa09",
    image = "gcr.io/distroless/base-debian12",
)

# Pull fedora container-disk preconfigured with ci tooling
# like stress and qemu guest agent pre-configured
oci_pull(
    name = "fedora_with_test_tooling",
    digest = "sha256:a53fd982787799c2d8cfaa37a2b6fbac4f416437768a25d2eb246dff46bb9d79",
    image = "quay.io/kubevirtci/fedora-with-test-tooling",
)

oci_pull(
    name = "alpine_with_test_tooling",
    digest = "sha256:8c8e8bb6cd81c75e492c678abb3e5f186d52eba2174ebabc328316250acfea58",
    image = "quay.io/kubevirtci/alpine-with-test-tooling-container-disk",
)

oci_pull(
    name = "alpine_with_test_tooling_arm64",
    digest = "sha256:5b443506b62f29f5ef5ac1bbf709338212b0b289ee2579e4feead42205685f43",
    image = "quay.io/kubevirtci/alpine-with-test-tooling-container-disk",
)

oci_pull(
    name = "alpine_with_test_tooling_s390x",
    digest = "sha256:1a52903133c00507607e8a82308a34923e89288d852762b9f4d5da227767e965",
    image = "quay.io/kubevirtci/alpine-with-test-tooling-container-disk",
)

oci_pull(
    name = "fedora_with_test_tooling_aarch64",
    digest = "sha256:0b29f1b32b2f8d75e35de165a121a9cb211741978972f27ed47e4879c1122b18",
    image = "quay.io/kubevirtci/fedora-with-test-tooling",
)

oci_pull(
    name = "fedora_with_test_tooling_s390x",
    digest = "sha256:ae6d6510dfb1e1cbcf09ad85c2c0b3e58494fe10bdaa720362934422037d42a2",
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
        "https://storage.googleapis.com/builddeps/e5c1d6460330fabe5ef57fb4b13d46ab0840f93556d898b5179f1b267f34455f",
    ],
)

rpm(
    name = "acl-0__2.3.2-4.el10.s390x",
    sha256 = "295d62b3d46571e5327671616bff8d1872af066f41719e09d5e0554d00001e49",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/acl-2.3.2-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/295d62b3d46571e5327671616bff8d1872af066f41719e09d5e0554d00001e49",
    ],
)

rpm(
    name = "acl-0__2.3.2-4.el10.x86_64",
    sha256 = "fd89f3c793d09fe633bf7721da719d29d599d01f65aaaa355b1b308a6fa580f2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/acl-2.3.2-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd89f3c793d09fe633bf7721da719d29d599d01f65aaaa355b1b308a6fa580f2",
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
        "https://storage.googleapis.com/builddeps/13d1cae28aecbc13bee2cf23391ec2ee41d39c51c9bb47f466fbad133d38f5c9",
    ],
)

rpm(
    name = "alternatives-0__1.30-2.el10.s390x",
    sha256 = "ab4f800759f602c25f483681b126b4eced6ba81331c9b613dd47a229379c71e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/alternatives-1.30-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ab4f800759f602c25f483681b126b4eced6ba81331c9b613dd47a229379c71e1",
    ],
)

rpm(
    name = "alternatives-0__1.30-2.el10.x86_64",
    sha256 = "1c8b83bf3dd0fa8d998a3c801986f50ea3661c2f8a21c60971c0391c381919c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/alternatives-1.30-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1c8b83bf3dd0fa8d998a3c801986f50ea3661c2f8a21c60971c0391c381919c8",
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
        "https://storage.googleapis.com/builddeps/f45973727e2dea77b2209bc9795c890abac187383a596b3cb81ab066b11ddb90",
    ],
)

rpm(
    name = "audit-libs-0__4.0.3-5.el10.s390x",
    sha256 = "1d7617a754258f58b0986c6f944621819381543eee344f60f348fc44bc2274c1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/audit-libs-4.0.3-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1d7617a754258f58b0986c6f944621819381543eee344f60f348fc44bc2274c1",
    ],
)

rpm(
    name = "audit-libs-0__4.0.3-5.el10.x86_64",
    sha256 = "a2be49cd9497b28aa9688b6e58bce216797c868559d249e3a08034e22d1e86f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/audit-libs-4.0.3-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a2be49cd9497b28aa9688b6e58bce216797c868559d249e3a08034e22d1e86f7",
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
        "https://storage.googleapis.com/builddeps/03bb6de04d7f5c64cf30fd0fe26301508b8fad5520c25fd1fc291132b01e0c7a",
    ],
)

rpm(
    name = "augeas-libs-0__1.14.2-0.9.20260120gitf4135e3.el10.x86_64",
    sha256 = "479f9bb17e1ede3ae449e0cc47474a935b519febdb65c214eedccc1e836aeb8b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/augeas-libs-1.14.2-0.9.20260120gitf4135e3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/479f9bb17e1ede3ae449e0cc47474a935b519febdb65c214eedccc1e836aeb8b",
    ],
)

rpm(
    name = "authselect-0__1.5.2-1.el10.aarch64",
    sha256 = "0354c0147a3921b6c3b67e4e38cc1d23be21533aec9e1935d666770258a67994",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/authselect-1.5.2-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0354c0147a3921b6c3b67e4e38cc1d23be21533aec9e1935d666770258a67994",
    ],
)

rpm(
    name = "authselect-0__1.5.2-1.el10.s390x",
    sha256 = "fe23c46f2bb599099c783986fcb202b8b8f4282294bdd8f068f22a0f3bc07644",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/authselect-1.5.2-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fe23c46f2bb599099c783986fcb202b8b8f4282294bdd8f068f22a0f3bc07644",
    ],
)

rpm(
    name = "authselect-0__1.5.2-1.el10.x86_64",
    sha256 = "f3fea14e3138deb4d1e6c28a9922eaae4c99baf511bd98c179acfd03fb0d61b1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/authselect-1.5.2-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f3fea14e3138deb4d1e6c28a9922eaae4c99baf511bd98c179acfd03fb0d61b1",
    ],
)

rpm(
    name = "authselect-libs-0__1.5.2-1.el10.aarch64",
    sha256 = "d90bd5fda96963f4c7a1359eed58974472b76374a116c3fea6eec601b52e5361",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/authselect-libs-1.5.2-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d90bd5fda96963f4c7a1359eed58974472b76374a116c3fea6eec601b52e5361",
    ],
)

rpm(
    name = "authselect-libs-0__1.5.2-1.el10.s390x",
    sha256 = "a5a6cd4f6eecab16227e57ac314fd7aafb0ebe7b31535834b847f15c69b235c9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/authselect-libs-1.5.2-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a5a6cd4f6eecab16227e57ac314fd7aafb0ebe7b31535834b847f15c69b235c9",
    ],
)

rpm(
    name = "authselect-libs-0__1.5.2-1.el10.x86_64",
    sha256 = "dfa12036f818655945efa17bfeac35e5bcb40a08ba89e74f57e3bb36e083000b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/authselect-libs-1.5.2-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dfa12036f818655945efa17bfeac35e5bcb40a08ba89e74f57e3bb36e083000b",
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
        "https://storage.googleapis.com/builddeps/76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    ],
)

rpm(
    name = "basesystem-0__11-22.el10.s390x",
    sha256 = "76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/basesystem-11-22.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    ],
)

rpm(
    name = "basesystem-0__11-22.el10.x86_64",
    sha256 = "76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/basesystem-11-22.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76ff57f4d7565cd0e49f5e6dc38f3707dfe6a6b61317d883c2701be4277f2abf",
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
        "https://storage.googleapis.com/builddeps/3f42c3de9fddc6e6c08f7c603ce29ed96d8d66f4425ce1c27bcb0d7d0e0490b5",
    ],
)

rpm(
    name = "bash-0__5.2.26-6.el10.s390x",
    sha256 = "07261872bd05c23366da7c2529b776dccfdf1a33c99d784370ebfde32d8909d7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/bash-5.2.26-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/07261872bd05c23366da7c2529b776dccfdf1a33c99d784370ebfde32d8909d7",
    ],
)

rpm(
    name = "bash-0__5.2.26-6.el10.x86_64",
    sha256 = "31eaf885847a6671a93e2b6e0d48e937ae5520f0442265aae19f4294260b5618",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/bash-5.2.26-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31eaf885847a6671a93e2b6e0d48e937ae5520f0442265aae19f4294260b5618",
    ],
)

rpm(
    name = "binutils-0__2.35.2-72.el9.aarch64",
    sha256 = "ca9bc2fa692d098e6dff6a0465cdc9955a7966e52357029d7d8c24d9b05864c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-2.35.2-72.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ca9bc2fa692d098e6dff6a0465cdc9955a7966e52357029d7d8c24d9b05864c9",
    ],
)

rpm(
    name = "binutils-0__2.35.2-72.el9.s390x",
    sha256 = "42c2bc7510f3adaed5d43ed7fd8e3ffe819a40059db6a0d92c9556c56ea34bd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-2.35.2-72.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/42c2bc7510f3adaed5d43ed7fd8e3ffe819a40059db6a0d92c9556c56ea34bd1",
    ],
)

rpm(
    name = "binutils-0__2.35.2-72.el9.x86_64",
    sha256 = "6f9b078ceaae9d8f4b87158b1fa911efe08e54287513232def5730d125b25900",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-2.35.2-72.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6f9b078ceaae9d8f4b87158b1fa911efe08e54287513232def5730d125b25900",
    ],
)

rpm(
    name = "binutils-0__2.41-65.el10.aarch64",
    sha256 = "82862b86b242ca6183a367206b7c36fe93ab11fddd4959b246f57f99597dd193",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/binutils-2.41-65.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/82862b86b242ca6183a367206b7c36fe93ab11fddd4959b246f57f99597dd193",
    ],
)

rpm(
    name = "binutils-0__2.41-65.el10.s390x",
    sha256 = "3a0ab5b2297739f6f7d2553c86f3e5d63581423552a28631cfbddc5713427181",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/binutils-2.41-65.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3a0ab5b2297739f6f7d2553c86f3e5d63581423552a28631cfbddc5713427181",
    ],
)

rpm(
    name = "binutils-0__2.41-65.el10.x86_64",
    sha256 = "2635e8e26f13f4195aeed6a16ef058180bec6b09b8b72f14900cebee2c88946a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/binutils-2.41-65.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2635e8e26f13f4195aeed6a16ef058180bec6b09b8b72f14900cebee2c88946a",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-72.el9.aarch64",
    sha256 = "49c74e9a687e54a746409fc7ef3684cae84f650b75cd88321fd1cb6f6078fff0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.35.2-72.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/49c74e9a687e54a746409fc7ef3684cae84f650b75cd88321fd1cb6f6078fff0",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-72.el9.s390x",
    sha256 = "1afe7937836aebdc3afb78947755f98dc0eb27c78e8172cec3f4b45123ebd16a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/binutils-gold-2.35.2-72.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1afe7937836aebdc3afb78947755f98dc0eb27c78e8172cec3f4b45123ebd16a",
    ],
)

rpm(
    name = "binutils-gold-0__2.35.2-72.el9.x86_64",
    sha256 = "6377a4a321c4725702d34dcc7293a8bc21fb85562ce95343c41cd69016bcaf62",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.35.2-72.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6377a4a321c4725702d34dcc7293a8bc21fb85562ce95343c41cd69016bcaf62",
    ],
)

rpm(
    name = "binutils-gold-0__2.41-65.el10.aarch64",
    sha256 = "ed7edaaad448f9674079934d5614eb6244bc86f3e21dd3d21314d60e5e63a731",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/binutils-gold-2.41-65.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ed7edaaad448f9674079934d5614eb6244bc86f3e21dd3d21314d60e5e63a731",
    ],
)

rpm(
    name = "binutils-gold-0__2.41-65.el10.s390x",
    sha256 = "bf9e62c6226edec75677293dc9d05c6329c01e27b205da53673d124ca4152b68",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/binutils-gold-2.41-65.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bf9e62c6226edec75677293dc9d05c6329c01e27b205da53673d124ca4152b68",
    ],
)

rpm(
    name = "binutils-gold-0__2.41-65.el10.x86_64",
    sha256 = "23077a4e89aed59b3981d49c8df69a9f364b8e811290f769cdb6776fd0f98073",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/binutils-gold-2.41-65.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23077a4e89aed59b3981d49c8df69a9f364b8e811290f769cdb6776fd0f98073",
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
        "https://storage.googleapis.com/builddeps/30fd7d37e3f06d0b06b6f3e6fda58fd9d54582b0e497795719d81dc68ac88ba7",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-25.el10.s390x",
    sha256 = "c9208b97a6a3e2cb7fc84a7bea4e330399cc6ef892c3f0abe20d5df10797eade",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/bzip2-1.0.8-25.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c9208b97a6a3e2cb7fc84a7bea4e330399cc6ef892c3f0abe20d5df10797eade",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-25.el10.x86_64",
    sha256 = "ff7f8e9c3cc936d35033ec40545ee4a836db27c30c240d3aa39be4c8b0fda631",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.8-25.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff7f8e9c3cc936d35033ec40545ee4a836db27c30c240d3aa39be4c8b0fda631",
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
        "https://storage.googleapis.com/builddeps/ac836c2c133077d0e71092f2c21e69d3985ace8458af527440e13b7edf165beb",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-25.el10.s390x",
    sha256 = "219adea56b92ecf22cb63fad38638e16115df270b78ea1fbd3cc1b183caf69a4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/bzip2-libs-1.0.8-25.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/219adea56b92ecf22cb63fad38638e16115df270b78ea1fbd3cc1b183caf69a4",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-25.el10.x86_64",
    sha256 = "309c7dbb857254655c51c4ab02d8038137c1363058542d8701c9272609f5b433",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.8-25.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/309c7dbb857254655c51c4ab02d8038137c1363058542d8701c9272609f5b433",
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
        "https://storage.googleapis.com/builddeps/a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    ],
)

rpm(
    name = "ca-certificates-0__2025.2.80_v9.0.305-102.el10.s390x",
    sha256 = "a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/ca-certificates-2025.2.80_v9.0.305-102.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    ],
)

rpm(
    name = "ca-certificates-0__2025.2.80_v9.0.305-102.el10.x86_64",
    sha256 = "a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/ca-certificates-2025.2.80_v9.0.305-102.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a5a8cf95b7cae489df2f6b4448b6d5100593256b0033376d25b2705985fad9dc",
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
    name = "capstone-0__4.0.2-10.el9.x86_64",
    sha256 = "f6a9fdc6bcb5da1b2ce44ca7ed6289759c37add7adbb19916dd36d5bb4624a41",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/capstone-4.0.2-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f6a9fdc6bcb5da1b2ce44ca7ed6289759c37add7adbb19916dd36d5bb4624a41",
    ],
)

rpm(
    name = "capstone-0__4.0.2-12.el9.aarch64",
    sha256 = "d14f5d381bb865c5b83a34bf30c1eecb6be136a2f7c41230f58a9a2aba4237c7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/capstone-4.0.2-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d14f5d381bb865c5b83a34bf30c1eecb6be136a2f7c41230f58a9a2aba4237c7",
    ],
)

rpm(
    name = "capstone-0__4.0.2-12.el9.s390x",
    sha256 = "05270815c439fc33e0ea250b6008c5f1ba0c1ff5a0905b3508e98e59e02458ba",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/capstone-4.0.2-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/05270815c439fc33e0ea250b6008c5f1ba0c1ff5a0905b3508e98e59e02458ba",
    ],
)

rpm(
    name = "capstone-0__4.0.2-12.el9.x86_64",
    sha256 = "0fb298dd8d35902a9176ada92bc7254bc7dac4fca217a692f133ecb7bb48a166",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/capstone-4.0.2-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0fb298dd8d35902a9176ada92bc7254bc7dac4fca217a692f133ecb7bb48a166",
    ],
)

rpm(
    name = "capstone-0__5.0.1-8.el10.aarch64",
    sha256 = "253c4dbe9b3095d10c425d206f908ee373b46cae1f9af21e7962739f69d36751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/capstone-5.0.1-8.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/253c4dbe9b3095d10c425d206f908ee373b46cae1f9af21e7962739f69d36751",
    ],
)

rpm(
    name = "capstone-0__5.0.1-8.el10.s390x",
    sha256 = "84687f2a15f9b18d800a9ae3b77423a6361210f640bdf4474343926e099f5cf1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/capstone-5.0.1-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/84687f2a15f9b18d800a9ae3b77423a6361210f640bdf4474343926e099f5cf1",
    ],
)

rpm(
    name = "capstone-0__5.0.1-8.el10.x86_64",
    sha256 = "279bbf70ea7cf2d4d65ce34c926d569feaaa903c268179397830e9bbd7698d87",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/capstone-5.0.1-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/279bbf70ea7cf2d4d65ce34c926d569feaaa903c268179397830e9bbd7698d87",
    ],
)

rpm(
    name = "centos-gpg-keys-0__10.0-23.el10.aarch64",
    sha256 = "f223143fc00d956376d9f6361a538037e738a469ad79d4d82555e99a307c510f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f223143fc00d956376d9f6361a538037e738a469ad79d4d82555e99a307c510f",
    ],
)

rpm(
    name = "centos-gpg-keys-0__10.0-23.el10.s390x",
    sha256 = "f223143fc00d956376d9f6361a538037e738a469ad79d4d82555e99a307c510f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/centos-gpg-keys-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f223143fc00d956376d9f6361a538037e738a469ad79d4d82555e99a307c510f",
    ],
)

rpm(
    name = "centos-gpg-keys-0__10.0-23.el10.x86_64",
    sha256 = "f223143fc00d956376d9f6361a538037e738a469ad79d4d82555e99a307c510f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f223143fc00d956376d9f6361a538037e738a469ad79d4d82555e99a307c510f",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-26.el9.x86_64",
    sha256 = "8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca",
    urls = ["https://storage.googleapis.com/builddeps/8d601d9f96356a200ad6ed8e5cb49bbac4aa3c4b762d10a23e11311daa5711ca"],
)

rpm(
    name = "centos-gpg-keys-0__9.0-38.el9.aarch64",
    sha256 = "b6dcd5a16160ab017bf5e871975aef477d34ce61b660eeb3f4aa6973dcc6f916",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b6dcd5a16160ab017bf5e871975aef477d34ce61b660eeb3f4aa6973dcc6f916",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-38.el9.s390x",
    sha256 = "b6dcd5a16160ab017bf5e871975aef477d34ce61b660eeb3f4aa6973dcc6f916",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-gpg-keys-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b6dcd5a16160ab017bf5e871975aef477d34ce61b660eeb3f4aa6973dcc6f916",
    ],
)

rpm(
    name = "centos-gpg-keys-0__9.0-38.el9.x86_64",
    sha256 = "b6dcd5a16160ab017bf5e871975aef477d34ce61b660eeb3f4aa6973dcc6f916",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b6dcd5a16160ab017bf5e871975aef477d34ce61b660eeb3f4aa6973dcc6f916",
    ],
)

rpm(
    name = "centos-stream-release-0__10.0-23.el10.aarch64",
    sha256 = "3c0ca8ad94e7b4dfdbe3a0c6ed088c560c57815779cdee33b00b8980894d760f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/centos-stream-release-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c0ca8ad94e7b4dfdbe3a0c6ed088c560c57815779cdee33b00b8980894d760f",
    ],
)

rpm(
    name = "centos-stream-release-0__10.0-23.el10.s390x",
    sha256 = "3c0ca8ad94e7b4dfdbe3a0c6ed088c560c57815779cdee33b00b8980894d760f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/centos-stream-release-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c0ca8ad94e7b4dfdbe3a0c6ed088c560c57815779cdee33b00b8980894d760f",
    ],
)

rpm(
    name = "centos-stream-release-0__10.0-23.el10.x86_64",
    sha256 = "3c0ca8ad94e7b4dfdbe3a0c6ed088c560c57815779cdee33b00b8980894d760f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/centos-stream-release-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c0ca8ad94e7b4dfdbe3a0c6ed088c560c57815779cdee33b00b8980894d760f",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-26.el9.x86_64",
    sha256 = "3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3",
    urls = ["https://storage.googleapis.com/builddeps/3d60dc8ed86717f68394fc7468b8024557c43ac2ad97b8e40911d056cd6d64d3"],
)

rpm(
    name = "centos-stream-release-0__9.0-38.el9.aarch64",
    sha256 = "04a24747b2884f59d8ac5583f162dbc5ed043f5ccf602ef6274349f1d7ca9a8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-release-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/04a24747b2884f59d8ac5583f162dbc5ed043f5ccf602ef6274349f1d7ca9a8e",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-38.el9.s390x",
    sha256 = "04a24747b2884f59d8ac5583f162dbc5ed043f5ccf602ef6274349f1d7ca9a8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-release-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/04a24747b2884f59d8ac5583f162dbc5ed043f5ccf602ef6274349f1d7ca9a8e",
    ],
)

rpm(
    name = "centos-stream-release-0__9.0-38.el9.x86_64",
    sha256 = "04a24747b2884f59d8ac5583f162dbc5ed043f5ccf602ef6274349f1d7ca9a8e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-release-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/04a24747b2884f59d8ac5583f162dbc5ed043f5ccf602ef6274349f1d7ca9a8e",
    ],
)

rpm(
    name = "centos-stream-repos-0__10.0-23.el10.aarch64",
    sha256 = "f08be10e29ff190f941fad1335dd0528f264ee1afaf1836758609a554c028d00",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f08be10e29ff190f941fad1335dd0528f264ee1afaf1836758609a554c028d00",
    ],
)

rpm(
    name = "centos-stream-repos-0__10.0-23.el10.s390x",
    sha256 = "f08be10e29ff190f941fad1335dd0528f264ee1afaf1836758609a554c028d00",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/centos-stream-repos-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f08be10e29ff190f941fad1335dd0528f264ee1afaf1836758609a554c028d00",
    ],
)

rpm(
    name = "centos-stream-repos-0__10.0-23.el10.x86_64",
    sha256 = "f08be10e29ff190f941fad1335dd0528f264ee1afaf1836758609a554c028d00",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-10.0-23.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f08be10e29ff190f941fad1335dd0528f264ee1afaf1836758609a554c028d00",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-26.el9.x86_64",
    sha256 = "eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035",
    urls = ["https://storage.googleapis.com/builddeps/eb3b55a5cf0e1a93a91cd2d39035bd1754b46f69ff3d062b3331e765b2345035"],
)

rpm(
    name = "centos-stream-repos-0__9.0-38.el9.aarch64",
    sha256 = "ae98322c35b3eb7b013c8989a8b318993e3bec71018e213f729a156c2bbd508d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ae98322c35b3eb7b013c8989a8b318993e3bec71018e213f729a156c2bbd508d",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-38.el9.s390x",
    sha256 = "ae98322c35b3eb7b013c8989a8b318993e3bec71018e213f729a156c2bbd508d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/centos-stream-repos-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ae98322c35b3eb7b013c8989a8b318993e3bec71018e213f729a156c2bbd508d",
    ],
)

rpm(
    name = "centos-stream-repos-0__9.0-38.el9.x86_64",
    sha256 = "ae98322c35b3eb7b013c8989a8b318993e3bec71018e213f729a156c2bbd508d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-9.0-38.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ae98322c35b3eb7b013c8989a8b318993e3bec71018e213f729a156c2bbd508d",
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
    name = "coreutils-single-0__8.32-43.el9.aarch64",
    sha256 = "b0a8cc4a393db9851ce5efc9fe58557a71ed9d0d92a4619ae31d951b6db2dcff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.32-43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b0a8cc4a393db9851ce5efc9fe58557a71ed9d0d92a4619ae31d951b6db2dcff",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-43.el9.s390x",
    sha256 = "2032300685afbeacc818cfd43a1e9c1ae991e4d0674a9dd0475a5ffcd7ceba16",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/coreutils-single-8.32-43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2032300685afbeacc818cfd43a1e9c1ae991e4d0674a9dd0475a5ffcd7ceba16",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-43.el9.x86_64",
    sha256 = "7cb048fc00e364bfbc1c12f26b9b39a062e426b0e45cfb4421ccf5493ec8a2c8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.32-43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7cb048fc00e364bfbc1c12f26b9b39a062e426b0e45cfb4421ccf5493ec8a2c8",
    ],
)

rpm(
    name = "coreutils-single-0__9.5-12.el10.aarch64",
    sha256 = "791e3316ebe21ad54ec241f2e3a19d331175931f094dc24658920077894a6209",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/coreutils-single-9.5-12.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/791e3316ebe21ad54ec241f2e3a19d331175931f094dc24658920077894a6209",
    ],
)

rpm(
    name = "coreutils-single-0__9.5-12.el10.s390x",
    sha256 = "039b2a725e48ab4d9b59c51b6edb150db5e732a2afbdf7d453b23b06cae52b37",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/coreutils-single-9.5-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/039b2a725e48ab4d9b59c51b6edb150db5e732a2afbdf7d453b23b06cae52b37",
    ],
)

rpm(
    name = "coreutils-single-0__9.5-12.el10.x86_64",
    sha256 = "92c85bde455c4760b21d47af28d5ebf78cb09598f71248924099c2afef9d819a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/coreutils-single-9.5-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/92c85bde455c4760b21d47af28d5ebf78cb09598f71248924099c2afef9d819a",
    ],
)

rpm(
    name = "cpp-0__11.5.0-15.el9.aarch64",
    sha256 = "1852f13d050f440b08d920035fd13d51ac8876a2aa6a15ef26add0665fdb6e93",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/cpp-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1852f13d050f440b08d920035fd13d51ac8876a2aa6a15ef26add0665fdb6e93",
    ],
)

rpm(
    name = "cpp-0__11.5.0-15.el9.s390x",
    sha256 = "0cbc70b5f6989aa304b8d334a778cae05cfa0bc19405b85a09679aba369363d0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/cpp-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0cbc70b5f6989aa304b8d334a778cae05cfa0bc19405b85a09679aba369363d0",
    ],
)

rpm(
    name = "cpp-0__11.5.0-15.el9.x86_64",
    sha256 = "1c1e4c8785b647ea94981f83c20ffe23d660e6a2b6ba88841a180dd1de102102",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/cpp-11.5.0-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1c1e4c8785b647ea94981f83c20ffe23d660e6a2b6ba88841a180dd1de102102",
    ],
)

rpm(
    name = "cpp-0__14.3.1-4.4.el10.aarch64",
    sha256 = "642cd24d9944392b6d55e2a32b137a0a4f31857a11b6e1c1c423bf66061b4864",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/cpp-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/642cd24d9944392b6d55e2a32b137a0a4f31857a11b6e1c1c423bf66061b4864",
    ],
)

rpm(
    name = "cpp-0__14.3.1-4.4.el10.s390x",
    sha256 = "29ee0eb088163e4b0ffa4166141bb2e40db37742870d57a96e461c3a2ad42fdb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/cpp-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29ee0eb088163e4b0ffa4166141bb2e40db37742870d57a96e461c3a2ad42fdb",
    ],
)

rpm(
    name = "cpp-0__14.3.1-4.4.el10.x86_64",
    sha256 = "7668973ca6c7706025e30e4b557eb1ac8b55d05bda1ba3d8410d0b502ebfbf72",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/cpp-14.3.1-4.4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7668973ca6c7706025e30e4b557eb1ac8b55d05bda1ba3d8410d0b502ebfbf72",
    ],
)

rpm(
    name = "cracklib-0__2.9.11-8.el10.aarch64",
    sha256 = "04112224e2f1b7027ef15ee4cb9ede5bb89426b29f150692778d8f7ca155eea9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/cracklib-2.9.11-8.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/04112224e2f1b7027ef15ee4cb9ede5bb89426b29f150692778d8f7ca155eea9",
    ],
)

rpm(
    name = "cracklib-0__2.9.11-8.el10.s390x",
    sha256 = "2e0c0ba830f1a497461b1a7f6e76f5d409c9bf87d2c4a6874957abe3fdb74be3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cracklib-2.9.11-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2e0c0ba830f1a497461b1a7f6e76f5d409c9bf87d2c4a6874957abe3fdb74be3",
    ],
)

rpm(
    name = "cracklib-0__2.9.11-8.el10.x86_64",
    sha256 = "4d648a415fe67550a22ff50befdaf9a33ccb55dbc9a2e3d4121ddfbe2ee843f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cracklib-2.9.11-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d648a415fe67550a22ff50befdaf9a33ccb55dbc9a2e3d4121ddfbe2ee843f7",
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
        "https://storage.googleapis.com/builddeps/51210426186039c77239cbb3c710acbc9f7778ca44292204ffa2ecf1448e2c1e",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.11-8.el10.s390x",
    sha256 = "45cf94fabce8c9c035df7db91b19fefec5cfef5cee54505cabebce1822e3099d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cracklib-dicts-2.9.11-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/45cf94fabce8c9c035df7db91b19fefec5cfef5cee54505cabebce1822e3099d",
    ],
)

rpm(
    name = "cracklib-dicts-0__2.9.11-8.el10.x86_64",
    sha256 = "79dd2684b0ae0cbc47739c0e292f17243eb448b92f74bae893cf1eb4aba14703",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cracklib-dicts-2.9.11-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/79dd2684b0ae0cbc47739c0e292f17243eb448b92f74bae893cf1eb4aba14703",
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
    name = "crypto-policies-0__20260525-1.gitf5f5370.el10.aarch64",
    sha256 = "0327b92908d19206074563c489ed11472a59e1abec3d724282d0d91025287553",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/crypto-policies-20260525-1.gitf5f5370.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0327b92908d19206074563c489ed11472a59e1abec3d724282d0d91025287553",
    ],
)

rpm(
    name = "crypto-policies-0__20260525-1.gitf5f5370.el10.s390x",
    sha256 = "0327b92908d19206074563c489ed11472a59e1abec3d724282d0d91025287553",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/crypto-policies-20260525-1.gitf5f5370.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0327b92908d19206074563c489ed11472a59e1abec3d724282d0d91025287553",
    ],
)

rpm(
    name = "crypto-policies-0__20260525-1.gitf5f5370.el10.x86_64",
    sha256 = "0327b92908d19206074563c489ed11472a59e1abec3d724282d0d91025287553",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/crypto-policies-20260525-1.gitf5f5370.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0327b92908d19206074563c489ed11472a59e1abec3d724282d0d91025287553",
    ],
)

rpm(
    name = "crypto-policies-0__20260610-1.git0798a9f.el9.aarch64",
    sha256 = "d48356afbf6145460f785753b59a2ead2e343c740b77c97d54022c5dd1b2c154",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/crypto-policies-20260610-1.git0798a9f.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d48356afbf6145460f785753b59a2ead2e343c740b77c97d54022c5dd1b2c154",
    ],
)

rpm(
    name = "crypto-policies-0__20260610-1.git0798a9f.el9.s390x",
    sha256 = "d48356afbf6145460f785753b59a2ead2e343c740b77c97d54022c5dd1b2c154",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/crypto-policies-20260610-1.git0798a9f.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d48356afbf6145460f785753b59a2ead2e343c740b77c97d54022c5dd1b2c154",
    ],
)

rpm(
    name = "crypto-policies-0__20260610-1.git0798a9f.el9.x86_64",
    sha256 = "d48356afbf6145460f785753b59a2ead2e343c740b77c97d54022c5dd1b2c154",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/crypto-policies-20260610-1.git0798a9f.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d48356afbf6145460f785753b59a2ead2e343c740b77c97d54022c5dd1b2c154",
    ],
)

rpm(
    name = "curl-0__8.12.1-6.el10.aarch64",
    sha256 = "14e6e2fda7a9c16c792133ac5544e7b5047ac373baf0333040d73891e95ce851",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/curl-8.12.1-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/14e6e2fda7a9c16c792133ac5544e7b5047ac373baf0333040d73891e95ce851",
    ],
)

rpm(
    name = "curl-0__8.12.1-6.el10.s390x",
    sha256 = "94daebf335228f60880e99a641b504018ba762d4b1d0c0e239862dfba2f69962",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/curl-8.12.1-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/94daebf335228f60880e99a641b504018ba762d4b1d0c0e239862dfba2f69962",
    ],
)

rpm(
    name = "curl-0__8.12.1-6.el10.x86_64",
    sha256 = "4727cb56888d3dc433a70e836b45c1cc61075b5987583166e8c7ec07a5af523f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/curl-8.12.1-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4727cb56888d3dc433a70e836b45c1cc61075b5987583166e8c7ec07a5af523f",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-31.el9.x86_64",
    sha256 = "be145eb1684cb38553b6611bca6c0fb562ff8485902c49131c5ed0b9ac0733f4",
    urls = ["https://storage.googleapis.com/builddeps/be145eb1684cb38553b6611bca6c0fb562ff8485902c49131c5ed0b9ac0733f4"],
)

rpm(
    name = "curl-minimal-0__7.76.1-43.el9.aarch64",
    sha256 = "e52f5971cbd6079130b1c6ef4ed35269b7b280a4e63ed1a3e7188ff8f15d610d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/curl-minimal-7.76.1-43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e52f5971cbd6079130b1c6ef4ed35269b7b280a4e63ed1a3e7188ff8f15d610d",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-43.el9.s390x",
    sha256 = "f6c4616125d9d5e98b7fcb50551d555f8ed7be5c141e2676057b3f8f8dd4e106",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/curl-minimal-7.76.1-43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f6c4616125d9d5e98b7fcb50551d555f8ed7be5c141e2676057b3f8f8dd4e106",
    ],
)

rpm(
    name = "curl-minimal-0__7.76.1-43.el9.x86_64",
    sha256 = "0ddf97bb566dd3c6c877b2a2fb895b252d13982a12f1cd407c9ca21a53ad0777",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/curl-minimal-7.76.1-43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0ddf97bb566dd3c6c877b2a2fb895b252d13982a12f1cd407c9ca21a53ad0777",
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
        "https://storage.googleapis.com/builddeps/f030977f59727e389143e1813c5fc848799abbea48ed60aca460dc2eb1a79637",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.28-27.el10.s390x",
    sha256 = "28c75a50cf3f092920ac56fb65805e9c875fc95d4e76bce0e1cc6b6d21e3fba3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cyrus-sasl-gssapi-2.1.28-27.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/28c75a50cf3f092920ac56fb65805e9c875fc95d4e76bce0e1cc6b6d21e3fba3",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.28-27.el10.x86_64",
    sha256 = "f9ab02ca832fe4d5c1e1ee3abd7ff3db3815d164561350316032a82b44d68b6c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-gssapi-2.1.28-27.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f9ab02ca832fe4d5c1e1ee3abd7ff3db3815d164561350316032a82b44d68b6c",
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
        "https://storage.googleapis.com/builddeps/917d6b8d2eff0dd71b55646c758b938ac7b9f0a298f2dffae5948c9865215067",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.28-27.el10.s390x",
    sha256 = "b40557a0d21461db27adf093fe6a72ec17a243f6743a3d1e26c32601753e97ee",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/cyrus-sasl-lib-2.1.28-27.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b40557a0d21461db27adf093fe6a72ec17a243f6743a3d1e26c32601753e97ee",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.28-27.el10.x86_64",
    sha256 = "ea78a83980b03f3709266f5e4c96b41699fe8d5f7003fb9503c3a7529c6ca46a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-lib-2.1.28-27.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ea78a83980b03f3709266f5e4c96b41699fe8d5f7003fb9503c3a7529c6ca46a",
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
    name = "dbus-1__1.12.20-9.el9.aarch64",
    sha256 = "f2f2f80cf9c11b7f4e1c27ba65a416b1dad9a48c2991ed1cb77c038a62319754",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-1.12.20-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f2f2f80cf9c11b7f4e1c27ba65a416b1dad9a48c2991ed1cb77c038a62319754",
    ],
)

rpm(
    name = "dbus-1__1.12.20-9.el9.s390x",
    sha256 = "62f819b14f1fec3a9eeb91b6367ba8b1ff464875414477157d61ca04da3aeede",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-1.12.20-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/62f819b14f1fec3a9eeb91b6367ba8b1ff464875414477157d61ca04da3aeede",
    ],
)

rpm(
    name = "dbus-1__1.12.20-9.el9.x86_64",
    sha256 = "9e0a4fc4da86a68b0366601580a9b2af73901440b85219370f60d773c344cc7c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-1.12.20-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9e0a4fc4da86a68b0366601580a9b2af73901440b85219370f60d773c344cc7c",
    ],
)

rpm(
    name = "dbus-1__1.14.10-5.el10.aarch64",
    sha256 = "2f00025969ff8b32c254ec38919908120f83847e98285413c718d1ad0b2a8766",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-1.14.10-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2f00025969ff8b32c254ec38919908120f83847e98285413c718d1ad0b2a8766",
    ],
)

rpm(
    name = "dbus-1__1.14.10-5.el10.s390x",
    sha256 = "2a746bab9a5c03b6bc2f680ad3be8ecf935404c17f6488de44e77ab61bdfedb8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-1.14.10-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2a746bab9a5c03b6bc2f680ad3be8ecf935404c17f6488de44e77ab61bdfedb8",
    ],
)

rpm(
    name = "dbus-1__1.14.10-5.el10.x86_64",
    sha256 = "c71f38667ecebd3ba0adf415ccf181209330bb0e2ca9ad0bf4de9828b370b9e4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-1.14.10-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c71f38667ecebd3ba0adf415ccf181209330bb0e2ca9ad0bf4de9828b370b9e4",
    ],
)

rpm(
    name = "dbus-broker-0__28-9.el9.aarch64",
    sha256 = "724c1f8142deda976f1b5c9a714e4e49864e61544b4ff507fc5078d416db6acc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-broker-28-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/724c1f8142deda976f1b5c9a714e4e49864e61544b4ff507fc5078d416db6acc",
    ],
)

rpm(
    name = "dbus-broker-0__28-9.el9.s390x",
    sha256 = "83140ba4112cd7b62ebd9673d74af5d11153428652eacf84a2b254f4597b174e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-broker-28-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/83140ba4112cd7b62ebd9673d74af5d11153428652eacf84a2b254f4597b174e",
    ],
)

rpm(
    name = "dbus-broker-0__28-9.el9.x86_64",
    sha256 = "2b0215cb658182a774d7eb1e965c12d0512fddb1be983d8da746620dc183d0c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-broker-28-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2b0215cb658182a774d7eb1e965c12d0512fddb1be983d8da746620dc183d0c2",
    ],
)

rpm(
    name = "dbus-broker-0__36-4.el10.aarch64",
    sha256 = "3716b1d4daa23c6fd965175473464ddfa91ea5651a68298a2e0b139021e23035",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-broker-36-4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3716b1d4daa23c6fd965175473464ddfa91ea5651a68298a2e0b139021e23035",
    ],
)

rpm(
    name = "dbus-broker-0__36-4.el10.s390x",
    sha256 = "3d1ec31218c8925602bb7fcd88150c628a0e24ab5cc4e7c63b85785202756283",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-broker-36-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3d1ec31218c8925602bb7fcd88150c628a0e24ab5cc4e7c63b85785202756283",
    ],
)

rpm(
    name = "dbus-broker-0__36-4.el10.x86_64",
    sha256 = "a0778052571fe74351500a06e765219fcf53c0ca2eeb4969a2682a36ee9f9c10",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-broker-36-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0778052571fe74351500a06e765219fcf53c0ca2eeb4969a2682a36ee9f9c10",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-9.el9.aarch64",
    sha256 = "c9e2580b234cf5591cdecd5472ae14b7886392dcf4e91d63751f18b320e7694b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.20-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c9e2580b234cf5591cdecd5472ae14b7886392dcf4e91d63751f18b320e7694b",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-9.el9.s390x",
    sha256 = "c9e2580b234cf5591cdecd5472ae14b7886392dcf4e91d63751f18b320e7694b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-common-1.12.20-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c9e2580b234cf5591cdecd5472ae14b7886392dcf4e91d63751f18b320e7694b",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-9.el9.x86_64",
    sha256 = "c9e2580b234cf5591cdecd5472ae14b7886392dcf4e91d63751f18b320e7694b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.20-9.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c9e2580b234cf5591cdecd5472ae14b7886392dcf4e91d63751f18b320e7694b",
    ],
)

rpm(
    name = "dbus-common-1__1.14.10-5.el10.aarch64",
    sha256 = "1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-common-1.14.10-5.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    ],
)

rpm(
    name = "dbus-common-1__1.14.10-5.el10.s390x",
    sha256 = "1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-common-1.14.10-5.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    ],
)

rpm(
    name = "dbus-common-1__1.14.10-5.el10.x86_64",
    sha256 = "1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-common-1.14.10-5.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1cf5e00ed550daa874c5ec81be43f4606717a2465d72b733d3b9012015dfa751",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-9.el9.aarch64",
    sha256 = "bef5e394cc943eb6cc1e3dd15e47cf0eb7b6f9acb33fbb08ba3263076f61517f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.12.20-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bef5e394cc943eb6cc1e3dd15e47cf0eb7b6f9acb33fbb08ba3263076f61517f",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-9.el9.s390x",
    sha256 = "9c6a1e3eb8f67f0bb2a949c19c8725f7e87b91454008bc407fd7ea90f1eea82d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/dbus-libs-1.12.20-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9c6a1e3eb8f67f0bb2a949c19c8725f7e87b91454008bc407fd7ea90f1eea82d",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-9.el9.x86_64",
    sha256 = "04bd4d47f5e6dd97d1f68a8ae66ef9b850c7347cefca5c9ec09b015c36d894a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.20-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/04bd4d47f5e6dd97d1f68a8ae66ef9b850c7347cefca5c9ec09b015c36d894a2",
    ],
)

rpm(
    name = "dbus-libs-1__1.14.10-5.el10.aarch64",
    sha256 = "976a662683dc4f8235303cd6065f589c4d4728671116827b2002ac1fd4a74a72",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.14.10-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/976a662683dc4f8235303cd6065f589c4d4728671116827b2002ac1fd4a74a72",
    ],
)

rpm(
    name = "dbus-libs-1__1.14.10-5.el10.s390x",
    sha256 = "261a5aee8fd8417bdb0b629b7ae4141cec92de79d32b45982c66cc82878f3175",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/dbus-libs-1.14.10-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/261a5aee8fd8417bdb0b629b7ae4141cec92de79d32b45982c66cc82878f3175",
    ],
)

rpm(
    name = "dbus-libs-1__1.14.10-5.el10.x86_64",
    sha256 = "7cd5d99568a89ef7100ae60d44aa270cbf5882e95cbc8f43497696f81c664284",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.14.10-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7cd5d99568a89ef7100ae60d44aa270cbf5882e95cbc8f43497696f81c664284",
    ],
)

rpm(
    name = "device-mapper-10__1.02.215-2.el10.aarch64",
    sha256 = "4dbbdff2b5f6dfd3bcd717e4168bbd2e9c7bc4ad67d2c04f18e85c12f3f6707c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.215-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4dbbdff2b5f6dfd3bcd717e4168bbd2e9c7bc4ad67d2c04f18e85c12f3f6707c",
    ],
)

rpm(
    name = "device-mapper-10__1.02.215-2.el10.s390x",
    sha256 = "4d6a93ad1d45f2483c0482753f9d314f22527b7b73455b438c556b5970f4be31",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/device-mapper-1.02.215-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4d6a93ad1d45f2483c0482753f9d314f22527b7b73455b438c556b5970f4be31",
    ],
)

rpm(
    name = "device-mapper-10__1.02.215-2.el10.x86_64",
    sha256 = "67b8f40730488c8fc20da510ee281dc1831283aaf0da2754e869366797803502",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.215-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/67b8f40730488c8fc20da510ee281dc1831283aaf0da2754e869366797803502",
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
    name = "device-mapper-9__1.02.207-6.el9.aarch64",
    sha256 = "c6631f4b5aa5c4e9d758f643cc259823b24d2d3c8af9d3343448d3ca2fe3a8e7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.207-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c6631f4b5aa5c4e9d758f643cc259823b24d2d3c8af9d3343448d3ca2fe3a8e7",
    ],
)

rpm(
    name = "device-mapper-9__1.02.207-6.el9.s390x",
    sha256 = "46cce0d6b50c559a43214f8faa2c0c7de7669626c3412077968fa5f4e2d27319",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-1.02.207-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/46cce0d6b50c559a43214f8faa2c0c7de7669626c3412077968fa5f4e2d27319",
    ],
)

rpm(
    name = "device-mapper-9__1.02.207-6.el9.x86_64",
    sha256 = "ffedc788a02c0bd900e38fbd755a7d2911e4dc4f58c7359db26c31772b0f4563",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.207-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffedc788a02c0bd900e38fbd755a7d2911e4dc4f58c7359db26c31772b0f4563",
    ],
)

rpm(
    name = "device-mapper-libs-10__1.02.215-2.el10.aarch64",
    sha256 = "e28a0c18b33f7ffd6d5b35e0864a7d18fab50b8a29f4936f396ee2ebf9b5af95",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.215-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e28a0c18b33f7ffd6d5b35e0864a7d18fab50b8a29f4936f396ee2ebf9b5af95",
    ],
)

rpm(
    name = "device-mapper-libs-10__1.02.215-2.el10.s390x",
    sha256 = "f26254d06ad26fa28020b2d2b57c0f315742459cd0b64fafdb00e04f6b81af15",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/device-mapper-libs-1.02.215-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f26254d06ad26fa28020b2d2b57c0f315742459cd0b64fafdb00e04f6b81af15",
    ],
)

rpm(
    name = "device-mapper-libs-10__1.02.215-2.el10.x86_64",
    sha256 = "81a9a4a5aca29c225db36f448b7a9344ed193a49aa11ea5d23c5701640e60557",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.215-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/81a9a4a5aca29c225db36f448b7a9344ed193a49aa11ea5d23c5701640e60557",
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
    name = "device-mapper-libs-9__1.02.207-6.el9.aarch64",
    sha256 = "afb5d271de7b817f416f248e3e219ebaae5ab79e409cb16ac01f7523b61a6cfd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.207-6.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/afb5d271de7b817f416f248e3e219ebaae5ab79e409cb16ac01f7523b61a6cfd",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.207-6.el9.s390x",
    sha256 = "29ddf28275ea3140ca05ab9dce9016dce36ef87564971717bfa5242e9f2d9c92",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/device-mapper-libs-1.02.207-6.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29ddf28275ea3140ca05ab9dce9016dce36ef87564971717bfa5242e9f2d9c92",
    ],
)

rpm(
    name = "device-mapper-libs-9__1.02.207-6.el9.x86_64",
    sha256 = "a1ed5a79d3d61cfdfa44f1d4f71a38da0e56f6bb3826d6c86a6131e7ebd0c8db",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.207-6.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a1ed5a79d3d61cfdfa44f1d4f71a38da0e56f6bb3826d6c86a6131e7ebd0c8db",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-47.el9.aarch64",
    sha256 = "14439600a76dd7083ffebd7417d0f993a83886f73b36e0b5893845cda2e1d414",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.7-47.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/14439600a76dd7083ffebd7417d0f993a83886f73b36e0b5893845cda2e1d414",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.7-47.el9.x86_64",
    sha256 = "c1ccd00ddb56ff3823a2ae1f6471133a58c48c70e16a697bea55bbe69c9c485e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.7-47.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c1ccd00ddb56ff3823a2ae1f6471133a58c48c70e16a697bea55bbe69c9c485e",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.9.9-19.el10.aarch64",
    sha256 = "30b2021d8fb814a0f665e11965025fed8837be948f5ac86b61c13632fed82e2f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.9.9-19.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/30b2021d8fb814a0f665e11965025fed8837be948f5ac86b61c13632fed82e2f",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.9.9-19.el10.x86_64",
    sha256 = "c6c1884b8f49236f591c30d7ba112575a527792cf84350cd9d80ecd7aafea56a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.9.9-19.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c6c1884b8f49236f591c30d7ba112575a527792cf84350cd9d80ecd7aafea56a",
    ],
)

rpm(
    name = "diffutils-0__3.10-8.el10.aarch64",
    sha256 = "d06031d2cd612618343d29186bc873cafd52c9e71efae6d04dcb494de2b53b58",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/diffutils-3.10-8.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d06031d2cd612618343d29186bc873cafd52c9e71efae6d04dcb494de2b53b58",
    ],
)

rpm(
    name = "diffutils-0__3.10-8.el10.s390x",
    sha256 = "4668ee01492723f3a4fd094ff49ef2485ab3f17d1e30b19103a70e4b24a7c3e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/diffutils-3.10-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4668ee01492723f3a4fd094ff49ef2485ab3f17d1e30b19103a70e4b24a7c3e2",
    ],
)

rpm(
    name = "diffutils-0__3.10-8.el10.x86_64",
    sha256 = "96882ec03cfc01ae557f0ec547fb8d346179eb705c899bec0533eafda7c1bd80",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/diffutils-3.10-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/96882ec03cfc01ae557f0ec547fb8d346179eb705c899bec0533eafda7c1bd80",
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
        "https://storage.googleapis.com/builddeps/381d5765cc5b1346f47dea4818c013bc308eb2cd9a76a9a3c4046a6982910956",
    ],
)

rpm(
    name = "dmidecode-1__3.6-5.el10.x86_64",
    sha256 = "332cfc77ea06aab27c93c1cf2382e50bf62ddad534c526795083a98ec10668c8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/dmidecode-3.6-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/332cfc77ea06aab27c93c1cf2382e50bf62ddad534c526795083a98ec10668c8",
    ],
)

rpm(
    name = "duktape-0__2.7.0-10.el10.aarch64",
    sha256 = "c390a43273231fec4a25199690e0106268e3eb46a1592d4cd68cf56909efce5e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/duktape-2.7.0-10.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c390a43273231fec4a25199690e0106268e3eb46a1592d4cd68cf56909efce5e",
    ],
)

rpm(
    name = "duktape-0__2.7.0-10.el10.s390x",
    sha256 = "7cafae00eb1aa432b96c9fb9a6df9789d3ccf03515b7714c16ff8dcbaa7210d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/duktape-2.7.0-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7cafae00eb1aa432b96c9fb9a6df9789d3ccf03515b7714c16ff8dcbaa7210d6",
    ],
)

rpm(
    name = "duktape-0__2.7.0-10.el10.x86_64",
    sha256 = "23b7d2905723ed7adabe3362c54d54f0745c908029ec3be79bd881770d2c591a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/duktape-2.7.0-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23b7d2905723ed7adabe3362c54d54f0745c908029ec3be79bd881770d2c591a",
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
        "https://storage.googleapis.com/builddeps/fd5592fb0e7c1ae9ae023eafb55c7ae3ac71c94c44e1f498f1eb56c1940f3c40",
    ],
)

rpm(
    name = "e2fsprogs-0__1.47.1-5.el10.s390x",
    sha256 = "23803262e02ed5ad895284267c828bee4620aa498326a36c659a36dcd12bce9e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/e2fsprogs-1.47.1-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/23803262e02ed5ad895284267c828bee4620aa498326a36c659a36dcd12bce9e",
    ],
)

rpm(
    name = "e2fsprogs-0__1.47.1-5.el10.x86_64",
    sha256 = "736291b66f30c8ad543f5bed5375c92bc8a2e3bce1704a77f5b727ee844fb0dd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.47.1-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/736291b66f30c8ad543f5bed5375c92bc8a2e3bce1704a77f5b727ee844fb0dd",
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
        "https://storage.googleapis.com/builddeps/e8b7d03d574363beaebef73048b8fe8461ed7b1206152b81eb0852f5c01d533b",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.47.1-5.el10.s390x",
    sha256 = "25bb41764aefa735e891df10d2846b4c86f00f8eaabaf9a66acf08ebf290b700",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/e2fsprogs-libs-1.47.1-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/25bb41764aefa735e891df10d2846b4c86f00f8eaabaf9a66acf08ebf290b700",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.47.1-5.el10.x86_64",
    sha256 = "d73c79a7bda1ce465707d82fa6b9777fcd2776301a6f6722ca323b4c9337c64b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.47.1-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d73c79a7bda1ce465707d82fa6b9777fcd2776301a6f6722ca323b4c9337c64b",
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
    name = "edk2-aarch64-0__20260221-3.el10.aarch64",
    sha256 = "7b21ef0c38e0630485c95761f7865f59c38aae73ae7b49ccad33f6f465066e59",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20260221-3.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7b21ef0c38e0630485c95761f7865f59c38aae73ae7b49ccad33f6f465066e59",
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
    name = "edk2-ovmf-0__20260221-3.el10.s390x",
    sha256 = "c31b41656d0b2908e4a3363948649059a4a93d5a67582694b0ee3792bdfb428e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/edk2-ovmf-20260221-3.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c31b41656d0b2908e4a3363948649059a4a93d5a67582694b0ee3792bdfb428e",
    ],
)

rpm(
    name = "edk2-ovmf-0__20260221-3.el10.x86_64",
    sha256 = "c31b41656d0b2908e4a3363948649059a4a93d5a67582694b0ee3792bdfb428e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20260221-3.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c31b41656d0b2908e4a3363948649059a4a93d5a67582694b0ee3792bdfb428e",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.195-1.el10.aarch64",
    sha256 = "ad4cdfc5ed2cedfd4f495809665e0f5f1899bb6b9d04ea1dceb7d35e6ce50834",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.195-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad4cdfc5ed2cedfd4f495809665e0f5f1899bb6b9d04ea1dceb7d35e6ce50834",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.195-1.el10.s390x",
    sha256 = "1f73b021efbb172a067a0673a62b8297ac9205b6b3bb4b594874f65894a4b670",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.195-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1f73b021efbb172a067a0673a62b8297ac9205b6b3bb4b594874f65894a4b670",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.195-1.el10.x86_64",
    sha256 = "9477e02b036d53fe69c9c26f4fd7007666948cb04fe38098e138ecc7897e6f55",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.195-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9477e02b036d53fe69c9c26f4fd7007666948cb04fe38098e138ecc7897e6f55",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.195-1.el9.aarch64",
    sha256 = "8ce8a270f43d718f2c3e35b73efe86d23ac92ce47d721197cbb54dd13b30b9aa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-debuginfod-client-0.195-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8ce8a270f43d718f2c3e35b73efe86d23ac92ce47d721197cbb54dd13b30b9aa",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.195-1.el9.s390x",
    sha256 = "051f95bc2f558c7279da05b374c2944f7c71a29107eef56b0d4358b8298625f7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-debuginfod-client-0.195-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/051f95bc2f558c7279da05b374c2944f7c71a29107eef56b0d4358b8298625f7",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.195-1.el9.x86_64",
    sha256 = "d02550867f3c42888e7287e1973864ae7d4d9a11d99716e7b0ea1803bbf0523a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-debuginfod-client-0.195-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d02550867f3c42888e7287e1973864ae7d4d9a11d99716e7b0ea1803bbf0523a",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.195-1.el10.aarch64",
    sha256 = "afe64f03cff5fa2d07e4acee01266f783c6d95f4f59edd6f8e41fe6782942309",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.195-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/afe64f03cff5fa2d07e4acee01266f783c6d95f4f59edd6f8e41fe6782942309",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.195-1.el10.s390x",
    sha256 = "afe64f03cff5fa2d07e4acee01266f783c6d95f4f59edd6f8e41fe6782942309",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.195-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/afe64f03cff5fa2d07e4acee01266f783c6d95f4f59edd6f8e41fe6782942309",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.195-1.el10.x86_64",
    sha256 = "afe64f03cff5fa2d07e4acee01266f783c6d95f4f59edd6f8e41fe6782942309",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.195-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/afe64f03cff5fa2d07e4acee01266f783c6d95f4f59edd6f8e41fe6782942309",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.195-1.el9.aarch64",
    sha256 = "cd17c962a443c95895c4b2598630105679ffc065228d1e9a0709ced5d8abb9ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.195-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/cd17c962a443c95895c4b2598630105679ffc065228d1e9a0709ced5d8abb9ae",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.195-1.el9.s390x",
    sha256 = "cd17c962a443c95895c4b2598630105679ffc065228d1e9a0709ced5d8abb9ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-default-yama-scope-0.195-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/cd17c962a443c95895c4b2598630105679ffc065228d1e9a0709ced5d8abb9ae",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.195-1.el9.x86_64",
    sha256 = "cd17c962a443c95895c4b2598630105679ffc065228d1e9a0709ced5d8abb9ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.195-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/cd17c962a443c95895c4b2598630105679ffc065228d1e9a0709ced5d8abb9ae",
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
    name = "elfutils-libelf-0__0.195-1.el10.aarch64",
    sha256 = "16db84cf3bc63008615829837daf36dd278ec7c37bf04c293a9369f393333241",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.195-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/16db84cf3bc63008615829837daf36dd278ec7c37bf04c293a9369f393333241",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.195-1.el10.s390x",
    sha256 = "1df504356dda3e84b0344511bd9c714e398a9c217d1641dfda22584656e320e0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.195-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1df504356dda3e84b0344511bd9c714e398a9c217d1641dfda22584656e320e0",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.195-1.el10.x86_64",
    sha256 = "d88ef9e1bf7ec73a377e0fad5a7d8f5a8bcab8332f0fd15d4e57971d1919e262",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.195-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d88ef9e1bf7ec73a377e0fad5a7d8f5a8bcab8332f0fd15d4e57971d1919e262",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.195-1.el9.aarch64",
    sha256 = "fc87bbf10413b1cbc986e246972d9074db9adda26a21d007cbbe2b19ed5b33ff",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.195-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fc87bbf10413b1cbc986e246972d9074db9adda26a21d007cbbe2b19ed5b33ff",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.195-1.el9.s390x",
    sha256 = "d3f3051455c76c71955ae1601a236ab867e0b119c90be6e060f6c0d851ac7092",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libelf-0.195-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d3f3051455c76c71955ae1601a236ab867e0b119c90be6e060f6c0d851ac7092",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.195-1.el9.x86_64",
    sha256 = "a40d7e70ab22bd27d37ce9bbc6be25aceaee9cdc4dc9989863d6ed8bd3fe6727",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.195-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a40d7e70ab22bd27d37ce9bbc6be25aceaee9cdc4dc9989863d6ed8bd3fe6727",
    ],
)

rpm(
    name = "elfutils-libs-0__0.195-1.el10.aarch64",
    sha256 = "63fb3cab162b3aecce1e4fc012f09294cb06069302dd5c90d2d36c8035e08f7c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.195-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/63fb3cab162b3aecce1e4fc012f09294cb06069302dd5c90d2d36c8035e08f7c",
    ],
)

rpm(
    name = "elfutils-libs-0__0.195-1.el10.s390x",
    sha256 = "a8db7c92474cfd1b95e9d6c720e5964582deab7298efce50ca88c64e6ca1cd2d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.195-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a8db7c92474cfd1b95e9d6c720e5964582deab7298efce50ca88c64e6ca1cd2d",
    ],
)

rpm(
    name = "elfutils-libs-0__0.195-1.el10.x86_64",
    sha256 = "05e14cc51ea510c43a071e2775decb64cdc144508351c0478c2bda3eace5acb9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.195-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/05e14cc51ea510c43a071e2775decb64cdc144508351c0478c2bda3eace5acb9",
    ],
)

rpm(
    name = "elfutils-libs-0__0.195-1.el9.aarch64",
    sha256 = "b8fcc84a0a27f16cac56c2a733eb47cd5aeebd0c241653f2a9f451e132e09a95",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.195-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b8fcc84a0a27f16cac56c2a733eb47cd5aeebd0c241653f2a9f451e132e09a95",
    ],
)

rpm(
    name = "elfutils-libs-0__0.195-1.el9.s390x",
    sha256 = "de52f1ca0a82b3d996ac5c7e9f56f5389e834b88d60aad08a9417b30798785ad",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/elfutils-libs-0.195-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/de52f1ca0a82b3d996ac5c7e9f56f5389e834b88d60aad08a9417b30798785ad",
    ],
)

rpm(
    name = "elfutils-libs-0__0.195-1.el9.x86_64",
    sha256 = "e00a7e2026f79552aa9ef5bcb6d4a674505c4cb6821f3e125ed298f66feba39f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.195-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e00a7e2026f79552aa9ef5bcb6d4a674505c4cb6821f3e125ed298f66feba39f",
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
        "https://storage.googleapis.com/builddeps/9d093b8a289a4fbac304097d8d628744fa0ea88f3a50a64c4ee1c657cb42a5c8",
    ],
)

rpm(
    name = "expat-0__2.7.3-1.el10.s390x",
    sha256 = "5fce4ab3c8a5e188f560bdbac6f780e36af2e71210f765153ee2c9328b8a2a5f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/expat-2.7.3-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5fce4ab3c8a5e188f560bdbac6f780e36af2e71210f765153ee2c9328b8a2a5f",
    ],
)

rpm(
    name = "expat-0__2.7.3-1.el10.x86_64",
    sha256 = "e00c0876574daba5e70a3e2c86e21823fae1269b7a123d08ff5493a59dde3f36",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/expat-2.7.3-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e00c0876574daba5e70a3e2c86e21823fae1269b7a123d08ff5493a59dde3f36",
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
        "https://storage.googleapis.com/builddeps/6c4d8ecaf8b45c8d7d588c6ebe368a77805ed84830d0bc3b38e4c8e499514aba",
    ],
)

rpm(
    name = "filesystem-0__3.18-17.el10.s390x",
    sha256 = "087e8def18ded2dd2a96f7a4292a3654704807d05f4424c43c0f5c873d7f9cb5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/filesystem-3.18-17.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/087e8def18ded2dd2a96f7a4292a3654704807d05f4424c43c0f5c873d7f9cb5",
    ],
)

rpm(
    name = "filesystem-0__3.18-17.el10.x86_64",
    sha256 = "bcfb13f67c813d645f47e0a56d4bb76c0863deaf64ba93be8e0c30eecdc1e45e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/filesystem-3.18-17.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bcfb13f67c813d645f47e0a56d4bb76c0863deaf64ba93be8e0c30eecdc1e45e",
    ],
)

rpm(
    name = "findutils-1__4.10.0-5.el10.aarch64",
    sha256 = "f0e4db5b6e713c75e097e80218c592de4e6cb85d353f0933f64714df11b178b2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/findutils-4.10.0-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f0e4db5b6e713c75e097e80218c592de4e6cb85d353f0933f64714df11b178b2",
    ],
)

rpm(
    name = "findutils-1__4.10.0-5.el10.s390x",
    sha256 = "da20bdfeb9053ac3a1689d2ee2281298ee119175a8d486e4bb3eed1bc2857a94",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/findutils-4.10.0-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/da20bdfeb9053ac3a1689d2ee2281298ee119175a8d486e4bb3eed1bc2857a94",
    ],
)

rpm(
    name = "findutils-1__4.10.0-5.el10.x86_64",
    sha256 = "c646c7c108a007d62792aa66e0bc9326312089a0f8bc1c9e9300b301fd2e4276",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/findutils-4.10.0-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c646c7c108a007d62792aa66e0bc9326312089a0f8bc1c9e9300b301fd2e4276",
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
    name = "fips-provider-next-0__1.5.0-7.el10.s390x",
    sha256 = "d4e0cdc757d1775d8dc0c4f700eb4941a5665c3b69178d8a30839ae9add1f219",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/fips-provider-next-1.5.0-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d4e0cdc757d1775d8dc0c4f700eb4941a5665c3b69178d8a30839ae9add1f219",
    ],
)

rpm(
    name = "fips-provider-next-0__1.5.0-7.el10.x86_64",
    sha256 = "98f054a5f65c3e729905e7b62290ee31af04e848b5414b67f7c50a50f4707055",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/fips-provider-next-1.5.0-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/98f054a5f65c3e729905e7b62290ee31af04e848b5414b67f7c50a50f4707055",
    ],
)

rpm(
    name = "fips-provider-next-0__1.5.2-1.el9.aarch64",
    sha256 = "25054341a65e27a82eb68dec4b5d0412081da1491e3a67523204599115e69f86",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/fips-provider-next-1.5.2-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/25054341a65e27a82eb68dec4b5d0412081da1491e3a67523204599115e69f86",
    ],
)

rpm(
    name = "fips-provider-next-0__1.5.2-1.el9.s390x",
    sha256 = "beb4b139e60a3390e1e451fa0dfc0df28dde86637d968fa33100934a1ffbe848",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/fips-provider-next-1.5.2-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/beb4b139e60a3390e1e451fa0dfc0df28dde86637d968fa33100934a1ffbe848",
    ],
)

rpm(
    name = "fips-provider-next-0__1.5.2-1.el9.x86_64",
    sha256 = "99f056cbdfff2b0b441f8652e9338a404e3439f6fcbe6ce313ce7158767e1058",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/fips-provider-next-1.5.2-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/99f056cbdfff2b0b441f8652e9338a404e3439f6fcbe6ce313ce7158767e1058",
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
        "https://storage.googleapis.com/builddeps/6d0dd7c5dc828fc93d96ff215d90324f8efd9e88a9512081f4cf6d6323387a2f",
    ],
)

rpm(
    name = "fuse-0__2.9.9-25.el10.x86_64",
    sha256 = "0707885f1d8074b5d36d85b4c60a68a10867894b379225302a94f3d54b6d4934",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse-2.9.9-25.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0707885f1d8074b5d36d85b4c60a68a10867894b379225302a94f3d54b6d4934",
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
        "https://storage.googleapis.com/builddeps/86983857ec56f535e57283f302d9f344a348b55a9dc5e6e81ef388b397a14e2a",
    ],
)

rpm(
    name = "fuse-common-0__3.16.2-5.el10.x86_64",
    sha256 = "eecc51472bf7713a97821ae02898b6811752aa513aa40dc5d380459fce590a40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse-common-3.16.2-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eecc51472bf7713a97821ae02898b6811752aa513aa40dc5d380459fce590a40",
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
        "https://storage.googleapis.com/builddeps/65b86c79a139100f7d61acbef829a0a345c70316988cd7eb0f573f0c57e98647",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-25.el10.x86_64",
    sha256 = "a8b094d60b9a7f83a84d8c7b0cdeed565be044dc2ecd170965b2c55ee4fa40f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.9-25.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a8b094d60b9a7f83a84d8c7b0cdeed565be044dc2ecd170965b2c55ee4fa40f7",
    ],
)

rpm(
    name = "fuse3-libs-0__3.16.2-5.el10.aarch64",
    sha256 = "919f632731bc755d7c9c81d6faebb3bb703d7460ed72fdd65c453541d3999a72",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/fuse3-libs-3.16.2-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/919f632731bc755d7c9c81d6faebb3bb703d7460ed72fdd65c453541d3999a72",
    ],
)

rpm(
    name = "fuse3-libs-0__3.16.2-5.el10.s390x",
    sha256 = "68501eaef0f538ca7e3731a4968f308ced9ae9c2a1b3b4d310890dd86b1843c5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/fuse3-libs-3.16.2-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/68501eaef0f538ca7e3731a4968f308ced9ae9c2a1b3b4d310890dd86b1843c5",
    ],
)

rpm(
    name = "fuse3-libs-0__3.16.2-5.el10.x86_64",
    sha256 = "3482d8de135a306e94f7a35c1f8315b4e6acb699c1871ef28ddb02dc0fbdf7d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/fuse3-libs-3.16.2-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3482d8de135a306e94f7a35c1f8315b4e6acb699c1871ef28ddb02dc0fbdf7d6",
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
        "https://storage.googleapis.com/builddeps/16d7b639936dd4c8c977cd5b2ee3f5a02d3235954f67aa7485765a6b146683de",
    ],
)

rpm(
    name = "gawk-0__5.3.0-6.el10.s390x",
    sha256 = "0c918acd6aed7bbe461611db414bed4c1871b9ee9e4e5369460e016eb0c6bcbb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gawk-5.3.0-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0c918acd6aed7bbe461611db414bed4c1871b9ee9e4e5369460e016eb0c6bcbb",
    ],
)

rpm(
    name = "gawk-0__5.3.0-6.el10.x86_64",
    sha256 = "ba59a3a4ee8741ed4e0c2517086164a76dc85309947f8b5ca7884f05c08ed959",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gawk-5.3.0-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ba59a3a4ee8741ed4e0c2517086164a76dc85309947f8b5ca7884f05c08ed959",
    ],
)

rpm(
    name = "gcc-0__11.5.0-15.el9.aarch64",
    sha256 = "7f8eaaa493c3949c2bf647357760104af20a261f19e663badbbdcd4ea69d99d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gcc-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7f8eaaa493c3949c2bf647357760104af20a261f19e663badbbdcd4ea69d99d8",
    ],
)

rpm(
    name = "gcc-0__11.5.0-15.el9.s390x",
    sha256 = "db239fb69e76c4d364dc1cccb6ab016b42e2dcf8367af952300ddf0349d293ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gcc-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/db239fb69e76c4d364dc1cccb6ab016b42e2dcf8367af952300ddf0349d293ee",
    ],
)

rpm(
    name = "gcc-0__11.5.0-15.el9.x86_64",
    sha256 = "99e891f10bc6497834668940313d2e8c7fdba72547499d5be8a6ec6fceabf878",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gcc-11.5.0-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/99e891f10bc6497834668940313d2e8c7fdba72547499d5be8a6ec6fceabf878",
    ],
)

rpm(
    name = "gcc-0__14.3.1-4.4.el10.aarch64",
    sha256 = "fe754a0edcf74767003728f0f1b2ba99bcd070a07e3f68785886272abd40818d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/gcc-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fe754a0edcf74767003728f0f1b2ba99bcd070a07e3f68785886272abd40818d",
    ],
)

rpm(
    name = "gcc-0__14.3.1-4.4.el10.s390x",
    sha256 = "227501eb019e43934d83194f34ade61c6f0fee7ffbf6bc5e75aeb9035375965a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gcc-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/227501eb019e43934d83194f34ade61c6f0fee7ffbf6bc5e75aeb9035375965a",
    ],
)

rpm(
    name = "gcc-0__14.3.1-4.4.el10.x86_64",
    sha256 = "e93acc10b23ee2b9ddef5e593307cdf68ebc72551be03057fa11f44095675803",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/gcc-14.3.1-4.4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e93acc10b23ee2b9ddef5e593307cdf68ebc72551be03057fa11f44095675803",
    ],
)

rpm(
    name = "gdbm-1__1.23-14.el10.aarch64",
    sha256 = "0db16e24bf3d297cc3543842d63143f583de6ee157806b0a3dc51b5740a2722f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gdbm-1.23-14.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0db16e24bf3d297cc3543842d63143f583de6ee157806b0a3dc51b5740a2722f",
    ],
)

rpm(
    name = "gdbm-1__1.23-14.el10.s390x",
    sha256 = "95c556f933af240938736727df962465928b7a556a8586b01e90c647facc2839",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gdbm-1.23-14.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/95c556f933af240938736727df962465928b7a556a8586b01e90c647facc2839",
    ],
)

rpm(
    name = "gdbm-1__1.23-14.el10.x86_64",
    sha256 = "159a6f1affc65d960c11a8726472699f693cec90a54a0862ad8340d0968f4838",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gdbm-1.23-14.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/159a6f1affc65d960c11a8726472699f693cec90a54a0862ad8340d0968f4838",
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
        "https://storage.googleapis.com/builddeps/b46628d13eba77191aad6905de11fff87d6f45e52168e5b5365cb1f62078fd4d",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-14.el10.s390x",
    sha256 = "38f1f8006c38c8fffa7f298bf3a143943f8611acaee7aad8200edc6bcde534aa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gdbm-libs-1.23-14.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/38f1f8006c38c8fffa7f298bf3a143943f8611acaee7aad8200edc6bcde534aa",
    ],
)

rpm(
    name = "gdbm-libs-1__1.23-14.el10.x86_64",
    sha256 = "b5f678293062eb1fcba572501d62e215dccfd222c26f5b76d9424f3c188cedee",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.23-14.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b5f678293062eb1fcba572501d62e215dccfd222c26f5b76d9424f3c188cedee",
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
        "https://storage.googleapis.com/builddeps/27cba50dbb800aaf7f46bffa04003338c797472b334b08344b3633a60e0f1755",
    ],
)

rpm(
    name = "gettext-0__0.22.5-6.el10.s390x",
    sha256 = "02ab0b35769a517e0a2c255c4e4f23cfb9f661179355f81687a2d5b5198289d6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-0.22.5-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/02ab0b35769a517e0a2c255c4e4f23cfb9f661179355f81687a2d5b5198289d6",
    ],
)

rpm(
    name = "gettext-0__0.22.5-6.el10.x86_64",
    sha256 = "19430ae2b77a7e4637bfcb70501748a27011f6c1e144a195b7046ecd9e6a96b4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-0.22.5-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19430ae2b77a7e4637bfcb70501748a27011f6c1e144a195b7046ecd9e6a96b4",
    ],
)

rpm(
    name = "gettext-envsubst-0__0.22.5-6.el10.aarch64",
    sha256 = "ae3a179fff748702f7ad12bc2d8e58910d724a1d42f4cafb22af8ddfaf2eb216",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gettext-envsubst-0.22.5-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ae3a179fff748702f7ad12bc2d8e58910d724a1d42f4cafb22af8ddfaf2eb216",
    ],
)

rpm(
    name = "gettext-envsubst-0__0.22.5-6.el10.s390x",
    sha256 = "a9f2345a5875671c4d3a14ae491ea02b535d52ecb6ff65813aba392660d96065",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-envsubst-0.22.5-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a9f2345a5875671c4d3a14ae491ea02b535d52ecb6ff65813aba392660d96065",
    ],
)

rpm(
    name = "gettext-envsubst-0__0.22.5-6.el10.x86_64",
    sha256 = "f7b90e29f350fd67a2425a9d06c404371f1bbcdc43727452bf27c6c855d9eccf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-envsubst-0.22.5-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7b90e29f350fd67a2425a9d06c404371f1bbcdc43727452bf27c6c855d9eccf",
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
        "https://storage.googleapis.com/builddeps/460e9216dbdd5a5a42bcd49162639e5515020a1caf9a246734ab7c19d5747b8e",
    ],
)

rpm(
    name = "gettext-libs-0__0.22.5-6.el10.s390x",
    sha256 = "aebeafee7bc7b3513b5210039214447304f4734e8ba4e590cbf74cfe0fb04393",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-libs-0.22.5-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/aebeafee7bc7b3513b5210039214447304f4734e8ba4e590cbf74cfe0fb04393",
    ],
)

rpm(
    name = "gettext-libs-0__0.22.5-6.el10.x86_64",
    sha256 = "de538283e9cc0281d53e05c235905a9e5c64ad1ac2533afb915ba75052f540a3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-libs-0.22.5-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de538283e9cc0281d53e05c235905a9e5c64ad1ac2533afb915ba75052f540a3",
    ],
)

rpm(
    name = "gettext-runtime-0__0.22.5-6.el10.aarch64",
    sha256 = "76d58cbcdddca202c4eecc30df7692d5f6e847f0ac233227349942b6f860a5da",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gettext-runtime-0.22.5-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76d58cbcdddca202c4eecc30df7692d5f6e847f0ac233227349942b6f860a5da",
    ],
)

rpm(
    name = "gettext-runtime-0__0.22.5-6.el10.s390x",
    sha256 = "59a0988b7180c5b0c78c02b40c60f902f60d363beb3acc379c5d1ffd8fa6dfeb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gettext-runtime-0.22.5-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/59a0988b7180c5b0c78c02b40c60f902f60d363beb3acc379c5d1ffd8fa6dfeb",
    ],
)

rpm(
    name = "gettext-runtime-0__0.22.5-6.el10.x86_64",
    sha256 = "aec2ce3c3805190c65667c617e1ed100b65c251d16896819b0bc933ec3084ebf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gettext-runtime-0.22.5-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aec2ce3c3805190c65667c617e1ed100b65c251d16896819b0bc933ec3084ebf",
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
    name = "glib2-0__2.68.4-20.el9.aarch64",
    sha256 = "3911bba0d89cc320479fefd6ede6cec6c3c4537c198419ee4784bf6ae3bf60d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glib2-2.68.4-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3911bba0d89cc320479fefd6ede6cec6c3c4537c198419ee4784bf6ae3bf60d6",
    ],
)

rpm(
    name = "glib2-0__2.68.4-20.el9.s390x",
    sha256 = "d5f084b1534e680bf72f1a2b7dafccb0775e77c1c6f76de2d133022f5a6feacb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glib2-2.68.4-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d5f084b1534e680bf72f1a2b7dafccb0775e77c1c6f76de2d133022f5a6feacb",
    ],
)

rpm(
    name = "glib2-0__2.68.4-20.el9.x86_64",
    sha256 = "ce540bb580908bb7f025e06c4dab863658f15b1e9f89c232eea0a2d511c2b0ac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glib2-2.68.4-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce540bb580908bb7f025e06c4dab863658f15b1e9f89c232eea0a2d511c2b0ac",
    ],
)

rpm(
    name = "glib2-0__2.80.4-13.el10.aarch64",
    sha256 = "731257ffac939b14ee500a0f4cf30fe2403bf63d06b186312c418f5f1a02da7a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glib2-2.80.4-13.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/731257ffac939b14ee500a0f4cf30fe2403bf63d06b186312c418f5f1a02da7a",
    ],
)

rpm(
    name = "glib2-0__2.80.4-13.el10.s390x",
    sha256 = "cfbba10cc41a7b2d7f03b0e2552ef941be650a8ca27c7eb25495e22480f20736",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glib2-2.80.4-13.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cfbba10cc41a7b2d7f03b0e2552ef941be650a8ca27c7eb25495e22480f20736",
    ],
)

rpm(
    name = "glib2-0__2.80.4-13.el10.x86_64",
    sha256 = "3ae3929c5469f48f86744c8db5a4c76b3aa8f37d2009d5739b04735efe02079f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glib2-2.80.4-13.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3ae3929c5469f48f86744c8db5a4c76b3aa8f37d2009d5739b04735efe02079f",
    ],
)

rpm(
    name = "glibc-0__2.34-168.el9.x86_64",
    sha256 = "e06212b1cac1d9fd9857a00ddefefe9fb9f406199cb84fdd1153589c15e16289",
    urls = ["https://storage.googleapis.com/builddeps/e06212b1cac1d9fd9857a00ddefefe9fb9f406199cb84fdd1153589c15e16289"],
)

rpm(
    name = "glibc-0__2.34-274.el9.aarch64",
    sha256 = "0899538b1898831cec9a36c0fcfd313b6f5f8564e769051471fc41636966f5df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-2.34-274.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0899538b1898831cec9a36c0fcfd313b6f5f8564e769051471fc41636966f5df",
    ],
)

rpm(
    name = "glibc-0__2.34-274.el9.s390x",
    sha256 = "d1aeeebcae9556702ade9a6d2d825e205a8578aaf053cb2f7948058bec7a5932",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d1aeeebcae9556702ade9a6d2d825e205a8578aaf053cb2f7948058bec7a5932",
    ],
)

rpm(
    name = "glibc-0__2.34-274.el9.x86_64",
    sha256 = "d7935171d45e6a51658a0da6853a053d64db39e1bf1a09054bf5e24e5d7bead1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d7935171d45e6a51658a0da6853a053d64db39e1bf1a09054bf5e24e5d7bead1",
    ],
)

rpm(
    name = "glibc-0__2.39-124.el10.aarch64",
    sha256 = "98835b98adc90ed4396e3ad149af18b6f8d1d742dcac6548da8c5cc03fbedce4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-2.39-124.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98835b98adc90ed4396e3ad149af18b6f8d1d742dcac6548da8c5cc03fbedce4",
    ],
)

rpm(
    name = "glibc-0__2.39-124.el10.s390x",
    sha256 = "78dfca58dc26ee192678e7714a7564d798e530716412ae13e582d8cb0ad22588",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-2.39-124.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/78dfca58dc26ee192678e7714a7564d798e530716412ae13e582d8cb0ad22588",
    ],
)

rpm(
    name = "glibc-0__2.39-124.el10.x86_64",
    sha256 = "c90fe609c529f1536094a8b6071a99ea6048d73bff6878d4480ff8e17a7a0313",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-2.39-124.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c90fe609c529f1536094a8b6071a99ea6048d73bff6878d4480ff8e17a7a0313",
    ],
)

rpm(
    name = "glibc-common-0__2.34-168.el9.x86_64",
    sha256 = "531650744909efd0284bf6c16a45dbaf455b214c0cac4197cf6d43e8c7d83af8",
    urls = ["https://storage.googleapis.com/builddeps/531650744909efd0284bf6c16a45dbaf455b214c0cac4197cf6d43e8c7d83af8"],
)

rpm(
    name = "glibc-common-0__2.34-274.el9.aarch64",
    sha256 = "9b0305d168fd789f52e4f5a60c1cb0ea990a0ed6974fbd68aeb74163a668bd13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-common-2.34-274.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9b0305d168fd789f52e4f5a60c1cb0ea990a0ed6974fbd68aeb74163a668bd13",
    ],
)

rpm(
    name = "glibc-common-0__2.34-274.el9.s390x",
    sha256 = "9379bfc37b5bd14124ffabc6c208c45608f8f0af887b8e8d72cf884da54cebcd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-common-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9379bfc37b5bd14124ffabc6c208c45608f8f0af887b8e8d72cf884da54cebcd",
    ],
)

rpm(
    name = "glibc-common-0__2.34-274.el9.x86_64",
    sha256 = "3cc56a48533d9d59ab2501e4d7916b70a74eb181f46542c549bf99bbeb46d18f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-common-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3cc56a48533d9d59ab2501e4d7916b70a74eb181f46542c549bf99bbeb46d18f",
    ],
)

rpm(
    name = "glibc-common-0__2.39-124.el10.aarch64",
    sha256 = "277ec9b0c2be22dd0641f23b7680c9520b37522e65438d1994ca5a18ef68b7a9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-common-2.39-124.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/277ec9b0c2be22dd0641f23b7680c9520b37522e65438d1994ca5a18ef68b7a9",
    ],
)

rpm(
    name = "glibc-common-0__2.39-124.el10.s390x",
    sha256 = "367ca6b643c35ea214677ec4b74c2af2c3b24b7b8af16d7addb8a4cc758da547",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-common-2.39-124.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/367ca6b643c35ea214677ec4b74c2af2c3b24b7b8af16d7addb8a4cc758da547",
    ],
)

rpm(
    name = "glibc-common-0__2.39-124.el10.x86_64",
    sha256 = "efe77412c5d1cd057c4d4b212f932033e54cf87a343fd4f2a55c1bc698911c4a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-common-2.39-124.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/efe77412c5d1cd057c4d4b212f932033e54cf87a343fd4f2a55c1bc698911c4a",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-274.el9.aarch64",
    sha256 = "4988958d1c19fcb9f51712679101565fec746df7d3279486b91a12d19fc8046f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/glibc-devel-2.34-274.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4988958d1c19fcb9f51712679101565fec746df7d3279486b91a12d19fc8046f",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-274.el9.s390x",
    sha256 = "e83c99cc4e48b35a8962a3e160a63ba6884830cb88bca37dd905d05d7adc2be7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-devel-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e83c99cc4e48b35a8962a3e160a63ba6884830cb88bca37dd905d05d7adc2be7",
    ],
)

rpm(
    name = "glibc-devel-0__2.34-274.el9.x86_64",
    sha256 = "8f66bea8ca3dc96841f94fadba4caafb2f550c6aa565be88a053ba9152eb4a8f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-devel-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8f66bea8ca3dc96841f94fadba4caafb2f550c6aa565be88a053ba9152eb4a8f",
    ],
)

rpm(
    name = "glibc-devel-0__2.39-124.el10.aarch64",
    sha256 = "332f5790df85fbbcdef78e1a1344b3f0a9c7d329f822f7c5ecf698e81ff3efcf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/glibc-devel-2.39-124.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/332f5790df85fbbcdef78e1a1344b3f0a9c7d329f822f7c5ecf698e81ff3efcf",
    ],
)

rpm(
    name = "glibc-devel-0__2.39-124.el10.s390x",
    sha256 = "66fb444f4d7378cf6a83154e5d7d82b3a8bfc9c4bf4d8248f71df0eed3c6e124",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/glibc-devel-2.39-124.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/66fb444f4d7378cf6a83154e5d7d82b3a8bfc9c4bf4d8248f71df0eed3c6e124",
    ],
)

rpm(
    name = "glibc-devel-0__2.39-124.el10.x86_64",
    sha256 = "91c986c1654b3f2bf12cd61d677a5fc0c937ed86a56472b34243c15aa15a3297",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/glibc-devel-2.39-124.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/91c986c1654b3f2bf12cd61d677a5fc0c937ed86a56472b34243c15aa15a3297",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-274.el9.s390x",
    sha256 = "9d93a57a2133baf1532094af7828ded470b00d0d863a536ad58e197e3371d0de",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/glibc-headers-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9d93a57a2133baf1532094af7828ded470b00d0d863a536ad58e197e3371d0de",
    ],
)

rpm(
    name = "glibc-headers-0__2.34-274.el9.x86_64",
    sha256 = "0269445872f8cf03e4fb61840c35957e6732bbd95ce2db75f65002bea5c7b9be",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/glibc-headers-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0269445872f8cf03e4fb61840c35957e6732bbd95ce2db75f65002bea5c7b9be",
    ],
)

rpm(
    name = "glibc-langpack-el-0__2.34-274.el9.s390x",
    sha256 = "913ccc6641c0eae1085ae588904bfcc40f10449695d3f2535f68e916ea40371f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-langpack-el-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/913ccc6641c0eae1085ae588904bfcc40f10449695d3f2535f68e916ea40371f",
    ],
)

rpm(
    name = "glibc-langpack-es-0__2.34-274.el9.x86_64",
    sha256 = "463b4e480a77e3baba15ff2387fca9cd491d5c19430a0e87106d71e3cd659fa0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-langpack-es-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/463b4e480a77e3baba15ff2387fca9cd491d5c19430a0e87106d71e3cd659fa0",
    ],
)

rpm(
    name = "glibc-langpack-ff-0__2.34-274.el9.aarch64",
    sha256 = "bf356e4a74ab3cb751508c48153a3d86366acda266129510bd44e4dd1c666cbf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-langpack-ff-2.34-274.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bf356e4a74ab3cb751508c48153a3d86366acda266129510bd44e4dd1c666cbf",
    ],
)

rpm(
    name = "glibc-langpack-fi-0__2.39-124.el10.s390x",
    sha256 = "faca72b19ba92f0141c4093460ec33cfabc0500b941b6c1d0b91f64b7fb3e700",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-langpack-fi-2.39-124.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/faca72b19ba92f0141c4093460ec33cfabc0500b941b6c1d0b91f64b7fb3e700",
    ],
)

rpm(
    name = "glibc-langpack-fi-0__2.39-124.el10.x86_64",
    sha256 = "1f466f674ba7f4793ceb4bdd081264d73c4cc376e651f792599d078220577171",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-langpack-fi-2.39-124.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f466f674ba7f4793ceb4bdd081264d73c4cc376e651f792599d078220577171",
    ],
)

rpm(
    name = "glibc-langpack-hak-0__2.39-124.el10.aarch64",
    sha256 = "76f6f824dec8c3bc39bd28ed4c0a4350b706a9ad0ed206acd7280962efbbca59",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-langpack-hak-2.39-124.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76f6f824dec8c3bc39bd28ed4c0a4350b706a9ad0ed206acd7280962efbbca59",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-168.el9.x86_64",
    sha256 = "991b6d7370b237a3d576536a517d01a1ccc997959f4ea30ba07bd779641f79e8",
    urls = ["https://storage.googleapis.com/builddeps/991b6d7370b237a3d576536a517d01a1ccc997959f4ea30ba07bd779641f79e8"],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-274.el9.aarch64",
    sha256 = "876c0ec99ddce022a244d0f92aa2d334197367a0851a1ee5ed09654e0cc0fa86",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.34-274.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/876c0ec99ddce022a244d0f92aa2d334197367a0851a1ee5ed09654e0cc0fa86",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-274.el9.s390x",
    sha256 = "eddf8722af56331c1f30d75623c3a767a27c5caac21c69c50300802c84b15f0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eddf8722af56331c1f30d75623c3a767a27c5caac21c69c50300802c84b15f0a",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.34-274.el9.x86_64",
    sha256 = "17d4c21da562a51a12ba6a1561aa0dd8334b3ea745d5f95984fc72d0469ca586",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/17d4c21da562a51a12ba6a1561aa0dd8334b3ea745d5f95984fc72d0469ca586",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.39-124.el10.aarch64",
    sha256 = "c625e3601b62225aee9150daba3b8c199b9b53ae7dc5b45e05cfc0e3c2e57519",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.39-124.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c625e3601b62225aee9150daba3b8c199b9b53ae7dc5b45e05cfc0e3c2e57519",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.39-124.el10.s390x",
    sha256 = "c54131f7ef7f44b3e2e313f38ffbe8cef622620e364eef7bb51760a245010375",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/glibc-minimal-langpack-2.39-124.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c54131f7ef7f44b3e2e313f38ffbe8cef622620e364eef7bb51760a245010375",
    ],
)

rpm(
    name = "glibc-minimal-langpack-0__2.39-124.el10.x86_64",
    sha256 = "c5d787e339171f8d691aebbdc6cc66db9951aaf7b24b22a6036838b2029d6e62",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.39-124.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c5d787e339171f8d691aebbdc6cc66db9951aaf7b24b22a6036838b2029d6e62",
    ],
)

rpm(
    name = "glibc-static-0__2.34-274.el9.aarch64",
    sha256 = "28295dc42df5a6cd5e4de4cb3f059707f567297a9b1d90db1d254660428cf139",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/glibc-static-2.34-274.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/28295dc42df5a6cd5e4de4cb3f059707f567297a9b1d90db1d254660428cf139",
    ],
)

rpm(
    name = "glibc-static-0__2.34-274.el9.s390x",
    sha256 = "98cdd264b160f8d64e58bdbd186ca1e7768520585a1e1ec6783752a8302ff33f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/glibc-static-2.34-274.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/98cdd264b160f8d64e58bdbd186ca1e7768520585a1e1ec6783752a8302ff33f",
    ],
)

rpm(
    name = "glibc-static-0__2.34-274.el9.x86_64",
    sha256 = "b7bb43d2e3febf801f46a8ee6c91ac43ea5cab05030680909227ae52503b0e3a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/glibc-static-2.34-274.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7bb43d2e3febf801f46a8ee6c91ac43ea5cab05030680909227ae52503b0e3a",
    ],
)

rpm(
    name = "glibc-static-0__2.39-124.el10.aarch64",
    sha256 = "7c366f18eb29f63310c38e74257352de896cd13ec38a82efbf0f6abe7a47c9f7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/aarch64/os/Packages/glibc-static-2.39-124.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7c366f18eb29f63310c38e74257352de896cd13ec38a82efbf0f6abe7a47c9f7",
    ],
)

rpm(
    name = "glibc-static-0__2.39-124.el10.s390x",
    sha256 = "b42abd2dd44169ef18ed7208bbc2fde3cc6758fac3795dfdec46ab0e5b5e9a22",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/glibc-static-2.39-124.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b42abd2dd44169ef18ed7208bbc2fde3cc6758fac3795dfdec46ab0e5b5e9a22",
    ],
)

rpm(
    name = "glibc-static-0__2.39-124.el10.x86_64",
    sha256 = "ec989ef10e4dad3278ab23e26d08c8abd0c4592d5ec96c518973274f96bb5961",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/glibc-static-2.39-124.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ec989ef10e4dad3278ab23e26d08c8abd0c4592d5ec96c518973274f96bb5961",
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
        "https://storage.googleapis.com/builddeps/9bbe58df2a29320daf9b4c36305fcc7f781ab0bdd486736c6d8c685838141a41",
    ],
)

rpm(
    name = "gmp-1__6.2.1-12.el10.s390x",
    sha256 = "54d437788539933aa6de0963c6b1303e50b07f17db9ea847a71e19d1b4ef6a66",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gmp-6.2.1-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/54d437788539933aa6de0963c6b1303e50b07f17db9ea847a71e19d1b4ef6a66",
    ],
)

rpm(
    name = "gmp-1__6.2.1-12.el10.x86_64",
    sha256 = "6678824b5d45f9b66e8bfeb8f32736e0d710e3b38531a85548f55702d96b63a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gmp-6.2.1-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6678824b5d45f9b66e8bfeb8f32736e0d710e3b38531a85548f55702d96b63a8",
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
    name = "gnupg2-0__2.4.5-4.el10.s390x",
    sha256 = "2b576fbf4dafb91c6ed3ccb439f6ff4fa4057efd10d07949cf51fa2ef1f076f0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gnupg2-2.4.5-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2b576fbf4dafb91c6ed3ccb439f6ff4fa4057efd10d07949cf51fa2ef1f076f0",
    ],
)

rpm(
    name = "gnupg2-0__2.4.5-4.el10.x86_64",
    sha256 = "0531cbe23b63a37972751c9fefb904041d304df93f21b97ad18271659b0a5643",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gnupg2-2.4.5-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0531cbe23b63a37972751c9fefb904041d304df93f21b97ad18271659b0a5643",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-4.el10.aarch64",
    sha256 = "83f76e499060a6a5670315e2bc6c997c2a89d9390c18b16c005b60d1b142120f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/gnutls-3.8.10-4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/83f76e499060a6a5670315e2bc6c997c2a89d9390c18b16c005b60d1b142120f",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-4.el10.s390x",
    sha256 = "3c65e060da2e1bef16cca56a30d1031fdace4fcecb318f6381a454dc7dd511f2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gnutls-3.8.10-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3c65e060da2e1bef16cca56a30d1031fdace4fcecb318f6381a454dc7dd511f2",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-4.el10.x86_64",
    sha256 = "813d3ad511f57547777731aa3b7c7002c3d6f1e24eca211432228607dd55a0d0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gnutls-3.8.10-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/813d3ad511f57547777731aa3b7c7002c3d6f1e24eca211432228607dd55a0d0",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-8.el9.aarch64",
    sha256 = "530f18b1cdf0a56ff8ce81d5d9e1617f026224f39a9f494df73cd0b88e6f7774",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/gnutls-3.8.10-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/530f18b1cdf0a56ff8ce81d5d9e1617f026224f39a9f494df73cd0b88e6f7774",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-8.el9.s390x",
    sha256 = "0d3a7b77a92bc9da0594366818e6bb70db3e6e01a7491565dff1143dbf3d9145",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/gnutls-3.8.10-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0d3a7b77a92bc9da0594366818e6bb70db3e6e01a7491565dff1143dbf3d9145",
    ],
)

rpm(
    name = "gnutls-0__3.8.10-8.el9.x86_64",
    sha256 = "7995375d84e6592cdba3d94ba01e47f5607aecb2ff73b7af67a756def78e8a31",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/gnutls-3.8.10-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7995375d84e6592cdba3d94ba01e47f5607aecb2ff73b7af67a756def78e8a31",
    ],
)

rpm(
    name = "gnutls-0__3.8.3-6.el9.x86_64",
    sha256 = "97364bd099856650cdbcc18448e85a3cc6a3cebc9513190a1b4d7016132920d9",
    urls = ["https://storage.googleapis.com/builddeps/97364bd099856650cdbcc18448e85a3cc6a3cebc9513190a1b4d7016132920d9"],
)

rpm(
    name = "gnutls-dane-0__3.8.10-4.el10.aarch64",
    sha256 = "cc9ae625f41867420753d409edd1cca199e23b99c131548624becc3d48f30807",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.8.10-4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cc9ae625f41867420753d409edd1cca199e23b99c131548624becc3d48f30807",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-4.el10.s390x",
    sha256 = "5c6a3e0b06d1557ed07648966707a8c3d13baaaf9e8eada12481c86e4a7508de",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gnutls-dane-3.8.10-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c6a3e0b06d1557ed07648966707a8c3d13baaaf9e8eada12481c86e4a7508de",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-4.el10.x86_64",
    sha256 = "0eacc66d0e40ebf42f94860474b12c44125c10f59091174d780d8062063a97fa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.8.10-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0eacc66d0e40ebf42f94860474b12c44125c10f59091174d780d8062063a97fa",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-8.el9.aarch64",
    sha256 = "5560b3a0875f8111a7e095bd9fb99fd17f8c73f101d215675b728fb844bad849",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.8.10-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5560b3a0875f8111a7e095bd9fb99fd17f8c73f101d215675b728fb844bad849",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-8.el9.s390x",
    sha256 = "204861a2d273cd4a23e48d549df0b903a617523689d04f7d49eb52618c0e5784",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-dane-3.8.10-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/204861a2d273cd4a23e48d549df0b903a617523689d04f7d49eb52618c0e5784",
    ],
)

rpm(
    name = "gnutls-dane-0__3.8.10-8.el9.x86_64",
    sha256 = "ab529b1c8fe7589a3f078d12cccea1e2ceabc1ec46f8a04361e629b44f2584a2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.8.10-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ab529b1c8fe7589a3f078d12cccea1e2ceabc1ec46f8a04361e629b44f2584a2",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-4.el10.aarch64",
    sha256 = "ee6553c9e7a1fd26221e57a8d237facc44c517dcfd879eb31e2d9bbddc813ae4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.8.10-4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ee6553c9e7a1fd26221e57a8d237facc44c517dcfd879eb31e2d9bbddc813ae4",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-4.el10.s390x",
    sha256 = "b816ed9a43419c54080d910dd35894ea63c22f506ff235646cb0391fe46011be",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/gnutls-utils-3.8.10-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b816ed9a43419c54080d910dd35894ea63c22f506ff235646cb0391fe46011be",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-4.el10.x86_64",
    sha256 = "5a80e43797ca35419714bab58016a566bd26282799a547ca7a16b51c2114dede",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.8.10-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a80e43797ca35419714bab58016a566bd26282799a547ca7a16b51c2114dede",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-8.el9.aarch64",
    sha256 = "3b05270198ed4124ce60e78ed6ae8b65dc78783dd0356feb722cd58297e272e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.8.10-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3b05270198ed4124ce60e78ed6ae8b65dc78783dd0356feb722cd58297e272e3",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-8.el9.s390x",
    sha256 = "477d1e2f02088710becd8cae864c1f69a6c80490dbb362e0b17a90d8b67a2c7a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/gnutls-utils-3.8.10-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/477d1e2f02088710becd8cae864c1f69a6c80490dbb362e0b17a90d8b67a2c7a",
    ],
)

rpm(
    name = "gnutls-utils-0__3.8.10-8.el9.x86_64",
    sha256 = "7f4487409ec614298b1ddeb1856c53cfcb7a175621a4773703d602b7f79967f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.8.10-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f4487409ec614298b1ddeb1856c53cfcb7a175621a4773703d602b7f79967f4",
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
        "https://storage.googleapis.com/builddeps/a3bd85b169c321602bafe23ca724dfa2b897379a89384dfd453cbb3a03d25e66",
    ],
)

rpm(
    name = "gobject-introspection-0__1.79.1-6.el10.s390x",
    sha256 = "440ef891180126b7d295bca67df47a23bdf05dff3a43d535826b8aa82ad26bb3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gobject-introspection-1.79.1-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/440ef891180126b7d295bca67df47a23bdf05dff3a43d535826b8aa82ad26bb3",
    ],
)

rpm(
    name = "gobject-introspection-0__1.79.1-6.el10.x86_64",
    sha256 = "80913f97462db46c9962d539f325cef09bf85ab4c415a2c47b445fe96bba84b6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gobject-introspection-1.79.1-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/80913f97462db46c9962d539f325cef09bf85ab4c415a2c47b445fe96bba84b6",
    ],
)

rpm(
    name = "grep-0__3.11-10.el10.aarch64",
    sha256 = "d797740f7c738e5e7729949bde3d82274c5c6422242a82c1058fbe71ea0c37e9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/grep-3.11-10.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d797740f7c738e5e7729949bde3d82274c5c6422242a82c1058fbe71ea0c37e9",
    ],
)

rpm(
    name = "grep-0__3.11-10.el10.s390x",
    sha256 = "d30a1ab1991131978b67f26d6c119f97bb5408a4bebac0294f2ac5417fe12276",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/grep-3.11-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d30a1ab1991131978b67f26d6c119f97bb5408a4bebac0294f2ac5417fe12276",
    ],
)

rpm(
    name = "grep-0__3.11-10.el10.x86_64",
    sha256 = "a0eb701c640cd0a0c9195493a8fc9206fff62174d958ba4af2d92527191f803f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/grep-3.11-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0eb701c640cd0a0c9195493a8fc9206fff62174d958ba4af2d92527191f803f",
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
    name = "guestfs-tools-0__1.55.8-1.el10.s390x",
    sha256 = "2189a5506deecf82b2d978ee3176601a3cc71697fe204c3a0e91a5cf2824bdb9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/guestfs-tools-1.55.8-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2189a5506deecf82b2d978ee3176601a3cc71697fe204c3a0e91a5cf2824bdb9",
    ],
)

rpm(
    name = "guestfs-tools-0__1.55.8-1.el10.x86_64",
    sha256 = "16e3957449d5097f0fc3473c153182a9a3dea900c453a39982491ad6ffa8cb6c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/guestfs-tools-1.55.8-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/16e3957449d5097f0fc3473c153182a9a3dea900c453a39982491ad6ffa8cb6c",
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
        "https://storage.googleapis.com/builddeps/9b276d61a13e3c996f059a095881630fab9ec5a4a56a07ddc711e4db0a3362d4",
    ],
)

rpm(
    name = "gzip-0__1.13-3.el10.s390x",
    sha256 = "d76be88d032b4f7525f5414d11081fb930fe338f108830b01b24f8501de3c2d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/gzip-1.13-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d76be88d032b4f7525f5414d11081fb930fe338f108830b01b24f8501de3c2d5",
    ],
)

rpm(
    name = "gzip-0__1.13-3.el10.x86_64",
    sha256 = "b7117230deceaba8bcd1341f0528df5855e54997cea04379fd3cc2c7c1e07ba8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/gzip-1.13-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7117230deceaba8bcd1341f0528df5855e54997cea04379fd3cc2c7c1e07ba8",
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
        "https://storage.googleapis.com/builddeps/3b3fa64ec84f359ff667cf1c9f0c66e5300d08284d6a944e295fefbd3fd1e720",
    ],
)

rpm(
    name = "hexedit-0__1.6-8.el10.x86_64",
    sha256 = "b4e61671ac71d0dc721f67b6a7d5ff28e3ec9c8e1b8251104bcd34d6f0611ce3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/hexedit-1.6-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4e61671ac71d0dc721f67b6a7d5ff28e3ec9c8e1b8251104bcd34d6f0611ce3",
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
        "https://storage.googleapis.com/builddeps/967989dfab46ed23e33b59c956d27ad881051582f1c59995824b93249c5ff004",
    ],
)

rpm(
    name = "hivex-libs-0__1.3.24-2.el10.x86_64",
    sha256 = "ecf17b83680af8d8a3cef0a632ec3a7163d01c3dddd2364c9b47a4ba79e23150",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/hivex-libs-1.3.24-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ecf17b83680af8d8a3cef0a632ec3a7163d01c3dddd2364c9b47a4ba79e23150",
    ],
)

rpm(
    name = "hwdata-0__0.348-9.18.el9.x86_64",
    sha256 = "b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137",
    urls = ["https://storage.googleapis.com/builddeps/b25f5743e2f54a34d41bb6b37602b301260629ef91713f0b894c8ed9dd37c137"],
)

rpm(
    name = "hwdata-0__0.348-9.23.el9.s390x",
    sha256 = "09bf38157cef58e43785108b6a3dc59a746f3356feec967b6f908e628f4bf137",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/hwdata-0.348-9.23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/09bf38157cef58e43785108b6a3dc59a746f3356feec967b6f908e628f4bf137",
    ],
)

rpm(
    name = "hwdata-0__0.348-9.23.el9.x86_64",
    sha256 = "09bf38157cef58e43785108b6a3dc59a746f3356feec967b6f908e628f4bf137",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/hwdata-0.348-9.23.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/09bf38157cef58e43785108b6a3dc59a746f3356feec967b6f908e628f4bf137",
    ],
)

rpm(
    name = "hwdata-0__0.379-10.9.el10.s390x",
    sha256 = "9c96f8bdd87e9681af26c8b43198e6a9a55f658e2fb1463a777372a85fc8d7e0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/hwdata-0.379-10.9.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9c96f8bdd87e9681af26c8b43198e6a9a55f658e2fb1463a777372a85fc8d7e0",
    ],
)

rpm(
    name = "hwdata-0__0.379-10.9.el10.x86_64",
    sha256 = "9c96f8bdd87e9681af26c8b43198e6a9a55f658e2fb1463a777372a85fc8d7e0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/hwdata-0.379-10.9.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9c96f8bdd87e9681af26c8b43198e6a9a55f658e2fb1463a777372a85fc8d7e0",
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
    name = "iproute-0__6.17.0-2.el10.aarch64",
    sha256 = "76b2db02bfca3ddcd80182c582b997178c4397609ce549a04aa90463271215cc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iproute-6.17.0-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76b2db02bfca3ddcd80182c582b997178c4397609ce549a04aa90463271215cc",
    ],
)

rpm(
    name = "iproute-0__6.17.0-2.el10.s390x",
    sha256 = "6b9d4766562c00ccc69159a9dc02ba26707c7b17bc3359920d59f4419a20deb3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iproute-6.17.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6b9d4766562c00ccc69159a9dc02ba26707c7b17bc3359920d59f4419a20deb3",
    ],
)

rpm(
    name = "iproute-0__6.17.0-2.el10.x86_64",
    sha256 = "687e6ec55a55c01b2d3489fad6705a2a8638b0ce54a77c1718a903b998072238",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iproute-6.17.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/687e6ec55a55c01b2d3489fad6705a2a8638b0ce54a77c1718a903b998072238",
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
    name = "iproute-tc-0__6.17.0-2.el10.aarch64",
    sha256 = "a55e37a358bda187c442321168748437e1c18ea53fa207c6ee89555f024adb89",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iproute-tc-6.17.0-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a55e37a358bda187c442321168748437e1c18ea53fa207c6ee89555f024adb89",
    ],
)

rpm(
    name = "iproute-tc-0__6.17.0-2.el10.s390x",
    sha256 = "6df073feb9854e653e085c5f66452643cc8567b24d71e81d8d02ab63303ad626",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iproute-tc-6.17.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6df073feb9854e653e085c5f66452643cc8567b24d71e81d8d02ab63303ad626",
    ],
)

rpm(
    name = "iproute-tc-0__6.17.0-2.el10.x86_64",
    sha256 = "6a73b20c012ba9f4e9e9f19a7a8d0cd7c6b61f79c5c4ddad1a8967461a70ba35",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iproute-tc-6.17.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a73b20c012ba9f4e9e9f19a7a8d0cd7c6b61f79c5c4ddad1a8967461a70ba35",
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
    name = "iptables-libs-0__1.8.11-14.el10.aarch64",
    sha256 = "5b8bed1d8b9c8a60f1174c40dda6ae1989baa6086a9538cfd289313907c33118",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.11-14.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5b8bed1d8b9c8a60f1174c40dda6ae1989baa6086a9538cfd289313907c33118",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.11-14.el10.s390x",
    sha256 = "fa116a206a5543dd5124d41e384a99c263a3e30690cb72552090684167c6a4c2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iptables-libs-1.8.11-14.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fa116a206a5543dd5124d41e384a99c263a3e30690cb72552090684167c6a4c2",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.11-14.el10.x86_64",
    sha256 = "318fc14dd7649d02925b3346a05ec7003ae0785625b3197eaf9521aeb9b288d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.11-14.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/318fc14dd7649d02925b3346a05ec7003ae0785625b3197eaf9521aeb9b288d8",
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
        "https://storage.googleapis.com/builddeps/3ed67cca3fbb5f60f14f85ea712b1822f2a80c58287e744795ef995ebebc3761",
    ],
)

rpm(
    name = "iputils-0__20240905-5.el10.s390x",
    sha256 = "bf09f778f68c47515f0763e7c4aa952ed32dea57608a9473cb8edd50742e8a6a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/iputils-20240905-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bf09f778f68c47515f0763e7c4aa952ed32dea57608a9473cb8edd50742e8a6a",
    ],
)

rpm(
    name = "iputils-0__20240905-5.el10.x86_64",
    sha256 = "adfa1b26bf1cd23d0998c85da06ad787f0fa745bfd232f7acec225c1e88b05d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/iputils-20240905-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/adfa1b26bf1cd23d0998c85da06ad787f0fa745bfd232f7acec225c1e88b05d8",
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
        "https://storage.googleapis.com/builddeps/0b834df444ffe592d164f1dd5a2ce690417e459b8cd6d6c69b2075bbb9c8b4cb",
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
        "https://storage.googleapis.com/builddeps/a838d217420f9f10eb80a221b6cda50ff65e729c15be94f33cbb420f206ee880",
    ],
)

rpm(
    name = "jansson-0__2.14-3.el10.s390x",
    sha256 = "cc054f4efd4b779ec708061759de28acd9eb9df0ad8f3b32f9fe4752b1dcb06c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/jansson-2.14-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cc054f4efd4b779ec708061759de28acd9eb9df0ad8f3b32f9fe4752b1dcb06c",
    ],
)

rpm(
    name = "jansson-0__2.14-3.el10.x86_64",
    sha256 = "25d2ef852d5941b27ae105ec780aa367605a6f8b86e6c6a13abdee1c1065979f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/jansson-2.14-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/25d2ef852d5941b27ae105ec780aa367605a6f8b86e6c6a13abdee1c1065979f",
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
        "https://storage.googleapis.com/builddeps/d3ecfebff7515c94e971c9584b0815202712cc2642526ee4fe5e424ec8ff2fae",
    ],
)

rpm(
    name = "json-c-0__0.18-3.el10.s390x",
    sha256 = "d4bc7597af6496e70ffa04858c8d2418267b302959db9da3f89c6edfc723ccd4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/json-c-0.18-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d4bc7597af6496e70ffa04858c8d2418267b302959db9da3f89c6edfc723ccd4",
    ],
)

rpm(
    name = "json-c-0__0.18-3.el10.x86_64",
    sha256 = "e73ae01d509fb9bef1bbd675be1c0003b0ee942a4187e9b14ef43e56e508245b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/json-c-0.18-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e73ae01d509fb9bef1bbd675be1c0003b0ee942a4187e9b14ef43e56e508245b",
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
        "https://storage.googleapis.com/builddeps/41de435cef6d704c1bb85066b9711e44f60b1dbff3574997094e5ed166e2b95e",
    ],
)

rpm(
    name = "json-glib-0__1.8.0-5.el10.s390x",
    sha256 = "851e663120bae993deed48bd36f06a84d9082b51b35c87244e7a8d3735bf422f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/json-glib-1.8.0-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/851e663120bae993deed48bd36f06a84d9082b51b35c87244e7a8d3735bf422f",
    ],
)

rpm(
    name = "json-glib-0__1.8.0-5.el10.x86_64",
    sha256 = "156fddb0053ab256ec6ecbe7818c0ec8e957228eb2ed1d7cd244ecda85e1197e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/json-glib-1.8.0-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/156fddb0053ab256ec6ecbe7818c0ec8e957228eb2ed1d7cd244ecda85e1197e",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-722.el9.aarch64",
    sha256 = "838d9375d044f9f0f03d1a4195d6c1360aa793aef73147e4fd828b9ae165396c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/kernel-headers-5.14.0-722.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/838d9375d044f9f0f03d1a4195d6c1360aa793aef73147e4fd828b9ae165396c",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-722.el9.s390x",
    sha256 = "3456701f21c6f93ebc91880d617440665fec328fd2c98408988d962cac5c439c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/kernel-headers-5.14.0-722.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3456701f21c6f93ebc91880d617440665fec328fd2c98408988d962cac5c439c",
    ],
)

rpm(
    name = "kernel-headers-0__5.14.0-722.el9.x86_64",
    sha256 = "de5e7f0c5d5d4128243277052ddd4db46422241ffb81adba3e2c3aba8b256cac",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/kernel-headers-5.14.0-722.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de5e7f0c5d5d4128243277052ddd4db46422241ffb81adba3e2c3aba8b256cac",
    ],
)

rpm(
    name = "kernel-headers-0__6.12.0-233.el10.aarch64",
    sha256 = "37bf6765ef7676e5a05e5200a3e9a0d06f542d02e3ee4066c44cf8e66331c26a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/kernel-headers-6.12.0-233.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/37bf6765ef7676e5a05e5200a3e9a0d06f542d02e3ee4066c44cf8e66331c26a",
    ],
)

rpm(
    name = "kernel-headers-0__6.12.0-233.el10.s390x",
    sha256 = "46120491d1ed732a5234b8e17b9d3470f85af4e0971827a1961b312c03bc964a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/kernel-headers-6.12.0-233.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/46120491d1ed732a5234b8e17b9d3470f85af4e0971827a1961b312c03bc964a",
    ],
)

rpm(
    name = "kernel-headers-0__6.12.0-233.el10.x86_64",
    sha256 = "5c769c60ced5878cc4b89063ea9196445fa9d020c822306050b782d498ca7189",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/kernel-headers-6.12.0-233.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5c769c60ced5878cc4b89063ea9196445fa9d020c822306050b782d498ca7189",
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
        "https://storage.googleapis.com/builddeps/a6ff394736256d5c2317ab5503a056d0f60155a92090853179506358bfd2333f",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-5.el10.s390x",
    sha256 = "f2ed690c8ec6ef2a0b912ba324268d354d282f437b30044a44304543e78d9238",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/keyutils-libs-1.6.3-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f2ed690c8ec6ef2a0b912ba324268d354d282f437b30044a44304543e78d9238",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.3-5.el10.x86_64",
    sha256 = "312e0bf42841bb330f7721012d1ee5816e5ea223e54fc5dfd1a95c6f1d7516b0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/keyutils-libs-1.6.3-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/312e0bf42841bb330f7721012d1ee5816e5ea223e54fc5dfd1a95c6f1d7516b0",
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
    name = "kmod-0__31-13.el10.aarch64",
    sha256 = "a3db8298e46e1ba852dec2537f616ae51c4ef14989932059c4c57563d30c4c82",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/kmod-31-13.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a3db8298e46e1ba852dec2537f616ae51c4ef14989932059c4c57563d30c4c82",
    ],
)

rpm(
    name = "kmod-0__31-13.el10.s390x",
    sha256 = "e238e6f1772a648243c41cc4ccce6a592b12c16a460e0a90f662f23c1cabebc5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/kmod-31-13.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e238e6f1772a648243c41cc4ccce6a592b12c16a460e0a90f662f23c1cabebc5",
    ],
)

rpm(
    name = "kmod-0__31-13.el10.x86_64",
    sha256 = "eccbe0bed15e27471945b486d82b6bc4b1003b63e579b99b79afc7cf897656a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/kmod-31-13.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eccbe0bed15e27471945b486d82b6bc4b1003b63e579b99b79afc7cf897656a2",
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
    name = "kmod-libs-0__31-13.el10.aarch64",
    sha256 = "9bd10be61b682fb1263c2c6afff852efc714f0cf04acbaa1ed1180f5abeca919",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/kmod-libs-31-13.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9bd10be61b682fb1263c2c6afff852efc714f0cf04acbaa1ed1180f5abeca919",
    ],
)

rpm(
    name = "kmod-libs-0__31-13.el10.s390x",
    sha256 = "b164ddc883444eae10d296a4478e5fe2576a193c4ee0921ccdd4df7211e1aa55",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/kmod-libs-31-13.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b164ddc883444eae10d296a4478e5fe2576a193c4ee0921ccdd4df7211e1aa55",
    ],
)

rpm(
    name = "kmod-libs-0__31-13.el10.x86_64",
    sha256 = "bdf857119d993a27da5b7b4bdfa581dd92f1c1e0a7a29596c21080e1ac8b4fc8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/kmod-libs-31-13.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bdf857119d993a27da5b7b4bdfa581dd92f1c1e0a7a29596c21080e1ac8b4fc8",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-10.el9.aarch64",
    sha256 = "02c094878ceb99014307c07aee6a95422d67b856571ee1f2c65b67f556b0a008",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.1-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/02c094878ceb99014307c07aee6a95422d67b856571ee1f2c65b67f556b0a008",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-10.el9.s390x",
    sha256 = "7f79794f0adc0b7f0ede5dd6d8536068c7f8de948d947e42ce1cdafeb96fe8e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/krb5-libs-1.21.1-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7f79794f0adc0b7f0ede5dd6d8536068c7f8de948d947e42ce1cdafeb96fe8e3",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.1-10.el9.x86_64",
    sha256 = "55f585ca5ceb611bcd44ce845179769fa42a2316fe23b83b1e13947fd54b7e0d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.1-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/55f585ca5ceb611bcd44ce845179769fa42a2316fe23b83b1e13947fd54b7e0d",
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
    name = "krb5-libs-0__1.21.3-11.el10.aarch64",
    sha256 = "a41b1ba1b82b994022ca5d6426e44aac92db1ec36123ddbfc8740ef58c9191ce",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.21.3-11.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a41b1ba1b82b994022ca5d6426e44aac92db1ec36123ddbfc8740ef58c9191ce",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.3-11.el10.s390x",
    sha256 = "f61dcd7aad1df818e151d2db57321c719455b6a50b2dd189c5a58ad6d24ff966",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/krb5-libs-1.21.3-11.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f61dcd7aad1df818e151d2db57321c719455b6a50b2dd189c5a58ad6d24ff966",
    ],
)

rpm(
    name = "krb5-libs-0__1.21.3-11.el10.x86_64",
    sha256 = "0423bea388f0f3e8a9b08f9bb3cd07c6fdd9303a8602e8c1d58b5ca5f32b8d65",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.21.3-11.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0423bea388f0f3e8a9b08f9bb3cd07c6fdd9303a8602e8c1d58b5ca5f32b8d65",
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
        "https://storage.googleapis.com/builddeps/6eb8705527ce26aa64dd692e6103e6b2197fb134a9e4f03e5db78d4c45035ddf",
    ],
)

rpm(
    name = "less-0__661-3.el10.x86_64",
    sha256 = "1cf4afdf660772f65668cc702722facd7ed79d849150e14cab623cffd4167516",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/less-661-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cf4afdf660772f65668cc702722facd7ed79d849150e14cab623cffd4167516",
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
        "https://storage.googleapis.com/builddeps/20f3eeb53bf86dd2c7152fcdc33df3efd60777edd11f31f633739c0fdc0bdbf5",
    ],
)

rpm(
    name = "libacl-0__2.3.2-4.el10.s390x",
    sha256 = "5f26a314b6e88e87516979610e9c0bda3dc55c67c9339bf79739574162eb1fa6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libacl-2.3.2-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5f26a314b6e88e87516979610e9c0bda3dc55c67c9339bf79739574162eb1fa6",
    ],
)

rpm(
    name = "libacl-0__2.3.2-4.el10.x86_64",
    sha256 = "dd06cfe883fcdf7cb14b749180abfd9fe9924723341a8644a9f65c086febc647",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libacl-2.3.2-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd06cfe883fcdf7cb14b749180abfd9fe9924723341a8644a9f65c086febc647",
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
        "https://storage.googleapis.com/builddeps/99660f7b25fdb5503e0414e263ad91d0c1b61f1dc4e106721c0d1380b239d17f",
    ],
)

rpm(
    name = "libaio-0__0.3.111-22.el10.s390x",
    sha256 = "5ee6ef6f4625016ae0746586e62fcf1c70596f8b557d8e6ad54cc67a4ae26690",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libaio-0.3.111-22.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5ee6ef6f4625016ae0746586e62fcf1c70596f8b557d8e6ad54cc67a4ae26690",
    ],
)

rpm(
    name = "libaio-0__0.3.111-22.el10.x86_64",
    sha256 = "ea807b22c77a37a766e62ad533dc3f9b80fd5b260016487cecea55b095092446",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libaio-0.3.111-22.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ea807b22c77a37a766e62ad533dc3f9b80fd5b260016487cecea55b095092446",
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
    name = "libarchive-0__3.5.3-9.el9.aarch64",
    sha256 = "8bb25bc13e7f5dfd0f450e9a9c246a4160dedfdd38de25f7742bfd9750d6b59b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libarchive-3.5.3-9.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8bb25bc13e7f5dfd0f450e9a9c246a4160dedfdd38de25f7742bfd9750d6b59b",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-9.el9.s390x",
    sha256 = "d8884a6cc3b9e9c4ec548ed50f259d3915947f61fccd66e698a90f1fd173ce9e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libarchive-3.5.3-9.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d8884a6cc3b9e9c4ec548ed50f259d3915947f61fccd66e698a90f1fd173ce9e",
    ],
)

rpm(
    name = "libarchive-0__3.5.3-9.el9.x86_64",
    sha256 = "7f22c226edb997c41ccda04737e850b406b2e6db1194748797a9e67dde2c1ab5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libarchive-3.5.3-9.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f22c226edb997c41ccda04737e850b406b2e6db1194748797a9e67dde2c1ab5",
    ],
)

rpm(
    name = "libarchive-0__3.7.7-8.el10.s390x",
    sha256 = "6977dfd3ec170ccafbddf652cb91323f436b769afe4bf2cd4ce9ad6f1cb4d952",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libarchive-3.7.7-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6977dfd3ec170ccafbddf652cb91323f436b769afe4bf2cd4ce9ad6f1cb4d952",
    ],
)

rpm(
    name = "libarchive-0__3.7.7-8.el10.x86_64",
    sha256 = "de418d6521ce376c970dd34872917824686785b050ca5cfa91e0900aa95bc9ab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libarchive-3.7.7-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de418d6521ce376c970dd34872917824686785b050ca5cfa91e0900aa95bc9ab",
    ],
)

rpm(
    name = "libasan-0__11.5.0-15.el9.aarch64",
    sha256 = "1df42bb9ae39a790a4ab0199663e27708cfa60b8a89d3214e6a9e2e568165db1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libasan-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1df42bb9ae39a790a4ab0199663e27708cfa60b8a89d3214e6a9e2e568165db1",
    ],
)

rpm(
    name = "libasan-0__11.5.0-15.el9.s390x",
    sha256 = "34bcb1f382a2eb56a59c0c90db994fa8caed462d10d5838171de7cc7bfaf6020",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libasan-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/34bcb1f382a2eb56a59c0c90db994fa8caed462d10d5838171de7cc7bfaf6020",
    ],
)

rpm(
    name = "libasan-0__14.3.1-4.4.el10.aarch64",
    sha256 = "1d1728b4d1ac7cdc05a696ef8b4f65346e788a79e11e25ddd38f4cc189539a8e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libasan-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1d1728b4d1ac7cdc05a696ef8b4f65346e788a79e11e25ddd38f4cc189539a8e",
    ],
)

rpm(
    name = "libasan-0__14.3.1-4.4.el10.s390x",
    sha256 = "cb18cf1281793fac52d882eb27c367da6551ee7e9802d1a329770f4dd2edd8a7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libasan-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cb18cf1281793fac52d882eb27c367da6551ee7e9802d1a329770f4dd2edd8a7",
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
        "https://storage.googleapis.com/builddeps/d31b659dd6036b990ea71c10c426df13f2f395685ea8674fcca49d6a5fbeb580",
    ],
)

rpm(
    name = "libassuan-0__2.5.6-6.el10.x86_64",
    sha256 = "5cb1eff4efadf906bb8060bb41c205bf77eabe73142599053ae892ee7d85e9c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libassuan-2.5.6-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5cb1eff4efadf906bb8060bb41c205bf77eabe73142599053ae892ee7d85e9c0",
    ],
)

rpm(
    name = "libatomic-0__11.5.0-15.el9.aarch64",
    sha256 = "71a203d4137dfe5794be5b272b1ffe1fe2ed00be9269223b8ac4e0d360e87fbc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libatomic-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/71a203d4137dfe5794be5b272b1ffe1fe2ed00be9269223b8ac4e0d360e87fbc",
    ],
)

rpm(
    name = "libatomic-0__11.5.0-15.el9.s390x",
    sha256 = "b02620556a043b4f68ffa3a7ea5d89290e3d5158e2033990ce5ed1f03bd84bd1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libatomic-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b02620556a043b4f68ffa3a7ea5d89290e3d5158e2033990ce5ed1f03bd84bd1",
    ],
)

rpm(
    name = "libatomic-0__14.3.1-4.4.el10.aarch64",
    sha256 = "be72dc41f36eb4e8f01003b4ed4f45b921db5b8eced1fcde29b055a224b5d6d2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libatomic-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/be72dc41f36eb4e8f01003b4ed4f45b921db5b8eced1fcde29b055a224b5d6d2",
    ],
)

rpm(
    name = "libatomic-0__14.3.1-4.4.el10.s390x",
    sha256 = "bde0e892caec40bb5a2460e0b470d2e996b60678d50700927373fbcdd011fc7e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libatomic-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bde0e892caec40bb5a2460e0b470d2e996b60678d50700927373fbcdd011fc7e",
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
        "https://storage.googleapis.com/builddeps/37a06ff130ff4112ca431839607e4d7c583ec4b0191431aa9913bba754880040",
    ],
)

rpm(
    name = "libattr-0__2.5.2-5.el10.s390x",
    sha256 = "583ef53e42b6928c6a707baee521a3161f0d00d094db96ce05b39ae4409c73f8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libattr-2.5.2-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/583ef53e42b6928c6a707baee521a3161f0d00d094db96ce05b39ae4409c73f8",
    ],
)

rpm(
    name = "libattr-0__2.5.2-5.el10.x86_64",
    sha256 = "2ec3c5ba70aaae97db5226f07476c3fd0adfeed15d7cce3b676288273c829274",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libattr-2.5.2-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ec3c5ba70aaae97db5226f07476c3fd0adfeed15d7cce3b676288273c829274",
    ],
)

rpm(
    name = "libattr-0__2.6.0-2.el9.aarch64",
    sha256 = "111a4f7ffe93fc2dd3b3155146d31045b04a1a74d1a4e58d56386db414d28c05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libattr-2.6.0-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/111a4f7ffe93fc2dd3b3155146d31045b04a1a74d1a4e58d56386db414d28c05",
    ],
)

rpm(
    name = "libattr-0__2.6.0-2.el9.s390x",
    sha256 = "8aceb6196eb757cf3810e0dd864b4bb1866b50dea1f9231bdb9c06e077bcd14b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libattr-2.6.0-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8aceb6196eb757cf3810e0dd864b4bb1866b50dea1f9231bdb9c06e077bcd14b",
    ],
)

rpm(
    name = "libattr-0__2.6.0-2.el9.x86_64",
    sha256 = "4077e93ea08373cc9c65bcf6cb7cad8e88831c3a91d5814af238dfb3c9b54d10",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libattr-2.6.0-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4077e93ea08373cc9c65bcf6cb7cad8e88831c3a91d5814af238dfb3c9b54d10",
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
    name = "libblkid-0__2.37.4-25.el9.aarch64",
    sha256 = "40de20b6cbd0d5bf61e1576d47c154b349779be6790d8ad05d54cad94a8f9a3b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libblkid-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/40de20b6cbd0d5bf61e1576d47c154b349779be6790d8ad05d54cad94a8f9a3b",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-25.el9.s390x",
    sha256 = "62d6027ed230599196800f12bbd058670aa4a8759c829c934e0b829c3996c288",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libblkid-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/62d6027ed230599196800f12bbd058670aa4a8759c829c934e0b829c3996c288",
    ],
)

rpm(
    name = "libblkid-0__2.37.4-25.el9.x86_64",
    sha256 = "2309af12b80fec77070d354fdae370ffa3e57209137b46098286895be5a484f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libblkid-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2309af12b80fec77070d354fdae370ffa3e57209137b46098286895be5a484f5",
    ],
)

rpm(
    name = "libblkid-0__2.40.2-20.el10.aarch64",
    sha256 = "058bc31675b778637801e3aa3af79e6c227efc6e592f8c9558606f8e48599eb4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libblkid-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/058bc31675b778637801e3aa3af79e6c227efc6e592f8c9558606f8e48599eb4",
    ],
)

rpm(
    name = "libblkid-0__2.40.2-20.el10.s390x",
    sha256 = "d717e9c7fc02f7672780882aa91f139c9c74977ce37f86b61655e60600559c66",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libblkid-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d717e9c7fc02f7672780882aa91f139c9c74977ce37f86b61655e60600559c66",
    ],
)

rpm(
    name = "libblkid-0__2.40.2-20.el10.x86_64",
    sha256 = "84e56581aa7ef8d87d02f0b7ed9bc4c0a04bb080a53fe6a90cfc53a601612662",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libblkid-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84e56581aa7ef8d87d02f0b7ed9bc4c0a04bb080a53fe6a90cfc53a601612662",
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
        "https://storage.googleapis.com/builddeps/f89a67afcbc8eacc5c8c40e7c30ec5a5aaa78e89bca1dd1032b89c8634bbc605",
    ],
)

rpm(
    name = "libbpf-2__1.7.0-1.el10.s390x",
    sha256 = "0d45b7988fb7d679b8f4f9e88439b086c0c086a1ba7853bfd376b5061ac3c12b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libbpf-1.7.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0d45b7988fb7d679b8f4f9e88439b086c0c086a1ba7853bfd376b5061ac3c12b",
    ],
)

rpm(
    name = "libbpf-2__1.7.0-1.el10.x86_64",
    sha256 = "1379b88512429975bbbdd65c8737cbc793664bef2f0e8c2e04c1481b939c85ca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libbpf-1.7.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1379b88512429975bbbdd65c8737cbc793664bef2f0e8c2e04c1481b939c85ca",
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
        "https://storage.googleapis.com/builddeps/6deea1eafedaa040d5c7c4af870f2ae2edb6742f02cd56cdf07789cf2acd1359",
    ],
)

rpm(
    name = "libbrotli-0__1.1.0-7.el10.x86_64",
    sha256 = "9b397443a3ffe381380af22b55b0cac0f02412b859f92888138fba5e1df5e15d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libbrotli-1.1.0-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9b397443a3ffe381380af22b55b0cac0f02412b859f92888138fba5e1df5e15d",
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
        "https://storage.googleapis.com/builddeps/221ced92933bca63eb94d1ed60699f364e5d0b0b9915a0fba39d8b98d513d887",
    ],
)

rpm(
    name = "libburn-0__1.5.6-6.el10.s390x",
    sha256 = "6ac13f60b7e3bee622332411d4ec87c77dbd23f5710edf73209d29bf85904e4d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libburn-1.5.6-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6ac13f60b7e3bee622332411d4ec87c77dbd23f5710edf73209d29bf85904e4d",
    ],
)

rpm(
    name = "libburn-0__1.5.6-6.el10.x86_64",
    sha256 = "83ba66223a60bf93d13710b632ecc8c057c294a17f36ba31a18e16e7d4b97819",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libburn-1.5.6-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/83ba66223a60bf93d13710b632ecc8c057c294a17f36ba31a18e16e7d4b97819",
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
        "https://storage.googleapis.com/builddeps/38c8ab1a8883b39bf46006ed39b7834cfa0df2ca0a4825908f7da4ed631c8fc6",
    ],
)

rpm(
    name = "libcap-0__2.69-7.el10.s390x",
    sha256 = "3f7016928e759a177e4103d6c31dc0833ec43eae5836d871bc5ea540fd3d0c7b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcap-2.69-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3f7016928e759a177e4103d6c31dc0833ec43eae5836d871bc5ea540fd3d0c7b",
    ],
)

rpm(
    name = "libcap-0__2.69-7.el10.x86_64",
    sha256 = "54c14cb5c8dc3536f43d632d766ed302a8ecad2ad8efd6aa2d079dafc11d1cd9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcap-2.69-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54c14cb5c8dc3536f43d632d766ed302a8ecad2ad8efd6aa2d079dafc11d1cd9",
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
    name = "libcap-ng-0__0.9.3-1.el10.aarch64",
    sha256 = "ceb3d2d1e48c55c80f1b5dc78e77e54adb0f69dccc35b4c49829e8d145a4502b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcap-ng-0.9.3-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ceb3d2d1e48c55c80f1b5dc78e77e54adb0f69dccc35b4c49829e8d145a4502b",
    ],
)

rpm(
    name = "libcap-ng-0__0.9.3-1.el10.s390x",
    sha256 = "9c492cc5094dfe6bd2b3b1032aae692d30c62a81d945427e522c2d72d02214f1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcap-ng-0.9.3-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9c492cc5094dfe6bd2b3b1032aae692d30c62a81d945427e522c2d72d02214f1",
    ],
)

rpm(
    name = "libcap-ng-0__0.9.3-1.el10.x86_64",
    sha256 = "86b80421ec51da7cbe3f80ab53acde4f37a62ffe27b224df59ede07a3baedbbf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcap-ng-0.9.3-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86b80421ec51da7cbe3f80ab53acde4f37a62ffe27b224df59ede07a3baedbbf",
    ],
)

rpm(
    name = "libcbor-0__0.11.0-3.el10.aarch64",
    sha256 = "588538490e8d295a4b0833bfcec72eeb378e7175e5d1df3c50341f6537ba15a9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcbor-0.11.0-3.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/588538490e8d295a4b0833bfcec72eeb378e7175e5d1df3c50341f6537ba15a9",
    ],
)

rpm(
    name = "libcbor-0__0.11.0-3.el10.s390x",
    sha256 = "cebf5e31b218be5724ff76f34f60e3ef673e821ca5b5a7ea60e1dea67cfd271a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcbor-0.11.0-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cebf5e31b218be5724ff76f34f60e3ef673e821ca5b5a7ea60e1dea67cfd271a",
    ],
)

rpm(
    name = "libcbor-0__0.11.0-3.el10.x86_64",
    sha256 = "49a800e8569ac3e22fbe1b3595683f9934cbeabc6fd03c1a224b97441af7d3ab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcbor-0.11.0-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/49a800e8569ac3e22fbe1b3595683f9934cbeabc6fd03c1a224b97441af7d3ab",
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
        "https://storage.googleapis.com/builddeps/97380e5fe0fce42be70418333bae2d5d9044c5f7fbda30c9b28f7776718e76a2",
    ],
)

rpm(
    name = "libcom_err-0__1.47.1-5.el10.s390x",
    sha256 = "09f3032ebafe4e8d93152c940f160b68df98b1a1c978fae86b8524a8d7713467",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcom_err-1.47.1-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/09f3032ebafe4e8d93152c940f160b68df98b1a1c978fae86b8524a8d7713467",
    ],
)

rpm(
    name = "libcom_err-0__1.47.1-5.el10.x86_64",
    sha256 = "37b036aa4cb44adade9c4206f2eea389035387082be0d2eda0249d9fd07fb842",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcom_err-1.47.1-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/37b036aa4cb44adade9c4206f2eea389035387082be0d2eda0249d9fd07fb842",
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
        "https://storage.googleapis.com/builddeps/2df21ddec6f917d330835cee60c9c71a604fa3f988aabe5f1deeebd76582dc5a",
    ],
)

rpm(
    name = "libconfig-0__1.7.3-10.el10.x86_64",
    sha256 = "5bee52a5f0599fc6a59df28222e6e831c441887471412a3d18e6d13ddfaaa881",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libconfig-1.7.3-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5bee52a5f0599fc6a59df28222e6e831c441887471412a3d18e6d13ddfaaa881",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-31.el9.x86_64",
    sha256 = "6438485e38465ee944e25abedcf4a1761564fe5202f05a02c71e4c880255b539",
    urls = ["https://storage.googleapis.com/builddeps/6438485e38465ee944e25abedcf4a1761564fe5202f05a02c71e4c880255b539"],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-43.el9.aarch64",
    sha256 = "dde117f183a44553b98c14ac3ed29bf6c7a302522e436eda909cdb44980afe66",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.76.1-43.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dde117f183a44553b98c14ac3ed29bf6c7a302522e436eda909cdb44980afe66",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-43.el9.s390x",
    sha256 = "c2807b0788883480e4c1ecae130f66e1463672461d1ca33bee6160be5e7fe2b8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libcurl-minimal-7.76.1-43.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c2807b0788883480e4c1ecae130f66e1463672461d1ca33bee6160be5e7fe2b8",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.76.1-43.el9.x86_64",
    sha256 = "ca12a88c313df73ce0e8f5a652b57daded8733183c0d44f85f3dca780b356c67",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.76.1-43.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ca12a88c313df73ce0e8f5a652b57daded8733183c0d44f85f3dca780b356c67",
    ],
)

rpm(
    name = "libcurl-minimal-0__8.12.1-6.el10.aarch64",
    sha256 = "071dc30ed569560fcecaed51e0e3e668658e0be90d1f3507f32d32e122fad27b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-8.12.1-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/071dc30ed569560fcecaed51e0e3e668658e0be90d1f3507f32d32e122fad27b",
    ],
)

rpm(
    name = "libcurl-minimal-0__8.12.1-6.el10.s390x",
    sha256 = "db9efa00b840bf9a5662f48247b7b97b16c7f1ef2063827a259c9cb54bbc0f43",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libcurl-minimal-8.12.1-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/db9efa00b840bf9a5662f48247b7b97b16c7f1ef2063827a259c9cb54bbc0f43",
    ],
)

rpm(
    name = "libcurl-minimal-0__8.12.1-6.el10.x86_64",
    sha256 = "0463229bec1ce920e576e777ba8457b00db91d9e0ea43b4d772d4a241ab98d39",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-8.12.1-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0463229bec1ce920e576e777ba8457b00db91d9e0ea43b4d772d4a241ab98d39",
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
    name = "libeconf-0__0.4.1-7.el9.aarch64",
    sha256 = "d2adf4f7d6c66c2962c1b7024d0b9514895d813aa50010ca6d1d652f3f73a87f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libeconf-0.4.1-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d2adf4f7d6c66c2962c1b7024d0b9514895d813aa50010ca6d1d652f3f73a87f",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-7.el9.s390x",
    sha256 = "19b54d80020f15ff5753d0d116faa4dd2b358f1a55c4854ea7843aa89379954a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libeconf-0.4.1-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/19b54d80020f15ff5753d0d116faa4dd2b358f1a55c4854ea7843aa89379954a",
    ],
)

rpm(
    name = "libeconf-0__0.4.1-7.el9.x86_64",
    sha256 = "5d852e2a7fbb298efeb05303c783afcebb369021337ca934df518362618de8f3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libeconf-0.4.1-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5d852e2a7fbb298efeb05303c783afcebb369021337ca934df518362618de8f3",
    ],
)

rpm(
    name = "libeconf-0__0.6.2-4.el10.aarch64",
    sha256 = "1bb73420b4f72fb200ccc560224107e1ae62b8a7156051a88c9239c9def47983",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libeconf-0.6.2-4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1bb73420b4f72fb200ccc560224107e1ae62b8a7156051a88c9239c9def47983",
    ],
)

rpm(
    name = "libeconf-0__0.6.2-4.el10.s390x",
    sha256 = "eb314cc56ffe80a641f97664b4d5a7313e3be7f1bef79e6636bcaab5751b350f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libeconf-0.6.2-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eb314cc56ffe80a641f97664b4d5a7313e3be7f1bef79e6636bcaab5751b350f",
    ],
)

rpm(
    name = "libeconf-0__0.6.2-4.el10.x86_64",
    sha256 = "1cdb8e5bf4d7680e41ebb2b76da3aab34c1ece4bba2fed952d8f49da69117dfa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libeconf-0.6.2-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cdb8e5bf4d7680e41ebb2b76da3aab34c1ece4bba2fed952d8f49da69117dfa",
    ],
)

rpm(
    name = "libevent-0__2.1.12-16.el10.aarch64",
    sha256 = "275530b6896bec203e5cdf0cb427c78da43f5b01d3d26b0a2b239f2ad49fcda2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libevent-2.1.12-16.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/275530b6896bec203e5cdf0cb427c78da43f5b01d3d26b0a2b239f2ad49fcda2",
    ],
)

rpm(
    name = "libevent-0__2.1.12-16.el10.s390x",
    sha256 = "11e041d07e9f2f30736efa420ed437e11676bc796914f22c7b3fc322f08e997a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libevent-2.1.12-16.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/11e041d07e9f2f30736efa420ed437e11676bc796914f22c7b3fc322f08e997a",
    ],
)

rpm(
    name = "libevent-0__2.1.12-16.el10.x86_64",
    sha256 = "f8f5c3946bbd53590978e9aeca3064d81ab580492c4ff8044c48797870276f47",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libevent-2.1.12-16.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f8f5c3946bbd53590978e9aeca3064d81ab580492c4ff8044c48797870276f47",
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
    name = "libfdisk-0__2.37.4-21.el9.x86_64",
    sha256 = "9a594c51e3bf09cb5016485ee2f143de6db960ff1c7e135c0097f59fa51b2edb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a594c51e3bf09cb5016485ee2f143de6db960ff1c7e135c0097f59fa51b2edb",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-25.el9.aarch64",
    sha256 = "d724b6dd4dc886b1d598edc24d30ebb06dfc675252073e04838c56d0ed18e173",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libfdisk-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d724b6dd4dc886b1d598edc24d30ebb06dfc675252073e04838c56d0ed18e173",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-25.el9.s390x",
    sha256 = "7584b9f892c5378bfa976d40c1e02e5a9ee058fd09ee14658aa13b1ab3448b6b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libfdisk-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7584b9f892c5378bfa976d40c1e02e5a9ee058fd09ee14658aa13b1ab3448b6b",
    ],
)

rpm(
    name = "libfdisk-0__2.37.4-25.el9.x86_64",
    sha256 = "57e990f6940ce2caed0d9578838549576535ad83f93ffc97df3bcbaf1ae72567",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libfdisk-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/57e990f6940ce2caed0d9578838549576535ad83f93ffc97df3bcbaf1ae72567",
    ],
)

rpm(
    name = "libfdisk-0__2.40.2-20.el10.aarch64",
    sha256 = "46b25eff7b01ebb8986db226a96f3ce7bf770ddad16f05d9576933e6cc99b01b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libfdisk-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/46b25eff7b01ebb8986db226a96f3ce7bf770ddad16f05d9576933e6cc99b01b",
    ],
)

rpm(
    name = "libfdisk-0__2.40.2-20.el10.s390x",
    sha256 = "3b692f35ac9c63e402f7bb36005ad9737338aabce9032e1064be719c5e6bd09e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libfdisk-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3b692f35ac9c63e402f7bb36005ad9737338aabce9032e1064be719c5e6bd09e",
    ],
)

rpm(
    name = "libfdisk-0__2.40.2-20.el10.x86_64",
    sha256 = "536a0b3ca1fd39c25c5c6be7e18fdeca6c1359b48ec69d2170117f2eac231448",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libfdisk-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/536a0b3ca1fd39c25c5c6be7e18fdeca6c1359b48ec69d2170117f2eac231448",
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
        "https://storage.googleapis.com/builddeps/2a568810d2b8fbd8425eeb64738491a514feaec766f1b90d320caf320e543134",
    ],
)

rpm(
    name = "libfdt-0__1.7.0-12.el10.s390x",
    sha256 = "bd49a0dac4411aca4319b6dc670e0fab0fb9672eaedd83aab40452878fbe46ac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libfdt-1.7.0-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bd49a0dac4411aca4319b6dc670e0fab0fb9672eaedd83aab40452878fbe46ac",
    ],
)

rpm(
    name = "libfdt-0__1.7.0-12.el10.x86_64",
    sha256 = "9c519693ffe97be0dda3e06ea9708446b2692b5fd72b8cfb93f0b78ddc6418d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libfdt-1.7.0-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9c519693ffe97be0dda3e06ea9708446b2692b5fd72b8cfb93f0b78ddc6418d5",
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
        "https://storage.googleapis.com/builddeps/87b620ad4069f0a9623913acc568a2659bcee3695293b275a9f09b809437bf6e",
    ],
)

rpm(
    name = "libffi-0__3.4.4-10.el10.s390x",
    sha256 = "592be60a3f4ee70236cc254894f587012ce533a6f4fc74031bf6674792782338",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libffi-3.4.4-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/592be60a3f4ee70236cc254894f587012ce533a6f4fc74031bf6674792782338",
    ],
)

rpm(
    name = "libffi-0__3.4.4-10.el10.x86_64",
    sha256 = "72aff2f3b4291f5418491e612be4f92d65a9239224a4906c0c63dbc4fc668e73",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libffi-3.4.4-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/72aff2f3b4291f5418491e612be4f92d65a9239224a4906c0c63dbc4fc668e73",
    ],
)

rpm(
    name = "libfido2-0__1.14.0-7.el10.aarch64",
    sha256 = "8ac9963cf2e29f6450213363a9f963747e185cbab7b9d472d0cd275768139d4a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libfido2-1.14.0-7.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8ac9963cf2e29f6450213363a9f963747e185cbab7b9d472d0cd275768139d4a",
    ],
)

rpm(
    name = "libfido2-0__1.14.0-7.el10.s390x",
    sha256 = "3ff583eaa691f84cb0f3f4b49b7454880f2aca54db749a2532c03d0f361de529",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libfido2-1.14.0-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3ff583eaa691f84cb0f3f4b49b7454880f2aca54db749a2532c03d0f361de529",
    ],
)

rpm(
    name = "libfido2-0__1.14.0-7.el10.x86_64",
    sha256 = "5652d77a1394ba80ece520e817832224a03b693d172b7b32587d0fc502419bdd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libfido2-1.14.0-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5652d77a1394ba80ece520e817832224a03b693d172b7b32587d0fc502419bdd",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-15.el9.aarch64",
    sha256 = "9fa96a86f8bfe2a03390874f4e794cac76a82edbd47d6bfe881ab0de0a59efb1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcc-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9fa96a86f8bfe2a03390874f4e794cac76a82edbd47d6bfe881ab0de0a59efb1",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-15.el9.s390x",
    sha256 = "330c7cf21f7b3adb9f48ffc732d5c7470abc68a33775e9b99da7070e241e1b46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcc-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/330c7cf21f7b3adb9f48ffc732d5c7470abc68a33775e9b99da7070e241e1b46",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-15.el9.x86_64",
    sha256 = "522e07f3a8a09a6d5c9174340031d3002d319d4f6ecad8a483d30d68f02fc36d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcc-11.5.0-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/522e07f3a8a09a6d5c9174340031d3002d319d4f6ecad8a483d30d68f02fc36d",
    ],
)

rpm(
    name = "libgcc-0__11.5.0-5.el9.x86_64",
    sha256 = "442c065a815212ac21760ff9f0bd93e9f5d5972925d9e987a421cbf6ebba41d2",
    urls = ["https://storage.googleapis.com/builddeps/442c065a815212ac21760ff9f0bd93e9f5d5972925d9e987a421cbf6ebba41d2"],
)

rpm(
    name = "libgcc-0__14.3.1-4.4.el10.aarch64",
    sha256 = "0f93f59846c59092b90d4170499f7b4c61de30c2b359cf853f4251dc09c7629e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libgcc-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0f93f59846c59092b90d4170499f7b4c61de30c2b359cf853f4251dc09c7629e",
    ],
)

rpm(
    name = "libgcc-0__14.3.1-4.4.el10.s390x",
    sha256 = "bbbc0d0e0ca942e402d9ab30b693566e596ce4b6a404f2e2c9f3d9e5fcb56924",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgcc-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bbbc0d0e0ca942e402d9ab30b693566e596ce4b6a404f2e2c9f3d9e5fcb56924",
    ],
)

rpm(
    name = "libgcc-0__14.3.1-4.4.el10.x86_64",
    sha256 = "ffbc36198762cf0ea31e129109744f85fbc9b708faf5a34201e9d4c54c9a1422",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgcc-14.3.1-4.4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffbc36198762cf0ea31e129109744f85fbc9b708faf5a34201e9d4c54c9a1422",
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
    name = "libgcrypt-0__1.10.0-13.el9.aarch64",
    sha256 = "dd1b8da929a138573303d096391893f6ebf72a2964771d793b2c65334dbefe0f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.10.0-13.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dd1b8da929a138573303d096391893f6ebf72a2964771d793b2c65334dbefe0f",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-13.el9.s390x",
    sha256 = "46bc0123d8d988b3cd91c76cf6ef7c4c4c61482a83ad919bb001a6f7809ce69e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgcrypt-1.10.0-13.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/46bc0123d8d988b3cd91c76cf6ef7c4c4c61482a83ad919bb001a6f7809ce69e",
    ],
)

rpm(
    name = "libgcrypt-0__1.10.0-13.el9.x86_64",
    sha256 = "71026b5a461fc0c4777db81a529dea0f9205f74c175bbdfbc0f5bab8475d05c9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.10.0-13.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/71026b5a461fc0c4777db81a529dea0f9205f74c175bbdfbc0f5bab8475d05c9",
    ],
)

rpm(
    name = "libgcrypt-0__1.11.0-6.el10.s390x",
    sha256 = "b4383e8187d076c47b732487866eeebc0e40c1474e0dd6d927dfca6172cb274f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgcrypt-1.11.0-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b4383e8187d076c47b732487866eeebc0e40c1474e0dd6d927dfca6172cb274f",
    ],
)

rpm(
    name = "libgcrypt-0__1.11.0-6.el10.x86_64",
    sha256 = "1be7cfbc9f69f9e2b3d3f0621e14ded96e27d1c334decb5c88d1e396edf825e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.11.0-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1be7cfbc9f69f9e2b3d3f0621e14ded96e27d1c334decb5c88d1e396edf825e2",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-15.el9.aarch64",
    sha256 = "d6973d3c5018128efb0fc5ec80118fc40684e94c3ddf88e983b6509603e34652",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libgomp-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d6973d3c5018128efb0fc5ec80118fc40684e94c3ddf88e983b6509603e34652",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-15.el9.s390x",
    sha256 = "546fb268fdf18a42f8fcab3482bb5740ba90af974da1e79912d65917c44b1ab6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libgomp-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/546fb268fdf18a42f8fcab3482bb5740ba90af974da1e79912d65917c44b1ab6",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-15.el9.x86_64",
    sha256 = "9374cbca43271f1267cf265deb1d0078ef68fdb40507aad74044202a68028924",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libgomp-11.5.0-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9374cbca43271f1267cf265deb1d0078ef68fdb40507aad74044202a68028924",
    ],
)

rpm(
    name = "libgomp-0__11.5.0-5.el9.x86_64",
    sha256 = "0158d5640d1f4b3841b681fa26a17361c56d7b1231e64eb163e3d75155913053",
    urls = ["https://storage.googleapis.com/builddeps/0158d5640d1f4b3841b681fa26a17361c56d7b1231e64eb163e3d75155913053"],
)

rpm(
    name = "libgomp-0__14.3.1-4.4.el10.aarch64",
    sha256 = "63c5829dbcc69c44de060cee7ee1f8a68c93d81107808f7b6941642b52694e3c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libgomp-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/63c5829dbcc69c44de060cee7ee1f8a68c93d81107808f7b6941642b52694e3c",
    ],
)

rpm(
    name = "libgomp-0__14.3.1-4.4.el10.s390x",
    sha256 = "a9e15dd3591b45f9c71bac57a880f9e2edee06842ff475e67d27469dbb7b573c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libgomp-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a9e15dd3591b45f9c71bac57a880f9e2edee06842ff475e67d27469dbb7b573c",
    ],
)

rpm(
    name = "libgomp-0__14.3.1-4.4.el10.x86_64",
    sha256 = "6f9e43fe4ff233974a5f9405c8ee5535b6999e2214cab947f57250133816c6a0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgomp-14.3.1-4.4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6f9e43fe4ff233974a5f9405c8ee5535b6999e2214cab947f57250133816c6a0",
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
        "https://storage.googleapis.com/builddeps/d2ae277868010dcf3003a59234b70743f2ff0846e0f8aba6cf349de19d6c2173",
    ],
)

rpm(
    name = "libgpg-error-0__1.50-2.el10.x86_64",
    sha256 = "b7d74c79f82abf581fdb5b9fbd0b3792640c26780652036be284347b7b339fff",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libgpg-error-1.50-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7d74c79f82abf581fdb5b9fbd0b3792640c26780652036be284347b7b339fff",
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
    name = "libguestfs-1__1.59.8-1.el10.s390x",
    sha256 = "80c9c27365588abd2da248b726ff92e070ba1c90533d73bf8c218a1f45ad22f1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libguestfs-1.59.8-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/80c9c27365588abd2da248b726ff92e070ba1c90533d73bf8c218a1f45ad22f1",
    ],
)

rpm(
    name = "libguestfs-1__1.59.8-1.el10.x86_64",
    sha256 = "083146cc1fa8c4ed1b5e341e9c9987001c1097654c9885efb4ff2045b21aa4a5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libguestfs-1.59.8-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/083146cc1fa8c4ed1b5e341e9c9987001c1097654c9885efb4ff2045b21aa4a5",
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
    name = "libibverbs-0__62.0-2.el10.aarch64",
    sha256 = "1728f1d3c16fc4a7cb7c17479b93e748fb72e370ea2f2fe8b98db63cead3ffb6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libibverbs-62.0-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1728f1d3c16fc4a7cb7c17479b93e748fb72e370ea2f2fe8b98db63cead3ffb6",
    ],
)

rpm(
    name = "libibverbs-0__62.0-2.el10.s390x",
    sha256 = "b1e2c677c73285cd616c87d9041f2fbf15738ef1711c86ef5638dd038cc08a83",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libibverbs-62.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b1e2c677c73285cd616c87d9041f2fbf15738ef1711c86ef5638dd038cc08a83",
    ],
)

rpm(
    name = "libibverbs-0__62.0-2.el10.x86_64",
    sha256 = "5f81183d1d4679aabd7b81eb11a72d1e8b3e4b992fd940c57942cafbc518b8e6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libibverbs-62.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5f81183d1d4679aabd7b81eb11a72d1e8b3e4b992fd940c57942cafbc518b8e6",
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
        "https://storage.googleapis.com/builddeps/947248aeedd08f88d9490f3020dee6416595cf8d25e15738c306d55e9cae8bfc",
    ],
)

rpm(
    name = "libidn2-0__2.3.7-3.el10.s390x",
    sha256 = "7d75542211c9b7b8e53aaf6aee0dc430f091414fb2ca755baadc35f844bded75",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libidn2-2.3.7-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7d75542211c9b7b8e53aaf6aee0dc430f091414fb2ca755baadc35f844bded75",
    ],
)

rpm(
    name = "libidn2-0__2.3.7-3.el10.x86_64",
    sha256 = "04ae61bbe2cc0db7581d6f96a562b9b87a8a4dba714a0cf2c73bba6306e94c27",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libidn2-2.3.7-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/04ae61bbe2cc0db7581d6f96a562b9b87a8a4dba714a0cf2c73bba6306e94c27",
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
        "https://storage.googleapis.com/builddeps/038aa1a45c117b4c4abbeedc3f67faa01a66570bc554f0e53955ce83a26e7281",
    ],
)

rpm(
    name = "libisoburn-0__1.5.6-6.el10.s390x",
    sha256 = "52fa5b55814330bf00ce834de3fa86c0dff29691feb242800f04cc13c44f2f3a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libisoburn-1.5.6-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/52fa5b55814330bf00ce834de3fa86c0dff29691feb242800f04cc13c44f2f3a",
    ],
)

rpm(
    name = "libisoburn-0__1.5.6-6.el10.x86_64",
    sha256 = "6f41c5e8e0d9dedf3c0b07f2ddc0748870cb18ed15a23dc22038a022c1d38e74",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libisoburn-1.5.6-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6f41c5e8e0d9dedf3c0b07f2ddc0748870cb18ed15a23dc22038a022c1d38e74",
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
        "https://storage.googleapis.com/builddeps/cacea726d7dd126a364ec2431f49417b884287c84664042d51d59011acac34d6",
    ],
)

rpm(
    name = "libisofs-0__1.5.6-6.el10.s390x",
    sha256 = "655af71fa634d1bb1c86b6c4810c452c98c1772dc9d55da3e3e9f2413bafc293",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libisofs-1.5.6-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/655af71fa634d1bb1c86b6c4810c452c98c1772dc9d55da3e3e9f2413bafc293",
    ],
)

rpm(
    name = "libisofs-0__1.5.6-6.el10.x86_64",
    sha256 = "a61eda57352e86657ca99fc774d0d74c0fbd67dad5bcd02139ff6de466a038ac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libisofs-1.5.6-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a61eda57352e86657ca99fc774d0d74c0fbd67dad5bcd02139ff6de466a038ac",
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
        "https://storage.googleapis.com/builddeps/f4a0e294968ce54ad30e4de5baabcdebd7a9db7900266e4feb7cadecd7f18cba",
    ],
)

rpm(
    name = "libksba-0__1.6.7-2.el10.x86_64",
    sha256 = "2bfa8330ad9c63eaecd2bd1d0989625e812d853a85c505cb759a2d1c06750607",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libksba-1.6.7-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2bfa8330ad9c63eaecd2bd1d0989625e812d853a85c505cb759a2d1c06750607",
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
        "https://storage.googleapis.com/builddeps/786a9caea9de8f4529e5ec07ac24c9cfccd50ee5f4b045c37dbd4eb074b34f34",
    ],
)

rpm(
    name = "libmnl-0__1.0.5-7.el10.s390x",
    sha256 = "bb6229f5c62e69828f94cd16f0086115e17ea9c9ab831ad14781f9b9807cd3d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libmnl-1.0.5-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bb6229f5c62e69828f94cd16f0086115e17ea9c9ab831ad14781f9b9807cd3d8",
    ],
)

rpm(
    name = "libmnl-0__1.0.5-7.el10.x86_64",
    sha256 = "1e1d36725d958fc3f9016cc85b238bf5462a43ae7be144d0a4a72be0982d68ce",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libmnl-1.0.5-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e1d36725d958fc3f9016cc85b238bf5462a43ae7be144d0a4a72be0982d68ce",
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
    name = "libmount-0__2.37.4-25.el9.aarch64",
    sha256 = "903e1c5a61a57eafa8b68d5d23b1288cae061b65fdd4a942933cf8862ee4b1e3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libmount-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/903e1c5a61a57eafa8b68d5d23b1288cae061b65fdd4a942933cf8862ee4b1e3",
    ],
)

rpm(
    name = "libmount-0__2.37.4-25.el9.s390x",
    sha256 = "e4f81986fd3609aeaf6099697a7aebcd72dc96f160ee79c3dc2e8c8c5f1df10b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libmount-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e4f81986fd3609aeaf6099697a7aebcd72dc96f160ee79c3dc2e8c8c5f1df10b",
    ],
)

rpm(
    name = "libmount-0__2.37.4-25.el9.x86_64",
    sha256 = "ffb1ab2134b59539b097ce4a3c5287c61d2d4a626f512dbb93036d90ce2d755a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libmount-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffb1ab2134b59539b097ce4a3c5287c61d2d4a626f512dbb93036d90ce2d755a",
    ],
)

rpm(
    name = "libmount-0__2.40.2-20.el10.aarch64",
    sha256 = "9609baaf1bb7b7201297c80722dec9f319d6c66bf6c47b13f0e48b103282cfb7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libmount-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9609baaf1bb7b7201297c80722dec9f319d6c66bf6c47b13f0e48b103282cfb7",
    ],
)

rpm(
    name = "libmount-0__2.40.2-20.el10.s390x",
    sha256 = "e36b57c784cb7e3c1e8b3fecf195dc1b3f1e32422ae60dd9a6cee152bb45206c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libmount-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e36b57c784cb7e3c1e8b3fecf195dc1b3f1e32422ae60dd9a6cee152bb45206c",
    ],
)

rpm(
    name = "libmount-0__2.40.2-20.el10.x86_64",
    sha256 = "44f67b8aeedf32d9b399aabd9d199c4a96fc4421087e4304bc545c27296ec8c9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libmount-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/44f67b8aeedf32d9b399aabd9d199c4a96fc4421087e4304bc545c27296ec8c9",
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
        "https://storage.googleapis.com/builddeps/bb46a7465559a26c085bf1c02f0764332430a6c1b8fb3f08c8cee184e3d1f02a",
    ],
)

rpm(
    name = "libmpc-0__1.3.1-7.el10.s390x",
    sha256 = "ad956e3c217ba500101acb4219f4e07390ca5ac8a14f99ca9cca85220b525da1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libmpc-1.3.1-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ad956e3c217ba500101acb4219f4e07390ca5ac8a14f99ca9cca85220b525da1",
    ],
)

rpm(
    name = "libmpc-0__1.3.1-7.el10.x86_64",
    sha256 = "daaa73a35dfe21a8201581e333b79ccd296ae87a93f9796ba522e58edc23777c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libmpc-1.3.1-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/daaa73a35dfe21a8201581e333b79ccd296ae87a93f9796ba522e58edc23777c",
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
    name = "libnbd-0__1.24.1-1.el10.aarch64",
    sha256 = "0c10ff0bfd22ba853ab4d21287afa51791a9ce1302176ff63d502d38d7334277",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libnbd-1.24.1-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0c10ff0bfd22ba853ab4d21287afa51791a9ce1302176ff63d502d38d7334277",
    ],
)

rpm(
    name = "libnbd-0__1.24.1-1.el10.s390x",
    sha256 = "223da7e9c0b17ca7b0a831d7222b38247d801aa282463e110ca25142fcc2b075",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libnbd-1.24.1-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/223da7e9c0b17ca7b0a831d7222b38247d801aa282463e110ca25142fcc2b075",
    ],
)

rpm(
    name = "libnbd-0__1.24.1-1.el10.x86_64",
    sha256 = "9bd98e418aa33430c4c576c8ae13c5141e64960a21a7fcd4ea13ed0eee5ddda4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libnbd-1.24.1-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9bd98e418aa33430c4c576c8ae13c5141e64960a21a7fcd4ea13ed0eee5ddda4",
    ],
)

rpm(
    name = "libnbd-devel-0__1.20.3-4.el9.aarch64",
    sha256 = "cb38d0fc674e15a84caa1606fdb7430ba3c0e61bfc2a7dd3a9719ff82dfa920f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libnbd-devel-1.20.3-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cb38d0fc674e15a84caa1606fdb7430ba3c0e61bfc2a7dd3a9719ff82dfa920f",
    ],
)

rpm(
    name = "libnbd-devel-0__1.20.3-4.el9.s390x",
    sha256 = "3289f3e0c7d6f290c79faca35a80b3b3170941bdd478468d339c692c3e35c882",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libnbd-devel-1.20.3-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3289f3e0c7d6f290c79faca35a80b3b3170941bdd478468d339c692c3e35c882",
    ],
)

rpm(
    name = "libnbd-devel-0__1.20.3-4.el9.x86_64",
    sha256 = "88dbe3a521391c371075dc20f70de464004ea8d3bca120f5fa8fdd66ef244847",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libnbd-devel-1.20.3-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/88dbe3a521391c371075dc20f70de464004ea8d3bca120f5fa8fdd66ef244847",
    ],
)

rpm(
    name = "libnbd-devel-0__1.24.1-1.el10.aarch64",
    sha256 = "f4e05d53cf0e8c6d062aa0deda4704eff44160025ed8ad75a382fe1582a99f51",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/aarch64/os/Packages/libnbd-devel-1.24.1-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f4e05d53cf0e8c6d062aa0deda4704eff44160025ed8ad75a382fe1582a99f51",
    ],
)

rpm(
    name = "libnbd-devel-0__1.24.1-1.el10.s390x",
    sha256 = "93b7164eda1d37d47cc19f75b942cb9054c02efd4c0d5813933a3a0fcbef00db",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/libnbd-devel-1.24.1-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/93b7164eda1d37d47cc19f75b942cb9054c02efd4c0d5813933a3a0fcbef00db",
    ],
)

rpm(
    name = "libnbd-devel-0__1.24.1-1.el10.x86_64",
    sha256 = "dd4d89a9a64d82e468bd9686cc420d4266294f358a6ac3d267d136fa341a4f77",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/libnbd-devel-1.24.1-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd4d89a9a64d82e468bd9686cc420d4266294f358a6ac3d267d136fa341a4f77",
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
        "https://storage.googleapis.com/builddeps/53c1b45e66ef040f6486052395cfda198d9b8b3058834ae7e8b7864b04f9c766",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-12.el10.s390x",
    sha256 = "f7f15dc88c380db9cbf6e62c1a04b872867d7f5a4f2cc0b2f9988db963d8401d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnetfilter_conntrack-1.0.9-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f7f15dc88c380db9cbf6e62c1a04b872867d7f5a4f2cc0b2f9988db963d8401d",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.9-12.el10.x86_64",
    sha256 = "71af0b9fb8b790e3d471a74ef463dc3cbb0267c9bdbaf876160fa9821f63200f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnetfilter_conntrack-1.0.9-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/71af0b9fb8b790e3d471a74ef463dc3cbb0267c9bdbaf876160fa9821f63200f",
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
        "https://storage.googleapis.com/builddeps/0ac9c6ebd2c5652bea632435fd73bfb36785d2f15eb840ca53049d6bb3abe639",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.2-3.el10.s390x",
    sha256 = "433915ab5525daf404ae37d47f1f735d257a33ff8a2565741e3449d5b39b0cab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnfnetlink-1.0.2-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/433915ab5525daf404ae37d47f1f735d257a33ff8a2565741e3449d5b39b0cab",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.2-3.el10.x86_64",
    sha256 = "2988f90762058160e4071b79b96523901a2170bc6242488878ce51fbc0d871ca",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.2-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2988f90762058160e4071b79b96523901a2170bc6242488878ce51fbc0d871ca",
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
    name = "libnftnl-0__1.3.0-3.el10.aarch64",
    sha256 = "d5467150ed76b14237d86554566ef07c0025d59f0ebc34348b503cc823784c80",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnftnl-1.3.0-3.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d5467150ed76b14237d86554566ef07c0025d59f0ebc34348b503cc823784c80",
    ],
)

rpm(
    name = "libnftnl-0__1.3.0-3.el10.s390x",
    sha256 = "b7f11ace02687c972f83c414b2a7a9e9a58979fa4630f4aae50608c686145a54",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnftnl-1.3.0-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b7f11ace02687c972f83c414b2a7a9e9a58979fa4630f4aae50608c686145a54",
    ],
)

rpm(
    name = "libnftnl-0__1.3.0-3.el10.x86_64",
    sha256 = "4721d71306d4330cac0868ea25e2f0dfd816fc3c3e618c91212eb9a854322e60",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnftnl-1.3.0-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4721d71306d4330cac0868ea25e2f0dfd816fc3c3e618c91212eb9a854322e60",
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
    name = "libnghttp2-0__1.43.0-7.el9.aarch64",
    sha256 = "7702676980b7c34cc834be8da466c0381f846ca00d7e4bf41d54be77795c1027",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.43.0-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7702676980b7c34cc834be8da466c0381f846ca00d7e4bf41d54be77795c1027",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-7.el9.s390x",
    sha256 = "6ce8782fd5fd6484df8206ad3f90d2f6b278ffcca82d5f2eab98a583f33563ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libnghttp2-1.43.0-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6ce8782fd5fd6484df8206ad3f90d2f6b278ffcca82d5f2eab98a583f33563ed",
    ],
)

rpm(
    name = "libnghttp2-0__1.43.0-7.el9.x86_64",
    sha256 = "2966ee44488ecd822e67ae030eeea4dc19b0323fa9f3da1fbd35dbbb42bc50aa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.43.0-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2966ee44488ecd822e67ae030eeea4dc19b0323fa9f3da1fbd35dbbb42bc50aa",
    ],
)

rpm(
    name = "libnghttp2-0__1.68.0-4.el10.aarch64",
    sha256 = "e6fa5c18018f1b6c07f62e67f6cb758bc50572d4f0a7c57b993dc442d8c1db8d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.68.0-4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e6fa5c18018f1b6c07f62e67f6cb758bc50572d4f0a7c57b993dc442d8c1db8d",
    ],
)

rpm(
    name = "libnghttp2-0__1.68.0-4.el10.s390x",
    sha256 = "391030b3996fd16e8f96c4af8949b909eef2150ccf3b7356cf8c41249e1d5290",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnghttp2-1.68.0-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/391030b3996fd16e8f96c4af8949b909eef2150ccf3b7356cf8c41249e1d5290",
    ],
)

rpm(
    name = "libnghttp2-0__1.68.0-4.el10.x86_64",
    sha256 = "6a092502fdd243940576cd79bbe99e94f1e6791357d2dc85ad51624658869d02",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.68.0-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a092502fdd243940576cd79bbe99e94f1e6791357d2dc85ad51624658869d02",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el10.aarch64",
    sha256 = "b27497d441cd6ae6fbf6a077913eff334c64ace7b6030d6c11a221316a8c8d92",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libnl3-3.11.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b27497d441cd6ae6fbf6a077913eff334c64ace7b6030d6c11a221316a8c8d92",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el10.s390x",
    sha256 = "4171ef398ec29504e828c461581fa44e6afe6b36ff84da66f902edd932271b34",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libnl3-3.11.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4171ef398ec29504e828c461581fa44e6afe6b36ff84da66f902edd932271b34",
    ],
)

rpm(
    name = "libnl3-0__3.11.0-1.el10.x86_64",
    sha256 = "886324f9d4b8c95a46d5c77c0f5cd90051c83462e78ca752c70cc709f5a01d90",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libnl3-3.11.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/886324f9d4b8c95a46d5c77c0f5cd90051c83462e78ca752c70cc709f5a01d90",
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
        "https://storage.googleapis.com/builddeps/3ae1d93313b6b63d6cf2887307a539e9a371d64134ad7a363d5b31c64ee2734d",
    ],
)

rpm(
    name = "libosinfo-0__1.11.0-8.el10.x86_64",
    sha256 = "e632610473056869e88ddaa33511e043721d89f8c2216590a274e638f64e98fa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libosinfo-1.11.0-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e632610473056869e88ddaa33511e043721d89f8c2216590a274e638f64e98fa",
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
        "https://storage.googleapis.com/builddeps/f15a71822ba8269643911bdd5455e2c24c2489927217d3512c9453a6ff8af5bf",
    ],
)

rpm(
    name = "libpcap-14__1.10.4-7.el10.s390x",
    sha256 = "98f721f0b8b731a77d27bcb3178c7297e84372095a452ab52a98192c76953d3f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpcap-1.10.4-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/98f721f0b8b731a77d27bcb3178c7297e84372095a452ab52a98192c76953d3f",
    ],
)

rpm(
    name = "libpcap-14__1.10.4-7.el10.x86_64",
    sha256 = "a933eb7fba1535c9df52f7e44504535b25f8b6fb79c5cf68a0b6e80eb4b9dbf8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpcap-1.10.4-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a933eb7fba1535c9df52f7e44504535b25f8b6fb79c5cf68a0b6e80eb4b9dbf8",
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
        "https://storage.googleapis.com/builddeps/fc2db71a801f4cd03425463d0aea745da36837f25d8cc2042eb747c8a336f989",
    ],
)

rpm(
    name = "libpkgconf-0__2.1.0-3.el10.s390x",
    sha256 = "7d503fcdd8154231531ec1e076ac2552b9d0a5fe096fb50d3a9ff0ebce07d92d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpkgconf-2.1.0-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7d503fcdd8154231531ec1e076ac2552b9d0a5fe096fb50d3a9ff0ebce07d92d",
    ],
)

rpm(
    name = "libpkgconf-0__2.1.0-3.el10.x86_64",
    sha256 = "813f59114413d5e14fc566262ee3d4b56b621beacbe40eda6f28d31f464de1a6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpkgconf-2.1.0-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/813f59114413d5e14fc566262ee3d4b56b621beacbe40eda6f28d31f464de1a6",
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
    urls = ["https://storage.googleapis.com/builddeps/b3f3a689918dc50a9bc41c33abf1a36bdb8e4a707daac77a91e0814407b07ae3"],
)

rpm(
    name = "libpng-2__1.6.37-17.el9.aarch64",
    sha256 = "7dc02f8279ac6310a1b6b7da7e30aad988289f0199325c508361c7a09569ba4e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libpng-1.6.37-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7dc02f8279ac6310a1b6b7da7e30aad988289f0199325c508361c7a09569ba4e",
    ],
)

rpm(
    name = "libpng-2__1.6.37-17.el9.s390x",
    sha256 = "9553cad040d3b6c4b972392148365b82ba117a11d33113a24bf63d1d8094afb4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libpng-1.6.37-17.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9553cad040d3b6c4b972392148365b82ba117a11d33113a24bf63d1d8094afb4",
    ],
)

rpm(
    name = "libpng-2__1.6.37-17.el9.x86_64",
    sha256 = "d9b73980df44529d8abf0be618640cbb0651ca5c8dd545f1612ed35c6fd74fab",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libpng-1.6.37-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d9b73980df44529d8abf0be618640cbb0651ca5c8dd545f1612ed35c6fd74fab",
    ],
)

rpm(
    name = "libpng-2__1.6.40-13.el10.aarch64",
    sha256 = "e6c2e27d169c0680fe57d9c18f98557deb4fe0bcf850ce3cff078198b3f49211",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libpng-1.6.40-13.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e6c2e27d169c0680fe57d9c18f98557deb4fe0bcf850ce3cff078198b3f49211",
    ],
)

rpm(
    name = "libpng-2__1.6.40-13.el10.s390x",
    sha256 = "cf6422feb6f79e476e1989e1de703c6c6543ea9aefb767767ea2e602eeb4028f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libpng-1.6.40-13.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cf6422feb6f79e476e1989e1de703c6c6543ea9aefb767767ea2e602eeb4028f",
    ],
)

rpm(
    name = "libpng-2__1.6.40-13.el10.x86_64",
    sha256 = "423c631be6dc2149783917b27c9f4afad7346aa25a9c37944b8a16ca8abc3b96",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpng-1.6.40-13.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/423c631be6dc2149783917b27c9f4afad7346aa25a9c37944b8a16ca8abc3b96",
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
        "https://storage.googleapis.com/builddeps/ef89d923a60bc9658e5524a80960a865d805aa136b7dd3761a162d58b2aff46d",
    ],
)

rpm(
    name = "libpsl-0__0.21.5-6.el10.x86_64",
    sha256 = "1dca94a85aabd9730bc731fa8a6abb138fec28b75c6a39694d862135c2ade0f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpsl-0.21.5-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1dca94a85aabd9730bc731fa8a6abb138fec28b75c6a39694d862135c2ade0f3",
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
        "https://storage.googleapis.com/builddeps/0d0d6a0e741f94889796b551935f72cf551587067f0c9b64531b5c34b03ab1d8",
    ],
)

rpm(
    name = "libpwquality-0__1.4.5-12.el10.s390x",
    sha256 = "bff94322487bd0bd36640c27e56d7b0167187772cab630758bc56aada0038aea",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libpwquality-1.4.5-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bff94322487bd0bd36640c27e56d7b0167187772cab630758bc56aada0038aea",
    ],
)

rpm(
    name = "libpwquality-0__1.4.5-12.el10.x86_64",
    sha256 = "eda9e6acc99c2c9fa058a9db428da1b0c7441f2be174b9aa7f1628359e36e6ab",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.5-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eda9e6acc99c2c9fa058a9db428da1b0c7441f2be174b9aa7f1628359e36e6ab",
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
        "https://storage.googleapis.com/builddeps/322aa4ea140a63645c7f086b58a08346617eea2efee8044287b76373d633b65f",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el10.s390x",
    sha256 = "e652de14f8c0d52480c2bba779daf5da7b7fd66e65090e1781f41d3bee250840",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libseccomp-2.5.6-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e652de14f8c0d52480c2bba779daf5da7b7fd66e65090e1781f41d3bee250840",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el10.x86_64",
    sha256 = "654051862cc301ed43501ab36b687ed5adeb3ca57689f54a80bf760ad9686e54",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libseccomp-2.5.6-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/654051862cc301ed43501ab36b687ed5adeb3ca57689f54a80bf760ad9686e54",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el9.aarch64",
    sha256 = "74a99b069ffe2fdd6f2ee19c73197c0ad1b71353df39c5af8c404932a5817974",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libseccomp-2.5.6-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/74a99b069ffe2fdd6f2ee19c73197c0ad1b71353df39c5af8c404932a5817974",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el9.s390x",
    sha256 = "155ef4319fc1fffa926ba688e12cd3d49e616f55474278b5df2e3a75d971d1a8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libseccomp-2.5.6-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/155ef4319fc1fffa926ba688e12cd3d49e616f55474278b5df2e3a75d971d1a8",
    ],
)

rpm(
    name = "libseccomp-0__2.5.6-1.el9.x86_64",
    sha256 = "73779d9eb83b4334fb312a7a6bcf7764780777f168724d7e57f6477fd912ac0a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libseccomp-2.5.6-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/73779d9eb83b4334fb312a7a6bcf7764780777f168724d7e57f6477fd912ac0a",
    ],
)

rpm(
    name = "libselinux-0__3.10-2.el10.aarch64",
    sha256 = "4d89a50881ed6bf60128ecc8d70f5f2ca51fb9b507cd725f38875253758eae19",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libselinux-3.10-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4d89a50881ed6bf60128ecc8d70f5f2ca51fb9b507cd725f38875253758eae19",
    ],
)

rpm(
    name = "libselinux-0__3.10-2.el10.s390x",
    sha256 = "692fa7172f8341ab955e30ac20f15bd34f5f2719fbf56391a675f573edce80e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libselinux-3.10-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/692fa7172f8341ab955e30ac20f15bd34f5f2719fbf56391a675f573edce80e1",
    ],
)

rpm(
    name = "libselinux-0__3.10-2.el10.x86_64",
    sha256 = "4d8f5835000e84e79ae26af7a04c8dcbe664408cccaf2a7a92d0d530dba298ec",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libselinux-3.10-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d8f5835000e84e79ae26af7a04c8dcbe664408cccaf2a7a92d0d530dba298ec",
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
    name = "libselinux-0__3.6-4.el9.aarch64",
    sha256 = "b33fc63c93f3f1194c542c443f6c9b511fa149002fddd527d73e2ee0ddc1f774",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-3.6-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b33fc63c93f3f1194c542c443f6c9b511fa149002fddd527d73e2ee0ddc1f774",
    ],
)

rpm(
    name = "libselinux-0__3.6-4.el9.s390x",
    sha256 = "98e1519df815f0f878f4c49810432c0ee305b1a52bb87c8f979e10570b3e1362",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-3.6-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/98e1519df815f0f878f4c49810432c0ee305b1a52bb87c8f979e10570b3e1362",
    ],
)

rpm(
    name = "libselinux-0__3.6-4.el9.x86_64",
    sha256 = "856d614fa2ba1a9d87ebc1ab78554a62c7fa6b7f37594dd9faaff1aac601ae94",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-3.6-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/856d614fa2ba1a9d87ebc1ab78554a62c7fa6b7f37594dd9faaff1aac601ae94",
    ],
)

rpm(
    name = "libselinux-utils-0__3.10-2.el10.aarch64",
    sha256 = "d216e457cf11f481e9db8e351f43f2cd3fa8ff8e8e7925728a04614dd4e263a0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.10-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d216e457cf11f481e9db8e351f43f2cd3fa8ff8e8e7925728a04614dd4e263a0",
    ],
)

rpm(
    name = "libselinux-utils-0__3.10-2.el10.s390x",
    sha256 = "ca5c1952165ce2750b34a7334446876cae157f5063f607325759a2e36bcc8007",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libselinux-utils-3.10-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ca5c1952165ce2750b34a7334446876cae157f5063f607325759a2e36bcc8007",
    ],
)

rpm(
    name = "libselinux-utils-0__3.10-2.el10.x86_64",
    sha256 = "e9a34cf4727868fb1be4949cdbdbfcc8f1cb43f6c5ccf23fb82fa014de9d1d19",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.10-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e9a34cf4727868fb1be4949cdbdbfcc8f1cb43f6c5ccf23fb82fa014de9d1d19",
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
    name = "libselinux-utils-0__3.6-4.el9.aarch64",
    sha256 = "c7a3c7c94a37095a8c115e810d290c0adc5711ddf30e8a6672f5140fa31c9532",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libselinux-utils-3.6-4.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c7a3c7c94a37095a8c115e810d290c0adc5711ddf30e8a6672f5140fa31c9532",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-4.el9.s390x",
    sha256 = "2c196e504d1300ecf005f3dc0051dc6331e27654a3e9e3d00f7fa50fcc09c710",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libselinux-utils-3.6-4.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2c196e504d1300ecf005f3dc0051dc6331e27654a3e9e3d00f7fa50fcc09c710",
    ],
)

rpm(
    name = "libselinux-utils-0__3.6-4.el9.x86_64",
    sha256 = "8c8dbef25e272d647d58496eaf292cf874b27b329a8e78d0422c45ddd3eadf1a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libselinux-utils-3.6-4.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c8dbef25e272d647d58496eaf292cf874b27b329a8e78d0422c45ddd3eadf1a",
    ],
)

rpm(
    name = "libsemanage-0__3.10-1.el10.aarch64",
    sha256 = "1bcf6028098bf2fbb99b9aa822f1ba67f8701fbdab080ce47ff3b9964f6e20fd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsemanage-3.10-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1bcf6028098bf2fbb99b9aa822f1ba67f8701fbdab080ce47ff3b9964f6e20fd",
    ],
)

rpm(
    name = "libsemanage-0__3.10-1.el10.s390x",
    sha256 = "f9796dbdb7ede125766680a5f32df1a325d8f5a749df6e9bf0e660c5aaa74778",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsemanage-3.10-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f9796dbdb7ede125766680a5f32df1a325d8f5a749df6e9bf0e660c5aaa74778",
    ],
)

rpm(
    name = "libsemanage-0__3.10-1.el10.x86_64",
    sha256 = "3f5fabb9a3e0d90c3d94d4340eece1afe1311209ea8d25b94e1e75cc129b7020",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsemanage-3.10-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f5fabb9a3e0d90c3d94d4340eece1afe1311209ea8d25b94e1e75cc129b7020",
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
    name = "libsepol-0__3.10-2.el10.aarch64",
    sha256 = "bdf018a1bd326aef2cbe47f2b02aa5763f6be061e9f49c105045e5bb91d69a03",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsepol-3.10-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bdf018a1bd326aef2cbe47f2b02aa5763f6be061e9f49c105045e5bb91d69a03",
    ],
)

rpm(
    name = "libsepol-0__3.10-2.el10.s390x",
    sha256 = "d9e6a76b4114868eb655c70eeb48d847c23f4bf61e242519d5b83019c5d43c1a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsepol-3.10-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d9e6a76b4114868eb655c70eeb48d847c23f4bf61e242519d5b83019c5d43c1a",
    ],
)

rpm(
    name = "libsepol-0__3.10-2.el10.x86_64",
    sha256 = "9ccabb603a5bd6a61f72086faff8be53da29d059ff5bac0ba7c113b7cd5dab0e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsepol-3.10-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ccabb603a5bd6a61f72086faff8be53da29d059ff5bac0ba7c113b7cd5dab0e",
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
    name = "libslirp-0__4.7.0-10.el10.aarch64",
    sha256 = "077e56fc67d139c2569bdd6b920777df742773a155425b11401406aac39f4e7b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libslirp-4.7.0-10.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/077e56fc67d139c2569bdd6b920777df742773a155425b11401406aac39f4e7b",
    ],
)

rpm(
    name = "libslirp-0__4.7.0-10.el10.s390x",
    sha256 = "392067b3525f2d603a121d6a2b7e5683c7a903a7be677ffc94fe2f1b278d3a11",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libslirp-4.7.0-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/392067b3525f2d603a121d6a2b7e5683c7a903a7be677ffc94fe2f1b278d3a11",
    ],
)

rpm(
    name = "libslirp-0__4.7.0-10.el10.x86_64",
    sha256 = "bc98bf4c15d226b809474c1237700e4e3158d77c4b7488611599672ac0b570af",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libslirp-4.7.0-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc98bf4c15d226b809474c1237700e4e3158d77c4b7488611599672ac0b570af",
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
    name = "libsmartcols-0__2.37.4-25.el9.aarch64",
    sha256 = "a6c8e44ec15936163ca5075ede209fe4f4ec96a2b8656b517962f4db3f082951",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a6c8e44ec15936163ca5075ede209fe4f4ec96a2b8656b517962f4db3f082951",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-25.el9.s390x",
    sha256 = "b9f7f3209532892849db09656f9c2ccffbdda7c60fe1a0cc0c32d9efaeaf065e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsmartcols-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b9f7f3209532892849db09656f9c2ccffbdda7c60fe1a0cc0c32d9efaeaf065e",
    ],
)

rpm(
    name = "libsmartcols-0__2.37.4-25.el9.x86_64",
    sha256 = "d3cc89b398cd94f8ead47a313ce1988b1b887b065842368b6a994559bca02b28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d3cc89b398cd94f8ead47a313ce1988b1b887b065842368b6a994559bca02b28",
    ],
)

rpm(
    name = "libsmartcols-0__2.40.2-20.el10.aarch64",
    sha256 = "1138fcdb3d8eaa86dbff32e2693076bac23ce56530a8d395447b159b93899e76",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1138fcdb3d8eaa86dbff32e2693076bac23ce56530a8d395447b159b93899e76",
    ],
)

rpm(
    name = "libsmartcols-0__2.40.2-20.el10.s390x",
    sha256 = "bbddb6193dab96605facea603e9446a03fe865f1c4fc0d575355da20f5fb241e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsmartcols-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bbddb6193dab96605facea603e9446a03fe865f1c4fc0d575355da20f5fb241e",
    ],
)

rpm(
    name = "libsmartcols-0__2.40.2-20.el10.x86_64",
    sha256 = "92a60d8d706d57bdc0b6c18f022b31f490235040aaa6281299381b887291e52c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/92a60d8d706d57bdc0b6c18f022b31f490235040aaa6281299381b887291e52c",
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
    name = "libsoup-0__2.72.0-17.el9.s390x",
    sha256 = "ffc16a07b7fd75142ce013066456cf2428d488de4ca4debbc94343e15e0218ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libsoup-2.72.0-17.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ffc16a07b7fd75142ce013066456cf2428d488de4ca4debbc94343e15e0218ce",
    ],
)

rpm(
    name = "libsoup-0__2.72.0-17.el9.x86_64",
    sha256 = "13f038612ba8ee5e1fe7781f8ec124e9a24539556d5b2535b240dd4fc6bb821d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libsoup-2.72.0-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/13f038612ba8ee5e1fe7781f8ec124e9a24539556d5b2535b240dd4fc6bb821d",
    ],
)

rpm(
    name = "libsoup3-0__3.6.6-1.el10.s390x",
    sha256 = "78c7d2d698876a94e6eee4071f837ce5f0f3834a55a65cfd3c06ae396f0e0a6c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libsoup3-3.6.6-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/78c7d2d698876a94e6eee4071f837ce5f0f3834a55a65cfd3c06ae396f0e0a6c",
    ],
)

rpm(
    name = "libsoup3-0__3.6.6-1.el10.x86_64",
    sha256 = "999361f436e93fabc26ab695ba0b7b66bc7afa9116090dfe5e44e94007123ff0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libsoup3-3.6.6-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/999361f436e93fabc26ab695ba0b7b66bc7afa9116090dfe5e44e94007123ff0",
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
        "https://storage.googleapis.com/builddeps/edb0a7af06913af8ce5a72ab62de780b8b08ad7a7db6cabc4bae9b95d4253607",
    ],
)

rpm(
    name = "libss-0__1.47.1-5.el10.s390x",
    sha256 = "1a38ebb62511e39bae10e1cbde35152562e90d8014f243696bed5c3deef6bf9f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libss-1.47.1-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1a38ebb62511e39bae10e1cbde35152562e90d8014f243696bed5c3deef6bf9f",
    ],
)

rpm(
    name = "libss-0__1.47.1-5.el10.x86_64",
    sha256 = "2dea843b06f0bd161807d8cfd7e3ef05f5944f5ea57332e5b58700bd93262765",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libss-1.47.1-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2dea843b06f0bd161807d8cfd7e3ef05f5944f5ea57332e5b58700bd93262765",
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
    name = "libssh-0__0.10.4-18.el9.aarch64",
    sha256 = "ad1d0008dc4b2e1e211c62f190129396f054b5c67233dce61fef8165991be9fc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-0.10.4-18.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad1d0008dc4b2e1e211c62f190129396f054b5c67233dce61fef8165991be9fc",
    ],
)

rpm(
    name = "libssh-0__0.10.4-18.el9.s390x",
    sha256 = "b9a86fb974893d1f7f973e706ea04fab2a03c68773270da126c2712e171090e5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libssh-0.10.4-18.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b9a86fb974893d1f7f973e706ea04fab2a03c68773270da126c2712e171090e5",
    ],
)

rpm(
    name = "libssh-0__0.10.4-18.el9.x86_64",
    sha256 = "d3ebec2c728844706676568eaf049c84d5731c741456809585b7507377b18625",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-0.10.4-18.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d3ebec2c728844706676568eaf049c84d5731c741456809585b7507377b18625",
    ],
)

rpm(
    name = "libssh-0__0.12.0-2.el10.aarch64",
    sha256 = "3c3d290af7c115d4021a7181e762dcfe9b877f86283cdf0ae84a6a89870ba391",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libssh-0.12.0-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3c3d290af7c115d4021a7181e762dcfe9b877f86283cdf0ae84a6a89870ba391",
    ],
)

rpm(
    name = "libssh-0__0.12.0-2.el10.s390x",
    sha256 = "94a3568a4e8767a31ca1e199c6c7ec8a8557a2e7373994247963753f1acc6b51",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libssh-0.12.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/94a3568a4e8767a31ca1e199c6c7ec8a8557a2e7373994247963753f1acc6b51",
    ],
)

rpm(
    name = "libssh-0__0.12.0-2.el10.x86_64",
    sha256 = "4fff653207041b86cb0aaf4d8617e1768a71674bc4aedf848617cb509a5fe867",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libssh-0.12.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4fff653207041b86cb0aaf4d8617e1768a71674bc4aedf848617cb509a5fe867",
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
    name = "libssh-config-0__0.10.4-18.el9.aarch64",
    sha256 = "76ac00246277076bafcc2adaf2a8d0b6eba2dbc175cd99fe58b936ee222ef22c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libssh-config-0.10.4-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76ac00246277076bafcc2adaf2a8d0b6eba2dbc175cd99fe58b936ee222ef22c",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-18.el9.s390x",
    sha256 = "76ac00246277076bafcc2adaf2a8d0b6eba2dbc175cd99fe58b936ee222ef22c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libssh-config-0.10.4-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76ac00246277076bafcc2adaf2a8d0b6eba2dbc175cd99fe58b936ee222ef22c",
    ],
)

rpm(
    name = "libssh-config-0__0.10.4-18.el9.x86_64",
    sha256 = "76ac00246277076bafcc2adaf2a8d0b6eba2dbc175cd99fe58b936ee222ef22c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libssh-config-0.10.4-18.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/76ac00246277076bafcc2adaf2a8d0b6eba2dbc175cd99fe58b936ee222ef22c",
    ],
)

rpm(
    name = "libssh-config-0__0.12.0-2.el10.aarch64",
    sha256 = "0f026e18f36d57bb55567368920420a28f5f7384612fa0348d154241f4171b42",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libssh-config-0.12.0-2.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0f026e18f36d57bb55567368920420a28f5f7384612fa0348d154241f4171b42",
    ],
)

rpm(
    name = "libssh-config-0__0.12.0-2.el10.s390x",
    sha256 = "0f026e18f36d57bb55567368920420a28f5f7384612fa0348d154241f4171b42",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libssh-config-0.12.0-2.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0f026e18f36d57bb55567368920420a28f5f7384612fa0348d154241f4171b42",
    ],
)

rpm(
    name = "libssh-config-0__0.12.0-2.el10.x86_64",
    sha256 = "0f026e18f36d57bb55567368920420a28f5f7384612fa0348d154241f4171b42",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libssh-config-0.12.0-2.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0f026e18f36d57bb55567368920420a28f5f7384612fa0348d154241f4171b42",
    ],
)

rpm(
    name = "libsss_idmap-0__2.13.0-1.el10.aarch64",
    sha256 = "8a2bbe553d42d1290cf3d6f0247e5e953623da8682a7caa15dfd86afd3f358e1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.13.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8a2bbe553d42d1290cf3d6f0247e5e953623da8682a7caa15dfd86afd3f358e1",
    ],
)

rpm(
    name = "libsss_idmap-0__2.13.0-1.el10.s390x",
    sha256 = "9e4f60a12b6a243f296eed56e368c1f6c90fcc7a311589051a2ff21411f5bc77",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.13.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9e4f60a12b6a243f296eed56e368c1f6c90fcc7a311589051a2ff21411f5bc77",
    ],
)

rpm(
    name = "libsss_idmap-0__2.13.0-1.el10.x86_64",
    sha256 = "fd9a13bdc32c10e7ee8aab203111764ffb8dc995b45a4a1a281040b79cc6e0d0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.13.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fd9a13bdc32c10e7ee8aab203111764ffb8dc995b45a4a1a281040b79cc6e0d0",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.9-3.el9.aarch64",
    sha256 = "3a78abfde3c2242ec520dd2607f25a73c266e8944a5da8b9dca7b6bc5625529e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_idmap-2.9.9-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3a78abfde3c2242ec520dd2607f25a73c266e8944a5da8b9dca7b6bc5625529e",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.9-3.el9.s390x",
    sha256 = "e065bcc9c6af5af7a73c7fab765543330892e888717f58d3e5aa5e948dbb466a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_idmap-2.9.9-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e065bcc9c6af5af7a73c7fab765543330892e888717f58d3e5aa5e948dbb466a",
    ],
)

rpm(
    name = "libsss_idmap-0__2.9.9-3.el9.x86_64",
    sha256 = "1e393b302c744e64f94e23d0832d1d537dce3ed361bdbd38c5cf592a877e896b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_idmap-2.9.9-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e393b302c744e64f94e23d0832d1d537dce3ed361bdbd38c5cf592a877e896b",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.13.0-1.el10.aarch64",
    sha256 = "0b7beca62932fc779be2c8fed68d8d304bc5050a5774aaf3e9eff033b078e054",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.13.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0b7beca62932fc779be2c8fed68d8d304bc5050a5774aaf3e9eff033b078e054",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.13.0-1.el10.s390x",
    sha256 = "cfa3c73fdd5caab50914a55f3737691782b0e3fc6fb3d13b20e3b94058e8a855",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.13.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cfa3c73fdd5caab50914a55f3737691782b0e3fc6fb3d13b20e3b94058e8a855",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.13.0-1.el10.x86_64",
    sha256 = "ae951fa828b492ccf74005ebbcafc00ee75d388ff4a0f95520cf4f4e4327b455",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.13.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ae951fa828b492ccf74005ebbcafc00ee75d388ff4a0f95520cf4f4e4327b455",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.9-3.el9.aarch64",
    sha256 = "19db46c992aa144e84c918f6cc5bedb0818b7dd0590c6b7402464f3e57111548",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libsss_nss_idmap-2.9.9-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19db46c992aa144e84c918f6cc5bedb0818b7dd0590c6b7402464f3e57111548",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.9-3.el9.s390x",
    sha256 = "d45eedb35de04f80f5fb680580dc4fb08dfdf417855a1487510a48b65f8f433d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libsss_nss_idmap-2.9.9-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d45eedb35de04f80f5fb680580dc4fb08dfdf417855a1487510a48b65f8f433d",
    ],
)

rpm(
    name = "libsss_nss_idmap-0__2.9.9-3.el9.x86_64",
    sha256 = "52d5caf6563701a1cfb641fa542340b285c8c7593ea72d5b9e61fab8c6a256ed",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libsss_nss_idmap-2.9.9-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/52d5caf6563701a1cfb641fa542340b285c8c7593ea72d5b9e61fab8c6a256ed",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-15.el9.aarch64",
    sha256 = "ff179801d1aebc179103fe94c5b4d2455f1e3efb7b4e0423dd125143fd720d55",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libstdc++-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ff179801d1aebc179103fe94c5b4d2455f1e3efb7b4e0423dd125143fd720d55",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-15.el9.s390x",
    sha256 = "9c0197756b9b25906f65ab12230bf3577e55a934d2a81a257638e956b2bae684",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libstdc++-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9c0197756b9b25906f65ab12230bf3577e55a934d2a81a257638e956b2bae684",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-15.el9.x86_64",
    sha256 = "aef7b17304d056eb7cbd07ecf8cb75e10de85b010b15905d3127d06bdf045ffe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libstdc++-11.5.0-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aef7b17304d056eb7cbd07ecf8cb75e10de85b010b15905d3127d06bdf045ffe",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__11.5.0-5.el9.x86_64",
    sha256 = "6628a0027a113c8687d0cd52ed5725ee6cb1ee2a02897349289d683fc6453223",
    urls = ["https://storage.googleapis.com/builddeps/6628a0027a113c8687d0cd52ed5725ee6cb1ee2a02897349289d683fc6453223"],
)

rpm(
    name = "libstdc__plus____plus__-0__14.3.1-4.4.el10.aarch64",
    sha256 = "315a351328eee84276c8a8d955b815ec952a8644af3bd16e51fa38d3b48fa6a5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libstdc++-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/315a351328eee84276c8a8d955b815ec952a8644af3bd16e51fa38d3b48fa6a5",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__14.3.1-4.4.el10.s390x",
    sha256 = "8fc9f5265996b38b2f60128c3c77372db20a9d86aeab260e6c6d391d4d5c478b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libstdc++-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8fc9f5265996b38b2f60128c3c77372db20a9d86aeab260e6c6d391d4d5c478b",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__14.3.1-4.4.el10.x86_64",
    sha256 = "b218d06d603585ed9e29b5aae21ecaafa2619620fa56f5a8afb3a010252a32ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libstdc++-14.3.1-4.4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b218d06d603585ed9e29b5aae21ecaafa2619620fa56f5a8afb3a010252a32ba",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-10.el9.aarch64",
    sha256 = "18fee5d9b7dc486f774d1fac61238a6d6ac1a2dbdf61fdc38496838015e61712",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libtasn1-4.16.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/18fee5d9b7dc486f774d1fac61238a6d6ac1a2dbdf61fdc38496838015e61712",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-10.el9.s390x",
    sha256 = "ce71d8eb0cfb625616683e3db2db40bcb8bb7506c46dbea6097c2cc2b2d360fe",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libtasn1-4.16.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ce71d8eb0cfb625616683e3db2db40bcb8bb7506c46dbea6097c2cc2b2d360fe",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-10.el9.x86_64",
    sha256 = "05f75ceb9f083ec511756eb9ed4078368c56ad55a6fe0abb819b8948e50b0d90",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libtasn1-4.16.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/05f75ceb9f083ec511756eb9ed4078368c56ad55a6fe0abb819b8948e50b0d90",
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
    name = "libtasn1-0__4.20.0-5.el10.aarch64",
    sha256 = "7620003f6a891292015f8db155c6d9c0717deb2f5ebfc30821da519637825ecc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libtasn1-4.20.0-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7620003f6a891292015f8db155c6d9c0717deb2f5ebfc30821da519637825ecc",
    ],
)

rpm(
    name = "libtasn1-0__4.20.0-5.el10.s390x",
    sha256 = "04bac1902d0e9dbec5e67637b0952ff8f274b86e0d226fae09351675c7058b6f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libtasn1-4.20.0-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/04bac1902d0e9dbec5e67637b0952ff8f274b86e0d226fae09351675c7058b6f",
    ],
)

rpm(
    name = "libtasn1-0__4.20.0-5.el10.x86_64",
    sha256 = "6caa5fabfeda372b6cadce6329b3a9132ccde0cc1d1d2999a1fb968b052cf5e7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libtasn1-4.20.0-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6caa5fabfeda372b6cadce6329b3a9132ccde0cc1d1d2999a1fb968b052cf5e7",
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
        "https://storage.googleapis.com/builddeps/6e0345c38ef8c15d2f1743892063241f0273a6b14c1844ab127ad0c085b510e1",
    ],
)

rpm(
    name = "libtirpc-0__1.3.5-1.el10.s390x",
    sha256 = "5f6160af1ea75ef4df15281225e50e13d7de69aad58a7bc08d558ffb88c86086",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libtirpc-1.3.5-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5f6160af1ea75ef4df15281225e50e13d7de69aad58a7bc08d558ffb88c86086",
    ],
)

rpm(
    name = "libtirpc-0__1.3.5-1.el10.x86_64",
    sha256 = "8692d388ed8b7fa6ffe56c9403576ea7b49d55c305e9a64dd44a59fb592fe295",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libtirpc-1.3.5-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8692d388ed8b7fa6ffe56c9403576ea7b49d55c305e9a64dd44a59fb592fe295",
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
        "https://storage.googleapis.com/builddeps/3c666376aabf7fa14a76232e8709a390587bcebfebb24897de6c7c693703fed0",
    ],
)

rpm(
    name = "libtpms-0__0.9.6-11.el10.s390x",
    sha256 = "55e810e2e6a3c8b166c1fa48a38e2430a2f3b1587014009f48b88fc17ba2f5c4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libtpms-0.9.6-11.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/55e810e2e6a3c8b166c1fa48a38e2430a2f3b1587014009f48b88fc17ba2f5c4",
    ],
)

rpm(
    name = "libtpms-0__0.9.6-11.el10.x86_64",
    sha256 = "595a554e74b9e9515d2d615b05ed13118f4a9e0c21f1afc74bf5b4e677b56c9a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libtpms-0.9.6-11.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/595a554e74b9e9515d2d615b05ed13118f4a9e0c21f1afc74bf5b4e677b56c9a",
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
    name = "libubsan-0__11.5.0-15.el9.aarch64",
    sha256 = "7fe98d7f12f79698f467fde4e14d13a5909cc44ff49e00e525390d28914f5055",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libubsan-11.5.0-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7fe98d7f12f79698f467fde4e14d13a5909cc44ff49e00e525390d28914f5055",
    ],
)

rpm(
    name = "libubsan-0__11.5.0-15.el9.s390x",
    sha256 = "e862addf532f5f101f0ace2ac53813e145050493c483791e339f167619f8c647",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libubsan-11.5.0-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e862addf532f5f101f0ace2ac53813e145050493c483791e339f167619f8c647",
    ],
)

rpm(
    name = "libubsan-0__14.3.1-4.4.el10.aarch64",
    sha256 = "db7102b99ef2e935f97f0fbf9dff45076c0fd3fb44cce77ef33d17b07302083d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libubsan-14.3.1-4.4.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/db7102b99ef2e935f97f0fbf9dff45076c0fd3fb44cce77ef33d17b07302083d",
    ],
)

rpm(
    name = "libubsan-0__14.3.1-4.4.el10.s390x",
    sha256 = "79db4832af64077ba6546b8aa25dac7821f6292f65e8090df960eaa0648f5a80",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libubsan-14.3.1-4.4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/79db4832af64077ba6546b8aa25dac7821f6292f65e8090df960eaa0648f5a80",
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
        "https://storage.googleapis.com/builddeps/aa793b61f51cb8727c37520bc4b261845831b9a5789649a798c4e8a2cc207f4f",
    ],
)

rpm(
    name = "libunistring-0__1.1-10.el10.s390x",
    sha256 = "9c45ddec6ffa51201a570b9881e9277b8f22c8eb40ee62b45d3c2b86bdc8eeac",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libunistring-1.1-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/9c45ddec6ffa51201a570b9881e9277b8f22c8eb40ee62b45d3c2b86bdc8eeac",
    ],
)

rpm(
    name = "libunistring-0__1.1-10.el10.x86_64",
    sha256 = "603c06593a43f5766a53588d9ba18855ddb7b238963b8e09d8a328a17959b774",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libunistring-1.1-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/603c06593a43f5766a53588d9ba18855ddb7b238963b8e09d8a328a17959b774",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el10.aarch64",
    sha256 = "29f16a2950ef7ddaae31d7806d98961bf1f7d1772623782ae45cc687a3980c62",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/liburing-2.12-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/29f16a2950ef7ddaae31d7806d98961bf1f7d1772623782ae45cc687a3980c62",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el10.s390x",
    sha256 = "53f3015878c4044a13caf8060a4afa6c10aff93452aa4b8de84cd2374d456a51",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/liburing-2.12-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/53f3015878c4044a13caf8060a4afa6c10aff93452aa4b8de84cd2374d456a51",
    ],
)

rpm(
    name = "liburing-0__2.12-1.el10.x86_64",
    sha256 = "133309fc854ab7859713d7944e5a14e8cbc3f3916bbcd9f9e6af4d4850424c15",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/liburing-2.12-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/133309fc854ab7859713d7944e5a14e8cbc3f3916bbcd9f9e6af4d4850424c15",
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
    name = "libusb1-0__1.0.30-1.el10.aarch64",
    sha256 = "e8a1d29a7b2e19f3fa76ad0d7a3bb01a6264902953bf2bbe0f2e75721d06fd7b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libusb1-1.0.30-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e8a1d29a7b2e19f3fa76ad0d7a3bb01a6264902953bf2bbe0f2e75721d06fd7b",
    ],
)

rpm(
    name = "libusb1-0__1.0.30-1.el10.s390x",
    sha256 = "fc0a07324c5465e31d1cf9f389acf7f69028ccedc3946647fa24df24bbe70460",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libusb1-1.0.30-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fc0a07324c5465e31d1cf9f389acf7f69028ccedc3946647fa24df24bbe70460",
    ],
)

rpm(
    name = "libusb1-0__1.0.30-1.el10.x86_64",
    sha256 = "195b5fd0b84c6214d16544e97146037f1cbad6393829edf5060448213d9021ad",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libusb1-1.0.30-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/195b5fd0b84c6214d16544e97146037f1cbad6393829edf5060448213d9021ad",
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
    name = "libusbx-0__1.0.30-1.el9.aarch64",
    sha256 = "b480be150230167d7b9fb230eca6017471a400587b51cdff52a542b2802fe4f4",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libusbx-1.0.30-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b480be150230167d7b9fb230eca6017471a400587b51cdff52a542b2802fe4f4",
    ],
)

rpm(
    name = "libusbx-0__1.0.30-1.el9.s390x",
    sha256 = "fdebd9892ed1b44bde02308b0017b73c78696f60659a4bc8fd6331c7e9147fcf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libusbx-1.0.30-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fdebd9892ed1b44bde02308b0017b73c78696f60659a4bc8fd6331c7e9147fcf",
    ],
)

rpm(
    name = "libusbx-0__1.0.30-1.el9.x86_64",
    sha256 = "89937542b4af7b56fc1f13a93bff7e601597e77159a0007a929bc01469a739df",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libusbx-1.0.30-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89937542b4af7b56fc1f13a93bff7e601597e77159a0007a929bc01469a739df",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-15.el10.aarch64",
    sha256 = "6444bf715fdd137bd1bd096d9903e29516c609d41113139756df2e9316825d6a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libutempter-1.2.1-15.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6444bf715fdd137bd1bd096d9903e29516c609d41113139756df2e9316825d6a",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-15.el10.s390x",
    sha256 = "1314f6b74597ad5a5a85b51f5243f3802d2f722a5ed5a41a3ebca827eb7d6a6f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libutempter-1.2.1-15.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1314f6b74597ad5a5a85b51f5243f3802d2f722a5ed5a41a3ebca827eb7d6a6f",
    ],
)

rpm(
    name = "libutempter-0__1.2.1-15.el10.x86_64",
    sha256 = "db498c4b6ce6f223597f8ea955fe4e286f4fc5838e81579de877ca0e80c2d6eb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libutempter-1.2.1-15.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/db498c4b6ce6f223597f8ea955fe4e286f4fc5838e81579de877ca0e80c2d6eb",
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
    name = "libuuid-0__2.37.4-21.el9.x86_64",
    sha256 = "be4793be5af11772206abe023746ec4021a8b7bc124fdc7e7cdb92b57c46d125",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-21.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be4793be5af11772206abe023746ec4021a8b7bc124fdc7e7cdb92b57c46d125",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-25.el9.aarch64",
    sha256 = "5e740b232a2ab7deb56916d28ef026f16e3d5d11bedc7ceaa7381717193b3836",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libuuid-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5e740b232a2ab7deb56916d28ef026f16e3d5d11bedc7ceaa7381717193b3836",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-25.el9.s390x",
    sha256 = "608adf99d9ad76624ef9d526748b8f0e95cc682edbe16e11ac22561b690dc0cd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libuuid-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/608adf99d9ad76624ef9d526748b8f0e95cc682edbe16e11ac22561b690dc0cd",
    ],
)

rpm(
    name = "libuuid-0__2.37.4-25.el9.x86_64",
    sha256 = "2305b6ddfd73d94cee66c8071d6ec30f7bd7e91792d76628b008c0d919e0c75e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libuuid-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2305b6ddfd73d94cee66c8071d6ec30f7bd7e91792d76628b008c0d919e0c75e",
    ],
)

rpm(
    name = "libuuid-0__2.40.2-20.el10.aarch64",
    sha256 = "c4f8390e6fed0586e7cb36cbf8bfa6c3b99669d96c680d60e54723d498e8fd8f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libuuid-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c4f8390e6fed0586e7cb36cbf8bfa6c3b99669d96c680d60e54723d498e8fd8f",
    ],
)

rpm(
    name = "libuuid-0__2.40.2-20.el10.s390x",
    sha256 = "629eb0b9b052a44f94e2457fc2d114664b53545b1d6ff385a19e593666050f61",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libuuid-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/629eb0b9b052a44f94e2457fc2d114664b53545b1d6ff385a19e593666050f61",
    ],
)

rpm(
    name = "libuuid-0__2.40.2-20.el10.x86_64",
    sha256 = "8c992f36d651b712f182ce9dc1186f4d2301d645115cc53c4d0615cf8291d89c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libuuid-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c992f36d651b712f182ce9dc1186f4d2301d645115cc53c4d0615cf8291d89c",
    ],
)

rpm(
    name = "libverto-0__0.3.2-10.el10.aarch64",
    sha256 = "0583db7823a8f33a1e09db1e4aa389c10bc98b58de3bd985b6f67be5351d814a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libverto-0.3.2-10.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0583db7823a8f33a1e09db1e4aa389c10bc98b58de3bd985b6f67be5351d814a",
    ],
)

rpm(
    name = "libverto-0__0.3.2-10.el10.s390x",
    sha256 = "80757eae2999d4dbc8975747eb4d8fdfb64b144826ba58215672a0f34d313228",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libverto-0.3.2-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/80757eae2999d4dbc8975747eb4d8fdfb64b144826ba58215672a0f34d313228",
    ],
)

rpm(
    name = "libverto-0__0.3.2-10.el10.x86_64",
    sha256 = "52777e532dc2351c83b72b5033c40df20494afb6504100f7413a65f74368c284",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libverto-0.3.2-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/52777e532dc2351c83b72b5033c40df20494afb6504100f7413a65f74368c284",
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
    name = "libvirt-client-0__11.10.0-12.el9.aarch64",
    sha256 = "58b59ac93cc9e75bb8b00337676526ffedd0d8cc6d8f7a932542dafb340e4582",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-client-11.10.0-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/58b59ac93cc9e75bb8b00337676526ffedd0d8cc6d8f7a932542dafb340e4582",
    ],
)

rpm(
    name = "libvirt-client-0__11.10.0-12.el9.s390x",
    sha256 = "080d62204758598339a3f180edd7669e4f30b6723a5a6a5684b4cfcb6fe7e47e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-client-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/080d62204758598339a3f180edd7669e4f30b6723a5a6a5684b4cfcb6fe7e47e",
    ],
)

rpm(
    name = "libvirt-client-0__11.10.0-12.el9.x86_64",
    sha256 = "01c25c54915ec841add7d151e18a555d0d0d6e7788a35eb3b7f70ac5803ad280",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-client-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/01c25c54915ec841add7d151e18a555d0d0d6e7788a35eb3b7f70ac5803ad280",
    ],
)

rpm(
    name = "libvirt-client-0__12.4.0-1.el10.aarch64",
    sha256 = "22b4047763a8fb70cd3432de6543009c8923c95e5e16eed3cbfede682c65cf7c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-client-12.4.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/22b4047763a8fb70cd3432de6543009c8923c95e5e16eed3cbfede682c65cf7c",
    ],
)

rpm(
    name = "libvirt-client-0__12.4.0-1.el10.s390x",
    sha256 = "6962d093e6bd525e6e35770a7e1b38bc84d1c5f60a6a5e823347adafae3d8e2d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-client-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6962d093e6bd525e6e35770a7e1b38bc84d1c5f60a6a5e823347adafae3d8e2d",
    ],
)

rpm(
    name = "libvirt-client-0__12.4.0-1.el10.x86_64",
    sha256 = "95953c1acf3a3d37b60cc7b53f2b59cae11b1716ed2444366d827b06847e67bb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-client-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95953c1acf3a3d37b60cc7b53f2b59cae11b1716ed2444366d827b06847e67bb",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__10.10.0-7.el9.x86_64",
    sha256 = "ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af",
    urls = ["https://storage.googleapis.com/builddeps/ce303675dd62e81a3d946c15e2938373be0988d9d64e62e620ef846a98be87af"],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-12.el9.aarch64",
    sha256 = "8da50fec7d5e5c4f7bf6c9e5717e400d8e9901cf6ae6bf785d86aa1723b03dbf",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-11.10.0-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8da50fec7d5e5c4f7bf6c9e5717e400d8e9901cf6ae6bf785d86aa1723b03dbf",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-12.el9.s390x",
    sha256 = "8497d1ac8aa90b87363d3d61e42d7fbad74097f441468c8f92aa2bc944f9bd60",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8497d1ac8aa90b87363d3d61e42d7fbad74097f441468c8f92aa2bc944f9bd60",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__11.10.0-12.el9.x86_64",
    sha256 = "e6ba5e43edacc5f3301eb7ec927ad011d464a98175afb56c4142b6becf02b1c0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e6ba5e43edacc5f3301eb7ec927ad011d464a98175afb56c4142b6becf02b1c0",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__12.4.0-1.el10.aarch64",
    sha256 = "385cd1cffa5063ede94ad4e48450313d55b1b9b4efbdd969338200930769b78d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-daemon-common-12.4.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/385cd1cffa5063ede94ad4e48450313d55b1b9b4efbdd969338200930769b78d",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__12.4.0-1.el10.s390x",
    sha256 = "10df2805cf7fcb3a0b55dca1759f16fffb964ca2c3cb3d8c834178c216f9c026",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-common-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/10df2805cf7fcb3a0b55dca1759f16fffb964ca2c3cb3d8c834178c216f9c026",
    ],
)

rpm(
    name = "libvirt-daemon-common-0__12.4.0-1.el10.x86_64",
    sha256 = "e4ccaa39cdf4929b68b17af8616927b68d9176ae8456a76b39f6d78f728d5b71",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-common-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e4ccaa39cdf4929b68b17af8616927b68d9176ae8456a76b39f6d78f728d5b71",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__10.10.0-7.el9.x86_64",
    sha256 = "13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1",
    urls = ["https://storage.googleapis.com/builddeps/13031a6b2bae44c50808b89b820e47879ef6b7884e21e2a0c0e8aba52accd0b1"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-12.el9.aarch64",
    sha256 = "57f744c631e0b3d928faec59a90d5c7194b410d64f7bfd42462b3d770e0afc20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-11.10.0-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/57f744c631e0b3d928faec59a90d5c7194b410d64f7bfd42462b3d770e0afc20",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-12.el9.s390x",
    sha256 = "c3d5aeb515fd40bc96a7f650b2800eeaad14ea7d75f3f500cc0c7245c013a21b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c3d5aeb515fd40bc96a7f650b2800eeaad14ea7d75f3f500cc0c7245c013a21b",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__11.10.0-12.el9.x86_64",
    sha256 = "6f8c2c7624dee63ec5cf4ffff973c8328bdc3104303a6d6ca28f71723de8245f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6f8c2c7624dee63ec5cf4ffff973c8328bdc3104303a6d6ca28f71723de8245f",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__12.4.0-1.el10.aarch64",
    sha256 = "8f15b281abed19a97524dc8a5e08f366fe2e80a38c6de80c5303fd6aa3d7a9dc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-daemon-driver-qemu-12.4.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8f15b281abed19a97524dc8a5e08f366fe2e80a38c6de80c5303fd6aa3d7a9dc",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__12.4.0-1.el10.s390x",
    sha256 = "1db4a1a2e50e4f3e470e22199b9bd210ef5b482276abb91cb92f99dd56f9fcba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-qemu-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1db4a1a2e50e4f3e470e22199b9bd210ef5b482276abb91cb92f99dd56f9fcba",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__12.4.0-1.el10.x86_64",
    sha256 = "c8cb1cc8a813763d059aafd9cc9ae18b6553d4e924da85a9412ddcc94086d31b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-qemu-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c8cb1cc8a813763d059aafd9cc9ae18b6553d4e924da85a9412ddcc94086d31b",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__10.10.0-7.el9.x86_64",
    sha256 = "8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3",
    urls = ["https://storage.googleapis.com/builddeps/8d6d2229cde16e57787fd0125ca75dca31d89008446ff344d577ef3eaefcd0f3"],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.10.0-12.el9.s390x",
    sha256 = "931183d6dac53eee0e85bec4b56cf0e9a83b89bc72afe84f4fc40959de3ed62a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-secret-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/931183d6dac53eee0e85bec4b56cf0e9a83b89bc72afe84f4fc40959de3ed62a",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__11.10.0-12.el9.x86_64",
    sha256 = "b2a17a3f40ca9418ff470f9dded6489762b593408dc6fed153b9cdc71a3fe73d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2a17a3f40ca9418ff470f9dded6489762b593408dc6fed153b9cdc71a3fe73d",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__12.4.0-1.el10.s390x",
    sha256 = "a2fb0039b9e2fcab1f4626d0e40b084fefa1761a12cace1739d9d6488955904c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-secret-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a2fb0039b9e2fcab1f4626d0e40b084fefa1761a12cace1739d9d6488955904c",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__12.4.0-1.el10.x86_64",
    sha256 = "315a0427687d1cd6e295ae87bc3ecdc7f40068abfcf52fc0d0ab0698540910cd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-secret-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/315a0427687d1cd6e295ae87bc3ecdc7f40068abfcf52fc0d0ab0698540910cd",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__10.10.0-7.el9.x86_64",
    sha256 = "a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067",
    urls = ["https://storage.googleapis.com/builddeps/a95615f05b0ca4349c571b5a25c2e7151ae7a2d6e7205b5e5c3be26c89a98067"],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.10.0-12.el9.s390x",
    sha256 = "ffb64201602f20dc35d977146ae7ffe229d1255e95f20df286c62ae36de7beec",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-storage-core-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ffb64201602f20dc35d977146ae7ffe229d1255e95f20df286c62ae36de7beec",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__11.10.0-12.el9.x86_64",
    sha256 = "c42f7e455e3634cf7e44b604d464471f6644cbb8fadec0873777fd076c395dbc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c42f7e455e3634cf7e44b604d464471f6644cbb8fadec0873777fd076c395dbc",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__12.4.0-1.el10.s390x",
    sha256 = "693a4e7dbeb0b74f394af0759bd47a8121cce4ed1768e5a3ed469e47dc4b7296",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-driver-storage-core-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/693a4e7dbeb0b74f394af0759bd47a8121cce4ed1768e5a3ed469e47dc4b7296",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__12.4.0-1.el10.x86_64",
    sha256 = "82057b7951eb703211a79780fe34da13b9c30ae3b5836e3652a1ba3b943b6cbd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-driver-storage-core-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82057b7951eb703211a79780fe34da13b9c30ae3b5836e3652a1ba3b943b6cbd",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__10.10.0-7.el9.x86_64",
    sha256 = "7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff",
    urls = ["https://storage.googleapis.com/builddeps/7fa94e83fcae83614c5c4c95a92f4cb3f0065d8971f4a4025c9fd262e68cddff"],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-12.el9.aarch64",
    sha256 = "c438b4b66bbf5cc7f5c47b62d437d2d1695561c28356b9b994ebeb878ac43d92",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-11.10.0-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c438b4b66bbf5cc7f5c47b62d437d2d1695561c28356b9b994ebeb878ac43d92",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-12.el9.s390x",
    sha256 = "4a55d96fb5a9e757cf08415286b6b07b404105479cac7b701a54f417f9d1c944",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4a55d96fb5a9e757cf08415286b6b07b404105479cac7b701a54f417f9d1c944",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__11.10.0-12.el9.x86_64",
    sha256 = "621cf5dc7025359dd65cc5ac30b1648ac3d67b1e1349d661a3f2b0bf3d0b9ad0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/621cf5dc7025359dd65cc5ac30b1648ac3d67b1e1349d661a3f2b0bf3d0b9ad0",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__12.4.0-1.el10.aarch64",
    sha256 = "77c5c596037c551e1f22e65374c144e4acaac572592b263c61bcf1801773db49",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-daemon-log-12.4.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/77c5c596037c551e1f22e65374c144e4acaac572592b263c61bcf1801773db49",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__12.4.0-1.el10.s390x",
    sha256 = "c691083efbee115b7cd444e8eaf72d21b98acc257dff453ffd9a5da0fb00a636",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-daemon-log-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c691083efbee115b7cd444e8eaf72d21b98acc257dff453ffd9a5da0fb00a636",
    ],
)

rpm(
    name = "libvirt-daemon-log-0__12.4.0-1.el10.x86_64",
    sha256 = "f145a13ef206079df73c15addc98819649d27e1a283f76ebb7f8f055723a7854",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-daemon-log-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f145a13ef206079df73c15addc98819649d27e1a283f76ebb7f8f055723a7854",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-12.el9.aarch64",
    sha256 = "bc51c66d476b47c6e81dc154aa1d88e423017289b1bfa9248fde05a80703cb59",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/aarch64/os/Packages/libvirt-devel-11.10.0-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bc51c66d476b47c6e81dc154aa1d88e423017289b1bfa9248fde05a80703cb59",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-12.el9.s390x",
    sha256 = "3eb2c036a810a842e4ea51133d0f1bd285cc8ff9384774a2832cd7d0439da035",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/s390x/os/Packages/libvirt-devel-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3eb2c036a810a842e4ea51133d0f1bd285cc8ff9384774a2832cd7d0439da035",
    ],
)

rpm(
    name = "libvirt-devel-0__11.10.0-12.el9.x86_64",
    sha256 = "fb1ba88c314d63287c7a2a0198c3e5e6ba2a0c3767e99d677343e5405c4d8a84",
    urls = [
        "http://mirror.stream.centos.org/9-stream/CRB/x86_64/os/Packages/libvirt-devel-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb1ba88c314d63287c7a2a0198c3e5e6ba2a0c3767e99d677343e5405c4d8a84",
    ],
)

rpm(
    name = "libvirt-devel-0__12.4.0-1.el10.aarch64",
    sha256 = "75f5a260cbb24655537c13630021f9b871dfaed43dcdb66cb4d0a799239e0a18",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/aarch64/os/Packages/libvirt-devel-12.4.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/75f5a260cbb24655537c13630021f9b871dfaed43dcdb66cb4d0a799239e0a18",
    ],
)

rpm(
    name = "libvirt-devel-0__12.4.0-1.el10.s390x",
    sha256 = "8d42230a7b94325f821203240f0820318b5ee30016aed52a1e33449b21ab4871",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/libvirt-devel-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8d42230a7b94325f821203240f0820318b5ee30016aed52a1e33449b21ab4871",
    ],
)

rpm(
    name = "libvirt-devel-0__12.4.0-1.el10.x86_64",
    sha256 = "754e2523dd46c326f183e5473efb4d4a75f2878fa9ce53d03b0ea3305f1d85d8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/libvirt-devel-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/754e2523dd46c326f183e5473efb4d4a75f2878fa9ce53d03b0ea3305f1d85d8",
    ],
)

rpm(
    name = "libvirt-libs-0__10.10.0-7.el9.x86_64",
    sha256 = "72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39",
    urls = ["https://storage.googleapis.com/builddeps/72e64da467f4afbff2c96b6e46c779fa3abfaba2ddaf85ad0de6087c3d5ccc39"],
)

rpm(
    name = "libvirt-libs-0__11.10.0-12.el9.aarch64",
    sha256 = "5435685a9cad1b387cfd9366ccf14928d10ebd4d43c3cd101178627ac9bb6a71",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/libvirt-libs-11.10.0-12.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5435685a9cad1b387cfd9366ccf14928d10ebd4d43c3cd101178627ac9bb6a71",
    ],
)

rpm(
    name = "libvirt-libs-0__11.10.0-12.el9.s390x",
    sha256 = "03311af212d61a4d3db541a8972507d7ad4ca53876497d7b53d0def19dc1cf30",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libvirt-libs-11.10.0-12.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/03311af212d61a4d3db541a8972507d7ad4ca53876497d7b53d0def19dc1cf30",
    ],
)

rpm(
    name = "libvirt-libs-0__11.10.0-12.el9.x86_64",
    sha256 = "832b6dabc1b09149353299206082be3ee3233cfbdaa3793b7f6b515f283c9fc3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libvirt-libs-11.10.0-12.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/832b6dabc1b09149353299206082be3ee3233cfbdaa3793b7f6b515f283c9fc3",
    ],
)

rpm(
    name = "libvirt-libs-0__12.4.0-1.el10.aarch64",
    sha256 = "cecbf2811bca392ed0c0b690ba5e0ef43888f5c5f9a93db9e98db20c6ad9ee09",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/libvirt-libs-12.4.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cecbf2811bca392ed0c0b690ba5e0ef43888f5c5f9a93db9e98db20c6ad9ee09",
    ],
)

rpm(
    name = "libvirt-libs-0__12.4.0-1.el10.s390x",
    sha256 = "e0bb5e291ead1a316da42355087af3861dcb4df07ba3ce0e5a9b7e4e16bf327e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libvirt-libs-12.4.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e0bb5e291ead1a316da42355087af3861dcb4df07ba3ce0e5a9b7e4e16bf327e",
    ],
)

rpm(
    name = "libvirt-libs-0__12.4.0-1.el10.x86_64",
    sha256 = "fbd229e4691e12354a531589a8a37e9d2367d500c5828cea3f3a39918ff77763",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libvirt-libs-12.4.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fbd229e4691e12354a531589a8a37e9d2367d500c5828cea3f3a39918ff77763",
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
        "https://storage.googleapis.com/builddeps/465ade16c8f369b5abc1a39671f882bc645ac90b1aeaa29cdfc3958e57640144",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.36-10.el10.s390x",
    sha256 = "d14c5523dd6c7f233277acbbb11fb2644f26e91da18e6184ae6ad445e3835a36",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libxcrypt-4.4.36-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d14c5523dd6c7f233277acbbb11fb2644f26e91da18e6184ae6ad445e3835a36",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.36-10.el10.x86_64",
    sha256 = "503a29c4c767637d810c7e89ed4355fe0b588381cb360517585fb56a2cf5ee46",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libxcrypt-4.4.36-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/503a29c4c767637d810c7e89ed4355fe0b588381cb360517585fb56a2cf5ee46",
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
        "https://storage.googleapis.com/builddeps/2f86c95726f3c3efdcb2d97f5d0020e86d254defebb084df7c13a5fa51442b5a",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.4.36-10.el10.s390x",
    sha256 = "a3f57faa74cefedf8baddec91311a5e0cafe73878e83c3447335495c7ed7934b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libxcrypt-devel-4.4.36-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a3f57faa74cefedf8baddec91311a5e0cafe73878e83c3447335495c7ed7934b",
    ],
)

rpm(
    name = "libxcrypt-devel-0__4.4.36-10.el10.x86_64",
    sha256 = "ccee1b09985e24bfed47cf7b5c965d7e0e869862ac55f3b7f783cdcec93716f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libxcrypt-devel-4.4.36-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ccee1b09985e24bfed47cf7b5c965d7e0e869862ac55f3b7f783cdcec93716f3",
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
        "https://storage.googleapis.com/builddeps/a2d13d4bb5d7ca66384346f0b90801862e9e58e016870565929a699bb1c15feb",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.4.36-10.el10.s390x",
    sha256 = "42d6494724ad9eb96949ac0632b9658b488c5db84d7aa2f9db3d20198a29f9fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/s390x/os/Packages/libxcrypt-static-4.4.36-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/42d6494724ad9eb96949ac0632b9658b488c5db84d7aa2f9db3d20198a29f9fe",
    ],
)

rpm(
    name = "libxcrypt-static-0__4.4.36-10.el10.x86_64",
    sha256 = "a4b6f28908fa252bac1c366f91bb37117c0b56ebce1743933dcf2029f10f86c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/CRB/x86_64/os/Packages/libxcrypt-static-4.4.36-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a4b6f28908fa252bac1c366f91bb37117c0b56ebce1743933dcf2029f10f86c0",
    ],
)

rpm(
    name = "libxml2-0__2.12.5-12.el10.aarch64",
    sha256 = "fb4ae7d4812e5a999d357e25bb10fc3bd8a42ffb92dc384d17059677cc9dca52",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/libxml2-2.12.5-12.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fb4ae7d4812e5a999d357e25bb10fc3bd8a42ffb92dc384d17059677cc9dca52",
    ],
)

rpm(
    name = "libxml2-0__2.12.5-12.el10.s390x",
    sha256 = "ca7706034d06accf2bc4a4890ad550201bb64f9a6f431b4cd52fd93159079fc4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libxml2-2.12.5-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ca7706034d06accf2bc4a4890ad550201bb64f9a6f431b4cd52fd93159079fc4",
    ],
)

rpm(
    name = "libxml2-0__2.12.5-12.el10.x86_64",
    sha256 = "ade0f2dff228a19be75fe4807e967bea4ec660558dabe1879c4a3cd3f7201c0f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libxml2-2.12.5-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ade0f2dff228a19be75fe4807e967bea4ec660558dabe1879c4a3cd3f7201c0f",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-16.el9.aarch64",
    sha256 = "aadb7ca54b54a4d976a6ebb9c6a780e74d33f294a71f9bfb808a3f6a89d5f2f6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.13-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aadb7ca54b54a4d976a6ebb9c6a780e74d33f294a71f9bfb808a3f6a89d5f2f6",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-16.el9.s390x",
    sha256 = "f33487680484e3a9bc67b939862b9f0ad9872db5ab29d33f53d0ee582c21f4ce",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/libxml2-2.9.13-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f33487680484e3a9bc67b939862b9f0ad9872db5ab29d33f53d0ee582c21f4ce",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-16.el9.x86_64",
    sha256 = "66a9bffc0993810538f3532a1964d56e7a073c128a9dbe8e394bbe56cd29f5d6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.13-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/66a9bffc0993810538f3532a1964d56e7a073c128a9dbe8e394bbe56cd29f5d6",
    ],
)

rpm(
    name = "libxml2-0__2.9.13-9.el9.x86_64",
    sha256 = "70b74fdfab02d40caad350cf83bc676a782de69b25beb3d37dc193aaf381d9e0",
    urls = ["https://storage.googleapis.com/builddeps/70b74fdfab02d40caad350cf83bc676a782de69b25beb3d37dc193aaf381d9e0"],
)

rpm(
    name = "libxslt-0__1.1.34-16.el9.s390x",
    sha256 = "7accf4895407cb33ca81659341b9977af5daf769eb72dc81d1f72400fe3148b7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/libxslt-1.1.34-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7accf4895407cb33ca81659341b9977af5daf769eb72dc81d1f72400fe3148b7",
    ],
)

rpm(
    name = "libxslt-0__1.1.34-16.el9.x86_64",
    sha256 = "7695356471d253e2017cf4b3aa56e1d00cca841950bdf01c6963fc6a9e3162dc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/libxslt-1.1.34-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7695356471d253e2017cf4b3aa56e1d00cca841950bdf01c6963fc6a9e3162dc",
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
    name = "libxslt-0__1.1.39-9.el10.s390x",
    sha256 = "12574f78d41c3e456f17ffc82c9e90075a68530ec71ab310ebcf23c1fed869db",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/libxslt-1.1.39-9.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/12574f78d41c3e456f17ffc82c9e90075a68530ec71ab310ebcf23c1fed869db",
    ],
)

rpm(
    name = "libxslt-0__1.1.39-9.el10.x86_64",
    sha256 = "e1186794783a61b71f8e2288d62cbdf154f0099b5a3d814ae0e2e928a3147613",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/libxslt-1.1.39-9.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e1186794783a61b71f8e2288d62cbdf154f0099b5a3d814ae0e2e928a3147613",
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
        "https://storage.googleapis.com/builddeps/474a4497b7901176be4a59895cd02bba744300fd673668ef068bd1dfc5e129c7",
    ],
)

rpm(
    name = "libzstd-0__1.5.5-9.el10.s390x",
    sha256 = "59d29a77a5792bbc4ce42b3ac700a1df776ace058e040f391374f011d39f0eef",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/libzstd-1.5.5-9.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/59d29a77a5792bbc4ce42b3ac700a1df776ace058e040f391374f011d39f0eef",
    ],
)

rpm(
    name = "libzstd-0__1.5.5-9.el10.x86_64",
    sha256 = "86f3cb406d56283119c45ec8c1f4689aa37ff6c04cf44f6608c10cfdcccdb2c1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/libzstd-1.5.5-9.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86f3cb406d56283119c45ec8c1f4689aa37ff6c04cf44f6608c10cfdcccdb2c1",
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
    name = "lua-libs-0__5.4.8-1.el10.s390x",
    sha256 = "caeb2e8f53e19b8a55b6b5a72a708c68756ff5d889658eed4dd1207998f72ca0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lua-libs-5.4.8-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/caeb2e8f53e19b8a55b6b5a72a708c68756ff5d889658eed4dd1207998f72ca0",
    ],
)

rpm(
    name = "lua-libs-0__5.4.8-1.el10.x86_64",
    sha256 = "4dbf818acf497edc275470d16520f121d93a853c5b700d768c13a4300dfbdcaa",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lua-libs-5.4.8-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4dbf818acf497edc275470d16520f121d93a853c5b700d768c13a4300dfbdcaa",
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
        "https://storage.googleapis.com/builddeps/7db176282f02ed0243d66b9136e1269e4db85da61157392ecc0febeac418ec85",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.4-8.el10.s390x",
    sha256 = "bd0ba485141caa931c930540a150a55a89ab3dfc6bba448aa592e5b9551dee2e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lz4-libs-1.9.4-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bd0ba485141caa931c930540a150a55a89ab3dfc6bba448aa592e5b9551dee2e",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.4-8.el10.x86_64",
    sha256 = "de360e857e8465c4b38990375e9435efc78e20d022afe42dbf2986d11fc2c759",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lz4-libs-1.9.4-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de360e857e8465c4b38990375e9435efc78e20d022afe42dbf2986d11fc2c759",
    ],
)

rpm(
    name = "lzo-0__2.10-14.el10.aarch64",
    sha256 = "677b7730dfa8e554a8ddd22940c5c6288b0d51cb09e9547c150905e856fb0575",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/lzo-2.10-14.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/677b7730dfa8e554a8ddd22940c5c6288b0d51cb09e9547c150905e856fb0575",
    ],
)

rpm(
    name = "lzo-0__2.10-14.el10.s390x",
    sha256 = "32bde43a3a00f4b5d078b2c831270f8cc195664e0e3ab5b1c5bcc6dc802e33d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lzo-2.10-14.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/32bde43a3a00f4b5d078b2c831270f8cc195664e0e3ab5b1c5bcc6dc802e33d5",
    ],
)

rpm(
    name = "lzo-0__2.10-14.el10.x86_64",
    sha256 = "9e4f4e6dc19d15eb865805a43f5834b0ce3a405dcc6df0fba72f0b73f59685a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lzo-2.10-14.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9e4f4e6dc19d15eb865805a43f5834b0ce3a405dcc6df0fba72f0b73f59685a2",
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
        "https://storage.googleapis.com/builddeps/e463088918132202d22ada263686b6b723af02b6a49066fd6f9d48cf191cb25e",
    ],
)

rpm(
    name = "lzop-0__1.04-16.el10.s390x",
    sha256 = "5eeeda50a19223224ac6de6428853904e6210b0c11223e71aa39848e613bcb0b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/lzop-1.04-16.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5eeeda50a19223224ac6de6428853904e6210b0c11223e71aa39848e613bcb0b",
    ],
)

rpm(
    name = "lzop-0__1.04-16.el10.x86_64",
    sha256 = "925d4dfbf179f00032be3a3a1ec7cf8ed8f9b9b2cd8ea87c2a4da1e97fcfd180",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/lzop-1.04-16.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/925d4dfbf179f00032be3a3a1ec7cf8ed8f9b9b2cd8ea87c2a4da1e97fcfd180",
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
        "https://storage.googleapis.com/builddeps/4cd069f5132c87ad16d02ff648b6389e3e303b41661362252134519993afc45c",
    ],
)

rpm(
    name = "make-1__4.4.1-9.el10.s390x",
    sha256 = "aa138cd7a41f8b054dbecd74462e796b47d58cd9058ad8b56734c0cf242dcd80",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/make-4.4.1-9.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/aa138cd7a41f8b054dbecd74462e796b47d58cd9058ad8b56734c0cf242dcd80",
    ],
)

rpm(
    name = "make-1__4.4.1-9.el10.x86_64",
    sha256 = "7d0b52fe16c826f8b08656abd70509987e69e2b8a9f0c42fda803d41a9e7c74e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/make-4.4.1-9.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d0b52fe16c826f8b08656abd70509987e69e2b8a9f0c42fda803d41a9e7c74e",
    ],
)

rpm(
    name = "mpdecimal-0__2.5.1-12.el10.aarch64",
    sha256 = "f7755f98208b3f400c950ba46acf568f113029893fede5770d19eedadfa0b3ea",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/mpdecimal-2.5.1-12.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f7755f98208b3f400c950ba46acf568f113029893fede5770d19eedadfa0b3ea",
    ],
)

rpm(
    name = "mpdecimal-0__2.5.1-12.el10.s390x",
    sha256 = "2dd0dbab48a3481fab6bcb4554b0854cb66c8d142ef28b8e97b7dfc96d4c2c93",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/mpdecimal-2.5.1-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2dd0dbab48a3481fab6bcb4554b0854cb66c8d142ef28b8e97b7dfc96d4c2c93",
    ],
)

rpm(
    name = "mpdecimal-0__2.5.1-12.el10.x86_64",
    sha256 = "7d1762e4770170efa93ff4f7e07cf523f62d3e3378f50d87d7b307cd8a73ee77",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/mpdecimal-2.5.1-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d1762e4770170efa93ff4f7e07cf523f62d3e3378f50d87d7b307cd8a73ee77",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-10.el9.aarch64",
    sha256 = "bea56ccc46a2a14f3f2c8d9624675abc135e4f002e87c76541784b047d51764d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/mpfr-4.1.0-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bea56ccc46a2a14f3f2c8d9624675abc135e4f002e87c76541784b047d51764d",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-10.el9.s390x",
    sha256 = "b166f1d2ae951d053a5761c826cd5bd8735412e465ce7cbfe78b1292c27aa10e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/mpfr-4.1.0-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b166f1d2ae951d053a5761c826cd5bd8735412e465ce7cbfe78b1292c27aa10e",
    ],
)

rpm(
    name = "mpfr-0__4.1.0-10.el9.x86_64",
    sha256 = "11c1d6b33b7e64ddc40faf45b949618c829bd2e3d3661132417e4c8aee6ab0fd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/mpfr-4.1.0-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11c1d6b33b7e64ddc40faf45b949618c829bd2e3d3661132417e4c8aee6ab0fd",
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
    name = "mpfr-0__4.2.1-8.el10.aarch64",
    sha256 = "df1662c3221a03d86963ca2f2b4db1b745493ce805e95fa07ebde2a8df7b3628",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/mpfr-4.2.1-8.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/df1662c3221a03d86963ca2f2b4db1b745493ce805e95fa07ebde2a8df7b3628",
    ],
)

rpm(
    name = "mpfr-0__4.2.1-8.el10.s390x",
    sha256 = "3b4858c266478b5173b90c650c9d0bbe7a9f84c31bde1fb85ac709051531f98f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/mpfr-4.2.1-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3b4858c266478b5173b90c650c9d0bbe7a9f84c31bde1fb85ac709051531f98f",
    ],
)

rpm(
    name = "mpfr-0__4.2.1-8.el10.x86_64",
    sha256 = "351ac05205dd7c29daecbeeb7e53bab4a130060d1d1d1f610097d8b5fc30289f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/mpfr-4.2.1-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/351ac05205dd7c29daecbeeb7e53bab4a130060d1d1d1f610097d8b5fc30289f",
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
    name = "ncurses-base-0__6.4-15.20240127.el10.aarch64",
    sha256 = "9feb09b8a1f86ca76ed7dd6f66476b05fde5084757501cf24be4b8e836a99693",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.4-15.20240127.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9feb09b8a1f86ca76ed7dd6f66476b05fde5084757501cf24be4b8e836a99693",
    ],
)

rpm(
    name = "ncurses-base-0__6.4-15.20240127.el10.s390x",
    sha256 = "9feb09b8a1f86ca76ed7dd6f66476b05fde5084757501cf24be4b8e836a99693",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/ncurses-base-6.4-15.20240127.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9feb09b8a1f86ca76ed7dd6f66476b05fde5084757501cf24be4b8e836a99693",
    ],
)

rpm(
    name = "ncurses-base-0__6.4-15.20240127.el10.x86_64",
    sha256 = "9feb09b8a1f86ca76ed7dd6f66476b05fde5084757501cf24be4b8e836a99693",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.4-15.20240127.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9feb09b8a1f86ca76ed7dd6f66476b05fde5084757501cf24be4b8e836a99693",
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
    name = "ncurses-libs-0__6.4-15.20240127.el10.aarch64",
    sha256 = "3436974a653ed58ff9bb3fe4cab25f3b2b81e672b5e034cc8c951ccd947f3257",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.4-15.20240127.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3436974a653ed58ff9bb3fe4cab25f3b2b81e672b5e034cc8c951ccd947f3257",
    ],
)

rpm(
    name = "ncurses-libs-0__6.4-15.20240127.el10.s390x",
    sha256 = "1fa5412403feb16c29c7558a9a7ee39e0332ea31b28134dfe6c7110294b78a68",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/ncurses-libs-6.4-15.20240127.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/1fa5412403feb16c29c7558a9a7ee39e0332ea31b28134dfe6c7110294b78a68",
    ],
)

rpm(
    name = "ncurses-libs-0__6.4-15.20240127.el10.x86_64",
    sha256 = "2bd5fc0a9343904137f1d2fdf6188d0f40d72971810dabf8fb1052f7b3915095",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.4-15.20240127.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2bd5fc0a9343904137f1d2fdf6188d0f40d72971810dabf8fb1052f7b3915095",
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
    name = "nftables-1__1.0.9-7.el9.aarch64",
    sha256 = "b91eb3193da58eabccce8146270c9370550702e6590c02aa1371b21d2f198f76",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/nftables-1.0.9-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b91eb3193da58eabccce8146270c9370550702e6590c02aa1371b21d2f198f76",
    ],
)

rpm(
    name = "nftables-1__1.0.9-7.el9.s390x",
    sha256 = "efb7e3971382ce36fa24a08b106cc726175aa71135e387a94c4d8b1d570fbce8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/nftables-1.0.9-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/efb7e3971382ce36fa24a08b106cc726175aa71135e387a94c4d8b1d570fbce8",
    ],
)

rpm(
    name = "nftables-1__1.0.9-7.el9.x86_64",
    sha256 = "f315ae294239ab2486c817938d6ba30ca7e6eebaa66084203322fb5f245e129b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/nftables-1.0.9-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f315ae294239ab2486c817938d6ba30ca7e6eebaa66084203322fb5f245e129b",
    ],
)

rpm(
    name = "nftables-1__1.1.5-5.el10.aarch64",
    sha256 = "5d8752baa436ee262917a84609917c11bb7bd9a359fce65fb03ca462ef5367ed",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/nftables-1.1.5-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5d8752baa436ee262917a84609917c11bb7bd9a359fce65fb03ca462ef5367ed",
    ],
)

rpm(
    name = "nftables-1__1.1.5-5.el10.s390x",
    sha256 = "f97b1cf9330e1a097a34ba934279636f93bd9b653b75a12e6d57039607196fdf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/nftables-1.1.5-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f97b1cf9330e1a097a34ba934279636f93bd9b653b75a12e6d57039607196fdf",
    ],
)

rpm(
    name = "nftables-1__1.1.5-5.el10.x86_64",
    sha256 = "0e95fb106ce2165cbe9c2a1419a8c2f1340ee48aad99ec2bd9ac4b207e8286ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/nftables-1.1.5-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0e95fb106ce2165cbe9c2a1419a8c2f1340ee48aad99ec2bd9ac4b207e8286ba",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-5.el9.aarch64",
    sha256 = "b19deb6d714c11d77f7ec5c3fe517346570099811885f56bb1ae93385886744b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.92-5.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b19deb6d714c11d77f7ec5c3fe517346570099811885f56bb1ae93385886744b",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-5.el9.s390x",
    sha256 = "7389016c95bbc4adb59b9925a35924c21e821c63bfdf34d2bbe02867346bb4b9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/nmap-ncat-7.92-5.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7389016c95bbc4adb59b9925a35924c21e821c63bfdf34d2bbe02867346bb4b9",
    ],
)

rpm(
    name = "nmap-ncat-3__7.92-5.el9.x86_64",
    sha256 = "988deb2cb2041d9c7feec5bf829985dc15af58ef167006fa91e23a80f2103d96",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.92-5.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/988deb2cb2041d9c7feec5bf829985dc15af58ef167006fa91e23a80f2103d96",
    ],
)

rpm(
    name = "nmap-ncat-4__7.92-5.el10.aarch64",
    sha256 = "7975edb3d4e9c583a41707bdd6f4d21dee67e571f7b7338352d75cf09130612e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.92-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7975edb3d4e9c583a41707bdd6f4d21dee67e571f7b7338352d75cf09130612e",
    ],
)

rpm(
    name = "nmap-ncat-4__7.92-5.el10.s390x",
    sha256 = "a0dd5969f49cf59a2448a75498a25007ea270fa25eed6b9d70c1148b88e37a96",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/nmap-ncat-7.92-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a0dd5969f49cf59a2448a75498a25007ea270fa25eed6b9d70c1148b88e37a96",
    ],
)

rpm(
    name = "nmap-ncat-4__7.92-5.el10.x86_64",
    sha256 = "30fcce0936e6fad42a6cca2d6999758fd637ad6d7057bd5c155eb8f49af53157",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.92-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/30fcce0936e6fad42a6cca2d6999758fd637ad6d7057bd5c155eb8f49af53157",
    ],
)

rpm(
    name = "npth-0__1.6-21.el10.s390x",
    sha256 = "47f1f79ad844c4d845591871bc752bf8677fb257fa2cc4d58778fab215965bf1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/npth-1.6-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/47f1f79ad844c4d845591871bc752bf8677fb257fa2cc4d58778fab215965bf1",
    ],
)

rpm(
    name = "npth-0__1.6-21.el10.x86_64",
    sha256 = "9d5de697dd346d3eeac85008ab93fbfce90ea49342418402959eda90829578d0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/npth-1.6-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d5de697dd346d3eeac85008ab93fbfce90ea49342418402959eda90829578d0",
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
        "https://storage.googleapis.com/builddeps/81016ab56c83cb8c221216794571ae58bb914e21dd3794c242f7ce8a8d8fbf8f",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el10.s390x",
    sha256 = "184bd0085cc03d74e317a2aad472fa7638fffc35e4e1a314700e31b00398a6cc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/numactl-libs-2.0.19-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/184bd0085cc03d74e317a2aad472fa7638fffc35e4e1a314700e31b00398a6cc",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.19-3.el10.x86_64",
    sha256 = "263ee2cba1d57996778f70045fbc4657067f73edafd6c6b04f4599c3eb12fbfd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.19-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/263ee2cba1d57996778f70045fbc4657067f73edafd6c6b04f4599c3eb12fbfd",
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
        "https://storage.googleapis.com/builddeps/942f7db59b047cc56e6c53c5bb9a2a84ba4715088021e85526e973d9485bc8fa",
    ],
)

rpm(
    name = "numad-0__0.5-50.20251104git.el10.x86_64",
    sha256 = "91def9a46ee7b6ee35a11276987c7eace5b18e97d6d967fe09139e0a01cb0731",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/numad-0.5-50.20251104git.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/91def9a46ee7b6ee35a11276987c7eace5b18e97d6d967fe09139e0a01cb0731",
    ],
)

rpm(
    name = "numad-0__0.5-50.20251104git.el9.aarch64",
    sha256 = "4aad2f9ae73cee80a2cb7d4443f08ec77de9e24f6ccf9b5b0ba0e1d8b2833244",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/numad-0.5-50.20251104git.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4aad2f9ae73cee80a2cb7d4443f08ec77de9e24f6ccf9b5b0ba0e1d8b2833244",
    ],
)

rpm(
    name = "numad-0__0.5-50.20251104git.el9.x86_64",
    sha256 = "e35e13a43887728741cb976b02ab2a99455964f374c44a23421a7970c814f19e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/numad-0.5-50.20251104git.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e35e13a43887728741cb976b02ab2a99455964f374c44a23421a7970c814f19e",
    ],
)

rpm(
    name = "openldap-0__2.6.13-1.el10.s390x",
    sha256 = "51c1bcd25a49f81085101b74e089e9ac065f763a147d07c62b4f0978413c9dcc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/openldap-2.6.13-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/51c1bcd25a49f81085101b74e089e9ac065f763a147d07c62b4f0978413c9dcc",
    ],
)

rpm(
    name = "openldap-0__2.6.13-1.el10.x86_64",
    sha256 = "668bf98af411e595330006c5eb8834cfe0fd8b11e90525d0f72709b0c200e224",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/openldap-2.6.13-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/668bf98af411e595330006c5eb8834cfe0fd8b11e90525d0f72709b0c200e224",
    ],
)

rpm(
    name = "openldap-0__2.6.13-1.el9.s390x",
    sha256 = "b8a4974ea6b1e8b307bf73054a22b6f4d3c34724ca6fa960d0b97978ab52290f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openldap-2.6.13-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b8a4974ea6b1e8b307bf73054a22b6f4d3c34724ca6fa960d0b97978ab52290f",
    ],
)

rpm(
    name = "openldap-0__2.6.13-1.el9.x86_64",
    sha256 = "6bed5684275d340e78f9300c4da665a6a0ea6779f7cee7217ddee868af81d8eb",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openldap-2.6.13-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6bed5684275d340e78f9300c4da665a6a0ea6779f7cee7217ddee868af81d8eb",
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
    name = "openssl-1__3.5.7-2.el9.aarch64",
    sha256 = "56ff7e328cb66a757a59c11e4176029d93c0c3586928268a7c6355cc6814e75f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-3.5.7-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56ff7e328cb66a757a59c11e4176029d93c0c3586928268a7c6355cc6814e75f",
    ],
)

rpm(
    name = "openssl-1__3.5.7-2.el9.s390x",
    sha256 = "d53822ad7049bb5e9bb4730cb99d92760f58e1d7936fb8ca62aaf1e4aba927a3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-3.5.7-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d53822ad7049bb5e9bb4730cb99d92760f58e1d7936fb8ca62aaf1e4aba927a3",
    ],
)

rpm(
    name = "openssl-1__3.5.7-2.el9.x86_64",
    sha256 = "f300e9e401691c215d3a09d90c6d71bf11e4028824c1f996139da7afb89f0c22",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-3.5.7-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f300e9e401691c215d3a09d90c6d71bf11e4028824c1f996139da7afb89f0c22",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-3.el10.aarch64",
    sha256 = "894577918ac27d68f522b6b28d1c670b03732f9e60d0e75cfd2cae0e5811a9f1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/openssl-fips-provider-3.5.5-3.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/894577918ac27d68f522b6b28d1c670b03732f9e60d0e75cfd2cae0e5811a9f1",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-3.el10.s390x",
    sha256 = "64eae78eca4cb52c0cf0bab883bd2b9a874161fe4fd066540a5dd2121175e39b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/openssl-fips-provider-3.5.5-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/64eae78eca4cb52c0cf0bab883bd2b9a874161fe4fd066540a5dd2121175e39b",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.5-3.el10.x86_64",
    sha256 = "6795928e79183c7ea391fa0ec89d95e9d224362a2db3ee2a11572fb4d579e2ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/openssl-fips-provider-3.5.5-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6795928e79183c7ea391fa0ec89d95e9d224362a2db3ee2a11572fb4d579e2ba",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.7-2.el9.aarch64",
    sha256 = "53025a629656559c6f5a8c97425e99736e271c4bb26857a1f3be6fbf413c50e1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-fips-provider-3.5.7-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/53025a629656559c6f5a8c97425e99736e271c4bb26857a1f3be6fbf413c50e1",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.7-2.el9.s390x",
    sha256 = "043ab9cd2d088bbf81cc4ebd5e77eb6ea9831190e6ba1de273e9b91ee89af8d3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-fips-provider-3.5.7-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/043ab9cd2d088bbf81cc4ebd5e77eb6ea9831190e6ba1de273e9b91ee89af8d3",
    ],
)

rpm(
    name = "openssl-fips-provider-1__3.5.7-2.el9.x86_64",
    sha256 = "62a7e1658907277f217d3f11681fe86878fe68e43d4a2fe7e7f08bd174d60dc9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-fips-provider-3.5.7-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/62a7e1658907277f217d3f11681fe86878fe68e43d4a2fe7e7f08bd174d60dc9",
    ],
)

rpm(
    name = "openssl-libs-1__3.2.2-6.el9.x86_64",
    sha256 = "4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738",
    urls = ["https://storage.googleapis.com/builddeps/4a0a29a309f72ba65a2d0b2d4b51637253520f6a0a1bd4640f0a09f7d7555738"],
)

rpm(
    name = "openssl-libs-1__3.5.5-3.el10.aarch64",
    sha256 = "cf82c121c174ba6b5f14d0c6cf31d6836bc024a0edec2c55775ffdc8289eab2e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.5.5-3.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cf82c121c174ba6b5f14d0c6cf31d6836bc024a0edec2c55775ffdc8289eab2e",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.5-3.el10.s390x",
    sha256 = "eb5daab99b3ef441de5b65f4f0d70a0a14f9195e0c9ce823534cd39716402b25",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/openssl-libs-3.5.5-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/eb5daab99b3ef441de5b65f4f0d70a0a14f9195e0c9ce823534cd39716402b25",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.5-3.el10.x86_64",
    sha256 = "e1a42c8f252acffd4617b6893e7533d16c1bca653746ea6aeb87ab0791359741",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.5.5-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e1a42c8f252acffd4617b6893e7533d16c1bca653746ea6aeb87ab0791359741",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.7-2.el9.aarch64",
    sha256 = "ab9bf090dc882977ecfd66911d7a0a1f6c9275e6c01738bc197a778646630336",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/openssl-libs-3.5.7-2.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab9bf090dc882977ecfd66911d7a0a1f6c9275e6c01738bc197a778646630336",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.7-2.el9.s390x",
    sha256 = "e4d4f983ac38e1b5e4c8ccdadd6375e9bef0372d08c5094acf99d37bb4a91578",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/openssl-libs-3.5.7-2.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e4d4f983ac38e1b5e4c8ccdadd6375e9bef0372d08c5094acf99d37bb4a91578",
    ],
)

rpm(
    name = "openssl-libs-1__3.5.7-2.el9.x86_64",
    sha256 = "e6309deacba826ea59e8c706da0f130015143a4acf0d0dab2411426b518b8363",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/openssl-libs-3.5.7-2.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e6309deacba826ea59e8c706da0f130015143a4acf0d0dab2411426b518b8363",
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
    name = "osinfo-db-0__20250606-2.el10.s390x",
    sha256 = "6af76d4696257961279e6c6443eb057c16355cdb1a97cd59559b22822f5ac289",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/osinfo-db-20250606-2.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6af76d4696257961279e6c6443eb057c16355cdb1a97cd59559b22822f5ac289",
    ],
)

rpm(
    name = "osinfo-db-0__20250606-2.el10.x86_64",
    sha256 = "6af76d4696257961279e6c6443eb057c16355cdb1a97cd59559b22822f5ac289",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/osinfo-db-20250606-2.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6af76d4696257961279e6c6443eb057c16355cdb1a97cd59559b22822f5ac289",
    ],
)

rpm(
    name = "osinfo-db-0__20250606-2.el9.s390x",
    sha256 = "187cd0d251cf29bf4837eb2bd2c363195bb11aa25b86a1e3f934b43a0dd6e0c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/osinfo-db-20250606-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/187cd0d251cf29bf4837eb2bd2c363195bb11aa25b86a1e3f934b43a0dd6e0c6",
    ],
)

rpm(
    name = "osinfo-db-0__20250606-2.el9.x86_64",
    sha256 = "187cd0d251cf29bf4837eb2bd2c363195bb11aa25b86a1e3f934b43a0dd6e0c6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/osinfo-db-20250606-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/187cd0d251cf29bf4837eb2bd2c363195bb11aa25b86a1e3f934b43a0dd6e0c6",
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
        "https://storage.googleapis.com/builddeps/fed0a8870fb28338db4b8b2bb6f57d44fcbfcaafe88187d787e3bf6cd5f911f0",
    ],
)

rpm(
    name = "osinfo-db-tools-0__1.11.0-8.el10.x86_64",
    sha256 = "31d38586cdd723e3de145e9b03dd1f4eaa2e63681323a7d0bcce3a47cc2e1d62",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/osinfo-db-tools-1.11.0-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31d38586cdd723e3de145e9b03dd1f4eaa2e63681323a7d0bcce3a47cc2e1d62",
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
    name = "p11-kit-0__0.26.2-1.el10.aarch64",
    sha256 = "93518eadefcbe6fb07cfa3d5d4cddd0820915ab44b9f4ce753af503558653d71",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/p11-kit-0.26.2-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/93518eadefcbe6fb07cfa3d5d4cddd0820915ab44b9f4ce753af503558653d71",
    ],
)

rpm(
    name = "p11-kit-0__0.26.2-1.el10.s390x",
    sha256 = "2252c8486d84092c6dca6488d39d52c31c348441d311474943a37c3192cc62fb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/p11-kit-0.26.2-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2252c8486d84092c6dca6488d39d52c31c348441d311474943a37c3192cc62fb",
    ],
)

rpm(
    name = "p11-kit-0__0.26.2-1.el10.x86_64",
    sha256 = "5787928d07bc6241e895500dee79ad8aa85de7ff997402c9c08e11dc9732240a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/p11-kit-0.26.2-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5787928d07bc6241e895500dee79ad8aa85de7ff997402c9c08e11dc9732240a",
    ],
)

rpm(
    name = "p11-kit-0__0.26.4-1.el9.aarch64",
    sha256 = "9769d2d2bfa7db493218bdba14075df2428543817e125c45f8775482236ce2aa",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-0.26.4-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9769d2d2bfa7db493218bdba14075df2428543817e125c45f8775482236ce2aa",
    ],
)

rpm(
    name = "p11-kit-0__0.26.4-1.el9.s390x",
    sha256 = "b9a14789372202a1b4d0d30c57756d8bce2a22342c90f63479343415b866495e",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-0.26.4-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b9a14789372202a1b4d0d30c57756d8bce2a22342c90f63479343415b866495e",
    ],
)

rpm(
    name = "p11-kit-0__0.26.4-1.el9.x86_64",
    sha256 = "185c0ee8f5470fe94b492f11c1e0c1674be3051062e906cdd32473a5ff8db045",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-0.26.4-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/185c0ee8f5470fe94b492f11c1e0c1674be3051062e906cdd32473a5ff8db045",
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
    name = "p11-kit-trust-0__0.26.2-1.el10.aarch64",
    sha256 = "87acee4c95360a0858299917ce916696d037cfea50e5d2326c7c0e002ac04470",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.26.2-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/87acee4c95360a0858299917ce916696d037cfea50e5d2326c7c0e002ac04470",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.2-1.el10.s390x",
    sha256 = "04dfaf06b01ee63eac7a2cf52bff14b94379a34cca6ec548a9cd7ecefc124c71",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/p11-kit-trust-0.26.2-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/04dfaf06b01ee63eac7a2cf52bff14b94379a34cca6ec548a9cd7ecefc124c71",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.2-1.el10.x86_64",
    sha256 = "21f8ff9b21243fbe951260c4dfd47058074c818be68bb274db3eb20378433f2a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.26.2-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21f8ff9b21243fbe951260c4dfd47058074c818be68bb274db3eb20378433f2a",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.4-1.el9.aarch64",
    sha256 = "89535f163fd9f6d78be992d9145e7e628fd1d7308bc8f9d6fb4744b5e14a7628",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.26.4-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/89535f163fd9f6d78be992d9145e7e628fd1d7308bc8f9d6fb4744b5e14a7628",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.4-1.el9.s390x",
    sha256 = "66c57115a29f40b23a6cf8b522033e5731850ea4e7df2769871db794816588f5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/p11-kit-trust-0.26.4-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/66c57115a29f40b23a6cf8b522033e5731850ea4e7df2769871db794816588f5",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.26.4-1.el9.x86_64",
    sha256 = "8dc277e3855df15bf2db2e8acd03e08311cd565152fa958ac75cbb862f98ebe1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.26.4-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8dc277e3855df15bf2db2e8acd03e08311cd565152fa958ac75cbb862f98ebe1",
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
    name = "pam-0__1.5.1-29.el9.aarch64",
    sha256 = "090c497dc32e6bc3a95c0200f1aa1dfcd696f25ba5b082f0ff7ec249b25a8923",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/pam-1.5.1-29.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/090c497dc32e6bc3a95c0200f1aa1dfcd696f25ba5b082f0ff7ec249b25a8923",
    ],
)

rpm(
    name = "pam-0__1.5.1-29.el9.s390x",
    sha256 = "692016ce57b3dd1a8a79640fc86c8ef6b2968e94ae59055532cf358b6704e652",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/pam-1.5.1-29.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/692016ce57b3dd1a8a79640fc86c8ef6b2968e94ae59055532cf358b6704e652",
    ],
)

rpm(
    name = "pam-0__1.5.1-29.el9.x86_64",
    sha256 = "fb6521a7339de9b9be954d07aef4787867b85b45fdd78f65703bbd8819f6d585",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/pam-1.5.1-29.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb6521a7339de9b9be954d07aef4787867b85b45fdd78f65703bbd8819f6d585",
    ],
)

rpm(
    name = "pam-0__1.6.1-10.el10.aarch64",
    sha256 = "5ea2b4778a5f93bcff868bb55fe4a02c199bcece84e656d196d020f95002556f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pam-1.6.1-10.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5ea2b4778a5f93bcff868bb55fe4a02c199bcece84e656d196d020f95002556f",
    ],
)

rpm(
    name = "pam-0__1.6.1-10.el10.s390x",
    sha256 = "93b14245d37206fa348179228a269844d5404ab5e781574499825b49ba3d0323",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pam-1.6.1-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/93b14245d37206fa348179228a269844d5404ab5e781574499825b49ba3d0323",
    ],
)

rpm(
    name = "pam-0__1.6.1-10.el10.x86_64",
    sha256 = "7ded51d01f4149806646a8e37862dc653200a529242251a7cac73f15e3361a52",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pam-1.6.1-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7ded51d01f4149806646a8e37862dc653200a529242251a7cac73f15e3361a52",
    ],
)

rpm(
    name = "pam-libs-0__1.6.1-10.el10.aarch64",
    sha256 = "0442795f4217fba235c6e795e7c7c88e46205cd8d72bebddb26da20a55104a08",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/pam-libs-1.6.1-10.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0442795f4217fba235c6e795e7c7c88e46205cd8d72bebddb26da20a55104a08",
    ],
)

rpm(
    name = "pam-libs-0__1.6.1-10.el10.s390x",
    sha256 = "52f36a9350a8c0b1fcadb9c96a1ecc5ba8b1665cb714bb28d0e20bda4c593dde",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pam-libs-1.6.1-10.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/52f36a9350a8c0b1fcadb9c96a1ecc5ba8b1665cb714bb28d0e20bda4c593dde",
    ],
)

rpm(
    name = "pam-libs-0__1.6.1-10.el10.x86_64",
    sha256 = "e0f3cd3d4d036e84f8e1dea5c4bcc41ad1064a780c4e6bb0038d37713f2071c4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pam-libs-1.6.1-10.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e0f3cd3d4d036e84f8e1dea5c4bcc41ad1064a780c4e6bb0038d37713f2071c4",
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
        "https://storage.googleapis.com/builddeps/bba170cbf71b85ebc299c466f4a15c14ce80fae6a138ec87014b772bab377f02",
    ],
)

rpm(
    name = "parted-0__3.6-7.el10.x86_64",
    sha256 = "bb339dd10bc7951376243b5a9ae11e18f3ecd235db576739da006a147c6bf412",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/parted-3.6-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb339dd10bc7951376243b5a9ae11e18f3ecd235db576739da006a147c6bf412",
    ],
)

rpm(
    name = "passt-0__0__caret__20260611.ga9c61ff-1.el10.aarch64",
    sha256 = "553173929764fc739b4927eb59c78146759866a23056e495769e7fdfa8e40359",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/passt-0%5E20260611.ga9c61ff-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/553173929764fc739b4927eb59c78146759866a23056e495769e7fdfa8e40359",
    ],
)

rpm(
    name = "passt-0__0__caret__20260611.ga9c61ff-1.el10.s390x",
    sha256 = "af219ff394db184ed6f0e269fc384b07cfc25f4da97353e795fadaa32d2bc5d2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/passt-0%5E20260611.ga9c61ff-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/af219ff394db184ed6f0e269fc384b07cfc25f4da97353e795fadaa32d2bc5d2",
    ],
)

rpm(
    name = "passt-0__0__caret__20260611.ga9c61ff-1.el10.x86_64",
    sha256 = "2e2aa85f27786d7b7cd3e307804939a1dec03c670b09764ba8d2b27a55bf2fd5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/passt-0%5E20260611.ga9c61ff-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2e2aa85f27786d7b7cd3e307804939a1dec03c670b09764ba8d2b27a55bf2fd5",
    ],
)

rpm(
    name = "passt-0__0__caret__20260611.ga9c61ff-1.el9.aarch64",
    sha256 = "347afb5eb0087bfa21b66a0ab3b40d70c60927aa811602166f0738ff33864fdd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/passt-0%5E20260611.ga9c61ff-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/347afb5eb0087bfa21b66a0ab3b40d70c60927aa811602166f0738ff33864fdd",
    ],
)

rpm(
    name = "passt-0__0__caret__20260611.ga9c61ff-1.el9.s390x",
    sha256 = "29f4e733a0533287b7e4b64e8f5e7aa8206b456e501a6d3ad6b575fe30d98b16",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/passt-0%5E20260611.ga9c61ff-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/29f4e733a0533287b7e4b64e8f5e7aa8206b456e501a6d3ad6b575fe30d98b16",
    ],
)

rpm(
    name = "passt-0__0__caret__20260611.ga9c61ff-1.el9.x86_64",
    sha256 = "5f071cad84c47839767aa3750ac08a166fa4bebf708e86ec5b114647aa55270c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/passt-0%5E20260611.ga9c61ff-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5f071cad84c47839767aa3750ac08a166fa4bebf708e86ec5b114647aa55270c",
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
        "https://storage.googleapis.com/builddeps/23f2a34aa9bc9c8c6662e93d184e07d7e01d45d0fb1b554fd3ed92c03ba2ae3c",
    ],
)

rpm(
    name = "pcre2-0__10.44-1.el10.3.s390x",
    sha256 = "7577ec5ef81b0aa96e340c6d292c4b828e957508962ec1ed68c1f048dff3998e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pcre2-10.44-1.el10.3.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7577ec5ef81b0aa96e340c6d292c4b828e957508962ec1ed68c1f048dff3998e",
    ],
)

rpm(
    name = "pcre2-0__10.44-1.el10.3.x86_64",
    sha256 = "773781e3aa9994fa8d6105ddc0b3d00fdd735bd589a5d9fe40fe96be6a7d89a7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pcre2-10.44-1.el10.3.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/773781e3aa9994fa8d6105ddc0b3d00fdd735bd589a5d9fe40fe96be6a7d89a7",
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
        "https://storage.googleapis.com/builddeps/71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.44-1.el10.3.s390x",
    sha256 = "71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pcre2-syntax-10.44-1.el10.3.noarch.rpm",
        "https://storage.googleapis.com/builddeps/71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.44-1.el10.3.x86_64",
    sha256 = "71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pcre2-syntax-10.44-1.el10.3.noarch.rpm",
        "https://storage.googleapis.com/builddeps/71de87112a846df439b0b3381b35fbba8c6e72109c6a4795c1de96e48bbc5d40",
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
        "https://storage.googleapis.com/builddeps/dc2c0f98c210e8209690b1d2a4fffa348b6ad22062461f4b3ebc7d7f6dd0246e",
    ],
)

rpm(
    name = "pixman-0__0.43.4-2.el10.s390x",
    sha256 = "269fcda361ff485d379f8a773e47752758b8c58e288f78196f169149570af637",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/pixman-0.43.4-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/269fcda361ff485d379f8a773e47752758b8c58e288f78196f169149570af637",
    ],
)

rpm(
    name = "pixman-0__0.43.4-2.el10.x86_64",
    sha256 = "c91d0077a917e843a009c69b63793de4b8d2f9a81414f63e319ac31bbf6a08cb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/pixman-0.43.4-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c91d0077a917e843a009c69b63793de4b8d2f9a81414f63e319ac31bbf6a08cb",
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
        "https://storage.googleapis.com/builddeps/5bd76130128a85e6275c6e56f7e519532425cd2a5d2db7a795a4d1d15f7d0d57",
    ],
)

rpm(
    name = "pkgconf-0__2.1.0-3.el10.s390x",
    sha256 = "010973bdd551e8489eb97446701fdf3100b8dd0b1ea7efd650412d8869d8181a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pkgconf-2.1.0-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/010973bdd551e8489eb97446701fdf3100b8dd0b1ea7efd650412d8869d8181a",
    ],
)

rpm(
    name = "pkgconf-0__2.1.0-3.el10.x86_64",
    sha256 = "ced8f494b664667d52245ff94ce6c0b2cad135586a36a9ff7f81281d1533f178",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pkgconf-2.1.0-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ced8f494b664667d52245ff94ce6c0b2cad135586a36a9ff7f81281d1533f178",
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
        "https://storage.googleapis.com/builddeps/4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    ],
)

rpm(
    name = "pkgconf-m4-0__2.1.0-3.el10.s390x",
    sha256 = "4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pkgconf-m4-2.1.0-3.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    ],
)

rpm(
    name = "pkgconf-m4-0__2.1.0-3.el10.x86_64",
    sha256 = "4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pkgconf-m4-2.1.0-3.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4de2147846658c2849aa28f756e5e906a3012be53e656b4a39ae77076286e828",
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
        "https://storage.googleapis.com/builddeps/3d646b74ccc730b097ceba50c5a054a6017b61e354b0e8731c66b5c266d55e40",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__2.1.0-3.el10.s390x",
    sha256 = "f895f22efbaa5a4f978600b3df480eabab7bb2eea7f0f8e90897ab4d76ae2102",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/pkgconf-pkg-config-2.1.0-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f895f22efbaa5a4f978600b3df480eabab7bb2eea7f0f8e90897ab4d76ae2102",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__2.1.0-3.el10.x86_64",
    sha256 = "4f5231ffccc59b5f1c42d85cc0dafea9b6901107660931071cf1e65f99af1e0b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/pkgconf-pkg-config-2.1.0-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4f5231ffccc59b5f1c42d85cc0dafea9b6901107660931071cf1e65f99af1e0b",
    ],
)

rpm(
    name = "policycoreutils-0__3.10-3.el10.aarch64",
    sha256 = "49dfce0d5fa6c20245b16f9765dc0cef8dba3c00cf02bfe5ad23a8eade19532c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.10-3.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/49dfce0d5fa6c20245b16f9765dc0cef8dba3c00cf02bfe5ad23a8eade19532c",
    ],
)

rpm(
    name = "policycoreutils-0__3.10-3.el10.s390x",
    sha256 = "0d96befdc527f3a07f27da06361e927c514325cdc289920019988f376c26a227",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/policycoreutils-3.10-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0d96befdc527f3a07f27da06361e927c514325cdc289920019988f376c26a227",
    ],
)

rpm(
    name = "policycoreutils-0__3.10-3.el10.x86_64",
    sha256 = "9147997eacc0b4aa97fb3de3212546dac39c491e2c6cd1adc325ab9aaca4a46c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.10-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9147997eacc0b4aa97fb3de3212546dac39c491e2c6cd1adc325ab9aaca4a46c",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-2.1.el9.x86_64",
    sha256 = "a87874363af6432b1c96b40f8b79b90616df22bff3bd4f9aa39da24f5bddd3e9",
    urls = ["https://storage.googleapis.com/builddeps/a87874363af6432b1c96b40f8b79b90616df22bff3bd4f9aa39da24f5bddd3e9"],
)

rpm(
    name = "policycoreutils-0__3.6-7.el9.aarch64",
    sha256 = "87243058a01b79b132e9b280f3cb85032d61cb111ce9926c93e4f458e4519095",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/policycoreutils-3.6-7.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/87243058a01b79b132e9b280f3cb85032d61cb111ce9926c93e4f458e4519095",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-7.el9.s390x",
    sha256 = "e35dce1062a3cc50fb55e6e1e3a2aeff687fa33ddb0a243a85c7f741a2879f3b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/policycoreutils-3.6-7.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e35dce1062a3cc50fb55e6e1e3a2aeff687fa33ddb0a243a85c7f741a2879f3b",
    ],
)

rpm(
    name = "policycoreutils-0__3.6-7.el9.x86_64",
    sha256 = "765137818d4a72c824bbf2dcf2323bf8b4740d89910a18f9e09d2605ee276e13",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/policycoreutils-3.6-7.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/765137818d4a72c824bbf2dcf2323bf8b4740d89910a18f9e09d2605ee276e13",
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
    name = "polkit-0__0.117-16.el9.aarch64",
    sha256 = "e20ae48a247d67d7652f1be006e091f3b7d700b96a0c0a1327ed5262bba4308f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-0.117-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e20ae48a247d67d7652f1be006e091f3b7d700b96a0c0a1327ed5262bba4308f",
    ],
)

rpm(
    name = "polkit-0__0.117-16.el9.s390x",
    sha256 = "d56371add066e8459a01f1c9689fc77e1d9b64c8134a7f3f20385b381006b1cc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/polkit-0.117-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d56371add066e8459a01f1c9689fc77e1d9b64c8134a7f3f20385b381006b1cc",
    ],
)

rpm(
    name = "polkit-0__0.117-16.el9.x86_64",
    sha256 = "c3a1e822081a1b68b558de9d3484577c07a69538e6fb845b2be323fc53c9bb09",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-0.117-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c3a1e822081a1b68b558de9d3484577c07a69538e6fb845b2be323fc53c9bb09",
    ],
)

rpm(
    name = "polkit-0__125-6.el10.aarch64",
    sha256 = "7141776ea2f5d731b90812d7a34e29ee9f10bf8eb7d77874e37761124ae203bc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/polkit-125-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7141776ea2f5d731b90812d7a34e29ee9f10bf8eb7d77874e37761124ae203bc",
    ],
)

rpm(
    name = "polkit-0__125-6.el10.s390x",
    sha256 = "69ce1973f3be11c127f43f1108ec698384b6bf26572135ba5bf8ada7b42882f2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/polkit-125-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/69ce1973f3be11c127f43f1108ec698384b6bf26572135ba5bf8ada7b42882f2",
    ],
)

rpm(
    name = "polkit-0__125-6.el10.x86_64",
    sha256 = "ea24cedd7e5e520fb396c649100355704ad8c5cab205735cd535e4e3e51e643f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/polkit-125-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ea24cedd7e5e520fb396c649100355704ad8c5cab205735cd535e4e3e51e643f",
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
    name = "polkit-libs-0__0.117-16.el9.aarch64",
    sha256 = "f4e7bdf11050e796b52adaf515d1a73b75bb53abfc7352f762ae35849a7b9312",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.117-16.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f4e7bdf11050e796b52adaf515d1a73b75bb53abfc7352f762ae35849a7b9312",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-16.el9.s390x",
    sha256 = "4e7d9fb88baf3956c939d42bdac698b0b8f65e713ce2f9bdf5ccf81813f4c27b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/polkit-libs-0.117-16.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4e7d9fb88baf3956c939d42bdac698b0b8f65e713ce2f9bdf5ccf81813f4c27b",
    ],
)

rpm(
    name = "polkit-libs-0__0.117-16.el9.x86_64",
    sha256 = "4b8616d20a53c5e7d48ee04020359e6090e32064387c441dde3fe8b486aeee09",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.117-16.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4b8616d20a53c5e7d48ee04020359e6090e32064387c441dde3fe8b486aeee09",
    ],
)

rpm(
    name = "polkit-libs-0__125-6.el10.aarch64",
    sha256 = "cc7d5267b1d6b3c77267de9cba7d6b4fea2afa3472042bf926215280c5d5f36c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/polkit-libs-125-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cc7d5267b1d6b3c77267de9cba7d6b4fea2afa3472042bf926215280c5d5f36c",
    ],
)

rpm(
    name = "polkit-libs-0__125-6.el10.s390x",
    sha256 = "d9be964cef3d7ce8061bd95088d4a20562e9f944ed7c9e6dbd9dc9d86fc5eb3e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/polkit-libs-125-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d9be964cef3d7ce8061bd95088d4a20562e9f944ed7c9e6dbd9dc9d86fc5eb3e",
    ],
)

rpm(
    name = "polkit-libs-0__125-6.el10.x86_64",
    sha256 = "c91efd1338f3d60d68ae20c24ccc29c7edbe7b9f823737a568f5ac779b27175a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/polkit-libs-125-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c91efd1338f3d60d68ae20c24ccc29c7edbe7b9f823737a568f5ac779b27175a",
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
        "https://storage.googleapis.com/builddeps/4c727d11de14d8bf1bc0df2be55c75cb0200a685c2737c740e636dedd3edbb0c",
    ],
)

rpm(
    name = "popt-0__1.19-8.el10.s390x",
    sha256 = "3dca46c310266fc9cce48d39651984d726cf727e12f446b70db78cd6f96e3515",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/popt-1.19-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3dca46c310266fc9cce48d39651984d726cf727e12f446b70db78cd6f96e3515",
    ],
)

rpm(
    name = "popt-0__1.19-8.el10.x86_64",
    sha256 = "bd15d2816600655a5241bc3efe6e1ac386061ba6ff2d05e53c70683db8761e5f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/popt-1.19-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bd15d2816600655a5241bc3efe6e1ac386061ba6ff2d05e53c70683db8761e5f",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-15.el9.aarch64",
    sha256 = "f761273cb213c1c5644298dbb0c00a6e85b3b47b0a03d07357dfc88d7e5404fc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.17-15.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f761273cb213c1c5644298dbb0c00a6e85b3b47b0a03d07357dfc88d7e5404fc",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-15.el9.s390x",
    sha256 = "4e5cf7e16181b83dbe0c4cba44def345038f8224bc7cb8535350c440805e1311",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/procps-ng-3.3.17-15.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4e5cf7e16181b83dbe0c4cba44def345038f8224bc7cb8535350c440805e1311",
    ],
)

rpm(
    name = "procps-ng-0__3.3.17-15.el9.x86_64",
    sha256 = "92732e06c5bd35dd5eb4d8e0c568a6563db0cd2dbef65031f0bb2e4341de2fc3",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.17-15.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/92732e06c5bd35dd5eb4d8e0c568a6563db0cd2dbef65031f0bb2e4341de2fc3",
    ],
)

rpm(
    name = "procps-ng-0__4.0.4-13.el10.aarch64",
    sha256 = "b74d17d9a9714bd136fac18fe7d29f5edd8cf53d357cec63959384d9f35692be",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/procps-ng-4.0.4-13.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b74d17d9a9714bd136fac18fe7d29f5edd8cf53d357cec63959384d9f35692be",
    ],
)

rpm(
    name = "procps-ng-0__4.0.4-13.el10.s390x",
    sha256 = "8132a2fb83a92c4e7e154c954c147972bd51505a545897f39aac120bf2cfd160",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/procps-ng-4.0.4-13.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8132a2fb83a92c4e7e154c954c147972bd51505a545897f39aac120bf2cfd160",
    ],
)

rpm(
    name = "procps-ng-0__4.0.4-13.el10.x86_64",
    sha256 = "bd91d403871fc30eb969117d61f2be75255bdd218fe33567c8aa0c9d505ddff1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/procps-ng-4.0.4-13.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bd91d403871fc30eb969117d61f2be75255bdd218fe33567c8aa0c9d505ddff1",
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
        "https://storage.googleapis.com/builddeps/3e17d0b103c0444852bbb952e39a64bb18b36f797cf5d4fd1bea57d9bc2c4cbe",
    ],
)

rpm(
    name = "protobuf-c-0__1.5.0-6.el10.s390x",
    sha256 = "70c17f3805e9ecb6eaef0c13ae83b889405c22bdf09eafc74d3f0ba26e0882c0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/protobuf-c-1.5.0-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/70c17f3805e9ecb6eaef0c13ae83b889405c22bdf09eafc74d3f0ba26e0882c0",
    ],
)

rpm(
    name = "protobuf-c-0__1.5.0-6.el10.x86_64",
    sha256 = "a7e792e5ed4f89d5f48eb60453619a6e0ad5cd34469526952c1748f1a99ce3ba",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/protobuf-c-1.5.0-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a7e792e5ed4f89d5f48eb60453619a6e0ad5cd34469526952c1748f1a99ce3ba",
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
        "https://storage.googleapis.com/builddeps/cca0153b72dfb9c9e2f9f8386514ff9591b6166e0746ff32d5cda0eeb9adbaba",
    ],
)

rpm(
    name = "psmisc-0__23.6-8.el10.s390x",
    sha256 = "3c0f3724b4040c7a6c07f5405850ed172f906daa9a7904cd78c8e52c680d4611",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/psmisc-23.6-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3c0f3724b4040c7a6c07f5405850ed172f906daa9a7904cd78c8e52c680d4611",
    ],
)

rpm(
    name = "psmisc-0__23.6-8.el10.x86_64",
    sha256 = "9fea410c82d95565a4cbb178da9557ec9cef3512d573efd4dd940c9f2c4219cf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/psmisc-23.6-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9fea410c82d95565a4cbb178da9557ec9cef3512d573efd4dd940c9f2c4219cf",
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
        "https://storage.googleapis.com/builddeps/440cb6e03187dfd68f62abf1dd751ace84ec8e2179d7de45dde348cf2e7dba11",
    ],
)

rpm(
    name = "publicsuffix-list-dafsa-0__20240107-5.el10.x86_64",
    sha256 = "440cb6e03187dfd68f62abf1dd751ace84ec8e2179d7de45dde348cf2e7dba11",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/publicsuffix-list-dafsa-20240107-5.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/440cb6e03187dfd68f62abf1dd751ace84ec8e2179d7de45dde348cf2e7dba11",
    ],
)

rpm(
    name = "python3-0__3.12.13-2.el10.aarch64",
    sha256 = "bbb1b5a0d1d0529a41bf0769e4e47b579670b4578e8475316b349b27b43b66d9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-3.12.13-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bbb1b5a0d1d0529a41bf0769e4e47b579670b4578e8475316b349b27b43b66d9",
    ],
)

rpm(
    name = "python3-0__3.12.13-2.el10.s390x",
    sha256 = "222774463326a3ff4ccda57c85f87e923c0550c1705db6a7389a546535f3dac1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-3.12.13-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/222774463326a3ff4ccda57c85f87e923c0550c1705db6a7389a546535f3dac1",
    ],
)

rpm(
    name = "python3-0__3.12.13-2.el10.x86_64",
    sha256 = "86b5c4d13bd2d1c85e42721ccf73e00d9a31fa9265e43999e3fa15e5756c0491",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-3.12.13-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86b5c4d13bd2d1c85e42721ccf73e00d9a31fa9265e43999e3fa15e5756c0491",
    ],
)

rpm(
    name = "python3-0__3.9.25-8.el9.aarch64",
    sha256 = "f3926cdabbd297fbf781d45c237b82707702a25083e39af6a1e38546a864c752",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-3.9.25-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f3926cdabbd297fbf781d45c237b82707702a25083e39af6a1e38546a864c752",
    ],
)

rpm(
    name = "python3-0__3.9.25-8.el9.s390x",
    sha256 = "3a500cae099197dbb3819567f6f9716dfa7ecb574d2c48ddc307a2dcb23aa8f2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-3.9.25-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/3a500cae099197dbb3819567f6f9716dfa7ecb574d2c48ddc307a2dcb23aa8f2",
    ],
)

rpm(
    name = "python3-0__3.9.25-8.el9.x86_64",
    sha256 = "46c7f847042b55a7a546263d57ee220940310caec4f2265aa31d7997918e8ea0",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-3.9.25-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46c7f847042b55a7a546263d57ee220940310caec4f2265aa31d7997918e8ea0",
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
        "https://storage.googleapis.com/builddeps/bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-9.el10.s390x",
    sha256 = "bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-configshell-1.1.30-9.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    ],
)

rpm(
    name = "python3-configshell-1__1.1.30-9.el10.x86_64",
    sha256 = "bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-configshell-1.1.30-9.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd3efd00e70a1cfcc68c0d973a5fb3fb34bd9863f30a1330070ba9b718acdf1b",
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
        "https://storage.googleapis.com/builddeps/6508de73c7fb8966b0d05f631af1002cb5238237791b0bd1b085384b9d6e15fd",
    ],
)

rpm(
    name = "python3-dbus-0__1.3.2-8.el10.s390x",
    sha256 = "98bf40e2dadd95cc0640650673b54d2dc7ebfa48bbba64a7857aede9c42550a5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-dbus-1.3.2-8.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/98bf40e2dadd95cc0640650673b54d2dc7ebfa48bbba64a7857aede9c42550a5",
    ],
)

rpm(
    name = "python3-dbus-0__1.3.2-8.el10.x86_64",
    sha256 = "95455a0bc5c76704ba2e46f5dd68b8bd47027b83ae551d62f02b15415a78a164",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-dbus-1.3.2-8.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95455a0bc5c76704ba2e46f5dd68b8bd47027b83ae551d62f02b15415a78a164",
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
        "https://storage.googleapis.com/builddeps/66267f4d40ef4d29b7084c60752688856a78949de1347292d4fe25be501e024b",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.46.0-7.el10.s390x",
    sha256 = "19a92f5cebbd47d89e69c63172d504154790e0ef013967d872ceb7ab0bc4b3f3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-gobject-base-3.46.0-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/19a92f5cebbd47d89e69c63172d504154790e0ef013967d872ceb7ab0bc4b3f3",
    ],
)

rpm(
    name = "python3-gobject-base-0__3.46.0-7.el10.x86_64",
    sha256 = "dd8582c736f50481252e556960f885c63af2d6a64888027d9345e35ec9bc0e27",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-gobject-base-3.46.0-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd8582c736f50481252e556960f885c63af2d6a64888027d9345e35ec9bc0e27",
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
        "https://storage.googleapis.com/builddeps/2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.46.0-7.el10.s390x",
    sha256 = "2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-gobject-base-noarch-3.46.0-7.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    ],
)

rpm(
    name = "python3-gobject-base-noarch-0__3.46.0-7.el10.x86_64",
    sha256 = "2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-gobject-base-noarch-3.46.0-7.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2c6d337336442bb1286b43facf50f9d7dad1398f86b312c93a3268d18d230824",
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
        "https://storage.googleapis.com/builddeps/75e70ceef21104220cd13f4de2cd23669c5660c23a07d15e471b39fa61418e8e",
    ],
)

rpm(
    name = "python3-kmod-0__0.9.2-6.el10.s390x",
    sha256 = "cc4bf90e94d8a7ac36762f21899fc174fe734aae502af314933685c0a342e832",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-kmod-0.9.2-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cc4bf90e94d8a7ac36762f21899fc174fe734aae502af314933685c0a342e832",
    ],
)

rpm(
    name = "python3-kmod-0__0.9.2-6.el10.x86_64",
    sha256 = "c9f50b595ee5a45bb14d571974662ed42c84bcb8660c3a78da891038847da651",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-kmod-0.9.2-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c9f50b595ee5a45bb14d571974662ed42c84bcb8660c3a78da891038847da651",
    ],
)

rpm(
    name = "python3-libs-0__3.12.13-2.el10.aarch64",
    sha256 = "744701a8fc1f9be467e9049697ded1344e63ddd85272881bdefe511f4c587ab9",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-libs-3.12.13-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/744701a8fc1f9be467e9049697ded1344e63ddd85272881bdefe511f4c587ab9",
    ],
)

rpm(
    name = "python3-libs-0__3.12.13-2.el10.s390x",
    sha256 = "23c01ab5a1c75268cd378150d584e77792b308e713438fdaf78ab3d8977eb92b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-libs-3.12.13-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/23c01ab5a1c75268cd378150d584e77792b308e713438fdaf78ab3d8977eb92b",
    ],
)

rpm(
    name = "python3-libs-0__3.12.13-2.el10.x86_64",
    sha256 = "f7349fd4bd4f3edf369f16771c87b4556b4a46e78ef8555c36fe8fb3367b25a0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-libs-3.12.13-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7349fd4bd4f3edf369f16771c87b4556b4a46e78ef8555c36fe8fb3367b25a0",
    ],
)

rpm(
    name = "python3-libs-0__3.9.25-8.el9.aarch64",
    sha256 = "1873e71815801127bf3dcb3d8e457b7963aaef7e7f02ee357912e71710f5032f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-libs-3.9.25-8.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1873e71815801127bf3dcb3d8e457b7963aaef7e7f02ee357912e71710f5032f",
    ],
)

rpm(
    name = "python3-libs-0__3.9.25-8.el9.s390x",
    sha256 = "a8d3b203708664f8b53f8dd201d3d52894002ea975cbe85df8bf5a7097095d35",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-libs-3.9.25-8.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a8d3b203708664f8b53f8dd201d3d52894002ea975cbe85df8bf5a7097095d35",
    ],
)

rpm(
    name = "python3-libs-0__3.9.25-8.el9.x86_64",
    sha256 = "fabc863ae6c1eb2d5e0aa768c5df5a4dfd2525490a7ecd6382758159814a394c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-libs-3.9.25-8.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fabc863ae6c1eb2d5e0aa768c5df5a4dfd2525490a7ecd6382758159814a394c",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.3.1-2.el9.aarch64",
    sha256 = "c8a53917081942a659da7f98c64137c5a7aab2b25fc6cb948a3ce4bef0b59309",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-21.3.1-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c8a53917081942a659da7f98c64137c5a7aab2b25fc6cb948a3ce4bef0b59309",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.3.1-2.el9.s390x",
    sha256 = "c8a53917081942a659da7f98c64137c5a7aab2b25fc6cb948a3ce4bef0b59309",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/python3-pip-wheel-21.3.1-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c8a53917081942a659da7f98c64137c5a7aab2b25fc6cb948a3ce4bef0b59309",
    ],
)

rpm(
    name = "python3-pip-wheel-0__21.3.1-2.el9.x86_64",
    sha256 = "c8a53917081942a659da7f98c64137c5a7aab2b25fc6cb948a3ce4bef0b59309",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-21.3.1-2.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c8a53917081942a659da7f98c64137c5a7aab2b25fc6cb948a3ce4bef0b59309",
    ],
)

rpm(
    name = "python3-pip-wheel-0__23.3.2-11.el10.aarch64",
    sha256 = "6bcd3976b56848076c8be3cacd495d39049ebeacc3dfd8dc70975df2f21d078a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-23.3.2-11.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6bcd3976b56848076c8be3cacd495d39049ebeacc3dfd8dc70975df2f21d078a",
    ],
)

rpm(
    name = "python3-pip-wheel-0__23.3.2-11.el10.s390x",
    sha256 = "6bcd3976b56848076c8be3cacd495d39049ebeacc3dfd8dc70975df2f21d078a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-pip-wheel-23.3.2-11.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6bcd3976b56848076c8be3cacd495d39049ebeacc3dfd8dc70975df2f21d078a",
    ],
)

rpm(
    name = "python3-pip-wheel-0__23.3.2-11.el10.x86_64",
    sha256 = "6bcd3976b56848076c8be3cacd495d39049ebeacc3dfd8dc70975df2f21d078a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-23.3.2-11.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6bcd3976b56848076c8be3cacd495d39049ebeacc3dfd8dc70975df2f21d078a",
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
        "https://storage.googleapis.com/builddeps/8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    ],
)

rpm(
    name = "python3-pyparsing-0__3.1.1-7.el10.s390x",
    sha256 = "8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-pyparsing-3.1.1-7.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    ],
)

rpm(
    name = "python3-pyparsing-0__3.1.1-7.el10.x86_64",
    sha256 = "8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-pyparsing-3.1.1-7.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8aef56a037934c4132e83b49893c0082351e96ab2c34cf3e14ee41472bb315e2",
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
        "https://storage.googleapis.com/builddeps/69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    ],
)

rpm(
    name = "python3-pyudev-0__0.24.1-10.el10.s390x",
    sha256 = "69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-pyudev-0.24.1-10.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    ],
)

rpm(
    name = "python3-pyudev-0__0.24.1-10.el10.x86_64",
    sha256 = "69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-pyudev-0.24.1-10.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/69e5069331c66c49738f7c558b3b78ec5aab81741af1d810629fb3a878a3f540",
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
        "https://storage.googleapis.com/builddeps/4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-12.el10.s390x",
    sha256 = "4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/python3-rtslib-2.1.76-12.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    ],
)

rpm(
    name = "python3-rtslib-0__2.1.76-12.el10.x86_64",
    sha256 = "4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/python3-rtslib-2.1.76-12.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4e91e035feac9802dedfe00460866864100d0c32f9b2c2a3a0c32789e307b63c",
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
        "https://storage.googleapis.com/builddeps/587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    ],
)

rpm(
    name = "python3-six-0__1.16.0-16.el10.s390x",
    sha256 = "587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-six-1.16.0-16.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    ],
)

rpm(
    name = "python3-six-0__1.16.0-16.el10.x86_64",
    sha256 = "587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-six-1.16.0-16.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/587391f25be67ed7389c4623f1260a16b33dfab99b5b7376e9eb72dafbc78403",
    ],
)

rpm(
    name = "python3-typing-extensions-0__4.9.0-6.el10.aarch64",
    sha256 = "d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-typing-extensions-4.9.0-6.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    ],
)

rpm(
    name = "python3-typing-extensions-0__4.9.0-6.el10.s390x",
    sha256 = "d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-typing-extensions-4.9.0-6.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    ],
)

rpm(
    name = "python3-typing-extensions-0__4.9.0-6.el10.x86_64",
    sha256 = "d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-typing-extensions-4.9.0-6.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d5e02bc63a658039701accb13f243d421082d21a64267e18fd04954d7d2938a8",
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
        "https://storage.googleapis.com/builddeps/b754bb3fe723d716e43404b58f706b506b14123fbfd3cad0fa016da10ce7aaf0",
    ],
)

rpm(
    name = "python3-urwid-0__2.5.3-4.el10.s390x",
    sha256 = "e73588c982971858102bb9a6804391709e72e5dd765034f5e4b1d7bea8c59332",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-urwid-2.5.3-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e73588c982971858102bb9a6804391709e72e5dd765034f5e4b1d7bea8c59332",
    ],
)

rpm(
    name = "python3-urwid-0__2.5.3-4.el10.x86_64",
    sha256 = "8ccc08409b227ee8b2cc6879a3b2f84a0e9cc792b720f3b5b7dd1381109a51cd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-urwid-2.5.3-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8ccc08409b227ee8b2cc6879a3b2f84a0e9cc792b720f3b5b7dd1381109a51cd",
    ],
)

rpm(
    name = "python3-wcwidth-0__0.2.6-6.el10.aarch64",
    sha256 = "0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/python3-wcwidth-0.2.6-6.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    ],
)

rpm(
    name = "python3-wcwidth-0__0.2.6-6.el10.s390x",
    sha256 = "0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/python3-wcwidth-0.2.6-6.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    ],
)

rpm(
    name = "python3-wcwidth-0__0.2.6-6.el10.x86_64",
    sha256 = "0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/python3-wcwidth-0.2.6-6.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0477cede1c6397494f32acfba7e6fba166e6f73811cf8c8e62a30aa7b3ae1af8",
    ],
)

rpm(
    name = "qemu-img-17__10.1.0-20.el9.aarch64",
    sha256 = "cfe398f86a3052183f19dc282c37a38fe2a1dfe3179215536cf47497079a080d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-img-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cfe398f86a3052183f19dc282c37a38fe2a1dfe3179215536cf47497079a080d",
    ],
)

rpm(
    name = "qemu-img-17__10.1.0-20.el9.s390x",
    sha256 = "f6aa0f452d223d8e44a9f40d302d256b66cfca4069305d5c9136296e859a5e71",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-img-10.1.0-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f6aa0f452d223d8e44a9f40d302d256b66cfca4069305d5c9136296e859a5e71",
    ],
)

rpm(
    name = "qemu-img-17__10.1.0-20.el9.x86_64",
    sha256 = "e869ced6b7c60507bb907ef1739e02e19f9cf2b0007b0e5d6247cca158d54b9a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-img-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e869ced6b7c60507bb907ef1739e02e19f9cf2b0007b0e5d6247cca158d54b9a",
    ],
)

rpm(
    name = "qemu-img-17__9.1.0-15.el9.x86_64",
    sha256 = "6149224d6968142db7c12330dd4d9f6882af2ad73a97e591214a3869603b663f",
    urls = ["https://storage.googleapis.com/builddeps/6149224d6968142db7c12330dd4d9f6882af2ad73a97e591214a3869603b663f"],
)

rpm(
    name = "qemu-img-18__10.1.0-21.el10.aarch64",
    sha256 = "6c4da550f28d3862815cd257ed0fe1a1b1c636ec3e210e2276445d14aad431d5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-img-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6c4da550f28d3862815cd257ed0fe1a1b1c636ec3e210e2276445d14aad431d5",
    ],
)

rpm(
    name = "qemu-img-18__10.1.0-21.el10.s390x",
    sha256 = "c77af19453f49911e2d691cc2aae6ee51fbf752121db93f311bcbb14c464b078",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-img-10.1.0-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c77af19453f49911e2d691cc2aae6ee51fbf752121db93f311bcbb14c464b078",
    ],
)

rpm(
    name = "qemu-img-18__10.1.0-21.el10.x86_64",
    sha256 = "c6010a3d572ff7f580d99bb2ea4f87c32cad8f8b5acfe6d6b77c8ffc64e4b62d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-img-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c6010a3d572ff7f580d99bb2ea4f87c32cad8f8b5acfe6d6b77c8ffc64e4b62d",
    ],
)

rpm(
    name = "qemu-kvm-common-17__10.1.0-20.el9.aarch64",
    sha256 = "ecd76c559c705ed2aa1ec392e798776feeecd1d10879ee689baf01e43c051a20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ecd76c559c705ed2aa1ec392e798776feeecd1d10879ee689baf01e43c051a20",
    ],
)

rpm(
    name = "qemu-kvm-common-17__10.1.0-20.el9.s390x",
    sha256 = "ff314df4856e3a0aca55baee198fe965ae658fbd003820ebf8ff38ce1bb94f9b",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-common-10.1.0-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ff314df4856e3a0aca55baee198fe965ae658fbd003820ebf8ff38ce1bb94f9b",
    ],
)

rpm(
    name = "qemu-kvm-common-17__10.1.0-20.el9.x86_64",
    sha256 = "196f38725da6b4f32f27ef93cb6c7a302c44b117b765a741a3177b24ed6cd329",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/196f38725da6b4f32f27ef93cb6c7a302c44b117b765a741a3177b24ed6cd329",
    ],
)

rpm(
    name = "qemu-kvm-common-17__9.1.0-15.el9.x86_64",
    sha256 = "345b3dae626a756f160321e025774d3d3e193a767388e69ffc832ea75988b166",
    urls = ["https://storage.googleapis.com/builddeps/345b3dae626a756f160321e025774d3d3e193a767388e69ffc832ea75988b166"],
)

rpm(
    name = "qemu-kvm-common-18__10.1.0-21.el10.aarch64",
    sha256 = "d360aa4073a15f5d61ccae2b7135bb18a748313ed53fff18d24c8a6cd2bf0a5f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-common-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d360aa4073a15f5d61ccae2b7135bb18a748313ed53fff18d24c8a6cd2bf0a5f",
    ],
)

rpm(
    name = "qemu-kvm-common-18__10.1.0-21.el10.s390x",
    sha256 = "78636b5994096a1a88d4b1b9f40269827b86d33e4918762dac6b3b188723b20c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-common-10.1.0-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/78636b5994096a1a88d4b1b9f40269827b86d33e4918762dac6b3b188723b20c",
    ],
)

rpm(
    name = "qemu-kvm-common-18__10.1.0-21.el10.x86_64",
    sha256 = "16f362c83a507dcad02f91953f2d53015aa21fc23c34744baaf0366bd242871a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-common-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/16f362c83a507dcad02f91953f2d53015aa21fc23c34744baaf0366bd242871a",
    ],
)

rpm(
    name = "qemu-kvm-core-17__10.1.0-20.el9.aarch64",
    sha256 = "8a3d5650a1d9643311a54c35912aa3d777ee37a2b9fb4a921dab3d2215589fd8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8a3d5650a1d9643311a54c35912aa3d777ee37a2b9fb4a921dab3d2215589fd8",
    ],
)

rpm(
    name = "qemu-kvm-core-17__10.1.0-20.el9.s390x",
    sha256 = "380fc78a907fd0353dcbcd0945258ab7e45a077766c51ec530b1e87c7dd08d32",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-core-10.1.0-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/380fc78a907fd0353dcbcd0945258ab7e45a077766c51ec530b1e87c7dd08d32",
    ],
)

rpm(
    name = "qemu-kvm-core-17__10.1.0-20.el9.x86_64",
    sha256 = "b800570361d1e67448aa6f05d127bd134a3792efbc5abc9f0ae5e176cb14149c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b800570361d1e67448aa6f05d127bd134a3792efbc5abc9f0ae5e176cb14149c",
    ],
)

rpm(
    name = "qemu-kvm-core-17__9.1.0-15.el9.x86_64",
    sha256 = "aa36521b947a78d2d06d90e1a8f5d74bab5ffbbb6d8ca8d939497477c4878565",
    urls = ["https://storage.googleapis.com/builddeps/aa36521b947a78d2d06d90e1a8f5d74bab5ffbbb6d8ca8d939497477c4878565"],
)

rpm(
    name = "qemu-kvm-core-18__10.1.0-21.el10.aarch64",
    sha256 = "bc94e30206d4e6c24d9b6ec995fac0991f2e39262559d7e5588f9de573ed1154",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-core-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bc94e30206d4e6c24d9b6ec995fac0991f2e39262559d7e5588f9de573ed1154",
    ],
)

rpm(
    name = "qemu-kvm-core-18__10.1.0-21.el10.s390x",
    sha256 = "e4ef76e75660a1d317c0a9241875aa22d0a065c9e14799e78b2afade45bd5003",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-core-10.1.0-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/e4ef76e75660a1d317c0a9241875aa22d0a065c9e14799e78b2afade45bd5003",
    ],
)

rpm(
    name = "qemu-kvm-core-18__10.1.0-21.el10.x86_64",
    sha256 = "2c486c95e57d003d3185e19d0ba8a4345c02c4d03650c8a9be5fc5eea0a26211",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-core-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c486c95e57d003d3185e19d0ba8a4345c02c4d03650c8a9be5fc5eea0a26211",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__10.1.0-20.el9.aarch64",
    sha256 = "b23daaabe4691f11e648133768ce120c88ba36e8de0abc3d26027797a4784b48",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b23daaabe4691f11e648133768ce120c88ba36e8de0abc3d26027797a4784b48",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__10.1.0-20.el9.s390x",
    sha256 = "b1bccb256ac06f42b50f88809d92248dd1d6accb351103e564ba560e36af6506",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b1bccb256ac06f42b50f88809d92248dd1d6accb351103e564ba560e36af6506",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-17__10.1.0-20.el9.x86_64",
    sha256 = "fbf645b1438bc7c43d808c9dfc87365391c7441432a88f7546d848c75cf4858f",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fbf645b1438bc7c43d808c9dfc87365391c7441432a88f7546d848c75cf4858f",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-18__10.1.0-21.el10.aarch64",
    sha256 = "6f3f2a6fdd5f8771f19c86d2fd02ff55f8b200c9216c8a4f162d3c950843dcdd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6f3f2a6fdd5f8771f19c86d2fd02ff55f8b200c9216c8a4f162d3c950843dcdd",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-18__10.1.0-21.el10.s390x",
    sha256 = "ee45155af64e77883a68c8c63dfab2a55cc8ef74affc705c59ac6f6c46c9ab50",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ee45155af64e77883a68c8c63dfab2a55cc8ef74affc705c59ac6f6c46c9ab50",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-18__10.1.0-21.el10.x86_64",
    sha256 = "778ec7d982d81e1106e9bd6b770e696084aea63a65de9a5646143ac86bc7a55c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/778ec7d982d81e1106e9bd6b770e696084aea63a65de9a5646143ac86bc7a55c",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-ccw-17__10.1.0-20.el9.s390x",
    sha256 = "af0a3f71ef2460585fb92411dd7736c6eab70f4673b828054e27756c9edc789c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-10.1.0-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/af0a3f71ef2460585fb92411dd7736c6eab70f4673b828054e27756c9edc789c",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-ccw-18__10.1.0-21.el10.s390x",
    sha256 = "bb985991022f6e8c472ed9aa544dcf86e63e5da115f78976967c1be72db7fa1b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-device-display-virtio-gpu-ccw-10.1.0-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bb985991022f6e8c472ed9aa544dcf86e63e5da115f78976967c1be72db7fa1b",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__10.1.0-20.el9.aarch64",
    sha256 = "d277a987023b83f1b82ea1350e30dd1b8936f4ad47842f802afc2df353760a97",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d277a987023b83f1b82ea1350e30dd1b8936f4ad47842f802afc2df353760a97",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-17__10.1.0-20.el9.x86_64",
    sha256 = "3096876abf4d2d7429a006cc9c0958057e58cd69d3a553f356d169abf5b6b886",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3096876abf4d2d7429a006cc9c0958057e58cd69d3a553f356d169abf5b6b886",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-18__10.1.0-21.el10.aarch64",
    sha256 = "8f86641d41b6026075d31c83ddd71c704df8abf1ea07c8ea975bd9b67eb5520c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8f86641d41b6026075d31c83ddd71c704df8abf1ea07c8ea975bd9b67eb5520c",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-gpu-pci-18__10.1.0-21.el10.x86_64",
    sha256 = "a63669a773db34dc5b0cb994c61cc9a99b7bb85471011c7781e40b55002298b6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-gpu-pci-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a63669a773db34dc5b0cb994c61cc9a99b7bb85471011c7781e40b55002298b6",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-vga-17__10.1.0-20.el9.x86_64",
    sha256 = "7bc9d686247abacbc87d910277c669bc103097ddd9d9c666185d2595cac96497",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-vga-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7bc9d686247abacbc87d910277c669bc103097ddd9d9c666185d2595cac96497",
    ],
)

rpm(
    name = "qemu-kvm-device-display-virtio-vga-18__10.1.0-21.el10.x86_64",
    sha256 = "c860f5354e689ed68d99ef43efa46135ea2ccc8e422eaef3b72ceed29164081c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-display-virtio-vga-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c860f5354e689ed68d99ef43efa46135ea2ccc8e422eaef3b72ceed29164081c",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__10.1.0-20.el9.aarch64",
    sha256 = "f760249c3bd6fc11c22d42e4a949d03f74fb0e6e0ec56a1ef1e93c2d780ee2b9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f760249c3bd6fc11c22d42e4a949d03f74fb0e6e0ec56a1ef1e93c2d780ee2b9",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__10.1.0-20.el9.s390x",
    sha256 = "48bb4b56049c5c12c12a2e68381e56febcaac31e136d77a7e9354accf8542dbd",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-10.1.0-20.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/48bb4b56049c5c12c12a2e68381e56febcaac31e136d77a7e9354accf8542dbd",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-17__10.1.0-20.el9.x86_64",
    sha256 = "8038ea2415a8b8d77ff4ccd33dd4af952b0239f9c520415851ec029c6c6a89e6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8038ea2415a8b8d77ff4ccd33dd4af952b0239f9c520415851ec029c6c6a89e6",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-18__10.1.0-21.el10.aarch64",
    sha256 = "6820d475b369fba3ad123091c2f05ff3d59063ad44078e54969087c10a839145",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-host-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6820d475b369fba3ad123091c2f05ff3d59063ad44078e54969087c10a839145",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-18__10.1.0-21.el10.s390x",
    sha256 = "71c9c2e7f1e118400f185e66b6f027cc1a33e5d853d82bc9f26891411932a598",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/qemu-kvm-device-usb-host-10.1.0-21.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/71c9c2e7f1e118400f185e66b6f027cc1a33e5d853d82bc9f26891411932a598",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-host-18__10.1.0-21.el10.x86_64",
    sha256 = "165d0e6282f9231607a3be519e9ff4312ba8ab628ad01129ab29bc5ac996f7eb",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-host-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/165d0e6282f9231607a3be519e9ff4312ba8ab628ad01129ab29bc5ac996f7eb",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__10.1.0-20.el9.aarch64",
    sha256 = "1dc328c0870ae0a476e372b9d89e809df7fe744f1bcfd11dd3d3b1089bff1118",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-20.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1dc328c0870ae0a476e372b9d89e809df7fe744f1bcfd11dd3d3b1089bff1118",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-17__10.1.0-20.el9.x86_64",
    sha256 = "4a6eccdfadab9c0df838f22ced89abb556a36f82aaa820cb2302eb365acc3b42",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-20.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a6eccdfadab9c0df838f22ced89abb556a36f82aaa820cb2302eb365acc3b42",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-18__10.1.0-21.el10.aarch64",
    sha256 = "98dc7179dc66f1c09d8a880b01915da756dd4203bac233935578d3471ab45b91",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98dc7179dc66f1c09d8a880b01915da756dd4203bac233935578d3471ab45b91",
    ],
)

rpm(
    name = "qemu-kvm-device-usb-redirect-18__10.1.0-21.el10.x86_64",
    sha256 = "6877294d99da12ab3c94dc56b5b075d1ed85f52a2fb8f49e799ad9dfb80b1452",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-kvm-device-usb-redirect-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6877294d99da12ab3c94dc56b5b075d1ed85f52a2fb8f49e799ad9dfb80b1452",
    ],
)

rpm(
    name = "qemu-pr-helper-17__10.1.0-22.el9.aarch64",
    sha256 = "04a3ce6259271841df1c3c60de1707fdece399a4c704b7e7a69baf0bad60886d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-10.1.0-22.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/04a3ce6259271841df1c3c60de1707fdece399a4c704b7e7a69baf0bad60886d",
    ],
)

rpm(
    name = "qemu-pr-helper-17__10.1.0-22.el9.x86_64",
    sha256 = "21f992daf97353f0fee544a3e6c1b4be4f6e8206eec5185cc59086e87164f60a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-10.1.0-22.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21f992daf97353f0fee544a3e6c1b4be4f6e8206eec5185cc59086e87164f60a",
    ],
)

rpm(
    name = "qemu-pr-helper-18__10.1.0-21.el10.aarch64",
    sha256 = "b302013dd4c020067e1d323acacc47befefccf2fccdde72944389f4f124a6ab2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/qemu-pr-helper-10.1.0-21.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b302013dd4c020067e1d323acacc47befefccf2fccdde72944389f4f124a6ab2",
    ],
)

rpm(
    name = "qemu-pr-helper-18__10.1.0-21.el10.x86_64",
    sha256 = "bc70157551d5f57bbf294c5190923f362bd24865100da396301426dd8f78bd23",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/qemu-pr-helper-10.1.0-21.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc70157551d5f57bbf294c5190923f362bd24865100da396301426dd8f78bd23",
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
        "https://storage.googleapis.com/builddeps/a1f1fe411d40cb802c7a3e3b105faffe05c2376563bec2d59c71ed28778684cd",
    ],
)

rpm(
    name = "readline-0__8.2-11.el10.s390x",
    sha256 = "fd8cc3c7dd19bf773afb0488b2521d1710dfc1d8337fbff55c9f8d84572a4f9f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/readline-8.2-11.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/fd8cc3c7dd19bf773afb0488b2521d1710dfc1d8337fbff55c9f8d84572a4f9f",
    ],
)

rpm(
    name = "readline-0__8.2-11.el10.x86_64",
    sha256 = "d8e2d7c011d0e5c56b6875919ce036605862db02d59d6983d470bc5757021783",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/readline-8.2-11.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d8e2d7c011d0e5c56b6875919ce036605862db02d59d6983d470bc5757021783",
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
    name = "rpm-0__4.19.1.1-25.el10.s390x",
    sha256 = "8b7b1548695a2db69449ccb15ca22b152f00a66deeb3e9d1705538a92b76321e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/rpm-4.19.1.1-25.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8b7b1548695a2db69449ccb15ca22b152f00a66deeb3e9d1705538a92b76321e",
    ],
)

rpm(
    name = "rpm-0__4.19.1.1-25.el10.x86_64",
    sha256 = "835c9acc3b89ed1f0a0f96831b65809f346797b8cc51ef62f13840194fdac93c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/rpm-4.19.1.1-25.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/835c9acc3b89ed1f0a0f96831b65809f346797b8cc51ef62f13840194fdac93c",
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
    name = "rpm-libs-0__4.19.1.1-25.el10.s390x",
    sha256 = "0ccde9a354f5a8f881be107f840074e437f2d95a510625b39080876dc1581d45",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/rpm-libs-4.19.1.1-25.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/0ccde9a354f5a8f881be107f840074e437f2d95a510625b39080876dc1581d45",
    ],
)

rpm(
    name = "rpm-libs-0__4.19.1.1-25.el10.x86_64",
    sha256 = "39243f122fa092a46e03dbb2bf44213bc8d6cc1062f292a8e408d86a53e877fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.19.1.1-25.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/39243f122fa092a46e03dbb2bf44213bc8d6cc1062f292a8e408d86a53e877fe",
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
    name = "rpm-sequoia-0__1.10.2.1-1.el10.s390x",
    sha256 = "4318a227b602ca15fd936a77ae87c5c2787433ddd3f21ae81340bd3bbea8a97f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/rpm-sequoia-1.10.2.1-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4318a227b602ca15fd936a77ae87c5c2787433ddd3f21ae81340bd3bbea8a97f",
    ],
)

rpm(
    name = "rpm-sequoia-0__1.10.2.1-1.el10.x86_64",
    sha256 = "c862dede85225722f77d720a52dc0c8cb5ca74d982c8491de7ebc47f170ca1cf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/rpm-sequoia-1.10.2.1-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c862dede85225722f77d720a52dc0c8cb5ca74d982c8491de7ebc47f170ca1cf",
    ],
)

rpm(
    name = "scrub-0__2.6.1-11.el10.s390x",
    sha256 = "527e3cf6d20579cbc13efd1b13c639c243e538b6e6121a22efd46ec13d2fa557",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/scrub-2.6.1-11.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/527e3cf6d20579cbc13efd1b13c639c243e538b6e6121a22efd46ec13d2fa557",
    ],
)

rpm(
    name = "scrub-0__2.6.1-11.el10.x86_64",
    sha256 = "935258a3ef8ada2d8cba193df349c9d1dc62d38018b9613aab6f727f93655f85",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/scrub-2.6.1-11.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/935258a3ef8ada2d8cba193df349c9d1dc62d38018b9613aab6f727f93655f85",
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
        "https://storage.googleapis.com/builddeps/18044b16fa0f0256167f42ba6ab1f8b5ac338747e150d3c9aead064cd28255c9",
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
        "https://storage.googleapis.com/builddeps/5edf7ad5039c74faab0fe3bc7f9741db6153c6f9ebe3367d20701d4e659d930d",
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
        "https://storage.googleapis.com/builddeps/5ed6563e3d13189aa28fe86d0fef8540d61539aa44dd2d5558ca068e79df4ea2",
    ],
)

rpm(
    name = "sed-0__4.8-10.el9.aarch64",
    sha256 = "5a2930318f5ca770e800b2a42c05c945ccb02cd8ea3ed2b177d759d0e9090d5d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sed-4.8-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5a2930318f5ca770e800b2a42c05c945ccb02cd8ea3ed2b177d759d0e9090d5d",
    ],
)

rpm(
    name = "sed-0__4.8-10.el9.s390x",
    sha256 = "a515c69e92880844e6fbcf690421bd0d44304b642e5e56392a00ede362da5056",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sed-4.8-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a515c69e92880844e6fbcf690421bd0d44304b642e5e56392a00ede362da5056",
    ],
)

rpm(
    name = "sed-0__4.8-10.el9.x86_64",
    sha256 = "8db670e1de34148e71c07f4ed8dbd5f41e1d6717325d5912a8651aa4e063b9e7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sed-4.8-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8db670e1de34148e71c07f4ed8dbd5f41e1d6717325d5912a8651aa4e063b9e7",
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
    name = "sed-0__4.9-5.el10.aarch64",
    sha256 = "9c03d6148a319111bce62ba46e859c17c6615f51648788b499e28cbb429d1390",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/sed-4.9-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9c03d6148a319111bce62ba46e859c17c6615f51648788b499e28cbb429d1390",
    ],
)

rpm(
    name = "sed-0__4.9-5.el10.s390x",
    sha256 = "428403e0e45d47510ffc7bebbe47a7e981bbe5347432f4b055e23c199482cbd3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/sed-4.9-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/428403e0e45d47510ffc7bebbe47a7e981bbe5347432f4b055e23c199482cbd3",
    ],
)

rpm(
    name = "sed-0__4.9-5.el10.x86_64",
    sha256 = "b5c61c0b1b90892375c36acaf52f8a19e68397a18703687df8b1883792485b7c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/sed-4.9-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b5c61c0b1b90892375c36acaf52f8a19e68397a18703687df8b1883792485b7c",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.53-2.el9.x86_64",
    sha256 = "6840efbf87f7f4782c332e0e0a3e3567075a804c070b1d501ff7e7a44a09448c",
    urls = ["https://storage.googleapis.com/builddeps/6840efbf87f7f4782c332e0e0a3e3567075a804c070b1d501ff7e7a44a09448c"],
)

rpm(
    name = "selinux-policy-0__38.1.83-1.el9.aarch64",
    sha256 = "61b55486bc8fe2f401a9961b9a643ad8f97f123648a047fdd312d341bcdb5572",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-38.1.83-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61b55486bc8fe2f401a9961b9a643ad8f97f123648a047fdd312d341bcdb5572",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.83-1.el9.s390x",
    sha256 = "61b55486bc8fe2f401a9961b9a643ad8f97f123648a047fdd312d341bcdb5572",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-38.1.83-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61b55486bc8fe2f401a9961b9a643ad8f97f123648a047fdd312d341bcdb5572",
    ],
)

rpm(
    name = "selinux-policy-0__38.1.83-1.el9.x86_64",
    sha256 = "61b55486bc8fe2f401a9961b9a643ad8f97f123648a047fdd312d341bcdb5572",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-38.1.83-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61b55486bc8fe2f401a9961b9a643ad8f97f123648a047fdd312d341bcdb5572",
    ],
)

rpm(
    name = "selinux-policy-0__42.1.23-1.el10.aarch64",
    sha256 = "aa8ea6ed2a6169d60def98335a7a84635a62a3974eafd87da3662f213877c4ec",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/selinux-policy-42.1.23-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aa8ea6ed2a6169d60def98335a7a84635a62a3974eafd87da3662f213877c4ec",
    ],
)

rpm(
    name = "selinux-policy-0__42.1.23-1.el10.s390x",
    sha256 = "aa8ea6ed2a6169d60def98335a7a84635a62a3974eafd87da3662f213877c4ec",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/selinux-policy-42.1.23-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aa8ea6ed2a6169d60def98335a7a84635a62a3974eafd87da3662f213877c4ec",
    ],
)

rpm(
    name = "selinux-policy-0__42.1.23-1.el10.x86_64",
    sha256 = "aa8ea6ed2a6169d60def98335a7a84635a62a3974eafd87da3662f213877c4ec",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/selinux-policy-42.1.23-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aa8ea6ed2a6169d60def98335a7a84635a62a3974eafd87da3662f213877c4ec",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.53-2.el9.x86_64",
    sha256 = "b9f921bdc764af3b8c5c8580fc9db4f75b0fb3b2c0a3ea1f541536de091664b1",
    urls = ["https://storage.googleapis.com/builddeps/b9f921bdc764af3b8c5c8580fc9db4f75b0fb3b2c0a3ea1f541536de091664b1"],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.83-1.el9.aarch64",
    sha256 = "d30be63386d8d453a7f753206d4640195ce52c64576a3138f5075f8e4ac582ec",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-38.1.83-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d30be63386d8d453a7f753206d4640195ce52c64576a3138f5075f8e4ac582ec",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.83-1.el9.s390x",
    sha256 = "d30be63386d8d453a7f753206d4640195ce52c64576a3138f5075f8e4ac582ec",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-38.1.83-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d30be63386d8d453a7f753206d4640195ce52c64576a3138f5075f8e4ac582ec",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__38.1.83-1.el9.x86_64",
    sha256 = "d30be63386d8d453a7f753206d4640195ce52c64576a3138f5075f8e4ac582ec",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-38.1.83-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d30be63386d8d453a7f753206d4640195ce52c64576a3138f5075f8e4ac582ec",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__42.1.23-1.el10.aarch64",
    sha256 = "a67fe41a6cd6d0a3bd4b8c9a2fde166bc29424d68211380feba1f0c60c96f6f5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-42.1.23-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a67fe41a6cd6d0a3bd4b8c9a2fde166bc29424d68211380feba1f0c60c96f6f5",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__42.1.23-1.el10.s390x",
    sha256 = "a67fe41a6cd6d0a3bd4b8c9a2fde166bc29424d68211380feba1f0c60c96f6f5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/selinux-policy-targeted-42.1.23-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a67fe41a6cd6d0a3bd4b8c9a2fde166bc29424d68211380feba1f0c60c96f6f5",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__42.1.23-1.el10.x86_64",
    sha256 = "a67fe41a6cd6d0a3bd4b8c9a2fde166bc29424d68211380feba1f0c60c96f6f5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-42.1.23-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a67fe41a6cd6d0a3bd4b8c9a2fde166bc29424d68211380feba1f0c60c96f6f5",
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
        "https://storage.googleapis.com/builddeps/bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    ],
)

rpm(
    name = "setup-0__2.14.5-7.el10.s390x",
    sha256 = "bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/setup-2.14.5-7.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    ],
)

rpm(
    name = "setup-0__2.14.5-7.el10.x86_64",
    sha256 = "bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/setup-2.14.5-7.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/bd7fb604e635ec8e49abc330cb15e9f30dcc1c6f248495308acd83e41896b29e",
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
        "https://storage.googleapis.com/builddeps/790b23bb704c9c42b9478b859f909186751771d9c8e864b5b1eb7a0158e690ca",
    ],
)

rpm(
    name = "shadow-utils-2__4.15.0-12.el10.aarch64",
    sha256 = "e7baf38a6d041494ea36a13015aa76d33077516b77d51f0f6775416200b04f64",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.15.0-12.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e7baf38a6d041494ea36a13015aa76d33077516b77d51f0f6775416200b04f64",
    ],
)

rpm(
    name = "shadow-utils-2__4.15.0-12.el10.s390x",
    sha256 = "54e7b369b033569d7552dd8569d8923186f5df23b56effb9f32d22df91578f9b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/shadow-utils-4.15.0-12.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/54e7b369b033569d7552dd8569d8923186f5df23b56effb9f32d22df91578f9b",
    ],
)

rpm(
    name = "shadow-utils-2__4.15.0-12.el10.x86_64",
    sha256 = "e65efc46ae8363edad023760b922d1bb51d418ce386a41cb4ae3f77aec3f1fef",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.15.0-12.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e65efc46ae8363edad023760b922d1bb51d418ce386a41cb4ae3f77aec3f1fef",
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
    name = "shadow-utils-2__4.9-17.el9.aarch64",
    sha256 = "3edd4c583815a1e74b05972137144264bd2fe062106f63697fddffc4a5fc957d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.9-17.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3edd4c583815a1e74b05972137144264bd2fe062106f63697fddffc4a5fc957d",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-17.el9.s390x",
    sha256 = "df349811eb3501d6653321753b4bd37f15c69a024f9601208c974b53058b66ae",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/shadow-utils-4.9-17.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/df349811eb3501d6653321753b4bd37f15c69a024f9601208c974b53058b66ae",
    ],
)

rpm(
    name = "shadow-utils-2__4.9-17.el9.x86_64",
    sha256 = "1b9b0829668ce68f0ff0904fa651005e9c0c5e53b7481adb41f8f6b758d9e36a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.9-17.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1b9b0829668ce68f0ff0904fa651005e9c0c5e53b7481adb41f8f6b758d9e36a",
    ],
)

rpm(
    name = "snappy-0__1.1.10-7.el10.aarch64",
    sha256 = "cc7bc94dc673d8d6d5b4559036648410e790e2c59e2254bc6acd1578fb5e6781",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/snappy-1.1.10-7.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cc7bc94dc673d8d6d5b4559036648410e790e2c59e2254bc6acd1578fb5e6781",
    ],
)

rpm(
    name = "snappy-0__1.1.10-7.el10.s390x",
    sha256 = "03a4ac68f64e146332224557a251a9d051dada615815dd6eb2f4bb22b73826e0",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/snappy-1.1.10-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/03a4ac68f64e146332224557a251a9d051dada615815dd6eb2f4bb22b73826e0",
    ],
)

rpm(
    name = "snappy-0__1.1.10-7.el10.x86_64",
    sha256 = "952dcfbe66d93bece4a4f3753ce721594acbd2af82cd5ca02bf9028375c136b3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/snappy-1.1.10-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/952dcfbe66d93bece4a4f3753ce721594acbd2af82cd5ca02bf9028375c136b3",
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
    name = "sqlite-libs-0__3.34.1-10.el9.aarch64",
    sha256 = "249e02ba4ebd53311c9fa9e5604d88e9a6642edfa8873f274463feec0438d24d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.34.1-10.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/249e02ba4ebd53311c9fa9e5604d88e9a6642edfa8873f274463feec0438d24d",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-10.el9.s390x",
    sha256 = "46ddfde17c746f5c93e562064f1f9759a9c334fd65e199ef4f2a0fd32d70e077",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sqlite-libs-3.34.1-10.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/46ddfde17c746f5c93e562064f1f9759a9c334fd65e199ef4f2a0fd32d70e077",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.1-10.el9.x86_64",
    sha256 = "33e446234418090d66106865df8d65aa32d9021c9105cd3029e7a2a912fffac9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.34.1-10.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/33e446234418090d66106865df8d65aa32d9021c9105cd3029e7a2a912fffac9",
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
    name = "sqlite-libs-0__3.46.1-5.el10.aarch64",
    sha256 = "217f00d515ac790fd028f0fd70a195a288258d0e1157ce6293ab65d29a965cf1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.46.1-5.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/217f00d515ac790fd028f0fd70a195a288258d0e1157ce6293ab65d29a965cf1",
    ],
)

rpm(
    name = "sqlite-libs-0__3.46.1-5.el10.s390x",
    sha256 = "34d97be1a2df9d53a327cec2ca15887897168b880a19a4b7af2b860ad80b35fe",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/sqlite-libs-3.46.1-5.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/34d97be1a2df9d53a327cec2ca15887897168b880a19a4b7af2b860ad80b35fe",
    ],
)

rpm(
    name = "sqlite-libs-0__3.46.1-5.el10.x86_64",
    sha256 = "fa8bd71adaf88ff1b893731fd5f49c949cf3f618332c9b80390113237699f8e7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.46.1-5.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fa8bd71adaf88ff1b893731fd5f49c949cf3f618332c9b80390113237699f8e7",
    ],
)

rpm(
    name = "sssd-client-0__2.13.0-1.el10.aarch64",
    sha256 = "18a4cf6ce2a80722f80c14eaabb71fd2de34894922de6039bff55aebaa072360",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/sssd-client-2.13.0-1.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/18a4cf6ce2a80722f80c14eaabb71fd2de34894922de6039bff55aebaa072360",
    ],
)

rpm(
    name = "sssd-client-0__2.13.0-1.el10.s390x",
    sha256 = "cdc999bd2fb09a5865b0a0996accd75a98d9d5e03dc0639b3978229a2a063d44",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/sssd-client-2.13.0-1.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/cdc999bd2fb09a5865b0a0996accd75a98d9d5e03dc0639b3978229a2a063d44",
    ],
)

rpm(
    name = "sssd-client-0__2.13.0-1.el10.x86_64",
    sha256 = "6ea9a201c36016d8d1fc94fccc7c3e1399e62a5ce62b10776efdfcb4ac6aa198",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/sssd-client-2.13.0-1.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6ea9a201c36016d8d1fc94fccc7c3e1399e62a5ce62b10776efdfcb4ac6aa198",
    ],
)

rpm(
    name = "sssd-client-0__2.9.9-3.el9.aarch64",
    sha256 = "0bee71fbf2ab25584fb75c037c8278312cb189b278019d97e65c2969a303b7be",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/sssd-client-2.9.9-3.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0bee71fbf2ab25584fb75c037c8278312cb189b278019d97e65c2969a303b7be",
    ],
)

rpm(
    name = "sssd-client-0__2.9.9-3.el9.s390x",
    sha256 = "103b8097c4557c5041d492bdba1a3ebf3ced3d0740a25501e234cae5c18d343d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/sssd-client-2.9.9-3.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/103b8097c4557c5041d492bdba1a3ebf3ced3d0740a25501e234cae5c18d343d",
    ],
)

rpm(
    name = "sssd-client-0__2.9.9-3.el9.x86_64",
    sha256 = "bb0f6fe7ee225bdb68d71b87253435bbb16acdd950f385524aa7a427cc838696",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/sssd-client-2.9.9-3.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb0f6fe7ee225bdb68d71b87253435bbb16acdd950f385524aa7a427cc838696",
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
        "https://storage.googleapis.com/builddeps/2ab32944b56a5d288754d90a3758d667cdc3703631488a2c2f4ac357880bff0b",
    ],
)

rpm(
    name = "swtpm-0__0.9.0-2.el10.s390x",
    sha256 = "4fc33cbd8611b571b7952968cd67999ca3d457f7d331290ed5850f70d292f89d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/swtpm-0.9.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4fc33cbd8611b571b7952968cd67999ca3d457f7d331290ed5850f70d292f89d",
    ],
)

rpm(
    name = "swtpm-0__0.9.0-2.el10.x86_64",
    sha256 = "2754a70eda7d481964e28e610d493f0c705ae966e75dda10ee901a6cf2ef5919",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/swtpm-0.9.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2754a70eda7d481964e28e610d493f0c705ae966e75dda10ee901a6cf2ef5919",
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
        "https://storage.googleapis.com/builddeps/e4233f1d21b64737a8c42fecaa652b2388b897a8748b416cf4bd599f30dd7fe2",
    ],
)

rpm(
    name = "swtpm-libs-0__0.9.0-2.el10.s390x",
    sha256 = "2cd497257c5a03b6e579f3ada2bef874350a0fbb0aad78cff5d38e247af20c0f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/swtpm-libs-0.9.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/2cd497257c5a03b6e579f3ada2bef874350a0fbb0aad78cff5d38e247af20c0f",
    ],
)

rpm(
    name = "swtpm-libs-0__0.9.0-2.el10.x86_64",
    sha256 = "57b1c9b2ab6540e9504f32e1aa58331fc98ad9476e47b82907f42ab17ab5288a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/swtpm-libs-0.9.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/57b1c9b2ab6540e9504f32e1aa58331fc98ad9476e47b82907f42ab17ab5288a",
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
        "https://storage.googleapis.com/builddeps/7da6702303b52d8724152e1235f9e3a1eca4dbb7e2dc2ce51f0b32ac6e04aef9",
    ],
)

rpm(
    name = "swtpm-tools-0__0.9.0-2.el10.s390x",
    sha256 = "764ead866ad117155085de09e2d0cf5c483ef9aeb134421ae5cc00892aed146e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/swtpm-tools-0.9.0-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/764ead866ad117155085de09e2d0cf5c483ef9aeb134421ae5cc00892aed146e",
    ],
)

rpm(
    name = "swtpm-tools-0__0.9.0-2.el10.x86_64",
    sha256 = "4c16e59ac5ef48d0e83d6e0a83ba1838c25c6531fb3ef3564bf98643e1e70503",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/swtpm-tools-0.9.0-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c16e59ac5ef48d0e83d6e0a83ba1838c25c6531fb3ef3564bf98643e1e70503",
    ],
)

rpm(
    name = "systemd-0__252-51.el9.x86_64",
    sha256 = "c5e5ae6f65f085c9f811a2a7950920eecb0c7ddf3d82c3f63b5662231cfc5de0",
    urls = ["https://storage.googleapis.com/builddeps/c5e5ae6f65f085c9f811a2a7950920eecb0c7ddf3d82c3f63b5662231cfc5de0"],
)

rpm(
    name = "systemd-0__252-71.el9.aarch64",
    sha256 = "69791f61b592609e6a178f6c8879aeff9f574a8c5d52fafd5f4e7bfadf593ff7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-252-71.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/69791f61b592609e6a178f6c8879aeff9f574a8c5d52fafd5f4e7bfadf593ff7",
    ],
)

rpm(
    name = "systemd-0__252-71.el9.s390x",
    sha256 = "f92f629607bfab87b1e3475e2ceb51d1b161db6ae304baf761e994d35c1929ee",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-252-71.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/f92f629607bfab87b1e3475e2ceb51d1b161db6ae304baf761e994d35c1929ee",
    ],
)

rpm(
    name = "systemd-0__252-71.el9.x86_64",
    sha256 = "762f43784312f9e19c5f1e39381fbe91f2d1601fc6a711c586063e8909ef7378",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-252-71.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/762f43784312f9e19c5f1e39381fbe91f2d1601fc6a711c586063e8909ef7378",
    ],
)

rpm(
    name = "systemd-0__257-27.el10.aarch64",
    sha256 = "4a40d05483fc57e6a77bbbcfced04e9e5edcd0997c646a4bc3a0807e1e1c6c95",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-257-27.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4a40d05483fc57e6a77bbbcfced04e9e5edcd0997c646a4bc3a0807e1e1c6c95",
    ],
)

rpm(
    name = "systemd-0__257-27.el10.s390x",
    sha256 = "bcf764fa541dcbdd6181d434aaacd3850dbb5e5f0adcbdd2d6d922fd6f832b56",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-257-27.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/bcf764fa541dcbdd6181d434aaacd3850dbb5e5f0adcbdd2d6d922fd6f832b56",
    ],
)

rpm(
    name = "systemd-0__257-27.el10.x86_64",
    sha256 = "9f3a21ad44080df6df29834c64e05767b10bf4fcbcef13e2c1e0e5b8e9dd821a",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-257-27.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9f3a21ad44080df6df29834c64e05767b10bf4fcbcef13e2c1e0e5b8e9dd821a",
    ],
)

rpm(
    name = "systemd-container-0__252-51.el9.x86_64",
    sha256 = "653fcd14047fb557e3a3f5da47c83d6ceb2194169f3ef42a27566bb4e2102dde",
    urls = ["https://storage.googleapis.com/builddeps/653fcd14047fb557e3a3f5da47c83d6ceb2194169f3ef42a27566bb4e2102dde"],
)

rpm(
    name = "systemd-container-0__252-71.el9.aarch64",
    sha256 = "bf3a401d43b2e20742a10256863f1044119261d4e9e2afdd1c0b18e8f61996c7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-container-252-71.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bf3a401d43b2e20742a10256863f1044119261d4e9e2afdd1c0b18e8f61996c7",
    ],
)

rpm(
    name = "systemd-container-0__252-71.el9.s390x",
    sha256 = "49a2b868ba9d973337000582a69299d8b2aa4bed40bb854dc780c62db55646d8",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-container-252-71.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/49a2b868ba9d973337000582a69299d8b2aa4bed40bb854dc780c62db55646d8",
    ],
)

rpm(
    name = "systemd-container-0__252-71.el9.x86_64",
    sha256 = "c2ebf5db03becec286c6ea1886374f591c8a6afe9a2edad6900dce0cd2443f83",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-container-252-71.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c2ebf5db03becec286c6ea1886374f591c8a6afe9a2edad6900dce0cd2443f83",
    ],
)

rpm(
    name = "systemd-container-0__257-27.el10.aarch64",
    sha256 = "76b16b371d2693b4014ad6a3c9953847efada170aa54f978ba7ce78813aab1cf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-container-257-27.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76b16b371d2693b4014ad6a3c9953847efada170aa54f978ba7ce78813aab1cf",
    ],
)

rpm(
    name = "systemd-container-0__257-27.el10.s390x",
    sha256 = "b16a30d164257d13fd74bfd764f308237eebea37860a83581f030a2e949c29bf",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-container-257-27.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b16a30d164257d13fd74bfd764f308237eebea37860a83581f030a2e949c29bf",
    ],
)

rpm(
    name = "systemd-container-0__257-27.el10.x86_64",
    sha256 = "0b62a25ddd5e5c2335f70a424b8bb7e623ada4a0a80ef5cff6a9662491a362a2",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-container-257-27.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0b62a25ddd5e5c2335f70a424b8bb7e623ada4a0a80ef5cff6a9662491a362a2",
    ],
)

rpm(
    name = "systemd-libs-0__252-51.el9.x86_64",
    sha256 = "a9d02a16bbc778ad3a2b46b8740fa821df065cdacd6ba8570c3301dacad79f0f",
    urls = ["https://storage.googleapis.com/builddeps/a9d02a16bbc778ad3a2b46b8740fa821df065cdacd6ba8570c3301dacad79f0f"],
)

rpm(
    name = "systemd-libs-0__252-71.el9.aarch64",
    sha256 = "51586771ef916e4a75535c08cf08a25a38c919c9917c04ac936a24fad20b8970",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-libs-252-71.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/51586771ef916e4a75535c08cf08a25a38c919c9917c04ac936a24fad20b8970",
    ],
)

rpm(
    name = "systemd-libs-0__252-71.el9.s390x",
    sha256 = "ae54dec9d1234ca18234af994eaf78453b012aa96522fb9c369814a1ea2b320a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-libs-252-71.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ae54dec9d1234ca18234af994eaf78453b012aa96522fb9c369814a1ea2b320a",
    ],
)

rpm(
    name = "systemd-libs-0__252-71.el9.x86_64",
    sha256 = "8c64338faba581c7348243b5f451a17acdfd1aacf8494ee7d20634c890bd5d98",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-libs-252-71.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c64338faba581c7348243b5f451a17acdfd1aacf8494ee7d20634c890bd5d98",
    ],
)

rpm(
    name = "systemd-libs-0__257-27.el10.aarch64",
    sha256 = "9e5b16499a0cd0bdc6d90b37f9a22800578be443b5385fdf15c0ea5b4948e582",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-libs-257-27.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9e5b16499a0cd0bdc6d90b37f9a22800578be443b5385fdf15c0ea5b4948e582",
    ],
)

rpm(
    name = "systemd-libs-0__257-27.el10.s390x",
    sha256 = "18024f333e74121bc230c60cae70d5d676c6add60adc6448273e3139d8abd750",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-libs-257-27.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/18024f333e74121bc230c60cae70d5d676c6add60adc6448273e3139d8abd750",
    ],
)

rpm(
    name = "systemd-libs-0__257-27.el10.x86_64",
    sha256 = "c9d0bc2471017e9a8e55bebfa9ed958ac266ff0169243fcaec0633f9c2744139",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-libs-257-27.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c9d0bc2471017e9a8e55bebfa9ed958ac266ff0169243fcaec0633f9c2744139",
    ],
)

rpm(
    name = "systemd-pam-0__252-51.el9.x86_64",
    sha256 = "26014995c59a6d43c7cc0ba55b829cc14513491bc901fe60faf5a10b43c8fb03",
    urls = ["https://storage.googleapis.com/builddeps/26014995c59a6d43c7cc0ba55b829cc14513491bc901fe60faf5a10b43c8fb03"],
)

rpm(
    name = "systemd-pam-0__252-71.el9.aarch64",
    sha256 = "010885ff62b730d57a6a0b2a0ceca256c94d5be962564480463478962635b6cc",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-pam-252-71.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/010885ff62b730d57a6a0b2a0ceca256c94d5be962564480463478962635b6cc",
    ],
)

rpm(
    name = "systemd-pam-0__252-71.el9.s390x",
    sha256 = "c64856c81b9ca765d5937d80cf93afcf6a56f93bdf204c02f3e95dc5ceee2dd7",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-pam-252-71.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/c64856c81b9ca765d5937d80cf93afcf6a56f93bdf204c02f3e95dc5ceee2dd7",
    ],
)

rpm(
    name = "systemd-pam-0__252-71.el9.x86_64",
    sha256 = "787a3e9c31a3fa5825e9046849c91dd2e5ec87a7e55f99143d01fb4e3685409c",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-pam-252-71.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/787a3e9c31a3fa5825e9046849c91dd2e5ec87a7e55f99143d01fb4e3685409c",
    ],
)

rpm(
    name = "systemd-pam-0__257-27.el10.aarch64",
    sha256 = "d7ca26a37cd1791f0faf98f464f1cd0639ff0c881b01dbccfbde7ed4e7a09221",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/systemd-pam-257-27.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d7ca26a37cd1791f0faf98f464f1cd0639ff0c881b01dbccfbde7ed4e7a09221",
    ],
)

rpm(
    name = "systemd-pam-0__257-27.el10.s390x",
    sha256 = "d05c7a6b12521d317161592602c7a20f6ef582e254eb9a8492263f852ef2abe6",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/systemd-pam-257-27.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/d05c7a6b12521d317161592602c7a20f6ef582e254eb9a8492263f852ef2abe6",
    ],
)

rpm(
    name = "systemd-pam-0__257-27.el10.x86_64",
    sha256 = "6989b7394667bd9b957c3df7ec5113d98a28136cf26881c9777ae453f0941884",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/systemd-pam-257-27.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6989b7394667bd9b957c3df7ec5113d98a28136cf26881c9777ae453f0941884",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-51.el9.x86_64",
    sha256 = "afa84ccbac79bb3950cca69bbfa9868429ed3aa464c96f5b2a15405a9c49f56c",
    urls = ["https://storage.googleapis.com/builddeps/afa84ccbac79bb3950cca69bbfa9868429ed3aa464c96f5b2a15405a9c49f56c"],
)

rpm(
    name = "systemd-rpm-macros-0__252-71.el9.aarch64",
    sha256 = "374e3c9291d2f083e71d56df22eb1c75f76d7339777b9231b732df6f0f2560b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/systemd-rpm-macros-252-71.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/374e3c9291d2f083e71d56df22eb1c75f76d7339777b9231b732df6f0f2560b1",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-71.el9.s390x",
    sha256 = "374e3c9291d2f083e71d56df22eb1c75f76d7339777b9231b732df6f0f2560b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/systemd-rpm-macros-252-71.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/374e3c9291d2f083e71d56df22eb1c75f76d7339777b9231b732df6f0f2560b1",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__252-71.el9.x86_64",
    sha256 = "374e3c9291d2f083e71d56df22eb1c75f76d7339777b9231b732df6f0f2560b1",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/systemd-rpm-macros-252-71.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/374e3c9291d2f083e71d56df22eb1c75f76d7339777b9231b732df6f0f2560b1",
    ],
)

rpm(
    name = "tar-2__1.34-11.el9.aarch64",
    sha256 = "c9df1ef5362dca84f7731244d7cf09f70ccaf5ffdae6a45f78be6c0edb168330",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tar-1.34-11.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c9df1ef5362dca84f7731244d7cf09f70ccaf5ffdae6a45f78be6c0edb168330",
    ],
)

rpm(
    name = "tar-2__1.34-11.el9.s390x",
    sha256 = "b309cdde22cd13ac6c89924b0b7e891d900c19a9181a2bb2b9e7c143924a940a",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tar-1.34-11.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/b309cdde22cd13ac6c89924b0b7e891d900c19a9181a2bb2b9e7c143924a940a",
    ],
)

rpm(
    name = "tar-2__1.34-11.el9.x86_64",
    sha256 = "bd851918dd8d5df94f8a88a2e1825125fdc9bc7c6d8e8961f7b50d8299df9906",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tar-1.34-11.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bd851918dd8d5df94f8a88a2e1825125fdc9bc7c6d8e8961f7b50d8299df9906",
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
    name = "tar-2__1.35-11.el10.aarch64",
    sha256 = "9cea19adb21443f7c47330f99987348750c8a389fd42544f0d0b161a255d6b78",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/tar-1.35-11.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9cea19adb21443f7c47330f99987348750c8a389fd42544f0d0b161a255d6b78",
    ],
)

rpm(
    name = "tar-2__1.35-11.el10.s390x",
    sha256 = "69bc4390eb6bfc9b60c3243c764d301947a703128dd8ea53e3ed408e584b5ad1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/tar-1.35-11.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/69bc4390eb6bfc9b60c3243c764d301947a703128dd8ea53e3ed408e584b5ad1",
    ],
)

rpm(
    name = "tar-2__1.35-11.el10.x86_64",
    sha256 = "c7461d8aa0cc1c9e51244356f4270a5384b220557d3131a5c6f146d6837f3e6d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/tar-1.35-11.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c7461d8aa0cc1c9e51244356f4270a5384b220557d3131a5c6f146d6837f3e6d",
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
        "https://storage.googleapis.com/builddeps/aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-12.el10.s390x",
    sha256 = "aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/target-restore-2.1.76-12.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    ],
)

rpm(
    name = "target-restore-0__2.1.76-12.el10.x86_64",
    sha256 = "aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/target-restore-2.1.76-12.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/aca595d2a389cf5be70543dbe4b428efdced6358fe31d59db7d2608bedfdbde5",
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
        "https://storage.googleapis.com/builddeps/687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    ],
)

rpm(
    name = "targetcli-0__2.1.58-5.el10.s390x",
    sha256 = "687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/targetcli-2.1.58-5.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    ],
)

rpm(
    name = "targetcli-0__2.1.58-5.el10.x86_64",
    sha256 = "687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/targetcli-2.1.58-5.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/687abcde3940a6867baf0ed5f204e383a731fd3d8023ca0672969b80f7a83422",
    ],
)

rpm(
    name = "tpm2-tss-0__4.1.3-6.el10.aarch64",
    sha256 = "22bdfcc4af5dd47fa52bad5f8ebca5d80f6caa6e548ad18c7c037899195d4bd3",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/tpm2-tss-4.1.3-6.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/22bdfcc4af5dd47fa52bad5f8ebca5d80f6caa6e548ad18c7c037899195d4bd3",
    ],
)

rpm(
    name = "tpm2-tss-0__4.1.3-6.el10.s390x",
    sha256 = "6c572b1029f932a26e9b4e6f15f2ab3554c13c0ec041ba9ffd4b525514ddee7d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/tpm2-tss-4.1.3-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/6c572b1029f932a26e9b4e6f15f2ab3554c13c0ec041ba9ffd4b525514ddee7d",
    ],
)

rpm(
    name = "tpm2-tss-0__4.1.3-6.el10.x86_64",
    sha256 = "95f4b1e56a2c511de79a18c0dd24a03b22175e8cc3d79c9cbd4eb4fd62207fbc",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/tpm2-tss-4.1.3-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95f4b1e56a2c511de79a18c0dd24a03b22175e8cc3d79c9cbd4eb4fd62207fbc",
    ],
)

rpm(
    name = "tzdata-0__2025a-1.el9.x86_64",
    sha256 = "655945e6a0e95b960a422828bc1cb3bac2232fe9b76590e35ad00069097f087a",
    urls = ["https://storage.googleapis.com/builddeps/655945e6a0e95b960a422828bc1cb3bac2232fe9b76590e35ad00069097f087a"],
)

rpm(
    name = "tzdata-0__2026b-1.el10.aarch64",
    sha256 = "3c44406ccd61907760b1224af8b807a03ddb4613d44ff9504e26d0b797eb91a1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/tzdata-2026b-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c44406ccd61907760b1224af8b807a03ddb4613d44ff9504e26d0b797eb91a1",
    ],
)

rpm(
    name = "tzdata-0__2026b-1.el10.s390x",
    sha256 = "3c44406ccd61907760b1224af8b807a03ddb4613d44ff9504e26d0b797eb91a1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/tzdata-2026b-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c44406ccd61907760b1224af8b807a03ddb4613d44ff9504e26d0b797eb91a1",
    ],
)

rpm(
    name = "tzdata-0__2026b-1.el10.x86_64",
    sha256 = "3c44406ccd61907760b1224af8b807a03ddb4613d44ff9504e26d0b797eb91a1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/tzdata-2026b-1.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3c44406ccd61907760b1224af8b807a03ddb4613d44ff9504e26d0b797eb91a1",
    ],
)

rpm(
    name = "tzdata-0__2026b-1.el9.aarch64",
    sha256 = "579c30aeaede82f71525e9252f22dd5b1ad41e5ecc3bfa13393c4f8d2baaca46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/tzdata-2026b-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/579c30aeaede82f71525e9252f22dd5b1ad41e5ecc3bfa13393c4f8d2baaca46",
    ],
)

rpm(
    name = "tzdata-0__2026b-1.el9.s390x",
    sha256 = "579c30aeaede82f71525e9252f22dd5b1ad41e5ecc3bfa13393c4f8d2baaca46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/tzdata-2026b-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/579c30aeaede82f71525e9252f22dd5b1ad41e5ecc3bfa13393c4f8d2baaca46",
    ],
)

rpm(
    name = "tzdata-0__2026b-1.el9.x86_64",
    sha256 = "579c30aeaede82f71525e9252f22dd5b1ad41e5ecc3bfa13393c4f8d2baaca46",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/tzdata-2026b-1.el9.noarch.rpm",
        "https://storage.googleapis.com/builddeps/579c30aeaede82f71525e9252f22dd5b1ad41e5ecc3bfa13393c4f8d2baaca46",
    ],
)

rpm(
    name = "unbound-libs-0__1.24.2-7.el10.aarch64",
    sha256 = "7ec96614c29a408d5ec4748822db054ca0bb44cb2706e7bf97e349beef9a7190",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/unbound-libs-1.24.2-7.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7ec96614c29a408d5ec4748822db054ca0bb44cb2706e7bf97e349beef9a7190",
    ],
)

rpm(
    name = "unbound-libs-0__1.24.2-7.el10.s390x",
    sha256 = "8cdf1580c4e364b7113c92ac593c200fa2d95d8893b1505753a91d98845acd1d",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/unbound-libs-1.24.2-7.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/8cdf1580c4e364b7113c92ac593c200fa2d95d8893b1505753a91d98845acd1d",
    ],
)

rpm(
    name = "unbound-libs-0__1.24.2-7.el10.x86_64",
    sha256 = "93606094853637132592f4b7d678ce2073ee2c5bd73a2bcb1dce208c629221ae",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/unbound-libs-1.24.2-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/93606094853637132592f4b7d678ce2073ee2c5bd73a2bcb1dce208c629221ae",
    ],
)

rpm(
    name = "unbound-libs-0__1.25.1-1.el9.aarch64",
    sha256 = "efe9405521dc74d9f379eddf4b9a57a3ce63d2c6ccdd32197fed374f19068b83",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/unbound-libs-1.25.1-1.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/efe9405521dc74d9f379eddf4b9a57a3ce63d2c6ccdd32197fed374f19068b83",
    ],
)

rpm(
    name = "unbound-libs-0__1.25.1-1.el9.s390x",
    sha256 = "72450c22c6cafacf6e213e38ff532322ec0568304ca4deb6f46e6131326a2ed9",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/unbound-libs-1.25.1-1.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/72450c22c6cafacf6e213e38ff532322ec0568304ca4deb6f46e6131326a2ed9",
    ],
)

rpm(
    name = "unbound-libs-0__1.25.1-1.el9.x86_64",
    sha256 = "25fee3475a48a0aae6f2f81534b538ad76ef4f9d80373eb1251c9f1f6b4038c2",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/unbound-libs-1.25.1-1.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/25fee3475a48a0aae6f2f81534b538ad76ef4f9d80373eb1251c9f1f6b4038c2",
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
        "https://storage.googleapis.com/builddeps/12a672104464f85388819600c8b2e4eee38cdb67342107485183bc2076b54fe2",
    ],
)

rpm(
    name = "usbredir-0__0.13.0-6.el10.x86_64",
    sha256 = "11551f45b3e60a80530431dfcd5a1d29c5624d34aeaf86dcfd6bcd0a4f87337f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/usbredir-0.13.0-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11551f45b3e60a80530431dfcd5a1d29c5624d34aeaf86dcfd6bcd0a4f87337f",
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
        "https://storage.googleapis.com/builddeps/4c68e72d9cf6b3ae7b001c181998eeff4514058621e7517ebff26f315757c11d",
    ],
)

rpm(
    name = "userspace-rcu-0__0.14.0-7.el10.x86_64",
    sha256 = "2ff9144b446e979b4d014fac8912e7ea9f9dbc2ebbe913c715629bf82aa34082",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/userspace-rcu-0.14.0-7.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ff9144b446e979b4d014fac8912e7ea9f9dbc2ebbe913c715629bf82aa34082",
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
    name = "util-linux-0__2.37.4-25.el9.aarch64",
    sha256 = "619d39f84e40856b19475294d7e50417541261f852d5feeab75028a9a8f2fb20",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/619d39f84e40856b19475294d7e50417541261f852d5feeab75028a9a8f2fb20",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-25.el9.s390x",
    sha256 = "46a49c017dd8aefaa0d2f9353ecde0477fb9acf048e8e5c9d99ebf404775de05",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/46a49c017dd8aefaa0d2f9353ecde0477fb9acf048e8e5c9d99ebf404775de05",
    ],
)

rpm(
    name = "util-linux-0__2.37.4-25.el9.x86_64",
    sha256 = "2d2b2ba4dea25b829031788e6afdc640412a42ac9b9e70a691aad219f744d0ec",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d2b2ba4dea25b829031788e6afdc640412a42ac9b9e70a691aad219f744d0ec",
    ],
)

rpm(
    name = "util-linux-0__2.40.2-20.el10.aarch64",
    sha256 = "09f93feb4068b441c3329806f93679ef7f72ea7bee51c184cc9377d74e3421d4",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/util-linux-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/09f93feb4068b441c3329806f93679ef7f72ea7bee51c184cc9377d74e3421d4",
    ],
)

rpm(
    name = "util-linux-0__2.40.2-20.el10.s390x",
    sha256 = "88271d9f1bb582f9a85a125af2dcbf5c7ba0ed8b7b66a343c89698bb22e516bd",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/util-linux-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/88271d9f1bb582f9a85a125af2dcbf5c7ba0ed8b7b66a343c89698bb22e516bd",
    ],
)

rpm(
    name = "util-linux-0__2.40.2-20.el10.x86_64",
    sha256 = "b20b43e27ecb494e231035b3a7e3b841efc4935491f3cad76e2695ea19215851",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/util-linux-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b20b43e27ecb494e231035b3a7e3b841efc4935491f3cad76e2695ea19215851",
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
    name = "util-linux-core-0__2.37.4-25.el9.aarch64",
    sha256 = "a31732e9e6c968665ff53330435674fdaa12f9812b309bda9babb29e0d2ca62d",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.37.4-25.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a31732e9e6c968665ff53330435674fdaa12f9812b309bda9babb29e0d2ca62d",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-25.el9.s390x",
    sha256 = "a9c0f4b1c76cc105f42d9763d7a7df522e76f3668086a9cbf2b8318a4a4688e5",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/util-linux-core-2.37.4-25.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a9c0f4b1c76cc105f42d9763d7a7df522e76f3668086a9cbf2b8318a4a4688e5",
    ],
)

rpm(
    name = "util-linux-core-0__2.37.4-25.el9.x86_64",
    sha256 = "15c9e658afed9d50ce20908fd4080cd12042f4bf508f67b2ecbc889ae41c7414",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.37.4-25.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/15c9e658afed9d50ce20908fd4080cd12042f4bf508f67b2ecbc889ae41c7414",
    ],
)

rpm(
    name = "util-linux-core-0__2.40.2-20.el10.aarch64",
    sha256 = "b381d9e796b2c8a4b56a14c9da17b4a9824661ff1e3f12d621c2dff91af56311",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/util-linux-core-2.40.2-20.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b381d9e796b2c8a4b56a14c9da17b4a9824661ff1e3f12d621c2dff91af56311",
    ],
)

rpm(
    name = "util-linux-core-0__2.40.2-20.el10.s390x",
    sha256 = "52532a73135628f2b8d2415865bb3d20d77de1f5a971ac8e8c0c96656bc5ea59",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/util-linux-core-2.40.2-20.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/52532a73135628f2b8d2415865bb3d20d77de1f5a971ac8e8c0c96656bc5ea59",
    ],
)

rpm(
    name = "util-linux-core-0__2.40.2-20.el10.x86_64",
    sha256 = "2a4a0d6756845cf3d719503615e7f3e7d87513f20fcba4c3c0a45f269fc122ee",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/util-linux-core-2.40.2-20.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2a4a0d6756845cf3d719503615e7f3e7d87513f20fcba4c3c0a45f269fc122ee",
    ],
)

rpm(
    name = "vim-data-2__9.1.083-13.el10.aarch64",
    sha256 = "142c899ec40a749f836f3c0f6167df3817b03a7aed59fb47d4ecfc53481f1014",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/vim-data-9.1.083-13.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/142c899ec40a749f836f3c0f6167df3817b03a7aed59fb47d4ecfc53481f1014",
    ],
)

rpm(
    name = "vim-data-2__9.1.083-13.el10.s390x",
    sha256 = "142c899ec40a749f836f3c0f6167df3817b03a7aed59fb47d4ecfc53481f1014",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/vim-data-9.1.083-13.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/142c899ec40a749f836f3c0f6167df3817b03a7aed59fb47d4ecfc53481f1014",
    ],
)

rpm(
    name = "vim-data-2__9.1.083-13.el10.x86_64",
    sha256 = "142c899ec40a749f836f3c0f6167df3817b03a7aed59fb47d4ecfc53481f1014",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/vim-data-9.1.083-13.el10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/142c899ec40a749f836f3c0f6167df3817b03a7aed59fb47d4ecfc53481f1014",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-21.el9.x86_64",
    sha256 = "1b15304790e4b2e7d4ff378b7bf0363b6ecb1c852fc42f984267296538de0c16",
    urls = ["https://storage.googleapis.com/builddeps/1b15304790e4b2e7d4ff378b7bf0363b6ecb1c852fc42f984267296538de0c16"],
)

rpm(
    name = "vim-minimal-2__8.2.2637-31.el9.aarch64",
    sha256 = "de52f15dc69a763d8264f2416c84cd88fbf4944a32fe2b35b62e7136ebc22ae6",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.2.2637-31.el9.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de52f15dc69a763d8264f2416c84cd88fbf4944a32fe2b35b62e7136ebc22ae6",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-31.el9.s390x",
    sha256 = "5c12af0ee160414916ef342b229287696da0eb296469f5c5810a356a220af535",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/s390x/os/Packages/vim-minimal-8.2.2637-31.el9.s390x.rpm",
        "https://storage.googleapis.com/builddeps/5c12af0ee160414916ef342b229287696da0eb296469f5c5810a356a220af535",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2637-31.el9.x86_64",
    sha256 = "b7311b04f63c5c4c8cc055d3f0c2dc9c2aa7bb569a35f5aedb997c3dbd8c9f28",
    urls = [
        "http://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.2.2637-31.el9.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7311b04f63c5c4c8cc055d3f0c2dc9c2aa7bb569a35f5aedb997c3dbd8c9f28",
    ],
)

rpm(
    name = "vim-minimal-2__9.1.083-13.el10.aarch64",
    sha256 = "26778904efdca80bf5fac5c419a2fe74ea3ed1918557f09c9c9ed342404e0f07",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/aarch64/os/Packages/vim-minimal-9.1.083-13.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/26778904efdca80bf5fac5c419a2fe74ea3ed1918557f09c9c9ed342404e0f07",
    ],
)

rpm(
    name = "vim-minimal-2__9.1.083-13.el10.s390x",
    sha256 = "a18d8dba3a5a9b41e57aae37bdc7c0c8dc3ce8cc32aa165b4e96396e9a62da4f",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/vim-minimal-9.1.083-13.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/a18d8dba3a5a9b41e57aae37bdc7c0c8dc3ce8cc32aa165b4e96396e9a62da4f",
    ],
)

rpm(
    name = "vim-minimal-2__9.1.083-13.el10.x86_64",
    sha256 = "680a603d8aff4cf25f371b884e22d31d5f811100855493065438ab3b278fb92b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/vim-minimal-9.1.083-13.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/680a603d8aff4cf25f371b884e22d31d5f811100855493065438ab3b278fb92b",
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
    name = "virtiofsd-0__1.13.3-2.el10.aarch64",
    sha256 = "8b007c2675c76b7149a40f70b5c16e19917bf3d7d9b82d8b1d184e125820630b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/aarch64/os/Packages/virtiofsd-1.13.3-2.el10.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8b007c2675c76b7149a40f70b5c16e19917bf3d7d9b82d8b1d184e125820630b",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.3-2.el10.s390x",
    sha256 = "4f105c2954237826d8f434a287d71c1c0ab6ea630856e4f642469a0c4621fcc1",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/virtiofsd-1.13.3-2.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/4f105c2954237826d8f434a287d71c1c0ab6ea630856e4f642469a0c4621fcc1",
    ],
)

rpm(
    name = "virtiofsd-0__1.13.3-2.el10.x86_64",
    sha256 = "600a02a49b5f023f633595f46752d4398342357178cd6331701a035c409b9427",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/virtiofsd-1.13.3-2.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/600a02a49b5f023f633595f46752d4398342357178cd6331701a035c409b9427",
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
        "https://storage.googleapis.com/builddeps/369a215b68f7dd87ce2b0c7be20425b63a19ba8b18a74775b474a717524388fe",
    ],
)

rpm(
    name = "which-0__2.21-44.el10.s390x",
    sha256 = "93c0edc58db280e4bcc7a7568fc9eb935a27666fea4659c0de6aa93e1017d0d7",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/which-2.21-44.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/93c0edc58db280e4bcc7a7568fc9eb935a27666fea4659c0de6aa93e1017d0d7",
    ],
)

rpm(
    name = "which-0__2.21-44.el10.x86_64",
    sha256 = "8817b5d8ce0a8a07e38daa93a72d0cca53934e1631322b990d389ccb34376e1c",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/which-2.21-44.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8817b5d8ce0a8a07e38daa93a72d0cca53934e1631322b990d389ccb34376e1c",
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
        "https://storage.googleapis.com/builddeps/8e152db322abfb8b173f703a0af4be1ef294abeb3dd78da974f800b074a06530",
    ],
)

rpm(
    name = "xorriso-0__1.5.6-6.el10.s390x",
    sha256 = "ab7d3d43d22e8a4920453c73de0e60dbc3df69ab28042cad271df8a51cfa0b4b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/s390x/os/Packages/xorriso-1.5.6-6.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/ab7d3d43d22e8a4920453c73de0e60dbc3df69ab28042cad271df8a51cfa0b4b",
    ],
)

rpm(
    name = "xorriso-0__1.5.6-6.el10.x86_64",
    sha256 = "2077b91e476836bec242f0fbf4a83384bfc785e9531bedb81ee825936105b017",
    urls = [
        "http://mirror.stream.centos.org/10-stream/AppStream/x86_64/os/Packages/xorriso-1.5.6-6.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2077b91e476836bec242f0fbf4a83384bfc785e9531bedb81ee825936105b017",
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
        "https://storage.googleapis.com/builddeps/7bf62608392ae9fd5dd59add39723086f5a052c2064e0498c1641c572cd46460",
    ],
)

rpm(
    name = "xz-1__5.6.2-4.el10.s390x",
    sha256 = "37e1052ce13b55ef1f4e33a8997728963f51c76223d165e2534f0cd6e8f9ba59",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/xz-5.6.2-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/37e1052ce13b55ef1f4e33a8997728963f51c76223d165e2534f0cd6e8f9ba59",
    ],
)

rpm(
    name = "xz-1__5.6.2-4.el10.x86_64",
    sha256 = "dc71c8e5b558c9f9fdea14a7d38819fc12ad8bdcb6834989188b225ca191eded",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/xz-5.6.2-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dc71c8e5b558c9f9fdea14a7d38819fc12ad8bdcb6834989188b225ca191eded",
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
        "https://storage.googleapis.com/builddeps/fcf207b0e6fe443fafe62fa43fc44ce16c8c118dd5e69491b3ad4b9eda72cc61",
    ],
)

rpm(
    name = "xz-libs-1__5.6.2-4.el10.s390x",
    sha256 = "7edd13c2a8dfb66b1e8c8a0d1d9259a1ff5cfb4891d568cc8f990664a02f7e32",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/xz-libs-5.6.2-4.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/7edd13c2a8dfb66b1e8c8a0d1d9259a1ff5cfb4891d568cc8f990664a02f7e32",
    ],
)

rpm(
    name = "xz-libs-1__5.6.2-4.el10.x86_64",
    sha256 = "21733e8b6bf26b20633618adb074706972479080527f3f7a51246e83b3d4342e",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/xz-libs-5.6.2-4.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21733e8b6bf26b20633618adb074706972479080527f3f7a51246e83b3d4342e",
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
        "https://storage.googleapis.com/builddeps/a7870bf73b68086ae1fdd3e2fb6191bf79dff1ab5ae16b907efbb0befe590dca",
    ],
)

rpm(
    name = "zlib-ng-compat-0__2.2.3-3.el10.s390x",
    sha256 = "89c8decb9febd474ba2f3fbb38c37577dd7098b349a9e766267723fb94f25962",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/zlib-ng-compat-2.2.3-3.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/89c8decb9febd474ba2f3fbb38c37577dd7098b349a9e766267723fb94f25962",
    ],
)

rpm(
    name = "zlib-ng-compat-0__2.2.3-3.el10.x86_64",
    sha256 = "8fe3c2d5203810828fa3e4a5d84ae53172ffd27f4f0eec9d192b42b187795c09",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/zlib-ng-compat-2.2.3-3.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8fe3c2d5203810828fa3e4a5d84ae53172ffd27f4f0eec9d192b42b187795c09",
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
        "https://storage.googleapis.com/builddeps/b45bf236f2a5a034295eb933b3c056b302785fa122ecba44b98fec6d2d8b39a2",
    ],
)

rpm(
    name = "zstd-0__1.5.5-9.el10.s390x",
    sha256 = "def19135b3b6f01e46d9ee17e69ae1227e9addaf1fcd231596836000917fe393",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/s390x/os/Packages/zstd-1.5.5-9.el10.s390x.rpm",
        "https://storage.googleapis.com/builddeps/def19135b3b6f01e46d9ee17e69ae1227e9addaf1fcd231596836000917fe393",
    ],
)

rpm(
    name = "zstd-0__1.5.5-9.el10.x86_64",
    sha256 = "4ef415b98ddbe28f836b86699f4cec6002817ea20fb47499d3c6bb0814db6d4b",
    urls = [
        "http://mirror.stream.centos.org/10-stream/BaseOS/x86_64/os/Packages/zstd-1.5.5-9.el10.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4ef415b98ddbe28f836b86699f4cec6002817ea20fb47499d3c6bb0814db6d4b",
    ],
)
