# ConfigMaps
HCO creates ConfigMaps on deployment for supplying default values configuration.

# Config local storage class name
Use 'LocalStorageClassName' spec to specify the name of the local class name.

## Example
```
apiVersion: hco.kubevirt.io/v1beta1
kind: HyperConverged
metadata:
  name: hyperconverged-cluster
spec:
  LocalStorageClassName: "local-sc"
```
