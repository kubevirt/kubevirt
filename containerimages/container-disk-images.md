# KubeVirt container-disk images
The following images stored at `dockerhub.com/kubevirt` and can be used in Kubevirt tests:

- **alpine-container-disk-demo**
   Lightweight image for test suite.

- **cirros-container-disk-demo**
    Lightweight image for test suite.

- **cirros-custom-container-disk-demo**
    Used for e2e testing of custom base paths.

- **fedora-cloud-container-disk-demo**
    Fedora cloud edition image.

- **fedora-sriov-lane-container-disk**
    Fedora cloud edition image with contains all necessary configuration and drivers for sriov lane tests.    
    - This image contained the packages:  
        - kernel-modules (includes sriov drivers)  
        - qemu-guest-agent  
    - Configurations:  
        - Enable and start qemu-guest-agent  
        - Load kernel modules needed for sriov  
          mlx4, mlx5, i40e, igb  

## How to create new customized container-disk for tests

Work-in-progress PR to automate this process https://github.com/kubevirt/kubevirtci/pull/428

Customize an image can be done by:
- Editing the image with `virt-customize`  
  Adding new package to an image for example:  
  `virt-customize -a "fedora32.qcow2" --install dpdk`

- Spin-up live VM with the image you want using `virt-install`  
  Once the VM is up apply the changes you want and when done
  shut down the VM gracefully with `sudo shutdown -h now`

Next, it is necessary to prepare the image so there will be no unique files or configurations   
so each VM that will be created with the new image will have unique machine-id, mac-address etc.. 
We can do that with `virt-sysprep`:
 ```bash
 virt-sysprep -a fedora32.qcow --operations machine-id,bash-history,logfiles,tmp-files,net-hostname,net-hwaddr  
 ```
￼

Once the image is ready it is necessary to convert it to   
container image with `kubevirt/container-disk-v1alpha` layer, 
so KubeVirt VM's can consume it according to  
https://github.com/kubevirt/kubevirt/blob/main/docs/container-register-disks.md

```bash
cat > Dockerfile <<EOF
FROM kubevirt/container-disk-v1alpha
ADD fedora32.qcow2 /disk/
EOF

docker build -t kubevirt/fedora-sriov-testing:latest .
```


## Use image in Kubevirt tests

First we need pull the image from the remote registry (or local registry) by adding `container_pull` rule to `WORKSPACE` file:
```bash
container_pull(
    name = "fedora_sriov_lane",
    # digest = ""
    registry = "localhost:5000",
    repository = "kubevirt/fedora-sriov-testing",
    tag = "latest",
)
```
Once you verified the image works reach out to kubevirt CI maintainers and ask to upload the new image 
then update the `container_pull` rule accordingly.
```bash
container_pull(
    name = "fedora_sriov_lane",
    digest = ""
    registry = "index.docker.io",
    repository = "kubevirt/fedora-sriov-testing",
    # tag = "32",
)
```

Next we need to add `contaier_image` rule for the new image to `containerdisks/BUILD.bazel` file;
```bash
container_image(
    name = "fedora-sriov-lane-container-disk-image",
    architecture = select({
        "@io_bazel_rules_go//go/platform:linux_arm64": "arm64",
        "//conditions:default": "amd64",
    }),
    base = select({
        "@io_bazel_rules_go//go/platform:linux_arm64": "@fedora_sriov_lane_aarch64//image",
        "//conditions:default": "@fedora_sriov_lane//image",
    }),
    visibility = ["//visibility:public"],
)
```

Then add new line for the container_bundle rule at the pojects`BUILD.bazel` file
```bash
container_bundle(
    name = "build-other-images",
    images = {
        ...
        "$(container_prefix)/$(image_prefix)fedora-sriov-lane-container-disk:$(container_tag)": "//containerimages:fedora-extended-container-disk-image",
        ...
    }
)
```

Finally, in order to use the image in tests suite it is necessary to add it to `tests/containerdisks/containerdisks.go` file.
