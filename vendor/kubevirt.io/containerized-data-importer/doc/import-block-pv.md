# How to import an image to a block raw PV User Guide
The purpose of this document is to show how to import an image to a raw block PV.
For now, this functionality only works for importing an image from a URL (http/s3) without authentication.

## Prerequisites
- You have a Kubernetes cluster up and running with CDI installed and at least one PersistentVolume is available.
- Feature-Gate 'BlockVolume' is enabled.

## Import an image to a Local raw block PV
In case you do not import an image to a local PV, please skip to the next section (Import an image with DataVolume manifest).
There are a few steps to follow, to be able to import an image to a local raw block PV.

First, create a local PV with the volumeMode field set to 'Block':

```bash
kind: PersistentVolume
apiVersion: v1
metadata:
  name: import-block-pv
  annotations:
spec:
  nodeAffinity:
    required:
      nodeSelectorTerms:
      - matchExpressions:
        - key: kubernetes.io/hostname
          operator: In
          values:
          - node01
  volumeMode: Block  # This is Block PV
  storageClassName: local
  capacity:
    storage: 1Gi
  local:
    path: /dev/loop10   # This is the local path on the node where we import the image to
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Delete 
```

Then, create the local block device on the node itself (in that case node01 as defined in the PV nodeAffinity), by running the following commands as root on the node:
```bash
dd if=/dev/zero of=loop10 bs=100M count=10
```
```bash
losetup /dev/loop10 loop10
```

Note: the path you create on the node has to be the same path that you define in the PV.


## Import an image with DataVolume manifest

Create the following DataVolume manifest (import-block-pv-datavolume.yaml):

```bash
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: import-block-pv-datavolume
spec:
  # Optional: Set the storage class or omit to accept the default
  storageClassName: local
  source:
      http:
         url: "http://distro.ibiblio.org/tinycorelinux/9.x/x86/release/Core-current.iso"
  pvc:
    volumeMode: Block
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi  
```

Deploy the DataVolume manifest:

```bash
kubectl create -f import-block-pv-datavolume.yaml
```

An importer pod will be spawned and the new image will be created on your data volume.








