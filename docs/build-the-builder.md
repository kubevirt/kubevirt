# Build The Builder Container

The KubeVirt build system starts with a builder helper container, where the entire toolchain and its dependencies are stored, in order to
ensure consistency between builds. Yet, there are sometimes CVEs that must be remedied with an update of the build toolchain, and from time
to time, KubeVirt must be enabled for a new system architecture, e.g. `s390x`. 

The KubeVirt official build stores its builder container and built images in a protected quay.io registry. If there is no builder image for the architecture
of the machine you are trying to build KubeVirt on and for, `make cluster-up` will fail in its attempt to pull a builder image. You will need to build your own builder image, and tweak the build scripts to refer to it. 


This document will provide the steps to:
 - prepare the local environment to build the builder container
 - build the builder container
 - push the builder image to a custom registry
 - how to update the rpm dependencies with bazeldnf
 - invoke the new test builder image to build new kubevirt operator images
 - push the new kubevirt operator images to a custom registry


## Prerequisites

### Podman or Docker

install podman or docker

On Ubuntu,

```
$ apt get install -y podman
```

or 

```
$ apt get install -y docker
```

On RHEL or CENTOS

```
$ dnf install -y podman
```

### Configure your read/write access to a container registry

You will first need to create read and write access to an appropriate namespace in a container registry

[`docker.io`](https://docs.docker.com/docker-id/) or [`quay.io`](https://docs.quay.io/solution/getting-started.html) or [`icr.io`](https://cloud.ibm.com/docs/Registry?topic=Registry-getting-started) are popular registries to use for this purpose.  Then set up a repository or namespace to hold the container you are building, e.g. `builder` or, where the namespace needs to be more unique, `kubevirt-builder`. 

##  Install bazel and dependencies in order to containerize the builder 

Bazel binary distribution images for version 5.3.0 for the Arm and x86 architectures are available from https://github.com/bazelbuild/bazel/releases/tag/5.3.0

This, the dependent libraries and build script configuration will constitute the containerized KubeVirt build toolchain.  For architectures other than arm and x86, e.g. IBM s390x, you need to build bazel from scratch on your build-the-builder box. 

Currently, [a scripted build for bazel is available for Ubuntu on s390x](https://github.com/linux-on-ibm-z/docs/wiki/Building-TensorFlow) step 1.2 'Build Bazel v5.3.0'.

The Dockerfile you'll be building with that bazel installation expects to 
find the bazel executable in `kubevirt/hack/builder/bazel`, so copy it there, e.g.:

```
$ cp /usr/local/bin/bazel ~/kubevirt/hack/builder/bazel
```

## Build and publish the Bazel builder container

### Set environment variables for building the builder -- all architectures, all registries

Note that the default system architecture is amd64, so if you do not set ARCH as an environment variable, you will build a docker container with x86 object format binaries in it. 

```
export ARCH="<your-system-architecture>"
export DOCKER_PREFIX="<your-registry-URL>/<your-kubevirt-namespace>" 
export DOCKER_IMAGE="builder" 
export VERSION="<your-docker-tag>"
export KUBEVIRT_BUILDER_IMAGE="${DOCKER_PREFIX}/${DOCKER_IMAGE}/${VERSION}"
```
where <your-system-architecture> is either `amd64` or `arm64` or `s390x` for x86, ARM, s390x or Power systems, respectively.


`make builder-build` invokes the build of the bazel builder container. 

This tags your image with namespace you set up in your environment variables.

Now you can

```
$ make builder-publish
```

which, among many other things, pushes the image up to the container registry of choice. Note that you will be pushing these changes to your test repository, in order to use this builder image to build kubevirt test operators in further steps. 

### Update the rpms

edit your local copy of this section of the file `hack/rpm-deps.sh`:

```
LIBVIRT_VERSION=${LIBVIRT_VERSION:-0:9.5.0-5.el9}
QEMU_VERSION=${QEMU_VERSION:-17:8.0.0-13.el9}
SEABIOS_VERSION=${SEABIOS_VERSION:-0:1.16.1-1.el9}
EDK2_VERSION=${EDK2_VERSION:-0:20221207gitfff6d81270b5-9}
LIBGUESTFS_VERSION=${LIBGUESTFS_VERSION:-1:1.48.4-4.el9}
GUESTFSTOOLS_VERSION=${GUESTFSTOOLS_VERSION:-0:1.48.2-8.el9}
PASST_VERSION=${PASST_VERSION:-0:0^20230818.g0af928e-4.el9}
VIRTIOFSD_VERSION=${VIRTIOFSD_VERSION:-0:1.5.0-1.el9}
SWTPM_VERSION=${SWTPM_VERSION:-0:0.8.0-1.el9}
SINGLE_ARCH=${SINGLE_ARCH:-""}
BASESYSTEM=${BASESYSTEM:-"centos-stream-release"}
```

Update the version numbers of the rpms in accordance with the latest versions observable in https://mirror.stream.centos.org/9-stream/AppStream/$ARCH/os/Packages/

If you do not have a version that can be looked up in the CentOS rpm collection, the build of the bazel helper container for KubeVirt will fail. 

Once you get it working, you can consider incorporating your local versions of the RPM dependencies in a PR, but careful to check for CVEs and other issues in the dependencies you specified. 

### Obtain the `bazeldnf` utility

If you are building the builder on Power, x86 or arm architecture you can get a pre-built binary of the `bazeldnf` utility from https://github.com/rmohr/bazeldnf/releases

If you are building the builder container natively on s390x, you will need to compile it from source, which is also provided at https://github.com/rmohr/bazeldnf/releases


Now run a script which you can name `./fix-rpm-deps.sh` to update these versions throughout the installation with bazeldnf:

```
#!/bin/bash

set -ex

BASESYSTEM=${BASESYSTEM:-"centos-stream-release"}
bazeldnf_repos="--repofile rpm/repo.yaml"

centos_main="
  acl
  curl-minimal
  vim-minimal
"
centos_extra="
  coreutils-single
  glibc-minimal-langpack
  libcurl-minimal
"

sandboxroot_main="
  findutils
  gcc
  glibc-static
  python3
  sssd-client
"
bazeldnf fetch \
    ${bazeldnf_repos}

bazeldnf rpmtree \
        --public --nobest \
        --name sandboxroot_s390x --arch s390x \
        --basesystem ${BASESYSTEM} \
        ${bazeldnf_repos} \
        $centos_main \
        $centos_extra \
        $sandboxroot_main

SINGLE_ARCH=<your-system-architecture> make rpm-deps
```

where <your-system-architecture> is `amd64`, `arm64` or `s390x`

## Build & push KubeVirt images to your test registry 

Because in this instance, you are using your test registry and potentially an architecture other than x86, before running `make && make bazel-push-images && make manifests` as in [getting-started.md](getting-started.md),
set the necessary environment variables for these make targets:

```
export BUILD_ARCH="<your-system-architecture"
export DOCKER_PREFIX="<your-registry-URL>/<your-kubevirt-namespace>" 
export QUAY_REPOSITORY="kubevirt" 
export PACKAGE_NAME="kubevirt-operatorhub"
```

