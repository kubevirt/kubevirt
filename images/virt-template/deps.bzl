"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:fbeaa4de09cdb4177a71d65e8dc952e03f431a3e9820d9b7d021e5d9b701064f"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:c03684e65e6e179fe4222bd0dbe9d68455067f80caea17bbfe48c39663eac209"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:d020ff395abc3a75c3f4d4983799126163d46c580d3a625902fa669c86b5672a"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:19a1e9b9614e1160d22ef6af6058c24ef52cea40b2f9dbfab61ee36320594222"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:b23854bb88705729f314045aa465c4fc674c3b3bf82d3a38633e33358d7bc13a"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:64ead4ca6aacac8532e7eb0605e4bb1b44ab58a1ce468337c9d5cfd88159bc51"

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
