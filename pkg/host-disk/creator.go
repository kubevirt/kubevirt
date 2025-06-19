package hostdisk

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

func newDiskImgCreator(recorder record.EventRecorder, lessPVCSpaceToleration int, minimumPVCReserveBytes uint64) diskImgCreator {
	return diskImgCreator{
		dirBytesAvailableFunc:  dirBytesAvailable,
		recorder:               recorder,
		lessPVCSpaceToleration: lessPVCSpaceToleration,
		minimumPVCReserveBytes: minimumPVCReserveBytes,
	}
}

type diskImgCreator struct {
	dirBytesAvailableFunc  func(path string, reserve uint64) (uint64, error)
	recorder               record.EventRecorder
	lessPVCSpaceToleration int
	minimumPVCReserveBytes uint64
}

func (c diskImgCreator) CreateDiskAndSetOwnership(vmi *v1.VirtualMachineInstance, diskDir, diskPath, volumeName string, requestedSize int64) error {
	fileExists, err := ephemeraldiskutils.FileExists(diskPath)
	if err != nil {
		return err
	}
	if !fileExists {
		if err := c.handleRequestedSizeAndCreateQcow2(vmi, diskDir, diskPath, volumeName, requestedSize); err != nil {
			return err
		}
	}
	// Change file ownership to the qemu user.
	if err := ephemeraldiskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(diskPath); err != nil {
		log.Log.Reason(err).Errorf("Couldn't set Ownership on %s: %v", diskPath, err)
		return err
	}
	return nil
}

func (c diskImgCreator) handleRequestedSizeAndCreateQcow2(vmi *v1.VirtualMachineInstance, diskDir, diskPath, volumeName string, requestedSize int64) error {
	size, err := c.dirBytesAvailableFunc(diskDir, c.minimumPVCReserveBytes)
	availableSize := int64(size)
	if err != nil {
		return err
	}
	if requestedSize > availableSize {
		requestedSize, err = c.shrinkRequestedSize(vmi, requestedSize, availableSize, volumeName)
		if err != nil {
			return err
		}
	}
	err = createQcow2(diskPath, requestedSize)
	if err != nil {
		log.Log.Reason(err).Errorf("Couldn't create a qcow2 file for disk path: %s, error: %v", diskPath, err)
		return err
	}
	return nil
}

func (c diskImgCreator) shrinkRequestedSize(vmi *v1.VirtualMachineInstance, requestedSize int64, availableSize int64, volumeName string) (int64, error) {
	// Some storage provisioners provide less space than requested, due to filesystem overhead etc.
	// We tolerate some difference in requested and available capacity up to some degree.
	// This can be configured with the "pvc-tolerate-less-space-up-to-percent" parameter in the kubevirt-config ConfigMap.
	// It is provided as argument to virt-launcher.
	toleratedSize := requestedSize * (100 - int64(c.lessPVCSpaceToleration)) / 100
	if toleratedSize > availableSize {
		return 0, fmt.Errorf("unable to create image on volume %s, not enough space, demanded size %d B is bigger than available space %d B, also after taking %v %% toleration into account",
			volumeName, uint64(requestedSize), availableSize, c.lessPVCSpaceToleration)
	}

	requestedSizeQ := resource.NewQuantity(requestedSize, resource.BinarySI)
	availableSizeQ := resource.NewQuantity(availableSize, resource.BinarySI)

	msg := fmt.Sprintf("PV size too small: expected %s, found %s. Using it anyway, it is within %v %% toleration", requestedSizeQ.String(), availableSizeQ.String(), c.lessPVCSpaceToleration)
	log.Log.Info(msg)
	c.recorder.Event(vmi, EventTypeToleratedSmallPV, EventReasonToleratedSmallPV, msg)
	return availableSize, nil
}
