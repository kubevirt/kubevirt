# Usage of libguestfs-tools and virtctl guestfs

[Libguestfs tools](https://libguestfs.org/) are a set of utilities for accessing and modifying VM disk images. The command `virtctl guestfs` helps to deploy an interactive container with the libguestfs-tools and the PVC attached to it. This command is particurarly useful if the users need to modify, inspect or debug VM disks on a PVC.
```bash
$ virtctl guestfs -h
Create a pod with libguestfs-tools, mount the pvc and attach a shell to it. The pvc is mounted under the /disks directory inside the pod for filesystem-based pvcs, or as /dev/vda for block-based pvcs

Usage:
  virtctl guestfs [flags]

Examples:
  # Create a pod with libguestfs-tools, mount the pvc and attach a shell to it:
  virtctl guestfs <pvc-name>

Flags:
  -h, --help                 help for guestfs
      --image string         libguestfs-tools container image
      --kvm                  Use kvm for the libguestfs-tools container (default true)
      --pull-policy string   pull policy for the libguestfs image (default "IfNotPresent")

Use "virtctl options" for a list of global command-line options (applies to all commands).
```

By default `virtctl guestfs` sets up `kvm` for the interactive container. This considerably speeds up the execution of the libguestfs-tools since they use QEMU. If the cluster doesn't have any kvm supporting nodes, the user must disable kvm by setting the option `--kvm=false`. If not set, the libguestfs-tools pod will remain pending because it cannot be scheduled on any node.

The command automatically uses the image exposed by KubeVirt under the http endpoint `/apis/subresources.kubevirt.io/<kubevirt-version>/guestfs`, but it can be configured to use a custom image by using the option `--image`. Users can also overwrite the pull policy of the image by setting the option `pull-policy`.

The command checks if a PVC is used by another pod in which case it will fail. However, once libguestfs-tools has started, the setup doesn't prevent a new pod starting and using the same PVC. The user needs to verify that there are no active virtctl guestfs pods before starting the VM which accesses the same PVC.

Currently, `virtctl guestfs` supports only a single PVC. Future versions might support multiple PVCs attached to the interactive pod.

## Examples and use-cases
Generally, the user can take advantage of the `virtctl guestfs` command for all typical usage of libguestfs-tools. It is strongly recommended to consult the [official documentation](https://libguestfs.org/guestfs-recipes.1.html). This command simply aims to help in configuring the correct containerized environment in the Kubernetes cluster where KubeVirt is installed.

For all the examples, the user has to start the interactive container by referencing the PVC in the `virtctl guestfs` command. This will deploy the interactive pod and attach the stdin and stdout. 

Example:

```bash
$ virtctl guestfs pvc-test
Use image: registry:5000/kubevirt/libguestfs-tools@sha256:6644792751b2ba9442e06475a809448b37d02d1937dbd15ad8da4d424b5c87dd 
The PVC has been mounted at /disk 
Waiting for container libguestfs still in pending, reason: ContainerCreating, message:  
Waiting for container libguestfs still in pending, reason: ContainerCreating, message:  
bash-5.0#
```
Once the libguestfs-tools pod has been deployed, the user can access the disk and execute the desired commands. Later, once the user has completed the operations on the disk, simply `exit` the container and the pod be will automatically terminated.
 
1. Inspect the disk filesystem to retrive the version of the OS on the disk:
```bash
bash-5.0# virt-cat -a disk.img /etc/os-release 
NAME=Fedora
VERSION="34 (Cloud Edition)"
ID=fedora
VERSION_ID=34
VERSION_CODENAME=""
PLATFORM_ID="platform:f34"
PRETTY_NAME="Fedora 34 (Cloud Edition)"
ANSI_COLOR="0;38;2;60;110;180"
LOGO=fedora-logo-icon
CPE_NAME="cpe:/o:fedoraproject:fedora:34"
HOME_URL="https://fedoraproject.org/"
DOCUMENTATION_URL="https://docs.fedoraproject.org/en-US/fedora/34/system-administrators-guide/"
SUPPORT_URL="https://fedoraproject.org/wiki/Communicating_and_getting_help"
BUG_REPORT_URL="https://bugzilla.redhat.com/"
REDHAT_BUGZILLA_PRODUCT="Fedora"
REDHAT_BUGZILLA_PRODUCT_VERSION=34
REDHAT_SUPPORT_PRODUCT="Fedora"
REDHAT_SUPPORT_PRODUCT_VERSION=34
PRIVACY_POLICY_URL="https://fedoraproject.org/wiki/Legal:PrivacyPolicy"
VARIANT="Cloud Edition"
VARIANT_ID=cloud
```
2. Add users (for example after the disk has been imported using [CDI](https://github.com/kubevirt/containerized-data-importer))
```bash
bash-5.0# virt-customize -a disk.img --run-command 'useradd -m test-user -s /bin/bash' --password  'test-user:password:test-password'
[   0.0] Examining the guest ...
[   4.1] Setting a random seed
[   4.2] Setting the machine ID in /etc/machine-id
[   4.2] Running: useradd -m test-user -s /bin/bash
[   4.3] Setting passwords
[   5.3] Finishing off
```
3. Run virt-rescue and repair a broken partition or initrd (for example by running dracut)
```bash
bash-5.0# virt-rescue -a disk.img
[...]
The virt-rescue escape key is ‘^]’.  Type ‘^] h’ for help.

------------------------------------------------------------

Welcome to virt-rescue, the libguestfs rescue shell.

Note: The contents of / (root) are the rescue appliance.
You have to mount the guest’s partitions under /sysroot
before you can examine them.
><rescue> fdisk -l
Disk /dev/sda: 6 GiB, 6442450944 bytes, 12582912 sectors
Disk model: QEMU HARDDISK   
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
Disklabel type: gpt
Disk identifier: F8DC0844-9194-4B34-B432-13FA4B70F278

Device       Start      End  Sectors Size Type
/dev/sda1     2048     4095     2048   1M BIOS boot
/dev/sda2     4096  2101247  2097152   1G Linux filesystem
/dev/sda3  2101248 12580863 10479616   5G Linux filesystem


Disk /dev/sdb: 4 GiB, 4294967296 bytes, 8388608 sectors
Disk model: QEMU HARDDISK   
Units: sectors of 1 * 512 = 512 bytes
Sector size (logical/physical): 512 bytes / 512 bytes
I/O size (minimum/optimal): 512 bytes / 512 bytes
><rescue> mount /dev/sda3 sysroot/
><rescue> mount /dev/sda2 sysroot/boot
><rescue> chroot sysroot/
><rescue> ls boot/
System.map-5.11.12-300.fc34.x86_64
config-5.11.12-300.fc34.x86_64
efi
grub2
initramfs-0-rescue-8afb5b540fab48728e48e4196a3a48ee.img
initramfs-5.11.12-300.fc34.x86_64.img
loader
vmlinuz-0-rescue-8afb5b540fab48728e48e4196a3a48ee
><rescue> dracut -f boot/initramfs-5.11.12-300.fc34.x86_64.img 5.11.12-300.fc34.x86_64
[...]
><rescue> exit # <- exit from chroot
><rescue> umount sysroot/boot
><rescue> umount sysroot
><rescue> exit
```

4. Install an OS from scratch
```bash
bash-5.0# virt-builder centos-8.2 -o disk.img --root-password password:password-test
[   1.5] Downloading: http://builder.libguestfs.org/centos-8.2.xz
######################################################################## 100.0%#=#=#                                                    ######################################################################## 100.0%
[  58.3] Planning how to build this image
[  58.3] Uncompressing
[  65.7] Opening the new disk
[  70.8] Setting a random seed
[  70.8] Setting passwords
[  72.0] Finishing off
                   Output file: disk.img
                   Output size: 6.0G
                 Output format: raw
            Total usable space: 5.3G
                    Free space: 4.0G (74%)

```
5. Identify the partition and filesystem on the disk
````bash
bash-5.0# virt-filesystems -a disk.img --partitions --filesystem --long
Name       Type        VFS   Label  MBR  Size        Parent
/dev/sda2  filesystem  ext4  -      -    1023303680  -
/dev/sda4  filesystem  xfs   -      -    4710203392  -
/dev/sda1  partition   -     -      -    1048576     /dev/sda
/dev/sda2  partition   -     -      -    1073741824  /dev/sda
/dev/sda3  partition   -     -      -    644874240   /dev/sda
/dev/sda4  partition   -     -      -    4720689152  /dev/sda
````
## Limitations
Currently, it is not possible to resize the xfs filesystem.
