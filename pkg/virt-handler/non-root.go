package virthandler

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	//"kubevirt.io/client-go/log"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

func changeOwnershipOfBlockDevices(vmiWithOnlyBlockPVCs *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	passthoughFSVolumes := make(map[string]struct{})
	for i := range vmiWithOnlyBlockPVCs.Spec.Domain.Devices.Filesystems {
		passthoughFSVolumes[vmiWithOnlyBlockPVCs.Spec.Domain.Devices.Filesystems[i].Name] = struct{}{}
	}
	for i := range vmiWithOnlyBlockPVCs.Spec.Volumes {
		if volumeSource := &vmiWithOnlyBlockPVCs.Spec.Volumes[i].VolumeSource; volumeSource.PersistentVolumeClaim != nil {
			volumeName := vmiWithOnlyBlockPVCs.Spec.Volumes[i].Name
			if _, isPassthoughFSVolume := passthoughFSVolumes[volumeName]; isPassthoughFSVolume {
				continue
			}
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
	passthoughFSVolumes := make(map[string]struct{})
	for i := range vmiWithAllPVCs.Spec.Domain.Devices.Filesystems {
		passthoughFSVolumes[vmiWithAllPVCs.Spec.Domain.Devices.Filesystems[i].Name] = struct{}{}
	}
	for i := range vmiWithAllPVCs.Spec.Volumes {
		/*volumeName := vmiWithAllPVCs.Spec.Volumes[i].Name
		if _, isPassthoughFSVolume := passthoughFSVolumes[volumeName]; isPassthoughFSVolume {
			err := changeOwnershipAndRelabel(filepath.Join(res.MountRoot(), fmt.Sprintf("/%s", volumeName)))
			if err != nil {
				return fmt.Errorf("Failed to change ownership of filesystem directory : %s", err)
			}
		}*/
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

func (*VirtualMachineController) prepareVFIO(vmi *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	vfioPath := filepath.Join(res.MountRoot(), "dev", "vfio")
	err := os.Chmod(filepath.Join(vfioPath, "vfio"), 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	groups, err := ioutil.ReadDir(vfioPath)
	if err != nil {
		return err
	}

	for _, group := range groups {
		if group.Name() == "vfio" {
			continue
		}
		if err := diskutils.DefaultOwnershipManager.SetFileOwnership(filepath.Join(vfioPath, group.Name())); err != nil {
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

	sock1 := filepath.Join(res.MountRoot(), "/var/run/kubevirt/virtiofs-containers/findMe.sock")
	res1, err := d.podIsolationDetector.DetectForSocket(origVMI, sock1)
	if err != nil {
		//log.Log.Reason(err).Errorf("failed to connect to socket")
		return err
	}
	err = changeOwnershipAndRelabel(filepath.Join(res1.MountRoot(), "/disk1"))
	if err != nil {
		return fmt.Errorf("Failed to change ownership of filesystem directory test! : %s", err)
	}

	if err := d.prepareStorage(vmi, origVMI, res); err != nil {
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
