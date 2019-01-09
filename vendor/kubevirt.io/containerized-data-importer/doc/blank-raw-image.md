# How to create Blank Raw Image User Guide
The purpose of this document is to show how to create a new blank raw image in your PersistentVolumeClaim resource.

## Prerequesites
You have a Kubernetes cluster up and running with CDI installed and at least one PersistentVolume is available.

## Create Blank Raw Image with DataVolume manifest

Create the following DataVolume manifest (blank-image-datavolume.yaml):

```bash
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: blank-image-datavolume
spec:
  source:
      blank: {}
  pvc:
    # Optional: Set the storage class or omit to accept the default
    # storageClassName: "hostpath"
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 500Mi
```

Deploy the DataVolume manifest:

```bash
kubectl create -f blank-image-datavolume.yaml
```

An importer pod will be spawned and the new image will be created on your PV.

Another way to create the image is by creating a PVC (blank-image-pvc.yaml) and deploy it:

```bash
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "blank-image-pvc"
  labels:
    app: containerized-data-importer
  annotations:
    cdi.kubevirt.io/storage.import.source: "none"
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 4Gi
  # Optional: Set the storage class or omit to accept the default
  # storageClassName: local
```

Deploy the PVC:

```bash
kubectl create -f blank-image-pvc.yaml
```







