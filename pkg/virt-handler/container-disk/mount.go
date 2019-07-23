package container_disk

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	v1 "kubevirt.io/client-go/api/v1"
)

type Mounter struct {
	PodIsolationDetector isolation.PodIsolationDetector
}

// Mount takes a vmi and mounts all container disks of the VMI, so that they are visible for the qemu process.
// Additionally qcow2 images are validated if "verify" is true. The validation happens with rlimits set, to avoid DOS.
func (m *Mounter) Mount(vmi *v1.VirtualMachineInstance, verify bool) error {
	for i, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			targetFile := containerdisk.GenerateDiskTargetPathFromHostView(vmi, i)
			nodeRes := isolation.NodeIsolationResult()

			if isMounted, err := nodeRes.IsMounted(targetFile); err != nil {
				return fmt.Errorf("failed to determine if %s is already mounted: %v", targetFile, err)
			} else if !isMounted {
				res, err := m.PodIsolationDetector.DetectForSocket(vmi, containerdisk.GenerateSocketPathFromHostView(vmi, i))
				if err != nil {
					return fmt.Errorf("failed to detect socket for containerDisk %v: %v", volume.Name, err)
				}
				mountInfo, err := res.MountInfoRoot()
				if err != nil {
					return fmt.Errorf("failed to detect root mount info of containerDisk  %v: %v", volume.Name, err)
				}
				nodeMountInfo, err := nodeRes.ParentMountInfoFor(mountInfo)
				if err != nil {
					return fmt.Errorf("failed to detect root mount point of containerDisk %v on the node: %v", volume.Name, err)
				}
				sourceFile, err := containerdisk.GetImage(filepath.Join(nodeRes.MountRoot(), nodeMountInfo.MountPoint), volume.ContainerDisk.Path)
				if err != nil {
					return fmt.Errorf("failed to find a sourceFile in containerDisk %v: %v", volume.Name, err)
				}
				f, err := os.Create(targetFile)
				if err != nil {
					return fmt.Errorf("failed to create mount point target %v: %v", targetFile, err)
				}
				f.Close()

				out, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "mount", "-o", "ro,bind", strings.TrimPrefix(sourceFile, nodeRes.MountRoot()), targetFile).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to bindmount containerDisk %v: %v : %v", volume.Name, string(out), err)
				}
			}
			if verify {
				res, err := m.PodIsolationDetector.Detect(vmi)
				if err != nil {
					return fmt.Errorf("failed to detect VMI pod: %v", err)
				}
				imageInfo, err := isolation.GetImageInfo(containerdisk.GenerateDiskTargetPathFromLauncherView(i), res)
				if err != nil {
					return fmt.Errorf("failed to get image info: %v", err)
				}

				if err := containerdisk.VerifyImage(imageInfo); err != nil {
					return fmt.Errorf("invalid image in containerDisk %v: %v", volume.Name, err)
				}
			}
		}
	}
	return nil
}

// Unmount unmounts all container disks of a given VMI.
func (m *Mounter) Unmount(vmi *v1.VirtualMachineInstance) error {
	mountDir := containerdisk.GenerateVolumeMountDir(vmi)

	files, err := ioutil.ReadDir(mountDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to list container disk mounts: %v", err)
	}

	if vmi.UID != "" {
		for _, file := range files {
			path := filepath.Join(mountDir, file.Name())
			if strings.HasSuffix(path, ".sock") {
				continue
			}
			if mounted, err := isolation.NodeIsolationResult().IsMounted(path); err != nil {
				return fmt.Errorf("failed to check mount point for containerDisk %v: %v", path, err)
			} else if mounted {
				out, err := exec.Command("/usr/bin/chroot", "--mount", "/proc/1/ns/mnt", "umount", path).CombinedOutput()
				if err != nil {
					return fmt.Errorf("failed to unmount containerDisk %v: %v : %v", path, string(out), err)
				}
			}
		}

		if err := os.RemoveAll(mountDir); err != nil {
			return fmt.Errorf("failed to remove containerDisk files: %v", err)
		}
	}
	return nil
}
