package virthandler

import (
	"path/filepath"

	v1 "kubevirt.io/client-go/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

func changeOwnershipOfBlockDevices(vmiWithOnlyBlockPVCs *v1.VirtualMachineInstance, res isolation.IsolationResult) error {
	for i := range vmiWithOnlyBlockPVCs.Spec.Volumes {
		if volumeSource := &vmiWithOnlyBlockPVCs.Spec.Volumes[i].VolumeSource; volumeSource.PersistentVolumeClaim != nil {
			devPath := filepath.Join(string(filepath.Separator), "dev", vmiWithOnlyBlockPVCs.Spec.Volumes[i].Name)
			if err := diskutils.DefaultOwnershipManager.SetFileOwnership(filepath.Join(res.MountRoot(), devPath)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *VirtualMachineController) prepareStorage(vmiWithOnlyBlockPVCS, vmiWithAllPVCs *v1.VirtualMachineInstance) error {
	res, err := d.podIsolationDetector.Detect(vmiWithOnlyBlockPVCS)
	if err != nil {
		return err
	}

	return changeOwnershipOfBlockDevices(vmiWithOnlyBlockPVCS, res)
}
