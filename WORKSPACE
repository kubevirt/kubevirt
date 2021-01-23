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
        "https://storage.googleapis.com/builddeps/29813b426161f1f09f940e62224f4e54e5737686f2bd22146807d933fa1fa768",
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

http_archive(
    name = "bazeldnf",
    sha256 = "d0833cfb1d60d1e9ca4b3e9619553f22867b15dd827e33010f9b6d093eb7d5f7",
    strip_prefix = "bazeldnf-0.0.14",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.0.14.tar.gz",
        "https://storage.googleapis.com/builddeps/d0833cfb1d60d1e9ca4b3e9619553f22867b15dd827e33010f9b6d093eb7d5f7",
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
    digest = "sha256:9ae61a4649c643caff7c667456a139eb47bd396517e18f4e37312fe95cccba19",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
    #tag = "20201210-917a01f",
)

# TODO: Update this once we have PPC builds of the base image available
container_pull(
    name = "libvirt_ppc64le",
    digest = "sha256:NOT_AVAILABLE",  # Make sure we don't use outdated image by mistake
    puller_linux = "@go_puller_linux_ppc64le//file:downloaded",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
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

bazeldnf_dependencies()

rpm(
    name = "acl-0__2.2.53-5.fc32.ppc64le",
    sha256 = "b05fca142fabc67f663e92e9f23a7bb675af5f9402169486b837ebf8a84eed07",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/acl-2.2.53-5.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/acl-2.2.53-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/acl-2.2.53-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/acl-2.2.53-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/b05fca142fabc67f663e92e9f23a7bb675af5f9402169486b837ebf8a84eed07",
    ],
)

rpm(
    name = "acl-0__2.2.53-5.fc32.x86_64",
    sha256 = "705bdb96aab3a0f9d9e2ff48ead1208e2dbc1927d713d8637632af936235217b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/705bdb96aab3a0f9d9e2ff48ead1208e2dbc1927d713d8637632af936235217b",
    ],
)

rpm(
    name = "alsa-lib-0__1.2.3.2-1.fc32.x86_64",
    sha256 = "d43af5355572121b48f033650311099260bec4a2c2622103a8031725c71aa813",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/alsa-lib-1.2.3.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/a/alsa-lib-1.2.3.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/alsa-lib-1.2.3.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/alsa-lib-1.2.3.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d43af5355572121b48f033650311099260bec4a2c2622103a8031725c71aa813",
    ],
)

rpm(
    name = "alternatives-0__1.11-6.fc32.ppc64le",
    sha256 = "c9a702b4afeb8dc2a023a6a09d463b2aed52b23fa2f7a5f504cdff1f76e83420",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/alternatives-1.11-6.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/alternatives-1.11-6.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/alternatives-1.11-6.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/a/alternatives-1.11-6.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/c9a702b4afeb8dc2a023a6a09d463b2aed52b23fa2f7a5f504cdff1f76e83420",
    ],
)

rpm(
    name = "alternatives-0__1.11-6.fc32.x86_64",
    sha256 = "c574c5432197acbe08ea15c7837be7577cd0b49902a3e65227792f051d73ce5c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c574c5432197acbe08ea15c7837be7577cd0b49902a3e65227792f051d73ce5c",
    ],
)

rpm(
    name = "audit-libs-0__3.0-1.fc32.ppc64le",
    sha256 = "2d21b6f222197cb9516b9abe4d620e5d3e0494693a0ba5d55a7ef651acf408a5",
    urls = [
        "https://fr2.rpmfind.net/linux/fedora-secondary/updates/32/Everything/ppc64le/Packages/a/audit-libs-3.0-1.fc32.ppc64le.rpm",
        "https://fedora-secondary.mirrors.theom.nz/updates/32/Everything/ppc64le/Packages/a/audit-libs-3.0-1.fc32.ppc64le.rpm",
        "https://download-ib01.fedoraproject.org/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/a/audit-libs-3.0-1.fc32.ppc64le.rpm",
        "https://mirror1.cl.netactuate.com/fedora-secondary/updates/32/Everything/ppc64le/Packages/a/audit-libs-3.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/2d21b6f222197cb9516b9abe4d620e5d3e0494693a0ba5d55a7ef651acf408a5",
    ],
)

rpm(
    name = "audit-libs-0__3.0-1.fc32.x86_64",
    sha256 = "8d876200b5d4cba6781733de91b557d7622eb8c4dfcf389bf1e3e26dba305e2d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8d876200b5d4cba6781733de91b557d7622eb8c4dfcf389bf1e3e26dba305e2d",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.16-4.fc32.x86_64",
    sha256 = "df529905e3527b66b059518c181512396e7cbc0e07fc8710dc53d3565941bf65",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df529905e3527b66b059518c181512396e7cbc0e07fc8710dc53d3565941bf65",
    ],
)

rpm(
    name = "basesystem-0__11-9.fc32.ppc64le",
    sha256 = "a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    ],
)

rpm(
    name = "basesystem-0__11-9.fc32.x86_64",
    sha256 = "a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    ],
)

rpm(
    name = "bash-0__5.0.17-1.fc32.ppc64le",
    sha256 = "6da2ea902a71198df5938701873c66f4442cb8a6cd975757a66c630f4fef1094",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/b/bash-5.0.17-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/b/bash-5.0.17-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/b/bash-5.0.17-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/b/bash-5.0.17-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/6da2ea902a71198df5938701873c66f4442cb8a6cd975757a66c630f4fef1094",
    ],
)

rpm(
    name = "bash-0__5.0.17-1.fc32.x86_64",
    sha256 = "31d92d4ef9080bd349188c6f835db0f8b7cf3fe57c6dcff37582f9ee14860ec0",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31d92d4ef9080bd349188c6f835db0f8b7cf3fe57c6dcff37582f9ee14860ec0",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-2.fc32.x86_64",
    sha256 = "b6601c8208b1fa3c41b582bd648c737798bf639da1a049efc0e78c37058280f2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b6601c8208b1fa3c41b582bd648c737798bf639da1a049efc0e78c37058280f2",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-2.fc32.ppc64le",
    sha256 = "3f525e95f91c27b3c128d41678c3c9269e0957fcec738bc76494abd582f1a4bd",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/bzip2-libs-1.0.8-2.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/bzip2-libs-1.0.8-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/bzip2-libs-1.0.8-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/b/bzip2-libs-1.0.8-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3f525e95f91c27b3c128d41678c3c9269e0957fcec738bc76494abd582f1a4bd",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-2.fc32.x86_64",
    sha256 = "842f7a38be2e8dbb14eff3ede4091db214ebe241e1fde7a128e88c4e686b63b0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/842f7a38be2e8dbb14eff3ede4091db214ebe241e1fde7a128e88c4e686b63b0",
    ],
)

rpm(
    name = "ca-certificates-0__2020.2.41-1.1.fc32.ppc64le",
    sha256 = "0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    ],
)

rpm(
    name = "ca-certificates-0__2020.2.41-1.1.fc32.x86_64",
    sha256 = "0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    ],
)

rpm(
    name = "cairo-0__1.16.0-8.fc32.x86_64",
    sha256 = "5722f37e14a7642b42b068e22aebf2d5167ec09e8459c8a12bd5e53f89796d43",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cairo-1.16.0-8.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/c/cairo-1.16.0-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cairo-1.16.0-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cairo-1.16.0-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5722f37e14a7642b42b068e22aebf2d5167ec09e8459c8a12bd5e53f89796d43",
    ],
)

rpm(
    name = "cdparanoia-libs-0__10.2-31.fc32.x86_64",
    sha256 = "a43a5fac13f3fa150d4a26e812223b9e765ff7e31e0c17b2beb690fe667a74ab",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cdparanoia-libs-10.2-31.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cdparanoia-libs-10.2-31.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cdparanoia-libs-10.2-31.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cdparanoia-libs-10.2-31.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a43a5fac13f3fa150d4a26e812223b9e765ff7e31e0c17b2beb690fe667a74ab",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-4.fc32.1.ppc64le",
    sha256 = "92209119047818820222e280ff1a3657bbe676f145608b31992916e590de7f17",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/coreutils-single-8.32-4.fc32.1.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/coreutils-single-8.32-4.fc32.1.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/coreutils-single-8.32-4.fc32.1.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/coreutils-single-8.32-4.fc32.1.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/92209119047818820222e280ff1a3657bbe676f145608b31992916e590de7f17",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-4.fc32.1.x86_64",
    sha256 = "95d659a94bee1464eb8f358b79db363b86eba5a9a472d01970a12d2a27f5d54e",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.1.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.1.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/95d659a94bee1464eb8f358b79db363b86eba5a9a472d01970a12d2a27f5d54e",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-22.fc32.ppc64le",
    sha256 = "34b4f0300b3136bbf30ca3d6cb3b444560965171ced819d228e0a2d0f4edc243",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/c/cracklib-2.9.6-22.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/c/cracklib-2.9.6-22.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/c/cracklib-2.9.6-22.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/c/cracklib-2.9.6-22.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/34b4f0300b3136bbf30ca3d6cb3b444560965171ced819d228e0a2d0f4edc243",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-22.fc32.x86_64",
    sha256 = "862e75c10377098a9cc50407a0395e5f3a81d14b5b6fecfb3f223325c8867829",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/862e75c10377098a9cc50407a0395e5f3a81d14b5b6fecfb3f223325c8867829",
    ],
)

rpm(
    name = "crypto-policies-0__20200619-1.git781bbd4.fc32.ppc64le",
    sha256 = "de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    ],
)

rpm(
    name = "crypto-policies-0__20200619-1.git781bbd4.fc32.x86_64",
    sha256 = "de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.4-1.fc32.ppc64le",
    sha256 = "0b621a7616827415a72d2f4d900ff8ca3f185d5b9f33fe0d63c635b69e7ad736",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/cryptsetup-libs-2.3.4-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/cryptsetup-libs-2.3.4-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/cryptsetup-libs-2.3.4-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/cryptsetup-libs-2.3.4-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0b621a7616827415a72d2f4d900ff8ca3f185d5b9f33fe0d63c635b69e7ad736",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.4-1.fc32.x86_64",
    sha256 = "fbea6919ace47f5be733d8828957d03ce473bb15d3381ce0d52bb1be3775f38a",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.4-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.4-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.4-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fbea6919ace47f5be733d8828957d03ce473bb15d3381ce0d52bb1be3775f38a",
    ],
)

rpm(
    name = "curl-minimal-0__7.69.1-7.fc32.ppc64le",
    sha256 = "21591b98b42413f4273eebe5bb0f8a9cd71420254374821f745141be1004ccd2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/curl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/curl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/curl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/c/curl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/21591b98b42413f4273eebe5bb0f8a9cd71420254374821f745141be1004ccd2",
    ],
)

rpm(
    name = "curl-minimal-0__7.69.1-7.fc32.x86_64",
    sha256 = "b207963fe20d2a09a9a68fa4a136a9a6573e39f03c718297673f1c852eb28528",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b207963fe20d2a09a9a68fa4a136a9a6573e39f03c718297673f1c852eb28528",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-4.fc32.x86_64",
    sha256 = "2878e39b646fc03b9405c99987598da839acd126db5dcfc1ce0c51107993a9ae",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2878e39b646fc03b9405c99987598da839acd126db5dcfc1ce0c51107993a9ae",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-4.fc32.x86_64",
    sha256 = "5b24460fa98b7d7004ea7d1d3b688c3b6e328eea1b23881d4c17f872e372d3cb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5b24460fa98b7d7004ea7d1d3b688c3b6e328eea1b23881d4c17f872e372d3cb",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-4.fc32.x86_64",
    sha256 = "fefa4162a563eba24714ac43874c508d1ba036afb5127c5d21bbcbeaf238a740",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fefa4162a563eba24714ac43874c508d1ba036afb5127c5d21bbcbeaf238a740",
    ],
)

rpm(
    name = "dbus-1__1.12.20-1.fc32.ppc64le",
    sha256 = "737b64465fd081fa65df98caffca7117c6aff47847f74083995e51baa657973e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-1.12.20-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-1.12.20-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-1.12.20-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-1.12.20-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/737b64465fd081fa65df98caffca7117c6aff47847f74083995e51baa657973e",
    ],
)

rpm(
    name = "dbus-1__1.12.20-1.fc32.x86_64",
    sha256 = "0f4bac9a18a2535b85a7b9d8ac4c652edbb0047224f89548122f6f1257a169eb",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0f4bac9a18a2535b85a7b9d8ac4c652edbb0047224f89548122f6f1257a169eb",
    ],
)

rpm(
    name = "dbus-broker-0__24-1.fc32.ppc64le",
    sha256 = "4dcc642bee9aec84b7a3ec45260c2470b914a17d33d22b0d7607406ddd9c3337",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-broker-24-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-broker-24-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-broker-24-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-broker-24-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/4dcc642bee9aec84b7a3ec45260c2470b914a17d33d22b0d7607406ddd9c3337",
    ],
)

rpm(
    name = "dbus-broker-0__24-1.fc32.x86_64",
    sha256 = "8f896f77cd4c268115b2e8b8a64e5cdcb63016c9a3e3ac02df8c2161894a82f8",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-24-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-24-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-24-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-24-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8f896f77cd4c268115b2e8b8a64e5cdcb63016c9a3e3ac02df8c2161894a82f8",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-1.fc32.ppc64le",
    sha256 = "0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-1.fc32.x86_64",
    sha256 = "0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    ],
)

rpm(
    name = "dbus-libs-1__1.12.20-1.fc32.x86_64",
    sha256 = "a967348d3ee8b4db1405921ff8bef14d3bcb3d04f4b3c67f6821cc65ec612fef",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-libs-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-libs-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-libs-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-libs-1.12.20-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a967348d3ee8b4db1405921ff8bef14d3bcb3d04f4b3c67f6821cc65ec612fef",
    ],
)

rpm(
    name = "dejavu-sans-fonts-0__2.37-8.fc32.x86_64",
    sha256 = "f22e2fb4cbe5ffe60879f3801db3b4afe7898bdeaf16246aab76520c613dfb36",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dejavu-sans-fonts-2.37-8.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/d/dejavu-sans-fonts-2.37-8.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dejavu-sans-fonts-2.37-8.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dejavu-sans-fonts-2.37-8.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f22e2fb4cbe5ffe60879f3801db3b4afe7898bdeaf16246aab76520c613dfb36",
    ],
)

rpm(
    name = "device-mapper-0__1.02.171-1.fc32.ppc64le",
    sha256 = "5c6b25abd51b079bc1b7e29b351dfd59b926481a099e907d67d61f2c4e1405a9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-1.02.171-1.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-1.02.171-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-1.02.171-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-1.02.171-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/5c6b25abd51b079bc1b7e29b351dfd59b926481a099e907d67d61f2c4e1405a9",
    ],
)

rpm(
    name = "device-mapper-0__1.02.171-1.fc32.x86_64",
    sha256 = "c132999a3f110029cd427f7578965ad558e91374637087d5230ee11c626ebcd4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c132999a3f110029cd427f7578965ad558e91374637087d5230ee11c626ebcd4",
    ],
)

rpm(
    name = "device-mapper-libs-0__1.02.171-1.fc32.ppc64le",
    sha256 = "8352b687fa2ce6ec5cfa6558042045efb2ec883fe854ab1b9d71f6c32fd553bd",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8352b687fa2ce6ec5cfa6558042045efb2ec883fe854ab1b9d71f6c32fd553bd",
    ],
)

rpm(
    name = "device-mapper-libs-0__1.02.171-1.fc32.x86_64",
    sha256 = "61cae80187ef2924857fdfc48a240646d23b331482cf181e7d8c661b02c15949",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61cae80187ef2924857fdfc48a240646d23b331482cf181e7d8c661b02c15949",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.2-4.fc32.x86_64",
    sha256 = "1513c0cd213973c6939ac22b9faf7b21347559f2702eb398301a2c68b12e0b8c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1513c0cd213973c6939ac22b9faf7b21347559f2702eb398301a2c68b12e0b8c",
    ],
)

rpm(
    name = "diffutils-0__3.7-4.fc32.x86_64",
    sha256 = "187dd61be71efcca6adf9819a523d432217abb335afcb2b95ef27b72928aff4b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/187dd61be71efcca6adf9819a523d432217abb335afcb2b95ef27b72928aff4b",
    ],
)

rpm(
    name = "dmidecode-1__3.2-5.fc32.x86_64",
    sha256 = "e40be03bd5808e640bb5fb18196499680a7b7b1d3fce47617f987baee849c0e5",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e40be03bd5808e640bb5fb18196499680a7b7b1d3fce47617f987baee849c0e5",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.5-3.fc32.ppc64le",
    sha256 = "e8772a8fab827cef27fe0781be26c2dd5cf55d0f9a8882ba20cb66712ca0d3d7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-1.45.5-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-1.45.5-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-1.45.5-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-1.45.5-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/e8772a8fab827cef27fe0781be26c2dd5cf55d0f9a8882ba20cb66712ca0d3d7",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.5-3.fc32.x86_64",
    sha256 = "2fa5e252441852dae918b522a2ff3f46a5bbee4ce8936e06702bf65f57d7ff99",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2fa5e252441852dae918b522a2ff3f46a5bbee4ce8936e06702bf65f57d7ff99",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.5-3.fc32.ppc64le",
    sha256 = "406cda55432888a29ba3c0f60deb8a6735cfaf4fd802bd9635c13c50d5c93032",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/406cda55432888a29ba3c0f60deb8a6735cfaf4fd802bd9635c13c50d5c93032",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.5-3.fc32.x86_64",
    sha256 = "26db62c2bc52c3eee5f3039cdbdf19498f675d0f45aec0c2a1c61c635f01479e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/26db62c2bc52c3eee5f3039cdbdf19498f675d0f45aec0c2a1c61c635f01479e",
    ],
)

rpm(
    name = "edk2-ovmf-0__20200801stable-1.fc32.x86_64",
    sha256 = "220848c197bd0e464172b94a4af9b681f788a916976d05036825241e1aaf753a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/220848c197bd0e464172b94a4af9b681f788a916976d05036825241e1aaf753a",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.182-1.fc32.ppc64le",
    sha256 = "114a84338752fe0a8bd6762b3065c5751958f9ef12002fb6a0dbe7144e218f20",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/114a84338752fe0a8bd6762b3065c5751958f9ef12002fb6a0dbe7144e218f20",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.182-1.fc32.x86_64",
    sha256 = "114a84338752fe0a8bd6762b3065c5751958f9ef12002fb6a0dbe7144e218f20",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.182-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/114a84338752fe0a8bd6762b3065c5751958f9ef12002fb6a0dbe7144e218f20",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.182-1.fc32.ppc64le",
    sha256 = "d44f8953d316c38db1ac44644c9e92bebabc51045e9524998d0411ec854cf017",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libelf-0.182-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libelf-0.182-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libelf-0.182-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libelf-0.182-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d44f8953d316c38db1ac44644c9e92bebabc51045e9524998d0411ec854cf017",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.182-1.fc32.x86_64",
    sha256 = "fecfca5bbcaebcc634e024c00939abca27bc19221ec2bde2b92d572513c5b3cc",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.182-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.182-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.182-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.182-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fecfca5bbcaebcc634e024c00939abca27bc19221ec2bde2b92d572513c5b3cc",
    ],
)

rpm(
    name = "elfutils-libs-0__0.182-1.fc32.ppc64le",
    sha256 = "3887755c8c0d7dd3a1bfc5b82793bd7d5ed4bb8e1f85ef623d445a6a5e2595e8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libs-0.182-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libs-0.182-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libs-0.182-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/e/elfutils-libs-0.182-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3887755c8c0d7dd3a1bfc5b82793bd7d5ed4bb8e1f85ef623d445a6a5e2595e8",
    ],
)

rpm(
    name = "elfutils-libs-0__0.182-1.fc32.x86_64",
    sha256 = "5ac887e322350ebccd4c195245ac9f3c6a31cccf435f41bd13ca9da2b59ac1f6",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.182-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.182-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.182-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.182-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5ac887e322350ebccd4c195245ac9f3c6a31cccf435f41bd13ca9da2b59ac1f6",
    ],
)

rpm(
    name = "expat-0__2.2.8-2.fc32.ppc64le",
    sha256 = "08c49261b79f0f9e5fa52914116e375451df8b84828ef4247cbab59ceac7f385",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/expat-2.2.8-2.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/expat-2.2.8-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/expat-2.2.8-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/e/expat-2.2.8-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/08c49261b79f0f9e5fa52914116e375451df8b84828ef4247cbab59ceac7f385",
    ],
)

rpm(
    name = "expat-0__2.2.8-2.fc32.x86_64",
    sha256 = "8fc2ae85f242105987d8fa7f05e4fa19358a7c81dff5fa163cf021eb6b9905e9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8fc2ae85f242105987d8fa7f05e4fa19358a7c81dff5fa163cf021eb6b9905e9",
    ],
)

rpm(
    name = "fedora-gpg-keys-0__32-10.ppc64le",
    sha256 = "e68a4a5857d66762df1970c624071987780aea7aaaa5c4a561263c39080d397f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e68a4a5857d66762df1970c624071987780aea7aaaa5c4a561263c39080d397f",
    ],
)

rpm(
    name = "fedora-gpg-keys-0__32-10.x86_64",
    sha256 = "e68a4a5857d66762df1970c624071987780aea7aaaa5c4a561263c39080d397f",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e68a4a5857d66762df1970c624071987780aea7aaaa5c4a561263c39080d397f",
    ],
)

rpm(
    name = "fedora-logos-httpd-0__30.0.2-4.fc32.ppc64le",
    sha256 = "458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    ],
)

rpm(
    name = "fedora-logos-httpd-0__30.0.2-4.fc32.x86_64",
    sha256 = "458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    urls = [
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    ],
)

rpm(
    name = "fedora-release-common-0__32-4.ppc64le",
    sha256 = "829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    ],
)

rpm(
    name = "fedora-release-common-0__32-4.x86_64",
    sha256 = "829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    ],
)

rpm(
    name = "fedora-release-container-0__32-4.ppc64le",
    sha256 = "21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    ],
)

rpm(
    name = "fedora-release-container-0__32-4.x86_64",
    sha256 = "21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    ],
)

rpm(
    name = "fedora-repos-0__32-10.ppc64le",
    sha256 = "61554ad6ee72e41b74df5ce56ce00b3d5600bc05c146948eb076d47e680af855",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61554ad6ee72e41b74df5ce56ce00b3d5600bc05c146948eb076d47e680af855",
    ],
)

rpm(
    name = "fedora-repos-0__32-10.x86_64",
    sha256 = "61554ad6ee72e41b74df5ce56ce00b3d5600bc05c146948eb076d47e680af855",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-10.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61554ad6ee72e41b74df5ce56ce00b3d5600bc05c146948eb076d47e680af855",
    ],
)

rpm(
    name = "file-libs-0__5.38-4.fc32.x86_64",
    sha256 = "9b3d457f2950fcb0170d1608151ea27f0dd7c970d2d3ff53e2a6644398a6571c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9b3d457f2950fcb0170d1608151ea27f0dd7c970d2d3ff53e2a6644398a6571c",
    ],
)

rpm(
    name = "filesystem-0__3.14-2.fc32.ppc64le",
    sha256 = "7b8c9fd5c225a6fc77390f44acbae7266629d89372a55d454f36c86762d5cf46",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/filesystem-3.14-2.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/filesystem-3.14-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/filesystem-3.14-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/filesystem-3.14-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/7b8c9fd5c225a6fc77390f44acbae7266629d89372a55d454f36c86762d5cf46",
    ],
)

rpm(
    name = "filesystem-0__3.14-2.fc32.x86_64",
    sha256 = "1110261787146443e089955912255d99daf7ba042c3743e13648a9eb3d80ceb4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1110261787146443e089955912255d99daf7ba042c3743e13648a9eb3d80ceb4",
    ],
)

rpm(
    name = "findutils-1__4.7.0-4.fc32.x86_64",
    sha256 = "c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    ],
)

rpm(
    name = "fontconfig-0__2.13.92-9.fc32.x86_64",
    sha256 = "5a8e8fc2734100b61fbb7065d1b9629e7cd0c0f3601632ae8c85a4421e8838b3",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fontconfig-2.13.92-9.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/fontconfig-2.13.92-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fontconfig-2.13.92-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fontconfig-2.13.92-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a8e8fc2734100b61fbb7065d1b9629e7cd0c0f3601632ae8c85a4421e8838b3",
    ],
)

rpm(
    name = "fonts-filesystem-0__2.0.3-1.fc32.x86_64",
    sha256 = "879ba2533610771dbf3fa103fdbde878edf255b771b53aa8a170009d01446012",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fonts-filesystem-2.0.3-1.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fonts-filesystem-2.0.3-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fonts-filesystem-2.0.3-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fonts-filesystem-2.0.3-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/879ba2533610771dbf3fa103fdbde878edf255b771b53aa8a170009d01446012",
    ],
)

rpm(
    name = "freetype-0__2.10.4-1.fc32.x86_64",
    sha256 = "c885b78a6304db062cb135ba2803b4ee97f371d139b1758d1da06edf2fd1457b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/freetype-2.10.4-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/f/freetype-2.10.4-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/freetype-2.10.4-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/freetype-2.10.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c885b78a6304db062cb135ba2803b4ee97f371d139b1758d1da06edf2fd1457b",
    ],
)

rpm(
    name = "fribidi-0__1.0.9-1.fc32.x86_64",
    sha256 = "488ca8e6d4c79948c317b57fcb2dfbf1bab334804813bd0a415edf311f724b29",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fribidi-1.0.9-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fribidi-1.0.9-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fribidi-1.0.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fribidi-1.0.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/488ca8e6d4c79948c317b57fcb2dfbf1bab334804813bd0a415edf311f724b29",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-9.fc32.ppc64le",
    sha256 = "72f486337a12b78e944a543bce96079ae12d167462f1b0e94a409ac586e40970",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fuse-libs-2.9.9-9.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fuse-libs-2.9.9-9.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fuse-libs-2.9.9-9.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/f/fuse-libs-2.9.9-9.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/72f486337a12b78e944a543bce96079ae12d167462f1b0e94a409ac586e40970",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-9.fc32.x86_64",
    sha256 = "53992752850779218421994f61f1589eda5d368e28d340dccaae3f67de06e7f2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/53992752850779218421994f61f1589eda5d368e28d340dccaae3f67de06e7f2",
    ],
)

rpm(
    name = "gawk-0__5.0.1-7.fc32.ppc64le",
    sha256 = "d4ef3087f3582b5f1817fe9fe233c06a9242f0d2411208dca0331437c3891075",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gawk-5.0.1-7.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gawk-5.0.1-7.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gawk-5.0.1-7.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gawk-5.0.1-7.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d4ef3087f3582b5f1817fe9fe233c06a9242f0d2411208dca0331437c3891075",
    ],
)

rpm(
    name = "gawk-0__5.0.1-7.fc32.x86_64",
    sha256 = "d0e5d0104cf20c8dd332053a5903aab9b7fdadb84b35a1bfb3a6456f3399eb32",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d0e5d0104cf20c8dd332053a5903aab9b7fdadb84b35a1bfb3a6456f3399eb32",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18.1-3.fc32.ppc64le",
    sha256 = "0276f3c6e381cddc3c782960858e98404813f6c48e22a9bcbc4588b7104e845f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gdbm-libs-1.18.1-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gdbm-libs-1.18.1-3.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gdbm-libs-1.18.1-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gdbm-libs-1.18.1-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0276f3c6e381cddc3c782960858e98404813f6c48e22a9bcbc4588b7104e845f",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18.1-3.fc32.x86_64",
    sha256 = "9899cfd32ada2537693af30b60051da21c6264b0d0db51ba709fceb179d4c836",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9899cfd32ada2537693af30b60051da21c6264b0d0db51ba709fceb179d4c836",
    ],
)

rpm(
    name = "genisoimage-0__1.1.11-44.fc32.x86_64",
    sha256 = "b76928dc67709dbd9ab39aff18ea7bf4355ca74e5d0f969d21e3499aa05fb8e1",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b76928dc67709dbd9ab39aff18ea7bf4355ca74e5d0f969d21e3499aa05fb8e1",
    ],
)

rpm(
    name = "gettext-0__0.21-1.fc32.x86_64",
    sha256 = "671a966b860875e2da6f95e1c8c33413ca6bc5f75ac32203a5645ba44e6ee398",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/671a966b860875e2da6f95e1c8c33413ca6bc5f75ac32203a5645ba44e6ee398",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-1.fc32.x86_64",
    sha256 = "34d66d6d5891be611cca8b2c3e13f5832679094c2e7cf0b3edc5e538b6d3eefe",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/34d66d6d5891be611cca8b2c3e13f5832679094c2e7cf0b3edc5e538b6d3eefe",
    ],
)

rpm(
    name = "glib2-0__2.64.6-1.fc32.ppc64le",
    sha256 = "d3d7a2d14a8c71619921e35a468d226f97a19d8926834310b4a02db129b62f2b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glib2-2.64.6-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glib2-2.64.6-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glib2-2.64.6-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glib2-2.64.6-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d3d7a2d14a8c71619921e35a468d226f97a19d8926834310b4a02db129b62f2b",
    ],
)

rpm(
    name = "glib2-0__2.64.6-1.fc32.x86_64",
    sha256 = "2f0f896eff6611e668944c83a63cbbe3a677802c89e4507975da1dba7ed82fed",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2f0f896eff6611e668944c83a63cbbe3a677802c89e4507975da1dba7ed82fed",
    ],
)

rpm(
    name = "glibc-0__2.31-4.fc32.ppc64le",
    sha256 = "fd0bf2cba563e4474a5ac48a1a477d2cd22ea0a603ab259d6d23790be28ce4bf",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-2.31-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-2.31-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-2.31-4.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-2.31-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/fd0bf2cba563e4474a5ac48a1a477d2cd22ea0a603ab259d6d23790be28ce4bf",
    ],
)

rpm(
    name = "glibc-0__2.31-4.fc32.x86_64",
    sha256 = "2145ee8f8af2d8c1023fc2ace21483e246f31fb4d0294d39e48551300d919a5d",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2145ee8f8af2d8c1023fc2ace21483e246f31fb4d0294d39e48551300d919a5d",
    ],
)

rpm(
    name = "glibc-common-0__2.31-4.fc32.ppc64le",
    sha256 = "820b0f1213f89b37fe358dfeaee5f4b3ffe1e14001b42a3d0924053681cd3363",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-common-2.31-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-common-2.31-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-common-2.31-4.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-common-2.31-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/820b0f1213f89b37fe358dfeaee5f4b3ffe1e14001b42a3d0924053681cd3363",
    ],
)

rpm(
    name = "glibc-common-0__2.31-4.fc32.x86_64",
    sha256 = "41b67774fd01311ee1813ac1c1c1f555fc1cf938fa67668b727f863c244948b8",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/41b67774fd01311ee1813ac1c1c1f555fc1cf938fa67668b727f863c244948b8",
    ],
)

rpm(
    name = "glibc-langpack-en-0__2.31-4.fc32.ppc64le",
    sha256 = "db30a1c3cb519607ef222f7fc93ce3bd7f9e42a9592c6c7ae883817bbe5b9245",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-langpack-en-2.31-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-langpack-en-2.31-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-langpack-en-2.31-4.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/glibc-langpack-en-2.31-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/db30a1c3cb519607ef222f7fc93ce3bd7f9e42a9592c6c7ae883817bbe5b9245",
    ],
)

rpm(
    name = "glibc-langpack-en-0__2.31-4.fc32.x86_64",
    sha256 = "41c695d8f1555ecc4d58a00f81a9e13e84640933f5c3d12d677fed728e2a01e5",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/41c695d8f1555ecc4d58a00f81a9e13e84640933f5c3d12d677fed728e2a01e5",
    ],
)

rpm(
    name = "glusterfs-0__7.9-1.fc32.x86_64",
    sha256 = "6ce9b3afd7c13ec7c37021e66107ea67aa190bd52803b6bf7545b61ddde5e9d9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-7.9-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-7.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-7.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-7.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6ce9b3afd7c13ec7c37021e66107ea67aa190bd52803b6bf7545b61ddde5e9d9",
    ],
)

rpm(
    name = "glusterfs-api-0__7.9-1.fc32.x86_64",
    sha256 = "15fbf267ce6453f58c2e922f638c209c4646220c55eb20d1fe8263b87a0a6eee",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-api-7.9-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-api-7.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-api-7.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-api-7.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/15fbf267ce6453f58c2e922f638c209c4646220c55eb20d1fe8263b87a0a6eee",
    ],
)

rpm(
    name = "glusterfs-client-xlators-0__7.9-1.fc32.x86_64",
    sha256 = "313b75a2d920f100319163dbbfd88fb863ec8ac732d4c386f52b973e14359aa1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-client-xlators-7.9-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-client-xlators-7.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-client-xlators-7.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-client-xlators-7.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/313b75a2d920f100319163dbbfd88fb863ec8ac732d4c386f52b973e14359aa1",
    ],
)

rpm(
    name = "glusterfs-libs-0__7.9-1.fc32.x86_64",
    sha256 = "0c25b4580dae9021962c7a814be1fe1c6d0bd39ffe0c1296e17d9655e5d2a727",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-libs-7.9-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-libs-7.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-libs-7.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glusterfs-libs-7.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c25b4580dae9021962c7a814be1fe1c6d0bd39ffe0c1296e17d9655e5d2a727",
    ],
)

rpm(
    name = "gmp-1__6.1.2-13.fc32.ppc64le",
    sha256 = "07e3b05e88b77fe26c766ec10ffe9e045b1ebe345aea44cb421c4085f4d5bf9c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gmp-6.1.2-13.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gmp-6.1.2-13.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gmp-6.1.2-13.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gmp-6.1.2-13.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/07e3b05e88b77fe26c766ec10ffe9e045b1ebe345aea44cb421c4085f4d5bf9c",
    ],
)

rpm(
    name = "gmp-1__6.1.2-13.fc32.x86_64",
    sha256 = "178e4470a6dfca84ec133932606737bfe167094560bf473940504c511354ddc9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/178e4470a6dfca84ec133932606737bfe167094560bf473940504c511354ddc9",
    ],
)

rpm(
    name = "gnutls-0__3.6.15-1.fc32.ppc64le",
    sha256 = "8d5d8faf33c257af5741151d06a29ef151fbe66f28397a936320f24fc842958b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/gnutls-3.6.15-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/gnutls-3.6.15-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/gnutls-3.6.15-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/gnutls-3.6.15-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8d5d8faf33c257af5741151d06a29ef151fbe66f28397a936320f24fc842958b",
    ],
)

rpm(
    name = "gnutls-0__3.6.15-1.fc32.x86_64",
    sha256 = "802c67682c05190dd720928dbd4e5bad394e8b2eecc88af42db0007161aa9738",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/802c67682c05190dd720928dbd4e5bad394e8b2eecc88af42db0007161aa9738",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.15-1.fc32.x86_64",
    sha256 = "e18b2f511d01bfb2ec0bf42e6ba99ed3db88b133784c1011458e72df154a722e",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e18b2f511d01bfb2ec0bf42e6ba99ed3db88b133784c1011458e72df154a722e",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.15-1.fc32.x86_64",
    sha256 = "484c3af6f704a48aa67d210c7afd977a8229cd875a9d04e570761e1246ad596f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/484c3af6f704a48aa67d210c7afd977a8229cd875a9d04e570761e1246ad596f",
    ],
)

rpm(
    name = "gperftools-libs-0__2.7-7.fc32.x86_64",
    sha256 = "4bde0737a685e82c732b9a5d2daf08a0b6a66c0abd699defcfefc0c7bd2ecdf6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4bde0737a685e82c732b9a5d2daf08a0b6a66c0abd699defcfefc0c7bd2ecdf6",
    ],
)

rpm(
    name = "graphene-0__1.10.2-1.fc32.x86_64",
    sha256 = "516714018f8e91f044f30d27e491cc0a6fd2d0e0f3df5b9e481d05cc6cc8fdde",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphene-1.10.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphene-1.10.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphene-1.10.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphene-1.10.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/516714018f8e91f044f30d27e491cc0a6fd2d0e0f3df5b9e481d05cc6cc8fdde",
    ],
)

rpm(
    name = "graphite2-0__1.3.14-1.fc32.x86_64",
    sha256 = "08731b385dda9a5636ce6ab7a9f0e795e7664d5359207c78ba505d981441fe2b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphite2-1.3.14-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphite2-1.3.14-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphite2-1.3.14-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/graphite2-1.3.14-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/08731b385dda9a5636ce6ab7a9f0e795e7664d5359207c78ba505d981441fe2b",
    ],
)

rpm(
    name = "grep-0__3.3-4.fc32.ppc64le",
    sha256 = "09a2d223b009611fe0ee2cbe98b913daa820b6034df310e522d71e374f3fd61a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/grep-3.3-4.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/grep-3.3-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/grep-3.3-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/grep-3.3-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/09a2d223b009611fe0ee2cbe98b913daa820b6034df310e522d71e374f3fd61a",
    ],
)

rpm(
    name = "grep-0__3.3-4.fc32.x86_64",
    sha256 = "759165656ac8141b0c0ada230c258ffcd4516c4c8d132d7fbaf762cd5a5e4095",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/759165656ac8141b0c0ada230c258ffcd4516c4c8d132d7fbaf762cd5a5e4095",
    ],
)

rpm(
    name = "groff-base-0__1.22.3-22.fc32.ppc64le",
    sha256 = "0d80f840544f25aa787a6d3fceb075948aec24f865d42d4ca779ab3e0f0d3615",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/groff-base-1.22.3-22.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/groff-base-1.22.3-22.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/groff-base-1.22.3-22.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/g/groff-base-1.22.3-22.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0d80f840544f25aa787a6d3fceb075948aec24f865d42d4ca779ab3e0f0d3615",
    ],
)

rpm(
    name = "groff-base-0__1.22.3-22.fc32.x86_64",
    sha256 = "a81e62e044a9cb5c752e55b3e6e40c3248ca0b595236d8f6f62e42251379454d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a81e62e044a9cb5c752e55b3e6e40c3248ca0b595236d8f6f62e42251379454d",
    ],
)

rpm(
    name = "gssproxy-0__0.8.2-8.fc32.x86_64",
    sha256 = "31ec30af8b97915f9bb0606549aa6970500f09065fa0a4f1f1b7aeed10a688d4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gssproxy-0.8.2-8.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gssproxy-0.8.2-8.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gssproxy-0.8.2-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gssproxy-0.8.2-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31ec30af8b97915f9bb0606549aa6970500f09065fa0a4f1f1b7aeed10a688d4",
    ],
)

rpm(
    name = "gstreamer1-0__1.16.2-2.fc32.x86_64",
    sha256 = "3cf5d0e1c6dd09b4b047f13bdafd7413d8e4c4f586fa5de932f3410c5cbcf82e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gstreamer1-1.16.2-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gstreamer1-1.16.2-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gstreamer1-1.16.2-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gstreamer1-1.16.2-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3cf5d0e1c6dd09b4b047f13bdafd7413d8e4c4f586fa5de932f3410c5cbcf82e",
    ],
)

rpm(
    name = "gstreamer1-plugins-base-0__1.16.2-3.fc32.x86_64",
    sha256 = "72b497a0b2f6f85a11b74ec1d7c010f48496f272cf7bace5addf39f3f2cf15a0",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gstreamer1-plugins-base-1.16.2-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/g/gstreamer1-plugins-base-1.16.2-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gstreamer1-plugins-base-1.16.2-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gstreamer1-plugins-base-1.16.2-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/72b497a0b2f6f85a11b74ec1d7c010f48496f272cf7bace5addf39f3f2cf15a0",
    ],
)

rpm(
    name = "gzip-0__1.10-2.fc32.ppc64le",
    sha256 = "9a6e21e353d104c46c796559ec973fd0b73d25633473929ee7910ab57556205d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gzip-1.10-2.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gzip-1.10-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gzip-1.10-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/g/gzip-1.10-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/9a6e21e353d104c46c796559ec973fd0b73d25633473929ee7910ab57556205d",
    ],
)

rpm(
    name = "gzip-0__1.10-2.fc32.x86_64",
    sha256 = "53f1e8570b175e8b58895646df6d8068a7e1f3cb1bafdde714ddd038bcf91e85",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/53f1e8570b175e8b58895646df6d8068a7e1f3cb1bafdde714ddd038bcf91e85",
    ],
)

rpm(
    name = "harfbuzz-0__2.6.4-3.fc32.x86_64",
    sha256 = "48756e87aba1d8757243a9d8cf4649e8ab9b129adb3cc13c2b8d425c58bb0cb5",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/harfbuzz-2.6.4-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/harfbuzz-2.6.4-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/harfbuzz-2.6.4-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/harfbuzz-2.6.4-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/48756e87aba1d8757243a9d8cf4649e8ab9b129adb3cc13c2b8d425c58bb0cb5",
    ],
)

rpm(
    name = "hwdata-0__0.342-1.fc32.ppc64le",
    sha256 = "117d7e2287769f0f9d14b8d9b967c3bacf99150427b322fcb0643578abaf1d40",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/117d7e2287769f0f9d14b8d9b967c3bacf99150427b322fcb0643578abaf1d40",
    ],
)

rpm(
    name = "hwdata-0__0.342-1.fc32.x86_64",
    sha256 = "117d7e2287769f0f9d14b8d9b967c3bacf99150427b322fcb0643578abaf1d40",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.342-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/117d7e2287769f0f9d14b8d9b967c3bacf99150427b322fcb0643578abaf1d40",
    ],
)

rpm(
    name = "iproute-0__5.7.0-1.fc32.x86_64",
    sha256 = "a3b5786f7050c97ccd7e1a5f71fea1500578b1d36a5e28b47a6afdacd0a25c06",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.7.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.7.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.7.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.7.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a3b5786f7050c97ccd7e1a5f71fea1500578b1d36a5e28b47a6afdacd0a25c06",
    ],
)

rpm(
    name = "iproute-tc-0__5.7.0-1.fc32.x86_64",
    sha256 = "2fc37a0b615dc9ac31d51467a39209005e4f181337ba0e3aa0990c7d1f701210",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.7.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.7.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.7.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.7.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2fc37a0b615dc9ac31d51467a39209005e4f181337ba0e3aa0990c7d1f701210",
    ],
)

rpm(
    name = "iptables-0__1.8.4-9.fc32.x86_64",
    sha256 = "6505f2881185c2a09eb623bbc1e10c47573386b385879484251ea48d196f2094",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6505f2881185c2a09eb623bbc1e10c47573386b385879484251ea48d196f2094",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-9.fc32.ppc64le",
    sha256 = "cb62f6d86bd402c9168983bd258c4083caffff34f64a7ae304e6a511b715d5dc",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iptables-libs-1.8.4-9.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iptables-libs-1.8.4-9.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iptables-libs-1.8.4-9.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iptables-libs-1.8.4-9.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cb62f6d86bd402c9168983bd258c4083caffff34f64a7ae304e6a511b715d5dc",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-9.fc32.x86_64",
    sha256 = "dcf038adbb690e6aa3dcc020576eccf1ee3eeecb0cddd3011fa5f99e85c8bf3a",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dcf038adbb690e6aa3dcc020576eccf1ee3eeecb0cddd3011fa5f99e85c8bf3a",
    ],
)

rpm(
    name = "iputils-0__20200821-1.fc32.ppc64le",
    sha256 = "51ea808a5c502e341b0d7de07a465f6a5282edae170d6c0ca25904b84510ff91",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iputils-20200821-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iputils-20200821-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iputils-20200821-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/i/iputils-20200821-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/51ea808a5c502e341b0d7de07a465f6a5282edae170d6c0ca25904b84510ff91",
    ],
)

rpm(
    name = "iputils-0__20200821-1.fc32.x86_64",
    sha256 = "a5c17f8a29defceb5d33ff860c205ba1db36d36828aecd9b96609207697a8047",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a5c17f8a29defceb5d33ff860c205ba1db36d36828aecd9b96609207697a8047",
    ],
)

rpm(
    name = "ipxe-roms-qemu-0__20190125-4.git36a4c85f.fc32.x86_64",
    sha256 = "6e298c78b9ccc125b0fdfdda4c39a993aaacd1accc62a23df9f30e46c17b64a9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6e298c78b9ccc125b0fdfdda4c39a993aaacd1accc62a23df9f30e46c17b64a9",
    ],
)

rpm(
    name = "iso-codes-0__4.4-2.fc32.x86_64",
    sha256 = "7080e23ec4ff5c0cacfa3476daa3c5c4f837b53d3806613f6ffcf1a352a480e2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/iso-codes-4.4-2.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/iso-codes-4.4-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/iso-codes-4.4-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/iso-codes-4.4-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7080e23ec4ff5c0cacfa3476daa3c5c4f837b53d3806613f6ffcf1a352a480e2",
    ],
)

rpm(
    name = "jansson-0__2.12-5.fc32.x86_64",
    sha256 = "975719a0c73cf5cb5bcbc8ad11b816ed75923dccd9c091baa4a6c6000753dcd8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/975719a0c73cf5cb5bcbc8ad11b816ed75923dccd9c091baa4a6c6000753dcd8",
    ],
)

rpm(
    name = "json-c-0__0.13.1-13.fc32.ppc64le",
    sha256 = "389e0f8a92ad1f6224fb1ca9bdfa76573de4d7c157ccef78627c89348c95193c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/j/json-c-0.13.1-13.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/j/json-c-0.13.1-13.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/j/json-c-0.13.1-13.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/j/json-c-0.13.1-13.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/389e0f8a92ad1f6224fb1ca9bdfa76573de4d7c157ccef78627c89348c95193c",
    ],
)

rpm(
    name = "json-c-0__0.13.1-13.fc32.x86_64",
    sha256 = "56ecdfc358f2149bc9f6fd38161d33fe45177c11059fd813143c8d314b1019fc",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56ecdfc358f2149bc9f6fd38161d33fe45177c11059fd813143c8d314b1019fc",
    ],
)

rpm(
    name = "keyutils-0__1.6-4.fc32.x86_64",
    sha256 = "9ee276ed9d036a4483bb8d1bd6f6d697b161a04276c6ce5f1d3f40d48e04bccb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-1.6-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-1.6-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-1.6-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-1.6-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ee276ed9d036a4483bb8d1bd6f6d697b161a04276c6ce5f1d3f40d48e04bccb",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6-4.fc32.ppc64le",
    sha256 = "c1126b5d7a02a6327f1d300915f42be3e2135c7cb013860b62854100fb39bd11",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/keyutils-libs-1.6-4.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/keyutils-libs-1.6-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/keyutils-libs-1.6-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/keyutils-libs-1.6-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/c1126b5d7a02a6327f1d300915f42be3e2135c7cb013860b62854100fb39bd11",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6-4.fc32.x86_64",
    sha256 = "ccc3cb2dcb7a534361cc911f27ff4e869902a150b68e236cf6eb209a99d4ee22",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-libs-1.6-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-libs-1.6-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-libs-1.6-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/keyutils-libs-1.6-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ccc3cb2dcb7a534361cc911f27ff4e869902a150b68e236cf6eb209a99d4ee22",
    ],
)

rpm(
    name = "kmod-0__27-1.fc32.x86_64",
    sha256 = "3f9c95f3827b785f49ac4a270d4c3a703dceba673c452838744ec5064cf43cbd",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f9c95f3827b785f49ac4a270d4c3a703dceba673c452838744ec5064cf43cbd",
    ],
)

rpm(
    name = "kmod-libs-0__27-1.fc32.ppc64le",
    sha256 = "b912fd82d8f5bc762f5f99b81fb915d55ae699fca9e05930e01e080126e3ea98",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/kmod-libs-27-1.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/kmod-libs-27-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/kmod-libs-27-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/k/kmod-libs-27-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/b912fd82d8f5bc762f5f99b81fb915d55ae699fca9e05930e01e080126e3ea98",
    ],
)

rpm(
    name = "kmod-libs-0__27-1.fc32.x86_64",
    sha256 = "56187c1c980cc0680f4dbc433ed2c8507e7dc9ab00000615b63ea08c086b7ab2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56187c1c980cc0680f4dbc433ed2c8507e7dc9ab00000615b63ea08c086b7ab2",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-29.fc32.ppc64le",
    sha256 = "5c0e007ca93c468a15c7d906ed08abac72097031ceb63fc4b5bb6cd1490f9ec2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/k/krb5-libs-1.18.2-29.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/k/krb5-libs-1.18.2-29.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/k/krb5-libs-1.18.2-29.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/k/krb5-libs-1.18.2-29.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/5c0e007ca93c468a15c7d906ed08abac72097031ceb63fc4b5bb6cd1490f9ec2",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-29.fc32.x86_64",
    sha256 = "f1ad00906636a2e01b4e978233a9e4a622f4c42f9bc4ec0dd0e294ba75351394",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1ad00906636a2e01b4e978233a9e4a622f4c42f9bc4ec0dd0e294ba75351394",
    ],
)

rpm(
    name = "langpacks-core-font-en-0__3.0-3.fc32.x86_64",
    sha256 = "6923881ce751e903ea2a2e8a58733aad24b33d042b0954fb5c9159360be6d389",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/langpacks-core-font-en-3.0-3.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/langpacks-core-font-en-3.0-3.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/langpacks-core-font-en-3.0-3.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/langpacks-core-font-en-3.0-3.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6923881ce751e903ea2a2e8a58733aad24b33d042b0954fb5c9159360be6d389",
    ],
)

rpm(
    name = "libX11-0__1.6.12-1.fc32.x86_64",
    sha256 = "4f1bc9a0e25664e27144999368545c5c105e9c30b39111b186680a6a99ceb26b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-1.6.12-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-1.6.12-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-1.6.12-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-1.6.12-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4f1bc9a0e25664e27144999368545c5c105e9c30b39111b186680a6a99ceb26b",
    ],
)

rpm(
    name = "libX11-common-0__1.6.12-1.fc32.x86_64",
    sha256 = "84c7f17faabea48121ac787bfce0edcad0e86e55f43351431ba408332f54f21f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-common-1.6.12-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-common-1.6.12-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-common-1.6.12-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-common-1.6.12-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/84c7f17faabea48121ac787bfce0edcad0e86e55f43351431ba408332f54f21f",
    ],
)

rpm(
    name = "libX11-xcb-0__1.6.12-1.fc32.x86_64",
    sha256 = "1e6f4865567365909ab15a9cb6c5926f983808dfddc98d9078f38a818c17d40d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-xcb-1.6.12-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-xcb-1.6.12-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-xcb-1.6.12-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libX11-xcb-1.6.12-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1e6f4865567365909ab15a9cb6c5926f983808dfddc98d9078f38a818c17d40d",
    ],
)

rpm(
    name = "libXau-0__1.0.9-3.fc32.x86_64",
    sha256 = "0f002cc5ef09532fb9ef309c6e24ab064cb1babda1514a1258e214fe9fe60ab1",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXau-1.0.9-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXau-1.0.9-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXau-1.0.9-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXau-1.0.9-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0f002cc5ef09532fb9ef309c6e24ab064cb1babda1514a1258e214fe9fe60ab1",
    ],
)

rpm(
    name = "libXdamage-0__1.1.5-2.fc32.x86_64",
    sha256 = "6d6a67c5991730234401595ea915e275413da0c17dc6ea89a4a602428119e36b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXdamage-1.1.5-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXdamage-1.1.5-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXdamage-1.1.5-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXdamage-1.1.5-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6d6a67c5991730234401595ea915e275413da0c17dc6ea89a4a602428119e36b",
    ],
)

rpm(
    name = "libXext-0__1.3.4-3.fc32.x86_64",
    sha256 = "2ba020fbb3c5aa9079b6e49903e1525ddd0722135a5fc7ce92e7dea2140102da",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXext-1.3.4-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXext-1.3.4-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXext-1.3.4-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXext-1.3.4-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ba020fbb3c5aa9079b6e49903e1525ddd0722135a5fc7ce92e7dea2140102da",
    ],
)

rpm(
    name = "libXfixes-0__5.0.3-11.fc32.x86_64",
    sha256 = "ee57347efdda4eaffdcfca30cb04d3089ab36d270a570ae8132dacdce766f2c4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXfixes-5.0.3-11.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXfixes-5.0.3-11.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXfixes-5.0.3-11.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXfixes-5.0.3-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ee57347efdda4eaffdcfca30cb04d3089ab36d270a570ae8132dacdce766f2c4",
    ],
)

rpm(
    name = "libXft-0__2.3.3-3.fc32.x86_64",
    sha256 = "e958dcf75c7b5d76dc02fea8fe27e4b41292900d15d6584ec8127469e9c9cc45",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXft-2.3.3-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXft-2.3.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXft-2.3.3-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXft-2.3.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e958dcf75c7b5d76dc02fea8fe27e4b41292900d15d6584ec8127469e9c9cc45",
    ],
)

rpm(
    name = "libXrender-0__0.9.10-11.fc32.x86_64",
    sha256 = "656bbeddecee15f2b638da3c5cdfbcf0ec7bff14ea1eb8179b30b6b389ffa4db",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXrender-0.9.10-11.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXrender-0.9.10-11.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXrender-0.9.10-11.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXrender-0.9.10-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/656bbeddecee15f2b638da3c5cdfbcf0ec7bff14ea1eb8179b30b6b389ffa4db",
    ],
)

rpm(
    name = "libXv-0__1.0.11-11.fc32.x86_64",
    sha256 = "28009ee073a5cd0c6182d7ce20fc4ef160f028cd9e34cfd53e91500ebcdb8a08",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXv-1.0.11-11.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXv-1.0.11-11.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXv-1.0.11-11.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXv-1.0.11-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/28009ee073a5cd0c6182d7ce20fc4ef160f028cd9e34cfd53e91500ebcdb8a08",
    ],
)

rpm(
    name = "libXxf86vm-0__1.1.4-13.fc32.x86_64",
    sha256 = "3b4f270b0fdd0250306f15ee8b98ba4add4eb81c2ec4bc392b037d3d74745bb8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXxf86vm-1.1.4-13.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXxf86vm-1.1.4-13.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXxf86vm-1.1.4-13.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libXxf86vm-1.1.4-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b4f270b0fdd0250306f15ee8b98ba4add4eb81c2ec4bc392b037d3d74745bb8",
    ],
)

rpm(
    name = "libacl-0__2.2.53-5.fc32.ppc64le",
    sha256 = "4cbe6a8a483ec8444314c0659de6941974e5460612caba416edc84bda0a648b4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libacl-2.2.53-5.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libacl-2.2.53-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libacl-2.2.53-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libacl-2.2.53-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/4cbe6a8a483ec8444314c0659de6941974e5460612caba416edc84bda0a648b4",
    ],
)

rpm(
    name = "libacl-0__2.2.53-5.fc32.x86_64",
    sha256 = "f826f984b23d0701a1b72de5882b9c0e7bae87ef49d9edfea156654f489f8b2b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f826f984b23d0701a1b72de5882b9c0e7bae87ef49d9edfea156654f489f8b2b",
    ],
)

rpm(
    name = "libaio-0__0.3.111-7.fc32.ppc64le",
    sha256 = "03e5a42873709ae7a6cab5fa2f5f41618b8ec03791ccb355afd57887818902d9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libaio-0.3.111-7.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libaio-0.3.111-7.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libaio-0.3.111-7.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libaio-0.3.111-7.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/03e5a42873709ae7a6cab5fa2f5f41618b8ec03791ccb355afd57887818902d9",
    ],
)

rpm(
    name = "libaio-0__0.3.111-7.fc32.x86_64",
    sha256 = "a410db5c56d4f39f6ea71e7d5bb6d4a2bd518015d1e34f38fbc0d7bbd4e872d4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a410db5c56d4f39f6ea71e7d5bb6d4a2bd518015d1e34f38fbc0d7bbd4e872d4",
    ],
)

rpm(
    name = "libarchive-0__3.4.3-1.fc32.x86_64",
    sha256 = "b4053e72ef23e59a43b4f77b0f3abeb92c4b72c9df7d4f62572560edfbfa02a4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4053e72ef23e59a43b4f77b0f3abeb92c4b72c9df7d4f62572560edfbfa02a4",
    ],
)

rpm(
    name = "libargon2-0__20171227-4.fc32.ppc64le",
    sha256 = "6d66a2ef8e9c13923d1f79c7887c0965ff774f5b2c8af1f90552744770fbc77b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libargon2-20171227-4.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libargon2-20171227-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libargon2-20171227-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libargon2-20171227-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/6d66a2ef8e9c13923d1f79c7887c0965ff774f5b2c8af1f90552744770fbc77b",
    ],
)

rpm(
    name = "libargon2-0__20171227-4.fc32.x86_64",
    sha256 = "7d9bd2fe016ca8860e8fab4a430b3aae4c7b7bea55f8ccd7775ad470172e2886",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d9bd2fe016ca8860e8fab4a430b3aae4c7b7bea55f8ccd7775ad470172e2886",
    ],
)

rpm(
    name = "libattr-0__2.4.48-8.fc32.ppc64le",
    sha256 = "a380dc7607e801b6339848a0ff9afe1c827a62c7db991f8142003f621cbffbd8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libattr-2.4.48-8.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libattr-2.4.48-8.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libattr-2.4.48-8.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libattr-2.4.48-8.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a380dc7607e801b6339848a0ff9afe1c827a62c7db991f8142003f621cbffbd8",
    ],
)

rpm(
    name = "libattr-0__2.4.48-8.fc32.x86_64",
    sha256 = "65e0cfe367ae4d54cf8bf509cb05e063c9eb6f2fea8dadcf746cdd85adc31d88",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65e0cfe367ae4d54cf8bf509cb05e063c9eb6f2fea8dadcf746cdd85adc31d88",
    ],
)

rpm(
    name = "libbasicobjects-0__0.1.1-44.fc32.x86_64",
    sha256 = "26cca773f1f258105f0f825a8e0faebcaa183655b5d91cd972afbe791b586478",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libbasicobjects-0.1.1-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libbasicobjects-0.1.1-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libbasicobjects-0.1.1-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libbasicobjects-0.1.1-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/26cca773f1f258105f0f825a8e0faebcaa183655b5d91cd972afbe791b586478",
    ],
)

rpm(
    name = "libblkid-0__2.35.2-1.fc32.ppc64le",
    sha256 = "932ad2321a66032018811f3981e6bc89a1e14245fb5cb3a8dfdd30dad3655d3a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libblkid-2.35.2-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libblkid-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libblkid-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libblkid-2.35.2-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/932ad2321a66032018811f3981e6bc89a1e14245fb5cb3a8dfdd30dad3655d3a",
    ],
)

rpm(
    name = "libblkid-0__2.35.2-1.fc32.x86_64",
    sha256 = "d43d17930e5fedbbeb2a45bdbfff713485c6cd01ca6cbb9443370192e73daf40",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d43d17930e5fedbbeb2a45bdbfff713485c6cd01ca6cbb9443370192e73daf40",
    ],
)

rpm(
    name = "libcacard-3__2.7.0-4.fc32.x86_64",
    sha256 = "a10a03a79cea36cf52f23120cf9fb040d289dd269903ab606f404cfee65cc429",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcacard-2.7.0-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcacard-2.7.0-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcacard-2.7.0-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcacard-2.7.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a10a03a79cea36cf52f23120cf9fb040d289dd269903ab606f404cfee65cc429",
    ],
)

rpm(
    name = "libcap-0__2.26-7.fc32.ppc64le",
    sha256 = "86ee8a93db4b0b12a29b97adb5a56a526382e3fc4b3ab228a87083c93dec07be",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcap-2.26-7.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcap-2.26-7.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcap-2.26-7.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcap-2.26-7.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/86ee8a93db4b0b12a29b97adb5a56a526382e3fc4b3ab228a87083c93dec07be",
    ],
)

rpm(
    name = "libcap-0__2.26-7.fc32.x86_64",
    sha256 = "1bc0542cf8a3746d0fe25c397a93c8206963f1f287246c6fb864eedfc9ffa4a7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1bc0542cf8a3746d0fe25c397a93c8206963f1f287246c6fb864eedfc9ffa4a7",
    ],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.fc32.ppc64le",
    sha256 = "b0ae618dce32fa69c4a05b2cde88dfff9674d80449174bd8c7a29e038f92c6d3",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcap-ng-0.7.11-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcap-ng-0.7.11-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcap-ng-0.7.11-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcap-ng-0.7.11-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/b0ae618dce32fa69c4a05b2cde88dfff9674d80449174bd8c7a29e038f92c6d3",
    ],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.fc32.x86_64",
    sha256 = "6fc5b00896f95b99a6c9785eedae9e6e522a9340fa0da0b0b1f4665708f0245f",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6fc5b00896f95b99a6c9785eedae9e6e522a9340fa0da0b0b1f4665708f0245f",
    ],
)

rpm(
    name = "libcollection-0__0.7.0-44.fc32.x86_64",
    sha256 = "9d4691918d7add910e8c03d61cccda0316b62d77b30ac2e502cf2e55289d1823",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcollection-0.7.0-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcollection-0.7.0-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcollection-0.7.0-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcollection-0.7.0-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d4691918d7add910e8c03d61cccda0316b62d77b30ac2e502cf2e55289d1823",
    ],
)

rpm(
    name = "libcom_err-0__1.45.5-3.fc32.ppc64le",
    sha256 = "b2291d14ebfa0effe8f3d3b14b0b4965c1825c3ed406073696a3c276a3d0eaa0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcom_err-1.45.5-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcom_err-1.45.5-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcom_err-1.45.5-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libcom_err-1.45.5-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/b2291d14ebfa0effe8f3d3b14b0b4965c1825c3ed406073696a3c276a3d0eaa0",
    ],
)

rpm(
    name = "libcom_err-0__1.45.5-3.fc32.x86_64",
    sha256 = "4494013eac1ad337673f084242aa8ebffb4a149243475b448bee9266401f2896",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4494013eac1ad337673f084242aa8ebffb4a149243475b448bee9266401f2896",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.69.1-7.fc32.ppc64le",
    sha256 = "21b1145d167c9c733b5a8ec6c9109e2d7f91df987c598c98292b46998741b955",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcurl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcurl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcurl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libcurl-minimal-7.69.1-7.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/21b1145d167c9c733b5a8ec6c9109e2d7f91df987c598c98292b46998741b955",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.69.1-7.fc32.x86_64",
    sha256 = "a4670ddc06d9a8ceaf6d47b51ba921199333afbf6bf43e87a71db9a999c8e1b3",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a4670ddc06d9a8ceaf6d47b51ba921199333afbf6bf43e87a71db9a999c8e1b3",
    ],
)

rpm(
    name = "libdatrie-0__0.2.9-11.fc32.x86_64",
    sha256 = "adad80ffa96cb9335f9dd466f673fb74513c651033bbab24af17c382f7dcddcc",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdatrie-0.2.9-11.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdatrie-0.2.9-11.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdatrie-0.2.9-11.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdatrie-0.2.9-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/adad80ffa96cb9335f9dd466f673fb74513c651033bbab24af17c382f7dcddcc",
    ],
)

rpm(
    name = "libdb-0__5.3.28-40.fc32.ppc64le",
    sha256 = "4ae6446eb25acf715a00fde46f17968079fbed0d13df6cc01dd48d8827d599aa",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libdb-5.3.28-40.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libdb-5.3.28-40.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libdb-5.3.28-40.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libdb-5.3.28-40.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/4ae6446eb25acf715a00fde46f17968079fbed0d13df6cc01dd48d8827d599aa",
    ],
)

rpm(
    name = "libdb-0__5.3.28-40.fc32.x86_64",
    sha256 = "688fcc0b7ef3c48cf7d602eefd7fefae7bcad4f0dc71c9fe9432c2ce5bbd9daa",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/688fcc0b7ef3c48cf7d602eefd7fefae7bcad4f0dc71c9fe9432c2ce5bbd9daa",
    ],
)

rpm(
    name = "libdb-utils-0__5.3.28-40.fc32.x86_64",
    sha256 = "431d836b2be015212d8c15b4290d5ce5bb45282cbf3fc52696f632d84ce34dfe",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/431d836b2be015212d8c15b4290d5ce5bb45282cbf3fc52696f632d84ce34dfe",
    ],
)

rpm(
    name = "libdrm-0__2.4.102-1.fc32.x86_64",
    sha256 = "ce3a4ebb0d63bd38268a9ce982b50c8ec67aea6928579943d133e1f9e0a99189",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libdrm-2.4.102-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libdrm-2.4.102-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libdrm-2.4.102-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libdrm-2.4.102-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce3a4ebb0d63bd38268a9ce982b50c8ec67aea6928579943d133e1f9e0a99189",
    ],
)

rpm(
    name = "libedit-0__3.1-32.20191231cvs.fc32.x86_64",
    sha256 = "9a12db30090023c60e3d7bcd5b07142cdc6d84c77e25ddb1cf41a4c490e52f09",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libedit-3.1-32.20191231cvs.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libedit-3.1-32.20191231cvs.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libedit-3.1-32.20191231cvs.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libedit-3.1-32.20191231cvs.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a12db30090023c60e3d7bcd5b07142cdc6d84c77e25ddb1cf41a4c490e52f09",
    ],
)

rpm(
    name = "libepoxy-0__1.5.4-2.fc32.x86_64",
    sha256 = "76c22bfdaab3fe575abe813cbb9a1c3816d8cf50d3c25c9280d07031403ccf30",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/76c22bfdaab3fe575abe813cbb9a1c3816d8cf50d3c25c9280d07031403ccf30",
    ],
)

rpm(
    name = "libev-0__4.31-2.fc32.x86_64",
    sha256 = "765176863b582a8de7db2d076bd7dbcef05c6f9993f44ad9918e0ca1f9507d11",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libev-4.31-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libev-4.31-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libev-4.31-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libev-4.31-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/765176863b582a8de7db2d076bd7dbcef05c6f9993f44ad9918e0ca1f9507d11",
    ],
)

rpm(
    name = "libevent-0__2.1.8-8.fc32.x86_64",
    sha256 = "7bf42ff57ce2a31db0da7d6c5926552f4e51e9f25cded77bd634eb5cd35eadab",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7bf42ff57ce2a31db0da7d6c5926552f4e51e9f25cded77bd634eb5cd35eadab",
    ],
)

rpm(
    name = "libfdisk-0__2.35.2-1.fc32.ppc64le",
    sha256 = "5539e6f1b420c278f4f481b49ae879005fe347bfe381b2f6bd68ee031fcc53b8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libfdisk-2.35.2-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libfdisk-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libfdisk-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libfdisk-2.35.2-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/5539e6f1b420c278f4f481b49ae879005fe347bfe381b2f6bd68ee031fcc53b8",
    ],
)

rpm(
    name = "libfdisk-0__2.35.2-1.fc32.x86_64",
    sha256 = "d7a895002e2291f776c8bf40dc99848105ca8c8e1651ba4692cc44ab838bc0a1",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d7a895002e2291f776c8bf40dc99848105ca8c8e1651ba4692cc44ab838bc0a1",
    ],
)

rpm(
    name = "libffi-0__3.1-24.fc32.ppc64le",
    sha256 = "766226682ef1e67213900ab8d71c4dbe4d954e2057fde0aa10b4791ef4931a2b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libffi-3.1-24.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libffi-3.1-24.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libffi-3.1-24.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libffi-3.1-24.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/766226682ef1e67213900ab8d71c4dbe4d954e2057fde0aa10b4791ef4931a2b",
    ],
)

rpm(
    name = "libffi-0__3.1-24.fc32.x86_64",
    sha256 = "86c87a4169bdf75c6d3a2f11d3a7e20b6364b2db97c74bc7eb62b1b22bc54401",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86c87a4169bdf75c6d3a2f11d3a7e20b6364b2db97c74bc7eb62b1b22bc54401",
    ],
)

rpm(
    name = "libgcc-0__10.2.1-9.fc32.ppc64le",
    sha256 = "0955989b418231a5f48ff9acaf22dd85d30f2c2c5d56461c99bf6a2881045f5b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libgcc-10.2.1-9.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libgcc-10.2.1-9.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libgcc-10.2.1-9.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libgcc-10.2.1-9.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0955989b418231a5f48ff9acaf22dd85d30f2c2c5d56461c99bf6a2881045f5b",
    ],
)

rpm(
    name = "libgcc-0__10.2.1-9.fc32.x86_64",
    sha256 = "cc25429520d34e963a91d47204cfc8112592d5406e4c7a0a883f7201b5f70915",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.2.1-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.2.1-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cc25429520d34e963a91d47204cfc8112592d5406e4c7a0a883f7201b5f70915",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-3.fc32.ppc64le",
    sha256 = "0b106d31cdc58cc060ff1a335c48acde08e2ccd39479f7070ddb0a27af98a945",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgcrypt-1.8.5-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgcrypt-1.8.5-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgcrypt-1.8.5-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgcrypt-1.8.5-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0b106d31cdc58cc060ff1a335c48acde08e2ccd39479f7070ddb0a27af98a945",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-3.fc32.x86_64",
    sha256 = "5f0ae954b5955c86623e68cd81ccf8505a89f260003b8a3be6a93bd76f18452c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5f0ae954b5955c86623e68cd81ccf8505a89f260003b8a3be6a93bd76f18452c",
    ],
)

rpm(
    name = "libglvnd-1__1.3.2-1.fc32.x86_64",
    sha256 = "cc5896d567a39e9ddcf8621f4746e5992795224f1dfe69af17aff9619dbadb1f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-1.3.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cc5896d567a39e9ddcf8621f4746e5992795224f1dfe69af17aff9619dbadb1f",
    ],
)

rpm(
    name = "libglvnd-egl-1__1.3.2-1.fc32.x86_64",
    sha256 = "9e27b2c2ee45d5602c2b31495edc20f475ee9c985a3910d936a4eb6c7a6204f0",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-egl-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-egl-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-egl-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-egl-1.3.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9e27b2c2ee45d5602c2b31495edc20f475ee9c985a3910d936a4eb6c7a6204f0",
    ],
)

rpm(
    name = "libglvnd-gles-1__1.3.2-1.fc32.x86_64",
    sha256 = "1f829a8f335326a49c008febcb984ef2d7d791b16b356419a0d409a8d2ab15f1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-gles-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-gles-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-gles-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-gles-1.3.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f829a8f335326a49c008febcb984ef2d7d791b16b356419a0d409a8d2ab15f1",
    ],
)

rpm(
    name = "libglvnd-glx-1__1.3.2-1.fc32.x86_64",
    sha256 = "d145c5bf5cb40f8c7736fd51931b1bdd192e6c1bf5ba3673f9fdf3311868c37f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-glx-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-glx-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-glx-1.3.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libglvnd-glx-1.3.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d145c5bf5cb40f8c7736fd51931b1bdd192e6c1bf5ba3673f9fdf3311868c37f",
    ],
)

rpm(
    name = "libgomp-0__10.2.1-9.fc32.x86_64",
    sha256 = "9895b6a204d4c4d83ee1cbe5a50b2bd1470daffe38448a09f704c7c6da78d4ef",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.2.1-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9895b6a204d4c4d83ee1cbe5a50b2bd1470daffe38448a09f704c7c6da78d4ef",
    ],
)

rpm(
    name = "libgpg-error-0__1.36-3.fc32.ppc64le",
    sha256 = "647765c89567faed5ac0577e9f5b0ef9aff33e14774a24e124f0d44e7102f303",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgpg-error-1.36-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgpg-error-1.36-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgpg-error-1.36-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libgpg-error-1.36-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/647765c89567faed5ac0577e9f5b0ef9aff33e14774a24e124f0d44e7102f303",
    ],
)

rpm(
    name = "libgpg-error-0__1.36-3.fc32.x86_64",
    sha256 = "9bd5cb588664e8427bc8bebde0cdf5e14315916624ab6b1979dde60f6eae4278",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9bd5cb588664e8427bc8bebde0cdf5e14315916624ab6b1979dde60f6eae4278",
    ],
)

rpm(
    name = "libibumad-0__32.0-1.fc32.x86_64",
    sha256 = "b9fce55848b4e815850746ecbb04292437716f30043a2dd1a13fc13a59088d15",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibumad-32.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibumad-32.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibumad-32.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibumad-32.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b9fce55848b4e815850746ecbb04292437716f30043a2dd1a13fc13a59088d15",
    ],
)

rpm(
    name = "libibverbs-0__32.0-1.fc32.ppc64le",
    sha256 = "08d6d19f13c7ba575063de779fd47b70da6e6d69b986d1b2d7b4cb48aac78123",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libibverbs-32.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libibverbs-32.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libibverbs-32.0-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libibverbs-32.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/08d6d19f13c7ba575063de779fd47b70da6e6d69b986d1b2d7b4cb48aac78123",
    ],
)

rpm(
    name = "libibverbs-0__32.0-1.fc32.x86_64",
    sha256 = "9c2fbc1cd13624d6cadb40c07dfc8ef70658cae0777008113e2a694b94725002",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-32.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-32.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-32.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-32.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9c2fbc1cd13624d6cadb40c07dfc8ef70658cae0777008113e2a694b94725002",
    ],
)

rpm(
    name = "libidn2-0__2.3.0-2.fc32.ppc64le",
    sha256 = "a3b7710531c353d7d192950440fa0bee2fc3b77c4e8a47f9a70e11f233a52173",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libidn2-2.3.0-2.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libidn2-2.3.0-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libidn2-2.3.0-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libidn2-2.3.0-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a3b7710531c353d7d192950440fa0bee2fc3b77c4e8a47f9a70e11f233a52173",
    ],
)

rpm(
    name = "libidn2-0__2.3.0-2.fc32.x86_64",
    sha256 = "20787251df57a108bbf9c40e30f041b71ac36c8a10900fb699e574ee7e259bf2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20787251df57a108bbf9c40e30f041b71ac36c8a10900fb699e574ee7e259bf2",
    ],
)

rpm(
    name = "libini_config-0__1.3.1-44.fc32.x86_64",
    sha256 = "fed6244fce0c0b48933d622dfbb00cf73931b2f746b27675a025a33b591d9820",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libini_config-1.3.1-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libini_config-1.3.1-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libini_config-1.3.1-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libini_config-1.3.1-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fed6244fce0c0b48933d622dfbb00cf73931b2f746b27675a025a33b591d9820",
    ],
)

rpm(
    name = "libiscsi-0__1.18.0-9.fc32.x86_64",
    sha256 = "022eee58fbb40ae59cbe7752df8d99c578603fc12de5407d2b78fd3ba88d8166",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libiscsi-1.18.0-9.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libiscsi-1.18.0-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libiscsi-1.18.0-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libiscsi-1.18.0-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/022eee58fbb40ae59cbe7752df8d99c578603fc12de5407d2b78fd3ba88d8166",
    ],
)

rpm(
    name = "libjpeg-turbo-0__2.0.4-3.fc32.x86_64",
    sha256 = "37c8e4091216342f58b83acf030f94cb776e95d2f0f86b548e33e2c4cba341eb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/37c8e4091216342f58b83acf030f94cb776e95d2f0f86b548e33e2c4cba341eb",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-11.fc32.ppc64le",
    sha256 = "81e40999d6d84df00de60f0ab415980e8ce7bb46c5ae18228a56bd9299a983b7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libmnl-1.0.4-11.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libmnl-1.0.4-11.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libmnl-1.0.4-11.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libmnl-1.0.4-11.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/81e40999d6d84df00de60f0ab415980e8ce7bb46c5ae18228a56bd9299a983b7",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-11.fc32.x86_64",
    sha256 = "1c68255945533ed4e3368125bc46e19f3fe348d7ec507a85a35038dbb976003f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1c68255945533ed4e3368125bc46e19f3fe348d7ec507a85a35038dbb976003f",
    ],
)

rpm(
    name = "libmount-0__2.35.2-1.fc32.ppc64le",
    sha256 = "75e3ebacfee38f86fea41a3287d03134f1246c4c29cedc00a4b41eaf10319f8d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libmount-2.35.2-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libmount-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libmount-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libmount-2.35.2-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/75e3ebacfee38f86fea41a3287d03134f1246c4c29cedc00a4b41eaf10319f8d",
    ],
)

rpm(
    name = "libmount-0__2.35.2-1.fc32.x86_64",
    sha256 = "2c8e76fcc1ad8197ffdb66d06fb498a1129e71e0f7c04a05176867e5788bbf05",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c8e76fcc1ad8197ffdb66d06fb498a1129e71e0f7c04a05176867e5788bbf05",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.7-4.fc32.ppc64le",
    sha256 = "ca4b7eac409c58e7812e7dfa5c45fceb1aeba148e8230fec83ced107e61382a0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ca4b7eac409c58e7812e7dfa5c45fceb1aeba148e8230fec83ced107e61382a0",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.7-4.fc32.x86_64",
    sha256 = "884357540f4be2a74e608e2c7a31f2371ee3b4d29be2fe39a371c0b131d84aa6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/884357540f4be2a74e608e2c7a31f2371ee3b4d29be2fe39a371c0b131d84aa6",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-17.fc32.ppc64le",
    sha256 = "8f0b020d30297c05b38807cdcde056b8d18b7a9093a14c13126d118d97dba248",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnfnetlink-1.0.1-17.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnfnetlink-1.0.1-17.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnfnetlink-1.0.1-17.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnfnetlink-1.0.1-17.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8f0b020d30297c05b38807cdcde056b8d18b7a9093a14c13126d118d97dba248",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-17.fc32.x86_64",
    sha256 = "ec6abd65541b5bded814de19c9d064e6c21e3d8b424dba7cb25b2fdc52d45a2b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ec6abd65541b5bded814de19c9d064e6c21e3d8b424dba7cb25b2fdc52d45a2b",
    ],
)

rpm(
    name = "libnfsidmap-1__2.5.2-1.rc3.fc32.x86_64",
    sha256 = "33715b7cbaae636cb36d83b80bc01cbdf548270f0365c48ad156cdab2593ca57",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnfsidmap-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnfsidmap-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnfsidmap-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnfsidmap-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/33715b7cbaae636cb36d83b80bc01cbdf548270f0365c48ad156cdab2593ca57",
    ],
)

rpm(
    name = "libnftnl-0__1.1.5-2.fc32.x86_64",
    sha256 = "3afab9512fd4d56a13c95b530c805ac8b2bc872572ec5bb435eccdd59fbbc8b6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3afab9512fd4d56a13c95b530c805ac8b2bc872572ec5bb435eccdd59fbbc8b6",
    ],
)

rpm(
    name = "libnghttp2-0__1.41.0-1.fc32.ppc64le",
    sha256 = "b0859c56ac001e74a93487a4a49188bd8c1730d9b7a503d8538000c827864ef2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libnghttp2-1.41.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libnghttp2-1.41.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libnghttp2-1.41.0-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libnghttp2-1.41.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/b0859c56ac001e74a93487a4a49188bd8c1730d9b7a503d8538000c827864ef2",
    ],
)

rpm(
    name = "libnghttp2-0__1.41.0-1.fc32.x86_64",
    sha256 = "a22b0bbe8feeb6bf43b6fb2ebae8c869061df791549f0b958a77cd44cdb05bd3",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a22b0bbe8feeb6bf43b6fb2ebae8c869061df791549f0b958a77cd44cdb05bd3",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-2.fc32.ppc64le",
    sha256 = "ae23b19c0ed73a696e485cbbf2cf6371f4962074704a308489125645e2fe0be2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnl3-3.5.0-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnl3-3.5.0-2.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnl3-3.5.0-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnl3-3.5.0-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ae23b19c0ed73a696e485cbbf2cf6371f4962074704a308489125645e2fe0be2",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-2.fc32.x86_64",
    sha256 = "8dfdbe51193bdcfc3db41b5b9f317f009bfab6373e6ed3c5475466b8772a85e1",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8dfdbe51193bdcfc3db41b5b9f317f009bfab6373e6ed3c5475466b8772a85e1",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-6.20180605git4a062cf.fc32.ppc64le",
    sha256 = "afb8984b65a2d41e08dbe2b349e54a42e76230ea0b96120d892a09575684b0ae",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/afb8984b65a2d41e08dbe2b349e54a42e76230ea0b96120d892a09575684b0ae",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-6.20180605git4a062cf.fc32.x86_64",
    sha256 = "3b4ce7fc4e2778758881feedf6ea19b65e99aa3672e19a7dd62977efe3b910b9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b4ce7fc4e2778758881feedf6ea19b65e99aa3672e19a7dd62977efe3b910b9",
    ],
)

rpm(
    name = "libogg-2__1.3.4-2.fc32.x86_64",
    sha256 = "ce5bbbe570b841c67066a03f9426cad43eefdd20431b80f0760ce418f031c6cc",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libogg-1.3.4-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libogg-1.3.4-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libogg-1.3.4-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libogg-1.3.4-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce5bbbe570b841c67066a03f9426cad43eefdd20431b80f0760ce418f031c6cc",
    ],
)

rpm(
    name = "libpath_utils-0__0.2.1-44.fc32.x86_64",
    sha256 = "ce1a80b0ba6f09a7e5a188436088dd31d10c5f7b43dbe9c1788f705127c8e6b2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpath_utils-0.2.1-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpath_utils-0.2.1-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpath_utils-0.2.1-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpath_utils-0.2.1-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce1a80b0ba6f09a7e5a188436088dd31d10c5f7b43dbe9c1788f705127c8e6b2",
    ],
)

rpm(
    name = "libpcap-14__1.9.1-3.fc32.ppc64le",
    sha256 = "63deebdfa67db81cebb5dc54ff80538f2de1c48b8ad0c579f0bae1e2e20e6298",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libpcap-1.9.1-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libpcap-1.9.1-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libpcap-1.9.1-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libpcap-1.9.1-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/63deebdfa67db81cebb5dc54ff80538f2de1c48b8ad0c579f0bae1e2e20e6298",
    ],
)

rpm(
    name = "libpcap-14__1.9.1-3.fc32.x86_64",
    sha256 = "b3230630a471b806a9153669d187508350cdb2b368a68f8c439c82abad038c3f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpcap-1.9.1-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpcap-1.9.1-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpcap-1.9.1-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpcap-1.9.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3230630a471b806a9153669d187508350cdb2b368a68f8c439c82abad038c3f",
    ],
)

rpm(
    name = "libpciaccess-0__0.16-2.fc32.x86_64",
    sha256 = "4630ea4b3e4c439f2d3bc4cf33e71388e65cd5394338c86fa1074c6f7fa0d801",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4630ea4b3e4c439f2d3bc4cf33e71388e65cd5394338c86fa1074c6f7fa0d801",
    ],
)

rpm(
    name = "libpkgconf-0__1.6.3-3.fc32.x86_64",
    sha256 = "6952dfc6a8f583c9aeafb16d5d34208d7e39fd7ec8628c5aa8ccde039acbe548",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6952dfc6a8f583c9aeafb16d5d34208d7e39fd7ec8628c5aa8ccde039acbe548",
    ],
)

rpm(
    name = "libpmem-0__1.8-2.fc32.x86_64",
    sha256 = "23115d324b418df21e859a11182b3570ce50407eecdeecb17257d72089c96c9f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23115d324b418df21e859a11182b3570ce50407eecdeecb17257d72089c96c9f",
    ],
)

rpm(
    name = "libpng-2__1.6.37-3.fc32.x86_64",
    sha256 = "8e4e38b74567a3b0df7c951a100956fa6b91868bec34717015ae667f6d4cbfcf",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e4e38b74567a3b0df7c951a100956fa6b91868bec34717015ae667f6d4cbfcf",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-1.fc32.ppc64le",
    sha256 = "1516ab47a1281dd2f3209ddfcc01a05932f66f95cf1cbbc6883a3b647d02772b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libpwquality-1.4.4-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libpwquality-1.4.4-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libpwquality-1.4.4-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libpwquality-1.4.4-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/1516ab47a1281dd2f3209ddfcc01a05932f66f95cf1cbbc6883a3b647d02772b",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-1.fc32.x86_64",
    sha256 = "583e4f689dc478f68942fa650e2b495db63bd29d13b3e075a3effcccf29260da",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/583e4f689dc478f68942fa650e2b495db63bd29d13b3e075a3effcccf29260da",
    ],
)

rpm(
    name = "librados2-2__14.2.16-1.fc32.x86_64",
    sha256 = "3aaba70870e20deca4a02163b84546f9a5268d07cc338f060099f1e4553c8d42",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librados2-14.2.16-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librados2-14.2.16-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/librados2-14.2.16-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/librados2-14.2.16-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3aaba70870e20deca4a02163b84546f9a5268d07cc338f060099f1e4553c8d42",
    ],
)

rpm(
    name = "librbd1-2__14.2.16-1.fc32.x86_64",
    sha256 = "20573a696bf613c792d8793a03ff6c8bcc2e66793f348e36d5259c1dc6493a77",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librbd1-14.2.16-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librbd1-14.2.16-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/librbd1-14.2.16-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/librbd1-14.2.16-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20573a696bf613c792d8793a03ff6c8bcc2e66793f348e36d5259c1dc6493a77",
    ],
)

rpm(
    name = "librdmacm-0__32.0-1.fc32.ppc64le",
    sha256 = "07dd938bf69459dbc3a8a6979093ecd2e0ad5e6fa7feb2cf216fa1115048a6f8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/librdmacm-32.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/librdmacm-32.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/librdmacm-32.0-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/librdmacm-32.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/07dd938bf69459dbc3a8a6979093ecd2e0ad5e6fa7feb2cf216fa1115048a6f8",
    ],
)

rpm(
    name = "librdmacm-0__32.0-1.fc32.x86_64",
    sha256 = "7e218af230a7cd2f9ac75faffa16286eb6bc574984d82183416eee68f4ae2f18",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-32.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-32.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-32.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-32.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7e218af230a7cd2f9ac75faffa16286eb6bc574984d82183416eee68f4ae2f18",
    ],
)

rpm(
    name = "libref_array-0__0.1.5-44.fc32.x86_64",
    sha256 = "6fc1181ed798617668f195579acfcf8fb212513a776f376eecfb3d086c66b2d9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libref_array-0.1.5-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libref_array-0.1.5-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libref_array-0.1.5-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libref_array-0.1.5-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6fc1181ed798617668f195579acfcf8fb212513a776f376eecfb3d086c66b2d9",
    ],
)

rpm(
    name = "librtas-0__2.0.2-5.fc32.ppc64le",
    sha256 = "0f4b44e9a95770b17ecceb0d3fac8879246d25541366d41f4f1072a9fe19b5cf",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/librtas-2.0.2-5.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/librtas-2.0.2-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/librtas-2.0.2-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/librtas-2.0.2-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0f4b44e9a95770b17ecceb0d3fac8879246d25541366d41f4f1072a9fe19b5cf",
    ],
)

rpm(
    name = "libseccomp-0__2.5.0-3.fc32.ppc64le",
    sha256 = "a9787055a6be7b2a805620a41e6450ef19927cb804ad507bfbe6046abae029a8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libseccomp-2.5.0-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libseccomp-2.5.0-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libseccomp-2.5.0-3.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libseccomp-2.5.0-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a9787055a6be7b2a805620a41e6450ef19927cb804ad507bfbe6046abae029a8",
    ],
)

rpm(
    name = "libseccomp-0__2.5.0-3.fc32.x86_64",
    sha256 = "7cb644e997c1f247f18ff981a0b03479cc3369871f16199e32da70370ead6faf",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7cb644e997c1f247f18ff981a0b03479cc3369871f16199e32da70370ead6faf",
    ],
)

rpm(
    name = "libselinux-0__3.0-5.fc32.ppc64le",
    sha256 = "27834817f16d4e586018c12c025976d217017ce083da1cc434151fb9d3c2fc2d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libselinux-3.0-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libselinux-3.0-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libselinux-3.0-5.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libselinux-3.0-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/27834817f16d4e586018c12c025976d217017ce083da1cc434151fb9d3c2fc2d",
    ],
)

rpm(
    name = "libselinux-0__3.0-5.fc32.x86_64",
    sha256 = "89a698ab28668b4374abb505de1cc140ffec611014622e8841ecb6fac8c888a3",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89a698ab28668b4374abb505de1cc140ffec611014622e8841ecb6fac8c888a3",
    ],
)

rpm(
    name = "libselinux-utils-0__3.0-5.fc32.x86_64",
    sha256 = "fa31d8d160d66400d6a7d87a3686f70d5a471b70263043ca5c24d96307a82d85",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fa31d8d160d66400d6a7d87a3686f70d5a471b70263043ca5c24d96307a82d85",
    ],
)

rpm(
    name = "libsemanage-0__3.0-3.fc32.ppc64le",
    sha256 = "48077519f89d9548228d850cd0a92d442b4e10e6e954a1e32773ca3e44252032",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsemanage-3.0-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsemanage-3.0-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsemanage-3.0-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsemanage-3.0-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/48077519f89d9548228d850cd0a92d442b4e10e6e954a1e32773ca3e44252032",
    ],
)

rpm(
    name = "libsemanage-0__3.0-3.fc32.x86_64",
    sha256 = "54cb827278ae474cbab1f05e0fbee0355bee2674d46a804f1c2b78ff80a48caa",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54cb827278ae474cbab1f05e0fbee0355bee2674d46a804f1c2b78ff80a48caa",
    ],
)

rpm(
    name = "libsepol-0__3.0-4.fc32.ppc64le",
    sha256 = "9c1fa51330e83fae0533f0829ee79c0832e3d756146b0583ac0582b9deb31f68",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsepol-3.0-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsepol-3.0-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsepol-3.0-4.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsepol-3.0-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/9c1fa51330e83fae0533f0829ee79c0832e3d756146b0583ac0582b9deb31f68",
    ],
)

rpm(
    name = "libsepol-0__3.0-4.fc32.x86_64",
    sha256 = "bcf4ca8e5e1d71a12c5e4d966c248b53ef0300a794ca607b9072145f4212e7a1",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bcf4ca8e5e1d71a12c5e4d966c248b53ef0300a794ca607b9072145f4212e7a1",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-10.fc32.ppc64le",
    sha256 = "ae500cc3eea78e6ae0f1dc883c6c136c00b45d0659a66d653034c8797afd5a1e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsigsegv-2.11-10.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsigsegv-2.11-10.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsigsegv-2.11-10.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libsigsegv-2.11-10.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ae500cc3eea78e6ae0f1dc883c6c136c00b45d0659a66d653034c8797afd5a1e",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-10.fc32.x86_64",
    sha256 = "942707884401498938fba6e2439dc923d4e2d81f4bac205f4e73d458e9879927",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/942707884401498938fba6e2439dc923d4e2d81f4bac205f4e73d458e9879927",
    ],
)

rpm(
    name = "libsmartcols-0__2.35.2-1.fc32.ppc64le",
    sha256 = "e99eff1b254b71b5c74f7dddf2691b53a9219446d3beee0e3187957177a106c9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsmartcols-2.35.2-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsmartcols-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsmartcols-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libsmartcols-2.35.2-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/e99eff1b254b71b5c74f7dddf2691b53a9219446d3beee0e3187957177a106c9",
    ],
)

rpm(
    name = "libsmartcols-0__2.35.2-1.fc32.x86_64",
    sha256 = "82a0c6703444fa28ab032b3e4aa355deabff92f3f39d5490faa5c9b9150eaceb",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82a0c6703444fa28ab032b3e4aa355deabff92f3f39d5490faa5c9b9150eaceb",
    ],
)

rpm(
    name = "libss-0__1.45.5-3.fc32.ppc64le",
    sha256 = "19070be3ad2825a07c417640df775b323f3f3456f01dd0e6d7c04012fc603f81",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libss-1.45.5-3.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libss-1.45.5-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libss-1.45.5-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libss-1.45.5-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/19070be3ad2825a07c417640df775b323f3f3456f01dd0e6d7c04012fc603f81",
    ],
)

rpm(
    name = "libss-0__1.45.5-3.fc32.x86_64",
    sha256 = "27701cda24f5f6386e0173745aabc4f6df28052975e73529854432c35399cfc8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/27701cda24f5f6386e0173745aabc4f6df28052975e73529854432c35399cfc8",
    ],
)

rpm(
    name = "libssh-0__0.9.5-1.fc32.x86_64",
    sha256 = "1f92b99b1ea563cf2642ce9b9c831d3f1fb8547922d6ca0d728de2c0d751c4e9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f92b99b1ea563cf2642ce9b9c831d3f1fb8547922d6ca0d728de2c0d751c4e9",
    ],
)

rpm(
    name = "libssh-config-0__0.9.5-1.fc32.x86_64",
    sha256 = "a11725cb639c2dc043ae096c450cb20b3a10116e55e2ec4eee80d8f111037428",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a11725cb639c2dc043ae096c450cb20b3a10116e55e2ec4eee80d8f111037428",
    ],
)

rpm(
    name = "libssh2-0__1.9.0-5.fc32.x86_64",
    sha256 = "2c811783245ad49873157c16b26cd79efc62afccaaf5077ae2821b8afad175b8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c811783245ad49873157c16b26cd79efc62afccaaf5077ae2821b8afad175b8",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__10.2.1-9.fc32.ppc64le",
    sha256 = "3b2def16c868c45a6c455bafa54ddb826201510cfecffa0d0ee03c379e74db7f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libstdc++-10.2.1-9.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libstdc++-10.2.1-9.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libstdc++-10.2.1-9.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libstdc++-10.2.1-9.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3b2def16c868c45a6c455bafa54ddb826201510cfecffa0d0ee03c379e74db7f",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__10.2.1-9.fc32.x86_64",
    sha256 = "05c67ab4e848cbf57040a693580c9303a2154281b2e3a91fc13e26280369dc81",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.2.1-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.2.1-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.2.1-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/05c67ab4e848cbf57040a693580c9303a2154281b2e3a91fc13e26280369dc81",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-1.fc32.ppc64le",
    sha256 = "87e1164f14eb31656d8695c57a2ceac1d02b49e2ccba6c3a0abd10b59780fb34",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libtasn1-4.16.0-1.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libtasn1-4.16.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libtasn1-4.16.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libtasn1-4.16.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/87e1164f14eb31656d8695c57a2ceac1d02b49e2ccba6c3a0abd10b59780fb34",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-1.fc32.x86_64",
    sha256 = "052d04c9a6697c6e5aa546546ae5058d547fc4a4f474d2805a3e45dbf69193c6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/052d04c9a6697c6e5aa546546ae5058d547fc4a4f474d2805a3e45dbf69193c6",
    ],
)

rpm(
    name = "libtextstyle-0__0.21-1.fc32.x86_64",
    sha256 = "2ee0dd5aa79c4327da8616903ff4d5956b687b3e28d2f45b7ab219fbd1326b0b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ee0dd5aa79c4327da8616903ff4d5956b687b3e28d2f45b7ab219fbd1326b0b",
    ],
)

rpm(
    name = "libthai-0__0.1.28-4.fc32.x86_64",
    sha256 = "6b9e2f0a21bafd58b7e62cac2e494ff15bdf83f1bd308238d9ec49440f1ecc06",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libthai-0.1.28-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libthai-0.1.28-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libthai-0.1.28-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libthai-0.1.28-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6b9e2f0a21bafd58b7e62cac2e494ff15bdf83f1bd308238d9ec49440f1ecc06",
    ],
)

rpm(
    name = "libtheora-1__1.1.1-25.fc32.x86_64",
    sha256 = "74193a7d5cc52fe6f77cf9282d70ea046eec8e3e40a331f956f721d3a71ceae9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtheora-1.1.1-25.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtheora-1.1.1-25.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtheora-1.1.1-25.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtheora-1.1.1-25.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/74193a7d5cc52fe6f77cf9282d70ea046eec8e3e40a331f956f721d3a71ceae9",
    ],
)

rpm(
    name = "libtirpc-0__1.2.6-1.rc4.fc32.ppc64le",
    sha256 = "030b02bffdb21fb166e9ab08d5a2e23842f94fccab5fca489467e1915c4fc4f7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libtirpc-1.2.6-1.rc4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libtirpc-1.2.6-1.rc4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libtirpc-1.2.6-1.rc4.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libtirpc-1.2.6-1.rc4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/030b02bffdb21fb166e9ab08d5a2e23842f94fccab5fca489467e1915c4fc4f7",
    ],
)

rpm(
    name = "libtirpc-0__1.2.6-1.rc4.fc32.x86_64",
    sha256 = "84c6b2d0dbb6181611816f642725005992522009993716482a3037294ef22954",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84c6b2d0dbb6181611816f642725005992522009993716482a3037294ef22954",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-7.fc32.ppc64le",
    sha256 = "96f495c08c4fd0a3dd0b098a31549a49f680c291597e336a9718d60143627ed6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libunistring-0.9.10-7.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libunistring-0.9.10-7.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libunistring-0.9.10-7.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libunistring-0.9.10-7.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/96f495c08c4fd0a3dd0b098a31549a49f680c291597e336a9718d60143627ed6",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-7.fc32.x86_64",
    sha256 = "fb06aa3d8059406a23694ddafe0ef340ca627dd68bf3f351f094de58ef30fb2c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb06aa3d8059406a23694ddafe0ef340ca627dd68bf3f351f094de58ef30fb2c",
    ],
)

rpm(
    name = "libunwind-0__1.3.1-7.fc32.x86_64",
    sha256 = "b5e581f7a60b4b4164b700bf3ba47c6de1fb74ef6102687c418c56b29b861e34",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b5e581f7a60b4b4164b700bf3ba47c6de1fb74ef6102687c418c56b29b861e34",
    ],
)

rpm(
    name = "libusal-0__1.1.11-44.fc32.x86_64",
    sha256 = "7c9f879b67d1d5a6f40fa551b75f69ac744e5d0d795154655b8d55ef15512af0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7c9f879b67d1d5a6f40fa551b75f69ac744e5d0d795154655b8d55ef15512af0",
    ],
)

rpm(
    name = "libusbx-0__1.0.23-1.fc32.x86_64",
    sha256 = "729fb595040f1e2e71ff0a8c1c22ebe4b7187f78b816af8e6a8d93c03fc5d844",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusbx-1.0.23-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusbx-1.0.23-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusbx-1.0.23-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusbx-1.0.23-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/729fb595040f1e2e71ff0a8c1c22ebe4b7187f78b816af8e6a8d93c03fc5d844",
    ],
)

rpm(
    name = "libutempter-0__1.1.6-18.fc32.ppc64le",
    sha256 = "b47100effff994060a88c7509949df3e7949efb486a98c417d0b21b82e838c90",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libutempter-1.1.6-18.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libutempter-1.1.6-18.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libutempter-1.1.6-18.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libutempter-1.1.6-18.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/b47100effff994060a88c7509949df3e7949efb486a98c417d0b21b82e838c90",
    ],
)

rpm(
    name = "libutempter-0__1.1.6-18.fc32.x86_64",
    sha256 = "f9ccea65ecf98f4dfac65d25986d08efa62a1d1c0db9db0a061e7408d6805a1a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f9ccea65ecf98f4dfac65d25986d08efa62a1d1c0db9db0a061e7408d6805a1a",
    ],
)

rpm(
    name = "libuuid-0__2.35.2-1.fc32.ppc64le",
    sha256 = "d1fc5c9b3128a03c4fc8ac3b5025f53bf5e1a2fa1b8eb2aa9fb396d350c556bd",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libuuid-2.35.2-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libuuid-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libuuid-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libuuid-2.35.2-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d1fc5c9b3128a03c4fc8ac3b5025f53bf5e1a2fa1b8eb2aa9fb396d350c556bd",
    ],
)

rpm(
    name = "libuuid-0__2.35.2-1.fc32.x86_64",
    sha256 = "20ad2f907034a1c3e76dd4691886223bf588ff946fd57545ecdfcd58bc4c3b4b",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20ad2f907034a1c3e76dd4691886223bf588ff946fd57545ecdfcd58bc4c3b4b",
    ],
)

rpm(
    name = "libverto-0__0.3.0-9.fc32.ppc64le",
    sha256 = "59c5db787decda7dfffd534787a061dfbffbd1b0972dae368fb288ffb4b3108d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libverto-0.3.0-9.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libverto-0.3.0-9.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libverto-0.3.0-9.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/libverto-0.3.0-9.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/59c5db787decda7dfffd534787a061dfbffbd1b0972dae368fb288ffb4b3108d",
    ],
)

rpm(
    name = "libverto-0__0.3.0-9.fc32.x86_64",
    sha256 = "ed84414c9b2190d3026f58db78dffd8bc3a9ad40311cb0adb8ff8e3c7c06ca60",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed84414c9b2190d3026f58db78dffd8bc3a9ad40311cb0adb8ff8e3c7c06ca60",
    ],
)

rpm(
    name = "libverto-libev-0__0.3.0-9.fc32.x86_64",
    sha256 = "ce35e4c3f232c62b75288a7dd7ada055d5742f25ac9411a2d0877a5b93e8ae58",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-libev-0.3.0-9.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-libev-0.3.0-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-libev-0.3.0-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-libev-0.3.0-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ce35e4c3f232c62b75288a7dd7ada055d5742f25ac9411a2d0877a5b93e8ae58",
    ],
)

rpm(
    name = "libvirt-bash-completion-0__6.6.0-13.fc32.x86_64",
    sha256 = "0ce8da7305bc3106abe5a3f183abbffe744e72639fd06a3e5236866531f5b872",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-bash-completion-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0ce8da7305bc3106abe5a3f183abbffe744e72639fd06a3e5236866531f5b872",
    ],
)

rpm(
    name = "libvirt-client-0__6.6.0-13.fc32.x86_64",
    sha256 = "afa500bbffb08635b40b50a75962bee754ab1ebc88f5f500eb98a74462e1e073",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-client-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/afa500bbffb08635b40b50a75962bee754ab1ebc88f5f500eb98a74462e1e073",
    ],
)

rpm(
    name = "libvirt-daemon-0__6.6.0-13.fc32.x86_64",
    sha256 = "274b926ae5b90f738c4a173a31ec9b2b2e121a41a2818b9108200cf104fc4d99",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-daemon-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/274b926ae5b90f738c4a173a31ec9b2b2e121a41a2818b9108200cf104fc4d99",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__6.6.0-13.fc32.x86_64",
    sha256 = "df08b0255cf659360cdc48fe94bb40f2bdcbf5e612fce7a5cd6deeac3cd9a814",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-daemon-driver-qemu-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df08b0255cf659360cdc48fe94bb40f2bdcbf5e612fce7a5cd6deeac3cd9a814",
    ],
)

rpm(
    name = "libvirt-daemon-driver-storage-core-0__6.6.0-13.fc32.x86_64",
    sha256 = "a47eee27df1cf3bff97e9593fd058024f8924721782e6d3baa1fa1dc914b35fe",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-daemon-driver-storage-core-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a47eee27df1cf3bff97e9593fd058024f8924721782e6d3baa1fa1dc914b35fe",
    ],
)

rpm(
    name = "libvirt-devel-0__6.6.0-13.fc32.x86_64",
    sha256 = "b76e99071145bac819bac28be726c89ccc6e81556a4553147154cefd068a0932",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-devel-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b76e99071145bac819bac28be726c89ccc6e81556a4553147154cefd068a0932",
    ],
)

rpm(
    name = "libvirt-libs-0__6.6.0-13.fc32.x86_64",
    sha256 = "e1c80ac577ff9bf07ed5f4b45bcc195d67ed717f15056807e331c9b31a40c14b",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-6.6.0-13.el8/fedora-32-x86_64/01890529-libvirt/libvirt-libs-6.6.0-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e1c80ac577ff9bf07ed5f4b45bcc195d67ed717f15056807e331c9b31a40c14b",
    ],
)

rpm(
    name = "libvisual-1__0.4.0-28.fc32.x86_64",
    sha256 = "65dbb4bc09a7228504b946cc1444c9f50e6eb7b654264df00dc991deb38dbcdb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvisual-0.4.0-28.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvisual-0.4.0-28.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvisual-0.4.0-28.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvisual-0.4.0-28.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65dbb4bc09a7228504b946cc1444c9f50e6eb7b654264df00dc991deb38dbcdb",
    ],
)

rpm(
    name = "libvorbis-1__1.3.6-6.fc32.x86_64",
    sha256 = "b6b2622496fa2f49b720bcd63990e83130d656b801b8d1039b95a52e0752731f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvorbis-1.3.6-6.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvorbis-1.3.6-6.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvorbis-1.3.6-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libvorbis-1.3.6-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b6b2622496fa2f49b720bcd63990e83130d656b801b8d1039b95a52e0752731f",
    ],
)

rpm(
    name = "libwayland-client-0__1.18.0-1.fc32.x86_64",
    sha256 = "94834f7101758c47f3b13209d4330dec69febb15504a3f254aa2618eefcbf1e0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-client-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-client-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-client-1.18.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-client-1.18.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/94834f7101758c47f3b13209d4330dec69febb15504a3f254aa2618eefcbf1e0",
    ],
)

rpm(
    name = "libwayland-cursor-0__1.18.0-1.fc32.x86_64",
    sha256 = "9a923d0eab654a5de3e8dbb4e0480d7e8e85b8b003585e98897569d21dc9d2ab",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-cursor-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-cursor-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-cursor-1.18.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-cursor-1.18.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a923d0eab654a5de3e8dbb4e0480d7e8e85b8b003585e98897569d21dc9d2ab",
    ],
)

rpm(
    name = "libwayland-egl-0__1.18.0-1.fc32.x86_64",
    sha256 = "f13e026e9ccfe312611fecea525d11726676052a1af91c7dd98261c1494cf2de",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-egl-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-egl-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-egl-1.18.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-egl-1.18.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f13e026e9ccfe312611fecea525d11726676052a1af91c7dd98261c1494cf2de",
    ],
)

rpm(
    name = "libwayland-server-0__1.18.0-1.fc32.x86_64",
    sha256 = "29dc93dd98ad04291c586bb372e7277b1abb404c9db70e7e233a1e8b82bba153",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-server-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-server-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-server-1.18.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwayland-server-1.18.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/29dc93dd98ad04291c586bb372e7277b1abb404c9db70e7e233a1e8b82bba153",
    ],
)

rpm(
    name = "libwsman1-0__2.6.8-12.fc32.x86_64",
    sha256 = "01e436bc768c2aa1e3ecac82ab5d3f9efc3d9b49b1879202182054abfcf8a618",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/01e436bc768c2aa1e3ecac82ab5d3f9efc3d9b49b1879202182054abfcf8a618",
    ],
)

rpm(
    name = "libxcb-0__1.13.1-4.fc32.x86_64",
    sha256 = "a067d65415265d19aa2d0adcbd9234029eda8245a8c58262b3ced83edc6ddf38",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxcb-1.13.1-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxcb-1.13.1-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxcb-1.13.1-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxcb-1.13.1-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a067d65415265d19aa2d0adcbd9234029eda8245a8c58262b3ced83edc6ddf38",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.17-1.fc32.ppc64le",
    sha256 = "91fe383f5a7a98a205b0483070e9496ee39f4ebe8cbe24ceace4200867638c49",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxcrypt-4.4.17-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxcrypt-4.4.17-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxcrypt-4.4.17-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxcrypt-4.4.17-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/91fe383f5a7a98a205b0483070e9496ee39f4ebe8cbe24ceace4200867638c49",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.17-1.fc32.x86_64",
    sha256 = "cef23f715be31a5f22f31c1485fbc2669b69a683c4e6bb3047f0479d8f3f50b2",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.17-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.17-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.17-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.17-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cef23f715be31a5f22f31c1485fbc2669b69a683c4e6bb3047f0479d8f3f50b2",
    ],
)

rpm(
    name = "libxkbcommon-0__0.10.0-2.fc32.x86_64",
    sha256 = "ae219ad5ecc0233271c3fd61263f817c646eecece19a8f075e7aa4dd9ff8698e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ae219ad5ecc0233271c3fd61263f817c646eecece19a8f075e7aa4dd9ff8698e",
    ],
)

rpm(
    name = "libxml2-0__2.9.10-8.fc32.ppc64le",
    sha256 = "265f887432fd69b340a13baa8e3b56830a548632e7f72c8d0ce4cffe587b4a79",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxml2-2.9.10-8.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxml2-2.9.10-8.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxml2-2.9.10-8.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libxml2-2.9.10-8.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/265f887432fd69b340a13baa8e3b56830a548632e7f72c8d0ce4cffe587b4a79",
    ],
)

rpm(
    name = "libxml2-0__2.9.10-8.fc32.x86_64",
    sha256 = "60f2deeac94c8d58b305a8faea0701a3fe5dd74909953bf8fe5e9c26169facd1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/60f2deeac94c8d58b305a8faea0701a3fe5dd74909953bf8fe5e9c26169facd1",
    ],
)

rpm(
    name = "libxshmfence-0__1.3-6.fc32.x86_64",
    sha256 = "f0e4157fcaa6a93aae86142d4fc566d33cbb210458eeb57e1efabde160bd31e0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxshmfence-1.3-6.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxshmfence-1.3-6.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxshmfence-1.3-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxshmfence-1.3-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f0e4157fcaa6a93aae86142d4fc566d33cbb210458eeb57e1efabde160bd31e0",
    ],
)

rpm(
    name = "libzstd-0__1.4.7-1.fc32.ppc64le",
    sha256 = "c41feb1f6593db8a6b80215eab924dc12c08915da3011fe2729d435d50502b94",
    urls = [
        "https://fr2.rpmfind.net/linux/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libzstd-1.4.7-1.fc32.ppc64le.rpm",
        "https://fedora-secondary.mirrors.theom.nz/updates/32/Everything/ppc64le/Packages/l/libzstd-1.4.7-1.fc32.ppc64le.rpm",
        "https://download-ib01.fedoraproject.org/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libzstd-1.4.7-1.fc32.ppc64le.rpm",
        "https://mirror1.cl.netactuate.com/fedora-secondary/updates/32/Everything/ppc64le/Packages/l/libzstd-1.4.7-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/c41feb1f6593db8a6b80215eab924dc12c08915da3011fe2729d435d50502b94",
    ],
)

rpm(
    name = "libzstd-0__1.4.7-1.fc32.x86_64",
    sha256 = "2dd84d0037419612e9007ee329ff689a388a32213410fcce975a283035dccf77",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.7-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.7-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.7-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.7-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2dd84d0037419612e9007ee329ff689a388a32213410fcce975a283035dccf77",
    ],
)

rpm(
    name = "linux-atm-libs-0__2.5.1-26.fc32.x86_64",
    sha256 = "c9ba05cb46a9cb52e3325ca20c457a377361abcd0e5a7dda776ba19481770467",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c9ba05cb46a9cb52e3325ca20c457a377361abcd0e5a7dda776ba19481770467",
    ],
)

rpm(
    name = "llvm-libs-0__10.0.1-4.fc32.x86_64",
    sha256 = "2553c52dfcf5b6b8edabb89bfd3af8c4a7cecb81169a48cd7af9460ea20e108f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/llvm-libs-10.0.1-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/llvm-libs-10.0.1-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/llvm-libs-10.0.1-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/llvm-libs-10.0.1-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2553c52dfcf5b6b8edabb89bfd3af8c4a7cecb81169a48cd7af9460ea20e108f",
    ],
)

rpm(
    name = "lsof-0__4.93.2-3.fc32.ppc64le",
    sha256 = "cb7a1938aa44d8d097945e7950e9e694ca543e39fb9e3384d597f5d37f206508",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lsof-4.93.2-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lsof-4.93.2-3.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lsof-4.93.2-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lsof-4.93.2-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cb7a1938aa44d8d097945e7950e9e694ca543e39fb9e3384d597f5d37f206508",
    ],
)

rpm(
    name = "lsof-0__4.93.2-3.fc32.x86_64",
    sha256 = "465b7317f0a979c92d76713fbe61761ee3e2afd1a59c01c6d3c6323767e1a115",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/465b7317f0a979c92d76713fbe61761ee3e2afd1a59c01c6d3c6323767e1a115",
    ],
)

rpm(
    name = "lttng-ust-0__2.11.0-4.fc32.x86_64",
    sha256 = "57e332e2a794bc5fa36a45327a46145c553012d9f502e1821c9ffd74514c9bde",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lttng-ust-2.11.0-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lttng-ust-2.11.0-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lttng-ust-2.11.0-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lttng-ust-2.11.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/57e332e2a794bc5fa36a45327a46145c553012d9f502e1821c9ffd74514c9bde",
    ],
)

rpm(
    name = "lua-libs-0__5.3.5-8.fc32.x86_64",
    sha256 = "09c4524a71762b4ba0aa95e296059130e1ef4007963c2b4d2803ea4eb5fdfed4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09c4524a71762b4ba0aa95e296059130e1ef4007963c2b4d2803ea4eb5fdfed4",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.1-2.fc32.ppc64le",
    sha256 = "a6ce9b3d585c7ba81a6f1d6919a8448d1e669f131113361285c2e9c348e39c42",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lz4-libs-1.9.1-2.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lz4-libs-1.9.1-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lz4-libs-1.9.1-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/l/lz4-libs-1.9.1-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a6ce9b3d585c7ba81a6f1d6919a8448d1e669f131113361285c2e9c348e39c42",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.1-2.fc32.x86_64",
    sha256 = "44cfb58b368fba586981aa838a7f3974ac1d66d2b3b695f88d7b1d2e9c81a0b6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/44cfb58b368fba586981aa838a7f3974ac1d66d2b3b695f88d7b1d2e9c81a0b6",
    ],
)

rpm(
    name = "lzo-0__2.10-2.fc32.x86_64",
    sha256 = "4375c398dff722a29bd1700bc8dc8b528345412d1e17d8d9d1176d9774962957",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4375c398dff722a29bd1700bc8dc8b528345412d1e17d8d9d1176d9774962957",
    ],
)

rpm(
    name = "lzop-0__1.04-3.fc32.x86_64",
    sha256 = "dd0aec170afc0e2113845e9d107a58d72b234414b548880e3154be49ffbaf64a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd0aec170afc0e2113845e9d107a58d72b234414b548880e3154be49ffbaf64a",
    ],
)

rpm(
    name = "mesa-dri-drivers-0__20.2.3-1.fc32.x86_64",
    sha256 = "b8b27ae4f693072199a30d571a3055785fdd681996b67494249a087af6b30b01",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-dri-drivers-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-dri-drivers-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-dri-drivers-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-dri-drivers-20.2.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b8b27ae4f693072199a30d571a3055785fdd681996b67494249a087af6b30b01",
    ],
)

rpm(
    name = "mesa-filesystem-0__20.2.3-1.fc32.x86_64",
    sha256 = "b189c84faa0ee7f1c5ab5404914113b138475c46d5a068afc9bb431944aea1ea",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-filesystem-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-filesystem-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-filesystem-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-filesystem-20.2.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b189c84faa0ee7f1c5ab5404914113b138475c46d5a068afc9bb431944aea1ea",
    ],
)

rpm(
    name = "mesa-libEGL-0__20.2.3-1.fc32.x86_64",
    sha256 = "89e5cb4f6a5a413a01211cd3fa9acf0661113a2c77a1ab5ab86a052b9a361215",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libEGL-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libEGL-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libEGL-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libEGL-20.2.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89e5cb4f6a5a413a01211cd3fa9acf0661113a2c77a1ab5ab86a052b9a361215",
    ],
)

rpm(
    name = "mesa-libGL-0__20.2.3-1.fc32.x86_64",
    sha256 = "791b549e19e7e004fb3af9353d95bd1283444c572d6f81416614b528c6207577",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libGL-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libGL-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libGL-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libGL-20.2.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/791b549e19e7e004fb3af9353d95bd1283444c572d6f81416614b528c6207577",
    ],
)

rpm(
    name = "mesa-libgbm-0__20.2.3-1.fc32.x86_64",
    sha256 = "b73a5c920123fb6ce9f8ca37e2878d8c27459a499997e2e9dd44323e090fb739",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libgbm-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libgbm-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libgbm-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libgbm-20.2.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b73a5c920123fb6ce9f8ca37e2878d8c27459a499997e2e9dd44323e090fb739",
    ],
)

rpm(
    name = "mesa-libglapi-0__20.2.3-1.fc32.x86_64",
    sha256 = "3d6a0c966d75c45ae3a158cd2a356a2ea10c746638d3a551dccf0da2c08517be",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libglapi-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libglapi-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libglapi-20.2.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mesa-libglapi-20.2.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3d6a0c966d75c45ae3a158cd2a356a2ea10c746638d3a551dccf0da2c08517be",
    ],
)

rpm(
    name = "mozjs60-0__60.9.0-5.fc32.x86_64",
    sha256 = "80cf220a3314f965c088e03d2b750426767db0b36b6b7c5e8059b9217ff4de6d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/m/mozjs60-60.9.0-5.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/m/mozjs60-60.9.0-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/m/mozjs60-60.9.0-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/m/mozjs60-60.9.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/80cf220a3314f965c088e03d2b750426767db0b36b6b7c5e8059b9217ff4de6d",
    ],
)

rpm(
    name = "mpfr-0__4.0.2-5.fc32.ppc64le",
    sha256 = "8fc2f7280385773b428b3c2d94178567bd8a69c5a4e1cb2fa98fc94c76ea01e8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/m/mpfr-4.0.2-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/m/mpfr-4.0.2-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/m/mpfr-4.0.2-5.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/m/mpfr-4.0.2-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8fc2f7280385773b428b3c2d94178567bd8a69c5a4e1cb2fa98fc94c76ea01e8",
    ],
)

rpm(
    name = "mpfr-0__4.0.2-5.fc32.x86_64",
    sha256 = "6a97b2d7b510dba87d67436c097dde860dcca5a3464c9b3489ec65fcfe101f22",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a97b2d7b510dba87d67436c097dde860dcca5a3464c9b3489ec65fcfe101f22",
    ],
)

rpm(
    name = "ncurses-0__6.1-15.20191109.fc32.ppc64le",
    sha256 = "4eeec77631c4997b8d42eb532845561a21ebad3ad1e86cfee75a019e3569650e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/4eeec77631c4997b8d42eb532845561a21ebad3ad1e86cfee75a019e3569650e",
    ],
)

rpm(
    name = "ncurses-0__6.1-15.20191109.fc32.x86_64",
    sha256 = "b2e862283ac97b1d8b1ede2034ead452ac7dc4ff308593306275b1b0ae5b4102",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2e862283ac97b1d8b1ede2034ead452ac7dc4ff308593306275b1b0ae5b4102",
    ],
)

rpm(
    name = "ncurses-base-0__6.1-15.20191109.fc32.ppc64le",
    sha256 = "25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    ],
)

rpm(
    name = "ncurses-base-0__6.1-15.20191109.fc32.x86_64",
    sha256 = "25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    ],
)

rpm(
    name = "ncurses-libs-0__6.1-15.20191109.fc32.ppc64le",
    sha256 = "d86c6934a7177c85408e89c0af80b78a2198644530ef0bd077de2bd5bbe94793",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d86c6934a7177c85408e89c0af80b78a2198644530ef0bd077de2bd5bbe94793",
    ],
)

rpm(
    name = "ncurses-libs-0__6.1-15.20191109.fc32.x86_64",
    sha256 = "04152a3a608d022a58830c0e3dac0818e2c060469b0f41d8d731f659981a4464",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/04152a3a608d022a58830c0e3dac0818e2c060469b0f41d8d731f659981a4464",
    ],
)

rpm(
    name = "nettle-0__3.5.1-5.fc32.ppc64le",
    sha256 = "27de3436d048d1f8685d9c13f53066649275a779e6c7cae82564ea4f9e249f19",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nettle-3.5.1-5.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nettle-3.5.1-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nettle-3.5.1-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nettle-3.5.1-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/27de3436d048d1f8685d9c13f53066649275a779e6c7cae82564ea4f9e249f19",
    ],
)

rpm(
    name = "nettle-0__3.5.1-5.fc32.x86_64",
    sha256 = "c019d23ed2cb3ceb0ac9757a72c3e8b1d31f2a524b889e18049cc7d923bc9466",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c019d23ed2cb3ceb0ac9757a72c3e8b1d31f2a524b889e18049cc7d923bc9466",
    ],
)

rpm(
    name = "nfs-utils-1__2.5.2-1.rc3.fc32.x86_64",
    sha256 = "c26af91ffb94d1776ca9064279ec1ae4a07cd2f65eaaa1f4445de05bf474803c",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nfs-utils-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nfs-utils-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nfs-utils-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/n/nfs-utils-2.5.2-1.rc3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c26af91ffb94d1776ca9064279ec1ae4a07cd2f65eaaa1f4445de05bf474803c",
    ],
)

rpm(
    name = "nftables-1__0.9.3-3.fc32.x86_64",
    sha256 = "f844f75efe3aca759eb80d426f6530ab03de0467665239bd21e5129ced6bae1c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f844f75efe3aca759eb80d426f6530ab03de0467665239bd21e5129ced6bae1c",
    ],
)

rpm(
    name = "nginx-1__1.18.0-1.fc32.ppc64le",
    sha256 = "2f707111d2d0db5e29b3a767cb020a97d7e533f874778675bdabebc4b7e95de1",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-1.18.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-1.18.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-1.18.0-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-1.18.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/2f707111d2d0db5e29b3a767cb020a97d7e533f874778675bdabebc4b7e95de1",
    ],
)

rpm(
    name = "nginx-1__1.18.0-1.fc32.x86_64",
    sha256 = "b31bb2d93bffcd0429aeda63d55906b3db7d3621cea21513f8d57a1b0abbd408",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.18.0-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.18.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.18.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b31bb2d93bffcd0429aeda63d55906b3db7d3621cea21513f8d57a1b0abbd408",
    ],
)

rpm(
    name = "nginx-filesystem-1__1.18.0-1.fc32.ppc64le",
    sha256 = "7bf90b5aecb556664c3e8e16d88804b756ebc67350ee0f5b6d86d8187cb35221",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7bf90b5aecb556664c3e8e16d88804b756ebc67350ee0f5b6d86d8187cb35221",
    ],
)

rpm(
    name = "nginx-filesystem-1__1.18.0-1.fc32.x86_64",
    sha256 = "7bf90b5aecb556664c3e8e16d88804b756ebc67350ee0f5b6d86d8187cb35221",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.18.0-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7bf90b5aecb556664c3e8e16d88804b756ebc67350ee0f5b6d86d8187cb35221",
    ],
)

rpm(
    name = "nginx-mimetypes-0__2.1.48-7.fc32.ppc64le",
    sha256 = "657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    ],
)

rpm(
    name = "nginx-mimetypes-0__2.1.48-7.fc32.x86_64",
    sha256 = "657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    ],
)

rpm(
    name = "nmap-ncat-2__7.80-4.fc32.ppc64le",
    sha256 = "42efa76ba186ce078ae4d7035445f5fd6ac26f42ca8203b516e1cf87702f0d50",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nmap-ncat-7.80-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nmap-ncat-7.80-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nmap-ncat-7.80-4.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/n/nmap-ncat-7.80-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/42efa76ba186ce078ae4d7035445f5fd6ac26f42ca8203b516e1cf87702f0d50",
    ],
)

rpm(
    name = "nmap-ncat-2__7.80-4.fc32.x86_64",
    sha256 = "35642d97e77aa48070364b7cd2e4704bb53b87b732c7dc484b59f51446aaaca8",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/35642d97e77aa48070364b7cd2e4704bb53b87b732c7dc484b59f51446aaaca8",
    ],
)

rpm(
    name = "nspr-0__4.29.0-1.fc32.x86_64",
    sha256 = "f6cd80d0da16a6437ca6346c7715946eab657b37aaf10022afe97b5e86c0bfba",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nspr-4.29.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nspr-4.29.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nspr-4.29.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nspr-4.29.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f6cd80d0da16a6437ca6346c7715946eab657b37aaf10022afe97b5e86c0bfba",
    ],
)

rpm(
    name = "nss-0__3.59.0-3.fc32.x86_64",
    sha256 = "e22050e96714fe1b62234337977440cd2f0c2c2071c931a7bc68cc91480293cd",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-3.59.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e22050e96714fe1b62234337977440cd2f0c2c2071c931a7bc68cc91480293cd",
    ],
)

rpm(
    name = "nss-softokn-0__3.59.0-3.fc32.x86_64",
    sha256 = "0c1749f3bd5277aeb18a3cf5496eb15c16b69e1185696e05e83d5789f0e49d0f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-3.59.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0c1749f3bd5277aeb18a3cf5496eb15c16b69e1185696e05e83d5789f0e49d0f",
    ],
)

rpm(
    name = "nss-softokn-freebl-0__3.59.0-3.fc32.x86_64",
    sha256 = "b7152ac4dc60830ab0c85543dd3cd4ae59156de6a3a685fe365328cce9fdd640",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-freebl-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-freebl-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-freebl-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-softokn-freebl-3.59.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b7152ac4dc60830ab0c85543dd3cd4ae59156de6a3a685fe365328cce9fdd640",
    ],
)

rpm(
    name = "nss-sysinit-0__3.59.0-3.fc32.x86_64",
    sha256 = "4c7bba88c5b69f6831e588a487913609086a2eadd15a7dc65ce1dc57581b2e30",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-sysinit-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-sysinit-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-sysinit-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-sysinit-3.59.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c7bba88c5b69f6831e588a487913609086a2eadd15a7dc65ce1dc57581b2e30",
    ],
)

rpm(
    name = "nss-util-0__3.59.0-3.fc32.x86_64",
    sha256 = "e63c75f46c76b7e6e03f4aba1e522293513218dfda8e10b7216da04bbe046033",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-util-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-util-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-util-3.59.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nss-util-3.59.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e63c75f46c76b7e6e03f4aba1e522293513218dfda8e10b7216da04bbe046033",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.12-4.fc32.x86_64",
    sha256 = "af4f4317249ad46956ada6c23dd5966a6679581672b52510763ef6324aee95b7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/af4f4317249ad46956ada6c23dd5966a6679581672b52510763ef6324aee95b7",
    ],
)

rpm(
    name = "numad-0__0.5-31.20150602git.fc32.x86_64",
    sha256 = "6bef82ea8e1006735181860421272e799da79568be2ed31e93267de664915992",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6bef82ea8e1006735181860421272e799da79568be2ed31e93267de664915992",
    ],
)

rpm(
    name = "openldap-0__2.4.47-5.fc32.x86_64",
    sha256 = "d528d4c020ec729776d30db52c141c47afaf991847c0c541870f97613d4b3f1f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d528d4c020ec729776d30db52c141c47afaf991847c0c541870f97613d4b3f1f",
    ],
)

rpm(
    name = "openssl-1__1.1.1i-1.fc32.ppc64le",
    sha256 = "cf339e64c63b1023082ef3851e8d529cc0dfccda89168bce6ab58d0ad7c16d91",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-1.1.1i-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-1.1.1i-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-1.1.1i-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-1.1.1i-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cf339e64c63b1023082ef3851e8d529cc0dfccda89168bce6ab58d0ad7c16d91",
    ],
)

rpm(
    name = "openssl-1__1.1.1i-1.fc32.x86_64",
    sha256 = "77f067961c3701c365191055698bde7bea91af41f1d0870e882b24db2cd354b4",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1i-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1i-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1i-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1i-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/77f067961c3701c365191055698bde7bea91af41f1d0870e882b24db2cd354b4",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1i-1.fc32.ppc64le",
    sha256 = "96dddb5cd4a0e8b3e81ffa3469fa117520629af955e4dcb5cb300c7a7a517846",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-libs-1.1.1i-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-libs-1.1.1i-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-libs-1.1.1i-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/o/openssl-libs-1.1.1i-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/96dddb5cd4a0e8b3e81ffa3469fa117520629af955e4dcb5cb300c7a7a517846",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1i-1.fc32.x86_64",
    sha256 = "45212bcfa4dffead244269e536aa8f072d46c0a80bb282b790fc4c1689d236ed",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1i-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1i-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1i-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1i-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/45212bcfa4dffead244269e536aa8f072d46c0a80bb282b790fc4c1689d236ed",
    ],
)

rpm(
    name = "opus-0__1.3.1-3.fc32.x86_64",
    sha256 = "fe9c792920ecdecbbc9cb37bcb870a8dc87ea5e046997c2149ebb9d621f56889",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe9c792920ecdecbbc9cb37bcb870a8dc87ea5e046997c2149ebb9d621f56889",
    ],
)

rpm(
    name = "orc-0__0.4.31-2.fc32.x86_64",
    sha256 = "fb6f26c0f7a3a559b79e8afe1f5c8cbe9df2ee1c440d34868e6e252e7a8401c2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb6f26c0f7a3a559b79e8afe1f5c8cbe9df2ee1c440d34868e6e252e7a8401c2",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-1.fc32.ppc64le",
    sha256 = "99223672835415e306626c0b68acac790ac6c5d8edc20c16cfa0925e9c5bfa00",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-0.23.22-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-0.23.22-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-0.23.22-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-0.23.22-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/99223672835415e306626c0b68acac790ac6c5d8edc20c16cfa0925e9c5bfa00",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-1.fc32.x86_64",
    sha256 = "d61d13e6fbd7bf1c197460b88add298295fa907fbb30cc2a3e060bf1dcd5c416",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d61d13e6fbd7bf1c197460b88add298295fa907fbb30cc2a3e060bf1dcd5c416",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-1.fc32.ppc64le",
    sha256 = "1b5521b037b4648bdce283771c94ac12bb0e254c1b30919fa80bf13008f462be",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-trust-0.23.22-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-trust-0.23.22-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-trust-0.23.22-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/p11-kit-trust-0.23.22-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/1b5521b037b4648bdce283771c94ac12bb0e254c1b30919fa80bf13008f462be",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-1.fc32.x86_64",
    sha256 = "11dce8bdfc524bb1e6d45256fbecd7d085ced11e0f03c2c9997f1d5d05a7ca69",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/11dce8bdfc524bb1e6d45256fbecd7d085ced11e0f03c2c9997f1d5d05a7ca69",
    ],
)

rpm(
    name = "pam-0__1.3.1-30.fc32.ppc64le",
    sha256 = "70dd17024926a421a62ced1e09fe33b10e184e39153ee6c2ed78f3f831c1244d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pam-1.3.1-30.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pam-1.3.1-30.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pam-1.3.1-30.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pam-1.3.1-30.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/70dd17024926a421a62ced1e09fe33b10e184e39153ee6c2ed78f3f831c1244d",
    ],
)

rpm(
    name = "pam-0__1.3.1-30.fc32.x86_64",
    sha256 = "b4c15f65d7d7a33d673da78088490ff384eb04006b2a460e049c72dbb0ee0691",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4c15f65d7d7a33d673da78088490ff384eb04006b2a460e049c72dbb0ee0691",
    ],
)

rpm(
    name = "pango-0__1.44.7-2.fc32.x86_64",
    sha256 = "de94c1578d3efd87e474f48f7d86f55f07cbf5ef84be49bfa84c603ea4733ac8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pango-1.44.7-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pango-1.44.7-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pango-1.44.7-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pango-1.44.7-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/de94c1578d3efd87e474f48f7d86f55f07cbf5ef84be49bfa84c603ea4733ac8",
    ],
)

rpm(
    name = "parted-0__3.3-3.fc32.x86_64",
    sha256 = "4b9da0cedada16eb80a2d66b86aa62c9b026501e13a3e947a5a03a06f41d9dbb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4b9da0cedada16eb80a2d66b86aa62c9b026501e13a3e947a5a03a06f41d9dbb",
    ],
)

rpm(
    name = "pciutils-0__3.6.4-1.fc32.ppc64le",
    sha256 = "ffc500180562dc8df134e65130afe9cd8dd2f83426b9140e13d389bf72657215",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-3.6.4-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-3.6.4-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-3.6.4-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-3.6.4-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ffc500180562dc8df134e65130afe9cd8dd2f83426b9140e13d389bf72657215",
    ],
)

rpm(
    name = "pciutils-0__3.6.4-1.fc32.x86_64",
    sha256 = "444f18dc1d8f6d0a4ff8ca9816e21e8faaeb4c31ac7997774a9454d4d336c21b",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-3.6.4-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-3.6.4-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-3.6.4-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-3.6.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/444f18dc1d8f6d0a4ff8ca9816e21e8faaeb4c31ac7997774a9454d4d336c21b",
    ],
)

rpm(
    name = "pciutils-libs-0__3.6.4-1.fc32.ppc64le",
    sha256 = "8c0846ffe61509b94be8c839e68e609c00781040c3ed2b499a580b55b731c191",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-libs-3.6.4-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-libs-3.6.4-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-libs-3.6.4-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/pciutils-libs-3.6.4-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8c0846ffe61509b94be8c839e68e609c00781040c3ed2b499a580b55b731c191",
    ],
)

rpm(
    name = "pciutils-libs-0__3.6.4-1.fc32.x86_64",
    sha256 = "e5efc87172d7081559137feaa221047385a5e248ffafd9794c2bfc73b61f8f37",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-libs-3.6.4-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-libs-3.6.4-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-libs-3.6.4-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pciutils-libs-3.6.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e5efc87172d7081559137feaa221047385a5e248ffafd9794c2bfc73b61f8f37",
    ],
)

rpm(
    name = "pcre-0__8.44-2.fc32.ppc64le",
    sha256 = "c45d212efde883a3aca61b87686f2b8e1512779467eed44bdf342de24a35e090",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre-8.44-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre-8.44-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre-8.44-2.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre-8.44-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/c45d212efde883a3aca61b87686f2b8e1512779467eed44bdf342de24a35e090",
    ],
)

rpm(
    name = "pcre-0__8.44-2.fc32.x86_64",
    sha256 = "3d6d8a95ef1416fa148f9776f4d8ca347d3346c5f4b7b066563d52d1562aaabd",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3d6d8a95ef1416fa148f9776f4d8ca347d3346c5f4b7b066563d52d1562aaabd",
    ],
)

rpm(
    name = "pcre2-0__10.36-1.fc32.ppc64le",
    sha256 = "bda1037dc189f0602f8c48e6c47bd43fc2753f8a13730e2a3b6aa4d6767e0e4b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-10.36-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-10.36-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-10.36-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-10.36-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/bda1037dc189f0602f8c48e6c47bd43fc2753f8a13730e2a3b6aa4d6767e0e4b",
    ],
)

rpm(
    name = "pcre2-0__10.36-1.fc32.x86_64",
    sha256 = "ff6a5f0fa8a59c7d727c7d19e7edce1ae8e089bd5fc12f3717d25f7631beb82e",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff6a5f0fa8a59c7d727c7d19e7edce1ae8e089bd5fc12f3717d25f7631beb82e",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.36-1.fc32.ppc64le",
    sha256 = "b73e858a35c21059b045bac2f4526d39160f3ab3de4a12a363bf4c3bb086e027",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b73e858a35c21059b045bac2f4526d39160f3ab3de4a12a363bf4c3bb086e027",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.36-1.fc32.x86_64",
    sha256 = "b73e858a35c21059b045bac2f4526d39160f3ab3de4a12a363bf4c3bb086e027",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b73e858a35c21059b045bac2f4526d39160f3ab3de4a12a363bf4c3bb086e027",
    ],
)

rpm(
    name = "perl-Carp-0__1.50-440.fc32.ppc64le",
    sha256 = "79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    ],
)

rpm(
    name = "perl-Carp-0__1.50-440.fc32.x86_64",
    sha256 = "79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    ],
)

rpm(
    name = "perl-Config-General-0__2.63-11.fc32.ppc64le",
    sha256 = "9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    ],
)

rpm(
    name = "perl-Config-General-0__2.63-11.fc32.x86_64",
    sha256 = "9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    ],
)

rpm(
    name = "perl-Encode-4__3.08-458.fc32.ppc64le",
    sha256 = "8aefb1ad4ae6f55fa99dc679b98c95c62623df03f08b9b5914cae0a7fa128355",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Encode-3.08-458.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Encode-3.08-458.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Encode-3.08-458.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Encode-3.08-458.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/8aefb1ad4ae6f55fa99dc679b98c95c62623df03f08b9b5914cae0a7fa128355",
    ],
)

rpm(
    name = "perl-Encode-4__3.08-458.fc32.x86_64",
    sha256 = "3443414bc9203145a26290ab9aecfc04dc2c272647411db03d09194f8ff69277",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3443414bc9203145a26290ab9aecfc04dc2c272647411db03d09194f8ff69277",
    ],
)

rpm(
    name = "perl-Errno-0__1.30-458.fc32.ppc64le",
    sha256 = "d8ff23a8936fb113e626730b251db37330e4b50e542d526854f3b793e48bf320",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Errno-1.30-458.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Errno-1.30-458.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Errno-1.30-458.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Errno-1.30-458.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/d8ff23a8936fb113e626730b251db37330e4b50e542d526854f3b793e48bf320",
    ],
)

rpm(
    name = "perl-Errno-0__1.30-458.fc32.x86_64",
    sha256 = "a522a8fd92f6d16334563dd15c5ff39d25a48761419550701f557870460c5028",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-458.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-458.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-458.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-458.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a522a8fd92f6d16334563dd15c5ff39d25a48761419550701f557870460c5028",
    ],
)

rpm(
    name = "perl-Exporter-0__5.74-2.fc32.ppc64le",
    sha256 = "9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    ],
)

rpm(
    name = "perl-Exporter-0__5.74-2.fc32.x86_64",
    sha256 = "9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    ],
)

rpm(
    name = "perl-File-Path-0__2.17-1.fc32.ppc64le",
    sha256 = "0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    ],
)

rpm(
    name = "perl-File-Path-0__2.17-1.fc32.x86_64",
    sha256 = "0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    ],
)

rpm(
    name = "perl-File-Temp-1__0.230.900-440.fc32.ppc64le",
    sha256 = "006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    ],
)

rpm(
    name = "perl-File-Temp-1__0.230.900-440.fc32.x86_64",
    sha256 = "006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.52-1.fc32.ppc64le",
    sha256 = "4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.52-1.fc32.x86_64",
    sha256 = "4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.076-440.fc32.ppc64le",
    sha256 = "af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.076-440.fc32.x86_64",
    sha256 = "af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    ],
)

rpm(
    name = "perl-IO-0__1.40-458.fc32.ppc64le",
    sha256 = "9352eec11a968e8d357ba9b24ec514f24bb89a835ef9b2e61fb9a6accfa5f22e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-IO-1.40-458.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-IO-1.40-458.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-IO-1.40-458.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-IO-1.40-458.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/9352eec11a968e8d357ba9b24ec514f24bb89a835ef9b2e61fb9a6accfa5f22e",
    ],
)

rpm(
    name = "perl-IO-0__1.40-458.fc32.x86_64",
    sha256 = "22f3b3ea68328ac99d853ee5f1777844c012f2fdc841bd544cd04071de6f52b2",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-458.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-458.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-458.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-458.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/22f3b3ea68328ac99d853ee5f1777844c012f2fdc841bd544cd04071de6f52b2",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.15-440.fc32.ppc64le",
    sha256 = "54f06ab20a347cc406236df3908bab0db9754edc8e9967e4e38dadf9cf0f8d75",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/54f06ab20a347cc406236df3908bab0db9754edc8e9967e4e38dadf9cf0f8d75",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.15-440.fc32.x86_64",
    sha256 = "86695db247813a6aec340c481e41b747deb588a3abec1528213087d84f99d430",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86695db247813a6aec340c481e41b747deb588a3abec1528213087d84f99d430",
    ],
)

rpm(
    name = "perl-PathTools-0__3.78-442.fc32.ppc64le",
    sha256 = "6112420cdc6ef3fd479c4d423993e058a7d729918c3ae52b4b39528620828426",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-PathTools-3.78-442.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-PathTools-3.78-442.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-PathTools-3.78-442.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-PathTools-3.78-442.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/6112420cdc6ef3fd479c4d423993e058a7d729918c3ae52b4b39528620828426",
    ],
)

rpm(
    name = "perl-PathTools-0__3.78-442.fc32.x86_64",
    sha256 = "79ac869bf8d4d4c322134d6b256faacd46476e3ede94d2a9ccf8b289e450d771",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/79ac869bf8d4d4c322134d6b256faacd46476e3ede94d2a9ccf8b289e450d771",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-440.fc32.ppc64le",
    sha256 = "32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-440.fc32.x86_64",
    sha256 = "32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28.01-443.fc32.ppc64le",
    sha256 = "03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28.01-443.fc32.x86_64",
    sha256 = "03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.40-2.fc32.ppc64le",
    sha256 = "c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.40-2.fc32.x86_64",
    sha256 = "c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__2.01-1.fc32.ppc64le",
    sha256 = "ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__2.01-1.fc32.x86_64",
    sha256 = "ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-3__1.54-440.fc32.ppc64le",
    sha256 = "cea5ef2933c028b9179969890242268ba798a039d39182302312cd4947dd90a4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cea5ef2933c028b9179969890242268ba798a039d39182302312cd4947dd90a4",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-3__1.54-440.fc32.x86_64",
    sha256 = "4a2c7d2dfbb0b6813b5fc4d73e791b011ef2353ca5793474cdffd240ae4295fd",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a2c7d2dfbb0b6813b5fc4d73e791b011ef2353ca5793474cdffd240ae4295fd",
    ],
)

rpm(
    name = "perl-Socket-4__2.030-1.fc32.ppc64le",
    sha256 = "999500414978014ac2054523975b0baca29cc2bf8f19ba63a454198d7d0eea5e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Socket-2.030-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Socket-2.030-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Socket-2.030-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-Socket-2.030-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/999500414978014ac2054523975b0baca29cc2bf8f19ba63a454198d7d0eea5e",
    ],
)

rpm(
    name = "perl-Socket-4__2.030-1.fc32.x86_64",
    sha256 = "d37ea6fe187724f6cb4a5a77cb14320880165b20a976b914cb0d1e6684ece3ff",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.030-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.030-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.030-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.030-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d37ea6fe187724f6cb4a5a77cb14320880165b20a976b914cb0d1e6684ece3ff",
    ],
)

rpm(
    name = "perl-Storable-1__3.15-443.fc32.ppc64le",
    sha256 = "a09206dfa71cf127d7efc76932bf7a76c4b0a27505a90d421ef081ca1980b43e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Storable-3.15-443.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Storable-3.15-443.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Storable-3.15-443.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Storable-3.15-443.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a09206dfa71cf127d7efc76932bf7a76c4b0a27505a90d421ef081ca1980b43e",
    ],
)

rpm(
    name = "perl-Storable-1__3.15-443.fc32.x86_64",
    sha256 = "e2e9c4b18e6a65182e8368a8446a9031550b32c27443c0fda580d3d1d110792b",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e2e9c4b18e6a65182e8368a8446a9031550b32c27443c0fda580d3d1d110792b",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__5.01-2.fc32.ppc64le",
    sha256 = "5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__5.01-2.fc32.x86_64",
    sha256 = "5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-440.fc32.ppc64le",
    sha256 = "48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-440.fc32.x86_64",
    sha256 = "48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-440.fc32.ppc64le",
    sha256 = "48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-440.fc32.x86_64",
    sha256 = "48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-440.fc32.ppc64le",
    sha256 = "f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-440.fc32.x86_64",
    sha256 = "f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    ],
)

rpm(
    name = "perl-Time-Local-2__1.300-2.fc32.ppc64le",
    sha256 = "2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    ],
)

rpm(
    name = "perl-Time-Local-2__1.300-2.fc32.x86_64",
    sha256 = "2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    ],
)

rpm(
    name = "perl-Unicode-Normalize-0__1.26-440.fc32.ppc64le",
    sha256 = "f8a0613b8e2358890ae0fd6ae7de69bae84eb66b01623f45c069c7f425760542",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/f8a0613b8e2358890ae0fd6ae7de69bae84eb66b01623f45c069c7f425760542",
    ],
)

rpm(
    name = "perl-Unicode-Normalize-0__1.26-440.fc32.x86_64",
    sha256 = "962ab865d9e38bb3e67284dd7c1ea1aac1e83074b72f381b50e6f7b4a65d3e84",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/962ab865d9e38bb3e67284dd7c1ea1aac1e83074b72f381b50e6f7b4a65d3e84",
    ],
)

rpm(
    name = "perl-constant-0__1.33-441.fc32.ppc64le",
    sha256 = "965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    ],
)

rpm(
    name = "perl-constant-0__1.33-441.fc32.x86_64",
    sha256 = "965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    ],
)

rpm(
    name = "perl-interpreter-4__5.30.3-458.fc32.ppc64le",
    sha256 = "3c5647ac9f27e5a9171fcc38598dfdead2dde0dedfce20c33d8dc0fc8d170997",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-interpreter-5.30.3-458.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-interpreter-5.30.3-458.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-interpreter-5.30.3-458.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-interpreter-5.30.3-458.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3c5647ac9f27e5a9171fcc38598dfdead2dde0dedfce20c33d8dc0fc8d170997",
    ],
)

rpm(
    name = "perl-interpreter-4__5.30.3-458.fc32.x86_64",
    sha256 = "026cbc03f348c3cd5de4dd839e110babba30d1a02bfdd215fc1bc70ce3807ddf",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-458.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-458.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-458.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-458.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/026cbc03f348c3cd5de4dd839e110babba30d1a02bfdd215fc1bc70ce3807ddf",
    ],
)

rpm(
    name = "perl-libs-4__5.30.3-458.fc32.ppc64le",
    sha256 = "3b25cb88c3e97872a531bb569c548230a47cb040f5995ab036ade6a7d278a22b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-libs-5.30.3-458.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-libs-5.30.3-458.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-libs-5.30.3-458.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-libs-5.30.3-458.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3b25cb88c3e97872a531bb569c548230a47cb040f5995ab036ade6a7d278a22b",
    ],
)

rpm(
    name = "perl-libs-4__5.30.3-458.fc32.x86_64",
    sha256 = "68961899edf4caedd28f3cc549ad2d83bb23bf9d066f41eb27e194b06d90d9c6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-458.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-458.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-458.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-458.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/68961899edf4caedd28f3cc549ad2d83bb23bf9d066f41eb27e194b06d90d9c6",
    ],
)

rpm(
    name = "perl-macros-4__5.30.3-458.fc32.ppc64le",
    sha256 = "0d959450cabbb0bace1227f09116d6cf2b65bd663305cdca600640ae1c029a7d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0d959450cabbb0bace1227f09116d6cf2b65bd663305cdca600640ae1c029a7d",
    ],
)

rpm(
    name = "perl-macros-4__5.30.3-458.fc32.x86_64",
    sha256 = "0d959450cabbb0bace1227f09116d6cf2b65bd663305cdca600640ae1c029a7d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-458.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0d959450cabbb0bace1227f09116d6cf2b65bd663305cdca600640ae1c029a7d",
    ],
)

rpm(
    name = "perl-parent-1__0.238-1.fc32.ppc64le",
    sha256 = "4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    ],
)

rpm(
    name = "perl-parent-1__0.238-1.fc32.x86_64",
    sha256 = "4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    ],
)

rpm(
    name = "perl-podlators-1__4.14-2.fc32.ppc64le",
    sha256 = "92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    ],
)

rpm(
    name = "perl-podlators-1__4.14-2.fc32.x86_64",
    sha256 = "92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    ],
)

rpm(
    name = "perl-threads-1__2.22-442.fc32.ppc64le",
    sha256 = "88a220789fbb007176665be19847b6e1022bb3f766d5857c8ac69521b846f470",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-2.22-442.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-2.22-442.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-2.22-442.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-2.22-442.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/88a220789fbb007176665be19847b6e1022bb3f766d5857c8ac69521b846f470",
    ],
)

rpm(
    name = "perl-threads-1__2.22-442.fc32.x86_64",
    sha256 = "ac8f21162d3353c4f65d0e10d72abf6a9c5b5a09c3a3b49aa27d96031ca5923c",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ac8f21162d3353c4f65d0e10d72abf6a9c5b5a09c3a3b49aa27d96031ca5923c",
    ],
)

rpm(
    name = "perl-threads-shared-0__1.60-441.fc32.ppc64le",
    sha256 = "f506e98026e62237309361f1a908ad94aa13b3afb88f9513081ccb02726ba926",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-shared-1.60-441.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-shared-1.60-441.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-shared-1.60-441.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/p/perl-threads-shared-1.60-441.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/f506e98026e62237309361f1a908ad94aa13b3afb88f9513081ccb02726ba926",
    ],
)

rpm(
    name = "perl-threads-shared-0__1.60-441.fc32.x86_64",
    sha256 = "61797e7bdacb824cea1c1dbe5702a60b1f853bc76e6f9e1cddc2cddb98320b40",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61797e7bdacb824cea1c1dbe5702a60b1f853bc76e6f9e1cddc2cddb98320b40",
    ],
)

rpm(
    name = "pixman-0__0.40.0-1.fc32.ppc64le",
    sha256 = "1345d15c775292065f7c4306dfe6acebd253154273c8932cb6af0731527af779",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pixman-0.40.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pixman-0.40.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pixman-0.40.0-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/pixman-0.40.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/1345d15c775292065f7c4306dfe6acebd253154273c8932cb6af0731527af779",
    ],
)

rpm(
    name = "pixman-0__0.40.0-1.fc32.x86_64",
    sha256 = "ca51912d142cc9b61657fe53e0297a53200cb2b3a96bb5b6b07c8fc1019d44af",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ca51912d142cc9b61657fe53e0297a53200cb2b3a96bb5b6b07c8fc1019d44af",
    ],
)

rpm(
    name = "pkgconf-0__1.6.3-3.fc32.x86_64",
    sha256 = "5c91890bf33527b9fb422cbed17600e761750a4e596fad3f0d0fa419070e82b0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5c91890bf33527b9fb422cbed17600e761750a4e596fad3f0d0fa419070e82b0",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.6.3-3.fc32.x86_64",
    sha256 = "0bace0cf41921db39247c99bfccb228818b83b68c7b8be7c8c4a92ea298a9a29",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0bace0cf41921db39247c99bfccb228818b83b68c7b8be7c8c4a92ea298a9a29",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.6.3-3.fc32.x86_64",
    sha256 = "4a7b63b32f176b8861f6ac7363bc8010caea0c323eaa83167227118f05603022",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a7b63b32f176b8861f6ac7363bc8010caea0c323eaa83167227118f05603022",
    ],
)

rpm(
    name = "policycoreutils-0__3.0-2.fc32.x86_64",
    sha256 = "8df97dcfb42c1667b5d2e4150012eaf96f58eeac4f7b879e0928c8c36e3a7604",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8df97dcfb42c1667b5d2e4150012eaf96f58eeac4f7b879e0928c8c36e3a7604",
    ],
)

rpm(
    name = "polkit-0__0.116-7.fc32.x86_64",
    sha256 = "d49f0b1c8ecf9bc808ae93e9298a40fbcc124fe67c3bbdd37705b6b5d8cfdd87",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d49f0b1c8ecf9bc808ae93e9298a40fbcc124fe67c3bbdd37705b6b5d8cfdd87",
    ],
)

rpm(
    name = "polkit-libs-0__0.116-7.fc32.x86_64",
    sha256 = "d439ffbe20c8c0e8244e31c0324d60cf959dc1cd6cecc575d7b34509a73e9386",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d439ffbe20c8c0e8244e31c0324d60cf959dc1cd6cecc575d7b34509a73e9386",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-16.fc32.x86_64",
    sha256 = "7c7eff31251dedcc3285a8b08c1b18f7fd9ee2e07dff86ad090f45a81e19e85e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7c7eff31251dedcc3285a8b08c1b18f7fd9ee2e07dff86ad090f45a81e19e85e",
    ],
)

rpm(
    name = "popt-0__1.16-19.fc32.x86_64",
    sha256 = "8a0c00a69f9cb3a9ffacaf1cdc162c38a1faca76c9b976cb177bdc988902f2d4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a0c00a69f9cb3a9ffacaf1cdc162c38a1faca76c9b976cb177bdc988902f2d4",
    ],
)

rpm(
    name = "procps-ng-0__3.3.16-2.fc32.ppc64le",
    sha256 = "71bc98a89df9891edf10bb08833c063972c15f5997a11bbaa94d21d3e395e961",
    urls = [
        "https://download-ib01.fedoraproject.org/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/procps-ng-3.3.16-2.fc32.ppc64le.rpm",
        "https://download-cc-rdu01.fedoraproject.org/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/procps-ng-3.3.16-2.fc32.ppc64le.rpm",
        "https://d2lzkl7pfhq30w.cloudfront.net/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/procps-ng-3.3.16-2.fc32.ppc64le.rpm",
        "https://pubmirror2.math.uh.edu/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/p/procps-ng-3.3.16-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/71bc98a89df9891edf10bb08833c063972c15f5997a11bbaa94d21d3e395e961",
    ],
)

rpm(
    name = "procps-ng-0__3.3.16-2.fc32.x86_64",
    sha256 = "9d360e29f7a54d585853407d778b194ebe299663e83d59394ac122ae1687f61a",
    urls = [
        "https://mirror.ette.biz/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://sjc.edge.kernel.org/fedora-buffet/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://mirror.genesisadaptive.com/fedora/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://mirror.umd.edu/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d360e29f7a54d585853407d778b194ebe299663e83d59394ac122ae1687f61a",
    ],
)

rpm(
    name = "psmisc-0__23.3-3.fc32.x86_64",
    sha256 = "be7ba234b6c48717ac0f69fb5868b3caa6ef09fbfc76c42a47b367578cd19444",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be7ba234b6c48717ac0f69fb5868b3caa6ef09fbfc76c42a47b367578cd19444",
    ],
)

rpm(
    name = "python-pip-wheel-0__19.3.1-4.fc32.x86_64",
    sha256 = "3fa72c1b9e6ff09048ad10ccd6339669144595f5e10e905055b25c3d6f0fd3d6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3fa72c1b9e6ff09048ad10ccd6339669144595f5e10e905055b25c3d6f0fd3d6",
    ],
)

rpm(
    name = "python-setuptools-wheel-0__41.6.0-2.fc32.x86_64",
    sha256 = "7dd93baaf69a8004ae2cd3b9e6660b862d0b6f399d53c05a27a48a2e276ef1ee",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7dd93baaf69a8004ae2cd3b9e6660b862d0b6f399d53c05a27a48a2e276ef1ee",
    ],
)

rpm(
    name = "python3-0__3.8.6-1.fc32.x86_64",
    sha256 = "e7c920a873c4ac9741289f3a51d649a9d741d9daa81d6f20bae56f0270feca94",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.6-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.6-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.6-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.6-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e7c920a873c4ac9741289f3a51d649a9d741d9daa81d6f20bae56f0270feca94",
    ],
)

rpm(
    name = "python3-libs-0__3.8.6-1.fc32.x86_64",
    sha256 = "44dd717d4062698ca3dd8ce1b93336ee62a3eed4441e7eddaae12e2690b58d10",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.6-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.6-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.6-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.6-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/44dd717d4062698ca3dd8ce1b93336ee62a3eed4441e7eddaae12e2690b58d10",
    ],
)

rpm(
    name = "qemu-guest-agent-15__5.1.0-18.fc32.x86_64",
    sha256 = "9eaac49247c2e8575b7e9ea794cf6c42f3837a2f37e94540f973963305f835d1",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-guest-agent-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9eaac49247c2e8575b7e9ea794cf6c42f3837a2f37e94540f973963305f835d1",
    ],
)

rpm(
    name = "qemu-guest-agent-2__4.2.1-1.fc32.ppc64le",
    sha256 = "deaf273d0c687aa17e64f1c57b94df250e7630527770b5a5ea3bfc152ffb1976",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-guest-agent-4.2.1-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-guest-agent-4.2.1-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-guest-agent-4.2.1-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-guest-agent-4.2.1-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/deaf273d0c687aa17e64f1c57b94df250e7630527770b5a5ea3bfc152ffb1976",
    ],
)

rpm(
    name = "qemu-img-15__5.1.0-18.fc32.x86_64",
    sha256 = "92e84606ca802240fab3ee8fcdec40c42ee7131b3203208b4be4d7f984cf6423",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-img-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/92e84606ca802240fab3ee8fcdec40c42ee7131b3203208b4be4d7f984cf6423",
    ],
)

rpm(
    name = "qemu-img-2__4.2.1-1.fc32.ppc64le",
    sha256 = "0681f4b8590f0747f17d580e5b0812bf03a03a3b7dcdeb5ec142b97a56c2689a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-img-4.2.1-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-img-4.2.1-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-img-4.2.1-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/q/qemu-img-4.2.1-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0681f4b8590f0747f17d580e5b0812bf03a03a3b7dcdeb5ec142b97a56c2689a",
    ],
)

rpm(
    name = "qemu-kvm-15__5.1.0-18.fc32.x86_64",
    sha256 = "e655dddabc8fa8c64a2afbb670058d011a16397a7905c2b33978f40e7717ab9b",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e655dddabc8fa8c64a2afbb670058d011a16397a7905c2b33978f40e7717ab9b",
    ],
)

rpm(
    name = "qemu-kvm-block-curl-15__5.1.0-18.fc32.x86_64",
    sha256 = "2378aebaef7afbc384cc6f7552af1f119e1fd8edee64d0840b45352642efaa0d",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-block-curl-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2378aebaef7afbc384cc6f7552af1f119e1fd8edee64d0840b45352642efaa0d",
    ],
)

rpm(
    name = "qemu-kvm-block-gluster-15__5.1.0-18.fc32.x86_64",
    sha256 = "f982f376d68ec75cd72ce7c7174e5e5251d0de2f2006472d278a3397c8a93fe2",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-block-gluster-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f982f376d68ec75cd72ce7c7174e5e5251d0de2f2006472d278a3397c8a93fe2",
    ],
)

rpm(
    name = "qemu-kvm-block-iscsi-15__5.1.0-18.fc32.x86_64",
    sha256 = "fb9a193820329f8a82959d02778a00a2aebb6060914ca47916f9766643164672",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-block-iscsi-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb9a193820329f8a82959d02778a00a2aebb6060914ca47916f9766643164672",
    ],
)

rpm(
    name = "qemu-kvm-block-rbd-15__5.1.0-18.fc32.x86_64",
    sha256 = "56123ff97720288ff94ef73ff5639bc70c32310e5106a72cd51c6f5003a23d82",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-block-rbd-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56123ff97720288ff94ef73ff5639bc70c32310e5106a72cd51c6f5003a23d82",
    ],
)

rpm(
    name = "qemu-kvm-block-ssh-15__5.1.0-18.fc32.x86_64",
    sha256 = "4fb227e77d3aff12725b04f6348a32ad9abd8c447020f11e4734caad18a702a5",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-block-ssh-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4fb227e77d3aff12725b04f6348a32ad9abd8c447020f11e4734caad18a702a5",
    ],
)

rpm(
    name = "qemu-kvm-common-15__5.1.0-18.fc32.x86_64",
    sha256 = "3ba3330400f5a267141e906378410c99a0a4702e27fce25653db78d9bbbe056a",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-common-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3ba3330400f5a267141e906378410c99a0a4702e27fce25653db78d9bbbe056a",
    ],
)

rpm(
    name = "qemu-kvm-core-15__5.1.0-18.fc32.x86_64",
    sha256 = "b70030acb6a9be539a60cad4d3f4f0dd3143868d265f73a04a5d6fa3deeded1b",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.1.0-18.el8/fedora-32-x86_64/01890581-qemu-kvm/qemu-kvm-core-5.1.0-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b70030acb6a9be539a60cad4d3f4f0dd3143868d265f73a04a5d6fa3deeded1b",
    ],
)

rpm(
    name = "qrencode-libs-0__4.0.2-5.fc32.ppc64le",
    sha256 = "a84918ba07f40ee371f37b4fe78a7c5fb3b3ed65c7e4befd111f110d64ac77a9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/q/qrencode-libs-4.0.2-5.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/q/qrencode-libs-4.0.2-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/q/qrencode-libs-4.0.2-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/q/qrencode-libs-4.0.2-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a84918ba07f40ee371f37b4fe78a7c5fb3b3ed65c7e4befd111f110d64ac77a9",
    ],
)

rpm(
    name = "qrencode-libs-0__4.0.2-5.fc32.x86_64",
    sha256 = "f1150f9e17beaef09aca0f291e10db8c3ee5566fbf4c929b7672334410fa74e9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1150f9e17beaef09aca0f291e10db8c3ee5566fbf4c929b7672334410fa74e9",
    ],
)

rpm(
    name = "quota-1__4.05-9.fc32.x86_64",
    sha256 = "aaa2d529c6d1b67a668a1e4b6d4bbd406e7328df57a2ab1ed034b3f70edeab4d",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-4.05-9.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-4.05-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-4.05-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-4.05-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aaa2d529c6d1b67a668a1e4b6d4bbd406e7328df57a2ab1ed034b3f70edeab4d",
    ],
)

rpm(
    name = "quota-nls-1__4.05-9.fc32.x86_64",
    sha256 = "ca1a77a822f87973f14ed740c640183c8d9cf4ce0f3b2410b92bdf05e5337d90",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-nls-4.05-9.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-nls-4.05-9.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-nls-4.05-9.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/quota-nls-4.05-9.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ca1a77a822f87973f14ed740c640183c8d9cf4ce0f3b2410b92bdf05e5337d90",
    ],
)

rpm(
    name = "rdma-core-0__32.0-1.fc32.ppc64le",
    sha256 = "a0de046b756ba963e7e177cba1f9d579c73482b65b8200fbdc15d6411cb446be",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/r/rdma-core-32.0-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/r/rdma-core-32.0-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/r/rdma-core-32.0-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/r/rdma-core-32.0-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a0de046b756ba963e7e177cba1f9d579c73482b65b8200fbdc15d6411cb446be",
    ],
)

rpm(
    name = "rdma-core-0__32.0-1.fc32.x86_64",
    sha256 = "022f24e47721d43dd78a3b95a589b44447cead91d4a6e848fabefe9027e01691",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-32.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-32.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-32.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-32.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/022f24e47721d43dd78a3b95a589b44447cead91d4a6e848fabefe9027e01691",
    ],
)

rpm(
    name = "readline-0__8.0-4.fc32.ppc64le",
    sha256 = "a4d76ad397446d08d2493d53231a337fef783321507d3b75ca9342f0b2a6ddf5",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/r/readline-8.0-4.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/r/readline-8.0-4.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/r/readline-8.0-4.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/r/readline-8.0-4.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/a4d76ad397446d08d2493d53231a337fef783321507d3b75ca9342f0b2a6ddf5",
    ],
)

rpm(
    name = "readline-0__8.0-4.fc32.x86_64",
    sha256 = "f1c79039f4c6ba0fad88590c2cb55a96489449c334a671cc18c0bf424a4548b8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1c79039f4c6ba0fad88590c2cb55a96489449c334a671cc18c0bf424a4548b8",
    ],
)

rpm(
    name = "rpcbind-0__1.2.5-5.rc1.fc32.1.x86_64",
    sha256 = "aa9f354fbf50678bf453b47de17f45e89dd286dea09313c816b8e756fcbd191b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/rpcbind-1.2.5-5.rc1.fc32.1.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/rpcbind-1.2.5-5.rc1.fc32.1.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/rpcbind-1.2.5-5.rc1.fc32.1.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/rpcbind-1.2.5-5.rc1.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aa9f354fbf50678bf453b47de17f45e89dd286dea09313c816b8e756fcbd191b",
    ],
)

rpm(
    name = "rpm-0__4.15.1-3.fc32.1.x86_64",
    sha256 = "c84a5e00f504e0f60aa02cac7eb5cde57ed7958f2965ea2fce0c7e9cd0ff532b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1-3.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c84a5e00f504e0f60aa02cac7eb5cde57ed7958f2965ea2fce0c7e9cd0ff532b",
    ],
)

rpm(
    name = "rpm-libs-0__4.15.1-3.fc32.1.x86_64",
    sha256 = "506b1d8ba1f1fbf138a94de6a1705da12360c81e6797cb60df488c87e9a366fe",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1-3.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/506b1d8ba1f1fbf138a94de6a1705da12360c81e6797cb60df488c87e9a366fe",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.15.1-3.fc32.1.x86_64",
    sha256 = "53823a669b4ab38d3ab58a6d8524e47f9d7c6998e006b9b5cab59dc1e5b47f88",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1-3.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1-3.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/53823a669b4ab38d3ab58a6d8524e47f9d7c6998e006b9b5cab59dc1e5b47f88",
    ],
)

rpm(
    name = "scsi-target-utils-0__1.0.79-1.fc32.ppc64le",
    sha256 = "22a67ab8ad8d794beb5d84f242baaf59d5917a5430330c24fce0e388fa955c1c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/22a67ab8ad8d794beb5d84f242baaf59d5917a5430330c24fce0e388fa955c1c",
    ],
)

rpm(
    name = "scsi-target-utils-0__1.0.79-1.fc32.x86_64",
    sha256 = "361a48d36c608a4790d2811fecb98503d4afc7da14f53ebb82d53d2e3994d786",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/361a48d36c608a4790d2811fecb98503d4afc7da14f53ebb82d53d2e3994d786",
    ],
)

rpm(
    name = "seabios-0__1.14.0-1.fc32.x86_64",
    sha256 = "20bcd764367d4bce5299356c594be9efe8fddbc7e6e52fc9cd8d0bf4bfe8c6ee",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/seabios-1.14.0-1.el8/fedora-32-x86_64/01822781-seabios/seabios-1.14.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20bcd764367d4bce5299356c594be9efe8fddbc7e6e52fc9cd8d0bf4bfe8c6ee",
    ],
)

rpm(
    name = "seabios-bin-0__1.14.0-1.fc32.x86_64",
    sha256 = "7f404656ca5c58917f13c2e1d123a35951a85302644db709203179baa03f8ef4",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/seabios-1.14.0-1.el8/fedora-32-x86_64/01822781-seabios/seabios-bin-1.14.0-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7f404656ca5c58917f13c2e1d123a35951a85302644db709203179baa03f8ef4",
    ],
)

rpm(
    name = "seavgabios-bin-0__1.14.0-1.fc32.x86_64",
    sha256 = "397e6ea1734cea4400703ec6d17406cc026fef4508517932f2a8dcfd4cb7f9f8",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/seabios-1.14.0-1.el8/fedora-32-x86_64/01822781-seabios/seavgabios-bin-1.14.0-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/397e6ea1734cea4400703ec6d17406cc026fef4508517932f2a8dcfd4cb7f9f8",
    ],
)

rpm(
    name = "sed-0__4.5-5.fc32.ppc64le",
    sha256 = "ebbd44f6edbdc250094608871fced452e3845369dd7648950298970b47069a1f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sed-4.5-5.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sed-4.5-5.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sed-4.5-5.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sed-4.5-5.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/ebbd44f6edbdc250094608871fced452e3845369dd7648950298970b47069a1f",
    ],
)

rpm(
    name = "sed-0__4.5-5.fc32.x86_64",
    sha256 = "ffe5076b9018efdb1612c487f637af39ab6c3c79ec37311978935cfa357ecd61",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffe5076b9018efdb1612c487f637af39ab6c3c79ec37311978935cfa357ecd61",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.5-45.fc32.x86_64",
    sha256 = "e0bab50d42fbf63594c80fc3f3bb0084bd34fba4d3ad7319972829171a5a2e2b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-45.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-45.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-45.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-45.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e0bab50d42fbf63594c80fc3f3bb0084bd34fba4d3ad7319972829171a5a2e2b",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.5-45.fc32.x86_64",
    sha256 = "8e5ee7a15b169c5df0e7d195061b0f4c839f65998027b1076afabb66119ce240",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-45.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-45.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-45.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-45.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8e5ee7a15b169c5df0e7d195061b0f4c839f65998027b1076afabb66119ce240",
    ],
)

rpm(
    name = "setup-0__2.13.6-2.fc32.ppc64le",
    sha256 = "a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    ],
)

rpm(
    name = "setup-0__2.13.6-2.fc32.x86_64",
    sha256 = "a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    ],
)

rpm(
    name = "sg3_utils-0__1.44-3.fc32.ppc64le",
    sha256 = "cc15fd5766197814074c3381cbf46f648965ce8e5064234d2430d91ce79e9f6a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-1.44-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-1.44-3.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-1.44-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-1.44-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/cc15fd5766197814074c3381cbf46f648965ce8e5064234d2430d91ce79e9f6a",
    ],
)

rpm(
    name = "sg3_utils-0__1.44-3.fc32.x86_64",
    sha256 = "cd3d9eb488859202bb6820830d7bb5622219492484e9d98c279ccb1211750eae",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cd3d9eb488859202bb6820830d7bb5622219492484e9d98c279ccb1211750eae",
    ],
)

rpm(
    name = "sg3_utils-libs-0__1.44-3.fc32.ppc64le",
    sha256 = "7939a4685023c073725f2c775de1c3ffaebe590454543bc838575f85006b660b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-libs-1.44-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-libs-1.44-3.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-libs-1.44-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/sg3_utils-libs-1.44-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/7939a4685023c073725f2c775de1c3ffaebe590454543bc838575f85006b660b",
    ],
)

rpm(
    name = "sg3_utils-libs-0__1.44-3.fc32.x86_64",
    sha256 = "acafd54a39135c9ac45e5046f3b4d8b3712eba4acd99d44bd044557ad3c3939c",
    urls = [
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/acafd54a39135c9ac45e5046f3b4d8b3712eba4acd99d44bd044557ad3c3939c",
    ],
)

rpm(
    name = "sgabios-bin-1__0.20180715git-4.fc32.x86_64",
    sha256 = "e89eb0796febd3e1108bc3551180eb0a2dffcbf34aa150be6abb40c6f7cb140c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e89eb0796febd3e1108bc3551180eb0a2dffcbf34aa150be6abb40c6f7cb140c",
    ],
)

rpm(
    name = "shadow-utils-2__4.8.1-3.fc32.ppc64le",
    sha256 = "439392d72f9c3af11cd9ba1b3fd3d320a44dc49e78f0fe10220c8a317f7c0e0a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/shadow-utils-4.8.1-3.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/shadow-utils-4.8.1-3.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/shadow-utils-4.8.1-3.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/shadow-utils-4.8.1-3.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/439392d72f9c3af11cd9ba1b3fd3d320a44dc49e78f0fe10220c8a317f7c0e0a",
    ],
)

rpm(
    name = "shadow-utils-2__4.8.1-3.fc32.x86_64",
    sha256 = "696768dc6f369a52d2c431eb7c76461237c2804d591cee418c04f97f3660b667",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/696768dc6f369a52d2c431eb7c76461237c2804d591cee418c04f97f3660b667",
    ],
)

rpm(
    name = "snappy-0__1.1.8-2.fc32.x86_64",
    sha256 = "186a33671176e2cd2a6d036bc6cc45fa6e331a28f022c495019c3f26ef2ee383",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/186a33671176e2cd2a6d036bc6cc45fa6e331a28f022c495019c3f26ef2ee383",
    ],
)

rpm(
    name = "spice-server-0__0.14.3-1.fc32.x86_64",
    sha256 = "4f8be162f46533c87bd62a6293d49f80ae273a257029653413c4c0c12fe656fe",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4f8be162f46533c87bd62a6293d49f80ae273a257029653413c4c0c12fe656fe",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.0-1.fc32.x86_64",
    sha256 = "19c1021f1729ea60b0b8f6056953150847196732e84745a118d442c2dea19cfb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19c1021f1729ea60b0b8f6056953150847196732e84745a118d442c2dea19cfb",
    ],
)

rpm(
    name = "stress-0__1.0.4-24.fc32.ppc64le",
    sha256 = "e84d0ad191bd5497d4f9eb1b3f3fb8676293e35e674b57ac336bd68bb8e82f80",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/stress-1.0.4-24.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/stress-1.0.4-24.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/stress-1.0.4-24.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/s/stress-1.0.4-24.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/e84d0ad191bd5497d4f9eb1b3f3fb8676293e35e674b57ac336bd68bb8e82f80",
    ],
)

rpm(
    name = "stress-0__1.0.4-24.fc32.x86_64",
    sha256 = "047ed8a6297cbf06c336d9f853c2e121528638b76dcebb5453d744e6d75a96d5",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/stress-1.0.4-24.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/stress-1.0.4-24.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/stress-1.0.4-24.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/stress-1.0.4-24.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/047ed8a6297cbf06c336d9f853c2e121528638b76dcebb5453d744e6d75a96d5",
    ],
)

rpm(
    name = "systemd-0__245.8-2.fc32.ppc64le",
    sha256 = "3c884547bb1150bf8bd7d8b98ca2246bd4fe4e6bd6ac9be19398682f3bcb259e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-245.8-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-245.8-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-245.8-2.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-245.8-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/3c884547bb1150bf8bd7d8b98ca2246bd4fe4e6bd6ac9be19398682f3bcb259e",
    ],
)

rpm(
    name = "systemd-0__245.8-2.fc32.x86_64",
    sha256 = "f5c70db708d429037e23467f5c60d10893fc4a6017a67dd91d0bd5344ecdb0eb",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.8-2.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f5c70db708d429037e23467f5c60d10893fc4a6017a67dd91d0bd5344ecdb0eb",
    ],
)

rpm(
    name = "systemd-container-0__245.8-2.fc32.x86_64",
    sha256 = "f08ea7093e489ada80331d33cb2d801520aa0d70e037582f698aa2fff456a8c7",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.8-2.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.8-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f08ea7093e489ada80331d33cb2d801520aa0d70e037582f698aa2fff456a8c7",
    ],
)

rpm(
    name = "systemd-libs-0__245.8-2.fc32.ppc64le",
    sha256 = "7ab65fca6ce88160e8b5cb949433ac31955e1815ce4d69bb920409198ed9c8c6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-libs-245.8-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-libs-245.8-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-libs-245.8-2.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-libs-245.8-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/7ab65fca6ce88160e8b5cb949433ac31955e1815ce4d69bb920409198ed9c8c6",
    ],
)

rpm(
    name = "systemd-libs-0__245.8-2.fc32.x86_64",
    sha256 = "45deeabf816fa801c66573c17136ed402670d3f8f627a371a7e87f43f4e9caab",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.8-2.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/45deeabf816fa801c66573c17136ed402670d3f8f627a371a7e87f43f4e9caab",
    ],
)

rpm(
    name = "systemd-pam-0__245.8-2.fc32.ppc64le",
    sha256 = "c0393d2d992998a236a42f8625768086d9681cb123de065ca7e6904ea99ef8a8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-pam-245.8-2.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-pam-245.8-2.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-pam-245.8-2.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-pam-245.8-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/c0393d2d992998a236a42f8625768086d9681cb123de065ca7e6904ea99ef8a8",
    ],
)

rpm(
    name = "systemd-pam-0__245.8-2.fc32.x86_64",
    sha256 = "592828b40ea5f0ade6658e6b849f39501723d6599965dea6709bd40afe36bcf8",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.8-2.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/592828b40ea5f0ade6658e6b849f39501723d6599965dea6709bd40afe36bcf8",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__245.8-2.fc32.ppc64le",
    sha256 = "1a6e9f366e262e95f3e5c89ae897cf254d3f655d377103e7c6e0796ff5fdbfec",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1a6e9f366e262e95f3e5c89ae897cf254d3f655d377103e7c6e0796ff5fdbfec",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__245.8-2.fc32.x86_64",
    sha256 = "1a6e9f366e262e95f3e5c89ae897cf254d3f655d377103e7c6e0796ff5fdbfec",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.8-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1a6e9f366e262e95f3e5c89ae897cf254d3f655d377103e7c6e0796ff5fdbfec",
    ],
)

rpm(
    name = "tzdata-0__2020d-1.fc32.ppc64le",
    sha256 = "fe396a1e023f5d0b3e9f7355fe81242cbb9b13aa72d12ea78bff1ece86c71e87",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/fe396a1e023f5d0b3e9f7355fe81242cbb9b13aa72d12ea78bff1ece86c71e87",
    ],
)

rpm(
    name = "tzdata-0__2020d-1.fc32.x86_64",
    sha256 = "fe396a1e023f5d0b3e9f7355fe81242cbb9b13aa72d12ea78bff1ece86c71e87",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2020d-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/fe396a1e023f5d0b3e9f7355fe81242cbb9b13aa72d12ea78bff1ece86c71e87",
    ],
)

rpm(
    name = "unbound-libs-0__1.10.1-1.fc32.x86_64",
    sha256 = "463d8259936601d23a02c407a4fb294c0e3bc9a213282c4525137bc1b18bbf6b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/463d8259936601d23a02c407a4fb294c0e3bc9a213282c4525137bc1b18bbf6b",
    ],
)

rpm(
    name = "usbredir-0__0.8.0-4.fc32.x86_64",
    sha256 = "62cdf815d1c8235745dc40a314480e972a0b7ba7f69fb68355eeacf1333e72b4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/usbredir-0.8.0-4.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/usbredir-0.8.0-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/usbredir-0.8.0-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/usbredir-0.8.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/62cdf815d1c8235745dc40a314480e972a0b7ba7f69fb68355eeacf1333e72b4",
    ],
)

rpm(
    name = "userspace-rcu-0__0.11.1-3.fc32.x86_64",
    sha256 = "1f5b2264d9f9a3f423124dbac73ef04fc9ad0df44afd80e8ba79d21a8942ca79",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f5b2264d9f9a3f423124dbac73ef04fc9ad0df44afd80e8ba79d21a8942ca79",
    ],
)

rpm(
    name = "util-linux-0__2.35.2-1.fc32.ppc64le",
    sha256 = "94c09354887a88e0f6602ae03559296cd59c91314130c58ae04b6281739d7ec8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/u/util-linux-2.35.2-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/updates/32/Everything/ppc64le/Packages/u/util-linux-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/u/util-linux-2.35.2-1.fc32.ppc64le.rpm",
        "https://mirror.yandex.ru/fedora-secondary/updates/32/Everything/ppc64le/Packages/u/util-linux-2.35.2-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/94c09354887a88e0f6602ae03559296cd59c91314130c58ae04b6281739d7ec8",
    ],
)

rpm(
    name = "util-linux-0__2.35.2-1.fc32.x86_64",
    sha256 = "4d80736f9a52519104eeb228eb1ea95d0d6e9addc766eebacc9a5137fb2a5977",
    urls = [
        "https://ftp.plusline.net/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d80736f9a52519104eeb228eb1ea95d0d6e9addc766eebacc9a5137fb2a5977",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2146-2.fc32.ppc64le",
    sha256 = "134adf9ed3ae4e375ef7f1f6d2299e63a940a2fc6f7655c640ae11e062eda3ea",
    urls = [
        "https://download-ib01.fedoraproject.org/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/v/vim-minimal-8.2.2146-2.fc32.ppc64le.rpm",
        "https://download-cc-rdu01.fedoraproject.org/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/v/vim-minimal-8.2.2146-2.fc32.ppc64le.rpm",
        "https://d2lzkl7pfhq30w.cloudfront.net/pub/fedora-secondary/updates/32/Everything/ppc64le/Packages/v/vim-minimal-8.2.2146-2.fc32.ppc64le.rpm",
        "https://pubmirror2.math.uh.edu/fedora-buffet/fedora-secondary/updates/32/Everything/ppc64le/Packages/v/vim-minimal-8.2.2146-2.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/134adf9ed3ae4e375ef7f1f6d2299e63a940a2fc6f7655c640ae11e062eda3ea",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2146-2.fc32.x86_64",
    sha256 = "1cf36a5d4a96954167ebd75ca34a21b0b6fd00a7935820528b515ab936ee6393",
    urls = [
        "https://mirror.ette.biz/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://sjc.edge.kernel.org/fedora-buffet/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://mirror.genesisadaptive.com/fedora/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://mirror.umd.edu/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2146-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1cf36a5d4a96954167ebd75ca34a21b0b6fd00a7935820528b515ab936ee6393",
    ],
)

rpm(
    name = "virt-what-0__1.20-2.fc32.x86_64",
    sha256 = "36f626b9f8c7fe218b893333385756681d18ffb4f888c8f14d1c1aae5e8df465",
    urls = [
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/v/virt-what-1.20-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/v/virt-what-1.20-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/v/virt-what-1.20-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/v/virt-what-1.20-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/36f626b9f8c7fe218b893333385756681d18ffb4f888c8f14d1c1aae5e8df465",
    ],
)

rpm(
    name = "which-0__2.21-19.fc32.x86_64",
    sha256 = "82e0d8f1e0dccc6d18acd04b7806350343140d9c91da7a216f93167dcf650a61",
    urls = [
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82e0d8f1e0dccc6d18acd04b7806350343140d9c91da7a216f93167dcf650a61",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.29-1.fc32.x86_64",
    sha256 = "ec12fef82d73314e3e4cb6e962f8de27e78989fa104dde0599a4480a53817647",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ec12fef82d73314e3e4cb6e962f8de27e78989fa104dde0599a4480a53817647",
    ],
)

rpm(
    name = "xml-common-0__0.6.3-54.fc32.x86_64",
    sha256 = "e2d176040423fe41e45d14a89ca079415ff811f932431c2cbee22faa476f5e47",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xml-common-0.6.3-54.fc32.noarch.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xml-common-0.6.3-54.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xml-common-0.6.3-54.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xml-common-0.6.3-54.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e2d176040423fe41e45d14a89ca079415ff811f932431c2cbee22faa476f5e47",
    ],
)

rpm(
    name = "xz-0__5.2.5-1.fc32.x86_64",
    sha256 = "1bdde5dc99a5588a8983f70b7b3e45e7006215d529c72adfec118c3bcbf7b01c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1bdde5dc99a5588a8983f70b7b3e45e7006215d529c72adfec118c3bcbf7b01c",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-1.fc32.ppc64le",
    sha256 = "f2eccae89552646dec2475f00e8a66139052c8d2a6628a8ea17628432efdd2a6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/x/xz-libs-5.2.5-1.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/x/xz-libs-5.2.5-1.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/x/xz-libs-5.2.5-1.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/x/xz-libs-5.2.5-1.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/f2eccae89552646dec2475f00e8a66139052c8d2a6628a8ea17628432efdd2a6",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-1.fc32.x86_64",
    sha256 = "84702d6395a9577c1a268184f123cfd4b15bc2287f01033625ba388a34ec2338",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84702d6395a9577c1a268184f123cfd4b15bc2287f01033625ba388a34ec2338",
    ],
)

rpm(
    name = "yajl-0__2.1.0-14.fc32.x86_64",
    sha256 = "9194788f87e4a1aa8835f1305d290cc2cd67cee6a5b1ab82643d3a068c0145b6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://ftp.wrz.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9194788f87e4a1aa8835f1305d290cc2cd67cee6a5b1ab82643d3a068c0145b6",
    ],
)

rpm(
    name = "zlib-0__1.2.11-21.fc32.ppc64le",
    sha256 = "0834015f23c15b3fd40ffac92e9ac3d3aaed9c33b7bccbbb493f45ea6540958a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/z/zlib-1.2.11-21.fc32.ppc64le.rpm",
        "https://fr2.rpmfind.net/linux/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/z/zlib-1.2.11-21.fc32.ppc64le.rpm",
        "https://ftp.icm.edu.pl/pub/Linux/dist/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/z/zlib-1.2.11-21.fc32.ppc64le.rpm",
        "https://mirrors.dotsrc.org/fedora-buffet/fedora-secondary/releases/32/Everything/ppc64le/os/Packages/z/zlib-1.2.11-21.fc32.ppc64le.rpm",
        "https://storage.googleapis.com/builddeps/0834015f23c15b3fd40ffac92e9ac3d3aaed9c33b7bccbbb493f45ea6540958a",
    ],
)

rpm(
    name = "zlib-0__1.2.11-21.fc32.x86_64",
    sha256 = "c0fff40dc1092e18ed3e608bc6143c89a0d7775b9e0553319bb2caca7d324d80",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c0fff40dc1092e18ed3e608bc6143c89a0d7775b9e0553319bb2caca7d324d80",
    ],
)
