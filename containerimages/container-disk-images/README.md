# Customize VM image with cloud-init

To customize a VM image execute `create-push-vm-image.sh` script.
What this script does is:
- Create could-init ISO file from the `user-cloud-config` file
- Create VM with cloud-init attached disk
- Export the VM image to qcow2 file

This script uses cloud-init in order to customize the VM image,
edit `user-cloud-config` file with the changes you would like
to apply according to cloud-config API:
https://cloudinit.readthedocs.io/en/latest/topics/examples.html

this script requires:
- cloud-utils
- virt-install
- qemu-img
    
TODO: convert this script to a container

Once executed you should have a login prompt to the VM.
If extra steps needed login with username fedora and password fedora, execute whats needed.
When done shutdown the VM:
```bash
sudo shutdown -h now
```

Example:
```bash
# Pass the source VM image file path, and the path to save
image_path="fedora.qcow2"
new_image_path="fedora-extended.qcow2"

./customize-vm-image.sh $image_path $new_image_path
```

### Build container-disk image

To build container-disk image form qcow2 image
file, use `build-container-disk.sh` script

Example:
```bash
image_name='fedora-extended'
tag='tests'
vm_image=source-image.qcow2

./build-container-disk.sh $image_name $tag $vm_image
```
This script also exports the container image to .tar
file, so it will be easier to store or send.

### Push container-disk image

In order the new container-disk image use
`publish-container-disk.sh` 

Example:
```bash
image_tag="fedora-extended:tests"
target_tag="docker.io/kubevirt/fedora-extended:tests"

./publish.sh $image_tag $target_tag
```

Push the new image to cluster registry:
```bash
# From kubevirt directory
cluster_registry="localhost:$(./cluster-up/cli.sh ports registry | tr -d '\r')"

cd  containerimages/container-disk-images
image_tag="fedora-extended:tests"
target_tag="${cluster_registry}:fedora-extended:tests"

./publish.sh $image_tag $target_tag
```

# Automation script

In order to automate the process of customizing VM image,
build container-disk from it and push it to a registry use
`create-push-vm-image.sh` script.

What this script does:
- Download VM image from given URL
- Customize VM image using `customize-image.sh` script, according to `user-cloud-config` file.
  Keeping customizing image method loosely-coupled, so it will be easy to maintain.
- Building container-disk image using `build-container-disk.sh`
  according to the doc: 
  https://github.com/kubevirt/kubevirt/blob/master/docs/container-register-disks.md
- Push the new container-disk image to registry using `publish-container-dish.sh`

This script requires the following packages:
- cloud-utils
- docker-ce
- virt-install
- qemu-img

### How to use:
```bash
# Export image name and tag:
export IMAGE_NAME=fedora-extended
export TAG=32

# Export VM image URL:
export IMAGE_URL=https://download.fedoraproject.org/pub/fedora/linux/releases/32/Cloud/x86_64/images/Fedora-Cloud-Base-32-1.6.x86_64.qcow2

cd containerimages/container-disk-images
./create-push-vm-image.sh
```
