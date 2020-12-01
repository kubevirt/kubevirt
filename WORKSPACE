load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# Additional bazel rules

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "08369b54a7cbe9348eea474e36c9bbb19d47101e8860cec75cbf1ccd4f749281",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.24.0/rules_go-v0.24.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.24.0/rules_go-v0.24.0.tar.gz",
        "https://storage.googleapis.com/builddeps/08369b54a7cbe9348eea474e36c9bbb19d47101e8860cec75cbf1ccd4f749281",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "d4113967ab451dd4d2d767c3ca5f927fec4b30f3b2c6f8135a2033b9c05a5687",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.0/bazel-gazelle-v0.22.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.0/bazel-gazelle-v0.22.0.tar.gz",
        "https://storage.googleapis.com/builddeps/d4113967ab451dd4d2d767c3ca5f927fec4b30f3b2c6f8135a2033b9c05a5687",
    ],
)

http_archive(
    name = "com_google_protobuf",
    sha256 = "cd218dc003eacc167e51e3ce856f6c2e607857225ef86b938d95650fcbb2f8e4",
    strip_prefix = "protobuf-6d4e7fd7966c989e38024a8ea693db83758944f1",
    # version 3.10.0
    urls = [
        "https://github.com/google/protobuf/archive/6d4e7fd7966c989e38024a8ea693db83758944f1.zip",
        "https://storage.googleapis.com/builddeps/cd218dc003eacc167e51e3ce856f6c2e607857225ef86b938d95650fcbb2f8e4",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "4521794f0fba2e20f3bf15846ab5e01d5332e587e9ce81629c7f96c793bb7036",
    strip_prefix = "rules_docker-0.14.4",
    urls = [
        "https://github.com/bazelbuild/rules_docker/releases/download/v0.14.4/rules_docker-v0.14.4.tar.gz",
        "https://storage.googleapis.com/builddeps/4521794f0fba2e20f3bf15846ab5e01d5332e587e9ce81629c7f96c793bb7036",
    ],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "29813b426161f1f09f940e62224f4e54e5737686f2bd22146807d933fa1fa768",
    strip_prefix = "bazel-tools-82b58b374e3b1746d6d6a58a37f7ada4400a13ce",
    urls = [
        "https://github.com/atlassian/bazel-tools/archive/82b58b374e3b1746d6d6a58a37f7ada4400a13ce.zip",
    ],
)

# Libvirt dependencies
http_file(
    name = "libvirt_libs",
    sha256 = "3a0a3d88c6cb90008fbe49fe05e7025056fb9fa3a887c4a78f79e63f8745c845",
    urls = [
        "https://download-ib01.fedoraproject.org/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvirt-libs-6.1.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3a0a3d88c6cb90008fbe49fe05e7025056fb9fa3a887c4a78f79e63f8745c845",
    ],
)

http_file(
    name = "libvirt_devel",
    sha256 = "2ebb715341b57a74759aff415e0ff53df528c49abaa7ba5b794b4047461fa8d6",
    urls = [
        "https://download-ib01.fedoraproject.org/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvirt-devel-6.1.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ebb715341b57a74759aff415e0ff53df528c49abaa7ba5b794b4047461fa8d6",
    ],
)

# Disk images
http_file(
    name = "alpine_image",
    sha256 = "5a4b2588afd32e7024dd61d9558b77b03a4f3189cb4c9fc05e9e944fb780acdd",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso",
        "https://storage.googleapis.com/builddeps/5a4b2588afd32e7024dd61d9558b77b03a4f3189cb4c9fc05e9e944fb780acdd",
    ],
)

http_file(
    name = "alpine_image_ppc64le",
    sha256 = "4b1d35e7cc9f5e1f5774ed9ea47c85893f20dc8713625e1f8fa7fbddca243a15",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/ppc64le/alpine-vanilla-3.7.0-ppc64le.iso",
        "https://storage.googleapis.com/builddeps/4b1d35e7cc9f5e1f5774ed9ea47c85893f20dc8713625e1f8fa7fbddca243a15",
    ],
)

http_file(
    name = "cirros_image",
    sha256 = "a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    urls = [
        "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img",
        "https://storage.googleapis.com/builddeps/a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    ],
)

http_file(
    name = "cirros_image_ppc64le",
    sha256 = "175063e409f4019acb760478eb1a94819628a1bec9376d26d3aa333449fe061d",
    urls = [
        "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-ppc64le-disk.img",
        "https://storage.googleapis.com/builddeps/175063e409f4019acb760478eb1a94819628a1bec9376d26d3aa333449fe061d",
    ],
)

http_file(
    name = "fedora_image",
    sha256 = "423a4ce32fa32c50c11e3d3ff392db97a762533b81bef9d00599de518a7469c8",
    urls = [
        "https://download.fedoraproject.org/pub/fedora/linux/releases/32/Cloud/x86_64/images/Fedora-Cloud-Base-32-1.6.x86_64.qcow2",
        "https://storage.googleapis.com/builddeps/423a4ce32fa32c50c11e3d3ff392db97a762533b81bef9d00599de518a7469c8",
    ],
)

http_file(
    name = "fedora_image_ppc64le",
    sha256 = "dd989a078d641713c55720ba3e4320b204ade6954e2bfe4570c8058dc36e2e5d",
    urls = [
        "https://kojipkgs.fedoraproject.org/compose/32/Fedora-32-20200422.0/compose/Cloud/ppc64le/images/Fedora-Cloud-Base-32-1.6.ppc64le.qcow2",
        "https://storage.googleapis.com/builddeps/dd989a078d641713c55720ba3e4320b204ade6954e2bfe4570c8058dc36e2e5d",
    ],
)

http_file(
    name = "microlivecd_image",
    sha256 = "ae449ae8c0f73b1a7e2c394bc5385e7ab01d8fc000f5b074bc8b2aaabf931eac",
    urls = [
        "https://github.com/jean-edouard/microlivecd/releases/download/0.1/microlivecd_amd64.iso",
        "https://storage.googleapis.com/builddeps/ae449ae8c0f73b1a7e2c394bc5385e7ab01d8fc000f5b074bc8b2aaabf931eac",
    ],
)

http_file(
    name = "microlivecd_image_ppc64el",
    sha256 = "eae431d68b9dc5fab422f4b90d4204cbc28c39518780c4822970a4bef42f7c7f",
    urls = [
        "https://github.com/jean-edouard/microlivecd/releases/download/0.1/microlivecd_ppc64el.iso",
        "https://storage.googleapis.com/builddeps/eae431d68b9dc5fab422f4b90d4204cbc28c39518780c4822970a4bef42f7c7f",
    ],
)

http_file(
    name = "virtio_win_image",
    sha256 = "7bf7f53e30c69a360f89abb3d2cc19cc978f533766b1b2270c2d8344edf9b3ef",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.171-1/virtio-win-0.1.171.iso",
        "https://storage.googleapis.com/builddeps/7bf7f53e30c69a360f89abb3d2cc19cc978f533766b1b2270c2d8344edf9b3ef",
    ],
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.13.14",
    nogo = "@//:nogo_vet",
)

load("@com_github_atlassian_bazel_tools//goimports:deps.bzl", "goimports_dependencies")

goimports_dependencies()

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

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

load("@io_bazel_rules_docker//repositories:pip_repositories.bzl", "pip_deps")

pip_deps()

http_file(
    name = "go_puller_linux_ppc64le",
    executable = True,
    sha256 = "540f4d7b2a3d627d7c3190f11c4fab5f8aad48bd42a9dffb037786e26270b6bd",
    urls = [
        "https://storage.googleapis.com/builddeps/540f4d7b2a3d627d7c3190f11c4fab5f8aad48bd42a9dffb037786e26270b6bd",
    ],
)

http_file(
    name = "go_pusher_linux_ppc64le",
    executable = True,
    sha256 = "961e5a11677ab5ebf9d7ada76864ca271abcc795a6808ac2e7e98b3458b4e435",
    urls = [
        "https://storage.googleapis.com/builddeps/961e5a11677ab5ebf9d7ada76864ca271abcc795a6808ac2e7e98b3458b4e435",
    ],
)

# Pull base image fedora31
container_pull(
    name = "fedora",
    digest = "sha256:5e2b864cfe165fa7da6606b29a9e60549eb7cc9ae7fb574614110d1494b0f0c2",
    registry = "index.docker.io",
    repository = "library/fedora",
    tag = "31",
)

container_pull(
    name = "fedora_ppc64le",
    digest = "sha256:50ab81a4619f7e94793aba65f3a40505bdfb9b59dcf6ae6deb8f974723e966d9",
    puller_linux = "@go_puller_linux_ppc64le//file:downloaded",
    registry = "index.docker.io",
    repository = "library/fedora",
    tag = "31",
)

# Pull fedora 32 customize container-disk
container_pull(
    name = "fedora_sriov_lane",
    digest = "sha256:2d332d28863d0e415d58e335e836bd4f8a8c714e7a9d1f8f87418ef3db7c0afb",
    registry = "index.docker.io",
    repository = "kubevirt/fedora-sriov-testing",
    #tag = "32",
)

# Pull base image libvirt
container_pull(
    name = "libvirt",
    digest = "sha256:a95f0d6e15796c4a7dc3e5358505691482eecd3f3286f3914bc744a5ce250cbd",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
    #tag = "20201125-c4405e2",
)

# TODO: Update this once we have PPC builds of the base image available
container_pull(
    name = "libvirt_ppc64le",
    digest = "sha256:NOT_AVAILABLE",  # Make sure we don't use outdated image by mistake
    puller_linux = "@go_puller_linux_ppc64le//file:downloaded",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
)

# Pull kubevirt-testing image
container_pull(
    name = "kubevirt-testing",
    digest = "sha256:eb86f7388217bb18611c8c4e6169af3463c2a18f420314eb4d742b3d3669b16f",
    registry = "index.docker.io",
    repository = "kubevirtci/kubevirt-testing",
    #tag = "28",
)

container_pull(
    name = "kubevirt-testing_ppc64le",
    digest = "sha256:eb86f7388217bb18611c8c4e6169af3463c2a18f420314eb4d742b3d3669b16f",
    #tag = "28",
    puller_linux = "@go_puller_linux_ppc64le//file:downloaded",
    registry = "index.docker.io",
    repository = "kubevirtci/kubevirt-testing",
)

# Pull nfs-server image
container_pull(
    name = "nfs-server",
    digest = "sha256:8c1fa882dddb2885c4152e9ce632c466f4b8dce29339455e9b6bfe71f0a3d3ef",
    registry = "index.docker.io",
    repository = "kubevirtci/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
)

container_pull(
    name = "nfs-server_ppc64le",
    digest = "sha256:8c1fa882dddb2885c4152e9ce632c466f4b8dce29339455e9b6bfe71f0a3d3ef",
    puller_linux = "@go_puller_linux_ppc64le//file:downloaded",
    registry = "index.docker.io",
    repository = "kubevirtci/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
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

# Get container-disk-v1alpha RPM's
http_file(
    name = "qemu-img",
    sha256 = "669250ad47aad5939cf4d1b88036fd95a94845d8e0bbdb05e933f3d2fe262fea",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/q/qemu-img-4.1.0-2.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/669250ad47aad5939cf4d1b88036fd95a94845d8e0bbdb05e933f3d2fe262fea",
    ],
)

http_file(
    name = "qemu-img_ppc64le",
    sha256 = "c6629cb5b44a7adbedf5f84324933d02f6ecfaf931b90d034354cdc9516e7adb",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/q/qemu-img-4.1.0-2.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/c6629cb5b44a7adbedf5f84324933d02f6ecfaf931b90d034354cdc9516e7adb",
    ],
)

# qemu-img library dependencies
http_file(
    name = "nettle",
    sha256 = "429d5b9a845285710b7baad1cdc96be74addbf878011642cfc7c14b5636e9bcc",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/n/nettle-3.5.1-3.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/429d5b9a845285710b7baad1cdc96be74addbf878011642cfc7c14b5636e9bcc",
    ],
)

http_file(
    name = "nettle_ppc64le",
    sha256 = "ae530f297a7159653ee26eacec17052b8a3e628b24ff256447ee4383c963be50",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/n/nettle-3.5.1-3.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ae530f297a7159653ee26eacec17052b8a3e628b24ff256447ee4383c963be50",
    ],
)

http_file(
    name = "glibc",
    sha256 = "33e0ad9b92d40c4e09d6407df1c8549b3d4d3d64fdd482439e66d12af6004f13",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/g/glibc-2.30-5.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/33e0ad9b92d40c4e09d6407df1c8549b3d4d3d64fdd482439e66d12af6004f13",
    ],
)

http_file(
    name = "glibc_ppc64le",
    sha256 = "7761a68e16aafe728b8a45187903010001b5cb086f2f5fe929703f0c7fe8a43b",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/g/glibc-2.30-5.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/7761a68e16aafe728b8a45187903010001b5cb086f2f5fe929703f0c7fe8a43b",
    ],
)

http_file(
    name = "bzip2",
    sha256 = "d334fe6e150349148b9cb77e32523029311ce8cb10d222d11c951b66637bbd3a",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/b/bzip2-1.0.8-1.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d334fe6e150349148b9cb77e32523029311ce8cb10d222d11c951b66637bbd3a",
    ],
)

http_file(
    name = "bzip2_ppc64le",
    sha256 = "a3c696a53507cb5d6e45c1d84b874ed030de8416e4da86e8b6eb21ea5d0f0d81",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/b/bzip2-1.0.8-1.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a3c696a53507cb5d6e45c1d84b874ed030de8416e4da86e8b6eb21ea5d0f0d81",
    ],
)

http_file(
    name = "capstone",
    sha256 = "4d2671bc78b11650e8ccf75926e34295c641433759eab8f8932b8403bfa15319",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/c/capstone-4.0.1-4.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d2671bc78b11650e8ccf75926e34295c641433759eab8f8932b8403bfa15319",
    ],
)

http_file(
    name = "capstone_ppc64le",
    sha256 = "d835db14b1dda9601cd208edeed76cffd3b14de37330684e9ec751e67b0827cf",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/c/capstone-4.0.1-11.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d835db14b1dda9601cd208edeed76cffd3b14de37330684e9ec751e67b0827cf",
    ],
)

http_file(
    name = "libaio",
    sha256 = "ee6596a5010c2b4a038861828ecca240aa03c592dacd83c3a70d44cb8ee50408",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/l/libaio-0.3.111-6.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ee6596a5010c2b4a038861828ecca240aa03c592dacd83c3a70d44cb8ee50408",
    ],
)

http_file(
    name = "libaio_ppc64le",
    sha256 = "cf0889003cdc23fa251553e96f294b253416641ddd638533c1da328deb82ec9e",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/l/libaio-0.3.111-6.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cf0889003cdc23fa251553e96f294b253416641ddd638533c1da328deb82ec9e",
    ],
)

http_file(
    name = "libstdc",
    sha256 = "2a89e768507364310d03fe54362b30fb90c6bb7d1b558ab52f74a596548c234f",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/l/libstdc++-9.2.1-1.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2a89e768507364310d03fe54362b30fb90c6bb7d1b558ab52f74a596548c234f",
    ],
)

http_file(
    name = "libstdc_ppc64le",
    sha256 = "9cda89750c2d01be49024fd0c57253cb2bbe5682fbad37e366c11c2fa802c68b",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/l/libstdc++-9.2.1-1.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/9cda89750c2d01be49024fd0c57253cb2bbe5682fbad37e366c11c2fa802c68b",
    ],
)

http_file(
    name = "qemu-guest-agent",
    sha256 = "41edf2ba208309eb1cde80d5d227c4fdf43906ef47ed76aa37a51c344dfed3ee",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/q/qemu-guest-agent-4.1.0-2.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/41edf2ba208309eb1cde80d5d227c4fdf43906ef47ed76aa37a51c344dfed3ee",
    ],
)

http_file(
    name = "qemu-guest-agent_ppc64le",
    sha256 = "cb352f4f5de837d6ab954f34c9b65f590c76547d5f59ba4c1f63343fce8d13c8",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/q/qemu-guest-agent-4.1.0-2.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cb352f4f5de837d6ab954f34c9b65f590c76547d5f59ba4c1f63343fce8d13c8",
    ],
)

# qemu-ga links against libpixman-1.so
http_file(
    name = "pixman-1",
    sha256 = "913aa9517093ce768a0fab78c9ef4012efdf8364af52e8c8b27cd043517616ba",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/p/pixman-0.38.4-1.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/913aa9517093ce768a0fab78c9ef4012efdf8364af52e8c8b27cd043517616ba",
    ],
)

http_file(
    name = "pixman-1_ppc64le",
    sha256 = "f29e86dcaeadaeb5ecb12e5a4f2d447e711f4bf1513b8923a63e69fa1d4f0f66",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/p/pixman-0.38.4-1.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f29e86dcaeadaeb5ecb12e5a4f2d447e711f4bf1513b8923a63e69fa1d4f0f66",
    ],
)

http_file(
    name = "stress",
    sha256 = "fe1037e4dca31eabf013e48a0cbc08a10bafa7fb77e3adcdd0ce376fafc21218",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/s/stress-1.0.4-23.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe1037e4dca31eabf013e48a0cbc08a10bafa7fb77e3adcdd0ce376fafc21218",
    ],
)

http_file(
    name = "stress_ppc64le",
    sha256 = "07b0cd6cce8ef6fc28f2187e128872e4a64054ddc24563c04c103abedb7c3ebd",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/s/stress-1.0.4-23.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/07b0cd6cce8ef6fc28f2187e128872e4a64054ddc24563c04c103abedb7c3ebd",
    ],
)

http_file(
    name = "e2fsprogs",
    sha256 = "71c02de0e50e07999d0f4f40bce06ca4904e0ab786220bd7ffebc4a60a4d3cd7",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.3-1.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/71c02de0e50e07999d0f4f40bce06ca4904e0ab786220bd7ffebc4a60a4d3cd7",
    ],
)

http_file(
    name = "e2fsprogs_ppc64le",
    sha256 = "9d47c2d29dd8cff1c2b646c068fbf080b4e974b6140ca184f1799e819525694c",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora-secondary/releases/31/Everything/ppc64le/os/Packages/e/e2fsprogs-1.45.3-1.fc31.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/9d47c2d29dd8cff1c2b646c068fbf080b4e974b6140ca184f1799e819525694c",
    ],
)

http_file(
    name = "dmidecode",
    sha256 = "254a243b2d6b4246d675742f4467665b6d1c639af64dae6ee60bd6c01f2f6084",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/d/dmidecode-3.2-3.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/254a243b2d6b4246d675742f4467665b6d1c639af64dae6ee60bd6c01f2f6084",
    ],
)

http_file(
    name = "which",
    sha256 = "ed94cc657a0cca686fcea9274f24053e13dc17f770e269cab0b151f18212ddaa",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/w/which-2.21-15.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed94cc657a0cca686fcea9274f24053e13dc17f770e269cab0b151f18212ddaa",
    ],
)

http_file(
    name = "virt-what",
    sha256 = "4c3b6e527de5c72ba44c7e10ec7bceba1a7922aefbd3ea34f99e378885729928",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/31/Everything/x86_64/os/Packages/v/virt-what-1.19-3.fc31.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c3b6e527de5c72ba44c7e10ec7bceba1a7922aefbd3ea34f99e378885729928",
    ],
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
