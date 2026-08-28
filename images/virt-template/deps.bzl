"""Dependencies for virt-template images."""

load("@rules_img//img:pull.bzl", "pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:9909065604cdfaa826e7c1a48ffba908370b5eb966b9d3affe0fdc22d9196255"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:e60170345f89d03fde41cfd65129901f474cbfafba3e47ed10788c6603a78b53"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:5a084bf838c0308a39b9ad54b94bc46cb73db7a16fff0cb4b93c068a059da8ac"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:abe3385a7c5ec1ddba0735423e00b783f6b464e0877d58cf0ab76eea012eb89e"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:fdc9755438258b1adc6c7a0148b1bf0a53128a319e7cc7c6b9d579e08d9e608f"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:6eed3581fc6610519cb1280253a2113a5cdf916c454328966079a74f6aa09971"

def virt_template_images():
    """Pull virt-template images for all architectures."""
    pull(
        name = "virt_template_apiserver",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_AMD64,
        registry = "quay.io",
        repository = "kubevirt/virt-template-apiserver",
    )

    pull(
        name = "virt_template_apiserver_aarch64",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_ARM64,
        registry = "quay.io",
        repository = "kubevirt/virt-template-apiserver",
    )

    pull(
        name = "virt_template_apiserver_s390x",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_S390X,
        registry = "quay.io",
        repository = "kubevirt/virt-template-apiserver",
    )

    pull(
        name = "virt_template_controller",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64,
        registry = "quay.io",
        repository = "kubevirt/virt-template-controller",
    )

    pull(
        name = "virt_template_controller_aarch64",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64,
        registry = "quay.io",
        repository = "kubevirt/virt-template-controller",
    )

    pull(
        name = "virt_template_controller_s390x",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X,
        registry = "quay.io",
        repository = "kubevirt/virt-template-controller",
    )
