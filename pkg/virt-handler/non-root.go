package virthandler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
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

func (c *BaseController) prepareStorage(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	if err := changeOwnershipOfBlockDevices(vmi, res); err != nil {
		return err
	}
	return changeOwnershipOfHostDisks(vmi, res)
}

func (*BaseController) prepareVFIO(res isolation.IsolationResult) error {
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

func (c *BaseController) prepareNetwork(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	rootMount, err := res.MountRoot()
	if err != nil {
		return err
	}

	if netvmispec.RequiresVirtioNetDevice(vmi, c.clusterConfig.AllowEmulation()) {
		if err := c.claimDeviceOwnership(rootMount, "vhost-net"); err != nil {
			return neterrors.CreateCriticalNetworkError(fmt.Errorf("failed to set up vhost-net device, %s", err))
		}
	}

	if netvmispec.RequiresTunDevice(vmi) {
		if err := c.claimDeviceOwnership(rootMount, "/net/tun"); err != nil {
			return neterrors.CreateCriticalNetworkError(fmt.Errorf("failed to set up tun device, %s", err))
		}
	}

	return nil
}

func (c *BaseController) nonRootSetup(vmi *v1.VirtualMachineInstance) error {
	res, err := c.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}
	if err := c.prepareStorage(vmi, res); err != nil {
		return err
	}
	if err := c.prepareVFIO(res); err != nil {
		return err
	}
	if err := c.prepareNetwork(vmi, res); err != nil {
		return err
	}
	return nil
}
