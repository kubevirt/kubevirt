load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# Additional bazel rules

# fix for https://github.com/bazelbuild/rules_go/issues/1999 will be included in > v0.20.3
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "aca4d0081b8bf96dce8f1395904ab1f21f5fea9c08afb9a9c67358982d4491eb",
    strip_prefix = "rules_go-b2c7df325cf96bf8daa9c5d329baf85df4b405eb",
    urls = [
        "https://github.com/bazelbuild/rules_go/archive/b2c7df325cf96bf8daa9c5d329baf85df4b405eb.tar.gz",
        "https://storage.googleapis.com/builddeps/aca4d0081b8bf96dce8f1395904ab1f21f5fea9c08afb9a9c67358982d4491eb",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "41bff2a0b32b02f20c227d234aa25ef3783998e5453f7eade929704dcff7cd4b",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.0/bazel-gazelle-v0.19.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.0/bazel-gazelle-v0.19.0.tar.gz",
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
    name = "com_github_bazelbuild_buildtools",
    sha256 = "875d0c49953e221cfc35d2a3846e502f366dfa4024b271fa266b186ca4664b37",
    strip_prefix = "buildtools-5bcc31df55ec1de770cb52887f2e989e7068301f",
    # version 0.29.0
    url = "https://github.com/bazelbuild/buildtools/archive/5bcc31df55ec1de770cb52887f2e989e7068301f.zip",
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "3efbd23e195727a67f87b2a04fb4388cc7a11a0c0c2cf33eec225fb8ffbb27ea",
    strip_prefix = "rules_docker-0.14.2",
    urls = [
        "https://github.com/bazelbuild/rules_docker/releases/download/v0.14.2/rules_docker-v0.14.2.tar.gz",
        "https://storage.googleapis.com/builddeps/3efbd23e195727a67f87b2a04fb4388cc7a11a0c0c2cf33eec225fb8ffbb27ea",
    ],
)

http_archive(
    name = "com_github_atlassian_bazel_tools",
    sha256 = "e4737fd3636d23f12cd3f9880b1cfa75c1bbdd4a967852785e227f3b0ab11844",
    strip_prefix = "bazel-tools-7d296003f478325b4a933c2b1372426d3a0926f0",
    urls = [
        "https://github.com/atlassian/bazel-tools/archive/7d296003f478325b4a933c2b1372426d3a0926f0.zip",
        "https://storage.googleapis.com/builddeps/e4737fd3636d23f12cd3f9880b1cfa75c1bbdd4a967852785e227f3b0ab11844",
    ],
)

# Libvirt dependencies
http_file(
    name = "libvirt_libs",
    sha256 = "9ff62eb0ed238882a652444e0feb876a9d7878a45fd026a18f64e7ab98faebf0",
    urls = [
        "https://copr-be.cloud.fedoraproject.org/results/%40virtmaint-sig/for-kubevirt/fedora-30-x86_64/01034621-libvirt/libvirt-libs-5.0.0-2.fc30.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ff62eb0ed238882a652444e0feb876a9d7878a45fd026a18f64e7ab98faebf0",
    ],
)

http_file(
    name = "libvirt_libs_ppc64le",
    sha256 = "7993e9fa2a3455aacde56833c476f5900fb0c4a76dc9f0468676a94422d53e3b",
    urls = [
        "https://copr-be.cloud.fedoraproject.org/results/%40virtmaint-sig/for-kubevirt/fedora-30-ppc64le/01113510-libvirt/libvirt-libs-5.0.0-2.fc30.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/7993e9fa2a3455aacde56833c476f5900fb0c4a76dc9f0468676a94422d53e3b",
    ],
)

http_file(
    name = "libvirt_devel",
    sha256 = "e34ca39bb94786b901a626143527f86cae1b6f1c8d1414165465d89e146a612e",
    urls = [
        "https://copr-be.cloud.fedoraproject.org/results/%40virtmaint-sig/for-kubevirt/fedora-30-x86_64/01034621-libvirt/libvirt-devel-5.0.0-2.fc30.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e34ca39bb94786b901a626143527f86cae1b6f1c8d1414165465d89e146a612e",
    ],
)

http_file(
    name = "libvirt_devel_ppc64le",
    sha256 = "45ddd9ba4508c72c4b6a6f03c2198ae4f918fa1324abf566e7b2e80ab51618f2",
    urls = [
        "https://copr-be.cloud.fedoraproject.org/results/%40virtmaint-sig/for-kubevirt/fedora-30-ppc64le/01113510-libvirt/libvirt-devel-5.0.0-2.fc30.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/45ddd9ba4508c72c4b6a6f03c2198ae4f918fa1324abf566e7b2e80ab51618f2",
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
    go_version = "1.12.8",
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

# Pull base image fedora28
container_pull(
    name = "fedora",
    digest = "sha256:f4ca3e9814bef8468b59cc3dea90ab52bcb3747057731f7ab7645d923731fed6",
    registry = "index.docker.io",
    repository = "library/fedora",
    #tag = "30",
)

container_pull(
    name = "fedora_ppc64le",
    digest = "sha256:de1e23d807365800621d10efe27fa7eea865ee0bf9f1beca316f919972312695",
    puller_linux = "@go_puller_linux_ppc64le//file:downloaded",
    registry = "index.docker.io",
    repository = "library/fedora",
)

# Pull base image libvirt
container_pull(
    name = "libvirt",
    digest = "sha256:6a817da0fa85d6698e93594ef9d1d084358d62c93ed1b1237f3d147f77f92b58",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
    #tag = "av-8.2-20200629-e049c89",
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
    sha256 = "eadbd81fe25827a9d7712d0d96b134ab834bfab9e7332a8e9cf54dedd3c02581",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-img-2.11.2-5.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/eadbd81fe25827a9d7712d0d96b134ab834bfab9e7332a8e9cf54dedd3c02581",
    ],
)

http_file(
    name = "qemu-img_ppc64le",
    sha256 = "ce725a0eb493657380ff2d7bb207820d57acdb810c965a47d05e642c4c730984",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/updates/28/Everything/ppc64le/Packages/q/qemu-img-2.11.2-5.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ce725a0eb493657380ff2d7bb207820d57acdb810c965a47d05e642c4c730984",
    ],
)

http_file(
    name = "bzip2",
    sha256 = "5248b7b85e58319c6c16e9b545dc04f1b25ba236e0507a7c923d74b00fe8741c",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/b/bzip2-1.0.6-26.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5248b7b85e58319c6c16e9b545dc04f1b25ba236e0507a7c923d74b00fe8741c",
    ],
)

http_file(
    name = "bzip2_ppc64le",
    sha256 = "e593e694a232829765969e7270cc355d2353436cd2f950029cfa4c0549125f7f",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/b/bzip2-1.0.6-26.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/e593e694a232829765969e7270cc355d2353436cd2f950029cfa4c0549125f7f",
    ],
)

http_file(
    name = "capstone",
    sha256 = "847ebb3a852f82cfe932230d1700cb8b90f602acbe9f9dcf5de7129a1d222c6b",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/c/capstone-3.0.5-1.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/847ebb3a852f82cfe932230d1700cb8b90f602acbe9f9dcf5de7129a1d222c6b",
    ],
)

http_file(
    name = "capstone_ppc64le",
    sha256 = "3bca5e7fe2045e064032572ed58026a21262198a4590b5b8cde5aefcb89b071e",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/updates/28/Everything/ppc64le/Packages/c/capstone-3.0.5-1.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3bca5e7fe2045e064032572ed58026a21262198a4590b5b8cde5aefcb89b071e",
    ],
)

http_file(
    name = "libaio",
    sha256 = "da731218ec1a8e8f690c880d7a45d09722ea01090caba0ae25d9202e0d521404",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/l/libaio-0.3.110-11.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da731218ec1a8e8f690c880d7a45d09722ea01090caba0ae25d9202e0d521404",
    ],
)

http_file(
    name = "libaio_ppc64le",
    sha256 = "2bad2d833f2a572c41dc5e71f03029f697e42a05bf729d9957479e9bd9ee3342",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/l/libaio-0.3.110-11.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/2bad2d833f2a572c41dc5e71f03029f697e42a05bf729d9957479e9bd9ee3342",
    ],
)

http_file(
    name = "libstdc",
    sha256 = "61743bc70033f02604fc18991f2a06efebd3b0f55abcbf5b1f7bd3e3cdca6293",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/l/libstdc++-8.3.1-2.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61743bc70033f02604fc18991f2a06efebd3b0f55abcbf5b1f7bd3e3cdca6293",
    ],
)

http_file(
    name = "libstdc_ppc64le",
    sha256 = "bfe7fc768dbbb23263c3641485d07c374858a1f853fe1261bffa92051790762c",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/updates/28/Everything/ppc64le/Packages/l/libstdc++-8.3.1-2.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/bfe7fc768dbbb23263c3641485d07c374858a1f853fe1261bffa92051790762c",
    ],
)

http_file(
    name = "qemu-guest-agent",
    sha256 = "d9c5072ab2476fbf9e50dd374b2bc680d3accadd2e70d52cae4eb515b6a893f4",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-guest-agent-2.11.2-5.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d9c5072ab2476fbf9e50dd374b2bc680d3accadd2e70d52cae4eb515b6a893f4",
    ],
)

http_file(
    name = "qemu-guest-agent_ppc64le",
    sha256 = "8bc1ec56aa9d9426b31a5cbf6971a5b01456e3ac717456c5b681b8769a8c12da",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/updates/28/Everything/ppc64le/Packages/q/qemu-guest-agent-2.11.2-5.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8bc1ec56aa9d9426b31a5cbf6971a5b01456e3ac717456c5b681b8769a8c12da",
    ],
)

http_file(
    name = "stress",
    sha256 = "bd93021d826c98cbec15b4bf7e0800f723f986e7ed89357c56284a7efa6394b5",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/s/stress-1.0.4-20.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bd93021d826c98cbec15b4bf7e0800f723f986e7ed89357c56284a7efa6394b5",
    ],
)

http_file(
    name = "stress_ppc64le",
    sha256 = "09a33a7a8571bb09519ed01bd98cb71b84bd0a2ad8c21907d80e955bc542a962",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/releases/28/Everything/ppc64le/os/Packages/s/stress-1.0.4-20.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/09a33a7a8571bb09519ed01bd98cb71b84bd0a2ad8c21907d80e955bc542a962",
    ],
)

http_file(
    name = "e2fsprogs",
    sha256 = "d6db37d587a2a0f7cd19e42aea8bd3e5e7c3a9c39c324d40be7514624f9f8f5f",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/e/e2fsprogs-1.44.2-0.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d6db37d587a2a0f7cd19e42aea8bd3e5e7c3a9c39c324d40be7514624f9f8f5f",
    ],
)

http_file(
    name = "e2fsprogs_ppc64le",
    sha256 = "08ebdd9925968fe8dc0655d0a9e98cb410fa68c04349d53c06ccd3ddaae3e3d8",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora-secondary/updates/28/Everything/ppc64le/Packages/e/e2fsprogs-1.44.2-0.fc28.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/08ebdd9925968fe8dc0655d0a9e98cb410fa68c04349d53c06ccd3ddaae3e3d8",
    ],
)

http_file(
    name = "dmidecode",
    sha256 = "5694c041bcebc273cbf9a67f7210b2dd93c517aba55d93d20980b5bdf4be3751",
    urls = [
        "https://dl.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/d/dmidecode-3.1-5.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5694c041bcebc273cbf9a67f7210b2dd93c517aba55d93d20980b5bdf4be3751",
    ],
)

http_file(
    name = "which",
    sha256 = "a7557e0f91d7d710bb7cd939d4263ebbc84aeec9594d7dc4e062ace4b090e3b6",
    urls = [
        "https://dl.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/w/which-2.21-8.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a7557e0f91d7d710bb7cd939d4263ebbc84aeec9594d7dc4e062ace4b090e3b6",
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
