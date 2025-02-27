package virtiofs

import (
	"fmt"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	PlaceholderSocketVolumeMountPoint = "/var/run/sockets"
	PlaceholderSocketVolumeName       = "virtiofs-sockets"
)

// This is empty dir
var VirtioFSContainers = "virtiofs-containers"
var VirtioFSContainersMountBaseDir = filepath.Join(util.VirtShareDir, VirtioFSContainers)

func VirtioFSSocketPath(volumeName string) string {
	socketName := fmt.Sprintf("%s.sock", volumeName)
	return filepath.Join(VirtioFSContainersMountBaseDir, socketName)
}

func PlaceholderSocketName(volumeName string) string {
	return fmt.Sprintf("%s.sock", volumeName)
}

func PlaceholderSocketPath(volumeName string) string {
	return filepath.Join(PlaceholderSocketVolumeMountPoint, PlaceholderSocketName(volumeName))
}

func GetFilesystemPersistentVolumes(vmi *v1.VirtualMachineInstance) []v1.Volume {
	var vols []v1.Volume
	fss := storagetypes.GetFilesystemsFromVolumes(vmi)
	for _, volume := range vmi.Spec.Volumes {
		if _, ok := fss[volume.Name]; !ok {
			continue
		}
		if volume.VolumeSource.PersistentVolumeClaim != nil ||
			volume.VolumeSource.DataVolume != nil {
			vols = append(vols, volume)
		}
	}

	return vols
}

func HasFilesystemPersistentVolumes(vmi *v1.VirtualMachineInstance) bool {
	return len(GetFilesystemPersistentVolumes(vmi)) > 0
}

func FSMountPoint(volume *v1.Volume) string {
	switch {
	case volume.ConfigMap != nil:
		return config.GetConfigMapSourcePath(volume.Name)
	case volume.Secret != nil:
		return config.GetSecretSourcePath(volume.Name)
	case volume.ServiceAccount != nil:
		return config.ServiceAccountSourceDir
	case volume.DownwardAPI != nil:
		return config.GetDownwardAPISourcePath(volume.Name)
	default:
		return fmt.Sprintf("/%s", volume.Name)
	}
}
