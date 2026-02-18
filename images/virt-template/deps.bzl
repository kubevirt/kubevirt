"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:c3b20cd9bc83cc9065998b78cffe1c1cea323231ad1c5678aeefe82a5d172846"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:dc6f6773fe84412f1a6cd6087523e44c6727d6e519f1223ea96e4d607aa90c85"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:0a2891de175e312a6ec9f4a4670c11e713b3a0ead2ff8c6bad30ff72e71bcf7b"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:e68cf77970e57aaec88ead5da9862a83632c5ba35ee3e4511355b6fbd7d421a9"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:54122356de3461714e3dbbf2942bb6182e48b294f89ba08e57472cfc777534c5"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:c178ec45bf32c9553f6e962e020cdfed427e31d20a0c1fda9f8e148c58d39504"

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
