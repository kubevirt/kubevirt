"""Dependencies for virt-template images."""

load("@rules_img//img:pull.bzl", "pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:88b0c03b786bbdb35c4c9efef6fe8717675f3c8829d7d1e45822e5e9386bd69e"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:59153062cc399d80f8829fd5dbebf41a16d773c4c96e0552330ba16463f50c51"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:036864aa81c830dfdbe0c3ab134119fdb110bc0927a9fe955c743efb3c345ca3"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:ea58f7d3bbd71e173bcf7a59593c987d776ceed85f58d2ac9cc9961471dbc1d5"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:2f44b269bbb9684d74c66839b156ed2afd929b6eae500f334600574fbfec6054"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:1c14a04ee2de6d4262211b3a8509be43178681a1f2f019af23f5b0b2203ddba3"

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
