# Purpose

This image can be used to export a file from a volume as an
iSCSI lun.
LIO is used to achieve this, thus the container needs to be
privileged.

# Try in docker

```bash
$ docker build -t iscsi-target .
$ docker run --privileged \
  -v /lib/modules:/lib/modules:ro \
  -p 3260:3260 \
  -e GENERATE_DEMO_OS_SEED=true -it iscsi-target

# In another terminal
$ qemu-system-x86_64 \
  -drive file=iscsi://127.0.0.1/iqn.2017-01.io.kubevirt:sn.42/0
```
