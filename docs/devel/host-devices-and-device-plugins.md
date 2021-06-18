## Host Devices Assignment

KubeVirt provides a mechanism for assigning host devices to a virtual machine. This mechanism is generic and allows various types of PCI devices, such as GPU or any other devices attached to a PCI bus, to be assigned. It also allows Mediated devices, such as pre-configured virtual GPUs to be assigned using the same mechanism.

NOTE: This document doesn't cover KubeVirt's [SR-IOV Support](https://github.com/kubevirt/kubevirt/blob/main/docs/sriov.md).
### Permitting Host Devices to be used in the cluster:

Administrators can control which host devices will be permitted for use in the cluster.
Permitted host devices in the cluster will need to be listed in KubeVirt CR by its `vendor:product` selector for PCI devices or mediated device names

```
configuration:
  permittedHostDevices:
    pciHostDevices:
    - pciVendorSelector: "10DE:1EB8"
      resourceName: "nvidia.com/TU104GL_Tesla_T4"
    - pciVendorSelector: "8086:6F54"
      resourceName: "intel.com/qat"
    mediatedDevices:
    - mdevNameSelector: "GRID T4-1Q"
      resourceName: "nvidia.com/GRID_T4-1Q"
```

### Device plugins for host devices assignment in KubeVirt

KubeVirt provides integrated generic device plugins for the assignment of PCI and Mediated devices.
These device plugins can discover, allocate and provide basic monitoring.
Any host device that is bound to a VFIO driver and permitted for use in the cluster can be assigned to a virtual machine.

KubeVirt can also assign host devices allocated by "external" device plugins, such as the NVIDIA GPU device plugin for KubeVirt.

### KubeVirt interface for device plugins

To assign the allocated devices to virtual machines, KubeVirt expects the device plugins to provide a list of allocated devices via an environment
variables that encode the name of the resource with its relevant type.

The prefixes are PCI_RESOURCE_ for PCI devices and MDEV_PCI_RESOURCE_ for MDEVs.

Here is an example of an expected naming of the variables:
```
PCI_RESOURCE_NVIDIA_COM_TU104GL_Tesla_T4=PCIADDRESS1
```
This encodes a PCI device with its resource name (provided in the KubeVirt CR) "nvidia.com/TU104GL_Tesla_T4"
```
PCI_RESOURCE_INTEL_QAT=PCIADDRESS2,PCIADDRESS3,...
MDEV_PCI_RESOURCE_NVIDIA_COM_GRID_T4-1Q=UUID1,UUID2,UUID3,...
```
Both the internal and the external device plugins are expected to follow the same naming convention.

### Starting a Virtual Machine
HostDevices, as well as the existing GPUs field, will be able to reference both PCI and Mediated devices

```
kind: VirtualMachineInstance
spec:
  domain:
    devices:
      gpus:
      - deviceName: nvidia.com/TU104GL_Tesla_T4
        name: gpu1
      - deviceName: nvidia.com/GRID_T4-1Q
        name: gpu2
      hostDevices:
      - deviceName: intel.com/qat
        name: quickaccess1
```
