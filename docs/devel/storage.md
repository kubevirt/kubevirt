# Getting started with storage

This document assumes you have a working [development environment](https://github.com/kubevirt/kubevirt/blob/master/docs/getting-started.md).


**We are using hostPath PVs, which are NOT suitable for a production environment, this document assumes you are setting up a SINGLE node development environment.**

In order to use disk images one must configure an appropriate storage. In this document we will be using hostPath Persistent Volumes (PVs) and making Persistent Volume Claims against that PV. If you have some other mechanism to get PVCs you can skip the 'deploy hostPath PVC provisioner' section of this document. Throughout the entire document, I am assuming we are deploying into the 'default' namespace. You can also deploy into a different namespace if you want.

## Create hostpath directory on node.
The hostPath provisioner needs to write to a directory on the node, we will need to create this directory before installing the hostPath provisioner

```bash
cluster/cli.sh ssh node01 -- sudo mkdir /var/run/kubevirt/hostpath
```

## Deploy hostPath PVC provisioner

We are using the hostPath provisioner from [here](https://github.com/MaZderMind/hostpath-provisioner). Simply run from :

```bash
cluster/kubectl.sh apply -f docs/devel/hostpath-provisioner.yaml
```

This will create
- hostpath-provisioner Service Account
- hostpath-provisioner Cluster Role
- hostpath-provisioner Cluster Role Binding
- hostpath-provisioner Deployment
- Create hostpath Storage Class, set as the default, so any Persistent Volume Claims (PVCs) will be serviced by this storage class

After a few moments the hostpath-provisioner pod should be running, we are now ready to create PVCs against it. Any requested PVCs will be stored in `/var/run/kubevirt/hostpath/<pvc-name>-<hash>`. The PV retention policy will be `delete`, which means that when you delete the PVC, the matching PV is also removed and the directory created on the host is removed as well.

## Deploy CDI controller

The Containerized Data Importer controller is a controller that watches for PVCs with a specific annotation. If that annotation is detected, it will use the 'storage.image.endpoint' URL to download the image, convert it if needed and write it to the requested PVC. One can then use the PVC to start a VM using the image in it. To deploy the CDI controller run this:

```bash
cluster/kubectl.sh apply -f https://raw.githubusercontent.com/kubevirt/containerized-data-importer/master/manifests/controller/cdi-controller-deployment.yaml
```

This will create
- cdi-sa Service Account
- cdi Cluster Role
- cdi-sa Cluster Role Binding
- cdi-deploy Deployment

After a few moments the cdi-deployment pod should be running, we are now ready to import images using a PVC annotation.

## Create a PVC with annotation

Now that we can request PVCs and we have the CDI controller running, we are ready to import an image. An example yaml file looks like this:
```YAML
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: "image-pvc"
  labels:
    app: containerized-data-importer
  annotations:
    cdi.kubevirt.io/storage.import.endpoint: "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img"   # Required.  Format: (http||s3)://www.myUrl.com/path/of/data
    cdi.kubevirt.io/storage.import.secretName: "" # Optional.  The name of the secret containing credentials for the data source
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Mi
```

As you can see we are creating a PVC named 'image-pvc' with 2 annotations. In this case we are requesting the cirros image because it's a small image, but any http or s3 end point will do. If you need to provide a secret the second annotation can be used to provide it. In our example we don't need a secret.

The CDI controller will see the annotation and spawn an 'importer' pod that does the actual work. Once the pod is finished there will be a disk.img in the PVC.

## Create a VM using the image

Now that we have a populated PVC, we can create a VM using the disk image. For instance using this yaml:
```YAML
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstance
metadata:
  creationTimestamp: null
  name: vm-pvc-cirrus
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: virtio
        name: pvcdisk
        volumeName: pvcvolume
    machine:
      type: ""
    resources:
      requests:
        memory: 64M
  terminationGracePeriodSeconds: 0
  volumes:
  - name: pvcvolume
    persistentVolumeClaim:
      claimName: image-pvc
status: {}
```

You can now use the steps described in the [development getting started guide](https://github.com/kubevirt/kubevirt/blob/master/docs/getting-started.md) to connect a VNC console to it.
