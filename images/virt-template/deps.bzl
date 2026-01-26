"""Dependencies for virt-template images."""

load("@rules_oci//oci:pull.bzl", "oci_pull")

# Image digests for virt-template components
VIRT_TEMPLATE_APISERVER_DIGESTS = {
    "amd64": "sha256:9b5d7fde015a467cf11cf84a63639a5457897fc77bbc7324fc6632e08124ae12",
    "arm64": "sha256:1933e0f78b0a58bb195fb450b629fe5bd12cad50a0fa18eb702ef69872a07462",
    "s390x": "sha256:135695e94e90145e2fa986982b3098ac720f73c0cc5376f44f11e3645127ebbc",
}

VIRT_TEMPLATE_CONTROLLER_DIGESTS = {
    "amd64": "sha256:910a8fffa74571e299075a1436db553f529f9b717ff979a3200e8cfcef92009f",
    "arm64": "sha256:1e2c35d07089bc5ddd14e3457410d9817fe0745cbf9321b017ebfd36e748c0dc",
    "s390x": "sha256:304a1df0a1671e2d1ecc7b9970910dafcce7f530f94c822cf50c52316ac6382a",
}

def virt_template_images():
    """Pull virt-template images for all architectures."""
    oci_pull(
        name = "virt_template_apiserver",
        digest = VIRT_TEMPLATE_APISERVER_DIGESTS["amd64"],
        image = "quay.io/kubevirt/virt-template-apiserver",
    )

    oci_pull(
        name = "virt_template_apiserver_aarch64",
        digest = VIRT_TEMPLATE_APISERVER_DIGESTS["arm64"],
        image = "quay.io/kubevirt/virt-template-apiserver",
    )

    oci_pull(
        name = "virt_template_apiserver_s390x",
        digest = VIRT_TEMPLATE_APISERVER_DIGESTS["s390x"],
        image = "quay.io/kubevirt/virt-template-apiserver",
    )

    oci_pull(
        name = "virt_template_controller",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGESTS["amd64"],
        image = "quay.io/kubevirt/virt-template-controller",
    )

    oci_pull(
        name = "virt_template_controller_aarch64",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGESTS["arm64"],
        image = "quay.io/kubevirt/virt-template-controller",
    )

    oci_pull(
        name = "virt_template_controller_s390x",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGESTS["s390x"],
        image = "quay.io/kubevirt/virt-template-controller",
    )
