# Build custom libvirt rpms from source code to KubeVirt builds

This setup illustrates how to build and integrate custom libvirt rpms in KubeVirt. This can be particularly useful if you need to test a fix in libvirt for KubeVirt.

## Build libvirt and the rpms

If you already have the rpms available you can skip this section.

  * Create a volume for the rpms that will be shared with the http server container
```bash
$ docker volume create rpms
```
  * Start build environment for libvirt source code. This setup uses the [container images](https://gitlab.com/libvirt/libvirt/container_registry) used by the libvirt CI. This setup is just an example for reference, and this can be achieved in many ways.
Start container inside the libvirt directory with your changes and enter in the build container
```bash
$ docker run -td -w /libvirt-src --security-opt label=disable --name libvirt-build -v $(pwd):/libvirt-src -v rpms:/root/rpmbuild/RPMS registry.gitlab.com/libvirt/libvirt/ci-centos-stream-8
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

## Start the http server for the rpms

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

  * Create `custom-repo.yaml` pointing to the local http server:
```yaml
repositories:
- arch: x86_64
  baseurl: http://172.17.0.4:80/x86_64/ # The IP corresponding to the rpms-http-server container
  name: custom-build
  gpgcheck: 0
  repo_gpgcheck: 0
```
  * Update the rpms in KubeVirt repository.
  * If you only want to update a single architecture, set `SINGLE_ARCH="x86_64"`.
  * It is sometimes necessary to change `basesystem` when using custom rpms packages. This can be achieved by setting `BASESYSTEM=xyz` env variable.
  * If you want to change version of some packages you can set env variables. See [`hack/rpm-deps.sh`](/hack/rpm-deps.sh) script for all variables that can be changed.
```bash
$ make CUSTOM_REPO=custom-repo.yaml LIBVIRT_VERSION=0:7.2.0-1.el8 rpm-deps
```
Afterwards, the `WORKSPACE` and `rpm/BUILD.bazel` are automatically updated and KubeVirt can be built with the custom rpms.
