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

	diskDir := filepath.Dir(diskPath)
	return c.diskImgCreator.CreateDiskAndSetOwnership(vmi, diskDir, diskPath, pvcInfo.ClaimName, requestedSize)
}
