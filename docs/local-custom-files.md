# Build KubeVirt using local files

KubeVirt can be built using custom local files when you want to replace a file with your local copy. For example for replacing an rpm or, as illustrated below, the libguestfs-appliance, or a binary.

The guide [custom-rpms.md](https://github.com/kubevirt/kubevirt/blob/main/docs/custom-rpms.md) already explains how to build KubeVirt using custom rpms. Here, we specifically focus in using local files, but the 2 methods can be combined based on your needs. The custom-rpms method might fit better for the cases where you also want to resolve the package dependencies automatically.

## Using Local Files in Builds

To use local files in the build:

1. Copy your custom files into the kubevirt directory. Files will be automatically synchronized by the `hack/dockerized` command into the build container.

```bash
# Local directory with the custom files
$ ls output/
latest-version.txt  libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz

# Sync build container and check the file 
$ ./hack/dockerized ls output
go version go1.19.2 linux/amd64

latest-version.txt  libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz
```

2. Modify the relevant Containerfile in the `build/` directory to use your local file instead of downloading from a remote URL.

3. Build the image with your custom files: `make build-images`

## Example: Custom libguestfs-appliance

To use a custom libguestfs-appliance:

1. Place your appliance tarball in the kubevirt directory
2. Modify `build/libguestfs-tools/Containerfile` to copy from your local path instead of downloading
3. Run `make build-images`
