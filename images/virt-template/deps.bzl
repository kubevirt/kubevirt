"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:1284f3a1e3cdfd6af3600ed321b5b186de3c5469dd6c9d471437e3ac81c98064"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:5f458945a539423b5afbf21bdd7028d6b1181860e370a6fbcd0eded420c5a079"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:5bedc3a902ff5ef6d4faca1b7cd61fcd7aa3701c59c3955e0e9d609036e0fb38"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:23d93c1cb5b9b051bed7efb2f6491e11d665a0b9c7a796fbe4b248d46b60b2c0"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:56c075fd8b127d10f55ec941cb4abcbde1d311dd9b1ac6253823f9521f73e934"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:46f3a16bc2c61913de88188550fea17cf5c834797e1a06e86c7c39712de26f54"

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
