workspace(name = "kubevirt")

load("//third_party:deps.bzl", "deps")

deps()

# register crosscompiler toolchains
load("//bazel/toolchain:toolchain.bzl", "register_all_toolchains")

register_all_toolchains()

load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "rules_python",
    sha256 = "934c9ceb552e84577b0faf1e5a2f0450314985b4d8712b2b70717dc679fdc01b",
    urls = [
        "https://github.com/bazelbuild/rules_python/releases/download/0.3.0/rules_python-0.3.0.tar.gz",
        "https://storage.googleapis.com/builddeps/934c9ceb552e84577b0faf1e5a2f0450314985b4d8712b2b70717dc679fdc01b",
    ],
)

# Additional bazel rules
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

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
    sha256 = "69de5c704a05ff37862f7e0f5534d4f479418afc21806c887db544a316f3cb6b",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
        "https://storage.googleapis.com/builddeps/69de5c704a05ff37862f7e0f5534d4f479418afc21806c887db544a316f3cb6b",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "62ca106be173579c0a167deb23358fdfe71ffa1e4cfdddf5582af26520f1c66f",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
        "https://storage.googleapis.com/builddeps/62ca106be173579c0a167deb23358fdfe71ffa1e4cfdddf5582af26520f1c66f",
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
    sha256 = "80ba082177c93e43a7c085a8566c7f11654dbae41da7da0da52e0ed2e917cd12",
    strip_prefix = "bazel-tools-6e2a416f565062955735edcfae881cdba2b7abf7",
    urls = [
        "https://github.com/ash2k/bazel-tools/archive/6e2a416f565062955735edcfae881cdba2b7abf7.zip",
        "https://storage.googleapis.com/builddeps/80ba082177c93e43a7c085a8566c7f11654dbae41da7da0da52e0ed2e917cd12",
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
    name = "alpine_image_aarch64",
    sha256 = "1a37be6e94bf2e102e9997c3c5bd433787299640a42d3df1597fd9dc5e1a81c7",
    urls = [
        "http://dl-cdn.alpinelinux.org/alpine/v3.11/releases/aarch64/alpine-virt-3.11.5-aarch64.iso",
        "https://storage.googleapis.com/builddeps/1a37be6e94bf2e102e9997c3c5bd433787299640a42d3df1597fd9dc5e1a81c7",
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
    name = "cirros_image_aarch64",
    sha256 = "46c4bd31c1b39152bafe3265c8e3551dd6bc672dfee6713dc736f5e20a348e63",
    urls = [
        "https://download.cirros-cloud.net/0.5.0/cirros-0.5.0-aarch64-disk.img",
        "https://storage.googleapis.com/builddeps/46c4bd31c1b39152bafe3265c8e3551dd6bc672dfee6713dc736f5e20a348e63",
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
    name = "microlivecd_image_aarch64",
    sha256 = "2d9a7790fa6347251aacd997384b30962bc60dfe4eb9f0c2bd76b42f54d04b8d",
    urls = [
        "https://github.com/jean-edouard/microlivecd/releases/download/0.2/microlivecd_arm64.iso",
        "https://storage.googleapis.com/builddeps/2d9a7790fa6347251aacd997384b30962bc60dfe4eb9f0c2bd76b42f54d04b8d",
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
    sha256 = "6a2af09c6a598a3c4e4fec9af78334fbec2b3c16473f4e2c692fe2e567dc6f56",
    strip_prefix = "bazeldnf-0.5.1",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.5.1.tar.gz",
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
    go_version = "1.16.6",
    nogo = "@//:nogo_vet",
)

load("@com_github_ash2k_bazel_tools//goimports:deps.bzl", "goimports_dependencies")

goimports_dependencies()

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

# override rules_docker issue with this dependency
# rules_docker 0.16 uses 0.1.4, bit since there the checksum changed, which is very weird, going with 0.1.4.1 to
go_repository(
    name = "com_github_google_go_containerregistry",
    importpath = "github.com/google/go-containerregistry",
    sha256 = "bc0136a33f9c1e4578a700f7afcdaa1241cfff997d6bba695c710d24c5ae26bd",
    strip_prefix = "google-go-containerregistry-efb2d62",
    type = "tar.gz",
    urls = ["https://api.github.com/repos/google/go-containerregistry/tarball/efb2d62d93a7705315b841d0544cb5b13565ff2a"],  # v0.1.4.1
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

# Pull base image fedora31
# WARNING: please update any automated process to push this image to quay.io
# instead of index.docker.io
container_pull(
    name = "fedora",
    digest = "sha256:5e2b864cfe165fa7da6606b29a9e60549eb7cc9ae7fb574614110d1494b0f0c2",
    registry = "quay.io",
    repository = "kubevirtci/fedora",
    tag = "31",
)

# As rpm package in https://dl.fedoraproject.org/pub/fedora/linux/releases/31 is empty, we use fedora 32 here.
# TODO add fedora image to quay.io
container_pull(
    name = "fedora_aarch64",
    digest = "sha256:425676dd30f2c85ba3593b82040ce03341cd6dc4e38838e57c8bc5eef95b5f81",
    registry = "index.docker.io",
    repository = "library/fedora",
    tag = "32",
)

# Pull go_image_base
container_pull(
    name = "go_image_base",
    digest = "sha256:f65536ce108fcc41cdcd5cb101006fcb82b9a1527409263feb9e34032f00bda0",
    registry = "gcr.io",
    repository = "distroless/base",
)

container_pull(
    name = "go_image_base_aarch64",
    digest = "sha256:789c477fbd30a7d85435450306e54f20c53938e40af644284a229d852db30dde",
    registry = "gcr.io",
    repository = "distroless/base",
)

# Pull nfs-server image
# WARNING: please update any automated process to push this image to quay.io
# instead of index.docker.io
# TODO build nfs-server for multi-arch
container_pull(
    name = "nfs-server",
    digest = "sha256:8c1fa882dddb2885c4152e9ce632c466f4b8dce29339455e9b6bfe71f0a3d3ef",
    registry = "quay.io",
    repository = "kubevirtci/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
)

container_pull(
    name = "nfs-server_aarch64",
    digest = "sha256:8c1fa882dddb2885c4152e9ce632c466f4b8dce29339455e9b6bfe71f0a3d3ef",
    registry = "quay.io",
    repository = "kubevirtci/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
)

# Pull fedora container-disk preconfigured with ci tooling
# like stress and qemu guest agent pre-configured
# TODO build fedora_with_test_tooling for multi-arch
container_pull(
    name = "fedora_with_test_tooling",
    digest = "sha256:14193941e1fe74f2189536263c71479abbd296dc93b75e8a7f97f0b31e78b71e",
    registry = "quay.io",
    repository = "kubevirtci/fedora-with-test-tooling",
)

container_pull(
    name = "fedora_with_test_tooling_aarch64",
    digest = "sha256:9ec3e137bff093597d192f5a4e346f25b614c3a94216b857de0e3d75b68bfb17",
    registry = "quay.io",
    repository = "kubevirt/fedora-with-test-tooling",
)

container_pull(
    name = "alpine-ext-kernel-boot-demo-container-base",
    digest = "sha256:a2ddb2f568bf3814e594a14bc793d5a655a61d5983f3561d60d02afa7bbc56b4",
    registry = "quay.io",
    repository = "kubevirt/alpine-ext-kernel-boot-demo",
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

http_file(
    name = "libguestfs-appliance",
    sha256 = "3e49a80d688b165ce8535369aa352f80eb832001f0ae6f2e1af1b4e9986421c5",
    urls = [
        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/appliance-1.44.0-linux-5.11.22-100-fedora32.tar.xz",
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
    name = "acl-0__2.2.53-5.fc32.aarch64",
    sha256 = "e8941c0abaa3ce527b14bc19013088149be9c5aacceb788718293cdef9132d18",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/acl-2.2.53-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/a/acl-2.2.53-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/acl-2.2.53-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/acl-2.2.53-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e8941c0abaa3ce527b14bc19013088149be9c5aacceb788718293cdef9132d18",
    ],
)

rpm(
    name = "acl-0__2.2.53-5.fc32.x86_64",
    sha256 = "705bdb96aab3a0f9d9e2ff48ead1208e2dbc1927d713d8637632af936235217b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/acl-2.2.53-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/705bdb96aab3a0f9d9e2ff48ead1208e2dbc1927d713d8637632af936235217b",
    ],
)

rpm(
    name = "alternatives-0__1.11-6.fc32.aarch64",
    sha256 = "10d828cc7803aca9b59e3bb9b52e0af45a2828250f1eab7f0fc08cdb981f191d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/alternatives-1.11-6.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/a/alternatives-1.11-6.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/alternatives-1.11-6.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/alternatives-1.11-6.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/10d828cc7803aca9b59e3bb9b52e0af45a2828250f1eab7f0fc08cdb981f191d",
    ],
)

rpm(
    name = "alternatives-0__1.11-6.fc32.x86_64",
    sha256 = "c574c5432197acbe08ea15c7837be7577cd0b49902a3e65227792f051d73ce5c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/alternatives-1.11-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c574c5432197acbe08ea15c7837be7577cd0b49902a3e65227792f051d73ce5c",
    ],
)

rpm(
    name = "attr-0__2.4.48-8.fc32.x86_64",
    sha256 = "00d0bb6a08f20bea2b6bd0d2c4de99b51c770b2dab266d1d3da85891efeded01",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/attr-2.4.48-8.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/attr-2.4.48-8.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/attr-2.4.48-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/attr-2.4.48-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/00d0bb6a08f20bea2b6bd0d2c4de99b51c770b2dab266d1d3da85891efeded01",
    ],
)

rpm(
    name = "audit-libs-0__3.0.1-2.fc32.aarch64",
    sha256 = "8532c3f01b7ff237c891f18ccb7d3efb26e55dd88fd3d74662ab16ca548ba865",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/a/audit-libs-3.0.1-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/a/audit-libs-3.0.1-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/a/audit-libs-3.0.1-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/a/audit-libs-3.0.1-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8532c3f01b7ff237c891f18ccb7d3efb26e55dd88fd3d74662ab16ca548ba865",
    ],
)

rpm(
    name = "audit-libs-0__3.0.1-2.fc32.x86_64",
    sha256 = "a3e2a70974370ab574d5157717323750f3e06a08d997fa95b0f72fca10eefdfc",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0.1-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0.1-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0.1-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/a/audit-libs-3.0.1-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a3e2a70974370ab574d5157717323750f3e06a08d997fa95b0f72fca10eefdfc",
    ],
)

rpm(
    name = "augeas-libs-0__1.12.0-3.fc32.x86_64",
    sha256 = "90e322415ad6e6b7b120d28ae34a37d9f89bb25a3c79936a34a2ce7bd67cffaf",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/augeas-libs-1.12.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/augeas-libs-1.12.0-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/augeas-libs-1.12.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/augeas-libs-1.12.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/90e322415ad6e6b7b120d28ae34a37d9f89bb25a3c79936a34a2ce7bd67cffaf",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.16-4.fc32.aarch64",
    sha256 = "8335b0a64a2db879d2abf477abea414b9a8889d9f859f1d4f64c001815f88cb2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8335b0a64a2db879d2abf477abea414b9a8889d9f859f1d4f64c001815f88cb2",
    ],
)

rpm(
    name = "autogen-libopts-0__5.18.16-4.fc32.x86_64",
    sha256 = "df529905e3527b66b059518c181512396e7cbc0e07fc8710dc53d3565941bf65",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/a/autogen-libopts-5.18.16-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/df529905e3527b66b059518c181512396e7cbc0e07fc8710dc53d3565941bf65",
    ],
)

rpm(
    name = "basesystem-0__11-9.fc32.aarch64",
    sha256 = "a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    ],
)

rpm(
    name = "basesystem-0__11-9.fc32.x86_64",
    sha256 = "a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/basesystem-11-9.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a346990bb07adca8c323a15f31b093ef6e639bde6ca84adf1a3abebc4dc9adce",
    ],
)

rpm(
    name = "bash-0__5.0.17-1.fc32.aarch64",
    sha256 = "6573d9dd93a1f3204f33f2f3b899e953e68b750b3c114fa9462f528ed13b89cb",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/b/bash-5.0.17-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/b/bash-5.0.17-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/b/bash-5.0.17-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/b/bash-5.0.17-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6573d9dd93a1f3204f33f2f3b899e953e68b750b3c114fa9462f528ed13b89cb",
    ],
)

rpm(
    name = "bash-0__5.0.17-1.fc32.x86_64",
    sha256 = "31d92d4ef9080bd349188c6f835db0f8b7cf3fe57c6dcff37582f9ee14860ec0",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/bash-5.0.17-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/31d92d4ef9080bd349188c6f835db0f8b7cf3fe57c6dcff37582f9ee14860ec0",
    ],
)

rpm(
    name = "binutils-0__2.34-6.fc32.x86_64",
    sha256 = "0605263ffb280a7f4281e14be1b2e12c5f8445256f925c1fdee6ebabd492c74c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-2.34-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-2.34-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-2.34-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-2.34-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0605263ffb280a7f4281e14be1b2e12c5f8445256f925c1fdee6ebabd492c74c",
    ],
)

rpm(
    name = "binutils-gold-0__2.34-6.fc32.x86_64",
    sha256 = "9d0d05430860b629f3831a039c719ac4c12a7cdc0ebda6030ee1d1cafae8d412",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-gold-2.34-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-gold-2.34-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-gold-2.34-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/binutils-gold-2.34-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d0d05430860b629f3831a039c719ac4c12a7cdc0ebda6030ee1d1cafae8d412",
    ],
)

rpm(
    name = "bridge-utils-0__1.6-5.fc32.x86_64",
    sha256 = "09a933edc58e851f731b08d0258cd0b61d8caa6f479fe565be3c9cccd96dc06e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bridge-utils-1.6-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bridge-utils-1.6-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bridge-utils-1.6-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bridge-utils-1.6-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09a933edc58e851f731b08d0258cd0b61d8caa6f479fe565be3c9cccd96dc06e",
    ],
)

rpm(
    name = "btrfs-progs-0__5.11.1-1.fc32.x86_64",
    sha256 = "998848c39cabd3b61c6a7d0b25f6c3bb230bebeafac07ccceaf2ece7ec08af4f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/btrfs-progs-5.11.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/btrfs-progs-5.11.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/b/btrfs-progs-5.11.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/b/btrfs-progs-5.11.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/998848c39cabd3b61c6a7d0b25f6c3bb230bebeafac07ccceaf2ece7ec08af4f",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-2.fc32.aarch64",
    sha256 = "44a39ebda134e725f49d7467795c58fb67fce745e17ec9a6e01e89ee12145508",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/bzip2-1.0.8-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/b/bzip2-1.0.8-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/bzip2-1.0.8-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/bzip2-1.0.8-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/44a39ebda134e725f49d7467795c58fb67fce745e17ec9a6e01e89ee12145508",
    ],
)

rpm(
    name = "bzip2-0__1.0.8-2.fc32.x86_64",
    sha256 = "b6601c8208b1fa3c41b582bd648c737798bf639da1a049efc0e78c37058280f2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-1.0.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b6601c8208b1fa3c41b582bd648c737798bf639da1a049efc0e78c37058280f2",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-2.fc32.aarch64",
    sha256 = "caf76966e150fbe796865d2d18479b080657cb0bada9283048a4586cf034d4e6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/caf76966e150fbe796865d2d18479b080657cb0bada9283048a4586cf034d4e6",
    ],
)

rpm(
    name = "bzip2-libs-0__1.0.8-2.fc32.x86_64",
    sha256 = "842f7a38be2e8dbb14eff3ede4091db214ebe241e1fde7a128e88c4e686b63b0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/b/bzip2-libs-1.0.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/842f7a38be2e8dbb14eff3ede4091db214ebe241e1fde7a128e88c4e686b63b0",
    ],
)

rpm(
    name = "ca-certificates-0__2020.2.41-1.1.fc32.aarch64",
    sha256 = "0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    ],
)

rpm(
    name = "ca-certificates-0__2020.2.41-1.1.fc32.x86_64",
    sha256 = "0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/ca-certificates-2020.2.41-1.1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0a87bedd7687620ce85224027c0cfebc603b92962f67db432eb5a7b00d405cde",
    ],
)

rpm(
    name = "checkpolicy-0__3.0-3.fc32.aarch64",
    sha256 = "ad6f711174c59ffb9116d792068cc8fd0585b46eb5d9bf18a3c9937727b9a379",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/checkpolicy-3.0-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/c/checkpolicy-3.0-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/checkpolicy-3.0-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/checkpolicy-3.0-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad6f711174c59ffb9116d792068cc8fd0585b46eb5d9bf18a3c9937727b9a379",
    ],
)

rpm(
    name = "checkpolicy-0__3.0-3.fc32.x86_64",
    sha256 = "703fb5ca1651bb72d8ab58576ce3d78c9479cbb2e78ff8666ae3a3d1cd9bb0da",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/checkpolicy-3.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/checkpolicy-3.0-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/checkpolicy-3.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/checkpolicy-3.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/703fb5ca1651bb72d8ab58576ce3d78c9479cbb2e78ff8666ae3a3d1cd9bb0da",
    ],
)

rpm(
    name = "compat-readline5-0__5.2-36.fc32.x86_64",
    sha256 = "449d2888d6b835d207a55a2d9b4478eff1b926581fcead6260b6508e4db1b782",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/compat-readline5-5.2-36.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/compat-readline5-5.2-36.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/compat-readline5-5.2-36.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/compat-readline5-5.2-36.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/449d2888d6b835d207a55a2d9b4478eff1b926581fcead6260b6508e4db1b782",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-4.fc32.2.aarch64",
    sha256 = "dd887703ae5bd046631e57095f1fa421a121d09880cbd173d58dc82411b8544b",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/coreutils-single-8.32-4.fc32.2.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/coreutils-single-8.32-4.fc32.2.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/c/coreutils-single-8.32-4.fc32.2.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/c/coreutils-single-8.32-4.fc32.2.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dd887703ae5bd046631e57095f1fa421a121d09880cbd173d58dc82411b8544b",
    ],
)

rpm(
    name = "coreutils-single-0__8.32-4.fc32.2.x86_64",
    sha256 = "5bb4cd5c46fde994f72998b37c9ef17654f3f91614d450a340ce8b1233d1a422",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.2.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.2.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.2.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/coreutils-single-8.32-4.fc32.2.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5bb4cd5c46fde994f72998b37c9ef17654f3f91614d450a340ce8b1233d1a422",
    ],
)

rpm(
    name = "cpio-0__2.13-6.fc32.x86_64",
    sha256 = "6a47ab3a8c8fcb5a9fe629fed1a1ecc4ca878d12e828fca2b0099dab15ca0148",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cpio-2.13-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cpio-2.13-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/cpio-2.13-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cpio-2.13-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a47ab3a8c8fcb5a9fe629fed1a1ecc4ca878d12e828fca2b0099dab15ca0148",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-22.fc32.aarch64",
    sha256 = "081d831528796c3e5c47b89c363a0f530bf77e3e2e0098cd586d814bea9a12f0",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cracklib-2.9.6-22.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/c/cracklib-2.9.6-22.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cracklib-2.9.6-22.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cracklib-2.9.6-22.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/081d831528796c3e5c47b89c363a0f530bf77e3e2e0098cd586d814bea9a12f0",
    ],
)

rpm(
    name = "cracklib-0__2.9.6-22.fc32.x86_64",
    sha256 = "862e75c10377098a9cc50407a0395e5f3a81d14b5b6fecfb3f223325c8867829",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cracklib-2.9.6-22.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/862e75c10377098a9cc50407a0395e5f3a81d14b5b6fecfb3f223325c8867829",
    ],
)

rpm(
    name = "crypto-policies-0__20200619-1.git781bbd4.fc32.aarch64",
    sha256 = "de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    ],
)

rpm(
    name = "crypto-policies-0__20200619-1.git781bbd4.fc32.x86_64",
    sha256 = "de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/crypto-policies-20200619-1.git781bbd4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/de8a3bb7cc8634b62e359fabfd2f8e07065b97fb3d6ce974dd3875c7bbd75683",
    ],
)

rpm(
    name = "cryptsetup-0__2.3.5-2.fc32.x86_64",
    sha256 = "469b4f319e409f480ab3adec717abb440682ae7e13aa2d764b8b7a254bcd5da1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-2.3.5-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-2.3.5-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-2.3.5-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-2.3.5-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/469b4f319e409f480ab3adec717abb440682ae7e13aa2d764b8b7a254bcd5da1",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.5-2.fc32.aarch64",
    sha256 = "7255c4ac3193e07b689308094368fb8e8b4c03cae258016a5147ce6e98f4adb4",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7255c4ac3193e07b689308094368fb8e8b4c03cae258016a5147ce6e98f4adb4",
    ],
)

rpm(
    name = "cryptsetup-libs-0__2.3.5-2.fc32.x86_64",
    sha256 = "23481b5ed3b47a509bdb15a7c8898cb4631181b7bc3b058af062fcd25c505139",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/cryptsetup-libs-2.3.5-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23481b5ed3b47a509bdb15a7c8898cb4631181b7bc3b058af062fcd25c505139",
    ],
)

rpm(
    name = "curl-minimal-0__7.69.1-8.fc32.aarch64",
    sha256 = "2079033b266c0d9eb662d89d7884643879271675a1536c2cc08377af02d7acfc",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/curl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/c/curl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/c/curl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/c/curl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2079033b266c0d9eb662d89d7884643879271675a1536c2cc08377af02d7acfc",
    ],
)

rpm(
    name = "curl-minimal-0__7.69.1-8.fc32.x86_64",
    sha256 = "86af7207a8beae934caa17a8308618c909ad10d7cfe9eb8f97281d4291a57fe4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/c/curl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86af7207a8beae934caa17a8308618c909ad10d7cfe9eb8f97281d4291a57fe4",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-4.fc32.aarch64",
    sha256 = "5f2f0e765440c2514be906c46e7edd82ed70ba3eea20eafe832a22495e88f0f0",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5f2f0e765440c2514be906c46e7edd82ed70ba3eea20eafe832a22495e88f0f0",
    ],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-4.fc32.x86_64",
    sha256 = "2878e39b646fc03b9405c99987598da839acd126db5dcfc1ce0c51107993a9ae",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-2.1.27-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2878e39b646fc03b9405c99987598da839acd126db5dcfc1ce0c51107993a9ae",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-4.fc32.aarch64",
    sha256 = "264417fd1b07b1ec1a53fc209f3833de61655e72c48ae49ed658402c4ee505ef",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/264417fd1b07b1ec1a53fc209f3833de61655e72c48ae49ed658402c4ee505ef",
    ],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-4.fc32.x86_64",
    sha256 = "5b24460fa98b7d7004ea7d1d3b688c3b6e328eea1b23881d4c17f872e372d3cb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-gssapi-2.1.27-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5b24460fa98b7d7004ea7d1d3b688c3b6e328eea1b23881d4c17f872e372d3cb",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-4.fc32.aarch64",
    sha256 = "b9904d16c86c28074bfdba38a3a740b61ad5de50a9945d550021027130fcfd41",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b9904d16c86c28074bfdba38a3a740b61ad5de50a9945d550021027130fcfd41",
    ],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-4.fc32.x86_64",
    sha256 = "fefa4162a563eba24714ac43874c508d1ba036afb5127c5d21bbcbeaf238a740",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/c/cyrus-sasl-lib-2.1.27-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fefa4162a563eba24714ac43874c508d1ba036afb5127c5d21bbcbeaf238a740",
    ],
)

rpm(
    name = "dbus-1__1.12.20-1.fc32.aarch64",
    sha256 = "e36a47ff624d27a0a7059bde2fe022302ffe335571ba8cf84e7e5e3646000557",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-1.12.20-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-1.12.20-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-1.12.20-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-1.12.20-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e36a47ff624d27a0a7059bde2fe022302ffe335571ba8cf84e7e5e3646000557",
    ],
)

rpm(
    name = "dbus-1__1.12.20-1.fc32.x86_64",
    sha256 = "0f4bac9a18a2535b85a7b9d8ac4c652edbb0047224f89548122f6f1257a169eb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-1.12.20-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0f4bac9a18a2535b85a7b9d8ac4c652edbb0047224f89548122f6f1257a169eb",
    ],
)

rpm(
    name = "dbus-broker-0__27-2.fc32.aarch64",
    sha256 = "bd3d1c8221895b6fa4f90087ac130233edf285e89d7c225aaf755e8cbc5baed4",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-broker-27-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-broker-27-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-broker-27-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-broker-27-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bd3d1c8221895b6fa4f90087ac130233edf285e89d7c225aaf755e8cbc5baed4",
    ],
)

rpm(
    name = "dbus-broker-0__27-2.fc32.x86_64",
    sha256 = "1169ea08c30c8fed6eded63cf2b2c77d7b4df8575bec971f80ed8d85c231506a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-27-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-27-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-27-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-broker-27-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1169ea08c30c8fed6eded63cf2b2c77d7b4df8575bec971f80ed8d85c231506a",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-1.fc32.aarch64",
    sha256 = "0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    ],
)

rpm(
    name = "dbus-common-1__1.12.20-1.fc32.x86_64",
    sha256 = "0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dbus-common-1.12.20-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0edabb437c55618b1c31ace707e827075eb4ef633d82ffde82f57ff45f0931a3",
    ],
)

rpm(
    name = "device-mapper-0__1.02.171-1.fc32.aarch64",
    sha256 = "18c188f63504b8cf3bc88d95de458a1eb216bca268378a6839618ef7468dc635",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/device-mapper-1.02.171-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/d/device-mapper-1.02.171-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/device-mapper-1.02.171-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/device-mapper-1.02.171-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/18c188f63504b8cf3bc88d95de458a1eb216bca268378a6839618ef7468dc635",
    ],
)

rpm(
    name = "device-mapper-0__1.02.171-1.fc32.x86_64",
    sha256 = "c132999a3f110029cd427f7578965ad558e91374637087d5230ee11c626ebcd4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-1.02.171-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c132999a3f110029cd427f7578965ad558e91374637087d5230ee11c626ebcd4",
    ],
)

rpm(
    name = "device-mapper-event-0__1.02.171-1.fc32.x86_64",
    sha256 = "9a2beeeede69d8910115608c2d98efa6a8dba73ab2df246df5b0f10e2fa37f54",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-1.02.171-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a2beeeede69d8910115608c2d98efa6a8dba73ab2df246df5b0f10e2fa37f54",
    ],
)

rpm(
    name = "device-mapper-event-libs-0__1.02.171-1.fc32.x86_64",
    sha256 = "8aa8258a1a13c1120d6c28321f618385111cb9363dae09eea2e4af481053e28b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-event-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8aa8258a1a13c1120d6c28321f618385111cb9363dae09eea2e4af481053e28b",
    ],
)

rpm(
    name = "device-mapper-libs-0__1.02.171-1.fc32.aarch64",
    sha256 = "5d52cffee2d5360db8cf7e6ed4b19a68de4a0ae55f42ed279d4fdb3a70bb72f3",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5d52cffee2d5360db8cf7e6ed4b19a68de4a0ae55f42ed279d4fdb3a70bb72f3",
    ],
)

rpm(
    name = "device-mapper-libs-0__1.02.171-1.fc32.x86_64",
    sha256 = "61cae80187ef2924857fdfc48a240646d23b331482cf181e7d8c661b02c15949",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-libs-1.02.171-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61cae80187ef2924857fdfc48a240646d23b331482cf181e7d8c661b02c15949",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.2-4.fc32.aarch64",
    sha256 = "b01d3754bb2784fcde52dbaab892726f95cea89aac263444d92cddb7e70e4d45",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b01d3754bb2784fcde52dbaab892726f95cea89aac263444d92cddb7e70e4d45",
    ],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.2-4.fc32.x86_64",
    sha256 = "1513c0cd213973c6939ac22b9faf7b21347559f2702eb398301a2c68b12e0b8c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/device-mapper-multipath-libs-0.8.2-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1513c0cd213973c6939ac22b9faf7b21347559f2702eb398301a2c68b12e0b8c",
    ],
)

rpm(
    name = "device-mapper-persistent-data-0__0.8.5-3.fc32.x86_64",
    sha256 = "7a525abda7230bfbc87763dfe58bf7684e385b3c78ca242a1685a589300909e9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/device-mapper-persistent-data-0.8.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7a525abda7230bfbc87763dfe58bf7684e385b3c78ca242a1685a589300909e9",
    ],
)

rpm(
    name = "dhcp-client-12__4.4.2-5.b1.fc32.x86_64",
    sha256 = "8c047278ce11e7ba7169a9c9537c3c6544f52c7eee8754179acd839ed59c028f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-client-4.4.2-5.b1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-client-4.4.2-5.b1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-client-4.4.2-5.b1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-client-4.4.2-5.b1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8c047278ce11e7ba7169a9c9537c3c6544f52c7eee8754179acd839ed59c028f",
    ],
)

rpm(
    name = "dhcp-common-12__4.4.2-5.b1.fc32.x86_64",
    sha256 = "48c36b8950b7e79e5f36317da9349392dae0b739177e9c60653a8a1a62814013",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-common-4.4.2-5.b1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-common-4.4.2-5.b1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-common-4.4.2-5.b1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dhcp-common-4.4.2-5.b1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48c36b8950b7e79e5f36317da9349392dae0b739177e9c60653a8a1a62814013",
    ],
)

rpm(
    name = "diffutils-0__3.7-4.fc32.aarch64",
    sha256 = "13290758e03b977aed5e23b7ba9a01157b6802fd78baf75bc1fc184864e9e31e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/diffutils-3.7-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/d/diffutils-3.7-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/diffutils-3.7-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/d/diffutils-3.7-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/13290758e03b977aed5e23b7ba9a01157b6802fd78baf75bc1fc184864e9e31e",
    ],
)

rpm(
    name = "diffutils-0__3.7-4.fc32.x86_64",
    sha256 = "187dd61be71efcca6adf9819a523d432217abb335afcb2b95ef27b72928aff4b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/diffutils-3.7-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/187dd61be71efcca6adf9819a523d432217abb335afcb2b95ef27b72928aff4b",
    ],
)

rpm(
    name = "dmidecode-1__3.2-5.fc32.x86_64",
    sha256 = "e40be03bd5808e640bb5fb18196499680a7b7b1d3fce47617f987baee849c0e5",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dmidecode-3.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e40be03bd5808e640bb5fb18196499680a7b7b1d3fce47617f987baee849c0e5",
    ],
)

rpm(
    name = "dnf-0__4.6.0-1.fc32.x86_64",
    sha256 = "72ba0d0e686458fbe109ca133ba9bae466a7f690832c82c5dac7130162611682",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-4.6.0-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-4.6.0-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-4.6.0-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-4.6.0-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/72ba0d0e686458fbe109ca133ba9bae466a7f690832c82c5dac7130162611682",
    ],
)

rpm(
    name = "dnf-plugins-core-0__4.0.18-1.fc32.x86_64",
    sha256 = "b3c435c9084c3b0e3b69a8ed4c6337d785a9bd70ab5ad15c0dad117efb1a666b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.18-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.18-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.18-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnf-plugins-core-4.0.18-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b3c435c9084c3b0e3b69a8ed4c6337d785a9bd70ab5ad15c0dad117efb1a666b",
    ],
)

rpm(
    name = "dnsmasq-0__2.85-1.fc32.x86_64",
    sha256 = "ed1a9727d5a94b2cecab876b33913f71af45baa3911f90a4c69ba87a74bef76a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnsmasq-2.85-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnsmasq-2.85-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnsmasq-2.85-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/d/dnsmasq-2.85-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed1a9727d5a94b2cecab876b33913f71af45baa3911f90a4c69ba87a74bef76a",
    ],
)

rpm(
    name = "dosfstools-0__4.1-10.fc32.x86_64",
    sha256 = "c3f7089ae50f79cf4d2cb59e01091d33c70cab89ae08f95c547339a87404c3ec",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dosfstools-4.1-10.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dosfstools-4.1-10.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dosfstools-4.1-10.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/d/dosfstools-4.1-10.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c3f7089ae50f79cf4d2cb59e01091d33c70cab89ae08f95c547339a87404c3ec",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.5-3.fc32.aarch64",
    sha256 = "d3281a3ef4de5e13ef1a76effd68169c0965467039059141609a078520f3db04",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d3281a3ef4de5e13ef1a76effd68169c0965467039059141609a078520f3db04",
    ],
)

rpm(
    name = "e2fsprogs-0__1.45.5-3.fc32.x86_64",
    sha256 = "2fa5e252441852dae918b522a2ff3f46a5bbee4ce8936e06702bf65f57d7ff99",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2fa5e252441852dae918b522a2ff3f46a5bbee4ce8936e06702bf65f57d7ff99",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.5-3.fc32.aarch64",
    sha256 = "7f667fb609062e966720bf1bb1fa97a91ca245925c68e36d2770caba57aa4db2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7f667fb609062e966720bf1bb1fa97a91ca245925c68e36d2770caba57aa4db2",
    ],
)

rpm(
    name = "e2fsprogs-libs-0__1.45.5-3.fc32.x86_64",
    sha256 = "26db62c2bc52c3eee5f3039cdbdf19498f675d0f45aec0c2a1c61c635f01479e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/e2fsprogs-libs-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/26db62c2bc52c3eee5f3039cdbdf19498f675d0f45aec0c2a1c61c635f01479e",
    ],
)

rpm(
    name = "edk2-aarch64-0__20200801stable-1.fc32.aarch64",
    sha256 = "672d0a48607a2b2532f31647163fab0a659a8a9e02f6d9abfe626b75e4d8f7f8",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/e/edk2-aarch64-20200801stable-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/672d0a48607a2b2532f31647163fab0a659a8a9e02f6d9abfe626b75e4d8f7f8",
    ],
)

rpm(
    name = "edk2-ovmf-0__20200801stable-1.fc32.x86_64",
    sha256 = "220848c197bd0e464172b94a4af9b681f788a916976d05036825241e1aaf753a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/edk2-ovmf-20200801stable-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/220848c197bd0e464172b94a4af9b681f788a916976d05036825241e1aaf753a",
    ],
)

rpm(
    name = "elfutils-debuginfod-client-0__0.183-1.fc32.x86_64",
    sha256 = "44bd353c3550b6c605ca90f2f949a84a99f72072a6f83a49874f962b849febf9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.183-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.183-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.183-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-debuginfod-client-0.183-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/44bd353c3550b6c605ca90f2f949a84a99f72072a6f83a49874f962b849febf9",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.183-1.fc32.aarch64",
    sha256 = "d163b7ae73ba9bc1760988833bdbbfce5ceaa99e53b9aba8e2392ec35ab4a004",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d163b7ae73ba9bc1760988833bdbbfce5ceaa99e53b9aba8e2392ec35ab4a004",
    ],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.183-1.fc32.x86_64",
    sha256 = "d163b7ae73ba9bc1760988833bdbbfce5ceaa99e53b9aba8e2392ec35ab4a004",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-default-yama-scope-0.183-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d163b7ae73ba9bc1760988833bdbbfce5ceaa99e53b9aba8e2392ec35ab4a004",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.183-1.fc32.aarch64",
    sha256 = "854c4722d44389841da0111ed6b55dba7c5f2b442390022426581fe75f9fae84",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libelf-0.183-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libelf-0.183-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libelf-0.183-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libelf-0.183-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/854c4722d44389841da0111ed6b55dba7c5f2b442390022426581fe75f9fae84",
    ],
)

rpm(
    name = "elfutils-libelf-0__0.183-1.fc32.x86_64",
    sha256 = "d3529f0d1e385ba4411f1afd0e2b4a6f34636ed75f795242f552aaccdfb34fc5",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.183-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.183-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.183-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libelf-0.183-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d3529f0d1e385ba4411f1afd0e2b4a6f34636ed75f795242f552aaccdfb34fc5",
    ],
)

rpm(
    name = "elfutils-libs-0__0.183-1.fc32.aarch64",
    sha256 = "794ece1ddd3279f799b4a9440abdad94430cac1a93ac3e200d2ca91b0879a296",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libs-0.183-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libs-0.183-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libs-0.183-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/e/elfutils-libs-0.183-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/794ece1ddd3279f799b4a9440abdad94430cac1a93ac3e200d2ca91b0879a296",
    ],
)

rpm(
    name = "elfutils-libs-0__0.183-1.fc32.x86_64",
    sha256 = "8d63771abe3810f232a512b1ca432b615d6c4a5d4c3845724b4200cf14cd158a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.183-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.183-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.183-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/e/elfutils-libs-0.183-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8d63771abe3810f232a512b1ca432b615d6c4a5d4c3845724b4200cf14cd158a",
    ],
)

rpm(
    name = "expat-0__2.2.8-2.fc32.aarch64",
    sha256 = "4940f6e26a93fe638667adb6e12969fe915b3a7b0cfeb58877dd6d7bccf46c1a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/expat-2.2.8-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/e/expat-2.2.8-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/expat-2.2.8-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/e/expat-2.2.8-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4940f6e26a93fe638667adb6e12969fe915b3a7b0cfeb58877dd6d7bccf46c1a",
    ],
)

rpm(
    name = "expat-0__2.2.8-2.fc32.x86_64",
    sha256 = "8fc2ae85f242105987d8fa7f05e4fa19358a7c81dff5fa163cf021eb6b9905e9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/e/expat-2.2.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8fc2ae85f242105987d8fa7f05e4fa19358a7c81dff5fa163cf021eb6b9905e9",
    ],
)

rpm(
    name = "fedora-gpg-keys-0__32-13.aarch64",
    sha256 = "22f0cc59f3d312291a54df89e1d0e4615c4c51575c11eb7e736705f1927e148f",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://storage.googleapis.com/builddeps/22f0cc59f3d312291a54df89e1d0e4615c4c51575c11eb7e736705f1927e148f",
    ],
)

rpm(
    name = "fedora-gpg-keys-0__32-13.x86_64",
    sha256 = "22f0cc59f3d312291a54df89e1d0e4615c4c51575c11eb7e736705f1927e148f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-gpg-keys-32-13.noarch.rpm",
        "https://storage.googleapis.com/builddeps/22f0cc59f3d312291a54df89e1d0e4615c4c51575c11eb7e736705f1927e148f",
    ],
)

rpm(
    name = "fedora-logos-httpd-0__30.0.2-4.fc32.aarch64",
    sha256 = "458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    ],
)

rpm(
    name = "fedora-logos-httpd-0__30.0.2-4.fc32.x86_64",
    sha256 = "458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fedora-logos-httpd-30.0.2-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/458d5c1745ca1c0f428fc99308e8089df64024bb75e6528ba5a02fb11a2e8af7",
    ],
)

rpm(
    name = "fedora-release-common-0__32-4.aarch64",
    sha256 = "829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    ],
)

rpm(
    name = "fedora-release-common-0__32-4.x86_64",
    sha256 = "829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-common-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/829b134f82e478fafdca34d407489f26b59e2ddf457e5a02dade40faa84034c6",
    ],
)

rpm(
    name = "fedora-release-container-0__32-4.aarch64",
    sha256 = "21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    ],
)

rpm(
    name = "fedora-release-container-0__32-4.x86_64",
    sha256 = "21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-release-container-32-4.noarch.rpm",
        "https://storage.googleapis.com/builddeps/21394dc70614bc031f60888c8070d67b9a5a434cc409059e755e7dc8cf515cb0",
    ],
)

rpm(
    name = "fedora-repos-0__32-13.aarch64",
    sha256 = "d2a2c4166d673deaf4ef60c943aba4296880350b6a670318d504d33c36f55b72",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d2a2c4166d673deaf4ef60c943aba4296880350b6a670318d504d33c36f55b72",
    ],
)

rpm(
    name = "fedora-repos-0__32-13.x86_64",
    sha256 = "d2a2c4166d673deaf4ef60c943aba4296880350b6a670318d504d33c36f55b72",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/fedora-repos-32-13.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d2a2c4166d673deaf4ef60c943aba4296880350b6a670318d504d33c36f55b72",
    ],
)

rpm(
    name = "file-0__5.38-4.fc32.x86_64",
    sha256 = "6eab58ed8f7333edab87b7d128f60e57a799ce39faa055fd45230bd077a0ec01",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-5.38-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-5.38-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-5.38-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-5.38-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6eab58ed8f7333edab87b7d128f60e57a799ce39faa055fd45230bd077a0ec01",
    ],
)

rpm(
    name = "file-libs-0__5.38-4.fc32.x86_64",
    sha256 = "9b3d457f2950fcb0170d1608151ea27f0dd7c970d2d3ff53e2a6644398a6571c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/file-libs-5.38-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9b3d457f2950fcb0170d1608151ea27f0dd7c970d2d3ff53e2a6644398a6571c",
    ],
)

rpm(
    name = "filesystem-0__3.14-2.fc32.aarch64",
    sha256 = "f8f3ec395d7d96c45cbd370f2376fe6266397ce091ab8fdaf884256ae8ae159f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/filesystem-3.14-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/f/filesystem-3.14-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/filesystem-3.14-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/filesystem-3.14-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f8f3ec395d7d96c45cbd370f2376fe6266397ce091ab8fdaf884256ae8ae159f",
    ],
)

rpm(
    name = "filesystem-0__3.14-2.fc32.x86_64",
    sha256 = "1110261787146443e089955912255d99daf7ba042c3743e13648a9eb3d80ceb4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/filesystem-3.14-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1110261787146443e089955912255d99daf7ba042c3743e13648a9eb3d80ceb4",
    ],
)

rpm(
    name = "findutils-1__4.7.0-4.fc32.aarch64",
    sha256 = "c9bb7dab27acdc263a7ff6b2e2a11844cfb3c172db93daa85cd221482e3a0892",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/findutils-4.7.0-4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/f/findutils-4.7.0-4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/f/findutils-4.7.0-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/f/findutils-4.7.0-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c9bb7dab27acdc263a7ff6b2e2a11844cfb3c172db93daa85cd221482e3a0892",
    ],
)

rpm(
    name = "findutils-1__4.7.0-4.fc32.x86_64",
    sha256 = "c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/f/findutils-4.7.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c7e5d5de11d4c791596ca39d1587c50caba0e06f12a7c24c5d40421d291cd661",
    ],
)

rpm(
    name = "fuse-0__2.9.9-9.fc32.x86_64",
    sha256 = "9369d4fed30402f45705b7a5cb51b6eeefb1dabbe0942c84514c6fdf1edac5e0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-2.9.9-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-2.9.9-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-2.9.9-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-2.9.9-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9369d4fed30402f45705b7a5cb51b6eeefb1dabbe0942c84514c6fdf1edac5e0",
    ],
)

rpm(
    name = "fuse-common-0__3.9.1-1.fc32.x86_64",
    sha256 = "45132e53c649def28d63c199d8c3a3b9fd16fa8bca7426ad4e9c202e52a233b4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-common-3.9.1-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-common-3.9.1-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-common-3.9.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-common-3.9.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/45132e53c649def28d63c199d8c3a3b9fd16fa8bca7426ad4e9c202e52a233b4",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-9.fc32.aarch64",
    sha256 = "5cc385c1ca3df73a1dd7865159628a6b0ce186f8679c6bc95dda0b4791e4a9fc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-9.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-9.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-9.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/f/fuse-libs-2.9.9-9.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5cc385c1ca3df73a1dd7865159628a6b0ce186f8679c6bc95dda0b4791e4a9fc",
    ],
)

rpm(
    name = "fuse-libs-0__2.9.9-9.fc32.x86_64",
    sha256 = "53992752850779218421994f61f1589eda5d368e28d340dccaae3f67de06e7f2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/f/fuse-libs-2.9.9-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/53992752850779218421994f61f1589eda5d368e28d340dccaae3f67de06e7f2",
    ],
)

rpm(
    name = "gawk-0__5.0.1-7.fc32.aarch64",
    sha256 = "62bafab5a0f37fdec29ce38bc1d635e0a81ab165061faaf5d83f5246ca4e2db0",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gawk-5.0.1-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/g/gawk-5.0.1-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gawk-5.0.1-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gawk-5.0.1-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/62bafab5a0f37fdec29ce38bc1d635e0a81ab165061faaf5d83f5246ca4e2db0",
    ],
)

rpm(
    name = "gawk-0__5.0.1-7.fc32.x86_64",
    sha256 = "d0e5d0104cf20c8dd332053a5903aab9b7fdadb84b35a1bfb3a6456f3399eb32",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gawk-5.0.1-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d0e5d0104cf20c8dd332053a5903aab9b7fdadb84b35a1bfb3a6456f3399eb32",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18.1-3.fc32.aarch64",
    sha256 = "aa667df83abb5a675444e898fb7554527b2967f3bdc793e6b4b56d794f74b9ef",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/aa667df83abb5a675444e898fb7554527b2967f3bdc793e6b4b56d794f74b9ef",
    ],
)

rpm(
    name = "gdbm-libs-1__1.18.1-3.fc32.x86_64",
    sha256 = "9899cfd32ada2537693af30b60051da21c6264b0d0db51ba709fceb179d4c836",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gdbm-libs-1.18.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9899cfd32ada2537693af30b60051da21c6264b0d0db51ba709fceb179d4c836",
    ],
)

rpm(
    name = "gdisk-0__1.0.7-1.fc32.x86_64",
    sha256 = "e06b782fb3e04800200d91b27594245383e43008ed67029745fdbe28f0d2e3a5",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gdisk-1.0.7-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gdisk-1.0.7-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gdisk-1.0.7-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gdisk-1.0.7-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e06b782fb3e04800200d91b27594245383e43008ed67029745fdbe28f0d2e3a5",
    ],
)

rpm(
    name = "genisoimage-0__1.1.11-44.fc32.x86_64",
    sha256 = "b76928dc67709dbd9ab39aff18ea7bf4355ca74e5d0f969d21e3499aa05fb8e1",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/genisoimage-1.1.11-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b76928dc67709dbd9ab39aff18ea7bf4355ca74e5d0f969d21e3499aa05fb8e1",
    ],
)

rpm(
    name = "gettext-0__0.21-1.fc32.aarch64",
    sha256 = "f9ba645870768c588621ae92fdb4c99053a4d1f12c85f71384a1c169f4feb52f",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-0.21-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-0.21-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-0.21-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-0.21-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f9ba645870768c588621ae92fdb4c99053a4d1f12c85f71384a1c169f4feb52f",
    ],
)

rpm(
    name = "gettext-0__0.21-1.fc32.x86_64",
    sha256 = "671a966b860875e2da6f95e1c8c33413ca6bc5f75ac32203a5645ba44e6ee398",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-0.21-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/671a966b860875e2da6f95e1c8c33413ca6bc5f75ac32203a5645ba44e6ee398",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-1.fc32.aarch64",
    sha256 = "0077e7f38b3af0fc32b28b5ebd92e620aedc6a9501596d64a602744fe9819648",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-libs-0.21-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-libs-0.21-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-libs-0.21-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/gettext-libs-0.21-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0077e7f38b3af0fc32b28b5ebd92e620aedc6a9501596d64a602744fe9819648",
    ],
)

rpm(
    name = "gettext-libs-0__0.21-1.fc32.x86_64",
    sha256 = "34d66d6d5891be611cca8b2c3e13f5832679094c2e7cf0b3edc5e538b6d3eefe",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gettext-libs-0.21-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/34d66d6d5891be611cca8b2c3e13f5832679094c2e7cf0b3edc5e538b6d3eefe",
    ],
)

rpm(
    name = "glib2-0__2.64.6-1.fc32.aarch64",
    sha256 = "feddf00207ca82d70cb885fe6cf45e6f7cf0d6dc66e89caeb5e06bd10404a058",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glib2-2.64.6-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glib2-2.64.6-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/glib2-2.64.6-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/glib2-2.64.6-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/feddf00207ca82d70cb885fe6cf45e6f7cf0d6dc66e89caeb5e06bd10404a058",
    ],
)

rpm(
    name = "glib2-0__2.64.6-1.fc32.x86_64",
    sha256 = "2f0f896eff6611e668944c83a63cbbe3a677802c89e4507975da1dba7ed82fed",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glib2-2.64.6-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2f0f896eff6611e668944c83a63cbbe3a677802c89e4507975da1dba7ed82fed",
    ],
)

rpm(
    name = "glibc-0__2.31-6.fc32.aarch64",
    sha256 = "b1624ca88bba72224661447ca35076f914e4c921b3a12b55bbee67798565e868",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-2.31-6.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-2.31-6.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-2.31-6.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-2.31-6.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b1624ca88bba72224661447ca35076f914e4c921b3a12b55bbee67798565e868",
    ],
)

rpm(
    name = "glibc-0__2.31-6.fc32.x86_64",
    sha256 = "642e4412d4fe796ce59aaf7d811c1a17d647fcc80565c14877be0881a3dbc4dc",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-2.31-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/642e4412d4fe796ce59aaf7d811c1a17d647fcc80565c14877be0881a3dbc4dc",
    ],
)

rpm(
    name = "glibc-common-0__2.31-6.fc32.aarch64",
    sha256 = "251b3d74106005f00314a071e26803fff7c6dd2f3a406938b665dc1e4bd66f9d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-common-2.31-6.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-common-2.31-6.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-common-2.31-6.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-common-2.31-6.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/251b3d74106005f00314a071e26803fff7c6dd2f3a406938b665dc1e4bd66f9d",
    ],
)

rpm(
    name = "glibc-common-0__2.31-6.fc32.x86_64",
    sha256 = "4e6994d189687c3728f554d94f92cce23281fb5f7a69578f64711284018a0099",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-common-2.31-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4e6994d189687c3728f554d94f92cce23281fb5f7a69578f64711284018a0099",
    ],
)

rpm(
    name = "glibc-langpack-en-0__2.31-6.fc32.aarch64",
    sha256 = "6356cbb650552271b46b328a2af627dd151fd4abe15de5fdde35d26af0bc60ec",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-langpack-en-2.31-6.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-langpack-en-2.31-6.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-langpack-en-2.31-6.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/glibc-langpack-en-2.31-6.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6356cbb650552271b46b328a2af627dd151fd4abe15de5fdde35d26af0bc60ec",
    ],
)

rpm(
    name = "glibc-langpack-en-0__2.31-6.fc32.x86_64",
    sha256 = "163e8b65f3e4f9c50011457e4cd2b64adb9b63bb178a0d3e62f8095e49d27152",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-6.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-6.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-6.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/glibc-langpack-en-2.31-6.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/163e8b65f3e4f9c50011457e4cd2b64adb9b63bb178a0d3e62f8095e49d27152",
    ],
)

rpm(
    name = "gmp-1__6.1.2-13.fc32.aarch64",
    sha256 = "5b7a135c35562e64344cc9f1ca37a5239649152cc055e14e7bf9bf84843eccab",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gmp-6.1.2-13.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/g/gmp-6.1.2-13.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gmp-6.1.2-13.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gmp-6.1.2-13.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5b7a135c35562e64344cc9f1ca37a5239649152cc055e14e7bf9bf84843eccab",
    ],
)

rpm(
    name = "gmp-1__6.1.2-13.fc32.x86_64",
    sha256 = "178e4470a6dfca84ec133932606737bfe167094560bf473940504c511354ddc9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gmp-6.1.2-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/178e4470a6dfca84ec133932606737bfe167094560bf473940504c511354ddc9",
    ],
)

rpm(
    name = "gnupg2-0__2.2.20-2.fc32.x86_64",
    sha256 = "88c92b121d4cec2a457772ef54270f07b3e755ece739a2fdbc446ffd08f21f00",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnupg2-2.2.20-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnupg2-2.2.20-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnupg2-2.2.20-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnupg2-2.2.20-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/88c92b121d4cec2a457772ef54270f07b3e755ece739a2fdbc446ffd08f21f00",
    ],
)

rpm(
    name = "gnutls-0__3.6.15-1.fc32.aarch64",
    sha256 = "be304b305cfbd74a2fcb869db5906921f181c7fd725cbe7b1bd53c62b207fc02",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-3.6.15-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-3.6.15-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-3.6.15-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-3.6.15-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/be304b305cfbd74a2fcb869db5906921f181c7fd725cbe7b1bd53c62b207fc02",
    ],
)

rpm(
    name = "gnutls-0__3.6.15-1.fc32.x86_64",
    sha256 = "802c67682c05190dd720928dbd4e5bad394e8b2eecc88af42db0007161aa9738",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-3.6.15-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/802c67682c05190dd720928dbd4e5bad394e8b2eecc88af42db0007161aa9738",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.15-1.fc32.aarch64",
    sha256 = "b04e8c03109373047e1021cba7816a5c5254687c260edcdcc5e0f8605feaf495",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-dane-3.6.15-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-dane-3.6.15-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-dane-3.6.15-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-dane-3.6.15-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b04e8c03109373047e1021cba7816a5c5254687c260edcdcc5e0f8605feaf495",
    ],
)

rpm(
    name = "gnutls-dane-0__3.6.15-1.fc32.x86_64",
    sha256 = "e18b2f511d01bfb2ec0bf42e6ba99ed3db88b133784c1011458e72df154a722e",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-dane-3.6.15-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e18b2f511d01bfb2ec0bf42e6ba99ed3db88b133784c1011458e72df154a722e",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.15-1.fc32.aarch64",
    sha256 = "d54c7b68cbae504d3fbafc56ab064e2958724ac557c73da6f999d52f0bb78604",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-utils-3.6.15-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-utils-3.6.15-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-utils-3.6.15-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/gnutls-utils-3.6.15-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d54c7b68cbae504d3fbafc56ab064e2958724ac557c73da6f999d52f0bb78604",
    ],
)

rpm(
    name = "gnutls-utils-0__3.6.15-1.fc32.x86_64",
    sha256 = "484c3af6f704a48aa67d210c7afd977a8229cd875a9d04e570761e1246ad596f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/gnutls-utils-3.6.15-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/484c3af6f704a48aa67d210c7afd977a8229cd875a9d04e570761e1246ad596f",
    ],
)

rpm(
    name = "grep-0__3.3-4.fc32.aarch64",
    sha256 = "f148b87e6bf64242dad504997f730c11706e5c0da52b036b8faebb5807d252d9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/grep-3.3-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/g/grep-3.3-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/grep-3.3-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/grep-3.3-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f148b87e6bf64242dad504997f730c11706e5c0da52b036b8faebb5807d252d9",
    ],
)

rpm(
    name = "grep-0__3.3-4.fc32.x86_64",
    sha256 = "759165656ac8141b0c0ada230c258ffcd4516c4c8d132d7fbaf762cd5a5e4095",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/grep-3.3-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/759165656ac8141b0c0ada230c258ffcd4516c4c8d132d7fbaf762cd5a5e4095",
    ],
)

rpm(
    name = "gzip-0__1.10-2.fc32.aarch64",
    sha256 = "50b7b06e94253cb4eacc1bfb68f8343b73cbd6dae427f8ad81367f7b8ebf58a8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gzip-1.10-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/g/gzip-1.10-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gzip-1.10-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gzip-1.10-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/50b7b06e94253cb4eacc1bfb68f8343b73cbd6dae427f8ad81367f7b8ebf58a8",
    ],
)

rpm(
    name = "gzip-0__1.10-2.fc32.x86_64",
    sha256 = "53f1e8570b175e8b58895646df6d8068a7e1f3cb1bafdde714ddd038bcf91e85",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gzip-1.10-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/53f1e8570b175e8b58895646df6d8068a7e1f3cb1bafdde714ddd038bcf91e85",
    ],
)

rpm(
    name = "hexedit-0__1.2.13-17.fc32.x86_64",
    sha256 = "5369a639d16f506abb37b3bea86b4551f7b4c4d8bd34f3441fd0fc0c47f1b759",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hexedit-1.2.13-17.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hexedit-1.2.13-17.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hexedit-1.2.13-17.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hexedit-1.2.13-17.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5369a639d16f506abb37b3bea86b4551f7b4c4d8bd34f3441fd0fc0c47f1b759",
    ],
)

rpm(
    name = "hivex-0__1.3.18-21.fc32.1.x86_64",
    sha256 = "7a54828dd544270274e5853b502b670a4dece679d0fbe2cebd3e3e1d0affab60",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hivex-1.3.18-21.fc32.1.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hivex-1.3.18-21.fc32.1.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hivex-1.3.18-21.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/h/hivex-1.3.18-21.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7a54828dd544270274e5853b502b670a4dece679d0fbe2cebd3e3e1d0affab60",
    ],
)

rpm(
    name = "hwdata-0__0.347-1.fc32.aarch64",
    sha256 = "0b056228b1044471af96b0c4182ac0c20d101af1137b87ef5f5dfca0f57a80ca",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0b056228b1044471af96b0c4182ac0c20d101af1137b87ef5f5dfca0f57a80ca",
    ],
)

rpm(
    name = "hwdata-0__0.347-1.fc32.x86_64",
    sha256 = "0b056228b1044471af96b0c4182ac0c20d101af1137b87ef5f5dfca0f57a80ca",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/h/hwdata-0.347-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0b056228b1044471af96b0c4182ac0c20d101af1137b87ef5f5dfca0f57a80ca",
    ],
)

rpm(
    name = "ipcalc-0__0.4.1-1.fc32.x86_64",
    sha256 = "287e95efe7718018c583b49892b89257f2dec97e0a144ffb4ea0863865a0a926",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/ipcalc-0.4.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/ipcalc-0.4.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/ipcalc-0.4.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/ipcalc-0.4.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/287e95efe7718018c583b49892b89257f2dec97e0a144ffb4ea0863865a0a926",
    ],
)

rpm(
    name = "iproute-0__5.9.0-1.fc32.aarch64",
    sha256 = "054376a061567f9c721865b581949ce59c4a83e4a2171b3e1508e32ee58375af",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-5.9.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/054376a061567f9c721865b581949ce59c4a83e4a2171b3e1508e32ee58375af",
    ],
)

rpm(
    name = "iproute-0__5.9.0-1.fc32.x86_64",
    sha256 = "5a80e7a7b6df263c6941df73e548c11de2ad2a093c06887879be1ea9e0b84b61",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-5.9.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5a80e7a7b6df263c6941df73e548c11de2ad2a093c06887879be1ea9e0b84b61",
    ],
)

rpm(
    name = "iproute-tc-0__5.9.0-1.fc32.aarch64",
    sha256 = "91be8c0765d4aade72cacea29df0dce8833616881867282cbb5bee864cf729e4",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/i/iproute-tc-5.9.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/91be8c0765d4aade72cacea29df0dce8833616881867282cbb5bee864cf729e4",
    ],
)

rpm(
    name = "iproute-tc-0__5.9.0-1.fc32.x86_64",
    sha256 = "78734a6d33175c9ee502cba0042794cef8fb292fe7fd9129b4972932100bf41b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iproute-tc-5.9.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/78734a6d33175c9ee502cba0042794cef8fb292fe7fd9129b4972932100bf41b",
    ],
)

rpm(
    name = "iptables-0__1.8.4-9.fc32.aarch64",
    sha256 = "4560c18ef4e856b4aefd33f519864341849a2ebac560aa399b0300f35dae5236",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-1.8.4-9.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-1.8.4-9.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-1.8.4-9.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-1.8.4-9.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4560c18ef4e856b4aefd33f519864341849a2ebac560aa399b0300f35dae5236",
    ],
)

rpm(
    name = "iptables-0__1.8.4-9.fc32.x86_64",
    sha256 = "6505f2881185c2a09eb623bbc1e10c47573386b385879484251ea48d196f2094",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-1.8.4-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6505f2881185c2a09eb623bbc1e10c47573386b385879484251ea48d196f2094",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-9.fc32.aarch64",
    sha256 = "0850c3829d13d16438d6b685aaf20079e51e9db4941fac4fdea901200762686d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-libs-1.8.4-9.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-libs-1.8.4-9.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-libs-1.8.4-9.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/i/iptables-libs-1.8.4-9.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0850c3829d13d16438d6b685aaf20079e51e9db4941fac4fdea901200762686d",
    ],
)

rpm(
    name = "iptables-libs-0__1.8.4-9.fc32.x86_64",
    sha256 = "dcf038adbb690e6aa3dcc020576eccf1ee3eeecb0cddd3011fa5f99e85c8bf3a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-libs-1.8.4-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dcf038adbb690e6aa3dcc020576eccf1ee3eeecb0cddd3011fa5f99e85c8bf3a",
    ],
)

rpm(
    name = "iptables-nft-0__1.8.4-9.fc32.x86_64",
    sha256 = "b81b7935915e07fe9fec3cfb0a25c27baa051ad9ee5a3f3a0ffd22b1615eccb9",
    urls = [
        "https://mirror.atl.genesisadaptive.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-nft-1.8.4-9.fc32.x86_64.rpm",
        "https://sjc.edge.kernel.org/fedora-buffet/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-nft-1.8.4-9.fc32.x86_64.rpm",
        "https://mirror.us-midwest-1.nexcess.net/fedora/updates/32/Everything/x86_64/Packages/i/iptables-nft-1.8.4-9.fc32.x86_64.rpm",
        "https://mirrors.rit.edu/fedora/fedora/linux/updates/32/Everything/x86_64/Packages/i/iptables-nft-1.8.4-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b81b7935915e07fe9fec3cfb0a25c27baa051ad9ee5a3f3a0ffd22b1615eccb9",
    ],
)

rpm(
    name = "iputils-0__20200821-1.fc32.aarch64",
    sha256 = "315899519ad1c8cc335b8e63579b012d98aaf0400a602322935c82f07b4f063a",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iputils-20200821-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/i/iputils-20200821-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/i/iputils-20200821-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/i/iputils-20200821-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/315899519ad1c8cc335b8e63579b012d98aaf0400a602322935c82f07b4f063a",
    ],
)

rpm(
    name = "iputils-0__20200821-1.fc32.x86_64",
    sha256 = "a5c17f8a29defceb5d33ff860c205ba1db36d36828aecd9b96609207697a8047",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/i/iputils-20200821-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a5c17f8a29defceb5d33ff860c205ba1db36d36828aecd9b96609207697a8047",
    ],
)

rpm(
    name = "ipxe-roms-qemu-0__20190125-4.git36a4c85f.fc32.x86_64",
    sha256 = "6e298c78b9ccc125b0fdfdda4c39a993aaacd1accc62a23df9f30e46c17b64a9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/i/ipxe-roms-qemu-20190125-4.git36a4c85f.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/6e298c78b9ccc125b0fdfdda4c39a993aaacd1accc62a23df9f30e46c17b64a9",
    ],
)

rpm(
    name = "jansson-0__2.12-5.fc32.aarch64",
    sha256 = "da4e2994692c9ed4d0760528139f6437bcb0d54862fac1a4afa55e329393d254",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/j/jansson-2.12-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/j/jansson-2.12-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/j/jansson-2.12-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/j/jansson-2.12-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/da4e2994692c9ed4d0760528139f6437bcb0d54862fac1a4afa55e329393d254",
    ],
)

rpm(
    name = "jansson-0__2.12-5.fc32.x86_64",
    sha256 = "975719a0c73cf5cb5bcbc8ad11b816ed75923dccd9c091baa4a6c6000753dcd8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jansson-2.12-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/975719a0c73cf5cb5bcbc8ad11b816ed75923dccd9c091baa4a6c6000753dcd8",
    ],
)

rpm(
    name = "jq-0__1.6-4.fc32.x86_64",
    sha256 = "65d0f0dac8fdc74058e40be7c7f14287932576a82ebc0d2249d311cdc9045b07",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jq-1.6-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jq-1.6-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jq-1.6-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/j/jq-1.6-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65d0f0dac8fdc74058e40be7c7f14287932576a82ebc0d2249d311cdc9045b07",
    ],
)

rpm(
    name = "json-c-0__0.13.1-13.fc32.aarch64",
    sha256 = "f3827b333133bda6bbfdc82609e1cfce8233c3c34b108104b0033188ca942093",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/j/json-c-0.13.1-13.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/j/json-c-0.13.1-13.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/j/json-c-0.13.1-13.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/j/json-c-0.13.1-13.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f3827b333133bda6bbfdc82609e1cfce8233c3c34b108104b0033188ca942093",
    ],
)

rpm(
    name = "json-c-0__0.13.1-13.fc32.x86_64",
    sha256 = "56ecdfc358f2149bc9f6fd38161d33fe45177c11059fd813143c8d314b1019fc",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/j/json-c-0.13.1-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56ecdfc358f2149bc9f6fd38161d33fe45177c11059fd813143c8d314b1019fc",
    ],
)

rpm(
    name = "kde-filesystem-0__4-63.fc32.aarch64",
    sha256 = "e545e03430221aed4a4eac22857a7ed1fc3c9b8b9e7df53ed46aea7af96321aa",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kde-filesystem-4-63.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/k/kde-filesystem-4-63.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kde-filesystem-4-63.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kde-filesystem-4-63.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e545e03430221aed4a4eac22857a7ed1fc3c9b8b9e7df53ed46aea7af96321aa",
    ],
)

rpm(
    name = "kde-filesystem-0__4-63.fc32.x86_64",
    sha256 = "f8aecd3ff4786a15d434ef8366f2e35e5d70e256c7fe2d521e7923064c232402",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kde-filesystem-4-63.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kde-filesystem-4-63.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kde-filesystem-4-63.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kde-filesystem-4-63.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f8aecd3ff4786a15d434ef8366f2e35e5d70e256c7fe2d521e7923064c232402",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.1-1.fc32.aarch64",
    sha256 = "819cdb2efbfe33fc8d2592d93f77e5b4d8516efc349409c0785294f32920ec81",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/k/keyutils-libs-1.6.1-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/819cdb2efbfe33fc8d2592d93f77e5b4d8516efc349409c0785294f32920ec81",
    ],
)

rpm(
    name = "keyutils-libs-0__1.6.1-1.fc32.x86_64",
    sha256 = "4b40eb8bce5cce20be6bb7693f27b61bb0ad9d1e4f9d38b89a38841eed7fb894",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/keyutils-libs-1.6.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4b40eb8bce5cce20be6bb7693f27b61bb0ad9d1e4f9d38b89a38841eed7fb894",
    ],
)

rpm(
    name = "kf5-filesystem-0__5.75.0-1.fc32.aarch64",
    sha256 = "d5f8ad9874db4e71dacec64086f3d24dcec9eda5314b0ca999ec5cac5b2678ba",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/k/kf5-filesystem-5.75.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/k/kf5-filesystem-5.75.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/k/kf5-filesystem-5.75.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/k/kf5-filesystem-5.75.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d5f8ad9874db4e71dacec64086f3d24dcec9eda5314b0ca999ec5cac5b2678ba",
    ],
)

rpm(
    name = "kf5-filesystem-0__5.75.0-1.fc32.x86_64",
    sha256 = "0876037d1952c7855c38ddbacfcdbce0c8a54c39f26beab2f6e6ea88b2c02deb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/kf5-filesystem-5.75.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/kf5-filesystem-5.75.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/k/kf5-filesystem-5.75.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/kf5-filesystem-5.75.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0876037d1952c7855c38ddbacfcdbce0c8a54c39f26beab2f6e6ea88b2c02deb",
    ],
)

rpm(
    name = "kmod-0__27-1.fc32.aarch64",
    sha256 = "fe512ddf337568ca1e4d1c0cce66dda461ca570587c7beb1e1be3960540e394f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kmod-27-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/k/kmod-27-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kmod-27-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kmod-27-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fe512ddf337568ca1e4d1c0cce66dda461ca570587c7beb1e1be3960540e394f",
    ],
)

rpm(
    name = "kmod-0__27-1.fc32.x86_64",
    sha256 = "3f9c95f3827b785f49ac4a270d4c3a703dceba673c452838744ec5064cf43cbd",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-27-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3f9c95f3827b785f49ac4a270d4c3a703dceba673c452838744ec5064cf43cbd",
    ],
)

rpm(
    name = "kmod-libs-0__27-1.fc32.aarch64",
    sha256 = "7684be07a8e054660705f8d6b1522d9a829be6614293096dc7b871682e445709",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kmod-libs-27-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/k/kmod-libs-27-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kmod-libs-27-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/k/kmod-libs-27-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7684be07a8e054660705f8d6b1522d9a829be6614293096dc7b871682e445709",
    ],
)

rpm(
    name = "kmod-libs-0__27-1.fc32.x86_64",
    sha256 = "56187c1c980cc0680f4dbc433ed2c8507e7dc9ab00000615b63ea08c086b7ab2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/k/kmod-libs-27-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56187c1c980cc0680f4dbc433ed2c8507e7dc9ab00000615b63ea08c086b7ab2",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-29.fc32.aarch64",
    sha256 = "9349fde1714397d1c548bb2975bd9b3ca360658a921c23c241acf02607d3f958",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/k/krb5-libs-1.18.2-29.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/k/krb5-libs-1.18.2-29.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/k/krb5-libs-1.18.2-29.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/k/krb5-libs-1.18.2-29.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9349fde1714397d1c548bb2975bd9b3ca360658a921c23c241acf02607d3f958",
    ],
)

rpm(
    name = "krb5-libs-0__1.18.2-29.fc32.x86_64",
    sha256 = "f1ad00906636a2e01b4e978233a9e4a622f4c42f9bc4ec0dd0e294ba75351394",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/k/krb5-libs-1.18.2-29.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1ad00906636a2e01b4e978233a9e4a622f4c42f9bc4ec0dd0e294ba75351394",
    ],
)

rpm(
    name = "less-0__551-3.fc32.x86_64",
    sha256 = "2b783576612dcf10ab151fee03084f8ae1667c044a9e2e9404a2a139e7c6c884",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/less-551-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/less-551-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/less-551-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/less-551-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2b783576612dcf10ab151fee03084f8ae1667c044a9e2e9404a2a139e7c6c884",
    ],
)

rpm(
    name = "libacl-0__2.2.53-5.fc32.aarch64",
    sha256 = "98d58695f22a613ff6ffcb2b738b4127be7b72e5d56f7d0dbd3c999f189ba323",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libacl-2.2.53-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libacl-2.2.53-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libacl-2.2.53-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libacl-2.2.53-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/98d58695f22a613ff6ffcb2b738b4127be7b72e5d56f7d0dbd3c999f189ba323",
    ],
)

rpm(
    name = "libacl-0__2.2.53-5.fc32.x86_64",
    sha256 = "f826f984b23d0701a1b72de5882b9c0e7bae87ef49d9edfea156654f489f8b2b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libacl-2.2.53-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f826f984b23d0701a1b72de5882b9c0e7bae87ef49d9edfea156654f489f8b2b",
    ],
)

rpm(
    name = "libaio-0__0.3.111-7.fc32.aarch64",
    sha256 = "e7b49bf8e3183d7604c7f7f51dfbc1e03bc599ddd7eac459a86f4ffdc8432533",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libaio-0.3.111-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libaio-0.3.111-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libaio-0.3.111-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libaio-0.3.111-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e7b49bf8e3183d7604c7f7f51dfbc1e03bc599ddd7eac459a86f4ffdc8432533",
    ],
)

rpm(
    name = "libaio-0__0.3.111-7.fc32.x86_64",
    sha256 = "a410db5c56d4f39f6ea71e7d5bb6d4a2bd518015d1e34f38fbc0d7bbd4e872d4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libaio-0.3.111-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a410db5c56d4f39f6ea71e7d5bb6d4a2bd518015d1e34f38fbc0d7bbd4e872d4",
    ],
)

rpm(
    name = "libarchive-0__3.4.3-1.fc32.aarch64",
    sha256 = "e99dfc3552f563f42caf08b9aa2a75e7bfeb44d14fdfffc955462d6824c1b44b",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libarchive-3.4.3-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libarchive-3.4.3-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libarchive-3.4.3-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libarchive-3.4.3-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e99dfc3552f563f42caf08b9aa2a75e7bfeb44d14fdfffc955462d6824c1b44b",
    ],
)

rpm(
    name = "libarchive-0__3.4.3-1.fc32.x86_64",
    sha256 = "b4053e72ef23e59a43b4f77b0f3abeb92c4b72c9df7d4f62572560edfbfa02a4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libarchive-3.4.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4053e72ef23e59a43b4f77b0f3abeb92c4b72c9df7d4f62572560edfbfa02a4",
    ],
)

rpm(
    name = "libargon2-0__20171227-4.fc32.aarch64",
    sha256 = "6ef55c2aa000adea432676010756cf69e8851587ad17277b21bde362e369bf3e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libargon2-20171227-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libargon2-20171227-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libargon2-20171227-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libargon2-20171227-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6ef55c2aa000adea432676010756cf69e8851587ad17277b21bde362e369bf3e",
    ],
)

rpm(
    name = "libargon2-0__20171227-4.fc32.x86_64",
    sha256 = "7d9bd2fe016ca8860e8fab4a430b3aae4c7b7bea55f8ccd7775ad470172e2886",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libargon2-20171227-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7d9bd2fe016ca8860e8fab4a430b3aae4c7b7bea55f8ccd7775ad470172e2886",
    ],
)

rpm(
    name = "libassuan-0__2.5.3-3.fc32.x86_64",
    sha256 = "598a136b7027cb9b4fef6bfa34715979d41c2f62c9d8bec5d50b633a17790f7b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libassuan-2.5.3-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libassuan-2.5.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libassuan-2.5.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libassuan-2.5.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/598a136b7027cb9b4fef6bfa34715979d41c2f62c9d8bec5d50b633a17790f7b",
    ],
)

rpm(
    name = "libattr-0__2.4.48-8.fc32.aarch64",
    sha256 = "caa6fe00c6e322e961c4b7a02ba4a10cc939b84121e09d07d331adcdc2ae1af2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libattr-2.4.48-8.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libattr-2.4.48-8.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libattr-2.4.48-8.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libattr-2.4.48-8.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/caa6fe00c6e322e961c4b7a02ba4a10cc939b84121e09d07d331adcdc2ae1af2",
    ],
)

rpm(
    name = "libattr-0__2.4.48-8.fc32.x86_64",
    sha256 = "65e0cfe367ae4d54cf8bf509cb05e063c9eb6f2fea8dadcf746cdd85adc31d88",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libattr-2.4.48-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/65e0cfe367ae4d54cf8bf509cb05e063c9eb6f2fea8dadcf746cdd85adc31d88",
    ],
)

rpm(
    name = "libblkid-0__2.35.2-1.fc32.aarch64",
    sha256 = "bb803701a499375f204ef0ff3af8c7056c46ffb05d658537721d3c75dd7f33cc",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libblkid-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libblkid-2.35.2-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libblkid-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libblkid-2.35.2-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bb803701a499375f204ef0ff3af8c7056c46ffb05d658537721d3c75dd7f33cc",
    ],
)

rpm(
    name = "libblkid-0__2.35.2-1.fc32.x86_64",
    sha256 = "d43d17930e5fedbbeb2a45bdbfff713485c6cd01ca6cbb9443370192e73daf40",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libblkid-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d43d17930e5fedbbeb2a45bdbfff713485c6cd01ca6cbb9443370192e73daf40",
    ],
)

rpm(
    name = "libburn-0__1.5.4-2.fc32.aarch64",
    sha256 = "50a792dd4d00cada9ec37e947fe425058d07844eec79087b8ee79839479c5750",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libburn-1.5.4-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/50a792dd4d00cada9ec37e947fe425058d07844eec79087b8ee79839479c5750",
    ],
)

rpm(
    name = "libburn-0__1.5.4-2.fc32.x86_64",
    sha256 = "9b92acf62e9cd4ee78748fe28393f9feb37b0d6e01ad1a270c966f841a8bb503",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libburn-1.5.4-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9b92acf62e9cd4ee78748fe28393f9feb37b0d6e01ad1a270c966f841a8bb503",
    ],
)

rpm(
    name = "libcap-0__2.26-7.fc32.aarch64",
    sha256 = "0a2eadd29cc53df942d3f0acc016b281efa4347fc2e9de1d7b8b61d9c5f0d894",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libcap-2.26-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libcap-2.26-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libcap-2.26-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libcap-2.26-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0a2eadd29cc53df942d3f0acc016b281efa4347fc2e9de1d7b8b61d9c5f0d894",
    ],
)

rpm(
    name = "libcap-0__2.26-7.fc32.x86_64",
    sha256 = "1bc0542cf8a3746d0fe25c397a93c8206963f1f287246c6fb864eedfc9ffa4a7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcap-2.26-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1bc0542cf8a3746d0fe25c397a93c8206963f1f287246c6fb864eedfc9ffa4a7",
    ],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.fc32.aarch64",
    sha256 = "fd7a4b3682c04d0f97a1e71f4cf2d6f705835db462fcd0986fa02b4ef89d4d69",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcap-ng-0.7.11-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcap-ng-0.7.11-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcap-ng-0.7.11-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcap-ng-0.7.11-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fd7a4b3682c04d0f97a1e71f4cf2d6f705835db462fcd0986fa02b4ef89d4d69",
    ],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.fc32.x86_64",
    sha256 = "6fc5b00896f95b99a6c9785eedae9e6e522a9340fa0da0b0b1f4665708f0245f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcap-ng-0.7.11-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6fc5b00896f95b99a6c9785eedae9e6e522a9340fa0da0b0b1f4665708f0245f",
    ],
)

rpm(
    name = "libcom_err-0__1.45.5-3.fc32.aarch64",
    sha256 = "93c5fe6589243bff8f4d6934d82616a4cce0f30d071c513cc56f8e53bfc19d17",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libcom_err-1.45.5-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libcom_err-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libcom_err-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libcom_err-1.45.5-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/93c5fe6589243bff8f4d6934d82616a4cce0f30d071c513cc56f8e53bfc19d17",
    ],
)

rpm(
    name = "libcom_err-0__1.45.5-3.fc32.x86_64",
    sha256 = "4494013eac1ad337673f084242aa8ebffb4a149243475b448bee9266401f2896",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libcom_err-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4494013eac1ad337673f084242aa8ebffb4a149243475b448bee9266401f2896",
    ],
)

rpm(
    name = "libconfig-0__1.7.2-5.fc32.x86_64",
    sha256 = "d94ff106e0f1af1e6f0cbe459594041da7713cec89db1cbd3573a84ee1fe233a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libconfig-1.7.2-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libconfig-1.7.2-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libconfig-1.7.2-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libconfig-1.7.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d94ff106e0f1af1e6f0cbe459594041da7713cec89db1cbd3573a84ee1fe233a",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.69.1-8.fc32.aarch64",
    sha256 = "3952a03162c1f1954b169055b12022aa49f1a5c3a11e7363743544bad6f155b2",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcurl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcurl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcurl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libcurl-minimal-7.69.1-8.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3952a03162c1f1954b169055b12022aa49f1a5c3a11e7363743544bad6f155b2",
    ],
)

rpm(
    name = "libcurl-minimal-0__7.69.1-8.fc32.x86_64",
    sha256 = "2d92df47bd26d6619bbcc7e1a0b88abb74a8bc841f064a7a0817b091510311d5",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libcurl-minimal-7.69.1-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2d92df47bd26d6619bbcc7e1a0b88abb74a8bc841f064a7a0817b091510311d5",
    ],
)

rpm(
    name = "libdb-0__5.3.28-40.fc32.aarch64",
    sha256 = "7bfb33bfa3c3a952c54cb61b7f7c7047c1fd91e8e334f53f54faea6f34e6c0bb",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libdb-5.3.28-40.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libdb-5.3.28-40.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libdb-5.3.28-40.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libdb-5.3.28-40.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7bfb33bfa3c3a952c54cb61b7f7c7047c1fd91e8e334f53f54faea6f34e6c0bb",
    ],
)

rpm(
    name = "libdb-0__5.3.28-40.fc32.x86_64",
    sha256 = "688fcc0b7ef3c48cf7d602eefd7fefae7bcad4f0dc71c9fe9432c2ce5bbd9daa",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-5.3.28-40.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/688fcc0b7ef3c48cf7d602eefd7fefae7bcad4f0dc71c9fe9432c2ce5bbd9daa",
    ],
)

rpm(
    name = "libdb-utils-0__5.3.28-40.fc32.aarch64",
    sha256 = "435530a0b9a086018694034ce48e9589348fc66389d884977b400f2f74814ac8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libdb-utils-5.3.28-40.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libdb-utils-5.3.28-40.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libdb-utils-5.3.28-40.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libdb-utils-5.3.28-40.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/435530a0b9a086018694034ce48e9589348fc66389d884977b400f2f74814ac8",
    ],
)

rpm(
    name = "libdb-utils-0__5.3.28-40.fc32.x86_64",
    sha256 = "431d836b2be015212d8c15b4290d5ce5bb45282cbf3fc52696f632d84ce34dfe",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libdb-utils-5.3.28-40.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/431d836b2be015212d8c15b4290d5ce5bb45282cbf3fc52696f632d84ce34dfe",
    ],
)

rpm(
    name = "libepoxy-0__1.5.4-2.fc32.x86_64",
    sha256 = "76c22bfdaab3fe575abe813cbb9a1c3816d8cf50d3c25c9280d07031403ccf30",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libepoxy-1.5.4-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/76c22bfdaab3fe575abe813cbb9a1c3816d8cf50d3c25c9280d07031403ccf30",
    ],
)

rpm(
    name = "libevent-0__2.1.8-8.fc32.aarch64",
    sha256 = "ad874e09de00dbdb887eb6a94351869950ead7f6409dfa191d1443d3bb9dd255",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libevent-2.1.8-8.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libevent-2.1.8-8.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libevent-2.1.8-8.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libevent-2.1.8-8.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad874e09de00dbdb887eb6a94351869950ead7f6409dfa191d1443d3bb9dd255",
    ],
)

rpm(
    name = "libevent-0__2.1.8-8.fc32.x86_64",
    sha256 = "7bf42ff57ce2a31db0da7d6c5926552f4e51e9f25cded77bd634eb5cd35eadab",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libevent-2.1.8-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7bf42ff57ce2a31db0da7d6c5926552f4e51e9f25cded77bd634eb5cd35eadab",
    ],
)

rpm(
    name = "libfdisk-0__2.35.2-1.fc32.aarch64",
    sha256 = "fa922d6606ca15a60059506366bff2e5f17be6c41189e24bc748f596cbc4b4d0",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libfdisk-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libfdisk-2.35.2-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libfdisk-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libfdisk-2.35.2-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fa922d6606ca15a60059506366bff2e5f17be6c41189e24bc748f596cbc4b4d0",
    ],
)

rpm(
    name = "libfdisk-0__2.35.2-1.fc32.x86_64",
    sha256 = "d7a895002e2291f776c8bf40dc99848105ca8c8e1651ba4692cc44ab838bc0a1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libfdisk-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d7a895002e2291f776c8bf40dc99848105ca8c8e1651ba4692cc44ab838bc0a1",
    ],
)

rpm(
    name = "libfdt-0__1.6.0-1.fc32.aarch64",
    sha256 = "a78e345faac8293fa2c05560869eb610ce53b5c851db932fd8915128b27d0c1e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libfdt-1.6.0-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libfdt-1.6.0-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libfdt-1.6.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libfdt-1.6.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a78e345faac8293fa2c05560869eb610ce53b5c851db932fd8915128b27d0c1e",
    ],
)

rpm(
    name = "libffi-0__3.1-24.fc32.aarch64",
    sha256 = "291df16c0ae66fa5685cd033c84ae92765be4f4e17ce4936e47dc602ac6ff93e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libffi-3.1-24.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libffi-3.1-24.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libffi-3.1-24.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libffi-3.1-24.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/291df16c0ae66fa5685cd033c84ae92765be4f4e17ce4936e47dc602ac6ff93e",
    ],
)

rpm(
    name = "libffi-0__3.1-24.fc32.x86_64",
    sha256 = "86c87a4169bdf75c6d3a2f11d3a7e20b6364b2db97c74bc7eb62b1b22bc54401",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libffi-3.1-24.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86c87a4169bdf75c6d3a2f11d3a7e20b6364b2db97c74bc7eb62b1b22bc54401",
    ],
)

rpm(
    name = "libgcc-0__10.3.1-1.fc32.aarch64",
    sha256 = "1db8dfa80dadc1c754d5384b6f4c2b09fb5327d57536816161f95757e53b64e5",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgcc-10.3.1-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1db8dfa80dadc1c754d5384b6f4c2b09fb5327d57536816161f95757e53b64e5",
    ],
)

rpm(
    name = "libgcc-0__10.3.1-1.fc32.x86_64",
    sha256 = "5138e9691a2d1d9573535804528834834c05ab87a0daf920ccd01dac462a7358",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgcc-10.3.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5138e9691a2d1d9573535804528834834c05ab87a0daf920ccd01dac462a7358",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-3.fc32.aarch64",
    sha256 = "e96e4caf6c98faa5fb61bd3b13ee7afa0d7510d3176fe3d3cbf485847ce985fd",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libgcrypt-1.8.5-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libgcrypt-1.8.5-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libgcrypt-1.8.5-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libgcrypt-1.8.5-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e96e4caf6c98faa5fb61bd3b13ee7afa0d7510d3176fe3d3cbf485847ce985fd",
    ],
)

rpm(
    name = "libgcrypt-0__1.8.5-3.fc32.x86_64",
    sha256 = "5f0ae954b5955c86623e68cd81ccf8505a89f260003b8a3be6a93bd76f18452c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgcrypt-1.8.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5f0ae954b5955c86623e68cd81ccf8505a89f260003b8a3be6a93bd76f18452c",
    ],
)

rpm(
    name = "libgomp-0__10.3.1-1.fc32.aarch64",
    sha256 = "8394b5dbae892e4b95085bdd17ab95387e3434a99dfb7a113ef4ad51cf0553e4",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libgomp-10.3.1-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8394b5dbae892e4b95085bdd17ab95387e3434a99dfb7a113ef4ad51cf0553e4",
    ],
)

rpm(
    name = "libgomp-0__10.3.1-1.fc32.x86_64",
    sha256 = "0d307615b31fa96e6a6d27e292dc16fd9bcd2ce36d50157cb100041abd100be4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libgomp-10.3.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0d307615b31fa96e6a6d27e292dc16fd9bcd2ce36d50157cb100041abd100be4",
    ],
)

rpm(
    name = "libgpg-error-0__1.36-3.fc32.aarch64",
    sha256 = "cffbab9f6052ee2c7b8bcc369a411e319174de094fb94eaf71555ce485049a74",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libgpg-error-1.36-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libgpg-error-1.36-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libgpg-error-1.36-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libgpg-error-1.36-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/cffbab9f6052ee2c7b8bcc369a411e319174de094fb94eaf71555ce485049a74",
    ],
)

rpm(
    name = "libgpg-error-0__1.36-3.fc32.x86_64",
    sha256 = "9bd5cb588664e8427bc8bebde0cdf5e14315916624ab6b1979dde60f6eae4278",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libgpg-error-1.36-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9bd5cb588664e8427bc8bebde0cdf5e14315916624ab6b1979dde60f6eae4278",
    ],
)

rpm(
    name = "libguestfs-1__1.44.0-1.fc32.x86_64",
    sha256 = "a0481a1fdd6cefb2a1038c0803f856345cfdd527b3803e5d54382c901c4318f2",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-1.44.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-1.44.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-1.44.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-1.44.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a0481a1fdd6cefb2a1038c0803f856345cfdd527b3803e5d54382c901c4318f2",
    ],
)

rpm(
    name = "libguestfs-tools-1__1.44.0-1.fc32.x86_64",
    sha256 = "29c3f7a36cbffe3e87032ffa60534f7a27e28af6aaa35da7e587721569788917",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-1.44.0-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-1.44.0-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-1.44.0-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-1.44.0-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/29c3f7a36cbffe3e87032ffa60534f7a27e28af6aaa35da7e587721569788917",
    ],
)

rpm(
    name = "libguestfs-tools-c-1__1.44.0-1.fc32.x86_64",
    sha256 = "db0da0fcd98ce2658df832b9fa6719a2f5e1d023b3a0e0703ac76a21cbc3f614",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libguestfs-tools-c-1.44.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/db0da0fcd98ce2658df832b9fa6719a2f5e1d023b3a0e0703ac76a21cbc3f614",
    ],
)

rpm(
    name = "libibverbs-0__33.0-2.fc32.aarch64",
    sha256 = "b98d83016c9746eb5e7b94e7d4fd40187de0dbd4355d01dcdd36c0b7e4c5b324",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libibverbs-33.0-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libibverbs-33.0-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libibverbs-33.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libibverbs-33.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b98d83016c9746eb5e7b94e7d4fd40187de0dbd4355d01dcdd36c0b7e4c5b324",
    ],
)

rpm(
    name = "libibverbs-0__33.0-2.fc32.x86_64",
    sha256 = "f3f0cb33d3a5aabc448e7f3520eced2946c2f391eb4315ddf7c5423148cf0969",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-33.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-33.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-33.0-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libibverbs-33.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f3f0cb33d3a5aabc448e7f3520eced2946c2f391eb4315ddf7c5423148cf0969",
    ],
)

rpm(
    name = "libidn2-0__2.3.0-2.fc32.aarch64",
    sha256 = "500c4abc34ff58e6f06c7194034b2d68b618c5e6afa89b551ab74ef226e1880a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libidn2-2.3.0-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libidn2-2.3.0-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libidn2-2.3.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libidn2-2.3.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/500c4abc34ff58e6f06c7194034b2d68b618c5e6afa89b551ab74ef226e1880a",
    ],
)

rpm(
    name = "libidn2-0__2.3.0-2.fc32.x86_64",
    sha256 = "20787251df57a108bbf9c40e30f041b71ac36c8a10900fb699e574ee7e259bf2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libidn2-2.3.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20787251df57a108bbf9c40e30f041b71ac36c8a10900fb699e574ee7e259bf2",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-2.fc32.aarch64",
    sha256 = "f2dca49457a25ac07e660ccb208851c5cb40e624c2a2f6a6251a4264204024af",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisoburn-1.5.4-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f2dca49457a25ac07e660ccb208851c5cb40e624c2a2f6a6251a4264204024af",
    ],
)

rpm(
    name = "libisoburn-0__1.5.4-2.fc32.x86_64",
    sha256 = "987f36bf84c5435fd99d961eab69cec5626f039ab414d361854d14bc2f77914a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisoburn-1.5.4-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/987f36bf84c5435fd99d961eab69cec5626f039ab414d361854d14bc2f77914a",
    ],
)

rpm(
    name = "libisofs-0__1.5.4-1.fc32.aarch64",
    sha256 = "f6460d264552bf1a4b1630dd7bfca3eec2ef7bdd5d31753aef7bcf9c2c1e43f3",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libisofs-1.5.4-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f6460d264552bf1a4b1630dd7bfca3eec2ef7bdd5d31753aef7bcf9c2c1e43f3",
    ],
)

rpm(
    name = "libisofs-0__1.5.4-1.fc32.x86_64",
    sha256 = "e024cfa9312d6752cd7677fb396ffec3895e4452e133d131f4715f6d7317ee9d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libisofs-1.5.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e024cfa9312d6752cd7677fb396ffec3895e4452e133d131f4715f6d7317ee9d",
    ],
)

rpm(
    name = "libjpeg-turbo-0__2.0.4-3.fc32.x86_64",
    sha256 = "37c8e4091216342f58b83acf030f94cb776e95d2f0f86b548e33e2c4cba341eb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libjpeg-turbo-2.0.4-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/37c8e4091216342f58b83acf030f94cb776e95d2f0f86b548e33e2c4cba341eb",
    ],
)

rpm(
    name = "libksba-0__1.3.5-11.fc32.x86_64",
    sha256 = "1b05dd5abad5a31380c859bc33e7851158c24333fda837ca9facf869005f81fe",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libksba-1.3.5-11.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libksba-1.3.5-11.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libksba-1.3.5-11.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libksba-1.3.5-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1b05dd5abad5a31380c859bc33e7851158c24333fda837ca9facf869005f81fe",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-11.fc32.aarch64",
    sha256 = "2356581880df7b8275896b18de24e432a362ee159fc3127f92476ffe8d0432fd",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libmnl-1.0.4-11.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libmnl-1.0.4-11.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libmnl-1.0.4-11.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libmnl-1.0.4-11.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2356581880df7b8275896b18de24e432a362ee159fc3127f92476ffe8d0432fd",
    ],
)

rpm(
    name = "libmnl-0__1.0.4-11.fc32.x86_64",
    sha256 = "1c68255945533ed4e3368125bc46e19f3fe348d7ec507a85a35038dbb976003f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libmnl-1.0.4-11.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1c68255945533ed4e3368125bc46e19f3fe348d7ec507a85a35038dbb976003f",
    ],
)

rpm(
    name = "libmount-0__2.35.2-1.fc32.aarch64",
    sha256 = "06d375e2045df7a9b491f314e4724bed4bbb415da967e0309f552ad2c95a5521",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libmount-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libmount-2.35.2-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libmount-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libmount-2.35.2-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/06d375e2045df7a9b491f314e4724bed4bbb415da967e0309f552ad2c95a5521",
    ],
)

rpm(
    name = "libmount-0__2.35.2-1.fc32.x86_64",
    sha256 = "2c8e76fcc1ad8197ffdb66d06fb498a1129e71e0f7c04a05176867e5788bbf05",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libmount-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c8e76fcc1ad8197ffdb66d06fb498a1129e71e0f7c04a05176867e5788bbf05",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.7-4.fc32.aarch64",
    sha256 = "400c91d4d6d1125ec891c16ea72aa4123fc4c96e02f8668a8ae6dbc27113d408",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/400c91d4d6d1125ec891c16ea72aa4123fc4c96e02f8668a8ae6dbc27113d408",
    ],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.7-4.fc32.x86_64",
    sha256 = "884357540f4be2a74e608e2c7a31f2371ee3b4d29be2fe39a371c0b131d84aa6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnetfilter_conntrack-1.0.7-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/884357540f4be2a74e608e2c7a31f2371ee3b4d29be2fe39a371c0b131d84aa6",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-17.fc32.aarch64",
    sha256 = "a0260a37707734c6f97885687a6ad5967c23cb0c693668bf1402e6ee5d4abe1e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a0260a37707734c6f97885687a6ad5967c23cb0c693668bf1402e6ee5d4abe1e",
    ],
)

rpm(
    name = "libnfnetlink-0__1.0.1-17.fc32.x86_64",
    sha256 = "ec6abd65541b5bded814de19c9d064e6c21e3d8b424dba7cb25b2fdc52d45a2b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnfnetlink-1.0.1-17.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ec6abd65541b5bded814de19c9d064e6c21e3d8b424dba7cb25b2fdc52d45a2b",
    ],
)

rpm(
    name = "libnftnl-0__1.1.5-2.fc32.aarch64",
    sha256 = "07cf4ae85cb34a38b22eff66e1fd996b32a5beda0c60644b06ecdff33c224ce9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnftnl-1.1.5-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libnftnl-1.1.5-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnftnl-1.1.5-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnftnl-1.1.5-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/07cf4ae85cb34a38b22eff66e1fd996b32a5beda0c60644b06ecdff33c224ce9",
    ],
)

rpm(
    name = "libnftnl-0__1.1.5-2.fc32.x86_64",
    sha256 = "3afab9512fd4d56a13c95b530c805ac8b2bc872572ec5bb435eccdd59fbbc8b6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnftnl-1.1.5-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3afab9512fd4d56a13c95b530c805ac8b2bc872572ec5bb435eccdd59fbbc8b6",
    ],
)

rpm(
    name = "libnghttp2-0__1.41.0-1.fc32.aarch64",
    sha256 = "286221dc6c1f1bee95ae1380bfd72c6c4c7ded5a5e40f3cff5cb43b002175280",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libnghttp2-1.41.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libnghttp2-1.41.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libnghttp2-1.41.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libnghttp2-1.41.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/286221dc6c1f1bee95ae1380bfd72c6c4c7ded5a5e40f3cff5cb43b002175280",
    ],
)

rpm(
    name = "libnghttp2-0__1.41.0-1.fc32.x86_64",
    sha256 = "a22b0bbe8feeb6bf43b6fb2ebae8c869061df791549f0b958a77cd44cdb05bd3",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libnghttp2-1.41.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a22b0bbe8feeb6bf43b6fb2ebae8c869061df791549f0b958a77cd44cdb05bd3",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-2.fc32.aarch64",
    sha256 = "231cefc11eb5a9ac8f23bbd294cef0bf3a690040df3048e063f8a269f2db75f8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnl3-3.5.0-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libnl3-3.5.0-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnl3-3.5.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnl3-3.5.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/231cefc11eb5a9ac8f23bbd294cef0bf3a690040df3048e063f8a269f2db75f8",
    ],
)

rpm(
    name = "libnl3-0__3.5.0-2.fc32.x86_64",
    sha256 = "8dfdbe51193bdcfc3db41b5b9f317f009bfab6373e6ed3c5475466b8772a85e1",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnl3-3.5.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8dfdbe51193bdcfc3db41b5b9f317f009bfab6373e6ed3c5475466b8772a85e1",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-6.20180605git4a062cf.fc32.aarch64",
    sha256 = "4139803076f102e2224b81b4f1da3f6d066b89e272201d2720557763f9acfcd5",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4139803076f102e2224b81b4f1da3f6d066b89e272201d2720557763f9acfcd5",
    ],
)

rpm(
    name = "libnsl2-0__1.2.0-6.20180605git4a062cf.fc32.x86_64",
    sha256 = "3b4ce7fc4e2778758881feedf6ea19b65e99aa3672e19a7dd62977efe3b910b9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libnsl2-1.2.0-6.20180605git4a062cf.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b4ce7fc4e2778758881feedf6ea19b65e99aa3672e19a7dd62977efe3b910b9",
    ],
)

rpm(
    name = "libpcap-14__1.10.0-1.fc32.aarch64",
    sha256 = "54290c3171279c693a8f49d40f4bac2111af55421da74293bd1fa07b0ba7d396",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpcap-1.10.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpcap-1.10.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpcap-1.10.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpcap-1.10.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/54290c3171279c693a8f49d40f4bac2111af55421da74293bd1fa07b0ba7d396",
    ],
)

rpm(
    name = "libpcap-14__1.10.0-1.fc32.x86_64",
    sha256 = "f5fd842fae691bfc41dd107db33bc0457508a5e1c82ebc277e5fe10d64c9659e",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpcap-1.10.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpcap-1.10.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpcap-1.10.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpcap-1.10.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f5fd842fae691bfc41dd107db33bc0457508a5e1c82ebc277e5fe10d64c9659e",
    ],
)

rpm(
    name = "libpciaccess-0__0.16-2.fc32.x86_64",
    sha256 = "4630ea4b3e4c439f2d3bc4cf33e71388e65cd5394338c86fa1074c6f7fa0d801",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpciaccess-0.16-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4630ea4b3e4c439f2d3bc4cf33e71388e65cd5394338c86fa1074c6f7fa0d801",
    ],
)

rpm(
    name = "libpkgconf-0__1.6.3-3.fc32.aarch64",
    sha256 = "fa2ea31650026dd9e700272f7da76066fda950f23c9126a7898ccbdd9468402d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libpkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libpkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libpkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libpkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fa2ea31650026dd9e700272f7da76066fda950f23c9126a7898ccbdd9468402d",
    ],
)

rpm(
    name = "libpkgconf-0__1.6.3-3.fc32.x86_64",
    sha256 = "6952dfc6a8f583c9aeafb16d5d34208d7e39fd7ec8628c5aa8ccde039acbe548",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6952dfc6a8f583c9aeafb16d5d34208d7e39fd7ec8628c5aa8ccde039acbe548",
    ],
)

rpm(
    name = "libpmem-0__1.8-2.fc32.x86_64",
    sha256 = "23115d324b418df21e859a11182b3570ce50407eecdeecb17257d72089c96c9f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpmem-1.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/23115d324b418df21e859a11182b3570ce50407eecdeecb17257d72089c96c9f",
    ],
)

rpm(
    name = "libpng-2__1.6.37-3.fc32.aarch64",
    sha256 = "f8289df7145ef4a5c57e53552ebf710d363ac8ce3b814e2fcd8470dc522ad9ab",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libpng-1.6.37-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libpng-1.6.37-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libpng-1.6.37-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libpng-1.6.37-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f8289df7145ef4a5c57e53552ebf710d363ac8ce3b814e2fcd8470dc522ad9ab",
    ],
)

rpm(
    name = "libpng-2__1.6.37-3.fc32.x86_64",
    sha256 = "8e4e38b74567a3b0df7c951a100956fa6b91868bec34717015ae667f6d4cbfcf",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libpng-1.6.37-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8e4e38b74567a3b0df7c951a100956fa6b91868bec34717015ae667f6d4cbfcf",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-1.fc32.aarch64",
    sha256 = "e38a8997526c03cd55aebe038679fc5dd56dffaacf6daec1e16b698335c87081",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpwquality-1.4.4-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpwquality-1.4.4-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpwquality-1.4.4-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libpwquality-1.4.4-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e38a8997526c03cd55aebe038679fc5dd56dffaacf6daec1e16b698335c87081",
    ],
)

rpm(
    name = "libpwquality-0__1.4.4-1.fc32.x86_64",
    sha256 = "583e4f689dc478f68942fa650e2b495db63bd29d13b3e075a3effcccf29260da",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libpwquality-1.4.4-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/583e4f689dc478f68942fa650e2b495db63bd29d13b3e075a3effcccf29260da",
    ],
)

rpm(
    name = "librdmacm-0__33.0-2.fc32.aarch64",
    sha256 = "e9dd0497df2c66cd91a328e4275e0b45744213816a5a16f7db5450314d65c27e",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/librdmacm-33.0-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/librdmacm-33.0-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/librdmacm-33.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/librdmacm-33.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e9dd0497df2c66cd91a328e4275e0b45744213816a5a16f7db5450314d65c27e",
    ],
)

rpm(
    name = "librdmacm-0__33.0-2.fc32.x86_64",
    sha256 = "f7e254fe335fb88d352c4bd883038de2a6b0ebee28ea88b2e3a20ecd329445da",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-33.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-33.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-33.0-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/librdmacm-33.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7e254fe335fb88d352c4bd883038de2a6b0ebee28ea88b2e3a20ecd329445da",
    ],
)

rpm(
    name = "libreport-filesystem-0__2.13.1-5.fc32.x86_64",
    sha256 = "b7bb6b8871a4b71a4364baa3127dca98e5359029b5111af4fc4d932edd056d7d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libreport-filesystem-2.13.1-5.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libreport-filesystem-2.13.1-5.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libreport-filesystem-2.13.1-5.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libreport-filesystem-2.13.1-5.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/b7bb6b8871a4b71a4364baa3127dca98e5359029b5111af4fc4d932edd056d7d",
    ],
)

rpm(
    name = "libseccomp-0__2.5.0-3.fc32.aarch64",
    sha256 = "5b79bd153f79c699f98ecdb9fd87958bb2e4d13840a7d6110f777709af88e812",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libseccomp-2.5.0-3.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libseccomp-2.5.0-3.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libseccomp-2.5.0-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libseccomp-2.5.0-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5b79bd153f79c699f98ecdb9fd87958bb2e4d13840a7d6110f777709af88e812",
    ],
)

rpm(
    name = "libseccomp-0__2.5.0-3.fc32.x86_64",
    sha256 = "7cb644e997c1f247f18ff981a0b03479cc3369871f16199e32da70370ead6faf",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libseccomp-2.5.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7cb644e997c1f247f18ff981a0b03479cc3369871f16199e32da70370ead6faf",
    ],
)

rpm(
    name = "libselinux-0__3.0-5.fc32.aarch64",
    sha256 = "0c63919d8af7844dde1feb17b949dac124091eeb38caa934a0d52621ea3e23f3",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-3.0-5.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-3.0-5.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-3.0-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-3.0-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0c63919d8af7844dde1feb17b949dac124091eeb38caa934a0d52621ea3e23f3",
    ],
)

rpm(
    name = "libselinux-0__3.0-5.fc32.x86_64",
    sha256 = "89a698ab28668b4374abb505de1cc140ffec611014622e8841ecb6fac8c888a3",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-3.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/89a698ab28668b4374abb505de1cc140ffec611014622e8841ecb6fac8c888a3",
    ],
)

rpm(
    name = "libselinux-utils-0__3.0-5.fc32.aarch64",
    sha256 = "9cb1ba92b46da8b777d96287042210b246f8519eb8313063ab94fd704aabd461",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-utils-3.0-5.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-utils-3.0-5.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-utils-3.0-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libselinux-utils-3.0-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9cb1ba92b46da8b777d96287042210b246f8519eb8313063ab94fd704aabd461",
    ],
)

rpm(
    name = "libselinux-utils-0__3.0-5.fc32.x86_64",
    sha256 = "fa31d8d160d66400d6a7d87a3686f70d5a471b70263043ca5c24d96307a82d85",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libselinux-utils-3.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fa31d8d160d66400d6a7d87a3686f70d5a471b70263043ca5c24d96307a82d85",
    ],
)

rpm(
    name = "libsemanage-0__3.0-3.fc32.aarch64",
    sha256 = "b78889f3a2ac801456c643fd5603017383221aa33eac381e4f74b9a13fbf3830",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b78889f3a2ac801456c643fd5603017383221aa33eac381e4f74b9a13fbf3830",
    ],
)

rpm(
    name = "libsemanage-0__3.0-3.fc32.x86_64",
    sha256 = "54cb827278ae474cbab1f05e0fbee0355bee2674d46a804f1c2b78ff80a48caa",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/54cb827278ae474cbab1f05e0fbee0355bee2674d46a804f1c2b78ff80a48caa",
    ],
)

rpm(
    name = "libsepol-0__3.0-4.fc32.aarch64",
    sha256 = "8a4e47749ccb657f8bb5e941cab30b5a74d618893f791cc682076b865d8f54fa",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsepol-3.0-4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsepol-3.0-4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsepol-3.0-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsepol-3.0-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8a4e47749ccb657f8bb5e941cab30b5a74d618893f791cc682076b865d8f54fa",
    ],
)

rpm(
    name = "libsepol-0__3.0-4.fc32.x86_64",
    sha256 = "bcf4ca8e5e1d71a12c5e4d966c248b53ef0300a794ca607b9072145f4212e7a1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsepol-3.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bcf4ca8e5e1d71a12c5e4d966c248b53ef0300a794ca607b9072145f4212e7a1",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-10.fc32.aarch64",
    sha256 = "836a45edfd4e2cda0b6bac254b2e6225aad36f9bae0f96f2fe7da42896db0dae",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libsigsegv-2.11-10.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libsigsegv-2.11-10.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libsigsegv-2.11-10.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libsigsegv-2.11-10.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/836a45edfd4e2cda0b6bac254b2e6225aad36f9bae0f96f2fe7da42896db0dae",
    ],
)

rpm(
    name = "libsigsegv-0__2.11-10.fc32.x86_64",
    sha256 = "942707884401498938fba6e2439dc923d4e2d81f4bac205f4e73d458e9879927",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libsigsegv-2.11-10.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/942707884401498938fba6e2439dc923d4e2d81f4bac205f4e73d458e9879927",
    ],
)

rpm(
    name = "libsmartcols-0__2.35.2-1.fc32.aarch64",
    sha256 = "c4a38ef54c313e9c201bdc93ec3f9f6cd0546e546489e453f2fc0c6d15752006",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsmartcols-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsmartcols-2.35.2-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsmartcols-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libsmartcols-2.35.2-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c4a38ef54c313e9c201bdc93ec3f9f6cd0546e546489e453f2fc0c6d15752006",
    ],
)

rpm(
    name = "libsmartcols-0__2.35.2-1.fc32.x86_64",
    sha256 = "82a0c6703444fa28ab032b3e4aa355deabff92f3f39d5490faa5c9b9150eaceb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libsmartcols-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82a0c6703444fa28ab032b3e4aa355deabff92f3f39d5490faa5c9b9150eaceb",
    ],
)

rpm(
    name = "libss-0__1.45.5-3.fc32.aarch64",
    sha256 = "a830bb13938bedaf5cc91b13ab78e2cf9172b06727b7e9e1bec2cddce8dd9e2d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libss-1.45.5-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libss-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libss-1.45.5-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libss-1.45.5-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a830bb13938bedaf5cc91b13ab78e2cf9172b06727b7e9e1bec2cddce8dd9e2d",
    ],
)

rpm(
    name = "libss-0__1.45.5-3.fc32.x86_64",
    sha256 = "27701cda24f5f6386e0173745aabc4f6df28052975e73529854432c35399cfc8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libss-1.45.5-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/27701cda24f5f6386e0173745aabc4f6df28052975e73529854432c35399cfc8",
    ],
)

rpm(
    name = "libssh-0__0.9.5-1.fc32.aarch64",
    sha256 = "7bb2567834a0425528560872eb0b3b21aa8a22b9b861af9fba9e1a2471070c4e",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-0.9.5-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-0.9.5-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-0.9.5-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-0.9.5-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7bb2567834a0425528560872eb0b3b21aa8a22b9b861af9fba9e1a2471070c4e",
    ],
)

rpm(
    name = "libssh-0__0.9.5-1.fc32.x86_64",
    sha256 = "1f92b99b1ea563cf2642ce9b9c831d3f1fb8547922d6ca0d728de2c0d751c4e9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-0.9.5-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f92b99b1ea563cf2642ce9b9c831d3f1fb8547922d6ca0d728de2c0d751c4e9",
    ],
)

rpm(
    name = "libssh-config-0__0.9.5-1.fc32.aarch64",
    sha256 = "a11725cb639c2dc043ae096c450cb20b3a10116e55e2ec4eee80d8f111037428",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a11725cb639c2dc043ae096c450cb20b3a10116e55e2ec4eee80d8f111037428",
    ],
)

rpm(
    name = "libssh-config-0__0.9.5-1.fc32.x86_64",
    sha256 = "a11725cb639c2dc043ae096c450cb20b3a10116e55e2ec4eee80d8f111037428",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libssh-config-0.9.5-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a11725cb639c2dc043ae096c450cb20b3a10116e55e2ec4eee80d8f111037428",
    ],
)

rpm(
    name = "libssh2-0__1.9.0-5.fc32.aarch64",
    sha256 = "fc19146120ceea3eb37c062eaea70f65099d94a4c503b0cbc1a0c316ca4177ab",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libssh2-1.9.0-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libssh2-1.9.0-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libssh2-1.9.0-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libssh2-1.9.0-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fc19146120ceea3eb37c062eaea70f65099d94a4c503b0cbc1a0c316ca4177ab",
    ],
)

rpm(
    name = "libssh2-0__1.9.0-5.fc32.x86_64",
    sha256 = "2c811783245ad49873157c16b26cd79efc62afccaaf5077ae2821b8afad175b8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libssh2-1.9.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2c811783245ad49873157c16b26cd79efc62afccaaf5077ae2821b8afad175b8",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__10.3.1-1.fc32.aarch64",
    sha256 = "f8a11dfcc43e1c18d502db508fd4f119acf853968484787b4cce2e0c1780f336",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libstdc++-10.3.1-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f8a11dfcc43e1c18d502db508fd4f119acf853968484787b4cce2e0c1780f336",
    ],
)

rpm(
    name = "libstdc__plus____plus__-0__10.3.1-1.fc32.x86_64",
    sha256 = "cc6e2a4308fbe33c586fb844fef192778538c3c637b9f0ed873205786090b80d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libstdc++-10.3.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cc6e2a4308fbe33c586fb844fef192778538c3c637b9f0ed873205786090b80d",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-1.fc32.aarch64",
    sha256 = "ea44ae1c951d3d4b30ff2a2d898c041ce9072acc94d6ea1e0e305c45e802019f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libtasn1-4.16.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ea44ae1c951d3d4b30ff2a2d898c041ce9072acc94d6ea1e0e305c45e802019f",
    ],
)

rpm(
    name = "libtasn1-0__4.16.0-1.fc32.x86_64",
    sha256 = "052d04c9a6697c6e5aa546546ae5058d547fc4a4f474d2805a3e45dbf69193c6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libtasn1-4.16.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/052d04c9a6697c6e5aa546546ae5058d547fc4a4f474d2805a3e45dbf69193c6",
    ],
)

rpm(
    name = "libtextstyle-0__0.21-1.fc32.aarch64",
    sha256 = "16ebf0b3907e26ffcc80b5a2ad206a93a0aeedb8f40fef9113117b10ba6a60f3",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtextstyle-0.21-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtextstyle-0.21-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtextstyle-0.21-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtextstyle-0.21-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/16ebf0b3907e26ffcc80b5a2ad206a93a0aeedb8f40fef9113117b10ba6a60f3",
    ],
)

rpm(
    name = "libtextstyle-0__0.21-1.fc32.x86_64",
    sha256 = "2ee0dd5aa79c4327da8616903ff4d5956b687b3e28d2f45b7ab219fbd1326b0b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtextstyle-0.21-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2ee0dd5aa79c4327da8616903ff4d5956b687b3e28d2f45b7ab219fbd1326b0b",
    ],
)

rpm(
    name = "libtirpc-0__1.2.6-1.rc4.fc32.aarch64",
    sha256 = "4dae321b67e99ce300240104e698b1ebee8ad7c8e18f1f9b649d5dea8d05a0f2",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4dae321b67e99ce300240104e698b1ebee8ad7c8e18f1f9b649d5dea8d05a0f2",
    ],
)

rpm(
    name = "libtirpc-0__1.2.6-1.rc4.fc32.x86_64",
    sha256 = "84c6b2d0dbb6181611816f642725005992522009993716482a3037294ef22954",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtirpc-1.2.6-1.rc4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84c6b2d0dbb6181611816f642725005992522009993716482a3037294ef22954",
    ],
)

rpm(
    name = "libtpms-0__0.7.7-0.20210302gitfd5bd3fb1d.fc32.aarch64",
    sha256 = "0434c4ac983350c721f7aca1c1932d97610e6d26da48499c89068d777a557d39",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0434c4ac983350c721f7aca1c1932d97610e6d26da48499c89068d777a557d39",
    ],
)

rpm(
    name = "libtpms-0__0.7.7-0.20210302gitfd5bd3fb1d.fc32.x86_64",
    sha256 = "8deef9482fe7c7fa97f44396481b3c47d0a650948affbbdaa0e2f0f47e65903c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libtpms-0.7.7-0.20210302gitfd5bd3fb1d.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8deef9482fe7c7fa97f44396481b3c47d0a650948affbbdaa0e2f0f47e65903c",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-7.fc32.aarch64",
    sha256 = "2d7ad38e86f5109c732a32bf9bea612c4c674aba6ad4cca2d211d826edc7fd6f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libunistring-0.9.10-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libunistring-0.9.10-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libunistring-0.9.10-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libunistring-0.9.10-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2d7ad38e86f5109c732a32bf9bea612c4c674aba6ad4cca2d211d826edc7fd6f",
    ],
)

rpm(
    name = "libunistring-0__0.9.10-7.fc32.x86_64",
    sha256 = "fb06aa3d8059406a23694ddafe0ef340ca627dd68bf3f351f094de58ef30fb2c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libunistring-0.9.10-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb06aa3d8059406a23694ddafe0ef340ca627dd68bf3f351f094de58ef30fb2c",
    ],
)

rpm(
    name = "libusal-0__1.1.11-44.fc32.x86_64",
    sha256 = "7c9f879b67d1d5a6f40fa551b75f69ac744e5d0d795154655b8d55ef15512af0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libusal-1.1.11-44.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7c9f879b67d1d5a6f40fa551b75f69ac744e5d0d795154655b8d55ef15512af0",
    ],
)

rpm(
    name = "libusbx-0__1.0.24-2.fc32.aarch64",
    sha256 = "dbb6addbbc17bd1533a5e395b8b92a4317c71a7d489e42597288c7659dec1d2a",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libusbx-1.0.24-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dbb6addbbc17bd1533a5e395b8b92a4317c71a7d489e42597288c7659dec1d2a",
    ],
)

rpm(
    name = "libusbx-0__1.0.24-2.fc32.x86_64",
    sha256 = "120ab6d685396674d4ce0d9c53f1dbfde9be617e9fa091a2ed1e8fb199c6699e",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libusbx-1.0.24-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/120ab6d685396674d4ce0d9c53f1dbfde9be617e9fa091a2ed1e8fb199c6699e",
    ],
)

rpm(
    name = "libutempter-0__1.1.6-18.fc32.aarch64",
    sha256 = "22954219a63638d7418204d818c01a0e3c914e2b2eb970f2e4638dcf5a7a5634",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libutempter-1.1.6-18.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libutempter-1.1.6-18.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libutempter-1.1.6-18.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libutempter-1.1.6-18.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/22954219a63638d7418204d818c01a0e3c914e2b2eb970f2e4638dcf5a7a5634",
    ],
)

rpm(
    name = "libutempter-0__1.1.6-18.fc32.x86_64",
    sha256 = "f9ccea65ecf98f4dfac65d25986d08efa62a1d1c0db9db0a061e7408d6805a1a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libutempter-1.1.6-18.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f9ccea65ecf98f4dfac65d25986d08efa62a1d1c0db9db0a061e7408d6805a1a",
    ],
)

rpm(
    name = "libuuid-0__2.35.2-1.fc32.aarch64",
    sha256 = "a38c52fddd22df199cff6ba1c8b7ca051098a9c456afc775465c6b5b500cd59f",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libuuid-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libuuid-2.35.2-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libuuid-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libuuid-2.35.2-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a38c52fddd22df199cff6ba1c8b7ca051098a9c456afc775465c6b5b500cd59f",
    ],
)

rpm(
    name = "libuuid-0__2.35.2-1.fc32.x86_64",
    sha256 = "20ad2f907034a1c3e76dd4691886223bf588ff946fd57545ecdfcd58bc4c3b4b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libuuid-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/20ad2f907034a1c3e76dd4691886223bf588ff946fd57545ecdfcd58bc4c3b4b",
    ],
)

rpm(
    name = "libverto-0__0.3.0-9.fc32.aarch64",
    sha256 = "c494a613443f49b6cca4845f9c3410a1267f609c503a81a9a26a272443708fee",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libverto-0.3.0-9.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libverto-0.3.0-9.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libverto-0.3.0-9.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libverto-0.3.0-9.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c494a613443f49b6cca4845f9c3410a1267f609c503a81a9a26a272443708fee",
    ],
)

rpm(
    name = "libverto-0__0.3.0-9.fc32.x86_64",
    sha256 = "ed84414c9b2190d3026f58db78dffd8bc3a9ad40311cb0adb8ff8e3c7c06ca60",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libverto-0.3.0-9.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ed84414c9b2190d3026f58db78dffd8bc3a9ad40311cb0adb8ff8e3c7c06ca60",
    ],
)

rpm(
    name = "libvirt-bash-completion-0__7.0.0-12.fc32.aarch64",
    sha256 = "804cf3f136cd05283f56ef5f1a606b013e43b8bb43d48df8f436b83fe78ef8b6",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-aarch64/02116091-libvirt/libvirt-bash-completion-7.0.0-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/804cf3f136cd05283f56ef5f1a606b013e43b8bb43d48df8f436b83fe78ef8b6",
    ],
)

rpm(
    name = "libvirt-bash-completion-0__7.0.0-12.fc32.x86_64",
    sha256 = "b3f501741bab94b412ebc8b359c29a9f2213e11ec110bbf57a131aff52fd6ef2",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-bash-completion-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b3f501741bab94b412ebc8b359c29a9f2213e11ec110bbf57a131aff52fd6ef2",
    ],
)

rpm(
    name = "libvirt-client-0__7.0.0-12.fc32.aarch64",
    sha256 = "a1666f035f7efab04a51a3782884c1fdfd1cb3e036db8ebb39fbdecb88f5645c",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-aarch64/02116091-libvirt/libvirt-client-7.0.0-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a1666f035f7efab04a51a3782884c1fdfd1cb3e036db8ebb39fbdecb88f5645c",
    ],
)

rpm(
    name = "libvirt-client-0__7.0.0-12.fc32.x86_64",
    sha256 = "9ea2e9893c9180fa31f2a0b91c6eaef3a12804c261224ad5150cd20e27292acc",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-client-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9ea2e9893c9180fa31f2a0b91c6eaef3a12804c261224ad5150cd20e27292acc",
    ],
)

rpm(
    name = "libvirt-daemon-0__7.0.0-12.fc32.aarch64",
    sha256 = "ccb5939f97bcfa66887c46aeb79e6469d3be20bf41e7d68b5abcdd6520343882",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-aarch64/02116091-libvirt/libvirt-daemon-7.0.0-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ccb5939f97bcfa66887c46aeb79e6469d3be20bf41e7d68b5abcdd6520343882",
    ],
)

rpm(
    name = "libvirt-daemon-0__7.0.0-12.fc32.x86_64",
    sha256 = "2657470eabb741afa7ffffd0fd7b74bf2ed12bed0517874060b2e2d28c107986",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2657470eabb741afa7ffffd0fd7b74bf2ed12bed0517874060b2e2d28c107986",
    ],
)

rpm(
    name = "libvirt-daemon-driver-interface-0__7.0.0-12.fc32.x86_64",
    sha256 = "be1be0e962a115f79bf7d67649130cc109ef551d4e8c289f40dea21109927496",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-driver-interface-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be1be0e962a115f79bf7d67649130cc109ef551d4e8c289f40dea21109927496",
    ],
)

rpm(
    name = "libvirt-daemon-driver-network-0__7.0.0-12.fc32.x86_64",
    sha256 = "aa1b9e85ef0e6c49781f1fc3ee466ad4e4d45a26049c468f68fd756898093045",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-driver-network-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/aa1b9e85ef0e6c49781f1fc3ee466ad4e4d45a26049c468f68fd756898093045",
    ],
)

rpm(
    name = "libvirt-daemon-driver-nodedev-0__7.0.0-12.fc32.x86_64",
    sha256 = "deac2aa9f8e67841ae3d30253bccc2d1682426e4e7623dedd7d6c2a84f878488",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-driver-nodedev-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/deac2aa9f8e67841ae3d30253bccc2d1682426e4e7623dedd7d6c2a84f878488",
    ],
)

rpm(
    name = "libvirt-daemon-driver-nwfilter-0__7.0.0-12.fc32.x86_64",
    sha256 = "72388797f69c60d32347cf09fd3b705a1212405d20781ad1b0ee6f07a5182514",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-driver-nwfilter-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/72388797f69c60d32347cf09fd3b705a1212405d20781ad1b0ee6f07a5182514",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.0.0-12.fc32.aarch64",
    sha256 = "1a6160b0e50f951908185c7a9cbb4a0f252a4aae9cdaa014f576c5493f27fdec",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-aarch64/02116091-libvirt/libvirt-daemon-driver-qemu-7.0.0-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1a6160b0e50f951908185c7a9cbb4a0f252a4aae9cdaa014f576c5493f27fdec",
    ],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.0.0-12.fc32.x86_64",
    sha256 = "66a9dfaa290d4114a1b249043d1d02e8f19893dd055e6abf485447b61e1a5384",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-driver-qemu-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/66a9dfaa290d4114a1b249043d1d02e8f19893dd055e6abf485447b61e1a5384",
    ],
)

rpm(
    name = "libvirt-daemon-driver-secret-0__7.0.0-12.fc32.x86_64",
    sha256 = "bc9d5d54cc62456800019f12b3fbedc0907c1be0617e0ae31d429286019b9290",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-driver-secret-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc9d5d54cc62456800019f12b3fbedc0907c1be0617e0ae31d429286019b9290",
    ],
)

rpm(
    name = "libvirt-daemon-kvm-0__7.0.0-12.fc32.x86_64",
    sha256 = "1889d4f8ba4fa9963689e824f918c7c4e642ab258f2dba186d3e7ef893cecca2",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-daemon-kvm-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1889d4f8ba4fa9963689e824f918c7c4e642ab258f2dba186d3e7ef893cecca2",
    ],
)

rpm(
    name = "libvirt-devel-0__7.0.0-12.fc32.aarch64",
    sha256 = "eab4911332e62d44cbbdaafba3a7c756be6d45ab68d186d2b15fa6f0da547f1b",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-aarch64/02116091-libvirt/libvirt-devel-7.0.0-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eab4911332e62d44cbbdaafba3a7c756be6d45ab68d186d2b15fa6f0da547f1b",
    ],
)

rpm(
    name = "libvirt-devel-0__7.0.0-12.fc32.x86_64",
    sha256 = "1aa01815a6f9e94bc23b3a3014345fb3a3abc0d0f89e607268baab95de264823",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-devel-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1aa01815a6f9e94bc23b3a3014345fb3a3abc0d0f89e607268baab95de264823",
    ],
)

rpm(
    name = "libvirt-libs-0__7.0.0-12.fc32.aarch64",
    sha256 = "42da6eb3bf7c2c3bddf5e1184c7fcba8bc40da42072c817fe51ff14e005577a6",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-aarch64/02116091-libvirt/libvirt-libs-7.0.0-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/42da6eb3bf7c2c3bddf5e1184c7fcba8bc40da42072c817fe51ff14e005577a6",
    ],
)

rpm(
    name = "libvirt-libs-0__7.0.0-12.fc32.x86_64",
    sha256 = "0cff0aa7af597987bed2654d7145da48c5cf70927c5bf4000170baf9908db93a",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/libvirt-7.0.0-12.el8/fedora-32-x86_64/02116091-libvirt/libvirt-libs-7.0.0-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0cff0aa7af597987bed2654d7145da48c5cf70927c5bf4000170baf9908db93a",
    ],
)

rpm(
    name = "libwsman1-0__2.6.8-12.fc32.aarch64",
    sha256 = "00c47d07358e79269feafa053c2c19095eeb9d936a3ec5e541ca115f889c8756",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libwsman1-2.6.8-12.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libwsman1-2.6.8-12.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libwsman1-2.6.8-12.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libwsman1-2.6.8-12.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/00c47d07358e79269feafa053c2c19095eeb9d936a3ec5e541ca115f889c8756",
    ],
)

rpm(
    name = "libwsman1-0__2.6.8-12.fc32.x86_64",
    sha256 = "01e436bc768c2aa1e3ecac82ab5d3f9efc3d9b49b1879202182054abfcf8a618",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libwsman1-2.6.8-12.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/01e436bc768c2aa1e3ecac82ab5d3f9efc3d9b49b1879202182054abfcf8a618",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.20-2.fc32.aarch64",
    sha256 = "d7876cd851019cce6e8599975a1706384465fc288cc2b3ca05073ce61da0242f",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxcrypt-4.4.20-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxcrypt-4.4.20-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxcrypt-4.4.20-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxcrypt-4.4.20-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d7876cd851019cce6e8599975a1706384465fc288cc2b3ca05073ce61da0242f",
    ],
)

rpm(
    name = "libxcrypt-0__4.4.20-2.fc32.x86_64",
    sha256 = "f9d0ff28aba32a66943a9c347d98a48f0508b9ae2431ce2c4f84e87b91714946",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.20-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.20-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.20-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxcrypt-4.4.20-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f9d0ff28aba32a66943a9c347d98a48f0508b9ae2431ce2c4f84e87b91714946",
    ],
)

rpm(
    name = "libxkbcommon-0__0.10.0-2.fc32.aarch64",
    sha256 = "9db3ade981c564c361eed9068cd35acac93c1b1db54b6fb2a74070ce68141cff",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9db3ade981c564c361eed9068cd35acac93c1b1db54b6fb2a74070ce68141cff",
    ],
)

rpm(
    name = "libxkbcommon-0__0.10.0-2.fc32.x86_64",
    sha256 = "ae219ad5ecc0233271c3fd61263f817c646eecece19a8f075e7aa4dd9ff8698e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/libxkbcommon-0.10.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ae219ad5ecc0233271c3fd61263f817c646eecece19a8f075e7aa4dd9ff8698e",
    ],
)

rpm(
    name = "libxml2-0__2.9.10-8.fc32.aarch64",
    sha256 = "6eeedd222b9def68c260de99b3dbfb2d764b78de5b70e112e9cf2b0f70376cf7",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxml2-2.9.10-8.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxml2-2.9.10-8.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxml2-2.9.10-8.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libxml2-2.9.10-8.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6eeedd222b9def68c260de99b3dbfb2d764b78de5b70e112e9cf2b0f70376cf7",
    ],
)

rpm(
    name = "libxml2-0__2.9.10-8.fc32.x86_64",
    sha256 = "60f2deeac94c8d58b305a8faea0701a3fe5dd74909953bf8fe5e9c26169facd1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxml2-2.9.10-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/60f2deeac94c8d58b305a8faea0701a3fe5dd74909953bf8fe5e9c26169facd1",
    ],
)

rpm(
    name = "libxslt-0__1.1.34-4.fc32.x86_64",
    sha256 = "bc6eb130bf4eafd0978e17e72f7f8896570ad95b6abb1a5a3402b261a0727b4f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxslt-1.1.34-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxslt-1.1.34-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxslt-1.1.34-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libxslt-1.1.34-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bc6eb130bf4eafd0978e17e72f7f8896570ad95b6abb1a5a3402b261a0727b4f",
    ],
)

rpm(
    name = "libzstd-0__1.4.9-1.fc32.aarch64",
    sha256 = "0c974d2ea735aa5132fc6e412a4b0f5979e428f4d9bc491771003c885107f4f7",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libzstd-1.4.9-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libzstd-1.4.9-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libzstd-1.4.9-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libzstd-1.4.9-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0c974d2ea735aa5132fc6e412a4b0f5979e428f4d9bc491771003c885107f4f7",
    ],
)

rpm(
    name = "libzstd-0__1.4.9-1.fc32.x86_64",
    sha256 = "08b63b18fb640a131a05982355c65105fd3295935f7e7a6f495a574440116ff9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libzstd-1.4.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/08b63b18fb640a131a05982355c65105fd3295935f7e7a6f495a574440116ff9",
    ],
)

rpm(
    name = "linux-atm-libs-0__2.5.1-26.fc32.aarch64",
    sha256 = "ae08e152061808ccc334cc611d8ea4d18c05daa6b68731e255a533d0572594ae",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ae08e152061808ccc334cc611d8ea4d18c05daa6b68731e255a533d0572594ae",
    ],
)

rpm(
    name = "linux-atm-libs-0__2.5.1-26.fc32.x86_64",
    sha256 = "c9ba05cb46a9cb52e3325ca20c457a377361abcd0e5a7dda776ba19481770467",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/linux-atm-libs-2.5.1-26.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c9ba05cb46a9cb52e3325ca20c457a377361abcd0e5a7dda776ba19481770467",
    ],
)

rpm(
    name = "lsscsi-0__0.31-1.fc32.x86_64",
    sha256 = "c8e2e020a86d8d10cf77db32a9ff452fa64318cc96552884bcc3f7ea8833da0e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsscsi-0.31-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsscsi-0.31-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsscsi-0.31-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsscsi-0.31-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c8e2e020a86d8d10cf77db32a9ff452fa64318cc96552884bcc3f7ea8833da0e",
    ],
)

rpm(
    name = "lua-libs-0__5.3.5-8.fc32.aarch64",
    sha256 = "780510fc38d55eb6e1e732fef11ea077bdc67ceaae9bb11c38b0cfbc2f6c3062",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/lua-libs-5.3.5-8.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/lua-libs-5.3.5-8.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/lua-libs-5.3.5-8.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/lua-libs-5.3.5-8.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/780510fc38d55eb6e1e732fef11ea077bdc67ceaae9bb11c38b0cfbc2f6c3062",
    ],
)

rpm(
    name = "lua-libs-0__5.3.5-8.fc32.x86_64",
    sha256 = "09c4524a71762b4ba0aa95e296059130e1ef4007963c2b4d2803ea4eb5fdfed4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/lua-libs-5.3.5-8.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09c4524a71762b4ba0aa95e296059130e1ef4007963c2b4d2803ea4eb5fdfed4",
    ],
)

rpm(
    name = "lvm2-0__2.03.09-1.fc32.x86_64",
    sha256 = "3b76bc46dd279404408d34946cfdb0c3899359a1c6b48e614e63d1259a94262a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-2.03.09-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-2.03.09-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-2.03.09-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-2.03.09-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3b76bc46dd279404408d34946cfdb0c3899359a1c6b48e614e63d1259a94262a",
    ],
)

rpm(
    name = "lvm2-libs-0__2.03.09-1.fc32.x86_64",
    sha256 = "e851ba0019baa83e1bebbe92e1a1cf629694ccf3b42c5ff84e0ed7bea74931d3",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.09-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.09-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.09-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lvm2-libs-2.03.09-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e851ba0019baa83e1bebbe92e1a1cf629694ccf3b42c5ff84e0ed7bea74931d3",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.1-2.fc32.aarch64",
    sha256 = "a7394cd1b11a1b25efaab43a30b1d9687683884babc162f43e29fdee4f00bda8",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lz4-libs-1.9.1-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a7394cd1b11a1b25efaab43a30b1d9687683884babc162f43e29fdee4f00bda8",
    ],
)

rpm(
    name = "lz4-libs-0__1.9.1-2.fc32.x86_64",
    sha256 = "44cfb58b368fba586981aa838a7f3974ac1d66d2b3b695f88d7b1d2e9c81a0b6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lz4-libs-1.9.1-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/44cfb58b368fba586981aa838a7f3974ac1d66d2b3b695f88d7b1d2e9c81a0b6",
    ],
)

rpm(
    name = "lzo-0__2.10-2.fc32.aarch64",
    sha256 = "e460eef82814077925930af888cca4a6788477de26eadeecd0b0f35eb84e8621",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lzo-2.10-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/lzo-2.10-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lzo-2.10-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lzo-2.10-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e460eef82814077925930af888cca4a6788477de26eadeecd0b0f35eb84e8621",
    ],
)

rpm(
    name = "lzo-0__2.10-2.fc32.x86_64",
    sha256 = "4375c398dff722a29bd1700bc8dc8b528345412d1e17d8d9d1176d9774962957",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzo-2.10-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4375c398dff722a29bd1700bc8dc8b528345412d1e17d8d9d1176d9774962957",
    ],
)

rpm(
    name = "lzop-0__1.04-3.fc32.aarch64",
    sha256 = "fd7b84ea759fbe99858dae9cd0f204e7746e4433f7f1c6f627028008916135ec",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lzop-1.04-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/lzop-1.04-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lzop-1.04-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lzop-1.04-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fd7b84ea759fbe99858dae9cd0f204e7746e4433f7f1c6f627028008916135ec",
    ],
)

rpm(
    name = "lzop-0__1.04-3.fc32.x86_64",
    sha256 = "dd0aec170afc0e2113845e9d107a58d72b234414b548880e3154be49ffbaf64a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lzop-1.04-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dd0aec170afc0e2113845e9d107a58d72b234414b548880e3154be49ffbaf64a",
    ],
)

rpm(
    name = "mdadm-0__4.1-5.fc32.x86_64",
    sha256 = "580d0c2839364345d061cf9d133ef3bc43e1c6dfcb552c7aa2104b9ff9f6d912",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdadm-4.1-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdadm-4.1-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdadm-4.1-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdadm-4.1-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/580d0c2839364345d061cf9d133ef3bc43e1c6dfcb552c7aa2104b9ff9f6d912",
    ],
)

rpm(
    name = "mdevctl-0__0.78-1.fc32.x86_64",
    sha256 = "8ca56e4f9ba1a6c89359c1c6d219f10b933571f77965c6ca6f2d5fcda135794f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdevctl-0.78-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdevctl-0.78-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdevctl-0.78-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mdevctl-0.78-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/8ca56e4f9ba1a6c89359c1c6d219f10b933571f77965c6ca6f2d5fcda135794f",
    ],
)

rpm(
    name = "mpfr-0__4.0.2-5.fc32.aarch64",
    sha256 = "374a30310d65af1224208fcb579b6edce08aace53c118e9230233b06492f3622",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/m/mpfr-4.0.2-5.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/m/mpfr-4.0.2-5.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/m/mpfr-4.0.2-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/m/mpfr-4.0.2-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/374a30310d65af1224208fcb579b6edce08aace53c118e9230233b06492f3622",
    ],
)

rpm(
    name = "mpfr-0__4.0.2-5.fc32.x86_64",
    sha256 = "6a97b2d7b510dba87d67436c097dde860dcca5a3464c9b3489ec65fcfe101f22",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mpfr-4.0.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6a97b2d7b510dba87d67436c097dde860dcca5a3464c9b3489ec65fcfe101f22",
    ],
)

rpm(
    name = "mtools-0__4.0.27-1.fc32.x86_64",
    sha256 = "9a657de2e96660b5720fc9bbaad766c680e5c7717abd811d8bda61447297d8d8",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mtools-4.0.27-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mtools-4.0.27-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/m/mtools-4.0.27-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/m/mtools-4.0.27-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9a657de2e96660b5720fc9bbaad766c680e5c7717abd811d8bda61447297d8d8",
    ],
)

rpm(
    name = "ncurses-base-0__6.1-15.20191109.fc32.aarch64",
    sha256 = "25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    ],
)

rpm(
    name = "ncurses-base-0__6.1-15.20191109.fc32.x86_64",
    sha256 = "25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-base-6.1-15.20191109.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/25fc5d288536e1973436da38357690575ed58e03e17ca48d2b3840364f830659",
    ],
)

rpm(
    name = "ncurses-libs-0__6.1-15.20191109.fc32.aarch64",
    sha256 = "a973f92acb0afe61087a69d13a532c18a39dd60b3ba4826b38350f2c6b27e417",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a973f92acb0afe61087a69d13a532c18a39dd60b3ba4826b38350f2c6b27e417",
    ],
)

rpm(
    name = "ncurses-libs-0__6.1-15.20191109.fc32.x86_64",
    sha256 = "04152a3a608d022a58830c0e3dac0818e2c060469b0f41d8d731f659981a4464",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-libs-6.1-15.20191109.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/04152a3a608d022a58830c0e3dac0818e2c060469b0f41d8d731f659981a4464",
    ],
)

rpm(
    name = "netcf-libs-0__0.2.8-15.fc32.x86_64",
    sha256 = "09ec07a1f692918a4ff1ad9e6d33f4c3526e14083d953da2875c4abc65cbdcb6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/netcf-libs-0.2.8-15.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/netcf-libs-0.2.8-15.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/netcf-libs-0.2.8-15.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/netcf-libs-0.2.8-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/09ec07a1f692918a4ff1ad9e6d33f4c3526e14083d953da2875c4abc65cbdcb6",
    ],
)

rpm(
    name = "nettle-0__3.5.1-5.fc32.aarch64",
    sha256 = "15b2402e11402a6cb494bf7ea31ebf10bf1adb0759aab417e63d05916e56aa45",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/nettle-3.5.1-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/nettle-3.5.1-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/nettle-3.5.1-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/nettle-3.5.1-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/15b2402e11402a6cb494bf7ea31ebf10bf1adb0759aab417e63d05916e56aa45",
    ],
)

rpm(
    name = "nettle-0__3.5.1-5.fc32.x86_64",
    sha256 = "c019d23ed2cb3ceb0ac9757a72c3e8b1d31f2a524b889e18049cc7d923bc9466",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nettle-3.5.1-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c019d23ed2cb3ceb0ac9757a72c3e8b1d31f2a524b889e18049cc7d923bc9466",
    ],
)

rpm(
    name = "nftables-1__0.9.3-4.fc32.aarch64",
    sha256 = "b14bec017a48b5d14d3789970682f2cb4ff673c053a1fc8e5acabaa0b86acd27",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nftables-0.9.3-4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nftables-0.9.3-4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/n/nftables-0.9.3-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/n/nftables-0.9.3-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/b14bec017a48b5d14d3789970682f2cb4ff673c053a1fc8e5acabaa0b86acd27",
    ],
)

rpm(
    name = "nftables-1__0.9.3-4.fc32.x86_64",
    sha256 = "0f90219dad602725a9148be111f3ef973805597e38c51dae036c45b686708330",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nftables-0.9.3-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0f90219dad602725a9148be111f3ef973805597e38c51dae036c45b686708330",
    ],
)

rpm(
    name = "nmap-ncat-2__7.80-4.fc32.aarch64",
    sha256 = "9e5c178bc813d22e8545563acde1f91c8a7e7429145085650eed36d83862deed",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nmap-ncat-7.80-4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nmap-ncat-7.80-4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/n/nmap-ncat-7.80-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/n/nmap-ncat-7.80-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9e5c178bc813d22e8545563acde1f91c8a7e7429145085650eed36d83862deed",
    ],
)

rpm(
    name = "nmap-ncat-2__7.80-4.fc32.x86_64",
    sha256 = "35642d97e77aa48070364b7cd2e4704bb53b87b732c7dc484b59f51446aaaca8",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nmap-ncat-7.80-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/35642d97e77aa48070364b7cd2e4704bb53b87b732c7dc484b59f51446aaaca8",
    ],
)

rpm(
    name = "npth-0__1.6-4.fc32.x86_64",
    sha256 = "3c2a641f118ab2e8b08df6dd2da72a60121d02df8d932b4afa2920eb80392875",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/npth-1.6-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/npth-1.6-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/npth-1.6-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/npth-1.6-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3c2a641f118ab2e8b08df6dd2da72a60121d02df8d932b4afa2920eb80392875",
    ],
)

rpm(
    name = "ntfs-3g-2__2017.3.23-13.fc32.x86_64",
    sha256 = "be78c188a2b301e08efd8ebc706d913163c0fa63af0ab0e4f0e20b9783876f43",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-13.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-2017.3.23-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be78c188a2b301e08efd8ebc706d913163c0fa63af0ab0e4f0e20b9783876f43",
    ],
)

rpm(
    name = "ntfs-3g-system-compression-0__1.0-3.fc32.x86_64",
    sha256 = "6c9cffafc624879e806b7c822a4ba1984d36d164336dd3ec712867ac488f1851",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfs-3g-system-compression-1.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6c9cffafc624879e806b7c822a4ba1984d36d164336dd3ec712867ac488f1851",
    ],
)

rpm(
    name = "ntfsprogs-2__2017.3.23-13.fc32.x86_64",
    sha256 = "bf973d907c3ae07ef84df09f90561fe9146fb0d122616ae70cdd7f5c31f24dd8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-13.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ntfsprogs-2017.3.23-13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bf973d907c3ae07ef84df09f90561fe9146fb0d122616ae70cdd7f5c31f24dd8",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.12-4.fc32.aarch64",
    sha256 = "0868dc649de9822dedec8886b90b0abb5e99300d6af5e70f280280d8f738ab8a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/numactl-libs-2.0.12-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/numactl-libs-2.0.12-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/numactl-libs-2.0.12-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/numactl-libs-2.0.12-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/0868dc649de9822dedec8886b90b0abb5e99300d6af5e70f280280d8f738ab8a",
    ],
)

rpm(
    name = "numactl-libs-0__2.0.12-4.fc32.x86_64",
    sha256 = "af4f4317249ad46956ada6c23dd5966a6679581672b52510763ef6324aee95b7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numactl-libs-2.0.12-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/af4f4317249ad46956ada6c23dd5966a6679581672b52510763ef6324aee95b7",
    ],
)

rpm(
    name = "numad-0__0.5-31.20150602git.fc32.aarch64",
    sha256 = "5cc3f288498404687a23168de64d39069c922e6a1cf25a5b4a55ac7714bfe778",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/numad-0.5-31.20150602git.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/numad-0.5-31.20150602git.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/numad-0.5-31.20150602git.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/numad-0.5-31.20150602git.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5cc3f288498404687a23168de64d39069c922e6a1cf25a5b4a55ac7714bfe778",
    ],
)

rpm(
    name = "numad-0__0.5-31.20150602git.fc32.x86_64",
    sha256 = "6bef82ea8e1006735181860421272e799da79568be2ed31e93267de664915992",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/numad-0.5-31.20150602git.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6bef82ea8e1006735181860421272e799da79568be2ed31e93267de664915992",
    ],
)

rpm(
    name = "oniguruma-0__6.9.5-4.rev1.fc32.x86_64",
    sha256 = "351548df2ae1db673e12db9e181171bf8699776e850580e23de581aa1562e7a0",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/oniguruma-6.9.5-4.rev1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/oniguruma-6.9.5-4.rev1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/o/oniguruma-6.9.5-4.rev1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/oniguruma-6.9.5-4.rev1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/351548df2ae1db673e12db9e181171bf8699776e850580e23de581aa1562e7a0",
    ],
)

rpm(
    name = "openldap-0__2.4.47-5.fc32.aarch64",
    sha256 = "76fe60efdd3fb14ea0de71c74b89c92ef5df3537773380acf909b75b1e29993d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/o/openldap-2.4.47-5.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/o/openldap-2.4.47-5.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/o/openldap-2.4.47-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/o/openldap-2.4.47-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/76fe60efdd3fb14ea0de71c74b89c92ef5df3537773380acf909b75b1e29993d",
    ],
)

rpm(
    name = "openldap-0__2.4.47-5.fc32.x86_64",
    sha256 = "d528d4c020ec729776d30db52c141c47afaf991847c0c541870f97613d4b3f1f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openldap-2.4.47-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d528d4c020ec729776d30db52c141c47afaf991847c0c541870f97613d4b3f1f",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-1.fc32.aarch64",
    sha256 = "fa23aa0a68ae4b048c5d5bb1f3af9ab08ea577f978cbb5acceb9a42c89793a9e",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-libs-1.1.1k-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fa23aa0a68ae4b048c5d5bb1f3af9ab08ea577f978cbb5acceb9a42c89793a9e",
    ],
)

rpm(
    name = "openssl-libs-1__1.1.1k-1.fc32.x86_64",
    sha256 = "551a817705d634c04b1ec1a267c7cc2a22f6006efa5fa93e155059c2e9e173fb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-libs-1.1.1k-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/551a817705d634c04b1ec1a267c7cc2a22f6006efa5fa93e155059c2e9e173fb",
    ],
)

rpm(
    name = "opus-0__1.3.1-3.fc32.x86_64",
    sha256 = "fe9c792920ecdecbbc9cb37bcb870a8dc87ea5e046997c2149ebb9d621f56889",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/opus-1.3.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fe9c792920ecdecbbc9cb37bcb870a8dc87ea5e046997c2149ebb9d621f56889",
    ],
)

rpm(
    name = "orc-0__0.4.31-2.fc32.x86_64",
    sha256 = "fb6f26c0f7a3a559b79e8afe1f5c8cbe9df2ee1c440d34868e6e252e7a8401c2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/o/orc-0.4.31-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/fb6f26c0f7a3a559b79e8afe1f5c8cbe9df2ee1c440d34868e6e252e7a8401c2",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-2.fc32.aarch64",
    sha256 = "815bc333fcf31fc6f24d09b128929e55c8b7c9128bd41a72fc48251f241a3bc9",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-0.23.22-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/815bc333fcf31fc6f24d09b128929e55c8b7c9128bd41a72fc48251f241a3bc9",
    ],
)

rpm(
    name = "p11-kit-0__0.23.22-2.fc32.x86_64",
    sha256 = "82c8a7b579114536ff8304dbe648dc0ceda9809035c0de32d7ec3cb70e6985f5",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-0.23.22-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82c8a7b579114536ff8304dbe648dc0ceda9809035c0de32d7ec3cb70e6985f5",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-2.fc32.aarch64",
    sha256 = "11225a6f91c1801ddc55fd254787a9933284e835e4c2cd41c5e41919bdc0383d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/p11-kit-trust-0.23.22-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/11225a6f91c1801ddc55fd254787a9933284e835e4c2cd41c5e41919bdc0383d",
    ],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-2.fc32.x86_64",
    sha256 = "f9d43d0d3b39ed651d08961771231acd0dda56f2afc2deff1d83b25698d85a2c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/p11-kit-trust-0.23.22-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f9d43d0d3b39ed651d08961771231acd0dda56f2afc2deff1d83b25698d85a2c",
    ],
)

rpm(
    name = "pam-0__1.3.1-30.fc32.aarch64",
    sha256 = "d79daf6b4b7a2ceaf71f1eb480ec5608a8ba87eac05161b3912e888b2311b6b8",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pam-1.3.1-30.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pam-1.3.1-30.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pam-1.3.1-30.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pam-1.3.1-30.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d79daf6b4b7a2ceaf71f1eb480ec5608a8ba87eac05161b3912e888b2311b6b8",
    ],
)

rpm(
    name = "pam-0__1.3.1-30.fc32.x86_64",
    sha256 = "b4c15f65d7d7a33d673da78088490ff384eb04006b2a460e049c72dbb0ee0691",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pam-1.3.1-30.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b4c15f65d7d7a33d673da78088490ff384eb04006b2a460e049c72dbb0ee0691",
    ],
)

rpm(
    name = "parted-0__3.3-3.fc32.x86_64",
    sha256 = "4b9da0cedada16eb80a2d66b86aa62c9b026501e13a3e947a5a03a06f41d9dbb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/parted-3.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4b9da0cedada16eb80a2d66b86aa62c9b026501e13a3e947a5a03a06f41d9dbb",
    ],
)

rpm(
    name = "pciutils-0__3.7.0-3.fc32.aarch64",
    sha256 = "36f43e3a742bc621bac39a491607a13ac04ef3bafb16371a9f84d1c32eeeac1a",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-3.7.0-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/36f43e3a742bc621bac39a491607a13ac04ef3bafb16371a9f84d1c32eeeac1a",
    ],
)

rpm(
    name = "pciutils-0__3.7.0-3.fc32.x86_64",
    sha256 = "58f6d6f5be084071f3dcaacf16cb345bcc9114499eeb82fa6cd22c74022cb59d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-3.7.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/58f6d6f5be084071f3dcaacf16cb345bcc9114499eeb82fa6cd22c74022cb59d",
    ],
)

rpm(
    name = "pciutils-libs-0__3.7.0-3.fc32.aarch64",
    sha256 = "13fbd7c0b3a263f52e39f9b984de6ed9ced48bb67dc6f113fe9b9f767a53bcd9",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pciutils-libs-3.7.0-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/13fbd7c0b3a263f52e39f9b984de6ed9ced48bb67dc6f113fe9b9f767a53bcd9",
    ],
)

rpm(
    name = "pciutils-libs-0__3.7.0-3.fc32.x86_64",
    sha256 = "2494975f6529cda1593041b744a4fb1ce3394862c7cb220b7f9993a1967ac2e2",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pciutils-libs-3.7.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/2494975f6529cda1593041b744a4fb1ce3394862c7cb220b7f9993a1967ac2e2",
    ],
)

rpm(
    name = "pcre-0__8.44-2.fc32.aarch64",
    sha256 = "f2437fdfed6aa62c8f0cac788e63a1972cadda3fa6ec0e83c7281f6d539cfa28",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre-8.44-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre-8.44-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre-8.44-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre-8.44-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f2437fdfed6aa62c8f0cac788e63a1972cadda3fa6ec0e83c7281f6d539cfa28",
    ],
)

rpm(
    name = "pcre-0__8.44-2.fc32.x86_64",
    sha256 = "3d6d8a95ef1416fa148f9776f4d8ca347d3346c5f4b7b066563d52d1562aaabd",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre-8.44-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3d6d8a95ef1416fa148f9776f4d8ca347d3346c5f4b7b066563d52d1562aaabd",
    ],
)

rpm(
    name = "pcre2-0__10.36-4.fc32.aarch64",
    sha256 = "ed21f8dde77a051c928481f6822c16a50138f621cdbd5f17c9d3a4c2e4dd8513",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-10.36-4.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-10.36-4.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-10.36-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-10.36-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ed21f8dde77a051c928481f6822c16a50138f621cdbd5f17c9d3a4c2e4dd8513",
    ],
)

rpm(
    name = "pcre2-0__10.36-4.fc32.x86_64",
    sha256 = "9dddbb2cc8577b2b5e2f5090d0098910752f124f194d83498e4db9abedc2754b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-4.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-4.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-10.36-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9dddbb2cc8577b2b5e2f5090d0098910752f124f194d83498e4db9abedc2754b",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.36-4.fc32.aarch64",
    sha256 = "7c9f22ee412d1d06426ece3967b677ef380ead108a20d8bbec24763e6b4e7a06",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7c9f22ee412d1d06426ece3967b677ef380ead108a20d8bbec24763e6b4e7a06",
    ],
)

rpm(
    name = "pcre2-syntax-0__10.36-4.fc32.x86_64",
    sha256 = "7c9f22ee412d1d06426ece3967b677ef380ead108a20d8bbec24763e6b4e7a06",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pcre2-syntax-10.36-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7c9f22ee412d1d06426ece3967b677ef380ead108a20d8bbec24763e6b4e7a06",
    ],
)

rpm(
    name = "pixman-0__0.40.0-1.fc32.aarch64",
    sha256 = "4a17154352dcb5bfdd44094afb06690b27bb998d8a177c90ae2c2cc1d70c8db6",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pixman-0.40.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/pixman-0.40.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/pixman-0.40.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/pixman-0.40.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4a17154352dcb5bfdd44094afb06690b27bb998d8a177c90ae2c2cc1d70c8db6",
    ],
)

rpm(
    name = "pixman-0__0.40.0-1.fc32.x86_64",
    sha256 = "ca51912d142cc9b61657fe53e0297a53200cb2b3a96bb5b6b07c8fc1019d44af",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/pixman-0.40.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ca51912d142cc9b61657fe53e0297a53200cb2b3a96bb5b6b07c8fc1019d44af",
    ],
)

rpm(
    name = "pkgconf-0__1.6.3-3.fc32.aarch64",
    sha256 = "086ee809c06522fba6ee29d354590ea20c7057c511c2cbcdcc277dc17e830c8c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/pkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-1.6.3-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/086ee809c06522fba6ee29d354590ea20c7057c511c2cbcdcc277dc17e830c8c",
    ],
)

rpm(
    name = "pkgconf-0__1.6.3-3.fc32.x86_64",
    sha256 = "5c91890bf33527b9fb422cbed17600e761750a4e596fad3f0d0fa419070e82b0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-1.6.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5c91890bf33527b9fb422cbed17600e761750a4e596fad3f0d0fa419070e82b0",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.6.3-3.fc32.aarch64",
    sha256 = "0bace0cf41921db39247c99bfccb228818b83b68c7b8be7c8c4a92ea298a9a29",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0bace0cf41921db39247c99bfccb228818b83b68c7b8be7c8c4a92ea298a9a29",
    ],
)

rpm(
    name = "pkgconf-m4-0__1.6.3-3.fc32.x86_64",
    sha256 = "0bace0cf41921db39247c99bfccb228818b83b68c7b8be7c8c4a92ea298a9a29",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-m4-1.6.3-3.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0bace0cf41921db39247c99bfccb228818b83b68c7b8be7c8c4a92ea298a9a29",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.6.3-3.fc32.aarch64",
    sha256 = "e42f5ab042161675c6297793f47422fcfc76bce37c9d9d54e8ba01e9cf019969",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e42f5ab042161675c6297793f47422fcfc76bce37c9d9d54e8ba01e9cf019969",
    ],
)

rpm(
    name = "pkgconf-pkg-config-0__1.6.3-3.fc32.x86_64",
    sha256 = "4a7b63b32f176b8861f6ac7363bc8010caea0c323eaa83167227118f05603022",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/pkgconf-pkg-config-1.6.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a7b63b32f176b8861f6ac7363bc8010caea0c323eaa83167227118f05603022",
    ],
)

rpm(
    name = "policycoreutils-0__3.0-2.fc32.aarch64",
    sha256 = "29bcc2f3f85ca7bdc22178af3e16743f55353bd9f25fb4c748d8c9f7117fe56f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-3.0-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-3.0-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-3.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-3.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/29bcc2f3f85ca7bdc22178af3e16743f55353bd9f25fb4c748d8c9f7117fe56f",
    ],
)

rpm(
    name = "policycoreutils-0__3.0-2.fc32.x86_64",
    sha256 = "8df97dcfb42c1667b5d2e4150012eaf96f58eeac4f7b879e0928c8c36e3a7604",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-3.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8df97dcfb42c1667b5d2e4150012eaf96f58eeac4f7b879e0928c8c36e3a7604",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.0-2.fc32.aarch64",
    sha256 = "3cd56dea57c00e2c4a9d5aac69a1e843ebef581ba76dde9d9878082fa1215485",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3cd56dea57c00e2c4a9d5aac69a1e843ebef581ba76dde9d9878082fa1215485",
    ],
)

rpm(
    name = "policycoreutils-python-utils-0__3.0-2.fc32.x86_64",
    sha256 = "3cd56dea57c00e2c4a9d5aac69a1e843ebef581ba76dde9d9878082fa1215485",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/policycoreutils-python-utils-3.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3cd56dea57c00e2c4a9d5aac69a1e843ebef581ba76dde9d9878082fa1215485",
    ],
)

rpm(
    name = "polkit-0__0.116-7.fc32.aarch64",
    sha256 = "056227b8324dbabe392ac9b3e8a28ae7fa1b630f5d06cc156e687b988a49c6bd",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-0.116-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/polkit-0.116-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-0.116-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-0.116-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/056227b8324dbabe392ac9b3e8a28ae7fa1b630f5d06cc156e687b988a49c6bd",
    ],
)

rpm(
    name = "polkit-0__0.116-7.fc32.x86_64",
    sha256 = "d49f0b1c8ecf9bc808ae93e9298a40fbcc124fe67c3bbdd37705b6b5d8cfdd87",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-0.116-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d49f0b1c8ecf9bc808ae93e9298a40fbcc124fe67c3bbdd37705b6b5d8cfdd87",
    ],
)

rpm(
    name = "polkit-libs-0__0.116-7.fc32.aarch64",
    sha256 = "54613bd9e0524bb992bd7779c80a24b12df744085031cb8f3defb5fae55ca0f5",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-libs-0.116-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/polkit-libs-0.116-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-libs-0.116-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-libs-0.116-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/54613bd9e0524bb992bd7779c80a24b12df744085031cb8f3defb5fae55ca0f5",
    ],
)

rpm(
    name = "polkit-libs-0__0.116-7.fc32.x86_64",
    sha256 = "d439ffbe20c8c0e8244e31c0324d60cf959dc1cd6cecc575d7b34509a73e9386",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-libs-0.116-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d439ffbe20c8c0e8244e31c0324d60cf959dc1cd6cecc575d7b34509a73e9386",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-16.fc32.aarch64",
    sha256 = "1bc0bced158db1fdd71c8c9211a6fae4e351720b8156d98059f62a945f97cf72",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1bc0bced158db1fdd71c8c9211a6fae4e351720b8156d98059f62a945f97cf72",
    ],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-16.fc32.x86_64",
    sha256 = "7c7eff31251dedcc3285a8b08c1b18f7fd9ee2e07dff86ad090f45a81e19e85e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/polkit-pkla-compat-0.1-16.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7c7eff31251dedcc3285a8b08c1b18f7fd9ee2e07dff86ad090f45a81e19e85e",
    ],
)

rpm(
    name = "popt-0__1.16-19.fc32.aarch64",
    sha256 = "8f4be33cb040f081bb1f863b92e94ac7838af743cb5a0ce9d8c8ec9a611f71a6",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/popt-1.16-19.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/popt-1.16-19.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/popt-1.16-19.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/popt-1.16-19.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/8f4be33cb040f081bb1f863b92e94ac7838af743cb5a0ce9d8c8ec9a611f71a6",
    ],
)

rpm(
    name = "popt-0__1.16-19.fc32.x86_64",
    sha256 = "8a0c00a69f9cb3a9ffacaf1cdc162c38a1faca76c9b976cb177bdc988902f2d4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/popt-1.16-19.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/8a0c00a69f9cb3a9ffacaf1cdc162c38a1faca76c9b976cb177bdc988902f2d4",
    ],
)

rpm(
    name = "procps-ng-0__3.3.16-2.fc32.aarch64",
    sha256 = "46e0e9c329489ca4f1beb4ed589ef5916e8aa7671f6210bcc02b205d4d547229",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/procps-ng-3.3.16-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/46e0e9c329489ca4f1beb4ed589ef5916e8aa7671f6210bcc02b205d4d547229",
    ],
)

rpm(
    name = "procps-ng-0__3.3.16-2.fc32.x86_64",
    sha256 = "9d360e29f7a54d585853407d778b194ebe299663e83d59394ac122ae1687f61a",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/procps-ng-3.3.16-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9d360e29f7a54d585853407d778b194ebe299663e83d59394ac122ae1687f61a",
    ],
)

rpm(
    name = "psmisc-0__23.3-3.fc32.aarch64",
    sha256 = "1eb386a258cebf600319b1f18344b047c9182485936d96da9c2b1067ac1c1bba",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/psmisc-23.3-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/psmisc-23.3-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/psmisc-23.3-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/psmisc-23.3-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1eb386a258cebf600319b1f18344b047c9182485936d96da9c2b1067ac1c1bba",
    ],
)

rpm(
    name = "psmisc-0__23.3-3.fc32.x86_64",
    sha256 = "be7ba234b6c48717ac0f69fb5868b3caa6ef09fbfc76c42a47b367578cd19444",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/psmisc-23.3-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/be7ba234b6c48717ac0f69fb5868b3caa6ef09fbfc76c42a47b367578cd19444",
    ],
)

rpm(
    name = "python-pip-wheel-0__19.3.1-4.fc32.aarch64",
    sha256 = "3fa72c1b9e6ff09048ad10ccd6339669144595f5e10e905055b25c3d6f0fd3d6",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3fa72c1b9e6ff09048ad10ccd6339669144595f5e10e905055b25c3d6f0fd3d6",
    ],
)

rpm(
    name = "python-pip-wheel-0__19.3.1-4.fc32.x86_64",
    sha256 = "3fa72c1b9e6ff09048ad10ccd6339669144595f5e10e905055b25c3d6f0fd3d6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python-pip-wheel-19.3.1-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3fa72c1b9e6ff09048ad10ccd6339669144595f5e10e905055b25c3d6f0fd3d6",
    ],
)

rpm(
    name = "python-setuptools-wheel-0__41.6.0-2.fc32.aarch64",
    sha256 = "7dd93baaf69a8004ae2cd3b9e6660b862d0b6f399d53c05a27a48a2e276ef1ee",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7dd93baaf69a8004ae2cd3b9e6660b862d0b6f399d53c05a27a48a2e276ef1ee",
    ],
)

rpm(
    name = "python-setuptools-wheel-0__41.6.0-2.fc32.x86_64",
    sha256 = "7dd93baaf69a8004ae2cd3b9e6660b862d0b6f399d53c05a27a48a2e276ef1ee",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python-setuptools-wheel-41.6.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/7dd93baaf69a8004ae2cd3b9e6660b862d0b6f399d53c05a27a48a2e276ef1ee",
    ],
)

rpm(
    name = "python3-0__3.8.10-1.fc32.aarch64",
    sha256 = "6be3f542137bf8321218bdcfe357dfce06db5d6abff48fe2a6b38124bf7840ca",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-3.8.10-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-3.8.10-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-3.8.10-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-3.8.10-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6be3f542137bf8321218bdcfe357dfce06db5d6abff48fe2a6b38124bf7840ca",
    ],
)

rpm(
    name = "python3-0__3.8.10-1.fc32.x86_64",
    sha256 = "714cf76e1206c7782ddb104d8b38cd8cbf4a3b0ad6a759dc6c7b2a71970e4099",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.10-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.10-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.10-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-3.8.10-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/714cf76e1206c7782ddb104d8b38cd8cbf4a3b0ad6a759dc6c7b2a71970e4099",
    ],
)

rpm(
    name = "python3-audit-0__3.0.1-2.fc32.aarch64",
    sha256 = "f71aaf8361ed54021ec46a486844d75d209da81dcf9805ee2759bf37a930f728",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-audit-3.0.1-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-audit-3.0.1-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-audit-3.0.1-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-audit-3.0.1-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/f71aaf8361ed54021ec46a486844d75d209da81dcf9805ee2759bf37a930f728",
    ],
)

rpm(
    name = "python3-audit-0__3.0.1-2.fc32.x86_64",
    sha256 = "21594d86b6e6b069267b83fec2d190ea261fd594eee7c37f18916227de84f0ba",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-audit-3.0.1-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-audit-3.0.1-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-audit-3.0.1-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-audit-3.0.1-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/21594d86b6e6b069267b83fec2d190ea261fd594eee7c37f18916227de84f0ba",
    ],
)

rpm(
    name = "python3-cffi-0__1.14.0-1.fc32.aarch64",
    sha256 = "844ee747d24d934104398be60747b407d19c8106ead11b06fe92fcc62bd765fc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-cffi-1.14.0-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-cffi-1.14.0-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-cffi-1.14.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-cffi-1.14.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/844ee747d24d934104398be60747b407d19c8106ead11b06fe92fcc62bd765fc",
    ],
)

rpm(
    name = "python3-cffi-0__1.14.0-1.fc32.x86_64",
    sha256 = "7124f9fedc862e3bab80f05b804b6c9580603ce3155727e888646d4d4f5ddc50",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cffi-1.14.0-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cffi-1.14.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cffi-1.14.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cffi-1.14.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7124f9fedc862e3bab80f05b804b6c9580603ce3155727e888646d4d4f5ddc50",
    ],
)

rpm(
    name = "python3-cryptography-0__2.8-3.fc32.aarch64",
    sha256 = "bbf9571bf10df55a90e73b372da33b6ac54fad5778cea58064b7b57dcbb17180",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-cryptography-2.8-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-cryptography-2.8-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-cryptography-2.8-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-cryptography-2.8-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bbf9571bf10df55a90e73b372da33b6ac54fad5778cea58064b7b57dcbb17180",
    ],
)

rpm(
    name = "python3-cryptography-0__2.8-3.fc32.x86_64",
    sha256 = "bb8942d19e594c0f4ca181bd58796bd5d3cb681c3f17cd2ec2654c3afe28e39a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cryptography-2.8-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cryptography-2.8-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cryptography-2.8-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-cryptography-2.8-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bb8942d19e594c0f4ca181bd58796bd5d3cb681c3f17cd2ec2654c3afe28e39a",
    ],
)

rpm(
    name = "python3-idna-0__2.8-6.fc32.aarch64",
    sha256 = "61c51596cc97f35177efe8dc5e2ca52d8fd528570f33c184497f419259b73c90",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61c51596cc97f35177efe8dc5e2ca52d8fd528570f33c184497f419259b73c90",
    ],
)

rpm(
    name = "python3-idna-0__2.8-6.fc32.x86_64",
    sha256 = "61c51596cc97f35177efe8dc5e2ca52d8fd528570f33c184497f419259b73c90",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-idna-2.8-6.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/61c51596cc97f35177efe8dc5e2ca52d8fd528570f33c184497f419259b73c90",
    ],
)

rpm(
    name = "python3-libs-0__3.8.10-1.fc32.aarch64",
    sha256 = "23ea540161b08791a66cb9e9975e79d8c03e2cd0e6258a66b4c6e0e0d00b4959",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libs-3.8.10-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libs-3.8.10-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libs-3.8.10-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libs-3.8.10-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/23ea540161b08791a66cb9e9975e79d8c03e2cd0e6258a66b4c6e0e0d00b4959",
    ],
)

rpm(
    name = "python3-libs-0__3.8.10-1.fc32.x86_64",
    sha256 = "79b78215cf83e351516dea7e580ecf2d3350a477dda5fbd4e9188873d0dd19e9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.10-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.10-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.10-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libs-3.8.10-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/79b78215cf83e351516dea7e580ecf2d3350a477dda5fbd4e9188873d0dd19e9",
    ],
)

rpm(
    name = "python3-libselinux-0__3.0-5.fc32.aarch64",
    sha256 = "99cf82c23c9aa1303dde95472a020c763a8ff0af58c78b57ced7c3a2b286da08",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libselinux-3.0-5.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libselinux-3.0-5.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libselinux-3.0-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/python3-libselinux-3.0-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/99cf82c23c9aa1303dde95472a020c763a8ff0af58c78b57ced7c3a2b286da08",
    ],
)

rpm(
    name = "python3-libselinux-0__3.0-5.fc32.x86_64",
    sha256 = "a5f9e91fbcf28dc4bfebcf8894b63758134044a6909b3b6061fd7c9f1b72cf39",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libselinux-3.0-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libselinux-3.0-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libselinux-3.0-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/python3-libselinux-3.0-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a5f9e91fbcf28dc4bfebcf8894b63758134044a6909b3b6061fd7c9f1b72cf39",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.0-3.fc32.aarch64",
    sha256 = "eded265cff5d22b89a955570eba030643d6730dd5987c2efed3110ef74cd0254",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-libsemanage-3.0-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/eded265cff5d22b89a955570eba030643d6730dd5987c2efed3110ef74cd0254",
    ],
)

rpm(
    name = "python3-libsemanage-0__3.0-3.fc32.x86_64",
    sha256 = "55bafcdf9c31b1456af3bf584bfe7ac745a03f4decd17197ea97b498d68b3b82",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-libsemanage-3.0-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/55bafcdf9c31b1456af3bf584bfe7ac745a03f4decd17197ea97b498d68b3b82",
    ],
)

rpm(
    name = "python3-ply-0__3.11-7.fc32.aarch64",
    sha256 = "f6203a41ed91197bb770a38a101d977f0f56de86ccc5a71cee9c0e198f26bcbc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f6203a41ed91197bb770a38a101d977f0f56de86ccc5a71cee9c0e198f26bcbc",
    ],
)

rpm(
    name = "python3-ply-0__3.11-7.fc32.x86_64",
    sha256 = "f6203a41ed91197bb770a38a101d977f0f56de86ccc5a71cee9c0e198f26bcbc",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-ply-3.11-7.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f6203a41ed91197bb770a38a101d977f0f56de86ccc5a71cee9c0e198f26bcbc",
    ],
)

rpm(
    name = "python3-policycoreutils-0__3.0-2.fc32.aarch64",
    sha256 = "15f2fc89b7bd39dcd3f6f8db30f56b76b65df311d7ad9852d498fbbc5c7d2aa2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/15f2fc89b7bd39dcd3f6f8db30f56b76b65df311d7ad9852d498fbbc5c7d2aa2",
    ],
)

rpm(
    name = "python3-policycoreutils-0__3.0-2.fc32.x86_64",
    sha256 = "15f2fc89b7bd39dcd3f6f8db30f56b76b65df311d7ad9852d498fbbc5c7d2aa2",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-policycoreutils-3.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/15f2fc89b7bd39dcd3f6f8db30f56b76b65df311d7ad9852d498fbbc5c7d2aa2",
    ],
)

rpm(
    name = "python3-pycparser-0__2.19-2.fc32.aarch64",
    sha256 = "a0b87b2dc3c5f536e94d6a4f3563a621dfbc067a62c3d1fe69bdb70c3cecec57",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a0b87b2dc3c5f536e94d6a4f3563a621dfbc067a62c3d1fe69bdb70c3cecec57",
    ],
)

rpm(
    name = "python3-pycparser-0__2.19-2.fc32.x86_64",
    sha256 = "a0b87b2dc3c5f536e94d6a4f3563a621dfbc067a62c3d1fe69bdb70c3cecec57",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-pycparser-2.19-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a0b87b2dc3c5f536e94d6a4f3563a621dfbc067a62c3d1fe69bdb70c3cecec57",
    ],
)

rpm(
    name = "python3-setools-0__4.3.0-1.fc32.aarch64",
    sha256 = "82d2eaad75cf45da9773298344dcbbaebb4da5b67526a6c43bc67d3f84d98616",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-setools-4.3.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/82d2eaad75cf45da9773298344dcbbaebb4da5b67526a6c43bc67d3f84d98616",
    ],
)

rpm(
    name = "python3-setools-0__4.3.0-1.fc32.x86_64",
    sha256 = "6f5f53b66f7c3bf6958f6f163788583265ff0360188620c3b0f7ddedeac3d1f4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setools-4.3.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6f5f53b66f7c3bf6958f6f163788583265ff0360188620c3b0f7ddedeac3d1f4",
    ],
)

rpm(
    name = "python3-setuptools-0__41.6.0-2.fc32.aarch64",
    sha256 = "724cca9919bb7b0183b030aca216d4d51de70bf35c2cc5e8325a21a52ca15ceb",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/724cca9919bb7b0183b030aca216d4d51de70bf35c2cc5e8325a21a52ca15ceb",
    ],
)

rpm(
    name = "python3-setuptools-0__41.6.0-2.fc32.x86_64",
    sha256 = "724cca9919bb7b0183b030aca216d4d51de70bf35c2cc5e8325a21a52ca15ceb",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-setuptools-41.6.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/724cca9919bb7b0183b030aca216d4d51de70bf35c2cc5e8325a21a52ca15ceb",
    ],
)

rpm(
    name = "python3-six-0__1.14.0-2.fc32.aarch64",
    sha256 = "02654432f3853c9ae39c7601b5b0606c9d5eb5eef1d95e3e6f0074501842941f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/02654432f3853c9ae39c7601b5b0606c9d5eb5eef1d95e3e6f0074501842941f",
    ],
)

rpm(
    name = "python3-six-0__1.14.0-2.fc32.x86_64",
    sha256 = "02654432f3853c9ae39c7601b5b0606c9d5eb5eef1d95e3e6f0074501842941f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/python3-six-1.14.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/02654432f3853c9ae39c7601b5b0606c9d5eb5eef1d95e3e6f0074501842941f",
    ],
)

rpm(
    name = "qemu-img-15__5.2.0-15.fc32.aarch64",
    sha256 = "75adc58ea9e74b82390fcfaeb6bf400575037336d1e4ebdcc7a3368bee7538b4",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-aarch64/02142033-qemu-kvm/qemu-img-5.2.0-15.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/75adc58ea9e74b82390fcfaeb6bf400575037336d1e4ebdcc7a3368bee7538b4",
    ],
)

rpm(
    name = "qemu-img-15__5.2.0-15.fc32.x86_64",
    sha256 = "4fec360d3d6ed31de0beaea40d4cdaa266ade9985c3d4fc8f2e0d13ee76a1e03",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-x86_64/02142033-qemu-kvm/qemu-img-5.2.0-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4fec360d3d6ed31de0beaea40d4cdaa266ade9985c3d4fc8f2e0d13ee76a1e03",
    ],
)

rpm(
    name = "qemu-kvm-15__5.2.0-15.fc32.x86_64",
    sha256 = "d13a7b9808c7ae59722bab3ff9816c5f41dd8ae56577dd49bdf39322ca052122",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-x86_64/02142033-qemu-kvm/qemu-kvm-5.2.0-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d13a7b9808c7ae59722bab3ff9816c5f41dd8ae56577dd49bdf39322ca052122",
    ],
)

rpm(
    name = "qemu-kvm-common-15__5.2.0-15.fc32.aarch64",
    sha256 = "c1fbbdcd522caa14f739ded634e605d6e046cd56d58e0daa8f662f9fed0ff50a",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-aarch64/02142033-qemu-kvm/qemu-kvm-common-5.2.0-15.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c1fbbdcd522caa14f739ded634e605d6e046cd56d58e0daa8f662f9fed0ff50a",
    ],
)

rpm(
    name = "qemu-kvm-common-15__5.2.0-15.fc32.x86_64",
    sha256 = "702f72282b5c87029ea4ff4a2b6ce7a5923f19892c3be15d3c2f84f4f5db1f42",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-x86_64/02142033-qemu-kvm/qemu-kvm-common-5.2.0-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/702f72282b5c87029ea4ff4a2b6ce7a5923f19892c3be15d3c2f84f4f5db1f42",
    ],
)

rpm(
    name = "qemu-kvm-core-15__5.2.0-15.fc32.aarch64",
    sha256 = "56e3f24555a17e24736049fe833f1dd3b6dc64b67bd6d0385a919964d9925f1b",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-aarch64/02142033-qemu-kvm/qemu-kvm-core-5.2.0-15.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/56e3f24555a17e24736049fe833f1dd3b6dc64b67bd6d0385a919964d9925f1b",
    ],
)

rpm(
    name = "qemu-kvm-core-15__5.2.0-15.fc32.x86_64",
    sha256 = "a926bb3f328a4ff72f0d8367328c785fc3a1827001f668e8599bdea6f4df1bb8",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-x86_64/02142033-qemu-kvm/qemu-kvm-core-5.2.0-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a926bb3f328a4ff72f0d8367328c785fc3a1827001f668e8599bdea6f4df1bb8",
    ],
)

rpm(
    name = "qemu-kvm-ui-opengl-15__5.2.0-15.fc32.x86_64",
    sha256 = "d4d8f22373bb83575f484cb2e1e28fecf2c829bef00bae2f246f68d566d75036",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-x86_64/02142033-qemu-kvm/qemu-kvm-ui-opengl-5.2.0-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d4d8f22373bb83575f484cb2e1e28fecf2c829bef00bae2f246f68d566d75036",
    ],
)

rpm(
    name = "qemu-kvm-ui-spice-15__5.2.0-15.fc32.x86_64",
    sha256 = "ff9206be699c8d4ce699588178809bf09ffe2d8642c84feb60b91eb0c7f462b2",
    urls = [
        "https://download.copr.fedorainfracloud.org/results/@kubevirt/qemu-5.2.0-15.el8/fedora-32-x86_64/02142033-qemu-kvm/qemu-kvm-ui-spice-5.2.0-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ff9206be699c8d4ce699588178809bf09ffe2d8642c84feb60b91eb0c7f462b2",
    ],
)

rpm(
    name = "qrencode-libs-0__4.0.2-5.fc32.aarch64",
    sha256 = "3d6ec574fe2c612bcc45395f7ee87c68f45016f005c6d7aeee6b37897f41b8d2",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/3d6ec574fe2c612bcc45395f7ee87c68f45016f005c6d7aeee6b37897f41b8d2",
    ],
)

rpm(
    name = "qrencode-libs-0__4.0.2-5.fc32.x86_64",
    sha256 = "f1150f9e17beaef09aca0f291e10db8c3ee5566fbf4c929b7672334410fa74e9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/q/qrencode-libs-4.0.2-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1150f9e17beaef09aca0f291e10db8c3ee5566fbf4c929b7672334410fa74e9",
    ],
)

rpm(
    name = "radvd-0__2.18-3.fc32.x86_64",
    sha256 = "5b483d192bc9c0d743d9d70acb417822f47e12d8ee732b510bef77f05dab4a8e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/radvd-2.18-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/radvd-2.18-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/radvd-2.18-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/radvd-2.18-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5b483d192bc9c0d743d9d70acb417822f47e12d8ee732b510bef77f05dab4a8e",
    ],
)

rpm(
    name = "rdma-core-0__33.0-2.fc32.aarch64",
    sha256 = "c2fb258612579316609cc97e941e2f7d9ac9f0c2063d274b70da51dd6ffe0123",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rdma-core-33.0-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rdma-core-33.0-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/r/rdma-core-33.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/r/rdma-core-33.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c2fb258612579316609cc97e941e2f7d9ac9f0c2063d274b70da51dd6ffe0123",
    ],
)

rpm(
    name = "rdma-core-0__33.0-2.fc32.x86_64",
    sha256 = "1dabfdd76b58fa155a4a697db55f398564fb0699a4cb0c17c2f7fb3b1db2fab9",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-33.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-33.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-33.0-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rdma-core-33.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1dabfdd76b58fa155a4a697db55f398564fb0699a4cb0c17c2f7fb3b1db2fab9",
    ],
)

rpm(
    name = "readline-0__8.0-4.fc32.aarch64",
    sha256 = "6007c88c459315a5e2ce354086bd0372a56e15cdd0dc14e6e889ab859f8d8365",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/r/readline-8.0-4.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/r/readline-8.0-4.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/r/readline-8.0-4.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/r/readline-8.0-4.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/6007c88c459315a5e2ce354086bd0372a56e15cdd0dc14e6e889ab859f8d8365",
    ],
)

rpm(
    name = "readline-0__8.0-4.fc32.x86_64",
    sha256 = "f1c79039f4c6ba0fad88590c2cb55a96489449c334a671cc18c0bf424a4548b8",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/r/readline-8.0-4.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f1c79039f4c6ba0fad88590c2cb55a96489449c334a671cc18c0bf424a4548b8",
    ],
)

rpm(
    name = "rpm-0__4.15.1.1-1.fc32.1.aarch64",
    sha256 = "714ecf43f412ec53c14672e7ea94ee52538e60ef6b5143ac1f28ae26293c329d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/714ecf43f412ec53c14672e7ea94ee52538e60ef6b5143ac1f28ae26293c329d",
    ],
)

rpm(
    name = "rpm-0__4.15.1.1-1.fc32.1.x86_64",
    sha256 = "0b7e2adc1d3a1aed75b5adad299c83ebe863fc11a6976fd5b14368e71c568203",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0b7e2adc1d3a1aed75b5adad299c83ebe863fc11a6976fd5b14368e71c568203",
    ],
)

rpm(
    name = "rpm-libs-0__4.15.1.1-1.fc32.1.aarch64",
    sha256 = "ab5212815679d5d7220e548b22658c048ece465e01e9781a998b3b605b36eb9e",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ab5212815679d5d7220e548b22658c048ece465e01e9781a998b3b605b36eb9e",
    ],
)

rpm(
    name = "rpm-libs-0__4.15.1.1-1.fc32.1.x86_64",
    sha256 = "f91eb167644766c3d51373f50e0869270f0994e5b32f5583f8ca43d217febd15",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-libs-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f91eb167644766c3d51373f50e0869270f0994e5b32f5583f8ca43d217febd15",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.15.1.1-1.fc32.1.aarch64",
    sha256 = "a024643c3b647e6b99e42861dd9d0d87c7182799bd02a9d2d4252dbc1df188d9",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a024643c3b647e6b99e42861dd9d0d87c7182799bd02a9d2d4252dbc1df188d9",
    ],
)

rpm(
    name = "rpm-plugin-selinux-0__4.15.1.1-1.fc32.1.x86_64",
    sha256 = "3ef5c9c7c0f123a306fbb6c8637435833c8a4dbafaad98466b66503177b146e3",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/r/rpm-plugin-selinux-4.15.1.1-1.fc32.1.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3ef5c9c7c0f123a306fbb6c8637435833c8a4dbafaad98466b66503177b146e3",
    ],
)

rpm(
    name = "scrub-0__2.6.1-2.fc32.x86_64",
    sha256 = "224f20d78534d31e8f93aa5898ac5ee26ea59bc29333bac39351d83e21136ade",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/scrub-2.6.1-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/224f20d78534d31e8f93aa5898ac5ee26ea59bc29333bac39351d83e21136ade",
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
    name = "sed-0__4.5-5.fc32.aarch64",
    sha256 = "ccf07a3682a1038a6224b3da69e20f201584ed1c879539cedb57e184aa14429a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sed-4.5-5.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/s/sed-4.5-5.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sed-4.5-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sed-4.5-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ccf07a3682a1038a6224b3da69e20f201584ed1c879539cedb57e184aa14429a",
    ],
)

rpm(
    name = "sed-0__4.5-5.fc32.x86_64",
    sha256 = "ffe5076b9018efdb1612c487f637af39ab6c3c79ec37311978935cfa357ecd61",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sed-4.5-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ffe5076b9018efdb1612c487f637af39ab6c3c79ec37311978935cfa357ecd61",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.5-46.fc32.aarch64",
    sha256 = "3bb33f22612d187d5acab1efb108b08270f8e568d0ebb7a3591598829eced000",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3bb33f22612d187d5acab1efb108b08270f8e568d0ebb7a3591598829eced000",
    ],
)

rpm(
    name = "selinux-policy-0__3.14.5-46.fc32.x86_64",
    sha256 = "3bb33f22612d187d5acab1efb108b08270f8e568d0ebb7a3591598829eced000",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-3.14.5-46.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/3bb33f22612d187d5acab1efb108b08270f8e568d0ebb7a3591598829eced000",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.5-46.fc32.aarch64",
    sha256 = "1fa0c0bd36639868ea3ef05a206f1fa7bfe0bd59d9c29e2fcdd59bc2ec8ce18c",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1fa0c0bd36639868ea3ef05a206f1fa7bfe0bd59d9c29e2fcdd59bc2ec8ce18c",
    ],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.5-46.fc32.x86_64",
    sha256 = "1fa0c0bd36639868ea3ef05a206f1fa7bfe0bd59d9c29e2fcdd59bc2ec8ce18c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/selinux-policy-targeted-3.14.5-46.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/1fa0c0bd36639868ea3ef05a206f1fa7bfe0bd59d9c29e2fcdd59bc2ec8ce18c",
    ],
)

rpm(
    name = "setup-0__2.13.6-2.fc32.aarch64",
    sha256 = "a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    ],
)

rpm(
    name = "setup-0__2.13.6-2.fc32.x86_64",
    sha256 = "a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/setup-2.13.6-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/a336d2e77255df4783f52762e44efcc8d77b044a3e39c7f577d5535212848280",
    ],
)

rpm(
    name = "sgabios-bin-1__0.20180715git-4.fc32.x86_64",
    sha256 = "e89eb0796febd3e1108bc3551180eb0a2dffcbf34aa150be6abb40c6f7cb140c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sgabios-bin-0.20180715git-4.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/e89eb0796febd3e1108bc3551180eb0a2dffcbf34aa150be6abb40c6f7cb140c",
    ],
)

rpm(
    name = "shadow-utils-2__4.8.1-3.fc32.aarch64",
    sha256 = "4946334a5901346fc9636d10be5da98668ffe369bc7b2df36051c19560f3f906",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/shadow-utils-4.8.1-3.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/shadow-utils-4.8.1-3.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/shadow-utils-4.8.1-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/shadow-utils-4.8.1-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4946334a5901346fc9636d10be5da98668ffe369bc7b2df36051c19560f3f906",
    ],
)

rpm(
    name = "shadow-utils-2__4.8.1-3.fc32.x86_64",
    sha256 = "696768dc6f369a52d2c431eb7c76461237c2804d591cee418c04f97f3660b667",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/shadow-utils-4.8.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/696768dc6f369a52d2c431eb7c76461237c2804d591cee418c04f97f3660b667",
    ],
)

rpm(
    name = "snappy-0__1.1.8-2.fc32.aarch64",
    sha256 = "bd7767b9634cdaac6eb2a5329a3170c35077b4d7ea1f919687e218b51df36517",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/snappy-1.1.8-2.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/s/snappy-1.1.8-2.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/snappy-1.1.8-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/snappy-1.1.8-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/bd7767b9634cdaac6eb2a5329a3170c35077b4d7ea1f919687e218b51df36517",
    ],
)

rpm(
    name = "snappy-0__1.1.8-2.fc32.x86_64",
    sha256 = "186a33671176e2cd2a6d036bc6cc45fa6e331a28f022c495019c3f26ef2ee383",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/snappy-1.1.8-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/186a33671176e2cd2a6d036bc6cc45fa6e331a28f022c495019c3f26ef2ee383",
    ],
)

rpm(
    name = "spice-server-0__0.14.3-1.fc32.x86_64",
    sha256 = "4f8be162f46533c87bd62a6293d49f80ae273a257029653413c4c0c12fe656fe",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/spice-server-0.14.3-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4f8be162f46533c87bd62a6293d49f80ae273a257029653413c4c0c12fe656fe",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.0-1.fc32.aarch64",
    sha256 = "2180540def7d02409ea703248dba91cc469b320511d31fb26051e9a8544e7bb8",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/sqlite-libs-3.34.0-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/sqlite-libs-3.34.0-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/sqlite-libs-3.34.0-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/sqlite-libs-3.34.0-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/2180540def7d02409ea703248dba91cc469b320511d31fb26051e9a8544e7bb8",
    ],
)

rpm(
    name = "sqlite-libs-0__3.34.0-1.fc32.x86_64",
    sha256 = "19c1021f1729ea60b0b8f6056953150847196732e84745a118d442c2dea19cfb",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/sqlite-libs-3.34.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/19c1021f1729ea60b0b8f6056953150847196732e84745a118d442c2dea19cfb",
    ],
)

rpm(
    name = "squashfs-tools-0__4.3-25.fc32.x86_64",
    sha256 = "219049fc4bb4b5aca1d1562e285a023eb16662b868eaaa96a39aff01987b88f4",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/squashfs-tools-4.3-25.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/squashfs-tools-4.3-25.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/squashfs-tools-4.3-25.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/squashfs-tools-4.3-25.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/219049fc4bb4b5aca1d1562e285a023eb16662b868eaaa96a39aff01987b88f4",
    ],
)

rpm(
    name = "supermin-0__5.2.1-1.fc32.x86_64",
    sha256 = "7f3bcd18c13e0f00f1060d79d56355bde9b7e2c433ca37f35780f5c56c799761",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/supermin-5.2.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/7f3bcd18c13e0f00f1060d79d56355bde9b7e2c433ca37f35780f5c56c799761",
    ],
)

rpm(
    name = "swtpm-0__0.5.2-0.20201226gite59c0c1.fc32.aarch64",
    sha256 = "fdb5d54e2337a3c4227fb8a72c55dc9ee8c9e850e81f11c31c3b2f86cc2f1c3f",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fdb5d54e2337a3c4227fb8a72c55dc9ee8c9e850e81f11c31c3b2f86cc2f1c3f",
    ],
)

rpm(
    name = "swtpm-0__0.5.2-0.20201226gite59c0c1.fc32.x86_64",
    sha256 = "56ec2b4489f6e8108f6613267058a9350d577c1c5359d9437e593a79194e2197",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56ec2b4489f6e8108f6613267058a9350d577c1c5359d9437e593a79194e2197",
    ],
)

rpm(
    name = "swtpm-libs-0__0.5.2-0.20201226gite59c0c1.fc32.aarch64",
    sha256 = "e2a9d7b9a828ea729acc00f75e648259253a2c8c279f4130a67f664e10680f42",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e2a9d7b9a828ea729acc00f75e648259253a2c8c279f4130a67f664e10680f42",
    ],
)

rpm(
    name = "swtpm-libs-0__0.5.2-0.20201226gite59c0c1.fc32.x86_64",
    sha256 = "56c95dad2db8fa41691863e8425cb2c66a38061dd47fa98eca0d3f5377578c59",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-libs-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/56c95dad2db8fa41691863e8425cb2c66a38061dd47fa98eca0d3f5377578c59",
    ],
)

rpm(
    name = "swtpm-tools-0__0.5.2-0.20201226gite59c0c1.fc32.aarch64",
    sha256 = "c6225b9b4e00a2d957f61a8539f30269625bed51794c50fdb053471549d39ef0",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c6225b9b4e00a2d957f61a8539f30269625bed51794c50fdb053471549d39ef0",
    ],
)

rpm(
    name = "swtpm-tools-0__0.5.2-0.20201226gite59c0c1.fc32.x86_64",
    sha256 = "5ee19ec49088e2e26918281ecc1abf89142e4072d7c24d5ebc670312d09e5d12",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/swtpm-tools-0.5.2-0.20201226gite59c0c1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5ee19ec49088e2e26918281ecc1abf89142e4072d7c24d5ebc670312d09e5d12",
    ],
)

rpm(
    name = "syslinux-0__6.04-0.13.fc32.x86_64",
    sha256 = "dbcf6ba5f6ab0cdc884572f07427114b8492d453420a80ffd521578a752b8064",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-6.04-0.13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-6.04-0.13.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-6.04-0.13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-6.04-0.13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/dbcf6ba5f6ab0cdc884572f07427114b8492d453420a80ffd521578a752b8064",
    ],
)

rpm(
    name = "syslinux-extlinux-0__6.04-0.13.fc32.x86_64",
    sha256 = "78de81d45789ae9bccf6dd82f2b2835ac868f1a584e05d0e9f89f40bb4a202c7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.13.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.13.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.13.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-6.04-0.13.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/78de81d45789ae9bccf6dd82f2b2835ac868f1a584e05d0e9f89f40bb4a202c7",
    ],
)

rpm(
    name = "syslinux-extlinux-nonlinux-0__6.04-0.13.fc32.x86_64",
    sha256 = "4e066ace8364aba9311fa74f4a87c21f98091bc1af5a6a4daa06441469234041",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-extlinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4e066ace8364aba9311fa74f4a87c21f98091bc1af5a6a4daa06441469234041",
    ],
)

rpm(
    name = "syslinux-nonlinux-0__6.04-0.13.fc32.x86_64",
    sha256 = "dc2558fdefc439075ebdf57bd9f2f691d6a6f278879713f2e093398dcb62c25a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/syslinux-nonlinux-6.04-0.13.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/dc2558fdefc439075ebdf57bd9f2f691d6a6f278879713f2e093398dcb62c25a",
    ],
)

rpm(
    name = "systemd-0__245.9-1.fc32.aarch64",
    sha256 = "46dbd6dd0bb0c9fc6d5136c98616981d6ecbd1fa47c3e7a0fb38b4cb319429e5",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-245.9-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-245.9-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-245.9-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-245.9-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/46dbd6dd0bb0c9fc6d5136c98616981d6ecbd1fa47c3e7a0fb38b4cb319429e5",
    ],
)

rpm(
    name = "systemd-0__245.9-1.fc32.x86_64",
    sha256 = "bffd499d9f853bf78721df922899f9cb631e938184ae6fac3bed9cc6ca048170",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-245.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/bffd499d9f853bf78721df922899f9cb631e938184ae6fac3bed9cc6ca048170",
    ],
)

rpm(
    name = "systemd-container-0__245.9-1.fc32.aarch64",
    sha256 = "51b6dd2bf85a8b27f16320af42fd1cceb6882239b91699d4d6feca28feeda90a",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-container-245.9-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-container-245.9-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-container-245.9-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-container-245.9-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/51b6dd2bf85a8b27f16320af42fd1cceb6882239b91699d4d6feca28feeda90a",
    ],
)

rpm(
    name = "systemd-container-0__245.9-1.fc32.x86_64",
    sha256 = "b995227f1bb1995ecd89831d537a5a06b1b9f561a5377f9a86ccee3e4e841be8",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-container-245.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b995227f1bb1995ecd89831d537a5a06b1b9f561a5377f9a86ccee3e4e841be8",
    ],
)

rpm(
    name = "systemd-libs-0__245.9-1.fc32.aarch64",
    sha256 = "d11c24204ef82ad5f13727835fb474156922eabe108e2657bf217a3f12c03ce6",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-libs-245.9-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-libs-245.9-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-libs-245.9-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-libs-245.9-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d11c24204ef82ad5f13727835fb474156922eabe108e2657bf217a3f12c03ce6",
    ],
)

rpm(
    name = "systemd-libs-0__245.9-1.fc32.x86_64",
    sha256 = "661c7bac2d828a41166d7675f62c58ff647037cf406210079da88847bcf13bd8",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-libs-245.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/661c7bac2d828a41166d7675f62c58ff647037cf406210079da88847bcf13bd8",
    ],
)

rpm(
    name = "systemd-pam-0__245.9-1.fc32.aarch64",
    sha256 = "1bcdb49ee70b3da2ad869ad0dda3b497e590c831b35418605d7d98d561a2c0f5",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-pam-245.9-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-pam-245.9-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-pam-245.9-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-pam-245.9-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/1bcdb49ee70b3da2ad869ad0dda3b497e590c831b35418605d7d98d561a2c0f5",
    ],
)

rpm(
    name = "systemd-pam-0__245.9-1.fc32.x86_64",
    sha256 = "6b2554f0c7ae3a19ff574e3085d473bcfed6fc6b83124bc4815e2d452403b472",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-pam-245.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6b2554f0c7ae3a19ff574e3085d473bcfed6fc6b83124bc4815e2d452403b472",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__245.9-1.fc32.aarch64",
    sha256 = "0e8bb875661f39c0a40a13419619ef3127682c9009b9f56cbb9ef833fac4cce6",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0e8bb875661f39c0a40a13419619ef3127682c9009b9f56cbb9ef833fac4cce6",
    ],
)

rpm(
    name = "systemd-rpm-macros-0__245.9-1.fc32.x86_64",
    sha256 = "0e8bb875661f39c0a40a13419619ef3127682c9009b9f56cbb9ef833fac4cce6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-rpm-macros-245.9-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0e8bb875661f39c0a40a13419619ef3127682c9009b9f56cbb9ef833fac4cce6",
    ],
)

rpm(
    name = "systemd-udev-0__245.9-1.fc32.x86_64",
    sha256 = "d7ff5450bd6197735b08ece1c97fe83aa36990dab0b178ddebf624404179cfa2",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-udev-245.9-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-udev-245.9-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-udev-245.9-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/s/systemd-udev-245.9-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d7ff5450bd6197735b08ece1c97fe83aa36990dab0b178ddebf624404179cfa2",
    ],
)

rpm(
    name = "tar-2__1.32-5.fc32.aarch64",
    sha256 = "a329d86c1035f428bc8b0e098664149e5f56c997d5a1e860ea26a0982e6cca8d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/tar-1.32-5.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/tar-1.32-5.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/t/tar-1.32-5.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/t/tar-1.32-5.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a329d86c1035f428bc8b0e098664149e5f56c997d5a1e860ea26a0982e6cca8d",
    ],
)

rpm(
    name = "tar-2__1.32-5.fc32.x86_64",
    sha256 = "6c0b8e09684f2b9526055205781048630a803be452f54b0bde72431554b4590f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tar-1.32-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tar-1.32-5.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/t/tar-1.32-5.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tar-1.32-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/6c0b8e09684f2b9526055205781048630a803be452f54b0bde72431554b4590f",
    ],
)

rpm(
    name = "trousers-0__0.3.13-15.fc32.aarch64",
    sha256 = "265ed9294e1c2ddb819e906dbb4b97b92fda5c0a032eb59de95e7e548fc6cdb7",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-0.3.13-15.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-0.3.13-15.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-0.3.13-15.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-0.3.13-15.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/265ed9294e1c2ddb819e906dbb4b97b92fda5c0a032eb59de95e7e548fc6cdb7",
    ],
)

rpm(
    name = "trousers-0__0.3.13-15.fc32.x86_64",
    sha256 = "4c1f241c759906e057d16be07747e14a658e5946ec519fb83959e411d4dd66a1",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-0.3.13-15.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-0.3.13-15.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-0.3.13-15.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-0.3.13-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4c1f241c759906e057d16be07747e14a658e5946ec519fb83959e411d4dd66a1",
    ],
)

rpm(
    name = "trousers-lib-0__0.3.13-15.fc32.aarch64",
    sha256 = "72be29109ba9d467bca37c32a800232c26d549083c8141b98c0b993c06cab2d6",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-lib-0.3.13-15.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-lib-0.3.13-15.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-lib-0.3.13-15.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/t/trousers-lib-0.3.13-15.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/72be29109ba9d467bca37c32a800232c26d549083c8141b98c0b993c06cab2d6",
    ],
)

rpm(
    name = "trousers-lib-0__0.3.13-15.fc32.x86_64",
    sha256 = "f509ff97b7769fcb8cef3210a77ee847f8855b55d5e2039ed9bbf8ccd900e8cc",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-lib-0.3.13-15.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-lib-0.3.13-15.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-lib-0.3.13-15.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/trousers-lib-0.3.13-15.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f509ff97b7769fcb8cef3210a77ee847f8855b55d5e2039ed9bbf8ccd900e8cc",
    ],
)

rpm(
    name = "tzdata-0__2021a-1.fc32.aarch64",
    sha256 = "f8dbb263b4b844d3d0ef4b93d7502a78384759d07987e6ab678cc565122595b8",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f8dbb263b4b844d3d0ef4b93d7502a78384759d07987e6ab678cc565122595b8",
    ],
)

rpm(
    name = "tzdata-0__2021a-1.fc32.x86_64",
    sha256 = "f8dbb263b4b844d3d0ef4b93d7502a78384759d07987e6ab678cc565122595b8",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/t/tzdata-2021a-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f8dbb263b4b844d3d0ef4b93d7502a78384759d07987e6ab678cc565122595b8",
    ],
)

rpm(
    name = "unbound-libs-0__1.10.1-1.fc32.aarch64",
    sha256 = "a7f2ddf14a7f796b1080962ca6ca0b5c3a653889f08d0501822a181154acb1db",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/u/unbound-libs-1.10.1-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/u/unbound-libs-1.10.1-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/u/unbound-libs-1.10.1-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/u/unbound-libs-1.10.1-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a7f2ddf14a7f796b1080962ca6ca0b5c3a653889f08d0501822a181154acb1db",
    ],
)

rpm(
    name = "unbound-libs-0__1.10.1-1.fc32.x86_64",
    sha256 = "463d8259936601d23a02c407a4fb294c0e3bc9a213282c4525137bc1b18bbf6b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/unbound-libs-1.10.1-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/463d8259936601d23a02c407a4fb294c0e3bc9a213282c4525137bc1b18bbf6b",
    ],
)

rpm(
    name = "usbredir-0__0.9.0-1.fc32.x86_64",
    sha256 = "565bf717a81dacd313cba21e25b4c8be55a9bf964258bc2f0d1b160a41096eb5",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/usbredir-0.9.0-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/usbredir-0.9.0-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/u/usbredir-0.9.0-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/usbredir-0.9.0-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/565bf717a81dacd313cba21e25b4c8be55a9bf964258bc2f0d1b160a41096eb5",
    ],
)

rpm(
    name = "userspace-rcu-0__0.11.1-3.fc32.aarch64",
    sha256 = "5b177815d30cfa5ca66e2ac0b33fe55d817aaed40256c8fd0eb9d537638e7f20",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/5b177815d30cfa5ca66e2ac0b33fe55d817aaed40256c8fd0eb9d537638e7f20",
    ],
)

rpm(
    name = "userspace-rcu-0__0.11.1-3.fc32.x86_64",
    sha256 = "1f5b2264d9f9a3f423124dbac73ef04fc9ad0df44afd80e8ba79d21a8942ca79",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/u/userspace-rcu-0.11.1-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1f5b2264d9f9a3f423124dbac73ef04fc9ad0df44afd80e8ba79d21a8942ca79",
    ],
)

rpm(
    name = "util-linux-0__2.35.2-1.fc32.aarch64",
    sha256 = "9e62445f73c927ee5efa10d79a1640eefe79159c6352b4c4b9621b232b8ad1c7",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/u/util-linux-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/u/util-linux-2.35.2-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/u/util-linux-2.35.2-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/u/util-linux-2.35.2-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9e62445f73c927ee5efa10d79a1640eefe79159c6352b4c4b9621b232b8ad1c7",
    ],
)

rpm(
    name = "util-linux-0__2.35.2-1.fc32.x86_64",
    sha256 = "4d80736f9a52519104eeb228eb1ea95d0d6e9addc766eebacc9a5137fb2a5977",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/u/util-linux-2.35.2-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4d80736f9a52519104eeb228eb1ea95d0d6e9addc766eebacc9a5137fb2a5977",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2787-1.fc32.aarch64",
    sha256 = "ace20b1fe8ce7d42fb02e5b35d0e1a079af0813b7d196e2174ff479467a500ba",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/v/vim-minimal-8.2.2787-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/v/vim-minimal-8.2.2787-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/v/vim-minimal-8.2.2787-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/v/vim-minimal-8.2.2787-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ace20b1fe8ce7d42fb02e5b35d0e1a079af0813b7d196e2174ff479467a500ba",
    ],
)

rpm(
    name = "vim-minimal-2__8.2.2787-1.fc32.x86_64",
    sha256 = "f7353010d28685511d2deee635051fc145bd911ff39c8f0d5676acaa396c9f7f",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2787-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2787-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2787-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/v/vim-minimal-8.2.2787-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/f7353010d28685511d2deee635051fc145bd911ff39c8f0d5676acaa396c9f7f",
    ],
)

rpm(
    name = "which-0__2.21-19.fc32.aarch64",
    sha256 = "d552c735d48fa647509605f524863eab28b69b9fc8d7c62a67479c3af0878024",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/w/which-2.21-19.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/w/which-2.21-19.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/w/which-2.21-19.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/w/which-2.21-19.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d552c735d48fa647509605f524863eab28b69b9fc8d7c62a67479c3af0878024",
    ],
)

rpm(
    name = "which-0__2.21-19.fc32.x86_64",
    sha256 = "82e0d8f1e0dccc6d18acd04b7806350343140d9c91da7a216f93167dcf650a61",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/w/which-2.21-19.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/82e0d8f1e0dccc6d18acd04b7806350343140d9c91da7a216f93167dcf650a61",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.29-1.fc32.aarch64",
    sha256 = "ec12fef82d73314e3e4cb6e962f8de27e78989fa104dde0599a4480a53817647",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ec12fef82d73314e3e4cb6e962f8de27e78989fa104dde0599a4480a53817647",
    ],
)

rpm(
    name = "xkeyboard-config-0__2.29-1.fc32.x86_64",
    sha256 = "ec12fef82d73314e3e4cb6e962f8de27e78989fa104dde0599a4480a53817647",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xkeyboard-config-2.29-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ec12fef82d73314e3e4cb6e962f8de27e78989fa104dde0599a4480a53817647",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-2.fc32.aarch64",
    sha256 = "de49c96d30ecc193263ecd2262f72d9e5bd2c8377a60005f51699c7bcd8c9bd9",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/x/xorriso-1.5.4-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/de49c96d30ecc193263ecd2262f72d9e5bd2c8377a60005f51699c7bcd8c9bd9",
    ],
)

rpm(
    name = "xorriso-0__1.5.4-2.fc32.x86_64",
    sha256 = "ada8be281e15fda282f2ce366082a4761deb4621a0d9d5bd93893699ded8c85b",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/x/xorriso-1.5.4-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ada8be281e15fda282f2ce366082a4761deb4621a0d9d5bd93893699ded8c85b",
    ],
)

rpm(
    name = "xz-0__5.2.5-1.fc32.aarch64",
    sha256 = "202d761caf4c9d4937c04388a7180d6687a79e8141136be0f7ecc3a54bf80594",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xz-5.2.5-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/x/xz-5.2.5-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xz-5.2.5-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xz-5.2.5-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/202d761caf4c9d4937c04388a7180d6687a79e8141136be0f7ecc3a54bf80594",
    ],
)

rpm(
    name = "xz-0__5.2.5-1.fc32.x86_64",
    sha256 = "1bdde5dc99a5588a8983f70b7b3e45e7006215d529c72adfec118c3bcbf7b01c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-5.2.5-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/1bdde5dc99a5588a8983f70b7b3e45e7006215d529c72adfec118c3bcbf7b01c",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-1.fc32.aarch64",
    sha256 = "48381163a3f2c524697efc07538f040fde0b69d4e0fdcbe3bcfbc9924dd7d5dd",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xz-libs-5.2.5-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/x/xz-libs-5.2.5-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xz-libs-5.2.5-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/x/xz-libs-5.2.5-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/48381163a3f2c524697efc07538f040fde0b69d4e0fdcbe3bcfbc9924dd7d5dd",
    ],
)

rpm(
    name = "xz-libs-0__5.2.5-1.fc32.x86_64",
    sha256 = "84702d6395a9577c1a268184f123cfd4b15bc2287f01033625ba388a34ec2338",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/x/xz-libs-5.2.5-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/84702d6395a9577c1a268184f123cfd4b15bc2287f01033625ba388a34ec2338",
    ],
)

rpm(
    name = "yajl-0__2.1.0-14.fc32.aarch64",
    sha256 = "c599bda69d6f4265be06e7206bfbf4a6a3c77b61bb960ddce807f5499736be4c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/y/yajl-2.1.0-14.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/y/yajl-2.1.0-14.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/y/yajl-2.1.0-14.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/y/yajl-2.1.0-14.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c599bda69d6f4265be06e7206bfbf4a6a3c77b61bb960ddce807f5499736be4c",
    ],
)

rpm(
    name = "yajl-0__2.1.0-14.fc32.x86_64",
    sha256 = "9194788f87e4a1aa8835f1305d290cc2cd67cee6a5b1ab82643d3a068c0145b6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/y/yajl-2.1.0-14.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/9194788f87e4a1aa8835f1305d290cc2cd67cee6a5b1ab82643d3a068c0145b6",
    ],
)

rpm(
    name = "zerofree-0__1.1.1-5.fc32.x86_64",
    sha256 = "a03c4cba5f8dccc36153ca72a46dd1935fa53e14767cb78aa02cdd025813419f",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zerofree-1.1.1-5.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zerofree-1.1.1-5.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zerofree-1.1.1-5.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zerofree-1.1.1-5.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a03c4cba5f8dccc36153ca72a46dd1935fa53e14767cb78aa02cdd025813419f",
    ],
)

rpm(
    name = "zlib-0__1.2.11-21.fc32.aarch64",
    sha256 = "df7184fef93e9f8f535d78349605595a812511db5e6dee26cbee15569a055422",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/z/zlib-1.2.11-21.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/z/zlib-1.2.11-21.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/z/zlib-1.2.11-21.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/z/zlib-1.2.11-21.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/df7184fef93e9f8f535d78349605595a812511db5e6dee26cbee15569a055422",
    ],
)

rpm(
    name = "zlib-0__1.2.11-21.fc32.x86_64",
    sha256 = "c0fff40dc1092e18ed3e608bc6143c89a0d7775b9e0553319bb2caca7d324d80",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/z/zlib-1.2.11-21.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/c0fff40dc1092e18ed3e608bc6143c89a0d7775b9e0553319bb2caca7d324d80",
    ],
)
