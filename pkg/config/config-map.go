package config

import (
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

// GetConfigMapDiskPath returns path to ConfigMap iso image created based on a volume name
func GetConfigMapDiskPath(volumeName string) string {
	return filepath.Join(ConfigMapDisksDir, volumeName+".iso")
}

// CreateConfigMapDisks creates ConfigMap iso disks which are attached to vmis
func CreateConfigMapDisks(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.ConfigMap != nil {

			var filesPath []string
			filesPath, err := getFilesLayout(filepath.Join(ConfigMapSourceDir, volume.Name), volume.Name)
			if err != nil {
				return err
			}

			err = createIsoConfigImage(GetConfigMapDiskPath(volume.Name), filesPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
