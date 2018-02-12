# Purpose

This container can be used to expose an empty and two demo images
as LUNs of an iSCSI target.
The iSCSI target is provided by `tgtd` which is running in userspace
and is thus not relying on kernel features (like the LIO iSCSI target).

Available LUNs:

0. (An tgt dinternal LUN, can not be used)
1. Empty image - for your data
2. Alpine
3. CirrOS


# Build

Easy:

```bash
$ docker build -t kubevirt/iscsi-demo-target-tgtd .
```

The build will download (and build in) two OS images:

- Alpine, because it's extensible
- CirrOS, because it's good for testing


# Usage

Just run the container and a client:

```bash
# Docker
$ docker run \
  -p 3260:3260 \
  -it iscsi-demo-target-tgtd


# To test:
# In another terminal
$ qemu-system-x86_64 \
  -snapshot \
  -drive file=iscsi://127.0.0.1/iqn.2017-01.io.kubevirt:sn.42/2

# Or just to discover
$ iscsiadm --mode discovery -t sendtargets --portal 127.0.0.1
```

## Usage in Kubernetes

Alongside the actual container, there is also a manifest to expose a ready to
- Create the iSCSI Demo target pod
- Expose the pod using a service
- Add named persistent volumes for each LUN
- Add persistent volume claims for each volume

The target itself can be used by `qemu` (see example below), the claims
can also be directly used by Pods.

Using the iSCSI target using `qemu`:

```bash
# Build all manifests in the top-level directory
$ make manifests

# Then create the pod, services, persistent volumes, and claims
$ kubectl create -f manifests/iscsi-demo-target.yaml
persistentvolumeclaim "disk-custom" created
persistentvolumeclaim "disk-alpine" created
persistentvolumeclaim "disk-cirros" created
persistentvolume "iscsi-disk-custom" created
persistentvolume "iscsi-disk-alpine" created
persistentvolume "iscsi-disk-cirros" created
service "iscsi-demo-target" created
pod "iscsi-demo-target-tgtd" created

# Run a qemu instance to see if the target can be used
# Note: This is not testing the PV or PVC, just the service and target
# Use ctrl-a c quit to quit
$ kubectl run --rm -it qemu-test --image=kubevirt/libvirt -- \
    qemu-system-x86_64 \
      -snapshot \
      -drive file=iscsi://iscsi-demo-target/iqn.2017-01.io.kubevirt:sn.42/2 \
      -nographic
```


## Exporting host paths as LUNs

Sometimes OS images are large or block devices, such that you do not want
to make them part of the demo image. In that case you can set
the `EXPORT_HOST_PATHS` environment variable to directly reference a path
on the host.
For this to work you obviously also need to bind mount in the host root
(`/`) into the `/host` path inside the container.

You can for example use `EXPORT_HOST_PATHS=/dev/sda /home/alice/beos.img`
to export `/dev/sda` and `/home/alice/beos.img` from the host.
Assuming that `/host` inside the container points to `/` on the host.


# Known issues

## It's not production ready
The purpose of this image and pod is to support demo and testing use-cases.

## Starts with LUN 1
tgtd is used inside the image, this is delivering the first LUN as 1 (not 0).

## Why tgtd and not targetcli/LIO?
LIO is a kernel based target, this would require privileged container.
tgt otoh is completely in user-space, and can thus be run as a regular container.

## Some files seem to be corrupted when booting VMs off them
Just kill the pod and let the RC recreate it. This can be due to caching.
