"""Dependencies for virt-template images."""

load("@rules_img//img:pull.bzl", "pull")

# Image digests for virt-template-apiserver
VIRT_TEMPLATE_APISERVER_DIGEST_AMD64 = "sha256:c98bf1c7561392c0d8de49ceb73bf14a8024f70b5c5e37db9b7b199852198502"
VIRT_TEMPLATE_APISERVER_DIGEST_ARM64 = "sha256:a3445b50d961eb29cc9f371de1715d62ab75f84f5979970ddf995465b62ce402"
VIRT_TEMPLATE_APISERVER_DIGEST_S390X = "sha256:78e2eb9d64063824ebd1992217bc8d653a2fce21b9238e82b81843a99ccee66c"
VIRT_TEMPLATE_APISERVER_DIGEST_PPC64LE = "sha256:4bc90b5be5ceedbb5c677c9b1ce62401b5d8e82b2b636720eb4332b771d75ee8"

# Image digests for virt-template-controller
VIRT_TEMPLATE_CONTROLLER_DIGEST_AMD64 = "sha256:d9ada5c6565b7f804c9785c53b364f902aef5dba73ca0535f1c04548afaa3333"
VIRT_TEMPLATE_CONTROLLER_DIGEST_ARM64 = "sha256:c0c28e3f44aaa5fb43f14880714bb90e999e2f274c22e4a17482f823d5cff92d"
VIRT_TEMPLATE_CONTROLLER_DIGEST_S390X = "sha256:86616e87f1ef496c5227f95d8644fefd4b29f4cc85d56b1b92294e7bf87c560d"
VIRT_TEMPLATE_CONTROLLER_DIGEST_PPC64LE = "sha256:cd0a603418f89a26be66a1b136e2de2be5ed97cb1aa7f98e35a7fb3c6d0790a5"

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
        name = "virt_template_apiserver_ppc64le",
        digest = VIRT_TEMPLATE_APISERVER_DIGEST_PPC64LE,
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

    pull(
        name = "virt_template_controller_ppc64le",
        digest = VIRT_TEMPLATE_CONTROLLER_DIGEST_PPC64LE,
        registry = "quay.io",
        repository = "kubevirt/virt-template-controller",
    )
