package main

import (
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"

	"kubevirt.io/kubevirt/pkg/monitoring/vmstats"
)

type fakeVMCollector struct {
}

func (fc fakeVMCollector) Describe(_ chan<- *prometheus.Desc) {
}

// Collect needs to report all metrics to see it in docs
func (fc fakeVMCollector) Collect(ch chan<- prometheus.Metric) {
	vmCollector := newVMCollector()
	vmCollector.Collect(ch)
}

func RegisterFakeVMCollector() {
	prometheus.MustRegister(fakeVMCollector{})
}

func newVMCollector() *vmstats.VMCollector {
	vm := createVM(k6tv1.VirtualMachineStatusRunning)
	vmInformer, _ := testutils.NewFakeInformerFor(&k6tv1.VirtualMachine{})
	vmInformer.GetIndexer().Add(vm)

	blockMode := corev1.PersistentVolumeBlock
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim.ClaimName,
			Namespace: vm.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "test-pv",
			VolumeMode: &(blockMode),
		},
	}
	pvcInformer, _ := testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
	pvcInformer.GetIndexer().Add(pvc)

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pv",
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					Driver: "test-driver",
					VolumeAttributes: map[string]string{
						"clusterID":     "cluster0",
						"mounter":       "rbd",
						"imageFeatures": "layering",
						"mapOptions":    "krbd:rxbounce",
					},
				},
			},
		},
	}
	pvInformer, _ := testutils.NewFakeInformerFor(&corev1.PersistentVolume{})
	pvInformer.GetIndexer().Add(pv)

	return vmstats.SetupVMCollector(vmInformer, pvcInformer, pvInformer)
}

func createVM(status k6tv1.VirtualMachinePrintableStatus) *k6tv1.VirtualMachine {
	vmVolumes := []k6tv1.Volume{
		{
			VolumeSource: k6tv1.VolumeSource{
				PersistentVolumeClaim: &k6tv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: v1.PersistentVolumeClaimVolumeSource{
						ClaimName: "test-pvc",
					},
				},
			},
		},
	}

	return &k6tv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-vm"},
		Spec: k6tv1.VirtualMachineSpec{
			Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
				Spec: k6tv1.VirtualMachineInstanceSpec{
					Volumes: vmVolumes,
				},
			},
		},
		Status: k6tv1.VirtualMachineStatus{
			PrintableStatus: status,
			Conditions: []k6tv1.VirtualMachineCondition{
				{
					Type:               k6tv1.VirtualMachineReady,
					Status:             "any",
					Reason:             "any",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
	}
}
