This document describes how to associate a k8s secret with a VMI for the purpose of iscsi initiator authentication.

*NOTE: Only client authentication is supported at this time, meaning that the iscsi target with authenticate an initiator has permissions to access a device, but that a initiator can not authenticate the target.*

## Workflow
1. create a k8s secret containing the password and username fields.

The k8s secret must be formatted in the same way kubernetes performs iscsi
authentication for volumes. https://github.com/kubernetes/examples/blob/master/volumes/iscsi/chap-secret.yaml

```
cat << END > my-chap-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: myK8sSecretID
  namespace: default

apiVersion: v1
kind: Secret
metadata:
  name: my-chap-secret
type: "kubernetes.io/iscsi-chap"  
data:
  node.session.auth.username: $(echo "myUsername" | base64 -w0)
  node.session.auth.password: $(echo "mySuperSecretPassword" | base64 -w0)
END
```
2. create a vm that references the **name** given to the k8s secret in the iscsi usage field
```
cat << END > my-vm.yaml
kind: VirtualMachine
metadata:
  name: testvm
  namespace: default
spec:
  domain:
    devices:
      disks:
      - auth:
          secret:
            type: iscsi
            usage: my-chap-secret
        type: network
        snapshot: external
        device: disk
        driver:
          name: qemu
          type: raw
          cache: none
        source:
          host:
            name: iscsi-demo-target.default
            port: "3260"
          protocol: iscsi
          name: iqn.2017-01.io.kubevirt:sn.42/2
        target:
          dev: vda
    memory:
      unit: MB
      value: 64
    os:
      type:
        os: hvm
    type: qemu
END
```
3.  Add the secret and vm to the cluster.
```
kubectl create -f my-chap-secret.yaml
kubectl create -f my-vm.yaml
```

From there, the password and username fields in the k8s secret will automatically be mapped to a libvirt secret when the VMI is scheduled to a node allowing the iscsi auth to work without any further configuration. 

