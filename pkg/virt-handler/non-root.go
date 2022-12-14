package virthandler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

func changeOwnershipOfBlockDevices(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	volumeModes := map[string]*k8sv1.PersistentVolumeMode{}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.PersistentVolumeClaimInfo != nil {
			volumeModes[volumeStatus.Name] = volumeStatus.PersistentVolumeClaimInfo.VolumeMode
		}
	}

	for i := range vmi.Spec.Volumes {
		volume := vmi.Spec.Volumes[i]
		if volume.VolumeSource.PersistentVolumeClaim == nil {
			continue
		}

		volumeMode, exists := volumeModes[volume.Name]
		if !exists {
			return fmt.Errorf("missing volume status for volume %s", volume.Name)
		}

		if !types.IsPVCBlock(volumeMode) {
			continue
		}
		devPath, err := isolation.SafeJoin(res, string(filepath.Separator), "dev", vmi.Spec.Volumes[i].Name)
		if err != nil {
			return nil
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(devPath); err != nil {
			return err
		}

	}
	return nil
}

func changeOwnership(path *safepath.Path) error {
	err := diskutils.DefaultOwnershipManager.SetFileOwnership(path)
	if err != nil {
		return err
	}
	return nil
}

// changeOwnershipOfHostDisks needs unmodified vmi (not passed to ReplacePVCByHostDisk function)
func changeOwnershipOfHostDisks(vmiWithAllPVCs *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	for i := range vmiWithAllPVCs.Spec.Volumes {
		if volumeSource := &vmiWithAllPVCs.Spec.Volumes[i].VolumeSource; volumeSource.HostDisk != nil {
			volumeName := vmiWithAllPVCs.Spec.Volumes[i].Name
			diskPath := hostdisk.GetMountedHostDiskPath(volumeName, volumeSource.HostDisk.Path)

			_, err := os.Stat(diskPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					diskDir := hostdisk.GetMountedHostDiskDir(volumeName)
					path, err := isolation.SafeJoin(res, diskDir)
					if err != nil {
						return fmt.Errorf("Failed to change ownership of HostDisk dir %s, %s", volumeName, err)
					}
					if err := changeOwnership(path); err != nil {
						return fmt.Errorf("Failed to change ownership of HostDisk dir %s, %s", volumeName, err)
					}
					continue
				}
				return fmt.Errorf("Failed to recognize if hostdisk contains image, %s", err)
			}

			path, err := isolation.SafeJoin(res, diskPath)
			if err != nil {
				return fmt.Errorf("Failed to change ownership of HostDisk image: %s", err)
			}
			err = changeOwnership(path)
			if err != nil {
				return fmt.Errorf("Failed to change ownership of HostDisk image: %s", err)
			}

		}
	}
	return nil
}

func (d *VirtualMachineController) prepareStorage(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	if err := changeOwnershipOfBlockDevices(vmi, res); err != nil {
		return err
	}
	return changeOwnershipOfHostDisks(vmi, res)
}

func getTapDevices(vmi *v1.VirtualMachineInstance) []string {
	macvtap := map[string]bool{}
	for _, inf := range vmi.Spec.Domain.Devices.Interfaces {
		if inf.Macvtap != nil {
			macvtap[inf.Name] = true
		}
	}

	networkNameScheme := namescheme.CreateNetworkNameScheme(vmi.Spec.Networks)
	tapDevices := []string{}
	for _, net := range vmi.Spec.Networks {
		_, isMacvtapNetwork := macvtap[net.Name]
		if podInterfaceName, exists := networkNameScheme[net.Name]; isMacvtapNetwork && exists {
			tapDevices = append(tapDevices, podInterfaceName)
		}
	}
	return tapDevices
}

func (d *VirtualMachineController) prepareTap(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	tapDevices := getTapDevices(vmi)
	for _, tap := range tapDevices {
		path, err := isolation.SafeJoin(res, "sys", "class", "net", tap, "ifindex")
		if err != nil {
			return err
		}
		index, err := func(path *safepath.Path) (int, error) {
			df, err := safepath.OpenAtNoFollow(path)
			if err != nil {
				return 0, err
			}
			defer df.Close()
			b, err := os.ReadFile(df.SafePath())
			if err != nil {
				return 0, fmt.Errorf("Failed to read if index, %v", err)
			}

			return strconv.Atoi(strings.TrimSpace(string(b)))
		}(path)
		if err != nil {
			return err
		}

		pathToTap, err := isolation.SafeJoin(res, "dev", fmt.Sprintf("tap%d", index))
		if err != nil {
			return err
		}

		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(pathToTap); err != nil {
			return err
		}
	}
	return nil

}

func (*VirtualMachineController) prepareVFIO(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	vfioBasePath, err := isolation.SafeJoin(res, "dev", "vfio")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
	}
	vfioPath, err := safepath.JoinNoFollow(vfioBasePath, "vfio")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
	}
	err = safepath.ChmodAtNoFollow(vfioPath, 0666)
	if err != nil {
		return err
	}

	var files []os.DirEntry
	err = vfioBasePath.ExecuteNoFollow(func(safePath string) (err error) {
		files, err = os.ReadDir(safePath)
		return err
	})
	if err != nil {
		return err
	}

	for _, group := range files {
		if group.Name() == "vfio" {
			continue
		}
		groupPath, err := safepath.JoinNoFollow(vfioBasePath, group.Name())
		if err != nil {
			return err
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(groupPath); err != nil {
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
	if err := d.prepareStorage(origVMI, res); err != nil {
		return err
	}
	if err := d.prepareTap(origVMI, res); err != nil {
		return err
	}
	if err := d.prepareVFIO(origVMI, res); err != nil {
		return err
	}
	return nil
}
