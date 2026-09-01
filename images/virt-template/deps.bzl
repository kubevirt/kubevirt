"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:bec24e53a36a9742ae931dcc596debd3617174bf7f373a7466b4234dca68f294"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:23c240c530ce64f39d3e89e93b83b53ea269ddf2cdd9ff97b7fbfd93edd2d080"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:fa4e578e7187c9a8c278a07430bda54206a58714e1b454ff5c23d6da45fc7529"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:ee7fdc21e77e958e8e8fc80c5ee81487b14625d10be65b0218492e435fb0a526"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:38a3700d6e5e2ea5e1eafac7de5178443a30e4606b1b6458cc906bc573d58f52"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:1f8a5776871a3bd16062bde3fcdef06188b74040133722d6919638dbc4d2d2c5"

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
