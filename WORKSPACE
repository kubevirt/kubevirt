load(
    "@bazel_tools//tools/build_defs/repo:http.bzl",
    "http_archive",
    "http_file",
)

# Additional bazel rules
http_archive(
    name = "io_bazel_rules_go",
    sha256 = "078f2a9569fa9ed846e60805fb5fb167d6f6c4ece48e6d409bf5fb2154eaf0d8",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.20.0/rules_go-v0.20.0.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.20.0/rules_go-v0.20.0.tar.gz",
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
    sha256 = "413bb1ec0895a8d3249a01edf24b82fd06af3c8633c9fb833a0cb1d4b234d46d",
    strip_prefix = "rules_docker-0.12.0",
    urls = [
        "https://github.com/bazelbuild/rules_docker/releases/download/v0.12.0/rules_docker-v0.12.0.tar.gz",
        "https://storage.googleapis.com/builddeps/413bb1ec0895a8d3249a01edf24b82fd06af3c8633c9fb833a0cb1d4b234d46d",
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
    name = "libvirt_devel",
    sha256 = "e34ca39bb94786b901a626143527f86cae1b6f1c8d1414165465d89e146a612e",
    urls = [
        "https://copr-be.cloud.fedoraproject.org/results/%40virtmaint-sig/for-kubevirt/fedora-30-x86_64/01034621-libvirt/libvirt-devel-5.0.0-2.fc30.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/e34ca39bb94786b901a626143527f86cae1b6f1c8d1414165465d89e146a612e",
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
    name = "cirros_image",
    sha256 = "a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    urls = [
        "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img",
        "https://storage.googleapis.com/builddeps/a8dd75ecffd4cdd96072d60c2237b448e0c8b2bc94d57f10fdbc8c481d9005b8",
    ],
)

http_file(
    name = "fedora_image",
    sha256 = "a30549d620bf6bf41d30a9a58626e59dfa70bb011fd7d50f6c4511ad2e479a39",
    urls = [
        "https://download.fedoraproject.org/pub/fedora/linux/releases/29/Cloud/x86_64/images/Fedora-Cloud-Base-29-1.2.x86_64.qcow2",
        "https://storage.googleapis.com/builddeps/a30549d620bf6bf41d30a9a58626e59dfa70bb011fd7d50f6c4511ad2e479a39",
    ],
)

http_file(
    name = "fedora30_image",
    sha256 = "72b6ae7b4ed09a4dccd6e966e1b3ac69bd97da419de9760b410e837ba00b4e26",
    urls = [
        "https://download.fedoraproject.org/pub/fedora/linux/releases/30/Cloud/x86_64/images/Fedora-Cloud-Base-30-1.2.x86_64.qcow2",
        "https://storage.googleapis.com/builddeps/72b6ae7b4ed09a4dccd6e966e1b3ac69bd97da419de9760b410e837ba00b4e26",
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

# Pull base image fedora28
container_pull(
    name = "fedora",
    digest = "sha256:f4ca3e9814bef8468b59cc3dea90ab52bcb3747057731f7ab7645d923731fed6",
    registry = "index.docker.io",
    repository = "library/fedora",
    #tag = "30",
)

# Pull base image libvirt
container_pull(
    name = "libvirt",
    digest = "sha256:840d4502f48f567f9030e22ba8aa8294003f1ac20b6a0c06c01b0236a8964a3a",
    registry = "index.docker.io",
    repository = "kubevirt/libvirt",
    #tag = "5.0.0-2.fc30",
)

# Pull kubevirt-testing image
container_pull(
    name = "kubevirt-testing",
    digest = "sha256:eb86f7388217bb18611c8c4e6169af3463c2a18f420314eb4d742b3d3669b16f",
    registry = "index.docker.io",
    repository = "kubevirtci/kubevirt-testing",
    #tag = "28",
)

# Pull nfs-server image
container_pull(
    name = "nfs-server",
    digest = "sha256:2a8c1cad9156165d0be1ee2c78fb2342f8368193acff72eb276ff276512508fa",
    registry = "index.docker.io",
    repository = "slintes/nfs-ganesha",  # see https://github.com/slintes/docker-nfs-ganesha
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
    name = "bzip2",
    sha256 = "5248b7b85e58319c6c16e9b545dc04f1b25ba236e0507a7c923d74b00fe8741c",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/b/bzip2-1.0.6-26.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/5248b7b85e58319c6c16e9b545dc04f1b25ba236e0507a7c923d74b00fe8741c",
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
    name = "libaio",
    sha256 = "da731218ec1a8e8f690c880d7a45d09722ea01090caba0ae25d9202e0d521404",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/releases/28/Everything/x86_64/os/Packages/l/libaio-0.3.110-11.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/da731218ec1a8e8f690c880d7a45d09722ea01090caba0ae25d9202e0d521404",
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
    name = "qemu-guest-agent",
    sha256 = "d9c5072ab2476fbf9e50dd374b2bc680d3accadd2e70d52cae4eb515b6a893f4",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/q/qemu-guest-agent-2.11.2-5.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d9c5072ab2476fbf9e50dd374b2bc680d3accadd2e70d52cae4eb515b6a893f4",
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
    name = "e2fsprogs",
    sha256 = "d6db37d587a2a0f7cd19e42aea8bd3e5e7c3a9c39c324d40be7514624f9f8f5f",
    urls = [
        "https://archives.fedoraproject.org/pub/archive/fedora/linux/updates/28/Everything/x86_64/Packages/e/e2fsprogs-1.44.2-0.fc28.x86_64.rpm",
        "https://storage.googleapis.com/builddeps/d6db37d587a2a0f7cd19e42aea8bd3e5e7c3a9c39c324d40be7514624f9f8f5f",
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
    sum = "h1:vb/1TCsVn3DcJlQ0Gs1yB1pKI6Do2/QNwxdKqmc/b0s=",
    version = "v1.24.0",
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
