load("@rules_img//img:image.bzl", "image_manifest")
load("@rules_img//img:layer.bzl", "layer_from_tar")

def kubevirt_image(name, tars = [], layers = [], **kwargs):
    """Creates an image_manifest, optionally auto-generating layer_from_tar targets.

    For each tar in `tars`, a layer_from_tar target is automatically created
    and prepended to `layers`. Use `layers` to include pre-existing layer
    targets alongside the auto-generated ones.

    Args:
        name: Name of the image_manifest target.
        tars: List of tar targets to auto-convert into layers.
        layers: Pre-existing layer targets to include after auto-generated ones.
        **kwargs: All other args forwarded to image_manifest
                  (base, entrypoint, user, visibility, testonly, etc.).
    """
    testonly = kwargs.get("testonly")
    auto_layers = []
    for i, tar in enumerate(tars):
        layer_name = "%s_layer_%d" % (name, i) if len(tars) > 1 else "%s_layer" % name
        if testonly:
            layer_from_tar(name = layer_name, src = tar, testonly = testonly)
        else:
            layer_from_tar(name = layer_name, src = tar)
        auto_layers.append(":" + layer_name)

    all_layers = auto_layers + layers
    if all_layers:
        kwargs["layers"] = all_layers

    image_manifest(name = name, **kwargs)
