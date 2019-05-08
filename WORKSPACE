load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)

# Additional bazel rules
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "301c8b39b0808c49f98895faa6aa8c92cbd605ab5ad4b6a3a652da33a1a2ba2e",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.18.0/rules_go-0.18.0.tar.gz"],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "7949fc6cc17b5b191103e97481cf8889217263acf52e00b560683413af204fcb",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.16.0/bazel-gazelle-0.16.0.tar.gz"],
)

http_archive(
    name = "com_github_bazelbuild_buildtools",
    sha256 = "e0b5b400cfef17d65886365dc7289cb4ef8dfe07066165607413a271a32aa2a4",
    strip_prefix = "buildtools-db073457c5a56d810e46efc18bb93a4fd7aa7b5e",
    # version 0.20.0
    url = "https://github.com/bazelbuild/buildtools/archive/db073457c5a56d810e46efc18bb93a4fd7aa7b5e.zip",
)

load(
    "@bazel_tools//tools/build_defs/repo:git.bzl",
    "git_repository",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz"],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "e4737fd3636d23f12cd3f9880b1cfa75c1bbdd4a967852785e227f3b0ab11844",
    strip_prefix = "bazel-tools-7d296003f478325b4a933c2b1372426d3a0926f0",
    urls = ["https://github.com/atlassian/bazel-tools/archive/7d296003f478325b4a933c2b1372426d3a0926f0.zip"],
)

# Libvirt dependencies
http_file(
    name = "libvirt_libs",
    sha256 = "0152d484d71abc50759775329fc9ee699fecfb9ff67154d6c6a5d91090f791a7",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/development/rawhide/Everything/ppc64le/os/Packages/l/libvirt-libs-5.1.0-3.fc31.ppc64le.rpm",
    ],
)

http_file(
    name = "libvirt_devel",
    sha256 = "dfb3f3108300cc9e62da822b3073639d9712110880e3025f99fc231874433147",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/development/rawhide/Everything/ppc64le/os/Packages/l/libvirt-devel-5.1.0-3.fc31.ppc64le.rpm",
    ],
)

# Disk images
http_file(
    name = "alpine_image",
    sha256 = "d87d89958991d03ce158119ffa359ee0d9d0024dd46df7010c8c633a24f47be3",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.9/releases/ppc64le/alpine-standard-3.9.2-ppc64le.iso",
    ],
)

http_file(
    name = "cirros_image",
    sha256 = "175063e409f4019acb760478eb1a94819628a1bec9376d26d3aa333449fe061d",
    urls = [
        "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-ppc64le-disk.img",
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
    sha256 = "594678f509ba6827c7b75d076ecfb64d45c6ad95e9fccba7258e6eee9a6a3560",
    urls = [
        "https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/latest-virtio/virtio-win.iso",
    ],
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

go_rules_dependencies()

go_register_toolchains(
    go_version = "1.11.5",
    nogo = "@//:nogo_vet",
)

load("@com_github_atlassian_bazel_tools//goimports:deps.bzl", "goimports_dependencies")

goimports_dependencies()

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
)
load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

# Pull base image fedora28
container_pull(
    name = "fedora",
    digest = "sha256:9b468a0c123715d5e1025876ee05aa5f2053aed4f9142937709b0145f7cd48cf",
    registry = "index.docker.io",
    repository = "ppc64le/fedora",
    #    tag = "28",
)

# Pull base image libvirt
container_pull(
    name = "libvirt",
    digest = "sha256:a815b109a2942796dad95ca846ea019a7879c7607e529a546537d3985273ac7c",
    registry = "index.docker.io",
    repository = "niteshkonkar/libvirt",
    #tag = "5.0.0",
)

# Pull kubevirt-testing image
container_pull(
    name = "kubevirt-testing",
    digest = "sha256:eb86f7388217bb18611c8c4e6169af3463c2a18f420314eb4d742b3d3669b16f",
    registry = "index.docker.io",
    repository = "niteshkonkar/kubevirt-testing",
    #tag = "28",
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
    urls = ["https://github.com/rmohr/rules_container_rpm/archive/v0.0.5.tar.gz"],
)

# Get container-disk-v1alpha RPM's
http_file(
    name = "qemu-img",
    sha256 = "1e4c0171edb631d07f2e45af734d378ea7fe8f70c2a646d9bdb9c8c39696f4f2",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/q/qemu-img-2.11.1-2.fc28.ppc64le.rpm",
    ],
)

http_file(
    name = "bzip2",
    sha256 = "e593e694a232829765969e7270cc355d2353436cd2f950029cfa4c0549125f7f",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/b/bzip2-1.0.6-26.fc28.ppc64le.rpm",
    ],
)

http_file(
    name = "capstone",
    sha256 = "ed0a40f96a28330b39f33965f7945de7373c08806998a2a827eb5def95448d2f",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/c/capstone-3.0.4-14.fc28.ppc64le.rpm",
    ],
)

http_file(
    name = "libaio",
    sha256 = "2bad2d833f2a572c41dc5e71f03029f697e42a05bf729d9957479e9bd9ee3342",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/l/libaio-0.3.110-11.fc28.ppc64le.rpm",
    ],
)

http_file(
    name = "libstdc",
    sha256 = "c7d739bf5e9ebe45aed803d3f39d01372f1c61e535cfe3adaf5fa28de9ec4992",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/l/libstdc++-8.0.1-0.20.fc28.ppc64le.rpm",
    ],
)

http_file(
    name = "qemu-guest-agent",
    sha256 = "7f9607d5eb5ff43877520fcb0fe0b7a385fb6b9e1ee3ebc2d7d9d07170f310b2",
    urls = [
        "https://rpmfind.net/linux/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/q/qemu-guest-agent-2.11.1-2.fc28.ppc64le.rpm",
    ],
)

http_file(
    name = "stress",
    sha256 = "bd93021d826c98cbec15b4bf7e0800f723f986e7ed89357c56284a7efa6394b5",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/28/Everything/x86_64/os/Packages/s/stress-1.0.4-20.fc28.x86_64.rpm",
    ],
)

http_file(
    name = "e2fsprogs",
    sha256 = "d6db37d587a2a0f7cd19e42aea8bd3e5e7c3a9c39c324d40be7514624f9f8f5f",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/updates/28/Everything/x86_64/Packages/e/e2fsprogs-1.44.2-0.fc28.x86_64.rpm",
    ],
)
