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
	PlaceholderSocketDir = "/var/run/sockets"
	ExtraVolName         = "virtiofs-sockets"
)

const (
	dispatcher = "virtiofs-dispatcher"
)

// This is empty dir
var VirtioFSContainers = "virtiofs-containers"
var VirtioFSContainersMountBaseDir = filepath.Join(util.VirtShareDir, VirtioFSContainers)

func VirtioFSSocketPath(volumeName string) string {
	socketName := fmt.Sprintf("%s.sock", volumeName)
	return filepath.Join(VirtioFSContainersMountBaseDir, socketName)
}

func VirtiofsPlaceholderSocketName(volumeName string) string {
	return fmt.Sprintf("%s.sock", volumeName)
}

func VirtiofsPlaceholderSocket(volumeName string) string {
	return filepath.Join(PlaceholderSocketDir, VirtiofsPlaceholderSocketName(volumeName))
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

func VirtioFSMountPoint(volume *v1.Volume) string {
	volumeMountPoint := fmt.Sprintf("/%s", volume.Name)

	if volume.ConfigMap != nil {
		volumeMountPoint = config.GetConfigMapSourcePath(volume.Name)
	} else if volume.Secret != nil {
		volumeMountPoint = config.GetSecretSourcePath(volume.Name)
	} else if volume.ServiceAccount != nil {
		volumeMountPoint = config.ServiceAccountSourceDir
	} else if volume.DownwardAPI != nil {
		volumeMountPoint = config.GetDownwardAPISourcePath(volume.Name)
	}

	return volumeMountPoint
}
