#!/bin/bash

set -xeo pipefail

# Default values for env vars, can be overridden by user input
KVM_HYPERVISOR_DEVICE="kvm"
KVM_VIRTTYPE="kvm"

if [ -z "$HYPERVISOR_DEVICE" ] || [ -z "$PREFERRED_VIRTTYPE" ]; then
    echo "Warning: Env vars HYPERVISOR_DEVICE or PREFERRED_VIRTTYPE not set. Defaulting to KVM values for both vars"
    echo "Currently specified values: HYPERVISOR_DEVICE='$HYPERVISOR_DEVICE', PREFERRED_VIRTTYPE='$PREFERRED_VIRTTYPE'"
    HYPERVISOR_DEVICE="$KVM_HYPERVISOR_DEVICE"
    PREFERRED_VIRTTYPE="$KVM_VIRTTYPE"
fi

ARCH=$(uname -m)
MACHINE=q35
if [ "$ARCH" == "aarch64" ]; then
  MACHINE=virt
elif [ "$ARCH" == "s390x" ]; then
  MACHINE=s390-ccw-virtio
elif [ "$ARCH" != "x86_64" ]; then
  exit 0
fi

set +o pipefail

HYPERVISOR_DEV_PATH="/dev/${HYPERVISOR_DEVICE}"
HYPERVISOR_DEV_MINOR=$(grep -w ${HYPERVISOR_DEVICE} /proc/misc | cut -f 1 -d' ')
set -o pipefail

VIRTTYPE=qemu

# Device setup is best-effort: in restricted environments (e.g. rootless podman)
# we may not be able to create or chmod the device, so we fall back to software
# emulation rather than failing the init container and blocking the deployment.
if [ ! -e "$HYPERVISOR_DEV_PATH" ] && [ -n "$HYPERVISOR_DEV_MINOR" ]; then
  mknod "$HYPERVISOR_DEV_PATH" c 10 "$HYPERVISOR_DEV_MINOR" || echo "Warning: cannot create $HYPERVISOR_DEV_PATH, falling back to software emulation (virttype=qemu)"
fi

if [ -e "$HYPERVISOR_DEV_PATH" ]; then
    # Keep hardware acceleration if the device is (or can be made) writable.
    if chmod o+rw "$HYPERVISOR_DEV_PATH" || [ -w "$HYPERVISOR_DEV_PATH" ]; then
        VIRTTYPE=${PREFERRED_VIRTTYPE}
    else
        echo "Warning: cannot access $HYPERVISOR_DEV_PATH, falling back to software emulation (virttype=qemu)"
    fi
fi

if [ -e /dev/sev ]; then
  # QEMU requires RW access to query SEV capabilities; skip if we cannot chmod it.
  chmod o+rw /dev/sev || echo "Warning: cannot chmod /dev/sev, SEV capabilities may not be detected"
fi

virtqemud -d

EXPAND_CPU_FEATURES=""
if virsh domcapabilities --help 2>&1 | grep -q -- '--expand-cpu-features'; then
   EXPAND_CPU_FEATURES="--expand-cpu-features"
fi

SUPPORTED_CPU_FEATURES=""
if virsh domcapabilities --help 2>&1 | grep -q -- '--supported-cpu-features'; then
   SUPPORTED_CPU_FEATURES="--supported-cpu-features"
fi

virsh domcapabilities --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE $EXPAND_CPU_FEATURES > /var/lib/kubevirt-node-labeller/virsh_domcapabilities.xml

# hypervisor-cpu-baseline command only works on x86 and s390x
if [ "$ARCH" == "x86_64" ] || [ "$ARCH" == "s390x" ]; then
   virsh domcapabilities --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE $EXPAND_CPU_FEATURES $SUPPORTED_CPU_FEATURES | virsh hypervisor-cpu-baseline --features /dev/stdin --machine $MACHINE --arch $ARCH --virttype $VIRTTYPE > /var/lib/kubevirt-node-labeller/supported_features.xml
fi

virsh capabilities > /var/lib/kubevirt-node-labeller/capabilities.xml

# Detect cross-architecture emulation capabilities by probing for the
# cross-arch QEMU emulator via virsh domcapabilities. The resulting XML
# file is read by the node labeller to decide whether to advertise the
# cross-arch vm-arch label.
if [ "$ARCH" == "x86_64" ]; then
  virsh domcapabilities --machine virt --arch aarch64 --virttype qemu > /var/lib/kubevirt-node-labeller/virsh_domcapabilities_aarch64.xml 2>/dev/null || true
elif [ "$ARCH" == "aarch64" ]; then
  virsh domcapabilities --machine q35 --arch x86_64 --virttype qemu > /var/lib/kubevirt-node-labeller/virsh_domcapabilities_x86_64.xml 2>/dev/null || true
fi
