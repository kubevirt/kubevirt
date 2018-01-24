# Virtual Machine Device Assignment

## Scope

For the first version of the document, the scope is reduced PCI device assignment.

## Introduction

Device assignment is a feature that allows virtual machines (VMs) to take advantage of hardware capabilities and performance available on the hypervisor (host/node). Unlike traditional VM resources such as virtual CPUs (vCPUs) or memory, assignment-capable hardware devices are limited resources that cannot be overcommitted.

### Device Assignment in Kubernetes

References:

* [Kubernetes device plugins](https://kubernetes.io/docs/concepts/cluster-administration/device-plugins/)
* [device plugins work tracker (mostly interesting is the 1.11 topology work)](kubernetes/kubernetes#53497)

The first version of device assignment was added to Kubernetes in version 1.3, and worked specifically with NVIDIA GPUs. Starting in version 1.8, the feature is slowly being deprecated by system called device plugins (DPIs). Both versions of assignment share similar traits: the node exposes a scalar resource indicating quantity of the device, and pods request the resource either directly or through node affinity rules. The legacy NVIDIA plugin was only able to expose NVIDIA GPU. In case of DPI, the devices are reported and managed through separate daemon(s), where exposed devices depend on the plugin's code. Both versions of the assignment worked with assumption that the container requires several node paths and container paths. The paths contained endpoints needed for actual GPU compute.

The system is required to provide the devices in working state - having the proper driver installed and bound to the device.

### (PCI) Device Assignment in KVM/QEMU (Linux)

Device assignment in KVM/QEMU has different requirements than in case of containers that stem from the higher level of isolation provided by VMs when compared to containers.

When using VFIO (Virtual Function I/O; modern device assignment framework), the system must be IOMMU (I/O Memory Management Unit) capable. That roughly translates to having Intel VT-d or AMD-Vi virtualization extensions, the kernel booted with IOMMU module, chipset that happens to work, ACS (PCI Access Control Services) capable CPU (= business class) for proper level of isolation, and possibly more.

The concept of IOMMU deserves an explanation on it's own: from high level point of view, the devices in the system may be isolated from each other. If the isolation is provided by the IOMMU, the device or devices within that isolated area are treated as one group. Imagine a system without any isolation - in that case, every device would be in one IOMMU group. On the other hand, extreme level of isolation between devices yields one IOMMU group for each device. In practice, the number of devices in a group varies quite a lot. Consumer level GPUs usually contain at least an audio device (that handles HDMI audio), so that makes a group of 2 devices. Network interface cards' ports are usually grouped into groups of 1-4 devices.

The most important knowledge about IOMMU is that not all devices within a group have to be assigned to a VM at the same time, but any device from the group that is not assigned cannot be used by other VM and must be bound to vfio-pci driver.

For example, consider following topology:

* IOMMU group `1` contains devices `A` and `B`
* IOMMU group `2` contains device `C`
* we have 2 VMs, `α` and `β`

If `A` is assigned to `α`, we bind both `A` and `B` to vfio-pci and consider them unusable for further assignment. That means `β` cannot be started if `B` was assigned to it. We are still able to start `β` with `C`, as `C` is in separate group.

## Use Cases

* ML / AI
* video processing
* hardware / OS compatibility

## KubeVirt Device Assignment

To enable device assignment in KubeVirt, we need to respect both Kubernetes resource handling and virtualization's complex system requirements. To expose PCI devices to Kubelet, that further communicates them as node resources, we use a [DPI with those capabilities](https://github.com/mpolednik/linux-vfio-k8s-dpi). The DPI must be able to expose devices in uniform manner across nodes and make sure that each device that is required in assignment or affected by IOMMU is bound to vfio-pci.

The VFIO endpoints, for devices requested by the pod are then exposed as host/container paths (/dev/vfio/vfio, /dev/vfio/{N, M, ...} where N, M are IOMMU group IDs) to the container in the pod.

The main difficulty of this approach is reliably identifying which system device was assigned. The KubeVirt API must be verbose enough to let users request a device by it's vendor ID/product ID tuple, yet hide the actual device address. This is difficult for libvirt, as VM XML requires users to specify the device address.

Because the actual device selection is done by kube-scheduler, and system may contain multiple devices with identical vendor ID/product ID that are in the same or different IOMMU groups, we need a mechanism to inspect which IOMMU group does the device belong to.

To demonstrate the issue, we present an example.

### Example

* IOMMU group `1` contains devices
 *	`A` (vendor ID = 1000, product ID = 1000)
 * `B` (vendor ID = 2000, product ID = 2000)
* IOMMU group `2` contains
 * device `C` (vendor ID = 1000, product ID = 1000)
* we have VM `α`, and it's corresponding pod `a`

```yaml
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
metadata:
  name: α
spec:
  domain:
    devices:
      ...
      passthrough:
      - type: pci
        vendor: 1000
        device: 1000
      ...
    memory:
      unit: MB
      value: 64
    os:
      type:
        os: hvm
    type: qemu
```

In this case, KubeVirt must generate pod specification that requests device with vendor ID/product ID 1000:1000. Such specification is shown below.

```yaml
spec:
  containers:
  - name: a
    ...
    resources:
      requests:
              mpolednik.github.io/1000_1000: 1
      limits:
              mpolednik.github.io/1000_1000: 1
```

The difficult part is: how should libvirt XML generated by virt-handler look like? It is able to query the underlying system for devices and determine addresses of all devices that match vendor ID/product ID 1000:1000, but choosing a different device than the scheduler scheduled would cause the VM not to start due to IOMMU violation.

To safely identify the selected device, we have to extend `virt-launcher` to report back the IOMMU groups inside the VMs pod. This could be done via a file in a directory shared between virt-launcher and virt-handler. Knowing which IOMMU groups were assigned to the pod, virt-handler can then probe the system and find device addresses to construct the final libvirt XML.

In case of selecting N similar devices from IOMMU group containing M similar devices, where N <= M, the device ordering is not relevant.

### API

The API must be able to abstract devices to the level consumable by cloud: an identifier for device that contains enough information that the user can understand what the device is. The tuple vendor ID/product ID does exactly that - all of the device addressing is abstracted while the information can be used to query the exact model of the device (e.g. from PCI IDs).

```yaml
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
metadata:
  name: α
spec:
  domain:
    devices:
      ...
      passthrough:
      - type: pci
        vendor: 1000
        device: 1000
      ...
    memory:
      unit: MB
      value: 64
    os:
      type:
        os: hvm
    type: qemu
```
