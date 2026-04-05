/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package admitters

import (
	"context"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type Admitter struct {
	virtClient    kubecli.KubevirtClient
	ctx           context.Context
	ar            *admissionv1.AdmissionRequest
	vm            *v1.VirtualMachine
	clusterConfig *virtconfig.ClusterConfig
}

func NewAdmitter(virtClient kubecli.KubevirtClient, ctx context.Context, ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) *Admitter {
	return &Admitter{
		virtClient:    virtClient,
		ctx:           ctx,
		ar:            ar,
		vm:            vm,
		clusterConfig: clusterConfig,
	}
}

func (a Admitter) AdmitStatus() []metav1.StatusCause {
	causes := a.validateSnapshotStatus()
	if len(causes) > 0 {
		return causes
	}

	causes = a.validateRestoreStatus()
	if len(causes) > 0 {
		return causes
	}
	return causes
}

func (a Admitter) Admit() ([]metav1.StatusCause, error) {
	causes, err := a.validateVirtualMachineDataVolumeTemplateNamespace()
	if err != nil || len(causes) > 0 {
		return causes, err
	}

	causes = a.AdmitStatus()
	if len(causes) > 0 {
		return causes, err
	}

	return causes, nil
}

func Admit(virtClient kubecli.KubevirtClient, ctx context.Context, ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) ([]metav1.StatusCause, error) {
	storageAdmitter := NewAdmitter(virtClient, ctx, ar, vm, clusterConfig)
	return storageAdmitter.Admit()
}

func AdmitStatus(virtClient kubecli.KubevirtClient, ctx context.Context, ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) []metav1.StatusCause {
	storageAdmitter := NewAdmitter(virtClient, ctx, ar, vm, clusterConfig)
	return storageAdmitter.AdmitStatus()
}
