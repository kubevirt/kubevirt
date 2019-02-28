load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)

# Additional bazel rules
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7be7dc01f1e0afdba6c8eb2b43d2fa01c743be1b9273ab1eaf6c233df078d705",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.16.5/rules_go-0.16.5.tar.gz",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    strip_prefix = "buildtools-db073457c5a56d810e46efc18bb93a4fd7aa7b5e",
    # version 0.20.0
    url = "https://github.com/bazelbuild/buildtools/archive/db073457c5a56d810e46efc18bb93a4fd7aa7b5e.zip",
)

load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    "git_repository",
)

git_repository(
    name = "io_bazel_rules_docker",
    remote = "https://github.com/bazelbuild/rules_docker.git",
    tag = "v0.5.1",
)

# Libvirt dependencies
http_file(
    name = "libvirt_libs",
    sha256 = "95d317fd645fb52745d642578263a9bcb0d796beadf00aeadebc0d692f5529ba",
    urls = [
        "https://libvirt.org/sources/libvirt-libs-5.0.0-1.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "libvirt_devel",
    sha256 = "6573a047d777ed00f6858c2e75c780053b1f898ae1c3f7415e991c94c5ccdd70",
    urls = [
        "https://libvirt.org/sources/libvirt-devel-5.0.0-1.fc28.x86_64.rpm",
    ],
)

# Disk images
http_file(
    name = "alpine_image",
    sha256 = "5a4b2588afd32e7024dd61d9558b77b03a4f3189cb4c9fc05e9e944fb780acdd",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.7/releases/x86_64/alpine-virt-3.7.0-x86_64.iso",
    ],
)

http_file(
    name = "cirros_image",
    sha256 = "a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    urls = [
        "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img",
    ],
)

http_file(
    name = "fedora_image",
    sha256 = "a30549d620bf6bf41d30a9a58626e59dfa70bb011fd7d50f6c4511ad2e479a39",
    urls = [
        "https://download.fedoraproject.org/pub/fedora/linux/releases/29/Cloud/x86_64/images/Fedora-Cloud-Base-29-1.2.x86_64.qcow2",
    ],
)

http_file(
    name = "virtio_win_image",
    sha256 = "7a770f738d5853378204648a583d559753920f4f95aa54b5d4c6ff16ab37eaea",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/latest-virtio/virtio-win.iso",
    ],
)

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains()

load(
    "@bazel_gazelle//:deps.bzl",
    "gazelle_dependencies",
    "go_repository",
)

gazelle_dependencies()

load("@com_github_bazelbuild_buildtools//buildifier:deps.bzl", "buildifier_dependencies")

buildifier_dependencies()

# Winrmcli dependencies
go_repository(
    name = "com_github_masterzen_winrmcli",
    commit = "c85a68ee8b6e3ac95af2a5fd62d2f41c9e9c5f32",
    importpath = "github.com/masterzen/winrm-cli",
)

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_image",
    "container_pull",
    container_repositories = "repositories",
)

container_repositories()

# Pull base image fedora28
container_pull(
    name = "fedora",
    digest = "sha256:57d86e03971841e79585379f8346289ceb5a3e8ee06933fbd5064b4f004659d1",
    registry = "index.docker.io",
    repository = "library/fedora",
    #tag = "28",
)

# Pull base image libvirt
container_pull(
    name = "libvirt",
    digest = "sha256:081f113a73748775e5f37d8fb877a574f595df1551e39e48ebbe8e8afd501d3b",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
    #tag = "5.0.0",
)

# Pull kubevirt-testing image
container_pull(
    name = "kubevirt-testing",
    digest = "sha256:2e43d16abaaea55672b125515e89ae69d8c6424fc2c110273aaf7db047f0b82f",
    registry = "index.docker.io",
    repository = "kubevirtci/kubevirt-testing",
    #tag = "28",
)

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

http_archive(
    name = "io_bazel_rules_container_rpm",
    sha256 = "7ace5bc7aa10697196d3c48ef5d73b0fc52b6e27c0bad3ce10751f66c5cc383c",
    strip_prefix = "rules_container_rpm-0.0.3",
    urls = ["https://github.com/rmohr/rules_container_rpm/archive/v0.0.3.tar.gz"],
)

# Get container-disk-v1alpha RPM's
http_file(
    name = "qemu-img",
    sha256 = "611d6ff8e08eca0d543486f3e8b437f7ddbc362e4d281c7422677d712d017d61",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-img-2.11.2-4.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "bzip2",
    sha256 = "5248b7b85e58319c6c16e9b545dc04f1b25ba236e0507a7c923d74b00fe8741c",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/28/Everything/x86_64/os/Packages/b/bzip2-1.0.6-26.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "capstone",
    sha256 = "847ebb3a852f82cfe932230d1700cb8b90f602acbe9f9dcf5de7129a1d222c6b",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/c/capstone-3.0.5-1.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "libaio",
    sha256 = "da731218ec1a8e8f690c880d7a45d09722ea01090caba0ae25d9202e0d521404",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/28/Everything/x86_64/os/Packages/l/libaio-0.3.110-11.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "libstdc",
    sha256 = "91e909dda77de31d5653e14e342bb4f0c0e44a82e688d343a192b3ed71479708",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/l/libstdc++-8.2.1-6.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "qemu-guest-agent",
    sha256 = "1d76ae3e482017439412e9cc8e179afb8f47ab415a8a9a7ceaa9b4ee4f9c85e8",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-guest-agent-2.11.2-4.fc28.x86_64.rpm",
    ],
)
