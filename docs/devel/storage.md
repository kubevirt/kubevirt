# Getting started with storage

This document assumes you have a working [development environment](https://github.com/kubevirt/kubevirt/blob/master/docs/getting-started.md). Kubernetes allows for a wide variety of storage. This document explains how to use some of those using local storage options.

- [hostPath based storage](#hostpath-based-storage)
- [local volume based storage](#local-volume-based-storage)

If you already have storage configured you can skip to deploying the [Containerized Data Importer](#deploy-cdi-controller) (CDI). It is adviced to pick one of the available options for your development environment as the yaml files will try to configure them as the default storage.

# hostPath based storage
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

# Local volume based storage

Local volume is in Beta since Kubernetes 1.10 and thus no longer requires you to enable it through the [feature-gates](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) parameter of the kubelet. To enable local volume for the development environment please follow the following steps. Note local volumes WILL work on multiple nodes due to the fact that PVCs don't get bound until a POD actually uses the PVC, at which point the local volume is bound on the node the pod is running on. The following steps are based on information found in the [Local Persistent Storage User Guide](https://github.com/kubernetes-incubator/external-storage/blob/master/local-volume/README.md)

## Preliminary steps

Since disks or volumes cannot be added to a node after a container is started (nodes are running in containers in the development environment). The mount source will have to be some directories that exist in the container. This can be achieved by using the --bind option when mounting. This example is creating 3 mount disks, but its possible to add or remove as many as one wants, as long as there is at least 1.

Make some directory based mount points:

```bash
$ ./cluster/cli.sh ssh node01
$ sudo su
$ mkdir /mnt/local-storage
$ mkdir /mnt/local-storage/dev
$ mkdir /mnt/local-storage/dev/disk1
$ mkdir /mnt/local-storage/dev/disk2
$ mkdir /mnt/local-storage/dev/disk3
$ mkdir /root/disk1
$ mkdir /root/disk2
$ mkdir /root/disk3
$ mount --bind /root/disk1 /mnt/local-storage/dev/disk1
$ mount --bind /root/disk2 /mnt/local-storage/dev/disk2
$ mount --bind /root/disk3 /mnt/local-storage/dev/disk3
$ chmod 777 /mnt/local-storage/dev/disk1
$ chmod 777 /mnt/local-storage/dev/disk2
$ chmod 777 /mnt/local-storage/dev/disk3
$ chcon -R unconfined_u:object_r:svirt_sandbox_file_t:s0 /mnt/local-storage/
```

This creates 3 mount points in /mnt/local-storage/dev one for each disk. For this example the source is in /root, but it can be anywhere really. In the yaml file that creates the provisioner for the PVs the ConfigMap is pointing to /mnt/local-storage/dev as the hostPath and mountPath for the provisioner. There is NO dynamic provisioning for local volumes, thus mount points have to be created ahead of time. If you add more mount points it should be detected automatically without restarting the provisioner pod.

The yaml file creates:

- Service Account
- Cluster Roles
- Cluster Role Bindings
- DaemonSet for the provisioner
- Storage Class (marked as default)
- ConfigMap

After the environment is running, and the mount points have been created using the above commands, simply 'create' the 'local-storage; yaml file and the Persistent volumes (PV) will be created. **NOTE: this yaml file is very specific to the development environment and it is NOT usable without modification for a more general environment**

```bash
$ ./cluster/kubectl.sh create -f docs/devel/local-storage.yaml
```

# Deploy CDI controller

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
    kubevirt.io/storage.import.endpoint: "https://download.cirros-cloud.net/0.4.0/cirros-0.4.0-x86_64-disk.img"   # Required.  Format: (http||s3)://www.myUrl.com/path/of/data
    kubevirt.io/storage.import.secretName: "" # Optional.  The name of the secret containing credentials for the data source
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 20Mi
```

As you can see we are creating a PVC named 'image-pvc' with 2 annotations. In this case we are requesting the cirros image because it's a small image, but any http or s3 end point will do. If you need to provide a secret the second annotation can be used to provide it. In our example we don't need a secret.

The CDI controller will see the annotation and spawn an 'importer' pod that does the actual work. Once the pod is finished there will be a disk.img in the PVC.

# Create a VM using the image

Now that we have a populated PVC, we can create a VM using the disk image. For instance using this yaml:
```YAML
apiVersion: kubevirt.io/v1alpha1
kind: VirtualMachine
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
