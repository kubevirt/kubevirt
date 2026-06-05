# Build KubeVirt using local files

KubeVirt system strongly relies on bazel and it is built mostly by files hosted remotely. It is handy to be able to build KubeVirt using custom local files when you want to replace a file with your local copy. For example for replacing an rpm or, as illustrated below, the libguestfs-appliance, or a binary.

The guide [custom-rpms.md](https://github.com/kubevirt/kubevirt/blob/main/docs/custom-rpms.md) already explains how to build KubeVirt using custom rpms. Here, we specifically focus in using local files, but the 2 methods can be combined based on your needs. The custom-rpms method might fit better for the cases where you also want to resolve the package dependencies automatically.

In the following example, we illustrate how to replace the `libguestfs-appliance` file, but it is valid for any cases using remote files.

1. Copy your custom appliance file in the building container. It is enough to have the directory or the file in the kubevirt directory, and it will be automatically synchronized by the `hack/dockerized` command

```bash
# Local directory with the custom files
$ ls output/
latest-version.txt  libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz
# Sync build container and check the file 
$ ./hack/dockerized ls output
go version go1.19.2 linux/amd64

latest-version.txt  libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz
```
Modify the WORKSPACE to point to your custom appliance:

2. Calculate the checksum of the file:
```bash
$  sha256sum output/libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz 
6bb9db7a4c83992f3e5fadb1dd51080d8cf53aabe6b546ebee6e2e9a52c569bb  output/libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz
```
3. Point the WORKSPACE to the file and replace the checksum. In the URL, we need to use the `file` protocol and the file is located in the KubeVirt workspace `/root/go/src/kubevirt.io/kubevirt` + the path of your custom file.

```diff
diff --git a/WORKSPACE b/WORKSPACE
index fa717cdcd..a27b05d29 100644
--- a/WORKSPACE
+++ b/WORKSPACE
@@ -386,9 +386,9 @@ http_archive(
 
 http_file(
     name = "libguestfs-appliance",
-    sha256 = "59fe17973fdaf4d969203b66b1446d855d406aea0736d06ee1cd624100942c8f",
+    sha256 = "6bb9db7a4c83992f3e5fadb1dd51080d8cf53aabe6b546ebee6e2e9a52c569bb",
     urls = [
-        "https://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/libguestfs-appliance/appliance-1.48.4-linux-5.14.0-176-centos9.tar.xz",
+        "file:///root/go/src/kubevirt.io/kubevirt/output/libguestfs-appliance-1.48.4-qcow2-linux-5.14.0-183-centos9.tar.xz",
     ],
 )
```
4. Build the image with your custom appliance `make bazel-build-images`
