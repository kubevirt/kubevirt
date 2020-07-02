package host_disk

import (
	"fmt"
	"os"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	v1 "kubevirt.io/client-go/api/v1"
)

func VerifyImages(vmi *v1.VirtualMachineInstance, podIsolationDetector isolation.PodIsolationDetector) error {
	res, err := podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect VMI pod: %v", err)
	}

	for _, volume := range vmi.Spec.Volumes {
		if hostDisk := volume.VolumeSource.HostDisk; hostDisk != nil && hostDisk.Path != "" {
			diskPath := hostdisk.GetMountedHostDiskPath(volume.Name, hostDisk.Path)
			if _, err := os.Stat(diskPath); os.IsNotExist(err) {
				continue
			}

			imageInfo, err := isolation.GetImageInfo(diskPath, res)
			if err != nil {
				return fmt.Errorf("failed to get image info: %v", err)
			}

			if err := isolation.VerifyImage(imageInfo); err != nil {
				return fmt.Errorf("invalid image in HostDisk %v: %v", volume.Name, err)
			}
		}
	}

	return nil
}
