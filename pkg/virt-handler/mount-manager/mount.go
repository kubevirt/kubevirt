package mount

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	container_disk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	hotplug_volume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

// MountInfo wraps all the mount information
type MountInfo struct {
	containerDisksInfo map[string]*containerdisk.DiskInfo
}

func (info *MountInfo) GetContainerDisksInfo() map[string]*containerdisk.DiskInfo {
	return info.containerDisksInfo
}

// MountManager handles all the mount operations required by KubeVirt
type MountManager interface {
	Mount(vmi *v1.VirtualMachineInstance) (MountInfo, error)
	Unmount(vmi *v1.VirtualMachineInstance) error
	ContainerDiskMountsReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error)
	SyncHotplugMounts(vmi *v1.VirtualMachineInstance) error
	SyncHotplugUnmounts(vmi *v1.VirtualMachineInstance) error
	IsHotplugVolumeMounted(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID) (bool, error)
}

type mountManager struct {
	containerDiskMounter container_disk.Mounter
	hotplugVolumeMounter hotplug_volume.VolumeMounter
}

func NewMounter(virtPrivateDir string, podIsolationDetector isolation.PodIsolationDetector, clusterConfig *virtconfig.ClusterConfig, kubeletPodsDir string) MountManager {
	mountRecorder := mountutils.NewMountRecorder(virtPrivateDir)
	return &mountManager{
		containerDiskMounter: container_disk.NewMounter(podIsolationDetector, clusterConfig, mountRecorder),
		hotplugVolumeMounter: hotplug_volume.NewVolumeMounter(mountRecorder, kubeletPodsDir),
	}
}

// ContainerDiskMountsReady returns if the mount points are ready to be used
func (m *mountManager) ContainerDiskMountsReady(vmi *v1.VirtualMachineInstance, notInitializedSince time.Time) (bool, error) {
	// Check container diks are ready to be mounted
	return m.containerDiskMounter.ContainerDisksReady(vmi, notInitializedSince)
}

// Mount mounts the volumes managed directly by KubeVirt
func (m *mountManager) Mount(vmi *v1.VirtualMachineInstance) (MountInfo, error) {
	disksInfo, err := m.containerDiskMounter.MountAndVerify(vmi)
	if err != nil {
		return MountInfo{}, fmt.Errorf("failed to mount container disks: %v", err)
	}

	attachmentPodUID := types.UID("")
	if vmi.Status.MigrationState != nil {
		attachmentPodUID = vmi.Status.MigrationState.TargetAttachmentPodUID
	}
	if attachmentPodUID != types.UID("") {
		if err = m.hotplugVolumeMounter.MountFromPod(vmi, attachmentPodUID); err != nil {
			return MountInfo{}, fmt.Errorf("failed to mount hotplug volumes: %v", err)
		}

	} else {
		if err = m.hotplugVolumeMounter.Mount(vmi); err != nil {
			return MountInfo{}, fmt.Errorf("failed to mount hotplug volumes: %v", err)
		}
	}
	return MountInfo{
		containerDisksInfo: disksInfo,
	}, nil
}

// Unmount unmounts the volumes managed directly by KubeVirt
func (m *mountManager) Unmount(vmi *v1.VirtualMachineInstance) error {
	errCd := m.containerDiskMounter.Unmount(vmi)
	errHp := m.hotplugVolumeMounter.UnmountAll(vmi)
	if errCd != nil {
		// An error occured for both kind of volumes
		if errHp != nil {
			return fmt.Errorf("failed unmounting container disks: %v and hotplugged volumes: %v", errCd, errHp)
		}
		return errCd
	}
	return errHp
}

// SyncHotplugMounts mounts the volumes managed directly by KubeVirt on a running VMI
func (m *mountManager) SyncHotplugMounts(vmi *v1.VirtualMachineInstance) error {
	return m.hotplugVolumeMounter.Mount(vmi)
}

// SyncHotplugUnmounts unmounts the volumes managed directly by KubeVirt on a running VMI
func (m *mountManager) SyncHotplugUnmounts(vmi *v1.VirtualMachineInstance) error {
	return m.hotplugVolumeMounter.Unmount(vmi)
}

// IsHotplugVolumeMounted returns if a volume is mounted
func (m *mountManager) IsHotplugVolumeMounted(vmi *v1.VirtualMachineInstance, volume string, sourceUID types.UID) (bool, error) {
	return m.hotplugVolumeMounter.IsMounted(vmi, volume, sourceUID)
}
