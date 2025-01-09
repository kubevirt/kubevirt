package image_volume

import (
	v1 "kubevirt.io/api/core/v1"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
)

func CreateEphemeralImages(
	vmi *v1.VirtualMachineInstance,
	diskCreator ephemeraldisk.EphemeralDiskCreatorInterface,
) error {
	for _, volume := range vmi.Spec.Volumes {
		if IsEphemeral(volume.Image) {
			path, err := GetImageVolumeDiskPath(volume.Name, volume.Image.Path)
			if err != nil {
				return err
			}
			info, err := GetImageVolumeDiskInfo(path)
			if err != nil {
				return err
			}
			if err = diskCreator.CreateBackedImageForVolume(volume, path, info.Format); err != nil {
				return err
			}
		}
	}

	return nil
}
