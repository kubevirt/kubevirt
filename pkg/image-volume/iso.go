package image_volume

import (
	v1 "kubevirt.io/api/core/v1"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	utildisk "kubevirt.io/kubevirt/pkg/util/disk"
)

func CreateISOImages(
	vmi *v1.VirtualMachineInstance,
) error {
	for _, volume := range vmi.Spec.Volumes {
		if !IsIsoArtifact(volume.Image) {
			continue
		}
		path := GetImageVolumeArtifactPath(volume.Name, volume.Image.Path)

		filesPath, err := utildisk.GetFilesLayoutForISO(path)
		if err != nil {
			return err
		}

		isoPath := GetImageVolumeISOPath(volume.Name)

		label := volume.Name
		if err = utildisk.CreateIsoImage(isoPath, label, filesPath); err != nil {
			return err
		}

		if err := ephemeraldiskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(isoPath); err != nil {
			return err
		}
	}

	return nil
}
