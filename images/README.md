## Steps to install new RPM's under the testing images

Some of our testing images

- `cdi-http-import-server`
- `vm-killer`

use base image `kubevirtci/kubevirt-testing` from the [KubeVirtCI repository](https://github.com/kubevirt/kubevirtci/tree/master/images).

To add new RPM's under this image:

- follow instructions under [README.md](https://github.com/kubevirt/kubevirtci/blob/master/images/README.md)
- use the hash the you got from the above step to update `WORKSPACE` target `kubevirt-testing`
