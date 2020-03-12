# ConfigMaps
HCO creates ConfigMaps on deployment for supplying default values configuration.

## Storage ConfigMap
Defines default values for storage specs: 'accessMode' and 'volumeMode'.

### Example
```
---
kind: ConfigMap
apiVersion: v1
metadata:
  name: kubevirt-storage-class-defaults
  namespace: openshift
data:
  accessMode: ReadWriteMany
  volumeMode: Filesystem  # (Block for BareMetal infrastructure)
  local-sc.accessMode: ReadWriteOnce
  local-sc.volumeMode: Filesystem
```

# Config BareMetal platform
Use 'BareMetalPlatform' spec when creating HyperConverged to enable BareMetal infrastructure.
This will result in 'volumeMode: Block' in storage ConfigMap.

# Config local storage class name
Use 'LocalStorageClassName' spec to specify the name of the local class name.

## Example
```
apiVersion: hco.kubevirt.io/v1alpha1
kind: HyperConverged
metadata:
  name: hyperconverged-cluster
spec:
  BareMetalPlatform: true
  LocalStorageClassName: "local-sc"
```
