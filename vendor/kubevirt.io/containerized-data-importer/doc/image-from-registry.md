# Creating a registry image with a VM disk
The purpose of this document is to show how to create registry image containing a Virtual Machine image that can be imported into a PV.

## Prerequisites
Import from registry should be able to consume the same container images as [containerDisk](https://github.com/kubevirt/kubevirt/blob/master/docs/container-register-disks.md).
Thus the VM disk image file to be consumed must be located under /disk directory in the container image. The file can be in any of the supported formats : qcow2, raw, archived image file. There are no special naming constraints for the VM disk file.

## Import VM disk image file from existing containerDisk images in kubevirt repository 
For example vmidisks/fedora25:latest as described in [containerDisk](https://github.com/kubevirt/kubevirt/blob/master/docs/container-register-disks.md)

## Create a container image with Buildah
Buildah is a tool that facilitates building Open Container Initiative (OCI) container images.
More information is available here: [Buildah tutorial](https://github.com/containers/buildah/blob/master/docs/tutorials/02-registries-repositories.md).

Create a new directory `/tmp/vmdisk` with the following Docker file and a vm image file (ex: `fedora28.qcow2`)
Create a new container image with the following docker file 

```bash
cat << END > Dockerfile
FROM kubevirt/container-disk-v1alpha
ADD fedora28.qcow2 /disk
END
```
Build and push image to a registry. 
Note: In development environment you can push to 
1. A cluster local `cdi-docker-registry-host` which hosts docker registry and is accessible within the cluster via `cdi-docker-registry-host.cdi`. The registry is initialized from `cluster-sync` flow and is used for functional tests purposes. 
2. Globally accessible registry that is used for image caching and is accessible via `registry:5000` host name

```bash
buildah bud -t vmidisk/fedora28:latest /tmp/vmdisk
buildah push --tls-verify=false vmidisk/fedora28:latest docker://cdi-docker-registry-host.cdi/fedora28:latest

```
## Create a container image with Docker

Create a Dockerfile with the following content in a new directory /tmp/vmdisk. Add an image file to the same directory (for example fedora28.qcow2)

```
FROM kubevirt/container-disk-v1alpha
ADD fedora28.qcow2 /disk
```

Build, tag and push the image:

```bash
docker build -t vmdisks/fedora28:latest /tmp/vmdisk
docker push vmdisks/fedora28:latest

```

# Import the registry image into a Data volume

Use the following to import a fedora cloud image from docker hub:
```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
metadata:
  name: registry-image-datavolume
spec:
  source:
    registry:
      url: "docker://kubevirt/fedora-cloud-registry-disk-demo"
  pvc:
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
```
Full example is available here: [registry-image-pvc](../manifests/example/registry-image-datavolume.yaml)

# Registry security

## Private registry

If your docker registry requires authentication:

Create a `Secret` in the same namespace as the DataVolume to store user credentials.  See [endpoint-secret](../manifests/example/endpoint-secret.yaml)

Add `SecretRef` to `DataVolume` spec.

```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
...
spec:
  source:
    registry: 
      url: "docker://my-private-registry:5000/my-username/my-image"
      secretRef: my-docker-creds 
...
```

## TLS certificate configuration

If your registry TLS certificate is not signed by a trusted CA:

Create a `ConfigMap` containing all certificates required to trust the registry.

```bash
kubectl create configmap my-registry-certs --from-file=my-registry.crt
```

The `ConfigMap` may contain multiple entries if necessary.  Key name is irrelevant.

Add `CertConfigMap` to `DataVolume` spec.

```yaml
apiVersion: cdi.kubevirt.io/v1alpha1
kind: DataVolume
...
spec:
  source:
    registry: 
      url: "docker://my-private-registry:5000/my-username/my-image"
      certConfigMap: my-registry-certs 
...
```

## Insecure registry

To disable TLS security for a registry:

Add the registry to the `cdi-insecure-registries` `ConfigMap` in the `cdi` namespace.

```bash
patch configmap cdi-insecure-registries -n cdi \
  --type merge -p '{"data":{"my-private-registry:5000": ""}}'
```
