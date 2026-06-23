"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:ccaf054ecf4cc35c92960eced366041c59932262665e43b933cf112615f03dfc"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:f60f4e0da3c01eeead20a371116b1f04de18047e24e275624ab9ce5cad864f00"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:8c408743fc7a9ecf5ab6479d3869274ff46dff7617ecf28fecfe54266ff38b23"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:27c1404e236611ac24b387144c40645685567adfec19554bb2bc2915407fa7fd"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:8f25d1f131091369014d122257f86d099fce8290041bdb923bf62e3b9d4b4d84"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:a42bca5a15e0752ce8aa92daf986f6d1a8e561229698469d1c6aaffef010e930"

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
