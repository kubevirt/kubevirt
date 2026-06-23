"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:f569e1acc6a8361ecf9ce386cddfdbb036696a2ab2fbf8d8c7f1fe1928e6b049"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:3bed4089e9d5f410f0f4fb883b7d84e6c2356db21f010d1493e5b9818c6fc94a"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:c30eeb6f95b10459bff7330fb2771c058bf454e78d92af1854090ee901f6e659"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:12bc0cec65e5dec4f1dc453e74b3c6d1f467d75015e73e3adb24d3be8e3a7586"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:5f41143055e9b716d8c4f6efe4e5f15f62353ac82b0f4a7ccc6b4c5d9b8ee922"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:a334996d54ff91e9472792fead645361b0a9a2f31fb5faa032c772870d60ee53"

def virt_template_images():
    """Pull virt-template images for all architectures."""
    oci_pull(
        name = "virt_template_apiserver",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_AMD64,
        image = "quay.io/kubevirt/virt-template-apiserver",
    )

    oci_pull(
        name = "virt_template_apiserver_aarch64",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_ARM64,
        image = "quay.io/kubevirt/virt-template-apiserver",
    )

    oci_pull(
        name = "virt_template_apiserver_s390x",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_S390X,
        image = "quay.io/kubevirt/virt-template-apiserver",
    )

    oci_pull(
        name = "virt_template_controller",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64,
        image = "quay.io/kubevirt/virt-template-controller",
    )

    oci_pull(
        name = "virt_template_controller_aarch64",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64,
        image = "quay.io/kubevirt/virt-template-controller",
    )

    oci_pull(
        name = "virt_template_controller_s390x",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X,
        image = "quay.io/kubevirt/virt-template-controller",
    )
