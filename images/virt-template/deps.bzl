"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:f052d1ff68c11991363b2367f8c9f622276b9c85b4a5a023422c62ec96052bf4"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:819ba032c6478f7e2e88abdf216e7fa9cc03ce8bf9ba5e67f5f747714a8064a2"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:5684c5fbda76bbb8779cc63dd621b8009c7ddd8f8a6cd6842c3162c4511bf134"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:481bdd4e14f27bec0fbcd3f77da397f3cd4367d8065f40aef9312db602526449"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:afcab3b4b3370c343927fdc2eb756fb8ca009827336a79b760249cc74aa6bd24"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:4c3a60af6f688f3ef50a43d57d38de6e69a6faf26eacf535f9fb2251da167a4f"

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
