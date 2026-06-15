# KubeVirt container-disk images

The following images are published to [quay.io/kubevirt](https://quay.io/organization/kubevirt)
and can be used in KubeVirt tests. The legacy `dockerhub.com/kubevirt` organization is no
longer maintained.

- **alpine-container-disk-demo**
   Lightweight image for test suite.

- **cirros-container-disk-demo**
    Lightweight image for test suite.

- **cirros-custom-container-disk-demo**
    Used for e2e testing of custom base paths.

- **virtio-container-disk**
    Windows virtio drivers image.

- **alpine-ext-kernel-boot-demo**
    Alpine image with kernel boot support.

- **alpine-with-test-tooling-container-disk**
    Alpine image preconfigured with CI tooling.

- **fedora-with-test-tooling-container-disk**
    Fedora image preconfigured with CI tooling.

- **fedora-realtime-container-disk**
    Fedora image with realtime kernel support.

- **s390x-guestless-kernel**
    s390x kernel boot image.

## How to create new customized container-disk for tests

Customizing an image can be done by:
- Editing the image with `virt-customize`
  Adding a new package to an image, for example:
  `virt-customize -a "fedora39.qcow2" --install dpdk`

- Spin up a live VM with the image you want using `virt-install`
  Once the VM is up apply the changes you want and when done
  shut down the VM gracefully with `sudo shutdown -h now`

Next, it is necessary to prepare the image so there will be no unique files or configurations
so each VM that will be created with the new image will have unique machine-id, mac-address etc..
We can do that with `virt-sysprep`:
 ```bash
 virt-sysprep -a fedora39.qcow2 --operations machine-id,bash-history,logfiles,tmp-files,net-hostname,net-hwaddr
 ```

Once the image is ready it is necessary to convert it to
a container image so KubeVirt VMs can consume it according to
[docs/container-register-disks.md](../docs/container-register-disks.md)

```bash
cat > Dockerfile <<EOF
FROM scratch
ADD --chown=107:107 fedora39.qcow2 /disk/
EOF

podman build -t quay.io/kubevirt/fedora-custom-testing:latest .
```


## Use image in KubeVirt tests

First we need to pull the image from the remote registry (or local registry) by adding `oci_pull` rule to `WORKSPACE` file:
```python
oci_pull(
    name = "fedora_custom_testing",
    # digest = ""
    image = "localhost:5000/kubevirt/fedora-custom-testing",
    tag = "latest",
)
```
Once you verified the image works reach out to kubevirt CI maintainers and ask to upload the new image
then update the `oci_pull` rule accordingly.
```python
oci_pull(
    name = "fedora_custom_testing",
    digest = "sha256:<digest>",
    image = "quay.io/kubevirt/fedora-custom-testing",
)
```

Next we need to add an `oci_image` rule for the new image to the `containerimages/BUILD.bazel` file.
```python
oci_image(
    name = "fedora-custom-container-disk-image",
    base = select({
        "@io_bazel_rules_go//go/platform:linux_arm64": "@fedora_custom_testing_aarch64",
        "//conditions:default": "@fedora_custom_testing",
    }),
    visibility = ["//visibility:public"],
)
```

Then add an `oci_push` rule in `BUILD.bazel`:
```python
oci_push(
    name = "push-fedora-custom-container-disk",
    image = "//containerimages:fedora-custom-container-disk-image",
    repository = "quay.io/kubevirt/fedora-custom-container-disk",
)
```

Finally, in order to use the image in test suite it is necessary to add it to `tests/containerdisk/containerdisk.go` file.
