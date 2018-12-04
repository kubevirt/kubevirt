package emptydisk

import (
	"os"
	"os/exec"
	"path"
	"strconv"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var EmptyDiskBaseDir = "/var/run/libvirt/empty-disks/"

func CreateTemporaryDisks(vmi *v1.VirtualMachineInstance) error {

	for _, volume := range vmi.Spec.Volumes {

		if volume.EmptyDisk != nil {
			// qemu-img takes the size in bytes or in Kibibytes/Mebibytes/...; lets take bytes
			size := strconv.FormatInt(volume.EmptyDisk.Capacity.ToDec().ScaledValue(0), 10)
			file := FilePathForVolumeName(volume.Name)
			if err := os.MkdirAll(EmptyDiskBaseDir, 0777); err != nil {
				return err
			}
			if _, err := os.Stat(file); os.IsNotExist(err) {
				if err := exec.Command("qemu-img", "create", "-f", "qcow2", file, size).Run(); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
	}

	return nil
}

func FilePathForVolumeName(volumeName string) string {
	return path.Join(EmptyDiskBaseDir, volumeName+".qcow2")
}
