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
 - how to update the rpm dependencies
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

## Build and publish the builder container

### Set environment variables for building the builder -- all architectures, all registries

Note that the default system architecture is amd64, so if you do not set ARCH as an environment variable, you will build a docker container with x86 object format binaries in it. 

```
export ARCH="<your-system-architecture>"
export DOCKER_PREFIX="<your-registry-URL>/<your-kubevirt-namespace>" 
export DOCKER_IMAGE="builder" 
export VERSION="<your-docker-tag>"
export KUBEVIRT_BUILDER_IMAGE="${DOCKER_PREFIX}/${DOCKER_IMAGE}/${VERSION}"
```
where <your-system-architecture> is either `amd64` or `arm64` or `s390x` for x86, ARM, s390x systems, respectively.


`make builder-build` invokes the build of the builder container. 

This tags your image with namespace you set up in your environment variables.

Now you can

```
$ make builder-publish
```

which, among many other things, pushes the image up to the container registry of choice. Note that you will be pushing these changes to your test repository, in order to use this builder image to build kubevirt test operators in further steps. 

### Update the RPM dependencies

Edit your local copy of `hack/rpm-packages.sh` to update the version numbers of the rpms in accordance with the latest versions observable in https://mirror.stream.centos.org/9-stream/AppStream/$ARCH/os/Packages/

Then run:

```
make rpm-deps
```

This will update the JSON lock files in `rpm-lockfiles/` with the new package versions.

## Build & push KubeVirt images to your test registry 

Set the necessary environment variables for these make targets:

```
export BUILD_ARCH="<your-system-architecture>"
export DOCKER_PREFIX="<your-registry-URL>/<your-kubevirt-namespace>" 
export QUAY_REPOSITORY="kubevirt" 
export PACKAGE_NAME="kubevirt-operatorhub"
```

Then build and push:

```
make && make push && make manifests
```
