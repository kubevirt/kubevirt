# Purpose

This image can be used to export a file from a volume as an
iSCSI lun - served by tgtd.

The container has two "modes":

1. Demo - The lun is prepoulated with a demo image (qemu demo image)
2. Persist - The lun is pointing to an empty file of a given size
   which resides on a persistent volume).


# Build

Easy:

```bash
$ docker build -t kubevirt/iscsi-target-tgtd .
```

# Demo flow

The demo flow will download a demo image into the container,
which is then exposed as a LUN.

```bash
# Docker
$ docker run \
  -p 3260:3260 \
  -e GENERATE_DEMO_OS_SEED=true -it iscsi-target-tgtd


# To test:
# In another terminal
$ qemu-system-x86_64 \
  -drive file=iscsi://127.0.0.1/iqn.2017-01.io.kubevirt:sn.42/1

# Or just to discover
$ iscsiadm --mode discovery -t sendtargets --portal 127.0.0.1
```

# Persistent flow

The persistent flow works as follows:

1. Choose a PV to use as a backingstore for the to be created LUN
2. Create a claim on the PV
3. Use the `iscsi-target-tgtd.yaml.tpl` to create a pod which is using
   the claim from 2. and mounting this claim into `/volume`.
4. Create the pod (which will claim the PV)
5. The LUN will store the data on the connected PV

```bash
# Copy `iscsi-target-tgtd.yaml.tpl` into a new file and put in the
# correct claim

$ kubectl create -f iscsi-target-tgtd.yaml.tpl
pod "iscsi-target-tgtd" created
service "iscsi-target-tgtd" created

# Inside the cluster run
$ qemu-system-x86_64 \
  -drive file=iscsi://${SERVICE_IP}/iqn.2017-01.io.kubevirt:sn.42/1

```

# Known issues

## Starts with LUN 1
tgtd is used inside the image, this is delivering the first LUN as 1 (not 0).

## Why tgtd and not targetcli/LIO?
LIO is a kernel based target, this would require privileged container.
tgt otoh is completely in user-space, and can thus be run as a regular container.
