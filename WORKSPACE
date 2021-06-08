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
    sha256 = "778197e26c5fbeb07ac2a2c5ae405b30f6cb7ad1f5510ea6fdac03bded96cc6f",
    urls = [
        "https://github.com/bazelbuild/rules_python/releases/download/0.2.0/rules_python-0.2.0.tar.gz",
        "https://storage.googleapis.com/builddeps/778197e26c5fbeb07ac2a2c5ae405b30f6cb7ad1f5510ea6fdac03bded96cc6f",
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
    name = "fedora_image",
    sha256 = "423a4ce32fa32c50c11e3d3ff392db97a762533b81bef9d00599de518a7469c8",
    urls = ["https://storage.googleapis.com/builddeps/423a4ce32fa32c50c11e3d3ff392db97a762533b81bef9d00599de518a7469c8"],
)

http_file(
    name = "fedora_image_aarch64",
    sha256 = "b367755c664a2d7a26955bbfff985855adfa2ca15e908baf15b4b176d68d3967",
    urls = [
        "https://dl.fedoraproject.org/pub/fedora/linux/releases/32/Cloud/aarch64/images/Fedora-Cloud-Base-32-1.6.aarch64.qcow2",
        "https://storage.googleapis.com/builddeps/b367755c664a2d7a26955bbfff985855adfa2ca15e908baf15b4b176d68d3967",
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
    sha256 = "0877c23a751aafad5467e3ea992759f6774b7539955b4431c8f00b96e88c2509",
    strip_prefix = "bazeldnf-0.4.0",
    urls = [
        "https://github.com/rmohr/bazeldnf/archive/v0.4.0.tar.gz",
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

# Pull fedora 32 customize container-disk
# WARNING: please update any automated process to push this image to quay.io
# instead of index.docker.io
# TODO build fedora_sriov_lane for multi-arch
container_pull(
    name = "fedora_sriov_lane",
    digest = "sha256:6f66ee747d62c354c0d36e640f8c97d6be0b6ad88a9e8c0180496ac55cba31bf",
    registry = "quay.io",
    repository = "kubevirtci/fedora-sriov-testing",
)

container_pull(
    name = "fedora_sriov_lane_aarch64",
    digest = "sha256:6f66ee747d62c354c0d36e640f8c97d6be0b6ad88a9e8c0180496ac55cba31bf",
    registry = "quay.io",
    repository = "kubevirtci/fedora-sriov-testing",
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
    digest = "sha256:ce36d2b4f81b038fba0b61b1bb1ac7f671d47687fb1f9d7ddedd22742cc79dd9",
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
    name = "acl-0__2.2.53-1.el8.aarch64",
    sha256 = "47c2cc5872174c548de1096dc5673ee91349209d89e0193a4793955d6865b3b1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/acl-2.2.53-1.el8.aarch64.rpm"],
)

rpm(
    name = "acl-0__2.2.53-1.el8.x86_64",
    sha256 = "227de6071cd3aeca7e10ad386beaf38737d081e06350d02208a3f6a2c9710385",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/acl-2.2.53-1.el8.x86_64.rpm"],
)

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
    name = "attr-0__2.4.48-3.el8.x86_64",
    sha256 = "da1464c73554bd77756428d592f0cb9a8f65604c22c3b3b2b7db14b35f5ad178",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/attr-2.4.48-3.el8.x86_64.rpm"],
)

rpm(
    name = "audit-libs-0__3.0-0.17.20191104git1c2f876.el8.aarch64",
    sha256 = "11811c556a3bdc9c572c0ab67d3106bd1de3406c9d471de03e028f041b5785c3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/audit-libs-3.0-0.17.20191104git1c2f876.el8.aarch64.rpm"],
)

rpm(
    name = "audit-libs-0__3.0-0.17.20191104git1c2f876.el8.x86_64",
    sha256 = "e7da6b155db78fb2015c40663fec6e475a44b21b1c2124496cf23f862e021db8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/audit-libs-3.0-0.17.20191104git1c2f876.el8.x86_64.rpm"],
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
    name = "augeas-libs-0__1.12.0-6.el8.x86_64",
    sha256 = "60e90e9c353066b0d08136aacfd6731a0eef918ca3ab59d7ee117e5c0b7ba723",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/augeas-libs-1.12.0-6.el8.x86_64.rpm"],
)

rpm(
    name = "autogen-libopts-0__5.18.12-8.el8.aarch64",
    sha256 = "a69b87111415322e6586ba6b35494d77af7d9d58b2d9dfaf0360e4f827622dd2",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/autogen-libopts-5.18.12-8.el8.aarch64.rpm"],
)

rpm(
    name = "autogen-libopts-0__5.18.12-8.el8.x86_64",
    sha256 = "c73af033015bfbdbe8a43e162b098364d148517d394910f8db5d33b76b93aa48",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/autogen-libopts-5.18.12-8.el8.x86_64.rpm"],
)

rpm(
    name = "basesystem-0__11-5.el8.aarch64",
    sha256 = "48226934763e4c412c1eb65df314e6879720b4b1ebcb3d07c126c9526639cb68",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/basesystem-11-5.el8.noarch.rpm"],
)

rpm(
    name = "basesystem-0__11-5.el8.x86_64",
    sha256 = "48226934763e4c412c1eb65df314e6879720b4b1ebcb3d07c126c9526639cb68",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/basesystem-11-5.el8.noarch.rpm"],
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
    name = "bash-0__4.4.20-2.el8.aarch64",
    sha256 = "75dc9e6f813392de179031656e2ff5a3cc92771e545bf029a8bdeb42909e98e8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bash-4.4.20-2.el8.aarch64.rpm"],
)

rpm(
    name = "bash-0__4.4.20-2.el8.x86_64",
    sha256 = "d54031866d4dcf7fb9c16e105379a335fbd805b7967336900f4a3cd5934095b8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bash-4.4.20-2.el8.x86_64.rpm"],
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
    name = "bind-export-libs-32__9.11.26-4.el8_4.x86_64",
    sha256 = "bb6b8ab618f130992d183a8df8f466bbca2293edbbf8c5f46dac5e4e3123457c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bind-export-libs-9.11.26-4.el8_4.x86_64.rpm"],
)

rpm(
    name = "binutils-0__2.30-108.el8.x86_64",
    sha256 = "57bdf5f44c38832ee276df320326aba08f4fa40cd4831f85349fe0055e44235a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/binutils-2.30-108.el8.x86_64.rpm"],
)

rpm(
    name = "boost-iostreams-0__1.66.0-10.el8.x86_64",
    sha256 = "785eb0669099397b2b7d5b2e5554f2c2cc19d053b7db994e3dd45cd69405d3bc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/boost-iostreams-1.66.0-10.el8.x86_64.rpm"],
)

rpm(
    name = "bzip2-0__1.0.6-26.el8.aarch64",
    sha256 = "b18d9f23161d7d5de93fa72a56c645762deefbc0f3e5a095bb8d9e3cf09521e6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bzip2-1.0.6-26.el8.aarch64.rpm"],
)

rpm(
    name = "bzip2-0__1.0.6-26.el8.x86_64",
    sha256 = "78596f457c3d737a97a4edfe9a03a01f593606379c281701ab7f7eba13ecaf18",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bzip2-1.0.6-26.el8.x86_64.rpm"],
)

rpm(
    name = "bzip2-libs-0__1.0.6-26.el8.aarch64",
    sha256 = "a4451cae0e8a3307228ed8ac7dc9bab7de77fcbf2004141daa7f986f5dc9b381",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/bzip2-libs-1.0.6-26.el8.aarch64.rpm"],
)

rpm(
    name = "bzip2-libs-0__1.0.6-26.el8.x86_64",
    sha256 = "19d66d152b745dbd49cea9d21c52aec0ec4d4321edef97a342acd3542404fa31",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/bzip2-libs-1.0.6-26.el8.x86_64.rpm"],
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
    name = "ca-certificates-0__2021.2.50-82.el8.aarch64",
    sha256 = "1fad1d1f8b56e6967863aeb60f5fa3615e6a35b0f6532d8a23066e6823b50860",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ca-certificates-2021.2.50-82.el8.noarch.rpm"],
)

rpm(
    name = "ca-certificates-0__2021.2.50-82.el8.x86_64",
    sha256 = "1fad1d1f8b56e6967863aeb60f5fa3615e6a35b0f6532d8a23066e6823b50860",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ca-certificates-2021.2.50-82.el8.noarch.rpm"],
)

rpm(
    name = "centos-gpg-keys-1__8-2.el8.aarch64",
    sha256 = "842ff55b80ac9a5c3357bf52646a5761a4c4786bb3e64b56d8fa5d8fe34ef8bb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-gpg-keys-8-2.el8.noarch.rpm"],
)

rpm(
    name = "centos-gpg-keys-1__8-2.el8.x86_64",
    sha256 = "842ff55b80ac9a5c3357bf52646a5761a4c4786bb3e64b56d8fa5d8fe34ef8bb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-gpg-keys-8-2.el8.noarch.rpm"],
)

rpm(
    name = "centos-stream-release-0__8.5-3.el8.aarch64",
    sha256 = "15b17b95cf9cb9fb64e0c8e56110836bdcaf70de81b8cbdb60e181fc90456e06",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-stream-release-8.5-3.el8.noarch.rpm"],
)

rpm(
    name = "centos-stream-release-0__8.5-3.el8.x86_64",
    sha256 = "15b17b95cf9cb9fb64e0c8e56110836bdcaf70de81b8cbdb60e181fc90456e06",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-release-8.5-3.el8.noarch.rpm"],
)

rpm(
    name = "centos-stream-repos-0__8-2.el8.aarch64",
    sha256 = "a82958266d292f4725fc1981ea57a861b7fc7feeb7a9551d0b61b98ca51a5662",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/centos-stream-repos-8-2.el8.noarch.rpm"],
)

rpm(
    name = "centos-stream-repos-0__8-2.el8.x86_64",
    sha256 = "a82958266d292f4725fc1981ea57a861b7fc7feeb7a9551d0b61b98ca51a5662",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/centos-stream-repos-8-2.el8.noarch.rpm"],
)

rpm(
    name = "checkpolicy-0__2.9-1.el8.aarch64",
    sha256 = "01b89be34e48d345ba14a3856bba0d1ff94e79798b5f7529a6a0803b97adca15",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/checkpolicy-2.9-1.el8.aarch64.rpm"],
)

rpm(
    name = "checkpolicy-0__2.9-1.el8.x86_64",
    sha256 = "d5c283da0d2666742635754626263f6f78e273cd46d83d2d66ed43730a731685",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/checkpolicy-2.9-1.el8.x86_64.rpm"],
)

rpm(
    name = "chkconfig-0__1.19.1-1.el8.aarch64",
    sha256 = "be370bfc2f375cdbfc1079b19423142236770cf67caf74cdb12a7aef8a29c8c5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/chkconfig-1.19.1-1.el8.aarch64.rpm"],
)

rpm(
    name = "chkconfig-0__1.19.1-1.el8.x86_64",
    sha256 = "561b5fdadd60370b5d0a91b7ed35df95d7f60650cbade8c7e744323982ac82db",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/chkconfig-1.19.1-1.el8.x86_64.rpm"],
)

rpm(
    name = "coreutils-single-0__8.30-12.el8.aarch64",
    sha256 = "2a72f27d58b3e9364a872fb089322a570477fc108ceeb5d304a2b831ab6f3e23",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/coreutils-single-8.30-12.el8.aarch64.rpm"],
)

rpm(
    name = "coreutils-single-0__8.30-12.el8.x86_64",
    sha256 = "2eb9c891de4f7281c7068351ffab36f93a8fa1e9d16b694c70968ed1a66a5f04",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/coreutils-single-8.30-12.el8.x86_64.rpm"],
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
    name = "cpio-0__2.12-10.el8.x86_64",
    sha256 = "10e88a1794107ea61f1441273be906704d66802b21800b0840a2870cad8b4d63",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cpio-2.12-10.el8.x86_64.rpm"],
)

rpm(
    name = "cracklib-0__2.9.6-15.el8.aarch64",
    sha256 = "54efb853142572e1c2872e351838fc3657b662722ff6b2913d1872d4752a0eb8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cracklib-2.9.6-15.el8.aarch64.rpm"],
)

rpm(
    name = "cracklib-0__2.9.6-15.el8.x86_64",
    sha256 = "dbbc9e20caabc30070354d91f61f383081f6d658e09d3c09e6df8764559e5aca",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cracklib-2.9.6-15.el8.x86_64.rpm"],
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
    name = "cracklib-dicts-0__2.9.6-15.el8.aarch64",
    sha256 = "d61741af0ffe96c55f588dd164b9c3c93e7c7175c7e616db25990ab3e16e0f22",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cracklib-dicts-2.9.6-15.el8.aarch64.rpm"],
)

rpm(
    name = "cracklib-dicts-0__2.9.6-15.el8.x86_64",
    sha256 = "f1ce23ee43c747a35367dada19ca200a7758c50955ccc44aa946b86b647077ca",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cracklib-dicts-2.9.6-15.el8.x86_64.rpm"],
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
    name = "crypto-policies-0__20210617-1.gitc776d3e.el8.aarch64",
    sha256 = "2a8f9e5119a034801904185dcbf1bc29db67e9e9b0cf5893615722d7bb33099c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/crypto-policies-20210617-1.gitc776d3e.el8.noarch.rpm"],
)

rpm(
    name = "crypto-policies-0__20210617-1.gitc776d3e.el8.x86_64",
    sha256 = "2a8f9e5119a034801904185dcbf1bc29db67e9e9b0cf5893615722d7bb33099c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/crypto-policies-20210617-1.gitc776d3e.el8.noarch.rpm"],
)

rpm(
    name = "cryptsetup-0__2.3.3-4.el8.x86_64",
    sha256 = "fab7d620fb953f64b8a01b93c835e4a8a59aa1e9459e58dc3211db493d6f2c35",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-2.3.3-4.el8.x86_64.rpm"],
)

rpm(
    name = "cryptsetup-libs-0__2.3.3-4.el8.aarch64",
    sha256 = "c94d212f77d5d83ba1bd22a5c6b5e92590d5c4cb412950ec22d1309d79e2fc0e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cryptsetup-libs-2.3.3-4.el8.aarch64.rpm"],
)

rpm(
    name = "cryptsetup-libs-0__2.3.3-4.el8.x86_64",
    sha256 = "679d78e677c3be4a5ee747feee9bbc4ccf59d489321da44253048b7d76beba97",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cryptsetup-libs-2.3.3-4.el8.x86_64.rpm"],
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
    name = "curl-0__7.61.1-18.el8.aarch64",
    sha256 = "89ebdd969468d9c9669fa65c9c92f0b66b306ef430f8c913663eefd789496e74",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/curl-7.61.1-18.el8.aarch64.rpm"],
)

rpm(
    name = "curl-0__7.61.1-18.el8.x86_64",
    sha256 = "51fdf97c00f76054ca2a795e077dc0ecb053f45a06052da9ab383578de796c75",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/curl-7.61.1-18.el8.x86_64.rpm"],
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
    name = "cyrus-sasl-0__2.1.27-5.el8.aarch64",
    sha256 = "7dcb85af91070dca195ad82b91476d6cbbb4fef192e2f5c0a318d228ffedfbac",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-2.1.27-5.el8.aarch64.rpm"],
)

rpm(
    name = "cyrus-sasl-0__2.1.27-5.el8.x86_64",
    sha256 = "41cf36b5d082794509fece3681e8b7a0000574efef834c611820d00ecf6f2d78",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-2.1.27-5.el8.x86_64.rpm"],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-5.el8.aarch64",
    sha256 = "f02f26dc5be5410aa233a0b50821df2c63f81772aef7f31c4be557319f1080e1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-gssapi-2.1.27-5.el8.aarch64.rpm"],
)

rpm(
    name = "cyrus-sasl-gssapi-0__2.1.27-5.el8.x86_64",
    sha256 = "a7af455a12a4df52523efc4be6f4da1065d1e83c73209844ba331c00d1d409a3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-gssapi-2.1.27-5.el8.x86_64.rpm"],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-5.el8.aarch64",
    sha256 = "36d4e208921238b99c822a5f1686120c0c227fc02dc6e3258c2c71d62492a1e7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/cyrus-sasl-lib-2.1.27-5.el8.aarch64.rpm"],
)

rpm(
    name = "cyrus-sasl-lib-0__2.1.27-5.el8.x86_64",
    sha256 = "c421b9c029abac796ade606f96d638e06a6d4ce5c2d499abd05812c306d25143",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/cyrus-sasl-lib-2.1.27-5.el8.x86_64.rpm"],
)

rpm(
    name = "daxctl-libs-0__71.1-2.el8.x86_64",
    sha256 = "c879ce8eea2780a4cf4ffe67661fe56aaf2e6c110b45a1ead147925065a2eeb2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/daxctl-libs-71.1-2.el8.x86_64.rpm"],
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
    name = "dbus-1__1.12.8-14.el8.aarch64",
    sha256 = "107a781be497f1a51ffd370aba59dbc4de3d7f89802830c66051dc51a5ec185b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-1.12.8-14.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-1__1.12.8-14.el8.x86_64",
    sha256 = "a61f7b7bccd0168f654f54e7a1acfb597bf018bbda267140d2049e58563c6f12",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-1.12.8-14.el8.x86_64.rpm"],
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
    name = "dbus-common-1__1.12.8-14.el8.aarch64",
    sha256 = "7baac88adafdc5958fb818c7685d3c6548f6e2e585e4435ceee4a168edc3597e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-common-1.12.8-14.el8.noarch.rpm"],
)

rpm(
    name = "dbus-common-1__1.12.8-14.el8.x86_64",
    sha256 = "7baac88adafdc5958fb818c7685d3c6548f6e2e585e4435ceee4a168edc3597e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-common-1.12.8-14.el8.noarch.rpm"],
)

rpm(
    name = "dbus-daemon-1__1.12.8-14.el8.aarch64",
    sha256 = "69e6fa2fa4a60384e21913b69cf4ddd6a21148e3d984a4ff0cbe651a2986f738",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-daemon-1.12.8-14.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-daemon-1__1.12.8-14.el8.x86_64",
    sha256 = "c15824e278323ba2ef0e3fab5c2c39d04137485dfa0298e43d19a6d2ca667f6c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-daemon-1.12.8-14.el8.x86_64.rpm"],
)

rpm(
    name = "dbus-libs-1__1.12.8-14.el8.aarch64",
    sha256 = "9738cb7597fa6dd4e3bee9159e813e6188894f98852fb896b95437f7fc8dbd8d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-libs-1.12.8-14.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-libs-1__1.12.8-14.el8.x86_64",
    sha256 = "7533e19781d1b7e354315b15ef3d3011a8f1eec8980f9e8a6c633af3a806db2a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-libs-1.12.8-14.el8.x86_64.rpm"],
)

rpm(
    name = "dbus-tools-1__1.12.8-14.el8.aarch64",
    sha256 = "da2dd7c4192fbafc3dfda1769b03fa27ec1855dd54963e774eb404f44a85b8e7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/dbus-tools-1.12.8-14.el8.aarch64.rpm"],
)

rpm(
    name = "dbus-tools-1__1.12.8-14.el8.x86_64",
    sha256 = "6032a05a8c33bc9d6be816d4172e8bb24a18ec873c127f5bee94da9210130e8b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dbus-tools-1.12.8-14.el8.x86_64.rpm"],
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
    name = "device-mapper-8__1.02.177-3.el8.aarch64",
    sha256 = "d3498552c92c4fbe1771096d2519a015240e25ccf4dbc6186e585efd11b89e26",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-1.02.177-3.el8.aarch64.rpm"],
)

rpm(
    name = "device-mapper-8__1.02.177-3.el8.x86_64",
    sha256 = "998ae36d127d39db5712a6942543eeda8fdcbb1f936f987ca5ff438c69598315",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-1.02.177-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-event-8__1.02.177-3.el8.x86_64",
    sha256 = "8ceeeefc9ae284d553b468b0e3c06594d8743e0a0ce258026f4b2e4f6d62dc62",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-1.02.177-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-event-libs-8__1.02.177-3.el8.x86_64",
    sha256 = "44f6c5b7a801f75b8e0b5c70897ef478b807a3a0c3f685e353bfe313573f25af",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-event-libs-1.02.177-3.el8.x86_64.rpm"],
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
    name = "device-mapper-libs-8__1.02.177-3.el8.aarch64",
    sha256 = "548b7899af4880221d7d2d9c30ebadb2bc044ba0b9d888303859cae84cf8ae29",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-libs-1.02.177-3.el8.aarch64.rpm"],
)

rpm(
    name = "device-mapper-libs-8__1.02.177-3.el8.x86_64",
    sha256 = "100239988a40cfc64d8e5aaed1811890b73501aed27d6df4e1915d4a9512194d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-libs-1.02.177-3.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-17.el8.aarch64",
    sha256 = "cf7f72e70414ffdb5e9f672d802f367077ce39f25fea1c76085ec87b64c394b5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/device-mapper-multipath-libs-0.8.4-17.el8.aarch64.rpm"],
)

rpm(
    name = "device-mapper-multipath-libs-0__0.8.4-17.el8.x86_64",
    sha256 = "bf38d632512ca8136ce50d8e75399b0c8ec1ef44d032e5277e52bb6e6d9cb571",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-multipath-libs-0.8.4-17.el8.x86_64.rpm"],
)

rpm(
    name = "device-mapper-persistent-data-0__0.9.0-1.el8.x86_64",
    sha256 = "1b5c47b9db3d22be28711759fcc3a5acaa8cbc034a467ca2d72c056e4e1c2c5d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/device-mapper-persistent-data-0.9.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "dhcp-client-12__4.3.6-45.el8.x86_64",
    sha256 = "17b8055a23085caee5db3a3b35d5571f228865bdfb06d0bdfbe4cdba52e82350",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-client-4.3.6-45.el8.x86_64.rpm"],
)

rpm(
    name = "dhcp-common-12__4.3.6-45.el8.x86_64",
    sha256 = "d64da85b0e013679fb06eeadad74465d9556d7ad21e97beb1b5f61fb0658cd10",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-common-4.3.6-45.el8.noarch.rpm"],
)

rpm(
    name = "dhcp-libs-12__4.3.6-45.el8.x86_64",
    sha256 = "7e55750fb56250d3f8ad91edbf4b865a09b7baf7797dc4eaa668b473043efeca",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dhcp-libs-4.3.6-45.el8.x86_64.rpm"],
)

rpm(
    name = "diffutils-0__3.6-6.el8.aarch64",
    sha256 = "8cbebc0fa970ceca4f479ee292eaad155084987be2cf7f97bbafe4a529319c98",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/diffutils-3.6-6.el8.aarch64.rpm"],
)

rpm(
    name = "diffutils-0__3.6-6.el8.x86_64",
    sha256 = "c515d78c64a93d8b469593bff5800eccd50f24b16697ab13bdce81238c38eb77",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/diffutils-3.6-6.el8.x86_64.rpm"],
)

rpm(
    name = "dmidecode-1__3.2-10.el8.x86_64",
    sha256 = "cd2f140bd1718b4403ad66568155264b69231748a3c813e2feaac1b704da62c6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dmidecode-3.2-10.el8.x86_64.rpm"],
)

rpm(
    name = "dnf-0__4.7.0-1.el8.x86_64",
    sha256 = "088b14b3c618d22eabff553dbba2737a03b9f864b542154be2a1b8fa5f196cba",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dnf-4.7.0-1.el8.noarch.rpm"],
)

rpm(
    name = "dnf-plugins-core-0__4.0.21-1.el8.x86_64",
    sha256 = "f8018a6754470faeb0538e1229191dab0f8bf449058811713aa7704654c820a5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dnf-plugins-core-4.0.21-1.el8.noarch.rpm"],
)

rpm(
    name = "dosfstools-0__4.1-6.el8.x86_64",
    sha256 = "40676b73567e195228ba2a8bb53692f88f88d43612564613fb168383eee57f6a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/dosfstools-4.1-6.el8.x86_64.rpm"],
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
    name = "e2fsprogs-0__1.45.6-2.el8.x86_64",
    sha256 = "f46dee25409f262173a127102dd9c7a3aced4feecf935a2340a043c17b1f8c61",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-1.45.6-2.el8.x86_64.rpm"],
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
    name = "e2fsprogs-libs-0__1.45.6-2.el8.x86_64",
    sha256 = "037d854bec991cd4f0827ff5903b33847ef6965f4aba44252f30f412d49afdac",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/e2fsprogs-libs-1.45.6-2.el8.x86_64.rpm"],
)

rpm(
    name = "edk2-aarch64-0__20200602gitca407c7246bf-4.el8.aarch64",
    sha256 = "1a2f9802fefaa0e6fab41557b0c2f02968d443a71d0bd83eae98882b0ce7df6d",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/edk2-aarch64-20200602gitca407c7246bf-4.el8.noarch.rpm"],
)

rpm(
    name = "edk2-ovmf-0__20200602gitca407c7246bf-4.el8.x86_64",
    sha256 = "3fe8746248ada6d0421d3108aa1db0c602a65b09aa5ddf9201ce8305529a129b",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/edk2-ovmf-20200602gitca407c7246bf-4.el8.noarch.rpm"],
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
    name = "elfutils-default-yama-scope-0__0.185-1.el8.aarch64",
    sha256 = "30ceeb5a6cadaeccdbde088bfb52ba88190fa530c11f4a2aafd62b4b4ad6b404",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-default-yama-scope-0.185-1.el8.noarch.rpm"],
)

rpm(
    name = "elfutils-default-yama-scope-0__0.185-1.el8.x86_64",
    sha256 = "30ceeb5a6cadaeccdbde088bfb52ba88190fa530c11f4a2aafd62b4b4ad6b404",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-default-yama-scope-0.185-1.el8.noarch.rpm"],
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
    name = "elfutils-libelf-0__0.185-1.el8.aarch64",
    sha256 = "25788279ab5869acfcaf46186ef08dc6908d6a90f6f4ff6ba9474a1fde3870fd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libelf-0.185-1.el8.aarch64.rpm"],
)

rpm(
    name = "elfutils-libelf-0__0.185-1.el8.x86_64",
    sha256 = "b56349ce3abac926fad2ef8366080e0823c4719235e72cb47306f4e9a39a0d66",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libelf-0.185-1.el8.x86_64.rpm"],
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
    name = "elfutils-libs-0__0.185-1.el8.aarch64",
    sha256 = "cb7464d6e1440b4218eb668edaa67b6a43ecd647d8915a6e96d5f955ad69f09c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/elfutils-libs-0.185-1.el8.aarch64.rpm"],
)

rpm(
    name = "elfutils-libs-0__0.185-1.el8.x86_64",
    sha256 = "abfb7d93009c64a38d1e938093eb109ad344b150272ac644dc8ea6a3bd64adef",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/elfutils-libs-0.185-1.el8.x86_64.rpm"],
)

rpm(
    name = "expat-0__2.2.5-4.el8.aarch64",
    sha256 = "16356a5f29d0b191e84e37c92f9b6a3cd2ef683c84dd37c065f3461ad5abef03",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/expat-2.2.5-4.el8.aarch64.rpm"],
)

rpm(
    name = "expat-0__2.2.5-4.el8.x86_64",
    sha256 = "0c451ef9a9cd603a35aaab1a6c4aba83103332bed7c2b7393c48631f9bb50158",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/expat-2.2.5-4.el8.x86_64.rpm"],
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
    name = "file-0__5.33-20.el8.x86_64",
    sha256 = "9729d5fd2ecbf6902329585a4acdd09f2f591673802ca89dd575ba8351991814",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/file-5.33-20.el8.x86_64.rpm"],
)

rpm(
    name = "file-libs-0__5.33-20.el8.x86_64",
    sha256 = "216250c4239243c7692981146d9a0eb08434c9f8d4b1321ef31302e9dbf08384",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/file-libs-5.33-20.el8.x86_64.rpm"],
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
    name = "filesystem-0__3.8-6.el8.aarch64",
    sha256 = "e6c3fa94860eda0bc2ae6b1b78acd1159cbed355a03e7bec8b3defa1d90782b6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/filesystem-3.8-6.el8.aarch64.rpm"],
)

rpm(
    name = "filesystem-0__3.8-6.el8.x86_64",
    sha256 = "50bdb81d578914e0e88fe6b13550b4c30aac4d72f064fdcd78523df7dd2f64da",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/filesystem-3.8-6.el8.x86_64.rpm"],
)

rpm(
    name = "findutils-1__4.6.0-20.el8.aarch64",
    sha256 = "985479064966d05aa82010ed5b8905942e47e2bebb919c9c1bd004a28addad1d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/findutils-4.6.0-20.el8.aarch64.rpm"],
)

rpm(
    name = "findutils-1__4.6.0-20.el8.x86_64",
    sha256 = "811eb112646b7d87773c65af47efdca975468f3e5df44aa9944e30de24d83890",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/findutils-4.6.0-20.el8.x86_64.rpm"],
)

rpm(
    name = "fuse-0__2.9.7-12.el8.x86_64",
    sha256 = "2465c0c3b3d9519a3f9ae2ffe3e2c0bc61dca6fcb6ae710a6c7951007f498864",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-2.9.7-12.el8.x86_64.rpm"],
)

rpm(
    name = "fuse-common-0__3.2.1-12.el8.x86_64",
    sha256 = "3f947e1e56d0b0210f9ccbc4483f8b6bfb100cfd79ea1efac3336a8d624ec0d6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-common-3.2.1-12.el8.x86_64.rpm"],
)

rpm(
    name = "fuse-libs-0__2.9.7-12.el8.x86_64",
    sha256 = "6c6c98e2ddc2210ca377b0ef0c6bb694abd23f33413acadaedc1760da5bcc079",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/fuse-libs-2.9.7-12.el8.x86_64.rpm"],
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
    name = "gawk-0__4.2.1-2.el8.aarch64",
    sha256 = "1597024288d637f0865ca9be73fb1f2e5c495005fa9ca5b3aacc6d8ab8f444a8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gawk-4.2.1-2.el8.aarch64.rpm"],
)

rpm(
    name = "gawk-0__4.2.1-2.el8.x86_64",
    sha256 = "bc0d36db80589a9797b8c343cd80f5ad5f42b9afc88f8a46666dc1d8f5317cfe",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gawk-4.2.1-2.el8.x86_64.rpm"],
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
    name = "gdbm-1__1.18-1.el8.aarch64",
    sha256 = "b7d0b4b922429354ffe7ddac90c8cd448229571b8d8e4c342110edadfe809f99",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gdbm-1.18-1.el8.aarch64.rpm"],
)

rpm(
    name = "gdbm-1__1.18-1.el8.x86_64",
    sha256 = "76d81e433a5291df491d2e289de9b33d4e5b98dcf48fd0a003c2767415d3e0aa",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdbm-1.18-1.el8.x86_64.rpm"],
)

rpm(
    name = "gdbm-libs-1__1.18-1.el8.aarch64",
    sha256 = "a7d04ae40ad91ba0ea93e4971a35585638f6adf8dbe1ed4849f643b6b64a5871",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gdbm-libs-1.18-1.el8.aarch64.rpm"],
)

rpm(
    name = "gdbm-libs-1__1.18-1.el8.x86_64",
    sha256 = "3a3cb5a11f8e844cd1bf7c0e7bb6c12cc63e743029df50916ce7e6a9f8a4e169",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdbm-libs-1.18-1.el8.x86_64.rpm"],
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
    name = "gdisk-0__1.0.3-6.el8.x86_64",
    sha256 = "fa0b90c4da7f7ca8bf40055be5641a2c57708931fec5f760a2f8944325669fe9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gdisk-1.0.3-6.el8.x86_64.rpm"],
)

rpm(
    name = "genisoimage-0__1.1.11-39.el8.x86_64",
    sha256 = "f98e67e6ed49e1ff2f4c1d8dea7aa139aaff69020013e458d2f3d8bd9d2c91b2",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/genisoimage-1.1.11-39.el8.x86_64.rpm"],
)

rpm(
    name = "gettext-0__0.19.8.1-17.el8.aarch64",
    sha256 = "5f0c37488d3017b052039ddb8d9189a38c252af97884264959334237109c7e7c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gettext-0.19.8.1-17.el8.aarch64.rpm"],
)

rpm(
    name = "gettext-0__0.19.8.1-17.el8.x86_64",
    sha256 = "829c842bbd79dca18d37198414626894c44e5b8faf0cce0054ca0ba6623ae136",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gettext-0.19.8.1-17.el8.x86_64.rpm"],
)

rpm(
    name = "gettext-libs-0__0.19.8.1-17.el8.aarch64",
    sha256 = "882f23e0250a2d4aea49abb4ec8e11a9a3869ccdd812c796b6f85341ff9d30a2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gettext-libs-0.19.8.1-17.el8.aarch64.rpm"],
)

rpm(
    name = "gettext-libs-0__0.19.8.1-17.el8.x86_64",
    sha256 = "ade52756aaf236e77dadd6cf97716821141c2759129ca7808524ab79607bb4c4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gettext-libs-0.19.8.1-17.el8.x86_64.rpm"],
)

rpm(
    name = "glib2-0__2.56.4-156.el8.aarch64",
    sha256 = "7518d3628201571931a67d1fda2f0a53107010603c0c74012469f05d701f66b8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glib2-2.56.4-156.el8.aarch64.rpm"],
)

rpm(
    name = "glib2-0__2.56.4-156.el8.x86_64",
    sha256 = "4f27488803c8949f7c6e07c0df73dd77dec6d75db43b7876a0ffc16ee751fe6b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glib2-2.56.4-156.el8.x86_64.rpm"],
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
    name = "glibc-0__2.28-162.el8.aarch64",
    sha256 = "7f4864ca5761b3e312b0cea1e88b6b9e59df9bb472e07dc088d9c78bc1135dfa",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-2.28-162.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-0__2.28-162.el8.x86_64",
    sha256 = "0adf23a6367b8f5a8fa0fac9044a444fa0105020c065a656355f41024ed0d115",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-2.28-162.el8.x86_64.rpm"],
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
    name = "glibc-common-0__2.28-162.el8.aarch64",
    sha256 = "f736ce73051513717bca871f0ef99f23c73b3d00fe44aa5efa8589d52bc3d30f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-common-2.28-162.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-common-0__2.28-162.el8.x86_64",
    sha256 = "a9bcf126ce77b8fb4c8710a03648c57eac2f21fe18ee5c9f7b3b15e5f0eb8f33",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-common-2.28-162.el8.x86_64.rpm"],
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
    name = "glibc-minimal-langpack-0__2.28-162.el8.aarch64",
    sha256 = "c9d2fc55d1648f1336bd3edb3e125ad2a054ca173428c730f078b43457eb53a1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/glibc-minimal-langpack-2.28-162.el8.aarch64.rpm"],
)

rpm(
    name = "glibc-minimal-langpack-0__2.28-162.el8.x86_64",
    sha256 = "c47e67323e17f2ecee6a6027f1bb1ce122c8566d3ad6a4179dbecdabc5a87428",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/glibc-minimal-langpack-2.28-162.el8.x86_64.rpm"],
)

rpm(
    name = "gmp-1__6.1.2-10.el8.aarch64",
    sha256 = "8d407f8ad961169fca2ee5e22e824cbc2d2b5fedca9701896cc492d4cb788603",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gmp-6.1.2-10.el8.aarch64.rpm"],
)

rpm(
    name = "gmp-1__6.1.2-10.el8.x86_64",
    sha256 = "3b96e2c7d5cd4b49bfde8e52c8af6ff595c91438e50856e468f14a049d8511e2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gmp-6.1.2-10.el8.x86_64.rpm"],
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
    name = "gnupg2-0__2.2.20-2.el8.x86_64",
    sha256 = "42842cc39272d095d01d076982d4e9aa4888c7b2a1c26ebed6fb6ef9a02680ba",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gnupg2-2.2.20-2.el8.x86_64.rpm"],
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
    name = "gnutls-0__3.6.16-4.el8.aarch64",
    sha256 = "f97d55f7bdf6fe126e7a1446563af7ee4c1bb7ee3a2a9b12b6df1cdd344da47e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gnutls-3.6.16-4.el8.aarch64.rpm"],
)

rpm(
    name = "gnutls-0__3.6.16-4.el8.x86_64",
    sha256 = "51bae480875ce4f8dd76b0af177c88eb1bd33faa910dbd64e574ef8c7ada1d03",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gnutls-3.6.16-4.el8.x86_64.rpm"],
)

rpm(
    name = "gnutls-dane-0__3.6.16-4.el8.aarch64",
    sha256 = "df78e84002d6ba09e37901b2f85f462a160beda734e98876c8baba0c71caf638",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gnutls-dane-3.6.16-4.el8.aarch64.rpm"],
)

rpm(
    name = "gnutls-dane-0__3.6.16-4.el8.x86_64",
    sha256 = "122d2a8e70c4cb857803e8b3673ca8dc572ba21ce790064abc4c99cca0f94b3f",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gnutls-dane-3.6.16-4.el8.x86_64.rpm"],
)

rpm(
    name = "gnutls-utils-0__3.6.16-4.el8.aarch64",
    sha256 = "1421e7f87f559b398b9bd289ee10c79b38a0505613761b4499ad9747aafb7da6",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/gnutls-utils-3.6.16-4.el8.aarch64.rpm"],
)

rpm(
    name = "gnutls-utils-0__3.6.16-4.el8.x86_64",
    sha256 = "58bc517e7d159bffa96db5cb5fd132e7e1798b8685ebb35d22a62ab6db51ced7",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/gnutls-utils-3.6.16-4.el8.x86_64.rpm"],
)

rpm(
    name = "gperftools-libs-0__2.7-7.fc32.aarch64",
    sha256 = "a6cc9ca54d874f5ca89954fbaa205d5979b411adc94962627aff78f2f5a69aef",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gperftools-libs-2.7-7.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/g/gperftools-libs-2.7-7.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gperftools-libs-2.7-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/g/gperftools-libs-2.7-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a6cc9ca54d874f5ca89954fbaa205d5979b411adc94962627aff78f2f5a69aef",
    ],
)

rpm(
    name = "gperftools-libs-0__2.7-7.fc32.x86_64",
    sha256 = "4bde0737a685e82c732b9a5d2daf08a0b6a66c0abd699defcfefc0c7bd2ecdf6",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/g/gperftools-libs-2.7-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4bde0737a685e82c732b9a5d2daf08a0b6a66c0abd699defcfefc0c7bd2ecdf6",
    ],
)

rpm(
    name = "grep-0__3.1-6.el8.aarch64",
    sha256 = "7ffd6e95b0554466e97346b2f41fb5279aedcb29ae07828f63d06a8dedd7cd51",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/grep-3.1-6.el8.aarch64.rpm"],
)

rpm(
    name = "grep-0__3.1-6.el8.x86_64",
    sha256 = "3f8ffe48bb481a5db7cbe42bf73b839d872351811e5df41b2f6697c61a030487",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/grep-3.1-6.el8.x86_64.rpm"],
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
    name = "groff-base-0__1.22.3-22.fc32.aarch64",
    sha256 = "93da9ee61ab9f2f0135b85c1656c39a923833c576c1cb6c0c551d4e8031170b0",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/groff-base-1.22.3-22.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/g/groff-base-1.22.3-22.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/g/groff-base-1.22.3-22.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/g/groff-base-1.22.3-22.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/93da9ee61ab9f2f0135b85c1656c39a923833c576c1cb6c0c551d4e8031170b0",
    ],
)

rpm(
    name = "groff-base-0__1.22.3-22.fc32.x86_64",
    sha256 = "a81e62e044a9cb5c752e55b3e6e40c3248ca0b595236d8f6f62e42251379454d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/g/groff-base-1.22.3-22.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/a81e62e044a9cb5c752e55b3e6e40c3248ca0b595236d8f6f62e42251379454d",
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
    name = "gzip-0__1.9-12.el8.aarch64",
    sha256 = "1fe57a2d38c0d449efd06fa3e498e49f1952829f612d657418a7496458c0cb7c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/gzip-1.9-12.el8.aarch64.rpm"],
)

rpm(
    name = "gzip-0__1.9-12.el8.x86_64",
    sha256 = "6d995888083240517e8eb5e0c8d8c22e63ac46de3b4bcd3c61e14959558800dd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/gzip-1.9-12.el8.x86_64.rpm"],
)

rpm(
    name = "hexedit-0__1.2.13-12.el8.x86_64",
    sha256 = "4538e44d3ebff3f9323b59171767bca2b7f5244dd90141de101856ad4f4643f5",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/hexedit-1.2.13-12.el8.x86_64.rpm"],
)

rpm(
    name = "hivex-0__1.3.18-21.module_el8.5.0__plus__821__plus__97472045.x86_64",
    sha256 = "f27b8da68c6f5f5b49bda20ece543cc7fd0b8693fa8255a0621d0a4041e10158",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/hivex-1.3.18-21.module_el8.5.0+821+97472045.x86_64.rpm"],
)

rpm(
    name = "hwdata-0__0.314-8.9.el8.aarch64",
    sha256 = "dfecfa1299d11d6a77503e626a2e9a14e1a75666bc8ea2abaf7ff515d2c86332",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/hwdata-0.314-8.9.el8.noarch.rpm"],
)

rpm(
    name = "hwdata-0__0.314-8.9.el8.x86_64",
    sha256 = "dfecfa1299d11d6a77503e626a2e9a14e1a75666bc8ea2abaf7ff515d2c86332",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/hwdata-0.314-8.9.el8.noarch.rpm"],
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
    name = "info-0__6.5-6.el8.aarch64",
    sha256 = "187a1fbb7e2992dfa777c7ca5c2f7369ecb85e4be4a483e6c0c6036e02bacf95",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/info-6.5-6.el8.aarch64.rpm"],
)

rpm(
    name = "info-0__6.5-6.el8.x86_64",
    sha256 = "611da4957e11f4621f53b5d7d491bcba09854de4fad8a5be34e762f4f36b1102",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/info-6.5-6.el8.x86_64.rpm"],
)

rpm(
    name = "ipcalc-0__0.2.4-4.el8.x86_64",
    sha256 = "dea18976861575d40ffca814dee08a225376c7828a5afc9e5d0a383edd3d8907",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ipcalc-0.2.4-4.el8.x86_64.rpm"],
)

rpm(
    name = "iproute-0__5.12.0-1.el8.aarch64",
    sha256 = "871915346b85123849c9fbbd066066ab1161ae530bd1f64b9fea25b1537868b9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-5.12.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "iproute-0__5.12.0-1.el8.x86_64",
    sha256 = "13fea0b2e3baac7914b26a9bf4a8c3115b23ddb4e67ae3b83d01397924fb9860",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-5.12.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "iproute-tc-0__5.12.0-1.el8.aarch64",
    sha256 = "cb25b4c1e537111429b2a06a535d224be5b683c8c692302ae06d3180045eab0e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iproute-tc-5.12.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "iproute-tc-0__5.12.0-1.el8.x86_64",
    sha256 = "eda8fcaf1354a90c8c3ab0ac23cad1ea312b8ff1995498ed19403e639d8ae693",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iproute-tc-5.12.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "iptables-0__1.8.4-19.el8.aarch64",
    sha256 = "f6ad5a8880d1ede67e240b47fa43ce77da84a37ce7119c7539b181a5f901c3c7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-1.8.4-19.el8.aarch64.rpm"],
)

rpm(
    name = "iptables-0__1.8.4-19.el8.x86_64",
    sha256 = "3adea7ef9abbeb0b4fc45f9cb429206733ac1ffd657d55ff87a3771366166702",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-1.8.4-19.el8.x86_64.rpm"],
)

rpm(
    name = "iptables-libs-0__1.8.4-19.el8.aarch64",
    sha256 = "2c9f44c42575e0605c88eaca65699fb5dcd9cf24109c94b74f8a5a848b93d96d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/iptables-libs-1.8.4-19.el8.aarch64.rpm"],
)

rpm(
    name = "iptables-libs-0__1.8.4-19.el8.x86_64",
    sha256 = "6f5911dd11991c689bb4f4586427d6af93021d1419a432dc554d0f70a027b0b5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iptables-libs-1.8.4-19.el8.x86_64.rpm"],
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
    name = "iputils-0__20180629-7.el8.x86_64",
    sha256 = "3c3d251a417e4d325b7075221d631df7411e25e3fc000528e3f2bd39a6bcc3af",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/iputils-20180629-7.el8.x86_64.rpm"],
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
    name = "ipxe-roms-qemu-0__20181214-8.git133f4c47.el8.x86_64",
    sha256 = "36d152d9372177f7418c609e71b3a3b3c683a505df85d1d1c43b1730955ff024",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/ipxe-roms-qemu-20181214-8.git133f4c47.el8.noarch.rpm"],
)

rpm(
    name = "jansson-0__2.11-3.el8.aarch64",
    sha256 = "b8bd21e036c68bb8fbb9f21e6b5f6998fc3558f55a4b902d5d85664d5929134a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/jansson-2.11-3.el8.aarch64.rpm"],
)

rpm(
    name = "jansson-0__2.11-3.el8.x86_64",
    sha256 = "a06e1d34df03aaf429d290d5c281356fefe0ad510c229189405b88b3c0f40374",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/jansson-2.11-3.el8.x86_64.rpm"],
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
    name = "json-c-0__0.13.1-2.el8.aarch64",
    sha256 = "2b9c17366280df2e2c05c9982bee55c6dd1e1774103ec6dfb2df92d73f0acf60",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/json-c-0.13.1-2.el8.aarch64.rpm"],
)

rpm(
    name = "json-c-0__0.13.1-2.el8.x86_64",
    sha256 = "3953ad29eb6ab29c845f28856d90d7c50cf252ce29654147efa6c8907b60bf28",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/json-c-0.13.1-2.el8.x86_64.rpm"],
)

rpm(
    name = "json-glib-0__1.4.4-1.el8.aarch64",
    sha256 = "01e70480bb032d5e0b60c5e732d4302d3a0ce73d1502a1729280d2b36e7e1c1a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/json-glib-1.4.4-1.el8.aarch64.rpm"],
)

rpm(
    name = "json-glib-0__1.4.4-1.el8.x86_64",
    sha256 = "98a6386df94fc9595365c3ecbc630708420fa68d1774614a723dec4a55e84b9c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/json-glib-1.4.4-1.el8.x86_64.rpm"],
)

rpm(
    name = "keyutils-libs-0__1.5.10-9.el8.aarch64",
    sha256 = "c5af4350099a98929777412fb23e74c3bd2d7d8bbd09c2969a59d45937738aad",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/keyutils-libs-1.5.10-9.el8.aarch64.rpm"],
)

rpm(
    name = "keyutils-libs-0__1.5.10-9.el8.x86_64",
    sha256 = "423329269c719b96ada88a27325e1923e764a70672e0dc6817e22eff07a9af7b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/keyutils-libs-1.5.10-9.el8.x86_64.rpm"],
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
    name = "kmod-0__25-18.el8.aarch64",
    sha256 = "22cd4d2563a814440d0c766e0153ef230d460ccb141c497f1cbd4723968832bc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-25-18.el8.aarch64.rpm"],
)

rpm(
    name = "kmod-0__25-18.el8.x86_64",
    sha256 = "d48173b5826ab4f09c3d06758266be6a9bfc992f58cdc1c1244982b71a75463c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-25-18.el8.x86_64.rpm"],
)

rpm(
    name = "kmod-libs-0__25-18.el8.aarch64",
    sha256 = "9fec275ea16aaea202613606599e262e9806ef791342a62366d7d6936bc2ec3c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/kmod-libs-25-18.el8.aarch64.rpm"],
)

rpm(
    name = "kmod-libs-0__25-18.el8.x86_64",
    sha256 = "8caf89ee7b7546fc39ebe58bf7447c9cd47ca8c4b2c0d9228de4b3087e0cb64e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/kmod-libs-25-18.el8.x86_64.rpm"],
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
    name = "krb5-libs-0__1.18.2-13.el8.aarch64",
    sha256 = "22df55141718774319ce71d6828da69f66936766538645f387d3dc41c3bdd78a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/krb5-libs-1.18.2-13.el8.aarch64.rpm"],
)

rpm(
    name = "krb5-libs-0__1.18.2-13.el8.x86_64",
    sha256 = "e48fa9191fc5799430e8fe1de413fb4ae6b2e92e227480b4972434500950e764",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/krb5-libs-1.18.2-13.el8.x86_64.rpm"],
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
    name = "less-0__530-1.el8.x86_64",
    sha256 = "f94172554b8ceeab97b560d0b05c2e2df4b2e737471adce6eca82fd3209be254",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/less-530-1.el8.x86_64.rpm"],
)

rpm(
    name = "libacl-0__2.2.53-1.el8.aarch64",
    sha256 = "c4cfed85e5a0db903ad134b4327b1714e5453fcf5c4348ec93ab344860a970ef",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libacl-2.2.53-1.el8.aarch64.rpm"],
)

rpm(
    name = "libacl-0__2.2.53-1.el8.x86_64",
    sha256 = "4973664648b7ed9278bf29074ec6a60a9f660aa97c23a283750483f64429d5bb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libacl-2.2.53-1.el8.x86_64.rpm"],
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
    name = "libaio-0__0.3.112-1.el8.aarch64",
    sha256 = "3bcb1ade26c217ead2da81c92b7ef78026c4a78383d28b6e825a7b840cae97fa",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libaio-0.3.112-1.el8.aarch64.rpm"],
)

rpm(
    name = "libaio-0__0.3.112-1.el8.x86_64",
    sha256 = "2c63399bee449fb6e921671a9bbf3356fda73f890b578820f7d926202e98a479",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libaio-0.3.112-1.el8.x86_64.rpm"],
)

rpm(
    name = "libarchive-0__3.3.3-1.el8.aarch64",
    sha256 = "e6ddc29b56fcbabe7bcd1ff1535a72c0d4477176a6321b13006d2aa65477ff9d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libarchive-3.3.3-1.el8.aarch64.rpm"],
)

rpm(
    name = "libarchive-0__3.3.3-1.el8.x86_64",
    sha256 = "57e908e16c0b5e63d0d97902e80660aa26543e0a17a5b78a41528889ea9cefb5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libarchive-3.3.3-1.el8.x86_64.rpm"],
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
    name = "libassuan-0__2.5.1-3.el8.x86_64",
    sha256 = "b49e8c674e462e3f494e825c5fca64002008cbf7a47bf131aa98b7f41678a6eb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libassuan-2.5.1-3.el8.x86_64.rpm"],
)

rpm(
    name = "libattr-0__2.4.48-3.el8.aarch64",
    sha256 = "6a6db7eab6e53dccc54116d2ddf86b02db4cff332a58b868f7ba778a99666c58",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libattr-2.4.48-3.el8.aarch64.rpm"],
)

rpm(
    name = "libattr-0__2.4.48-3.el8.x86_64",
    sha256 = "a02e1344ccde1747501ceeeff37df4f18149fb79b435aa22add08cff6bab3a5a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libattr-2.4.48-3.el8.x86_64.rpm"],
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
    name = "libblkid-0__2.32.1-28.el8.aarch64",
    sha256 = "4eb804f201b7ff9d79f5d5c82c898a8b6f0daf3e2a43e4032790868238ec9e6e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libblkid-2.32.1-28.el8.aarch64.rpm"],
)

rpm(
    name = "libblkid-0__2.32.1-28.el8.x86_64",
    sha256 = "20bcec1c3a9ca196c26749d9bb67dc6a77039f46ec078915531ae3f5205ee693",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libblkid-2.32.1-28.el8.x86_64.rpm"],
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
    name = "libburn-0__1.4.8-3.el8.aarch64",
    sha256 = "5ae88291a28b2a86efb6cdc8ff67baaf73dad1428c858c8b0fa9e8df0f0f041c",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libburn-1.4.8-3.el8.aarch64.rpm"],
)

rpm(
    name = "libburn-0__1.4.8-3.el8.x86_64",
    sha256 = "d4b0815ced6c1ec209b78fee4e2c1ee74efcd401d5462268b47d94a28ebfaf31",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libburn-1.4.8-3.el8.x86_64.rpm"],
)

rpm(
    name = "libcap-0__2.26-4.el8.aarch64",
    sha256 = "dae95e7b55eda5e7dd4cf016e129a88205de730796061e763fafda2876e8c196",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcap-2.26-4.el8.aarch64.rpm"],
)

rpm(
    name = "libcap-0__2.26-4.el8.x86_64",
    sha256 = "cfd15d82bb8e25b54c338f1eeb9e3b948edde7d73afb874e4a8a24171386fcb9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcap-2.26-4.el8.x86_64.rpm"],
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
    name = "libcap-ng-0__0.7.11-1.el8.aarch64",
    sha256 = "cbbbb1771fe9cfaa3284837e5e02cd2101190504ea0baa0278c9cfb2b169073c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcap-ng-0.7.11-1.el8.aarch64.rpm"],
)

rpm(
    name = "libcap-ng-0__0.7.11-1.el8.x86_64",
    sha256 = "15c3c696ec2e21f48e951f426d3c77b53b579605b8dd89843b35c9ab9b1d7e69",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcap-ng-0.7.11-1.el8.x86_64.rpm"],
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
    name = "libcom_err-0__1.45.6-2.el8.aarch64",
    sha256 = "adcc252cfead341c4258526cc6064d32f4a5709d3667ef15d66716e636a28783",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcom_err-1.45.6-2.el8.aarch64.rpm"],
)

rpm(
    name = "libcom_err-0__1.45.6-2.el8.x86_64",
    sha256 = "21ac150aca09ddc50c667bf369c4c4937630f959ebec5f19c62560576ca18fd3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcom_err-1.45.6-2.el8.x86_64.rpm"],
)

rpm(
    name = "libconfig-0__1.5-9.el8.x86_64",
    sha256 = "a4a2c7c0e2f454abae61dddbf4286a0b3617a8159fd20659bddbcedd8eaaa80c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libconfig-1.5-9.el8.x86_64.rpm"],
)

rpm(
    name = "libcroco-0__0.6.12-4.el8_2.1.aarch64",
    sha256 = "0022ec2580783f68e603e9d4751478c28f2b383c596b4e896469077748771bfe",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcroco-0.6.12-4.el8_2.1.aarch64.rpm"],
)

rpm(
    name = "libcroco-0__0.6.12-4.el8_2.1.x86_64",
    sha256 = "87f2a4d80cf4f6a958f3662c6a382edefc32a5ad2c364a7f3c40337cf2b1e8ba",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcroco-0.6.12-4.el8_2.1.x86_64.rpm"],
)

rpm(
    name = "libcurl-minimal-0__7.61.1-18.el8.aarch64",
    sha256 = "bbf49538863d96344ec55898f6c99dfaec0657cdd68d59ee7e4b002137305367",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libcurl-minimal-7.61.1-18.el8.aarch64.rpm"],
)

rpm(
    name = "libcurl-minimal-0__7.61.1-18.el8.x86_64",
    sha256 = "1f3fffb5d1bb3d3ec226f530c841cfe7fcaf66a87ca4b9eab5e506dbd8ce657a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libcurl-minimal-7.61.1-18.el8.x86_64.rpm"],
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
    name = "libdb-0__5.3.28-40.el8.aarch64",
    sha256 = "cab4f9caf4d9e51a7bcaa4d69e7550d5b9372ce817d956d2e5fa4e374c76a8ab",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libdb-5.3.28-40.el8.aarch64.rpm"],
)

rpm(
    name = "libdb-0__5.3.28-40.el8.x86_64",
    sha256 = "195dd3e3ba3366453faf28d3602b18a070fc3447cddd6ec45fe758490350aa0b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libdb-5.3.28-40.el8.x86_64.rpm"],
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
    name = "libdb-utils-0__5.3.28-40.el8.aarch64",
    sha256 = "47596a15abbe575d633c60d722e2bb3613d8622d6b44489957b3fca5f652b24a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libdb-utils-5.3.28-40.el8.aarch64.rpm"],
)

rpm(
    name = "libdb-utils-0__5.3.28-40.el8.x86_64",
    sha256 = "dff9459fe9602a6ae36b0f34b738c77121cb7f0a89fdce3a8a48ec78002f01c0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libdb-utils-5.3.28-40.el8.x86_64.rpm"],
)

rpm(
    name = "libevent-0__2.1.8-5.el8.aarch64",
    sha256 = "a7fed3b521d23e60539dcbd548bda2a62f0d745a99dd5feeb43b6539f7f88232",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libevent-2.1.8-5.el8.aarch64.rpm"],
)

rpm(
    name = "libevent-0__2.1.8-5.el8.x86_64",
    sha256 = "746bac6bb011a586d42bd82b2f8b25bac72c9e4bbd4c19a34cf88eadb1d83873",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libevent-2.1.8-5.el8.x86_64.rpm"],
)

rpm(
    name = "libfdisk-0__2.32.1-28.el8.aarch64",
    sha256 = "dbe365d0d44beafe99de8fa82e9789f873955e0ce1f66bebb785acca98ae3743",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libfdisk-2.32.1-28.el8.aarch64.rpm"],
)

rpm(
    name = "libfdisk-0__2.32.1-28.el8.x86_64",
    sha256 = "58cb137adc7edde3eb60b1ee8aac7de0198093621d6f37398b6023c79cd5bb06",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libfdisk-2.32.1-28.el8.x86_64.rpm"],
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
    name = "libfdt-0__1.6.0-1.el8.aarch64",
    sha256 = "a2f3c86d18ee25ce4764a1df0854c63b615db37291ef9780e649f0123a92acf5",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libfdt-1.6.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "libffi-0__3.1-22.el8.aarch64",
    sha256 = "9d7e9a47e16b3edd1f9ce69c44bf485e8498cb6ced68e354b4c24936cd015bb5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libffi-3.1-22.el8.aarch64.rpm"],
)

rpm(
    name = "libffi-0__3.1-22.el8.x86_64",
    sha256 = "3991890c6b556a06923002b0ad511c0e2d85e93cb0618758e68d72f95676b4e6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libffi-3.1-22.el8.x86_64.rpm"],
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
    name = "libgcc-0__8.5.0-3.el8.aarch64",
    sha256 = "f1f10022c95ef2ff496b3a358f6fa7f7474fecd4840ac0fac689075356a09689",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcc-8.5.0-3.el8.aarch64.rpm"],
)

rpm(
    name = "libgcc-0__8.5.0-3.el8.x86_64",
    sha256 = "59ce0d8c0aa6ed7ca399b20ffa125b096554e05127faea9da3602c215e811685",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcc-8.5.0-3.el8.x86_64.rpm"],
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
    name = "libgcrypt-0__1.8.5-6.el8.aarch64",
    sha256 = "e51932a986acc83e12f81396d532b58aacfa2b553fee84f1e62ffada1029bfd8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgcrypt-1.8.5-6.el8.aarch64.rpm"],
)

rpm(
    name = "libgcrypt-0__1.8.5-6.el8.x86_64",
    sha256 = "f53997b3c5a858b3f2c640b1a2f2fcc1ba9f698bf12ae1b6ff5097d9095caa5e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgcrypt-1.8.5-6.el8.x86_64.rpm"],
)

rpm(
    name = "libgomp-0__8.5.0-3.el8.aarch64",
    sha256 = "2d0278ec7c49b088bb7e1444a241154372d3ee560dc3d3564ffcf327c5e32c4f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgomp-8.5.0-3.el8.aarch64.rpm"],
)

rpm(
    name = "libgomp-0__8.5.0-3.el8.x86_64",
    sha256 = "f67d405acce5a03b67fa7a01726b9a697d7f2f810ae6f6d4db295f2b2a0fe7ff",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgomp-8.5.0-3.el8.x86_64.rpm"],
)

rpm(
    name = "libgpg-error-0__1.31-1.el8.aarch64",
    sha256 = "b953729a0a2be24749aeee9f00853fdc3227737971cf052a999a37ac36387cd9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libgpg-error-1.31-1.el8.aarch64.rpm"],
)

rpm(
    name = "libgpg-error-0__1.31-1.el8.x86_64",
    sha256 = "845a0732d9d7a01b909124cd8293204764235c2d856227c7a74dfa0e38113e34",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libgpg-error-1.31-1.el8.x86_64.rpm"],
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
    name = "libguestfs-1__1.44.0-3.el8s.x86_64",
    sha256 = "dd8b612fb7f8b199c989e3d15bbeb101f8cc710e7a7b9880c09ce285dfd1eaad",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libguestfs-1.44.0-3.el8s.x86_64.rpm"],
)

rpm(
    name = "libguestfs-tools-1__1.44.0-3.el8s.x86_64",
    sha256 = "de36b65d1686762617e567379f8d22bd7a5167b223aac5cfa0311cfcca7950a3",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libguestfs-tools-1.44.0-3.el8s.noarch.rpm"],
)

rpm(
    name = "libguestfs-tools-c-1__1.44.0-3.el8s.x86_64",
    sha256 = "8ae0f79d1a6b83b23c3b0d1d2bb126cadea908818d310a3c069c0c089418bd3f",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libguestfs-tools-c-1.44.0-3.el8s.x86_64.rpm"],
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
    name = "libibverbs-0__35.0-1.el8.aarch64",
    sha256 = "018bdb0811a2a05c1a3248d2a8c27120b3eefcba91677e5cd0ad56dca390a3ab",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libibverbs-35.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "libibverbs-0__35.0-1.el8.x86_64",
    sha256 = "19b78fad862eb25de4ec87c8ada45965ca90439611228f53e1e50f2b4335689e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libibverbs-35.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "libidn2-0__2.2.0-1.el8.aarch64",
    sha256 = "b62589101a60a365ef34447cae78f62e6dba560d403dc56c87036709ea00ad88",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libidn2-2.2.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "libidn2-0__2.2.0-1.el8.x86_64",
    sha256 = "7e08785bd3cc0e09f9ab4bf600b98b705203d552cbb655269a939087987f1694",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libidn2-2.2.0-1.el8.x86_64.rpm"],
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
    name = "libisoburn-0__1.4.8-4.el8.aarch64",
    sha256 = "3ff828ef16f6033227d71207bc1b00983b826172fe7c575cd7590a72d846d831",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libisoburn-1.4.8-4.el8.aarch64.rpm"],
)

rpm(
    name = "libisoburn-0__1.4.8-4.el8.x86_64",
    sha256 = "7aa030310250b462d90895d8c04ce47695722d86f5470930fdf8bfba0570c4dc",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libisoburn-1.4.8-4.el8.x86_64.rpm"],
)

rpm(
    name = "libisofs-0__1.4.8-3.el8.aarch64",
    sha256 = "2e5435efba38348be8d33a43e5abbffc85f7c5a9504ebe6451b87c44006b3b4c",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libisofs-1.4.8-3.el8.aarch64.rpm"],
)

rpm(
    name = "libisofs-0__1.4.8-3.el8.x86_64",
    sha256 = "66b7bcc256b62736f7b3d33fa65c6a91a17e08c61484a7c3748f4f86b4589bc7",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libisofs-1.4.8-3.el8.x86_64.rpm"],
)

rpm(
    name = "libksba-0__1.3.5-7.el8.x86_64",
    sha256 = "e6d3476e9996fb49632744be169f633d92900f5b7151db233501167a9018d240",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libksba-1.3.5-7.el8.x86_64.rpm"],
)

rpm(
    name = "libmetalink-0__0.1.3-7.el8.aarch64",
    sha256 = "b86423694dd6d12a0b608760046ef18f6ee97f96cb8ad661ace419a45525e200",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmetalink-0.1.3-7.el8.aarch64.rpm"],
)

rpm(
    name = "libmetalink-0__0.1.3-7.el8.x86_64",
    sha256 = "c4087dec9ffc6e6a164563c46ef09bc0c0bbb5cb992f5fbc8cd3bf20417750e1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmetalink-0.1.3-7.el8.x86_64.rpm"],
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
    name = "libmnl-0__1.0.4-6.el8.aarch64",
    sha256 = "fbe4f2cb2660ebe3cb90a73c7dfbd978059af138356e46c9a93049761c0467ef",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmnl-1.0.4-6.el8.aarch64.rpm"],
)

rpm(
    name = "libmnl-0__1.0.4-6.el8.x86_64",
    sha256 = "30fab73ee155f03dbbd99c1e30fe59dfba4ae8fdb2e7213451ccc36d6918bfcc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmnl-1.0.4-6.el8.x86_64.rpm"],
)

rpm(
    name = "libmount-0__2.32.1-28.el8.aarch64",
    sha256 = "2e0e94196aaaf205e6bda61d9379b789140d49ac3547d14fad573d012759e54d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libmount-2.32.1-28.el8.aarch64.rpm"],
)

rpm(
    name = "libmount-0__2.32.1-28.el8.x86_64",
    sha256 = "a20142c98ca558697f68d07aee33d98759c45d1307fdc86ed1553c52f5b7bb96",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libmount-2.32.1-28.el8.x86_64.rpm"],
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
    name = "libnetfilter_conntrack-0__1.0.6-5.el8.aarch64",
    sha256 = "4e43b0f85746f74064b082fdf6914ba4e9fe386651b1c39aeaecc702b2a59fc0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnetfilter_conntrack-1.0.6-5.el8.aarch64.rpm"],
)

rpm(
    name = "libnetfilter_conntrack-0__1.0.6-5.el8.x86_64",
    sha256 = "224100af3ecfc80c416796ec02c7c4dd113a38d42349d763485f3b42f260493f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnetfilter_conntrack-1.0.6-5.el8.x86_64.rpm"],
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
    name = "libnfnetlink-0__1.0.1-13.el8.aarch64",
    sha256 = "8422fbc84108abc9a89fe98cef9cd18ad1788b4dc6a9ec0bba1836b772fcaeda",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnfnetlink-1.0.1-13.el8.aarch64.rpm"],
)

rpm(
    name = "libnfnetlink-0__1.0.1-13.el8.x86_64",
    sha256 = "cec98aa5fbefcb99715921b493b4f92d34c4eeb823e9c8741aa75e280def89f1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnfnetlink-1.0.1-13.el8.x86_64.rpm"],
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
    name = "libnftnl-0__1.1.5-4.el8.aarch64",
    sha256 = "c85fbf0045e810a8a7df257799a82e32fee141db8119e9f1eb7abdb96553127f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnftnl-1.1.5-4.el8.aarch64.rpm"],
)

rpm(
    name = "libnftnl-0__1.1.5-4.el8.x86_64",
    sha256 = "c1bb77ed45ae47dc068445c6dfa4b70b273a3daf8cd82b9fa7a50e3d59abe3c1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnftnl-1.1.5-4.el8.x86_64.rpm"],
)

rpm(
    name = "libnghttp2-0__1.33.0-3.el8_2.1.aarch64",
    sha256 = "23e9ff009c2316652c3bcd96a8b69b5bc26f2acd46214f652a7ce26a572cbabb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnghttp2-1.33.0-3.el8_2.1.aarch64.rpm"],
)

rpm(
    name = "libnghttp2-0__1.33.0-3.el8_2.1.x86_64",
    sha256 = "0126a384853d46484dec98601a4cb4ce58b2e0411f8f7ef09937174dd5975bac",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnghttp2-1.33.0-3.el8_2.1.x86_64.rpm"],
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
    name = "libnl3-0__3.5.0-1.el8.aarch64",
    sha256 = "851a9cebfb68b8c301231b1121f573311fbb165ace0f4b1a599fa42f80113df9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnl3-3.5.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "libnl3-0__3.5.0-1.el8.x86_64",
    sha256 = "21c65dbf3b506a37828b13c205077f4b70fddb4b1d1c929dec01661238108059",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnl3-3.5.0-1.el8.x86_64.rpm"],
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
    name = "libnsl2-0__1.2.0-2.20180605git4a062cf.el8.aarch64",
    sha256 = "b33276781f442757afd5e066ead95ec79927f2aed608a368420f230d5ee28686",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libnsl2-1.2.0-2.20180605git4a062cf.el8.aarch64.rpm"],
)

rpm(
    name = "libnsl2-0__1.2.0-2.20180605git4a062cf.el8.x86_64",
    sha256 = "5846c73edfa2ff673989728e9621cce6a1369eb2f8a269ac5205c381a10d327a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libnsl2-1.2.0-2.20180605git4a062cf.el8.x86_64.rpm"],
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
    name = "libpcap-14__1.9.1-5.el8.aarch64",
    sha256 = "239019a8aadb26e4b015d99f7fe49e80c2d1dfa227f7c71322dca2a2a85c2de1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpcap-1.9.1-5.el8.aarch64.rpm"],
)

rpm(
    name = "libpcap-14__1.9.1-5.el8.x86_64",
    sha256 = "7f429477c26b4650a3eca4a27b3972ff0857c843bdb4d8fcb02086da111ce5fd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpcap-1.9.1-5.el8.x86_64.rpm"],
)

rpm(
    name = "libpkgconf-0__1.4.2-1.el8.aarch64",
    sha256 = "8f3e34df67e6c4a20bd7617f17d1199f0441a626fbab8059ddc6bf06c7ff4e78",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpkgconf-1.4.2-1.el8.aarch64.rpm"],
)

rpm(
    name = "libpkgconf-0__1.4.2-1.el8.x86_64",
    sha256 = "a76ff4cf270d2e38106a4bba1880c3a0899d186cd4e1986d7e97c01b934e13b7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpkgconf-1.4.2-1.el8.x86_64.rpm"],
)

rpm(
    name = "libpmem-0__1.9.2-1.module_el8.5.0__plus__756__plus__4cdc1762.x86_64",
    sha256 = "82a3a0bb6541ed6110a700488977ecf0aaf2650203356ea2f13fbc9410640706",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libpmem-1.9.2-1.module_el8.5.0+756+4cdc1762.x86_64.rpm"],
)

rpm(
    name = "libpng-2__1.6.34-5.el8.aarch64",
    sha256 = "d7bd4e7a7ff4424266c0f6030bf444de0bea88d0540ff4caf4f7f6c2bac175f6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpng-1.6.34-5.el8.aarch64.rpm"],
)

rpm(
    name = "libpng-2__1.6.34-5.el8.x86_64",
    sha256 = "cc2f054cf7ef006faf0b179701838ff8632c3ac5f45a0199a13f9c237f632b82",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpng-1.6.34-5.el8.x86_64.rpm"],
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
    name = "libpwquality-0__1.4.4-3.el8.aarch64",
    sha256 = "64e55ddddc1dd27e05097c9222e73052f6f20f9d2f7605f46922b7756adeb0b5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libpwquality-1.4.4-3.el8.aarch64.rpm"],
)

rpm(
    name = "libpwquality-0__1.4.4-3.el8.x86_64",
    sha256 = "e42ec1259c966909507a6b4c4cd25b183268d4516dd9a8d60078c8a4b6df0014",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libpwquality-1.4.4-3.el8.x86_64.rpm"],
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
    name = "librdmacm-0__35.0-1.el8.aarch64",
    sha256 = "91ba8226c6b88e23d9d9bf3f247adb6695bc2a15b940da83bab247c9cf62224e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/librdmacm-35.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "librdmacm-0__35.0-1.el8.x86_64",
    sha256 = "51927bf204955c81f0aea476df636000f208c6c225918d8516200cf7c0a9bbff",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/librdmacm-35.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "libreport-filesystem-0__2.9.5-15.el8.x86_64",
    sha256 = "b9cfde532f94e32540b51c74547da69bb06045e5d03c4e7d4be909dbcf929887",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libreport-filesystem-2.9.5-15.el8.x86_64.rpm"],
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
    name = "libseccomp-0__2.5.1-1.el8.aarch64",
    sha256 = "0e6fcdf916490d8538044bf2dc77aa67a5d7d2c51a654d5eee6dca8f69b06ba8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libseccomp-2.5.1-1.el8.aarch64.rpm"],
)

rpm(
    name = "libseccomp-0__2.5.1-1.el8.x86_64",
    sha256 = "423233a6617a132caf4d5876eb1c17d4388513d265b45025c4f061afc5588656",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libseccomp-2.5.1-1.el8.x86_64.rpm"],
)

rpm(
    name = "libselinux-0__2.9-5.el8.aarch64",
    sha256 = "9474fe348bd9e3a7a6ffe7813538e979e80ddb970b074e4e79bd122b4ece8b64",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libselinux-2.9-5.el8.aarch64.rpm"],
)

rpm(
    name = "libselinux-0__2.9-5.el8.x86_64",
    sha256 = "89e54e0975b9c87c45d3478d9f8bcc3f19a90e9ef16062a524af4a8efc059e1f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libselinux-2.9-5.el8.x86_64.rpm"],
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
    name = "libselinux-utils-0__2.9-5.el8.aarch64",
    sha256 = "e4613455147d283b222fcff5ef0f85b3a1a323893ed884db8950e51936e97c52",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libselinux-utils-2.9-5.el8.aarch64.rpm"],
)

rpm(
    name = "libselinux-utils-0__2.9-5.el8.x86_64",
    sha256 = "5063fe914f04ca203e3f28529021c40ef01ad8ed33330fafc0f658581a78b722",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libselinux-utils-2.9-5.el8.x86_64.rpm"],
)

rpm(
    name = "libsemanage-0__2.9-6.el8.aarch64",
    sha256 = "ccb929460b2e9f3fc477b5f040b8e9de1faab4492e696aac4d4eafd4d82b7ba3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsemanage-2.9-6.el8.aarch64.rpm"],
)

rpm(
    name = "libsemanage-0__2.9-6.el8.x86_64",
    sha256 = "6ba1f1f26bc8e261a813883e0cbcd7b0f542109e797fb6092afba8dc7f1ea269",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsemanage-2.9-6.el8.x86_64.rpm"],
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
    name = "libsepol-0__2.9-2.el8.aarch64",
    sha256 = "fa227d42012eb38ff357aa85387312a5a189fa143519b39d499dc9cf80896abb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsepol-2.9-2.el8.aarch64.rpm"],
)

rpm(
    name = "libsepol-0__2.9-2.el8.x86_64",
    sha256 = "6351d2d121e7a7e157a5c48086f243417327fd91b1c1f34f33aca64f741f26ee",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsepol-2.9-2.el8.x86_64.rpm"],
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
    name = "libsigsegv-0__2.11-5.el8.aarch64",
    sha256 = "b377f4e8bcdc750ed0be94f97bdbfbb12843c458fbc1d5d507f92ad04aaf592b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsigsegv-2.11-5.el8.aarch64.rpm"],
)

rpm(
    name = "libsigsegv-0__2.11-5.el8.x86_64",
    sha256 = "02d728cf74eb47005babeeab5ac68ca04472c643203a1faef0037b5f33710fe2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsigsegv-2.11-5.el8.x86_64.rpm"],
)

rpm(
    name = "libsmartcols-0__2.32.1-28.el8.aarch64",
    sha256 = "2f037740a6275018d75377a0b50a60866215f7e086f44c609835a1d08c629ce4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libsmartcols-2.32.1-28.el8.aarch64.rpm"],
)

rpm(
    name = "libsmartcols-0__2.32.1-28.el8.x86_64",
    sha256 = "da4240251c7d94968ef0b834751c789fbe49993655aa059a1533ff5f187cce6d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libsmartcols-2.32.1-28.el8.x86_64.rpm"],
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
    name = "libss-0__1.45.6-2.el8.x86_64",
    sha256 = "e66194044367e413e733c0aeebfa04ec7acaef3c330fb3f331b79976152fdf37",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libss-1.45.6-2.el8.x86_64.rpm"],
)

rpm(
    name = "libssh-0__0.9.4-3.el8.aarch64",
    sha256 = "552fdc18c6f4f1a233c808c907b43438c2059d54499b20afdb65247a7773b23f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libssh-0.9.4-3.el8.aarch64.rpm"],
)

rpm(
    name = "libssh-0__0.9.4-3.el8.x86_64",
    sha256 = "267e7ec17de7be49b118d9c717b9a4064cf620578520d9bf56c65d2a496aafa6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libssh-0.9.4-3.el8.x86_64.rpm"],
)

rpm(
    name = "libssh-config-0__0.9.4-3.el8.aarch64",
    sha256 = "f5bcb82a732c02d6f31bbf156887049883c76cedc9c6a11b049358a74f4d45d0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libssh-config-0.9.4-3.el8.noarch.rpm"],
)

rpm(
    name = "libssh-config-0__0.9.4-3.el8.x86_64",
    sha256 = "f5bcb82a732c02d6f31bbf156887049883c76cedc9c6a11b049358a74f4d45d0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libssh-config-0.9.4-3.el8.noarch.rpm"],
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
    name = "libstdc__plus____plus__-0__8.5.0-3.el8.aarch64",
    sha256 = "62b1ecd40aa76506162253dd1453f3ecd70994ae82fa86a972c2118793cb1d34",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libstdc++-8.5.0-3.el8.aarch64.rpm"],
)

rpm(
    name = "libstdc__plus____plus__-0__8.5.0-3.el8.x86_64",
    sha256 = "e204a911cf409a4da2a9c92841f2a27af38d1a249dadaff77df4bfd072345d1b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libstdc++-8.5.0-3.el8.x86_64.rpm"],
)

rpm(
    name = "libtasn1-0__4.13-3.el8.aarch64",
    sha256 = "3401ccfb7fd08c12578b6257b4dac7e94ba5f4cd70fc6a234fd90bb99d1bb108",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libtasn1-4.13-3.el8.aarch64.rpm"],
)

rpm(
    name = "libtasn1-0__4.13-3.el8.x86_64",
    sha256 = "e8d9697a8914226a2d3ed5a4523b85e8e70ac09cf90aae05395e6faee9858534",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libtasn1-4.13-3.el8.x86_64.rpm"],
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
    name = "libtirpc-0__1.1.4-5.el8.aarch64",
    sha256 = "c378aad0473ca944ce881d3d45bd76429e365216634e63213e0bdc19738d25db",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libtirpc-1.1.4-5.el8.aarch64.rpm"],
)

rpm(
    name = "libtirpc-0__1.1.4-5.el8.x86_64",
    sha256 = "71f2babdefc7c063cce7541f3f132d3fed6f5a1df94f360850d4dc3d95a7bf28",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libtirpc-1.1.4-5.el8.x86_64.rpm"],
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
    name = "libtpms-0__0.7.4-5.20201106git2452a24dab.el8s.aarch64",
    sha256 = "f42b42d370db82e6c7e2bdad4a7557e1ae9f0653f460c1a275f6d9b6734ad16f",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libtpms-0.7.4-5.20201106git2452a24dab.el8s.aarch64.rpm"],
)

rpm(
    name = "libtpms-0__0.7.4-5.20201106git2452a24dab.el8s.x86_64",
    sha256 = "7d0468e142b380529bb0b61c8242b9eb49981a5cec86ffa75a1f13d3337eb6aa",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libtpms-0.7.4-5.20201106git2452a24dab.el8s.x86_64.rpm"],
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
    name = "libunistring-0__0.9.9-3.el8.aarch64",
    sha256 = "707429ccb3223628d55097a162cd0d3de1bd00b48800677c1099931b0f019e80",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libunistring-0.9.9-3.el8.aarch64.rpm"],
)

rpm(
    name = "libunistring-0__0.9.9-3.el8.x86_64",
    sha256 = "20bb189228afa589141d9c9d4ed457729d13c11608305387602d0b00ed0a3093",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libunistring-0.9.9-3.el8.x86_64.rpm"],
)

rpm(
    name = "libunwind-0__1.3.1-7.fc32.aarch64",
    sha256 = "4b4e158b08f02a7e4c3138e02267e231d64394e17598103620a73164c65a40be",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libunwind-1.3.1-7.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/l/libunwind-1.3.1-7.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/l/libunwind-1.3.1-7.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/l/libunwind-1.3.1-7.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/4b4e158b08f02a7e4c3138e02267e231d64394e17598103620a73164c65a40be",
    ],
)

rpm(
    name = "libunwind-0__1.3.1-7.fc32.x86_64",
    sha256 = "b5e581f7a60b4b4164b700bf3ba47c6de1fb74ef6102687c418c56b29b861e34",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/l/libunwind-1.3.1-7.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b5e581f7a60b4b4164b700bf3ba47c6de1fb74ef6102687c418c56b29b861e34",
    ],
)

rpm(
    name = "libusal-0__1.1.11-39.el8.x86_64",
    sha256 = "0b2b79d9f8cd01090816386ad89852662b5489bbd43fbd04760f0e57c28bce4c",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libusal-1.1.11-39.el8.x86_64.rpm"],
)

rpm(
    name = "libusbx-0__1.0.23-4.el8.aarch64",
    sha256 = "ae797d004f3cafb89773fcc8a3f0d6d046546b7cb3f9741be200d095c637706f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libusbx-1.0.23-4.el8.aarch64.rpm"],
)

rpm(
    name = "libusbx-0__1.0.23-4.el8.x86_64",
    sha256 = "7e704756a93f07feec345a9748204e78994ce06a4667a2ef35b44964ff754306",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libusbx-1.0.23-4.el8.x86_64.rpm"],
)

rpm(
    name = "libutempter-0__1.1.6-14.el8.aarch64",
    sha256 = "8f6d9839a758fdacfdb4b4b0731e8023b8bbb0b633bd32dbf21c2ce85a933a8a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libutempter-1.1.6-14.el8.aarch64.rpm"],
)

rpm(
    name = "libutempter-0__1.1.6-14.el8.x86_64",
    sha256 = "c8c54c56bff9ca416c3ba6bccac483fb66c81a53d93a19420088715018ed5169",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libutempter-1.1.6-14.el8.x86_64.rpm"],
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
    name = "libuuid-0__2.32.1-28.el8.aarch64",
    sha256 = "3976c3648ef9503a771a8f2466bb854b68c1569d363018db9cc63e097ecff41b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libuuid-2.32.1-28.el8.aarch64.rpm"],
)

rpm(
    name = "libuuid-0__2.32.1-28.el8.x86_64",
    sha256 = "4338967a50af54b0392b45193be52959157e67122eef9ebf12aa2b0b3661b16c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libuuid-2.32.1-28.el8.x86_64.rpm"],
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
    name = "libverto-0__0.3.0-5.el8.aarch64",
    sha256 = "446f45706d78e80d4057d9d55dda32ce1cb823b2ca4dfe50f0ca5b515238130d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libverto-0.3.0-5.el8.aarch64.rpm"],
)

rpm(
    name = "libverto-0__0.3.0-5.el8.x86_64",
    sha256 = "f95f673fc9236dc712270a343807cdac06297d847001e78cd707482c751b2d0d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libverto-0.3.0-5.el8.x86_64.rpm"],
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
    name = "libvirt-bash-completion-0__7.0.0-14.el8s.aarch64",
    sha256 = "90074b171db001f563921d5d75be6c223069a560d6cddd6258dd8c39a7eac855",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-bash-completion-7.0.0-14.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-bash-completion-0__7.0.0-14.el8s.x86_64",
    sha256 = "ca4d8d1a5e367599212af381310f99d7569c3aa1b164874b32d068196f4622a8",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-bash-completion-7.0.0-14.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-client-0__7.0.0-14.el8s.aarch64",
    sha256 = "f363e3f0e4a8b046c4541ba569af996dcb72482482a49c7085cb0adfd5a9183c",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-client-7.0.0-14.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-client-0__7.0.0-14.el8s.x86_64",
    sha256 = "b3df1f07e3ea150c5869efee57e07a9ea1d1036971c68cae19d5b65f0bed3026",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-client-7.0.0-14.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-0__7.0.0-14.el8s.aarch64",
    sha256 = "232b530b6eca6700a15172b828a7a262ee07ed09bfce34d992f78f9024d25527",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-daemon-7.0.0-14.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-daemon-0__7.0.0-14.el8s.x86_64",
    sha256 = "0ced3add2c837801da22748d50f8de7335015c2c9811adb24f3a378e74865d45",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-7.0.0-14.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.0.0-14.el8s.aarch64",
    sha256 = "baa48a22babba1e13e00519a4e89b2ccb5db8843c029c17c266423ecb467f831",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-daemon-driver-qemu-7.0.0-14.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-daemon-driver-qemu-0__7.0.0-14.el8s.x86_64",
    sha256 = "4654d2deed80c40f943584ddfc2bd60597158fe703af71d30aecefa8d51d8f0a",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-driver-qemu-7.0.0-14.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-daemon-kvm-0__7.5.0-1.el8s.x86_64",
    sha256 = "f7c2dc57b81688e0ff82b41375e72f4c41add69055aeb64ac5eb0ee9cb6d8864",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-daemon-kvm-7.5.0-1.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-devel-0__7.0.0-14.el8s.aarch64",
    sha256 = "9e5181ed299dbcdb014060e1215378db71c7e6f260c88905e09066196e6241bc",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-devel-7.0.0-14.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-devel-0__7.0.0-14.el8s.x86_64",
    sha256 = "9620b55b56e43cfef0109cd5b46b34b0a5fbd009ec7fcade69a32de1d4aa7c19",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-devel-7.0.0-14.el8s.x86_64.rpm"],
)

rpm(
    name = "libvirt-libs-0__7.0.0-14.el8s.aarch64",
    sha256 = "15af6a089ac472c0f5f6cafc3bf5ff207758f742c99d0f17d6526da7d282a434",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/l/libvirt-libs-7.0.0-14.el8s.aarch64.rpm"],
)

rpm(
    name = "libvirt-libs-0__7.0.0-14.el8s.x86_64",
    sha256 = "98ccca2680f66c863843b035b6103b1a2419f2529976bfa3e780b815493c04a6",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/l/libvirt-libs-7.0.0-14.el8s.x86_64.rpm"],
)

rpm(
    name = "libxcrypt-0__4.1.1-6.el8.aarch64",
    sha256 = "4948420ee35381c71c619fab4b8deabfa93c04e7c5729620b02e4382a50550ad",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libxcrypt-4.1.1-6.el8.aarch64.rpm"],
)

rpm(
    name = "libxcrypt-0__4.1.1-6.el8.x86_64",
    sha256 = "645853feb85c921d979cb9cf9109663528429eda63cf5a1e31fe578d3d7e713a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libxcrypt-4.1.1-6.el8.x86_64.rpm"],
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
    name = "libxkbcommon-0__0.9.1-1.el8.aarch64",
    sha256 = "3aca03c788af2ecf8ef39421f246769d7ef7f37260ee9421fc68c1d1cc913600",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/libxkbcommon-0.9.1-1.el8.aarch64.rpm"],
)

rpm(
    name = "libxkbcommon-0__0.9.1-1.el8.x86_64",
    sha256 = "e03d462995326a4477dcebc8c12eae3c1776ce2f095617ace253c0c492c89082",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/libxkbcommon-0.9.1-1.el8.x86_64.rpm"],
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
    name = "libxml2-0__2.9.7-11.el8.aarch64",
    sha256 = "3514c1fa9f0ff57538e74e9b66991e4911e5176e250d49cd6fe079d4a9a3ba04",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libxml2-2.9.7-11.el8.aarch64.rpm"],
)

rpm(
    name = "libxml2-0__2.9.7-11.el8.x86_64",
    sha256 = "d13a830e42506b9ada2b719521e020e4857bc49aacef3c9a66368485690443da",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libxml2-2.9.7-11.el8.x86_64.rpm"],
)

rpm(
    name = "libzstd-0__1.4.4-1.el8.aarch64",
    sha256 = "b560a8a185100a7c80e6c32f69ba65ce17004156f7218cf183249b15c13295cc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/libzstd-1.4.4-1.el8.aarch64.rpm"],
)

rpm(
    name = "libzstd-0__1.4.4-1.el8.x86_64",
    sha256 = "7c2dc6044f13fe4ae04a4c1620da822a6be591b5129bf68ba98a3d8e9092f83b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/libzstd-1.4.4-1.el8.x86_64.rpm"],
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
    name = "lsof-0__4.93.2-3.fc32.aarch64",
    sha256 = "fb5b3b970d2f0b638d46cb8a0b283369a07a0c653a3d11fca4e0f1ba3be56b0f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lsof-4.93.2-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/l/lsof-4.93.2-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lsof-4.93.2-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/l/lsof-4.93.2-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fb5b3b970d2f0b638d46cb8a0b283369a07a0c653a3d11fca4e0f1ba3be56b0f",
    ],
)

rpm(
    name = "lsof-0__4.93.2-3.fc32.x86_64",
    sha256 = "465b7317f0a979c92d76713fbe61761ee3e2afd1a59c01c6d3c6323767e1a115",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/l/lsof-4.93.2-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/465b7317f0a979c92d76713fbe61761ee3e2afd1a59c01c6d3c6323767e1a115",
    ],
)

rpm(
    name = "lsscsi-0__0.32-2.el8.x86_64",
    sha256 = "c8fc05dd997477fc80e2f3195e719ca2e569fc8997f05a64a13c52c8358e8239",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lsscsi-0.32-2.el8.x86_64.rpm"],
)

rpm(
    name = "lua-libs-0__5.3.4-11.el8.aarch64",
    sha256 = "914f1d8cf5385ec874ac88b00f5ae99e77be48aa6c7157a2e0c1c5355c415c94",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lua-libs-5.3.4-11.el8.aarch64.rpm"],
)

rpm(
    name = "lua-libs-0__5.3.4-11.el8.x86_64",
    sha256 = "98a5f610c2ca116fa63f98302036eaa8ca725c1e8fd7afae4a285deb50605b35",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lua-libs-5.3.4-11.el8.x86_64.rpm"],
)

rpm(
    name = "lvm2-8__2.03.12-3.el8.x86_64",
    sha256 = "154bef16f948bae471bb1de3df840aca2c899dc56a26ed6dac93d225205bc985",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-2.03.12-3.el8.x86_64.rpm"],
)

rpm(
    name = "lvm2-libs-8__2.03.12-3.el8.x86_64",
    sha256 = "3abeff185da917c9378d4e3d0aceaf0a7ae5176802671d148ea980c541db8cdf",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lvm2-libs-2.03.12-3.el8.x86_64.rpm"],
)

rpm(
    name = "lz4-libs-0__1.8.3-3.el8.aarch64",
    sha256 = "1f757439c1aeafd3b42eac8962ea6672190fb5cb93a4f658220d31e21db4ce5b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lz4-libs-1.8.3-3.el8.aarch64.rpm"],
)

rpm(
    name = "lz4-libs-0__1.8.3-3.el8.x86_64",
    sha256 = "703da187965ae1fc8af40c271e1464acafb9f1181c778d33c1da65b4622dc4bd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lz4-libs-1.8.3-3.el8.x86_64.rpm"],
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
    name = "lzo-0__2.08-14.el8.aarch64",
    sha256 = "6809839757bd05082ca1b8d23eac617898eda3ce34844a0d31b0a030c8cc6653",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lzo-2.08-14.el8.aarch64.rpm"],
)

rpm(
    name = "lzo-0__2.08-14.el8.x86_64",
    sha256 = "5c68635cb03533a38d4a42f6547c21a1d5f9952351bb01f3cf865d2621a6e634",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lzo-2.08-14.el8.x86_64.rpm"],
)

rpm(
    name = "lzop-0__1.03-20.el8.aarch64",
    sha256 = "003b309833a1ed94ad97ed62f04c2fcda4a20fb8b7b5933c36459974f4e4986c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/lzop-1.03-20.el8.aarch64.rpm"],
)

rpm(
    name = "lzop-0__1.03-20.el8.x86_64",
    sha256 = "04eae61018a5be7656be832797016f97cd7b6e19d56f58cb658cd3969dedf2b0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/lzop-1.03-20.el8.x86_64.rpm"],
)

rpm(
    name = "mdadm-0__4.2-rc1_3.el8.x86_64",
    sha256 = "233cc6811ba790b7f31dd712d7f88e6c8f3b8c38d4df6744e50edc6064788841",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mdadm-4.2-rc1_3.el8.x86_64.rpm"],
)

rpm(
    name = "mozjs60-0__60.9.0-4.el8.aarch64",
    sha256 = "8a1da341e022af37e9861bb2e8f2b045ad0b36cd783547c0dee08b8097e73c80",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/mozjs60-60.9.0-4.el8.aarch64.rpm"],
)

rpm(
    name = "mozjs60-0__60.9.0-4.el8.x86_64",
    sha256 = "03b50a4ea5cf5655c67e2358fabb6e563eec4e7929e7fc6c4e92c92694f60fa0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mozjs60-60.9.0-4.el8.x86_64.rpm"],
)

rpm(
    name = "mpfr-0__3.1.6-1.el8.aarch64",
    sha256 = "97a998a1b93c21bf070f9a9a1dbb525234b00fccedfe67de8967cd9ec7132eb1",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/mpfr-3.1.6-1.el8.aarch64.rpm"],
)

rpm(
    name = "mpfr-0__3.1.6-1.el8.x86_64",
    sha256 = "e7f0c34f83c1ec2abb22951779e84d51e234c4ba0a05252e4ffd8917461891a5",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mpfr-3.1.6-1.el8.x86_64.rpm"],
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
    name = "mtools-0__4.0.18-14.el8.x86_64",
    sha256 = "f726efa5063fdb4b0bff847b20087a3286f9c069ce62f75561a6d1adee0dad5a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/mtools-4.0.18-14.el8.x86_64.rpm"],
)

rpm(
    name = "ncurses-0__6.1-15.20191109.fc32.aarch64",
    sha256 = "fe7ee39b0779c467c5d8a20daff4911e1967523e6fc748179e77584168e18bde",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-6.1-15.20191109.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/ncurses-6.1-15.20191109.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-6.1-15.20191109.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/ncurses-6.1-15.20191109.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/fe7ee39b0779c467c5d8a20daff4911e1967523e6fc748179e77584168e18bde",
    ],
)

rpm(
    name = "ncurses-0__6.1-15.20191109.fc32.x86_64",
    sha256 = "b2e862283ac97b1d8b1ede2034ead452ac7dc4ff308593306275b1b0ae5b4102",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/ncurses-6.1-15.20191109.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/b2e862283ac97b1d8b1ede2034ead452ac7dc4ff308593306275b1b0ae5b4102",
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
    name = "ncurses-base-0__6.1-9.20180224.el8.aarch64",
    sha256 = "41716536ea16798238ac89fbc3041b3f9dc80f9a64ea4b19d6e67ad2c909269a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ncurses-base-6.1-9.20180224.el8.noarch.rpm"],
)

rpm(
    name = "ncurses-base-0__6.1-9.20180224.el8.x86_64",
    sha256 = "41716536ea16798238ac89fbc3041b3f9dc80f9a64ea4b19d6e67ad2c909269a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ncurses-base-6.1-9.20180224.el8.noarch.rpm"],
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
    name = "ncurses-libs-0__6.1-9.20180224.el8.aarch64",
    sha256 = "b938a6facc8d8a3de12b369871738bb531c822b1ec5212501b06bcaaf6cd25fa",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/ncurses-libs-6.1-9.20180224.el8.aarch64.rpm"],
)

rpm(
    name = "ncurses-libs-0__6.1-9.20180224.el8.x86_64",
    sha256 = "54609dd070a57a14a6103f0c06bea99bb0a4e568d1fbc6a22b8ba67c954d90bf",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ncurses-libs-6.1-9.20180224.el8.x86_64.rpm"],
)

rpm(
    name = "ndctl-libs-0__71.1-2.el8.x86_64",
    sha256 = "7e8fa8aa5971b39e329905feb378545d8dad32a46e4d25e9a9daf5eb19e6c593",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/ndctl-libs-71.1-2.el8.x86_64.rpm"],
)

rpm(
    name = "nettle-0__3.4.1-7.el8.aarch64",
    sha256 = "5441222132ae52cd31063e9b9e3bb40f2e5711dfb0c84315b4aec2907278a075",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/nettle-3.4.1-7.el8.aarch64.rpm"],
)

rpm(
    name = "nettle-0__3.4.1-7.el8.x86_64",
    sha256 = "fe9a848502c595e0b7acc699d69c24b9c5ad0ac58a0b3933cd228f3633de31cb",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/nettle-3.4.1-7.el8.x86_64.rpm"],
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
    name = "nftables-1__0.9.3-20.el8.aarch64",
    sha256 = "21234c37dd57e15b357c5f0e506f25249e89ed6a1b78a3e50c386fa871ff7db4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/nftables-0.9.3-20.el8.aarch64.rpm"],
)

rpm(
    name = "nftables-1__0.9.3-20.el8.x86_64",
    sha256 = "1f24dd4dfd57b18c7d9148df6d564115d17e2be99119f164d5e3f887b83d6b14",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/nftables-0.9.3-20.el8.x86_64.rpm"],
)

rpm(
    name = "nginx-1__1.20.0-2.fc32.aarch64",
    sha256 = "dc5d325af1c543017603d7359057578793b755b444d6babce3214e3290e6a12b",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-1.20.0-2.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-1.20.0-2.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-1.20.0-2.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-1.20.0-2.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/dc5d325af1c543017603d7359057578793b755b444d6babce3214e3290e6a12b",
    ],
)

rpm(
    name = "nginx-1__1.20.0-2.fc32.x86_64",
    sha256 = "e372048f4c81be8f67d1d93b92a4fec4e8a76786336c0c022472d92f21d6474d",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.20.0-2.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.20.0-2.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.20.0-2.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-1.20.0-2.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e372048f4c81be8f67d1d93b92a4fec4e8a76786336c0c022472d92f21d6474d",
    ],
)

rpm(
    name = "nginx-filesystem-1__1.20.0-2.fc32.aarch64",
    sha256 = "5681c48f58996de37de65fd46b0f90146e33ffbba039b708d55a770196cc7028",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5681c48f58996de37de65fd46b0f90146e33ffbba039b708d55a770196cc7028",
    ],
)

rpm(
    name = "nginx-filesystem-1__1.20.0-2.fc32.x86_64",
    sha256 = "5681c48f58996de37de65fd46b0f90146e33ffbba039b708d55a770196cc7028",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/n/nginx-filesystem-1.20.0-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5681c48f58996de37de65fd46b0f90146e33ffbba039b708d55a770196cc7028",
    ],
)

rpm(
    name = "nginx-mimetypes-0__2.1.48-7.fc32.aarch64",
    sha256 = "657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    ],
)

rpm(
    name = "nginx-mimetypes-0__2.1.48-7.fc32.x86_64",
    sha256 = "657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/n/nginx-mimetypes-2.1.48-7.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/657909c0fc6fdf24f105a2579ea3a2fe17a73969339880809cc46dd6ff8d8773",
    ],
)

rpm(
    name = "nmap-ncat-2__7.70-5.el8.aarch64",
    sha256 = "8a71ce754f45c62be372e88350ef338d30bf3aa97c766a3d241f58ec12520c2d",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/nmap-ncat-7.70-5.el8.aarch64.rpm"],
)

rpm(
    name = "nmap-ncat-2__7.70-5.el8.x86_64",
    sha256 = "114353eb1f2c97125230c40385039824d83032672e64629f9c589393a361dfed",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/nmap-ncat-7.70-5.el8.x86_64.rpm"],
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
    name = "npth-0__1.5-4.el8.x86_64",
    sha256 = "168ab5dbc86b836b8742b2e63eee51d074f1d790728e3d30b0c59fff93cf1d8d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/npth-1.5-4.el8.x86_64.rpm"],
)

rpm(
    name = "numactl-libs-0__2.0.12-13.el8.aarch64",
    sha256 = "5f2d7a8db99ad318df35e60d43e5e7f462294c00ffa3d7c24207c16bfd3a6619",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/numactl-libs-2.0.12-13.el8.aarch64.rpm"],
)

rpm(
    name = "numactl-libs-0__2.0.12-13.el8.x86_64",
    sha256 = "b7b71ba34b3af893dc0acbb9d2228a2307da849d38e1c0007bd3d64f456640af",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/numactl-libs-2.0.12-13.el8.x86_64.rpm"],
)

rpm(
    name = "numad-0__0.5-26.20150602git.el8.aarch64",
    sha256 = "5b580f1a1c2193384a7c4c5171200d1e6f4ca6a19e6a01a327a75d03db916484",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/numad-0.5-26.20150602git.el8.aarch64.rpm"],
)

rpm(
    name = "numad-0__0.5-26.20150602git.el8.x86_64",
    sha256 = "5d975c08273b1629683275c32f16e52ca8e37e6836598e211092c915d38878bf",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/numad-0.5-26.20150602git.el8.x86_64.rpm"],
)

rpm(
    name = "openldap-0__2.4.46-17.el8_4.aarch64",
    sha256 = "2d3343015e291d718b75dc1f395e526429835685fe98f62d4f0d64bc99f830a6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/openldap-2.4.46-17.el8_4.aarch64.rpm"],
)

rpm(
    name = "openldap-0__2.4.46-17.el8_4.x86_64",
    sha256 = "94cadb1bf09facc08048e362726f5ea996aaf9e17fcf2a0b87eeb6ae38ac3ce7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/openldap-2.4.46-17.el8_4.x86_64.rpm"],
)

rpm(
    name = "openssl-1__1.1.1k-1.fc32.aarch64",
    sha256 = "d47d2d349de423f03fd30a33cf7f022dab2fa03ea786705686a1c52070cb411f",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/o/openssl-1.1.1k-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d47d2d349de423f03fd30a33cf7f022dab2fa03ea786705686a1c52070cb411f",
    ],
)

rpm(
    name = "openssl-1__1.1.1k-1.fc32.x86_64",
    sha256 = "06790d926d76c4b3f5e8dcfd4ae836d6b1d0cc5e19f8a306adb50c245ced17a4",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/o/openssl-1.1.1k-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/06790d926d76c4b3f5e8dcfd4ae836d6b1d0cc5e19f8a306adb50c245ced17a4",
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
    name = "openssl-libs-1__1.1.1k-4.el8.aarch64",
    sha256 = "032e8c0576f2743234369ed3a9d682e1b4467e27587a43fd427d2b5b5949e08a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/openssl-libs-1.1.1k-4.el8.aarch64.rpm"],
)

rpm(
    name = "openssl-libs-1__1.1.1k-4.el8.x86_64",
    sha256 = "7bf8f1f49b71019e960f5d5e93f8b0a069ec2743b05ed6e1344bdb4af2dade39",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/openssl-libs-1.1.1k-4.el8.x86_64.rpm"],
)

rpm(
    name = "p11-kit-0__0.23.22-1.el8.aarch64",
    sha256 = "cfee10a5ca5613896a4e84716aa393094fd97c09f2c585c9aa921e6063783867",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/p11-kit-0.23.22-1.el8.aarch64.rpm"],
)

rpm(
    name = "p11-kit-0__0.23.22-1.el8.x86_64",
    sha256 = "6a67c8721fe24af25ec56c6aae956a190d8463e46efed45adfbbd800086550c7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/p11-kit-0.23.22-1.el8.x86_64.rpm"],
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
    name = "p11-kit-trust-0__0.23.22-1.el8.aarch64",
    sha256 = "3fc181bf0f076fef283fdb63d36e7b84930c8822fa67dff6e1ccea9987d6dbf3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/p11-kit-trust-0.23.22-1.el8.aarch64.rpm"],
)

rpm(
    name = "p11-kit-trust-0__0.23.22-1.el8.x86_64",
    sha256 = "d218619a4859e002fe677703bc1767986314cd196ae2ac397ed057f3bec36516",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/p11-kit-trust-0.23.22-1.el8.x86_64.rpm"],
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
    name = "pam-0__1.3.1-15.el8.aarch64",
    sha256 = "a33349c435ef9b8348864e5b8f09ed050d0b7a79fb2db5a88b2f7a5d869231d7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pam-1.3.1-15.el8.aarch64.rpm"],
)

rpm(
    name = "pam-0__1.3.1-15.el8.x86_64",
    sha256 = "a0096af833462f915fe6474f7f85324992f641ea7ebeccca1f666815a4afad19",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pam-1.3.1-15.el8.x86_64.rpm"],
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
    name = "parted-0__3.2-39.el8.x86_64",
    sha256 = "2a9f8558c6c640d8f035004f3a9e607f6941e028785da562f01b61a142b5e282",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/parted-3.2-39.el8.x86_64.rpm"],
)

rpm(
    name = "pciutils-0__3.7.0-1.el8.aarch64",
    sha256 = "8337a6e98b7ae82d5263e08524381a5e396fd7cafca6f7195753286c0c082a04",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pciutils-3.7.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "pciutils-0__3.7.0-1.el8.x86_64",
    sha256 = "4d563d8048653b88a65b3b507e95ff911b39471935870684c0d6ead054a9a90e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pciutils-3.7.0-1.el8.x86_64.rpm"],
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
    name = "pciutils-libs-0__3.7.0-1.el8.aarch64",
    sha256 = "ae037b9b513dd2ce6b4ecce6255a8fddf94367bc9c348ba45c5aefbfeff29201",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pciutils-libs-3.7.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "pciutils-libs-0__3.7.0-1.el8.x86_64",
    sha256 = "4c4c1acb49227d6a71b7e7ae490d0f5f33031a4643e374b2e0b6c7d53045873c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pciutils-libs-3.7.0-1.el8.x86_64.rpm"],
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
    name = "pcre-0__8.42-6.el8.aarch64",
    sha256 = "5591faa4f51dc97067292938883b771d75ec2b3a749ec956eddc0408e689c369",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pcre-8.42-6.el8.aarch64.rpm"],
)

rpm(
    name = "pcre-0__8.42-6.el8.x86_64",
    sha256 = "876e9e99b0e50cb2752499045bafa903dd29e5c491d112daacef1ae16f614dad",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pcre-8.42-6.el8.x86_64.rpm"],
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
    name = "pcre2-0__10.32-2.el8.aarch64",
    sha256 = "3a386eca4550def1fef05213ddc8fe082e589a2fe2898f634265fbe8fe828296",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pcre2-10.32-2.el8.aarch64.rpm"],
)

rpm(
    name = "pcre2-0__10.32-2.el8.x86_64",
    sha256 = "fb29d2bd46a98affd617bbb243bb117ebbb3d074a6455036abb2aa5b507cce62",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pcre2-10.32-2.el8.x86_64.rpm"],
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
    name = "perl-Carp-0__1.50-440.fc32.aarch64",
    sha256 = "79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    ],
)

rpm(
    name = "perl-Carp-0__1.50-440.fc32.x86_64",
    sha256 = "79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Carp-1.50-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/79a464d82928b693b59dd775db69f8641abe211331514f304c8157e002ccd2c7",
    ],
)

rpm(
    name = "perl-Config-General-0__2.63-11.fc32.aarch64",
    sha256 = "9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    ],
)

rpm(
    name = "perl-Config-General-0__2.63-11.fc32.x86_64",
    sha256 = "9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Config-General-2.63-11.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9dcb140fe281a4d1d75033d9a933f9ac828ae1055de4c37c0aff24b42c512b66",
    ],
)

rpm(
    name = "perl-Encode-4__3.08-458.fc32.aarch64",
    sha256 = "facd41ab7e467f9b3567fd2660a2482f996ef2583de0b18c5ff8555250879f79",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Encode-3.08-458.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/facd41ab7e467f9b3567fd2660a2482f996ef2583de0b18c5ff8555250879f79",
    ],
)

rpm(
    name = "perl-Encode-4__3.08-458.fc32.x86_64",
    sha256 = "3443414bc9203145a26290ab9aecfc04dc2c272647411db03d09194f8ff69277",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Encode-3.08-458.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/3443414bc9203145a26290ab9aecfc04dc2c272647411db03d09194f8ff69277",
    ],
)

rpm(
    name = "perl-Errno-0__1.30-461.fc32.aarch64",
    sha256 = "7dcbacfb6352cbc0a18dd2b5c191f15841cadaf35bac8c5399bda6b29d1f3a75",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Errno-1.30-461.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Errno-1.30-461.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Errno-1.30-461.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Errno-1.30-461.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7dcbacfb6352cbc0a18dd2b5c191f15841cadaf35bac8c5399bda6b29d1f3a75",
    ],
)

rpm(
    name = "perl-Errno-0__1.30-461.fc32.x86_64",
    sha256 = "10e4932b74652ee184e9e9e684dc068a66c014b92a187d0aee40f1bdcb3347c6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-461.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-461.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-461.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Errno-1.30-461.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/10e4932b74652ee184e9e9e684dc068a66c014b92a187d0aee40f1bdcb3347c6",
    ],
)

rpm(
    name = "perl-Exporter-0__5.74-2.fc32.aarch64",
    sha256 = "9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    ],
)

rpm(
    name = "perl-Exporter-0__5.74-2.fc32.x86_64",
    sha256 = "9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Exporter-5.74-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/9d696e62b86d7a2ed5d7cb6c9484d4669955300d1b96f7a723f6f27aefdddb09",
    ],
)

rpm(
    name = "perl-File-Path-0__2.17-1.fc32.aarch64",
    sha256 = "0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    ],
)

rpm(
    name = "perl-File-Path-0__2.17-1.fc32.x86_64",
    sha256 = "0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-File-Path-2.17-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/0595b0078ddd6ff7caaf66db7f2c989b312eaa28bbb668b69e50a1c98f6d7454",
    ],
)

rpm(
    name = "perl-File-Temp-1__0.230.900-440.fc32.aarch64",
    sha256 = "006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    ],
)

rpm(
    name = "perl-File-Temp-1__0.230.900-440.fc32.x86_64",
    sha256 = "006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-File-Temp-0.230.900-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/006d36c836aa26fb2378465832d6579e61ce54ced4bc24817a463c6eb3b45f4b",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.52-1.fc32.aarch64",
    sha256 = "4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    ],
)

rpm(
    name = "perl-Getopt-Long-1__2.52-1.fc32.x86_64",
    sha256 = "4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Getopt-Long-2.52-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4ab8567b18b8349a60177413e87485cd5d630f8012fee4616420203c1d600e68",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.076-440.fc32.aarch64",
    sha256 = "af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    ],
)

rpm(
    name = "perl-HTTP-Tiny-0__0.076-440.fc32.x86_64",
    sha256 = "af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-HTTP-Tiny-0.076-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/af3ca7b72d7ebaaaad37b76e922ab7d542448d77ff73cb912e40cddc7fa506dc",
    ],
)

rpm(
    name = "perl-IO-0__1.40-461.fc32.aarch64",
    sha256 = "7968b16d624daa32c90e7dd024fddd2daf754d69e1b004599a498e5dc72bfb92",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-IO-1.40-461.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-IO-1.40-461.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-IO-1.40-461.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-IO-1.40-461.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/7968b16d624daa32c90e7dd024fddd2daf754d69e1b004599a498e5dc72bfb92",
    ],
)

rpm(
    name = "perl-IO-0__1.40-461.fc32.x86_64",
    sha256 = "0b1f8f7f1313632e72a951623863e752e1222b5f818884d7c653b19607dd9048",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-461.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-461.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-461.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-IO-1.40-461.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/0b1f8f7f1313632e72a951623863e752e1222b5f818884d7c653b19607dd9048",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.15-440.fc32.aarch64",
    sha256 = "d7c72e6ef23dbf1ff77a7f9a2d9bb368b0783b4e607fcaa17d04885077b94f2d",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/d7c72e6ef23dbf1ff77a7f9a2d9bb368b0783b4e607fcaa17d04885077b94f2d",
    ],
)

rpm(
    name = "perl-MIME-Base64-0__3.15-440.fc32.x86_64",
    sha256 = "86695db247813a6aec340c481e41b747deb588a3abec1528213087d84f99d430",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-MIME-Base64-3.15-440.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/86695db247813a6aec340c481e41b747deb588a3abec1528213087d84f99d430",
    ],
)

rpm(
    name = "perl-PathTools-0__3.78-442.fc32.aarch64",
    sha256 = "35af5d1d22c9a36f4b0465b4a31f5ddde5d09d6907e63f8d3e3441f0108d5791",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-PathTools-3.78-442.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-PathTools-3.78-442.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-PathTools-3.78-442.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-PathTools-3.78-442.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/35af5d1d22c9a36f4b0465b4a31f5ddde5d09d6907e63f8d3e3441f0108d5791",
    ],
)

rpm(
    name = "perl-PathTools-0__3.78-442.fc32.x86_64",
    sha256 = "79ac869bf8d4d4c322134d6b256faacd46476e3ede94d2a9ccf8b289e450d771",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-PathTools-3.78-442.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/79ac869bf8d4d4c322134d6b256faacd46476e3ede94d2a9ccf8b289e450d771",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-440.fc32.aarch64",
    sha256 = "32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    ],
)

rpm(
    name = "perl-Pod-Escapes-1__1.07-440.fc32.x86_64",
    sha256 = "32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Escapes-1.07-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/32a7608e47ecc6069c70dae86b4ad808850ce97b715f01806e87b2a7d3317a3c",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28.01-443.fc32.aarch64",
    sha256 = "03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    ],
)

rpm(
    name = "perl-Pod-Perldoc-0__3.28.01-443.fc32.x86_64",
    sha256 = "03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Perldoc-3.28.01-443.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/03e5fcaec5c3f2c180dc803b0aa5bba31af8fa3f59e1822d1d5a82b3e67da44a",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.40-2.fc32.aarch64",
    sha256 = "c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    ],
)

rpm(
    name = "perl-Pod-Simple-1__3.40-2.fc32.x86_64",
    sha256 = "c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Pod-Simple-3.40-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/c87dfbe6e0d11c6410f22a8dec3e6cf183497caa8fa26aafa052d82bcbd088f7",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__2.01-1.fc32.aarch64",
    sha256 = "ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    ],
)

rpm(
    name = "perl-Pod-Usage-4__2.01-1.fc32.x86_64",
    sha256 = "ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Pod-Usage-2.01-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/ccf730f0bc01083f4ad36f985c3cfff5be014ee02703dcb0a9e4f117036be217",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-3__1.54-440.fc32.aarch64",
    sha256 = "090511cf7961675b0697938608b90825fc032c607e25133d35906c34e50d1f51",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/090511cf7961675b0697938608b90825fc032c607e25133d35906c34e50d1f51",
    ],
)

rpm(
    name = "perl-Scalar-List-Utils-3__1.54-440.fc32.x86_64",
    sha256 = "4a2c7d2dfbb0b6813b5fc4d73e791b011ef2353ca5793474cdffd240ae4295fd",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Scalar-List-Utils-1.54-440.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/4a2c7d2dfbb0b6813b5fc4d73e791b011ef2353ca5793474cdffd240ae4295fd",
    ],
)

rpm(
    name = "perl-Socket-4__2.031-1.fc32.aarch64",
    sha256 = "c900378d0f79dc76fd2f82d2de6c7ead267c787eefdd11ba19166d4a1efbc2da",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-Socket-2.031-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/c900378d0f79dc76fd2f82d2de6c7ead267c787eefdd11ba19166d4a1efbc2da",
    ],
)

rpm(
    name = "perl-Socket-4__2.031-1.fc32.x86_64",
    sha256 = "96f0bb2811ab2b2538dac0e71e8a66b8f9a07f9191aef308d2a99f714e42bf7c",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-Socket-2.031-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/96f0bb2811ab2b2538dac0e71e8a66b8f9a07f9191aef308d2a99f714e42bf7c",
    ],
)

rpm(
    name = "perl-Storable-1__3.15-443.fc32.aarch64",
    sha256 = "e2b79f09f184c749b994522298ce66c7dad3d5b807549cea9f0b332123479479",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Storable-3.15-443.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Storable-3.15-443.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Storable-3.15-443.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Storable-3.15-443.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e2b79f09f184c749b994522298ce66c7dad3d5b807549cea9f0b332123479479",
    ],
)

rpm(
    name = "perl-Storable-1__3.15-443.fc32.x86_64",
    sha256 = "e2e9c4b18e6a65182e8368a8446a9031550b32c27443c0fda580d3d1d110792b",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Storable-3.15-443.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e2e9c4b18e6a65182e8368a8446a9031550b32c27443c0fda580d3d1d110792b",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__5.01-2.fc32.aarch64",
    sha256 = "5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    ],
)

rpm(
    name = "perl-Term-ANSIColor-0__5.01-2.fc32.x86_64",
    sha256 = "5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-ANSIColor-5.01-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/5faeaff5ad78dbe6dde7aff1fd548df6eefa051e8126d67f25053cb833102ae9",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-440.fc32.aarch64",
    sha256 = "48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    ],
)

rpm(
    name = "perl-Term-Cap-0__1.17-440.fc32.x86_64",
    sha256 = "48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Term-Cap-1.17-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48c1f06423d03965164b756807cea8e0c0b7486606c41d60b764fb9b0ce350a7",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-440.fc32.aarch64",
    sha256 = "48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    ],
)

rpm(
    name = "perl-Text-ParseWords-0__3.30-440.fc32.x86_64",
    sha256 = "48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-ParseWords-3.30-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/48bf5b99a29f8b7e7be798df28a29e858cb100dd6342341760cb375dee083cca",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-440.fc32.aarch64",
    sha256 = "f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    ],
)

rpm(
    name = "perl-Text-Tabs__plus__Wrap-0__2013.0523-440.fc32.x86_64",
    sha256 = "f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Text-Tabs+Wrap-2013.0523-440.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/f8fe1d9ec0f57d5013d6b286c4242455a8bbccbe3406a8f8758ba598d9d77a21",
    ],
)

rpm(
    name = "perl-Time-Local-2__1.300-2.fc32.aarch64",
    sha256 = "2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    ],
)

rpm(
    name = "perl-Time-Local-2__1.300-2.fc32.x86_64",
    sha256 = "2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Time-Local-1.300-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/2c1fd9ea78cfd28229e78ebc3758ef4fa5bbe839353402ca9bdfd228a6c5d33e",
    ],
)

rpm(
    name = "perl-Unicode-Normalize-0__1.26-440.fc32.aarch64",
    sha256 = "19a35c2f9bf8e1435b53181a24dcf8f2ecaa8a9f89967173cfe02b8054bc3d1f",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/19a35c2f9bf8e1435b53181a24dcf8f2ecaa8a9f89967173cfe02b8054bc3d1f",
    ],
)

rpm(
    name = "perl-Unicode-Normalize-0__1.26-440.fc32.x86_64",
    sha256 = "962ab865d9e38bb3e67284dd7c1ea1aac1e83074b72f381b50e6f7b4a65d3e84",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-Unicode-Normalize-1.26-440.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/962ab865d9e38bb3e67284dd7c1ea1aac1e83074b72f381b50e6f7b4a65d3e84",
    ],
)

rpm(
    name = "perl-constant-0__1.33-441.fc32.aarch64",
    sha256 = "965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    ],
)

rpm(
    name = "perl-constant-0__1.33-441.fc32.x86_64",
    sha256 = "965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-constant-1.33-441.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/965e2fd10921e81b597759823f0707f89d89a80feb1cb6fc5a7875bf33858705",
    ],
)

rpm(
    name = "perl-interpreter-4__5.30.3-461.fc32.aarch64",
    sha256 = "ad100c3503f4e2e79e4780ee6f27ef05998fd1842a5cc6bbe5da4f5fb714bab8",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-interpreter-5.30.3-461.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-interpreter-5.30.3-461.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-interpreter-5.30.3-461.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-interpreter-5.30.3-461.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/ad100c3503f4e2e79e4780ee6f27ef05998fd1842a5cc6bbe5da4f5fb714bab8",
    ],
)

rpm(
    name = "perl-interpreter-4__5.30.3-461.fc32.x86_64",
    sha256 = "46f1f1da9c2a0c215a0aa29fd39bdc918bd7f0c3d6eb8cc66701f4a34fbd1093",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-461.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-461.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-461.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-interpreter-5.30.3-461.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/46f1f1da9c2a0c215a0aa29fd39bdc918bd7f0c3d6eb8cc66701f4a34fbd1093",
    ],
)

rpm(
    name = "perl-libs-4__5.30.3-461.fc32.aarch64",
    sha256 = "a8d7741e46b275a7e89681bf6bbf3910bc97cce5b098937ea2b1c8f9f6653b1d",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-libs-5.30.3-461.fc32.aarch64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-libs-5.30.3-461.fc32.aarch64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-libs-5.30.3-461.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-libs-5.30.3-461.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/a8d7741e46b275a7e89681bf6bbf3910bc97cce5b098937ea2b1c8f9f6653b1d",
    ],
)

rpm(
    name = "perl-libs-4__5.30.3-461.fc32.x86_64",
    sha256 = "012cc20b8fb96a23fbd2b3bc96f405fbc0632e95c489e7abc3b8c295c21c2de6",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-461.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-461.fc32.x86_64.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-461.fc32.x86_64.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-libs-5.30.3-461.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/012cc20b8fb96a23fbd2b3bc96f405fbc0632e95c489e7abc3b8c295c21c2de6",
    ],
)

rpm(
    name = "perl-macros-4__5.30.3-461.fc32.aarch64",
    sha256 = "d23c018b926286c37be48bc81cf8d99beee6fc3167f02273c030eebd27508719",
    urls = [
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/updates/32/Everything/aarch64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d23c018b926286c37be48bc81cf8d99beee6fc3167f02273c030eebd27508719",
    ],
)

rpm(
    name = "perl-macros-4__5.30.3-461.fc32.x86_64",
    sha256 = "d23c018b926286c37be48bc81cf8d99beee6fc3167f02273c030eebd27508719",
    urls = [
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://mirror.23media.com/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/updates/32/Everything/x86_64/Packages/p/perl-macros-5.30.3-461.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/d23c018b926286c37be48bc81cf8d99beee6fc3167f02273c030eebd27508719",
    ],
)

rpm(
    name = "perl-parent-1__0.238-1.fc32.aarch64",
    sha256 = "4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    ],
)

rpm(
    name = "perl-parent-1__0.238-1.fc32.x86_64",
    sha256 = "4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-parent-0.238-1.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/4c453acd86df25c71b4ddc3de48d3b99481fc178167edf0fd622a02fabe96da0",
    ],
)

rpm(
    name = "perl-podlators-1__4.14-2.fc32.aarch64",
    sha256 = "92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    ],
)

rpm(
    name = "perl-podlators-1__4.14-2.fc32.x86_64",
    sha256 = "92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-podlators-4.14-2.fc32.noarch.rpm",
        "https://storage.googleapis.com/builddeps/92c02eedf425150cf7461f5c2a60257269a5520f865d1f1b8b55a90de2c19f87",
    ],
)

rpm(
    name = "perl-threads-1__2.22-442.fc32.aarch64",
    sha256 = "e5653553e1eb55aafbe0509ca0eba954bdaa6747f44090c5cc20250898a30ffa",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-threads-2.22-442.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-threads-2.22-442.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-threads-2.22-442.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-threads-2.22-442.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/e5653553e1eb55aafbe0509ca0eba954bdaa6747f44090c5cc20250898a30ffa",
    ],
)

rpm(
    name = "perl-threads-1__2.22-442.fc32.x86_64",
    sha256 = "ac8f21162d3353c4f65d0e10d72abf6a9c5b5a09c3a3b49aa27d96031ca5923c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-2.22-442.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/ac8f21162d3353c4f65d0e10d72abf6a9c5b5a09c3a3b49aa27d96031ca5923c",
    ],
)

rpm(
    name = "perl-threads-shared-0__1.60-441.fc32.aarch64",
    sha256 = "9215a9603f95634cdc1ebe6f305a926f2e5a604bfc8563b562ac77bbb3ec7078",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-threads-shared-1.60-441.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/p/perl-threads-shared-1.60-441.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-threads-shared-1.60-441.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/p/perl-threads-shared-1.60-441.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/9215a9603f95634cdc1ebe6f305a926f2e5a604bfc8563b562ac77bbb3ec7078",
    ],
)

rpm(
    name = "perl-threads-shared-0__1.60-441.fc32.x86_64",
    sha256 = "61797e7bdacb824cea1c1dbe5702a60b1f853bc76e6f9e1cddc2cddb98320b40",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/p/perl-threads-shared-1.60-441.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/61797e7bdacb824cea1c1dbe5702a60b1f853bc76e6f9e1cddc2cddb98320b40",
    ],
)

rpm(
    name = "pixman-0__0.38.4-1.el8.aarch64",
    sha256 = "9886953d4bc5b03f26b5c3164ce5b5fd86e9f80cf6358b91dd00f870f86052fe",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/pixman-0.38.4-1.el8.aarch64.rpm"],
)

rpm(
    name = "pixman-0__0.38.4-1.el8.x86_64",
    sha256 = "ddbbf3a8191dbc1a9fcb67ccf9cea0d34dbe9bbb74780e1359933cd03ee24451",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/pixman-0.38.4-1.el8.x86_64.rpm"],
)

rpm(
    name = "pkgconf-0__1.4.2-1.el8.aarch64",
    sha256 = "9a2c046a45d46e681f417f3b438d4bb5a21e1b93deacb59d906b8aa08a7535ad",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pkgconf-1.4.2-1.el8.aarch64.rpm"],
)

rpm(
    name = "pkgconf-0__1.4.2-1.el8.x86_64",
    sha256 = "dd08de48d25573f0a8492cf858ce8c37abb10eb560975d9df0e45a7f91b3b41d",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pkgconf-1.4.2-1.el8.x86_64.rpm"],
)

rpm(
    name = "pkgconf-m4-0__1.4.2-1.el8.aarch64",
    sha256 = "56187f25e8ae7c2a5ce228d13c6e93b9c6a701960d61dff8ad720a8879b6059e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pkgconf-m4-1.4.2-1.el8.noarch.rpm"],
)

rpm(
    name = "pkgconf-m4-0__1.4.2-1.el8.x86_64",
    sha256 = "56187f25e8ae7c2a5ce228d13c6e93b9c6a701960d61dff8ad720a8879b6059e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pkgconf-m4-1.4.2-1.el8.noarch.rpm"],
)

rpm(
    name = "pkgconf-pkg-config-0__1.4.2-1.el8.aarch64",
    sha256 = "aadca7b635ac2b30c3463a4edfe38eaee2c6064181cb090694619186747f3950",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/pkgconf-pkg-config-1.4.2-1.el8.aarch64.rpm"],
)

rpm(
    name = "pkgconf-pkg-config-0__1.4.2-1.el8.x86_64",
    sha256 = "bf5319e42dbe96c24cd64c974b17f422847cc658c4461d9d61cfe76ad76e9c67",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/pkgconf-pkg-config-1.4.2-1.el8.x86_64.rpm"],
)

rpm(
    name = "platform-python-0__3.6.8-39.el8.aarch64",
    sha256 = "e46f4a23d8a931b1cc0c839e6dd3cf3c0f1c94f7967b35e02431427539d8f1e9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-3.6.8-39.el8.aarch64.rpm"],
)

rpm(
    name = "platform-python-0__3.6.8-39.el8.x86_64",
    sha256 = "a70d0a027b10cd8bcbc5f4696fbf48fe481222705035e31590eef18cb66c96b0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-3.6.8-39.el8.x86_64.rpm"],
)

rpm(
    name = "platform-python-setuptools-0__39.2.0-6.el8.aarch64",
    sha256 = "946ba273a3a3b6fdf140f3c03112918c0a556a5871c477f5dbbb98600e6ca557",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/platform-python-setuptools-39.2.0-6.el8.noarch.rpm"],
)

rpm(
    name = "platform-python-setuptools-0__39.2.0-6.el8.x86_64",
    sha256 = "946ba273a3a3b6fdf140f3c03112918c0a556a5871c477f5dbbb98600e6ca557",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/platform-python-setuptools-39.2.0-6.el8.noarch.rpm"],
)

rpm(
    name = "policycoreutils-0__2.9-15.el8.aarch64",
    sha256 = "b9d4fd6e002ae3677727d878804e65f6f31bb05c84be3cf91efa8be8d5d4007b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/policycoreutils-2.9-15.el8.aarch64.rpm"],
)

rpm(
    name = "policycoreutils-0__2.9-15.el8.x86_64",
    sha256 = "2808973b830fa647768579804766842586340a79ce41c821b6c876bc5a8d15ec",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/policycoreutils-2.9-15.el8.x86_64.rpm"],
)

rpm(
    name = "policycoreutils-python-utils-0__2.9-15.el8.aarch64",
    sha256 = "a9d4dd71786b3147d53da1aa44f134e61b1e5527b6c7ece6c2b19474b6ca9270",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/policycoreutils-python-utils-2.9-15.el8.noarch.rpm"],
)

rpm(
    name = "policycoreutils-python-utils-0__2.9-15.el8.x86_64",
    sha256 = "a9d4dd71786b3147d53da1aa44f134e61b1e5527b6c7ece6c2b19474b6ca9270",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/policycoreutils-python-utils-2.9-15.el8.noarch.rpm"],
)

rpm(
    name = "polkit-0__0.115-12.el8.aarch64",
    sha256 = "cbd709de63c28a95b78bb32e8da27cf062a2008a47c8d799a8d8bb82a00a33e3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-0.115-12.el8.aarch64.rpm"],
)

rpm(
    name = "polkit-0__0.115-12.el8.x86_64",
    sha256 = "df82da310e172a5b40116b47b87e39cdf15b9a68b2f86d5b251203569c5d3c10",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-0.115-12.el8.x86_64.rpm"],
)

rpm(
    name = "polkit-libs-0__0.115-12.el8.aarch64",
    sha256 = "5de1ed82200ffe2d2fe91b0bf8362a6a7ff12d2f703db4eb63f6f162e510263b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-libs-0.115-12.el8.aarch64.rpm"],
)

rpm(
    name = "polkit-libs-0__0.115-12.el8.x86_64",
    sha256 = "07fbc8d163a0c526f5a6a4851c17dbc440011c25b164fe022f391ebeacfc2ebe",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-libs-0.115-12.el8.x86_64.rpm"],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-12.el8.aarch64",
    sha256 = "d25d562fe77f391458903ebf0d9078b6d38af6d9ced39d902b9afc7e717d2234",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/polkit-pkla-compat-0.1-12.el8.aarch64.rpm"],
)

rpm(
    name = "polkit-pkla-compat-0__0.1-12.el8.x86_64",
    sha256 = "e7ee4b6d6456cb7da0332f5a6fb8a7c47df977bcf616f12f0455413765367e89",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/polkit-pkla-compat-0.1-12.el8.x86_64.rpm"],
)

rpm(
    name = "popt-0__1.18-1.el8.aarch64",
    sha256 = "2596d6cba62bf9594e4fbb07df31e2459eb6fca8e479fd0be2b32c7561e9ad95",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/popt-1.18-1.el8.aarch64.rpm"],
)

rpm(
    name = "popt-0__1.18-1.el8.x86_64",
    sha256 = "3fc009f00388e66befab79be548ff3c7aa80ca70bd7f183d22f59137d8e2c2ae",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/popt-1.18-1.el8.x86_64.rpm"],
)

rpm(
    name = "procps-ng-0__3.3.15-6.el8.aarch64",
    sha256 = "dda0f9ad611135e6bee3459f183292cb1364b6c09795ead62cfe402426482212",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/procps-ng-3.3.15-6.el8.aarch64.rpm"],
)

rpm(
    name = "procps-ng-0__3.3.15-6.el8.x86_64",
    sha256 = "f5e5f477118224715f12a7151a5effcb6eda892898b5a176e1bde98b03ba7b77",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/procps-ng-3.3.15-6.el8.x86_64.rpm"],
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
    name = "psmisc-0__23.1-5.el8.x86_64",
    sha256 = "9d433d8c058e59c891c0852b95b3b87795ea30a85889c77ba0b12f965517d626",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/psmisc-23.1-5.el8.x86_64.rpm"],
)

rpm(
    name = "python3-audit-0__3.0-0.17.20191104git1c2f876.el8.aarch64",
    sha256 = "122fe05bd35778f2887e7f5cad32e8e93247fbbd71bd3da5ed78f788d529d028",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-audit-3.0-0.17.20191104git1c2f876.el8.aarch64.rpm"],
)

rpm(
    name = "python3-audit-0__3.0-0.17.20191104git1c2f876.el8.x86_64",
    sha256 = "addf80c52d794aed47874eb9d5ddbbaa90cb248fda1634d793054a41da0d92d7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-audit-3.0-0.17.20191104git1c2f876.el8.x86_64.rpm"],
)

rpm(
    name = "python3-libs-0__3.6.8-39.el8.aarch64",
    sha256 = "c4b1088492cb05631edda4cd62ea537911b79eb36e847270755bf083afafc6d9",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-libs-3.6.8-39.el8.aarch64.rpm"],
)

rpm(
    name = "python3-libs-0__3.6.8-39.el8.x86_64",
    sha256 = "958f4ef8efa6c2f1d7ac9be085ae1d90d4b9b128ef4fa31b46e72eb8d1d9a2fd",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-libs-3.6.8-39.el8.x86_64.rpm"],
)

rpm(
    name = "python3-libselinux-0__2.9-5.el8.aarch64",
    sha256 = "1a39d5db45d7e97f0a9b564b263ae22d20433bd2f40a6298b8e3ca6a80875da3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-libselinux-2.9-5.el8.aarch64.rpm"],
)

rpm(
    name = "python3-libselinux-0__2.9-5.el8.x86_64",
    sha256 = "59ba5bf69953a5a2e902b0f08b7187b66d84968852d46a3579a059477547d1a0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-libselinux-2.9-5.el8.x86_64.rpm"],
)

rpm(
    name = "python3-libsemanage-0__2.9-6.el8.aarch64",
    sha256 = "bc96ccd4671ee6a42d4ad5bbfbbd67ad397d276e29b4353ca6d67ae9705924a7",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-libsemanage-2.9-6.el8.aarch64.rpm"],
)

rpm(
    name = "python3-libsemanage-0__2.9-6.el8.x86_64",
    sha256 = "451d0cb6e2284578e3279b4cbb5ec98e9d98a687556c6b424adf8f1282d4ef58",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-libsemanage-2.9-6.el8.x86_64.rpm"],
)

rpm(
    name = "python3-pip-wheel-0__9.0.3-20.el8.aarch64",
    sha256 = "6c9dfb73e199975275633f05d31388cd61c5a77dec5678db958b4e6624eb21ba",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-pip-wheel-9.0.3-20.el8.noarch.rpm"],
)

rpm(
    name = "python3-pip-wheel-0__9.0.3-20.el8.x86_64",
    sha256 = "6c9dfb73e199975275633f05d31388cd61c5a77dec5678db958b4e6624eb21ba",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-pip-wheel-9.0.3-20.el8.noarch.rpm"],
)

rpm(
    name = "python3-policycoreutils-0__2.9-15.el8.aarch64",
    sha256 = "3eca41ba67f8121586709b6fa2c57b20b4fab73bbb8a4c133269d0cc964f362c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-policycoreutils-2.9-15.el8.noarch.rpm"],
)

rpm(
    name = "python3-policycoreutils-0__2.9-15.el8.x86_64",
    sha256 = "3eca41ba67f8121586709b6fa2c57b20b4fab73bbb8a4c133269d0cc964f362c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-policycoreutils-2.9-15.el8.noarch.rpm"],
)

rpm(
    name = "python3-setools-0__4.3.0-2.el8.aarch64",
    sha256 = "bd4efc248eee5517821027c94e937c69f92bac82243dc7798456fcef51521766",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-setools-4.3.0-2.el8.aarch64.rpm"],
)

rpm(
    name = "python3-setools-0__4.3.0-2.el8.x86_64",
    sha256 = "f56992135d789147285215cb062960fedcdf4b3296c62658fa430caa2c20165c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-setools-4.3.0-2.el8.x86_64.rpm"],
)

rpm(
    name = "python3-setuptools-wheel-0__39.2.0-6.el8.aarch64",
    sha256 = "b19bd4f106ce301ee21c860183cc1c2ef9c09bdf495059bdf16e8d8ccc71bbe8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/python3-setuptools-wheel-39.2.0-6.el8.noarch.rpm"],
)

rpm(
    name = "python3-setuptools-wheel-0__39.2.0-6.el8.x86_64",
    sha256 = "b19bd4f106ce301ee21c860183cc1c2ef9c09bdf495059bdf16e8d8ccc71bbe8",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/python3-setuptools-wheel-39.2.0-6.el8.noarch.rpm"],
)

rpm(
    name = "qemu-img-15__5.2.0-16.el8s.aarch64",
    sha256 = "4fc1041ce8d7fdcaf4f90f1ed33421b9e4967a18b06b54870cc96f3b84879bb6",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/q/qemu-img-5.2.0-16.el8s.aarch64.rpm"],
)

rpm(
    name = "qemu-img-15__5.2.0-16.el8s.x86_64",
    sha256 = "be18c7c4ce697a2d73f4f6952df1d32a668e5519d43af345e643cde16a57e3d3",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-img-5.2.0-16.el8s.x86_64.rpm"],
)

rpm(
    name = "qemu-img-2__4.2.1-1.fc32.aarch64",
    sha256 = "20f96cbd6c1b2b0c54436c10358e0e936a0b8ccddd3f30a0f328138176af333c",
    urls = [
        "https://fedora.mirror.garr.it/fedora/linux/updates/32/Everything/aarch64/Packages/q/qemu-img-4.2.1-1.fc32.aarch64.rpm",
        "https://mirror.telepoint.bg/fedora/updates/32/Everything/aarch64/Packages/q/qemu-img-4.2.1-1.fc32.aarch64.rpm",
        "https://mirror.serverion.com/fedora/updates/32/Everything/aarch64/Packages/q/qemu-img-4.2.1-1.fc32.aarch64.rpm",
        "https://mirror.init7.net/fedora/fedora/linux/updates/32/Everything/aarch64/Packages/q/qemu-img-4.2.1-1.fc32.aarch64.rpm",
    ],
)

rpm(
    name = "qemu-img-2__4.2.1-1.fc32.x86_64",
    sha256 = "ee4f4b67c1735283511a830ce98a259b5dff8c623ecd6c2ebb1dda74c43b0805",
    urls = [
        "https://fedora.mirror.garr.it/fedora/linux/updates/32/Everything/x86_64/Packages/q/qemu-img-4.2.1-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/updates/32/Everything/x86_64/Packages/q/qemu-img-4.2.1-1.fc32.x86_64.rpm",
        "https://mirror.init7.net/fedora/fedora/linux/updates/32/Everything/x86_64/Packages/q/qemu-img-4.2.1-1.fc32.x86_64.rpm",
        "https://fedora.ipacct.com/fedora/linux/updates/32/Everything/x86_64/Packages/q/qemu-img-4.2.1-1.fc32.x86_64.rpm",
    ],
)

rpm(
    name = "qemu-kvm-common-15__5.2.0-16.el8s.aarch64",
    sha256 = "bb229b5e4dee19fb4cebcc05dee464dc717370dfb11da451d42269f1bdec6ed1",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/q/qemu-kvm-common-5.2.0-16.el8s.aarch64.rpm"],
)

rpm(
    name = "qemu-kvm-common-15__5.2.0-16.el8s.x86_64",
    sha256 = "146311b6e085645a5b95b9edb30c37ffa86e623901d763d36ee2c3dc5d70a94f",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-kvm-common-5.2.0-16.el8s.x86_64.rpm"],
)

rpm(
    name = "qemu-kvm-core-15__5.2.0-16.el8s.aarch64",
    sha256 = "682ab321d801bdb35b9eed293694f5965c5c6953929ed3e6c3231fc0fa6a0abb",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/q/qemu-kvm-core-5.2.0-16.el8s.aarch64.rpm"],
)

rpm(
    name = "qemu-kvm-core-15__5.2.0-16.el8s.x86_64",
    sha256 = "ff7fbd6c095c6c642aeb25d329707f8fb57cb1aaa82db6cca53da32cc3638c01",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/q/qemu-kvm-core-5.2.0-16.el8s.x86_64.rpm"],
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
    name = "rdma-core-0__35.0-1.el8.aarch64",
    sha256 = "b170c69f99c0bacc01a003c70098e04132629589bf3b0737a73e229da75d0571",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rdma-core-35.0-1.el8.aarch64.rpm"],
)

rpm(
    name = "rdma-core-0__35.0-1.el8.x86_64",
    sha256 = "ee2d61287b724b6088902016728fc5d9a17744d0b69aa7ccab9e133264f18603",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rdma-core-35.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "readline-0__7.0-10.el8.aarch64",
    sha256 = "ef74f2c65ed0e38dd021177d6e59fcdf7fb8de8929b7544b7a6f0709eff6562c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/readline-7.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "readline-0__7.0-10.el8.x86_64",
    sha256 = "fea868a7d82a7b6f392260ed4afb472dc4428fd71eab1456319f423a845b5084",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/readline-7.0-10.el8.x86_64.rpm"],
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
    name = "rpm-0__4.14.3-15.el8.aarch64",
    sha256 = "58cad77e10210a5c9183ce5518d452c6a234c5553b93b94af0de3f470c0d944b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-4.14.3-15.el8.aarch64.rpm"],
)

rpm(
    name = "rpm-0__4.14.3-15.el8.x86_64",
    sha256 = "38e2d7fd362b10b71aa0b357331314d0f370a35185dfa31818c899cbb56ff610",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-4.14.3-15.el8.x86_64.rpm"],
)

rpm(
    name = "rpm-libs-0__4.14.3-15.el8.aarch64",
    sha256 = "deed01a583548b6283a78af0cae84c367e279d9e9e878244a73a9c61dec0cb59",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-libs-4.14.3-15.el8.aarch64.rpm"],
)

rpm(
    name = "rpm-libs-0__4.14.3-15.el8.x86_64",
    sha256 = "3705a3d6c18aea20fd77cd44d91a33fdb5f15e8a8a8eae658ba952323261de94",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-libs-4.14.3-15.el8.x86_64.rpm"],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-15.el8.aarch64",
    sha256 = "f03b364d736df39fa832009dde2678dae17493dffeb8eba374f0a919584f1b31",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/rpm-plugin-selinux-4.14.3-15.el8.aarch64.rpm"],
)

rpm(
    name = "rpm-plugin-selinux-0__4.14.3-15.el8.x86_64",
    sha256 = "d210335d60e89f32db0dfc86b68fa5dcbbc12b6f4972695aba93371470a9a78f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/rpm-plugin-selinux-4.14.3-15.el8.x86_64.rpm"],
)

rpm(
    name = "scrub-0__2.5.2-14.el8.x86_64",
    sha256 = "4973c48ebe26e5d97095abe45e4f628521589e310bfa3e1a3387e166d7ab8adc",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/scrub-2.5.2-14.el8.x86_64.rpm"],
)

rpm(
    name = "scsi-target-utils-0__1.0.79-1.fc32.aarch64",
    sha256 = "14f9875de4dbdef7a6a8d00490e770bb0f9379642534f523fe0cb7fd75302baf",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/14f9875de4dbdef7a6a8d00490e770bb0f9379642534f523fe0cb7fd75302baf",
    ],
)

rpm(
    name = "scsi-target-utils-0__1.0.79-1.fc32.x86_64",
    sha256 = "361a48d36c608a4790d2811fecb98503d4afc7da14f53ebb82d53d2e3994d786",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/scsi-target-utils-1.0.79-1.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/361a48d36c608a4790d2811fecb98503d4afc7da14f53ebb82d53d2e3994d786",
    ],
)

rpm(
    name = "seabios-0__1.14.0-1.el8s.x86_64",
    sha256 = "468be89248e2b4cf655832f7b156e8ce90d726f0203d7c729293ce708a16cc7f",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/seabios-1.14.0-1.el8s.x86_64.rpm"],
)

rpm(
    name = "seabios-bin-0__1.14.0-1.el8s.x86_64",
    sha256 = "89033ae80928e60ff0377703208e08fb57c31b4e6d31697936817a3125180abb",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/seabios-bin-1.14.0-1.el8s.noarch.rpm"],
)

rpm(
    name = "seavgabios-bin-0__1.14.0-1.el8s.x86_64",
    sha256 = "558868cab91b079c7a33e15602b715d38b31966a6f12f953b2b67ec8cad9ccf1",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/seavgabios-bin-1.14.0-1.el8s.noarch.rpm"],
)

rpm(
    name = "sed-0__4.5-2.el8.aarch64",
    sha256 = "f89de80c1d2c1c8ad2b1bb92055b1a4c7dca0ca0ffb6419b76e13617f1fe827e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sed-4.5-2.el8.aarch64.rpm"],
)

rpm(
    name = "sed-0__4.5-2.el8.x86_64",
    sha256 = "33aa5c86d596d06a2f199b65b61912cbd2c46c3923f13dadc6179650d45ba96c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sed-4.5-2.el8.x86_64.rpm"],
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
    name = "selinux-policy-0__3.14.3-75.el8.aarch64",
    sha256 = "c9ab1d91e55fbf8b2494b16915ea345bc9a1b8ae584ce9f31ed42b475432a0b3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-3.14.3-75.el8.noarch.rpm"],
)

rpm(
    name = "selinux-policy-0__3.14.3-75.el8.x86_64",
    sha256 = "c9ab1d91e55fbf8b2494b16915ea345bc9a1b8ae584ce9f31ed42b475432a0b3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-3.14.3-75.el8.noarch.rpm"],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-75.el8.aarch64",
    sha256 = "d5f9b9b7ab53b9ced59ea78dd95b3d962caf4045995fb84287af1467c9ac7ae4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/selinux-policy-targeted-3.14.3-75.el8.noarch.rpm"],
)

rpm(
    name = "selinux-policy-targeted-0__3.14.3-75.el8.x86_64",
    sha256 = "d5f9b9b7ab53b9ced59ea78dd95b3d962caf4045995fb84287af1467c9ac7ae4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/selinux-policy-targeted-3.14.3-75.el8.noarch.rpm"],
)

rpm(
    name = "setup-0__2.12.2-6.el8.aarch64",
    sha256 = "9e540fe1fcf866ba1e738e012eef5459d34cca30385df73973e6fc7c6eadb55f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/setup-2.12.2-6.el8.noarch.rpm"],
)

rpm(
    name = "setup-0__2.12.2-6.el8.x86_64",
    sha256 = "9e540fe1fcf866ba1e738e012eef5459d34cca30385df73973e6fc7c6eadb55f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/setup-2.12.2-6.el8.noarch.rpm"],
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
    name = "sg3_utils-0__1.44-3.fc32.aarch64",
    sha256 = "81a9b1386eaa107ad0bceac405bf1c89e36177272c2ad2253b60b9984aa91fdd",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-1.44-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-1.44-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-1.44-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-1.44-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/81a9b1386eaa107ad0bceac405bf1c89e36177272c2ad2253b60b9984aa91fdd",
    ],
)

rpm(
    name = "sg3_utils-0__1.44-3.fc32.x86_64",
    sha256 = "cd3d9eb488859202bb6820830d7bb5622219492484e9d98c279ccb1211750eae",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-1.44-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/cd3d9eb488859202bb6820830d7bb5622219492484e9d98c279ccb1211750eae",
    ],
)

rpm(
    name = "sg3_utils-libs-0__1.44-3.fc32.aarch64",
    sha256 = "572dfe51db31cb1082bf43ef109a36367a164e54e486aa1520eac6c43a8af26a",
    urls = [
        "https://ftp.halifax.rwth-aachen.de/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.aarch64.rpm",
        "https://mirrors.xtom.de/fedora/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.aarch64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.aarch64.rpm",
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/aarch64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.aarch64.rpm",
        "https://storage.googleapis.com/builddeps/572dfe51db31cb1082bf43ef109a36367a164e54e486aa1520eac6c43a8af26a",
    ],
)

rpm(
    name = "sg3_utils-libs-0__1.44-3.fc32.x86_64",
    sha256 = "acafd54a39135c9ac45e5046f3b4d8b3712eba4acd99d44bd044557ad3c3939c",
    urls = [
        "https://ftp-stud.hs-esslingen.de/pub/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://mirror.dogado.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://ftp.plusline.net/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://ftp.fau.de/fedora/linux/releases/32/Everything/x86_64/os/Packages/s/sg3_utils-libs-1.44-3.fc32.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/acafd54a39135c9ac45e5046f3b4d8b3712eba4acd99d44bd044557ad3c3939c",
    ],
)

rpm(
    name = "sgabios-bin-1__0.20170427git-3.module_el8.5.0__plus__746__plus__bbd5d70c.x86_64",
    sha256 = "1e6f37883269101bb80f8a51ce60243068fa801d0a8e6c83808d57102835cc8f",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/sgabios-bin-0.20170427git-3.module_el8.5.0+746+bbd5d70c.noarch.rpm"],
)

rpm(
    name = "shadow-utils-2__4.6-13.el8.aarch64",
    sha256 = "02ce11f42faaf6f58e3219afafa147cae68403295fb949da3f9322a49354776a",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/shadow-utils-4.6-13.el8.aarch64.rpm"],
)

rpm(
    name = "shadow-utils-2__4.6-13.el8.x86_64",
    sha256 = "4a74407bb9435709c22db70ed6fbcd5bd9e5e4e8b6ad8110a651cd14f159736b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/shadow-utils-4.6-13.el8.x86_64.rpm"],
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
    name = "snappy-0__1.1.8-3.el8.aarch64",
    sha256 = "4731985b22fc7b733ff89be6c1423396f27c94a78bb09fc89be5c2200bee893c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/snappy-1.1.8-3.el8.aarch64.rpm"],
)

rpm(
    name = "snappy-0__1.1.8-3.el8.x86_64",
    sha256 = "839c62cd7fc7e152decded6f28c80b5f7b8f34a5e319057867b38b26512cee67",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/snappy-1.1.8-3.el8.x86_64.rpm"],
)

rpm(
    name = "sqlite-libs-0__3.26.0-15.el8.aarch64",
    sha256 = "b3a0c27117c927795b1a3a1ef2c08c857a88199bcfad5603cd2303c9519671a4",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/sqlite-libs-3.26.0-15.el8.aarch64.rpm"],
)

rpm(
    name = "sqlite-libs-0__3.26.0-15.el8.x86_64",
    sha256 = "46d01b59aba3aaccaf32731ada7323f62ae848fe17ff2bd020589f282b3ccac3",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/sqlite-libs-3.26.0-15.el8.x86_64.rpm"],
)

rpm(
    name = "squashfs-tools-0__4.3-20.el8.x86_64",
    sha256 = "956da9a94f3f2331df649b8351ebeb0c102702486ff447e8252e7af3b96ab414",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/squashfs-tools-4.3-20.el8.x86_64.rpm"],
)

rpm(
    name = "supermin-0__5.2.1-1.el8s.x86_64",
    sha256 = "177087eed49db85288b6601f63a17bb0bd84135edd8aa1f4d0cecbe61f2f619e",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/supermin-5.2.1-1.el8s.x86_64.rpm"],
)

rpm(
    name = "swtpm-0__0.6.0-1.20210607gitea627b3.el8s.aarch64",
    sha256 = "e4907058884ebc37e22424c3294488db30aaf9e1f0a8e5087b3f9363ae30cd65",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.el8s.aarch64.rpm"],
)

rpm(
    name = "swtpm-0__0.6.0-1.20210607gitea627b3.el8s.x86_64",
    sha256 = "9f2003a8aca209936ffec799172d5658536c3b0a558d3a1b8b701b3aa035b8d4",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/swtpm-0.6.0-1.20210607gitea627b3.el8s.x86_64.rpm"],
)

rpm(
    name = "swtpm-libs-0__0.6.0-1.20210607gitea627b3.el8s.aarch64",
    sha256 = "f19f46009f0ec712a053fa23c22e4123aafa28553a6ae1aeebe4de63254dd163",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.el8s.aarch64.rpm"],
)

rpm(
    name = "swtpm-libs-0__0.6.0-1.20210607gitea627b3.el8s.x86_64",
    sha256 = "ca8365a7dc58f38f398cfe33dfc69091952f8f94b7f8336b405e0bb0c5ccc601",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/swtpm-libs-0.6.0-1.20210607gitea627b3.el8s.x86_64.rpm"],
)

rpm(
    name = "swtpm-tools-0__0.6.0-1.20210607gitea627b3.el8s.aarch64",
    sha256 = "a2e024a819533bee54313b327780bdfea9281d627d959352f4adf134da32c9f7",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/aarch64/advancedvirt-common/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.el8s.aarch64.rpm"],
)

rpm(
    name = "swtpm-tools-0__0.6.0-1.20210607gitea627b3.el8s.x86_64",
    sha256 = "296aef985000c146bb9be5f86960452a2c50b42134a31f0a632a2eedb03ba5a6",
    urls = ["http://mirror.centos.org/centos/8-stream/virt/x86_64/advancedvirt-common/Packages/s/swtpm-tools-0.6.0-1.20210607gitea627b3.el8s.x86_64.rpm"],
)

rpm(
    name = "syslinux-0__6.04-5.el8.x86_64",
    sha256 = "33996f2476ed82d68353ac5c6d22c204db4ee76821eefc4c0cc2dafcf44ae16b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-6.04-5.el8.x86_64.rpm"],
)

rpm(
    name = "syslinux-extlinux-0__6.04-5.el8.x86_64",
    sha256 = "28ede201bc3e0a3aae01dfb96d84260c933372b710be37c25d923504ee43acea",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-extlinux-6.04-5.el8.x86_64.rpm"],
)

rpm(
    name = "syslinux-extlinux-nonlinux-0__6.04-5.el8.x86_64",
    sha256 = "32b57460a7ce649954f813033d44a8feba1ab30cd1cf99c0a64f5826d2448167",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-extlinux-nonlinux-6.04-5.el8.noarch.rpm"],
)

rpm(
    name = "syslinux-nonlinux-0__6.04-5.el8.x86_64",
    sha256 = "89f2d9a00712110d283de570cd3212c204fdcf78a32cd71e0d6ee660e412941c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/syslinux-nonlinux-6.04-5.el8.noarch.rpm"],
)

rpm(
    name = "systemd-0__239-49.el8.aarch64",
    sha256 = "454ec267900a2b43ac86d9730b65e8b78bb0ef823c3b8777b531021029c5ce86",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-239-49.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-0__239-49.el8.x86_64",
    sha256 = "f16942e337972452994ad32c3e5a5385450eb46857b8fba61aa11b764c484eef",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-239-49.el8.x86_64.rpm"],
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
    name = "systemd-container-0__239-49.el8.aarch64",
    sha256 = "c44fd1680c387659ae37f04fa32dcabb086f6befb2458bc1a721f1ac29aa870b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-container-239-49.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-container-0__239-49.el8.x86_64",
    sha256 = "038db403f424302bf91224ede70d7cecb000c6d0b66f1f568ab0602b1915c6af",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-container-239-49.el8.x86_64.rpm"],
)

rpm(
    name = "systemd-libs-0__239-49.el8.aarch64",
    sha256 = "feb22f3d5529fce8a1d60fe9ddc9fba7dadf4d0864b6ae8bb8245985aef709ec",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-libs-239-49.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-libs-0__239-49.el8.x86_64",
    sha256 = "0854686cc3009d2fdf8baf48cbef87c58b10baf85302e7061b25cc64a1848e64",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-libs-239-49.el8.x86_64.rpm"],
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
    name = "systemd-pam-0__239-49.el8.aarch64",
    sha256 = "3f2566af901b766511de7fb13f6a675ea982c8d28e3cb7746635938cd41b877e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/systemd-pam-239-49.el8.aarch64.rpm"],
)

rpm(
    name = "systemd-pam-0__239-49.el8.x86_64",
    sha256 = "33e8bc88e97523512e0aba1f8cf03130fd72a180d1633497a46020ab72b69481",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-pam-239-49.el8.x86_64.rpm"],
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
    name = "systemd-udev-0__239-49.el8.x86_64",
    sha256 = "2dbf7a4f5bc6d801d4bfe921cf4b8070d57ea8fe1f563cca53a03465bb14305e",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/systemd-udev-239-49.el8.x86_64.rpm"],
)

rpm(
    name = "tar-2__1.30-5.el8.aarch64",
    sha256 = "3d527d861793fe3a74b6254540068e8b846e6df20d75754df39904e67f1e569f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/tar-1.30-5.el8.aarch64.rpm"],
)

rpm(
    name = "tar-2__1.30-5.el8.x86_64",
    sha256 = "ed1f7ab0225df75734034cb2aea426c48c089f2bd476ec66b66af879437c5393",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/tar-1.30-5.el8.x86_64.rpm"],
)

rpm(
    name = "tzdata-0__2021a-1.el8.aarch64",
    sha256 = "44999c555a6e4bb6cf5e6f6a79819e76912d036732cb50efaeefc20a180dd839",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/tzdata-2021a-1.el8.noarch.rpm"],
)

rpm(
    name = "tzdata-0__2021a-1.el8.x86_64",
    sha256 = "44999c555a6e4bb6cf5e6f6a79819e76912d036732cb50efaeefc20a180dd839",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/tzdata-2021a-1.el8.noarch.rpm"],
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
    name = "unbound-libs-0__1.7.3-17.el8.aarch64",
    sha256 = "406140d0a2d6fe921875898b24b91376870fb9ab1b1baf7778cff060bbbe0d72",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/unbound-libs-1.7.3-17.el8.aarch64.rpm"],
)

rpm(
    name = "unbound-libs-0__1.7.3-17.el8.x86_64",
    sha256 = "9a5380195d24327a8a2e059395d7902f9bc3b771275afe1533702998dc5be364",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/unbound-libs-1.7.3-17.el8.x86_64.rpm"],
)

rpm(
    name = "usbredir-0__0.8.0-1.el8.x86_64",
    sha256 = "359290c30476453554d970c0f5360b6039e8b92fb72018a65b7e56b38f260bda",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/usbredir-0.8.0-1.el8.x86_64.rpm"],
)

rpm(
    name = "userspace-rcu-0__0.10.1-4.el8.aarch64",
    sha256 = "c4b53c8f1121938c2c5ae3fabd48b9d8f77c7d26f47a76f5c0eab3fd7f0a6cfc",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/userspace-rcu-0.10.1-4.el8.aarch64.rpm"],
)

rpm(
    name = "userspace-rcu-0__0.10.1-4.el8.x86_64",
    sha256 = "4025900345c5125fd6c10c1780275139f56b63be2bfac10be83628758c225dd0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/userspace-rcu-0.10.1-4.el8.x86_64.rpm"],
)

rpm(
    name = "util-linux-0__2.32.1-28.el8.aarch64",
    sha256 = "1600cd7372ca2682c9bd58de6e783092d6bdb6412c9888e3dd767db92c9b5239",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/util-linux-2.32.1-28.el8.aarch64.rpm"],
)

rpm(
    name = "util-linux-0__2.32.1-28.el8.x86_64",
    sha256 = "8213aa26dbe71291c0cd5969256f3f0bce13ee8e8cd76e3b949c1f6752142471",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/util-linux-2.32.1-28.el8.x86_64.rpm"],
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
    name = "vim-minimal-2__8.0.1763-15.el8.aarch64",
    sha256 = "2b743f157f47b27a0528d53fd9ae3e4eb0553f4b8357d01d362dd1b0e4e87c06",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/vim-minimal-8.0.1763-15.el8.aarch64.rpm"],
)

rpm(
    name = "vim-minimal-2__8.0.1763-15.el8.x86_64",
    sha256 = "3efd6a2548813167fe37718546bc768a5aa8ba59aa80edcecd8ba408bec329b0",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/vim-minimal-8.0.1763-15.el8.x86_64.rpm"],
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
    name = "which-0__2.21-16.el8.x86_64",
    sha256 = "0a4bd60fba20ec837a384a52cc56c852f6d01b3b6ec810e3a3d538a42442b937",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/which-2.21-16.el8.x86_64.rpm"],
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
    name = "xkeyboard-config-0__2.28-1.el8.aarch64",
    sha256 = "a2aeabb3962859069a78acc288bc3bffb35485428e162caafec8134f5ce6ca67",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/xkeyboard-config-2.28-1.el8.noarch.rpm"],
)

rpm(
    name = "xkeyboard-config-0__2.28-1.el8.x86_64",
    sha256 = "a2aeabb3962859069a78acc288bc3bffb35485428e162caafec8134f5ce6ca67",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/xkeyboard-config-2.28-1.el8.noarch.rpm"],
)

rpm(
    name = "xorriso-0__1.4.8-4.el8.aarch64",
    sha256 = "4280064ab658525b486d7b8c2ca5f87aeef90002361a0925f2819fd7a7909500",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/xorriso-1.4.8-4.el8.aarch64.rpm"],
)

rpm(
    name = "xorriso-0__1.4.8-4.el8.x86_64",
    sha256 = "3a232d848da1ace286efef6c8c9cf0fcfab2c47dd58968ddb6a24718629a6220",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/xorriso-1.4.8-4.el8.x86_64.rpm"],
)

rpm(
    name = "xz-0__5.2.4-3.el8.aarch64",
    sha256 = "b9a899e715019e7002600005bcb2a9dd7b089eaef9c55c3764c326d745ad681f",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/xz-5.2.4-3.el8.aarch64.rpm"],
)

rpm(
    name = "xz-0__5.2.4-3.el8.x86_64",
    sha256 = "02f10beaf61212427e0cd57140d050948eea0b533cf432d7bc4c10266c8b33db",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/xz-5.2.4-3.el8.x86_64.rpm"],
)

rpm(
    name = "xz-libs-0__5.2.4-3.el8.aarch64",
    sha256 = "8f141db26834b1ec60028790b130d00b14b7fda256db0df1e51b7ba8d3d40c7b",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/xz-libs-5.2.4-3.el8.aarch64.rpm"],
)

rpm(
    name = "xz-libs-0__5.2.4-3.el8.x86_64",
    sha256 = "61553db2c5d1da168da53ec285de14d00ce91bb02dd902a1688725cf37a7b1a2",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/xz-libs-5.2.4-3.el8.x86_64.rpm"],
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
    name = "yajl-0__2.1.0-10.el8.aarch64",
    sha256 = "255e74b387f5e9b517d82cd00f3b62af88b32054095be91a63b3e5eb5db34939",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/aarch64/os/Packages/yajl-2.1.0-10.el8.aarch64.rpm"],
)

rpm(
    name = "yajl-0__2.1.0-10.el8.x86_64",
    sha256 = "a7797aa70d6a35116ec3253523dc91d1b08df44bad7442b94af07bb6c0a661f0",
    urls = ["http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/yajl-2.1.0-10.el8.x86_64.rpm"],
)

rpm(
    name = "zlib-0__1.2.11-17.el8.aarch64",
    sha256 = "19223c1996366de6f38c38f5d0163368fbff9c29149bb925ffe8d2eba79b239c",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/aarch64/os/Packages/zlib-1.2.11-17.el8.aarch64.rpm"],
)

rpm(
    name = "zlib-0__1.2.11-17.el8.x86_64",
    sha256 = "a604ffec838794e53b7721e4f113dbd780b5a0765f200df6c41ea19018fa7ea6",
    urls = ["http://mirror.centos.org/centos/8-stream/BaseOS/x86_64/os/Packages/zlib-1.2.11-17.el8.x86_64.rpm"],
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
