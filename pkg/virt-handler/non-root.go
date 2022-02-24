package virthandler

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"
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

	seLinux, selinuxEnabled, err := selinux.NewSELinux()
	if err == nil && selinuxEnabled {
		unprivilegedContainerSELinuxLabel := "system_u:object_r:container_file_t:s0"
		err = selinux.RelabelFiles(unprivilegedContainerSELinuxLabel, seLinux.IsPermissive(), filepath.Join(path))
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

func (d *VirtualMachineController) prepareStorage(vmiWithOnlyBlockPVCS, vmiWithAllPVCs *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	if err := changeOwnershipOfBlockDevices(vmiWithOnlyBlockPVCS, res); err != nil {
		return err
	}
	return changeOwnershipOfHostDisks(vmiWithAllPVCs, res)
}

func getTapDevices(vmi *v1.VirtualMachineInstance) []string {
	macvtap := map[string]bool{}
	for _, inf := range vmi.Spec.Domain.Devices.Interfaces {
		if inf.Macvtap != nil {
			macvtap[inf.Name] = true
		}
	}

	tapDevices := []string{}
	for _, net := range vmi.Spec.Networks {
		_, ok := macvtap[net.Name]
		if ok {
			tapDevices = append(tapDevices, net.Multus.NetworkName)
		}
	}
	return tapDevices
}

func (d *VirtualMachineController) prepareTap(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	tapDevices := getTapDevices(vmi)
	for _, tap := range tapDevices {
		path := filepath.Join(res.MountRoot(), "sys", "class", "net", tap, "ifindex")
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("Failed to read if index, %v", err)
		}

		index, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			return err
		}

		pathToTap := filepath.Join(res.MountRoot(), "dev", fmt.Sprintf("tap%d", index))

		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(pathToTap); err != nil {
			return err
		}
	}
	return nil

}

func (d *VirtualMachineController) nonRootSetup(origVMI, vmi *v1.VirtualMachineInstance) error {
	res, err := d.podIsolationDetector.Detect(origVMI)
	if err != nil {
		return err
	}
	if err := d.prepareStorage(vmi, origVMI, res); err != nil {
		return err
	}
	if err := d.prepareTap(origVMI, res); err != nil {
		return err
	}
	return nil
}
