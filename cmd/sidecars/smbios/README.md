# KubeVirt SMBIOS hook sidecar

To use this hook, use following annotations:

```yaml
annotations:
  # Request the hook sidecar
  hooks.kubevirt.io/hookSidecars: '[{"image": "registry:5000/kubevirt/example-hook-sidecar:devel"}]'
  # Overwrite base board manufacturer name
  smbios.vm.kubevirt.io/baseBoardManufacturer: "Radical Edward"
```

## Example

```shell
# Create a VM requesting the hook sidecar
cluster/kubectl.sh create -f examples/vmi-with-sidecar-hook.yaml

# Once the VM is ready, connect to its display and login using name and password "fedora"
cluster/virtctl.sh vnc vm-with-sidecar-hook

# Install dmidecode
sudo dnf install -y dmidecode

# Check whether the base board manufacturer value was successfully overwritten
sudo dmidecode -s baseboard-manufacturer
```
