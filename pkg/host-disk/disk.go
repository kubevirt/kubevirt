package hostdisk

import (
	"fmt"
	"path/filepath"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

type PVCDiskImgCreator struct {
	diskImgCreator diskImgCreator
}

func NewPVCDiskImgCreator(recorder record.EventRecorder, lessPVCSpaceToleration int, minimumPVCReserveBytes uint64) PVCDiskImgCreator {
	return PVCDiskImgCreator{
		diskImgCreator: newDiskImgCreator(recorder, lessPVCSpaceToleration, minimumPVCReserveBytes),
	}
}

func getPVCInfo(vmi *v1.VirtualMachineInstance, volumeName string) (*v1.PersistentVolumeClaimInfo, error) {
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName {
			if volumeStatus.PersistentVolumeClaimInfo.VolumeMode == nil || *volumeStatus.PersistentVolumeClaimInfo.VolumeMode != k8sv1.PersistentVolumeFilesystem {
				return nil, fmt.Errorf("volume %s is not in filesystem mode", volumeName)
			}
			return volumeStatus.PersistentVolumeClaimInfo, nil
		}
	}

	return nil, fmt.Errorf("no disk found")
}

func (c PVCDiskImgCreator) Create(vmi *v1.VirtualMachineInstance, volumeName, diskPath string) error {
	pvcInfo, err := getPVCInfo(vmi, volumeName)
	if err != nil {
		return err
	}

	capacity, hasCapacity := pvcInfo.Capacity[k8sv1.ResourceStorage]
	requested, hasRequests := pvcInfo.Requests[k8sv1.ResourceStorage]

	if !hasCapacity && !hasRequests {
		return fmt.Errorf("unable to determine capacity from PVC that provides no storage capacity or requests")
	}
	var requestedSize int64
	if hasRequests && (!hasCapacity || capacity.Value() > requested.Value()) {
		requestedSize = util.AlignImageSizeTo1MiB(requested.Value(), log.Log)
	} else {
		requestedSize = util.AlignImageSizeTo1MiB(capacity.Value(), log.Log)
	}

	if requestedSize == 0 {
		return fmt.Errorf("the size for volume %s is too low, must be at least 1MiB", pvcInfo.ClaimName)
	}

	// If it’s a PVC filesystem (we infer this indirectly by the presence of FilesystemOverhead), then prefer the actual
	// virtual size of the existing source image when creating the target qcow2.
	//
	// QEMU requires source and target images used for block migration to have
	// exactly the same virtual size. Creating a target image based solely on the
	// requested volume size may result in a size mismatch and cause migration
	// failures such as
	//
	//   "Source and target image have different sizes"
	//
	// Align the source image size to 1 MiB and use it for target image creation
	// to satisfy QEMU strict size checks.
	//
	// Hotplugged disks are not replaced with host disks, which is why we need to reapply this logic again."
	if isHotplugged(vmi, volumeName) {
		if volumeStatus := getVolumeStatus(vmi, volumeName); volumeStatus != nil {
			if volumeStatus.PersistentVolumeClaimInfo != nil && volumeStatus.PersistentVolumeClaimInfo.FilesystemOverhead != nil {
				if volumeStatus.Size > 0 && volumeStatus.Size < requestedSize {
					requestedSize = util.AlignImageSizeTo1MiB(volumeStatus.Size, log.Log)
				}
			}
		}
	}

	diskDir := filepath.Dir(diskPath)
	return c.diskImgCreator.CreateDiskAndSetOwnership(vmi, diskDir, diskPath, pvcInfo.ClaimName, requestedSize)
}

func isHotplugged(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	if status := getVolumeStatus(vmi, volumeName); status != nil {
		return status.HotplugVolume != nil
	}
	return false
}

func getVolumeStatus(vmi *v1.VirtualMachineInstance, volumeName string) *v1.VolumeStatus {
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name == volumeName {
			return &volumeStatus
		}
	}
	return nil
}
