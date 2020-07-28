#!/usr/bin/env bash
set -xe

IMAGE_PATH=$1
NEW_IMAGE=$2

readonly CLOUD_CONFIG_PATH="user-cloud-config"
readonly DOMAIN_NAME="provision-vm"
readonly CLOUD_INIT_ISO="cloudinit.iso"

# Create cloud-init user data ISO
cloud-localds $CLOUD_INIT_ISO $CLOUD_CONFIG_PATH

# Provision the VM by running could-init (ctrl+] to exit)
virt-install \
  --memory 2048 \
  --vcpus 2 \
  --name $DOMAIN_NAME \
  --disk $IMAGE_PATH,device=disk \
  --disk $CLOUD_INIT_ISO,device=cdrom \
  --os-type Linux \
  --os-variant fedora30 \
  --virt-type kvm \
  --graphics none \
  --network default \
  --import

# Remove VM
virsh destroy $DOMAIN_NAME || :
virsh undefine $DOMAIN_NAME

# Remove cloud-init image
rm -rf $CLOUD_INIT_ISO

# Convert image"
qemu-img convert -c -O qcow2 "$IMAGE_PATH" "$NEW_IMAGE"
