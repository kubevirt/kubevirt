"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:104545cf22d9d6ea0d675b650742f07fbffad18bdae5454e8ad79c93e0fa3eeb"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:04b06a30b7c03994ec5c2dcd3effbefffde9f2818483dc9273035faec21e4336"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:db633d78bf352b3109888c1d194507c2cc8940ae911fe550ee3390e4f311163a"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:1d7a44eeb45987852f796d1f4ac6a3ab9bed7ee9bfa553bb093e3b114dc72c9c"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:c29b7231aee92c56ebaea55ca6472c41857757258d0c18037be54dd216226601"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:007ee285eb7a789043babf323458395402dfc70c7e0b4ca3a76c2be3e42b1e2a"

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
