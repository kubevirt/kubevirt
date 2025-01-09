package image_volume

import (
	"fmt"
	"os"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	utildisk "kubevirt.io/kubevirt/pkg/util/disk"
)

const (
	mountBaseDir = "/var/run/kubevirt-image-volumes"
	IsosBaseDir  = "/var/run/kubevirt-private/image-volume-isos"
)

// GetImageVolumeSourcePath returns the base mount point path for the specified ImageVolume.
// This is where the volume is mounted in the system.
func GetImageVolumeSourcePath(volumeName string) string {
	return filepath.Join(mountBaseDir, volumeName)
}

// GetImageVolumeDiskPath resolves the path to a QCOW2 or RAW disk within the mounted ImageVolume.
// If a specific subPath is provided, it verifies the file exists at that location.
// Otherwise, it falls back to searching for a single disk file in a predefined default path within the volume.
func GetImageVolumeDiskPath(volumeName string, subPath string) (string, error) {
	return getImage(GetImageVolumeSourcePath(volumeName), subPath)
}

func GetImageVolumeDiskInfo(path string) (*utildisk.DiskInfo, error) {
	info, err := utildisk.FetchDiskInfo(path)
	if err != nil {
		return nil, err
	}
	if err = utildisk.VerifyImage(info); err != nil {
		return nil, err
	}
	return info, nil
}

// GetImageVolumeArtifactPath constructs the path to an artifact (file or directory) within the mounted ImageVolume.
// The artifact will later be used to create an ISO.
func GetImageVolumeArtifactPath(volumeName string, subPath string) string {
	return filepath.Join(GetImageVolumeSourcePath(volumeName), subPath)
}

// GetImageVolumeISOPath constructs the path to an ISO file derived from an artifact associated with the specified ImageVolume.
func GetImageVolumeISOPath(volumeName string) string {
	return filepath.Join(IsosBaseDir, volumeName+".iso")
}

func getImage(basePath, imagePath string) (string, error) {
	if imagePath != "" {
		p := filepath.Join(basePath, imagePath)
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("failed to determine custom image path %s: %w", imagePath, err)
		}
		return p, nil
	}
	fallbackPath := filepath.Join(basePath, utildisk.DiskSourceFallbackPath)
	files, err := os.ReadDir(fallbackPath)
	if err != nil {
		return "", fmt.Errorf("failed to check default image path %s: %w", fallbackPath, err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no file found in folder %s, no disk present", utildisk.DiskSourceFallbackPath)
	} else if len(files) > 1 {
		return "", fmt.Errorf("more than one file found in folder %s, only one disk is allowed", utildisk.DiskSourceFallbackPath)
	}

	return filepath.Join(fallbackPath, files[0].Name()), nil
}

func IsEphemeral(imageVolume *v1.ImageVolumeSource) bool {
	return imageVolume != nil && imageVolume.MountMode == v1.ImageVolumeMountModeEphemeral
}
func IsIsoArtifact(imageVolume *v1.ImageVolumeSource) bool {
	return imageVolume != nil && imageVolume.MountMode == v1.ImageVolumeMountModeIsoArtifact
}
