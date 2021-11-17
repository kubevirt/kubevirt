package emptydisk

import (
	"os"
	"os/exec"
	"path"
	"strconv"

	v1 "kubevirt.io/api/core/v1"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

const emptyDiskBaseDir = "/var/run/libvirt/empty-disks/"

type emptyDiskCreator struct {
	emptyDiskBaseDir string
	discCreateFunc   func(filePath string, size string) error
}

func (c *emptyDiskCreator) CreateTemporaryDisks(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {

		if volume.EmptyDisk != nil {
			// qemu-img takes the size in bytes or in Kibibytes/Mebibytes/...; lets take bytes
			size := strconv.FormatInt(volume.EmptyDisk.Capacity.ToDec().ScaledValue(0), 10)
			file := filePathForVolumeName(c.emptyDiskBaseDir, volume.Name)
			if err := util.MkdirAllWithNosec(c.emptyDiskBaseDir); err != nil {
				return err
			}
			if _, err := os.Stat(file); os.IsNotExist(err) {
				if err := c.discCreateFunc(file, size); err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(file); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *emptyDiskCreator) FilePathForVolumeName(volumeName string) string {
	return filePathForVolumeName(c.emptyDiskBaseDir, volumeName)
}

func filePathForVolumeName(basedir string, volumeName string) string {
	return path.Join(basedir, volumeName+".qcow2")
}

func createQCOW(file string, size string) error {
	// #nosec No risk for attacket injection. Parameters are predefined strings
	return exec.Command("qemu-img", "create", "-f", "qcow2", file, size).Run()
}

func NewEmptyDiskCreator() *emptyDiskCreator {
	return &emptyDiskCreator{
		emptyDiskBaseDir: emptyDiskBaseDir,
		discCreateFunc:   createQCOW,
	}
}
