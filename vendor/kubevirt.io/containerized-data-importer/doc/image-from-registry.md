# Creating a registry image with a VM disk
The purpose of this document is to show how to create registry image containing a Virtual Machine image that can be imported into a PV.

## Prerequisites
The VM disk image should be called 'disk.img' and placed in the root folder.

## Create an image with Buildah

Buildah is a tool that facilitates building Open Container Initiative (OCI) container images.

More information is available here: [Buildah tutorial](https://github.com/containers/buildah/blob/master/docs/tutorials/02-registries-repositories.md).

Create a new container image from scratch and keep its name in a variable:

```bash
$ newcontainer=$(buildah from scratch)
```

Add the VM disk to the container image and commit:

```bash
$ buildah copy $newcontainer ./disk.img /
buildah commit $newcontainer my-vm-img
```

To push an image from your local Buildah container storage, check the image name, then push it using the buildah push command. Remember to identify both the local image name and a new name that includes the location (<registry>:5000, in this case):
```
buildah push --tls-verify=false my-vm-img:latest <registry>:5000/my-vm-img:latest
```

## Create an image with Docker

Create a Dockerfile with the following content:

```
FROM scratch
ADD disk.img /
```

Build, tag and push the image:

```bash
$ docker build . -t my-vm-img
$ docker tag my-vm-img <registry>:5000/my-vm-img
$ docker push <registry>:5000/my-vm-img
```

# Import the registry image into a PVC

Use the following annotations in the PVC yaml:
```
...
annotations:
    cdi.kubevirt.io/storage.import.source: "registry"
    cdi.kubevirt.io/storage.import.endpoint: "docker://<registry>:5000/my-vm-img"
...
```

Full example is available here: [registry-image-pvc](https://github.com/kubevirt/containerized-data-importer/blob/master/manifests/example/registry-image-pvc.yaml)
