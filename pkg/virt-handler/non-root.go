package virthandler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	v1 "kubevirt.io/client-go/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

func changeOwnershipOfBlockDevices(vmiWithOnlyBlockPVCs *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	for i := range vmiWithOnlyBlockPVCs.Spec.Volumes {
		if volumeSource := &vmiWithOnlyBlockPVCs.Spec.Volumes[i].VolumeSource; volumeSource.PersistentVolumeClaim != nil {
			devPath := filepath.Join(string(filepath.Separator), "dev", vmiWithOnlyBlockPVCs.Spec.Volumes[i].Name)
			if err := diskutils.DefaultOwnershipManager.SetFileOwnership(filepath.Join(res.MountRoot(), devPath)); err != nil {
				return err
			}
		}
	}
	return nil
}

func changeOwnershipAndRelabel(path string) error {
	err := diskutils.DefaultOwnershipManager.SetFileOwnership(path)
	if err != nil {
		return err
	}

	_, selinuxEnabled, err := selinux.NewSELinux()
	if err == nil && selinuxEnabled {
		unprivilegedContainerSELinuxLabel := "system_u:object_r:container_file_t:s0"
		err = relabelFiles(unprivilegedContainerSELinuxLabel, filepath.Join(path))
		if err != nil {
			return (fmt.Errorf("error relabeling %s: %v", path, err))
		}

	}
	return err
}

func changeOwnershipOfHostDisks(vmiWithAllPVCs *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	for i := range vmiWithAllPVCs.Spec.Volumes {
		if volumeSource := &vmiWithAllPVCs.Spec.Volumes[i].VolumeSource; volumeSource.HostDisk != nil {
			volumeName := vmiWithAllPVCs.Spec.Volumes[i].Name
			diskPath := hostdisk.GetMountedHostDiskPath(volumeName, volumeSource.HostDisk.Path)

			_, err := os.Stat(diskPath)
			if err != nil {
				if os.IsNotExist(err) {
					diskDir := hostdisk.GetMountedHostDiskDir(volumeName)
					if err := changeOwnershipAndRelabel(filepath.Join(res.MountRoot(), diskDir)); err != nil {
						return fmt.Errorf("Failed to change ownership of HostDisk dir %s, %s", volumeName, err)
					}
					continue
				}
				return fmt.Errorf("Failed to recognize if hostdisk contains image, %s", err)
			}

			err = changeOwnershipAndRelabel(filepath.Join(res.MountRoot(), diskPath))
			if err != nil {
				return fmt.Errorf("Failed to change ownership of HostDisk image: %s", err)
			}

		}
	}
	return nil
}

func (d *VirtualMachineController) prepareStorage(vmiWithOnlyBlockPVCS, vmiWithAllPVCs *v1.VirtualMachineInstance) error {
	res, err := d.podIsolationDetector.Detect(vmiWithAllPVCs)
	if err != nil {
		return err
	}
	if err := changeOwnershipOfBlockDevices(vmiWithOnlyBlockPVCS, res); err != nil {
		return err
	}
	return changeOwnershipOfHostDisks(vmiWithAllPVCs, res)
}

func relabelFiles(newLabel string, files ...string) error {
	relabelArgs := []string{"selinux", "relabel", newLabel}
	for _, file := range files {
		cmd := exec.Command("virt-chroot", append(relabelArgs, file)...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error relabeling file %s with label %s. Reason: %v", file, newLabel, err)
		}
	}

	return nil
}
