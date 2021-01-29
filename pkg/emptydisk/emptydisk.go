package emptydisk

import (
	"os"
	"os/exec"
	"path"
	"strconv"

	v1 "kubevirt.io/client-go/api/v1"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

var EmptyDiskBaseDir = "/var/run/libvirt/empty-disks/"

func CreateTemporaryDisks(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.EmptyDisk != nil {
			if err := createEmptyDiskForVolume(volume); err != nil {
				return err
			}
		}
	}
	return nil
}

func createEmptyDiskForVolume(volume v1.Volume) error {
	if err := os.MkdirAll(EmptyDiskBaseDir, 0777); err != nil {
		return err
	}
	file := FilePathForVolumeName(volume.Name)
	if exists, err := ephemeraldiskutils.FileExists(file); !exists {
		// qemu-img takes the size in bytes or in Kibibytes/Mebibytes/...; lets take bytes
		size := strconv.FormatInt(volume.EmptyDisk.Capacity.ToDec().ScaledValue(0), 10)
		// #nosec No risk for attacker injection. Parameters are predefined strings
		if err := exec.Command("qemu-img", "create", "-f", "qcow2", file, size).Run(); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(file); err != nil {
		return err
	}
	return nil
}

func FilePathForVolumeName(volumeName string) string {
	return path.Join(EmptyDiskBaseDir, volumeName+".qcow2")
}
