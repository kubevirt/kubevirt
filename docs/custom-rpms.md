# Build custom libvirt rpms from source code to KubeVirt builds

This setup illustrates how to build and integrate custom libvirt and qemu rpms in KubeVirt. This can be particularly useful if you need to test a fix in libvirt or qemu for KubeVirt.

# Quick start
Set the required environment variables and run `make custom-rpms`

## Required environment variables
The build script by default builds both QEMU and libvirt from source. 
If you wish to only build one or the other you can set one of the following environment variables;
- QEMU_ONLY=1 
- LIBVIRT_ONLY=1

If QEMU build is enabled (LIBVIRT_ONLY not set), you must export the following environment variables
- QEMU_DIR: The path to the QEMU source directory to be built (can be a git repo or extracted tarball)
- QEMU_KVM_DIR: The path to the CentOS stream source directory providing rpmbuild spec and dependencies
  - See the CentOS stream [qemu-kvm repo](https://gitlab.com/redhat/centos-stream/rpms/qemu-kvm). You should check out the specification corresponding to the currently used version of CentOS stream for the build chain (CentOS 9 Stream, branch name 'c9s')

If libvirt build is enabled (QEMU_ONLY not set), you must export the following environment variables
- LIBVIRT_DIR: The path to the libvirt source directory to be built (can be a git repo or extracted tarball)

### A note on versions
Currently all local builds (cluster-up/cluster-sync) are based on CentOS Stream 9 images.
Accordingly, the RPM build containers are based on the same to avoid version issues where possible.
Should this change in the future the RPM build images, package names for built RPMs, and qemu-kvm spec branch would need to change.
The qemu-kvm branch 'c9s' is currently a good baseline for your builds.
Picking compatible versions of qemu and libvirt to compile is left to the user.

# Confirming the build
The simplest way to check that custom libvirt/qemu RPMs were integrated in kubevirt images is to set unique version numbers for each.
This can be done for qemu by modifying the **VERSION** file in the repository root, and for libvirt by modifying the **meson.build** project/version variable.

**Example: change default versions 10.1.0 and 12.1.0 to 10.1.99 and 12.1.99**

#### `qemu/VERSION`
```diff
-10.1.0
+10.1.99
```

#### `libvirt/meson.build`
```diff
project(
  'libvirt', 'c',
- version: '12.1.0',
+ version: '12.1.99',
  license: 'LGPLv2+',
  meson_version: '>= 0.57.0',
  ...
```

**WARNING: Libvirt builds by default require a clean git repo, so you must either commit your changes or override this requirement. QEMU does not have this requirement**

With uniqe versions set, run `make custom-rpms`, `make cluster-up` and `make cluster-sync` to build a local cluster with the new RPMs.

Next deploy, any VM or VM instance to the cluster (vmi-alpine-efi.yaml used for example)
```
./kubevirtci/cluster-up/kubectl.sh apply -f examples/vmi-alpine-efi.yaml
```
Run the `virsh version` command in virt-launcher pods "compute" container (this is the container that kicks off QEMU through libvirt, and the primary image that consumes libvirt/QEMU RPMs).
The "special" label is specified by **vmi-alpine-efi.yaml**, alternatively find the pod name with another method
#### Example:
```
./kubevirtci/cluster-up/kubectl.sh exec $(./kubevirtci/cluster-up/kubectl.sh get pods -l special=vmi-alpine-efi -o name) -c compute -i -t -- virsh version --daemon
```
#### Output:
```
Compiled against library: libvirt 12.1.99
Using library: libvirt 12.1.99
Using API: QEMU 12.1.99
Running hypervisor: QEMU 10.1.99
Running against daemon: 12.1.99
```

This confirms that the used versions match our special source versions.

### Another note on versions
The "package" versions of the RPM's are set by the corresponding spec file (qemu-kvm.spec and libvirt.spec) and don't necessarily match the executable version that is set in the example.
You can also increment the RPM package versions through the spec files, which may be necessary in rare circumstances to ensure your built package gets selected over the official packages.

# Manual build
This section attempts to document the build process performed by the scripts in as much detail as possible for troubleshooting/maintainence purposes.

## Container base
The **hack/custom-rpms/Dockerfile** file provides a base image with build dependencies for both qemu and libvirt installed.
It is based on the same CentOS 9 Stream image used by the kubevirt builder: `quay.io/centos/centos:stream9`.
This matches the version of packages expected by kubevirt by default (see [BASESYSTEM](#BASESYSTEM-anchor)).
If you are using a different basesystem for kubevirt builds, or the default version changes, the RPM base image will need to change as well

## Build libvirt and the rpms

If you already have the rpms available you can skip this section.

  * Create a volume for the rpms that will be shared with the http server container
```bash
$ docker volume create rpms
```
  * Start build environment for libvirt source code. This setup uses the [container images](https://gitlab.com/libvirt/libvirt/container_registry) used by the libvirt CI. This setup is just an example for reference, and this can be achieved in many ways.
Start container inside the libvirt directory with your changes and enter in the build container
```bash
$ docker run -td -w /libvirt-src --security-opt label=disable --name libvirt-build -v $(pwd):/libvirt-src -v rpms:/root/rpmbuild/RPMS registry.gitlab.com/libvirt/libvirt/ci-centos-stream-9
# Exec in the container
$ docker exec -ti libvirt-build bash
```
  * Steps inside the build environment to obtain the rpms. More details at https://libvirt.org/compiling.html
```bash
# Make sure we get all the latest packages
$ dnf update -y
# Compile and create the rpms
$ meson build
$ ninja -C build dist
```
The build environment might require additional dependencies and this may vary based on the libvirt version:
```bash
$ dnf install -y createrepo hostname
$ rpmbuild -ta    /libvirt-src/build/meson-dist/libvirt-*.tar.xz 
# Create repomd.xml
$ createrepo -v  /root/rpmbuild/RPMS/x86_64
```

The next concern is getting your source into the container while not confounding file ownership.
The scripts mirror the approach used by `hack/dockerized` by using rsync to copy source into the container before build and rsync the build artifacts out after.
There are other ways to solve, but keep in mind that by default libvirt will run git commands during build.
These commands will fail with error "dubious file permissions" if the source code is owned by the user while the "build" folder is owned by root, which is the default if simply using volume mounts.
You may opt to disable the git command part of the libvirt build and simply run those commands ahead of time, or choose another option to deal with docker user ownership in/out of the container.

The **hack/custom-rpms** folder contains build scripts for qemu and libvirt, and the dockerfile defining the base build image. 
Each build script will additionally get rsync-ed into the build container as necessary to build the respective RPM.
The build scripts and dockerfile are documented through comments in the files themselves.

The `dockerized` function defined in the **hack/custom-rpms.sh** build script is based on **hack/dockerized**, stripped down and tailored to the requirements of rpm builds.
It is responsible for starting a build container and rsync server container for transferring files, transferring, building, and copying build artifacts (including a version metadata file required for the `make rpm-deps` step) out of the container.

## Start the http server for the rpms
All built rpms will land in a named docker container (rpms), that was mounted into each build container.
This volume can then be mounted into a httpd container to serve as a RPM server for `make rpm-deps`
If you want to use other publicly available rpms or a private repository that is reachable from the KubeVirt build container, you can skip this section and substitute the custom repository.
The http server container allows to expose locally the rpms to the KubeVirt build server. It is reachable by the IP address from the KubeVirt build container.
  * Start the http server with the `rpms` volume where we created the rpms in the previous step (otherwise pass the directory that contains the rpms)
```bash
$ docker run -dit --name rpms-http-server -p 80 -v rpms:/usr/local/apache2/htdocs/ httpd:latest
```
  * Get the IP of the container `rpms-http-server`
```bash
$ docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' rpms-http-server
172.17.0.4
```
## Add the custom repository to KubeVirt
The build script retreives the httpd server IP automatically and generates a modified version of **hack/custom-rpms/custom-repo.yaml**.
This file must be saved in a kubevirt directory that isn't ignored by the **hack/dockerized** rsync copy process so that it is copied into the builder container.
  * Create `custom-repo.yaml` pointing to the local http server:
```yaml
repositories:
- arch: x86_64
  baseurl: http://172.17.0.4:80/x86_64/ # The IP corresponding to the rpms-http-server container
  name: custom-build
  gpgcheck: 0
  repo_gpgcheck: 0
```
The custom repo will be specified to the `make rpm-deps` command.
Afterwards, the `WORKSPACE` and `rpm/BUILD.bazel` are automatically updated and KubeVirt can be built with the custom rpms.


## The `make rpm-deps` command
TLDR: we need to set the LIBVIRT_VERSION/QEMU_VERSION variable to match that of the version we are building from source, and the SINGLE_ARCH variable to match what architecture we are building for (likely host arch). We *do not* need to set the BASESYSTEM variable, but the default value (centos-stream-release) implies what image version we should use for our dockerized build of libvirt

This command updates the **WORKSPACE** file and **rpms/BUILD.bazel** file according to pre-defined rules, and the currently used rpm repositories.
Inspection of the script shows that it uses several environment variables that can specify custom RPM versions to be used by the kubevirt build system.
The LIBVIRT_VERSION and QEMU_VERSION variables are the ones that are critical to running custom RPMs for libvirt and qemu respectively.
- LIBVIRT_VERSION
- QEMU_VERSION
- SEABIOS_VERSION
- EDK2_VERSION
- LIBGUESTFS_VERSION
- GUESTFSTOOLS_VERSION
- PASST_VERSION
- VIRTIOFSD_VERSION
- SWTPM_VERSION

Additionally, two more environment variables are used:
- **SINGLE_ARCH**: This variable will restrict which rpm specifications will be updated in the **WORKSPACE** and **rpms/BUILD.bazel** file.
This is required to be set in our situation, as otherwise the make command will attempt to update package requirements of other architectures (arm64 and s390x) to the version that we specify in LIBVIRT_VERSION, which will not exist in our self-hosted package repository, and likely not exist in the upstream repositories.
- **BASESYSTEM** <a id="BASESYSTEM-anchor"></a>: This will become the `--basesystem` argument to the `bazeldnf rpmtree` command, which resolves rpm trees into an individualized rpm list with precise versioning (see [bazeldnf docs](https://github.com/brianmcarey/bazeldnf) for the currently used fork of bazeldnf).
The default for this value in kubevirt build scripts is "centos-stream-release," which at the time of writing corresponds to CentOS Stream 9.
This is a reasonable value as the kubevirt builder is *currently* based on CentOS Stream 9 (see **hack/builder/Dockerfile** for the base version being used).
The **custom-rpms.md** docs mention that it is sometimes necessary to change this value, but changing the value is not necessary if using the standard kubevirt builder, and what values are valid to set for this is outside the scope of this document.

### Getting the correct LIBVIRT_VERSION
The build script automatically gets the libvirt version using the `meson introspect` within the build container and outputting the version value to **libvirt/build/version.txt**.
It gets the qemu version by grepping the qemu-kvm.spec rpmbuild specfile used to build the package.
The qemu approach doesn't work for libvirt, as libvirt.spec uses a variable to retrieve the meson package version.
The libvirt approach doesn't work for qemu, as the qemu source version is completely independent from the RPM package version.

#### Manual version setting
The libvirt version of the compiled code can be found manually in **libvirt/meson.build** as shown:
```
project(
  'libvirt', 'c',
  version: '12.1.0',    # libvirt-version-number
  license: 'LGPLv2+',
  meson_version: '>= 0.57.0',
  default_options: [
    'buildtype=debugoptimized',
    'b_pie=true',
    'c_std=gnu99',
    'warning_level=2',
  ],
)
```
The format of the LIBVIRT_VERSION/QEMU_VERSION environment variable is "<epoch>:<version-number>-<release>.el<centos-stream-version>", for example:
- Given the libvirt version shown (12.1.0) and using centos-stream-9 to build the rpms, you would have LIBVIRT_VERSION=0:12.1.0-1.el9 (Epoch is currently hardcoded to 0, Release to 1. See libvirt.spec if that changes).
- This ***does not*** exactly match the format of the built rpm file names (i.e. libvirt-devel-12.1.0-1.el9.x86_64.rpm)

The QEMU version is found in **qemu-kvm/qemu-kvm.spec**
```
Summary: QEMU is a machine emulator and virtualizer
Name: qemu-kvm
Version: 10.1.0
Release: 12%{?rcrel}%{?dist}%{?cc_suffix}
# Epoch because we pushed a qemu-1.0 package. AIUI this can't ever be dropped
# Epoch 15 used for RHEL 8
# Epoch 17 used for RHEL 9 (due to release versioning offset in RHEL 8.5)
Epoch: 17
License: GPLv2 and GPLv2+ and CC-BY
URL: http://www.qemu.org/
ExclusiveArch: x86_64 %{power64} aarch64 s390x
```

Release will by default collapse to just "12.el<centos-stream-version>" in this case, unless you explicitly set the rcrel version for a release candidate. Epoch and Version are self explanatory.

You may override the automatically populated value of LIBVIRT_VERSION/QEMU_VERSION by exporting the environment variable in the correct format

With the correct environment variables set, we can successfully run the `make rpm-deps` command. Running a `git diff` in the kubevirt source folder will show the updated packages being referenced. You may also see some non-libvirt packages that changed that are simply newer versions available on the remote. All libvirt dependencies should show a *url* field that references the rpm http server that is being run in docker.
