package migration

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	virtstorage "kubevirt.io/api/storage/v1alpha1"
	virtstoragev1alpha1 "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/api"
)

func createVMIWithPVCs(name, ns string, pvcs ...string) *virtv1.VirtualMachineInstance {
	vmi := api.NewMinimalVMIWithNS(ns, name)
	for _, p := range pvcs {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: p,
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.DiskBusVirtio,
				},
			},
		})
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: p,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: p,
					}},
			},
		})
	}
	return vmi
}

func createVMFromVMI(vmi *virtv1.VirtualMachineInstance) *virtv1.VirtualMachine {
	return &virtv1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachine",
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      vmi.Name,
			Namespace: vmi.Namespace,
		},
		Spec: virtv1.VirtualMachineSpec{
			Running: pointer.BoolPtr(true),
			Template: &virtv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: *(vmi.ObjectMeta.DeepCopy()),
				Spec:       *(vmi.Spec.DeepCopy()),
			},
		},
	}
}

func updateMigrateVolumesVMIStatus(vmi *virtv1.VirtualMachineInstance, migVols []virtstoragev1alpha1.MigratedVolume,
	phase *virtstorage.VolumeMigrationPhase) {
	for _, v := range migVols {
		vmi.Status.MigratedVolumes = append(vmi.Status.MigratedVolumes,
			virtv1.StorageMigratedVolumeInfo{
				VolumeName:         v.SourceClaim,
				SourcePVCInfo:      &virtv1.PersistentVolumeClaimInfo{ClaimName: v.SourceClaim},
				DestinationPVCInfo: &virtv1.PersistentVolumeClaimInfo{ClaimName: v.DestinationClaim},
				MigrationPhase:     phase,
			})
	}
}
