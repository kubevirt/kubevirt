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

To use a new container disk image in tests:

1. Build and push your image to a container registry
2. Add the image reference to `tests/containerdisks/containerdisks.go`
3. Use the image in your test

For official images, reach out to kubevirt CI maintainers to upload the image
to the official registry.
